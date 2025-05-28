package dto

// TagCreateRequest 创建标签请求
type TagCreateRequest struct {
	Name string `json:"name" binding:"required,max=50"`
}

// TagUpdateRequest 更新标签请求
type TagUpdateRequest struct {
	Name string `json:"name" binding:"required,max=50"`
}

// TagResponse 标签响应
type TagResponse struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	ArticleCount int    `json:"article_count"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// TagListRequest 标签列表请求
type TagListRequest struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Keyword  string `form:"keyword" binding:"omitempty,max=50"`
	OrderBy  string `form:"order_by" binding:"omitempty,oneof=id name article_count created_at"`
	Order    string `form:"order" binding:"omitempty,oneof=asc desc"`
}

// TagListResponse 标签列表响应
type TagListResponse struct {
	Total int64         `json:"total"`
	List  []TagResponse `json:"list"`
}

// TagCloudItem 标签云项
type TagCloudItem struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	ArticleCount int    `json:"article_count"`
}

// TagCloudResponse 标签云响应
type TagCloudResponse struct {
	List []TagCloudItem `json:"list"`
}
