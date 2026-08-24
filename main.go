package main

import (
	"fmt"

	config "github.com/nishuis/community-forum-backend/configs"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	cfg := config.LoadConfig()

	listenAddr := fmt.Sprintf(":%d", cfg.Server.Port)
	fmt.Printf("准备监听端口：%s\n", listenAddr)

	db, err := gorm.Open(mysql.Open(cfg.Mysql.Dsn()),
		&gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("mysql连接失败：%v", err))
	}
	fmt.Println("MySQL连接成功")

	var cnt int64
	err = db.Table("users").Count(&cnt).Error
	if err != nil {
		panic(err)
	}
	fmt.Printf("users表行数：%d\n", cnt)
}
