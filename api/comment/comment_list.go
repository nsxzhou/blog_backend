package comment

import (
	"blog/global"
	"blog/models"
	"blog/models/res"
	"blog/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type CommentListRequest struct {
	models.CommentRequest
	ArticleID string `form:"article_id" validate:"required"`
}

func (cm *Comment) CommentList(c *gin.Context) {
	var req CommentListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		global.Log.Error("c.ShouldBindQuery() failed", zap.String("error", err.Error()))
		res.Error(c, res.InvalidParameter, "请求参数格式错误")
		return
	}

	err := utils.Validate(req)
	if err != nil {
		global.Log.Error("utils.Validate() failed", zap.String("error", err.Error()))
		res.Error(c, res.InvalidParameter, utils.FormatValidationError(err.(validator.ValidationErrors)))
		return
	}

	comments, err := models.GetArticleCommentsWithTree(req.ArticleID)
	if err != nil {
		global.Log.Error("comment.GetArticleCommentsWithTree() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "获取评论失败")
		return
	}
	global.Log.Info("获取评论成功", zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))

	res.Success(c, comments)
}
