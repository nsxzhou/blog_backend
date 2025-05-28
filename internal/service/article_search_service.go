package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/nsxzhou1114/blog-api/internal/database"
	"github.com/nsxzhou1114/blog-api/internal/dto"
	"github.com/nsxzhou1114/blog-api/internal/logger"
	"github.com/nsxzhou1114/blog-api/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ArticleSearchService 文章搜索服务
type ArticleSearchService struct {
	db       *gorm.DB
	esClient *elasticsearch.Client
	log      *zap.SugaredLogger
}

// NewArticleSearchService 创建文章搜索服务实例
func NewArticleSearchService() *ArticleSearchService {
	return &ArticleSearchService{
		db:       database.GetDB(),
		esClient: database.GetES(),
		log: logger.GetSugaredLogger(),
	}
}

// Search 搜索文章
func (s *ArticleSearchService) Search(req *dto.ArticleSearchRequest) (*dto.ArticleSearchResponse, error) {
	ctx := context.Background()

	// 构建ES查询
	var buf bytes.Buffer
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"multi_match": map[string]interface{}{
							"query":  req.Keyword,
							"fields": []string{"title^3", "content^2", "summary^2", "tags"},
							"type":   "best_fields",
						},
					},
					{
						"term": map[string]interface{}{
							"status": "published",
						},
					},
				},
				"filter": []map[string]interface{}{
					{
						"term": map[string]interface{}{
							"access_type": "public",
						},
					},
				},
			},
		},
		"highlight": map[string]interface{}{
			"fields": map[string]interface{}{
				"title":   map[string]interface{}{},
				"content": map[string]interface{}{},
				"summary": map[string]interface{}{},
			},
			"pre_tags":            []string{"<em>"},
			"post_tags":           []string{"</em>"},
			"fragment_size":       150,
			"number_of_fragments": 3,
		},
		"from": (req.Page - 1) * req.PageSize,
		"size": req.PageSize,
		"sort": []map[string]interface{}{
			{"_score": map[string]interface{}{"order": "desc"}},
			{"published_at": map[string]interface{}{"order": "desc"}},
		},
	}

	// 添加可选过滤条件
	if req.CategoryID > 0 {
		filter := map[string]interface{}{
			"term": map[string]interface{}{
				"category_id": req.CategoryID,
			},
		}
		query["query"].(map[string]interface{})["bool"].(map[string]interface{})["filter"] = append(
			query["query"].(map[string]interface{})["bool"].(map[string]interface{})["filter"].([]map[string]interface{}),
			filter,
		)
	}

	if req.TagID > 0 {
		// 先获取标签名称
		var tag model.Tag
		if err := s.db.Select("name").First(&tag, req.TagID).Error; err != nil {
			return nil, err
		}

		filter := map[string]interface{}{
			"term": map[string]interface{}{
				"tags": tag.Name,
			},
		}
		query["query"].(map[string]interface{})["bool"].(map[string]interface{})["filter"] = append(
			query["query"].(map[string]interface{})["bool"].(map[string]interface{})["filter"].([]map[string]interface{}),
			filter,
		)
	}

	if req.AuthorID > 0 {
		filter := map[string]interface{}{
			"term": map[string]interface{}{
				"author_id": req.AuthorID,
			},
		}
		query["query"].(map[string]interface{})["bool"].(map[string]interface{})["filter"] = append(
			query["query"].(map[string]interface{})["bool"].(map[string]interface{})["filter"].([]map[string]interface{}),
			filter,
		)
	}

	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, err
	}

	// 执行搜索
	res, err := s.esClient.Search(
		s.esClient.Search.WithContext(ctx),
		s.esClient.Search.WithIndex("articles"),
		s.esClient.Search.WithBody(&buf),
		s.esClient.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("ES搜索错误: %s", res.String())
	}

	// 解析搜索结果
	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	// 提取总数
	total := int64(result["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))

	// 提取文章ID列表
	hits := result["hits"].(map[string]interface{})["hits"].([]interface{})
	articleIDs := make([]uint, 0, len(hits))
	highlightMap := make(map[uint]map[string][]string)

	for _, hit := range hits {
		hitMap := hit.(map[string]interface{})
		source := hitMap["_source"].(map[string]interface{})
		articleID := uint(source["article_id"].(float64))
		articleIDs = append(articleIDs, articleID)

		// 提取高亮信息
		if highlight, exists := hitMap["highlight"]; exists {
			highlightFields := highlight.(map[string]interface{})
			articleHighlight := make(map[string][]string)

			// 处理标题高亮
			if titleHighlight, exists := highlightFields["title"]; exists {
				titles := make([]string, 0)
				for _, title := range titleHighlight.([]interface{}) {
					titles = append(titles, title.(string))
				}
				articleHighlight["title"] = titles
			}

			// 处理内容高亮
			if contentHighlight, exists := highlightFields["content"]; exists {
				contents := make([]string, 0)
				for _, content := range contentHighlight.([]interface{}) {
					contents = append(contents, content.(string))
				}
				articleHighlight["content"] = contents
			}

			// 处理摘要高亮
			if summaryHighlight, exists := highlightFields["summary"]; exists {
				summaries := make([]string, 0)
				for _, summary := range summaryHighlight.([]interface{}) {
					summaries = append(summaries, summary.(string))
				}
				articleHighlight["summary"] = summaries
			}

			highlightMap[articleID] = articleHighlight
		}
	}

	// 如果没有找到文章，返回空结果
	if len(articleIDs) == 0 {
		return &dto.ArticleSearchResponse{
			Total: 0,
			Items: []dto.ArticleListItem{},
		}, nil
	}

	// 从MySQL获取完整的文章信息
	var articles []model.Article
	if err := s.db.Preload("Author").Preload("Category").Preload("Tags").
		Where("id IN ?", articleIDs).
		Find(&articles).Error; err != nil {
		return nil, err
	}

	// 按照搜索结果的顺序排序文章
	sortedArticles := make([]model.Article, len(articleIDs))
	articleMap := make(map[uint]model.Article)
	for _, article := range articles {
		articleMap[article.ID] = article
	}

	for i, id := range articleIDs {
		if article, exists := articleMap[id]; exists {
			sortedArticles[i] = article
		}
	}

	// 转换为响应格式
	items := make([]dto.ArticleListItem, 0, len(sortedArticles))
	for _, article := range sortedArticles {
		// 构建标签列表
		tags := make([]dto.TagInfo, 0, len(article.Tags))
		for _, tag := range article.Tags {
			tags = append(tags, dto.TagInfo{
				ID:   tag.ID,
				Name: tag.Name,
			})
		}

		// 使用高亮的摘要（如果有）
		summary := article.Summary
		if highlight, exists := highlightMap[article.ID]; exists {
			if contentHighlights, exists := highlight["content"]; exists && len(contentHighlights) > 0 {
				// 使用内容高亮作为摘要
				summary = strings.Join(contentHighlights, "...")
			} else if summaryHighlights, exists := highlight["summary"]; exists && len(summaryHighlights) > 0 {
				// 使用摘要高亮
				summary = strings.Join(summaryHighlights, "...")
			}
		}

		var publishedAt time.Time
		if article.PublishedAt != nil {
			publishedAt = *article.PublishedAt
		}

		items = append(items, dto.ArticleListItem{
			ID:            article.ID,
			Title:         article.Title,
			Summary:       summary,
			CategoryID:    article.CategoryID,
			CategoryName:  article.Category.Name,
			AuthorID:      article.AuthorID,
			AuthorName:    article.Author.Nickname,
			CoverImage:    article.CoverImage,
			ViewCount:     article.ViewCount,
			LikeCount:     article.LikeCount,
			CommentCount:  article.CommentCount,
			FavoriteCount: article.FavoriteCount,
			WordCount:     article.WordCount,
			Status:        article.Status,
			AccessType:    article.AccessType,
			IsTop:         article.IsTop,
			IsOriginal:    article.IsOriginal,
			Tags:          tags,
			CreatedAt:     article.CreatedAt,
			UpdatedAt:     article.UpdatedAt,
			PublishedAt:   publishedAt,
		})
	}

	return &dto.ArticleSearchResponse{
		Total: total,
		Items: items,
	}, nil
}

// SearchArticlesByTag 根据标签搜索文章
func (s *ArticleSearchService) SearchArticlesByTag(tagID uint, page, pageSize int) (*dto.ArticleListResponse, error) {
	var tag model.Tag
	if err := s.db.First(&tag, tagID).Error; err != nil {
		return nil, err
	}

	var articles []model.Article
	var total int64

	query := s.db.Model(&model.Article{}).
		Joins("JOIN article_tags ON articles.id = article_tags.article_id").
		Where("article_tags.tag_id = ? AND articles.status = 'published' AND articles.access_type = 'public'", tagID)

	// 获取总数
	query.Count(&total)

	// 执行分页查询
	err := query.
		Preload("Author").
		Preload("Category").
		Preload("Tags").
		Order("is_top DESC, published_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&articles).Error

	if err != nil {
		return nil, err
	}

	// 转换为响应格式
	items := make([]dto.ArticleListItem, 0, len(articles))
	for _, article := range articles {
		tags := make([]dto.TagInfo, 0, len(article.Tags))
		for _, tag := range article.Tags {
			tags = append(tags, dto.TagInfo{
				ID:   tag.ID,
				Name: tag.Name,
			})
		}

		var publishedAt time.Time
		if article.PublishedAt != nil {
			publishedAt = *article.PublishedAt
		}

		items = append(items, dto.ArticleListItem{
			ID:            article.ID,
			Title:         article.Title,
			Summary:       article.Summary,
			CategoryID:    article.CategoryID,
			CategoryName:  article.Category.Name,
			AuthorID:      article.AuthorID,
			AuthorName:    article.Author.Nickname,
			CoverImage:    article.CoverImage,
			ViewCount:     article.ViewCount,
			LikeCount:     article.LikeCount,
			CommentCount:  article.CommentCount,
			FavoriteCount: article.FavoriteCount,
			WordCount:     article.WordCount,
			Status:        article.Status,
			AccessType:    article.AccessType,
			IsTop:         article.IsTop,
			IsOriginal:    article.IsOriginal,
			Tags:          tags,
			CreatedAt:     article.CreatedAt,
			UpdatedAt:     article.UpdatedAt,
			PublishedAt:   publishedAt,
		})
	}

	return &dto.ArticleListResponse{
		Total: total,
		Items: items,
	}, nil
}

// SearchArticlesByCategory 根据分类搜索文章
func (s *ArticleSearchService) SearchArticlesByCategory(categoryID uint, page, pageSize int) (*dto.ArticleListResponse, error) {
	var category model.Category
	if err := s.db.First(&category, categoryID).Error; err != nil {
		return nil, err
	}

	var articles []model.Article
	var total int64

	query := s.db.Model(&model.Article{}).
		Where("category_id = ? AND status = 'published' AND access_type = 'public'", categoryID)

	// 获取总数
	query.Count(&total)

	// 执行分页查询
	err := query.
		Preload("Author").
		Preload("Category").
		Preload("Tags").
		Order("is_top DESC, published_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&articles).Error

	if err != nil {
		return nil, err
	}

	// 转换为响应格式
	items := make([]dto.ArticleListItem, 0, len(articles))
	for _, article := range articles {
		tags := make([]dto.TagInfo, 0, len(article.Tags))
		for _, tag := range article.Tags {
			tags = append(tags, dto.TagInfo{
				ID:   tag.ID,
				Name: tag.Name,
			})
		}

		var publishedAt time.Time
		if article.PublishedAt != nil {
			publishedAt = *article.PublishedAt
		}

		items = append(items, dto.ArticleListItem{
			ID:            article.ID,
			Title:         article.Title,
			Summary:       article.Summary,
			CategoryID:    article.CategoryID,
			CategoryName:  article.Category.Name,
			AuthorID:      article.AuthorID,
			AuthorName:    article.Author.Nickname,
			CoverImage:    article.CoverImage,
			ViewCount:     article.ViewCount,
			LikeCount:     article.LikeCount,
			CommentCount:  article.CommentCount,
			FavoriteCount: article.FavoriteCount,
			WordCount:     article.WordCount,
			Status:        article.Status,
			AccessType:    article.AccessType,
			IsTop:         article.IsTop,
			IsOriginal:    article.IsOriginal,
			Tags:          tags,
			CreatedAt:     article.CreatedAt,
			UpdatedAt:     article.UpdatedAt,
			PublishedAt:   publishedAt,
		})
	}

	return &dto.ArticleListResponse{
		Total: total,
		Items: items,
	}, nil
}

// GetHotArticles 获取热门文章
func (s *ArticleSearchService) GetHotArticles(limit int) ([]dto.ArticleListItem, error) {
	var articles []model.Article

	err := s.db.
		Where("status = 'published' AND access_type = 'public'").
		Order("view_count DESC, like_count DESC, comment_count DESC").
		Limit(limit).
		Preload("Author").
		Preload("Category").
		Preload("Tags").
		Find(&articles).Error

	if err != nil {
		return nil, err
	}

	// 转换为响应格式
	items := make([]dto.ArticleListItem, 0, len(articles))
	for _, article := range articles {
		tags := make([]dto.TagInfo, 0, len(article.Tags))
		for _, tag := range article.Tags {
			tags = append(tags, dto.TagInfo{
				ID:   tag.ID,
				Name: tag.Name,
			})
		}

		var publishedAt time.Time
		if article.PublishedAt != nil {
			publishedAt = *article.PublishedAt
		}

		items = append(items, dto.ArticleListItem{
			ID:            article.ID,
			Title:         article.Title,
			Summary:       article.Summary,
			CategoryID:    article.CategoryID,
			CategoryName:  article.Category.Name,
			AuthorID:      article.AuthorID,
			AuthorName:    article.Author.Nickname,
			CoverImage:    article.CoverImage,
			ViewCount:     article.ViewCount,
			LikeCount:     article.LikeCount,
			CommentCount:  article.CommentCount,
			FavoriteCount: article.FavoriteCount,
			WordCount:     article.WordCount,
			Status:        article.Status,
			AccessType:    article.AccessType,
			IsTop:         article.IsTop,
			IsOriginal:    article.IsOriginal,
			Tags:          tags,
			CreatedAt:     article.CreatedAt,
			UpdatedAt:     article.UpdatedAt,
			PublishedAt:   publishedAt,
		})
	}

	return items, nil
}

// GetLatestArticles 获取最新文章
func (s *ArticleSearchService) GetLatestArticles(limit int) ([]dto.ArticleListItem, error) {
	var articles []model.Article

	err := s.db.
		Where("status = 'published' AND access_type = 'public'").
		Order("published_at DESC").
		Limit(limit).
		Preload("Author").
		Preload("Category").
		Preload("Tags").
		Find(&articles).Error

	if err != nil {
		return nil, err
	}

	// 转换为响应格式
	items := make([]dto.ArticleListItem, 0, len(articles))
	for _, article := range articles {
		tags := make([]dto.TagInfo, 0, len(article.Tags))
		for _, tag := range article.Tags {
			tags = append(tags, dto.TagInfo{
				ID:   tag.ID,
				Name: tag.Name,
			})
		}

		var publishedAt time.Time
		if article.PublishedAt != nil {
			publishedAt = *article.PublishedAt
		}

		items = append(items, dto.ArticleListItem{
			ID:            article.ID,
			Title:         article.Title,
			Summary:       article.Summary,
			CategoryID:    article.CategoryID,
			CategoryName:  article.Category.Name,
			AuthorID:      article.AuthorID,
			AuthorName:    article.Author.Nickname,
			CoverImage:    article.CoverImage,
			ViewCount:     article.ViewCount,
			LikeCount:     article.LikeCount,
			CommentCount:  article.CommentCount,
			FavoriteCount: article.FavoriteCount,
			WordCount:     article.WordCount,
			Status:        article.Status,
			AccessType:    article.AccessType,
			IsTop:         article.IsTop,
			IsOriginal:    article.IsOriginal,
			Tags:          tags,
			CreatedAt:     article.CreatedAt,
			UpdatedAt:     article.UpdatedAt,
			PublishedAt:   publishedAt,
		})
	}

	return items, nil
}

// CreateESIndex 创建ES索引
func (s *ArticleSearchService) CreateESIndex() error {
	esArticle := model.ESArticle{}
	indexName := esArticle.ESIndexName()
	mapping := esArticle.ESMapping()

	ctx := context.Background()

	// 检查索引是否存在
	res, err := s.esClient.Indices.Exists([]string{indexName})
	if err != nil {
		return err
	}

	// 如果索引已存在，先删除
	if res.StatusCode == 200 {
		if _, err := s.esClient.Indices.Delete([]string{indexName}); err != nil {
			return err
		}
	}

	// 创建索引
	createRes, err := s.esClient.Indices.Create(
		indexName,
		s.esClient.Indices.Create.WithContext(ctx),
		s.esClient.Indices.Create.WithBody(strings.NewReader(mapping)),
	)
	if err != nil {
		return err
	}

	if createRes.IsError() {
		return fmt.Errorf("创建索引失败: %s", createRes.String())
	}

	return nil
}

// SyncArticlesToES 同步所有文章到ES
func (s *ArticleSearchService) SyncArticlesToES() error {
	var articles []model.Article

	// 获取所有文章（包括关联数据）
	if err := s.db.Preload("Author").Preload("Category").Preload("Tags").Find(&articles).Error; err != nil {
		return err
	}

	esArticle := model.ESArticle{}
	indexName := esArticle.ESIndexName()

	// 先清空索引中的所有文档
	_, err := s.esClient.DeleteByQuery(
		[]string{indexName},
		strings.NewReader(`{"query": {"match_all": {}}}`),
		s.esClient.DeleteByQuery.WithRefresh(true),
	)
	if err != nil {
		return err
	}

	// 遍历所有文章，逐个添加到ES
	for _, article := range articles {
		if article.ESDocID == "" {
			article.ESDocID = fmt.Sprintf("article_%d", article.ID)
			if err := s.db.Model(&article).Update("es_doc_id", article.ESDocID).Error; err != nil {
				s.log.Warnf("更新文章ES文档ID失败: %v", err)
			}
		}

		// 对于没有内容的文章，设置默认内容
		content := "内容已移至Elasticsearch存储"

		// 创建ES文档
		esDoc := article.ToSearchDocument(content)
		docJSON, err := json.Marshal(esDoc)
		if err != nil {
			s.log.Warnf("序列化文章 %d 失败: %v", article.ID, err)
			continue
		}

		// 添加到ES
		_, err = s.esClient.Index(
			indexName,
			strings.NewReader(string(docJSON)),
			s.esClient.Index.WithDocumentID(esDoc.ID),
			s.esClient.Index.WithRefresh("true"),
		)
		if err != nil {
			s.log.Warnf("添加文章 %d 到ES失败: %v", article.ID, err)
		}
	}

	// 刷新索引，确保数据可搜索
	_, err = s.esClient.Indices.Refresh(
		s.esClient.Indices.Refresh.WithIndex(indexName),
	)

	return err
}
