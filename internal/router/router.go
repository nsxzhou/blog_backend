package router

import (
	"github.com/gin-gonic/gin"
	"github.com/nsxzhou1114/blog-api/internal/config"
	"github.com/nsxzhou1114/blog-api/internal/controller"
	"github.com/nsxzhou1114/blog-api/internal/middleware"
)

// Setup 设置API路由
func Setup(r *gin.Engine) {
	// 静态文件服务 - 让前端可以访问本地上传的图片
	cfg := config.GetConfig()
	if cfg.Image.LocalEnabled {
		// 提供静态文件服务，将 /uploads/images 路径映射到本地存储目录
		r.Static("/uploads/images", cfg.Image.Upload.Local.UploadPath)
		
		// 为了兼容性，也提供 /images 路径的访问方式
		r.Static("/images", cfg.Image.Upload.Local.UploadPath)
	}

	// API 路由组
	api := r.Group("/api")

	// 用户相关路由
	setupUserRoutes(api)

	// 标签相关路由
	setupTagRoutes(api)

	// 分类相关路由
	setupCategoryRoutes(api)

	// 文章相关路由
	setupArticleRoutes(api)

	// 评论相关路由
	setupCommentRoutes(api)

	// 图片相关路由
	setupImageRoutes(api)
}

// setupUserRoutes 设置用户相关路由
func setupUserRoutes(api *gin.RouterGroup) {
	userApi := controller.NewUserApi()

	// 公开路由
	userRoutes := api.Group("/users")
	{
		// 注册
		userRoutes.POST("/register", userApi.Register)
		// 登录
		userRoutes.POST("/login", userApi.Login)
		// 忘记密码
		userRoutes.POST("/forgot-password", userApi.ForgotPassword)
		// 重置密码
		userRoutes.POST("/reset-password", userApi.ResetPassword)
		// 发送验证邮件
		//userRoutes.POST("/send-verification", userApi.SendVerificationEmail)
		// 验证邮箱
		//userRoutes.POST("/verify-email", userApi.VerifyEmail)
		// 获取用户详情（公开接口，可查看任何用户的公开信息）
		userRoutes.GET("/:id", userApi.GetUserDetail)
		// 获取指定用户粉丝列表（公开接口）
		//userRoutes.GET("/:id/followers", userApi.GetUserFollowers)
		// 获取指定用户关注列表（公开接口）
		//userRoutes.GET("/:id/following", userApi.GetUserFollowing)
	}

	// 需要刷新令牌的路由
	refreshRoutes := api.Group("/users", middleware.RefreshAuth())
	{
		// 刷新令牌
		refreshRoutes.POST("/refresh", userApi.RefreshToken)
		// 登出
		refreshRoutes.POST("/logout", userApi.Logout)
	}

	// 需要认证的路由
	authUserRoutes := api.Group("/users", middleware.JWTAuth())
	{
		// 获取当前用户信息
		authUserRoutes.GET("/me", userApi.GetUserInfo)
		// 更新用户信息
		authUserRoutes.PUT("/me", userApi.UpdateUserInfo)
		// 修改密码
		authUserRoutes.POST("/change-password", userApi.ChangePassword)
		// 关注用户
		authUserRoutes.POST("/follow/:id", userApi.FollowUser)
		// 取消关注
		authUserRoutes.DELETE("/follow/:id", userApi.UnfollowUser)
		// 获取当前用户粉丝列表
		authUserRoutes.GET("/followers", userApi.GetFollowers)
		// 获取当前用户关注列表
		authUserRoutes.GET("/following", userApi.GetFollowing)
	}

	// 需要管理员权限的路由
	adminUserRoutes := api.Group("/users", middleware.AdminAuth())
	{
		// 获取用户列表
		adminUserRoutes.GET("", userApi.GetUserList)
		// 更新用户状态
		adminUserRoutes.PUT("/:id/status", userApi.UpdateUserStatus)
		// 重置用户密码
		adminUserRoutes.POST("/:id/reset-password", userApi.ResetUserPassword)
		// 批量操作用户
		adminUserRoutes.POST("/batch-action", userApi.BatchUserAction)
		// 获取用户统计
		adminUserRoutes.GET("/stats", userApi.GetUserStats)
	}
}

