package res

// ResponseCode 响应码类型
type ResponseCode int

const (
	// 客户端错误码 (1000-1999)
	// 通用客户端错误 (1000-1099)
	BadRequest       ResponseCode = 1000 // 错误的请求
	Unauthorized     ResponseCode = 1001 // 未授权
	Forbidden        ResponseCode = 1003 // 禁止访问
	NotFound         ResponseCode = 1004 // 资源未找到
	MethodNotAllowed ResponseCode = 1005 // 方法不允许
	Timeout          ResponseCode = 1006 // 请求超时
	TooManyRequests  ResponseCode = 1007 // 请求过于频繁

	// 参数验证错误 (1100-1199)
	InvalidParameter ResponseCode = 1100 // 无效的参数
	MissingParameter ResponseCode = 1101 // 缺少参数
	InvalidFormat    ResponseCode = 1102 // 格式错误
	InvalidJSON      ResponseCode = 1103 // JSON解析错误

	// 认证授权错误 (1200-1299)
	TokenExpired       ResponseCode = 1200 // 令牌过期
	TokenInvalid       ResponseCode = 1201 // 令牌无效
	TokenMissing       ResponseCode = 1202 // 缺少令牌
	SignatureInvalid   ResponseCode = 1203 // 签名无效
	PermissionDenied   ResponseCode = 1204 // 权限不足
	TokenRefreshFailed ResponseCode = 1205 // 令牌刷新失败

	// 服务端错误码 (2000-2999)
	// 通用服务端错误 (2000-2099)
	ServerError        ResponseCode = 2000 // 服务器内部错误
	ServiceUnavailable ResponseCode = 2001 // 服务不可用
	GatewayError       ResponseCode = 2002 // 网关错误

	// 数据库相关错误 (2100-2199)
	DBError           ResponseCode = 2100 // 数据库错误
	DBConnectionError ResponseCode = 2101 // 数据库连接错误
	DBQueryError      ResponseCode = 2102 // 数据库查询错误
	DBUpdateError     ResponseCode = 2103 // 数据库更新错误
	DBDeleteError     ResponseCode = 2104 // 数据库删除错误

	// 缓存相关错误 (2200-2299)
	CacheError       ResponseCode = 2200 // 缓存错误
	CacheKeyNotFound ResponseCode = 2201 // 缓存键不存在
	CacheExpired     ResponseCode = 2202 // 缓存已过期

	// 第三方服务错误 (2300-2399)
	ThirdPartyError ResponseCode = 2300 // 第三方服务错误
	APICallError    ResponseCode = 2301 // API调用错误
	NetworkError    ResponseCode = 2302 // 网络错误

	// 业务错误码 (3000-3999)
	// 用户相关错误 (3000-3099)
	UserNotFound      ResponseCode = 3000 // 用户不存在
	UserAlreadyExists ResponseCode = 3001 // 用户已存在
	PasswordError     ResponseCode = 3002 // 密码错误
	AccountLocked     ResponseCode = 3003 // 账号已锁定
	AccountDisabled   ResponseCode = 3004 // 账号已禁用

	// 订单相关错误 (3100-3199)
	OrderNotFound ResponseCode = 3100 // 订单不存在
	OrderExpired  ResponseCode = 3101 // 订单已过期
	OrderPaid     ResponseCode = 3102 // 订单已支付
	OrderCanceled ResponseCode = 3103 // 订单已取消

	// 支付相关错误 (3200-3299)
	PaymentFailed       ResponseCode = 3200 // 支付失败
	InsufficientBalance ResponseCode = 3201 // 余额不足
	PaymentTimeout      ResponseCode = 3202 // 支付超时

	// 文件相关错误 (3300-3399)
	FileUploadFailed   ResponseCode = 3300 // 文件上传失败
	FileDownloadFailed ResponseCode = 3301 // 文件下载失败
	FileNotFound       ResponseCode = 3302 // 文件不存在
	FileTooLarge       ResponseCode = 3303 // 文件过大
	InvalidFileType    ResponseCode = 3304 // 无效的文件类型
)

// CodeMsg 错误码消息映射
var CodeMsg = map[ResponseCode]string{
	// 客户端错误
	BadRequest:       "请求参数错误",
	Unauthorized:     "未授权访问",
	Forbidden:        "禁止访问",
	NotFound:         "资源不存在",
	MethodNotAllowed: "请求方法不允许",
	Timeout:          "请求超时",
	TooManyRequests:  "请求过于频繁",

	// 参数验证错误
	InvalidParameter: "无效的参数",
	MissingParameter: "缺少必要参数",
	InvalidFormat:    "数据格式错误",
	InvalidJSON:      "JSON解析错误",

	// 认证授权错误
	TokenExpired:     "令牌已过期",
	TokenInvalid:     "令牌无效",
	TokenMissing:     "缺少令牌",
	SignatureInvalid: "签名无效",
	PermissionDenied: "权限不足",

	// 服务端错误
	ServerError:        "服务器内部错误",
	ServiceUnavailable: "服务不可用",
	GatewayError:       "网关错误",

	// 数据库相关错误
	DBError:           "数据库操作失败",
	DBConnectionError: "数据库连接失败",
	DBQueryError:      "数据库查询失败",
	DBUpdateError:     "数据库更新失败",
	DBDeleteError:     "数据库删除失败",

	// 缓存相关错误
	CacheError:       "缓存操作失败",
	CacheKeyNotFound: "缓存数据不存在",
	CacheExpired:     "缓存数据已过期",

	// 第三方服务错误
	ThirdPartyError: "第三方服务错误",
	APICallError:    "API调用失败",
	NetworkError:    "网络连接错误",

	// 业务错误
	UserNotFound:      "用户不存在",
	UserAlreadyExists: "用户已存在",
	PasswordError:     "密码错误",
	AccountLocked:     "账号已锁定",
	AccountDisabled:   "账号已禁用",

	// 订单相关错误
	OrderNotFound: "订单不存在",
	OrderExpired:  "订单已过期",
	OrderPaid:     "订单已支付",
	OrderCanceled: "订单已取消",

	// 支付相关错误
	PaymentFailed:       "支付失败",
	InsufficientBalance: "余额不足",
	PaymentTimeout:      "支付超时",

	// 文件相关错误
	FileUploadFailed:   "文件上传失败",
	FileDownloadFailed: "文件下载失败",
	FileNotFound:       "文件不存在",
	FileTooLarge:       "文件超过大小限制",
	InvalidFileType:    "不支持的文件类型",
}

// GetMsg 获取错误码对应的消息
func GetMsg(code ResponseCode) string {
	msg, ok := CodeMsg[code]
	if ok {
		return msg
	}
	return "未知错误"
}
