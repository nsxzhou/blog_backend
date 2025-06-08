package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"strings"
	"sync"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/nsxzhou1114/blog-api/internal/database"
	"github.com/nsxzhou1114/blog-api/internal/dto"
	"github.com/nsxzhou1114/blog-api/internal/logger"
	"github.com/nsxzhou1114/blog-api/internal/model"
	"github.com/nsxzhou1114/blog-api/pkg/cache"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	articleSearchService     *ArticleSearchService
	articleSearchServiceOnce sync.Once
)

// ArticleSearchService 文章搜索服务
type ArticleSearchService struct {
	db           *gorm.DB
	esClient     *elasticsearch.Client
	log          *zap.SugaredLogger
	articleCache *cache.ArticleCacheService
	cacheManager *cache.Manager
}

// NewArticleSearchService 创建文章搜索服务实例
func NewArticleSearchService() *ArticleSearchService {
	articleSearchServiceOnce.Do(func() {
		articleSearchService = &ArticleSearchService{
			db:           database.GetDB(),
			esClient:     database.GetES(),
			log:          logger.GetSugaredLogger(),
			cacheManager: cache.GetManager(),
		}

		articleSearchService.initializeCache()
	})
	return articleSearchService
}

// initializeCache 初始化缓存
func (s *ArticleSearchService) initializeCache() {
	if !s.cacheManager.IsInitialized() {
		redisClient := database.GetRedis()
		if err := s.cacheManager.Initialize(redisClient); err != nil {
			s.log.Errorf("初始化缓存失败: %v", err)
		}
	}
	s.articleCache = s.cacheManager.GetArticleCache()
}

// SearchArticlesByTag 根据标签搜索文章
func (s *ArticleSearchService) SearchArticlesByTag(tagID uint, page, pageSize int) (*dto.ArticleListResponse, error) {
	return s.searchWithCache(
		func(ctx context.Context) (*dto.ArticleListResponse, error) {
			var cachedResponse dto.ArticleListResponse
			err := s.articleCache.GetArticleTagList(ctx, tagID, page, pageSize, &cachedResponse)
			return &cachedResponse, err
		},
		func() (*dto.ArticleListResponse, error) {
			return s.queryArticlesByTag(tagID, page, pageSize)
		},
		func(ctx context.Context, response *dto.ArticleListResponse) error {
			return s.articleCache.SetArticleTagList(ctx, tagID, page, pageSize, response)
		},
		fmt.Sprintf("标签文章列表: tagID=%d, page=%d, pageSize=%d", tagID, page, pageSize),
	)
}

// SearchArticlesByCategory 根据分类搜索文章
func (s *ArticleSearchService) SearchArticlesByCategory(categoryID uint, page, pageSize int) (*dto.ArticleListResponse, error) {
	return s.searchWithCache(
		func(ctx context.Context) (*dto.ArticleListResponse, error) {
			var cachedResponse dto.ArticleListResponse
			err := s.articleCache.GetArticleCategoryList(ctx, categoryID, page, pageSize, &cachedResponse)
			return &cachedResponse, err
		},
		func() (*dto.ArticleListResponse, error) {
			return s.queryArticlesByCategory(categoryID, page, pageSize)
		},
		func(ctx context.Context, response *dto.ArticleListResponse) error {
			return s.articleCache.SetArticleCategoryList(ctx, categoryID, page, pageSize, response)
		},
		fmt.Sprintf("分类文章列表: categoryID=%d, page=%d, pageSize=%d", categoryID, page, pageSize),
	)
}

// GetArticleList 通用文章列表获取
func (s *ArticleSearchService) GetArticleList(req *dto.ArticleListRequest) (*dto.ArticleListResponse, error) {
	if req.Keyword != "" {
		return s.searchWithElasticsearch(req)
	}
	return s.queryWithMySQL(req)
}

