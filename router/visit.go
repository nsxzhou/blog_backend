package router

import (
	"blog/api"
	"blog/middleware"
)

func (router RouterGroup) VisitRouter() {
	visitApi := api.AppGroupApp.VisitApi
	visitRouter := router.Group("visit")
	visitRouter.GET("list", middleware.JwtAuth(), visitApi.VisitList)
	visitRouter.DELETE(":id", middleware.JwtAuth(), visitApi.VisitDelete)
}
