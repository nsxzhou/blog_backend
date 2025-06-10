package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/nsxzhou1114/blog-api/internal/database"
	"github.com/nsxzhou1114/blog-api/internal/dto"
	"github.com/nsxzhou1114/blog-api/internal/logger"
	"github.com/nsxzhou1114/blog-api/internal/model"
	"github.com/nsxzhou1114/blog-api/pkg/websocket"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	notificationService     *NotificationService
	notificationServiceOnce sync.Once
)

// NotificationService 通知服务
type NotificationService struct {
	db              *gorm.DB
	logger          *zap.SugaredLogger
	websocketManager *websocket.Manager
}

// NewNotificationService 创建通知服务实例
func NewNotificationService() *NotificationService {
	notificationServiceOnce.Do(func() {
		notificationService = &NotificationService{
			db:              database.GetDB(),
			logger:          logger.GetSugaredLogger(),
			websocketManager: websocket.GetManager(),
		}
	})
	return notificationService
}

// CreateNotification 创建通知（异步处理）
func (s *NotificationService) CreateNotification(senderID, receiverID uint, notificationType string, content string, articleID, commentID *uint) {
	// 异步处理通知创建，避免阻塞主业务流程
	go func() {
		if err := s.createNotificationSync(senderID, receiverID, notificationType, content, articleID, commentID); err != nil {
			s.logger.Errorf("创建通知失败: %v", err)
		}
	}()
}

// createNotificationSync 同步创建通知
func (s *NotificationService) createNotificationSync(senderID, receiverID uint, notificationType string, content string, articleID, commentID *uint) error {
	// 避免给自己发送通知
	if senderID == receiverID {
		return nil
	}

	// 检查是否存在相同的未读通知（防重复）
	var count int64
	query := s.db.Model(&model.Notification{}).
		Where("user_id = ? AND sender_id = ? AND type = ? AND is_read = 0", receiverID, senderID, notificationType)
	
	if articleID != nil {
		query = query.Where("article_id = ?", *articleID)
	}
	if commentID != nil {
		query = query.Where("comment_id = ?", *commentID)
	}
	
	if err := query.Count(&count).Error; err != nil {
		return fmt.Errorf("检查重复通知失败: %w", err)
	}
	
	if count > 0 {
		// 存在相同未读通知，更新时间而不创建新通知
		return s.updateExistingNotification(receiverID, senderID, notificationType, content, articleID, commentID)
	}

	// 创建新通知
	notification := &model.Notification{
		UserID:    receiverID,
		SenderID:  &senderID,
		ArticleID: articleID,
		CommentID: commentID,
		Type:      notificationType,
		Content:   content,
		IsRead:    0,
	}

	if err := s.db.Create(notification).Error; err != nil {
		return fmt.Errorf("创建通知记录失败: %w", err)
	}

	// 预加载关联数据
	if err := s.loadNotificationAssociations(notification); err != nil {
		s.logger.Warnf("加载通知关联数据失败: %v", err)
	}

	// 实时推送通知
	if err := s.websocketManager.SendToUser(receiverID, notification); err != nil {
		s.logger.Warnf("推送通知失败: %v", err)
	}

	s.logger.Infof("通知创建成功: 用户%d -> 用户%d, 类型: %s", senderID, receiverID, notificationType)
	return nil
}

// updateExistingNotification 更新已存在的通知
func (s *NotificationService) updateExistingNotification(receiverID, senderID uint, notificationType string, content string, articleID, commentID *uint) error {
	query := s.db.Model(&model.Notification{}).
		Where("user_id = ? AND sender_id = ? AND type = ? AND is_read = 0", receiverID, senderID, notificationType)
	
	if articleID != nil {
		query = query.Where("article_id = ?", *articleID)
	}
	if commentID != nil {
		query = query.Where("comment_id = ?", *commentID)
	}

	updates := map[string]interface{}{
		"content":    content,
		"updated_at": time.Now(),
	}

	return query.Updates(updates).Error
}

