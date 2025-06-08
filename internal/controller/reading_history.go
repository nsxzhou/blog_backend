package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nsxzhou1114/blog-api/internal/dto"
	"github.com/nsxzhou1114/blog-api/internal/logger"
	"github.com/nsxzhou1114/blog-api/internal/middleware"
	"github.com/nsxzhou1114/blog-api/internal/service"
	"github.com/nsxzhou1114/blog-api/pkg/response"
	"go.uber.org/zap"
)

// ReadingHistoryApi 阅读历史API控制器
type ReadingHistoryApi struct {
	logger                *zap.SugaredLogger
	readingHistoryService *service.ReadingHistoryService
}

// NewReadingHistoryApi 创建阅读历史API控制器
func NewReadingHistoryApi() *ReadingHistoryApi {
	return &ReadingHistoryApi{
		logger:                logger.GetSugaredLogger(),
		readingHistoryService: service.NewReadingHistoryService(),
	}
}

// CreateOrUpdate 创建或更新阅读历史记录
func (api *ReadingHistoryApi) CreateOrUpdate(c *gin.Context) {
	// 获取当前用户ID
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录", nil)
		return
	}

	var req dto.ReadingHistoryCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	// 获取客户端IP
	ip := c.ClientIP()

	if err := api.readingHistoryService.CreateOrUpdate(userID, &req, ip); err != nil {
		api.logger.Errorf("记录阅读历史失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "记录阅读历史失败", err)
		return
	}

	response.Success(c, "记录成功", nil)
}

// GetUserReadingHistory 获取用户阅读历史列表
func (api *ReadingHistoryApi) GetUserReadingHistory(c *gin.Context) {
	// 获取当前用户ID
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录", nil)
		return
	}

	var req dto.ReadingHistoryQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	histories, err := api.readingHistoryService.GetUserReadingHistory(userID, &req)
	if err != nil {
		api.logger.Errorf("获取阅读历史失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取阅读历史失败", err)
		return
	}

	response.Success(c, "获取成功", histories)
}

// DeleteReadingHistory 删除阅读历史记录
func (api *ReadingHistoryApi) DeleteReadingHistory(c *gin.Context) {
	// 获取当前用户ID
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录", nil)
		return
	}

	var req dto.ReadingHistoryDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	if err := api.readingHistoryService.DeleteReadingHistory(userID, &req); err != nil {
		api.logger.Errorf("删除阅读历史失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "删除阅读历史失败", err)
		return
	}

	response.Success(c, "删除成功", nil)
}

// ClearUserReadingHistory 清空用户所有阅读历史
func (api *ReadingHistoryApi) ClearUserReadingHistory(c *gin.Context) {
	// 获取当前用户ID
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录", nil)
		return
	}

	if err := api.readingHistoryService.ClearUserReadingHistory(userID); err != nil {
		api.logger.Errorf("清空阅读历史失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "清空阅读历史失败", err)
		return
	}

	response.Success(c, "清空成功", nil)
}

// DeleteSingleRecord 删除单条阅读历史记录
func (api *ReadingHistoryApi) DeleteSingleRecord(c *gin.Context) {
	// 获取当前用户ID
	userID, exists := middleware.GetUserID(c)
	if !exists {
		response.Unauthorized(c, "请先登录", nil)
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的记录ID", err)
		return
	}

	req := &dto.ReadingHistoryDeleteRequest{
		IDs: []uint{uint(id)},
	}

	if err := api.readingHistoryService.DeleteReadingHistory(userID, req); err != nil {
		api.logger.Errorf("删除阅读历史失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "删除阅读历史失败", err)
		return
	}

	response.Success(c, "删除成功", nil)
}
