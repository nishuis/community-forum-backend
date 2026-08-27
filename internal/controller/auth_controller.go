package controller

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nishuis/community-forum-backend/internal/dto/request"
	"github.com/nishuis/community-forum-backend/internal/dto/response"
	"github.com/nishuis/community-forum-backend/internal/errs"
	"github.com/nishuis/community-forum-backend/internal/service"
)

type AuthController struct {
	authService *service.AuthService
}

// NewAuthController 新建鉴权控制器
func NewAuthController(as *service.AuthService) *AuthController {
	return &AuthController{
		authService: as,
	}
}

// Loginer 用户登录接口
func (ac *AuthController) Login(ctx *gin.Context) {
	//1.接收前端JSON
	var req request.LoginRequest
	err := ctx.ShouldBindJSON(&req)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": errs.CodeParamError,
			"msg":  "请求参数错误",
		})
		return
	}

	//2.调用service层执行登录业务
	accessToken, refreshToken, err := ac.authService.Login(req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrPasswordWrong) || errors.Is(err, errs.ErrUserNotFound):
			ctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeAuthFail,
				"msg":  "用户名或密码错误",
			})
		default:
			log.Printf("登录业务异常 err: %v", err)
			ctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeServerInternal,
				"msg":  "服务器内部错误",
			})
		}
		return
	}

	//3.响应前端
	resp := response.LoginResp{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code": errs.CodeOk,
		"msg":  "登录成功",
		"data": resp,
	})
}
