package visit

import (
	"blog/global"
	"blog/models"
	"blog/models/res"
	"blog/service/search_ser"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (v *Visit) VisitList(c *gin.Context) {
	var req models.PageInfo
	if err := c.ShouldBindQuery(&req); err != nil {
		global.Log.Error("c.ShouldBindQuery() failed", zap.String("error", err.Error()))
		res.Error(c, res.InvalidParameter, "参数验证失败")
		return
	}

	list, count, err := search_ser.ComList(models.VisitModel{}, search_ser.Option{
		PageInfo: req,
	})
	if err != nil {
		global.Log.Error("search.ComList() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "加载失败")
	}
	global.Log.Info("访问记录列表成功", zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))
	res.SuccessWithPage(c, list, count, req.Page, req.PageSize)
}
