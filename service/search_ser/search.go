package search_ser

import (
	"fmt"

	"blog/global"
	"blog/models"

	"gorm.io/gorm"
)

// Option 查询选项结构体
type Option struct {
	models.PageInfo
	Likes   []string // 需要模糊匹配的字段列表
	Debug   bool     // 是否打印sql语句
	Where   *gorm.DB // 额外的查询条件
	Preload []string // 预加载的字段列表
	OrderBy string   // 排序字段，默认为 created_at desc
}

// buildLikeQuery 构建模糊查询条件
func buildLikeQuery(likes []string, key string) *gorm.DB {
	if key == "" || len(likes) == 0 {
		return nil
	}

	likeQuery := global.DB.Where("")
	for index, column := range likes {
		// 构建一个 OR 连接的模糊查询条件
		condition := fmt.Sprintf("%s LIKE ?", column)
		value := fmt.Sprintf("%%%s%%", key)
		if index == 0 {
			likeQuery = likeQuery.Where(condition, value)
		} else {
			likeQuery = likeQuery.Or(condition, value)
		}
	}
	return likeQuery
}

// ComList 通用列表查询函数
func ComList[T any](model T, option Option) (list []T, total int64, err error) {
	// 初始化查询构建器
	query := global.DB.Model(&model)

	// 开启调试模式

	if option.Debug {
		query = query.Debug()
	}

	// 设置默认页码
	if option.Page <= 0 {
		option.Page = 1
	}
	if option.PageSize <= 0 {
		option.PageSize = 100
	}
	// 设置默认排序
	if option.OrderBy == "" {
		option.OrderBy = "created_at desc"
	}

	// 构建查询条件
	query = query.Where(model)

	if option.Where != nil {
		query = query.Where(option.Where)
	}

	// 添加模糊查询条件
	if likeQuery := buildLikeQuery(option.Likes, option.Key); likeQuery != nil {
		query = query.Where(likeQuery)
	}

	// 获取总记录数
	if err = query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计总记录数失败: %w", err)
	}

	// 添加预加载关系
	for _, preload := range option.Preload {
		query = query.Preload(preload)
	}

	// 执行分页查询
	offset := (option.Page - 1) * option.PageSize
	err = query.
		Limit(option.PageSize).
		Offset(offset).
		Order(option.OrderBy).
		Find(&list).Error

	if err != nil {
		return nil, 0, fmt.Errorf("执行查询失败: %w", err)
	}

	return list, total, nil
}
