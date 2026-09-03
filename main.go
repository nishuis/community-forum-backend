package main

import (
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/nishuis/community-forum-backend/configs"
	"github.com/nishuis/community-forum-backend/internal/controller"
	"github.com/nishuis/community-forum-backend/internal/infra/db"
	infralog "github.com/nishuis/community-forum-backend/internal/infra/log"
	"github.com/nishuis/community-forum-backend/internal/middleware"
	"github.com/nishuis/community-forum-backend/internal/repository"
	"github.com/nishuis/community-forum-backend/internal/service"
)

func main() {
	//1.加载配置
	cfg, err := configs.LoadConfig("configs/config.yaml")
	if err != nil {
		panic(err)
	}

	//1.1 初始化全局日志（slog，JSON结构化输出）
	if err := infralog.InitLogger(cfg); err != nil {
		panic(err)
	}
	slog.Info("配置加载完成")

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
	slog.Info("数据库初始化完成")

	//3.组装依赖链
	//repo
	userRepo := repository.NewUserRepo(db)
	postRepo := repository.NewPostRepo(db)
	commentRepo := repository.NewCommentRepo(db)
	likeRepo := repository.NewLikeRepo(db)
	//service
	userService := service.NewUserService(userRepo, cfg)
	authService := service.NewAuthService(userRepo, cfg)
	postService := service.NewPostService(postRepo, userRepo, cfg)
	commentService := service.NewCommentService(commentRepo, postRepo)
	likeService := service.NewLikeService(likeRepo, postRepo, commentRepo)
	//controller
	userCtrl := controller.NewUserController(userService)
	authCtrl := controller.NewAuthController(authService)
	postCtrl := controller.NewPostController(postService)
	commentCtrl := controller.NewCommentController(commentService)
	likeCtrl := controller.NewLikeController(likeService, cfg)

	//4.初始化Gin
	r := gin.Default()
	//4.1 全局请求ID中间件：为每个请求生成/透传 request_id（第3步将调整到访问日志之前）
	r.Use(middleware.RequestID())

	//5.路由分组
	apiGroup := r.Group("/api/v1")
	{
		//公开接口组(含可选鉴权)
		publicGroup := apiGroup.Group("")
		{
			//用户注册、登录
			publicGroup.POST("/users/register", userCtrl.Register)
			publicGroup.POST("/auth/login", authCtrl.Login)

			//帖子公开查询
			publicGroup.GET("/posts/:post_id", postCtrl.GetPostByPostId)
			publicGroup.GET("/users/:author_id/posts", postCtrl.GetAuthorPostList)
			publicGroup.GET("/posts/search", postCtrl.GetPostByKeyWordOffset)

			//获取点赞状态
			publicGroup.GET("like/status", likeCtrl.GetLikeStatus)

			//获取评论列表
			publicGroup.GET("/post/:post_id/comments", commentCtrl.GetCommentList)
		}

		//需鉴权接口组
		authGroup := apiGroup.Group("")
		{
			//挂载中间件
			authGroup.Use(middleware.JWTAuth(cfg))

			//获取用户信息,更新用户信息，删除用户
			authGroup.GET("/users/me", userCtrl.GetCurrentUser)
			authGroup.PUT("/users/me", userCtrl.UpdateUserInfo)
			authGroup.DELETE("/user/me", userCtrl.DeleteUser)

			//发帖,删帖，更新帖子
			authGroup.POST("/posts", postCtrl.CreatePost)
			authGroup.DELETE("/posts/:post_id", postCtrl.DeletePost)
			authGroup.PUT("/posts/:post_id", postCtrl.UpdatePost)

			//发评论，删评论，编辑评论
			authGroup.POST("/comment/create", commentCtrl.CreateComment)
			authGroup.DELETE("/comment/:comment_id", commentCtrl.DeleteComment)
			authGroup.PUT("/comment/update", commentCtrl.UpdateComment)

			//点赞，取消点赞，获取点赞记录
			authGroup.POST("/like/do", likeCtrl.DoLike)
			authGroup.POST("/like/cancel", likeCtrl.CancelLike)
			authGroup.GET("/like/me", likeCtrl.GetMyLiked)

		}
	}
	//6.启动web，监听本地8080端口
	err = r.Run(fmt.Sprint(":", cfg.Server.Port))
	if err != nil {
		panic(err)
	}
}
