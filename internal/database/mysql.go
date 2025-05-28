package database

import (
	"fmt"
	"sync"
	"time"

	"github.com/nsxzhou1114/blog-api/internal/config"
	"github.com/nsxzhou1114/blog-api/internal/logger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// DB 全局数据库实例
var (
	db    *gorm.DB
	dbOne sync.Once
)

// InitMySQL 初始化MySQL数据库连接
func InitMySQL(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	var err error
	dsn := cfg.DSN()

	// GORM配置
	gormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // 使用单数表名
		},
		DisableForeignKeyConstraintWhenMigrating: true, // 禁用外键约束
	}

	db, err = gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("连接MySQL数据库失败: %v", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接池失败: %v", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	// 默认连接最大生命周期为一小时
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("测试数据库连接失败: %v", err)
	}

	logger.Info("MySQL数据库连接成功")
	return db, nil
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	var err error
	dbOne.Do(func() {
		db, err = InitMySQL(&config.GlobalConfig.MySQL)
		if err != nil {
			panic(fmt.Sprintf("MySQL数据库初始化失败: %v", err))
		}
	})
	return db
}
