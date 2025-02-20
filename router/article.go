package router

import (
	"blog/api"
	"blog/middleware"
)

func (router RouterGroup) ArticleRouter() {
	articleApi := api.AppGroupApp.ArticleApi
	articleRouter := router.Group("article")
	articleRouter.GET(":id", articleApi.ArticleDetail)
	articleRouter.POST("", middleware.JwtAdmin(), articleApi.ArticleCreate)
	articleRouter.GET("list", articleApi.ArticleList)
	articleRouter.POST("delete", middleware.JwtAdmin(), articleApi.ArticleDelete)
	articleRouter.PUT("", middleware.JwtAdmin(), articleApi.ArticleUpdate)
	articleRouter.GET("data", middleware.JwtAdmin(), articleApi.GetArticleData)
}
