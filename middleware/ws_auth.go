package middleware

import (
	"blog/global"
	"blog/service/redis_ser"
	"blog/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// WSAuth 为WebSocket连接提供的JWT认证中间件
func WSAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 添加详细日志
		global.Log.Info("开始WebSocket认证",
			zap.String("remote_addr", c.Request.RemoteAddr),
			zap.String("path", c.Request.URL.Path))

		// 从URL查询参数获取token
		tokenString := c.Query("token")
		if tokenString == "" {
			// 从cookie中尝试获取
			tokenString, _ = c.Cookie("token")
		}

		// 检查Token是否存在
		if tokenString == "" {
			global.Log.Warn("WebSocket认证失败: 缺少token",
				zap.String("remote_addr", c.Request.RemoteAddr))

			// WebSocket特殊处理：使用HTTP状态码来拒绝连接
			c.Header("Sec-WebSocket-Version", "13")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 检查令牌是否在黑名单中
		isBlacklisted, err := redis_ser.IsTokenBlacklisted(tokenString)
		if err != nil {
			global.Log.Error("检查令牌黑名单失败", zap.Error(err))
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if isBlacklisted {
			global.Log.Warn("WebSocket认证失败: token已失效",
				zap.String("remote_addr", c.Request.RemoteAddr))
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 解析Token
		claims, err := utils.ParseToken(tokenString)
		if err != nil {
			if err.Error() == "token已过期" {
				// 处理过期token
				expiredClaims, parseErr := utils.ParseExpiredToken(tokenString)
				if parseErr != nil {
					global.Log.Error("解析过期token失败", zap.Error(parseErr))
					c.AbortWithStatus(http.StatusUnauthorized)
					return
				}

				// 刷新token
				newAccessToken, refreshErr := utils.RefreshAccessToken(tokenString, expiredClaims.UserID)
				if refreshErr != nil || newAccessToken == "" {
					global.Log.Error("刷新token失败", zap.Error(refreshErr))
					c.AbortWithStatus(http.StatusUnauthorized)
					return
				}

				// 将新token设置到上下文
				c.Set("new_token", newAccessToken)
				c.Set("claims", expiredClaims)

				global.Log.Info("WebSocket认证成功(已刷新token)",
					zap.Uint64("user_id", uint64(expiredClaims.UserID)))
				c.Next()
				return
			}
			global.Log.Error("无效token", zap.Error(err))
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 将用户信息保存到上下文
		c.Set("claims", claims)
		global.Log.Info("WebSocket认证成功", zap.Uint64("user_id", uint64(claims.UserID)))
		c.Next()
	}
}
