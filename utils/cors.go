package utils

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

// Cors 跨域中间件
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 允许所有来源的请求
		c.Header("Access-Control-Allow-Origin", "*")
		// 允许的HTTP方法
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		// 允许的请求头
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept")
		// 如果是OPTIONS请求，直接返回204状态码
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

/*
这段代码解决了前后端分离开发中的跨域问题。比如：
如果您的前端网站在 http://example.com
后端 API 在 http://api.example.com
没有这个中间件，浏览器会阻止前端访问后端 API
有了这个中间件，前端就可以正常访问后端 API 了
*/