// setupTagRoutes 设置标签相关路由
func setupTagRoutes(api *gin.RouterGroup) {
	tagApi := controller.NewTagApi()

	// 公开路由
	tagRoutes := api.Group("/tags")
	{
		// 获取标签列表
		tagRoutes.GET("", tagApi.List)
		// 获取标签云
		// tagRoutes.GET("/cloud", tagApi.GetTagCloud)
		// 获取标签详情
		tagRoutes.GET("/:id", tagApi.GetByID)
	}

	// 需要管理员权限的路由
	adminTagRoutes := api.Group("/tags", middleware.AdminAuth())
	{
		// 创建标签
		adminTagRoutes.POST("", tagApi.Create)
		// 更新标签
		adminTagRoutes.PUT("/:id", tagApi.Update)
		// 删除标签
		adminTagRoutes.DELETE("/:id", tagApi.Delete)
	}
}

// setupCategoryRoutes 设置分类相关路由
func setupCategoryRoutes(api *gin.RouterGroup) {
	categoryApi := controller.NewCategoryApi()

	// 公开路由
	categoryRoutes := api.Group("/categories")
	{
		// 获取分类列表
		categoryRoutes.GET("", categoryApi.List)
		// 获取热门分类
		categoryRoutes.GET("/hot", categoryApi.GetHotCategories)
		// 获取分类详情
		categoryRoutes.GET("/:id", categoryApi.GetByID)
	}

	// 需要管理员权限的路由
	adminCategoryRoutes := api.Group("/categories", middleware.AdminAuth())
	{
		// 创建分类
		adminCategoryRoutes.POST("", categoryApi.Create)
		// 更新分类
		adminCategoryRoutes.PUT("/:id", categoryApi.Update)
		// 删除分类
		adminCategoryRoutes.DELETE("/:id", categoryApi.Delete)
	}
}

// setupArticleRoutes 设置文章相关路由
func setupArticleRoutes(api *gin.RouterGroup) {
	articleApi := controller.NewArticleApi()

	// 公开路由
	articleRoutes := api.Group("/articles")
	{
		// 获取文章详情
		articleRoutes.GET("/:id", articleApi.GetDetail)
		// 统一文章列表接口
		articleRoutes.GET("", articleApi.GetArticleList)
		// 根据标签获取文章
		articleRoutes.GET("/tag/:id", articleApi.GetArticlesByTag)
		// 根据分类获取文章
		articleRoutes.GET("/category/:id", articleApi.GetArticlesByCategory)
		// 获取点赞用户
		articleRoutes.GET("/:id/like-users", articleApi.GetArticleLikeUsers)
	}

	// 需要认证的路由
	authArticleRoutes := api.Group("/articles", middleware.JWTAuth())
	{
		// 创建文章
		authArticleRoutes.POST("", articleApi.Create)
		// 更新文章
		authArticleRoutes.PUT("/:id", articleApi.Update)
		// 删除文章
		authArticleRoutes.DELETE("/:id", articleApi.Delete)
		// 获取用户文章列表
		authArticleRoutes.GET("/my", articleApi.GetUserArticles)
		// 文章交互操作（点赞、收藏等）
		authArticleRoutes.POST("/:id/action", articleApi.ArticleAction)
		// 获取用户收藏的文章
		authArticleRoutes.GET("/favorites", articleApi.GetUserFavorites)
		// 获取文章统计数据
		authArticleRoutes.GET("/stats", articleApi.GetArticleStat)
		// 更新文章状态
		authArticleRoutes.PUT("/:id/status", articleApi.UpdateArticleStatus)
		// 更新文章访问权限
		authArticleRoutes.PUT("/:id/access", articleApi.UpdateArticleAccess)
	}

	// 需要管理员权限的路由
	adminArticleRoutes := api.Group("/articles", middleware.AdminAuth())
	{
		// 创建ES索引
		adminArticleRoutes.POST("/es/create-index", articleApi.CreateESIndex)
		// 同步文章到ES
		adminArticleRoutes.POST("/es/sync", articleApi.SyncArticlesToES)
	}
}

