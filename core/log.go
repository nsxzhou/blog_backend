package core

import (
	"blog/config"
	"blog/global"
	"blog/service/log_ser"
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogManager 创建新的日志管理器
func NewLogManager(config *config.Log) *zap.SugaredLogger {
	// 如果配置为空，则使用默认配置
	if config == nil {
		config = getDefaultConfig()
	}

	// 创建日志管理器实例
	sugarLogger, err := initialize(config)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
		return nil
	}

	return sugarLogger
}

// initialize 初始化日志管理器
func initialize(config *config.Log) (*zap.SugaredLogger, error) {
	// 创建文件写入器
	fileWriter := getLogWriter(
		config.Filename,
		config.MaxSize,
		config.MaxBackups,
		config.MaxAge,
	)

	// 创建数据库写入器
	logWriter := log_ser.NewDBWriter()

	// 合并多个写入器
	multiWriter := zapcore.NewMultiWriteSyncer(fileWriter, logWriter)

	// 获取编码器
	encoder := getEncoder()

	// 设置日志级别
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(config.Level)); err != nil {
		return nil, fmt.Errorf("invalid log level: %v", err)
	}

	// 创建核心
	var core zapcore.Core
	if global.Config.System.Env == "debug" {
		consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
		core = zapcore.NewTee(
			zapcore.NewCore(encoder, multiWriter, level),                    // 文件和数据库输出
			zapcore.NewCore(consoleEncoder, zapcore.Lock(os.Stdout), level), // 控制台输出
		)
	} else {
		core = zapcore.NewCore(encoder, multiWriter, level)
	}

	// 创建日志记录器
	logger := zap.New(
		core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	sugarLogger := logger.Sugar()
	zap.ReplaceGlobals(logger)

	return sugarLogger, nil
}

// getEncoder 获取日志编码器
func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()

	// 自定义时间格式
	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05"))
	}

	// 设置日志字段名
	encoderConfig.TimeKey = "time"
	encoderConfig.MessageKey = "msg"
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder

	// 自定义调用者格式
	encoderConfig.EncodeCaller = func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(caller.TrimmedPath())
	}

	return zapcore.NewJSONEncoder(encoderConfig)
}

// getLogWriter 获取日志写入器
func getLogWriter(filename string, maxSize, maxBackup, maxAge int) zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   filename,  // 日志文件路径
		MaxSize:    maxSize,   // 文件大小限制
		MaxBackups: maxBackup, // 备份数量
		MaxAge:     maxAge,    // 保留天数
		Compress:   true,      // 是否压缩
	}
	return zapcore.AddSync(lumberJackLogger)
}

// getDefaultConfig 获取默认日志配置
func getDefaultConfig() *config.Log {
	return &config.Log{
		Filename:   "./logs/app.log",
		MaxSize:    100,    // 100MB
		MaxBackups: 7,      // 保留7个备份
		MaxAge:     30,     // 保留30天
		Level:      "info", // 默认info级别
		Format:     "json", // 默认json格式
		BufferSize: 256,    // 256KB缓冲区
		Compress:   true,   // 启用压缩
	}
}

// GinMiddleware Gin框架的日志中间件
func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录开始时间
		start := time.Now()

		// 获取请求信息
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 记录请求体大小
		reqSize := c.Request.ContentLength

		// 包装ResponseWriter以获取响应大小
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// 处理请求
		c.Next()

		// 计算耗时
		cost := time.Since(start)

		// 记录访问日志
		global.Log.Info("access_log",
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.Duration("cost", cost),
			zap.Int64("request_size", reqSize),
			zap.Int("response_size", blw.body.Len()),
			zap.String("refer", c.Request.Referer()),
		)
	}
}

// GinRecovery Gin框架的错误恢复中间件
func GinRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 检查是否是客户端断开连接
				if isBrokenPipe(err) {
					global.Log.Warn("client_disconnected",
						zap.Any("error", err),
						zap.String("path", c.Request.URL.Path),
						zap.String("ip", c.ClientIP()),
					)
					c.Abort()
					return
				}

				// 检查上下文是否已取消
				if c.Request.Context().Err() != nil {
					global.Log.Warn("request_canceled",
						zap.Any("error", err),
						zap.String("path", c.Request.URL.Path),
					)
					c.Abort()
					return
				}

				// 记录系统错误
				stack := string(debug.Stack())
				global.Log.Error("system_error",
					zap.Any("error", err),
					zap.String("stack", stack),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.String("ip", c.ClientIP()),
					zap.String("user_agent", c.Request.UserAgent()),
					zap.String("query", c.Request.URL.RawQuery),
				)

				// 返回500错误
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    500,
					"message": "Internal Server Error",
				})
			}
		}()
		c.Next()
	}
}

// bodyLogWriter 用于记录响应体大小
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write 实现 ResponseWriter 接口
func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// isBrokenPipe 检查是否是连接中断错误
func isBrokenPipe(err interface{}) bool {
	if ne, ok := err.(*net.OpError); ok {
		if se, ok := ne.Err.(*os.SyscallError); ok {
			errMsg := strings.ToLower(se.Error())
			return strings.Contains(errMsg, "broken pipe") ||
				strings.Contains(errMsg, "connection reset by peer") ||
				strings.Contains(errMsg, "protocol wrong type for socket")
		}
	}
	return false
}
