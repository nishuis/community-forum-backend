// jwt 鉴权中间件
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nishuis/community-forum-backend/configs"
	"github.com/nishuis/community-forum-backend/internal/errs"
	ginutil "github.com/nishuis/community-forum-backend/pkg/gin_util"
	jwtutil "github.com/nishuis/community-forum-backend/pkg/jwt_util"
)

// JWTAuth jwt鉴权中间件
func JWTAuth(cfg *configs.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		//真正中间件逻辑

		//1.获取请求头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusOK, gin.H{
				"code": errs.CodeUnauthorized,
				"msg":  "未认证的操作，请登录",
			})
			//终止处理链，需要手动return
			c.Abort()
			return
		}

		//2.校验Bearer格式：bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		//兼容大小写“Bearer”
		if !(len(parts) == 2 && strings.EqualFold(parts[0], "Bearer")) {
			c.JSON(http.StatusOK, gin.H{
				"code": errs.CodeParamError,
				"msg":  "格式错误，认证失败",
			})
			c.Abort()
			return
		}

		//3.解析Token
		tokenStr := parts[1]
		claims, err := jwtutil.ParseToken(tokenStr, cfg.Jwt.Secret)
		if err != nil {
			ginutil.LogError(c, "JWT解析失败", "err", err)
			c.JSON(http.StatusOK, gin.H{
				"code": errs.CodeAuthFail,
				"msg":  "校验失败",
			})
			c.Abort()
			return
		}

		//4.解析成功，获取用户信息
		c.Set("userId", claims.UserId)
		c.Set("username", claims.Username)

		//5.放行
		c.Next()
	}

}
