package log_ser

import (
	"blog/global"
	"blog/models"
	"encoding/json"
	"time"
)

// DBWriter 实现 zapcore.WriteSyncer 接口
type DBWriter struct {
	// 使用通道来实现异步写入
	logChan chan *models.LogModel
}

// NewDBWriter 创建数据库日志写入器
func NewDBWriter() *DBWriter {
	w := &DBWriter{
		logChan: make(chan *models.LogModel, 1000), // 缓冲区大小可配置
	}

	// 启动异步写入协程
	go w.startWorker()

	return w
}

// Write 实现 zapcore.WriteSyncer 接口
func (w *DBWriter) Write(p []byte) (n int, err error) {
	// 解析日志内容
	var logEntry map[string]interface{}
	if err := json.Unmarshal(p, &logEntry); err != nil {
		return 0, err
	}

	// 转换为 LogModel
	logModel := &models.LogModel{
		Level:    getString(logEntry, "level"),
		Caller:   getString(logEntry, "caller"),
		Message:  getString(logEntry, "msg"),
		ErrorMsg: getString(logEntry, "error"),
	}

	// 异步写入
	select {
	case w.logChan <- logModel:
		return len(p), nil
	default:
		// 通道已满时直接返回，避免阻塞
		return len(p), nil
	}
}

// Sync 实现 zapcore.WriteSyncer 接口
func (w *DBWriter) Sync() error {
	return nil
}

// startWorker 启动异步写入工作协程
func (w *DBWriter) startWorker() {
	const batchSize = 100
	const flushInterval = 5 * time.Second

	var batch []*models.LogModel
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case log := <-w.logChan:
			batch = append(batch, log)

			// 达到批量大小时写入
			if len(batch) >= batchSize {
				w.writeBatch(batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			// 定时写入剩余的日志
			if len(batch) > 0 {
				w.writeBatch(batch)
				batch = batch[:0]
			}
		}
	}
}

// writeBatch 批量写入日志
func (w *DBWriter) writeBatch(logs []*models.LogModel) {
	if err := global.DB.CreateInBatches(logs, len(logs)).Error; err != nil {
		// 写入失败时打印错误
		println("Failed to write logs to database:", err.Error())
	}
}

// 辅助函数
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
