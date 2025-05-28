package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nsxzhou1114/blog-api/internal/dto"
	"github.com/nsxzhou1114/blog-api/internal/logger"
	"github.com/nsxzhou1114/blog-api/internal/service"
	"github.com/nsxzhou1114/blog-api/pkg/response"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserApi struct {
	logger      *zap.SugaredLogger
	userService *service.UserService
}

func NewUserApi() *UserApi {
	return &UserApi{
		logger:      logger.GetSugaredLogger(),
		userService: service.NewUserService(),
	}
}

// Register 用户注册
func (api *UserApi) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	user, tokenPair, err := api.userService.Register(&req)
	if err != nil {
		api.logger.Errorf("用户注册失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "注册失败", err)
		return
	}

	response.Success(c, "注册成功", gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_in":    tokenPair.ExpiresIn,
		"user":          api.userService.GenerateUserResponse(user),
	})
}

// Login 用户登录
func (api *UserApi) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	user, tokenPair, err := api.userService.Login(&req)
	if err != nil {
		api.logger.Warnf("用户登录失败: %v", err)
		response.Error(c, http.StatusUnauthorized, "登录失败", err)
		return
	}

	// 更新最后登录IP
	api.userService.UpdateLoginIP(user.ID, c.ClientIP())

	response.Success(c, "登录成功", gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_in":    tokenPair.ExpiresIn,
		"user":          api.userService.GenerateUserResponse(user),
	})
}

// GetUserInfo 获取用户信息
func (api *UserApi) GetUserInfo(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "未授权", err)
		return
	}

	user, err := api.userService.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "用户不存在", err)
			return
		}
		api.logger.Errorf("获取用户信息失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取用户信息失败", err)
		return
	}

	response.Success(c, "获取成功", gin.H{
		"user": api.userService.GenerateUserResponse(user),
	})
}

// UpdateUserInfo 更新用户信息
func (api *UserApi) UpdateUserInfo(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "未授权", err)
		return
	}

	var req dto.UserInfoUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	user, err := api.userService.UpdateUserInfo(userID, &req)
	if err != nil {
		api.logger.Errorf("更新用户信息失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "更新用户信息失败", err)
		return
	}

	response.Success(c, "更新成功", gin.H{
		"user": api.userService.GenerateUserResponse(user),
	})
}

// ChangePassword 修改密码
func (api *UserApi) ChangePassword(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "未授权", err)
		return
	}

	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	if err := api.userService.ChangePassword(userID, &req); err != nil {
		api.logger.Warnf("修改密码失败: %v", err)
		response.Error(c, http.StatusBadRequest, "修改密码失败", err)
		return
	}

	response.Success(c, "密码修改成功", nil)
}

// RefreshToken 刷新访问令牌
func (api *UserApi) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	tokenPair, err := api.userService.RefreshToken(req.RefreshToken)
	if err != nil {
		api.logger.Warnf("刷新令牌失败: %v", err)
		response.Error(c, http.StatusUnauthorized, "刷新令牌失败", err)
		return
	}

	response.Success(c, "刷新令牌成功", gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_in":    tokenPair.ExpiresIn,
		"token_id":      tokenPair.TokenID,
	})
}

// Logout 用户登出
func (api *UserApi) Logout(c *gin.Context) {
	var req dto.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	if err := api.userService.Logout(req.AccessToken, req.RefreshToken); err != nil {
		api.logger.Warnf("用户登出失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "登出失败", err)
		return
	}

	response.Success(c, "登出成功", nil)
}
