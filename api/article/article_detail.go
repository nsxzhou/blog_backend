package article

import (
	"blog/global"
	"blog/models"
	"blog/models/res"
	"blog/service/redis_ser"
	"blog/utils"

	"github.com/go-playground/validator/v10"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ArticleDetailRequest struct {
	ID string `uri:"id" validate:"required"`
}

func (a *Article) ArticleDetail(c *gin.Context) {
	var req ArticleDetailRequest
	err := c.ShouldBindUri(&req)
	if err != nil {
		global.Log.Error("c.ShouldBindUri() failed", zap.String("error", err.Error()))
		res.Error(c, res.InvalidParameter, "请求参数格式错误")
		return
	}

	err = utils.Validate(req)
	if err != nil {
		global.Log.Error("utils.Validate() failed", zap.String("error", err.Error()))
		res.Error(c, res.InvalidParameter, utils.FormatValidationError(err.(validator.ValidationErrors)))
		return
	}

	article, err := models.NewArticleService().ArticleGet(req.ID)
	if err != nil {
		global.Log.Error("models.NewArticleService().ArticleGet() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "加载失败")
		return
	}

	redis_ser.IncrArticleLookCount(req.ID, c.ClientIP())
	global.Log.Info("文章详情成功", zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))
	res.Success(c, article)
}
