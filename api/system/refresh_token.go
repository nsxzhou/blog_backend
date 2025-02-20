package system

import (
	"blog/global"
	"blog/models/res"
	"blog/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type RefreshTokenRequest struct {
	Token  string `json:"token" binding:"required"`
	UserID uint   `json:"user_id" binding:"required"`
}

func (s *System) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		res.Error(c, res.InvalidParameter, "请求参数格式错误")
		return
	}

	err := utils.Validate(req)
	if err != nil {
		global.Log.Error("utils.Validate() failed", zap.String("error", err.Error()))
		res.Error(c, res.InvalidParameter, utils.FormatValidationError(err.(validator.ValidationErrors)))
		return
	}

	accessToken, err := utils.RefreshAccessToken(req.Token, req.UserID)
	if err != nil {
		global.Log.Error("utils.RefreshAccessToken() failed", zap.String("error", err.Error()))
		res.Error(c, res.InvalidParameter, "刷新令牌失败")
		return
	}

	res.Success(c, accessToken)
}
