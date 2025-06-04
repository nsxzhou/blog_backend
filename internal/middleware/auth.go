package middleware

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nsxzhou1114/blog-api/internal/config"
	"github.com/nsxzhou1114/blog-api/internal/logger"
	"github.com/nsxzhou1114/blog-api/pkg/auth"
	"github.com/nsxzhou1114/blog-api/pkg/response"
)

// JWTAuth JWT认证中间件
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "请先登录", nil)
			c.Abort()
			return
		}

		// 检查格式
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			response.Unauthorized(c, "Authorization格式错误", nil)
			c.Abort()
			return
		}

		// 验证token
		claims, err := auth.ParseToken(parts[1])
		if err != nil {
			logger.Warnf("无效的令牌: %v", err)
			response.Unauthorized(c, "无效的令牌", err)
			c.Abort()
			return
		}

		// 验证token类型
		if claims.Type != auth.AccessToken {
			logger.Warnf("使用了错误类型的令牌: %v", claims.Type)
			response.Unauthorized(c, "使用了错误类型的令牌", errors.New("需要访问令牌"))
			c.Abort()
			return
		}

		// 检查令牌是否临近过期（可选）
		// 如果令牌将在缓冲时间内过期，设置一个标志，让API响应包含一个头部指示客户端应该刷新令牌
		bufferTime := time.Duration(config.GlobalConfig.JWT.BufferSeconds) * time.Second
		if time.Until(time.Unix(claims.ExpiresAt, 0)) < bufferTime {
			c.Header("X-Token-Expire-Soon", "true")
		}
		// 将用户ID存入上下文
		c.Set("userID", claims.UserID)
		c.Set("userRole", claims.Role)
		c.Set("tokenID", claims.TokenID)
		c.Next()
	}
}

// RefreshAuth 用于刷新访问令牌的中间件
func RefreshAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "请提供刷新令牌", nil)
			c.Abort()
			return
		}

		// 检查格式
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			response.Unauthorized(c, "Authorization格式错误", nil)
			c.Abort()
			return
		}

		// 验证token
		claims, err := auth.ParseToken(parts[1])
		if err != nil {
			logger.Warnf("无效的刷新令牌: %v", err)
			response.Unauthorized(c, "无效的刷新令牌", err)
			c.Abort()
			return
		}

		// 验证token类型
		if claims.Type != auth.RefreshToken {
			logger.Warnf("使用了错误类型的令牌: %v", claims.Type)
			response.Unauthorized(c, "使用了错误类型的令牌", errors.New("需要刷新令牌"))
			c.Abort()
			return
		}

		// 检查刷新令牌是否临近过期
		if time.Until(time.Unix(claims.ExpiresAt, 0)) < 24*time.Hour {
			// 设置响应头，通知客户端刷新令牌即将过期
			c.Header("X-Refresh-Token-Expire-Soon", "true")
		}

		// 将用户ID和令牌信息存入上下文
		c.Set("userID", claims.UserID)
		c.Set("userRole", claims.Role)
		c.Set("tokenID", claims.TokenID)
		c.Set("previousToken", claims.Previous)
		c.Next()
	}
}

// AdminAuth 管理员认证中间件
func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 先执行JWT认证
		JWTAuth()(c)
		if c.IsAborted() {
			return
		}

		// 获取用户角色
		role, exists := c.Get("userRole")
		if !exists {
			response.Unauthorized(c, "未授权", nil)
			c.Abort()
			return
		}

		// 检查是否为管理员
		if role != "admin" {
			response.Forbidden(c, "需要管理员权限", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetUserID 从上下文中获取用户ID
func GetUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		return 0, false
	}
	return userID.(uint), true
}

// GetUserRole 从上下文中获取用户角色
func GetUserRole(c *gin.Context) (string, bool) {
	userRole, exists := c.Get("userRole")
	if !exists {
		return "", false
	}
	return userRole.(string), true
}

// GetTokenID 从上下文中获取令牌ID
func GetTokenID(c *gin.Context) (string, bool) {
	tokenID, exists := c.Get("tokenID")
	if !exists {
		return "", false
	}
	return tokenID.(string), true
}

// GetTokenVersion 从上下文中获取令牌版本
func GetTokenVersion(c *gin.Context) (int, bool) {
	version, exists := c.Get("tokenVersion")
	if !exists {
		return 0, false
	}
	return version.(int), true
}

// OptionalAuth 可选的JWT认证中间件
// 不会阻止未认证的用户访问，但如果提供了有效的token会设置用户信息到上下文
func OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// 没有提供token，继续执行，但不设置用户信息
			c.Next()
			return
		}

		// 检查格式
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			// token格式错误，继续执行，但不设置用户信息
			logger.Warnf("Authorization格式错误: %s", authHeader)
			c.Next()
			return
		}

		// 验证token
		claims, err := auth.ParseToken(parts[1])
		if err != nil {
			// token无效，继续执行，但不设置用户信息
			logger.Warnf("无效的令牌: %v", err)
			c.Next()
			return
		}

		// 验证token类型
		if claims.Type != auth.AccessToken {
			// token类型错误，继续执行，但不设置用户信息
			logger.Warnf("使用了错误类型的令牌: %v", claims.Type)
			c.Next()
			return
		}

		// 检查令牌是否临近过期（可选）
		bufferTime := time.Duration(config.GlobalConfig.JWT.BufferSeconds) * time.Second
		if time.Until(time.Unix(claims.ExpiresAt, 0)) < bufferTime {
			c.Header("X-Token-Expire-Soon", "true")
		}

		// 将用户ID存入上下文
		c.Set("userID", claims.UserID)
		c.Set("userRole", claims.Role)
		c.Set("tokenID", claims.TokenID)
		
		c.Next()
	}
}
