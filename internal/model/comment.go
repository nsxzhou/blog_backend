package model

// Comment 评论模型
type Comment struct {
	Base
	Content      string `gorm:"type:text;not null" json:"content"`
	ArticleID    uint   `gorm:"type:int(11);not null;index" json:"article_id"`
	UserID       uint   `gorm:"type:int(11);not null;index" json:"user_id"`
	ParentID     *uint  `gorm:"type:int(11);index" json:"parent_id"`
	Status       string `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"` // 状态: pending  approved rejected
	RejectReason string `gorm:"type:text" json:"reject_reason"`
	LikeCount    int    `gorm:"type:int(11);not null;default:0" json:"like_count"`

	// 关联
	Article  Article    `gorm:"foreignKey:ArticleID" json:"article,omitempty"`
	User     User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Parent   *Comment   `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []*Comment `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Likes    []*User    `gorm:"many2many:comment_likes;" json:"likes,omitempty"`
}

// TableName 指定表名
func (Comment) TableName() string {
	return "comments"
}

// CommentLike 评论点赞关联模型
type CommentLike struct {
	Base
	UserID    uint `gorm:"type:int(11);not null;index" json:"user_id"`
	CommentID uint `gorm:"type:int(11);not null;index" json:"comment_id"`

	// 关联
	User    User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Comment Comment `gorm:"foreignKey:CommentID" json:"comment,omitempty"`
}

// TableName 指定表名
func (CommentLike) TableName() string {
	return "comment_likes"
}
