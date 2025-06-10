package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/nsxzhou1114/blog-api/internal/logger"
	"github.com/nsxzhou1114/blog-api/internal/model"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域
	},
}

// Client WebSocket客户端
type Client struct {
	UserID     uint
	Conn       *websocket.Conn
	Send       chan []byte
	Manager    *Manager
	LastActive time.Time
}

// Manager WebSocket管理器
type Manager struct {
	clients    map[uint]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	mutex      sync.RWMutex
	redis      *redis.Client
	logger     *zap.SugaredLogger
	ctx        context.Context
}

// NotificationMessage 通知消息结构
type NotificationMessage struct {
	Type      string      `json:"type"`      // notification, system
	Data      interface{} `json:"data"`      
	Timestamp int64       `json:"timestamp"`
}

var (
	manager     *Manager
	managerOnce sync.Once
)

// GetManager 获取WebSocket管理器单例
func GetManager() *Manager {
	managerOnce.Do(func() {
		manager = &Manager{
			clients:    make(map[uint]*Client),
			register:   make(chan *Client, 256),
			unregister: make(chan *Client, 256),
			broadcast:  make(chan []byte, 256),
			ctx:        context.Background(),
			logger:     logger.GetSugaredLogger(),
		}
	})
	return manager
}

// Initialize 初始化管理器
func (m *Manager) Initialize(redisClient *redis.Client) {
	m.redis = redisClient
	go m.run()
}

// run 运行管理器主循环
func (m *Manager) run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case client := <-m.register:
			m.registerClient(client)
		case client := <-m.unregister:
			m.unregisterClient(client)
		case message := <-m.broadcast:
			m.broadcastToAll(message)
		case <-ticker.C:
			m.cleanupInactiveClients()
		}
	}
}

// registerClient 注册客户端
func (m *Manager) registerClient(client *Client) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 如果用户已有连接，关闭旧连接
	if existingClient, exists := m.clients[client.UserID]; exists {
		close(existingClient.Send)
		existingClient.Conn.Close()
	}

	m.clients[client.UserID] = client
	m.logger.Infof("用户 %d WebSocket连接已建立，当前在线用户数: %d", client.UserID, len(m.clients))

	// 发送离线消息
	go m.sendOfflineMessages(client.UserID)
}

// unregisterClient 注销客户端
func (m *Manager) unregisterClient(client *Client) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.clients[client.UserID]; exists {
		delete(m.clients, client.UserID)
		close(client.Send)
		client.Conn.Close()
		m.logger.Infof("用户 %d WebSocket连接已断开，当前在线用户数: %d", client.UserID, len(m.clients))
	}
}

// broadcastToAll 广播消息给所有用户
func (m *Manager) broadcastToAll(message []byte) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for userID, client := range m.clients {
		select {
		case client.Send <- message:
		default:
			m.logger.Warnf("用户 %d 消息发送失败，连接可能已断开", userID)
			delete(m.clients, userID)
			close(client.Send)
			client.Conn.Close()
		}
	}
}

// SendToUser 发送消息给指定用户
func (m *Manager) SendToUser(userID uint, notification *model.Notification) error {
	m.mutex.RLock()
	client, exists := m.clients[userID]
	m.mutex.RUnlock()

	if !exists {
		// 用户不在线，存储到Redis离线消息队列
		return m.storeOfflineMessage(userID, notification)
	}

	message := NotificationMessage{
		Type:      "notification",
		Data:      notification,
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化通知消息失败: %w", err)
	}

	select {
	case client.Send <- data:
		return nil
	default:
		m.logger.Warnf("用户 %d 消息发送失败", userID)
		return m.storeOfflineMessage(userID, notification)
	}
}

// storeOfflineMessage 存储离线消息到Redis
func (m *Manager) storeOfflineMessage(userID uint, notification *model.Notification) error {
	if m.redis == nil {
		return fmt.Errorf("Redis客户端未初始化")
	}

	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("序列化离线消息失败: %w", err)
	}

	key := fmt.Sprintf("offline_notifications:%d", userID)
	if err := m.redis.LPush(m.ctx, key, data).Err(); err != nil {
		return fmt.Errorf("存储离线消息失败: %w", err)
	}

	// 设置过期时间（7天）
	m.redis.Expire(m.ctx, key, 7*24*time.Hour)
	return nil
}

// sendOfflineMessages 发送离线消息
func (m *Manager) sendOfflineMessages(userID uint) {
	if m.redis == nil {
		return
	}

	key := fmt.Sprintf("offline_notifications:%d", userID)
	messages, err := m.redis.LRange(m.ctx, key, 0, -1).Result()
	if err != nil {
		m.logger.Errorf("获取离线消息失败: %v", err)
		return
	}

	if len(messages) == 0 {
		return
	}

	m.mutex.RLock()
	client, exists := m.clients[userID]
	m.mutex.RUnlock()

	if !exists {
		return
	}

	// 发送离线消息
	for _, msgData := range messages {
		var notification model.Notification
		if err := json.Unmarshal([]byte(msgData), &notification); err != nil {
			continue
		}

		message := NotificationMessage{
			Type:      "notification",
			Data:      notification,
			Timestamp: time.Now().Unix(),
		}

		data, err := json.Marshal(message)
		if err != nil {
			continue
		}

		select {
		case client.Send <- data:
		default:
			// 发送失败，停止发送其他消息
			return
		}
	}

	// 清空离线消息
	m.redis.Del(m.ctx, key)
	m.logger.Infof("用户 %d 离线消息已发送，共 %d 条", userID, len(messages))
}

// cleanupInactiveClients 清理不活跃的客户端
func (m *Manager) cleanupInactiveClients() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	cutoff := time.Now().Add(-5 * time.Minute)
	for userID, client := range m.clients {
		if client.LastActive.Before(cutoff) {
			delete(m.clients, userID)
			close(client.Send)
			client.Conn.Close()
			m.logger.Infof("清理不活跃连接: 用户 %d", userID)
		}
	}
}

// IsUserOnline 检查用户是否在线
func (m *Manager) IsUserOnline(userID uint) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	_, exists := m.clients[userID]
	return exists
}

// GetOnlineUsers 获取在线用户列表
func (m *Manager) GetOnlineUsers() []uint {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	users := make([]uint, 0, len(m.clients))
	for userID := range m.clients {
		users = append(users, userID)
	}
	return users
}

// GetStats 获取统计信息
func (m *Manager) GetStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return map[string]interface{}{
		"online_users":  len(m.clients),
		"total_clients": len(m.clients),
	}
}

// HandleWebSocket 处理WebSocket连接
func (m *Manager) HandleWebSocket(c *gin.Context, userID uint) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		m.logger.Errorf("WebSocket升级失败: %v", err)
		return
	}

	client := &Client{
		UserID:     userID,
		Conn:       conn,
		Send:       make(chan []byte, 256),
		Manager:    m,
		LastActive: time.Now(),
	}

	client.Manager.register <- client

	// 启动goroutine处理客户端
	go client.writePump()
	go client.readPump()
}

// writePump 处理写入消息
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.Manager.logger.Errorf("写入消息失败: %v", err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump 处理读取消息
func (c *Client) readPump() {
	defer func() {
		c.Manager.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.LastActive = time.Now()
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Manager.logger.Errorf("WebSocket意外关闭: %v", err)
			}
			break
		}
		c.LastActive = time.Now()
	}
} 