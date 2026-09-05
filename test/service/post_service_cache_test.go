// test/service —— PostService 缓存行为黑盒单测。
// 组合 sqlmock（模拟 MySQL）+ miniredis（模拟 Redis）：
// 验证帖子详情 Cache-Aside 命中/未命中/空值占位，以及编辑/删除后的写后失效。
package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nishuis/community-forum-backend/configs"
	"github.com/nishuis/community-forum-backend/internal/errs"
	"github.com/nishuis/community-forum-backend/internal/repository"
	"github.com/nishuis/community-forum-backend/internal/service"
)

// postRow 构造 posts 表查询结果行。
func postRow(postID, authorID int64, title string) *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows([]string{
		"post_id", "created_at", "updated_at", "deleted_at", "author_id",
		"title", "summary", "content", "like_count", "is_top", "top_end_at", "status", "last_comment_at",
	}).AddRow(postID, now, now, nil, authorID, title, "", "内容", 0, 0, nil, 0, nil)
}

// TestGetPostById_MissThenHit 首次未命中查 DB 并回填，二次命中不再查 DB。
func TestGetPostById_MissThenHit(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	srv, cacheSvc := newMiniredisCache(t)

	svc := service.NewPostService(
		repository.NewPostRepo(newGormDB(t, db)),
		repository.NewUserRepo(newGormDB(t, db)),
		&configs.Config{},
		cacheSvc,
	)

	const postID = 1001
	// 第一次：未命中 → 查 DB → 回填
	mock.ExpectQuery("SELECT \\* FROM `posts`").WillReturnRows(postRow(postID, 7, "标题A"))
	got, err := svc.GetPostById(context.Background(), postID)
	if err != nil {
		t.Fatalf("首次查询失败: %v", err)
	}
	if got == nil || got.PostId != postID || got.Title != "标题A" {
		t.Fatalf("首次查询返回错误: %+v", got)
	}
	if !srv.Exists("cf:post:1001") {
		t.Fatal("首次查询后未回填缓存")
	}

	// 第二次：命中缓存，不再查 DB（mock 已无剩余期望，若查 DB 会报错）
	got2, err := svc.GetPostById(context.Background(), postID)
	if err != nil {
		t.Fatalf("缓存命中查询失败: %v", err)
	}
	if got2 == nil || got2.PostId != postID || got2.Title != "标题A" {
		t.Fatalf("缓存命中返回错误: %+v", got2)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock 期望未全部满足: %v", err)
	}
}

// TestGetPostById_RecordNotFound_EmptyMarker 不存在的帖子：DB 无记录写空值占位，二次命中直接返回不存在。
func TestGetPostById_RecordNotFound_EmptyMarker(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	srv, cacheSvc := newMiniredisCache(t)

	svc := service.NewPostService(
		repository.NewPostRepo(newGormDB(t, db)),
		repository.NewUserRepo(newGormDB(t, db)),
		&configs.Config{},
		cacheSvc,
	)

	const postID = 9999
	// 第一次：DB 无记录
	mock.ExpectQuery("SELECT \\* FROM `posts`").WillReturnRows(sqlmock.NewRows([]string{"post_id"}))
	_, err := svc.GetPostById(context.Background(), postID)
	if !errors.Is(err, errs.ErrPostNotExist) {
		t.Fatalf("期望 ErrPostNotExist，实际: %v", err)
	}
	if !srv.Exists("cf:post:9999") {
		t.Fatal("未写入空值占位")
	}

	// 第二次：命中空值占位，直接返回不存在，不再查 DB
	_, err = svc.GetPostById(context.Background(), postID)
	if !errors.Is(err, errs.ErrPostNotExist) {
		t.Fatalf("空值占位命中后应返回 ErrPostNotExist，实际: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock 期望未全部满足: %v", err)
	}
}

