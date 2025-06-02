package dto

// ImageUploadRequest 图片上传请求
type ImageUploadRequest struct {
	UsageType   string `form:"usage_type" binding:"required,oneof=avatar cover content"` // 使用类型：头像/封面/内容
	ArticleID   *uint  `form:"article_id"`                                               // 文章ID（内容图片时需要）
	StorageType string `form:"storage_type" binding:"omitempty,oneof=local cos"`         // 存储类型：本地/腾讯云COS
}

// ImageQueryRequest 图片查询请求
type ImageQueryRequest struct {
	UsageType   string `form:"usage_type" binding:"omitempty,oneof=avatar cover content"` // 使用类型
	ArticleID   uint  `form:"article_id"`                                                // 文章ID
	StorageType string `form:"storage_type" binding:"omitempty,oneof=local cos"`          // 存储类型
	IsExternal  int   `form:"is_external" binding:"omitempty,oneof=0 1 2"`                 // 是否外链
	StartDate   string `form:"start_date"`                                                // 开始日期
	EndDate     string `form:"end_date"`                                                  // 结束日期
	Page        int    `form:"page" binding:"required,min=1"`                             // 页码
	PageSize    int    `form:"page_size" binding:"required,min=5,max=50"`                 // 每页条数
}

// ImageUpdateRequest 图片更新请求
type ImageUpdateRequest struct {
	UsageType string `json:"usage_type" binding:"omitempty,oneof=avatar cover content"` // 使用类型
	ArticleID *uint  `json:"article_id"`                                                // 文章ID
}

// ImageListItem 图片列表项
type ImageListItem struct {
	ID           uint   `json:"id"`                      // 图片ID
	URL          string `json:"url"`                     // 图片URL
	Path         string `json:"path"`                    // 存储路径
	Filename     string `json:"filename"`                // 文件名
	Size         int    `json:"size"`                    // 文件大小（字节）
	Width        *int   `json:"width"`                   // 图片宽度
	Height       *int   `json:"height"`                  // 图片高度
	MimeType     string `json:"mime_type"`               // MIME类型
	UserID       uint   `json:"user_id"`                 // 用户ID
	UserName     string `json:"user_name"`               // 用户名
	UsageType    string `json:"usage_type"`              // 使用类型
	ArticleID    *uint  `json:"article_id"`              // 文章ID
	ArticleTitle string `json:"article_title,omitempty"` // 文章标题
	IsExternal   int    `json:"is_external"`             // 是否外链: 0=否 1=是 2=全部
	StorageType  string `json:"storage_type"`            // 存储类型: local cos
	CreatedAt    string `json:"created_at"`              // 创建时间
	UpdatedAt    string `json:"updated_at"`              // 更新时间
}

// ImageDetail 图片详情
type ImageDetail struct {
	ID           uint   `json:"id"`                      // 图片ID
	URL          string `json:"url"`                     // 图片URL
	Path         string `json:"path"`                    // 存储路径
	Filename     string `json:"filename"`                // 文件名
	Size         int    `json:"size"`                    // 文件大小（字节）
	Width        *int   `json:"width"`                   // 图片宽度
	Height       *int   `json:"height"`                  // 图片高度
	MimeType     string `json:"mime_type"`               // MIME类型
	UserID       uint   `json:"user_id"`                 // 用户ID
	UserName     string `json:"user_name"`               // 用户名
	UserAvatar   string `json:"user_avatar"`             // 用户头像
	UsageType    string `json:"usage_type"`              // 使用类型
	ArticleID    *uint  `json:"article_id"`              // 文章ID
	ArticleTitle string `json:"article_title,omitempty"` // 文章标题
	IsExternal   int    `json:"is_external"`             // 是否外链: 0=否 1=是 2=全部
	StorageType  string `json:"storage_type"`            // 存储类型: local cos
	CreatedAt    string `json:"created_at"`              // 创建时间
	UpdatedAt    string `json:"updated_at"`              // 更新时间
}

// ImageDetailResponse 图片详情响应
type ImageDetailResponse struct {
	Image ImageDetail `json:"image"` // 图片
}

// ImageListResponse 图片列表响应
type ImageListResponse struct {
	Total int64           `json:"total"` // 总数
	List  []ImageListItem `json:"list"`  // 列表项
}

// ImageUpload 图片上传
type ImageUpload struct {
	ID          uint   `json:"id"`           // 图片ID
	URL         string `json:"url"`          // 图片URL
	Path        string `json:"path"`         // 存储路径
	Filename    string `json:"filename"`     // 文件名
	Size        int    `json:"size"`         // 文件大小（字节）
	Width       *int   `json:"width"`        // 图片宽度
	Height      *int   `json:"height"`       // 图片高度
	MimeType    string `json:"mime_type"`    // MIME类型 
	UsageType   string `json:"usage_type"`   // 使用类型: avatar cover content
	StorageType string `json:"storage_type"` // 存储类型: local cos
}

// ImageUploadResponse 图片上传响应
type ImageUploadResponse struct {
	Image ImageUpload `json:"image"` // 图片
}

// ImageBatchDeleteRequest 批量删除图片请求
type ImageBatchDeleteRequest struct {
	ImageIDs []uint `json:"image_ids" binding:"required,dive,min=1"` // 图片ID列表
}

// ImageStatRequest 图片统计数据请求
type ImageStatRequest struct {
	StartDate string `form:"start_date"` // 开始日期
	EndDate   string `form:"end_date"`   // 结束日期
}

// ImageStatResponse 图片统计数据响应
type ImageStatResponse struct {
	TotalImages   int64 `json:"total_images"`   // 图片总数
	TotalSize     int64 `json:"total_size"`     // 总存储大小（字节）
	LocalImages   int64 `json:"local_images"`   // 本地存储图片数
	CosImages     int64 `json:"cos_images"`     // COS存储图片数
	AvatarImages  int64 `json:"avatar_images"`  // 头像图片数
	CoverImages   int64 `json:"cover_images"`   // 封面图片数
	ContentImages int64 `json:"content_images"` // 内容图片数
	// 按日期统计的数据
	DailyStats []ImageDailyStat `json:"daily_stats"`
}

// ImageDailyStat 图片每日统计数据
type ImageDailyStat struct {
	Date  string `json:"date"`  // 日期
	Count int    `json:"count"` // 上传数量
	Size  int64  `json:"size"`  // 存储大小（字节）
	Local int    `json:"local"` // 本地存储数量
	Cos   int    `json:"cos"`   // COS存储数量
}

// ImageStorageConfigResponse 图片存储配置响应
type ImageStorageConfigResponse struct {
	LocalEnabled    bool     `json:"local_enabled"`     // 是否启用本地存储
	CosEnabled      bool     `json:"cos_enabled"`       // 是否启用COS存储
	DefaultStorage  string   `json:"default_storage"`   // 默认存储类型
	MaxFileSize     int64    `json:"max_file_size"`     // 最大文件大小（字节）
	AllowedTypes    []string `json:"allowed_types"`     // 允许的文件类型
	LocalUploadPath string   `json:"local_upload_path"` // 本地上传路径
}
