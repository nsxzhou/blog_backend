package chat

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"blog/global"
	"blog/models"
	"blog/service/chat_ser"
	"blog/utils"
)

// 确保聊天室只被初始化一次
var (
	chatRoom *chat_ser.ChatRoom
	roomOnce sync.Once
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// 生产环境应验证来源
		origin := r.Header.Get("Origin")
		global.Log.Info("WebSocket连接请求", zap.String("origin", origin))
		return true
	},
	HandshakeTimeout: 15 * time.Second, // 增加握手超时
	ReadBufferSize:   2048,             // 增加缓冲区大小
	WriteBufferSize:  2048,
}

// HandleWebSocket 处理WebSocket连接
func (c *Chat) HandleWebSocket(ctx *gin.Context) {
	// 记录详细连接信息
	global.Log.Info("新的WebSocket连接请求",
		zap.String("remote_addr", ctx.Request.RemoteAddr),
		zap.String("user_agent", ctx.Request.UserAgent()))

	// 确保聊天室只初始化一次
	roomOnce.Do(func() {
		chatRoom = chat_ser.NewChatRoom()
		go chatRoom.Run()

		// 启动后台清理任务
		go chatRoom.StartHeartbeatCheck()
	})

	_claims, exists := ctx.Get("claims")
	if !exists {
		global.Log.Error("无法获取用户身份信息")
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	claims := _claims.(*utils.CustomClaims)
	global.Log.Info("WebSocket用户已认证", zap.Uint64("user_id", uint64(claims.UserID)))

	// 升级HTTP连接为WebSocket
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		global.Log.Error("WebSocket升级失败",
			zap.Error(err),
			zap.String("remote_addr", ctx.Request.RemoteAddr))
		return
	}

	global.Log.Info("WebSocket连接已建立",
		zap.String("remote_addr", ctx.Request.RemoteAddr),
		zap.Uint64("user_id", uint64(claims.UserID)))

	// 设置连接属性
	conn.SetReadLimit(1024 * 1024) // 1MB最大消息大小
	conn.SetReadDeadline(time.Now().Add(models.PongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(models.PongWait))
		return nil
	})

	// 获取用户信息
	user, err := models.GetUserByID(claims.UserID)
	if err != nil {
		global.Log.Error("获取用户信息失败", zap.Error(err))
		conn.Close()
		return
	}

	// 生成客户端ID
	id, err := utils.GenerateID()
	if err != nil {
		global.Log.Error("生成ID失败", zap.Error(err))
		conn.Close()
		return
	}

	// 检查用户是否已连接
	if existingClient := chatRoom.GetClient(uint64(user.ID)); existingClient != nil {
		// 关闭旧连接
		global.Log.Warn("用户已有连接，关闭旧连接",
			zap.Uint64("user_id", uint64(user.ID)))
		close(existingClient.Send)

		// 等待旧连接关闭
		time.Sleep(100 * time.Millisecond)
	}

	// 创建客户端
	client := &models.Client{
		ID:         uint64(id),
		UserID:     uint64(user.ID),
		Username:   user.Nickname,
		Conn:       conn,
		Send:       make(chan *models.ChatMessage, 256),
		Room:       chatRoom,
		JoinedAt:   time.Now(),
		LastActive: time.Now(),
	}

	// 先启动写入协程
	go client.WritePump()

	// 注册客户端
	joinMsg := &models.ChatMessage{
		Type:      models.MessageTypeJoin,
		UserID:    client.UserID,
		Content:   "加入了聊天室",
		CreatedAt: time.Now(),
	}

	chatRoom.Register <- client
	chatRoom.Broadcast <- joinMsg

	// 启动读取协程
	go c.handleMessages(client)
}

// handleMessages 处理客户端发来的消息
func (c *Chat) handleMessages(client *models.Client) {
	defer func() {
		if r := recover(); r != nil {
			global.Log.Error("处理消息时发生panic", zap.Any("error", r))
		}

		// 广播用户离开消息
		leaveMsg := &models.ChatMessage{
			Type:      models.MessageTypeLeave,
			UserID:    client.UserID,
			Content:   "离开了聊天室",
			CreatedAt: time.Now(),
		}
		chatRoom.Broadcast <- leaveMsg

		// 注销客户端
		chatRoom.Unregister <- client
		client.Conn.Close()
	}()

	// 设置连接参数
	client.Conn.SetReadLimit(models.MaxMessageSize)
	client.Conn.SetReadDeadline(time.Now().Add(models.PongWait))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(models.PongWait))
		return nil
	})

	for {
		var message models.ChatMessage
		err := client.Conn.ReadJSON(&message)
		global.Log.Info("收到消息", zap.Any("message", message))

		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure) {
				global.Log.Error("websocket unexpected close", zap.Error(err))
			} else {
				global.Log.Error("读取消息错误", zap.Error(err))
			}
			break
		}

		// 根据消息类型处理
		switch message.Type {
		case models.MessageTypeMessage:
			global.Log.Info("收到消息", zap.Any("message", message))
			// 处理普通聊天消息
			message.UserID = client.UserID
			message.Username = client.Username
			message.CreatedAt = time.Now()

			// 保存消息到数据库
			if err := chatRoom.StoreMessage(&message); err != nil {
				global.Log.Error("保存消息失败", zap.Error(err))
			}

			// 广播消息
			chatRoom.Broadcast <- &message
		case models.MessageTypeReceipt:
			global.Log.Info("收到消息回执", zap.Any("message", message))
			// 处理消息回执
			if message.MessageID > 0 {
				chatRoom.UpdateMessageStatus(message.MessageID, message.Status)

				// 通知消息发送者
				if msg, err := chatRoom.GetMessageByID(message.MessageID); err == nil {
					receiptMsg := &models.ChatMessage{
						Type:      models.MessageTypeReceipt,
						MessageID: message.MessageID,
						Status:    message.Status,
						UserID:    client.UserID,
					}

					if targetClient := chatRoom.GetClient(msg.UserID); targetClient != nil {
						targetClient.Send <- receiptMsg
					}
				}
			}
		case models.MessageTypeHistory:
			global.Log.Info("收到历史消息", zap.Any("message", message))
			// 获取历史消息
			limit := 50
			if message.Limit > 0 && message.Limit <= 100 {
				limit = message.Limit
			}

			history, err := chatRoom.GetMessageHistory(limit)
			if err != nil {
				global.Log.Error("获取历史消息失败", zap.Error(err))
				errorMsg := &models.ChatMessage{
					Type:    models.MessageTypeError,
					Content: "获取历史消息失败",
				}
				client.Send <- errorMsg
				continue
			}
			global.Log.Info("发送历史消息", zap.Any("history", history))
			// 发送历史消息
			historyMsg := &models.ChatMessage{
				Type:     models.MessageTypeHistory,
				Messages: history,
			}
			client.Send <- historyMsg
		case models.MessageTypeUsers:
			global.Log.Info("收到用户列表", zap.Any("message", message))
			// 获取在线用户列表
			users := chatRoom.GetOnlineUsers()
			global.Log.Info("在线用户列表", zap.Any("users", users))
			usersMsg := &models.ChatMessage{
				Type:  models.MessageTypeUsers,
				Users: users,
			}
			client.Send <- usersMsg
		}
	}
}
