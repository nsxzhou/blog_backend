package utils

import (
	"blog/global"
	"fmt"
	"net"
	"strings"
	"github.com/gin-gonic/gin"
)

// GetAddrByGin 获取客户端IP地址
func GetAddrByGin(c *gin.Context) (addr string) {
	ip := c.ClientIP()
	addr = GetAddrByIp(ip)
	return addr
}

// GetAddrByIp 获取IP地址
func GetAddrByIp(ip string) string {
	parseIP := net.ParseIP(ip)
	if IsIntranetIP(parseIP) {
		return fmt.Sprintf("%s-%s", "内网地址", ip)
	}
	record, err := global.AddrDB.SearchByStr(ip)
	if err != nil {
		return fmt.Sprintf("%s-%s", "错误的地址", ip)
	}
	fields := strings.Split(record, "|")
	regionName := "未知地区"
	if len(fields) >= 4 && fields[3] != "0" {
		regionName = fields[3] // 城市名
	} else if len(fields) >= 3 && fields[2] != "0" {
		regionName = fields[2] // 省份名
	} else if len(fields) >= 1 && fields[0] != "0" {
		regionName = fields[0] // 国家名
	}
	return fmt.Sprintf("%s-%s", regionName, ip)
}

// IsIntranetIP 判断IP是否为内网IP
func IsIntranetIP(ip net.IP) bool {
	if ip.IsLoopback() {
		return true
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return true
	}
	// 192.168
	// 172.16 - 172.31
	// 10
	// 169.254
	return (ip4[0] == 192 && ip4[1] == 168) ||
		(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 32) ||
		(ip4[0] == 10) ||
		(ip4[0] == 169 && ip4[1] == 254)
}
