package router

import (
	"blog/api"
	"blog/middleware"
)


func (router RouterGroup) LogRouter() {
	logApi := api.AppGroupApp.LogApi

	logRouter := router.Group("log")
	logRouter.GET("list", middleware.JwtAdmin(), logApi.LogList)
	logRouter.DELETE(":id", middleware.JwtAdmin(), logApi.LogDelete)


}