// setupCommentRoutes 设置评论相关路由
func setupCommentRoutes(api *gin.RouterGroup) {
	commentApi := controller.NewCommentApi()

	// 公开路由
	commentRoutes := api.Group("/comments")
	{
		// 获取评论列表
		commentRoutes.GET("", commentApi.List)
		// 获取评论详情
		commentRoutes.GET("/:id", commentApi.GetByID)
	}

	// 需要认证的路由
	authCommentRoutes := api.Group("/comments", middleware.JWTAuth())
	{
		// 创建评论
		authCommentRoutes.POST("", commentApi.Create)
		// 回复评论
		authCommentRoutes.POST("/reply", commentApi.Reply)
		// 更新评论
		authCommentRoutes.PUT("/:id", commentApi.Update)
		// 删除评论
		authCommentRoutes.DELETE("/:id", commentApi.Delete)
		// 点赞评论
		authCommentRoutes.POST("/like", commentApi.Like)

		// 评论通知相关
		// authCommentRoutes.GET("/notifications", commentApi.GetNotifications)
		// authCommentRoutes.PUT("/notifications/:id/read", commentApi.MarkNotificationAsRead)
		// authCommentRoutes.PUT("/notifications/read-all", commentApi.MarkAllNotificationsAsRead)
	}

	// 需要管理员权限的路由
	adminCommentRoutes := api.Group("/comments", middleware.AdminAuth())
	{
		// 更新评论状态
		adminCommentRoutes.PUT("/:id/status", commentApi.UpdateStatus)
		// 批量更新评论状态
		adminCommentRoutes.PUT("/batch-status", commentApi.BatchUpdateStatus)
	}
}

// setupImageRoutes 设置图片相关路由
func setupImageRoutes(api *gin.RouterGroup) {
	imageApi := controller.NewImageApi()

	// 公开路由
	imageRoutes := api.Group("/images")
	{
		// 获取图片详情
		imageRoutes.GET("/:id", imageApi.GetDetail)
		// 获取图片列表
		imageRoutes.GET("", imageApi.List)
		// 获取存储配置
		imageRoutes.GET("/config", imageApi.GetStorageConfig)
		// 根据使用类型获取图片
		imageRoutes.GET("/type/:type", imageApi.GetImagesByUsageType)
		// 根据文章ID获取图片
		imageRoutes.GET("/article/:article_id", imageApi.GetImagesByArticle)
		// 获取图片统计数据
		imageRoutes.GET("/statistics", imageApi.GetStatistics)
	}

	// 需要认证的路由
	authImageRoutes := api.Group("/images", middleware.AdminAuth())
	{
		// 上传图片
		authImageRoutes.POST("/upload", imageApi.Upload)
		// 更新图片信息
		authImageRoutes.PUT("/:id", imageApi.Update)
		// 删除图片
		authImageRoutes.DELETE("/:id", imageApi.Delete)
		// 批量删除图片
		authImageRoutes.POST("/batch-delete", imageApi.BatchDelete)
	}
}

// func setupNotificationRoutes(api *gin.RouterGroup) {
//     notificationApi := controller.NewNotificationApi()
//     // 需要认证的路由
//     authNotificationRoutes := api.Group("/notifications", middleware.JWTAuth())
//     {
//         // 获取用户通知列表
//         authNotificationRoutes.GET("", notificationApi.List)
//         // 获取未读通知数量
//         authNotificationRoutes.GET("/unread-count", notificationApi.GetUnreadCount)
//         // 标记通知为已读
//         authNotificationRoutes.PUT("/:id/read", notificationApi.MarkAsRead)
//         // 标记所有通知为已读
//         authNotificationRoutes.PUT("/read-all", notificationApi.MarkAllAsRead)
//         // 删除通知
//         authNotificationRoutes.DELETE("/:id", notificationApi.Delete)
//         // 批量删除通知
//         authNotificationRoutes.POST("/batch-delete", notificationApi.BatchDelete)
//     }
// }

// func setupSettingRoutes(api *gin.RouterGroup) {
//     settingApi := controller.NewSettingApi()
//     // 公开路由
//     settingRoutes := api.Group("/settings")
//     {
//         // 获取公开设置
//         settingRoutes.GET("/public", settingApi.GetPublicSettings)
//     }
//     // 管理员路由
//     adminSettingRoutes := api.Group("/settings", middleware.AdminAuth())
//     {
//         // 获取所有设置
//         adminSettingRoutes.GET("", settingApi.List)
//         // 更新设置
//         adminSettingRoutes.PUT("/:key", settingApi.Update)
//         // 批量更新设置
//         adminSettingRoutes.PUT("", settingApi.BatchUpdate)
//     }
// }

// func setupImportExportRoutes(api *gin.RouterGroup) {
//     importExportApi := controller.NewImportExportApi()  
//     adminRoutes := api.Group("/import-export", middleware.AdminAuth())
//     {
//         // 导出数据
//         adminRoutes.POST("/export/articles", importExportApi.ExportArticles)
//         adminRoutes.POST("/export/users", importExportApi.ExportUsers)
//         adminRoutes.POST("/export/comments", importExportApi.ExportComments)       
//         // 导入数据
//         adminRoutes.POST("/import/articles", importExportApi.ImportArticles)
//         adminRoutes.GET("/import/status/:task_id", importExportApi.GetImportStatus)
//     }
// }