// GetHotArticles 获取热门文章
func (s *ArticleSearchService) GetHotArticles(page, pageSize int) (*dto.ArticleListResponse, error) {
	return s.searchWithCache(
		func(ctx context.Context) (*dto.ArticleListResponse, error) {
			var cachedResponse dto.ArticleListResponse
			err := s.articleCache.GetHotArticles(ctx, page, pageSize, &cachedResponse)
			return &cachedResponse, err
		},
		func() (*dto.ArticleListResponse, error) {
			return s.queryHotArticles(page, pageSize)
		},
		func(ctx context.Context, response *dto.ArticleListResponse) error {
			return s.articleCache.SetHotArticles(ctx, page, pageSize, response)
		},
		fmt.Sprintf("热门文章: page=%d, pageSize=%d", page, pageSize),
	)
}

// 通用缓存处理方法

// searchWithCache 通用缓存搜索模板
func (s *ArticleSearchService) searchWithCache(
	getCacheFunc func(context.Context) (*dto.ArticleListResponse, error),
	queryFunc func() (*dto.ArticleListResponse, error),
	setCacheFunc func(context.Context, *dto.ArticleListResponse) error,
	logPrefix string,
) (*dto.ArticleListResponse, error) {
	ctx := context.Background()

	// 尝试从缓存获取
	if s.articleCache != nil {
		if cachedResponse, err := getCacheFunc(ctx); err == nil {
			s.log.Infof("%s缓存命中", logPrefix)
			return cachedResponse, nil
		}
	}

	// 缓存未命中，从数据库查询
	response, err := queryFunc()
	if err != nil {
		return nil, err
	}

	// 异步设置缓存
	if s.articleCache != nil {
		go func() {
			if err := setCacheFunc(context.Background(), response); err != nil {
				s.log.Warnf("设置%s缓存失败: %v", logPrefix, err)
			} else {
				s.log.Infof("%s缓存设置成功", logPrefix)
			}
		}()
	}

	return response, nil
}

// 具体查询方法

// queryArticlesByTag 从数据库查询标签文章
func (s *ArticleSearchService) queryArticlesByTag(tagID uint, page, pageSize int) (*dto.ArticleListResponse, error) {
	var tag model.Tag
	if err := s.db.First(&tag, tagID).Error; err != nil {
		return nil, err
	}

	var articles []model.Article
	var total int64

	query := s.db.Model(&model.Article{}).
		Joins("JOIN article_tags ON articles.id = article_tags.article_id").
		Where("article_tags.tag_id = ? AND articles.status = 'published' AND articles.access_type = 'public'", tagID)

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

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

	return &dto.ArticleListResponse{
		Total: total,
		List:  s.convertToArticleListItems(articles),
	}, nil
}

// queryArticlesByCategory 从数据库查询分类文章
func (s *ArticleSearchService) queryArticlesByCategory(categoryID uint, page, pageSize int) (*dto.ArticleListResponse, error) {
	var category model.Category
	if err := s.db.First(&category, categoryID).Error; err != nil {
		return nil, err
	}

	var articles []model.Article
	var total int64

	query := s.db.Model(&model.Article{}).
		Where("category_id = ? AND status = 'published' AND access_type = 'public'", categoryID)

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

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

	return &dto.ArticleListResponse{
		Total: total,
		List:  s.convertToArticleListItems(articles),
	}, nil
}

