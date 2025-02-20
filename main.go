package main

import (
	"fmt"

	"blog/core"
	"blog/flags"
	"blog/global"
	"blog/router"
	"blog/service/corn_ser"
	"blog/utils"

	"go.uber.org/zap"
)

func main() {
	var err error
	// 初始化配置
	core.InitConf()
	// 初始化日志
	global.Log = core.NewLogManager(&global.Config.Log)
	// 初始化数据库
	global.DB = core.InitGorm()
	// 初始化redis
	global.Redis = core.InitRedis()
	// 初始化es
	global.Es = core.InitEs()
	// 初始化地址数据库
	global.AddrDB = core.InitAddrDB()
	// 初始化系统信息
	utils.Init(global.Config.System.StartTime, global.Config.System.MachineID)
	// 初始化命令行参数
	flags.Newflags()
	// 初始化corn
	corn_ser.CornInit()
	// 打印系统信息
	utils.PrintSystem()
	// 初始化路由
	router := router.InitRouter()
	// 启动服务
	err = router.Run(fmt.Sprintf(":%d", global.Config.System.Port))
	if err != nil {
		global.Log.Fatal("启动服务失败", zap.String("error", err.Error()))
	}
}
