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

// TagApi 标签API控制器
type TagApi struct {
	logger     *zap.SugaredLogger
	tagService *service.TagService
}

// NewTagApi 创建标签API控制器
func NewTagApi() *TagApi {
	return &TagApi{
		logger:     logger.GetSugaredLogger(),
		tagService: service.NewTagService(),
	}
}

// Create 创建标签
func (api *TagApi) Create(c *gin.Context) {
	// 只有管理员可以创建标签
	role, exists := middleware.GetUserRole(c)
	if !exists || role != "admin" {
		response.Forbidden(c, "需要管理员权限", nil)
		return
	}

	var req dto.TagCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	tag, err := api.tagService.Create(&req)
	if err != nil {
		api.logger.Errorf("创建标签失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "创建标签失败", err)
		return
	}

	response.Success(c, "创建成功", gin.H{
		"tag": api.tagService.GenerateTagResponse(tag),
	})
}

// Update 更新标签
func (api *TagApi) Update(c *gin.Context) {
	// 只有管理员可以更新标签
	role, exists := middleware.GetUserRole(c)
	if !exists || role != "admin" {
		response.Forbidden(c, "需要管理员权限", nil)
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的标签ID", err)
		return
	}

	var req dto.TagUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	tag, err := api.tagService.Update(uint(id), &req)
	if err != nil {
		api.logger.Errorf("更新标签失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "更新标签失败", err)
		return
	}

	response.Success(c, "更新成功", gin.H{
		"tag": api.tagService.GenerateTagResponse(tag),
	})
}

// Delete 删除标签
func (api *TagApi) Delete(c *gin.Context) {
	// 只有管理员可以删除标签
	role, exists := middleware.GetUserRole(c)
	if !exists || role != "admin" {
		response.Forbidden(c, "需要管理员权限", nil)
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的标签ID", err)
		return
	}

	if err := api.tagService.Delete(uint(id)); err != nil {
		api.logger.Errorf("删除标签失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "删除标签失败", err)
		return
	}

	response.Success(c, "删除成功", nil)
}

// GetByID 根据ID获取标签
func (api *TagApi) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的标签ID", err)
		return
	}

	tag, err := api.tagService.GetByID(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "标签不存在", err)
			return
		}
		api.logger.Errorf("获取标签失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取标签失败", err)
		return
	}

	response.Success(c, "获取成功", gin.H{
		"tag": api.tagService.GenerateTagResponse(tag),
	})
}

// List 获取标签列表
func (api *TagApi) List(c *gin.Context) {
	var req dto.TagListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	tags, err := api.tagService.List(&req)
	if err != nil {
		api.logger.Errorf("获取标签列表失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取标签列表失败", err)
		return
	}

	response.Success(c, "获取成功", tags)
}

// GetTagCloud 获取标签云
func (api *TagApi) GetTagCloud(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "30")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 30 // 默认30个
	}

	tags, err := api.tagService.GetTagCloud(limit)
	if err != nil {
		api.logger.Errorf("获取标签云失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取标签云失败", err)
		return
	}

	response.Success(c, "获取成功", tags)
}
