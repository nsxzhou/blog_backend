package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8/esapi"
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
// 示例：./blog-api db export-mysql articles articles.json
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
// 示例：./blog-api db import-mysql articles articles.json
var importMySQLCmd = &cobra.Command{
	Use:   "import-mysql [table] [file]",
	Short: "导入MySQL数据",
	Long:  `从JSON文件导入数据到MySQL表`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		importMySQLData(args[0], args[1])
	},
}

// exportESCmd 导出ES数据命令
// 示例：./blog-api db export-es articles articles_es.json
var exportESCmd = &cobra.Command{
	Use:   "export-es [index] [file]",
	Short: "导出Elasticsearch数据",
	Long:  `导出Elasticsearch索引数据到JSON文件`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		exportESData(args[0], args[1])
	},
}

// importESCmd 导入ES数据命令
// 示例：./blog-api db import-es articles articles_es.json
var importESCmd = &cobra.Command{
	Use:   "import-es [index] [file]",
	Short: "导入Elasticsearch数据",
	Long:  `从JSON文件导入数据到Elasticsearch索引`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		importESData(args[0], args[1])
	},
}

// syncESCmd 同步ES数据命令
// 示例：./blog-api db sync-es articles
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
// 示例：./blog-api db cleanup
var cleanupDBCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "清理数据库",
	Long:  `清理过期的日志、通知等数据`,
	Run: func(cmd *cobra.Command, args []string) {
		cleanupDatabase()
	},
}

// initTablesCmd 初始化数据库表命令
// 示例：./blog-api db init-tables
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
	databaseCmd.AddCommand(exportESCmd)
	databaseCmd.AddCommand(importESCmd)
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

