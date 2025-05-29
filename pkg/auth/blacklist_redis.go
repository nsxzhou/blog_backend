package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nsxzhou1114/blog-api/internal/database"
	"github.com/nsxzhou1114/blog-api/internal/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisTokenBlacklist Redis令牌黑名单实现
type RedisTokenBlacklist struct {
	redis      *redis.Client
	localCache map[string]time.Time // 本地缓存，提高查询性能
	mutex      sync.RWMutex         // 本地缓存锁
	ctx        context.Context
}

var (
	redisBlacklist     *RedisTokenBlacklist
	redisBlacklistOnce sync.Once
)

const (
	// Redis键前缀
	blacklistKeyPrefix = "jwt:blacklist:"
	// 本地缓存同步间隔
	localCacheSyncInterval = 5 * time.Minute
	// 本地缓存最大条目数
	maxLocalCacheSize = 10000
)

// GetRedisTokenBlacklist 获取Redis令牌黑名单单例
func GetRedisTokenBlacklist() *RedisTokenBlacklist {
	redisBlacklistOnce.Do(func() {
		redisBlacklist = &RedisTokenBlacklist{
			redis:      database.GetRedis(),
			localCache: make(map[string]time.Time),
			ctx:        context.Background(),
		}
		// 启动本地缓存同步任务
		go redisBlacklist.syncLocalCache()
		// 启动定期清理任务
		go redisBlacklist.cleanupTask()
	})
	return redisBlacklist
}

// AddToBlacklist 将令牌添加到黑名单
func (b *RedisTokenBlacklist) AddToBlacklist(token string, expireAt time.Time) error {
	// 计算过期时间
	duration := time.Until(expireAt)
	if duration <= 0 {
		return nil // 已过期的令牌无需添加
	}

	// 添加到Redis
	key := blacklistKeyPrefix + token
	err := b.redis.Set(b.ctx, key, "1", duration).Err()
	if err != nil {
		logger.Error("添加令牌到Redis黑名单失败", 
			zap.String("token", token), 
			zap.Error(err))
		return fmt.Errorf("添加令牌到黑名单失败: %w", err)
	}

	// 添加到本地缓存
	b.mutex.Lock()
	defer b.mutex.Unlock()
	
	// 检查本地缓存大小，防止内存泄漏
	if len(b.localCache) >= maxLocalCacheSize {
		// 清理过期的本地缓存
		b.cleanupLocalCacheUnsafe()
	}
	
	b.localCache[token] = expireAt
	
	logger.Info("令牌已添加到黑名单", zap.String("token", token))
	return nil
}

// IsBlacklisted 检查令牌是否在黑名单中
func (b *RedisTokenBlacklist) IsBlacklisted(token string) bool {
	// 首先检查本地缓存
	b.mutex.RLock()
	expireAt, exists := b.localCache[token]
	b.mutex.RUnlock()
	
	if exists {
		// 检查是否过期
		if time.Now().After(expireAt) {
			// 从本地缓存中删除过期令牌
			b.mutex.Lock()
			delete(b.localCache, token)
			b.mutex.Unlock()
		} else {
			return true
		}
	}

	// 检查Redis
	key := blacklistKeyPrefix + token
	result, err := b.redis.Exists(b.ctx, key).Result()
	if err != nil {
		logger.Error("检查Redis黑名单失败", 
			zap.String("token", token), 
			zap.Error(err))
		// Redis异常时，仅依赖本地缓存
		return false
	}

	if result > 0 {
		// 获取过期时间并添加到本地缓存
		ttl := b.redis.TTL(b.ctx, key).Val()
		if ttl > 0 {
			expireAt := time.Now().Add(ttl)
			b.mutex.Lock()
			b.localCache[token] = expireAt
			b.mutex.Unlock()
		}
		return true
	}

	return false
}

// syncLocalCache 定期同步本地缓存
func (b *RedisTokenBlacklist) syncLocalCache() {
	ticker := time.NewTicker(localCacheSyncInterval)
	defer ticker.Stop()

	for range ticker.C {
		b.cleanupLocalCache()
	}
}

// cleanupLocalCache 清理本地缓存中的过期令牌
func (b *RedisTokenBlacklist) cleanupLocalCache() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.cleanupLocalCacheUnsafe()
}

// cleanupLocalCacheUnsafe 清理本地缓存中的过期令牌（不加锁版本）
func (b *RedisTokenBlacklist) cleanupLocalCacheUnsafe() {
	now := time.Now()
	for token, expireAt := range b.localCache {
		if now.After(expireAt) {
			delete(b.localCache, token)
		}
	}
}

// cleanupTask 定期清理Redis中的过期令牌（Redis会自动处理，这里主要用于监控）
func (b *RedisTokenBlacklist) cleanupTask() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		// 获取黑名单统计信息
		pattern := blacklistKeyPrefix + "*"
		keys, err := b.redis.Keys(b.ctx, pattern).Result()
		if err != nil {
			logger.Error("获取黑名单键失败", zap.Error(err))
			continue
		}

		b.mutex.RLock()
		localCount := len(b.localCache)
		b.mutex.RUnlock()

		logger.Info("黑名单统计", 
			zap.Int("redis_count", len(keys)),
			zap.Int("local_cache_count", localCount))
	}
}

// GetStats 获取黑名单统计信息
func (b *RedisTokenBlacklist) GetStats() (redisCount int, localCacheCount int, err error) {
	// 获取Redis中的数量
	pattern := blacklistKeyPrefix + "*"
	keys, err := b.redis.Keys(b.ctx, pattern).Result()
	if err != nil {
		return 0, 0, fmt.Errorf("获取Redis黑名单统计失败: %w", err)
	}

	// 获取本地缓存数量
	b.mutex.RLock()
	localCacheCount = len(b.localCache)
	b.mutex.RUnlock()

	return len(keys), localCacheCount, nil
}

// Clear 清空黑名单（谨慎使用）
func (b *RedisTokenBlacklist) Clear() error {
	// 清空Redis
	pattern := blacklistKeyPrefix + "*"
	keys, err := b.redis.Keys(b.ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("获取黑名单键失败: %w", err)
	}

	if len(keys) > 0 {
		err = b.redis.Del(b.ctx, keys...).Err()
		if err != nil {
			return fmt.Errorf("清空Redis黑名单失败: %w", err)
		}
	}

	// 清空本地缓存
	b.mutex.Lock()
	b.localCache = make(map[string]time.Time)
	b.mutex.Unlock()

	logger.Info("黑名单已清空")
	return nil
} 