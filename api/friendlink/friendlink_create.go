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

type FriendLinkCreate struct {
	Name string `json:"name" validate:"required,min=1,max=10"`
	Link string `json:"link" validate:"required,url"`
}

func (f *FriendLink) FriendLinkCreate(c *gin.Context) {
	var req FriendLinkCreate
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

	err = (&models.FriendLinkModel{
		Name: req.Name,
		Link: req.Link,
	}).Create()
	if err != nil {
		global.Log.Error("friendlink.Create() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "友链创建失败")
		return
	}
	global.Log.Info("友链创建成功", zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))
	res.Success(c, nil)
}
