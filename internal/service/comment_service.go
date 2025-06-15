package service

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/nsxzhou1114/blog-api/internal/database"
	"github.com/nsxzhou1114/blog-api/internal/dto"
	"github.com/nsxzhou1114/blog-api/internal/logger"
	"github.com/nsxzhou1114/blog-api/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	commentService     *CommentService
	commentServiceOnce sync.Once
)

// CommentService 评论服务
type CommentService struct {
	db     *gorm.DB
	logger *zap.SugaredLogger
}

// NewCommentService 创建评论服务实例
func NewCommentService() *CommentService {
	commentServiceOnce.Do(func() {
		commentService = &CommentService{
			db:     database.GetDB(),
			logger: logger.GetSugaredLogger(),
		}
	})
	return commentService
}

// Create 创建评论
func (s *CommentService) Create(userID uint, req *dto.CommentCreateRequest) (*model.Comment, error) {
	// 检查文章是否存在
	var article model.Article
	if err := s.db.First(&article, req.ArticleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("文章不存在")
		}
		return nil, err
	}

	// 如果是回复评论，检查父评论是否存在
	if req.ParentID != nil && *req.ParentID > 0 {
		var parentComment model.Comment
		if err := s.db.First(&parentComment, *req.ParentID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("回复的评论不存在")
			}
			return nil, err
		}

		// 检查父评论是否属于同一篇文章
		if parentComment.ArticleID != req.ArticleID {
			return nil, errors.New("不能回复其他文章的评论")
		}
	}

	// 使用敏感词过滤服务处理内容
	sensitiveService := NewSensitiveService()
	originalContent := req.Content
	filteredContent := sensitiveService.FilterSensitiveWords(originalContent)
	containsSensitive := sensitiveService.ContainsSensitiveWord(originalContent)

	// 创建评论
	comment := &model.Comment{
		Content:   filteredContent,
		ArticleID: req.ArticleID,
		UserID:    userID,
		ParentID:  req.ParentID,
		Status:    "approved", // 默认状态为已通过（敏感词已被过滤）
		LikeCount: 0,          // 初始化点赞数
	}

	// 如果包含敏感词，记录处理信息
	if containsSensitive {
		sensitiveWords := sensitiveService.GetSensitiveWords(originalContent)
		comment.RejectReason = "内容已过滤敏感词: " + strings.Join(sensitiveWords, ", ")
		s.logger.Infof("评论内容包含敏感词已被过滤: %s", comment.RejectReason)
	}

	// 开启事务
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 创建评论
		if err := tx.Create(comment).Error; err != nil {
			return err
		}

		// 更新文章评论计数
		if err := tx.Model(&article).
			UpdateColumn("comment_count", gorm.Expr("comment_count + ?", 1)).
			Error; err != nil {
			return err
		}

		// 预加载用户信息
		if err := tx.Preload("User").First(comment, comment.ID).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 异步创建通知
	go s.createCommentNotification(userID, comment, &article)

	return comment, nil
}

// Update 更新评论（仅允许评论作者或管理员）
func (s *CommentService) Update(id uint, userID uint, isAdmin bool, req *dto.CommentUpdateRequest) (*model.Comment, error) {
	// 获取评论
	var comment model.Comment
	if err := s.db.Preload("User").First(&comment, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("评论不存在")
		}
		return nil, err
	}

	// 检查权限
	if !isAdmin && comment.UserID != userID {
		return nil, errors.New("您无权修改该评论")
	}

	// 过滤敏感词
	sensitiveService := NewSensitiveService()
	filteredContent := sensitiveService.FilterSensitiveWords(req.Content)
	containsSensitive := sensitiveService.ContainsSensitiveWord(req.Content)

	// 更新内容
	updates := map[string]interface{}{
		"content": filteredContent,
	}

	// 如果包含敏感词，记录处理信息
	if containsSensitive {
		sensitiveWords := sensitiveService.GetSensitiveWords(req.Content)
		updates["reject_reason"] = "内容已过滤敏感词: " + strings.Join(sensitiveWords, ", ")
		s.logger.Infof("更新的评论内容包含敏感词已被过滤: %v", sensitiveWords)
	} else {
		updates["reject_reason"] = ""
	}

	// 管理员可以更新状态
	if isAdmin && req.Status != "" {
		updates["status"] = req.Status
	}

	// 更新评论
	if err := s.db.Model(&comment).Updates(updates).Error; err != nil {
		return nil, err
	}

	// 重新查询完整信息，包含User信息
	if err := s.db.Preload("User").First(&comment, id).Error; err != nil {
		return nil, err
	}

	return &comment, nil
}

