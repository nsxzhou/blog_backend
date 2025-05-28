package model

// Setting 系统设置模型
type Setting struct {
	Base
	Key         string `gorm:"type:varchar(50);not null;uniqueIndex" json:"key"`
	Value       string `gorm:"type:text" json:"value"`
	Description string `gorm:"type:varchar(255)" json:"description"`
}

// TableName 指定表名
func (Setting) TableName() string {
	return "settings"
}
