package websocket

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client 表示一个WebSocket客户端连接
type Client struct {
	ID         string          // 连接唯一标识符
	UserID     uint            // 用户ID
	Conn       *websocket.Conn // WebSocket连接
	Send       chan []byte     // 发送消息的通道
	manager    *Manager        // 所属的管理器
	lastActive time.Time       // 最后活跃时间
	closed     bool            // 连接是否已关闭
	closeMutex sync.RWMutex    // 关闭状态的互斥锁
}

// NewClient 创建新的客户端实例
func NewClient(userID uint, conn *websocket.Conn, manager *Manager) *Client {
	return &Client{
		ID:         generateConnID(userID),
		UserID:     userID,
		Conn:       conn,
		Send:       make(chan []byte, 256),
		manager:    manager,
		lastActive: time.Now(),
	}
}

// readPump 处理从客户端读取消息
func (c *Client) readPump() {
	defer func() {
		c.manager.unregister <- c
	}()

	c.Conn.SetReadLimit(4096)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.updateActivity()
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		c.updateActivity()
		if len(message) > 0 {
			c.handleMessage(message)
		}
	}
}

// writePump 处理向客户端发送消息
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				return
			}
			if err := c.writeMessage(message); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage 处理接收到的消息
func (c *Client) handleMessage(message []byte) {
	var msg struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(message, &msg); err != nil {
		return
	}

	switch msg.Type {
	case "ping":
		c.handlePing()
	}
}

// handlePing 处理ping消息
func (c *Client) handlePing() {
	response := struct {
		Type      string `json:"type"`
		Timestamp int64  `json:"timestamp"`
	}{
		Type:      "pong",
		Timestamp: time.Now().Unix(),
	}

	if data, err := json.Marshal(response); err == nil {
		c.Send <- data
	}
}

// writeMessage 发送消息到客户端
func (c *Client) writeMessage(message []byte) error {
	c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return c.Conn.WriteMessage(websocket.TextMessage, message)
}

// Close 关闭客户端连接
func (c *Client) Close() {
	c.closeMutex.Lock()
	defer c.closeMutex.Unlock()

	if !c.closed {
		c.closed = true
		close(c.Send)
		c.Conn.Close()
	}
}

// updateActivity 更新最后活跃时间
func (c *Client) updateActivity() {
	c.closeMutex.Lock()
	c.lastActive = time.Now()
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.closeMutex.Unlock()
}

// IsActive 检查客户端是否活跃
func (c *Client) IsActive(timeout time.Duration) bool {
	c.closeMutex.RLock()
	defer c.closeMutex.RUnlock()
	return !c.closed && time.Since(c.lastActive) < timeout
}

// generateConnID 生成连接ID
func generateConnID(userID uint) string {
	return fmt.Sprintf("%d_%d", userID, time.Now().UnixNano())
}
