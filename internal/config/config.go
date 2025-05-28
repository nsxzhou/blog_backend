package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/viper"
)

// Config 全局配置结构体
type Config struct {
	App           AppConfig           `mapstructure:"app"`
	MySQL         DatabaseConfig      `mapstructure:"mysql"`
	Redis         RedisConfig         `mapstructure:"redis"`
	Elasticsearch ElasticsearchConfig `mapstructure:"elasticsearch"`
	Log           LogConfig           `mapstructure:"log"`
	Storage       StorageConfig       `mapstructure:"storage"`
	JWT           JWTConfig           `mapstructure:"jwt"`
	Image         ImageConfig         `mapstructure:"image"`
}

// AppConfig 应用配置
type AppConfig struct {
	Name string     `mapstructure:"name"`
	Mode string     `mapstructure:"mode"`
	Port int        `mapstructure:"port"`
	Cors CorsConfig `mapstructure:"cors"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	SecretKey            string `mapstructure:"secret_key"`
	AccessExpireSeconds  int    `mapstructure:"access_expire_seconds"`
	RefreshExpireSeconds int    `mapstructure:"refresh_expire_seconds"`
	BufferSeconds        int    `mapstructure:"buffer_seconds"`
	Issuer               string `mapstructure:"issuer"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"`
	Charset      string `mapstructure:"charset"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	LogLevel     string `mapstructure:"log_level"`
}

// DSN 获取数据库连接字符串
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		c.Username, c.Password, c.Host, c.Port, c.Database, c.Charset)
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Password     string `mapstructure:"password"`
	DB           int    `mapstructure:"db"`
	PoolSize     int    `mapstructure:"pool_size"`
	MinIdleConns int    `mapstructure:"min_idle_conns"`
}

// Addr 获取Redis地址
func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// ElasticsearchConfig Elasticsearch配置
type ElasticsearchConfig struct {
	URLs        []string `mapstructure:"urls"`
	Username    string   `mapstructure:"username"`
	Password    string   `mapstructure:"password"`
	Sniff       bool     `mapstructure:"sniff"`
	Healthcheck bool     `mapstructure:"healthcheck"`
	InfoLog     string   `mapstructure:"infolog"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxAge     int    `mapstructure:"max_age"`
	MaxBackups int    `mapstructure:"max_backups"`
	Compress   bool   `mapstructure:"compress"`
	Stdout     bool   `mapstructure:"stdout"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Type  string       `mapstructure:"type"`
	Local LocalStorage `mapstructure:"local"`
	COS   COSStorage   `mapstructure:"cos"`
	Limit StorageLimit `mapstructure:"limit"`
}

// LocalStorage 本地存储配置
type LocalStorage struct {
	Path      string `mapstructure:"path"`
	URLPrefix string `mapstructure:"url_prefix"`
}

// COSStorage 腾讯云COS存储配置
type COSStorage struct {
	SecretID  string `mapstructure:"secret_id"`
	SecretKey string `mapstructure:"secret_key"`
	BucketURL string `mapstructure:"bucket_url"`
	Region    string `mapstructure:"region"`
	Bucket    string `mapstructure:"bucket"`
	URLPrefix string `mapstructure:"url_prefix"`
}

// StorageLimit 存储限制配置
type StorageLimit struct {
	MaxSize    int      `mapstructure:"max_size"`
	AllowTypes []string `mapstructure:"allow_types"`
}

// ImageConfig 图片配置
type ImageConfig struct {
	DefaultStorage string            `mapstructure:"default_storage"`
	LocalEnabled   bool              `mapstructure:"local_enabled"`
	CosEnabled     bool              `mapstructure:"cos_enabled"`
	Upload         ImageUploadConfig `mapstructure:"upload"`
}

// ImageUploadConfig 图片上传配置
type ImageUploadConfig struct {
	MaxFileSize  int64            `mapstructure:"max_file_size"`
	AllowedTypes []string         `mapstructure:"allowed_types"`
	Local        ImageLocalConfig `mapstructure:"local"`
	COS          ImageCOSConfig   `mapstructure:"cos"`
}

// ImageLocalConfig 图片本地存储配置
type ImageLocalConfig struct {
	UploadPath string `mapstructure:"upload_path"`
	URLPrefix  string `mapstructure:"url_prefix"`
}

// ImageCOSConfig 图片腾讯云COS配置
type ImageCOSConfig struct {
	SecretID  string `mapstructure:"secret_id"`
	SecretKey string `mapstructure:"secret_key"`
	BucketURL string `mapstructure:"bucket_url"`
	Region    string `mapstructure:"region"`
	Bucket    string `mapstructure:"bucket"`
}

// CorsConfig 跨域配置
type CorsConfig struct {
	AllowOrigins     []string `mapstructure:"allow_origins"`
	AllowMethods     []string `mapstructure:"allow_methods"`
	AllowHeaders     []string `mapstructure:"allow_headers"`
	ExposedHeaders   []string `mapstructure:"expose_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
}

var (
	// GlobalConfig 全局配置实例
	GlobalConfig *Config
	// 配置Viper实例
	viperInstance *viper.Viper
)

// Init 初始化配置
func Init(configPath string) error {
	v := viper.New()
	v.AddConfigPath(configPath)
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}

	GlobalConfig = &config
	viperInstance = v
	return nil
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) *Config {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("读取配置文件失败: %v", err)
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		log.Fatalf("解析配置文件失败: %v", err)
	}

	GlobalConfig = &config
	return &config
}

// GetString 获取字符串配置
func GetString(key string) string {
	return viperInstance.GetString(key)
}

// GetInt 获取整数配置
func GetInt(key string) int {
	return viperInstance.GetInt(key)
}

// GetBool 获取布尔值配置
func GetBool(key string) bool {
	return viperInstance.GetBool(key)
}

// GetStringSlice 获取字符串切片配置
func GetStringSlice(key string) []string {
	return viperInstance.GetStringSlice(key)
}

// GetConfig 获取全局配置
func GetConfig() *Config {
	return GlobalConfig
}
