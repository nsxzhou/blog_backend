package redis_ser

import (
	"blog/global"
	"context"
	"strconv"
	"time"
)

// 令牌黑名单相关
const (
	TokenBlacklist = "token_blacklist:"
	BlacklistTTL   = 30 * time.Minute
)

// 登出时令牌处理
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

// 检查令牌是否在黑名单中
func IsTokenBlacklisted(accessToken string) (bool, error) {
	accessTokenKey := GetRedisKey(TokenBlacklist + accessToken)
	result, err := global.Redis.Get(context.Background(), accessTokenKey).Result()
	if err != nil {
		return false, err
	}
	return result == "invalid", nil
}

// 设置 refresh token
func SetRefreshToken(userID uint, refreshToken string) error {
	expiration := time.Duration(global.Config.Jwt.Expires) * 24 * time.Hour
	key := RefreshToken + strconv.Itoa(int(userID))
	err := global.Redis.Set(context.Background(), GetRedisKey(key), refreshToken, expiration).Err()
	if err != nil {
		return err
	}
	return nil
}
