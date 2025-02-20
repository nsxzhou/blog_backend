package models

import (
	"blog/models/ctypes"

	"gorm.io/gorm"
)

type PageInfo struct {
	Page     int    `json:"page" form:"page" validate:"required,gt=0"`
	Key      string `json:"key" form:"key"`
	PageSize int    `json:"page_size" form:"page_size" validate:"required,gt=0"`
}

type MODEL struct {
	ID        uint           `gorm:"primaryKey;comment:id" json:"id" structs:"-"`
	CreatedAt ctypes.MyTime  `gorm:"type:datetime NOT NULL DEFAULT CURRENT_TIMESTAMP;comment:创建时间" json:"created_at" structs:"-"`
	UpdatedAt ctypes.MyTime  `gorm:"type:datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;comment:更新时间" json:"updated_at" structs:"-"`
	DeletedAt gorm.DeletedAt `gorm:"type:datetime NULL;index;comment:删除时间" json:"deleted_at" structs:"-"`
}

type IDRequest struct {
	ID uint `json:"id" uri:"id" form:"id" validate:"required,gt=0"`
}
