package service

import (
	"context"
	"encoding/json"
	"errors"
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
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	articleService     *ArticleService
	articleServiceOnce sync.Once
)

// ArticleService 文章服务
type ArticleService struct {
	db           *gorm.DB
	esClient     *elasticsearch.Client
	log          *zap.SugaredLogger
	articleCache *cache.ArticleCacheService
	cacheManager *cache.Manager
	tagService   *TagService
}

// NewArticleService 创建文章服务实例
func NewArticleService() *ArticleService {
	articleServiceOnce.Do(func() {
		articleService = &ArticleService{
			db:           database.GetDB(),
			esClient:     database.GetES(),
			log:          logger.GetSugaredLogger(),
			cacheManager: cache.GetManager(),
			tagService:   NewTagService(),
		}
	
		articleService.initializeCache()
	})
	return articleService
}

// initializeCache 初始化缓存
func (s *ArticleService) initializeCache() {
	if !s.cacheManager.IsInitialized() {
		redisClient := database.GetRedis()
		if err := s.cacheManager.Initialize(redisClient); err != nil {
			s.log.Errorf("初始化缓存失败: %v", err)
		}
	}
	s.articleCache = s.cacheManager.GetArticleCache()
}

// Create 创建文章
func (s *ArticleService) Create(userID uint, req *dto.ArticleCreateRequest) (*model.Article, error) {
	article := s.buildArticleFromRequest(userID, req)
	
	return s.executeTransaction(func(tx *gorm.DB) (*model.Article, error) {
		if err := tx.Create(article).Error; err != nil {
			return nil, err
		}

		if err := s.handleTags(tx, article, req.TagIDs); err != nil {
			return nil, err
		}

		if err := s.preloadArticleData(tx, article); err != nil {
			return nil, err
		}

		esDocID, err := s.saveArticleToES(article.ToSearchDocument(req.Content))
		if err != nil {
			return nil, err
		}

		article.ESDocID = esDocID
		if err := tx.Save(article).Error; err != nil {
			s.deleteArticleFromES(esDocID)
			return nil, err
		}

		s.handleCacheAsyncCreate(article)
		return article, nil
	})
}

// Update 更新文章
func (s *ArticleService) Update(userID uint, articleID uint, req *dto.ArticleUpdateRequest) (*model.Article, error) {
	var article model.Article
	if err := s.db.First(&article, articleID).Error; err != nil {
		return nil, err
	}

	if err := s.checkPermission(article.AuthorID, userID); err != nil {
		return nil, err
	}

	return s.executeTransaction(func(tx *gorm.DB) (*model.Article, error) {
		updates := s.buildUpdateData(req)
		if len(updates) > 0 {
			if err := tx.Model(&article).Updates(updates).Error; err != nil {
				return nil, err
			}
		}

		if err := s.handleTags(tx, &article, req.TagIDs); err != nil {
			return nil, err
		}

		if err := s.preloadArticleData(tx, &article); err != nil {
			return nil, err
		}

		if err := s.updateArticleContent(tx, &article, req.Content); err != nil {
			return nil, err
		}

		s.handleCacheAsyncUpdate(&article)
		return &article, nil
	})
}

// Delete 删除文章
func (s *ArticleService) Delete(userID uint, articleID uint, role string) error {
	var article model.Article
	if err := s.db.First(&article, articleID).Error; err != nil {
		return err
	}

	if article.AuthorID != userID && role != "admin" {
		return errors.New("没有权限删除此文章")
	}

	return s.executeDeleteTransaction(func(tx *gorm.DB) error {
		// 获取文章的标签列表，用于更新计数
		var tagIDs []uint
		if err := tx.Model(&article).Association("Tags").Find(&tagIDs); err != nil {
			s.log.Warnf("获取文章标签列表失败: %v", err)
		}

		if err := tx.Delete(&article).Error; err != nil {
			return err
		}

		// 更新标签文章计数
		if len(tagIDs) > 0 {
			if err := s.tagService.UpdateMultipleTagArticleCount(tx, tagIDs); err != nil {
				s.log.Warnf("更新标签文章计数失败: %v", err)
			}
		}

		if article.ESDocID != "" {
			if err := s.deleteArticleFromES(article.ESDocID); err != nil {
				s.log.Warnf("从ES删除文章 %s 失败: %v", article.ESDocID, err)
			}
		}

		s.handleCacheAsyncDelete(articleID, article.CategoryID)
		return nil
	})
}

