package service

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/nsxzhou1114/blog-api/internal/database"
	"github.com/nsxzhou1114/blog-api/internal/dto"
	"github.com/nsxzhou1114/blog-api/internal/logger"
	"github.com/nsxzhou1114/blog-api/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	readingHistoryService     *ReadingHistoryService
	readingHistoryServiceOnce sync.Once
)

// ReadingHistoryService 阅读历史服务
type ReadingHistoryService struct {
	db     *gorm.DB
	logger *zap.SugaredLogger
}

// NewReadingHistoryService 创建阅读历史服务实例
func NewReadingHistoryService() *ReadingHistoryService {
	readingHistoryServiceOnce.Do(func() {
		readingHistoryService = &ReadingHistoryService{
			db:     database.GetDB(),
			logger: logger.GetSugaredLogger(),
		}
	})
	return readingHistoryService
}

// CreateOrUpdate 创建或更新阅读历史记录
func (s *ReadingHistoryService) CreateOrUpdate(userID uint, req *dto.ReadingHistoryCreateRequest, ip string) error {
	// 验证文章是否存在且已发布
	var article model.Article
	if err := s.db.Where("id = ? AND status = ?", req.ArticleID, "published").First(&article).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("文章不存在或未发布")
		}
		return err
	}

	// 查找是否已有阅读记录
	var history model.ReadingHistory
	err := s.db.Where("user_id = ? AND article_id = ?", userID, req.ArticleID).First(&history).Error

	now := time.Now()

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 创建新记录
			history = model.ReadingHistory{
				UserID:    userID,
				ArticleID: req.ArticleID,
				ReadAt:    now,
				IP:        ip,
			}
			if err := s.db.Create(&history).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		// 更新现有记录
		updates := map[string]interface{}{
			"read_at": now,
			"ip":      ip,
		}
		if err := s.db.Model(&history).Updates(updates).Error; err != nil {
			return err
		}
	}

	return nil
}

// GetUserReadingHistory 获取用户阅读历史列表
func (s *ReadingHistoryService) GetUserReadingHistory(userID uint, req *dto.ReadingHistoryQueryRequest) (*dto.ReadingHistoryListResponse, error) {
	// 默认参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.OrderBy == "" {
		req.OrderBy = "read_at"
	}
	if req.Order == "" {
		req.Order = "desc"
	}

	// 构建查询
	query := s.db.Model(&model.ReadingHistory{}).
		Select("reading_histories.*, articles.title as article_title, articles.summary as article_summary, articles.cover_image as article_cover, articles.category_id, categories.name as category_name, articles.author_id, users.nickname as author_name").
		Joins("LEFT JOIN articles ON reading_histories.article_id = articles.id").
		Joins("LEFT JOIN categories ON articles.category_id = categories.id").
		Joins("LEFT JOIN users ON articles.author_id = users.id").
		Where("reading_histories.user_id = ?", userID)

	// 分类筛选
	if req.CategoryID > 0 {
		query = query.Where("articles.category_id = ?", req.CategoryID)
	}

	// 标签筛选
	if req.TagID > 0 {
		query = query.Joins("LEFT JOIN article_tags ON articles.id = article_tags.article_id").
			Where("article_tags.tag_id = ?", req.TagID)
	}

	// 关键词搜索
	if req.Keyword != "" {
		query = query.Where("articles.title LIKE ? OR articles.summary LIKE ?", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	// 时间范围筛选
	if req.StartDate != "" {
		query = query.Where("reading_histories.read_at >= ?", req.StartDate)
	}
	if req.EndDate != "" {
		query = query.Where("reading_histories.read_at <= ?", req.EndDate)
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 排序和分页
	orderStr := fmt.Sprintf("reading_histories.%s %s", req.OrderBy, req.Order)

	// 定义结果结构体
	type QueryResult struct {
		model.ReadingHistory
		ArticleTitle   string `json:"article_title"`
		ArticleSummary string `json:"article_summary"`
		ArticleCover   string `json:"article_cover"`
		CategoryID     uint   `json:"category_id"`
		CategoryName   string `json:"category_name"`
		AuthorID       uint   `json:"author_id"`
		AuthorName     string `json:"author_name"`
	}

	var results []QueryResult
	if err := query.Order(orderStr).
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Scan(&results).Error; err != nil {
		return nil, err
	}

	// 构建响应
	resp := &dto.ReadingHistoryListResponse{
		Total: total,
		List:  make([]dto.ReadingHistoryItem, 0, len(results)),
	}

	for _, result := range results {
		item := dto.ReadingHistoryItem{
			ID:             result.ID,
			ArticleID:      result.ArticleID,
			ArticleTitle:   result.ArticleTitle,
			ArticleSummary: result.ArticleSummary,
			ArticleCover:   result.ArticleCover,
			CategoryID:     result.CategoryID,
			CategoryName:   result.CategoryName,
			AuthorID:       result.AuthorID,
			AuthorName:     result.AuthorName,
			ReadAt:         result.ReadAt.Format("2006-01-02 15:04:05"),
			CreatedAt:      result.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		resp.List = append(resp.List, item)
	}

	return resp, nil
}

// DeleteReadingHistory 删除阅读历史记录
func (s *ReadingHistoryService) DeleteReadingHistory(userID uint, req *dto.ReadingHistoryDeleteRequest) error {
	if len(req.IDs) == 0 {
		return errors.New("请选择要删除的记录")
	}

	// 验证记录是否属于当前用户
	var count int64
	if err := s.db.Model(&model.ReadingHistory{}).
		Where("id IN ? AND user_id = ?", req.IDs, userID).
		Count(&count).Error; err != nil {
		return err
	}

	if count != int64(len(req.IDs)) {
		return errors.New("部分记录不存在或无权限删除")
	}

	// 删除记录
	if err := s.db.Where("id IN ? AND user_id = ?", req.IDs, userID).
		Delete(&model.ReadingHistory{}).Error; err != nil {
		return err
	}

	return nil
}

// ClearUserReadingHistory 清空用户所有阅读历史
func (s *ReadingHistoryService) ClearUserReadingHistory(userID uint) error {
	if err := s.db.Where("user_id = ?", userID).Delete(&model.ReadingHistory{}).Error; err != nil {
		return err
	}
	return nil
}
