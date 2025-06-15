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
	"github.com/nsxzhou1114/blog-api/pkg/cache"
	"github.com/nsxzhou1114/blog-api/pkg/websocket"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var configPath string

// rootCmd 根命令
var rootCmd = &cobra.Command{
	Use:   "blog-api",
	Short: "博客API服务",
	Long:  `一个功能完整的博客API服务，支持用户管理、文章发布、评论等功能`,
}

// serveCmd 启动服务命令
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "启动HTTP服务",
	Long:  `启动博客API的HTTP服务器`,
	Run: func(cmd *cobra.Command, args []string) {
		startServer()
	},
}

func init() {
	// 添加全局标志
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "./config", "配置文件路径")

	// 添加子命令
	rootCmd.AddCommand(serveCmd)
}

// Execute 执行根命令
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// initializeSystem 初始化系统
func initializeSystem() error {
	// 初始化配置
	if err := config.Init(configPath); err != nil {
		return fmt.Errorf("配置初始化失败: %v", err)
	}

	// 初始化日志
	if err := logger.Init(); err != nil {
		return fmt.Errorf("日志初始化失败: %v", err)
	}

	// 初始化MySQL数据库
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("MySQL数据库连接失败")
	}

	// 初始化数据库表
	if err := model.InitTables(db); err != nil {
		return fmt.Errorf("初始化数据库表失败: %v", err)
	}

	// 初始化Elasticsearch
	es := database.GetES()
	if es == nil {
		return fmt.Errorf("Elasticsearch连接失败")
	}

	// 初始化Elasticsearch索引
	if err := model.InitESIndices(es); err != nil {
		return fmt.Errorf("初始化Elasticsearch索引失败: %v", err)
	}

	// 初始化缓存
	if err := cache.InitializeCache(database.GetRedis(), database.GetDB()); err != nil {
		return fmt.Errorf("缓存初始化失败: %v", err)
	}

	// 初始化WebSocket管理器
	websocketManager := websocket.GetManager()
	websocketManager.Initialize(websocket.NewRedisMessageStore(database.GetRedis()))

	return nil
}

// startServer 启动HTTP服务
func startServer() {
	// 初始化系统
	if err := initializeSystem(); err != nil {
		fmt.Printf("系统初始化失败: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()
	defer cache.CleanupCache()

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

	// 优雅关闭WebSocket管理器
	websocketManager := websocket.GetManager()
	websocketManager.Shutdown()

	// 设置关闭超时
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
