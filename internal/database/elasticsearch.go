package database

import (
	"context"
	"fmt"
	"sync"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/nsxzhou1114/blog-api/internal/config"
	"github.com/nsxzhou1114/blog-api/internal/logger"
	"go.uber.org/zap"
)

// ES 全局Elasticsearch客户端实例
var (
	ES    *elasticsearch.Client
	esOne sync.Once
)

// InitElasticsearch 初始化Elasticsearch连接
func InitElasticsearch() (*elasticsearch.Client, error) {
	cfg := config.GlobalConfig.Elasticsearch

	// 创建客户端配置
	esConfig := elasticsearch.Config{
		Addresses: cfg.URLs,
	}

	// 如果设置了用户名和密码，则添加基本认证
	if cfg.Username != "" && cfg.Password != "" {
		esConfig.Username = cfg.Username
		esConfig.Password = cfg.Password
	}

	// 创建客户端
	client, err := elasticsearch.NewClient(esConfig)
	if err != nil {
		return nil, fmt.Errorf("连接elasticsearch失败: %v", err)
	}

	// 检查连接
	ctx := context.Background()
	info, err := client.Info(client.Info.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("elasticsearch健康检查失败: %v", err)
	}

	logger.Info("elasticsearch连接成功",
		zap.String("status", info.String()),
		zap.Strings("addresses", cfg.URLs),
	)

	return client, nil
}

// GetES 获取Elasticsearch客户端实例
func GetES() *elasticsearch.Client {
	var err error
	esOne.Do(func() {
		ES, err = InitElasticsearch()
		if err != nil {
			panic(fmt.Sprintf("elasticsearch初始化失败: %v", err))
		}
	})
	return ES
}
