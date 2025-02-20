package router

import (
	"blog/api"
	"blog/middleware"
)

func (router RouterGroup) DataRouter() {
	dataRouter := router.Group("/data")
	dataApi := api.AppGroupApp.DataApi
	dataRouter.GET("/statistics", middleware.JwtAdmin(), dataApi.GetStatistics)
	dataRouter.GET("/visit_trend", middleware.JwtAdmin(), dataApi.GetVisitTrend)
	dataRouter.GET("/user_distribution", middleware.JwtAdmin(), dataApi.GetUserDistribution)

}
