package models

import (
	"fmt"

	"blog/global"

	"gorm.io/gorm"
)

type FriendLinkModel struct {
	MODEL `json:","`
	Name  string `json:"name"`
	Link  string `json:"link"`
}

// Create 创建友链
func (c *FriendLinkModel) Create() error {
	return global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(c).Error; err != nil {
			return fmt.Errorf("创建友链失败: %w", err)
		}
		return nil
	})
}

// Delete 删除友链
func (c *FriendLinkModel) Delete() error {
	return global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(c).Error; err != nil {
			return fmt.Errorf("删除友链失败: %w", err)
		}
		return nil
	})
}
