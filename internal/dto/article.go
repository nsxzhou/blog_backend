package dto

import "time"

// ArticleCreateRequest 创建文章请求
type ArticleCreateRequest struct {
	Title      string `json:"title" binding:"required,max=255"`                             // 文章标题
	Content    string `json:"content" binding:"required"`                                   // 文章内容
	Summary    string `json:"summary" binding:"max=500"`                                    // 文章摘要
	CategoryID uint   `json:"category_id" binding:"required"`                               // 分类ID
	TagIDs     []uint `json:"tag_ids" binding:"dive,min=1"`                                 // 标签ID列表
	CoverImage string `json:"cover_image"`                                                  // 封面图片
	Status     string `json:"status" binding:"required,oneof=draft published"`              // 状态(草稿/已发布)
	AccessType string `json:"access_type" binding:"required,oneof=public private password"` // 访问类型
	Password   string `json:"password"`                                                     // 访问密码(当access_type为password时需要)
	IsTop      int    `json:"is_top" binding:"oneof=0 1"`                                   // 是否置顶
	IsOriginal int    `json:"is_original" binding:"required,oneof=0 1"`                     // 是否原创
	SourceURL  string `json:"source_url"`                                                   // 转载来源URL
	SourceName string `json:"source_name"`                                                  // 转载来源名称
}

// ArticleUpdateRequest 更新文章请求
type ArticleUpdateRequest struct {
	Title      string `json:"title" binding:"omitempty,max=255"`                             // 文章标题
	Content    string `json:"content"`                                                       // 文章内容
	Summary    string `json:"summary" binding:"omitempty,max=500"`                           // 文章摘要
	CategoryID uint   `json:"category_id"`                                                   // 分类ID
	TagIDs     []uint `json:"tag_ids" binding:"omitempty,dive,min=1"`                        // 标签ID列表
	CoverImage string `json:"cover_image"`                                                   // 封面图片
	Status     string `json:"status" binding:"omitempty,oneof=draft published"`              // 状态
	AccessType string `json:"access_type" binding:"omitempty,oneof=public private password"` // 访问类型
	Password   string `json:"password"`                                                      // 访问密码
	IsTop      int    `json:"is_top" binding:"omitempty,oneof=0 1"`                          // 是否置顶
	IsOriginal int    `json:"is_original" binding:"omitempty,oneof=0 1"`                     // 是否原创
	SourceURL  string `json:"source_url"`                                                    // 转载来源URL
	SourceName string `json:"source_name"`                                                   // 转载来源名称
}

// ArticleQueryRequest 文章查询请求
type ArticleQueryRequest struct {
	Keyword    string `form:"keyword"`                                                                                        // 关键词
	Status     string `form:"status" binding:"omitempty,oneof=draft published"`                                               // 状态
	CategoryID uint   `form:"category_id"`                                                                                    // 分类ID
	TagID      uint   `form:"tag_id"`                                                                                         // 标签ID
	AuthorID   uint   `form:"author_id"`                                                                                      // 作者ID
	IsTop      int    `form:"is_top" binding:"omitempty,oneof=0 1"`                                                           // 是否置顶
	IsOriginal int    `form:"is_original" binding:"omitempty,oneof=0 1"`                                                      // 是否原创
	AccessType string `form:"access_type" binding:"omitempty,oneof=public private password"`                                  // 访问类型
	StartDate  string `form:"start_date"`                                                                                     // 开始日期
	EndDate    string `form:"end_date"`                                                                                       // 结束日期
	OrderBy    string `form:"order_by" binding:"omitempty,oneof=created_at published_at view_count like_count comment_count"` // 排序字段
	Order      string `form:"order" binding:"omitempty,oneof=asc desc"`                                                       // 排序方向
	Page       int    `form:"page" binding:"required,min=1"`                                                                  // 页码
	PageSize   int    `form:"page_size" binding:"required,min=5,max=50"`                                                      // 每页条数
}

