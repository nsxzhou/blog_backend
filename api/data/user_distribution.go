package data

import (
	"blog/global"
	"blog/models"
	"blog/models/res"

	"github.com/gin-gonic/gin"
)

// UserDistribution 用户分布数据结构
type UserDistribution struct {
	Name  string `json:"name"`  // 地区名称
	Value int64  `json:"value"` // 用户数量
}

// GetUserDistribution 获取用户地理分布数据
func (d *Data) GetUserDistribution(c *gin.Context) {
	// 构建查询条件
	query := global.DB.Model(&models.VisitModel{})

	// 使用数据库聚合函数直接统计
	var result []UserDistribution
	err := query.Select("distribution as name, COUNT(DISTINCT visitor_id) as value").
		Where("distribution != ''").
		Group("distribution").
		Find(&result).Error

	if err != nil {
		res.Error(c, res.ServerError, "获取用户分布数据失败")
		return
	}

	res.Success(c, result)
}
