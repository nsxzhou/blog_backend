package service

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/nsxzhou1114/blog-api/internal/config"
	"github.com/nsxzhou1114/blog-api/internal/database"
	"github.com/nsxzhou1114/blog-api/internal/dto"
	"github.com/nsxzhou1114/blog-api/internal/logger"
	"github.com/nsxzhou1114/blog-api/internal/model"
	"github.com/nsxzhou1114/blog-api/pkg/auth"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	userService     *UserService
	userServiceOnce sync.Once
)

type UserService struct {
	db     *gorm.DB
	logger *zap.SugaredLogger
}

func NewUserService() *UserService {
	userServiceOnce.Do(func() {
		userService = &UserService{
			db:     database.GetDB(),
			logger: logger.GetSugaredLogger(),
		}
	})
	return userService
}

// Register 用户注册
func (s *UserService) Register(req *dto.RegisterRequest) (*model.User, *auth.TokenPair, error) {
	// 检查用户名是否已存在
	var count int64
	if err := s.db.Model(&model.User{}).Where("username = ?", req.Username).Count(&count).Error; err != nil {
		return nil, nil, err
	}
	if count > 0 {
		return nil, nil, errors.New("用户名已存在")
	}

	// 检查邮箱是否已存在
	if err := s.db.Model(&model.User{}).Where("email = ?", req.Email).Count(&count).Error; err != nil {
		return nil, nil, err
	}
	if count > 0 {
		return nil, nil, errors.New("邮箱已存在")
	}

	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, err
	}

	// 创建用户
	user := &model.User{
		Username:             req.Username,
		Password:             string(hashedPassword),
		Email:                req.Email,
		Nickname:             req.Nickname,
		Role:                 "user",
		Status:               1,          // 1 表示启用
		LastLoginAt:          time.Now(), // 设置初始登录时间为当前时间
		ResetPasswordExpires: time.Now(), // 设置密码重置过期时间的初始值为当前时间
	}


	if err := s.db.Create(user).Error; err != nil {
		return nil, nil, err
	}

	// 生成Token对
	tokenPair, err := auth.GenerateTokenPair(user.ID, user.Role, false)
	if err != nil {
		return nil, nil, err
	}

	return user, tokenPair, nil
}

// Login 用户登录
func (s *UserService) Login(req *dto.LoginRequest) (*model.User, *auth.TokenPair, error) {
	var user model.User
	query := s.db.Where("status = ?", 1) // 只查询状态正常的用户

	// 判断登录方式（用户名或邮箱）
	if strings.Contains(req.Username, "@") {
		query = query.Where("email = ?", req.Username)
	} else {
		query = query.Where("username = ?", req.Username)
	}

	if err := query.First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("用户不存在")
		}
		return nil, nil, err
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, nil, errors.New("密码错误")
	}

	// 更新最后登录时间和IP
	if err := s.db.Model(&user).Updates(map[string]interface{}{
		"last_login_at": time.Now(),
		"last_login_ip": "", // 在控制器中设置真实IP
	}).Error; err != nil {
		s.logger.Warnf("更新用户登录信息失败: %v", err)
	}

	// 生成Token对
	tokenPair, err := auth.GenerateTokenPair(user.ID, user.Role, req.Remember)
	if err != nil {
		return nil, nil, err
	}

	return &user, tokenPair, nil
}

// RefreshToken 刷新访问令牌
func (s *UserService) RefreshToken(refreshToken string) (*auth.TokenPair, error) {
	// 验证并刷新令牌
	tokenPair, err := auth.RefreshAccessToken(refreshToken)
	if err != nil {
		return nil, err
	}

	return tokenPair, nil
}

// Logout 用户登出
func (s *UserService) Logout(accessToken string, refreshToken string) error {
	// 将两个令牌都加入黑名单
	if accessToken != "" {
		if err := auth.RevokeToken(accessToken); err != nil {
			s.logger.Warnf("撤销访问令牌失败: %v", err)
			// 继续处理刷新令牌，不返回错误
		}
	}

	if refreshToken != "" {
		if err := auth.RevokeToken(refreshToken); err != nil {
			s.logger.Warnf("撤销刷新令牌失败: %v", err)
			return err
		}
	}

	return nil
}

// UpdateLoginIP 更新用户最后登录IP
func (s *UserService) UpdateLoginIP(userID uint, ip string) error {
	return s.db.Model(&model.User{}).Where("id = ?", userID).Update("last_login_ip", ip).Error
}

