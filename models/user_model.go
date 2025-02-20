package models

import (
	"errors"
	"fmt"

	"blog/global"
	"blog/models/ctypes"
	"blog/utils"

	"gorm.io/gorm"
)

// UserModel 用户模型
type UserModel struct {
	MODEL    `json:","`
	Nickname string          `json:"nick_name" gorm:"column:nick_name;size:50" validate:"required,min=2,max=50"`
	Account  string          `json:"account" gorm:"uniqueIndex:idx_account,length:191" validate:"required,min=5,max=191"`
	Password string          `json:"-" validate:"required,min=6"`
	Email    string          `json:"email"`
	Address  string          `json:"address"`
	Token    string          `json:"token"`
	QQOpenID string          `json:"qq_open_id"`
	Role     ctypes.UserRole `json:"role" validate:"required"`
}

// Create 创建用户
func (u *UserModel) Create(ip string) error {
	// 验证用户输入
	if err := utils.Validate(u); err != nil {
		return fmt.Errorf("输入验证失败: %w", err)
	}

	return global.DB.Transaction(func(tx *gorm.DB) error {
		// 检查用户是否存在
		if err := u.checkExist(); err != nil {
			return fmt.Errorf("用户检查失败: %w", err)
		}

		// 密码加密
		hashedPassword, err := utils.HashPassword(u.Password)
		if err != nil {
			return fmt.Errorf("密码处理失败: %w", err)
		}
		u.Password = hashedPassword

		// 获取地址信息
		u.Address = utils.GetAddrByIp(ip)

		// 创建用户
		if err := tx.Create(u).Error; err != nil {
			return fmt.Errorf("创建用户失败: %w", err)
		}

		return nil
	})
}

// checkExists 检查用户是否已存在
func (u *UserModel) checkExist() error {
	var exists bool
	err := global.DB.Model(&UserModel{}).
		Select("1").
		Where("nick_name = ? OR account = ?", u.Nickname, u.Account).
		Limit(1).
		Find(&exists).
		Error

	if err != nil {
		return fmt.Errorf("检查用户存在性失败: %w", err)
	}
	if exists {
		return errors.New("用户名或账号已存在")
	}
	return nil
}

// FindByNickname 根据昵称查找用户
func (u *UserModel) FindByNickname(nickname string) error {
	return global.DB.Where("nick_name = ?", nickname).Take(u).Error
}

// FindByAccount 根据账号查找用户
func (u *UserModel) FindByAccount(account string) error {
	return global.DB.Where("account = ?", account).Take(u).Error
}

// FindByQQOpenID 根据QQ OpenID查找用户
func (u *UserModel) FindByQQOpenID(qqOpenID string) error {
	return global.DB.Where("qq_open_id = ?", qqOpenID).Take(u).Error
}

// UpdatePassword 更新用户密码
func (u *UserModel) UpdatePassword(newPassword string) error {
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("密码处理失败: %w", err)
	}

	return global.DB.Model(u).Update("password", hashedPassword).Error
}

// UpdateProfile 更新用户信息
func (u *UserModel) UpdateProfile(updates map[string]interface{}) error {
	// 过滤敏感字段
	sensitiveFields := []string{"password", "account", "role", "token"}
	for _, field := range sensitiveFields {
		delete(updates, field)
	}

	return global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(u).Updates(updates).Error; err != nil {
			return fmt.Errorf("更新用户信息失败: %w", err)
		}
		return nil
	})
}

// UpdateToken 更新用户token
func (u *UserModel) UpdateToken(token string) error {
	return global.DB.Model(u).Update("token", token).Error
}

// Delete 删除用户
func (u *UserModel) Delete() error {
	return global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(u).Error; err != nil {
			return fmt.Errorf("删除用户失败: %w", err)
		}
		return nil
	})
}

// ValidatePassword 验证密码
func (u *UserModel) ValidatePassword(password string) bool {
	return utils.CheckPassword(password, u.Password)
}

// IsAdmin 检查是否为管理员
func (u *UserModel) IsAdmin() bool {
	return u.Role == ctypes.RoleAdmin
}

// GetTotalUsers 获取用户总数
func GetTotalUsers() (int64, error) {
	var count int64
	err := global.DB.Model(&UserModel{}).Count(&count).Error
	return count, err
}
