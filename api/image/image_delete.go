package image

import (
	"blog/global"
	"blog/models"
	"blog/models/res"
	"blog/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

func (i *Image) ImageDelete(c *gin.Context) {
	var req models.IDRequest
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

	var image models.ImageModel
	err = global.DB.First(&image, req.ID).Error
	if err != nil {
		global.Log.Error("global.DB.First() failed", zap.String("error", err.Error()))
		res.Error(c, res.NotFound, "图片不存在")
		return
	}

	err = global.DB.Delete(&image).Error
	if err != nil {
		global.Log.Error("image.Delete() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "图片删除失败")
		return
	}
	global.Log.Info("图片删除成功", zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))
	res.Success(c, nil)
}