// TestUpdatePost_InvalidatesCache 编辑成功后删除帖子详情缓存。
func TestUpdatePost_InvalidatesCache(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	srv, cacheSvc := newMiniredisCache(t)

	svc := service.NewPostService(
		repository.NewPostRepo(newGormDB(t, db)),
		repository.NewUserRepo(newGormDB(t, db)),
		&configs.Config{},
		cacheSvc,
	)

	const postID = 1001
	const userID = 7
	// 预置缓存
	cacheSvc.SetJSON(context.Background(), "cf:post:1001", map[string]any{"post_id": postID}, time.Minute)

	// 编辑流程：查帖子(作者校验) → 事务内 UPDATE
	mock.ExpectQuery("SELECT \\* FROM `posts`").WillReturnRows(postRow(postID, userID, "旧标题"))
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `posts`").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := svc.UpdatePost(context.Background(), userID, postID, "新标题", ""); err != nil {
		t.Fatalf("编辑帖子失败: %v", err)
	}
	if srv.Exists("cf:post:1001") {
		t.Fatal("更新成功后缓存未失效")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock 期望未全部满足: %v", err)
	}
}

// TestDeletePost_InvalidatesCache 删除成功后删除帖子详情缓存。
func TestDeletePost_InvalidatesCache(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	srv, cacheSvc := newMiniredisCache(t)

	svc := service.NewPostService(
		repository.NewPostRepo(newGormDB(t, db)),
		repository.NewUserRepo(newGormDB(t, db)),
		&configs.Config{},
		cacheSvc,
	)

	const postID = 1001
	const userID = 7
	cacheSvc.SetJSON(context.Background(), "cf:post:1001", map[string]any{"post_id": postID}, time.Minute)

	// 删除流程：查帖子(作者校验) → 事务内软删除 UPDATE
	mock.ExpectQuery("SELECT \\* FROM `posts`").WillReturnRows(postRow(postID, userID, "标题"))
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `posts`").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := svc.DeletePost(context.Background(), userID, postID); err != nil {
		t.Fatalf("删除帖子失败: %v", err)
	}
	if srv.Exists("cf:post:1001") {
		t.Fatal("删除成功后缓存未失效")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock 期望未全部满足: %v", err)
	}
}