// queryHotArticles 从数据库查询热门文章
func (s *ArticleSearchService) queryHotArticles(page, pageSize int) (*dto.ArticleListResponse, error) {
	var articles []model.Article
	var total int64

	query := s.db.Model(&model.Article{}).
		Where("status = 'published' AND access_type = 'public'")

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	err := query.
		Preload("Author").
		Preload("Category").
		Preload("Tags").
		Order("view_count DESC, like_count DESC, comment_count DESC, published_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&articles).Error

	if err != nil {
		return nil, err
	}

	return &dto.ArticleListResponse{
		Total: total,
		List:  s.convertToArticleListItems(articles),
	}, nil
}

// searchWithElasticsearch 使用Elasticsearch搜索
func (s *ArticleSearchService) searchWithElasticsearch(req *dto.ArticleListRequest) (*dto.ArticleListResponse, error) {
	ctx := context.Background()

	// 尝试从缓存获取搜索结果
	if s.articleCache != nil {
		var cachedResponse dto.ArticleListResponse
		err := s.articleCache.GetSearchResults(ctx, req.Keyword, req.Page, req.PageSize, &cachedResponse)
		if err == nil {
			s.log.Infof("搜索结果缓存命中: keyword=%s, page=%d, pageSize=%d", req.Keyword, req.Page, req.PageSize)
			return &cachedResponse, nil
		}
	}

	query := s.buildESQuery(req)
	response, err := s.executeESSearch(query)
	if err != nil {
		return nil, err
	}

	// 异步设置搜索结果缓存
	if s.articleCache != nil {
		go func() {
			if err := s.articleCache.SetSearchResults(context.Background(), req.Keyword, req.Page, req.PageSize, response); err != nil {
				s.log.Warnf("设置搜索结果缓存失败: %v", err)
			} else {
				s.log.Infof("搜索结果缓存设置成功: keyword=%s, page=%d, pageSize=%d", req.Keyword, req.Page, req.PageSize)
			}
		}()
	}

	return response, nil
}

// queryWithMySQL 使用MySQL查询
func (s *ArticleSearchService) queryWithMySQL(req *dto.ArticleListRequest) (*dto.ArticleListResponse, error) {
	ctx := context.Background()

	// 判断是否为基础查询，尝试从缓存获取
	if isBasicQuery := s.isBasicQuery(req); isBasicQuery && s.articleCache != nil {
		var cachedResponse dto.ArticleListResponse
		err := s.articleCache.GetArticleList(ctx, req.Page, req.PageSize, &cachedResponse)
		if err == nil {
			s.log.Infof("基础文章列表缓存命中: page=%d, pageSize=%d", req.Page, req.PageSize)
			return &cachedResponse, nil
		}
	}

	var articles []model.Article
	var total int64

	query := s.buildMySQLQuery(req)
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	query = s.applyMySQLPaginationAndOrder(query, req)
	query = query.Preload("Author").Preload("Category").Preload("Tags")

	if err := query.Find(&articles).Error; err != nil {
		return nil, err
	}

	response := &dto.ArticleListResponse{
		Total: total,
		List:  s.convertToArticleListItems(articles),
	}

	// 如果是基础查询，异步设置缓存
	if s.isBasicQuery(req) && s.articleCache != nil {
		go func() {
			if err := s.articleCache.SetArticleList(context.Background(), req.Page, req.PageSize, response); err != nil {
				s.log.Warnf("设置基础文章列表缓存失败: %v", err)
			} else {
				s.log.Infof("基础文章列表缓存设置成功: page=%d, pageSize=%d", req.Page, req.PageSize)
			}
		}()
	}

	return response, nil
}

// 查询构建方法

// buildESQuery 构建ES查询
func (s *ArticleSearchService) buildESQuery(req *dto.ArticleListRequest) map[string]interface{} {
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
				},
				"filter": s.buildESFilters(req),
			},
		},
		"highlight": map[string]interface{}{
			"fields": map[string]interface{}{
				"title": map[string]interface{}{
					"pre_tags":            []string{"<mark>"},
					"post_tags":           []string{"</mark>"},
					"number_of_fragments": 0, // 返回整个标题
				},
				"content": map[string]interface{}{
					"pre_tags":            []string{"<mark>"},
					"post_tags":           []string{"</mark>"},
					"fragment_size":       100, // 减小片段大小
					"number_of_fragments": 3,   // 减少片段数量
					"fragment_offset":     5,   // 减小偏移量
					"no_match_size":       80,  // 如果没有匹配，返回开头80个字符
				},
				"summary": map[string]interface{}{
					"pre_tags":            []string{"<mark>"},
					"post_tags":           []string{"</mark>"},
					"fragment_size":       80, // 减小摘要片段大小
					"number_of_fragments": 2,  // 最多2个摘要片段
				},
			},
			"order": "score",
		},
		"from": (req.Page - 1) * req.PageSize,
		"size": req.PageSize,
		"sort": s.buildESSort(req.SortBy, req.Order),
	}

	return query
}

