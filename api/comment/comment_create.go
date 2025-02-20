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

type CommentCreateRequest struct {
	Content         string `json:"content" validate:"required,min=1,max=100"`
	ArticleID       string `json:"article_id" validate:"required"`
	ParentCommentID *uint  `json:"parent_comment_id,omitempty" validate:"omitempty"`
}

func (cm *Comment) CommentCreate(c *gin.Context) {
	_claims, _ := c.Get("claims")
	claims := _claims.(*utils.CustomClaims)
	var req CommentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		global.Log.Error("c.ShouldBindJSON() failed", zap.String("error", err.Error()))
		res.Error(c, res.InvalidParameter, "请求参数格式错误")
		return
	}

	err := utils.Validate(req)
	if err != nil {
		global.Log.Error("utils.Validate() failed", zap.String("error", err.Error()))
		res.Error(c, res.InvalidParameter, utils.FormatValidationError(err.(validator.ValidationErrors)))
		return
	}

	comment := &models.CommentModel{
		Content:         req.Content,
		ArticleID:       req.ArticleID,
		UserID:          claims.UserID,
		ParentCommentID: req.ParentCommentID,
	}

	if err := models.CommentCreate(comment); err != nil {
		global.Log.Error("comment.CommentCreate() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "创建评论失败")
		return
	}

	redis_ser.IncrArticleCommentCount(req.ArticleID)
	global.Log.Info("创建评论成功", zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))
	res.Success(c, nil)

}
