package data

import (
	"blog/global"
	"blog/models"
	"blog/models/res"
	"context"
	"fmt"
	"time"

	"blog/models/ctypes"

	"crypto/md5"
	"encoding/hex"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// VisitTrend 表示访问趋势数据结构
type VisitTrend struct {
	Dates  []ctypes.MyTime `json:"dates"`  // 修改为使用 MyTime 类型
	Values []int64         `json:"values"` // 对应的访问量数组
}

func RecordVisit(c *gin.Context) {
	ctx := context.Background()

	publicIP := c.ClientIP()
	internalIP := getInternalIP(c)

	// 生成访问者的唯一标识
	visitorID := generateVisitorID(publicIP, internalIP)

	// 使用多个维度的key来控制访问频率
	key := fmt.Sprintf("visit:%s:%s:%s",
		publicIP,   // 公网IP
		internalIP, // 内网IP
		visitorID,  // 访问者唯一标识
	)

	// 检查是否在短时间内重复访问
	exists, err := global.Redis.Exists(ctx, key).Result()
	if err != nil {
		global.Log.Error("Redis check failed", zap.String("error", err.Error()))
	}
	if exists == 1 {
		return // 已存在记录，不重复统计
	}


	region, err := global.AddrDB.SearchByStr(publicIP)
	if err != nil {
		global.Log.Error("解析IP地址失败", zap.String("error", err.Error()))
	}


	// region 直接是字符串，格式如："中国|0|浙江省|杭州市|电信"
	fields := strings.Split(region, "|")
	regionName := "未知地区"
	if len(fields) >= 4 && fields[3] != "0" {
		regionName = fields[3] // 城市名
	} else if len(fields) >= 3 && fields[2] != "0" {
		regionName = fields[2] // 省份名
	} else if len(fields) >= 1 && fields[0] != "0" {
		regionName = fields[0] // 国家名
	}

	// 创建访问记录
	visit := &models.VisitModel{
		PublicIP:     publicIP,
		InternalIP:   internalIP,
		VisitorID:    visitorID,
		UserAgent:    c.Request.UserAgent(),
		Distribution: regionName,
	}

	// 创建访问记录
	err = visit.Create()
	if err != nil {
		global.Log.Error("创建访问记录失败", zap.String("error", err.Error()))
		return
	}


	// 设置Redis过期时间（可以根据需求调整）
	err = global.Redis.Set(ctx, key, 1, 30*time.Minute).Err()
	if err != nil {
		global.Log.Error("Redis set failed", zap.String("error", err.Error()))
	}

}

// 生成访问者唯一标识
func generateVisitorID(publicIP, internalIP string) string {
	// 组合多个特征值
	features := []string{
		publicIP,   // 公网IP
		internalIP, // 内网IP
	}

	// 生成唯一标识
	hash := md5.Sum([]byte(strings.Join(features, "|")))
	return hex.EncodeToString(hash[:])
}

func getInternalIP(c *gin.Context) string {
	// 从X-Forwarded-For获取原始客户端IP链
	if xff := c.Request.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 1 {
			return strings.TrimSpace(ips[len(ips)-1])
		}
	}

	// 从X-Real-IP获取
	if xRealIP := c.Request.Header.Get("X-Real-IP"); xRealIP != "" {
		return xRealIP
	}

	// 获取直接连接的IP
	return strings.Split(c.Request.RemoteAddr, ":")[0]
}

func (d *Data) GetVisitTrend(c *gin.Context) {
	// 获取最近7天的数据
	var results []struct {
		Date  ctypes.MyTime `json:"date"` // 修改为使用 MyTime 类型
		Count int64         `json:"count"`
	}

	err := global.DB.Model(&models.VisitModel{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at >= DATE_SUB(CURDATE(), INTERVAL 7 DAY)").
		Group("DATE(created_at)").
		Order("date ASC").
		Find(&results).Error

	if err != nil {
		global.Log.Error("获取访问趋势数据失败", zap.String("error", err.Error()))
		res.Error(c, res.ServerError, "获取访问趋势数据失败")
		return
	}

	global.Log.Info("获取访问趋势数据成功", zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path))

	// 构建返回数据
	trend := &VisitTrend{
		Dates:  make([]ctypes.MyTime, len(results)),
		Values: make([]int64, len(results)),
	}

	// 填充数据
	for i, result := range results {
		trend.Dates[i] = result.Date
		trend.Values[i] = result.Count
	}

	res.Success(c, trend)
}
