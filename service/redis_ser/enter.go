package redis_ser

const (
	Prefix              = "blog:"
	RefreshToken        = "refresh_token:user_id:"
)

func GetRedisKey(key string) string {
	return Prefix + key
}