// ArticleListItem 文章列表项
type ArticleListItem struct {
	ID            uint      `json:"id"`                     // 文章ID
	Title         string    `json:"title"`                  // 文章标题
	Summary       string    `json:"summary"`                // 文章摘要
	CategoryID    uint      `json:"category_id"`            // 分类ID
	CategoryName  string    `json:"category_name"`          // 分类名称
	AuthorID      uint      `json:"author_id"`              // 作者ID
	AuthorName    string    `json:"author_name"`            // 作者名称
	CoverImage    string    `json:"cover_image"`            // 封面图片
	ViewCount     int       `json:"view_count"`             // 浏览量
	LikeCount     int       `json:"like_count"`             // 点赞数
	CommentCount  int       `json:"comment_count"`          // 评论数
	FavoriteCount int       `json:"favorite_count"`         // 收藏数
	WordCount     int       `json:"word_count"`             // 字数
	Status        string    `json:"status"`                 // 状态
	AccessType    string    `json:"access_type"`            // 访问类型
	IsTop         int       `json:"is_top"`                 // 是否置顶
	IsOriginal    int       `json:"is_original"`            // 是否原创
	Tags          []TagInfo `json:"tags"`                   // 标签列表
	CreatedAt     time.Time `json:"created_at"`             // 创建时间
	UpdatedAt     time.Time `json:"updated_at"`             // 更新时间
	PublishedAt   time.Time `json:"published_at,omitempty"` // 发布时间
}

// ArticleDetailResponse 文章详情响应
type ArticleDetailResponse struct {
	ID            uint      `json:"id"`                     // 文章ID
	Title         string    `json:"title"`                  // 文章标题
	Content       string    `json:"content"`                // 文章内容
	Summary       string    `json:"summary"`                // 文章摘要
	CategoryID    uint      `json:"category_id"`            // 分类ID
	CategoryName  string    `json:"category_name"`          // 分类名称
	AuthorID      uint      `json:"author_id"`              // 作者ID
	AuthorName    string    `json:"author_name"`            // 作者名称
	AuthorAvatar  string    `json:"author_avatar"`          // 作者头像
	CoverImage    string    `json:"cover_image"`            // 封面图片
	ViewCount     int       `json:"view_count"`             // 浏览量
	LikeCount     int       `json:"like_count"`             // 点赞数
	CommentCount  int       `json:"comment_count"`          // 评论数
	FavoriteCount int       `json:"favorite_count"`         // 收藏数
	WordCount     int       `json:"word_count"`             // 字数
	Status        string    `json:"status"`                 // 状态
	AccessType    string    `json:"access_type"`            // 访问类型
	IsTop         int       `json:"is_top"`                 // 是否置顶
	IsOriginal    int       `json:"is_original"`            // 是否原创
	SourceURL     string    `json:"source_url"`             // 转载来源URL
	SourceName    string    `json:"source_name"`            // 转载来源名称
	Tags          []TagInfo `json:"tags"`                   // 标签列表
	CreatedAt     time.Time `json:"created_at"`             // 创建时间
	UpdatedAt     time.Time `json:"updated_at"`             // 更新时间
	PublishedAt   time.Time `json:"published_at,omitempty"` // 发布时间
	// 扩展字段
	IsLiked         bool            `json:"is_liked"`         // 当前用户是否已点赞
	IsFavorited     bool            `json:"is_favorited"`     // 当前用户是否已收藏
	NextArticle     *SimpleArticle  `json:"next_article"`     // 下一篇文章
	PrevArticle     *SimpleArticle  `json:"prev_article"`     // 上一篇文章
	RelatedArticles []SimpleArticle `json:"related_articles"` // 相关文章
}

// SimpleArticle 简化版文章信息(用于上一篇/下一篇/相关文章等场景)
type SimpleArticle struct {
	ID          uint      `json:"id"`           // 文章ID
	Title       string    `json:"title"`        // 文章标题
	CoverImage  string    `json:"cover_image"`  // 封面图片
	PublishedAt time.Time `json:"published_at"` // 发布时间
}

// ArticleListResponse 文章列表响应
type ArticleListResponse struct {
	Total int64             `json:"total"` // 总数
	List []ArticleListItem `json:"list"` // 列表项
}

// TagInfo 标签信息(简化版)
type TagInfo struct {
	ID   uint   `json:"id"`   // 标签ID
	Name string `json:"name"` // 标签名称
}

