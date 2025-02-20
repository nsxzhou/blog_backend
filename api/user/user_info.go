package user

import (
	"blog/global"
	"blog/models"
	"blog/models/res"
	"blog/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (u *User) Userinfo(c *gin.Context) {
	var user models.UserModel
	_claims, _ := c.Get("claims")
	claims := _claims.(*utils.CustomClaims)
	if err := global.DB.First(&user, claims.UserID).Error; err != nil {
		global.Log.Error("global.DB.First() failed", 	zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "获取用户信息失败")
		return
	}
	global.Log.Info("获取用户信息成功", zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))
	res.Success(c, user)
}
