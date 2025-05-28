package service

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nsxzhou1114/blog-api/internal/config"
	"github.com/nsxzhou1114/blog-api/internal/database"
	"github.com/nsxzhou1114/blog-api/internal/dto"
	"github.com/nsxzhou1114/blog-api/internal/logger"
	"github.com/nsxzhou1114/blog-api/internal/model"
	"github.com/tencentyun/cos-go-sdk-v5"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ImageService 图片服务
type ImageService struct {
	db     *gorm.DB
	log    *zap.SugaredLogger
	config *config.Config
}

// NewImageService 创建图片服务实例
func NewImageService() *ImageService {
	return &ImageService{
		db:     database.GetDB(),
		log:    logger.GetSugaredLogger(),
		config: config.GetConfig(),
	}
}

// Upload 上传图片
func (s *ImageService) Upload(userID uint, file *multipart.FileHeader, req *dto.ImageUploadRequest) (*dto.ImageUploadResponse, error) {
	// 1. 验证文件
	if err := s.validateImageFile(file); err != nil {
		return nil, err
	}

	// 2. 读取文件数据
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %v", err)
	}
	defer src.Close()

	fileData, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("读取文件数据失败: %v", err)
	}

	// 3. 获取图片尺寸
	width, height, err := s.getImageDimensions(fileData)
	if err != nil {
		s.log.Warnf("获取图片尺寸失败: %v", err)
		// 继续执行，只是没有尺寸信息
	}

	// 4. 确定存储类型
	storageType := req.StorageType
	if storageType == "" {
		storageType = s.config.Image.DefaultStorage // 使用配置文件中的默认存储类型
	}

	// 5. 根据存储类型上传文件
	var imageURL, imagePath string
	var isExternal int = 0

	if storageType == "cos" {
		if !s.config.Image.CosEnabled {
			return nil, errors.New("腾讯云COS存储未启用")
		}
		imageURL, err = s.uploadToTencentCOS(file, fileData)
		if err != nil {
			return nil, err
		}
		imagePath = imageURL
		isExternal = 1
	} else {
		if !s.config.Image.LocalEnabled {
			return nil, errors.New("本地存储未启用")
		}
		imagePath, imageURL, err = s.uploadToLocal(file, fileData)
		if err != nil {
			return nil, err
		}
		isExternal = 0
	}

	// 6. 保存图片信息到数据库
	imageModel := &model.Image{
		URL:         imageURL,
		Path:        imagePath,
		Filename:    file.Filename,
		Size:        int(file.Size),
		Width:       width,
		Height:      height,
		MimeType:    file.Header.Get("Content-Type"),
		UserID:      userID,
		UsageType:   req.UsageType,
		ArticleID:   req.ArticleID,
		IsExternal:  isExternal,
		StorageType: storageType,
	}

	if err := s.db.Create(imageModel).Error; err != nil {
		// 如果数据库保存失败，删除已上传的文件
		if storageType == "local" {
			os.Remove(imagePath)
		}
		return nil, fmt.Errorf("保存图片信息失败: %v", err)
	}

	// 7. 返回响应
	response := &dto.ImageUploadResponse{
		ID:          imageModel.ID,
		URL:         imageModel.URL,
		Path:        imageModel.Path,
		Filename:    imageModel.Filename,
		Size:        imageModel.Size,
		Width:       imageModel.Width,
		Height:      imageModel.Height,
		MimeType:    imageModel.MimeType,
		UsageType:   imageModel.UsageType,
		StorageType: imageModel.StorageType,
	}

	return response, nil
}

// GetByID 根据ID获取图片
func (s *ImageService) GetByID(imageID uint) (*model.Image, error) {
	var image model.Image
	err := s.db.Preload("User").Preload("Article").First(&image, imageID).Error
	if err != nil {
		return nil, err
	}
	return &image, nil
}

// GetDetail 获取图片详情
func (s *ImageService) GetDetail(imageID uint) (*dto.ImageDetailResponse, error) {
	image, err := s.GetByID(imageID)
	if err != nil {
		return nil, err
	}

	detail := &dto.ImageDetailResponse{
		ID:          image.ID,
		URL:         image.URL,
		Path:        image.Path,
		Filename:    image.Filename,
		Size:        image.Size,
		Width:       image.Width,
		Height:      image.Height,
		MimeType:    image.MimeType,
		UserID:      image.UserID,
		UsageType:   image.UsageType,
		ArticleID:   image.ArticleID,
		IsExternal:  image.IsExternal,
		StorageType: image.StorageType,
		CreatedAt:   image.CreatedAt,
		UpdatedAt:   image.UpdatedAt,
	}

	// 填充用户信息
	if image.User.ID > 0 {
		detail.UserName = image.User.Username
		detail.UserAvatar = image.User.Avatar
	}

	// 填充文章信息
	if image.Article != nil && image.Article.ID > 0 {
		detail.ArticleTitle = image.Article.Title
	}

	return detail, nil
}

