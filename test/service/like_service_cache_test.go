// test/service —— LikeService 缓存行为黑盒单测。
// 组合 sqlmock + miniredis：验证点赞状态/计数/我的点赞 Cache-Aside，
// 以及点赞/取消后 status/count/我的点赞/帖子详情/评论分页的连带失效。
package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nishuis/community-forum-backend/internal/domain"
	"github.com/nishuis/community-forum-backend/internal/repository"
	"github.com/nishuis/community-forum-backend/internal/service"
)

// likeRow 构造 likes 表查询结果行。
func likeRow(likeID, userID int64, targetType int8, targetID int64) *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows([]string{
		"like_id", "user_id", "target_type", "target_id", "created_at", "updated_at",
	}).AddRow(likeID, userID, targetType, targetID, now, now)
}

// TestIsUserLike_MissThenHit 点赞状态：首次查 DB 回填，二次命中缓存不再查 DB。
func TestIsUserLike_MissThenHit(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	srv, cacheSvc := newMiniredisCache(t)

	gdb := newGormDB(t, db)
	svc := service.NewLikeService(
		repository.NewLikeRepo(gdb),
		repository.NewPostRepo(gdb),
		repository.NewCommentRepo(gdb),
		cacheSvc,
	)

	const userID = 7
	const targetID = 1001
	// 第一次：Count 查询返回 0（未点赞）
	mock.ExpectQuery("count").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	liked, err := svc.IsUserLike(context.Background(), userID, domain.LikeTargetTypePost, targetID)
	if err != nil {
		t.Fatalf("首次查询失败: %v", err)
	}
	if liked {
		t.Fatal("首次查询应返回未点赞")
	}
	if !srv.Exists("cf:like:status:7:1:1001") {
		t.Fatal("首次查询后未回填点赞状态缓存")
	}

	// 第二次：命中缓存，不再查 DB
	liked2, err := svc.IsUserLike(context.Background(), userID, domain.LikeTargetTypePost, targetID)
	if err != nil {
		t.Fatalf("缓存命中查询失败: %v", err)
	}
	if liked2 {
		t.Fatal("缓存命中应返回未点赞")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock 期望未全部满足: %v", err)
	}
}

// TestCountLike_MissThenHit 点赞计数：首次查 DB 回填，二次命中缓存。
func TestCountLike_MissThenHit(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	srv, cacheSvc := newMiniredisCache(t)

	gdb := newGormDB(t, db)
	svc := service.NewLikeService(
		repository.NewLikeRepo(gdb),
		repository.NewPostRepo(gdb),
		repository.NewCommentRepo(gdb),
		cacheSvc,
	)

	const targetID = 1001
	mock.ExpectQuery("count").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	total, err := svc.CountLike(context.Background(), domain.LikeTargetTypePost, targetID)
	if err != nil {
		t.Fatalf("首次查询失败: %v", err)
	}
	if total != 5 {
		t.Fatalf("首次查询应返回 5, 实际: %d", total)
	}
	if !srv.Exists("cf:like:count:1:1001") {
		t.Fatal("首次查询后未回填点赞计数缓存")
	}

	total2, err := svc.CountLike(context.Background(), domain.LikeTargetTypePost, targetID)
	if err != nil {
		t.Fatalf("缓存命中查询失败: %v", err)
	}
	if total2 != 5 {
		t.Fatalf("缓存命中应返回 5, 实际: %d", total2)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock 期望未全部满足: %v", err)
	}
}

// TestGetUserLikedTarget_MissThenHit 我的点赞列表：首次查 DB(COUNT+SELECT)回填，二次命中缓存。
func TestGetUserLikedTarget_MissThenHit(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	srv, cacheSvc := newMiniredisCache(t)

	gdb := newGormDB(t, db)
	svc := service.NewLikeService(
		repository.NewLikeRepo(gdb),
		repository.NewPostRepo(gdb),
		repository.NewCommentRepo(gdb),
		cacheSvc,
	)

	const userID = 7
	// 第一次：COUNT + SELECT likes(空)
	mock.ExpectQuery("count").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT \\* FROM `likes`").WillReturnRows(sqlmock.NewRows([]string{"like_id"}))

	list, total, err := svc.GetUserLikedTarget(context.Background(), userID, domain.LikeTargetTypePost, 1, 20)
	if err != nil {
		t.Fatalf("首次查询失败: %v", err)
	}
	if total != 0 || len(list) != 0 {
		t.Fatalf("首次查询返回错误: list=%v total=%d", list, total)
	}
	if !srv.Exists("cf:like:user:7:1:1:20") {
		t.Fatal("首次查询后未回填我的点赞列表缓存")
	}

	// 第二次：命中缓存，不再查 DB
	list2, total2, err := svc.GetUserLikedTarget(context.Background(), userID, domain.LikeTargetTypePost, 1, 20)
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

// TestDoLike_InvalidatesCaches 帖子点赞后失效 status/count/我的点赞/帖子详情缓存。
func TestDoLike_InvalidatesCaches(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	srv, cacheSvc := newMiniredisCache(t)

	gdb := newGormDB(t, db)
	svc := service.NewLikeService(
		repository.NewLikeRepo(gdb),
		repository.NewPostRepo(gdb),
		repository.NewCommentRepo(gdb),
		cacheSvc,
	)

	const userID = 7
	const postID = 1001
	// 预置缓存
	cacheSvc.SetJSON(context.Background(), "cf:like:status:7:1:1001", false, time.Minute)
	cacheSvc.SetJSON(context.Background(), "cf:like:count:1:1001", 0, time.Minute)
	cacheSvc.SetJSON(context.Background(), "cf:like:user:7:1:1:20", map[string]any{"total": 0}, time.Minute)
	cacheSvc.SetJSON(context.Background(), "cf:post:1001", map[string]any{"post_id": postID}, time.Minute)

	// 点赞流程：查帖子(目标存在性) → Count(IsUserLike=0) → 事务(INSERT likes + UPDATE posts)
	mock.ExpectQuery("SELECT \\* FROM `posts`").WillReturnRows(postRow(postID, userID, "标题"))
	mock.ExpectQuery("count").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `likes`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE `posts`").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := svc.DoLike(context.Background(), userID, domain.LikeTargetTypePost, postID); err != nil {
		t.Fatalf("点赞失败: %v", err)
	}
	if srv.Exists("cf:like:status:7:1:1001") {
		t.Fatal("点赞后 status 缓存未失效")
	}
	if srv.Exists("cf:like:count:1:1001") {
		t.Fatal("点赞后 count 缓存未失效")
	}
	if srv.Exists("cf:like:user:7:1:1:20") {
		t.Fatal("点赞后我的点赞列表缓存未失效")
	}
	if srv.Exists("cf:post:1001") {
		t.Fatal("点赞后帖子详情缓存未失效（LikeCount 已变）")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock 期望未全部满足: %v", err)
	}
}

