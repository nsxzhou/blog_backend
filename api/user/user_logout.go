package user

import (
	"blog/global"
	"blog/models/res"
	"blog/service/redis_ser"
	"blog/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (u *User) UserLogout(c *gin.Context) {
	accessToken := c.GetHeader("Authorization")

	if len(accessToken) < 7 || accessToken[:7] != "Bearer " {
		global.Log.Error("缺少token")
		res.Error(c, res.TokenMissing, "缺少token")
		return
	}
	accessToken = accessToken[7:]

	claims, err := utils.ParseToken(accessToken)
	if err != nil {
		global.Log.Error("utils.ParseToken() failed", zap.String("error", err.Error()))
		res.Error(c, res.TokenInvalid, "token无效")
		return
	}

	err = redis_ser.InvalidateTokens(claims.UserID, accessToken)
	if err != nil {
		global.Log.Error("redis_ser.InvalidateTokens() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "登出失败")
		return
	}
	global.Log.Info("用户退出成功", zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))
	res.Success(c, nil)
}
