package models

import (
	"blog/global"
)

type CategoryModel struct {
	MODEL `json:","`
	Name  string `json:"name"`
}

// Create 创建分类
func (c *CategoryModel) Create() error {
	return global.DB.Create(c).Error
}

// Delete 删除分类
func (c *CategoryModel) Delete() error {
	return global.DB.Delete(c).Error
}
