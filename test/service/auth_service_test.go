// test/service —— service 层黑盒单元测试（package service_test）。
// 覆盖鉴权业务错误路径：登录时密码错误应返回 ErrPasswordWrong，且不签发 token。
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
	"golang.org/x/crypto/bcrypt"
)

// TestLoginWrongPassword 用户存在但密码错误时，Login 应返回业务错误 ErrPasswordWrong，
// 并且不会签发任何 token（未走到 JWT 生成逻辑）。
func TestLoginWrongPassword(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	svc := service.NewAuthService(repository.NewUserRepo(newGormDB(t, db)), &configs.Config{})

	// 预置一条用真实 bcrypt 哈希的“正确密码”用户记录
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-pass"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("生成密码哈希失败: %v", err)
	}

	// 模拟 repository.FindUserByUsername 命中该用户。
	// gorm 的 First 会把 LIMIT 参数化为 ?，因此有两个参数：username 和 1。
	mock.ExpectQuery("SELECT .* FROM .users. WHERE username").
		WithArgs("bob", 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "created_at", "updated_at", "deleted_at",
			"username", "password", "email", "nickname", "avatar", "role",
		}).AddRow(
			int64(1), time.Now(), time.Now(), nil,
			"bob", string(hash), "bob@example.com", "", "", int64(0),
		))

	accessToken, refreshToken, err := svc.Login(context.Background(), "bob", "wrong-pass")
	if err == nil {
		t.Fatal("期望返回错误，实际为 nil")
	}
	if !errors.Is(err, errs.ErrPasswordWrong) {
		t.Fatalf("期望业务错误 errs.ErrPasswordWrong，实际: %v", err)
	}
	if accessToken != "" || refreshToken != "" {
		t.Fatalf("密码错误时不应签发令牌: access=%q refresh=%q", accessToken, refreshToken)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock 期望未全部满足: %v", err)
	}
}
