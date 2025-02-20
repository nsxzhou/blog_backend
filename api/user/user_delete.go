package user

import (
	"blog/global"
	"blog/models"
	"blog/models/res"
	"blog/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

func (u *User) UserDelete(c *gin.Context) {
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

	var user models.UserModel

	if err := global.DB.First(&user, req.ID).Error; err != nil {
		global.Log.Error("global.DB.First() failed", zap.String("error", err.Error()))
		res.Error(c, res.NotFound, "用户不存在")
		return
	}


	if err := user.Delete(); err != nil {
		global.Log.Error("user.Delete() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "删除用户失败")
		return
	}
	global.Log.Info("用户删除成功", zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))

	res.Success(c, nil)
}
