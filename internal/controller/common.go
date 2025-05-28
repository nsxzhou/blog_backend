package controller

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/nsxzhou1114/blog-api/internal/middleware"
)

// getUserIDFromContext 从上下文中获取用户ID
func getUserIDFromContext(c *gin.Context) (uint, error) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		return 0, errors.New("用户未登录")
	}
	return userID, nil
}
