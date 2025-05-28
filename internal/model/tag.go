package model

// Tag 标签模型
type Tag struct {
	Base
	Name         string `gorm:"type:varchar(50);not null;uniqueIndex" json:"name"`
	ArticleCount int    `gorm:"type:int(11);not null;default:0" json:"article_count"`

	// 关联
	Articles []*Article `gorm:"many2many:article_tags;" json:"articles,omitempty"`
}

// TableName 指定表名
func (Tag) TableName() string {
	return "tags"
}

// ArticleTag 文章-标签关联模型
type ArticleTag struct {
	ArticleID uint `gorm:"primaryKey;type:int(11);not null" json:"article_id"`
	TagID     uint `gorm:"primaryKey;type:int(11);not null" json:"tag_id"`
}

// TableName 指定表名
func (ArticleTag) TableName() string {
	return "article_tags"
}
