package service

import (
	"errors"
	"fmt"
	"sync"

	"github.com/nsxzhou1114/blog-api/internal/database"
	"github.com/nsxzhou1114/blog-api/internal/dto"
	"github.com/nsxzhou1114/blog-api/internal/logger"
	"github.com/nsxzhou1114/blog-api/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	tagService     *TagService
	tagServiceOnce sync.Once
)

// TagService 标签服务
type TagService struct {
	db     *gorm.DB
	logger *zap.SugaredLogger
}

// NewTagService 创建标签服务实例
func NewTagService() *TagService {
	tagServiceOnce.Do(func() {
		tagService = &TagService{
			db:     database.GetDB(),
			logger: logger.GetSugaredLogger(),
		}
	})
	return tagService
}

// Create 创建标签
func (s *TagService) Create(req *dto.TagCreateRequest) (*model.Tag, error) {
	// 检查标签名是否已存在
	var count int64
	if err := s.db.Model(&model.Tag{}).Where("name = ?", req.Name).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, errors.New("标签名已存在")
	}

	// 创建标签
	tag := &model.Tag{
		Name: req.Name,
	}

	if err := s.db.Create(tag).Error; err != nil {
		return nil, err
	}

	return tag, nil
}

// Update 更新标签
func (s *TagService) Update(id uint, req *dto.TagUpdateRequest) (*model.Tag, error) {
	// 检查标签是否存在
	tag, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// 检查新标签名是否与其他标签冲突
	if tag.Name != req.Name {
		var count int64
		if err := s.db.Model(&model.Tag{}).Where("name = ? AND id != ?", req.Name, id).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, errors.New("标签名已存在")
		}
	}

	// 更新标签
	if err := s.db.Model(tag).Updates(map[string]interface{}{
		"name": req.Name,
	}).Error; err != nil {
		return nil, err
	}

	// 重新获取完整信息
	return s.GetByID(id)
}

// Delete 删除标签
func (s *TagService) Delete(id uint) error {
	// 检查标签是否存在
	tag, err := s.GetByID(id)
	if err != nil {
		return err
	}

	// 检查是否有关联的文章
	if tag.ArticleCount > 0 {
		return errors.New("该标签下还有关联的文章，无法删除")
	}

	// 开启事务
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 删除文章-标签关联
		if err := tx.Where("tag_id = ?", id).Delete(&model.ArticleTag{}).Error; err != nil {
			return err
		}

		// 删除标签
		if err := tx.Delete(tag).Error; err != nil {
			return err
		}

		return nil
	})
}

// GetByID 根据ID获取标签
func (s *TagService) GetByID(id uint) (*model.Tag, error) {
	var tag model.Tag
	if err := s.db.First(&tag, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("标签不存在")
		}
		return nil, err
	}
	return &tag, nil
}

// List 获取标签列表
func (s *TagService) List(req *dto.TagListRequest) (*dto.TagListResponse, error) {
	// 默认参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.OrderBy == "" {
		req.OrderBy = "id"
	}
	if req.Order == "" {
		req.Order = "desc"
	}

	// 构建查询
	query := s.db.Model(&model.Tag{})

	// 关键词搜索
	if req.Keyword != "" {
		query = query.Where("name LIKE ?", "%"+req.Keyword+"%")
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 排序和分页
	orderStr := fmt.Sprintf("%s %s", req.OrderBy, req.Order)
	var tags []model.Tag
	if err := query.Order(orderStr).
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Find(&tags).Error; err != nil {
		return nil, err
	}

	// 构建响应
	resp := &dto.TagListResponse{
		Total: total,
		List:  make([]dto.TagResponse, 0, len(tags)),
	}

	for _, tag := range tags {
		resp.List = append(resp.List, *s.GenerateTagResponse(&tag))
	}

	return resp, nil
}

// GetTagCloud 获取标签云
func (s *TagService) GetTagCloud(limit int) (*dto.TagCloudResponse, error) {
	if limit <= 0 {
		limit = 30 // 默认显示30个
	}

	var tags []model.Tag
	if err := s.db.Model(&model.Tag{}).
		Order("article_count DESC").
		Where("article_count > 0").
		Limit(limit).
		Find(&tags).Error; err != nil {
		return nil, err
	}

	resp := &dto.TagCloudResponse{
		List: make([]dto.TagCloudItem, 0, len(tags)),
	}

	for _, tag := range tags {
		resp.List = append(resp.List, dto.TagCloudItem{
			ID:           tag.ID,
			Name:         tag.Name,
			ArticleCount: tag.ArticleCount,
		})
	}

	return resp, nil
}

// UpdateTagArticleCount 更新标签文章计数
func (s *TagService) UpdateTagArticleCount(tx *gorm.DB, tagID uint) error {
	// 如果没有传入事务，使用默认数据库连接
	db := tx
	if db == nil {
		db = s.db
	}

	// 计算实际关联的已发布文章数
	var count int64
	if err := db.Model(&model.ArticleTag{}).
		Joins("JOIN articles ON articles.id = article_tags.article_id").
		Where("article_tags.tag_id = ? AND articles.status = 'published' ", tagID).
		Count(&count).Error; err != nil {
		return err
	}

	// 更新标签的文章计数
	return db.Model(&model.Tag{}).Where("id = ?", tagID).Update("article_count", count).Error
}

// UpdateMultipleTagArticleCount 批量更新多个标签的文章计数
func (s *TagService) UpdateMultipleTagArticleCount(tx *gorm.DB, tagIDs []uint) error {
	if len(tagIDs) == 0 {
		return nil
	}

	// 如果没有传入事务，使用默认数据库连接
	db := tx
	if db == nil {
		db = s.db
	}

	// 批量更新每个标签的文章计数
	for _, tagID := range tagIDs {
		if err := s.UpdateTagArticleCount(db, tagID); err != nil {
			return err
		}
	}

	return nil
}

// SyncAllTagArticleCount 同步所有标签的文章计数
func (s *TagService) SyncAllTagArticleCount() error {
	s.logger.Info("开始同步所有标签的文章计数...")

	// 获取所有标签
	var tags []model.Tag
	if err := s.db.Find(&tags).Error; err != nil {
		return err
	}

	// 使用事务更新所有标签计数
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, tag := range tags {
			if err := s.UpdateTagArticleCount(tx, tag.ID); err != nil {
				s.logger.Errorf("更新标签 %d (%s) 文章计数失败: %v", tag.ID, tag.Name, err)
				return err
			}
		}
		return nil
	})
}

// GetTagsByArticleID 获取文章的标签
func (s *TagService) GetTagsByArticleID(articleID uint) ([]model.Tag, error) {
	var tags []model.Tag
	if err := s.db.Joins("JOIN article_tags ON article_tags.tag_id = tags.id").
		Where("article_tags.article_id = ?", articleID).
		Find(&tags).Error; err != nil {
		return nil, err
	}
	return tags, nil
}

// GenerateTagResponse 生成标签响应DTO
func (s *TagService) GenerateTagResponse(tag *model.Tag) *dto.TagResponse {
	return &dto.TagResponse{
		ID:           tag.ID,
		Name:         tag.Name,
		ArticleCount: tag.ArticleCount,
		CreatedAt:    tag.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:    tag.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
