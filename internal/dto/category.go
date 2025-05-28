package dto

// CategoryCreateRequest 创建分类请求
type CategoryCreateRequest struct {
	Name        string `json:"name" binding:"required,max=50"`
	Description string `json:"description" binding:"max=500"`
	Icon        string `json:"icon" binding:"omitempty,max=255"`
	IsVisible   int    `json:"is_visible" binding:"omitempty,oneof=0 1"`
}

// CategoryUpdateRequest 更新分类请求
type CategoryUpdateRequest struct {
	Name        string `json:"name" binding:"required,max=50"`
	Description string `json:"description" binding:"max=500"`
	Icon        string `json:"icon" binding:"omitempty,max=255"`
	IsVisible   int    `json:"is_visible" binding:"omitempty,oneof=0 1"`
}

// CategoryResponse 分类响应
type CategoryResponse struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Icon         string `json:"icon"`
	ArticleCount int    `json:"article_count"`
	IsVisible    int    `json:"is_visible"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// CategoryBriefInfo 分类简要信息
type CategoryBriefInfo struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	IsVisible   int    `json:"is_visible"`
}

// CategoryListRequest 分类列表请求
type CategoryListRequest struct {
	Page      int    `form:"page" binding:"omitempty,min=1"`
	PageSize  int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Keyword   string `form:"keyword" binding:"omitempty,max=50"`
	IsVisible *int   `form:"is_visible" binding:"omitempty,oneof=0 1"`
	OrderBy   string `form:"order_by" binding:"omitempty,oneof=id name article_count created_at"`
	Order     string `form:"order" binding:"omitempty,oneof=asc desc"`
}

// CategoryListResponse 分类列表响应
type CategoryListResponse struct {
	Total int64              `json:"total"`
	List  []CategoryResponse `json:"list"`
}

// HotCategoryItem 热门分类项
type HotCategoryItem struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	Icon         string `json:"icon"`
	ArticleCount int    `json:"article_count"`
}

// HotCategoryResponse 热门分类响应
type HotCategoryResponse struct {
	List []HotCategoryItem `json:"list"`
}
