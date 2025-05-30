package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/nsxzhou1114/blog-api/internal/database"
	"github.com/nsxzhou1114/blog-api/internal/dto"
	"github.com/nsxzhou1114/blog-api/internal/logger"
	"github.com/nsxzhou1114/blog-api/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ArticleInteractionService 文章交互服务
type ArticleInteractionService struct {
	db  *gorm.DB
	log *zap.SugaredLogger
}

// NewArticleInteractionService 创建文章交互服务实例
func NewArticleInteractionService() *ArticleInteractionService {
	return &ArticleInteractionService{
		db:  database.GetDB(),
		log: logger.GetSugaredLogger(),
	}
}

// LikeArticle 点赞文章
func (s *ArticleInteractionService) LikeArticle(userID, articleID uint) error {
	var article model.Article
	if err := s.db.First(&article, articleID).Error; err != nil {
		return err
	}

	var like model.ArticleLike
	result := s.db.Where("user_id = ? AND article_id = ?", userID, articleID).First(&like)
	
	if result.Error == nil {
		// 已点赞，取消点赞
		return s.executeInteractionTransaction(func(tx *gorm.DB) error {
			if err := tx.Delete(&like).Error; err != nil {
				return err
			}
			return tx.Model(&article).Update("like_count", gorm.Expr("like_count - 1")).Error
		})
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error
	}

	// 创建点赞
	return s.executeInteractionTransaction(func(tx *gorm.DB) error {
		newLike := model.ArticleLike{
			UserID:    userID,
			ArticleID: articleID,
		}
		if err := tx.Create(&newLike).Error; err != nil {
			return err
		}

		if err := tx.Model(&article).Update("like_count", gorm.Expr("like_count + 1")).Error; err != nil {
			return err
		}

		// 创建通知（如果点赞的不是自己的文章）
		if article.AuthorID != userID {
			return s.createNotification(tx, userID, article.AuthorID, articleID, "article_like", "点赞了你的文章")
		}
		return nil
	})
}

// FavoriteArticle 收藏文章
func (s *ArticleInteractionService) FavoriteArticle(userID, articleID uint) error {
	var article model.Article
	if err := s.db.First(&article, articleID).Error; err != nil {
		return err
	}

	var favorite model.Favorite
	result := s.db.Where("user_id = ? AND article_id = ?", userID, articleID).First(&favorite)
	
	if result.Error == nil {
		// 已收藏，取消收藏
		return s.executeInteractionTransaction(func(tx *gorm.DB) error {
			if err := tx.Delete(&favorite).Error; err != nil {
				return err
			}
			return tx.Model(&article).Update("favorite_count", gorm.Expr("favorite_count - 1")).Error
		})
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error
	}

	// 创建收藏
	return s.executeInteractionTransaction(func(tx *gorm.DB) error {
		newFavorite := model.Favorite{
			UserID:    userID,
			ArticleID: articleID,
		}
		if err := tx.Create(&newFavorite).Error; err != nil {
			return err
		}

		if err := tx.Model(&article).Update("favorite_count", gorm.Expr("favorite_count + 1")).Error; err != nil {
			return err
		}

		// 创建通知（如果收藏的不是自己的文章）
		if article.AuthorID != userID {
			return s.createNotification(tx, userID, article.AuthorID, articleID, "article_favorite", "收藏了你的文章")
		}
		return nil
	})
}

// GetArticleLikes 获取文章点赞用户列表
func (s *ArticleInteractionService) GetArticleLikes(articleID uint, page, pageSize int) (*dto.UserListResponse, error) {
	if err := s.checkArticleExists(articleID); err != nil {
		return nil, err
	}

	var total int64
	var users []model.User

	// 获取点赞总数
	if err := s.db.Model(&model.ArticleLike{}).
		Where("article_id = ?", articleID).
		Count(&total).Error; err != nil {
		return nil, err
	}

	// 获取点赞用户列表
	if err := s.db.Table("users").
		Select("users.id, users.username, users.email, users.avatar, users.nickname, users.bio, users.role, users.status, users.created_at").
		Joins("JOIN article_likes ON users.id = article_likes.user_id").
		Where("article_likes.article_id = ?", articleID).
		Order("article_likes.created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&users).Error; err != nil {
		return nil, err
	}

	return &dto.UserListResponse{
		Total: total,
		List:  s.convertToUserResponses(users),
	}, nil
}

