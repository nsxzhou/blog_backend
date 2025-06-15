package controller

import (
	"errors"
	"net/http"
	"strconv"

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
		response.Error(c, http.StatusInternalServerError, "登录失败", err)
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
		"user": api.userService.GenerateUserResponseWithStats(user, &userID),
	})
}

// GetUserDetail 获取用户详情
func (api *UserApi) GetUserDetail(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的用户ID", err)
		return
	}

	// 获取当前用户ID（如果已登录）
	var currentUserID *uint
	if id, err := getUserIDFromContext(c); err == nil {
		currentUserID = &id
	}

	userDetail, err := api.userService.GetUserDetail(uint(userID), currentUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "用户不存在", err)
			return
		}
		api.logger.Errorf("获取用户详情失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取用户详情失败", err)
		return
	}

	response.Success(c, "获取成功", gin.H{
		"user": userDetail,
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

// FollowUser 关注用户
func (api *UserApi) FollowUser(c *gin.Context) {
	currentUserID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "未授权", err)
		return
	}

	followedIDStr := c.Param("id")
	followedID, err := strconv.ParseUint(followedIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的用户ID", err)
		return
	}

	if err := api.userService.FollowUser(currentUserID, uint(followedID)); err != nil {
		api.logger.Warnf("关注用户失败: %v", err)
		response.Error(c, http.StatusBadRequest, "关注失败", err)
		return
	}

	response.Success(c, "关注成功", nil)
}

// UnfollowUser 取消关注用户
func (api *UserApi) UnfollowUser(c *gin.Context) {
	currentUserID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "未授权", err)
		return
	}

	followedIDStr := c.Param("id")
	followedID, err := strconv.ParseUint(followedIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的用户ID", err)
		return
	}

	if err := api.userService.UnfollowUser(currentUserID, uint(followedID)); err != nil {
		api.logger.Warnf("取消关注用户失败: %v", err)
		response.Error(c, http.StatusBadRequest, "取消关注失败", err)
		return
	}

	response.Success(c, "取消关注成功", nil)
}

// GetFollowers 获取粉丝列表
func (api *UserApi) GetFollowers(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "未授权", err)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	followers, err := api.userService.GetFollowers(userID, page, pageSize)
	if err != nil {
		api.logger.Errorf("获取粉丝列表失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取粉丝列表失败", err)
		return
	}

	response.Success(c, "获取成功", followers)
}

// GetFollowing 获取关注列表
func (api *UserApi) GetFollowing(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "未授权", err)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	following, err := api.userService.GetFollowing(userID, page, pageSize)
	if err != nil {
		api.logger.Errorf("获取关注列表失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取关注列表失败", err)
		return
	}

	response.Success(c, "获取成功", following)
}

// GetUserFollowers 获取指定用户的粉丝列表
func (api *UserApi) GetUserFollowers(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的用户ID", err)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	followers, err := api.userService.GetFollowers(uint(userID), page, pageSize)
	if err != nil {
		api.logger.Errorf("获取用户粉丝列表失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取粉丝列表失败", err)
		return
	}

	response.Success(c, "获取成功", followers)
}

// GetUserFollowing 获取指定用户的关注列表
func (api *UserApi) GetUserFollowing(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的用户ID", err)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	following, err := api.userService.GetFollowing(uint(userID), page, pageSize)
	if err != nil {
		api.logger.Errorf("获取用户关注列表失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取关注列表失败", err)
		return
	}

	response.Success(c, "获取成功", following)
}

// ForgotPassword 忘记密码
func (api *UserApi) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	if err := api.userService.ForgotPassword(req.Email); err != nil {
		api.logger.Warnf("发送重置密码邮件失败: %v", err)
		response.Error(c, http.StatusBadRequest, "发送重置密码邮件失败", err)
		return
	}

	response.Success(c, "重置密码邮件已发送", nil)
}

