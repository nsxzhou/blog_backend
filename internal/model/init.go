package model

import (
	"context"
	"fmt"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"gorm.io/gorm"
)

// ESModel 定义支持Elasticsearch操作的模型接口
type ESModel interface {
	ESIndexName() string
	ESMapping() string
}

// 支持ES的模型列表
var esModels = []ESModel{
	&ESArticle{},
}

// 需要自动迁移的模型列表
var models = []interface{}{
	&User{},
	&Article{},
	&Category{},
	&Tag{},
	&ArticleTag{},
	&Comment{},
	&CommentLike{},
	&Favorite{},
	&ArticleLike{},
	&UserFollow{},
	&Notification{},
	&LoginLog{},
	&OperationLog{},
	&Setting{},
	&Image{},
}

// InitTables 初始化数据库表
func InitTables(db *gorm.DB) error {
	fmt.Println("开始初始化数据库表...")

	// 自动迁移表结构
	err := db.AutoMigrate(models...)
	if err != nil {
		return fmt.Errorf("自动迁移数据库表失败: %v", err)
	}

	fmt.Println("数据库表初始化完成")
	return nil
}

// InitESIndices 初始化Elasticsearch索引
func InitESIndices(client *elasticsearch.Client) error {
	fmt.Println("开始初始化Elasticsearch索引...")
	ctx := context.Background()

	for _, model := range esModels {
		indexName := model.ESIndexName()
		mapping := model.ESMapping()

		// 检查索引是否存在
		resp, err := client.Indices.Exists([]string{indexName})
		if err != nil {
			return fmt.Errorf("检查索引 %s 是否存在时出错: %v", indexName, err)
		}

		// 如果索引不存在，则创建
		if resp.StatusCode == 404 {
			createResp, err := client.Indices.Create(
				indexName,
				client.Indices.Create.WithContext(ctx),
				client.Indices.Create.WithBody(strings.NewReader(mapping)),
			)
			if err != nil {
				return fmt.Errorf("创建索引 %s 失败: %v", indexName, err)
			}
			if createResp.IsError() {
				return fmt.Errorf("创建索引 %s 返回错误: %s", indexName, createResp.String())
			}
			fmt.Printf("索引 %s 创建成功\n", indexName)
		} else {
			fmt.Printf("索引 %s 已存在，跳过创建\n", indexName)
		}
	}

	fmt.Println("Elasticsearch索引初始化完成")
	return nil
}
