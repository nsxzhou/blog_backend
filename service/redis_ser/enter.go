package redis_ser

const (
	Prefix        = "blog:"
	ArticlePrefix = Prefix + "article:"
	TokenPrefix   = Prefix + "token:"
	UserPrefix    = Prefix + "user:"
	RefreshToken  = "refresh_token:user_id:"
)

// 获取Redis键
func GetRedisKey(key string) string {
	return BuildKey(Prefix, key)
}

// 建议增加一个通用的键生成函数
func BuildKey(prefix string, parts ...string) string {
	result := prefix
	for _, part := range parts {
		result += part + ":"
	}
	return result[:len(result)-1] // 移除最后一个冒号
}