// TestCancelLike_InvalidatesCaches 帖子取消点赞后失效 status/count/我的点赞/帖子详情缓存。
func TestCancelLike_InvalidatesCaches(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	srv, cacheSvc := newMiniredisCache(t)

	gdb := newGormDB(t, db)
	svc := service.NewLikeService(
		repository.NewLikeRepo(gdb),
		repository.NewPostRepo(gdb),
		repository.NewCommentRepo(gdb),
		cacheSvc,
	)

	const userID = 7
	const postID = 1001
	cacheSvc.SetJSON(context.Background(), "cf:like:status:7:1:1001", true, time.Minute)
	cacheSvc.SetJSON(context.Background(), "cf:like:count:1:1001", 1, time.Minute)
	cacheSvc.SetJSON(context.Background(), "cf:like:user:7:1:1:20", map[string]any{"total": 1}, time.Minute)
	cacheSvc.SetJSON(context.Background(), "cf:post:1001", map[string]any{"post_id": postID}, time.Minute)

	// 取消点赞流程：查帖子 → Count(IsUserLike=1) → 事务(DELETE likes + UPDATE posts)
	mock.ExpectQuery("SELECT \\* FROM `posts`").WillReturnRows(postRow(postID, userID, "标题"))
	mock.ExpectQuery("count").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM `likes`").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE `posts`").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := svc.CancelLike(context.Background(), userID, domain.LikeTargetTypePost, postID); err != nil {
		t.Fatalf("取消点赞失败: %v", err)
	}
	if srv.Exists("cf:like:status:7:1:1001") {
		t.Fatal("取消点赞后 status 缓存未失效")
	}
	if srv.Exists("cf:like:count:1:1001") {
		t.Fatal("取消点赞后 count 缓存未失效")
	}
	if srv.Exists("cf:like:user:7:1:1:20") {
		t.Fatal("取消点赞后我的点赞列表缓存未失效")
	}
	if srv.Exists("cf:post:1001") {
		t.Fatal("取消点赞后帖子详情缓存未失效")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock 期望未全部满足: %v", err)
	}
}

// TestDoCommentLike_InvalidatesCommentPages 评论点赞后连带失效该帖评论分页缓存（LikeCount 已变）。
func TestDoCommentLike_InvalidatesCommentPages(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	srv, cacheSvc := newMiniredisCache(t)

	gdb := newGormDB(t, db)
	svc := service.NewLikeService(
		repository.NewLikeRepo(gdb),
		repository.NewPostRepo(gdb),
		repository.NewCommentRepo(gdb),
		cacheSvc,
	)

	const userID = 7
	const commentID = 500
	const postID = 1001
	// 预置该帖评论分页缓存 + 另一帖子评论分页缓存
	cacheSvc.SetJSON(context.Background(), "cf:comment:1001:1:20", map[string]any{"total": 1}, time.Minute)
	cacheSvc.SetJSON(context.Background(), "cf:comment:1002:1:20", map[string]any{"total": 9}, time.Minute)

	// 评论点赞流程：查评论(含 Preload Author) → Count(IsUserLike=0) → 事务(INSERT likes + UPDATE comments)
	mock.ExpectQuery("SELECT \\* FROM `comments`").WillReturnRows(commentRow(commentID, postID, userID, "内容"))
	mock.ExpectQuery("SELECT \\* FROM `users`").WillReturnRows(userRow(userID, "alice", "a@e.com", "hash"))
	mock.ExpectQuery("count").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `likes`").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE `comments`").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := svc.DoLike(context.Background(), userID, domain.LikeTargetTypeComment, commentID); err != nil {
		t.Fatalf("评论点赞失败: %v", err)
	}
	if srv.Exists("cf:comment:1001:1:20") {
		t.Fatal("评论点赞后该帖评论分页缓存未失效")
	}
	if !srv.Exists("cf:comment:1002:1:20") {
		t.Fatal("不应误删其他帖子的评论缓存")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock 期望未全部满足: %v", err)
	}
}