// List 获取图片列表
func (s *ImageService) List(req *dto.ImageQueryRequest) (*dto.ImageListResponse, error) {
	var images []model.Image
	var total int64
	// 构建查询
	query := s.db.Model(&model.Image{})
	// 添加查询条件
	if req.UsageType != "" {
		query = query.Where("usage_type = ?", req.UsageType)
	}
	if req.ArticleID != nil && *req.ArticleID > 0 {
		query = query.Where("article_id = ?", *req.ArticleID)
	}
	if req.StorageType != "" {
		query = query.Where("storage_type = ?", req.StorageType)
	}
	if req.IsExternal != nil {
		query = query.Where("is_external = ?", *req.IsExternal)
	}
	if req.StartDate != "" {
		query = query.Where("created_at >= ?", req.StartDate)
	}
	if req.EndDate != "" {
		query = query.Where("created_at <= ?", req.EndDate)
	}
	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	err := query.Preload("User").Preload("Article").
		Order("created_at DESC").
		Offset(offset).
		Limit(req.PageSize).
		Find(&images).Error
	if err != nil {
		return nil, err
	}

	// 转换为响应格式
	items := make([]dto.ImageListItem, len(images))
	for i, img := range images {
		items[i] = dto.ImageListItem{
			ID:          img.ID,
			URL:         img.URL,
			Path:        img.Path,
			Filename:    img.Filename,
			Size:        img.Size,
			Width:       img.Width,
			Height:      img.Height,
			MimeType:    img.MimeType,
			UserID:      img.UserID,
			UsageType:   img.UsageType,
			ArticleID:   img.ArticleID,
			IsExternal:  img.IsExternal,
			StorageType: img.StorageType,
			CreatedAt:   img.CreatedAt,
			UpdatedAt:   img.UpdatedAt,
		}

		// 填充用户信息
		if img.User.ID > 0 {
			items[i].UserName = img.User.Username
		}

		// 填充文章信息
		if img.Article != nil && img.Article.ID > 0 {
			items[i].ArticleTitle = img.Article.Title
		}
	}

	return &dto.ImageListResponse{
		Total: total,
		Items: items,
	}, nil
}

// Update 更新图片信息
func (s *ImageService) Update(userID uint, imageID uint, req *dto.ImageUpdateRequest) error {
	// 查找图片并验证权限
	var image model.Image
	if err := s.db.First(&image, imageID).Error; err != nil {
		return err
	}

	// 检查权限
	if image.UserID != userID {
		return errors.New("没有权限修改此图片")
	}

	// 更新字段
	updates := map[string]interface{}{}
	if req.UsageType != "" {
		updates["usage_type"] = req.UsageType
	}
	if req.ArticleID != nil {
		updates["article_id"] = req.ArticleID
	}

	if len(updates) > 0 {
		return s.db.Model(&image).Updates(updates).Error
	}

	return nil
}

// Delete 删除图片
func (s *ImageService) Delete(userID uint, imageID uint) error {
	// 查找图片并验证权限
	var image model.Image
	if err := s.db.First(&image, imageID).Error; err != nil {
		return err
	}

	// 软删除数据库记录
	if err := s.db.Delete(&image).Error; err != nil {
		return err
	}

	// 删除物理文件（本地存储）
	if image.StorageType == "local" && image.IsExternal == 0 {
		go func() {
			if err := os.Remove(image.Path); err != nil {
				s.log.Errorf("删除本地文件失败: %v", err)
			}
		}()
	}

	return nil
}

// BatchDelete 批量删除图片
func (s *ImageService) BatchDelete(userID uint, req *dto.ImageBatchDeleteRequest) error {
	// 查找所有要删除的图片
	var images []model.Image
	err := s.db.Where("id IN ?", req.ImageIDs).Find(&images).Error
	if err != nil {
		return err
	}

	if len(images) == 0 {
		return errors.New("没有找到可删除的图片")
	}

	// 启动事务
	tx := s.db.Begin()

	// 软删除数据库记录
	if err := tx.Delete(&images).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()

	// 异步删除物理文件
	go func() {
		for _, img := range images {
			if img.StorageType == "local" && img.IsExternal == 0 {
				if err := os.Remove(img.Path); err != nil {
					s.log.Errorf("删除本地文件失败: %v", err)
				}
			}
		}
	}()

	return nil
}

