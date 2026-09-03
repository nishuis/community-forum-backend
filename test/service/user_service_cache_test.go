// test/service —— UserService 缓存行为黑盒单测。
// 组合 sqlmock + miniredis：验证用户信息 Cache-Aside 命中/未命中/空值占位、
// 缓存中不含 Password，以及更新/删除后的写后失效。
package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nishuis/community-forum-backend/configs"
	"github.com/nishuis/community-forum-backend/internal/errs"
	"github.com/nishuis/community-forum-backend/internal/repository"
	"github.com/nishuis/community-forum-backend/internal/service"
)

// userRow 构造 users 表查询结果行。
func userRow(userID int64, username, email, password string) *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows([]string{
		"user_id", "created_at", "updated_at", "deleted_at",
		"username", "password", "email", "nickname", "avatar", "role",
	}).AddRow(userID, now, now, nil, username, password, email, "", "", 0)
}

// TestGetCurrentUser_MissThenHit_NoPasswordInCache 命中/未命中 + 缓存值不含 Password。
func TestGetCurrentUser_MissThenHit_NoPasswordInCache(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	srv, cacheSvc := newMiniredisCache(t)

	svc := service.NewUserService(repository.NewUserRepo(newGormDB(t, db)), &configs.Config{}, cacheSvc)

	const userID = 7
	// 第一次：未命中 → 查 DB → 回填
	mock.ExpectQuery("SELECT \\* FROM `users`").WillReturnRows(userRow(userID, "alice", "alice@example.com", "bcrypt-hash-xxx"))
	got, err := svc.GetCurrentUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("首次查询失败: %v", err)
	}
	if got == nil || got.Username != "alice" {
		t.Fatalf("首次查询返回错误: %+v", got)
	}
	if got.Password == "" {
		t.Fatal("首次从 DB 返回的用户应含 Password（用于后续对比缓存不含密码）")
	}

	// 缓存值不应包含 Password 字段
	raw, err := srv.Get("cf:user:7")
	if err != nil {
		t.Fatalf("读取缓存失败: %v", err)
	}
	if strings.Contains(strings.ToLower(raw), "password") {
		t.Fatalf("缓存中不应包含 Password 字段: %s", raw)
	}
	var cached map[string]any
	if err := json.Unmarshal([]byte(raw), &cached); err != nil {
		t.Fatalf("缓存不是合法 JSON: %v", err)
	}
	if cached["username"] != "alice" {
		t.Fatalf("缓存应包含用户名，实际: %v", cached)
	}

	// 第二次：命中缓存（Password 为空），不再查 DB
	got2, err := svc.GetCurrentUser(context.Background(), userID)
	if err != nil {
		t.Fatalf("缓存命中查询失败: %v", err)
	}
	if got2 == nil || got2.Username != "alice" {
		t.Fatalf("缓存命中返回错误: %+v", got2)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock 期望未全部满足: %v", err)
	}
}

// TestGetCurrentUser_RecordNotFound_EmptyMarker 不存在的用户写空值占位，二次直接返回不存在。
func TestGetCurrentUser_RecordNotFound_EmptyMarker(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	srv, cacheSvc := newMiniredisCache(t)

	svc := service.NewUserService(repository.NewUserRepo(newGormDB(t, db)), &configs.Config{}, cacheSvc)

	const userID = 9999
	mock.ExpectQuery("SELECT \\* FROM `users`").WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
	_, err := svc.GetCurrentUser(context.Background(), userID)
	if !errors.Is(err, errs.ErrUserNotFound) {
		t.Fatalf("期望 ErrUserNotFound，实际: %v", err)
	}
	if !srv.Exists("cf:user:9999") {
		t.Fatal("未写入空值占位")
	}

	_, err = svc.GetCurrentUser(context.Background(), userID)
	if !errors.Is(err, errs.ErrUserNotFound) {
		t.Fatalf("空值占位命中后应返回 ErrUserNotFound，实际: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock 期望未全部满足: %v", err)
	}
}

// TestDeleteUser_InvalidatesCache 删除成功后删除用户信息缓存。
func TestDeleteUser_InvalidatesCache(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	srv, cacheSvc := newMiniredisCache(t)

	svc := service.NewUserService(repository.NewUserRepo(newGormDB(t, db)), &configs.Config{}, cacheSvc)

	const userID = 7
	cacheSvc.SetJSON(context.Background(), "cf:user:7", map[string]any{"user_id": userID}, time.Minute)

	mock.ExpectQuery("SELECT \\* FROM `users`").WillReturnRows(userRow(userID, "alice", "a@e.com", "hash"))
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `users`").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := svc.DeleteUser(context.Background(), userID); err != nil {
		t.Fatalf("删除用户失败: %v", err)
	}
	if srv.Exists("cf:user:7") {
		t.Fatal("删除成功后缓存未失效")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock 期望未全部满足: %v", err)
	}
}

// TestUpdateUserInfo_InvalidatesCache 更新资料成功后删除用户信息缓存。
func TestUpdateUserInfo_InvalidatesCache(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	srv, cacheSvc := newMiniredisCache(t)

	svc := service.NewUserService(repository.NewUserRepo(newGormDB(t, db)), &configs.Config{}, cacheSvc)

	const userID = 7
	cacheSvc.SetJSON(context.Background(), "cf:user:7", map[string]any{"user_id": userID}, time.Minute)

	// 流程：查原用户 → 查新用户名是否被占用(无记录) → 事务内 UPDATE → 重新查
	mock.ExpectQuery("SELECT \\* FROM `users`").WillReturnRows(userRow(userID, "oldname", "old@e.com", "hash"))
	mock.ExpectQuery("SELECT \\* FROM `users`").WillReturnRows(sqlmock.NewRows([]string{"user_id"})) // 用户名未被占用
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `users`").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery("SELECT \\* FROM `users`").WillReturnRows(userRow(userID, "newname", "old@e.com", "hash"))

	updated, err := svc.UpdateUserInfo(context.Background(), userID, "newname", "")
	if err != nil {
		t.Fatalf("更新用户失败: %v", err)
	}
	if updated.Username != "newname" {
		t.Fatalf("返回用户错误: %+v", updated)
	}
	if srv.Exists("cf:user:7") {
		t.Fatal("更新成功后缓存未失效")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock 期望未全部满足: %v", err)
	}
}
