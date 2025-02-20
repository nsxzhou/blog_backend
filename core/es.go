package core

import (
	"blog/global"

	"github.com/elastic/go-elasticsearch/v8"
	"go.uber.org/zap"
)

func InitEs() *elasticsearch.TypedClient {
	// 从配置中获取ES连接信息
	esConfig := global.Config.Es
	cfg := elasticsearch.Config{
		Addresses: []string{esConfig.Dsn()},
	}
	// 创建ES客户端
	es, err := elasticsearch.NewTypedClient(cfg)
	if err != nil {
		global.Log.Error("ES客户端创建失败",
			zap.String("dsn", esConfig.Dsn()),
			zap.String("error", err.Error()))
			return nil
	}
	global.Log.Info("ES连接成功", zap.String("method", "InitEs"), zap.String("path", "core/es.go"))

	return es
}