// ResetPassword 重置密码
func (api *UserApi) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	if err := api.userService.ResetPassword(req.Token, req.NewPassword); err != nil {
		api.logger.Warnf("重置密码失败: %v", err)
		response.Error(c, http.StatusBadRequest, "重置密码失败", err)
		return
	}

	response.Success(c, "密码重置成功", nil)
}

// SendVerificationEmail 发送验证邮件
func (api *UserApi) SendVerificationEmail(c *gin.Context) {
	var req dto.ResendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	if err := api.userService.SendVerificationEmail(req.Email); err != nil {
		api.logger.Warnf("发送验证邮件失败: %v", err)
		response.Error(c, http.StatusBadRequest, "发送验证邮件失败", err)
		return
	}

	response.Success(c, "验证邮件已发送", nil)
}

// VerifyEmail 验证邮箱
func (api *UserApi) VerifyEmail(c *gin.Context) {
	var req dto.EmailVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	if err := api.userService.VerifyEmail(req.Token); err != nil {
		api.logger.Warnf("邮箱验证失败: %v", err)
		response.Error(c, http.StatusBadRequest, "邮箱验证失败", err)
		return
	}

	response.Success(c, "邮箱验证成功", nil)
}

// 管理员相关接口

// GetUserList 获取用户列表（管理员）
func (api *UserApi) GetUserList(c *gin.Context) {
	var req dto.UserListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	userList, err := api.userService.GetUserList(&req)
	if err != nil {
		api.logger.Errorf("获取用户列表失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取用户列表失败", err)
		return
	}

	response.Success(c, "获取成功", userList)
}

// UpdateUserStatus 更新用户状态（管理员）
func (api *UserApi) UpdateUserStatus(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的用户ID", err)
		return
	}

	var req dto.UpdateUserStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	if err := api.userService.UpdateUserStatus(uint(userID), req.Status); err != nil {
		api.logger.Errorf("更新用户状态失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "更新用户状态失败", err)
		return
	}

	response.Success(c, "更新成功", nil)
}

// ResetUserPassword 重置用户密码（管理员）
func (api *UserApi) ResetUserPassword(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的用户ID", err)
		return
	}

	var req dto.ResetUserPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	if err := api.userService.ResetUserPassword(uint(userID), req.NewPassword); err != nil {
		api.logger.Errorf("重置用户密码失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "重置用户密码失败", err)
		return
	}

	response.Success(c, "重置密码成功", nil)
}

// BatchUserAction 批量用户操作（管理员）
func (api *UserApi) BatchUserAction(c *gin.Context) {
	var req dto.BatchUserActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	if err := api.userService.BatchUserAction(req.UserIDs, req.Action); err != nil {
		api.logger.Errorf("批量用户操作失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "批量操作失败", err)
		return
	}

	response.Success(c, "操作成功", nil)
}

// GetUserStats 获取用户统计（管理员）
func (api *UserApi) GetUserStats(c *gin.Context) {
	stats, err := api.userService.GetUserStats()
	if err != nil {
		api.logger.Errorf("获取用户统计失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取用户统计失败", err)
		return
	}

	response.Success(c, "获取成功", stats)
}

// GetQQLoginURL 获取QQ登录URL
func (api *UserApi) GetQQLoginURL(c *gin.Context) {
	loginURL := api.userService.GetQQLoginURL()

	response.Success(c, "获取成功", dto.QQLoginURLResponse{
		URL: loginURL,
	})
}

// QQLoginCallback QQ登录回调
func (api *UserApi) QQLoginCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		response.Error(c, http.StatusBadRequest, "授权码不能为空", nil)
		return
	}

	user, tokenPair, err := api.userService.QQLoginCallback(code, c.ClientIP())
	if err != nil {
		api.logger.Warnf("QQ登录失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "QQ登录失败", err)
		return
	}

	api.logger.Infof("用户QQ登录成功: %s", user.Username)

	response.Success(c, "QQ登录成功", gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_in":    tokenPair.ExpiresIn,
		"user":          api.userService.GenerateUserResponse(user),
	})
}
