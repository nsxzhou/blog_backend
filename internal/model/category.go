package model

// Category 分类模型
type Category struct {
	Base
	Name         string `gorm:"type:varchar(50);not null;uniqueIndex" json:"name"`
	Description  string `gorm:"type:text" json:"description"`
	Icon         string `gorm:"type:varchar(255)" json:"icon"`
	ArticleCount int    `gorm:"type:int(11);not null;default:0" json:"article_count"`
	IsVisible    int    `gorm:"type:tinyint(1);not null;default:1;index" json:"is_visible"` // 0=隐藏 1=显示

	// 关联
	Articles []*Article `gorm:"foreignKey:CategoryID" json:"articles,omitempty"`
}

// TableName 指定表名
func (Category) TableName() string {
	return "categories"
}
