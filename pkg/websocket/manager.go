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
		origin := r.Header.Get("Origin")
	
		// 记录连接来源，便于调试
		if manager != nil && manager.logger != nil {
			manager.logger.Debugf("WebSocket连接来源: %s, User-Agent: %s", origin, r.Header.Get("User-Agent"))
		}
		
		// 在开发环境允许所有跨域
		// 生产环境可以根据需要限制特定域名
		return true
	},
}

// Client WebSocket客户端
type Client struct {
	UserID     uint
	Conn       *websocket.Conn
	Send       chan []byte
	Manager    *Manager
	LastActive time.Time
	closed     bool
	closeMutex sync.RWMutex 
	ConnID     string
	ConnTime   time.Time
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
	cancel     context.CancelFunc  
	stats      *Stats
	statsMutex sync.RWMutex
	messageBatch chan *BatchMessage
}

// Stats 统计信息
type Stats struct {
	TotalConnections    int64     `json:"total_connections"`
	ActiveConnections   int       `json:"active_connections"`
	MessagesSent        int64     `json:"messages_sent"`
	MessagesReceived    int64     `json:"messages_received"`
	ConnectionErrors    int64     `json:"connection_errors"`
	LastRestart         time.Time `json:"last_restart"`
}

// BatchMessage 批量消息
type BatchMessage struct {
	UserIDs []uint
	Message []byte
}

// NotificationMessage 通知消息结构
type NotificationMessage struct {
	Type      string      `json:"type"`      // notification, system, ping, pong
	Data      interface{} `json:"data"`      
	Timestamp int64       `json:"timestamp"`
	MessageID string      `json:"message_id,omitempty"` // 新增消息ID
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
			clients:      make(map[uint]*Client),
			register:     make(chan *Client, 512),    
			unregister:   make(chan *Client, 512),    
			broadcast:    make(chan []byte, 1024),    
			messageBatch: make(chan *BatchMessage, 256), 
			ctx:          ctx,
			cancel:       cancel,
			logger:       logger.GetSugaredLogger(),
			stats: &Stats{
				LastRestart: time.Now(),
			},
		}
	})
	return manager
}

// Initialize 初始化管理器
func (m *Manager) Initialize(redisClient *redis.Client) {
	m.redis = redisClient
	go m.run()
	go m.processBatchMessages() 
}

// Shutdown 优雅关闭管理器
func (m *Manager) Shutdown() {
	m.logger.Info("开始关闭WebSocket管理器...")
	
	// 取消context
	if m.cancel != nil {
		m.cancel()
	}
	
	// 关闭所有客户端连接
	m.mutex.RLock()
	clients := make([]*Client, 0, len(m.clients))
	for _, client := range m.clients {
		clients = append(clients, client)
	}
	m.mutex.RUnlock()
	
	// 并发关闭所有连接
	var wg sync.WaitGroup
	for _, client := range clients {
		wg.Add(1)
		go func(c *Client) {
			defer wg.Done()
			m.safeCloseClient(c)
		}(client)
	}
	
	// 使用带超时的通道等待连接关闭
	waitChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitChan)
	}()
	
	select {
	case <-waitChan:
		m.logger.Info("所有WebSocket连接已关闭")
	case <-time.After(10 * time.Second):
		m.logger.Warn("关闭WebSocket连接超时")
	}
	
	m.logger.Info("WebSocket管理器已关闭")
}

// run 运行管理器主循环
func (m *Manager) run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 监控goroutine
	go m.monitorHealth()

	for {
		select {
		case <-m.ctx.Done():
			m.logger.Info("WebSocket管理器收到关闭信号")
			return
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

// monitorHealth 监控系统健康状态
func (m *Manager) monitorHealth() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.logHealthStats()
		}
	}
}

// logHealthStats 记录健康统计
func (m *Manager) logHealthStats() {
	m.statsMutex.RLock()
	stats := *m.stats
	m.statsMutex.RUnlock()
	
	m.mutex.RLock()
	activeConnections := len(m.clients)
	m.mutex.RUnlock()
	
	m.logger.Infof("WebSocket健康状态 - 活跃连接: %d, 总连接数: %d, 发送消息: %d, 接收消息: %d, 连接错误: %d",
		activeConnections, stats.TotalConnections, stats.MessagesSent, stats.MessagesReceived, stats.ConnectionErrors)
}

