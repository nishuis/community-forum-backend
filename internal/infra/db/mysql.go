package db

import (
	"github.com/nishuis/community-forum-backend/configs"
	"github.com/nishuis/community-forum-backend/internal/domain"
	infralog "github.com/nishuis/community-forum-backend/internal/infra/log"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDB 初始化gorm连接
func InitDB(cfg *configs.Config) (*gorm.DB, error) {
	dsn := cfg.BuildMysqlDSN()

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		//驱动错误转gorm哨兵错误
		TranslateError: true,
		//自定义slog日志适配器：慢查询阈值100ms，出错/慢查询才记录
		Logger: infralog.NewGormLogger(logger.Warn, infralog.SlowThreshold),
	})
	if err != nil {
		return nil, err
	}

	//自动迁移建表
	err = db.AutoMigrate(&domain.User{},
		&domain.Post{}, &domain.Comment{}, &domain.Like{})
	if err != nil {
		return nil, err
	}

	return db, nil

}
