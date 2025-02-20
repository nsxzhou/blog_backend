package router

import (
	"blog/api"
)

func (router RouterGroup) SystemRouter() {
	systemRouter := router.Group("system")
	SystemApi := api.AppGroupApp.SystemApi
	systemRouter.GET("captcha", SystemApi.CaptchaCreate)
	systemRouter.POST("refreshToken", SystemApi.RefreshToken)
}
