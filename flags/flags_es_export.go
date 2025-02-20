package flags

import (
	"blog/global"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

// EsExport 从 Elasticsearch 导出指定索引的数据到 JSON 文件
// 参数 c *cli.Context 包含命令行参数，需要包含 "index" 参数指定要导出的索引名
// 返回 error 如果在导出过程中发生错误
func EsExport(c *cli.Context) error {
	// 获取要导出的索引名
	index := c.String("index")
	if index == "" {
		return fmt.Errorf("索引名不能为空")
	}

	// 查询指定索引的所有数据
	res, err := global.Es.Search().
		Index(index).
		Query(&types.Query{
			MatchAll: &types.MatchAllQuery{},
		}).
		Do(context.Background())
	if err != nil {
		global.Log.Error("查询 ES 数据失败",
			zap.String("index", index),
			zap.String("error", err.Error()))
		return fmt.Errorf("查询 ES 数据失败: %w", err)
	}

	// 构建导出数据结构
	var data ESIndexResponse
	data.Index = index
	for _, hit := range res.Hits.Hits {
		item := Data{
			ID:  hit.Id_,
			Doc: hit.Source_,
		}
		data.Data = append(data.Data, item)
	}

	// 生成导出文件名，格式：索引名_日期.json
	fileName := fmt.Sprintf("%s_%s.json", index, time.Now().Format("20060102"))

	// 创建并写入文件
	file, err := os.Create(fileName)
	if err != nil {
		global.Log.Error("创建导出文件失败",
			zap.String("fileName", fileName),
			zap.String("error", err.Error()))
		return fmt.Errorf("创建导出文件失败: %w", err)

	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("关闭文件失败: %w", cerr)
		}
	}()

	// 将数据转换为 JSON 格式
	byteData, err := json.Marshal(data)
	if err != nil {
		global.Log.Error("序列化数据失败", zap.String("error", err.Error()))
		return fmt.Errorf("序列化数据失败: %w", err)
	}

	// 写入文件
	if _, err = file.Write(byteData); err != nil {
		global.Log.Error("写入文件失败",
			zap.String("fileName", fileName),
			zap.String("error", err.Error()))
		return fmt.Errorf("写入文件失败: %w", err)
	}

	global.Log.Info("数据导出成功",
		zap.String("index", index),
		zap.String("fileName", fileName))
	return nil
}
