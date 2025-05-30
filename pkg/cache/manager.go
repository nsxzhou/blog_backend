package cache

import (
	"context"
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"
)

// Manager 缓存管理器
type Manager struct {
	cache          Cache
	bloomManager   *BloomFilterManager
	articleCache   *ArticleCacheService
	mutex          sync.RWMutex
	initialized    bool
}

var (
	instance *Manager
	once     sync.Once
)

// GetManager 获取缓存管理器单例
func GetManager() *Manager {
	once.Do(func() {
		instance = &Manager{}
	})
	return instance
}

// Initialize 初始化缓存管理器
func (m *Manager) Initialize(redisClient *redis.Client) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if m.initialized {
		return nil
	}
	
	// 创建Redis缓存实例
	m.cache = NewRedisCache(redisClient)
	
	// 创建布隆过滤器管理器
	m.bloomManager = NewBloomFilterManager(redisClient)
	
	// 初始化布隆过滤器（从Redis加载）
	ctx := context.Background()
	if err := m.bloomManager.InitializeFilters(ctx); err != nil {
		return fmt.Errorf("initialize bloom filters failed: %w", err)
	}
	
	// 创建文章缓存服务
	m.articleCache = NewArticleCacheService(m.cache, m.bloomManager.GetArticleFilter())
	
	m.initialized = true
	return nil
}

// GetCache 获取基础缓存接口
func (m *Manager) GetCache() Cache {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.cache
}

// GetArticleCache 获取文章缓存服务
func (m *Manager) GetArticleCache() *ArticleCacheService {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.articleCache
}

// GetBloomManager 获取布隆过滤器管理器
func (m *Manager) GetBloomManager() *BloomFilterManager {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.bloomManager
}

// SaveBloomFilters 保存布隆过滤器到Redis
func (m *Manager) SaveBloomFilters(ctx context.Context) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	if !m.initialized {
		return fmt.Errorf("cache manager not initialized")
	}
	
	return m.bloomManager.SaveFilters(ctx)
}

// IsInitialized 检查是否已初始化
func (m *Manager) IsInitialized() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.initialized
}

// Close 关闭缓存连接
func (m *Manager) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if !m.initialized {
		return nil
	}
	
	// 保存布隆过滤器
	ctx := context.Background()
	if err := m.bloomManager.SaveFilters(ctx); err != nil {
		// 记录错误但不阻止关闭过程
		fmt.Printf("Failed to save bloom filters: %v\n", err)
	}
	
	// 关闭缓存连接
	if err := m.cache.Close(); err != nil {
		return fmt.Errorf("close cache failed: %w", err)
	}
	
	m.initialized = false
	return nil
}

// WarmUpArticleBloomFilter 预热文章布隆过滤器
func (m *Manager) WarmUpArticleBloomFilter(ctx context.Context, articleIDs []uint) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	if !m.initialized {
		return fmt.Errorf("cache manager not initialized")
	}
	
	return m.articleCache.BatchAddArticlesToBloomFilter(ctx, articleIDs)
} 