// GetByID 根据ID获取文章基本信息
func (s *ArticleService) GetByID(articleID uint) (*model.Article, error) {
	var article model.Article
	err := s.db.First(&article, articleID).Error
	return &article, err
}

// GetArticleDetail 获取文章详情
func (s *ArticleService) GetArticleDetail(articleID uint, currentUserID uint) (*dto.ArticleDetailResponse, error) {
	ctx := context.Background()
	
	// 尝试从缓存获取
	if cachedResponse := s.tryGetCachedDetail(ctx, articleID, currentUserID); cachedResponse != nil {
		return cachedResponse, nil
	}
	
	// 从数据库获取
	var article model.Article
	if err := s.db.Preload("Author").Preload("Category").Preload("Tags").First(&article, articleID).Error; err != nil {
		return nil, err
	}

	s.incrementViewCount(article.ID)

	content, err := s.getArticleContentFromES(article.ESDocID)
	if err != nil {
		return nil, err
	}

	response := s.buildArticleDetailResponse(&article, content, currentUserID)
	s.setCacheAsync(ctx, articleID, response)
	
	return response, nil
}

// GetUserArticles 获取用户文章列表
func (s *ArticleService) GetUserArticles(userID uint, req *dto.ArticleQueryRequest) (*dto.ArticleListResponse, error) {
	var articles []model.Article
	var total int64

	query := s.db.Model(&model.Article{}).Where("author_id = ?", userID)
	query = s.applyUserArticleFilters(query, req)

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	query = s.applyPaginationAndOrder(query, req.Page, req.PageSize, req.OrderBy, req.Order)
	query = query.Preload("Author").Preload("Category").Preload("Tags")

	if err := query.Find(&articles).Error; err != nil {
		return nil, err
	}

	return &dto.ArticleListResponse{
		Total: total,
		List:  s.convertToArticleListItems(articles),
	}, nil
}

// 辅助方法

// executeTransaction 执行事务
func (s *ArticleService) executeTransaction(fn func(*gorm.DB) (*model.Article, error)) (*model.Article, error) {
	tx := s.db.Begin()
	result, err := fn(tx)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	tx.Commit()
	return result, nil
}

