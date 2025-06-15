package cache

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/redis/go-redis/v9"
)

// BloomFilter 布隆过滤器接口
type BloomFilter interface {
	// Add 添加元素到布隆过滤器
	Add(ctx context.Context, element string) error

	// Test 测试元素是否可能存在
	Test(ctx context.Context, element string) (bool, error)

	// BatchAdd 批量添加元素
	BatchAdd(ctx context.Context, elements []string) error

	// Reset 重置布隆过滤器
	Reset(ctx context.Context) error

	// SaveToRedis 保存布隆过滤器到Redis
	SaveToRedis(ctx context.Context) error

	// LoadFromRedis 从Redis加载布隆过滤器
	LoadFromRedis(ctx context.Context) error
}

// RedisBloomFilter 基于Redis的布隆过滤器实现
type RedisBloomFilter struct {
	filter    *bloom.BloomFilter
	redisKey  string
	client    *redis.Client
	mutex     sync.RWMutex
	capacity  uint    // 预期元素数量
	errorRate float64 // 误判率
}

// NewRedisBloomFilter 创建Redis布隆过滤器
func NewRedisBloomFilter(client *redis.Client, redisKey string, capacity uint, errorRate float64) BloomFilter {
	filter := bloom.NewWithEstimates(capacity, errorRate)

	return &RedisBloomFilter{
		filter:    filter,
		redisKey:  redisKey,
		client:    client,
		capacity:  capacity,
		errorRate: errorRate,
	}
}

// Add 添加元素到布隆过滤器
func (bf *RedisBloomFilter) Add(ctx context.Context, element string) error {
	bf.mutex.Lock()
	defer bf.mutex.Unlock()

	bf.filter.AddString(element)
	return nil
}

// Test 测试元素是否可能存在
func (bf *RedisBloomFilter) Test(ctx context.Context, element string) (bool, error) {
	bf.mutex.RLock()
	defer bf.mutex.RUnlock()

	return bf.filter.TestString(element), nil
}

// BatchAdd 批量添加元素
func (bf *RedisBloomFilter) BatchAdd(ctx context.Context, elements []string) error {
	bf.mutex.Lock()
	defer bf.mutex.Unlock()

	for _, element := range elements {
		bf.filter.AddString(element)
	}

	return nil
}

// Reset 重置布隆过滤器
func (bf *RedisBloomFilter) Reset(ctx context.Context) error {
	bf.mutex.Lock()
	defer bf.mutex.Unlock()

	bf.filter.ClearAll()

	// 从Redis中删除
	return bf.client.Del(ctx, bf.redisKey).Err()
}

// SaveToRedis 保存布隆过滤器到Redis
func (bf *RedisBloomFilter) SaveToRedis(ctx context.Context) error {
	bf.mutex.RLock()
	defer bf.mutex.RUnlock()

	// 获取布隆过滤器的二进制数据
	data, err := bf.filter.GobEncode()
	if err != nil {
		return fmt.Errorf("encode bloom filter failed: %w", err)
	}

	// Base64编码后存储到Redis
	encoded := base64.StdEncoding.EncodeToString(data)
	return bf.client.Set(ctx, bf.redisKey, encoded, BloomFilterExpiration).Err()
}

// LoadFromRedis 从Redis加载布隆过滤器
func (bf *RedisBloomFilter) LoadFromRedis(ctx context.Context) error {
	bf.mutex.Lock()
	defer bf.mutex.Unlock()

	// 从Redis获取数据
	encoded, err := bf.client.Get(ctx, bf.redisKey).Result()
	if err != nil {
		if err == redis.Nil {
			// 如果不存在，创建新的布隆过滤器
			bf.filter = bloom.NewWithEstimates(bf.capacity, bf.errorRate)
			return nil
		}
		return fmt.Errorf("get bloom filter from redis failed: %w", err)
	}

	// Base64解码
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("decode bloom filter data failed: %w", err)
	}

	// 反序列化布隆过滤器
	filter := &bloom.BloomFilter{}
	if err := filter.GobDecode(data); err != nil {
		return fmt.Errorf("decode bloom filter failed: %w", err)
	}

	bf.filter = filter
	return nil
}

// BloomFilterManager 布隆过滤器管理器
type BloomFilterManager struct {
	client        *redis.Client
	articleFilter BloomFilter
	userFilter    BloomFilter
}

// NewBloomFilterManager 创建布隆过滤器管理器
func NewBloomFilterManager(client *redis.Client) *BloomFilterManager {
	return &BloomFilterManager{
		client:        client,
		articleFilter: NewRedisBloomFilter(client, BloomFilterArticleKey, 1000000, 0.01), // 100万文章，1%误判率
		userFilter:    NewRedisBloomFilter(client, BloomFilterUserKey, 100000, 0.01),     // 10万用户，1%误判率
	}
}

// GetArticleFilter 获取文章布隆过滤器
func (m *BloomFilterManager) GetArticleFilter() BloomFilter {
	return m.articleFilter
}

// GetUserFilter 获取用户布隆过滤器
func (m *BloomFilterManager) GetUserFilter() BloomFilter {
	return m.userFilter
}

// InitializeFilters 初始化布隆过滤器（从Redis加载或创建新的）
func (m *BloomFilterManager) InitializeFilters(ctx context.Context) error {
	// 加载文章布隆过滤器
	if err := m.articleFilter.LoadFromRedis(ctx); err != nil {
		return fmt.Errorf("load article bloom filter failed: %w", err)
	}

	// 加载用户布隆过滤器
	if err := m.userFilter.LoadFromRedis(ctx); err != nil {
		return fmt.Errorf("load user bloom filter failed: %w", err)
	}

	return nil
}

// SaveFilters 保存所有布隆过滤器到Redis
func (m *BloomFilterManager) SaveFilters(ctx context.Context) error {
	// 保存文章布隆过滤器
	if err := m.articleFilter.SaveToRedis(ctx); err != nil {
		return fmt.Errorf("save article bloom filter failed: %w", err)
	}

	// 保存用户布隆过滤器
	if err := m.userFilter.SaveToRedis(ctx); err != nil {
		return fmt.Errorf("save user bloom filter failed: %w", err)
	}

	return nil
}
