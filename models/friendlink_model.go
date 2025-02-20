package models

import (

	"blog/global"

)

type FriendLinkModel struct {
	MODEL `json:","`
	Name  string `json:"name"`
	Link  string `json:"link"`
}

// Create 创建友链
func (c *FriendLinkModel) Create() error {
	return global.DB.Create(c).Error
}

// Delete 删除友链
func (c *FriendLinkModel) Delete() error {
	return global.DB.Delete(c).Error
}
