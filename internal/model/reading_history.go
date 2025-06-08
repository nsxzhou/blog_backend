package model

import (
	"time"
)

// ReadingHistory 用户阅读历史记录模型
type ReadingHistory struct {
	Base
	UserID    uint      `gorm:"type:int(11);not null;index:idx_user_article,priority:1" json:"user_id"`          // 用户ID
	ArticleID uint      `gorm:"type:int(11);not null;index:idx_user_article,priority:2;index" json:"article_id"` // 文章ID
	ReadAt    time.Time `gorm:"not null;index" json:"read_at"`                                                   // 最后阅读时间
	IP        string    `gorm:"type:varchar(50)" json:"ip"`                                                      // IP地址

	// 关联
	User    User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Article Article `gorm:"foreignKey:ArticleID" json:"article,omitempty"`
}

// TableName 指定表名
func (ReadingHistory) TableName() string {
	return "reading_histories"
}
