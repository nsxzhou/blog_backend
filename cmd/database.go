package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/nsxzhou1114/blog-api/internal/database"
	"github.com/nsxzhou1114/blog-api/internal/model"
	"github.com/nsxzhou1114/blog-api/internal/service"
	"github.com/spf13/cobra"
)

// databaseCmd 数据库管理命令
var databaseCmd = &cobra.Command{
	Use:   "db",
	Short: "数据库管理命令",
	Long:  `数据库管理相关的命令，包括导入导出、同步、清理等`,
}

// exportMySQLCmd 导出MySQL数据命令
var exportMySQLCmd = &cobra.Command{
	Use:   "export-mysql [table] [file]",
	Short: "导出MySQL数据",
	Long:  `导出MySQL表数据到JSON文件`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		exportMySQLData(args[0], args[1])
	},
}

// importMySQLCmd 导入MySQL数据命令
var importMySQLCmd = &cobra.Command{
	Use:   "import-mysql [table] [file]",
	Short: "导入MySQL数据",
	Long:  `从JSON文件导入数据到MySQL表`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		importMySQLData(args[0], args[1])
	},
}

// syncESCmd 同步ES数据命令
var syncESCmd = &cobra.Command{
	Use:   "sync-es [index]",
	Short: "同步数据到Elasticsearch",
	Long:  `将MySQL数据同步到Elasticsearch`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		index := "articles"
		if len(args) > 0 {
			index = args[0]
		}
		syncToElasticsearch(index)
	},
}

// cleanupDBCmd 清理数据库命令
var cleanupDBCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "清理数据库",
	Long:  `清理过期的日志、通知等数据`,
	Run: func(cmd *cobra.Command, args []string) {
		cleanupDatabase()
	},
}

// initTablesCmd 初始化数据库表命令
var initTablesCmd = &cobra.Command{
	Use:   "init-tables",
	Short: "初始化数据库表",
	Long:  `初始化数据库表和Elasticsearch索引`,
	Run: func(cmd *cobra.Command, args []string) {
		initializeTables()
	},
}

func init() {
	// 添加数据库相关子命令
	databaseCmd.AddCommand(exportMySQLCmd)
	databaseCmd.AddCommand(importMySQLCmd)
	databaseCmd.AddCommand(syncESCmd)
	databaseCmd.AddCommand(cleanupDBCmd)
	databaseCmd.AddCommand(initTablesCmd)
	
	// 将数据库命令添加到根命令
	rootCmd.AddCommand(databaseCmd)
}

// exportMySQLData 导出MySQL数据
func exportMySQLData(tableName, fileName string) {
	if err := initializeSystem(); err != nil {
		fmt.Printf("系统初始化失败: %v\n", err)
		os.Exit(1)
	}

	db := database.GetDB()
	
	var data []map[string]interface{}
	if err := db.Table(tableName).Find(&data).Error; err != nil {
		fmt.Printf("导出数据失败: %v\n", err)
		return
	}
	
	file, err := os.Create(fileName)
	if err != nil {
		fmt.Printf("创建文件失败: %v\n", err)
		return
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		fmt.Printf("写入文件失败: %v\n", err)
		return
	}
	
	fmt.Printf("成功导出 %d 条记录到 %s\n", len(data), fileName)
}

// importMySQLData 导入MySQL数据
func importMySQLData(tableName, fileName string) {
	if err := initializeSystem(); err != nil {
		fmt.Printf("系统初始化失败: %v\n", err)
		os.Exit(1)
	}

	file, err := os.Open(fileName)
	if err != nil {
		fmt.Printf("打开文件失败: %v\n", err)
		return
	}
	defer file.Close()
	
	var data []map[string]interface{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		fmt.Printf("解析文件失败: %v\n", err)
		return
	}
	
	db := database.GetDB()
	
	// 使用事务确保数据一致性
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	
	for _, record := range data {
		if err := tx.Table(tableName).Create(&record).Error; err != nil {
			tx.Rollback()
			fmt.Printf("导入数据失败: %v\n", err)
			return
		}
	}
	
	if err := tx.Commit().Error; err != nil {
		fmt.Printf("提交事务失败: %v\n", err)
		return
	}
	
	fmt.Printf("成功导入 %d 条记录到表 %s\n", len(data), tableName)
}

// syncToElasticsearch 同步数据到Elasticsearch
func syncToElasticsearch(index string) {
	if err := initializeSystem(); err != nil {
		fmt.Printf("系统初始化失败: %v\n", err)
		os.Exit(1)
	}

	switch index {
	case "articles":
		syncArticlesToES()
	default:
		fmt.Printf("不支持的索引: %s\n", index)
	}
}

// syncArticlesToES 同步文章到Elasticsearch
func syncArticlesToES() {
	searchService := service.NewArticleSearchService()
	
	fmt.Println("开始同步文章到Elasticsearch...")
	
	if err := searchService.SyncArticlesToES(); err != nil {
		fmt.Printf("同步文章到ES失败: %v\n", err)
		return
	}
	
	fmt.Println("文章同步到Elasticsearch完成！")
}

// cleanupDatabase 清理数据库
func cleanupDatabase() {
	if err := initializeSystem(); err != nil {
		fmt.Printf("系统初始化失败: %v\n", err)
		os.Exit(1)
	}

	db := database.GetDB()
	
	// 清理30天前的登录日志
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	result := db.Where("created_at < ?", thirtyDaysAgo).Delete(&model.LoginLog{})
	fmt.Printf("清理了 %d 条过期登录日志\n", result.RowsAffected)
	
	// 清理60天前的操作日志
	sixtyDaysAgo := time.Now().AddDate(0, 0, -60)
	result = db.Where("created_at < ?", sixtyDaysAgo).Delete(&model.OperationLog{})
	fmt.Printf("清理了 %d 条过期操作日志\n", result.RowsAffected)
	
	// 清理7天前的通知
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	result = db.Where("created_at < ? AND is_read = ?", sevenDaysAgo, 1).Delete(&model.Notification{})
	fmt.Printf("清理了 %d 条已读通知\n", result.RowsAffected)
	
	// 清理临时文件记录（如果有的话）
	// 这里可以添加更多清理逻辑
	
	fmt.Println("数据库清理完成")
}

// initializeTables 初始化数据库表
func initializeTables() {
	if err := initializeSystem(); err != nil {
		fmt.Printf("系统初始化失败: %v\n", err)
		os.Exit(1)
	}

	db := database.GetDB()
	es := database.GetES()
	
	// 初始化MySQL表
	if err := model.InitTables(db); err != nil {
		fmt.Printf("初始化MySQL表失败: %v\n", err)
		return
	}
	fmt.Println("MySQL表初始化成功")
	
	// 初始化Elasticsearch索引
	if err := model.InitESIndices(es); err != nil {
		fmt.Printf("初始化Elasticsearch索引失败: %v\n", err)
		return
	}
	fmt.Println("Elasticsearch索引初始化成功")
	
	fmt.Println("数据库初始化完成")
} 