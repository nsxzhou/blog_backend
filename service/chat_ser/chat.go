package chat_ser

import (
	"sync"
	"time"

	"blog/global"
	"blog/models"

	"go.uber.org/zap"
)

// ChatRoom 聊天室实现
type ChatRoom struct {
	// 注册客户端的通道
	Register chan *models.Client

	// 注销客户端的通道
	Unregister chan *models.Client

	// 广播消息的通道
	Broadcast chan *models.ChatMessage

	// 客户端映射表
	clients map[uint64]*models.Client

	// 消息历史记录
	messageHistory []*models.ChatMessage

	// 互斥锁，保护共享资源
	mutex sync.RWMutex
}

// NewChatRoom 创建新的聊天室
func NewChatRoom() *ChatRoom {
	return &ChatRoom{
		Register:       make(chan *models.Client),
		Unregister:     make(chan *models.Client),
		Broadcast:      make(chan *models.ChatMessage),
		clients:        make(map[uint64]*models.Client),
		messageHistory: make([]*models.ChatMessage, 0, 1000),
		mutex:          sync.RWMutex{},
	}
}

// Run 启动聊天室后台处理
func (cr *ChatRoom) Run() {
	for {
		select {
		case client := <-cr.Register:
			global.Log.Info("客户端已连接", zap.Any("client", client))
			// 注册新客户端
			cr.mutex.Lock()
			cr.clients[client.UserID] = client
			cr.mutex.Unlock()

			// 向新客户发送已送达回执
			go cr.sendDeliveredReceipts(client)

			// 广播在线用户列表更新
			go cr.broadcastUserList()

		case client := <-cr.Unregister:
			global.Log.Info("客户端已断开连接", zap.Any("client", client))
			// 注销客户端
			cr.mutex.Lock()
			if _, ok := cr.clients[client.UserID]; ok {
				delete(cr.clients, client.UserID)
				close(client.Send)
			}
			cr.mutex.Unlock()

			// 广播在线用户列表更新
			go cr.broadcastUserList()

		case message := <-cr.Broadcast:
			// 处理不同类型的消息
			switch message.Type {
			case models.MessageTypeMessage:
				// 存储消息到历史记录
				cr.storeMessageToHistory(message)

				// 设置消息初始状态
				message.Status = models.MessageStatusSent

				// 广播消息给所有客户端
				cr.broadcastMessage(message)

			case models.MessageTypeJoin, models.MessageTypeLeave:
				// 广播加入/离开消息并更新用户列表
				cr.broadcastMessage(message)
				go cr.broadcastUserList()
			default:
				// 其他类型消息直接广播
				cr.broadcastMessage(message)
			}
		}
	}
}

// broadcastMessage 广播消息给所有客户端
func (cr *ChatRoom) broadcastMessage(message *models.ChatMessage) {
	cr.mutex.RLock()

	// 创建需要删除的客户端列表，避免在遍历时修改map
	var clientsToRemove []uint64

	for userID, client := range cr.clients {
		select {
		case client.Send <- message:
			// 消息发送成功
			if message.Type == models.MessageTypeMessage &&
				message.UserID != client.UserID {
				// 为其他用户的消息自动更新为已送达状态
				go cr.sendDeliveredStatus(message.ID, client.UserID)
			}
		default:
			// 发送通道已满或关闭，标记客户端待删除
			clientsToRemove = append(clientsToRemove, userID)
		}
	}

	cr.mutex.RUnlock()

	// 如果有需要删除的客户端，加写锁后删除
	if len(clientsToRemove) > 0 {
		cr.mutex.Lock()
		for _, userID := range clientsToRemove {
			if client, ok := cr.clients[userID]; ok {
				close(client.Send)
				delete(cr.clients, userID)
			}
		}
		cr.mutex.Unlock()
	}
}

// storeMessageToHistory 存储消息到历史记录
func (cr *ChatRoom) storeMessageToHistory(message *models.ChatMessage) {
	// 只存储聊天消息
	if message.Type != models.MessageTypeMessage {
		return
	}

	cr.mutex.Lock()
	defer cr.mutex.Unlock()

	// 限制历史记录长度
	if len(cr.messageHistory) >= 1000 {
		cr.messageHistory = cr.messageHistory[1:]
	}

	// 为消息分配ID
	if message.ID == 0 {
		message.ID = uint64(time.Now().UnixNano())
	}

	// 添加到历史记录
	cr.messageHistory = append(cr.messageHistory, message)

	// TODO: 如果需要，也可以将消息保存到数据库
}

