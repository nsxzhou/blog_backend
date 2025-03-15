package router

import (
	"blog/api"
	"blog/middleware"
)

func (routerGroupApp *RouterGroup) ChatRouter() {
	chatApi := api.AppGroupApp.ChatApi
	routerGroupApp.GET("/ws", middleware.WSAuth(), chatApi.HandleWebSocket)
}