// loadNotificationAssociations 加载通知关联数据
func (s *NotificationService) loadNotificationAssociations(notification *model.Notification) error {
	return s.db.Preload("User").Preload("Sender").
		Preload("Article").Preload("Comment").
		First(notification, notification.ID).Error
}

// GetUserNotifications 获取用户通知列表
func (s *NotificationService) GetUserNotifications(userID uint, req *dto.NotificationListRequest) (*dto.NotificationListResponse, error) {
	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	query := s.db.Model(&model.Notification{}).Where("user_id = ?", userID)

	// 根据类型过滤
	if req.Type != "" {
		query = query.Where("type = ?", req.Type)
	}

	// 根据已读状态过滤
	if req.IsRead != nil {
		isReadValue := 0
		if *req.IsRead {
			isReadValue = 1
		}
		query = query.Where("is_read = ?", isReadValue)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("获取通知总数失败: %w", err)
	}

	// 分页查询
	var notifications []model.Notification
	offset := (req.Page - 1) * req.PageSize
	err := query.Preload("User").Preload("Sender").
		Preload("Article").Preload("Comment").
		Order("created_at DESC").
		Offset(offset).Limit(req.PageSize).
		Find(&notifications).Error

	if err != nil {
		return nil, fmt.Errorf("查询通知列表失败: %w", err)
	}

	// 转换为响应格式
	list := make([]dto.NotificationResponse, 0, len(notifications))
	for _, notification := range notifications {
		list = append(list, s.convertToNotificationResponse(&notification))
	}

	// 获取未读通知数量
	var unreadCount int64
	s.db.Model(&model.Notification{}).
		Where("user_id = ? AND is_read = 0", userID).
		Count(&unreadCount)

	return &dto.NotificationListResponse{
		Total:       total,
		UnreadCount: unreadCount,
		List:        list,
	}, nil
}

