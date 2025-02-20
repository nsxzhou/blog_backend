package core

import (
	"blog/global"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
	"go.uber.org/zap"
)

// InitAddrDB 初始化 IP 地理位置数据库
func InitAddrDB() *xdb.Searcher {
	dbPath := "ip2region.xdb"

	// 加载整个 xdb 到内存
	cBuff, err := xdb.LoadContentFromFile(dbPath)
	if err != nil {
		global.Log.Error("无法读取ip2region.xdb文件", zap.String("error", err.Error()))
		return nil
	}


	// 创建内存查询对象
	searcher, err := xdb.NewWithBuffer(cBuff)
	if err != nil {
		global.Log.Error("创建ip2region查询对象失败", zap.String("error", err.Error()))
		return nil
	}

	global.Log.Info("地址库初始化成功", zap.String("method", "InitAddrDB"), zap.String("path", "core/addr_db.go"))
	return searcher

}