// processBatchMessages 处理批量消息
func (m *Manager) processBatchMessages() {
	batchTicker := time.NewTicker(100 * time.Millisecond) // 100ms批量处理一次
	defer batchTicker.Stop()
	
	messageBatch := make([]*BatchMessage, 0, 20) // 预分配容量
	
	processAndClear := func() {
		if len(messageBatch) > 0 {
			m.flushBatchMessages(messageBatch)
			messageBatch = messageBatch[:0] // 复用底层数组
		}
	}
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case msg := <-m.messageBatch:
			messageBatch = append(messageBatch, msg)
			// 如果批量达到一定数量，立即处理
			if len(messageBatch) >= 10 {
				processAndClear()
			}
		case <-batchTicker.C:
			// 定时处理批量消息
			processAndClear()
		}
	}
}

// flushBatchMessages 刷新批量消息
func (m *Manager) flushBatchMessages(messages []*BatchMessage) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	for _, batchMsg := range messages {
		for _, userID := range batchMsg.UserIDs {
			if client, exists := m.clients[userID]; exists {
				client.closeMutex.RLock()
				if !client.closed {
					select {
					case client.Send <- batchMsg.Message:
						// 消息发送成功
					default:
						// channel满了，跳过此消息
						m.logger.Warnf("用户 %d 消息队列已满，跳过消息", userID)
					}
				}
				client.closeMutex.RUnlock()
			}
		}
	}
}

// registerClient 注册客户端
func (m *Manager) registerClient(client *Client) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 生成连接ID
	client.ConnID = fmt.Sprintf("%d_%d", client.UserID, time.Now().UnixNano())
	client.ConnTime = time.Now()

	// 检查是否已有相同用户的连接，强制替换
	if existingClient, exists := m.clients[client.UserID]; exists {
		m.logger.Infof("用户 %d 已有连接 %s，强制替换为新连接 %s", 
			client.UserID, existingClient.ConnID, client.ConnID)
		
		// 立即从map中移除旧连接
		delete(m.clients, client.UserID)
		
		// 异步关闭旧连接，避免阻塞注册流程
		go m.safeCloseClient(existingClient)
	}

	// 注册新连接
	m.clients[client.UserID] = client
	
	// 更新统计信息
	m.statsMutex.Lock()
	m.stats.TotalConnections++
	m.stats.ActiveConnections = len(m.clients)
	m.statsMutex.Unlock()
	
	m.logger.Infof("用户 %d WebSocket连接已建立，连接ID: %s，当前在线用户数: %d", 
		client.UserID, client.ConnID, len(m.clients))

	// 异步发送离线消息，避免阻塞
	go m.sendOfflineMessages(client.UserID)
}

// unregisterClient 注销客户端
func (m *Manager) unregisterClient(client *Client) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 检查客户端是否仍在管理器中
	if storedClient, exists := m.clients[client.UserID]; exists && storedClient == client {
		delete(m.clients, client.UserID)
		
		// 更新统计信息
		m.statsMutex.Lock()
		m.stats.ActiveConnections = len(m.clients)
		m.statsMutex.Unlock()
		
		m.logger.Infof("用户 %d WebSocket连接已从管理器移除，连接ID: %s，当前在线用户数: %d", 
			client.UserID, client.ConnID, len(m.clients))
	}
	
	// 安全关闭客户端连接
	go m.safeCloseClient(client)
}

// broadcastToAll 广播消息给所有用户
func (m *Manager) broadcastToAll(message []byte) {
	m.mutex.RLock()
	userIDs := make([]uint, 0, len(m.clients))
	for userID := range m.clients {
		userIDs = append(userIDs, userID)
	}
	m.mutex.RUnlock()

	// 使用批量消息处理
	if len(userIDs) > 0 {
		batchMsg := &BatchMessage{
			UserIDs: userIDs,
			Message: message,
		}
		
		select {
		case m.messageBatch <- batchMsg:
			// 批量消息已加入队列
		default:
			// 批量消息队列满了，直接处理
			go m.flushBatchMessages([]*BatchMessage{batchMsg})
		}
	}
}

