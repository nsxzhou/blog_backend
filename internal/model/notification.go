package model

// Notification 通知模型
type Notification struct {
	Base
	UserID    uint   `gorm:"type:int(11);not null;index" json:"user_id"`
	SenderID  *uint  `gorm:"type:int(11);index" json:"sender_id"`
	ArticleID *uint  `gorm:"type:int(11);index" json:"article_id"`
	CommentID *uint  `gorm:"type:int(11);index" json:"comment_id"`
	Type      string `gorm:"type:varchar(20);not null;index" json:"type"` // 通知类型: like comment follow favorite
	Content   string `gorm:"type:text;not null" json:"content"`
	IsRead    int    `gorm:"type:tinyint(1);not null;default:0;index" json:"is_read"` // 0=未读 1=已读 2=全部

	// 关联
	User    User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Sender  *User    `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
	Article *Article `gorm:"foreignKey:ArticleID" json:"article,omitempty"`
	Comment *Comment `gorm:"foreignKey:CommentID" json:"comment,omitempty"`
}

// TableName 指定表名
func (Notification) TableName() string {
	return "notifications"
}
