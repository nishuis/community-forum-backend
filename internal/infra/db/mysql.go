package db

import (
	"github.com/nishuis/community-forum-backend/configs"
	"github.com/nishuis/community-forum-backend/internal/domain"
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
		Logger:         logger.Default.LogMode(logger.Warn),
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
