package model

import (
	"time"
)

// User 用户模型
type User struct {
	Base
	Username             string    `gorm:"type:varchar(50);not null;uniqueIndex" json:"username"`
	Password             string    `gorm:"type:varchar(100);not null" json:"-"`
	Email                string    `gorm:"type:varchar(100);not null;uniqueIndex" json:"email"`
	Avatar               string    `gorm:"type:varchar(255)" json:"avatar"`
	Nickname             string    `gorm:"type:varchar(50)" json:"nickname"`
	Bio                  string    `gorm:"type:text" json:"bio"`
	Role                 string    `gorm:"type:varchar(20);not null;default:'user'" json:"role"`
	Status               int       `gorm:"type:tinyint(2);not null;default:1" json:"status"`           // 0=禁用 1=正常
	LastLoginAt          time.Time `json:"last_login_at"`
	LastLoginIP          string    `gorm:"type:varchar(50)" json:"last_login_ip"`
	IsVerified           int       `gorm:"type:tinyint(1);not null;default:0" json:"is_verified"` // 0=未验证 1=已验证
	VerificationToken    string    `gorm:"type:varchar(100)" json:"-"`
	ResetPasswordToken   string    `gorm:"type:varchar(100)" json:"-"`
	ResetPasswordExpires time.Time `json:"-"`
	Phone                string    `gorm:"type:varchar(20);uniqueIndex" json:"phone"`
	IsPhoneVerified      int       `gorm:"type:tinyint(1);not null;default:0" json:"is_phone_verified"` // 0=未验证 1=已验证
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}
