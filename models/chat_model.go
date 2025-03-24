package models

import (
	"blog/global"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// 消息类型常量
const (
	MessageTypeMessage = "message" // 消息
	MessageTypeJoin    = "join"    // 加入
	MessageTypeLeave   = "leave"   // 离开
	MessageTypeUsers   = "users"   // 用户列表
	MessageTypeReceipt = "receipt" // 已送达回执
	MessageTypeError   = "error"   // 错误
	MessageTypeHistory = "history" // 历史消息
	MessageTypePing    = "ping"    // 客户端心跳
	MessageTypePong    = "pong"    // 服务端回应
	MessageTypeTyping  = "typing"  // 正在输入
)

// 消息状态常量
const (
	MessageStatusSent      = "sent"      // 已发送
	MessageStatusDelivered = "delivered" // 已送达
	MessageStatusRead      = "read"      // 已读
	MessageStatusError     = "error"     // 错误
)

// ChatMessage 聊天消息结构
type ChatMessage struct {
	ID        uint64         `json:"id,omitempty" gorm:"primaryKey;autoIncrement"`
	Type      string         `json:"type"`                 // message, join, leave, typing, users, receipt, error
	MessageID uint64         `json:"message_id,omitempty"` // 用于消息回执
	UserID    uint64         `json:"user_id,omitempty"`    // 发送者ID
	Username  string         `json:"username,omitempty"`   // 发送者用户名
	Content   string         `json:"content,omitempty"`    // 消息内容
	Status    string         `json:"status,omitempty"`     // sent, delivered, read, error
	CreatedAt time.Time      `json:"created_at,omitempty"` // 消息创建时间
	Limit     int            `json:"limit,omitempty"`      // 历史消息请求的数量限制
	Messages  []*ChatMessage `json:"messages,omitempty"`   // 历史消息列表
	Users     []*User        `json:"users,omitempty"`      // 在线用户列表
}

// ChatRoom 聊天室接口
type ChatRoom interface {
	GetClient(userID uint64) *Client
	GetMessageByID(id uint64) (*ChatMessage, error)
	StoreMessage(msg *ChatMessage) error
	UpdateMessageStatus(id uint64, status string) error
	GetMessageHistory(limit int) ([]*ChatMessage, error)
	GetOnlineUsers() []*User
}

// Client 客户端连接
type Client struct {
	ID                uint64            // 客户端ID
	UserID            uint64            // 用户ID
	Username          string            // 用户名
	Conn              *websocket.Conn   // WebSocket连接
	Send              chan *ChatMessage // 发送消息的通道
	Room              ChatRoom          // 聊天室接口
	JoinedAt          time.Time         // 加入时间
	LastActive        time.Time         // 最后活跃时间
	ReconnectAttempts int               // 重连尝试次数
}

// User 用户信息结构
type User struct {
	ID     uint64 `json:"id"`
	Name   string `json:"name"`
	Online bool   `json:"online"`
}

const (
	// 向客户端写入消息的超时时间
	WriteWait = 10 * time.Second

	// 时间间隔，发送ping消息保持连接活跃
	PingPeriod = 50 * time.Second

	// 最长不活跃时间
	PongWait = 60 * time.Second

	// 消息最大长度
	MaxMessageSize = 4096

	// 客户端重连尝试次数
	MaxReconnectAttempts = 5

	// 重连间隔时间
	ReconnectInterval = 3 * time.Second

	// 心跳检查间隔
	HeartbeatInterval = 30 * time.Second

	// 非活跃超时（连接可能断开但未正确关闭）
	InactiveTimeout = 2 * PongWait
)

// WritePump 将消息从应用程序写入WebSocket连接
func (c *Client) WritePump() {
	ticker := time.NewTicker(PingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if !ok {
				// 通道已关闭
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 写入消息
			err := c.Conn.WriteJSON(message)
			if err != nil {
				global.Log.Error("写入消息失败", zap.Error(err),
					zap.Uint64("clientID", c.ID),
					zap.Uint64("userID", c.UserID))
				return
			}

			// 在消息写入成功后记录日志
			global.Log.Debug("消息已发送到客户端",
				zap.String("type", message.Type),
				zap.Uint64("clientID", c.ID),
				zap.Uint64("userID", c.UserID))

			// 处理消息回执
			if message.Type == MessageTypeMessage && message.UserID != c.UserID {
				// 为收到的消息发送已送达回执
				if message.Status != MessageStatusDelivered &&
					message.Status != MessageStatusRead {
					// 向发送者发送已送达回执
					c.sendDeliveredReceipt(message.ID)
				}
			}

		case <-ticker.C:
			// 发送Ping消息保持连接活跃
			c.Conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// 发送已送达回执
func (c *Client) sendDeliveredReceipt(messageID uint64) {
	if messageID == 0 {
		return
	}

	// 创建回执消息
	receipt := &ChatMessage{
		Type:      MessageTypeReceipt,
		MessageID: messageID,
		Status:    MessageStatusDelivered,
		UserID:    c.UserID,
	}

	// 获取消息发送者
	msg, err := c.Room.GetMessageByID(messageID)
	if err != nil || msg == nil {
		return
	}

	// 获取发送者客户端并发送回执
	if sender := c.Room.GetClient(msg.UserID); sender != nil {
		select {
		case sender.Send <- receipt:
			// 回执发送成功
		default:
			// 发送失败，忽略
		}
	}
}
