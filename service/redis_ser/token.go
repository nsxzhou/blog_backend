package redis_ser

import (
	"blog/global"
	"context"
	"strconv"
	"time"
)

// 添加令牌黑名单相关
const (
	TokenBlacklist = "token_blacklist:"
	BlacklistTTL   = 30 * time.Minute // 略大于 access token 的有效期
)

// 添加登出时令牌处理
func InvalidateTokens(userID uint, accessToken string) error {
	// 将 access token 加入黑名单
	accessTokenKey := GetRedisKey(TokenBlacklist + accessToken)
	err := global.Redis.Set(context.Background(),
		accessTokenKey,
		"invalid",
		BlacklistTTL).Err()
	if err != nil {
		return err
	}

	// 删除 refresh token
	refreshTokenKey := GetRedisKey(RefreshToken + strconv.Itoa(int(userID)))
	return global.Redis.Del(context.Background(), refreshTokenKey).Err()
}
