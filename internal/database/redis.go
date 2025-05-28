package database

import (
	"context"
	"fmt"
	"sync"

	"github.com/nsxzhou1114/blog-api/internal/config"
	"github.com/nsxzhou1114/blog-api/internal/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Redis 全局Redis客户端实例
var (
	Redis    *redis.Client
	redisOne sync.Once
)

// InitRedis 初始化Redis连接
func InitRedis() (*redis.Client, error) {
	cfg := config.GlobalConfig.Redis

	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr(),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
	})

	// 测试连接
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("连接redis失败: %v", err)
	}

	logger.Info("redis连接成功", zap.String("addr", cfg.Addr()))
	return client, nil
}

// GetRedis 获取Redis客户端实例
func GetRedis() *redis.Client {
	var err error
	redisOne.Do(func() {
		Redis, err = InitRedis()
		if err != nil {
			panic(fmt.Sprintf("redis初始化失败: %v", err))
		}
	})
	return Redis
}