// executeDeleteTransaction 执行删除事务
func (s *ArticleService) executeDeleteTransaction(fn func(*gorm.DB) error) error {
	tx := s.db.Begin()
	err := fn(tx)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

// buildArticleFromRequest 从请求构建文章
func (s *ArticleService) buildArticleFromRequest(userID uint, req *dto.ArticleCreateRequest) *model.Article {
	article := &model.Article{
		Title:      req.Title,
		Summary:    req.Summary,
		Status:     req.Status,
		AuthorID:   userID,
		CategoryID: req.CategoryID,
		CoverImage: req.CoverImage,
		WordCount:  calculateWordCount(req.Content),
		AccessType: req.AccessType,
		Password:   req.Password,
		IsTop:      req.IsTop,
		IsOriginal: req.IsOriginal,
		SourceURL:  req.SourceURL,
		SourceName: req.SourceName,
	}

	if req.Status == "published" {
		now := time.Now()
		article.PublishedAt = &now
	}

	return article
}

// buildUpdateData 构建更新数据
func (s *ArticleService) buildUpdateData(req *dto.ArticleUpdateRequest) map[string]interface{} {
	updates := map[string]interface{}{}
	
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Summary != "" {
		updates["summary"] = req.Summary
	}
	if req.CategoryID != 0 {
		updates["category_id"] = req.CategoryID
	}
	if req.CoverImage != "" {
		updates["cover_image"] = req.CoverImage
	}
	if req.Status != "" {
		updates["status"] = req.Status
		if req.Status == "published" {
			updates["published_at"] = time.Now()
		}
	}
	if req.AccessType != "" {
		updates["access_type"] = req.AccessType
	}
	if req.Password != "" {
		updates["password"] = req.Password
	}
	if req.IsTop >= 0 {
		updates["is_top"] = req.IsTop
	}
	if req.IsOriginal >= 0 {
		updates["is_original"] = req.IsOriginal
	}
	if req.SourceURL != "" {
		updates["source_url"] = req.SourceURL
	}
	if req.SourceName != "" {
		updates["source_name"] = req.SourceName
	}

	return updates
}

// handleTags 处理标签关联
func (s *ArticleService) handleTags(tx *gorm.DB, article *model.Article, tagIDs []uint) error {
	// 获取文章原有的标签IDs
	var oldTagIDs []uint
	if article.ID > 0 {
		if err := tx.Model(article).Association("Tags").Find(&oldTagIDs); err != nil {
			s.log.Warnf("获取文章原有标签失败: %v", err)
		}
	}

	if len(tagIDs) == 0 {
		// 如果新标签为空，清除所有关联
		if err := tx.Model(article).Association("Tags").Clear(); err != nil {
			return err
		}
		// 更新原有标签的计数
		if len(oldTagIDs) > 0 {
			s.tagService.UpdateMultipleTagArticleCount(tx, oldTagIDs)
		}
		return nil
	}

	var tags []model.Tag
	if err := tx.Where("id IN ?", tagIDs).Find(&tags).Error; err != nil {
		return err
	}

	if err := tx.Model(article).Association("Tags").Replace(tags); err != nil {
		return err
	}

	// 合并需要更新的标签ID（包括新的和旧的）
	allTagIDs := make(map[uint]bool)
	for _, id := range tagIDs {
		allTagIDs[id] = true
	}
	for _, id := range oldTagIDs {
		allTagIDs[id] = true
	}

	// 批量更新标签计数
	var updateTagIDs []uint
	for id := range allTagIDs {
		updateTagIDs = append(updateTagIDs, id)
	}

	if len(updateTagIDs) > 0 {
		if err := s.tagService.UpdateMultipleTagArticleCount(tx, updateTagIDs); err != nil {
			s.log.Warnf("更新标签文章计数失败: %v", err)
		}
	}

	return nil
}

// preloadArticleData 预加载文章关联数据
func (s *ArticleService) preloadArticleData(tx *gorm.DB, article *model.Article) error {
	return tx.Preload("Author").Preload("Category").Preload("Tags").First(article, article.ID).Error
}

// checkPermission 检查权限
func (s *ArticleService) checkPermission(authorID, userID uint) error {
	if authorID != userID {
		return errors.New("没有权限修改此文章")
	}
	return nil
}

// updateArticleContent 更新文章内容
func (s *ArticleService) updateArticleContent(tx *gorm.DB, article *model.Article, content string) error {
	if content == "" {
		return nil
	}

	if content != "" {
		wordCount := calculateWordCount(content)
		if err := tx.Model(article).Update("word_count", wordCount).Error; err != nil {
			return err
		}
		article.WordCount = wordCount
	}

	esDoc := article.ToSearchDocument(content)
	_, err := s.updateArticleInES(article.ESDocID, esDoc)
	return err
}

// tryGetCachedDetail 尝试从缓存获取文章详情
func (s *ArticleService) tryGetCachedDetail(ctx context.Context, articleID uint, currentUserID uint) *dto.ArticleDetailResponse {
	if s.articleCache == nil {
		return nil
	}

	var cachedResponse dto.ArticleDetailResponse
	err := s.articleCache.GetArticleDetail(ctx, articleID, &cachedResponse)
	if err == nil {
		s.log.Infof("文章详情缓存命中: articleID=%d", articleID)
		
		go s.incrementViewCount(articleID)
		
		if currentUserID > 0 {
			cachedResponse.IsLiked = s.checkUserLiked(currentUserID, articleID)
			cachedResponse.IsFavorited = s.checkUserFavorited(currentUserID, articleID)
		}
		
		return &cachedResponse
	} else if err != redis.Nil {
		s.log.Warnf("获取文章缓存失败: %v", err)
	}

	return nil
}

// buildArticleDetailResponse 构建文章详情响应
func (s *ArticleService) buildArticleDetailResponse(article *model.Article, content string, currentUserID uint) *dto.ArticleDetailResponse {
	// 查询用户交互信息
	isLiked := currentUserID > 0 && s.checkUserLiked(currentUserID, article.ID)
	isFavorited := currentUserID > 0 && s.checkUserFavorited(currentUserID, article.ID)

	// 获取相关文章
	prevArticle, nextArticle := s.getAdjacentArticles(article.ID)
	relatedArticles := s.getRelatedArticles(article.ID, article.CategoryID, article.Tags)

	// 构建响应
	var publishedAtStr string
	if article.PublishedAt != nil {
		publishedAtStr = article.PublishedAt.Format("2006-01-02 15:04:05")
	}

	return &dto.ArticleDetailResponse{
		ID:              article.ID,
		Title:           article.Title,
		Content:         content,
		Summary:         article.Summary,
		CategoryID:      article.CategoryID,
		CategoryName:    article.Category.Name,
		AuthorID:        article.AuthorID,
		AuthorName:      article.Author.Nickname,
		AuthorAvatar:    article.Author.Avatar,
		CoverImage:      article.CoverImage,
		ViewCount:       article.ViewCount,
		LikeCount:       article.LikeCount,
		CommentCount:    article.CommentCount,
		FavoriteCount:   article.FavoriteCount,
		WordCount:       article.WordCount,
		Status:          article.Status,
		AccessType:      article.AccessType,
		IsTop:           article.IsTop,
		IsOriginal:      article.IsOriginal,
		SourceURL:       article.SourceURL,
		SourceName:      article.SourceName,
		Tags:            s.convertToTagInfos(article.Tags),
		CreatedAt:       article.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:       article.UpdatedAt.Format("2006-01-02 15:04:05"),
		PublishedAt:     publishedAtStr,
		IsLiked:         isLiked,
		IsFavorited:     isFavorited,
		PrevArticle:     prevArticle,
		NextArticle:     nextArticle,
		RelatedArticles: relatedArticles,
	}
}

// getAdjacentArticles 获取相邻文章
func (s *ArticleService) getAdjacentArticles(articleID uint) (*dto.SimpleArticle, *dto.SimpleArticle) {
	var prevArticle, nextArticle model.Article
	
	s.db.Where("id < ? AND status = 'published' AND access_type = 'public'", articleID).
		Order("id DESC").Limit(1).
		Select("id, title, cover_image, published_at").
		First(&prevArticle)

	s.db.Where("id > ? AND status = 'published' AND access_type = 'public'", articleID).
		Order("id ASC").Limit(1).
		Select("id, title, cover_image, published_at").
		First(&nextArticle)

	var prevData, nextData *dto.SimpleArticle
	if prevArticle.ID > 0 {
		prevData = s.convertToSimpleArticle(&prevArticle)
	}
	if nextArticle.ID > 0 {
		nextData = s.convertToSimpleArticle(&nextArticle)
	}

	return prevData, nextData
}

// getRelatedArticles 获取相关文章
func (s *ArticleService) getRelatedArticles(articleID, categoryID uint, tags []model.Tag) []dto.SimpleArticle {
	var relatedArticles []model.Article
	tagIDs := make([]uint, 0, len(tags))
	for _, tag := range tags {
		tagIDs = append(tagIDs, tag.ID)
	}

	s.db.Where("(category_id = ? OR id IN (SELECT article_id FROM article_tags WHERE tag_id IN ?)) AND id != ? AND status = 'published' AND access_type = 'public'",
		categoryID, tagIDs, articleID).
		Order("published_at DESC").Limit(5).
		Select("id, title, cover_image, published_at").
		Find(&relatedArticles)

	result := make([]dto.SimpleArticle, 0, len(relatedArticles))
	for _, article := range relatedArticles {
		result = append(result, *s.convertToSimpleArticle(&article))
	}
	return result
}

// convertToSimpleArticle 转换为简单文章
func (s *ArticleService) convertToSimpleArticle(article *model.Article) *dto.SimpleArticle {
	var publishedAtStr string
	if article.PublishedAt != nil {
		publishedAtStr = article.PublishedAt.Format("2006-01-02 15:04:05")
	}
	
	return &dto.SimpleArticle{
		ID:          article.ID,
		Title:       article.Title,
		CoverImage:  article.CoverImage,
		PublishedAt: publishedAtStr,
	}
}

// convertToTagInfos 转换标签信息
func (s *ArticleService) convertToTagInfos(tags []model.Tag) []dto.TagInfo {
	result := make([]dto.TagInfo, 0, len(tags))
	for _, tag := range tags {
		result = append(result, dto.TagInfo{
			ID:   tag.ID,
			Name: tag.Name,
		})
	}
	return result
}

// convertToArticleListItems 转换为文章列表项
func (s *ArticleService) convertToArticleListItems(articles []model.Article) []dto.ArticleListItem {
	items := make([]dto.ArticleListItem, 0, len(articles))
	for _, article := range articles {
		var publishedAtStr string
		if article.PublishedAt != nil {
			publishedAtStr = article.PublishedAt.Format("2006-01-02 15:04:05")
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
			Tags:          s.convertToTagInfos(article.Tags),
			CreatedAt:     article.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:     article.UpdatedAt.Format("2006-01-02 15:04:05"),
			PublishedAt:   publishedAtStr,
		})
	}
	return items
}

