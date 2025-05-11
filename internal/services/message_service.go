package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"

	"im-go/internal/config"
	"im-go/internal/models"
	"im-go/internal/storage"

	// ws "im-go/internal/websocket" //不再直接导入websocket包来获取Message类型
	"im-go/internal/imtypes" // 导入新的imtypes包

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

	fmt.Printf("[ProcessKafkaMessage] 开始处理消息: Type=%s, SenderID=%s, ReceiverID=%s, Content=%s, ConversationID=%s\n",
		receivedInput.Type, receivedInput.SenderID, receivedInput.ReceiverID, string(receivedInput.Content), receivedInput.ConversationID)

	senderIDUint, err := storage.StrToUint(receivedInput.SenderID)
	if err != nil {
		return fmt.Errorf("转换发送者ID '%s' 失败: %w", receivedInput.SenderID, err)
	}

	// 验证发送者是否存在
	_, err = s.validateUserExists(ctx, senderIDUint)
	if err != nil {
		return fmt.Errorf("发送者用户验证失败: %w", err)
	}

	// 处理不同类型的消息
	var conversation *models.Conversation
	var conversationID uint

	// 判断是否有会话ID，有会话ID时直接使用
	if receivedInput.ConversationID != "" {
		// 已有会话ID，直接使用
		conversationIDUint, err := storage.StrToUint(receivedInput.ConversationID)
		if err != nil {
			return fmt.Errorf("转换会话ID '%s' 失败: %w", receivedInput.ConversationID, err)
		}

		// 查询会话是否存在
		conversation, err = s.convoRepo.GetConversationByID(ctx, conversationIDUint)
		if err != nil {
			return fmt.Errorf("查询会话ID=%d失败: %w", conversationIDUint, err)
		}

		// 验证发送者是否是会话参与者
		_, err = s.convoRepo.GetParticipant(ctx, conversationIDUint, senderIDUint)
		if err != nil {
			return fmt.Errorf("发送者ID=%d不是会话ID=%d的参与者: %w", senderIDUint, conversationIDUint, err)
		}

		conversationID = conversationIDUint
		fmt.Printf("[ProcessKafkaMessage] 使用现有会话ID=%d，会话类型=%s\n", conversationID, conversation.Type)

		// 如果是群组会话，不需要额外处理ReceiverID，因为消息已经关联到会话
		// 群聊中ReceiverID可能是会话ID而非用户ID，这是预期行为
	} else {
		// 没有会话ID，尝试查找或创建私聊会话
		// 只有私聊可以通过receiverID查找会话，群聊必须提供conversationId

		// 转换接收者ID为uint
		receiverIDUint, err := storage.StrToUint(receivedInput.ReceiverID)
		if err != nil {
			return fmt.Errorf("转换接收者ID '%s' 失败: %w", receivedInput.ReceiverID, err)
		}

		// 验证接收者是否存在 - 私聊时需要验证接收者
		_, err = s.validateUserExists(ctx, receiverIDUint)
		if err != nil {
			return fmt.Errorf("接收者用户验证失败: %w", err)
		}

		// 获取或创建私聊会话 (发送者和接收者的会话)
		var privateConversation *models.Conversation

		tx := s.convoRepo.GetDB().Begin()
		if tx.Error != nil {
			return fmt.Errorf("开始事务失败: %w", tx.Error)
		}

		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
				log.Printf("创建会话过程中发生异常，已回滚: %v", r)
			}
		}()

		// 查找或创建私聊会话
		privateConversation, err = s.convoRepo.FindOrCreatePrivateConversationWithTx(ctx, tx, senderIDUint, receiverIDUint)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("查找或创建私聊会话失败: %w", err)
		} else {
			conversation = privateConversation
			conversationID = conversation.ID
			fmt.Printf("[ProcessKafkaMessage] 找到现有私聊会话ID=%d\n", conversationID)
		}

		if err := tx.Commit().Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("提交事务失败: %w", err)
		}
	}

	// 创建消息
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

	// 使用事务保存消息和更新会话
	tx := s.convoRepo.GetDB().Begin()
	if tx.Error != nil {
		return fmt.Errorf("开始保存消息事务失败: %w", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Printf("保存消息过程中发生异常，已回滚: %v", r)
		}
	}()

	if err := tx.Create(dbMessage).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("存储消息到数据库失败: %w", err)
	}

	// 更新会话的最后一条消息
	conversation.LastMessageID = &dbMessage.ID
	if err := tx.Save(conversation).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新会话 %d 的 LastMessageID 失败: %w", conversationID, err)
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("提交消息保存事务失败: %w", err)
	}

	fmt.Printf("[ProcessKafkaMessage] 消息已保存: ID=%d, 会话ID=%d\n", dbMessage.ID, conversationID)

	// --- 新增：将消息推送到 WebSocketOutgoingTopic ---
	// 构建发送给客户端的 websocket.Message
	outgoingWsMsg := &imtypes.Message{
		ID:             dbMessage.IDString(),
		Type:           imtypes.MessageType(dbMessage.Type),
		Content:        dbMessage.Content,
		SenderID:       strconv.FormatUint(uint64(dbMessage.SenderID), 10),
		Timestamp:      dbMessage.SentAt,
		ConversationID: strconv.FormatUint(uint64(dbMessage.ConversationID), 10),
	}
	if dbMessage.Type == models.FileMessageTypeDB || dbMessage.Type == models.ImageMessageTypeDB {
		fileMeta, _ := dbMessage.GetFileMetadata()
		if fileMeta != nil {
			outgoingWsMsg.FileName = fileMeta.FileName
			outgoingWsMsg.FileSize = fileMeta.FileSize
		}
	}

	// 获取会话所有参与者，以便可以向他们发送消息
	participants, err := s.convoRepo.GetConversationParticipants(ctx, conversationID)
	if err != nil {
		log.Printf("获取会话参与者失败: %v", err)
	} else {
		fmt.Printf("[ProcessKafkaMessage] 会话ID=%d有%d名参与者\n", conversationID, len(participants))
	}

	// 群聊消息发给所有人，私聊只需发给接收者
	if conversation.Type == models.GroupConversation {
		// 群聊：向所有参与者发送消息
		for _, participant := range participants {
			// 给每个参与者单独发送一条消息
			if participant.UserID == senderIDUint {
				// 跳过发送者自己，因为他已经在前端看到了乐观更新的消息
				continue
			}

			// 为每个接收者单独设置ReceiverID
			outgoingWsMsg.ReceiverID = strconv.FormatUint(uint64(participant.UserID), 10)
			participantMessageBytes, err := json.Marshal(outgoingWsMsg)
			if err != nil {
				log.Printf("序列化发送给参与者 %d 的消息失败: %v", participant.UserID, err)
				continue
			}

			// 以接收者ID为key发送消息
			outgoingKey := []byte(outgoingWsMsg.ReceiverID)
			if err := s.producer.SendMessage(ctx, s.cfg.Kafka.WebSocketOutgoingTopic, outgoingKey, participantMessageBytes); err != nil {
				log.Printf("发送消息到参与者 %d 失败: %v", participant.UserID, err)
			}
			fmt.Printf("[ProcessKafkaMessage] 发送群聊消息给参与者ID=%d\n", participant.UserID)
		}
	} else {
		// 私聊：只向接收者发送消息
		outgoingWsMsg.ReceiverID = receivedInput.ReceiverID
		outgoingMsgBytes, _ := json.Marshal(outgoingWsMsg)
		outgoingKey := []byte(receivedInput.ReceiverID)

		if err := s.producer.SendMessage(ctx, s.cfg.Kafka.WebSocketOutgoingTopic, outgoingKey, outgoingMsgBytes); err != nil {
			log.Printf("发送私聊消息到接收者ID=%s失败: %v", receivedInput.ReceiverID, err)
		}
		fmt.Printf("[ProcessKafkaMessage] 发送私聊消息到接收者ID=%s\n", receivedInput.ReceiverID)
	}

	return nil
}

// validateUserExists 检查用户是否存在
func (s *messageService) validateUserExists(ctx context.Context, userID uint) (bool, error) {
	var count int64
	err := s.convoRepo.GetDB().WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("查询用户 %d 是否存在失败: %w", userID, err)
	}
	if count == 0 {
		return false, fmt.Errorf("用户 %d 不存在", userID)
	}
	return true, nil
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