// Reply 回复评论
func (s *CommentService) Reply(userID uint, req *dto.CommentReplyRequest) (*model.Comment, error) {
	// 获取父评论
	var parentComment model.Comment
	if err := s.db.First(&parentComment, req.CommentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("回复的评论不存在")
		}
		return nil, err
	}

	// 创建回复
	createReq := &dto.CommentCreateRequest{
		Content:   req.Content,
		ArticleID: parentComment.ArticleID,
		ParentID:  &parentComment.ID,
	}

	return s.Create(userID, createReq)
}

// Delete 删除评论（仅允许评论作者或管理员）
func (s *CommentService) Delete(id uint, userID uint, isAdmin bool) error {
	// 获取评论
	var comment model.Comment
	if err := s.db.First(&comment, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("评论不存在")
		}
		return err
	}

	// 检查权限
	if !isAdmin && comment.UserID != userID {
		return errors.New("您无权删除该评论")
	}

	// 开启事务
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 如果有子评论，将子评论的父ID设为null
		if err := tx.Model(&model.Comment{}).
			Where("parent_id = ?", id).
			Update("parent_id", nil).Error; err != nil {
			return err
		}

		// 删除评论点赞
		if err := tx.Where("comment_id = ?", id).Delete(&model.CommentLike{}).Error; err != nil {
			return err
		}

		// 删除评论
		if err := tx.Delete(&comment).Error; err != nil {
			return err
		}

		// 更新文章评论计数
		if comment.Status == "approved" {
			if err := tx.Model(&model.Article{}).
				Where("id = ?", comment.ArticleID).
				Where("comment_count > 0").
				UpdateColumn("comment_count", gorm.Expr("comment_count - ?", 1)).
				Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// GetByID 根据ID获取评论
func (s *CommentService) GetByID(id uint) (*model.Comment, error) {
	var comment model.Comment
	if err := s.db.Preload("User").First(&comment, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("评论不存在")
		}
		return nil, err
	}
	return &comment, nil
}

// Like 点赞评论
func (s *CommentService) Like(userID uint, commentID uint) error {
	// 检查评论是否存在
	var comment model.Comment
	if err := s.db.First(&comment, commentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("评论不存在")
		}
		return err
	}

	// 检查是否已点赞
	var count int64
	if err := s.db.Model(&model.CommentLike{}).
		Where("user_id = ? AND comment_id = ?", userID, commentID).
		Count(&count).Error; err != nil {
		return err
	}

	// 开启事务
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 已点赞则取消，未点赞则添加
		if count > 0 {
			// 取消点赞
			if err := tx.Where("user_id = ? AND comment_id = ?", userID, commentID).
				Delete(&model.CommentLike{}).Error; err != nil {
				return err
			}
			// 减少点赞数
			if err := tx.Model(&model.Comment{}).
				Where("id = ? AND like_count > 0", commentID).
				UpdateColumn("like_count", gorm.Expr("like_count - ?", 1)).
				Error; err != nil {
				return err
			}
		} else {
			// 添加点赞
			like := &model.CommentLike{
				UserID:    userID,
				CommentID: commentID,
			}
			if err := tx.Create(like).Error; err != nil {
				return err
			}
			// 增加点赞数
			if err := tx.Model(&model.Comment{}).
				Where("id = ?", commentID).
				UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).
				Error; err != nil {
				return err
			}

			// 异步创建点赞通知
			go s.createCommentLikeNotification(userID, &comment)
		}

		return nil
	})
}