// applyUserArticleFilters 应用用户文章过滤条件
func (s *ArticleService) applyUserArticleFilters(query *gorm.DB, req *dto.ArticleQueryRequest) *gorm.DB {
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	} else {
		query = query.Where("status IN ('draft', 'published')")
	}

	if req.CategoryID > 0 {
		query = query.Where("category_id = ?", req.CategoryID)
	}

	if req.IsTop >= 0 && req.IsTop != 2 {
		query = query.Where("is_top = ?", req.IsTop)
	}

	if req.IsOriginal >= 0 && req.IsOriginal != 2 {
		query = query.Where("is_original = ?", req.IsOriginal)
	}

	if req.AccessType != "" {
		query = query.Where("access_type = ?", req.AccessType)
	} else {
		query = query.Where("access_type IN ('public', 'private', 'password')")
	}

	if req.Keyword != "" {
		query = query.Where("title LIKE ? OR summary LIKE ?", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	if req.TagID > 0 {
		query = query.Joins("JOIN article_tags ON articles.id = article_tags.article_id").
			Where("article_tags.tag_id = ?", req.TagID)
	}

	if req.StartDate != "" {
		query = query.Where("created_at >= ?", req.StartDate)
	}

	if req.EndDate != "" {
		query = query.Where("created_at <= ?", req.EndDate)
	}

	return query
}

// applyPaginationAndOrder 应用分页和排序
func (s *ArticleService) applyPaginationAndOrder(query *gorm.DB, page, pageSize int, orderBy, order string) *gorm.DB {
	if orderBy == "" {
		orderBy = "created_at"
	}
	if order == "" {
		order = "DESC"
	}

	offset := (page - 1) * pageSize
	return query.Order(fmt.Sprintf("%s %s", orderBy, order)).Offset(offset).Limit(pageSize)
}

// incrementViewCount 增加浏览量
func (s *ArticleService) incrementViewCount(articleID uint) {
	if err := s.db.Model(&model.Article{}).Where("id = ?", articleID).
		Update("view_count", gorm.Expr("view_count + 1")).Error; err != nil {
		s.log.Warnf("更新文章浏览量失败: %v", err)
	}
}

// checkUserLiked 检查用户是否点赞
func (s *ArticleService) checkUserLiked(userID, articleID uint) bool {
	var count int64
	s.db.Model(&model.ArticleLike{}).Where("user_id = ? AND article_id = ?", userID, articleID).Count(&count)
	return count > 0
}

// checkUserFavorited 检查用户是否收藏
func (s *ArticleService) checkUserFavorited(userID, articleID uint) bool {
	var count int64
	s.db.Model(&model.Favorite{}).Where("user_id = ? AND article_id = ?", userID, articleID).Count(&count)
	return count > 0
}

// 缓存相关方法

// setCacheAsync 异步设置缓存
func (s *ArticleService) setCacheAsync(ctx context.Context, articleID uint, response *dto.ArticleDetailResponse) {
	if s.articleCache == nil {
		return
	}

	go func() {
		cacheResponse := *response
		cacheResponse.IsLiked = false
		cacheResponse.IsFavorited = false
		
		if err := s.articleCache.SetArticleDetail(context.Background(), articleID, &cacheResponse); err != nil {
			s.log.Warnf("设置文章详情缓存失败: %v", err)
		} else {
			s.log.Infof("文章详情缓存设置成功: articleID=%d", articleID)
		}
	}()
}

// handleCacheAsyncCreate 处理创建文章后的缓存操作
func (s *ArticleService) handleCacheAsyncCreate(article *model.Article) {
	if s.articleCache == nil {
		return
	}

	go func() {
		ctx := context.Background()
		
		if err := s.articleCache.BatchAddArticlesToBloomFilter(ctx, []uint{article.ID}); err != nil {
			s.log.Warnf("添加文章到布隆过滤器失败: %v", err)
		}
		
		tagIDs := make([]uint, 0, len(article.Tags))
		for _, tag := range article.Tags {
			tagIDs = append(tagIDs, tag.ID)
		}
		
		if err := s.articleCache.InvalidateArticleCaches(ctx, article.ID, article.CategoryID, tagIDs); err != nil {
			s.log.Warnf("清除文章相关缓存失败: %v", err)
		}
		
		s.log.Infof("文章缓存处理完成: articleID=%d", article.ID)
	}()
}

// handleCacheAsyncUpdate 处理更新文章后的缓存操作
func (s *ArticleService) handleCacheAsyncUpdate(article *model.Article) {
	if s.articleCache == nil {
		return
	}

	go func() {
		ctx := context.Background()
		
		tagIDs := make([]uint, 0, len(article.Tags))
		for _, tag := range article.Tags {
			tagIDs = append(tagIDs, tag.ID)
		}
		
		if err := s.articleCache.InvalidateArticleCaches(ctx, article.ID, article.CategoryID, tagIDs); err != nil {
			s.log.Warnf("清除文章相关缓存失败: %v", err)
		}
		
		s.log.Infof("文章更新缓存清理完成: articleID=%d", article.ID)
	}()
}

// handleCacheAsyncDelete 处理删除文章后的缓存操作
func (s *ArticleService) handleCacheAsyncDelete(articleID, categoryID uint) {
	if s.articleCache == nil {
		return
	}

	go func() {
		ctx := context.Background()
		
		if err := s.articleCache.InvalidateArticleCaches(ctx, articleID, categoryID, []uint{}); err != nil {
			s.log.Warnf("清除文章相关缓存失败: %v", err)
		}
		
		s.log.Infof("文章删除缓存清理完成: articleID=%d", articleID)
	}()
}

// ES相关方法

// saveArticleToES 保存文章到ES
func (s *ArticleService) saveArticleToES(article *model.ESArticle) (string, error) {
	ctx := context.Background()
	jsonData, err := json.Marshal(article)
	if err != nil {
		return "", err
	}

	req := esapi.IndexRequest{
		Index:      article.ESIndexName(),
		DocumentID: article.ID,
		Body:       strings.NewReader(string(jsonData)),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, s.esClient)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.IsError() {
		return "", fmt.Errorf("保存到ES失败: %s", res.String())
	}

	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return "", err
	}

	return article.ID, nil
}

