package ginutil

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nishuis/community-forum-backend/configs"
	"github.com/nishuis/community-forum-backend/internal/errs"
	jwtutil "github.com/nishuis/community-forum-backend/pkg/jwt_util"
)

// LogError 输出带请求上下文（request_id / user_id）的结构化错误日志。
// attrs 传键值对，例如：LogError(c, "发帖失败", "err", err)
func LogError(c *gin.Context, msg string, attrs ...any) {
	fields := []any{
		"request_id", c.GetString("requestId"),
		"user_id", c.GetInt64("userId"),
	}
	fields = append(fields, attrs...)
	slog.ErrorContext(c.Request.Context(), msg, fields...)
}

// GetCurrentUserInfo 获取当前登录用户 userId username
// return：ok = true 获取成功；ok=false 内部已经 c.JSON + c.Abort()，controller直接return即可
func GetCurrentUserInfo(c *gin.Context) (userId int64, username string, ok bool) {
	val, exist := c.Get("userId")
	if !exist {
		LogError(c, "jwt中间件异常放行，上下文不存在userId")
		c.JSON(200, gin.H{
			"code": errs.CodeServerInternal,
			"msg":  "服务器内部错误",
		})
		c.Abort() // 标记：不要再跑后面的handler链,做一层兜底防护
		return 0, "", false
	}
	userId, typeOk := val.(int64)
	if !typeOk {
		LogError(c, "jwt中间件异常放行，userId类型断言失败")
		c.JSON(200, gin.H{
			"code": errs.CodeServerInternal,
			"msg":  "服务器内部错误",
		})
		c.Abort()
		return 0, "", false
	}

	uName, exist := c.Get("username")
	if !exist {
		LogError(c, "jwt中间件异常放行，上下文不存在username")
		c.JSON(200, gin.H{
			"code": errs.CodeServerInternal,
			"msg":  "服务器内部错误",
		})
		c.Abort()
		return 0, "", false
	}
	username, typeOk = uName.(string)
	if !typeOk {
		LogError(c, "jwt中间件异常放行，username类型断言失败")
		c.JSON(200, gin.H{
			"code": errs.CodeServerInternal,
			"msg":  "服务器内部错误",
		})
		c.Abort()
		return 0, "", false
	}

	return userId, username, true
}

// HandleContextError 统一处理context取消、超时错误,
// logMsg：传入业务描述字符串，例如 "删帖请求"、"登录请求"、"发表帖子请求"，用来区分，打印日志
// 返回值 true: 已经识别是context错误，内部完成JSON响应；外部直接return
// 返回值 false: 不是context错误，controller继续处理业务错误
func HandleContextError(c *gin.Context, err error, logMsg string) bool {
	if errors.Is(err, context.Canceled) {
		LogError(c, logMsg+",客户端主动取消", "err", err)
		c.JSON(http.StatusOK, gin.H{
			"code": errs.CodeContextCancel,
			"msg":  "服务取消",
		})
		return true
	}
	if errors.Is(err, context.DeadlineExceeded) {
		LogError(c, logMsg+",请求超时", "err", err)
		c.JSON(http.StatusOK, gin.H{
			"code": errs.CodeDeadLineExceeded,
			"msg":  "超时未响应",
		})
		return true
	}
	return false
}

// TryGetUserIdFromHeader 可选鉴权：尝试从header解析userId
// 返回 userId=0 代表未登录 / token无效；不会c.Abort，不会中断请求
func TryGetUserIdFromHeader(c *gin.Context, cfg *configs.Config) int64 {
	//1.尝试获取请求头
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return 0
	}

	//2.尝试获取token
	parts := strings.SplitN(authHeader, " ", 2)
	if !(len(parts) == 2 && strings.EqualFold(parts[0], "Bearer")) {
		return 0
	}
	tokenStr := parts[1]

	//3.解析token
	claims, err := jwtutil.ParseToken(tokenStr, cfg.Jwt.Secret)
	if err != nil {
		LogError(c, "TryGetUserIdFromHeader parse token fail", "err", err)
		return 0
	}
	return claims.UserId
}