// GetUserByID 根据ID获取用户
func (s *UserService) GetUserByID(id uint) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *UserService) GetUserByUsername(username string) (*model.User, error) {
	var user model.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}
	return &user, nil
}

// UpdateUserInfo 更新用户信息
func (s *UserService) UpdateUserInfo(id uint, req *dto.UserInfoUpdateRequest) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}

	// 更新用户信息
	updates := map[string]interface{}{}
	if req.Nickname != "" {
		updates["nickname"] = req.Nickname
	}
	if req.Bio != "" {
		updates["bio"] = req.Bio
	}
	if req.Avatar != "" {
		updates["avatar"] = req.Avatar
	}
	if req.Email != "" && req.Email != user.Email {
		// 检查邮箱是否已被使用
		var count int64
		if err := s.db.Model(&model.User{}).Where("email = ? AND id != ?", req.Email, id).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, errors.New("邮箱已被使用")
		}
		updates["email"] = req.Email
		updates["is_verified"] = 0 // 新邮箱需要重新验证
	}
	if req.Phone != "" && req.Phone != user.Phone {
		// 检查手机号是否已被使用
		var count int64
		if err := s.db.Model(&model.User{}).Where("phone = ? AND id != ?", req.Phone, id).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, errors.New("手机号已被使用")
		}
		updates["phone"] = req.Phone
		updates["is_phone_verified"] = 0 // 新手机号需要重新验证
	}

	if len(updates) > 0 {
		if err := s.db.Model(&user).Updates(updates).Error; err != nil {
			return nil, err
		}
		// 重新查询完整信息
		if err := s.db.First(&user, id).Error; err != nil {
			return nil, err
		}
	}

	return &user, nil
}

// ChangePassword 修改密码
func (s *UserService) ChangePassword(id uint, req *dto.ChangePasswordRequest) error {
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		return err
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		return errors.New("旧密码错误")
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 更新密码
	if err := s.db.Model(&user).Update("password", string(hashedPassword)).Error; err != nil {
		return err
	}

	return nil
}

// GenerateUserResponse 生成用户响应DTO
func (s *UserService) GenerateUserResponse(user *model.User) *dto.UserResponse {
	return &dto.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Avatar:    user.Avatar,
		Nickname:  user.Nickname,
		Bio:       user.Bio,
		Role:      user.Role,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// GenerateUserResponseWithStats 生成带统计信息的用户响应DTO
func (s *UserService) GenerateUserResponseWithStats(user *model.User, currentUserID *uint) *dto.UserResponse {
	userResp := &dto.UserResponse{
		ID:              user.ID,
		Username:        user.Username,
		Email:           user.Email,
		Avatar:          user.Avatar,
		Nickname:        user.Nickname,
		Bio:             user.Bio,
		Role:            user.Role,
		Status:          user.Status,
		IsVerified:      user.IsVerified,
		IsPhoneVerified: user.IsPhoneVerified,
		Phone:           user.Phone,
		CreatedAt:       user.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	if !user.LastLoginAt.IsZero() {
		userResp.LastLoginAt = user.LastLoginAt.Format("2006-01-02 15:04:05")
	}
	if user.LastLoginIP != "" {
		userResp.LastLoginIP = user.LastLoginIP
	}

	// 获取关注统计
	s.db.Model(&model.UserFollow{}).Where("followed_id = ?", user.ID).Count(&userResp.FollowerCount)
	s.db.Model(&model.UserFollow{}).Where("follower_id = ?", user.ID).Count(&userResp.FollowingCount)

	// 如果有当前用户，检查是否已关注
	if currentUserID != nil && *currentUserID != user.ID {
		var count int64
		s.db.Model(&model.UserFollow{}).Where("follower_id = ? AND followed_id = ?", *currentUserID, user.ID).Count(&count)
		userResp.IsFollowedByMe = count > 0
	}

	return userResp
}

// GetUserByIDWithStats 根据ID获取用户（带统计信息）
func (s *UserService) GetUserByIDWithStats(id uint, currentUserID *uint) (*dto.UserResponse, error) {
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}

	return s.GenerateUserResponseWithStats(&user, currentUserID), nil
}

// GetUserDetail 获取用户详情
func (s *UserService) GetUserDetail(id uint, currentUserID *uint) (*dto.UserDetailResponse, error) {
	user, err := s.GetUserByIDWithStats(id, currentUserID)
	if err != nil {
		return nil, err
	}

	detail := &dto.UserDetailResponse{
		UserResponse: *user,
	}

	// 获取详细统计信息
	s.db.Model(&model.Article{}).Where("author_id = ?", id).Count(&detail.ArticleCount)
	s.db.Model(&model.Comment{}).Where("user_id = ?", id).Count(&detail.CommentCount)
	s.db.Model(&model.ArticleLike{}).Where("user_id = ?", id).Count(&detail.LikeCount)
	s.db.Model(&model.Favorite{}).Where("user_id = ?", id).Count(&detail.FavoriteCount)

	return detail, nil
}

// FollowUser 关注用户
func (s *UserService) FollowUser(followerID, followedID uint) error {
	if followerID == followedID {
		return errors.New("不能关注自己")
	}

	// 检查被关注用户是否存在
	var followedUser model.User
	if err := s.db.First(&followedUser, followedID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("用户不存在")
		}
		return err
	}

	// 检查是否已经关注
	var count int64
	if err := s.db.Model(&model.UserFollow{}).Where("follower_id = ? AND followed_id = ?", followerID, followedID).Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return errors.New("已经关注该用户")
	}

	// 创建关注关系
	follow := &model.UserFollow{
		FollowerID: followerID,
		FollowedID: followedID,
	}

	if err := s.db.Create(follow).Error; err != nil {
		return err
	}

	// 异步创建通知
	go func() {
		notificationService := NewNotificationService()
		var follower model.User
		if err := s.db.Select("nickname").First(&follower, followerID).Error; err == nil {
			notificationService.CreateFollowNotification(followerID, followedID, follower.Nickname)
		}
	}()

	return nil
}

