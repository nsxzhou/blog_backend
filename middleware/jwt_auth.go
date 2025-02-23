package middleware

import (
	"blog/global"
	"blog/models/ctypes"
	"blog/models/res"
	"blog/service/redis_ser"
	"blog/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// JwtAuth 中间件，负责验证 Token 并将用户信息存储到上下文
func JwtAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.Request.Header.Get("Authorization")
		// 检查 Token 是否存在并去除 "Bearer " 前缀
		if len(tokenString) < 7 || tokenString[:7] != "Bearer " {
			res.HttpError(c, http.StatusUnauthorized, res.TokenMissing, "缺少token")
			c.Abort()
			return
		}
		tokenString = tokenString[7:]

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

		// 解析 Token
		claims, err := utils.ParseToken(tokenString)
		if err != nil {
			if err.Error() == "token已过期" {
				// 尝试从过期的token中解析出用户ID
				expiredClaims, parseErr := utils.ParseExpiredToken(tokenString)
				if parseErr != nil {
					global.Log.Error("utils.ParseExpiredToken() failed", zap.String("error", parseErr.Error()))
					res.HttpError(c, http.StatusUnauthorized, res.TokenRefreshFailed, "token已过期且无法刷新")
					c.Abort()
					return
				}

				// 使用解析出的用户ID尝试刷新token
				newAccessToken, refreshErr := utils.RefreshAccessToken(tokenString, expiredClaims.UserID)
				if refreshErr != nil || newAccessToken == "" {
					global.Log.Error("utils.RefreshAccessToken() failed", zap.String("error", refreshErr.Error()))
					res.HttpError(c, http.StatusUnauthorized, res.TokenRefreshFailed, "token刷新失败")
					c.Abort()
					return
				}

				// 刷新成功，将新的 Token 设置到响应头中
				c.Request.Header.Set("Authorization", "Bearer "+newAccessToken)
				c.Set("claims", expiredClaims)
				c.Next()
				return
			}
			res.HttpError(c, http.StatusUnauthorized, res.TokenInvalid, "token无效")
			c.Abort()
			return
		}

		// 将用户信息保存到上下文中，方便后续使用
		c.Set("claims", claims)

		c.Next()
	}
}

// JwtAdmin 中间件，基于 JwtAuth 并检查用户角色
func JwtAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 先执行 JwtAuth 中间件的验证
		JwtAuth()(c)
		if c.IsAborted() {
			return // 如果 JwtAuth 验证失败，直接中止
		}

		// 验证用户角色是否为管理员
		_claims, _ := c.Get("claims")
		claims := _claims.(*utils.CustomClaims)
		if claims.Role != ctypes.RoleAdmin {
			res.HttpError(c, http.StatusForbidden, res.PermissionDenied, "权限不足")
			c.Abort()
			return
		}

		c.Next()
	}
}
