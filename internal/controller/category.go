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

// CategoryApi 分类API控制器
type CategoryApi struct {
	logger          *zap.SugaredLogger
	categoryService *service.CategoryService
}

// NewCategoryApi 创建分类API控制器
func NewCategoryApi() *CategoryApi {
	return &CategoryApi{
		logger:          logger.GetSugaredLogger(),
		categoryService: service.NewCategoryService(),
	}
}

// Create 创建分类
func (api *CategoryApi) Create(c *gin.Context) {
	// 只有管理员可以创建分类
	role, exists := middleware.GetUserRole(c)
	if !exists || role != "admin" {
		response.Forbidden(c, "需要管理员权限", nil)
		return
	}

	var req dto.CategoryCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	category, err := api.categoryService.Create(&req)
	if err != nil {
		api.logger.Errorf("创建分类失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "创建分类失败", err)
		return
	}

	// 获取完整的分类信息
	categoryResp, err := api.categoryService.GenerateCategoryResponse(category)
	if err != nil {
		api.logger.Errorf("生成分类响应失败: %v", err)
	}

	response.Success(c, "创建成功", gin.H{
		"category": categoryResp,
	})
}

// Update 更新分类
func (api *CategoryApi) Update(c *gin.Context) {
	// 只有管理员可以更新分类
	role, exists := middleware.GetUserRole(c)
	if !exists || role != "admin" {
		response.Forbidden(c, "需要管理员权限", nil)
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的分类ID", err)
		return
	}

	var req dto.CategoryUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	category, err := api.categoryService.Update(uint(id), &req)
	if err != nil {
		api.logger.Errorf("更新分类失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "更新分类失败", err)
		return
	}

	// 获取完整的分类信息
	categoryResp, err := api.categoryService.GenerateCategoryResponse(category)
	if err != nil {
		api.logger.Errorf("生成分类响应失败: %v", err)
	}

	response.Success(c, "更新成功", gin.H{
		"category": categoryResp,
	})
}

// Delete 删除分类
func (api *CategoryApi) Delete(c *gin.Context) {
	// 只有管理员可以删除分类
	role, exists := middleware.GetUserRole(c)
	if !exists || role != "admin" {
		response.Forbidden(c, "需要管理员权限", nil)
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的分类ID", err)
		return
	}

	if err := api.categoryService.Delete(uint(id)); err != nil {
		api.logger.Errorf("删除分类失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "删除分类失败", err)
		return
	}

	response.Success(c, "删除成功", nil)
}

// GetByID 根据ID获取分类
func (api *CategoryApi) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的分类ID", err)
		return
	}

	// 获取分类信息
	category, err := api.categoryService.GetByID(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "分类不存在", err)
			return
		}
		api.logger.Errorf("获取分类失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取分类失败", err)
		return
	}

	// 转换为响应DTO
	categoryResp, err := api.categoryService.GenerateCategoryResponse(category)
	if err != nil {
		api.logger.Errorf("生成分类响应失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "生成分类响应失败", err)
		return
	}

	response.Success(c, "获取成功", gin.H{
		"category": categoryResp,
	})
}

// List 获取分类列表
func (api *CategoryApi) List(c *gin.Context) {
	var req dto.CategoryListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	categories, err := api.categoryService.List(&req)
	if err != nil {
		api.logger.Errorf("获取分类列表失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取分类列表失败", err)
		return
	}

	response.Success(c, "获取成功", categories)
}

// GetHotCategories 获取热门分类
func (api *CategoryApi) GetHotCategories(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10 // 默认10个
	}

	categories, err := api.categoryService.GetHotCategories(limit)
	if err != nil {
		api.logger.Errorf("获取热门分类失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取热门分类失败", err)
		return
	}

	response.Success(c, "获取成功", categories)
}
