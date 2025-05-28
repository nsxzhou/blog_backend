package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/nsxzhou1114/blog-api/internal/config"
)

// TokenType 定义token类型
type TokenType string

const (
	// AccessToken 访问令牌，用于访问资源
	AccessToken TokenType = "access"
	// RefreshToken 刷新令牌，用于获取新的访问令牌
	RefreshToken TokenType = "refresh"
)

// Claims 自定义JWT声明结构体
type Claims struct {
	UserID   uint      `json:"user_id"`
	Role     string    `json:"role"`
	Type     TokenType `json:"type"`
	TokenID  string    `json:"jti,omitempty"`      // 令牌唯一ID，用于追踪和撤销
	Previous string    `json:"previous,omitempty"` // 前一个刷新令牌的ID，用于令牌轮换
	jwt.StandardClaims
}

// TokenPair 包含访问令牌和刷新令牌
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // 访问令牌过期时间（秒）
	TokenID      string `json:"token_id"`   // 令牌ID，用于客户端存储和管理
}

// GenerateTokenPair 生成访问令牌和刷新令牌对
func GenerateTokenPair(userID uint, role string, remember bool) (*TokenPair, error) {
	// 获取配置
	accessExpire := time.Duration(config.GlobalConfig.JWT.AccessExpireSeconds) * time.Second
	refreshExpire := time.Duration(config.GlobalConfig.JWT.RefreshExpireSeconds) * time.Second

	// 如果选择记住登录，延长token有效期
	if remember {
		accessExpire = time.Duration(7 * 24 * time.Hour)
		refreshExpire = time.Duration(30 * 24 * time.Hour)
	}

	// 生成令牌ID
	tokenID := generateTokenID()

	// 创建访问令牌
	accessToken, err := generateToken(userID, role, AccessToken, accessExpire, tokenID, "")
	if err != nil {
		return nil, err
	}

	// 创建刷新令牌
	refreshToken, err := generateToken(userID, role, RefreshToken, refreshExpire, tokenID, "")
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(accessExpire.Seconds()),
		TokenID:      tokenID,
	}, nil
}

// generateToken 创建指定类型的JWT令牌
func generateToken(userID uint, role string, tokenType TokenType, expiration time.Duration, tokenID string, previous string) (string, error) {
	// 设置token过期时间
	expireTime := time.Now().Add(expiration)

	claims := Claims{
		UserID:   userID,
		Role:     role,
		Type:     tokenType,
		TokenID:  tokenID,
		Previous: previous,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expireTime.Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    config.GlobalConfig.JWT.Issuer,
		},
	}

	// 创建签名方法
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 使用密钥签名并获得完整的编码字符串令牌
	tokenString, err := token.SignedString([]byte(config.GlobalConfig.JWT.SecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ParseToken 解析JWT令牌
func ParseToken(tokenString string) (*Claims, error) {
	// 检查令牌是否在黑名单中
	if GetTokenBlacklist().IsBlacklisted(tokenString) {
		return nil, errors.New("令牌已被撤销")
	}

	// 解析令牌
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.GlobalConfig.JWT.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	// 校验令牌
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("无效的令牌")
}

// RefreshAccessToken 使用刷新令牌获取新的访问令牌
func RefreshAccessToken(refreshTokenString string) (*TokenPair, error) {
	// 解析刷新令牌
	claims, err := ParseToken(refreshTokenString)
	if err != nil {
		return nil, err
	}

	// 验证是否为刷新令牌
	if claims.Type != RefreshToken {
		return nil, errors.New("无效的刷新令牌")
	}

	// 计算新令牌的版本和过期时间
	accessExpire := time.Duration(config.GlobalConfig.JWT.AccessExpireSeconds) * time.Second
	refreshExpire := time.Duration(config.GlobalConfig.JWT.RefreshExpireSeconds) * time.Second

	// 生成新的令牌ID
	newTokenID := generateTokenID()

	// 生成新的访问令牌
	accessToken, err := generateToken(claims.UserID, claims.Role, AccessToken, accessExpire, newTokenID, "")
	if err != nil {
		return nil, err
	}

	// 生成新的刷新令牌，并记录上一个刷新令牌的ID
	refreshToken, err := generateToken(claims.UserID, claims.Role, RefreshToken, refreshExpire, newTokenID, claims.TokenID)
	if err != nil {
		return nil, err
	}

	// 将旧的刷新令牌加入黑名单
	// 解析令牌并获取过期时间
	expireTime := time.Unix(claims.ExpiresAt, 0)
	GetTokenBlacklist().AddToBlacklist(refreshTokenString, expireTime)

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(accessExpire.Seconds()),
		TokenID:      newTokenID,
	}, nil
}

// RevokeToken 撤销令牌（登出时使用）
func RevokeToken(tokenString string) error {
	// 解析令牌
	claims, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.GlobalConfig.JWT.SecretKey), nil
	})

	if err != nil {
		return err
	}

	// 获取过期时间
	if claims, ok := claims.Claims.(*Claims); ok {
		expireTime := time.Unix(claims.ExpiresAt, 0)
		// 将令牌加入黑名单
		GetTokenBlacklist().AddToBlacklist(tokenString, expireTime)
		return nil
	}

	return errors.New("无效的令牌")
}

// generateTokenID 生成令牌唯一ID
func generateTokenID() string {
	// 简单实现，使用时间戳和随机数组合
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
}