// List 获取评论列表
func (s *CommentService) List(req *dto.CommentListRequest, currentUserID *uint) (*dto.CommentListResponse, error) {
	var comments []model.Comment

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
		req.Order = "desc"
	}

	// 构建查询
	query := s.db.Model(&model.Comment{})

	// 文章ID筛选
	if req.ArticleID != nil && *req.ArticleID != 0 {
		query = query.Where("article_id = ?", *req.ArticleID)
	}

	// 用户ID筛选
	if req.UserID != nil && *req.UserID != 0 {
		query = query.Where("user_id = ?", *req.UserID)
	}

	// 状态筛选
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	} else {
		query = query.Where("status IN ('pending', 'approved', 'rejected')")
	}

	// 父评论ID筛选（用于获取回复）
	if req.ParentID != nil && *req.ParentID != 0 {
		query = query.Where("parent_id = ?", *req.ParentID)
	} else {
		// 默认只查询顶级评论（无父评论）
		query = query.Where("parent_id IS NULL")
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 排序和分页
	orderStr := fmt.Sprintf("%s %s", req.OrderBy, req.Order)
	if err := query.Order(orderStr).
		Preload("User").
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Find(&comments).Error; err != nil {
		return nil, err
	}

	// 构建响应
	resp := &dto.CommentListResponse{
		Total: total,
		List:  make([]dto.CommentResponse, 0, len(comments)),
	}

	for _, comment := range comments {
		commentResp, err := s.GenerateCommentResponseWithStatus(&comment, currentUserID, req.Status)
		if err != nil {
			s.logger.Warnf("生成评论响应失败: %v", err)
			continue
		}
		resp.List = append(resp.List, *commentResp)
	}

	return resp, nil
}

// UpdateStatus 更新评论状态（管理员使用）
func (s *CommentService) UpdateStatus(id uint, req *dto.CommentStatusUpdateRequest) (*model.Comment, error) {
	// 获取评论
	var comment model.Comment
	if err := s.db.First(&comment, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("评论不存在")
		}
		return nil, err
	}

	// 更新状态
	if err := s.db.Model(&comment).Update("status", req.Status).Error; err != nil {
		return nil, err
	}

	// 重新查询完整信息
	if err := s.db.First(&comment, id).Error; err != nil {
		return nil, err
	}

	return &comment, nil
}

// BatchUpdateStatus 批量更新评论状态（管理员使用）
func (s *CommentService) BatchUpdateStatus(req *dto.CommentBatchStatusUpdateRequest) error {
	if len(req.IDs) == 0 {
		return nil
	}

	// 批量更新状态
	return s.db.Model(&model.Comment{}).
		Where("id IN ?", req.IDs).
		Update("status", req.Status).Error
}

// GenerateCommentResponse 生成评论响应DTO
func (s *CommentService) GenerateCommentResponse(comment *model.Comment, currentUserID *uint) (*dto.CommentResponse, error) {
	return s.GenerateCommentResponseWithStatus(comment, currentUserID, "")
}

