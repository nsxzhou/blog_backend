package flags

import (
	"blog/global"
	"blog/models"
	"context"
	"encoding/json"
	"os"

	"github.com/elastic/go-elasticsearch/v8/typedapi/core/bulk"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

// EsImport 从指定文件导入数据到 Elasticsearch
// 参数 c *cli.Context: CLI上下文，包含命令行参数
// 返回 error: 如果发生错误则返回错误信息
func EsImport(c *cli.Context) (err error) {
	// 获取命令行参数中指定的文件路径
	path := c.String("path")

	// 读取指定文件的内容
	byteData, err := os.ReadFile(path)
	if err != nil {
		global.Log.Error("读取文件失败", zap.String("error", err.Error()))
		return err
	}

	// 解析JSON文件内容到ESIndexResponse结构体
	var response ESIndexResponse
	err = json.Unmarshal(byteData, &response)
	if err != nil {
		global.Log.Error("解析文件失败", zap.String("error", err.Error()))
		return err
	}

	// 创建文章服务实例并初始化ES索引
	articleService := models.NewArticleService()
	err = articleService.IndexCreate()
	if err != nil {
		global.Log.Error("创建索引失败", zap.String("error", err.Error()))
		return err
	}

	// 准备批量导入请求
	var request bulk.Request
	for _, data := range response.Data {
		// 添加索引操作的元数据
		request = append(request, map[string]interface{}{
			"index": map[string]interface{}{
				"_index": response.Index,
				"_id":    data.ID,
			},
		})
		// 添加实际的文档数据
		request = append(request, data.Doc)
	}

	// 执行批量导入操作
	_, err = global.Es.Bulk().
		Index(response.Index).
		Request(&request).
		Do(context.Background())
	if err != nil {
		global.Log.Error("导入数据失败", zap.String("error", err.Error()))
		return err
	}

	// 记录成功导入的数据条数
	global.Log.Infof("Es数据添加成功,共添加 %d 条", len(response.Data))
	return nil
}
