package utils

import (
	"context"
	"strconv"

	"blog/global"
	"blog/service/redis_ser"
)

// RefreshAccessToken 通过 UserID 刷新 Access Token
func RefreshAccessToken(accessToken string, userID uint) (string, error) {
	// 从 Redis 获取对应的 refreshToken
	key := redis_ser.RefreshToken + strconv.Itoa(int(userID))
	storedRefreshToken, err := global.Redis.Get(context.Background(), redis_ser.GetRedisKey(key)).Result()
	if err != nil {
		return "", err
	}

	// 使用 refreshToken 生成新的 accessToken 和 refreshToken
	newAccessToken, newRefreshToken, err := RefreshToken(accessToken, storedRefreshToken)
	if err != nil {
		return "", err
	}

	// 更新新的 refreshToken 到 Redis 中
	if storedRefreshToken != newRefreshToken {
		err = redis_ser.SetRefreshToken(userID, newRefreshToken)
		if err != nil {
			return "", err
		}
	}

	return newAccessToken, nil
}
