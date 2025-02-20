package api

import (
	"blog/api/article"
	"blog/api/category"
	"blog/api/comment"
	"blog/api/data"
	"blog/api/friendlink"
	"blog/api/image"
	"blog/api/log"
	"blog/api/system"
	"blog/api/user"
	"blog/api/visit"
)

type AppGroup struct {
	SystemApi     system.System
	UserApi       user.User
	ImageApi      image.Image
	ArticleApi    article.Article
	CommentApi    comment.Comment
	CategoryApi   category.Category
	FriendLinkApi friendlink.FriendLink
	DataApi       data.Data
	VisitApi      visit.Visit
	LogApi        log.Log
}

var AppGroupApp = new(AppGroup)
