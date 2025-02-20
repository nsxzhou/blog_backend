package comment

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

type CommentDeleteRequest struct {
	ID        uint   `json:"id" validate:"required,gt=0"`
	ArticleID string `json:"article_id" validate:"required"`
}

func (cm *Comment) CommentDelete(c *gin.Context) {
	var req CommentDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		global.Log.Error("c.ShouldBindUri() failed", zap.String("error", err.Error()))
		res.Error(c, res.InvalidParameter, "请求参数格式错误")
		return
	}

	err := utils.Validate(req)
	if err != nil {
		global.Log.Error("utils.Validate() failed", zap.String("error", err.Error()))
		res.Error(c, res.InvalidParameter, utils.FormatValidationError(err.(validator.ValidationErrors)))
		return
	}

	if err := models.CommentDelete(req.ID, req.ArticleID); err != nil {
		global.Log.Error("comment.CommentDelete() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "删除评论失败")
		return
	}

	redis_ser.DecrArticleCommentCount(req.ArticleID)
	global.Log.Info("删除评论成功", zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))

	res.Success(c, nil)
}