// UnfollowUser 取消关注用户
func (s *UserService) UnfollowUser(followerID, followedID uint) error {
	if followerID == followedID {
		return errors.New("不能取消关注自己")
	}

	result := s.db.Where("follower_id = ? AND followed_id = ?", followerID, followedID).Delete(&model.UserFollow{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("未关注该用户")
	}

	return nil
}

// GetFollowers 获取粉丝列表
func (s *UserService) GetFollowers(userID uint, page, pageSize int) (*dto.FollowListResponse, error) {
	var total int64
	var follows []model.UserFollow

	// 获取总数
	if err := s.db.Model(&model.UserFollow{}).Where("followed_id = ?", userID).Count(&total).Error; err != nil {
		return nil, err
	}

	// 获取关注列表
	if err := s.db.Where("followed_id = ?", userID).
		Preload("Follower").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&follows).Error; err != nil {
		return nil, err
	}

	list := make([]dto.UserBriefInfo, 0, len(follows))
	for _, follow := range follows {
		list = append(list, dto.UserBriefInfo{
			ID:       follow.Follower.ID,
			Username: follow.Follower.Username,
			Nickname: follow.Follower.Nickname,
			Avatar:   follow.Follower.Avatar,
			Bio:      follow.Follower.Bio,
		})
	}

	return &dto.FollowListResponse{
		Total: total,
		List:  list,
	}, nil
}

// GetFollowing 获取关注列表
func (s *UserService) GetFollowing(userID uint, page, pageSize int) (*dto.FollowListResponse, error) {
	var total int64
	var follows []model.UserFollow

	// 获取总数
	if err := s.db.Model(&model.UserFollow{}).Where("follower_id = ?", userID).Count(&total).Error; err != nil {
		return nil, err
	}

	// 获取关注列表
	if err := s.db.Where("follower_id = ?", userID).
		Preload("Followed").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&follows).Error; err != nil {
		return nil, err
	}

	list := make([]dto.UserBriefInfo, 0, len(follows))
	for _, follow := range follows {
		list = append(list, dto.UserBriefInfo{
			ID:       follow.Followed.ID,
			Username: follow.Followed.Username,
			Nickname: follow.Followed.Nickname,
			Avatar:   follow.Followed.Avatar,
			Bio:      follow.Followed.Bio,
		})
	}

	return &dto.FollowListResponse{
		Total: total,
		List:  list,
	}, nil
}

