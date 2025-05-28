package auth

import (
	"sync"
	"time"
)

// TokenBlacklist 令牌黑名单，用于管理已失效的令牌
type TokenBlacklist struct {
	tokens map[string]time.Time // 令牌->过期时间映射
	mutex  sync.RWMutex         // 读写锁，保证并发安全
}

var (
	blacklist     *TokenBlacklist
	blacklistOnce sync.Once
)

// GetTokenBlacklist 获取令牌黑名单单例
func GetTokenBlacklist() *TokenBlacklist {
	blacklistOnce.Do(func() {
		blacklist = &TokenBlacklist{
			tokens: make(map[string]time.Time),
		}
		// 启动定期清理过期令牌的goroutine
		go blacklist.cleanupTask()
	})
	return blacklist
}

// AddToBlacklist 将令牌添加到黑名单
func (b *TokenBlacklist) AddToBlacklist(token string, expireAt time.Time) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.tokens[token] = expireAt
}

// IsBlacklisted 检查令牌是否在黑名单中
func (b *TokenBlacklist) IsBlacklisted(token string) bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	_, exists := b.tokens[token]
	return exists
}

// cleanupTask 定期清理过期的令牌
func (b *TokenBlacklist) cleanupTask() {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		b.cleanup()
	}
}

// cleanup 清理过期的令牌
func (b *TokenBlacklist) cleanup() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	now := time.Now()
	for token, expireAt := range b.tokens {
		if now.After(expireAt) {
			delete(b.tokens, token)
		}
	}
}
