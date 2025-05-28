package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nsxzhou1114/blog-api/internal/dto"
	"github.com/nsxzhou1114/blog-api/internal/logger"
	"github.com/nsxzhou1114/blog-api/internal/middleware"
	"github.com/nsxzhou1114/blog-api/internal/service"
	"github.com/nsxzhou1114/blog-api/pkg/response"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CommentApi 评论API控制器
type CommentApi struct {
	logger         *zap.SugaredLogger
	commentService *service.CommentService
}

// NewCommentApi 创建评论API控制器
func NewCommentApi() *CommentApi {
	return &CommentApi{
		logger:         logger.GetSugaredLogger(),
		commentService: service.NewCommentService(),
	}
}

// Create 创建评论
func (api *CommentApi) Create(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "需要登录", nil)
		return
	}

	var req dto.CommentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	comment, err := api.commentService.Create(userID, &req)
	if err != nil {
		api.logger.Errorf("创建评论失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "创建评论失败", err)
		return
	}

	api.logger.Infof("创建评论成功: %v", comment.User)

	// 转换为响应DTO
	commentResp, err := api.commentService.GenerateCommentResponse(comment, &userID)
	if err != nil {
		api.logger.Errorf("生成评论响应失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "生成评论响应失败", err)
		return
	}

	api.logger.Infof("生成评论响应成功: %v", commentResp)

	// 构建响应消息
	message := "评论发布成功"
	if comment.RejectReason != "" {
		message = "评论已发布，但包含敏感内容已被过滤"
	}

	response.Success(c, message, gin.H{
		"comment": commentResp,
	})
}

// Update 更新评论
func (api *CommentApi) Update(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "需要登录", nil)
		return
	}

	// 检查用户角色
	role, roleExists := middleware.GetUserRole(c)
	isAdmin := roleExists && role == "admin"

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的评论ID", err)
		return
	}

	var req dto.CommentUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	comment, err := api.commentService.Update(uint(id), userID, isAdmin, &req)
	if err != nil {
		api.logger.Errorf("更新评论失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "更新评论失败", err)
		return
	}

	// 转换为响应DTO
	commentResp, err := api.commentService.GenerateCommentResponse(comment, &userID)
	if err != nil {
		api.logger.Errorf("生成评论响应失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "生成评论响应失败", err)
		return
	}

	response.Success(c, "更新成功", gin.H{
		"comment": commentResp,
	})
}

// Reply 回复评论
func (api *CommentApi) Reply(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "需要登录", nil)
		return
	}

	var req dto.CommentReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	comment, err := api.commentService.Reply(userID, &req)
	if err != nil {
		api.logger.Errorf("回复评论失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "回复评论失败", err)
		return
	}

	// 转换为响应DTO
	commentResp, err := api.commentService.GenerateCommentResponse(comment, &userID)
	if err != nil {
		api.logger.Errorf("生成评论响应失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "生成评论响应失败", err)
		return
	}

	// 构建响应消息
	message := "回复发布成功"
	if comment.RejectReason != "" {
		message = "回复已发布，但包含敏感内容已被过滤"
	}

	response.Success(c, message, gin.H{
		"comment": commentResp,
	})
}

// Delete 删除评论
func (api *CommentApi) Delete(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "需要登录", nil)
		return
	}

	// 检查用户角色
	role, roleExists := middleware.GetUserRole(c)
	isAdmin := roleExists && role == "admin"

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的评论ID", err)
		return
	}

	if err := api.commentService.Delete(uint(id), userID, isAdmin); err != nil {
		api.logger.Errorf("删除评论失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "删除评论失败", err)
		return
	}

	response.Success(c, "删除成功", nil)
}