// convertToNotificationResponse 转换为通知响应格式
func (s *NotificationService) convertToNotificationResponse(notification *model.Notification) dto.NotificationResponse {
	resp := dto.NotificationResponse{
		ID:        notification.ID,
		Type:      notification.Type,
		Content:   notification.Content,
		IsRead:    notification.IsRead == 1,
		CreatedAt: notification.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: notification.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	// 发送者信息
	if notification.Sender != nil {
		resp.Sender = &dto.NotificationUserInfo{
			ID:       notification.Sender.ID,
			Username: notification.Sender.Username,
			Nickname: notification.Sender.Nickname,
			Avatar:   notification.Sender.Avatar,
		}
	}

	// 文章信息
	if notification.Article != nil {
		resp.Article = &dto.NotificationArticleInfo{
			ID:    notification.Article.ID,
			Title: notification.Article.Title,
			Slug:  fmt.Sprintf("article-%d", notification.Article.ID), // 生成简单的slug
		}
	}

	// 评论信息
	if notification.Comment != nil {
		resp.Comment = &dto.NotificationCommentInfo{
			ID:      notification.Comment.ID,
			Content: notification.Comment.Content,
		}
	}

	return resp
}

// MarkAsRead 标记通知为已读
func (s *NotificationService) MarkAsRead(userID, notificationID uint) error {
	result := s.db.Model(&model.Notification{}).
		Where("id = ? AND user_id = ?", notificationID, userID).
		Update("is_read", 1)

	if result.Error != nil {
		return fmt.Errorf("标记通知已读失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("通知不存在或无权限")
	}

	return nil
}

// MarkAllAsRead 标记所有通知为已读
func (s *NotificationService) MarkAllAsRead(userID uint) error {
	if err := s.db.Model(&model.Notification{}).
		Where("user_id = ? AND is_read = 0", userID).
		Update("is_read", 1).Error; err != nil {
		return fmt.Errorf("标记所有通知已读失败: %w", err)
	}

	return nil
}

// DeleteNotification 删除通知
func (s *NotificationService) DeleteNotification(userID, notificationID uint) error {
	result := s.db.Where("id = ? AND user_id = ?", notificationID, userID).
		Delete(&model.Notification{})

	if result.Error != nil {
		return fmt.Errorf("删除通知失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("通知不存在或无权限")
	}

	return nil
}

// BatchDeleteNotifications 批量删除通知
func (s *NotificationService) BatchDeleteNotifications(userID uint, notificationIDs []uint) error {
	if len(notificationIDs) == 0 {
		return fmt.Errorf("请选择要删除的通知")
	}

	result := s.db.Where("id IN ? AND user_id = ?", notificationIDs, userID).
		Delete(&model.Notification{})

	if result.Error != nil {
		return fmt.Errorf("批量删除通知失败: %w", result.Error)
	}

	s.logger.Infof("用户 %d 批量删除了 %d 条通知", userID, result.RowsAffected)
	return nil
}

// GetUnreadCount 获取未读通知数量
func (s *NotificationService) GetUnreadCount(userID uint) (int64, error) {
	var count int64
	err := s.db.Model(&model.Notification{}).
		Where("user_id = ? AND is_read = 0", userID).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("获取未读通知数量失败: %w", err)
	}

	return count, nil
}

// CleanupReadNotifications 清理已读通知（定期任务）
func (s *NotificationService) CleanupReadNotifications(days int) error {
	if days <= 0 {
		days = 30 // 默认清理30天前的已读通知
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	result := s.db.Where("is_read = 1 AND updated_at < ?", cutoff).
		Delete(&model.Notification{})

	if result.Error != nil {
		return fmt.Errorf("清理已读通知失败: %w", result.Error)
	}

	s.logger.Infof("清理了 %d 条过期的已读通知", result.RowsAffected)
	return nil
}

// 便捷方法：创建各种类型的通知

// CreateArticleLikeNotification 创建文章点赞通知
func (s *NotificationService) CreateArticleLikeNotification(senderID, authorID, articleID uint, articleTitle string) {
	content := fmt.Sprintf("点赞了你的文章《%s》", articleTitle)
	s.CreateNotification(senderID, authorID, "article_like", content, &articleID, nil)
}

// CreateArticleFavoriteNotification 创建文章收藏通知
func (s *NotificationService) CreateArticleFavoriteNotification(senderID, authorID, articleID uint, articleTitle string) {
	content := fmt.Sprintf("收藏了你的文章《%s》", articleTitle)
	s.CreateNotification(senderID, authorID, "article_favorite", content, &articleID, nil)
}

// CreateCommentNotification 创建评论通知
func (s *NotificationService) CreateCommentNotification(senderID, authorID, articleID, commentID uint, articleTitle string) {
	content := fmt.Sprintf("评论了你的文章《%s》", articleTitle)
	s.CreateNotification(senderID, authorID, "comment", content, &articleID, &commentID)
}

// CreateCommentReplyNotification 创建评论回复通知
func (s *NotificationService) CreateCommentReplyNotification(senderID, receiverID, articleID, commentID uint, articleTitle string) {
	content := fmt.Sprintf("回复了你在《%s》的评论", articleTitle)
	s.CreateNotification(senderID, receiverID, "comment_reply", content, &articleID, &commentID)
}

// CreateCommentLikeNotification 创建评论点赞通知
func (s *NotificationService) CreateCommentLikeNotification(senderID, receiverID, articleID, commentID uint, articleTitle string) {
	content := fmt.Sprintf("点赞了你在《%s》的评论", articleTitle)
	s.CreateNotification(senderID, receiverID, "comment_like", content, &articleID, &commentID)
}

// CreateFollowNotification 创建关注通知
func (s *NotificationService) CreateFollowNotification(senderID, receiverID uint, senderNickname string) {
	content := fmt.Sprintf("%s 关注了你", senderNickname)
	s.CreateNotification(senderID, receiverID, "follow", content, nil, nil)
} 