// GetUserList 获取用户列表（管理员）
func (s *UserService) GetUserList(req *dto.UserListRequest) (*dto.UserListResponse, error) {
	var total int64
	var users []model.User

	query := s.db.Model(&model.User{})

	// 条件过滤
	if req.Keyword != "" {
		query = query.Where("username LIKE ? OR nickname LIKE ? OR email LIKE ?", 
			"%"+req.Keyword+"%", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}
	if req.Role != "" {
		query = query.Where("role = ?", req.Role)
	}
	if req.Status >= 0 {
		if req.Status == 2 {
			query = query.Where("status >= 0")
		} else {
			query = query.Where("status = ?", req.Status)
		}
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 排序
	orderBy := "created_at"
	if req.OrderBy != "" {
		orderBy = req.OrderBy
	}
	order := "DESC"
	if req.Order != "" {
		order = strings.ToUpper(req.Order)
	}

	// 分页查询
	if err := query.Order(orderBy + " " + order).
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Find(&users).Error; err != nil {
		return nil, err
	}

	list := make([]dto.UserResponse, 0, len(users))
	for _, user := range users {
		list = append(list, *s.GenerateUserResponseWithStats(&user, nil))
	}

	return &dto.UserListResponse{
		Total: total,
		List:  list,
	}, nil
}

// UpdateUserStatus 更新用户状态（管理员）
func (s *UserService) UpdateUserStatus(id uint, status int) error {
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("用户不存在")
		}
		return err
	}

	return s.db.Model(&user).Update("status", status).Error
}

// ResetUserPassword 重置用户密码（管理员）
func (s *UserService) ResetUserPassword(id uint, newPassword string) error {
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("用户不存在")
		}
		return err
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.db.Model(&user).Update("password", string(hashedPassword)).Error
}

// BatchUserAction 批量用户操作（管理员）
func (s *UserService) BatchUserAction(userIDs []uint, action string) error {
	if len(userIDs) == 0 {
		return errors.New("请选择要操作的用户")
	}

	switch action {
	case "enable":
		return s.db.Model(&model.User{}).Where("id IN ?", userIDs).Update("status", 1).Error
	case "disable":
		return s.db.Model(&model.User{}).Where("id IN ?", userIDs).Update("status", 0).Error
	case "delete":
		return s.db.Where("id IN ?", userIDs).Delete(&model.User{}).Error
	default:
		return errors.New("不支持的操作类型")
	}
}

// ForgotPassword 忘记密码
func (s *UserService) ForgotPassword(email string) error {
	var user model.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("邮箱不存在")
		}
		return err
	}

	// 生成重置令牌（这里简化处理，实际应用中应该生成安全的随机令牌）
	resetToken := s.generateResetToken()
	resetExpires := time.Now().Add(24 * time.Hour) // 24小时后过期

	// 更新用户重置信息
	if err := s.db.Model(&user).Updates(map[string]interface{}{
		"reset_password_token":   resetToken,
		"reset_password_expires": resetExpires,
	}).Error; err != nil {
		return err
	}

	// 发送重置邮件（这里需要实现邮件发送功能）
	s.logger.Infof("重置密码邮件已发送到: %s, 令牌: %s", email, resetToken)

	return nil
}

// ResetPassword 重置密码
func (s *UserService) ResetPassword(token, newPassword string) error {
	var user model.User
	if err := s.db.Where("reset_password_token = ? AND reset_password_expires > ?", 
		token, time.Now()).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("重置令牌无效或已过期")
		}
		return err
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 更新密码并清除重置令牌
	return s.db.Model(&user).Updates(map[string]interface{}{
		"password":                string(hashedPassword),
		"reset_password_token":    "",
		"reset_password_expires":  time.Now(),
	}).Error
}

// SendVerificationEmail 发送验证邮件
func (s *UserService) SendVerificationEmail(email string) error {
	var user model.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("邮箱不存在")
		}
		return err
	}

	if user.IsVerified == 1 {
		return errors.New("邮箱已验证")
	}

	// 生成验证令牌
	verificationToken := s.generateVerificationToken()

	// 更新用户验证令牌
	if err := s.db.Model(&user).Update("verification_token", verificationToken).Error; err != nil {
		return err
	}

	// 发送验证邮件（这里需要实现邮件发送功能）
	s.logger.Infof("验证邮件已发送到: %s, 令牌: %s", email, verificationToken)

	return nil
}

// VerifyEmail 验证邮箱
func (s *UserService) VerifyEmail(token string) error {
	var user model.User
	if err := s.db.Where("verification_token = ?", token).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("验证令牌无效")
		}
		return err
	}

	// 更新验证状态并清除令牌
	return s.db.Model(&user).Updates(map[string]interface{}{
		"is_verified":        1,
		"verification_token": "",
	}).Error
}

