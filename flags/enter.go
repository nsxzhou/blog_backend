package flags

import (
	"os"

	"blog/global"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func Newflags() {
	var app = cli.NewApp()
	app.Name = "溺水寻舟的博客"
	app.Authors = []*cli.Author{
		{
			Name:  "溺水寻舟",
			Email: "1790146932@qq.com",
		},
	}
	app.Commands = []*cli.Command{
		{
			Name:    "database",
			Aliases: []string{"db"},
			Usage:   "建表",
			Action:  DB,
		},
		{
			Name:    "user",
			Aliases: []string{"u"},
			Usage:   "创建用户",
			Action:  User,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "nick_name",
					Aliases: []string{"n"},
					Usage:   "用户昵称",
					Value:   "admin",
				},
				&cli.StringFlag{
					Name:    "password",
					Aliases: []string{"p"},
					Usage:   "用户密码",
					Value:   "3.1415926535",
				},
				&cli.StringFlag{
					Name:    "role",
					Aliases: []string{"r"},
					Usage:   "用户角色 (admin/user)",
					Value:   "admin",
				},
			},
		},
		{
			Name:    "export-mysql",
			Aliases: []string{"e-m"},
			Usage:   "导出数据库",
			Action:  MysqlExport,
		},
		{
			Name:    "import-mysql",
			Aliases: []string{"i-m"},
			Usage:   "导入数据库",
			Action:  MysqlImport,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "path",
				},
			},
		},
		{
			Name:    "elasticsearch",
			Aliases: []string{"es"},
			Usage:   "创建索引",
			Action:  EsIndexCreate,
		},
		{
			Name:    "export-es",
			Aliases: []string{"e-e"},
			Usage:   "导出索引",
			Action:  EsExport,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "index",
					Usage: "索引名称",
					Value: "article_index",
				},
			},
		},
		{
			Name:    "import-es",
			Aliases: []string{"i-e"},
			Usage:   "导入索引",
			Action:  EsImport,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "path",
				},
			},
		},
	}
	if len(os.Args) > 1 {
		err := app.Run(os.Args)
		if err != nil {
			global.Log.Fatal("初始化命令失败", zap.String("error", err.Error()))
		}
		os.Exit(0)

	}
}