// SendToUser 发送消息给指定用户
func (m *Manager) SendToUser(userID uint, notification *model.Notification) error {
	// 生成消息ID
	messageID := fmt.Sprintf("%d_%d", userID, time.Now().UnixNano())
	
	message := NotificationMessage{
		Type:      "notification",
		Data:      notification,
		Timestamp: time.Now().Unix(),
		MessageID: messageID,
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化通知消息失败: %w", err)
	}

	m.mutex.RLock()
	client, exists := m.clients[userID]
	m.mutex.RUnlock()

	if !exists {
		// 用户不在线，存储到Redis离线消息队列
		return m.storeOfflineMessage(userID, notification)
	}

	// 检查客户端是否已关闭
	client.closeMutex.RLock()
	if client.closed {
		client.closeMutex.RUnlock()
		return m.storeOfflineMessage(userID, notification)
	}
	client.closeMutex.RUnlock()
	
	select {
	case client.Send <- data:
		// 更新发送统计
		m.statsMutex.Lock()
		m.stats.MessagesSent++
		m.statsMutex.Unlock()
		return nil
	default:
		m.logger.Warnf("用户 %d 消息发送失败，channel已满", userID)
		return m.storeOfflineMessage(userID, notification)
	}
}

// SendToUsers 批量发送消息给多个用户
func (m *Manager) SendToUsers(userIDs []uint, notification *model.Notification) error {
	message := NotificationMessage{
		Type:      "notification",
		Data:      notification,
		Timestamp: time.Now().Unix(),
		MessageID: fmt.Sprintf("batch_%d", time.Now().UnixNano()),
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化通知消息失败: %w", err)
	}

	// 分离在线和离线用户
	var onlineUsers, offlineUsers []uint
	
	m.mutex.RLock()
	for _, userID := range userIDs {
		if client, exists := m.clients[userID]; exists {
			client.closeMutex.RLock()
			if !client.closed {
				onlineUsers = append(onlineUsers, userID)
			} else {
				offlineUsers = append(offlineUsers, userID)
			}
			client.closeMutex.RUnlock()
		} else {
			offlineUsers = append(offlineUsers, userID)
		}
	}
	m.mutex.RUnlock()

	// 批量发送给在线用户
	if len(onlineUsers) > 0 {
		batchMsg := &BatchMessage{
			UserIDs: onlineUsers,
			Message: data,
		}
		
		select {
		case m.messageBatch <- batchMsg:
			// 批量消息已加入队列
		default:
			// 批量消息队列满了，直接处理
			go m.flushBatchMessages([]*BatchMessage{batchMsg})
		}
	}

	// 存储离线用户消息
	for _, userID := range offlineUsers {
		if err := m.storeOfflineMessage(userID, notification); err != nil {
			m.logger.Errorf("存储用户 %d 离线消息失败: %v", userID, err)
		}
	}

	return nil
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
	
	// 使用pipeline提升性能
	pipe := m.redis.Pipeline()
	pipe.LPush(m.ctx, key, data)
	pipe.Expire(m.ctx, key, 7*24*time.Hour)
	// 限制离线消息数量，避免内存过多占用
	pipe.LTrim(m.ctx, key, 0, 999) // 最多保存1000条离线消息
	
	if _, err := pipe.Exec(m.ctx); err != nil {
		return fmt.Errorf("存储离线消息失败: %w", err)
	}

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

	client.closeMutex.RLock()
	if client.closed {
		client.closeMutex.RUnlock()
		return
	}
	client.closeMutex.RUnlock()

	// 批量发送离线消息，限制发送速度
	sendCount := 0
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
			sendCount++
		default:
			// 发送失败，停止发送其他消息
			m.logger.Warnf("用户 %d 离线消息发送中断，已发送 %d/%d 条", userID, sendCount, len(messages))
			return
		}

		// 限制发送速度，避免过量消息
		if sendCount%10 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	// 清空离线消息
	m.redis.Del(m.ctx, key)
	m.logger.Infof("用户 %d 离线消息已发送，共 %d 条", userID, len(messages))
}