// GetByID 根据ID获取评论
func (api *CommentApi) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的评论ID", err)
		return
	}

	// 获取当前用户ID（如果已登录）
	var userID *uint
	if tempUserID, exists := middleware.GetUserID(c); exists {
		userID = &tempUserID
	}

	comment, err := api.commentService.GetByID(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "评论不存在", err)
			return
		}
		api.logger.Errorf("获取评论失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取评论失败", err)
		return
	}

	// 转换为响应DTO
	commentResp, err := api.commentService.GenerateCommentResponse(comment, userID)
	if err != nil {
		api.logger.Errorf("生成评论响应失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "生成评论响应失败", err)
		return
	}

	response.Success(c, "获取成功", gin.H{
		"comment": commentResp,
	})
}

// List 获取评论列表
func (api *CommentApi) List(c *gin.Context) {
	var req dto.CommentListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	// 获取当前用户ID（如果已登录）
	var userID *uint
	if tempUserID, exists := middleware.GetUserID(c); exists {
		userID = &tempUserID
	}

	comments, err := api.commentService.List(&req, userID)
	if err != nil {
		api.logger.Errorf("获取评论列表失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取评论列表失败", err)
		return
	}

	response.Success(c, "获取成功", comments)
}

// UpdateStatus 更新评论状态（管理员使用）
func (api *CommentApi) UpdateStatus(c *gin.Context) {
	// 检查是否为管理员
	role, exists := middleware.GetUserRole(c)
	if !exists || role != "admin" {
		response.Forbidden(c, "需要管理员权限", nil)
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的评论ID", err)
		return
	}

	var req dto.CommentStatusUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	comment, err := api.commentService.UpdateStatus(uint(id), &req)
	if err != nil {
		api.logger.Errorf("更新评论状态失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "更新评论状态失败", err)
		return
	}

	response.Success(c, "更新成功", gin.H{
		"comment": comment,
	})
}

// BatchUpdateStatus 批量更新评论状态（管理员使用）
func (api *CommentApi) BatchUpdateStatus(c *gin.Context) {
	// 检查是否为管理员
	role, exists := middleware.GetUserRole(c)
	if !exists || role != "admin" {
		response.Forbidden(c, "需要管理员权限", nil)
		return
	}

	var req dto.CommentBatchStatusUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	if err := api.commentService.BatchUpdateStatus(&req); err != nil {
		api.logger.Errorf("批量更新评论状态失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "批量更新评论状态失败", err)
		return
	}

	response.Success(c, "更新成功", nil)
}

// Like 点赞评论
func (api *CommentApi) Like(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "需要登录", nil)
		return
	}

	var req dto.CommentLikeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	if err := api.commentService.Like(userID, req.CommentID); err != nil {
		api.logger.Errorf("点赞评论失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "点赞失败", err)
		return
	}

	response.Success(c, "操作成功", nil)
}

// GetNotifications 获取评论通知
func (api *CommentApi) GetNotifications(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "需要登录", nil)
		return
	}

	var req dto.CommentNotificationListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	notifications, err := api.commentService.GetNotifications(userID, &req)
	if err != nil {
		api.logger.Errorf("获取评论通知失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取评论通知失败", err)
		return
	}

	response.Success(c, "获取成功", notifications)
}

// MarkNotificationAsRead 标记评论通知为已读
func (api *CommentApi) MarkNotificationAsRead(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "需要登录", nil)
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的通知ID", err)
		return
	}

	if err := api.commentService.MarkNotificationAsRead(userID, uint(id)); err != nil {
		api.logger.Errorf("标记通知已读失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "标记已读失败", err)
		return
	}

	response.Success(c, "标记已读成功", nil)
}

// MarkAllNotificationsAsRead 标记所有评论通知为已读
func (api *CommentApi) MarkAllNotificationsAsRead(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "需要登录", nil)
		return
	}

	if err := api.commentService.MarkAllNotificationsAsRead(userID); err != nil {
		api.logger.Errorf("标记所有通知已读失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "标记所有已读失败", err)
		return
	}

	response.Success(c, "标记所有已读成功", nil)
}
