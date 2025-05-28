package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nsxzhou1114/blog-api/internal/dto"
	"github.com/nsxzhou1114/blog-api/internal/logger"
	"github.com/nsxzhou1114/blog-api/internal/service"
	"github.com/nsxzhou1114/blog-api/pkg/response"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ImageApi 图片控制器
type ImageApi struct {
	logger       *zap.SugaredLogger
	imageService *service.ImageService
}

// NewImageApi 创建图片控制器实例
func NewImageApi() *ImageApi {
	return &ImageApi{
		logger:       logger.GetSugaredLogger(),
		imageService: service.NewImageService(),
	}
}

// Upload 上传图片
func (api *ImageApi) Upload(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "未授权", err)
		return
	}

	// 获取上传的文件
	file, err := c.FormFile("image")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "请选择要上传的图片", err)
		return
	}

	// 绑定请求参数
	var req dto.ImageUploadRequest
	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	// 调用服务层上传图片
	result, err := api.imageService.Upload(userID, file, &req)
	if err != nil {
		api.logger.Errorf("上传图片失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "上传图片失败", err)
		return
	}

	response.Success(c, "上传成功", result)
}

// GetDetail 获取图片详情
func (api *ImageApi) GetDetail(c *gin.Context) {
	imageIDStr := c.Param("id")
	imageID, err := strconv.ParseUint(imageIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的图片ID", err)
		return
	}

	detail, err := api.imageService.GetDetail(uint(imageID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "图片不存在", err)
			return
		}
		api.logger.Errorf("获取图片详情失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取图片详情失败", err)
		return
	}

	response.Success(c, "获取成功", detail)
}

// List 获取图片列表
func (api *ImageApi) List(c *gin.Context) {
	var req dto.ImageQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	images, err := api.imageService.List(&req)
	if err != nil {
		api.logger.Errorf("获取图片列表失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取图片列表失败", err)
		return
	}

	response.Success(c, "获取成功", images)
}

// Update 更新图片信息
func (api *ImageApi) Update(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "未授权", err)
		return
	}

	imageIDStr := c.Param("id")
	imageID, err := strconv.ParseUint(imageIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的图片ID", err)
		return
	}

	var req dto.ImageUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	err = api.imageService.Update(userID, uint(imageID), &req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "图片不存在", err)
			return
		}
		api.logger.Errorf("更新图片失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "更新图片失败", err)
		return
	}

	response.Success(c, "更新成功", nil)
}

// Delete 删除图片
func (api *ImageApi) Delete(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "未授权", err)
		return
	}

	imageIDStr := c.Param("id")
	imageID, err := strconv.ParseUint(imageIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的图片ID", err)
		return
	}

	err = api.imageService.Delete(userID, uint(imageID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "图片不存在", err)
			return
		}
		api.logger.Errorf("删除图片失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "删除图片失败", err)
		return
	}

	response.Success(c, "删除成功", nil)
}

// BatchDelete 批量删除图片
func (api *ImageApi) BatchDelete(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "未授权", err)
		return
	}

	var req dto.ImageBatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	err = api.imageService.BatchDelete(userID, &req)
	if err != nil {
		api.logger.Errorf("批量删除图片失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "批量删除图片失败", err)
		return
	}

	response.Success(c, "删除成功", nil)
}

// GetStatistics 获取图片统计数据
func (api *ImageApi) GetStatistics(c *gin.Context) {
	var req dto.ImageStatRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	stats, err := api.imageService.GetStatistics(&req)
	if err != nil {
		api.logger.Errorf("获取图片统计数据失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取图片统计数据失败", err)
		return
	}

	response.Success(c, "获取成功", stats)
}

// GetStorageConfig 获取存储配置
func (api *ImageApi) GetStorageConfig(c *gin.Context) {
	config := api.imageService.GetStorageConfig()
	response.Success(c, "获取成功", config)
}

// GetImagesByUsageType 根据使用类型获取图片
func (api *ImageApi) GetImagesByUsageType(c *gin.Context) {
	usageType := c.Param("type")
	if usageType == "" {
		response.Error(c, http.StatusBadRequest, "使用类型不能为空", nil)
		return
	}

	var req dto.ImageQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	// 设置使用类型
	req.UsageType = usageType

	images, err := api.imageService.List(&req)
	if err != nil {
		api.logger.Errorf("根据使用类型获取图片失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取图片失败", err)
		return
	}

	response.Success(c, "获取成功", images)
}

// GetImagesByArticle 根据文章ID获取图片
func (api *ImageApi) GetImagesByArticle(c *gin.Context) {
	articleIDStr := c.Param("article_id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的文章ID", err)
		return
	}

	var req dto.ImageQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	// 设置文章ID
	aid := uint(articleID)
	req.ArticleID = aid

	images, err := api.imageService.List(&req)
	if err != nil {
		api.logger.Errorf("根据文章ID获取图片失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取图片失败", err)
		return
	}

	response.Success(c, "获取成功", images)
}

