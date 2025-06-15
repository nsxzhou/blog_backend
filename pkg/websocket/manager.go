package websocket

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/nsxzhou1114/blog-api/internal/logger"
	"github.com/nsxzhou1114/blog-api/internal/model"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Manager WebSocket连接管理器
type Manager struct {
	clients    map[uint]*Client   // 客户端连接映射
	store      MessageStore       // 消息存储
	register   chan *Client       // 注册通道
	unregister chan *Client       // 注销通道
	logger     *zap.SugaredLogger // 日志记录器
	ctx        context.Context    // 上下文
	cancel     context.CancelFunc // 取消函数
	mutex      sync.RWMutex       // 并发锁
}

// Stats 统计信息
type Stats struct {
	TotalConnections  int64     `json:"total_connections"`
	ActiveConnections int       `json:"active_connections"`
	MessagesSent      int64     `json:"messages_sent"`
	MessagesReceived  int64     `json:"messages_received"`
	ConnectionErrors  int64     `json:"connection_errors"`
	LastRestart       time.Time `json:"last_restart"`
}

// BatchMessage 批量消息
type BatchMessage struct {
	UserIDs []uint
	Message []byte
}

var (
	manager     *Manager
	managerOnce sync.Once
)

// GetManager 获取WebSocket管理器单例
func GetManager() *Manager {
	managerOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		manager = &Manager{
			clients:    make(map[uint]*Client),
			register:   make(chan *Client, 32),
			unregister: make(chan *Client, 32),
			logger:     logger.GetSugaredLogger(),
			ctx:        ctx,
			cancel:     cancel,
		}
	})
	return manager
}

// Initialize 初始化管理器
func (m *Manager) Initialize(store MessageStore) {
	m.store = store
	go m.run()
}

// Shutdown 关闭管理器
func (m *Manager) Shutdown() {
	m.logger.Info("正在关闭WebSocket管理器...")
	m.cancel()

	m.mutex.Lock()
	for _, client := range m.clients {
		client.Close()
	}
	m.clients = make(map[uint]*Client)
	m.mutex.Unlock()

	m.logger.Info("WebSocket管理器已关闭")
}

// run 运行管理器主循环
func (m *Manager) run() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case client := <-m.register:
			m.handleRegister(client)
		case client := <-m.unregister:
			m.handleUnregister(client)
		case <-ticker.C:
			m.cleanInactiveConnections()
		}
	}
}

// handleRegister 处理客户端注册
func (m *Manager) handleRegister(client *Client) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 关闭已存在的连接
	if old, exists := m.clients[client.UserID]; exists {
		old.Close()
	}

	m.clients[client.UserID] = client
	m.logger.Infof("用户 %d 已连接，当前在线用户数: %d", client.UserID, len(m.clients))

	// 异步发送离线消息
	go m.sendOfflineMessages(client)
}

// handleUnregister 处理客户端注销
func (m *Manager) handleUnregister(client *Client) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if c, exists := m.clients[client.UserID]; exists && c == client {
		delete(m.clients, client.UserID)
		m.logger.Infof("用户 %d 已断开连接，当前在线用户数: %d", client.UserID, len(m.clients))
	}
}

// cleanInactiveConnections 清理不活跃的连接
func (m *Manager) cleanInactiveConnections() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	timeout := 5 * time.Minute
	for userID, client := range m.clients {
		if !client.IsActive(timeout) {
			client.Close()
			delete(m.clients, userID)
			m.logger.Infof("清理不活跃连接：用户 %d", userID)
		}
	}
}

// SendToUser 发送消息给指定用户
func (m *Manager) SendToUser(ctx context.Context, userID uint, notification *model.Notification) error {
	message := &NotificationMessage{
		Type:      "notification",
		Data:      notification,
		Timestamp: time.Now().Unix(),
		MessageID: generateConnID(userID),
	}

	data, err := message.ToJSON()
	if err != nil {
		return err
	}

	m.mutex.RLock()
	client, online := m.clients[userID]
	m.mutex.RUnlock()

	if !online || client.closed {
		return m.store.StoreOfflineMessage(ctx, userID, notification)
	}

	select {
	case client.Send <- data:
		return nil
	default:
		return m.store.StoreOfflineMessage(ctx, userID, notification)
	}
}

// SendToUsers 批量发送消息
func (m *Manager) SendToUsers(ctx context.Context, userIDs []uint, notification *model.Notification) error {
	message := &NotificationMessage{
		Type:      "notification",
		Data:      notification,
		Timestamp: time.Now().Unix(),
		MessageID: fmt.Sprintf("batch_%d", time.Now().UnixNano()),
	}

	data, err := message.ToJSON()
	if err != nil {
		return err
	}

	m.mutex.RLock()
	for _, userID := range userIDs {
		if client, exists := m.clients[userID]; exists && !client.closed {
			select {
			case client.Send <- data:
				// 消息已发送
			default:
				// 发送失败，存储为离线消息
				m.store.StoreOfflineMessage(ctx, userID, notification)
			}
		} else {
			// 用户离线，存储消息
			m.store.StoreOfflineMessage(ctx, userID, notification)
		}
	}
	m.mutex.RUnlock()

	return nil
}

// sendOfflineMessages 发送离线消息
func (m *Manager) sendOfflineMessages(client *Client) {
	messages, err := m.store.GetOfflineMessages(m.ctx, client.UserID)
	if err != nil {
		m.logger.Errorf("获取离线消息失败: %v", err)
		return
	}

	if len(messages) == 0 {
		return
	}

	for _, notification := range messages {
		message := &NotificationMessage{
			Type:      "notification",
			Data:      notification,
			Timestamp: time.Now().Unix(),
		}

		data, err := message.ToJSON()
		if err != nil {
			continue
		}

		select {
		case client.Send <- data:
			// 消息发送成功
		default:
			// 发送失败，保留消息
			return
		}
	}

	// 清理已发送的离线消息
	m.store.ClearOfflineMessages(m.ctx, client.UserID)
}

// HandleWebSocket 处理WebSocket连接
func (m *Manager) HandleWebSocket(c *gin.Context, userID uint) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		m.logger.Errorf("WebSocket升级失败: %v", err)
		return
	}

	client := NewClient(userID, conn, m)

	// 注册客户端
	m.register <- client

	// 启动客户端处理
	go client.readPump()
	go client.writePump()
}

// IsUserOnline 检查用户是否在线
func (m *Manager) IsUserOnline(userID uint) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	client, exists := m.clients[userID]
	return exists && !client.closed
}

// GetOnlineUsers 获取在线用户列表
func (m *Manager) GetOnlineUsers() []uint {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	users := make([]uint, 0, len(m.clients))
	for userID, client := range m.clients {
		if !client.closed {
			users = append(users, userID)
		}
	}
	return users
}
