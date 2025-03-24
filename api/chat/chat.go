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
		return true // 在生产环境中应该限制来源
	},
	HandshakeTimeout: 10 * time.Second,
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
}

// HandleWebSocket 处理WebSocket连接
func (c *Chat) HandleWebSocket(ctx *gin.Context) {
	// 确保聊天室只初始化一次
	roomOnce.Do(func() {
		chatRoom = chat_ser.NewChatRoom()
		go chatRoom.Run()
	})

	_claims, exists := ctx.Get("claims")
	if !exists {
		global.Log.Error("无法获取用户身份信息")
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	claims := _claims.(*utils.CustomClaims)

	// 升级HTTP连接为WebSocket连接
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		global.Log.Error("websocket upgrade error", zap.Error(err))
		return
	}

	user, err := models.GetUserByID(claims.UserID)
	if err != nil {
		global.Log.Error("获取用户信息失败", zap.Error(err))
		return
	}

	id, err := utils.GenerateID()
	if err != nil {
		global.Log.Error("生成ID失败", zap.Error(err))
		return
	}

	// 创建客户端
	client := &models.Client{
		ID:       uint64(id),
		UserID:   uint64(user.ID),
		Username: user.Nickname,
		Conn:     conn,
		Send:     make(chan *models.ChatMessage, 256),
		Room:     chatRoom,
		JoinedAt: time.Now(),
	}

	// 广播用户加入消息
	joinMsg := &models.ChatMessage{
		Type:      models.MessageTypeJoin,
		UserID:    client.UserID,
		Content:   "加入了聊天室",
		CreatedAt: time.Now(),
	}

	// 注册新客户端并广播加入消息
	chatRoom.Register <- client
	chatRoom.Broadcast <- joinMsg

	// 启动客户端消息处理
	go client.WritePump()
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

	for {
		var message models.ChatMessage
		err := client.Conn.ReadJSON(&message)
		global.Log.Info("收到消息", zap.Any("message", message))

		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure) {
				global.Log.Error("websocket unexpected close", zap.Error(err))
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
