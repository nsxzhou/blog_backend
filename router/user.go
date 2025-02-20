package router

import (
	"blog/api"
	"blog/middleware"
)

func (router RouterGroup) UserRouter() {
	userApi := api.AppGroupApp.UserApi
	userRouter := router.Group("user")
	userRouter.GET("", middleware.JwtAuth(), userApi.Userinfo)
	userRouter.POST("", middleware.JwtAdmin(), userApi.UserCreate)
	userRouter.POST("login", userApi.UserLogin)
	userRouter.GET("qq/login-url", userApi.GetQQLoginURL)
	userRouter.GET("qq/callback", userApi.QQLoginCallback)
	userRouter.POST("logout", middleware.JwtAuth(), userApi.UserLogout)
	userRouter.GET("list", middleware.JwtAdmin(), userApi.UserList)
	userRouter.DELETE(":id", middleware.JwtAdmin(), userApi.UserDelete)
}