// buildESFilters 构建ES过滤条件
func (s *ArticleSearchService) buildESFilters(req *dto.ArticleListRequest) []map[string]interface{} {
	filters := []map[string]interface{}{}

	// 基础过滤条件
	status := req.Status
	if status == "" {
		status = "published"
	}
	filters = append(filters, map[string]interface{}{
		"term": map[string]interface{}{"status": status},
	})

	accessType := req.AccessType
	if accessType == "" {
		accessType = "public"
	}
	filters = append(filters, map[string]interface{}{
		"term": map[string]interface{}{"access_type": accessType},
	})

	// 可选过滤条件
	if req.CategoryID > 0 {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{"category_id": req.CategoryID},
		})
	}

	if req.TagID > 0 {
		var tag model.Tag
		if err := s.db.Select("name").First(&tag, req.TagID).Error; err == nil {
			filters = append(filters, map[string]interface{}{
				"term": map[string]interface{}{"tags": tag.Name},
			})
		}
	}

	if req.AuthorID > 0 {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{"author_id": req.AuthorID},
		})
	}

	if req.IsTop > 0 {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{"is_top": req.IsTop},
		})
	}

	if req.IsOriginal > 0 {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{"is_original": req.IsOriginal},
		})
	}

	// 时间范围过滤
	if req.StartDate != "" || req.EndDate != "" {
		rangeFilter := map[string]interface{}{
			"range": map[string]interface{}{
				"published_at": map[string]interface{}{},
			},
		}
		if req.StartDate != "" {
			rangeFilter["range"].(map[string]interface{})["published_at"].(map[string]interface{})["gte"] = req.StartDate
		}
		if req.EndDate != "" {
			rangeFilter["range"].(map[string]interface{})["published_at"].(map[string]interface{})["lte"] = req.EndDate
		}
		filters = append(filters, rangeFilter)
	}

	return filters
}

// buildMySQLQuery 构建MySQL查询
func (s *ArticleSearchService) buildMySQLQuery(req *dto.ArticleListRequest) *gorm.DB {
	query := s.db.Model(&model.Article{})

	// 基础过滤条件
	if req.Status == "" {
		query = query.Where("status IN ('draft', 'published')")
	} else {
		query = query.Where("status = ?", req.Status)
	}

	if req.AccessType != "" {
		query = query.Where("access_type = ?", req.AccessType)
	} else {
		query = query.Where("access_type IN ('public', 'private', 'password')")
	}

	// 可选过滤条件
	if req.CategoryID > 0 {
		query = query.Where("category_id = ?", req.CategoryID)
	}

	if req.TagID > 0 {
		query = query.Joins("JOIN article_tags ON articles.id = article_tags.article_id").
			Where("article_tags.tag_id = ?", req.TagID)
	}

	if req.AuthorID > 0 {
		query = query.Where("author_id = ?", req.AuthorID)
	}

	if req.IsTop > 0 && req.IsTop != 2 {
		query = query.Where("is_top = ?", req.IsTop)
	}

	if req.IsOriginal > 0 && req.IsOriginal != 2 {
		query = query.Where("is_original = ?", req.IsOriginal)
	}

	// 时间范围过滤
	if req.StartDate != "" {
		query = query.Where("published_at >= ?", req.StartDate)
	}
	if req.EndDate != "" {
		query = query.Where("published_at <= ?", req.EndDate)
	}

	return query
}

