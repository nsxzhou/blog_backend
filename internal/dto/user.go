package dto

// RegisterRequest 用户注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6,max=32"`
	Email    string `json:"email" binding:"omitempty,email"`
	Phone    string `json:"phone" binding:"omitempty,len=11"`
}

// LoginRequest 用户登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"` // 用户名或邮箱
	Password string `json:"password" binding:"required"`
	Remember bool   `json:"remember"` // 是否记住登录状态
}

// RefreshTokenRequest 刷新令牌请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// LogoutRequest 登出请求
type LogoutRequest struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// UserInfoUpdateRequest 用户信息更新请求
type UserInfoUpdateRequest struct {
	Username string `json:"username" binding:"required,min=1,max=50"`
	Bio      string `json:"bio" binding:"omitempty,max=255"`
	Avatar   string `json:"avatar" binding:"omitempty"`
	Email    string `json:"email" binding:"omitempty,email"`
	Phone    string `json:"phone" binding:"omitempty,len=11"`
}

// ChangePasswordRequest 密码修改请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=32"`
}

// ForgotPasswordRequest 忘记密码请求
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest 重置密码请求
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=32"`
}

// EmailVerificationRequest 邮箱验证请求
type EmailVerificationRequest struct {
	Token string `json:"token" binding:"required"`
}

// ResendVerificationRequest 重新发送验证邮件请求
type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// QQLoginURLResponse QQ登录URL响应
type QQLoginURLResponse struct {
	URL string `json:"url"`
}

// QQLoginCallbackRequest QQ登录回调请求
type QQLoginCallbackRequest struct {
	Code string `form:"code" binding:"required"`
}

// UserListRequest 用户列表请求
type UserListRequest struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Keyword  string `form:"keyword"`
	Role     string `form:"role" binding:"omitempty,oneof=admin user"`
	Status   int    `form:"status" binding:"omitempty,oneof=0 1 2"`
	OrderBy  string `form:"order_by" binding:"omitempty,oneof=created_at last_login_at"`
	Order    string `form:"order" binding:"omitempty,oneof=asc desc"`
}

// UpdateUserStatusRequest 更新用户状态请求
type UpdateUserStatusRequest struct {
	Status int `json:"status" binding:"required,oneof=0 1"`
}

// ResetUserPasswordRequest 重置用户密码请求（管理员）
type ResetUserPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=6,max=32"`
}

// BatchUserActionRequest 批量用户操作请求
type BatchUserActionRequest struct {
	UserIDs []uint `json:"user_ids" binding:"required,min=1"`
	Action  string `json:"action" binding:"required,oneof=enable disable delete"`
}

// FollowUserRequest 关注用户请求（通过路径参数，这里保留备用）
type FollowUserRequest struct {
	UserID uint `json:"user_id" binding:"required"`
}

// UserResponse 用户信息响应
type UserResponse struct {
	ID              uint   `json:"id"`
	Username        string `json:"username"`
	Avatar          string `json:"avatar"`
	Bio             string `json:"bio"`
	Role            string `json:"role"`
	Status          int    `json:"status"` // 0=禁用 1=正常
	Email           string `json:"email"`
	IsEmailVerified int    `json:"is_email_verified"`
	Phone           string `json:"phone"`
	IsPhoneVerified int    `json:"is_phone_verified"`
	CreatedAt       string `json:"created_at"`
	LastLoginAt     string `json:"last_login_at"`
	LastLoginIP     string `json:"last_login_ip"`
	FollowerCount   int64  `json:"follower_count"`    // 粉丝数
	FollowingCount  int64  `json:"following_count"`   // 关注数
	IsFollowedByMe  bool   `json:"is_followed_by_me"` // 是否被当前用户关注
}

// UserListResponse 用户列表响应
type UserListResponse struct {
	Total int64          `json:"total"`
	List  []UserResponse `json:"list"`
}

// UserDetailResponse 用户详情响应
type UserDetailResponse struct {
	UserResponse
	ArticleCount  int64 `json:"article_count"`  // 文章数
	CommentCount  int64 `json:"comment_count"`  // 评论数
	LikeCount     int64 `json:"like_count"`     // 获赞数
	FavoriteCount int64 `json:"favorite_count"` // 收藏数
}

// UserBriefInfo 用户简要信息
type UserBriefInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
	Bio      string `json:"bio"`
}

// FollowListResponse 关注列表响应
type FollowListResponse struct {
	Total int64           `json:"total"`
	List  []UserBriefInfo `json:"list"`
}

// UserStatResponse 用户统计响应
type UserStatResponse struct {
	TotalUsers    int64 `json:"total_users"`
	ActiveUsers   int64 `json:"active_users"`   // 30天内活跃用户
	NewUsers      int64 `json:"new_users"`      // 本月新用户
	AdminUsers    int64 `json:"admin_users"`    // 管理员用户
	DisabledUsers int64 `json:"disabled_users"` // 禁用用户
}