// GetUserStats 获取用户统计（管理员）
func (s *UserService) GetUserStats() (*dto.UserStatResponse, error) {
	stats := &dto.UserStatResponse{}

	// 总用户数
	s.db.Model(&model.User{}).Count(&stats.TotalUsers)

	// 活跃用户数（30天内登录）
	thirtyDaysAgo := time.Now().AddDate(0, 0, -7	)
	s.db.Model(&model.User{}).Where("last_login_at > ?", thirtyDaysAgo).Count(&stats.ActiveUsers)

	// 本月新用户
	currentMonth := time.Now().Format("2006-01")
	s.db.Model(&model.User{}).Where("DATE_FORMAT(created_at, '%Y-%m') = ?", currentMonth).Count(&stats.NewUsers)

	// 管理员用户	
	s.db.Model(&model.User{}).Where("role = 'admin'").Count(&stats.AdminUsers)

	// 禁用用户
	s.db.Model(&model.User{}).Where("status = 0").Count(&stats.DisabledUsers)

	return stats, nil
}

// 私有辅助方法

// createFollowNotification 创建关注通知
func (s *UserService) createFollowNotification(followerID, followedID uint) {
	var follower model.User
	if err := s.db.Select("username, nickname").First(&follower, followerID).Error; err != nil {
		return
	}

	notification := &model.Notification{
		UserID:   followedID,
		SenderID: &followerID,
		Type:     "follow",
		Content:  follower.Nickname + " 关注了你",
		IsRead:   0,
	}

	s.db.Create(notification)
}

// generateResetToken 生成重置令牌
func (s *UserService) generateResetToken() string {
	// 这里应该生成安全的随机令牌，简化处理
	return fmt.Sprintf("reset_%d_%d", time.Now().Unix(), time.Now().Nanosecond())
}

// generateVerificationToken 生成验证令牌  
func (s *UserService) generateVerificationToken() string {
	// 这里应该生成安全的随机令牌，简化处理
	return fmt.Sprintf("verify_%d_%d", time.Now().Unix(), time.Now().Nanosecond())
}

// GetQQLoginURL 获取QQ登录URL
func (s *UserService) GetQQLoginURL() string {
	baseURL := "https://graph.qq.com/oauth2.0/authorize"
	params := url.Values{}
	params.Add("response_type", "code")
	params.Add("client_id", config.GlobalConfig.QQ.AppID)
	params.Add("redirect_uri", config.GlobalConfig.QQ.RedirectURL)
	params.Add("scope", "get_user_info")

	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}

// QQLoginCallback 处理QQ登录回调
func (s *UserService) QQLoginCallback(code string, clientIP string) (*model.User, *auth.TokenPair, error) {
	// 1. 获取access_token
	accessToken, err := s.getQQAccessToken(code)
	if err != nil {
		s.logger.Errorf("获取QQ access_token失败: %v", err)
		return nil, nil, fmt.Errorf("QQ登录失败: %v", err)
	}

	// 2. 获取OpenID
	openID, err := s.getQQOpenID(accessToken)
	if err != nil {
		s.logger.Errorf("获取QQ OpenID失败: %v", err)
		return nil, nil, fmt.Errorf("QQ登录失败: %v", err)
	}

	// 3. 获取用户信息
	qqUserInfo, err := s.getQQUserInfo(accessToken, openID)
	if err != nil {
		s.logger.Errorf("获取QQ用户信息失败: %v", err)
		return nil, nil, fmt.Errorf("QQ登录失败: %v", err)
	}

	// 4. 查找或创建用户
	var user model.User
	err = s.db.Where("qq_open_id = ?", openID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 创建新用户
			user, err = s.createUserFromQQ(openID, qqUserInfo, clientIP)
			if err != nil {
				s.logger.Errorf("创建QQ用户失败: %v", err)
				return nil, nil, fmt.Errorf("创建用户失败: %v", err)
			}
		} else {
			return nil, nil, err
		}
	}

	// 5. 更新最后登录信息
	if err := s.db.Model(&user).Updates(map[string]interface{}{
		"last_login_at": time.Now(),
		"last_login_ip": clientIP,
	}).Error; err != nil {
		s.logger.Warnf("更新用户登录信息失败: %v", err)
	}

	// 6. 生成Token对
	tokenPair, err := auth.GenerateTokenPair(user.ID, user.Role, false)
	if err != nil {
		return nil, nil, err
	}

	return &user, tokenPair, nil
}

