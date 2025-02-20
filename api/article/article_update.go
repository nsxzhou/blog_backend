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

type ArticleUpdateRequest struct {
	ID       string `json:"id" validate:"required"`
	Title    string `json:"title" validate:"required,min=1,max=50"`
	Abstract string `json:"abstract" validate:"required,min=1,max=100"`
	Content  string `json:"content" validate:"required,min=1,max=100000"`
	Category string `json:"category" validate:"required,min=1,max=10"`
	CoverID  uint   `json:"cover_id" validate:"required,gt=0"`
}

func (a *Article) ArticleUpdate(c *gin.Context) {
	var req ArticleUpdateRequest
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

	var coverUrl string
	if req.CoverID != 0 {
		err = global.DB.Model(models.ImageModel{}).Where("id = ?", req.CoverID).Select("path").Scan(&coverUrl).Error
		if err != nil {
			global.Log.Error("global.DB.Model(models.ImageModel{}).Where().Select().Scan() failed", zap.String("error", err.Error()))
			res.Error(c, res.ServerError, "选择图片路径失败")
			return
		}

	}
	article, err := models.NewArticleService().ArticleGet(req.ID)
	if err != nil {
		global.Log.Error("models.NewArticleService().ArticleGet() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "获取文章数据失败")
		return
	}
	article.Title = req.Title
	article.Abstract = req.Abstract
	article.Content = req.Content
	article.Category = req.Category
	article.CoverID = req.CoverID
	article.CoverURL = coverUrl
	err = models.NewArticleService().ArticleUpdate(article)
	if err != nil {
		global.Log.Error("models.NewArticleService().UpdateArticle() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "文章更新失败")
		return
	}
	global.Log.Info("文章更新成功", zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))
	res.Success(c, nil)
}
