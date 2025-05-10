package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"im-go/internal/config"
	"im-go/internal/models"
	"im-go/internal/storage"

	// ws "im-go/internal/websocket" //不再直接导入websocket包来获取Message类型
	"im-go/internal/imtypes" // 导入新的imtypes包

	"gorm.io/gorm"

	appKafka "im-go/internal/kafka" // Renamed alias for clarity

	confluentKafka "github.com/confluentinc/confluent-kafka-go/v2/kafka" // New import
)

// RawMessageInput 类型现在从 imtypes 包获取
// type RawMessageInput struct { ... } // 定义已移除

// MessageService 定义了消息相关服务的接口。
type MessageService interface {
	// SendMessage 处理从客户端收到的原始消息输入，负责初步处理、发送到 Kafka
	SendMessage(ctx context.Context, input imtypes.RawMessageInput) error // 使用 imtypes.RawMessageInput

	// ProcessKafkaMessage 作为 Kafka 消费者回调，处理从 Kafka 接收到的消息
	// 负责持久化消息、更新会话、可能的分发逻辑（如果 Hub 在此服务中管理）
	ProcessKafkaMessage(ctx context.Context, kafkaMsg *confluentKafka.Message) error

	GetMessagesForConversation(ctx context.Context, conversationID uint, limit int, offset int) ([]*models.Message, error)
	// MarkMessagesAsRead(ctx context.Context, userID uint, conversationID uint, messageIDs []uint) error
	GetMessageByID(ctx context.Context, messageID uint) (*models.Message, error)
}

// messageService 是 MessageService 的实现。
type messageService struct {
	msgRepo   storage.MessageRepository
	convoRepo storage.ConversationRepository
	producer  appKafka.MessageProducer
	cfg       config.Config
	// hub      *ws.Hub // 如果需要直接与 Hub 交互以分发消息
}

// NewMessageService 创建一个新的 MessageService 实例。
func NewMessageService(msgRepo storage.MessageRepository, convoRepo storage.ConversationRepository, producer appKafka.MessageProducer, cfg config.Config /*, hub *ws.Hub*/) MessageService {
	return &messageService{
		msgRepo:   msgRepo,
		convoRepo: convoRepo,
		producer:  producer,
		cfg:       cfg,
		// hub: hub,
	}
}

// SendMessage 处理用户发送的新消息，将其发送到 Kafka。
func (s *messageService) SendMessage(ctx context.Context, input imtypes.RawMessageInput) error {
	if input.SenderID == "" || input.ReceiverID == "" {
		return fmt.Errorf("发送者ID或接收者ID不能为空")
	}

	// RawMessageInput 已经包含了需要发送到 Kafka 的核心信息
	// 将 RawMessageInput 序列化为 []byte 以便发送到 Kafka
	msgBytes, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("序列化消息输入失败: %w", err)
	}

	topic := s.cfg.Kafka.MessagesTopic
	// 根据 input.Type 进一步确定 topic 的逻辑可以保留
	// if input.Type == string(ws.FileMessageType) || input.Type == string(ws.ImageMessageType) { ... }

	err = s.producer.SendMessage(ctx, topic, []byte(input.SenderID), msgBytes) // 可以用 SenderID 作为 Key
	if err != nil {
		return fmt.Errorf("发送消息到 Kafka 失败: %w", err)
	}
	return nil
}

