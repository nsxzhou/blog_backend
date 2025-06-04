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

// ArticleApi 文章控制器
type ArticleApi struct {
	logger                    *zap.SugaredLogger
	articleService            *service.ArticleService
	articleSearchService      *service.ArticleSearchService
	articleInteractionService *service.ArticleInteractionService
}

// NewArticleApi 创建文章控制器实例
func NewArticleApi() *ArticleApi {
	return &ArticleApi{
		logger:                    logger.GetSugaredLogger(),
		articleService:            service.NewArticleService(),
		articleSearchService:      service.NewArticleSearchService(),
		articleInteractionService: service.NewArticleInteractionService(),
	}
}

// Create 创建文章
func (api *ArticleApi) Create(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "未授权", err)
		return
	}

	var req dto.ArticleCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	article, err := api.articleService.Create(userID, &req)
	if err != nil {
		api.logger.Errorf("创建文章失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "创建文章失败", err)
		return
	}

	response.Success(c, "创建成功", gin.H{
		"article_id": article.ID,
	})
}

// Update 更新文章
func (api *ArticleApi) Update(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "未授权", err)
		return
	}

	articleIDStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的文章ID", err)
		return
	}

	var req dto.ArticleUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	article, err := api.articleService.Update(userID, uint(articleID), &req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "文章不存在", err)
			return
		}
		api.logger.Errorf("更新文章失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "更新文章失败", err)
		return
	}

	response.Success(c, "更新成功", gin.H{
		"article_id": article.ID,
	})
}

// Delete 删除文章
func (api *ArticleApi) Delete(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "未授权", err)
		return
	}

	role, exists := c.Get("userRole")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "未授权", err)
		return
	}

	articleIDStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的文章ID", err)
		return
	}

	err = api.articleService.Delete(userID, uint(articleID), role.(string))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "文章不存在", err)
			return
		}
		api.logger.Errorf("删除文章失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "删除文章失败", err)
		return
	}

	response.Success(c, "删除成功", nil)
}

// GetDetail 获取文章详情
func (api *ArticleApi) GetDetail(c *gin.Context) {
	// 获取当前用户ID（如果已登录）
	var currentUserID uint = 0
	userID, err := getUserIDFromContext(c)
	if err == nil {
		currentUserID = userID
	}
	articleIDStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的文章ID", err)
		return
	}

	// 如果是密码保护的文章，验证密码
	article, err := api.articleService.GetByID(uint(articleID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "文章不存在", err)
			return
		}
		api.logger.Errorf("获取文章失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取文章失败", err)
		return
	}

	if article.AccessType == "password" && article.AuthorID != currentUserID {
		password := c.Query("password")
		if password != article.Password {
			response.Error(c, http.StatusForbidden, "需要密码访问", nil)
			return
		}
	}

	detail, err := api.articleService.GetArticleDetail(uint(articleID), currentUserID)
	if err != nil {
		api.logger.Errorf("获取文章详情失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取文章详情失败", err)
		return
	}

	response.Success(c, "获取成功", detail)
}

// GetUserArticles 获取用户文章列表
func (api *ArticleApi) GetUserArticles(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "未授权", err)
		return
	}

	var req dto.ArticleQueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	articles, err := api.articleService.GetUserArticles(userID, &req)
	if err != nil {
		api.logger.Errorf("获取用户文章列表失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取用户文章列表失败", err)
		return
	}

	response.Success(c, "获取成功", articles)
}

