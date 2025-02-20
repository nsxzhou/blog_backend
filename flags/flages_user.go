package flags

import (
	"strconv"

	"blog/global"
	"blog/models"
	"blog/models/ctypes"
	"blog/utils"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func User(c *cli.Context) error {
	nickName := c.String("nick_name")
	password := c.String("password")
	role := c.String("role")
	ip := "127.0.0.1"
	userRole := ctypes.RoleUser
	if role == "admin" {
		userRole = ctypes.RoleAdmin
	}

	account, err := utils.GenerateID()
	if err != nil {
		global.Log.Error("生成account失败", zap.String("error", err.Error()))
		return err
	}

	user := &models.UserModel{
		Account:  strconv.FormatInt(account, 10),
		Nickname: nickName,
		Password: password,
		Role:     userRole,
	}

	if err := user.Create(ip); err != nil {
		global.Log.Error("用户创建失败",
			zap.String("error", err.Error()),
		)
		return err
	}

	global.Log.Infof("用户%s创建成功,account:%s,role:%s", nickName, user.Account, string(userRole))
	return nil
}