// GenerateCommentResponseWithStatus 生成评论响应DTO（带状态筛选）
func (s *CommentService) GenerateCommentResponseWithStatus(comment *model.Comment, currentUserID *uint, statusFilter string) (*dto.CommentResponse, error) {
	// 填充基本信息
	resp := &dto.CommentResponse{
		ID:           comment.ID,
		Content:      comment.Content,
		ArticleID:    comment.ArticleID,
		UserID:       comment.UserID,
		ParentID:     comment.ParentID,
		Status:       comment.Status,
		RejectReason: comment.RejectReason,
		CreatedAt:    comment.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:    comment.UpdatedAt.Format("2006-01-02 15:04:05"),
		LikeCount:    comment.LikeCount,
		LikedByMe:    false, // 默认为false
	}

	// 检查当前用户是否已点赞
	if currentUserID != nil {
		resp.LikedByMe = s.hasUserLiked(*currentUserID, comment.ID)
	}

	// 填充用户信息
	if comment.User.ID > 0 {
		resp.User = dto.CommentUserInfo{
			ID:       comment.User.ID,
			Username: comment.User.Username,
			Avatar:   comment.User.Avatar,
		}
	} else {
		// 如果User信息未预加载，手动查询
		var user model.User
		if err := s.db.First(&user, comment.UserID).Error; err == nil {
			resp.User = dto.CommentUserInfo{
				ID:       user.ID,
				Username: user.Username,
				Avatar:   user.Avatar,
			}
		}
	}

	// 获取父评论信息（如果有）
	if comment.ParentID != nil && *comment.ParentID > 0 {
		var parent model.Comment
		parentQuery := s.db.Preload("User")

		// 如果有状态筛选，也应用到父评论查询
		if statusFilter != "" {
			parentQuery = parentQuery.Where("status = ?", statusFilter)
		}

		if err := parentQuery.First(&parent, *comment.ParentID).Error; err == nil {
			parentResp := &dto.CommentResponse{
				ID:           parent.ID,
				Content:      parent.Content,
				ArticleID:    parent.ArticleID,
				UserID:       parent.UserID,
				ParentID:     parent.ParentID,
				Status:       parent.Status,
				RejectReason: parent.RejectReason,
				CreatedAt:    parent.CreatedAt.Format("2006-01-02 15:04:05"),
				UpdatedAt:    parent.UpdatedAt.Format("2006-01-02 15:04:05"),
				LikeCount:    parent.LikeCount,
				LikedByMe:    false, // 默认为false
			}

			// 只有在用户已登录时才检查点赞状态
			if currentUserID != nil {
				parentResp.LikedByMe = s.hasUserLiked(*currentUserID, parent.ID)
			}

			if parent.User.ID > 0 {
				parentResp.User = dto.CommentUserInfo{
					ID:       parent.User.ID,
					Username: parent.User.Username,
					Avatar:   parent.User.Avatar,
				}
			}
			resp.Parent = parentResp
		}
	}

	// 获取子评论（回复）信息
	var children []model.Comment
	childQuery := s.db.Where("parent_id = ?", comment.ID)

	// 如果有状态筛选，应用到子评论查询
	if statusFilter != "" {
		childQuery = childQuery.Where("status = ?", statusFilter)
	}

	if err := childQuery.Preload("User").
		Order("created_at ASC").
		Limit(5). // 只取前5条回复
		Find(&children).Error; err == nil {

		resp.Children = make([]dto.CommentResponse, 0, len(children))
		for _, child := range children {
			childInfo := dto.CommentResponse{
				ID:           child.ID,
				Content:      child.Content,
				ArticleID:    child.ArticleID,
				UserID:       child.UserID,
				ParentID:     child.ParentID,
				Status:       child.Status,
				RejectReason: child.RejectReason,
				CreatedAt:    child.CreatedAt.Format("2006-01-02 15:04:05"),
				UpdatedAt:    child.UpdatedAt.Format("2006-01-02 15:04:05"),
				LikeCount:    child.LikeCount,
				LikedByMe:    false, // 默认为false
			}

			// 只有在用户已登录时才检查点赞状态
			if currentUserID != nil {
				childInfo.LikedByMe = s.hasUserLiked(*currentUserID, child.ID)
			}

			// 检查用户信息是否有效
			if child.User.ID > 0 {
				childInfo.User = dto.CommentUserInfo{
					ID:       child.User.ID,
					Username: child.User.Username,
					Avatar:   child.User.Avatar,
				}
			} else {
				// 如果User信息无效，手动查询
				var user model.User
				if err := s.db.First(&user, child.UserID).Error; err == nil {
					childInfo.User = dto.CommentUserInfo{
						ID:       user.ID,
						Username: user.Username,
						Avatar:   user.Avatar,
					}
				} else {
					s.logger.Warnf("获取评论用户信息失败: %v", err)
				}
			}

			resp.Children = append(resp.Children, childInfo)
		}
	}

	return resp, nil
}