// applyMySQLPaginationAndOrder 应用MySQL分页和排序
func (s *ArticleSearchService) applyMySQLPaginationAndOrder(query *gorm.DB, req *dto.ArticleListRequest) *gorm.DB {
	orderBy := s.buildMySQLSort(req.SortBy, req.Order)
	return query.Order(orderBy).Offset((req.Page - 1) * req.PageSize).Limit(req.PageSize)
}

// isBasicQuery 判断是否为基础查询
func (s *ArticleSearchService) isBasicQuery(req *dto.ArticleListRequest) bool {
	return req.CategoryID == 0 && req.TagID == 0 && req.AuthorID == 0 &&
		req.Keyword == "" && req.Status == "" && req.AccessType == "" &&
		req.IsTop == 0 && req.IsOriginal == 0 && req.StartDate == "" && req.EndDate == ""
}

// 排序构建方法

// buildESSort 构建ES排序条件
func (s *ArticleSearchService) buildESSort(sortBy, order string) []map[string]interface{} {
	if order == "" {
		order = "desc"
	}

	switch sortBy {
	case "hot":
		return []map[string]interface{}{
			{"view_count": map[string]interface{}{"order": order}},
			{"like_count": map[string]interface{}{"order": order}},
			{"comment_count": map[string]interface{}{"order": order}},
			{"published_at": map[string]interface{}{"order": "desc"}},
		}
	case "latest":
		return []map[string]interface{}{
			{"published_at": map[string]interface{}{"order": order}},
		}
	case "score":
		return []map[string]interface{}{
			{"_score": map[string]interface{}{"order": order}},
			{"published_at": map[string]interface{}{"order": "desc"}},
		}
	case "view_count", "like_count", "comment_count", "created_at", "published_at":
		return []map[string]interface{}{
			{sortBy: map[string]interface{}{"order": order}},
		}
	default:
		return []map[string]interface{}{
			{"_score": map[string]interface{}{"order": "desc"}},
			{"published_at": map[string]interface{}{"order": "desc"}},
		}
	}
}

// buildMySQLSort 构建MySQL排序条件
func (s *ArticleSearchService) buildMySQLSort(sortBy, order string) string {
	if order == "" {
		order = "desc"
	}

	switch sortBy {
	case "hot":
		return fmt.Sprintf("view_count %s, like_count %s, comment_count %s, published_at desc", order, order, order)
	case "latest":
		return fmt.Sprintf("published_at %s", order)
	case "view_count", "like_count", "comment_count", "created_at", "published_at":
		return fmt.Sprintf("%s %s", sortBy, order)
	default:
		return "is_top desc, published_at desc"
	}
}

// ES操作方法

// executeESSearch 执行ES搜索
func (s *ArticleSearchService) executeESSearch(query map[string]interface{}) (*dto.ArticleListResponse, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, err
	}

	res, err := s.esClient.Search(
		s.esClient.Search.WithContext(context.Background()),
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

	return s.processESResponse(res)
}

