package dto

// ReadingHistoryCreateRequest 创建阅读历史记录请求
type ReadingHistoryCreateRequest struct {
	ArticleID uint `json:"article_id" binding:"required"` // 文章ID
}

// ReadingHistoryQueryRequest 阅读历史查询请求
type ReadingHistoryQueryRequest struct {
	CategoryID uint   `form:"category_id"`                                           // 分类ID
	TagID      uint   `form:"tag_id"`                                                // 标签ID
	Keyword    string `form:"keyword"`                                               // 关键词搜索
	StartDate  string `form:"start_date"`                                            // 开始日期
	EndDate    string `form:"end_date"`                                              // 结束日期
	OrderBy    string `form:"order_by" binding:"omitempty,oneof=read_at created_at"` // 排序字段
	Order      string `form:"order" binding:"omitempty,oneof=asc desc"`              // 排序方向
	Page       int    `form:"page" binding:"required,min=1"`                         // 页码
	PageSize   int    `form:"page_size" binding:"required,min=5,max=50"`             // 每页条数
}

// ReadingHistoryItem 阅读历史列表项
type ReadingHistoryItem struct {
	ID             uint   `json:"id"`              // 记录ID
	ArticleID      uint   `json:"article_id"`      // 文章ID
	ArticleTitle   string `json:"article_title"`   // 文章标题
	ArticleSummary string `json:"article_summary"` // 文章摘要
	ArticleCover   string `json:"article_cover"`   // 文章封面
	CategoryID     uint   `json:"category_id"`     // 分类ID
	CategoryName   string `json:"category_name"`   // 分类名称
	AuthorID       uint   `json:"author_id"`       // 作者ID
	AuthorName     string `json:"author_name"`     // 作者名称
	ReadAt         string `json:"read_at"`         // 最后阅读时间
	CreatedAt      string `json:"created_at"`      // 创建时间
}

// ReadingHistoryListResponse 阅读历史列表响应
type ReadingHistoryListResponse struct {
	Total int64                `json:"total"` // 总数
	List  []ReadingHistoryItem `json:"list"`  // 列表项
}

// ReadingHistoryDeleteRequest 删除阅读历史请求
type ReadingHistoryDeleteRequest struct {
	IDs []uint `json:"ids" binding:"required,dive,min=1"` // 要删除的记录ID列表
}
