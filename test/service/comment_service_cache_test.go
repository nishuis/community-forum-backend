// test/service —— CommentService 缓存行为黑盒单测。
// 组合 sqlmock + miniredis：验证评论分页列表 Cache-Aside 命中/未命中，
// 以及发评论/删评论/编辑评论后的按模式批量失效（不影响其他帖子缓存）。
package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nishuis/community-forum-backend/internal/repository"
	"github.com/nishuis/community-forum-backend/internal/service"
)

// commentRow 构造 comments 表查询结果行。
func commentRow(commentID, postID, authorID int64, content string) *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows([]string{
		"comment_id", "created_at", "updated_at", "deleted_at", "post_id", "parent_comment_id",
		"author_id", "is_top", "top_end_at", "status", "content", "like_count",
	}).AddRow(commentID, now, now, nil, postID, 0, authorID, 0, nil, 0, content, 0)
}

// TestGetCommentPage_MissThenHit 首次查 DB 回填，二次命中缓存（帖子存在性校验仍查 DB）。
func TestGetCommentPage_MissThenHit(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	srv, cacheSvc := newMiniredisCache(t)

	svc := service.NewCommentService(
		repository.NewCommentRepo(newGormDB(t, db)),
		repository.NewPostRepo(newGormDB(t, db)),
		cacheSvc,
	)

	const postID = 1001
	// 第一次：帖子存在性校验(SELECT posts) + COUNT + 查评论列表(空)
	mock.ExpectQuery("SELECT \\* FROM `posts`").WillReturnRows(postRow(postID, 7, "标题"))
	mock.ExpectQuery("count").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT \\* FROM `comments`").WillReturnRows(sqlmock.NewRows([]string{"comment_id"}))

	list, total, err := svc.GetCommentPageByPostId(context.Background(), postID, 1, 20)
	if err != nil {
		t.Fatalf("首次查询失败: %v", err)
	}
	if total != 0 || len(list) != 0 {
		t.Fatalf("首次查询返回错误: list=%v total=%d", list, total)
	}
	if !srv.Exists("cf:comment:1001:1:20") {
		t.Fatal("首次查询后未回填缓存")
	}

	// 第二次：命中缓存，只查帖子存在性，不再查评论（COUNT/SELECT comments 不应再出现）
	mock.ExpectQuery("SELECT \\* FROM `posts`").WillReturnRows(postRow(postID, 7, "标题"))
	list2, total2, err := svc.GetCommentPageByPostId(context.Background(), postID, 1, 20)
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

// TestCreateComment_InvalidatesCommentPages 发评论后按模式删除该帖所有分页缓存，不影响其他帖子。
func TestCreateComment_InvalidatesCommentPages(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	srv, cacheSvc := newMiniredisCache(t)

	svc := service.NewCommentService(
		repository.NewCommentRepo(newGormDB(t, db)),
		repository.NewPostRepo(newGormDB(t, db)),
		cacheSvc,
	)

	const postID = 1001
	// 预置该帖两个分页缓存 + 另一帖子的缓存
	cacheSvc.SetJSON(context.Background(), "cf:comment:1001:1:20", map[string]any{"total": 1}, time.Minute)
	cacheSvc.SetJSON(context.Background(), "cf:comment:1001:2:20", map[string]any{"total": 2}, time.Minute)
	cacheSvc.SetJSON(context.Background(), "cf:comment:1002:1:20", map[string]any{"total": 9}, time.Minute)

	// 发评论：帖子存在性校验 + 事务内 INSERT
	mock.ExpectQuery("SELECT \\* FROM `posts`").WillReturnRows(postRow(postID, 7, "标题"))
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `comments`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if _, err := svc.CreateComment(context.Background(), postID, 0, "新评论", 7); err != nil {
		t.Fatalf("发表评论失败: %v", err)
	}
	if srv.Exists("cf:comment:1001:1:20") || srv.Exists("cf:comment:1001:2:20") {
		t.Fatal("发评论后 1001 的评论分页缓存未失效")
	}
	if !srv.Exists("cf:comment:1002:1:20") {
		t.Fatal("不应误删其他帖子的评论缓存")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock 期望未全部满足: %v", err)
	}
}

// TestDeleteComment_InvalidatesCommentPages 删评论后按模式删除该帖所有分页缓存。
func TestDeleteComment_InvalidatesCommentPages(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	srv, cacheSvc := newMiniredisCache(t)

	svc := service.NewCommentService(
		repository.NewCommentRepo(newGormDB(t, db)),
		repository.NewPostRepo(newGormDB(t, db)),
		cacheSvc,
	)

	const postID = 1001
	const commentID = 500
	const userID = 7
	cacheSvc.SetJSON(context.Background(), "cf:comment:1001:1:20", map[string]any{"total": 1}, time.Minute)

	// 删评论：查评论(含作者预加载) → 事务内软删除
	mock.ExpectQuery("SELECT \\* FROM `comments`").WillReturnRows(commentRow(commentID, postID, userID, "内容"))
	mock.ExpectQuery("SELECT \\* FROM `users`").WillReturnRows(userRow(userID, "alice", "a@e.com", "hash"))
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `comments`").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := svc.DeleteComment(context.Background(), commentID, userID); err != nil {
		t.Fatalf("删除评论失败: %v", err)
	}
	if srv.Exists("cf:comment:1001:1:20") {
		t.Fatal("删评论后评论分页缓存未失效")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock 期望未全部满足: %v", err)
	}
}

// TestEditComment_InvalidatesCommentPages 编辑评论后按模式删除该帖所有分页缓存。
func TestEditComment_InvalidatesCommentPages(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	srv, cacheSvc := newMiniredisCache(t)

	svc := service.NewCommentService(
		repository.NewCommentRepo(newGormDB(t, db)),
		repository.NewPostRepo(newGormDB(t, db)),
		cacheSvc,
	)

	const postID = 1001
	const commentID = 500
	const userID = 7
	cacheSvc.SetJSON(context.Background(), "cf:comment:1001:1:20", map[string]any{"total": 1}, time.Minute)

	// 编辑评论：查评论+预加载 → 事务内更新内容 → 重新查评论+预加载
	mock.ExpectQuery("SELECT \\* FROM `comments`").WillReturnRows(commentRow(commentID, postID, userID, "旧内容"))
	mock.ExpectQuery("SELECT \\* FROM `users`").WillReturnRows(userRow(userID, "alice", "a@e.com", "hash"))
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `comments`").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery("SELECT \\* FROM `comments`").WillReturnRows(commentRow(commentID, postID, userID, "新内容"))
	mock.ExpectQuery("SELECT \\* FROM `users`").WillReturnRows(userRow(userID, "alice", "a@e.com", "hash"))

	if _, err := svc.EditComment(context.Background(), commentID, "新内容", userID); err != nil {
		t.Fatalf("编辑评论失败: %v", err)
	}
	if srv.Exists("cf:comment:1001:1:20") {
		t.Fatal("编辑评论后评论分页缓存未失效")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock 期望未全部满足: %v", err)
	}
}
