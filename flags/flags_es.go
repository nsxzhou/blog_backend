package flags

import (
	"encoding/json"

	"blog/global"
	"blog/models"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

type Data struct {
	ID  *string         `json:"id"`
	Doc json.RawMessage `json:"doc"`
}

type ESIndexResponse struct {
	Index string `json:"index"`
	Data  []Data `json:"data"`
}

func EsIndexCreate(c *cli.Context) (err error) {
	articleService := models.NewArticleService()
	err = articleService.IndexCreate()
	if err != nil {
		global.Log.Error("索引创建失败", zap.String("error", err.Error()))
		return err
	}
	return nil

}
