package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int    `json:"code"`           // 状态码
	Message string `json:"message"`        // 响应消息
	Data    any    `json:"data"`           // 响应数据
	Meta    any    `json:"meta,omitempty"` // 元数据，如分页信息
}

// PageMeta 分页元数据
type PageMeta struct {
	Page  int   `json:"page"`  // 当前页码
	Size  int   `json:"size"`  // 每页大小
	Total int64 `json:"total"` // 总记录数
}

// NewPageMeta 创建分页元数据
func NewPageMeta(page, size int, total int64) PageMeta {
	return PageMeta{
		Page:  page,
		Size:  size,
		Total: total,
	}
}

// Success 返回成功响应
func Success(c *gin.Context, message string, data any) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: message,
		Data:    data,
	})
}

// SuccessPage 返回分页成功响应
func SuccessPage(c *gin.Context, message string, data any, page, size int, total int64) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: message,
		Data:    data,
		Meta: PageMeta{
			Page:  page,
			Size:  size,
			Total: total,
		},
	})
}

// Error 错误响应
func Error(c *gin.Context, code int, message string, err error) {
	// 记录详细错误信息，但不向客户端暴露
	if err != nil {
		c.Error(err)
	}

	c.JSON(code, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

// BadRequest 400错误响应
func BadRequest(c *gin.Context, message string, err error) {
	Error(c, http.StatusBadRequest, message, err)
}

// Unauthorized 401错误响应
func Unauthorized(c *gin.Context, message string, err error) {
	Error(c, http.StatusUnauthorized, message, err)
}

// Forbidden 403错误响应
func Forbidden(c *gin.Context, message string, err error) {
	Error(c, http.StatusForbidden, message, err)
}

// NotFound 404错误响应
func NotFound(c *gin.Context, message string, err error) {
	Error(c, http.StatusNotFound, message, err)
}

// InternalServerError 500错误响应
func InternalServerError(c *gin.Context, message string, err error) {
	Error(c, http.StatusInternalServerError, message, err)
}

// CustomError 返回自定义错误
func CustomError(c *gin.Context, code int, message string, data any) {
	Error(c, code, message, nil)
}
