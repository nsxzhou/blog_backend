package dto

// CommentCreateRequest 创建评论请求
type CommentCreateRequest struct {
	Content   string `json:"content" binding:"required,min=1,max=1000"`
	ArticleID uint   `json:"article_id" binding:"required"`
	ParentID  *uint  `json:"parent_id"`
}

// CommentUpdateRequest 更新评论请求（管理员使用）
type CommentUpdateRequest struct {
	Content string `json:"content" binding:"required,min=1,max=1000"`
	Status  string `json:"status" binding:"omitempty,oneof=pending approved rejected"`
}

// CommentReplyRequest 回复评论请求
type CommentReplyRequest struct {
	Content   string `json:"content" binding:"required,min=1,max=1000"`
	CommentID uint   `json:"comment_id" binding:"required"`
}

// CommentResponse 评论响应
type CommentResponse struct {
	ID           uint              `json:"id"`
	Content      string            `json:"content"`
	ArticleID    uint              `json:"article_id"`
	UserID       uint              `json:"user_id"`
	ParentID     *uint             `json:"parent_id"`
	Status       string            `json:"status"`
	RejectReason string            `json:"reject_reason,omitempty"`
	CreatedAt    string            `json:"created_at"`
	UpdatedAt    string            `json:"updated_at"`
	User         CommentUserInfo   `json:"user"`
	Parent       *CommentResponse  `json:"parent,omitempty"`
	Children     []CommentResponse `json:"children,omitempty"`
	LikeCount    int               `json:"like_count"`
	LikedByMe    bool              `json:"liked_by_me"`
}

// CommentUserInfo 评论用户信息
type CommentUserInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}

// CommentBriefInfo 评论简要信息
type CommentBriefInfo struct {
	ID           uint            `json:"id"`
	Content      string          `json:"content"`
	UserID       uint            `json:"user_id"`
	Status       string          `json:"status"`
	RejectReason string          `json:"reject_reason,omitempty"`
	CreatedAt    string          `json:"created_at"`
	User         CommentUserInfo `json:"user"`
	LikeCount    int             `json:"like_count"`
}

// CommentListRequest 评论列表请求
type CommentListRequest struct {
	Page      int    `form:"page" binding:"omitempty,min=1"`
	PageSize  int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	ArticleID *uint  `form:"article_id"`
	UserID    *uint  `form:"user_id"`
	Status    string `form:"status" binding:"omitempty,oneof=pending approved rejected"`
	ParentID  *uint  `form:"parent_id"`
	OrderBy   string `form:"order_by" binding:"omitempty,oneof=id created_at like_count"`
	Order     string `form:"order" binding:"omitempty,oneof=asc desc"`
}

// CommentListResponse 评论列表响应
type CommentListResponse struct {
	Total int64             `json:"total"`
	List  []CommentResponse `json:"list"`
}

// CommentStatusUpdateRequest 更新评论状态请求（管理员使用）
type CommentStatusUpdateRequest struct {
	Status string `json:"status" binding:"required,oneof=pending approved rejected"`
}

// CommentBatchStatusUpdateRequest 批量更新评论状态请求
type CommentBatchStatusUpdateRequest struct {
	IDs    []uint `json:"ids" binding:"required,min=1"`
	Status string `json:"status" binding:"required,oneof=pending approved rejected"`
}

// CommentLikeRequest 评论点赞请求
type CommentLikeRequest struct {
	CommentID uint `json:"comment_id" binding:"required"`
}

// CommentNotificationResponse 评论通知响应
type CommentNotificationResponse struct {
	ID           uint            `json:"id"`
	ArticleID    uint            `json:"article_id"`
	ArticleTitle string          `json:"article_title"`
	CommentID    uint            `json:"comment_id"`
	Content      string          `json:"content"`
	UserID       uint            `json:"user_id"`
	User         CommentUserInfo `json:"user"`
	CreatedAt    string          `json:"created_at"`
	IsRead       bool            `json:"is_read"`
}

// CommentNotificationListRequest 评论通知列表请求
type CommentNotificationListRequest struct {
	Page     int   `form:"page" binding:"omitempty,min=1"`
	PageSize int   `form:"page_size" binding:"omitempty,min=1,max=100"`
	IsRead   *bool `form:"is_read"`
}

// CommentNotificationListResponse 评论通知列表响应
type CommentNotificationListResponse struct {
	Total       int64                         `json:"total"`
	UnreadCount int64                         `json:"unread_count"`
	List        []CommentNotificationResponse `json:"list"`
}
