package cache

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// InitializeCache 初始化缓存系统
func InitializeCache(redisClient *redis.Client, db *gorm.DB) error {
	// 获取缓存管理器
	manager := GetManager()
	
	// 初始化缓存管理器
	if err := manager.Initialize(redisClient); err != nil {
		return fmt.Errorf("initialize cache manager failed: %w", err)
	}
	
	// 预热布隆过滤器
	if err := warmUpBloomFilters(manager, db); err != nil {
		return fmt.Errorf("warm up bloom filters failed: %w", err)
	}
	
	return nil
}

// warmUpBloomFilters 预热布隆过滤器
func warmUpBloomFilters(manager *Manager, db *gorm.DB) error {
	ctx := context.Background()
	
	// 预热文章布隆过滤器
	if err := warmUpArticleBloomFilter(ctx, manager, db); err != nil {
		return fmt.Errorf("warm up article bloom filter failed: %w", err)
	}
	
	// 预热用户布隆过滤器
	if err := warmUpUserBloomFilter(ctx, manager, db); err != nil {
		return fmt.Errorf("warm up user bloom filter failed: %w", err)
	}
	
	return nil
}

// warmUpArticleBloomFilter 预热文章布隆过滤器
func warmUpArticleBloomFilter(ctx context.Context, manager *Manager, db *gorm.DB) error {
	// 获取所有文章ID
	var articleIDs []uint
	if err := db.Model(&struct {
		ID uint `gorm:"column:id"`
	}{}).Table("articles").Pluck("id", &articleIDs).Error; err != nil {
		return fmt.Errorf("get article ids failed: %w", err)
	}
	
	if len(articleIDs) == 0 {
		return nil
	}
	
	// 批量添加到布隆过滤器
	bloomManager := manager.GetBloomManager()
	articleFilter := bloomManager.GetArticleFilter()
	
	// 转换为字符串数组
	elements := make([]string, len(articleIDs))
	for i, id := range articleIDs {
		elements[i] = strconv.FormatUint(uint64(id), 10)
	}
	
	// 批量添加
	if err := articleFilter.BatchAdd(ctx, elements); err != nil {
		return fmt.Errorf("batch add articles to bloom filter failed: %w", err)
	}
	
	fmt.Printf("预热文章布隆过滤器完成，添加了 %d 个文章ID\n", len(articleIDs))
	return nil
}

// warmUpUserBloomFilter 预热用户布隆过滤器
func warmUpUserBloomFilter(ctx context.Context, manager *Manager, db *gorm.DB) error {
	// 获取所有用户ID
	var userIDs []uint
	if err := db.Model(&struct {
		ID uint `gorm:"column:id"`
	}{}).Table("users").Where("deleted_at IS NULL").Pluck("id", &userIDs).Error; err != nil {
		return fmt.Errorf("get user ids failed: %w", err)
	}
	
	if len(userIDs) == 0 {
		return nil
	}
	
	// 批量添加到布隆过滤器
	bloomManager := manager.GetBloomManager()
	userFilter := bloomManager.GetUserFilter()
	
	// 转换为字符串数组
	elements := make([]string, len(userIDs))
	for i, id := range userIDs {
		elements[i] = strconv.FormatUint(uint64(id), 10)
	}
	
	// 批量添加
	if err := userFilter.BatchAdd(ctx, elements); err != nil {
		return fmt.Errorf("batch add users to bloom filter failed: %w", err)
	}
	
	fmt.Printf("预热用户布隆过滤器完成，添加了 %d 个用户ID\n", len(userIDs))
	return nil
}

// CleanupCache 清理缓存资源
func CleanupCache() error {
	manager := GetManager()
	return manager.Close()
} 