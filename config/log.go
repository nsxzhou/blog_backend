package config

type Log struct {
	Filename   string `mapstructure:"filename"`    // 日志文件路径
	MaxSize    int    `mapstructure:"max_size"`    // 每个日志文件的最大大小（MB）
	MaxAge     int    `mapstructure:"max_age"`     // 保留的日志文件天数
	MaxBackups int    `mapstructure:"max_backups"` // 保留的旧日志文件数量
	Level      string `mapstructure:"level"`       // 日志级别：debug/info/warn/error
	Format     string `mapstructure:"format"`      // 日志格式：json/console
	BufferSize int    `mapstructure:"buffer_size"` // 缓冲区大小
	Compress   bool   `mapstructure:"compress"`    // 是否压缩旧日志文件
}
