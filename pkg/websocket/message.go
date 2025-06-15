package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nsxzhou1114/blog-api/internal/model"
	"github.com/redis/go-redis/v9"
)

// Message 消息接口
type Message interface {
	ToJSON() ([]byte, error)
}

// NotificationMessage 通知消息
type NotificationMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
	MessageID string      `json:"message_id,omitempty"`
}

// ToJSON 将消息转换为JSON
func (m *NotificationMessage) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// MessageStore 消息存储接口
type MessageStore interface {
	StoreOfflineMessage(ctx context.Context, userID uint, msg *model.Notification) error
	GetOfflineMessages(ctx context.Context, userID uint) ([]*model.Notification, error)
	ClearOfflineMessages(ctx context.Context, userID uint) error
}

// RedisMessageStore Redis消息存储实现
type RedisMessageStore struct {
	redis  *redis.Client
	prefix string
}

// NewRedisMessageStore 创建Redis消息存储实例
func NewRedisMessageStore(redis *redis.Client) *RedisMessageStore {
	return &RedisMessageStore{
		redis:  redis,
		prefix: "offline_notifications:",
	}
}

// StoreOfflineMessage 存储离线消息
func (s *RedisMessageStore) StoreOfflineMessage(ctx context.Context, userID uint, notification *model.Notification) error {
	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	key := s.getKey(userID)
	pipe := s.redis.Pipeline()
	pipe.LPush(ctx, key, data)
	pipe.Expire(ctx, key, 7*24*time.Hour) // 消息保存7天
	pipe.LTrim(ctx, key, 0, 99)           // 最多保存100条消息

	_, err = pipe.Exec(ctx)
	return err
}

// GetOfflineMessages 获取离线消息
func (s *RedisMessageStore) GetOfflineMessages(ctx context.Context, userID uint) ([]*model.Notification, error) {
	key := s.getKey(userID)
	data, err := s.redis.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	messages := make([]*model.Notification, 0, len(data))
	for _, item := range data {
		var notification model.Notification
		if err := json.Unmarshal([]byte(item), &notification); err != nil {
			continue
		}
		messages = append(messages, &notification)
	}

	return messages, nil
}

// ClearOfflineMessages 清除离线消息
func (s *RedisMessageStore) ClearOfflineMessages(ctx context.Context, userID uint) error {
	return s.redis.Del(ctx, s.getKey(userID)).Err()
}

// getKey 获取Redis键名
func (s *RedisMessageStore) getKey(userID uint) string {
	return fmt.Sprintf("%s%d", s.prefix, userID)
}