// TestGetPostByAuthorId_MissThenHit 作者帖子列表：首次查 DB 回填，二次命中缓存不再查 DB。
func TestGetPostByAuthorId_MissThenHit(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	srv, cacheSvc := newMiniredisCache(t)

	svc := service.NewPostService(
		repository.NewPostRepo(newGormDB(t, db)),
		repository.NewUserRepo(newGormDB(t, db)),
		&configs.Config{},
		cacheSvc,
	)

	const authorID = 7
	// 第一次：用户存在性校验 + 查帖子列表(空)
	mock.ExpectQuery("SELECT \\* FROM `users`").WillReturnRows(userRow(authorID, "alice", "a@e.com", "hash"))
	mock.ExpectQuery("SELECT \\* FROM `posts`").WillReturnRows(sqlmock.NewRows([]string{"post_id"}))

	list, err := svc.GetPostByAuthorId(context.Background(), authorID)
	if err != nil {
		t.Fatalf("首次查询失败: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("首次查询应返回空列表, 实际: %v", list)
	}
	if !srv.Exists("cf:user:7:posts") {
		t.Fatal("首次查询后未回填作者列表缓存")
	}

	// 第二次：帖子列表命中缓存，但用户存在性校验仍直查 DB（刻意保持正确性优先），不再查 posts 表
	mock.ExpectQuery("SELECT \\* FROM `users`").WillReturnRows(userRow(authorID, "alice", "a@e.com", "hash"))
	list2, err := svc.GetPostByAuthorId(context.Background(), authorID)
	if err != nil {
		t.Fatalf("缓存命中查询失败: %v", err)
	}
	if len(list2) != 0 {
		t.Fatalf("缓存命中返回错误: %v", list2)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock 期望未全部满足: %v", err)
	}
}

// TestGetPostByTitleExact_MissThenHit 精确标题查询：首次查 DB 回填，二次命中缓存。
func TestGetPostByTitleExact_MissThenHit(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	srv, cacheSvc := newMiniredisCache(t)

	svc := service.NewPostService(
		repository.NewPostRepo(newGormDB(t, db)),
		repository.NewUserRepo(newGormDB(t, db)),
		&configs.Config{},
		cacheSvc,
	)

	const title = "标题A"
	mock.ExpectQuery("SELECT \\* FROM `posts`").WillReturnRows(sqlmock.NewRows([]string{"post_id"}))

	list, err := svc.GetPostByTitleExact(context.Background(), title)
	if err != nil {
		t.Fatalf("首次查询失败: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("首次查询应返回空列表, 实际: %v", list)
	}
	if !srv.Exists("cf:post:title:标题A") {
		t.Fatal("首次查询后未回填标题缓存")
	}

	list2, err := svc.GetPostByTitleExact(context.Background(), title)
	if err != nil {
		t.Fatalf("缓存命中查询失败: %v", err)
	}
	if len(list2) != 0 {
		t.Fatalf("缓存命中返回错误: %v", list2)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock 期望未全部满足: %v", err)
	}
}

// TestShowByKeyWordOffset_MissThenHit 关键词搜索：首次查 DB(COUNT+SELECT)回填，二次命中缓存。
func TestShowByKeyWordOffset_MissThenHit(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	srv, cacheSvc := newMiniredisCache(t)

	svc := service.NewPostService(
		repository.NewPostRepo(newGormDB(t, db)),
		repository.NewUserRepo(newGormDB(t, db)),
		&configs.Config{},
		cacheSvc,
	)

	const kw = "golang"
	// 第一次：COUNT + SELECT posts(空)，空列表无 Preload 额外查询
	mock.ExpectQuery("count").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT \\* FROM `posts`").WillReturnRows(sqlmock.NewRows([]string{"post_id"}))

	list, total, err := svc.ShowByKeyWordOffset(context.Background(), kw, 1, 20)
	if err != nil {
		t.Fatalf("首次查询失败: %v", err)
	}
	if total != 0 || len(list) != 0 {
		t.Fatalf("首次查询返回错误: list=%v total=%d", list, total)
	}
	if !srv.Exists("cf:post:search:golang:1:20") {
		t.Fatal("首次查询后未回填搜索缓存")
	}

	// 第二次：命中缓存，不再查 DB
	list2, total2, err := svc.ShowByKeyWordOffset(context.Background(), kw, 1, 20)
	if err != nil {
		t.Fatalf("缓存命中查询失败: %v", err)
	}
	if total2 != 0 || len(list2) != 0 {
		t.Fatalf("缓存命中返回错误: list=%v total=%d", list2, total2)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock 期望未全部满足: %v", err)
	}
}

// TestCreatePost_InvalidatesListCaches 发帖后失效作者列表、标题、搜索缓存。
func TestCreatePost_InvalidatesListCaches(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	srv, cacheSvc := newMiniredisCache(t)

	svc := service.NewPostService(
		repository.NewPostRepo(newGormDB(t, db)),
		repository.NewUserRepo(newGormDB(t, db)),
		&configs.Config{},
		cacheSvc,
	)

	const authorID = 7
	const title = "新帖标题"
	// 预置缓存
	cacheSvc.SetJSON(context.Background(), "cf:user:7:posts", []any{map[string]any{"post_id": 1}}, time.Minute)
	cacheSvc.SetJSON(context.Background(), "cf:post:title:新帖标题", []any{}, time.Minute)
	cacheSvc.SetJSON(context.Background(), "cf:post:search:kw:1:20", map[string]any{"total": 1}, time.Minute)

	// 发帖：用户存在性校验 + 事务内 INSERT
	mock.ExpectQuery("SELECT \\* FROM `users`").WillReturnRows(userRow(authorID, "alice", "a@e.com", "hash"))
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `posts`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if _, err := svc.CreatePost(context.Background(), title, "内容", authorID); err != nil {
		t.Fatalf("发帖失败: %v", err)
	}
	if srv.Exists("cf:user:7:posts") {
		t.Fatal("发帖后作者列表缓存未失效")
	}
	if srv.Exists("cf:post:title:新帖标题") {
		t.Fatal("发帖后标题缓存未失效")
	}
	if srv.Exists("cf:post:search:kw:1:20") {
		t.Fatal("发帖后搜索缓存未失效")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock 期望未全部满足: %v", err)
	}
}

// TestUpdatePost_InvalidatesListCaches 编辑帖子后失效详情、作者列表、搜索、旧/新标题缓存。
func TestUpdatePost_InvalidatesListCaches(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	srv, cacheSvc := newMiniredisCache(t)

	svc := service.NewPostService(
		repository.NewPostRepo(newGormDB(t, db)),
		repository.NewUserRepo(newGormDB(t, db)),
		&configs.Config{},
		cacheSvc,
	)

	const postID = 1001
	const userID = 7
	// 预置缓存
	cacheSvc.SetJSON(context.Background(), "cf:post:1001", map[string]any{"post_id": postID}, time.Minute)
	cacheSvc.SetJSON(context.Background(), "cf:user:7:posts", []any{}, time.Minute)
	cacheSvc.SetJSON(context.Background(), "cf:post:title:旧标题", []any{}, time.Minute)
	cacheSvc.SetJSON(context.Background(), "cf:post:title:新标题", []any{}, time.Minute)
	cacheSvc.SetJSON(context.Background(), "cf:post:search:kw:1:20", map[string]any{"total": 1}, time.Minute)

	// 编辑：查帖子(作者校验) → 事务内 UPDATE
	mock.ExpectQuery("SELECT \\* FROM `posts`").WillReturnRows(postRow(postID, userID, "旧标题"))
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `posts`").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := svc.UpdatePost(context.Background(), userID, postID, "新标题", ""); err != nil {
		t.Fatalf("编辑帖子失败: %v", err)
	}
	if srv.Exists("cf:post:1001") {
		t.Fatal("编辑后详情缓存未失效")
	}
	if srv.Exists("cf:user:7:posts") {
		t.Fatal("编辑后作者列表缓存未失效")
	}
	if srv.Exists("cf:post:title:旧标题") {
		t.Fatal("编辑后旧标题缓存未失效")
	}
	if srv.Exists("cf:post:title:新标题") {
		t.Fatal("编辑后新标题缓存未失效")
	}
	if srv.Exists("cf:post:search:kw:1:20") {
		t.Fatal("编辑后搜索缓存未失效")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock 期望未全部满足: %v", err)
	}
}

// TestDeletePost_InvalidatesCommentPages 删帖后连带失效该帖评论分页缓存，不影响其他帖子。
func TestDeletePost_InvalidatesCommentPages(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	srv, cacheSvc := newMiniredisCache(t)

	svc := service.NewPostService(
		repository.NewPostRepo(newGormDB(t, db)),
		repository.NewUserRepo(newGormDB(t, db)),
		&configs.Config{},
		cacheSvc,
	)

	const postID = 1001
	const userID = 7
	// 预置该帖评论分页缓存 + 另一帖子评论分页缓存
	cacheSvc.SetJSON(context.Background(), "cf:comment:1001:1:20", map[string]any{"total": 1}, time.Minute)
	cacheSvc.SetJSON(context.Background(), "cf:comment:1002:1:20", map[string]any{"total": 9}, time.Minute)

	// 删帖：查帖子(作者校验) → 事务内软删除
	mock.ExpectQuery("SELECT \\* FROM `posts`").WillReturnRows(postRow(postID, userID, "标题"))
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `posts`").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := svc.DeletePost(context.Background(), userID, postID); err != nil {
		t.Fatalf("删帖失败: %v", err)
	}
	if srv.Exists("cf:comment:1001:1:20") {
		t.Fatal("删帖后该帖评论分页缓存未失效")
	}
	if !srv.Exists("cf:comment:1002:1:20") {
		t.Fatal("不应误删其他帖子的评论缓存")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock 期望未全部满足: %v", err)
	}
}