// GetStatistics 获取图片统计数据
func (s *ImageService) GetStatistics(req *dto.ImageStatRequest) (*dto.ImageStatResponse, error) {
	var stats dto.ImageStatResponse

	// 构建基础查询
	query := s.db.Model(&model.Image{})
	if req.StartDate != "" {
		query = query.Where("created_at >= ?", req.StartDate)
	}
	if req.EndDate != "" {
		query = query.Where("created_at <= ?", req.EndDate)
	}

	// 总图片数
	query.Count(&stats.TotalImages)

	// 总存储大小
	var totalSize sql.NullInt64
	query.Select("SUM(size)").Scan(&totalSize)
	if totalSize.Valid {
		stats.TotalSize = totalSize.Int64
	}

	// 按存储类型统计
	s.db.Model(&model.Image{}).Where("storage_type = ?", "local").Count(&stats.LocalImages)
	s.db.Model(&model.Image{}).Where("storage_type = ?", "cos").Count(&stats.CosImages)

	// 按使用类型统计
	s.db.Model(&model.Image{}).Where("usage_type = ?", "avatar").Count(&stats.AvatarImages)
	s.db.Model(&model.Image{}).Where("usage_type = ?", "cover").Count(&stats.CoverImages)
	s.db.Model(&model.Image{}).Where("usage_type = ?", "content").Count(&stats.ContentImages)

	// 按日期统计（最近30天）
	var dailyStats []dto.ImageDailyStat
	err := s.db.Raw(`
		SELECT 
			DATE(created_at) as date,
			COUNT(*) as count,
			SUM(size) as size,
			SUM(CASE WHEN storage_type = 'local' THEN 1 ELSE 0 END) as local,
			SUM(CASE WHEN storage_type = 'cos' THEN 1 ELSE 0 END) as cos
		FROM images 
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 30 DAY)
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`).Scan(&dailyStats).Error

	if err != nil {
		return nil, err
	}

	stats.DailyStats = dailyStats

	return &stats, nil
}

// GetStorageConfig 获取存储配置
func (s *ImageService) GetStorageConfig() *dto.ImageStorageConfigResponse {
	return &dto.ImageStorageConfigResponse{
		LocalEnabled:    s.config.Image.LocalEnabled,
		CosEnabled:      s.config.Image.CosEnabled,
		DefaultStorage:  s.config.Image.DefaultStorage,
		MaxFileSize:     s.config.Image.Upload.MaxFileSize,
		AllowedTypes:    s.config.Image.Upload.AllowedTypes,
		LocalUploadPath: s.config.Image.Upload.Local.URLPrefix,
	}
}

// validateImageFile 验证图片文件
func (s *ImageService) validateImageFile(file *multipart.FileHeader) error {
	// 检查文件大小
	maxSize := s.config.Image.Upload.MaxFileSize
	if file.Size > maxSize {
		return fmt.Errorf("文件大小超过限制，最大允许 %d MB", maxSize/(1024*1024))
	}

	// 检查文件类型
	allowedTypes := make(map[string]bool)
	for _, t := range s.config.Image.Upload.AllowedTypes {
		allowedTypes[t] = true
	}

	contentType := file.Header.Get("Content-Type")
	if !allowedTypes[contentType] {
		return fmt.Errorf("不支持的文件类型: %s", contentType)
	}

	return nil
}

// getImageDimensions 获取图片尺寸
func (s *ImageService) getImageDimensions(data []byte) (*int, *int, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	return &width, &height, nil
}

// uploadToLocal 上传到本地存储
func (s *ImageService) uploadToLocal(file *multipart.FileHeader, data []byte) (string, string, error) {
	// 使用配置文件中的上传目录
	uploadDir := s.config.Image.Upload.Local.UploadPath
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", "", fmt.Errorf("创建上传目录失败: %v", err)
	}

	// 生成文件名
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	filePath := filepath.Join(uploadDir, filename)

	// 保存文件
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", "", fmt.Errorf("保存文件失败: %v", err)
	}

	// 生成访问URL
	fileURL := fmt.Sprintf("%s/%s", s.config.Image.Upload.Local.URLPrefix, filename)

	return filePath, fileURL, nil
}

// uploadToTencentCOS 上传文件到腾讯云COS
func (s *ImageService) uploadToTencentCOS(file *multipart.FileHeader, data []byte) (string, error) {
	// 使用配置文件中的腾讯云配置
	cosConfig := s.config.Image.Upload.COS

	// 创建COS客户端
	u, err := url.Parse(cosConfig.BucketURL)
	if err != nil {
		return "", fmt.Errorf("解析COS URL失败: %v", err)
	}

	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  cosConfig.SecretID,
			SecretKey: cosConfig.SecretKey,
		},
	})

	// 生成文件名，使用时间戳避免重名
	ext := filepath.Ext(file.Filename)
	fileName := fmt.Sprintf("images/%d%s", time.Now().UnixNano(), ext)

	// 上传对象
	r := bytes.NewReader(data)
	_, err = client.Object.Put(context.Background(), fileName, r, nil)
	if err != nil {
		s.log.Errorf("上传到腾讯云失败: %v", err)
		return "", fmt.Errorf("上传到腾讯云失败: %v", err)
	}

	// 返回完整的访问URL
	return fmt.Sprintf("%s/%s", strings.TrimRight(cosConfig.BucketURL, "/"), fileName), nil
}
