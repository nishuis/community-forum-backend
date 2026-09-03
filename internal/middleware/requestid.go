package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

// ctxKey 自定义 context key 类型，避免与其他包/其他 key 冲突
type ctxKey string

// requestIDKey 用于在请求 context 中存取 request_id
const requestIDKey ctxKey = "requestId"

// RequestID 请求ID中间件：
// 从请求头 X-Request-Id 读取，为空则自动生成；
// 写入 gin context 与请求 context（供业务日志取用），并回写响应头，
// 便于压测与排查时按 request_id 串联整条调用链。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-Id")
		if rid == "" {
			rid = newRequestID()
		}

		c.Set("requestId", rid)
		ctx := context.WithValue(c.Request.Context(), requestIDKey, rid)
		c.Request = c.Request.WithContext(ctx)

		c.Writer.Header().Set("X-Request-Id", rid)
		c.Next()
	}
}

// GetRequestID 从请求 context 中取出 request_id（不存在返回空串）
func GetRequestID(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// newRequestID 生成 32 位十六进制随机请求ID（基于 crypto/rand，零依赖）
func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand 在主流平台不会失败，此处兜底避免 panic
		return ""
	}
	return hex.EncodeToString(b)
}
