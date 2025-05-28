package service

import (
	"errors"
	"strings"
	"sync"
	"time"

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