// processESResponse 处理ES响应
func (s *ArticleSearchService) processESResponse(res *esapi.Response) (*dto.ArticleListResponse, error) {
	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	total := int64(result["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))

	hits := result["hits"].(map[string]interface{})["hits"].([]interface{})
	articleIDs := make([]uint, 0, len(hits))
	highlightMap := make(map[uint]map[string][]string)

	for _, hit := range hits {
		hitMap := hit.(map[string]interface{})
		source := hitMap["_source"].(map[string]interface{})
		articleID := uint(source["article_id"].(float64))
		articleIDs = append(articleIDs, articleID)

		if highlight, exists := hitMap["highlight"]; exists {
			highlightFields := highlight.(map[string]interface{})
			articleHighlight := make(map[string][]string)

			for field, highlights := range highlightFields {
				fieldHighlights := make([]string, 0)
				for _, h := range highlights.([]interface{}) {
					fieldHighlights = append(fieldHighlights, h.(string))
				}
				articleHighlight[field] = fieldHighlights
			}

			highlightMap[articleID] = articleHighlight
		}
	}

	if len(articleIDs) == 0 {
		return &dto.ArticleListResponse{
			Total: 0,
			List:  []dto.ArticleListItem{},
		}, nil
	}

	var articles []model.Article
	if err := s.db.Preload("Author").Preload("Category").Preload("Tags").
		Where("id IN ?", articleIDs).
		Find(&articles).Error; err != nil {
		return nil, err
	}

	// 按搜索结果顺序排序
	sortedArticles := s.sortArticlesByIDs(articles, articleIDs)
	items := s.convertToArticleListItemsWithHighlight(sortedArticles, highlightMap)

	return &dto.ArticleListResponse{
		Total: total,
		List:  items,
	}, nil
}

// 转换方法

// convertToArticleListItems 转换为文章列表项
func (s *ArticleSearchService) convertToArticleListItems(articles []model.Article) []dto.ArticleListItem {
	items := make([]dto.ArticleListItem, 0, len(articles))
	for _, article := range articles {
		items = append(items, s.convertToArticleListItem(article, nil))
	}
	return items
}

// convertToArticleListItemsWithHighlight 转换为带高亮的文章列表项
func (s *ArticleSearchService) convertToArticleListItemsWithHighlight(articles []model.Article, highlightMap map[uint]map[string][]string) []dto.ArticleListItem {
	items := make([]dto.ArticleListItem, 0, len(articles))
	for _, article := range articles {
		highlight := highlightMap[article.ID]
		items = append(items, s.convertToArticleListItem(article, highlight))
	}
	return items
}

// convertToArticleListItem 转换单个文章列表项
func (s *ArticleSearchService) convertToArticleListItem(article model.Article, highlight map[string][]string) dto.ArticleListItem {
	tags := make([]dto.TagInfo, 0, len(article.Tags))
	for _, tag := range article.Tags {
		tags = append(tags, dto.TagInfo{
			ID:   tag.ID,
			Name: tag.Name,
		})
	}

	// 处理高亮摘要
	summary := article.Summary
	if highlight != nil {
		if contentHighlights, exists := highlight["content"]; exists && len(contentHighlights) > 0 {
			summary = strings.Join(contentHighlights, "...")
		} else if summaryHighlights, exists := highlight["summary"]; exists && len(summaryHighlights) > 0 {
			summary = strings.Join(summaryHighlights, "...")
		}
	}

	var publishedAtStr string
	if article.PublishedAt != nil {
		publishedAtStr = article.PublishedAt.Format("2006-01-02 15:04:05")
	}

	return dto.ArticleListItem{
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
		CreatedAt:     article.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:     article.UpdatedAt.Format("2006-01-02 15:04:05"),
		PublishedAt:   publishedAtStr,
	}
}

// sortArticlesByIDs 按ID顺序排序文章
func (s *ArticleSearchService) sortArticlesByIDs(articles []model.Article, articleIDs []uint) []model.Article {
	articleMap := make(map[uint]model.Article)
	for _, article := range articles {
		articleMap[article.ID] = article
	}

	sortedArticles := make([]model.Article, 0, len(articleIDs))
	for _, id := range articleIDs {
		if article, exists := articleMap[id]; exists {
			sortedArticles = append(sortedArticles, article)
		}
	}
	return sortedArticles
}

// ES索引管理方法

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

	if err := s.db.Preload("Author").Preload("Category").Preload("Tags").Find(&articles).Error; err != nil {
		return err
	}

	esArticle := model.ESArticle{}
	indexName := esArticle.ESIndexName()

	// 先清空索引
	_, err := s.esClient.DeleteByQuery(
		[]string{indexName},
		strings.NewReader(`{"query": {"match_all": {}}}`),
		s.esClient.DeleteByQuery.WithRefresh(true),
	)
	if err != nil {
		return err
	}

	// 同步所有文章
	for _, article := range articles {
		if article.ESDocID == "" {
			article.ESDocID = fmt.Sprintf("article_%d", article.ID)
			if err := s.db.Model(&article).Update("es_doc_id", article.ESDocID).Error; err != nil {
				s.log.Warnf("更新文章ES文档ID失败: %v", err)
			}
		}

		content := "内容已移至Elasticsearch存储"
		esDoc := article.ToSearchDocument(content)
		docJSON, err := json.Marshal(esDoc)
		if err != nil {
			s.log.Warnf("序列化文章 %d 失败: %v", article.ID, err)
			continue
		}

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

	// 刷新索引
	_, err = s.esClient.Indices.Refresh(
		s.esClient.Indices.Refresh.WithIndex(indexName),
	)

	return err
}

// FullTextSearch 全文搜索，返回内容片段
func (s *ArticleSearchService) FullTextSearch(req *dto.FullTextSearchRequest) (*dto.FullTextSearchResponse, error) {
	ctx := context.Background()

	// 尝试从缓存获取搜索结果
	if s.articleCache != nil {
		var cachedResponse dto.FullTextSearchResponse
		err := s.articleCache.GetFullTextSearchResults(ctx, req.Keyword, req.Page, req.PageSize, &cachedResponse)
		if err == nil {
			s.log.Infof("全文搜索结果缓存命中: keyword=%s, page=%d, pageSize=%d", req.Keyword, req.Page, req.PageSize)
			return &cachedResponse, nil
		}
	}

	query := s.buildFullTextSearchQuery(req)
	response, err := s.executeFullTextSearch(query)
	if err != nil {
		return nil, err
	}

	// 异步设置搜索结果缓存
	if s.articleCache != nil {
		go func() {
			if err := s.articleCache.SetFullTextSearchResults(context.Background(), req.Keyword, req.Page, req.PageSize, response); err != nil {
				s.log.Warnf("设置全文搜索结果缓存失败: %v", err)
			} else {
				s.log.Infof("全文搜索结果缓存设置成功: keyword=%s, page=%d, pageSize=%d", req.Keyword, req.Page, req.PageSize)
			}
		}()
	}

	return response, nil
}

// buildFullTextSearchQuery 构建全文搜索查询
func (s *ArticleSearchService) buildFullTextSearchQuery(req *dto.FullTextSearchRequest) map[string]interface{} {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"multi_match": map[string]interface{}{
							"query":  req.Keyword,
							"fields": []string{"title^3", "content^2", "summary^1.5"},
							"type":   "best_fields",
						},
					},
				},
				"filter": []map[string]interface{}{
					{
						"term": map[string]interface{}{"status": "published"},
					},
					{
						"term": map[string]interface{}{"access_type": "public"},
					},
				},
			},
		},
		"highlight": map[string]interface{}{
			"fields": map[string]interface{}{
				"title": map[string]interface{}{
					"pre_tags":            []string{"<mark>"},
					"post_tags":           []string{"</mark>"},
					"number_of_fragments": 0, // 返回整个标题
				},
				"content": map[string]interface{}{
					"pre_tags":            []string{"<mark>"},
					"post_tags":           []string{"</mark>"},
					"fragment_size":       100, // 减小片段大小
					"number_of_fragments": 3,   // 减少片段数量
					"fragment_offset":     5,   // 减小偏移量
				},
				"summary": map[string]interface{}{
					"pre_tags":            []string{"<mark>"},
					"post_tags":           []string{"</mark>"},
					"fragment_size":       80, // 减小摘要片段大小
					"number_of_fragments": 2,  // 最多2个摘要片段
				},
			},
			"order": "score",
		},
		"from": (req.Page - 1) * req.PageSize,
		"size": req.PageSize,
		"sort": []map[string]interface{}{
			{"_score": map[string]interface{}{"order": "desc"}},
			{"published_at": map[string]interface{}{"order": "desc"}},
		},
		"_source": []string{"article_id", "title", "author_name", "category_name", "published_at"},
	}

	return query
}

