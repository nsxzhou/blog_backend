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
	categoryService     *CategoryService
	categoryServiceOnce sync.Once
)

// CategoryService 分类服务
type CategoryService struct {
	db     *gorm.DB
	logger *zap.SugaredLogger
}

// NewCategoryService 创建分类服务实例
func NewCategoryService() *CategoryService {
	categoryServiceOnce.Do(func() {
		categoryService = &CategoryService{
			db:     database.GetDB(),
			logger: logger.GetSugaredLogger(),
		}
	})
	return categoryService
}

// Create 创建分类
func (s *CategoryService) Create(req *dto.CategoryCreateRequest) (*model.Category, error) {
	// 检查分类名是否已存在
	var count int64
	if err := s.db.Model(&model.Category{}).Where("name = ?", req.Name).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, errors.New("分类名已存在")
	}

	// 设置默认值
	if req.IsVisible == 0 {
		req.IsVisible = 1 // 默认可见
	}

	// 创建分类
	category := &model.Category{
		Name:        req.Name,
		Description: req.Description,
		Icon:        req.Icon,
		IsVisible:   req.IsVisible,
	}

	if err := s.db.Create(category).Error; err != nil {
		return nil, err
	}

	return category, nil
}

// Update 更新分类
func (s *CategoryService) Update(id uint, req *dto.CategoryUpdateRequest) (*model.Category, error) {
	// 检查分类是否存在
	category, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// 检查新分类名是否与其他分类冲突
	if category.Name != req.Name {
		var count int64
		if err := s.db.Model(&model.Category{}).Where("name = ? AND id != ?", req.Name, id).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, errors.New("分类名已存在")
		}
	}

	// 更新分类
	updates := map[string]interface{}{
		"name":        req.Name,
		"description": req.Description,
		"icon":        req.Icon,
		"is_visible":  req.IsVisible,
	}

	if err := s.db.Model(category).Updates(updates).Error; err != nil {
		return nil, err
	}

	// 重新获取完整信息
	return s.GetByID(id)
}

// Delete 删除分类
func (s *CategoryService) Delete(id uint) error {
	// 检查分类是否存在
	category, err := s.GetByID(id)
	if err != nil {
		return err
	}

	// 检查是否有关联的文章
	if category.ArticleCount > 0 {
		return errors.New("该分类下还有关联的文章，无法删除")
	}

	// 删除分类
	if err := s.db.Delete(category).Error; err != nil {
		return err
	}

	return nil
}

// GetByID 根据ID获取分类
func (s *CategoryService) GetByID(id uint) (*model.Category, error) {
	var category model.Category
	if err := s.db.First(&category, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("分类不存在")
		}
		return nil, err
	}
	return &category, nil
}

// List 获取分类列表
func (s *CategoryService) List(req *dto.CategoryListRequest) (*dto.CategoryListResponse, error) {
	// 默认参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.OrderBy == "" {
		req.OrderBy = "created_at"
	}
	if req.Order == "" {
		req.Order = "asc"
	}

	// 构建查询
	query := s.db.Model(&model.Category{})

	// 关键词搜索
	if req.Keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	// 可见性筛选
	if req.IsVisible >= 0 {
		if req.IsVisible == 2 {
			query = query.Where("is_visible >= 0")
		} else {
			query = query.Where("is_visible = ?", req.IsVisible)
		}
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 排序和分页
	orderStr := fmt.Sprintf("%s %s", req.OrderBy, req.Order)
	var categories []model.Category
	if err := query.Order(orderStr).
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Find(&categories).Error; err != nil {
		return nil, err
	}

	// 构建响应
	resp := &dto.CategoryListResponse{
		Total: total,
		List:  make([]dto.CategoryResponse, 0, len(categories)),
	}

	// 生成分类响应
	for _, category := range categories {
		categoryResp, err := s.GenerateCategoryResponse(&category)
		if err != nil {
			s.logger.Warnf("生成分类响应失败: %v", err)
			continue
		}
		resp.List = append(resp.List, *categoryResp)
	}

	return resp, nil
}

// GetHotCategories 获取热门分类
func (s *CategoryService) GetHotCategories(limit int) (*dto.HotCategoryResponse, error) {
	if limit <= 0 {
		limit = 10 // 默认显示10个
	}

	var categories []model.Category
	if err := s.db.Model(&model.Category{}).
		Where("is_visible = ?", 1).
		Order("article_count DESC").
		Limit(limit).
		Find(&categories).Error; err != nil {
		return nil, err
	}

	resp := &dto.HotCategoryResponse{
		List: make([]dto.HotCategoryItem, 0, len(categories)),
	}

	for _, category := range categories {
		resp.List = append(resp.List, dto.HotCategoryItem{
			ID:           category.ID,
			Name:         category.Name,
			Icon:         category.Icon,
			ArticleCount: category.ArticleCount,
		})
	}

	return resp, nil
}

// IncrementArticleCount 增加分类文章计数
// func (s *CategoryService) IncrementArticleCount(categoryID uint) error {
// 	return s.db.Model(&model.Category{}).
// 		Where("id = ?", categoryID).
// 		UpdateColumn("article_count", gorm.Expr("article_count + ?", 1)).
// 		Error
// }

// DecrementArticleCount 减少分类文章计数
// func (s *CategoryService) DecrementArticleCount(categoryID uint) error {
// 	return s.db.Model(&model.Category{}).
// 		Where("id = ? AND article_count > 0", categoryID).
// 		UpdateColumn("article_count", gorm.Expr("article_count - ?", 1)).
// 		Error
// }

// GenerateCategoryResponse 生成分类响应DTO
func (s *CategoryService) GenerateCategoryResponse(category *model.Category) (*dto.CategoryResponse, error) {
	resp := &dto.CategoryResponse{
		ID:           category.ID,
		Name:         category.Name,
		Description:  category.Description,
		Icon:         category.Icon,
		ArticleCount: category.ArticleCount,
		IsVisible:    category.IsVisible,
		CreatedAt:    category.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:    category.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	return resp, nil
}
