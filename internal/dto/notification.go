package dto

// NotificationListRequest 通知列表请求
type NotificationListRequest struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Type     string `form:"type" binding:"omitempty,oneof=article_like article_favorite comment comment_reply comment_like follow"`
	IsRead   *bool  `form:"is_read"`
}

// NotificationResponse 通知响应
type NotificationResponse struct {
	ID        uint                        `json:"id"`
	Type      string                      `json:"type"`
	Content   string                      `json:"content"`
	IsRead    bool                        `json:"is_read"`
	CreatedAt string                      `json:"created_at"`
	UpdatedAt string                      `json:"updated_at"`
	Sender    *NotificationUserInfo       `json:"sender,omitempty"`
	Article   *NotificationArticleInfo    `json:"article,omitempty"`
	Comment   *NotificationCommentInfo    `json:"comment,omitempty"`
}

// NotificationListResponse 通知列表响应
type NotificationListResponse struct {
	Total       int64                  `json:"total"`
	UnreadCount int64                  `json:"unread_count"`
	List        []NotificationResponse `json:"list"`
}

// NotificationUserInfo 通知中的用户信息
type NotificationUserInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

// NotificationArticleInfo 通知中的文章信息
type NotificationArticleInfo struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
	Slug  string `json:"slug"`
}

// NotificationCommentInfo 通知中的评论信息
type NotificationCommentInfo struct {
	ID      uint   `json:"id"`
	Content string `json:"content"`
}

// NotificationUnreadCountResponse 未读通知数量响应
type NotificationUnreadCountResponse struct {
	Count int64 `json:"count"`
}

// NotificationMarkAsReadRequest 标记通知已读请求
type NotificationMarkAsReadRequest struct {
	NotificationID uint `json:"notification_id" binding:"required"`
}

// NotificationBatchDeleteRequest 批量删除通知请求
type NotificationBatchDeleteRequest struct {
	NotificationIDs []uint `json:"notification_ids" binding:"required,min=1"`
} 