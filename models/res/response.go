package res

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// StandardResponse 标准响应结构
type StandardResponse struct {
	Success bool         `json:"success"` // 请求是否成功
	Code    ResponseCode `json:"code"`    // 业务状态码
	Message string       `json:"message"` // 响应信息
	Data    interface{}  `json:"data"`    // 响应数据
}

// PageData 分页数据结构
type PageData[T any] struct {
	List       T     `json:"list"`        // 数据列表
	Total      int64 `json:"total"`       // 总记录数
	Page       int   `json:"page"`        // 当前页码
	PageSize   int   `json:"page_size"`   // 每页大小
	TotalPages int   `json:"total_pages"` // 总页数
	HasMore    bool  `json:"has_more"`    // 是否有更多数据
}

// 成功响应
func Success(c *gin.Context, data interface{}) {
	response(c, http.StatusOK, 0, "success", data)
}

// 成功响应带消息
func SuccessWithMsg(c *gin.Context, data interface{}, msg string) {
	response(c, http.StatusOK, 0, msg, data)
}

// 分页响应
func SuccessWithPage[T any](c *gin.Context, list T, total int64, page, pageSize int) {
	totalPages := (int(total) + pageSize - 1) / pageSize
	pageData := PageData[T]{
		List:       list,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		HasMore:    page < totalPages,
	}
	Success(c, pageData)
}

// 错误响应
func Error(c *gin.Context, code ResponseCode, msg string) {
	response(c, http.StatusOK, code, msg, nil)
}

// HTTP错误响应
func HttpError(c *gin.Context, httpStatus int, code ResponseCode, msg string) {
	response(c, httpStatus, code, msg, nil)
}

// 统一响应处理
func response(c *gin.Context, httpStatus int, code ResponseCode, msg string, data interface{}) {
	response := StandardResponse{
		Success: code == 0,
		Code:    code,
		Message: msg,
		Data:    data,
	}

	c.JSON(httpStatus, response)
}
