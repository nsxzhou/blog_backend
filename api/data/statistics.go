package data

import (
	"blog/global"
	"blog/models"
	"blog/models/res"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Statistics struct {
	TotalUsers    int64 `json:"total_users"`
	TotalArticles int64 `json:"total_articles"`
	TotalComments int64 `json:"total_comments"`
	TotalViews    int64 `json:"total_views"`
}

func (d *Data) GetStatistics(c *gin.Context) {
	totalUsers, err := models.GetTotalUsers()
	if err != nil {
		global.Log.Error("models.GetTotalUsers() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "获取用户总数失败")
		return
	}

	articleService := models.NewArticleService()
	articleStats, err := articleService.GetArticleStats()
	if err != nil {
		global.Log.Error("models.GetTotalArticles() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "获取文章统计数据失败")
		return
	}


	statistics := &Statistics{
		TotalUsers:    totalUsers,
		TotalArticles: articleStats.TotalArticles,
		TotalComments: articleStats.TotalComments,
		TotalViews:    articleStats.TotalViews,
	}
	global.Log.Info("获取统计数据成功", zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))
	res.Success(c, statistics)
}

