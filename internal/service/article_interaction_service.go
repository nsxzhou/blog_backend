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
	// 检查文章是否存在
	var article model.Article
	if err := s.db.First(&article, articleID).Error; err != nil {
		return err
	}

	// 检查是否已经点赞
	var like model.ArticleLike
	result := s.db.Where("user_id = ? AND article_id = ?", userID, articleID).First(&like)
	if result.Error == nil {
		// 已经点赞，取消点赞
		tx := s.db.Begin()

		// 删除点赞记录
		if err := tx.Delete(&like).Error; err != nil {
			tx.Rollback()
			return err
		}

		// 减少文章点赞数
		if err := tx.Model(&article).Update("like_count", gorm.Expr("like_count - 1")).Error; err != nil {
			tx.Rollback()
			return err
		}

		tx.Commit()
		return nil
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// 查询错误
		return result.Error
	}

	// 创建点赞
	tx := s.db.Begin()

	// 创建点赞记录
	newLike := model.ArticleLike{
		UserID:    userID,
		ArticleID: articleID,
	}
	if err := tx.Create(&newLike).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 增加文章点赞数
	if err := tx.Model(&article).Update("like_count", gorm.Expr("like_count + 1")).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 获取用户信息，用于创建通知
	var user model.User
	if err := tx.Select("id, username, nickname, avatar").First(&user, userID).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 创建通知（如果点赞的不是自己的文章）
	if article.AuthorID != userID {
		notification := model.Notification{
			UserID:    article.AuthorID,
			SenderID:  &userID,
			ArticleID: &articleID,
			Type:      "article_like",
			Content:   fmt.Sprintf("用户 %s 点赞了你的文章《%s》", user.Nickname, article.Title),
			IsRead:    0,
		}
		if err := tx.Create(&notification).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	tx.Commit()
	return nil
}

