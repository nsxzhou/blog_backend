package user

import (
	"blog/models/ctypes"
	"strconv"

	"blog/global"
	"blog/models"
	"blog/models/res"
	"blog/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"github.com/go-playground/validator/v10"
)

type UserCreateRequest struct {
	Nickname string          `json:"nick_name" validate:"required,min=1,max=10"`
	Password string          `json:"password" validate:"required,min=6,max=16"`
	Role     ctypes.UserRole `json:"role" validate:"required,oneof=admin user"`
}

func (u *User) UserCreate(c *gin.Context) {
	var req UserCreateRequest
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

	account, err := utils.GenerateID()
	if err != nil {
		global.Log.Error("utils.GenerateID() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "生成ID失败")
		return
	}

	err = (&models.UserModel{
		Account:  strconv.FormatInt(account, 10),
		Nickname: req.Nickname,
		Password: req.Password,
		Role:     req.Role,
	}).Create(c.ClientIP())
	if err != nil {
		global.Log.Error("user.Create() failed", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "用户创建失败")
		return
	}
	global.Log.Info("用户创建成功", zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))
	res.Success(c, strconv.FormatInt(account, 10))
}
