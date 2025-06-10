package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nsxzhou1114/blog-api/internal/dto"
	"github.com/nsxzhou1114/blog-api/internal/service"
	"github.com/nsxzhou1114/blog-api/pkg/response"
)

// RSSApi RSS控制器
type RSSApi struct {
	rssService *service.RSSService
}

// NewRSSApi 创建RSS控制器实例
func NewRSSApi() *RSSApi {
	return &RSSApi{
		rssService: service.NewRSSService(),
	}
}

// GetRSSFeed 获取RSS订阅数据
func (api *RSSApi) GetRSSFeed(c *gin.Context) {
	// 绑定查询参数
	var query dto.RSSQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	// 构建基础URL
	baseURL := api.getBaseURL(c)

	// 生成RSS数据
	rssXML, err := api.rssService.GenerateRSSFeed(&query, baseURL)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "生成RSS失败", err)
		return
	}

	// 设置响应头
	c.Header("Content-Type", "application/rss+xml; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=3600") // 缓存1小时

	// 返回XML数据
	c.String(http.StatusOK, rssXML)
}

// GetRSSURL 获取RSS订阅链接
func (api *RSSApi) GetRSSURL(c *gin.Context) {
	// 绑定查询参数
	var query dto.RSSQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	// 构建基础URL
	baseURL := api.getBaseURL(c)

	// 生成RSS订阅链接
	rssURL := api.rssService.GetRSSURL(&query, baseURL)

	response.Success(c, "获取RSS链接成功", gin.H{
		"rss_url": rssURL,
	})
}

// getBaseURL 获取基础URL
func (api *RSSApi) getBaseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}

	host := c.Request.Host
	if forwardedHost := c.GetHeader("X-Forwarded-Host"); forwardedHost != "" {
		host = forwardedHost
	}

	return scheme + "://" + host
} 