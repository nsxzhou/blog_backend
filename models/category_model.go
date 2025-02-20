package models

import (
	"fmt"

	"blog/global"

	"gorm.io/gorm"
)

type CategoryModel struct {
	MODEL `json:","`
	Name  string `json:"name"`
}

// Create 创建分类
func (c *CategoryModel) Create() error {
	return global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(c).Error; err != nil {
			return fmt.Errorf("创建分类失败: %w", err)
		}
		return nil
	})
}

// Delete 删除分类
func (c *CategoryModel) Delete() error {
	return global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(c).Error; err != nil {
			return fmt.Errorf("删除分类失败: %w", err)
		}
		return nil
	})
}
