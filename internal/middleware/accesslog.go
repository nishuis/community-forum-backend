package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// AccessLog 访问日志中间件：
// 记录每个请求的方法、路径、状态码、耗时、request_id、user_id，输出结构化 JSON。
// 必须在 RequestID 之后挂载，才能取到 request_id。
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		slog.Info("access",
			"request_id", c.GetString("requestId"),
			"user_id", c.GetInt64("userId"),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
		)
	}
}