// cleanupInactiveClients 清理不活跃的客户端
func (m *Manager) cleanupInactiveClients() {
	m.mutex.Lock()
	
	cutoff := time.Now().Add(-5 * time.Minute) // 增加超时时间到5分钟
	var toRemove []*Client
	
	for userID, client := range m.clients {
		client.closeMutex.RLock()
		shouldRemove := client.closed || client.LastActive.Before(cutoff)
		client.closeMutex.RUnlock()
		
		if shouldRemove {
			toRemove = append(toRemove, client)
			delete(m.clients, userID)
		}
	}

	// 更新统计信息
	removeCount := len(toRemove)
	if removeCount > 0 {
		m.statsMutex.Lock()
		m.stats.ActiveConnections = len(m.clients)
		m.statsMutex.Unlock()
		
		m.logger.Infof("清理 %d 个不活跃连接", removeCount)
	}
	
	m.mutex.Unlock()
	
	// 锁外关闭客户端
	for _, client := range toRemove {
		go m.safeCloseClient(client) // 直接并发关闭每个客户端，不需要额外的WaitGroup
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

// GetStats 获取统计信息（优化版）
func (m *Manager) GetStats() map[string]interface{} {
	m.mutex.RLock()
	activeConnections := len(m.clients)
	m.mutex.RUnlock()
	
	m.statsMutex.RLock()
	stats := *m.stats
	m.statsMutex.RUnlock()
	
	stats.ActiveConnections = activeConnections

	return map[string]interface{}{
		"active_connections":  stats.ActiveConnections,
		"total_connections":   stats.TotalConnections,
		"messages_sent":       stats.MessagesSent,
		"messages_received":   stats.MessagesReceived,
		"connection_errors":   stats.ConnectionErrors,
		"last_restart":        stats.LastRestart,
		"uptime_seconds":     int64(time.Since(stats.LastRestart).Seconds()),
	}
}

// HandleWebSocket 处理WebSocket连接（优化版）
func (m *Manager) HandleWebSocket(c *gin.Context, userID uint) {
	// 记录连接尝试
	m.logger.Infof("用户 %d 尝试建立WebSocket连接，Origin: %s, User-Agent: %s", 
		userID, c.GetHeader("Origin"), c.GetHeader("User-Agent"))
	
	// 升级WebSocket连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		m.logger.Errorf("用户 %d WebSocket升级失败: %v", userID, err)
		// 更新错误统计
		m.statsMutex.Lock()
		m.stats.ConnectionErrors++
		m.statsMutex.Unlock()
		return
	}

	// 设置连接参数
	conn.SetReadLimit(4096)  // 增加读取限制
	conn.SetReadDeadline(time.Now().Add(90 * time.Second))   // 增加读取超时
	conn.SetWriteDeadline(time.Now().Add(15 * time.Second))  // 增加写入超时

	client := &Client{
		UserID:     userID,
		Conn:       conn,
		Send:       make(chan []byte, 512), // 增加channel缓冲区
		Manager:    m,
		LastActive: time.Now(),
		closed:     false,
	}

	// 设置连接状态回调
	conn.SetCloseHandler(func(code int, text string) error {
		m.logger.Infof("用户 %d WebSocket连接关闭: code=%d, text=%s, 连接ID: %s", 
			userID, code, text, client.ConnID)
		return nil
	})

	// 注册客户端
	select {
	case m.register <- client:
		m.logger.Debugf("用户 %d 客户端注册请求已发送", userID)
	default:
		m.logger.Errorf("注册队列已满，关闭用户 %d 的连接", userID)
		conn.Close()
		return
	}

	// 启动goroutine处理客户端
	go client.writePump()
	go client.readPump()
	
	m.logger.Infof("用户 %d WebSocket连接处理程序已启动", userID)
}

// readPump 处理读取消息
func (c *Client) readPump() {
	defer func() {
		// 确保连接被正确清理
		select {
		case c.Manager.unregister <- c:
		default:
			// 如果unregister channel满了，直接关闭
			go c.Manager.safeCloseClient(c)
		}
	}()

	// 设置读取参数
	c.Conn.SetReadLimit(4096)
	c.Conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	
	// 设置pong处理器
	c.Conn.SetPongHandler(func(string) error {
		c.closeMutex.RLock()
		closed := c.closed
		c.closeMutex.RUnlock()
		
		if !closed {
			c.closeMutex.Lock()
			c.LastActive = time.Now()
			c.Conn.SetReadDeadline(time.Now().Add(90 * time.Second))
			c.closeMutex.Unlock()
		}
		return nil
	})

	// 消息读取循环
	for {
		// 使用非阻塞方式检查context是否已取消
		select {
		case <-c.Manager.ctx.Done():
			return
		default:
		}
		
		c.closeMutex.RLock()
		closed := c.closed
		c.closeMutex.RUnlock()
		if closed {
			return
		}

		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, 
				websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				c.Manager.logger.Errorf("用户 %d WebSocket意外关闭: %v, 连接ID: %s", 
					c.UserID, err, c.ConnID)
			}
			return
		}
		
		// 更新活跃时间
		c.closeMutex.RLock()
		closed = c.closed
		c.closeMutex.RUnlock()
		
		if !closed {
			c.closeMutex.Lock()
			c.LastActive = time.Now()
			c.closeMutex.Unlock()
			
			// 更新接收统计
			c.Manager.statsMutex.Lock()
			c.Manager.stats.MessagesReceived++
			c.Manager.statsMutex.Unlock()
			
			// 处理客户端消息
			if len(message) > 0 {
				c.handleMessage(message)
			}
		}
	}
}