// getArticleContentFromES 从ES获取文章内容
func (s *ArticleService) getArticleContentFromES(esDocID string) (string, error) {
	if esDocID == "" {
		return "", errors.New("文章ES文档ID为空")
	}

	ctx := context.Background()
	req := esapi.GetRequest{
		Index:      "articles",
		DocumentID: esDocID,
	}

	res, err := req.Do(ctx, s.esClient)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.IsError() {
		return "", fmt.Errorf("从ES获取文章失败: %s", res.String())
	}

	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return "", err
	}

	source, exists := r["_source"]
	if !exists {
		return "", errors.New("文章内容不存在")
	}

	sourceMap, ok := source.(map[string]interface{})
	if !ok {
		return "", errors.New("解析文章内容失败")
	}

	content, exists := sourceMap["content"]
	if !exists {
		return "", errors.New("文章内容字段不存在")
	}

	contentStr, ok := content.(string)
	if !ok {
		return "", errors.New("文章内容不是字符串类型")
	}

	return contentStr, nil
}

// updateArticleInES 更新ES中的文章
func (s *ArticleService) updateArticleInES(esDocID string, article *model.ESArticle) (string, error) {
	ctx := context.Background()
	jsonData, err := json.Marshal(article)
	if err != nil {
		return "", err
	}

	req := esapi.UpdateRequest{
		Index:      article.ESIndexName(),
		DocumentID: esDocID,
		Body:       strings.NewReader(fmt.Sprintf(`{"doc":%s}`, string(jsonData))),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, s.esClient)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.IsError() {
		return "", fmt.Errorf("更新ES文档失败: %s", res.String())
	}

	return esDocID, nil
}

// deleteArticleFromES 从ES删除文章
func (s *ArticleService) deleteArticleFromES(esDocID string) error {
	if esDocID == "" {
		return errors.New("文章ES文档ID为空")
	}

	ctx := context.Background()
	req := esapi.DeleteRequest{
		Index:      "articles",
		DocumentID: esDocID,
		Refresh:    "true",
	}

	res, err := req.Do(ctx, s.esClient)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("从ES删除文章失败: %s", res.String())
	}

	return nil
}

// calculateWordCount 计算文章字数
func calculateWordCount(content string) int {
	return len(strings.Split(content, ""))
}
