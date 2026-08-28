package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/nishuis/community-forum-backend/configs"
	"github.com/nishuis/community-forum-backend/internal/controller"
	"github.com/nishuis/community-forum-backend/internal/infra/db"
	"github.com/nishuis/community-forum-backend/internal/repository"
	"github.com/nishuis/community-forum-backend/internal/service"
)

func main() {
	//1.加载配置
	cfg, err := configs.LoadConfig("configs/config.yaml")
	if err != nil {
		panic(err)
	}
	// 校验JWT配置
	jwtCfg := cfg.Jwt
	if jwtCfg.Secret == "" {
		panic("jwt secret 不能为空，请检查config.yaml")
	}
	if jwtCfg.AccessExpHour <= 0 {
		panic("Jwt.AccessExpHour 必须大于0")
	}
	if jwtCfg.RefreshExpDay <= 0 {
		panic("Jwt.RefreshExpDay 必须大于0")
	}

	//2.初始化，拿到gorm数据库连接实例db
	db, err := db.InitDB(cfg)
	if err != nil {
		panic(err)
	}
	fmt.Println("初始化完成")

	//3.组装依赖链
	//repo
	userRepo := repository.NewUserRepo(db)
	//service
	userService := service.NewUserService(userRepo, cfg)
	authService := service.NewAuthService(userRepo, cfg)
	//controller
	userCtrl := controller.NewUserController(userService)
	authCtrl := controller.NewAuthController(authService)

	//4.初始化Gin，组装依赖链
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.String(200, "hello ok")
	})

	//5.路由分组
	apiGroup := r.Group("/api")
	{
		authGroup := apiGroup.Group("/auth")
		{
			//注册接口 POST /api/auth/register
			authGroup.POST("/register", userCtrl.Register)

			//登录接口 POST /api/auth/login
			authGroup.POST("/login", authCtrl.Login)
		}
	}

	//6.启动web，监听本地8080端口
	err = r.Run(fmt.Sprint(":", cfg.Server.Port))
	if err != nil {
		panic(err)
	}
}
