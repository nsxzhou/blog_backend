package router

import (
	"net/http"

	"blog/core"
	"blog/global"
	"blog/middleware"

	"github.com/gin-gonic/gin"
)

type RouterGroup struct {
	*gin.RouterGroup
}

func InitRouter() *gin.Engine {
	//设置gin模式
	gin.SetMode(global.Config.System.Env)
	router := gin.New()
	router.Use(core.GinMiddleware(), core.GinRecovery())
	router.Use(middleware.VisitRecorder())
	//将指定目录下的文件提供给客户端
	//"uploads" 是URL路径前缀，http.Dir("uploads")是实际文件系统s中存储文件的目录
	router.StaticFS("uploads", http.Dir("uploads"))
	//创建路由组
	apiRouterGroup := router.Group("api")
	routerGroupApp := RouterGroup{apiRouterGroup}
	// 系统配置api
	routerGroupApp.SystemRouter()
	routerGroupApp.UserRouter()
	routerGroupApp.ImageRouter()
	routerGroupApp.ArticleRouter()
	routerGroupApp.CommentRouter()
	routerGroupApp.CategoryRouter()
	routerGroupApp.FriendLinkRouter()
	routerGroupApp.DataRouter()
	routerGroupApp.VisitRouter()
	routerGroupApp.LogRouter()
	routerGroupApp.ChatRouter()
	return router
}
