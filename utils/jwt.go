package utils

import (
	"blog/models/ctypes"
	"blog/service/redis_ser"
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"blog/global"

	"github.com/dgrijalva/jwt-go"
	"go.uber.org/zap"
)

type PayLoad struct {
	Account string          `json:"account"`
	Role    ctypes.UserRole `json:"role"`
	UserID  uint            `json:"user_id"`
}

type CustomClaims struct {
	PayLoad
	jwt.StandardClaims
}

type RefreshClaims struct {
	UserID uint `json:"user_id"`
	jwt.StandardClaims
}

// GenerateAccessToken 生成 Access Token
func GenerateAccessToken(payload PayLoad) (string, error) {
	claims := CustomClaims{
		PayLoad: payload,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(15 * time.Minute).Unix(),
			Issuer:    global.Config.Jwt.Issuer,
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(global.Config.Jwt.Secret))
}

// GenerateRefreshToken 生成 Refresh Token
func GenerateRefreshToken(userID uint) (string, error) {
	claims := RefreshClaims{
		UserID: userID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Duration(global.Config.Jwt.Expires) * 24 * time.Hour).Unix(),
			Issuer:    global.Config.Jwt.Issuer,
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(global.Config.Jwt.Secret))
}

// ParseToken 解析  Token
func ParseToken(tokenString string) (*CustomClaims, error) {
	var claims CustomClaims
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(global.Config.Jwt.Secret), nil
	})

	if err != nil {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorExpired != 0 {
				return nil, errors.New("token已过期")
			} else if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				return nil, errors.New("token格式错误")
			} else if ve.Errors&jwt.ValidationErrorSignatureInvalid != 0 {
				return nil, errors.New("token签名无效")
			} else if ve.Errors&jwt.ValidationErrorNotValidYet != 0 {
				return nil, errors.New("token尚未生效")
			}
		}
		return nil, errors.New("token无效")
	}

	if !token.Valid {
		return nil, errors.New("token验证失败")
	}

	return &claims, nil
}

// 解析过期token
func ParseExpiredToken(tokenString string) (*CustomClaims, error) {
	token, _ := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(global.Config.Jwt.Secret), nil
	})

	if claims, ok := token.Claims.(*CustomClaims); ok {
		return claims, nil
	}
	return nil, errors.New("无法解析token")
}

// RefreshToken 刷新访问令牌和刷新令牌
func RefreshToken(aToken, rToken string) (newAToken, newRToken string, err error) {
	// 步骤1: 解析并验证刷新令牌
	var rClaims RefreshClaims
	rToken, err = validateRefreshToken(rToken, &rClaims)
	if err != nil {
		return "", "", err
	}

	// 步骤2: 处理访问令牌
	aToken, err = handleAccessToken(aToken, rClaims.UserID)
	if err != nil {
		return "", "", err
	}

	return aToken, rToken, nil
}

// validateRefreshToken 验证刷新令牌并在需要时生成新的刷新令牌
func validateRefreshToken(rToken string, rClaims *RefreshClaims) (string, error) {
	// 解析刷新令牌
	token, err := jwt.ParseWithClaims(rToken, rClaims, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(global.Config.Jwt.Secret), nil
	})

	// 处理刷新令牌解析错误
	if err != nil || !token.Valid {
		global.Log.Error("刷新令牌验证失败",
			zap.String("token", rToken),
			zap.String("error", err.Error()),
		)
		return "", errors.New("refresh token无效")

	}

	// 检查刷新令牌是否即将过期
	refreshThreshold := time.Duration(global.Config.Jwt.RefreshThreshold) * 24 * time.Hour
	timeUntilExpiry := time.Until(time.Unix(rClaims.ExpiresAt, 0))

	// 如果刷新令牌即将过期，生成新的刷新令牌
	if timeUntilExpiry < refreshThreshold {
		newRToken, err := GenerateRefreshToken(rClaims.UserID)
		if err != nil {
			global.Log.Error("生成新的刷新令牌失败",
				zap.Uint("userID", rClaims.UserID),
				zap.String("error", err.Error()),
			)
			return "", errors.New("生成新的刷新令牌失败")

		}
		return newRToken, nil
	}

	// 如果刷新令牌仍然有效，返回原令牌
	return rToken, nil
}

// handleAccessToken 处理访问令牌的验证和刷新
func handleAccessToken(aToken string, userID uint) (string, error) {
	var aClaims CustomClaims

	// 解析访问令牌
	_, err := jwt.ParseWithClaims(aToken, &aClaims, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(global.Config.Jwt.Secret), nil
	})

	// 处理访问令牌解析结果
	if err != nil {
		// 检查是否是过期错误
		if vErr, ok := err.(*jwt.ValidationError); ok && vErr.Errors&jwt.ValidationErrorExpired != 0 {
			// 生成新的访问令牌
			newAToken, err := GenerateAccessToken(PayLoad{
				UserID:  aClaims.UserID,
				Account: aClaims.Account,
				Role:    aClaims.Role,
			})
			if err != nil {
				global.Log.Error("生成新的访问令牌失败",
					zap.Uint("userID", userID),
					zap.String("error", err.Error()),
				)
				return "", errors.New("生成新的访问令牌失败")

			}
			return newAToken, nil
		}

		// 其他错误情况
		global.Log.Error("访问令牌验证失败",
			zap.String("token", aToken),
			zap.String("error", err.Error()),
		)
		return "", errors.New("访问令牌无效")
	}

	// 访问令牌仍然有效，返回原令牌
	return aToken, nil
}

// 添加令牌黑名单相关
const (
	TokenBlacklist = "token_blacklist:"
	BlacklistTTL   = 30 * time.Minute // 略大于 access token 的有效期
)

// 添加登出时令牌处理
func InvalidateTokens(userID uint, accessToken string) error {
	// 将 access token 加入黑名单
	err := global.Redis.Set(context.Background(),
		TokenBlacklist+accessToken,
		"invalid",
		BlacklistTTL).Err()
	if err != nil {
		return err
	}

	// 删除 refresh token
	key := redis_ser.RefreshToken + strconv.Itoa(int(userID))
	return global.Redis.Del(context.Background(), redis_ser.GetRedisKey(key)).Err()
}