// GetArticlesByTag 根据标签获取文章
func (api *ArticleApi) GetArticlesByTag(c *gin.Context) {
	tagIDStr := c.Param("id")
	tagID, err := strconv.ParseUint(tagIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的标签ID", err)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	result, err := api.articleSearchService.SearchArticlesByTag(uint(tagID), page, pageSize)
	if err != nil {
		api.logger.Errorf("根据标签获取文章失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取文章失败", err)
		return
	}

	response.Success(c, "获取成功", result)
}

// GetArticlesByCategory 根据分类获取文章
func (api *ArticleApi) GetArticlesByCategory(c *gin.Context) {
	categoryIDStr := c.Param("id")
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的分类ID", err)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	result, err := api.articleSearchService.SearchArticlesByCategory(uint(categoryID), page, pageSize)
	if err != nil {
		api.logger.Errorf("根据分类获取文章失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取文章失败", err)
		return
	}

	response.Success(c, "获取成功", result)
}

// GetArticleList 通用文章列表获取（合并搜索、热门、最新等功能）
func (api *ArticleApi) GetArticleList(c *gin.Context) {
	var req dto.ArticleListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	result, err := api.articleSearchService.GetArticleList(&req)
	if err != nil {
		api.logger.Errorf("获取文章列表失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取文章列表失败", err)
		return
	}

	response.Success(c, "获取成功", result)
}

// ArticleAction 文章交互操作（点赞、收藏等）
func (api *ArticleApi) ArticleAction(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "未授权", err)
		return
	}

	var req dto.ArticleActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	articleIDStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的文章ID", err)
		return
	}

	err = api.articleInteractionService.ProcessArticleAction(userID, uint(articleID), req.Action)
	if err != nil {
		api.logger.Warnf("文章交互操作失败: %v", err)
		response.Error(c, http.StatusBadRequest, "操作失败", err)
		return
	}

	response.Success(c, "操作成功", nil)
}

// GetUserFavorites 获取用户收藏的文章
func (api *ArticleApi) 	GetUserFavorites(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "未授权", err)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	favorites, err := api.articleInteractionService.GetUserFavorites(userID, page, pageSize)
	if err != nil {
		api.logger.Errorf("获取用户收藏文章失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取收藏文章失败", err)
		return
	}

	response.Success(c, "获取成功", favorites)
}

// GetArticleLikeUsers 获取点赞文章的用户
func (api *ArticleApi) GetArticleLikeUsers(c *gin.Context) {
	articleIDStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的文章ID", err)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	users, err := api.articleInteractionService.GetArticleLikes(uint(articleID), page, pageSize)
	if err != nil {
		api.logger.Errorf("获取点赞用户失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取点赞用户失败", err)
		return
	}

	response.Success(c, "获取成功", users)
}

// GetArticleStat 获取文章统计数据
func (api *ArticleApi) GetArticleStat(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "未授权", err)
		return
	}

	stat, err := api.articleInteractionService.GetArticleStats(userID)
	if err != nil {
		api.logger.Errorf("获取文章统计数据失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "获取统计数据失败", err)
		return
	}

	response.Success(c, "获取成功", stat)
}

// CreateESIndex 创建ES索引（仅管理员）
func (api *ArticleApi) CreateESIndex(c *gin.Context) {
	err := api.articleSearchService.CreateESIndex()
	if err != nil {
		api.logger.Errorf("创建ES索引失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "创建索引失败", err)
		return
	}

	response.Success(c, "创建成功", nil)
}

// SyncArticlesToES 同步文章到ES（仅管理员）
func (api *ArticleApi) SyncArticlesToES(c *gin.Context) {
	err := api.articleSearchService.SyncArticlesToES()
	if err != nil {
		api.logger.Errorf("同步文章到ES失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "同步文章失败", err)
		return
	}

	response.Success(c, "同步成功", nil)
}

// UpdateArticleStatus 更新文章状态
func (api *ArticleApi) UpdateArticleStatus(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "未授权", err)
		return
	}

	articleIDStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的文章ID", err)
		return
	}

	var req dto.ArticleStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	err = api.articleInteractionService.UpdateArticleStatus(userID, uint(articleID), req.Status)
	if err != nil {
		api.logger.Errorf("更新文章状态失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "更新状态失败", err)
		return
	}

	response.Success(c, "更新成功", gin.H{
		"article_id": articleID,
		"status":     req.Status,
	})
}

// UpdateArticleAccess 更新文章访问权限
func (api *ArticleApi) UpdateArticleAccess(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "未授权", err)
		return
	}

	articleIDStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleIDStr, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的文章ID", err)
		return
	}

	var req dto.ArticleAccessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误", err)
		return
	}

	err = api.articleInteractionService.UpdateArticleAccess(userID, uint(articleID), req.AccessType, req.Password)
	if err != nil {
		api.logger.Errorf("更新文章访问权限失败: %v", err)
		response.Error(c, http.StatusInternalServerError, "更新访问权限失败", err)
		return
	}

	response.Success(c, "更新成功", gin.H{
		"article_id":  articleID,
		"access_type": req.AccessType,
	})
}
