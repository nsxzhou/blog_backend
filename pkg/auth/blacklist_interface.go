package auth

import "time"

// BlacklistInterface 黑名单接口
type BlacklistInterface interface {
	// AddToBlacklist 将令牌添加到黑名单
	AddToBlacklist(token string, expireAt time.Time) error

	// IsBlacklisted 检查令牌是否在黑名单中
	IsBlacklisted(token string) bool
}

// BlacklistType 黑名单类型
type BlacklistType string

const (
	// MemoryBlacklist 内存黑名单
	MemoryBlacklist BlacklistType = "memory"
	// RedisBlacklist Redis黑名单
	RedisBlacklist BlacklistType = "redis"
	// HybridBlacklist 混合黑名单（Redis + 本地缓存）
	HybridBlacklist BlacklistType = "hybrid"
)

// GetBlacklist 根据类型获取黑名单实例
func GetBlacklist(blacklistType BlacklistType) BlacklistInterface {
	switch blacklistType {
	case RedisBlacklist, HybridBlacklist:
		return GetRedisTokenBlacklist()
	case MemoryBlacklist:
		fallthrough
	default:
		return GetTokenBlacklist()
	}
}
