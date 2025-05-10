package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"im-go/internal/models"
	"im-go/internal/storage"

	"gorm.io/gorm"
)

// ConversationService 定义了会话相关服务的接口。
type ConversationService interface {
	// GetOrCreatePrivateConversation 获取或创建两个用户之间的私聊会话。
	// 返回会话对象以及一个布尔值，指示会话是否是新创建的。
	GetOrCreatePrivateConversation(ctx context.Context, userID1, userID2 uint) (*models.Conversation, bool, error)
	GetUserConversations(ctx context.Context, userID uint, limit, offset int) ([]*models.Conversation, error)
	GetConversationDetails(ctx context.Context, conversationID uint, userID uint) (*models.Conversation, error) // userID 用于权限检查或个性化信息
	GetConversationParticipants(ctx context.Context, conversationID uint) ([]*models.ConversationParticipant, error)
	GetUserByID(ctx context.Context, userID uint) (*models.User, error)
	// UpdateConversationSettings(ctx context.Context, userID, conversationID uint, settings map[string]interface{}) error
	// LeaveConversation(ctx context.Context, userID, conversationID uint) error
}

// conversationService 是 ConversationService 的实现。
type conversationService struct {
	convoRepo storage.ConversationRepository
	userRepo  storage.UserRepository // 可能需要用于获取参与者信息
}

// NewConversationService 创建一个新的 ConversationService 实例。
func NewConversationService(convoRepo storage.ConversationRepository, userRepo storage.UserRepository) ConversationService {
	return &conversationService{convoRepo: convoRepo, userRepo: userRepo}
}

// GetOrCreatePrivateConversation 获取或创建两个用户之间的私聊会话。
func (s *conversationService) GetOrCreatePrivateConversation(ctx context.Context, userID1, userID2 uint) (*models.Conversation, bool, error) {
	if userID1 == userID2 {
		return nil, false, fmt.Errorf("不能与自己创建私聊会话")
	}

	// 确保 userID1 < userID2，以使查找具有确定性，避免重复会话
	if userID1 > userID2 {
		userID1, userID2 = userID2, userID1
	}

	conversation, err := s.convoRepo.FindPrivateConversationByUsers(ctx, userID1, userID2)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) { // 如果是 gorm.ErrRecordNotFound 以外的错误
		return nil, false, fmt.Errorf("查找私聊会话失败: %w", err)
	}

	if conversation != nil {
		return conversation, false, nil // 会话已存在
	}

	// 会话不存在，创建新的私聊会话
	newConversation := &models.Conversation{
		Type: models.PrivateConversation,
		// TargetID 对于私聊可以不设置，或根据需要定义其含义
	}
	if err := s.convoRepo.CreateConversation(ctx, newConversation); err != nil {
		return nil, true, fmt.Errorf("创建新会话失败: %w", err)
	}

	// 添加参与者
	p1 := &models.ConversationParticipant{ConversationID: newConversation.ID, UserID: userID1, JoinedAt: time.Now()}
	p2 := &models.ConversationParticipant{ConversationID: newConversation.ID, UserID: userID2, JoinedAt: time.Now()}

	if err := s.convoRepo.AddParticipant(ctx, p1); err != nil {
		// 注意：这里可能需要事务回滚，如果创建会话成功但添加参与者失败
		return newConversation, true, fmt.Errorf("为会话 %d 添加参与者 %d 失败: %w", newConversation.ID, userID1, err)
	}
	if err := s.convoRepo.AddParticipant(ctx, p2); err != nil {
		return newConversation, true, fmt.Errorf("为会话 %d 添加参与者 %d 失败: %w", newConversation.ID, userID2, err)
	}

	return newConversation, true, nil
}

// GetUserConversations 获取用户参与的所有会话列表。
func (s *conversationService) GetUserConversations(ctx context.Context, userID uint, limit, offset int) ([]*models.Conversation, error) {
	// TODO: 在 Repository 或 Service 层填充会话的最后一条消息、未读计数、以及对方用户信息等
	// 例如，通过遍历 conversations，对每个 conversation 调用 convoRepo.GetConversationParticipants 和 msgRepo.GetByID(LastMessageID)
	// 并根据需要组合成更丰富的会话列表项返回给前端。
	return s.convoRepo.GetUserConversations(ctx, userID, limit, offset)
}

// GetConversationDetails 获取会话的详细信息，包括参与者等。
func (s *conversationService) GetConversationDetails(ctx context.Context, conversationID uint, userID uint) (*models.Conversation, error) {
	// 1. 检查用户是否有权限查看此会话
	_, err := s.convoRepo.GetParticipant(ctx, conversationID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("用户 %d 不是会话 %d 的成员，无权查看", userID, conversationID)
		}
		return nil, fmt.Errorf("检查用户 %d 在会话 %d 中的参与情况失败: %w", userID, conversationID, err)
	}

	conversation, err := s.convoRepo.GetConversationByID(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("获取会话 %d 详情失败: %w", conversationID, err)
	}

	// TODO: 预加载或手动加载会话的参与者信息 (User) 和最后一条消息 (Message)
	// participants, _ := s.convoRepo.GetConversationParticipants(ctx, conversationID)
	// conversation.Participants = participants // GORM 的 Preload 更佳

	return conversation, nil
}

func (s *conversationService) GetConversationParticipants(ctx context.Context, conversationID uint) ([]*models.ConversationParticipant, error) {
	return s.convoRepo.GetConversationParticipants(ctx, conversationID)
}

// GetUserByID retrieves a user by their ID using the user repository.
func (s *conversationService) GetUserByID(ctx context.Context, userID uint) (*models.User, error) {
	if s.userRepo == nil {
		return nil, errors.New("userRepository is not initialized in conversationService")
	}
	return s.userRepo.GetByID(ctx, userID)
}
