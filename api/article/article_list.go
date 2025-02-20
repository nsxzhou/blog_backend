package article

import (
	"blog/global"
	"blog/models"
	"blog/models/res"
	"blog/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type ArticleListRequest struct {
	models.SearchParams
}

func (a *Article) ArticleList(c *gin.Context) {
	var req ArticleListRequest
	err := c.ShouldBindQuery(&req)
	if err != nil {
		global.Log.Error("c.ShouldBindQuery() failed", zap.String("error", err.Error()))
		res.Error(c, res.InvalidParameter, "请求参数格式错误")
		return
	}

	err = utils.Validate(req)
	if err != nil {
		global.Log.Error("utils.Validate() failed", zap.String("error", err.Error()))
		res.Error(c, res.InvalidParameter, utils.FormatValidationError(err.(validator.ValidationErrors)))
		return
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}

	articles, err := models.NewArticleService().ArticleSearch(req.SearchParams)
	if err != nil {
		global.Log.Error("models.NewArticleService().ArticleSearch() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "搜索文章失败")
		return
	}
	global.Log.Info("文章列表成功", zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))

	res.SuccessWithPage(c, articles.Articles, articles.Total, req.Page, req.PageSize)
}
