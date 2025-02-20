package core

import (
	"blog/global"
	"context"

	"go.uber.org/zap"

	"github.com/redis/go-redis/v9"
)

// InitRedis 初始化Redis
func InitRedis() *redis.Client {
	// 获取Redis配置
	redisConf := global.Config.Redis

	// 创建Redis客户端配置
	opts := &redis.Options{
		Addr:     redisConf.Addr(),
		Password: redisConf.Password,
		DB:       redisConf.DB,
	}
	// 创建Redis客户端
	rdb := redis.NewClient(opts)
	// 测试连接
	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		global.Log.Error("Redis连接失败", zap.String("addr", redisConf.Addr()), zap.String("error", err.Error()))
		return nil
	}
	global.Log.Info("Redis连接成功", zap.String("method", "InitRedis"), zap.String("path", "core/redis.go"))
	return rdb

}
