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