// exportESData 导出ES数据
func exportESData(indexName, fileName string) {
	if err := initializeSystem(); err != nil {
		fmt.Printf("系统初始化失败: %v\n", err)
		os.Exit(1)
	}

	esClient := database.GetES()
	ctx := context.Background()

	fmt.Printf("开始导出ES索引 %s 的数据...\n", indexName)

	// 使用scroll API来获取所有数据
	var allDocs []map[string]interface{}
	scrollSize := 1000
	scrollDuration := 5 * time.Minute // For WithScroll method

	// 初始搜索请求
	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
		"size": scrollSize,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(searchQuery); err != nil {
		fmt.Printf("构建搜索查询失败: %v\n", err)
		return
	}

	// 执行初始搜索
	res, err := esClient.Search(
		esClient.Search.WithContext(ctx),
		esClient.Search.WithIndex(indexName),
		esClient.Search.WithBody(&buf),
		esClient.Search.WithScroll(scrollDuration),
		esClient.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		fmt.Printf("执行ES搜索失败: %v\n", err)
		return
	}
	defer res.Body.Close()

	if res.IsError() {
		fmt.Printf("ES搜索返回错误: %s\n", res.String())
		return
	}

	var searchResult map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&searchResult); err != nil {
		fmt.Printf("解析搜索结果失败: %v\n", err)
		return
	}

	// 获取总数
	total := int64(searchResult["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))
	fmt.Printf("索引 %s 共有 %d 条记录\n", indexName, total)

	// 处理第一批数据
	hits := searchResult["hits"].(map[string]interface{})["hits"].([]interface{})
	for _, hit := range hits {
		hitMap := hit.(map[string]interface{})
		doc := map[string]interface{}{
			"_id":     hitMap["_id"],
			"_source": hitMap["_source"],
		}
		allDocs = append(allDocs, doc)
	}

	// 获取scroll_id
	scrollID := searchResult["_scroll_id"].(string)

	// 继续滚动获取剩余数据
	for len(hits) > 0 {
		scrollReq := esapi.ScrollRequest{
			ScrollID: scrollID,
			Scroll:   scrollDuration,
		}

		scrollRes, err := scrollReq.Do(ctx, esClient)
		if err != nil {
			fmt.Printf("执行scroll请求失败: %v\n", err)
			break
		}

		if scrollRes.IsError() {
			fmt.Printf("Scroll请求返回错误: %s\n", scrollRes.String())
			scrollRes.Body.Close()
			break
		}

		var scrollResult map[string]interface{}
		if err := json.NewDecoder(scrollRes.Body).Decode(&scrollResult); err != nil {
			fmt.Printf("解析scroll结果失败: %v\n", err)
			scrollRes.Body.Close()
			break
		}
		scrollRes.Body.Close()

		hits = scrollResult["hits"].(map[string]interface{})["hits"].([]interface{})
		for _, hit := range hits {
			hitMap := hit.(map[string]interface{})
			doc := map[string]interface{}{
				"_id":     hitMap["_id"],
				"_source": hitMap["_source"],
			}
			allDocs = append(allDocs, doc)
		}

		fmt.Printf("已导出 %d/%d 条记录\n", len(allDocs), total)
	}

	// 清理scroll上下文
	clearScrollReq := esapi.ClearScrollRequest{
		ScrollID: []string{scrollID},
	}
	clearScrollReq.Do(ctx, esClient)

	// 写入文件
	file, err := os.Create(fileName)
	if err != nil {
		fmt.Printf("创建文件失败: %v\n", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(allDocs); err != nil {
		fmt.Printf("写入文件失败: %v\n", err)
		return
	}

	fmt.Printf("成功导出 %d 条记录到 %s\n", len(allDocs), fileName)
}

// importESData 导入ES数据
func importESData(indexName, fileName string) {
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

	var docs []map[string]interface{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&docs); err != nil {
		fmt.Printf("解析文件失败: %v\n", err)
		return
	}

	esClient := database.GetES()
	ctx := context.Background()

	fmt.Printf("开始导入 %d 条记录到ES索引 %s...\n", len(docs), indexName)

	// 批量导入数据
	batchSize := 100
	successCount := 0
	errorCount := 0

	for i := 0; i < len(docs); i += batchSize {
		end := i + batchSize
		if end > len(docs) {
			end = len(docs)
		}

		batch := docs[i:end]
		
		// 构建批量请求体
		var bulkBody strings.Builder
		for _, doc := range batch {
			// 添加index操作
			indexAction := map[string]interface{}{
				"index": map[string]interface{}{
					"_index": indexName,
				},
			}
			
			// 如果有_id字段，使用它作为文档ID
			if docID, exists := doc["_id"]; exists {
				indexAction["index"].(map[string]interface{})["_id"] = docID
			}

			indexActionJSON, _ := json.Marshal(indexAction)
			bulkBody.Write(indexActionJSON)
			bulkBody.WriteString("\n")

			// 添加文档数据
			var docData interface{}
			if source, exists := doc["_source"]; exists {
				docData = source
			} else {
				docData = doc
			}
			
			docJSON, _ := json.Marshal(docData)
			bulkBody.Write(docJSON)
			bulkBody.WriteString("\n")
		}

		// 执行批量请求
		bulkReq := esapi.BulkRequest{
			Body:    strings.NewReader(bulkBody.String()),
			Refresh: "true",
		}

		bulkRes, err := bulkReq.Do(ctx, esClient)
		if err != nil {
			fmt.Printf("批量导入失败: %v\n", err)
			errorCount += len(batch)
			continue
		}

		if bulkRes.IsError() {
			fmt.Printf("批量导入返回错误: %s\n", bulkRes.String())
			errorCount += len(batch)
			bulkRes.Body.Close()
			continue
		}

		// 解析批量响应
		var bulkResult map[string]interface{}
		if err := json.NewDecoder(bulkRes.Body).Decode(&bulkResult); err != nil {
			fmt.Printf("解析批量响应失败: %v\n", err)
			errorCount += len(batch)
			bulkRes.Body.Close()
			continue
		}
		bulkRes.Body.Close()

		// 统计成功和失败的数量
		items := bulkResult["items"].([]interface{})
		for _, item := range items {
			itemMap := item.(map[string]interface{})
			if indexResult, exists := itemMap["index"]; exists {
				indexMap := indexResult.(map[string]interface{})
				if status, exists := indexMap["status"]; exists {
					statusCode := int(status.(float64))
					if statusCode >= 200 && statusCode < 300 {
						successCount++
					} else {
						errorCount++
						if errorInfo, exists := indexMap["error"]; exists {
							fmt.Printf("文档导入失败: %v\n", errorInfo)
						}
					}
				}
			}
		}

		fmt.Printf("已处理 %d/%d 条记录，成功: %d，失败: %d\n", 
			end, len(docs), successCount, errorCount)
	}

	fmt.Printf("导入完成！成功导入 %d 条记录到索引 %s，失败 %d 条\n", 
		successCount, indexName, errorCount)
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