// handleMessage 处理客户端消息
func (c *Client) handleMessage(message []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		c.Manager.logger.Debugf("用户 %d 发送了无效JSON消息: %s", c.UserID, string(message))
		return
	}

	msgType, ok := msg["type"].(string)
	if !ok {
		return
	}

	// 只处理特定类型的消息
	if msgType == "ping" {
		// 处理ping消息
		response := NotificationMessage{
			Type:      "pong",
			Timestamp: time.Now().Unix(),
		}
		responseData, err := json.Marshal(response)
		if err != nil {
			return
		}
		
		c.closeMutex.RLock()
		closed := c.closed
		c.closeMutex.RUnlock()
		
		if !closed {
			select {
			case c.Send <- responseData:
			default:
				// Send channel满了，忽略此消息
			}
		}
	} else if msgType == "heartbeat" {
		// 处理心跳消息，仅更新活跃时间即可
		c.Manager.logger.Debugf("用户 %d 发送心跳消息", c.UserID)
	} else {
		// 处理其他类型消息
		c.Manager.logger.Debugf("用户 %d 发送未知类型消息: %s", c.UserID, msgType)
	}
}

// writePump 处理写入消息
func (c *Client) writePump() {
	// 心跳间隔调整为60秒
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Manager.ctx.Done():
			return
		case message, ok := <-c.Send:
			c.closeMutex.RLock()
			if c.closed {
				c.closeMutex.RUnlock()
				return
			}
			
			c.Conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
			if !ok {
				// Send channel已关闭
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				c.closeMutex.RUnlock()
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.Manager.logger.Errorf("用户 %d 写入消息失败: %v, 连接ID: %s", 
					c.UserID, err, c.ConnID)
				c.closeMutex.RUnlock()
				return
			}
			c.closeMutex.RUnlock()

		case <-ticker.C:
			c.closeMutex.RLock()
			if c.closed {
				c.closeMutex.RUnlock()
				return
			}
			
			// 发送ping消息
			c.Conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.Manager.logger.Debugf("用户 %d ping发送失败: %v, 连接ID: %s", 
					c.UserID, err, c.ConnID)
				c.closeMutex.RUnlock()
				return
			}
			c.closeMutex.RUnlock()
		}
	}
}

// safeCloseClient 安全关闭客户端连接
func (m *Manager) safeCloseClient(client *Client) {
	client.closeMutex.Lock()
	defer client.closeMutex.Unlock()

	if client.closed {
		return // 已经关闭过了
	}
	
	client.closed = true
	
	// 直接关闭Send channel，通知writePump退出
	close(client.Send)
	
	// 关闭WebSocket连接
	if err := client.Conn.Close(); err != nil {
		m.logger.Debugf("关闭WebSocket连接时出错: %v, 用户: %d, 连接ID: %s", 
			err, client.UserID, client.ConnID)
	}
	
	m.logger.Debugf("用户 %d 客户端连接已安全关闭，连接ID: %s, 连接时长: %v", 
		client.UserID, client.ConnID, time.Since(client.ConnTime))
}