package flags

import (
	"blog/global"
	"blog/models"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func DB(c *cli.Context) (err error) {
	err = global.DB.Set("gorm:table_options", "ENGINE=InnoDB").
		AutoMigrate(&models.UserModel{},
			&models.ImageModel{},
			&models.CommentModel{},
			&models.CategoryModel{},
			&models.FriendLinkModel{},
			&models.VisitModel{},
			&models.LogModel{},
		)
	if err != nil {
		global.Log.Error("生成数据库表结构失败", zap.String("error", err.Error()))
		return nil
	}
	global.Log.Info("生成数据库表结构成功", zap.String("method", "DB"), zap.String("path", "flags/flags_db.go"))
	return nil

}
