package article

import (
	"blog/global"
	"blog/models"
	"blog/models/res"
	"blog/service/redis_ser"
	"blog/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type ArticleDeleteRequest struct {
	IDList []string `json:"id_list" validate:"required"`
}

func (a *Article) ArticleDelete(c *gin.Context) {
	var req ArticleDeleteRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		global.Log.Error("c.ShouldBindJSON() failed", zap.String("error", err.Error()))
		res.Error(c, res.InvalidParameter, "请求参数格式错误")
		return
	}

	err = utils.Validate(req)
	if err != nil {
		global.Log.Error("utils.Validate() failed", zap.String("error", err.Error()))
		res.Error(c, res.InvalidParameter, utils.FormatValidationError(err.(validator.ValidationErrors)))
		return
	}

	err = models.NewArticleService().ArticleDelete(req.IDList)
	if err != nil {
		global.Log.Error("articleService.ArticleDelete() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "文章删除失败")
		return
	}

	for _, articleID := range req.IDList {
		err = redis_ser.DeleteArticleStats(articleID)
		if err != nil {
			global.Log.Error("redis_ser.DeleteArticleStats() failed", zap.String("error", err.Error()))
		}

	}
	global.Log.Info("文章删除成功", zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))
	res.Success(c, nil)
}
