package utils

import (
	"blog/global"
	"net"

	"go.uber.org/zap"
)

// GetIPList 获取所有IP地址
func GetIPList() (ipList []string) {
	interfaces, err := net.Interfaces()
	if err != nil {
		global.Log.Error("net.Interfaces() failed", zap.String("error", err.Error()))
	}

	for _, i2 := range interfaces {
		address, err := i2.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range address {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipNet.IP.To4()
			if ip4 == nil {
				continue
			}
			ipList = append(ipList, ip4.String())
		}
	}
	return
}
