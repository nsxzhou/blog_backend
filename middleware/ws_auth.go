package middleware

import (
	"blog/global"
	"blog/models/res"
	"blog/service/redis_ser"
	"blog/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// WSAuth 为WebSocket连接提供的JWT认证中间件
func WSAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从URL查询参数获取token
		tokenString := c.Query("token")
		if tokenString == "" {
			// 从cookie中尝试获取
			tokenString, _ = c.Cookie("token")
		}

		// 检查Token是否存在
		if tokenString == "" {
			res.HttpError(c, http.StatusUnauthorized, res.TokenMissing, "缺少token")
			c.Abort()
			return
		}

		// 检查令牌是否在黑名单中
		isBlacklisted, err := redis_ser.IsTokenBlacklisted(tokenString)
		if err != nil {
			global.Log.Error("检查令牌黑名单失败", zap.Error(err))
			res.HttpError(c, http.StatusInternalServerError, res.ServerError, "服务器错误")
			c.Abort()
			return
		}
		if isBlacklisted {
			res.HttpError(c, http.StatusUnauthorized, res.TokenInvalid, "token已失效")
			c.Abort()
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
					res.HttpError(c, http.StatusUnauthorized, res.TokenRefreshFailed, "token已过期且无法刷新")
					c.Abort()
					return
				}

				// 刷新token
				newAccessToken, refreshErr := utils.RefreshAccessToken(tokenString, expiredClaims.UserID)
				if refreshErr != nil || newAccessToken == "" {
					global.Log.Error("刷新token失败", zap.Error(refreshErr))
					res.HttpError(c, http.StatusUnauthorized, res.TokenRefreshFailed, "token刷新失败")
					c.Abort()
					return
				}

				// 将新token设置到上下文
				c.Set("new_token", newAccessToken)
				c.Set("claims", expiredClaims)
				c.Next()
				return
			}
			global.Log.Error("无效token", zap.Error(err))
			res.HttpError(c, http.StatusUnauthorized, res.TokenInvalid, "token无效")
			c.Abort()
			return
		}

		// 将用户信息保存到上下文
		c.Set("claims", claims)
		c.Next()
	}
}
