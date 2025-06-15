package service

import (
	"encoding/xml"
	"fmt"
	"html"
	"net/url"
	"strings"
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
	rssService     *RSSService
	rssServiceOnce sync.Once
)

// RSSService RSS服务
type RSSService struct {
	db *gorm.DB
	logger *zap.SugaredLogger
}

// NewRSSService 创建RSS服务实例
func NewRSSService() *RSSService {
	rssServiceOnce.Do(func() {
		rssService = &RSSService{
			db: database.GetDB(),
			logger: logger.GetSugaredLogger(),
		}
	})
	return rssService
}

// GenerateRSSFeed 生成RSS订阅数据
func (s *RSSService) GenerateRSSFeed(query *dto.RSSQuery, baseURL string) (string, error) {
	// 设置默认值
	if query.Limit <= 0 {
		query.Limit = 20
	}

	// 构建查询条件
	db := s.db.Model(&model.Article{}).
		Where("status = ?", "published").
		Where("access_type = ?", "public").
		Preload("Author").
		Preload("Category").
		Preload("Tags").
		Order("published_at DESC, created_at DESC")

	// 按分类筛选
	if query.CategoryID > 0 {
		db = db.Where("category_id = ?", query.CategoryID)
	}

	// 按标签筛选
	if query.TagID > 0 {
		db = db.Joins("JOIN article_tags ON articles.id = article_tags.article_id").
			Where("article_tags.tag_id = ?", query.TagID)
	}

	// 限制数量
	db = db.Limit(query.Limit)

	// 查询文章
	var articles []model.Article
	if err := db.Find(&articles).Error; err != nil {
		return "", fmt.Errorf("查询文章失败: %w", err)
	}

	// 生成RSS数据
	feed := s.buildRSSFeed(articles, baseURL, query)

	// 转换为XML
	xmlData, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		return "", fmt.Errorf("生成XML失败: %w", err)
	}

	// 添加XML声明
	xmlString := xml.Header + string(xmlData)
	return xmlString, nil
}

// buildRSSFeed 构建RSS订阅数据
func (s *RSSService) buildRSSFeed(articles []model.Article, baseURL string, query *dto.RSSQuery) *dto.RSSFeed {
	// 构建频道信息
	channel := dto.RSSChannel{
		Title:          s.getChannelTitle(query),
		Description:    s.getChannelDescription(query),
		Link:           baseURL,
		Language:       "zh-CN",
		LastBuildDate:  time.Now().Format(time.RFC1123Z),
		Generator:      "Blog API RSS Generator",
		WebMaster:      "1790146932@qq.com",
		ManagingEditor: "1790146932@qq.com",
		Items:          make([]dto.RSSItem, 0, len(articles)),
	}

	// 构建文章项
	for _, article := range articles {
		item := s.buildRSSItem(&article, baseURL)
		channel.Items = append(channel.Items, item)
	}

	return &dto.RSSFeed{
		Version: "2.0",
		Channel: channel,
	}
}

// buildRSSItem 构建RSS文章项
func (s *RSSService) buildRSSItem(article *model.Article, baseURL string) dto.RSSItem {
	// 构建文章链接
	articleURL := fmt.Sprintf("%s/article/%d", baseURL, article.ID)

	// 处理发布时间
	pubDate := ""
	if article.PublishedAt != nil {
		pubDate = article.PublishedAt.Format(time.RFC1123Z)
	} else {
		pubDate = article.CreatedAt.Format(time.RFC1123Z)
	}

	// 构建作者信息
	author := "匿名"
	if article.Author.Username != "" {
		author = article.Author.Username
	}

	// 构建分类信息
	category := "未分类"
	if article.Category.Name != "" {
		category = article.Category.Name
	}

	// 处理摘要，如果没有摘要则截取标题
	description := article.Summary
	if description == "" {
		description = article.Title
	}
	// HTML转义
	description = html.EscapeString(description)

	// 构建RSS项
	item := dto.RSSItem{
		Title:       html.EscapeString(article.Title),
		Description: description,
		Link:        articleURL,
		PubDate:     pubDate,
		GUID:        articleURL,
		Author:      html.EscapeString(author),
		Category:    html.EscapeString(category),
	}

	// 如果有封面图片，添加enclosure
	if article.CoverImage != "" {
		enclosureURL := s.buildImageURL(article.CoverImage, baseURL)
		if enclosureURL != "" {
			item.Enclosure = &dto.RSSEnclosure{
				URL:    enclosureURL,
				Type:   "image/jpeg", // 默认为JPEG，实际应该根据文件扩展名判断
				Length: 0,            // 暂时设为0，实际应该获取文件大小
			}
		}
	}

	return item
}

// getChannelTitle 获取频道标题
func (s *RSSService) getChannelTitle(query *dto.RSSQuery) string {
	baseTitle := "博客RSS订阅"

	if query.CategoryID > 0 {
		var category model.Category
		if err := s.db.First(&category, query.CategoryID).Error; err == nil {
			return fmt.Sprintf("%s - %s分类", baseTitle, category.Name)
		}
	}

	if query.TagID > 0 {
		var tag model.Tag
		if err := s.db.First(&tag, query.TagID).Error; err == nil {
			return fmt.Sprintf("%s - %s标签", baseTitle, tag.Name)
		}
	}

	return baseTitle
}

// getChannelDescription 获取频道描述
func (s *RSSService) getChannelDescription(query *dto.RSSQuery) string {
	baseDesc := "最新博客文章RSS订阅"

	if query.CategoryID > 0 {
		var category model.Category
		if err := s.db.First(&category, query.CategoryID).Error; err == nil {
			if category.Description != "" {
				return fmt.Sprintf("%s分类: %s", category.Name, category.Description)
			}
			return fmt.Sprintf("%s分类的最新文章", category.Name)
		}
	}

	if query.TagID > 0 {
		var tag model.Tag
		if err := s.db.First(&tag, query.TagID).Error; err == nil {
			return fmt.Sprintf("%s标签的最新文章", tag.Name)
		}
	}

	return baseDesc
}

// buildImageURL 构建图片URL
func (s *RSSService) buildImageURL(imagePath, baseURL string) string {
	if imagePath == "" {
		return ""
	}

	// 如果已经是完整URL，直接返回
	if strings.HasPrefix(imagePath, "http://") || strings.HasPrefix(imagePath, "https://") {
		return imagePath
	}

	// 构建完整的图片URL
	if strings.HasPrefix(imagePath, "/") {
		return baseURL + imagePath
	}

	return fmt.Sprintf("%s/%s", baseURL, imagePath)
}

// GetRSSURL 获取RSS订阅链接
func (s *RSSService) GetRSSURL(query *dto.RSSQuery, baseURL string) string {
	rssURL, _ := url.Parse(fmt.Sprintf("%s/api/rss", baseURL))
	
	params := url.Values{}
	if query.Limit > 0 {
		params.Add("limit", fmt.Sprintf("%d", query.Limit))
	}
	if query.CategoryID > 0 {
		params.Add("category_id", fmt.Sprintf("%d", query.CategoryID))
	}
	if query.TagID > 0 {
		params.Add("tag_id", fmt.Sprintf("%d", query.TagID))
	}

	rssURL.RawQuery = params.Encode()
	return rssURL.String()
} 