// ProcessKafkaMessage 处理从 Kafka 消费到的消息。
func (s *messageService) ProcessKafkaMessage(ctx context.Context, kafkaMsg *confluentKafka.Message) error {
	var receivedInput imtypes.RawMessageInput // 使用 imtypes.RawMessageInput
	if err := json.Unmarshal(kafkaMsg.Value, &receivedInput); err != nil {
		return fmt.Errorf("从 Kafka 反序列化消息输入失败: %w, 原始消息: %s", err, string(kafkaMsg.Value))
	}

	senderIDUint, err := storage.StrToUint(receivedInput.SenderID)
	if err != nil {
		return fmt.Errorf("转换发送者ID '%s' 失败: %w", receivedInput.SenderID, err)
	}
	receiverIDUint, err := storage.StrToUint(receivedInput.ReceiverID)
	if err != nil {
		return fmt.Errorf("转换接收者ID '%s' 失败: %w", receivedInput.ReceiverID, err)
	}

	var conversationID uint
	var conversation *models.Conversation
	conversation, err = s.convoRepo.FindPrivateConversationByUsers(ctx, senderIDUint, receiverIDUint)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查找私聊会话失败: %w", err)
	}

	if conversation == nil {
		newConversation := &models.Conversation{Type: models.PrivateConversation}
		if err := s.convoRepo.CreateConversation(ctx, newConversation); err != nil {
			return fmt.Errorf("创建新会话失败: %w", err)
		}
		conversationID = newConversation.ID
		conversation = newConversation
		p1 := &models.ConversationParticipant{ConversationID: conversationID, UserID: senderIDUint, JoinedAt: time.Now()}
		p2 := &models.ConversationParticipant{ConversationID: conversationID, UserID: receiverIDUint, JoinedAt: time.Now()}
		if err := s.convoRepo.AddParticipant(ctx, p1); err != nil {
			return fmt.Errorf("添加参与者1到会话 %d 失败: %w", conversationID, err)
		}
		if err := s.convoRepo.AddParticipant(ctx, p2); err != nil {
			return fmt.Errorf("添加参与者2到会话 %d 失败: %w", conversationID, err)
		}
	} else {
		conversationID = conversation.ID
	}

	dbMessage := &models.Message{
		ConversationID: conversationID,
		SenderID:       senderIDUint,
		Type:           models.MessageTypeDB(receivedInput.Type),
		Content:        string(receivedInput.Content),
		SentAt:         receivedInput.Timestamp,
	}

	if receivedInput.Type == string(models.FileMessageTypeDB) || receivedInput.Type == string(models.ImageMessageTypeDB) {
		metadata := map[string]interface{}{
			"fileName": receivedInput.FileName,
			"fileSize": receivedInput.FileSize,
		}
		metadataBytes, _ := json.Marshal(metadata)
		dbMessage.MetadataRaw = metadataBytes
	}

	if err := s.msgRepo.Create(ctx, dbMessage); err != nil {
		return fmt.Errorf("存储消息到数据库失败: %w", err)
	}

	conversation.LastMessageID = &dbMessage.ID
	if err := s.convoRepo.UpdateConversation(ctx, conversation); err != nil {
		return fmt.Errorf("更新会话 %d 的 LastMessageID 失败: %w", conversationID, err)
	}

	// --- 新增：将消息推送到 WebSocketOutgoingTopic ---
	// 构建发送给客户端的 websocket.Message
	// 注意：这里的 Type 需要是 websocket.MessageType
	outgoingWsMsg := &imtypes.Message{ // 使用 imtypes.Message
		ID:             dbMessage.IDString(),
		Type:           imtypes.MessageType(dbMessage.Type), // 从 models.MessageTypeDB 转为 imtypes.MessageType
		Content:        dbMessage.Content,
		SenderID:       strconv.FormatUint(uint64(dbMessage.SenderID), 10),
		ReceiverID:     receivedInput.ReceiverID,
		Timestamp:      dbMessage.SentAt,
		ConversationID: strconv.FormatUint(uint64(dbMessage.ConversationID), 10),
	}
	if dbMessage.Type == models.FileMessageTypeDB || dbMessage.Type == models.ImageMessageTypeDB {
		fileMeta, _ := dbMessage.GetFileMetadata() // 假设GetFileMetadata可以处理两种类型或有相应方法
		if fileMeta != nil {
			outgoingWsMsg.FileName = fileMeta.FileName
			outgoingWsMsg.FileSize = fileMeta.FileSize
			// outgoingWsMsg.Content 应该包含 URL，这部分逻辑可能需要在文件上传完成后补充
		}
	}

	outgoingMsgBytes, err := json.Marshal(outgoingWsMsg)
	if err != nil {
		log.Printf("序列化出站 WebSocket 消息失败: %v", err)
		return nil // 或者只记录错误，不中断主流程
	}

	// 发送到 WebSocketOutgoingTopic。消息的 Key 可以是 ReceiverID (私聊) 或 ConversationID (群聊)
	// 以便消费者可以根据 Key 做一些路由或分区优化（如果 Kafka 分区策略基于 Key）。
	var outgoingKey []byte
	// TODO: 确定是私聊还是群聊，并设置合适的 Key
	// if isGroupMessage(receivedInput.Type) { // 假设有这样的判断函数
	// 	outgoingKey = []byte(strconv.FormatUint(uint64(conversationID), 10))
	// } else {
	outgoingKey = []byte(receivedInput.ReceiverID)
	// }

	if err := s.producer.SendMessage(ctx, s.cfg.Kafka.WebSocketOutgoingTopic, outgoingKey, outgoingMsgBytes); err != nil {
		log.Printf("发送消息到 WebSocketOutgoingTopic 失败: %v", err)
		// 这个错误也可能不应该中断主流程，而是记录下来
	}
	return nil
}

// GetMessagesForConversation 获取指定会话的消息列表。
func (s *messageService) GetMessagesForConversation(ctx context.Context, conversationID uint, limit int, offset int) ([]*models.Message, error) {
	return s.msgRepo.GetByConversationID(ctx, conversationID, limit, offset)
}

// GetMessageByID retrieves a single message by its ID.
func (s *messageService) GetMessageByID(ctx context.Context, messageID uint) (*models.Message, error) {
	if s.msgRepo == nil {
		return nil, errors.New("messageRepository is not initialized in messageService")
	}
	return s.msgRepo.GetByID(ctx, messageID) // Changed FindMessageByID to FindByID
}
