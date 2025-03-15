package router

import (
	"blog/api"
	"blog/middleware"
)

func (routerGroupApp *RouterGroup) ChatRouter() {
	chatApi := api.AppGroupApp.ChatApi
	chatRouter := routerGroupApp.Group("chat")
	chatRouter.GET("/ws", middleware.WSAuth(), chatApi.HandleWebSocket)
}
