package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nsxzhou1114/blog-api/internal/config"
	"github.com/nsxzhou1114/blog-api/internal/database"
	"github.com/nsxzhou1114/blog-api/internal/logger"
	"github.com/nsxzhou1114/blog-api/internal/model"
	"github.com/nsxzhou1114/blog-api/internal/router"
	"go.uber.org/zap"
)

// Execute 执行主程序
func Execute() {
	// 初始化配置
	if err := config.Init("./config"); err != nil {
		panic(fmt.Sprintf("配置初始化失败: %v", err))
	}

	// 初始化日志
	if err := logger.Init(); err != nil {
		panic(fmt.Sprintf("日志初始化失败: %v", err))
	}
	defer logger.Sync()

	// 初始化MySQL数据库
	db := database.GetDB()
	if db == nil {
		logger.Fatal("MySQL数据库连接失败")
		return
	}

	// 初始化数据库表
	if err := model.InitTables(db); err != nil {
		logger.Fatal("初始化数据库表失败", zap.Error(err))
		return
	}

	// 初始化Elasticsearch
	es := database.GetES()
	if es == nil {
		logger.Fatal("Elasticsearch连接失败")
		return
	}

	// 初始化Elasticsearch索引
	if err := model.InitESIndices(es); err != nil {
		logger.Fatal("初始化Elasticsearch索引失败", zap.Error(err))
		return
	}

	// 设置Gin模式
	gin.SetMode(config.GlobalConfig.App.Mode)

	// 初始化路由
	r := initRouter()

	// 启动HTTP服务
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.GlobalConfig.App.Port),
		Handler: r,
	}

	// 优雅关闭
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP服务启动失败", zap.Error(err))
		}
	}()

	logger.Info("服务已启动", zap.String("addr", srv.Addr))

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("关闭服务...")

	// 设置关闭超时
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("服务关闭异常", zap.Error(err))
	}

	logger.Info("服务已关闭")
}

// 初始化路由
func initRouter() *gin.Engine {
	r := gin.New()

	// 使用中间件
	r.Use(gin.Recovery())
	r.Use(logger.GinLogger())

	// 初始化API路由
	router.Setup(r)

	return r
}