// GetUserFavorites 获取用户收藏的文章列表
func (s *ArticleInteractionService) GetUserFavorites(userID uint, page, pageSize int) (*dto.ArticleListResponse, error) {
	var total int64
	var articles []model.Article

	// 获取收藏总数
	if err := s.db.Model(&model.Favorite{}).
		Where("user_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, err
	}

	// 获取收藏文章列表
	if err := s.db.Table("articles").
		Joins("JOIN favorites ON articles.id = favorites.article_id").
		Where("favorites.user_id = ?", userID).
		Preload("Author").
		Preload("Category").
		Preload("Tags").
		Order("favorites.created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&articles).Error; err != nil {
		return nil, err
	}

	return &dto.ArticleListResponse{
		Total: total,
		List:  s.convertToArticleListItems(articles),
	}, nil
}

// GetArticleStats 获取文章统计信息
func (s *ArticleInteractionService) GetArticleStats(userID uint) (*dto.ArticleStatsResponse, error) {
	var stats dto.ArticleStatsResponse

	// 获取基础统计数据
	if err := s.getBasicStats(userID, &stats); err != nil {
		return nil, err
	}

	// 获取TOP文章
	if err := s.getTopArticles(userID, &stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// UpdateArticleStatus 更新文章状态
func (s *ArticleInteractionService) UpdateArticleStatus(userID, articleID uint, status string) error {
	var article model.Article
	if err := s.db.First(&article, articleID).Error; err != nil {
		return err
	}

	if err := s.checkArticlePermission(article.AuthorID, userID); err != nil {
		return err
	}

	if err := s.validateStatus(status); err != nil {
		return err
	}

	updates := map[string]interface{}{"status": status}
	if status == "published" && article.Status != "published" {
		updates["published_at"] = time.Now()
	}

	return s.db.Model(&article).Updates(updates).Error
}

// UpdateArticleAccess 更新文章访问权限
func (s *ArticleInteractionService) UpdateArticleAccess(userID, articleID uint, accessType string, password string) error {
	var article model.Article
	if err := s.db.First(&article, articleID).Error; err != nil {
		return err
	}

	if err := s.checkArticlePermission(article.AuthorID, userID); err != nil {
		return err
	}

	if err := s.validateAccessType(accessType, password); err != nil {
		return err
	}

	updates := map[string]interface{}{"access_type": accessType}
	if accessType == "password" {
		updates["password"] = password
	} else {
		updates["password"] = ""
	}

	return s.db.Model(&article).Updates(updates).Error
}

// ProcessArticleAction 处理文章交互操作
func (s *ArticleInteractionService) ProcessArticleAction(userID, articleID uint, action string) error {
	switch action {
	case "like":
		return s.LikeArticle(userID, articleID)
	case "favorite":
		return s.FavoriteArticle(userID, articleID)
	default:
		return errors.New("不支持的操作类型")
	}
}

// 辅助方法

// executeInteractionTransaction 执行交互事务
func (s *ArticleInteractionService) executeInteractionTransaction(fn func(*gorm.DB) error) error {
	tx := s.db.Begin()
	err := fn(tx)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

// createNotification 创建通知
func (s *ArticleInteractionService) createNotification(tx *gorm.DB, senderID, receiverID, articleID uint, notificationType, action string) error {
	var user model.User
	if err := tx.Select("id, username, nickname, avatar").First(&user, senderID).Error; err != nil {
		return err
	}

	var article model.Article
	if err := tx.Select("title").First(&article, articleID).Error; err != nil {
		return err
	}

	notification := model.Notification{
		UserID:    receiverID,
		SenderID:  &senderID,
		ArticleID: &articleID,
		Type:      notificationType,
		Content:   fmt.Sprintf("用户 %s %s《%s》", user.Nickname, action, article.Title),
		IsRead:    0,
	}

	return tx.Create(&notification).Error
}

// checkArticleExists 检查文章是否存在
func (s *ArticleInteractionService) checkArticleExists(articleID uint) error {
	var article model.Article
	return s.db.First(&article, articleID).Error
}

// checkArticlePermission 检查文章权限
func (s *ArticleInteractionService) checkArticlePermission(authorID, userID uint) error {
	if authorID != userID {
		return errors.New("没有权限更新此文章")
	}
	return nil
}

// validateStatus 验证状态值
func (s *ArticleInteractionService) validateStatus(status string) error {
	validStatuses := []string{"draft", "published", "archived"}
	for _, validStatus := range validStatuses {
		if status == validStatus {
			return nil
		}
	}
	return errors.New("无效的文章状态")
}

// validateAccessType 验证访问权限类型
func (s *ArticleInteractionService) validateAccessType(accessType, password string) error {
	validTypes := []string{"public", "private", "password"}
	for _, validType := range validTypes {
		if accessType == validType {
			if accessType == "password" && password == "" {
				return errors.New("需要提供密码")
			}
			return nil
		}
	}
	return errors.New("无效的访问权限类型")
}

// getBasicStats 获取基础统计数据
func (s *ArticleInteractionService) getBasicStats(userID uint, stats *dto.ArticleStatsResponse) error {
	queries := []struct {
		query  string
		result *int64
	}{
		{"author_id = ?", &stats.TotalArticles},
		{"author_id = ? AND status = 'published'", &stats.PublishedArticles},
		{"author_id = ? AND status = 'draft'", &stats.DraftArticles},
	}

	for _, q := range queries {
		if err := s.db.Model(&model.Article{}).Where(q.query, userID).Count(q.result).Error; err != nil {
			return err
		}
	}

	// 获取聚合统计
	aggregateQueries := []struct {
		field  string
		result *int64
	}{
		{"view_count", &stats.TotalViews},
		{"like_count", &stats.TotalLikes},
		{"comment_count", &stats.TotalComments},
		{"favorite_count", &stats.TotalFavorites},
		{"word_count", &stats.TotalWordCount},
	}

	for _, q := range aggregateQueries {
		if err := s.db.Model(&model.Article{}).
			Where("author_id = ?", userID).
			Select(fmt.Sprintf("COALESCE(SUM(%s), 0)", q.field)).
			Row().Scan(q.result); err != nil {
			return err
		}
	}

	return nil
}

// getTopArticles 获取TOP文章
func (s *ArticleInteractionService) getTopArticles(userID uint, stats *dto.ArticleStatsResponse) error {
	// 访问量最高的文章
	var topViewedArticles []struct {
		ID        uint   `json:"id"`
		Title     string `json:"title"`
		ViewCount int64  `json:"view_count"`
	}
	if err := s.db.Model(&model.Article{}).
		Where("author_id = ?", userID).
		Select("id, title, view_count").
		Order("view_count DESC").
		Limit(5).
		Find(&topViewedArticles).Error; err != nil {
		return err
	}

	stats.TopViewedArticles = make([]dto.ArticleStatItem, 0, len(topViewedArticles))
	for _, article := range topViewedArticles {
		stats.TopViewedArticles = append(stats.TopViewedArticles, dto.ArticleStatItem{
			ID:    article.ID,
			Title: article.Title,
			Count: article.ViewCount,
		})
	}

	// 点赞最高的文章
	var topLikedArticles []struct {
		ID        uint   `json:"id"`
		Title     string `json:"title"`
		LikeCount int64  `json:"like_count"`
	}
	if err := s.db.Model(&model.Article{}).
		Where("author_id = ?", userID).
		Select("id, title, like_count").
		Order("like_count DESC").
		Limit(5).
		Find(&topLikedArticles).Error; err != nil {
		return err
	}

	stats.TopLikedArticles = make([]dto.ArticleStatItem, 0, len(topLikedArticles))
	for _, article := range topLikedArticles {
		stats.TopLikedArticles = append(stats.TopLikedArticles, dto.ArticleStatItem{
			ID:    article.ID,
			Title: article.Title,
			Count: article.LikeCount,
		})
	}

	return nil
}

// 转换方法

// convertToUserResponses 转换用户响应列表
func (s *ArticleInteractionService) convertToUserResponses(users []model.User) []dto.UserResponse {
	userList := make([]dto.UserResponse, 0, len(users))
	for _, user := range users {
		userList = append(userList, dto.UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Avatar:    user.Avatar,
			Nickname:  user.Nickname,
			Bio:       user.Bio,
			Role:      user.Role,
			Status:    user.Status,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
		})
	}
	return userList
}

// convertToArticleListItems 转换文章列表项
func (s *ArticleInteractionService) convertToArticleListItems(articles []model.Article) []dto.ArticleListItem {
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
	return items
}
