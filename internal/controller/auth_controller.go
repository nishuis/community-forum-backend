package controller

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nishuis/community-forum-backend/internal/dto/request"
	"github.com/nishuis/community-forum-backend/internal/dto/response"
	"github.com/nishuis/community-forum-backend/internal/errs"
	"github.com/nishuis/community-forum-backend/internal/service"
)

// AuthController 只管凭证、身份认证，不操作用户业务字段
type AuthController struct {
	authService *service.AuthService
}

// NewAuthController 新建鉴权控制器
func NewAuthController(as *service.AuthService) *AuthController {
	return &AuthController{
		authService: as,
	}
}

// Login 用户登录接口
func (ac *AuthController) Login(ginctx *gin.Context) {
	//1.接收前端JSON
	var req request.LoginReq
	err := ginctx.ShouldBindJSON(&req)
	if err != nil {
		ginctx.JSON(http.StatusOK, gin.H{
			"code": errs.CodeParamError,
			"msg":  "请求参数错误",
		})
		return
	}

	//2.调用service层执行登录业务
	//获取context.Context
	ctx := ginctx.Request.Context()
	accessToken, refreshToken, err := ac.authService.Login(ctx, req.Username, req.Password)
	if err != nil {
		//处理context错误
		if errors.Is(err, context.Canceled) {
			log.Printf("登录请求客户端主动取消: %v", err)
			return
		}
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("请求超时：%v", err)
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeDeadLineExceeded,
				"msg":  "超时未响应",
			})
			return
		}

		//处理业务错误
		switch {
		case errors.Is(err, errs.ErrPasswordWrong) || errors.Is(err, errs.ErrUserNotFound):
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeAuthFail,
				"msg":  "用户名或密码错误",
			})
		default:
			log.Printf("登录业务异常 err: %v", err)
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeServerInternal,
				"msg":  "服务器内部错误",
			})
		}
		return
	}

	//3.响应前端
	resp := response.LoginResp{
		Username:     req.Username,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	ginctx.JSON(http.StatusOK, gin.H{
		"code": errs.CodeOK,
		"msg":  "登录成功",
		"data": resp,
	})
}
