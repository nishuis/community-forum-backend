// test/service —— service 层黑盒单元测试（package service_test）。
// 通过被测包的公开 API 验证业务规则，不依赖真实 MySQL，用 sqlmock 模拟数据库。
package service_test

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// newMockDB 创建基于 sqlmock 的内存数据库连接，不依赖真实 MySQL。
func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() 失败: %v", err)
	}
	return db, mock
}

// newGormDB 把 sqlmock 的 *sql.DB 包装成 *gorm.DB，供 repository 注入使用。
func newGormDB(t *testing.T, db *sql.DB) *gorm.DB {
	t.Helper()
	gdb, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() 失败: %v", err)
	}
	return gdb
}
