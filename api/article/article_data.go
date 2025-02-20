package article

import (
	"blog/global"
	"blog/models"
	"blog/models/res"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (a *Article) GetArticleData(c *gin.Context) {
	var data *models.ArticleStats
	articleService := models.NewArticleService()
	data, err := articleService.GetArticleStats()
	if err != nil {
		global.Log.Error("articleService.GetArticleStats() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "获取文章数据失败")
		return
	}
	global.Log.Info("获取文章数据成功", zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))
	res.Success(c, data)
}
