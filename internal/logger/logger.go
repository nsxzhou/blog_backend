package logger

import (
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/natefinch/lumberjack"
	"github.com/nsxzhou1114/blog-api/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Logger 全局日志实例
	Logger *zap.Logger
	// SugaredLogger 语法糖日志实例
	SugaredLogger *zap.SugaredLogger
	loggerOnce sync.Once
)

// Init 初始化日志
func Init() error {
	// 使用配置中的日志设置
	cfg := config.GlobalConfig.Log
	loggerOnce.Do(func() {
		InitLogger(&cfg)
	})
	return nil
}

// Sync 同步日志
func Sync() error {
	return Logger.Sync()
}

// InitLogger 初始化日志
func InitLogger(cfg *config.LogConfig) {
	// 设置日志级别
	var level zapcore.Level
	switch cfg.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	// 设置JSON编码器
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 设置日志输出
	var writeSyncer zapcore.WriteSyncer

	if cfg.Filename != "" {
		// 使用lumberjack进行日志轮转
		lumberjackLogger := &lumberjack.Logger{
			Filename:   cfg.Filename,
			MaxSize:    cfg.MaxSize, // MB
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge, // days
			Compress:   cfg.Compress,
		}

		// 如果同时需要输出到控制台
		if cfg.Stdout {
			writeSyncer = zapcore.NewMultiWriteSyncer(
				zapcore.AddSync(lumberjackLogger),
				zapcore.AddSync(os.Stdout),
			)
		} else {
			writeSyncer = zapcore.AddSync(lumberjackLogger)
		}
	} else {
		// 如果没有设置文件名，则输出到控制台
		writeSyncer = zapcore.AddSync(os.Stdout)
	}

	// 创建Core
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		writeSyncer,
		level,
	)

	// 创建Logger
	Logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	SugaredLogger = Logger.Sugar()
}

// GetLogger 获取日志实例
func GetLogger() *zap.Logger {
	return Logger	
}

// GetSugaredLogger 获取语法糖日志实例			
func GetSugaredLogger() *zap.SugaredLogger {
	return SugaredLogger
}

// GinLogger 返回Gin中间件日志处理函数
func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 计算耗时
		cost := time.Since(start)

		// 记录日志
		Logger.Info("HTTP请求",
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.Duration("cost", cost),
			zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()),
		)
	}
}

// Debug 调试日志
func Debug(msg string, fields ...zap.Field) {
	Logger.Debug(msg, fields...)
}

// Info 信息日志
func Info(msg string, fields ...zap.Field) {
	Logger.Info(msg, fields...)
}

// Warn 警告日志
func Warn(msg string, fields ...zap.Field) {
	Logger.Warn(msg, fields...)
}

// Error 错误日志
func Error(msg string, fields ...zap.Field) {
	Logger.Error(msg, fields...)
}

// Fatal 致命错误日志
func Fatal(msg string, fields ...zap.Field) {
	Logger.Fatal(msg, fields...)
}

// Debugf 格式化调试日志
func Debugf(format string, args ...interface{}) {
	SugaredLogger.Debugf(format, args...)
}

// Infof 格式化信息日志
func Infof(format string, args ...interface{}) {
	SugaredLogger.Infof(format, args...)
}

// Warnf 格式化警告日志
func Warnf(format string, args ...interface{}) {
	SugaredLogger.Warnf(format, args...)
}

// Errorf 格式化错误日志
func Errorf(format string, args ...interface{}) {
	SugaredLogger.Errorf(format, args...)
}

// Fatalf 格式化致命错误日志
func Fatalf(format string, args ...interface{}) {
	SugaredLogger.Fatalf(format, args...)
}

// Close 关闭日志
func Close() {
	_ = Logger.Sync()
}
