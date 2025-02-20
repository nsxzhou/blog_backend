package middleware

import (
	"blog/api/data"

	"github.com/gin-gonic/gin"
)

func VisitRecorder() gin.HandlerFunc {
	return func(c *gin.Context) {
		data.RecordVisit(c)
		c.Next()
	}
}
