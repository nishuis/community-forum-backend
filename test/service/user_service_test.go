// test/service —— service 层黑盒单元测试（package service_test）。
// 覆盖用户业务错误路径：注册时用户名唯一索引冲突应翻译为 ErrUsernameExisted。
package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/nishuis/community-forum-backend/configs"
	"github.com/nishuis/community-forum-backend/internal/errs"
	"github.com/nishuis/community-forum-backend/internal/repository"
	"github.com/nishuis/community-forum-backend/internal/service"
)

// TestRegisterDuplicateUsername 注册时用户名触发唯一索引冲突（MySQL 1062），
// service 层应把底层数据库错误翻译成业务错误 ErrUsernameExisted，且不返回用户。
func TestRegisterDuplicateUsername(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	svc := service.NewUserService(repository.NewUserRepo(newGormDB(t, db)), &configs.Config{}, nil)

	// 模拟 repository.CreateUser 插入时撞上 users.uk_username 唯一索引。
	// gorm 默认把单条 Create 包在事务里：BEGIN -> INSERT(报 1062) -> ROLLBACK。
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO .users.").
		WillReturnError(&mysql.MySQLError{
			Number:  1062,
			Message: "Duplicate entry 'alice' for key 'users.uk_username'",
		})
	mock.ExpectRollback()

	user, err := svc.Register(context.Background(), "alice", "secret123", "alice@example.com")
	if err == nil {
		t.Fatalf("期望返回错误，实际为 nil，user=%+v", user)
	}
	if !errors.Is(err, errs.ErrUsernameExisted) {
		t.Fatalf("期望业务错误 errs.ErrUsernameExisted，实际: %v", err)
	}
	if user != nil {
		t.Fatalf("注册失败时不应返回用户对象，实际: %+v", user)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock 期望未全部满足: %v", err)
	}
}
