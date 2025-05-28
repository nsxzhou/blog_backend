package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/nsxzhou1114/blog-api/internal/database"
	"github.com/nsxzhou1114/blog-api/internal/dto"
	"github.com/nsxzhou1114/blog-api/internal/logger"
	"github.com/nsxzhou1114/blog-api/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ArticleService 文章服务
type ArticleService struct {
	db       *gorm.DB
	esClient *elasticsearch.Client
	log      *zap.SugaredLogger
}

// NewArticleService 创建文章服务实例
func NewArticleService() *ArticleService {
	return &ArticleService{
		db:       database.GetDB(),
		esClient: database.GetES(),
		log:      logger.GetSugaredLogger(),
	}
}

// Create 创建文章
func (s *ArticleService) Create(userID uint, req *dto.ArticleCreateRequest) (*model.Article, error) {
	// 1. 创建文章基本信息到MySQL
	article := &model.Article{
		Title:      req.Title,
		Summary:    req.Summary,
		Status:     req.Status,
		AuthorID:   userID,
		CategoryID: req.CategoryID,
		CoverImage: req.CoverImage,
		WordCount:  calculateWordCount(req.Content), // 计算字数
		AccessType: req.AccessType,
		Password:   req.Password,
		IsTop:      req.IsTop,
		IsOriginal: req.IsOriginal,
		SourceURL:  req.SourceURL,
		SourceName: req.SourceName,
	}

	// 如果状态为已发布，设置发布时间
	if req.Status == "published" {
		now := time.Now()
		article.PublishedAt = &now
	} else {
		article.PublishedAt = nil
	}

	// 启动事务
	tx := s.db.Begin()

	if err := tx.Create(article).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 处理标签关联
	if len(req.TagIDs) > 0 {
		var tags []model.Tag
		if err := tx.Where("id IN ?", req.TagIDs).Find(&tags).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		if err := tx.Model(article).Association("Tags").Replace(tags); err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// 加载关联数据，用于创建ES文档
	if err := tx.Preload("Author").Preload("Category").Preload("Tags").First(article, article.ID).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 2. 创建文章内容到ES
	esDoc := article.ToSearchDocument(req.Content)

	esDocID, err := s.saveArticleToES(esDoc)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// 更新MySQL中的ES文档ID引用
	article.ESDocID = esDocID
	if err := tx.Save(article).Error; err != nil {
		tx.Rollback()
		// 尝试从ES中删除已创建的文档
		s.deleteArticleFromES(esDocID)
		return nil, err
	}

	tx.Commit()
	return article, nil
}

// Update 更新文章
func (s *ArticleService) Update(userID uint, articleID uint, req *dto.ArticleUpdateRequest) (*model.Article, error) {
	// 查找文章并验证权限
	var article model.Article
	if err := s.db.First(&article, articleID).Error; err != nil {
		return nil, err
	}

	// 检查权限
	if article.AuthorID != userID {
		return nil, errors.New("没有权限修改此文章")
	}

	// 启动事务
	tx := s.db.Begin()

	// 准备更新数据
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
		// 如果是首次发布，设置发布时间
		if req.Status == "published" && article.Status != "published" {
			now := time.Now()
			updates["published_at"] = &now
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

	// 更新MySQL中的文章信息
	if len(updates) > 0 {
		if err := tx.Model(&article).Updates(updates).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// 处理标签更新
	if len(req.TagIDs) > 0 {
		var tags []model.Tag
		if err := tx.Where("id IN ?", req.TagIDs).Find(&tags).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		if err := tx.Model(&article).Association("Tags").Replace(tags); err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// 重新加载文章及其关联数据
	if err := tx.Preload("Author").Preload("Category").Preload("Tags").First(&article, article.ID).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 如果有内容更新或其他字段更新，更新ES文档
	if req.Content != "" || len(updates) > 0 {
		// 获取当前ES中的内容
		content := req.Content
		if content == "" {
			var err error
			content, err = s.getArticleContentFromES(article.ESDocID)
			if err != nil {
				tx.Rollback()
				return nil, err
			}
		}

		if req.Content != "" {
			// 更新文章字数统计
			wordCount := calculateWordCount(req.Content)
			if err := tx.Model(&article).Update("word_count", wordCount).Error; err != nil {
				tx.Rollback()
				return nil, err
			}
			article.WordCount = wordCount
		}

		// 更新ES文档
		esDoc := article.ToSearchDocument(content)
		if _, err := s.updateArticleInES(article.ESDocID, esDoc); err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	tx.Commit()
	return &article, nil
}

// Delete 删除文章
func (s *ArticleService) Delete(userID uint, articleID uint, role string) error {
	// 查找文章并验证权限
	var article model.Article
	if err := s.db.First(&article, articleID).Error; err != nil {
		return err
	}	

	// 检查权限
	if article.AuthorID != userID && role != "admin" {
		return errors.New("没有权限删除此文章")
	}

	// 启动事务
	tx := s.db.Begin()

	// 1. 从MySQL中软删除文章
	if err := tx.Delete(&article).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 2. 从ES中删除文章内容
	if article.ESDocID != "" {
		if err := s.deleteArticleFromES(article.ESDocID); err != nil {
			s.log.Warnf("从ES删除文章 %s 失败: %v", article.ESDocID, err)
			// 不回滚事务，因为MySQL删除是主要操作
		}
	}

	tx.Commit()
	return nil
}

// GetByID 根据ID获取文章基本信息
func (s *ArticleService) GetByID(articleID uint) (*model.Article, error) {
	var article model.Article
	err := s.db.First(&article, articleID).Error
	return &article, err
}

// GetArticleDetail 获取文章详情
func (s *ArticleService) GetArticleDetail(articleID uint, currentUserID uint) (*dto.ArticleDetailResponse, error) {
	// 1. 从MySQL获取文章基本信息
	var article model.Article
	if err := s.db.Preload("Author").Preload("Category").Preload("Tags").First(&article, articleID).Error; err != nil {
		return nil, err
	}

	// 增加浏览量
	if err := s.db.Model(&article).Update("view_count", gorm.Expr("view_count + 1")).Error; err != nil {
		s.log.Warnf("更新文章浏览量失败: %v", err)
	}

	// 2. 从ES获取文章内容
	content, err := s.getArticleContentFromES(article.ESDocID)
	if err != nil {
		return nil, err
	}

	// 3. 查询用户交互信息
	isLiked := false
	isFavorited := false
	if currentUserID > 0 {
		// 检查是否点赞
		var likeCount int64
		s.db.Model(&model.ArticleLike{}).Where("user_id = ? AND article_id = ?", currentUserID, articleID).Count(&likeCount)
		isLiked = likeCount > 0

		// 检查是否收藏
		var favoriteCount int64
		s.db.Model(&model.Favorite{}).Where("user_id = ? AND article_id = ?", currentUserID, articleID).Count(&favoriteCount)
		isFavorited = favoriteCount > 0
	}

	// 4. 获取上一篇和下一篇文章
	var prevArticle, nextArticle model.Article
	s.db.Where("id < ? AND status = 'published' AND access_type = 'public'", articleID).
		Order("id DESC").Limit(1).
		Select("id, title, cover_image, published_at").
		First(&prevArticle)

	s.db.Where("id > ? AND status = 'published' AND access_type = 'public'", articleID).
		Order("id ASC").Limit(1).
		Select("id, title, cover_image, published_at").
		First(&nextArticle)

	// 5. 获取相关文章（同一分类或标签）
	var relatedArticles []model.Article
	tagIDs := make([]uint, 0, len(article.Tags))
	for _, tag := range article.Tags {
		tagIDs = append(tagIDs, tag.ID)
	}

	s.db.Where("(category_id = ? OR id IN (SELECT article_id FROM article_tags WHERE tag_id IN ?)) AND id != ? AND status = 'published' AND access_type = 'public'",
		article.CategoryID, tagIDs, articleID).
		Order("published_at DESC").Limit(5).
		Select("id, title, cover_image, published_at").
		Find(&relatedArticles)

	// 6. 组装响应
	var prevArticleData, nextArticleData *dto.SimpleArticle
	if prevArticle.ID > 0 {
		var publishedAt time.Time
		if prevArticle.PublishedAt != nil {
			publishedAt = *prevArticle.PublishedAt
		}
		prevArticleData = &dto.SimpleArticle{
			ID:          prevArticle.ID,
			Title:       prevArticle.Title,
			CoverImage:  prevArticle.CoverImage,
			PublishedAt: publishedAt,
		}
	}

	if nextArticle.ID > 0 {
		var publishedAt time.Time
		if nextArticle.PublishedAt != nil {
			publishedAt = *nextArticle.PublishedAt
		}
		nextArticleData = &dto.SimpleArticle{
			ID:          nextArticle.ID,
			Title:       nextArticle.Title,
			CoverImage:  nextArticle.CoverImage,
			PublishedAt: publishedAt,
		}
	}

	relatedArticlesData := make([]dto.SimpleArticle, 0, len(relatedArticles))
	for _, a := range relatedArticles {
		var publishedAt time.Time
		if a.PublishedAt != nil {
			publishedAt = *a.PublishedAt
		}
		relatedArticlesData = append(relatedArticlesData, dto.SimpleArticle{
			ID:          a.ID,
			Title:       a.Title,
			CoverImage:  a.CoverImage,
			PublishedAt: publishedAt,
		})
	}

	// 构建标签信息
	tags := make([]dto.TagInfo, 0, len(article.Tags))
	for _, tag := range article.Tags {
		tags = append(tags, dto.TagInfo{
			ID:   tag.ID,
			Name: tag.Name,
		})
	}

	// 构建文章详情响应
	var articlePublishedAt time.Time
	if article.PublishedAt != nil {
		articlePublishedAt = *article.PublishedAt
	}

	response := &dto.ArticleDetailResponse{
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
		Tags:            tags,
		CreatedAt:       article.CreatedAt,
		UpdatedAt:       article.UpdatedAt,
		PublishedAt:     articlePublishedAt,
		IsLiked:         isLiked,
		IsFavorited:     isFavorited,
		PrevArticle:     prevArticleData,
		NextArticle:     nextArticleData,
		RelatedArticles: relatedArticlesData,
	}

	return response, nil
}

// GetUserArticles 获取用户文章列表
func (s *ArticleService) GetUserArticles(userID uint, req *dto.ArticleQueryRequest) (*dto.ArticleListResponse, error) {
	var articles []model.Article
	var total int64
	s.log.Infof("GetUserArticles userID: %d, req: %+v", userID, req)
	query := s.db.Model(&model.Article{}).Where("author_id = ?", userID)

	// 应用过滤条件
	if req.Status != "" {
		if req.Status == "all" {
			query = query.Where("status IN ('draft', 'published')")
		} else {
			query = query.Where("status = ?", req.Status)
		}
	}
	if req.CategoryID > 0 {
		query = query.Where("category_id = ?", req.CategoryID)
	}

	if req.IsTop >= 0 {
		if req.IsTop == 2 {
			query = query.Where("is_top >= 0")
		} else {
			query = query.Where("is_top = ?", req.IsTop)
		}
	}
	if req.IsOriginal >= 0 {
		if req.IsOriginal == 2 {
			query = query.Where("is_original >= 0")
		} else {
			query = query.Where("is_original = ?", req.IsOriginal)
		}
	}

	if req.AccessType != "" {
		if req.AccessType == "all" {
			query = query.Where("access_type IN ('public', 'private', 'password')")
		} else {
			query = query.Where("access_type = ?", req.AccessType)
		}
	}

	// 应用关键词搜索
	if req.Keyword != "" {
		query = query.Where("title LIKE ? OR summary LIKE ?", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	// 应用标签过滤
	if req.TagID > 0 {
		query = query.Joins("JOIN article_tags ON articles.id = article_tags.article_id").
			Where("article_tags.tag_id = ?", req.TagID)
	}

	// 应用日期过滤
	if req.StartDate != "" {
		query = query.Where("created_at >= ?", req.StartDate)
	}

	if req.EndDate != "" {
		query = query.Where("created_at <= ?", req.EndDate)
	}

	// 获取总数
	query.Count(&total)

	// 应用排序
	orderBy := "created_at"
	if req.OrderBy != "" {
		orderBy = req.OrderBy
	}
	order := "DESC"
	if req.Order != "" {
		order = req.Order
	}
	query = query.Order(fmt.Sprintf("%s %s", orderBy, order))

	// 应用分页
	offset := (req.Page - 1) * req.PageSize
	query = query.Offset(offset).Limit(req.PageSize)

	// 预加载关联数据
	query = query.Preload("Author").Preload("Category").Preload("Tags")

	// 执行查询
	if err := query.Find(&articles).Error; err != nil {
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
		List:  items,
	}, nil
}

// 保存文章到ES
func (s *ArticleService) saveArticleToES(article *model.ESArticle) (string, error) {
	ctx := context.Background()

	// 准备JSON数据
	jsonData, err := json.Marshal(article)
	if err != nil {
		return "", err
	}

	// 准备ES请求
	req := esapi.IndexRequest{
		Index:      article.ESIndexName(),
		DocumentID: article.ID,
		Body:       strings.NewReader(string(jsonData)),
		Refresh:    "true",
	}

	// 执行ES请求
	res, err := req.Do(ctx, s.esClient)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	// 检查响应
	if res.IsError() {
		return "", fmt.Errorf("保存到ES失败: %s", res.String())
	}

	// 解析响应
	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return "", err
	}

	// 返回文档ID
	return article.ID, nil
}

// 从ES中获取文章内容
func (s *ArticleService) getArticleContentFromES(esDocID string) (string, error) {
	if esDocID == "" {
		return "", errors.New("文章ES文档ID为空")
	}

	ctx := context.Background()

	// 准备ES请求
	req := esapi.GetRequest{
		Index:      "articles",
		DocumentID: esDocID,
	}

	// 执行ES请求
	res, err := req.Do(ctx, s.esClient)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	// 检查响应
	if res.IsError() {
		return "", fmt.Errorf("从ES获取文章失败: %s", res.String())
	}

	// 解析响应
	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return "", err
	}

	// 获取文章内容
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

// 更新ES中的文章
func (s *ArticleService) updateArticleInES(esDocID string, article *model.ESArticle) (string, error) {
	ctx := context.Background()

	// 准备JSON数据
	jsonData, err := json.Marshal(article)
	if err != nil {
		return "", err
	}

	// 准备ES请求
	req := esapi.UpdateRequest{
		Index:      article.ESIndexName(),
		DocumentID: esDocID,
		Body:       strings.NewReader(fmt.Sprintf(`{"doc":%s}`, string(jsonData))),
		Refresh:    "true",
	}

	// 执行ES请求
	res, err := req.Do(ctx, s.esClient)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	// 检查响应
	if res.IsError() {
		return "", fmt.Errorf("更新ES文档失败: %s", res.String())
	}

	return esDocID, nil
}

// 从ES中删除文章
func (s *ArticleService) deleteArticleFromES(esDocID string) error {
	if esDocID == "" {
		return errors.New("文章ES文档ID为空")
	}

	ctx := context.Background()

	// 准备ES请求
	req := esapi.DeleteRequest{
		Index:      "articles",
		DocumentID: esDocID,
		Refresh:    "true",
	}

	// 执行ES请求
	res, err := req.Do(ctx, s.esClient)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// 检查响应
	if res.IsError() {
		return fmt.Errorf("从ES删除文章失败: %s", res.String())
	}

	return nil
}

// 计算文章字数
func calculateWordCount(content string) int {
	// 简单实现，实际可能需要更复杂的算法处理HTML标签等
	return len(strings.Split(content, ""))
}