// FavoriteArticle 收藏文章
func (s *ArticleInteractionService) FavoriteArticle(userID, articleID uint) error {
	// 检查文章是否存在
	var article model.Article
	if err := s.db.First(&article, articleID).Error; err != nil {
		return err
	}

	// 检查是否已经收藏
	var favorite model.Favorite
	result := s.db.Where("user_id = ? AND article_id = ?", userID, articleID).First(&favorite)
	if result.Error == nil {
		// 已经收藏，取消收藏
		tx := s.db.Begin()

		// 删除收藏记录
		if err := tx.Delete(&favorite).Error; err != nil {
			tx.Rollback()
			return err
		}

		// 减少文章收藏数
		if err := tx.Model(&article).Update("favorite_count", gorm.Expr("favorite_count - 1")).Error; err != nil {
			tx.Rollback()
			return err
		}

		tx.Commit()
		return nil
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// 查询错误
		return result.Error
	}

	// 创建收藏
	tx := s.db.Begin()

	// 创建收藏记录
	newFavorite := model.Favorite{
		UserID:    userID,
		ArticleID: articleID,
	}
	if err := tx.Create(&newFavorite).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 增加文章收藏数
	if err := tx.Model(&article).Update("favorite_count", gorm.Expr("favorite_count + 1")).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 获取用户信息，用于创建通知
	var user model.User
	if err := tx.Select("id, username, nickname, avatar").First(&user, userID).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 创建通知（如果收藏的不是自己的文章）
	if article.AuthorID != userID {
		notification := model.Notification{
			UserID:    article.AuthorID,
			SenderID:  &userID,
			ArticleID: &articleID,
			Type:      "article_favorite",
			Content:   fmt.Sprintf("用户 %s 收藏了你的文章《%s》", user.Nickname, article.Title),
			IsRead:    0,
		}
		if err := tx.Create(&notification).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	tx.Commit()
	return nil
}

// GetArticleLikes 获取文章点赞用户列表
func (s *ArticleInteractionService) GetArticleLikes(articleID uint, page, pageSize int) (*dto.UserListResponse, error) {
	// 检查文章是否存在
	var article model.Article
	if err := s.db.First(&article, articleID).Error; err != nil {
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

	// 转换为响应格式
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

	return &dto.UserListResponse{
		Total: total,
		List:  userList,
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

// GetArticleStats 获取文章统计信息
func (s *ArticleInteractionService) GetArticleStats(userID uint) (*dto.ArticleStatsResponse, error) {
	var stats dto.ArticleStatsResponse

	// 获取文章总数
	if err := s.db.Model(&model.Article{}).
		Where("author_id = ?", userID).
		Count(&stats.TotalArticles).Error; err != nil {
		return nil, err
	}

	// 获取已发布文章数
	if err := s.db.Model(&model.Article{}).
		Where("author_id = ? AND status = 'published'", userID).
		Count(&stats.PublishedArticles).Error; err != nil {
		return nil, err
	}

	// 获取草稿数
	if err := s.db.Model(&model.Article{}).
		Where("author_id = ? AND status = 'draft'", userID).
		Count(&stats.DraftArticles).Error; err != nil {
		return nil, err
	}

	// 获取文章总访问量
	var viewCount int64
	if err := s.db.Model(&model.Article{}).
		Where("author_id = ?", userID).
		Select("COALESCE(SUM(view_count), 0)").
		Row().Scan(&viewCount); err != nil {
		return nil, err
	}
	stats.TotalViews = viewCount

	// 获取文章总点赞数
	var likeCount int64
	if err := s.db.Model(&model.Article{}).
		Where("author_id = ?", userID).
		Select("COALESCE(SUM(like_count), 0)").
		Row().Scan(&likeCount); err != nil {
		return nil, err
	}
	stats.TotalLikes = likeCount

	// 获取文章总评论数
	var commentCount int64
	if err := s.db.Model(&model.Article{}).
		Where("author_id = ?", userID).
		Select("COALESCE(SUM(comment_count), 0)").
		Row().Scan(&commentCount); err != nil {
		return nil, err
	}
	stats.TotalComments = commentCount

	// 获取文章总收藏数
	var favoriteCount int64
	if err := s.db.Model(&model.Article{}).
		Where("author_id = ?", userID).
		Select("COALESCE(SUM(favorite_count), 0)").
		Row().Scan(&favoriteCount); err != nil {
		return nil, err
	}
	stats.TotalFavorites = favoriteCount

	// 获取文章总字数
	var wordCount int64
	if err := s.db.Model(&model.Article{}).
		Where("author_id = ?", userID).
		Select("COALESCE(SUM(word_count), 0)").
		Row().Scan(&wordCount); err != nil {
		return nil, err
	}
	stats.TotalWordCount = wordCount

	// 获取访问量最高的5篇文章
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
		return nil, err
	}

	stats.TopViewedArticles = make([]dto.ArticleStatItem, 0, len(topViewedArticles))
	for _, article := range topViewedArticles {
		stats.TopViewedArticles = append(stats.TopViewedArticles, dto.ArticleStatItem{
			ID:    article.ID,
			Title: article.Title,
			Count: article.ViewCount,
		})
	}

	// 获取点赞最高的5篇文章
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
		return nil, err
	}

	stats.TopLikedArticles = make([]dto.ArticleStatItem, 0, len(topLikedArticles))
	for _, article := range topLikedArticles {
		stats.TopLikedArticles = append(stats.TopLikedArticles, dto.ArticleStatItem{
			ID:    article.ID,
			Title: article.Title,
			Count: article.LikeCount,
		})
	}

	return &stats, nil
}

// UpdateArticleStatus 更新文章状态
func (s *ArticleInteractionService) UpdateArticleStatus(userID, articleID uint, status string) error {
	// 检查文章是否存在
	var article model.Article
	if err := s.db.First(&article, articleID).Error; err != nil {
		return err
	}

	// 检查权限
	if article.AuthorID != userID {
		return errors.New("没有权限更新此文章的状态")
	}

	// 验证状态值
	if status != "draft" && status != "published" && status != "archived" {
		return errors.New("无效的文章状态")
	}

	// 更新状态
	updates := map[string]interface{}{
		"status": status,
	}

	// 如果从草稿变为已发布，设置发布时间
	if status == "published" && article.Status != "published" {
		updates["published_at"] = time.Now()
	}

	if err := s.db.Model(&article).Updates(updates).Error; err != nil {
		return err
	}

	return nil
}

// UpdateArticleAccess 更新文章访问权限
func (s *ArticleInteractionService) UpdateArticleAccess(userID, articleID uint, accessType string, password string) error {
	// 检查文章是否存在
	var article model.Article
	if err := s.db.First(&article, articleID).Error; err != nil {
		return err
	}

	// 检查权限
	if article.AuthorID != userID {
		return errors.New("没有权限更新此文章的访问权限")
	}

	// 验证访问权限类型
	if accessType != "public" && accessType != "private" && accessType != "password" {
		return errors.New("无效的访问权限类型")
	}

	// 如果是密码访问，但没有提供密码
	if accessType == "password" && password == "" {
		return errors.New("需要提供密码")
	}

	// 更新访问权限
	updates := map[string]interface{}{
		"access_type": accessType,
	}

	// 如果是密码访问，更新密码
	if accessType == "password" {
		updates["password"] = password
	} else {
		updates["password"] = ""
	}

	if err := s.db.Model(&article).Updates(updates).Error; err != nil {
		return err
	}

	return nil
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
