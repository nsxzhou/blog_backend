package friendlink

import (
	"blog/global"
	"blog/models"
	"blog/models/res"
	"blog/service/search_ser"
	"blog/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

func (f *FriendLink) FriendLinkList(c *gin.Context) {
	var req models.PageInfo
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

	list, count, err := search_ser.ComList(models.FriendLinkModel{}, search_ser.Option{
		PageInfo: req,
	})
	if err != nil {
		global.Log.Error("search.ComList() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "加载失败")
		return
	}
	global.Log.Info("友链列表成功", zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))
	res.SuccessWithPage(c, list, count, req.Page, req.PageSize)
}
