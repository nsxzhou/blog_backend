package router

import (
	"blog/api"
	"blog/middleware"
)

func (r *RouterGroup) CategoryRouter() {
	categoryRouter := r.Group("category")
	categoryApi := api.AppGroupApp.CategoryApi
	categoryRouter.POST("", middleware.JwtAdmin(), categoryApi.CategoryCreate)
	categoryRouter.DELETE(":id", middleware.JwtAdmin(), categoryApi.CategoryDelete)
	categoryRouter.GET("list", categoryApi.CategoryList)
}
