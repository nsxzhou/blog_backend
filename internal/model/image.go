package model

// Image 图片模型
type Image struct {
	Base
	URL         string `gorm:"type:varchar(255);not null" json:"url"`
	Path        string `gorm:"type:varchar(255);not null" json:"path"`
	Filename    string `gorm:"type:varchar(100);not null" json:"filename"`
	Size        int    `gorm:"type:int(11);not null" json:"size"`
	Width       *int   `gorm:"type:int(11)" json:"width"`
	Height      *int   `gorm:"type:int(11)" json:"height"`
	MimeType    string `gorm:"type:varchar(50);not null" json:"mime_type"`
	UserID      uint   `gorm:"type:int(11);not null;index" json:"user_id"`
	UsageType   string `gorm:"type:varchar(20);index" json:"usage_type"` // 使用类型: avatar/cover/content
	ArticleID   *uint  `gorm:"type:int(11);index" json:"article_id"`
	IsExternal  int    `gorm:"type:tinyint(1);not null;default:0" json:"is_external"`               // 是否外链: 0=否 1=是
	StorageType string `gorm:"type:varchar(20);not null;default:'local';index" json:"storage_type"` // 存储类型: local/cos

	// 关联
	User    User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Article *Article `gorm:"foreignKey:ArticleID" json:"article,omitempty"`
}

// TableName 指定表名
func (Image) TableName() string {
	return "images"
}