// hasUserLiked 判断用户是否已点赞评论
func (s *CommentService) hasUserLiked(userID uint, commentID uint) bool {
	var count int64
	if err := s.db.Model(&model.CommentLike{}).
		Where("user_id = ? AND comment_id = ?", userID, commentID).
		Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

// FilterContent 过滤评论内容中的敏感词
func (s *CommentService) FilterContent(content string) string {
	sensitiveService := NewSensitiveService()
	return sensitiveService.FilterSensitiveWords(content)
}

// GetNotifications 获取评论通知
func (s *CommentService) GetNotifications(userID uint, req *dto.CommentNotificationListRequest) (*dto.CommentNotificationListResponse, error) {
	notificationService := NewNotificationService()

	// 转换请求参数
	notifyReq := &dto.NotificationListRequest{
		Page:     req.Page,
		PageSize: req.PageSize,
		Type:     "comment", // 只获取评论相关通知
	}
	if req.IsRead != nil {
		notifyReq.IsRead = req.IsRead
	}

	notifications, err := notificationService.GetUserNotifications(userID, notifyReq)
	if err != nil {
		return nil, fmt.Errorf("获取评论通知失败: %w", err)
	}

	// 转换为评论通知响应格式
	list := make([]dto.CommentNotificationResponse, 0, len(notifications.List))
	for _, notification := range notifications.List {
		if notification.Article != nil && notification.Comment != nil {
			commentNotify := dto.CommentNotificationResponse{
				ID:           notification.ID,
				ArticleID:    notification.Article.ID,
				ArticleTitle: notification.Article.Title,
				CommentID:    notification.Comment.ID,
				Content:      notification.Comment.Content,
				CreatedAt:    notification.CreatedAt,
				IsRead:       notification.IsRead,
			}

			if notification.Sender != nil {
				commentNotify.UserID = notification.Sender.ID
				commentNotify.User = dto.CommentUserInfo{
					ID:       notification.Sender.ID,
					Username: notification.Sender.Username,
					Avatar:   notification.Sender.Avatar,
				}
			}

			list = append(list, commentNotify)
		}
	}

	return &dto.CommentNotificationListResponse{
		Total:       notifications.Total,
		UnreadCount: notifications.UnreadCount,
		List:        list,
	}, nil
}

// MarkNotificationAsRead 标记评论通知为已读
func (s *CommentService) MarkNotificationAsRead(userID uint, notificationID uint) error {
	notificationService := NewNotificationService()
	return notificationService.MarkAsRead(userID, notificationID)
}

// MarkAllNotificationsAsRead 标记所有评论通知为已读
func (s *CommentService) MarkAllNotificationsAsRead(userID uint) error {
	notificationService := NewNotificationService()
	return notificationService.MarkAllAsRead(userID)
}

// createCommentNotification 创建评论通知
func (s *CommentService) createCommentNotification(senderID uint, comment *model.Comment, article *model.Article) {
	notificationService := NewNotificationService()

	// 如果是回复评论，通知父评论作者
	if comment.ParentID != nil {
		var parentComment model.Comment
		if err := s.db.Preload("User").First(&parentComment, *comment.ParentID).Error; err == nil {
			// 回复通知
			notificationService.CreateCommentReplyNotification(
				senderID,
				parentComment.UserID,
				article.ID,
				comment.ID,
				article.Title,
			)
		}
	}

	// 通知文章作者（如果不是自己评论自己的文章，且不是回复）
	if article.AuthorID != senderID && comment.ParentID == nil {
		notificationService.CreateCommentNotification(
			senderID,
			article.AuthorID,
			article.ID,
			comment.ID,
			article.Title,
		)
	}
}

// createCommentLikeNotification 创建评论点赞通知
func (s *CommentService) createCommentLikeNotification(senderID uint, comment *model.Comment) {
	// 不给自己发通知
	if senderID == comment.UserID {
		return
	}

	notificationService := NewNotificationService()

	// 获取文章信息
	var article model.Article
	if err := s.db.Select("id, title, author_id").First(&article, comment.ArticleID).Error; err != nil {
		s.logger.Warnf("获取文章信息失败: %v", err)
		return
	}

	// 创建评论点赞通知
	notificationService.CreateCommentLikeNotification(
		senderID,
		comment.UserID,
		article.ID,
		comment.ID,
		article.Title,
	)
}
