package visit

import (
	"blog/global"
	"blog/models"
	"blog/models/res"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (v *Visit) VisitDelete(c *gin.Context) {
	var req models.IDRequest
	if err := c.ShouldBindUri(&req); err != nil {

		global.Log.Error("c.ShouldBindUri() failed", zap.String("error", err.Error()))
		res.Error(c, res.InvalidParameter, "参数验证失败")
		return

	}
	err := (&models.VisitModel{
		MODEL: models.MODEL{
			ID: req.ID,
		},
	}).Delete()
	if err != nil {
		global.Log.Error("visit.Delete() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "删除失败")
	}
	global.Log.Info("访问记录删除成功", zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))
	res.Success(c, nil)
}


