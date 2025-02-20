package utils

import (
	"blog/global"
)

// PrintSystem 打印系统信息
func PrintSystem() {
	ip := global.Config.System.Host
	port := global.Config.System.Port

	if ip == "0.0.0.0" {
		ipList := GetIPList()
		for _, i := range ipList {
			global.Log.Infof("blog_server 运行在： http://%s:%d/api", i, port)
		}
	} else {
		global.Log.Infof("blog_server 运行在： http://%s:%d/api", ip, port)
	}
}