// executeFullTextSearch 执行全文搜索
func (s *ArticleSearchService) executeFullTextSearch(query map[string]interface{}) (*dto.FullTextSearchResponse, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, err
	}

	res, err := s.esClient.Search(
		s.esClient.Search.WithContext(context.Background()),
		s.esClient.Search.WithIndex("articles"),
		s.esClient.Search.WithBody(&buf),
		s.esClient.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("ES全文搜索错误: %s", res.String())
	}

	return s.processFullTextSearchResponse(res)
}

// processFullTextSearchResponse 处理全文搜索响应
func (s *ArticleSearchService) processFullTextSearchResponse(res *esapi.Response) (*dto.FullTextSearchResponse, error) {
	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	total := int64(result["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	hits := result["hits"].(map[string]interface{})["hits"].([]interface{})

	items := make([]dto.FullTextSearchItem, 0, len(hits))

	for _, hit := range hits {
		hitMap := hit.(map[string]interface{})
		source := hitMap["_source"].(map[string]interface{})
		score := hitMap["_score"].(float64)

		// 提取基本信息
		articleID := uint(source["article_id"].(float64))
		title := source["title"].(string)
		authorName := source["author_name"].(string)
		categoryName := source["category_name"].(string)
		publishedAt := source["published_at"].(string)

		// 处理高亮信息
		var highlightedTitle string
		var fragments []dto.ContentFragment

		if highlight, exists := hitMap["highlight"]; exists {
			highlightFields := highlight.(map[string]interface{})

			// 处理标题高亮
			if titleHighlights, exists := highlightFields["title"]; exists {
				titleArray := titleHighlights.([]interface{})
				if len(titleArray) > 0 {
					highlightedTitle = titleArray[0].(string)
				}
			}

			// 优先处理内容片段
			if contentHighlights, exists := highlightFields["content"]; exists {
				contentArray := contentHighlights.([]interface{})
				for _, fragment := range contentArray {
					fragmentStr := fragment.(string)
					// 清理内容片段
					if fragmentStr != "" { // 只添加非空片段
						fragments = append(fragments, dto.ContentFragment{
							Content:    fragmentStr,
							Position:   -1,          // 内容片段位置标记为-1
							MatchScore: score * 0.8, // 内容片段权重较高
						})
					}
				}
			}

			// 如果内容片段不足，补充摘要片段
			if len(fragments) < 3 {
				if summaryHighlights, exists := highlightFields["summary"]; exists {
					summaryArray := summaryHighlights.([]interface{})
					for _, fragment := range summaryArray {
						if len(fragments) >= 5 { // 最多5个片段
							break
						}
						fragmentStr := fragment.(string)
						// 清理摘要片段
						if fragmentStr != "" { // 只添加非空片段
							fragments = append(fragments, dto.ContentFragment{
								Content:    fragmentStr,
								Position:   -1,          // 摘要片段位置标记为-1
								MatchScore: score * 0.6, // 摘要片段权重较低
							})
						}
					}
				}
			}
		}

		// 如果没有高亮标题，使用原标题
		if highlightedTitle == "" {
			highlightedTitle = title
		}

		// 格式化发布时间
		if publishedAt != "" {
			if parsedTime, err := time.Parse(time.RFC3339, publishedAt); err == nil {
				publishedAt = parsedTime.Format("2006-01-02 15:04:05")
			}
		}

		item := dto.FullTextSearchItem{
			ArticleID:    articleID,
			Title:        highlightedTitle,
			AuthorName:   authorName,
			CategoryName: categoryName,
			PublishedAt:  publishedAt,
			Fragments:    fragments,
			Score:        score,
		}

		items = append(items, item)
	}

	return &dto.FullTextSearchResponse{
		Total: total,
		List:  items,
	}, nil
}