// ArticleListRequest 通用文章列表请求（合并搜索、热门、最新等功能）
type ArticleListRequest struct {
	// 搜索条件
	Keyword    string `form:"keyword"`     // 搜索关键词（为空时不进行关键词搜索）
	CategoryID uint   `form:"category_id"` // 分类ID
	TagID      uint   `form:"tag_id"`      // 标签ID
	AuthorID   uint   `form:"author_id"`   // 作者ID
	
	// 过滤条件
	Status     string `form:"status" binding:"omitempty,oneof=draft published"`              // 状态
	AccessType string `form:"access_type" binding:"omitempty,oneof=public private password"` // 访问类型
	IsTop      int    `form:"is_top" binding:"omitempty,oneof=0 1"`                          // 是否置顶
	IsOriginal int    `form:"is_original" binding:"omitempty,oneof=0 1"`                     // 是否原创
	
	// 时间范围
	StartDate string `form:"start_date"` // 开始日期
	EndDate   string `form:"end_date"`   // 结束日期
	
	// 排序方式
	SortBy string `form:"sort_by" binding:"omitempty,oneof=latest hot score created_at published_at view_count like_count comment_count"` // 排序类型
	Order  string `form:"order" binding:"omitempty,oneof=asc desc"`                                                                      // 排序方向
	
	// 分页
	Page     int `form:"page" binding:"required,min=1"`             // 页码
	PageSize int `form:"page_size" binding:"required,min=5,max=50"` // 每页条数
}

// ArticleStatusRequest 更新文章状态请求
type ArticleStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=draft published"` // 状态
}

// ArticleAccessRequest 更新文章访问控制请求
type ArticleAccessRequest struct {
	AccessType string `json:"access_type" binding:"required,oneof=public private password"` // 访问类型
	Password   string `json:"password"`                                                     // 访问密码
}

// ArticlePasswordVerifyRequest 文章密码验证请求
type ArticlePasswordVerifyRequest struct {
	Password string `json:"password" binding:"required"` // 访问密码
}

// ArticleActionRequest 文章操作请求(点赞/收藏)
type ArticleActionRequest struct {
	Action string `json:"action" binding:"required,oneof=like unlike favorite unfavorite"` // 操作类型
}

// ArticleDraftAutoSaveRequest 文章草稿自动保存请求
type ArticleDraftAutoSaveRequest struct {
	ArticleID uint   `json:"article_id"`                 // 文章ID(若为0则表示新文章)
	Title     string `json:"title"`                      // 标题
	Content   string `json:"content" binding:"required"` // 内容
}

// ArticleStatRequest 文章统计数据请求
type ArticleStatRequest struct {
	StartDate string `form:"start_date"` // 开始日期
	EndDate   string `form:"end_date"`   // 结束日期
}

// ArticleStatResponse 文章统计数据响应
type ArticleStatResponse struct {
	TotalArticles     int `json:"total_articles"`     // 文章总数
	PublishedArticles int `json:"published_articles"` // 已发布文章数
	DraftArticles     int `json:"draft_articles"`     // 草稿数
	TotalViews        int `json:"total_views"`        // 总浏览量
	TotalLikes        int `json:"total_likes"`        // 总点赞数
	TotalComments     int `json:"total_comments"`     // 总评论数
	TotalFavorites    int `json:"total_favorites"`    // 总收藏数
	// 按日期统计的数据
	DailyStats []DailyStat `json:"daily_stats"`
}

// DailyStat 每日统计数据
type DailyStat struct {
	Date      string `json:"date"`      // 日期
	Articles  int    `json:"articles"`  // 发表文章数
	Views     int    `json:"views"`     // 浏览量
	Likes     int    `json:"likes"`     // 点赞数
	Comments  int    `json:"comments"`  // 评论数
	Favorites int    `json:"favorites"` // 收藏数
}

// ArticleStatItem 文章统计项
type ArticleStatItem struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
	Count int64  `json:"count"`
}

// ArticleStatsResponse 文章统计响应
type ArticleStatsResponse struct {
	TotalArticles     int64             `json:"total_articles"`
	PublishedArticles int64             `json:"published_articles"`
	DraftArticles     int64             `json:"draft_articles"`
	TotalViews        int64             `json:"total_views"`
	TotalLikes        int64             `json:"total_likes"`
	TotalComments     int64             `json:"total_comments"`
	TotalFavorites    int64             `json:"total_favorites"`
	TotalWordCount    int64             `json:"total_word_count"`
	TopViewedArticles []ArticleStatItem `json:"top_viewed_articles"`
	TopLikedArticles  []ArticleStatItem `json:"top_liked_articles"`
}