// getQQAccessToken 获取QQ access_token
func (s *UserService) getQQAccessToken(code string) (string, error) {
	tokenURL := fmt.Sprintf("https://graph.qq.com/oauth2.0/token?grant_type=authorization_code&client_id=%s&client_secret=%s&code=%s&redirect_uri=%s",
		config.GlobalConfig.QQ.AppID,
		config.GlobalConfig.QQ.AppKey,
		code,
		config.GlobalConfig.QQ.RedirectURL,
	)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(tokenURL)
	if err != nil {
		return "", fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	// 解析响应 access_token=YOUR_TOKEN&expires_in=7776000&refresh_token=YOUR_REFRESH_TOKEN
	params := s.parseQueryString(string(body))
	accessToken := params["access_token"]
	if accessToken == "" {
		return "", fmt.Errorf("access_token为空，响应: %s", string(body))
	}

	return accessToken, nil
}

// getQQOpenID 获取QQ OpenID
func (s *UserService) getQQOpenID(accessToken string) (string, error) {
	openIDURL := fmt.Sprintf("https://graph.qq.com/oauth2.0/me?access_token=%s", accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(openIDURL)
	if err != nil {
		return "", fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	// 解析响应 callback( {"client_id":"YOUR_APPID","openid":"YOUR_OPENID"} );
	openID := s.extractOpenID(string(body))
	if openID == "" {
		return "", fmt.Errorf("解析OpenID失败，响应: %s", string(body))
	}

	return openID, nil
}

// QQUserInfo QQ用户信息结构体
type QQUserInfo struct {
	Nickname   string `json:"nickname"`
	Figureurl1 string `json:"figureurl_1"`
	Gender     string `json:"gender"`
}

// getQQUserInfo 获取QQ用户信息
func (s *UserService) getQQUserInfo(accessToken, openID string) (*QQUserInfo, error) {
	userInfoURL := fmt.Sprintf("https://graph.qq.com/user/get_user_info?access_token=%s&oauth_consumer_key=%s&openid=%s",
		accessToken,
		config.GlobalConfig.QQ.AppID,
		openID,
	)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	s.logger.Debugf("QQ用户信息响应: %s", string(body))

	var userInfo QQUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("解析用户信息失败: %v, 原始数据: %s", err, string(body))
	}

	if userInfo.Nickname == "" {
		return nil, fmt.Errorf("获取用户信息失败，昵称为空: %s", string(body))
	}

	return &userInfo, nil
}

// createUserFromQQ 从QQ信息创建用户
func (s *UserService) createUserFromQQ(openID string, qqUserInfo *QQUserInfo, clientIP string) (model.User, error) {
	user := model.User{
		Username:             qqUserInfo.Nickname,
		Password:             s.generateRandomString(32), // 随机密码，QQ用户无法使用密码登录
		Nickname:             qqUserInfo.Nickname,
		Avatar:               qqUserInfo.Figureurl1,
		Role:                 "user",
		Status:               1,
		QQOpenID:             openID,
		LastLoginAt:          time.Now(),
		LastLoginIP:          clientIP,
		ResetPasswordExpires: time.Now(),
	}

	if err := s.db.Create(&user).Error; err != nil {
		return user, err
	}

	return user, nil
}

// generateRandomString 生成随机字符串
func (s *UserService) generateRandomString(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	rand.Read(b)
	for i := range b {
		b[i] = chars[b[i]%byte(len(chars))]
	}
	return string(b)
}

// parseQueryString 解析查询字符串
func (s *UserService) parseQueryString(query string) map[string]string {
	params := make(map[string]string)
	pairs := strings.Split(query, "&")
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) == 2 {
			params[kv[0]] = kv[1]
		}
	}
	return params
}

// extractOpenID 从回调响应中提取OpenID
func (s *UserService) extractOpenID(response string) string {
	// QQ返回格式: callback( {"client_id":"YOUR_APPID","openid":"YOUR_OPENID"} );
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start == -1 || end == -1 {
		return ""
	}

	jsonStr := response[start : end+1]
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return ""
	}

	if openID, ok := result["openid"].(string); ok {
		return openID
	}

	return ""
}

