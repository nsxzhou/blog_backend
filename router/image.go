package router

import (
	"blog/api"
	"blog/middleware"
)

func (router RouterGroup) ImageRouter() {
	imageRouter := router.Group("image")
	imageApi := api.AppGroupApp.ImageApi
	imageRouter.POST("", middleware.JwtAdmin(), imageApi.ImageUpload)
	imageRouter.GET("list", middleware.JwtAdmin(), imageApi.ImageList)
	imageRouter.DELETE(":id", middleware.JwtAdmin(), imageApi.ImageDelete)
}
