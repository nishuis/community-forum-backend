package ginutil

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nishuis/community-forum-backend/internal/errs"
)

// GetCurrentUserInfo 获取当前登录用户 userId username
// return：ok = true 获取成功；ok=false 内部已经 c.JSON + c.Abort()，controller直接return即可
func GetCurrentUserInfo(c *gin.Context) (userId int64, username string, ok bool) {
	val, exist := c.Get("userId")
	if !exist {
		log.Printf("jwt中间件异常放行，上下文不存在userId")
		c.JSON(200, gin.H{
			"code": errs.CodeServerInternal,
			"msg":  "服务器内部错误",
		})
		c.Abort() // 标记：不要再跑后面的handler链,做一层兜底防护
		return 0, "", false
	}
	userId, typeOk := val.(int64)
	if !typeOk {
		log.Printf("jwt中间件异常放行，userId类型断言失败")
		c.JSON(200, gin.H{
			"code": errs.CodeServerInternal,
			"msg":  "服务器内部错误",
		})
		c.Abort()
		return 0, "", false
	}

	uName, exist := c.Get("username")
	if !exist {
		log.Printf("jwt中间件异常放行，上下文不存在username")
		c.JSON(200, gin.H{
			"code": errs.CodeServerInternal,
			"msg":  "服务器内部错误",
		})
		c.Abort()
		return 0, "", false
	}
	username, typeOk = uName.(string)
	if !typeOk {
		log.Printf("jwt中间件异常放行，username类型断言失败")
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
		log.Printf("%s,客户端主动取消: %v", logMsg, err)
		c.JSON(http.StatusOK, gin.H{
			"code": errs.CodeContextCancel,
			"msg":  "服务取消",
		})
		return true
	}
	if errors.Is(err, context.DeadlineExceeded) {
		log.Printf("%s,请求超时: %v", logMsg, err)
		c.JSON(http.StatusOK, gin.H{
			"code": errs.CodeDeadLineExceeded,
			"msg":  "超时未响应",
		})
		return true
	}
	return false
}