// sendDeliveredReceipts 向新客户端发送所有未读消息的已送达回执
func (cr *ChatRoom) sendDeliveredReceipts(client *models.Client) {
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()

	// 遍历历史消息，发送已送达回执
	for _, msg := range cr.messageHistory {
		// 如果消息不是自己发的，且状态不是已读
		if msg.UserID != client.UserID && msg.Status != models.MessageStatusRead {
			// 发送已送达回执给消息发送者
			receiptMsg := &models.ChatMessage{
				Type:      models.MessageTypeReceipt,
				MessageID: msg.ID,
				Status:    models.MessageStatusDelivered,
				UserID:    client.UserID,
			}

			if sender, ok := cr.clients[msg.UserID]; ok {
				select {
				case sender.Send <- receiptMsg:
					// 回执发送成功
				default:
					// 发送失败，忽略
				}
			}
		}
	}
}

// sendDeliveredStatus 发送已送达状态
func (cr *ChatRoom) sendDeliveredStatus(messageID uint64, userID uint64) {
	// 更新消息状态
	cr.UpdateMessageStatus(messageID, models.MessageStatusDelivered)

	// 创建回执消息
	receiptMsg := &models.ChatMessage{
		Type:      models.MessageTypeReceipt,
		MessageID: messageID,
		Status:    models.MessageStatusDelivered,
		UserID:    userID,
	}

	// 找到消息发送者
	msg, err := cr.GetMessageByID(messageID)
	if err != nil {
		return
	}

	// 发送回执给原消息发送者
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()

	if sender, ok := cr.clients[msg.UserID]; ok {
		select {
		case sender.Send <- receiptMsg:
			// 回执发送成功
		default:
			// 发送失败，忽略
		}
	}
}

// broadcastUserList 广播在线用户列表
func (cr *ChatRoom) broadcastUserList() {
	users := cr.GetOnlineUsers()

	// 创建用户列表消息
	message := &models.ChatMessage{
		Type:  models.MessageTypeUsers,
		Users: users,
	}

	// 广播给所有客户端
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()

	for _, client := range cr.clients {
		select {
		case client.Send <- message:
			// 发送成功
		default:
			// 发送失败，忽略
		}
	}
}

// GetClient 获取指定用户ID的客户端
func (cr *ChatRoom) GetClient(userID uint64) *models.Client {
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()

	if client, ok := cr.clients[userID]; ok {
		return client
	}
	return nil
}

// GetOnlineUsers 获取在线用户列表
func (cr *ChatRoom) GetOnlineUsers() []*models.User {
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()

	users := make([]*models.User, 0, len(cr.clients))

	for _, client := range cr.clients {
		users = append(users, &models.User{
			ID:     client.UserID,
			Name:   client.Username,
			Online: true,
		})
	}

	return users
}

// StoreMessage 存储消息到数据库
func (cr *ChatRoom) StoreMessage(msg *models.ChatMessage) error {
	return nil
}

// GetMessageByID 根据ID获取消息
func (cr *ChatRoom) GetMessageByID(id uint64) (*models.ChatMessage, error) {
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()

	for _, msg := range cr.messageHistory {
		if msg.ID == id {
			return msg, nil
		}
	}

	return nil, nil
}

// GetMessageHistory 获取历史消息
func (cr *ChatRoom) GetMessageHistory(limit int) ([]*models.ChatMessage, error) {
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()

	// 确保不超出历史记录长度
	if limit > len(cr.messageHistory) {
		limit = len(cr.messageHistory)
	}

	// 获取最近的消息
	startIdx := len(cr.messageHistory) - limit
	if startIdx < 0 {
		startIdx = 0
	}

	// 复制消息切片
	messages := make([]*models.ChatMessage, limit)
	copy(messages, cr.messageHistory[startIdx:])

	return messages, nil
}

// UpdateMessageStatus 更新消息状态
func (cr *ChatRoom) UpdateMessageStatus(id uint64, status string) error {
	cr.mutex.Lock()
	defer cr.mutex.Unlock()

	for i, msg := range cr.messageHistory {
		if msg.ID == id {
			cr.messageHistory[i].Status = status
			break
		}
	}

	return nil
}
