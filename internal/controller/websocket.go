package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nsxzhou1114/blog-api/internal/dto"
	"github.com/nsxzhou1114/blog-api/internal/logger"
	"github.com/nsxzhou1114/blog-api/internal/middleware"
	"github.com/nsxzhou1114/blog-api/internal/service"
	"github.com/nsxzhou1114/blog-api/pkg/auth"
	"github.com/nsxzhou1114/blog-api/pkg/response"
	"github.com/nsxzhou1114/blog-api/pkg/websocket"
	"go.uber.org/zap"
)

// WebSocketApi WebSocket API控制器
type WebSocketApi struct {
	logger              *zap.SugaredLogger
	websocketManager    *websocket.Manager
	notificationService *service.NotificationService
}

// NewWebSocketApi 创建WebSocket API实例
func NewWebSocketApi() *WebSocketApi {
	return &WebSocketApi{
		logger:              logger.GetSugaredLogger(),
		websocketManager:    websocket.GetManager(),
		notificationService: service.NewNotificationService(),
	}
}

// HandleWebSocket 处理WebSocket连接
func (api *WebSocketApi) HandleWebSocket(c *gin.Context) {
	// 从查询参数获取token
	token := c.Query("token")
	if token == "" {
		api.logger.Warnf("WebSocket连接缺少认证token")
		c.Status(http.StatusUnauthorized)
		c.Abort()
		return
	}

	// 验证token
	claims, err := auth.ParseToken(token)
	if err != nil {
		api.logger.Warnf("WebSocket连接token验证失败: %v", err)
		c.Status(http.StatusUnauthorized)
		c.Abort()
		return
	}

	// 验证token类型
	if claims.Type != auth.AccessToken {
		api.logger.Warnf("WebSocket连接使用了错误类型的token: %v", claims.Type)
		c.Status(http.StatusUnauthorized)
		c.Abort()
		return
	}

	userID := claims.UserID
	api.logger.Infof("用户 %d 尝试建立WebSocket连接", userID)

	// 直接调用WebSocket管理器处理连接
	api.websocketManager.HandleWebSocket(c, userID)
}

// GetNotifications 获取用户通知列表
func (api *WebSocketApi) GetNotifications(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "需要登录", nil)
		return
	}

	var req dto.NotificationListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	notifications, err := api.notificationService.GetUserNotifications(userID, &req)
	if err != nil {
		api.logger.Errorf("获取用户通知失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取通知失败", err)
		return
	}

	response.Success(c, "获取成功", notifications)
}

// GetUnreadCount 获取未读通知数量
func (api *WebSocketApi) GetUnreadCount(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "需要登录", nil)
		return
	}

	count, err := api.notificationService.GetUnreadCount(userID)
	if err != nil {
		api.logger.Errorf("获取未读通知数量失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取未读数量失败", err)
		return
	}

	response.Success(c, "获取成功", dto.NotificationUnreadCountResponse{
		Count: count,
	})
}

// MarkAsRead 标记通知为已读
func (api *WebSocketApi) MarkAsRead(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "需要登录", nil)
		return
	}

	idStr := c.Param("id")
	notificationID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的通知ID", err)
		return
	}

	if err := api.notificationService.MarkAsRead(userID, uint(notificationID)); err != nil {
		api.logger.Errorf("标记通知已读失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "标记已读失败", err)
		return
	}

	response.Success(c, "标记已读成功", nil)
}

// MarkAllAsRead 标记所有通知为已读
func (api *WebSocketApi) MarkAllAsRead(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "需要登录", nil)
		return
	}

	if err := api.notificationService.MarkAllAsRead(userID); err != nil {
		api.logger.Errorf("标记所有通知已读失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "标记所有已读失败", err)
		return
	}

	response.Success(c, "标记所有已读成功", nil)
}

// DeleteNotification 删除通知
func (api *WebSocketApi) DeleteNotification(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "需要登录", nil)
		return
	}

	idStr := c.Param("id")
	notificationID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的通知ID", err)
		return
	}

	if err := api.notificationService.DeleteNotification(userID, uint(notificationID)); err != nil {
		api.logger.Errorf("删除通知失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "删除通知失败", err)
		return
	}

	response.Success(c, "删除成功", nil)
}

// BatchDeleteNotifications 批量删除通知
func (api *WebSocketApi) BatchDeleteNotifications(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "需要登录", nil)
		return
	}

	var req dto.NotificationBatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	if err := api.notificationService.BatchDeleteNotifications(userID, req.NotificationIDs); err != nil {
		api.logger.Errorf("批量删除通知失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "批量删除失败", err)
		return
	}

	response.Success(c, "批量删除成功", nil)
}

// GetWebSocketStats 获取WebSocket统计信息（管理员）
func (api *WebSocketApi) GetWebSocketStats(c *gin.Context) {
	stats := api.websocketManager.GetStats()
	response.Success(c, "获取成功", stats)
}
