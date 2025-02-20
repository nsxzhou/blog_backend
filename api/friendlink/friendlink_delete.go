package friendlink

import (
	"blog/global"
	"blog/models"
	"blog/models/res"
	"blog/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

func (f *FriendLink) FriendLinkDelete(c *gin.Context) {
	var req models.IDRequest
	if err := c.ShouldBindUri(&req); err != nil {
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

	err = (&models.FriendLinkModel{
		MODEL: models.MODEL{
			ID: req.ID,
		},
	}).Delete()
	if err != nil {
		global.Log.Error("friendlink.Delete() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "友链删除失败")
		return
	}
	global.Log.Info("友链删除成功", zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))
	res.Success(c, nil)
}
