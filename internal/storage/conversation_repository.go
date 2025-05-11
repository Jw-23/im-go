package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"im-go/internal/models"
)

// ConversationRepository 定义了会话数据操作的接口。
type ConversationRepository interface {
	CreateConversation(ctx context.Context, conversation *models.Conversation) error
	GetConversationByID(ctx context.Context, id uint) (*models.Conversation, error)
	// GetUserConversations 获取用户参与的所有会话列表，可包含最后一条消息用于预览
	GetUserConversations(ctx context.Context, userID uint, limit int, offset int) ([]*models.Conversation, error)
	UpdateConversation(ctx context.Context, conversation *models.Conversation) error
	// FindPrivateConversationByUsers 尝试查找两个用户之间的私聊会话
	FindPrivateConversationByUsers(ctx context.Context, userID1 uint, userID2 uint) (*models.Conversation, error)
	// FindOrCreatePrivateConversationWithTx 在事务中查找或创建两个用户之间的私聊会话
	FindOrCreatePrivateConversationWithTx(ctx context.Context, tx *gorm.DB, senderID uint, receiverID uint) (*models.Conversation, error)

	AddParticipant(ctx context.Context, participant *models.ConversationParticipant) error
	GetParticipant(ctx context.Context, conversationID uint, userID uint) (*models.ConversationParticipant, error)
	UpdateParticipant(ctx context.Context, participant *models.ConversationParticipant) error
	RemoveParticipant(ctx context.Context, conversationID uint, userID uint) error
	GetConversationParticipants(ctx context.Context, conversationID uint) ([]*models.ConversationParticipant, error)

	// GetDB 返回底层数据库连接，用于事务操作
	GetDB() *gorm.DB
}

// gormConversationRepository 使用 GORM 实现 ConversationRepository。
type gormConversationRepository struct {
	db *gorm.DB
}

// NewGormConversationRepository 创建一个新的基于 GORM 的 ConversationRepository。
func NewGormConversationRepository(db *gorm.DB) ConversationRepository {
	return &gormConversationRepository{db: db}
}

// CreateConversation 创建一个新的会话。
func (r *gormConversationRepository) CreateConversation(ctx context.Context, conversation *models.Conversation) error {
	return r.db.WithContext(ctx).Create(conversation).Error
}

// GetConversationByID 通过ID检索会话。
func (r *gormConversationRepository) GetConversationByID(ctx context.Context, id uint) (*models.Conversation, error) {
	var conversation models.Conversation
	// 示例：预加载参与者和最后一条消息，具体按需调整
	// err := r.db.WithContext(ctx).Preload("Participants.User").Preload("LastMessage.Sender").First(&conversation, id).Error
	err := r.db.WithContext(ctx).First(&conversation, id).Error
	if err != nil {
		return nil, err
	}
	return &conversation, nil
}

// GetUserConversations 获取用户参与的所有会话列表。
func (r *gormConversationRepository) GetUserConversations(ctx context.Context, userID uint, limit int, offset int) ([]*models.Conversation, error) {
	var conversations []*models.Conversation
	// 此查询需要连接 conversation_participants 表
	query := r.db.WithContext(ctx).Joins("JOIN conversation_participants cp ON cp.conversation_id = conversations.id").
		Where("cp.user_id = ?", userID).Order("conversations.updated_at DESC") // 按会话更新时间排序

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	// err := query.Preload("LastMessage").Find(&conversations).Error // 预加载最后一条消息
	err := query.Find(&conversations).Error
	return conversations, err
}

// UpdateConversation 更新会话信息，例如最后一条消息ID。
func (r *gormConversationRepository) UpdateConversation(ctx context.Context, conversation *models.Conversation) error {
	return r.db.WithContext(ctx).Save(conversation).Error
}

// AddParticipant向会话中添加参与者。
func (r *gormConversationRepository) AddParticipant(ctx context.Context, participant *models.ConversationParticipant) error {
	// 首先检查用户是否存在
	var userExists int64
	if err := r.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", participant.UserID).
		Count(&userExists).Error; err != nil {
		return fmt.Errorf("检查用户 %d 是否存在时出错: %w", participant.UserID, err)
	}

	if userExists == 0 {
		return fmt.Errorf("用户 %d 不存在，无法添加为会话参与者", participant.UserID)
	}

	// 检查参与者是否已存在（避免重复添加）
	var participantExists int64
	if err := r.db.WithContext(ctx).Model(&models.ConversationParticipant{}).
		Where("conversation_id = ? AND user_id = ?", participant.ConversationID, participant.UserID).
		Count(&participantExists).Error; err != nil {
		return fmt.Errorf("检查参与者是否已存在时出错: %w", err)
	}

	if participantExists > 0 {
		// 参与者已存在，视为成功
		return nil
	}

	// 添加参与者
	return r.db.WithContext(ctx).Create(participant).Error
}

// GetParticipant 获取会话中的特定参与者信息。
func (r *gormConversationRepository) GetParticipant(ctx context.Context, conversationID uint, userID uint) (*models.ConversationParticipant, error) {
	var participant models.ConversationParticipant
	err := r.db.WithContext(ctx).Where("conversation_id = ? AND user_id = ?", conversationID, userID).First(&participant).Error
	return &participant, err
}

// UpdateParticipant 更新参与者信息，例如 last_read_at。
func (r *gormConversationRepository) UpdateParticipant(ctx context.Context, participant *models.ConversationParticipant) error {
	return r.db.WithContext(ctx).Save(participant).Error
}

// RemoveParticipant 从会话中移除参与者。
func (r *gormConversationRepository) RemoveParticipant(ctx context.Context, conversationID uint, userID uint) error {
	return r.db.WithContext(ctx).Where("conversation_id = ? AND user_id = ?", conversationID, userID).Delete(&models.ConversationParticipant{}).Error
}

// GetConversationParticipants 获取会话的所有参与者。
func (r *gormConversationRepository) GetConversationParticipants(ctx context.Context, conversationID uint) ([]*models.ConversationParticipant, error) {
	var participants []*models.ConversationParticipant
	// err := r.db.WithContext(ctx).Preload("User").Where("conversation_id = ?", conversationID).Find(&participants).Error
	err := r.db.WithContext(ctx).Where("conversation_id = ?", conversationID).Find(&participants).Error
	return participants, err
}

// FindPrivateConversationByUsers 尝试查找两个特定用户之间的私聊会话。
// 这对于防止重复创建相同的1v1会话很有用。
func (r *gormConversationRepository) FindPrivateConversationByUsers(ctx context.Context, userID1 uint, userID2 uint) (*models.Conversation, error) {
	var conversation models.Conversation

	// 正确的 GORM 实现：
	// 我们需要找到一个会话 'c'，它同时关联到两个不同的 conversation_participants 记录 (cp1 和 cp2)
	// cp1 的 user_id 是 userID1
	// cp2 的 user_id 是 userID2
	// 并且这两个参与者记录都指向同一个 conversation_id (即 c.id)
	err := r.db.WithContext(ctx).
		Table("conversations as c").                                                                               // 主表 conversations，别名 c
		Select("c.*").                                                                                             // 只选择 conversations 表的所有列
		Joins("JOIN conversation_participants as cp1 ON c.id = cp1.conversation_id AND cp1.user_id = ?", userID1). // 第一个参与者
		Joins("JOIN conversation_participants as cp2 ON c.id = cp2.conversation_id AND cp2.user_id = ?", userID2). // 第二个参与者
		Where("c.type = ?", models.PrivateConversation).                                                           // 会话类型为 private
		First(&conversation).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // RecordNotFound means no such conversation exists, which is not an application error.
		}
		return nil, err // Other errors (DB connection, syntax error not caught by GORM, etc.)
	}
	return &conversation, nil
}

// FindOrCreatePrivateConversationWithTx 在提供的事务中查找或创建私聊会话
func (r *gormConversationRepository) FindOrCreatePrivateConversationWithTx(ctx context.Context, tx *gorm.DB, senderID uint, receiverID uint) (*models.Conversation, error) {
	// 确保 userID1 < userID2，以使查找具有确定性，避免重复会话
	userID1, userID2 := senderID, receiverID
	if userID1 > userID2 {
		userID1, userID2 = userID2, userID1
	}

	// 使用加锁查询确保在并发情况下会话唯一性
	var conversation models.Conversation
	err := tx.WithContext(ctx).
		Set("gorm:query_option", "FOR UPDATE"). // 使用行锁防止并发问题
		Table("conversations AS c").
		Select("c.*").
		Joins("JOIN conversation_participants AS cp1 ON c.id = cp1.conversation_id AND cp1.user_id = ?", userID1).
		Joins("JOIN conversation_participants AS cp2 ON c.id = cp2.conversation_id AND cp2.user_id = ?", userID2).
		Where("c.type = ?", models.PrivateConversation).
		First(&conversation).Error

	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("查找私聊会话失败: %w", err)
		}

		// 会话不存在，创建新的私聊会话
		newConversation := &models.Conversation{Type: models.PrivateConversation}
		if err := tx.WithContext(ctx).Create(newConversation).Error; err != nil {
			return nil, fmt.Errorf("创建新会话失败: %w", err)
		}

		// 添加参与者
		p1 := &models.ConversationParticipant{ConversationID: newConversation.ID, UserID: userID1, JoinedAt: time.Now()}
		p2 := &models.ConversationParticipant{ConversationID: newConversation.ID, UserID: userID2, JoinedAt: time.Now()}

		// 检查并添加参与者1
		var count int64
		if err := tx.WithContext(ctx).Model(&models.ConversationParticipant{}).
			Where("conversation_id = ? AND user_id = ?", newConversation.ID, userID1).
			Count(&count).Error; err != nil {
			return nil, fmt.Errorf("检查参与者1是否已存在失败: %w", err)
		}

		if count == 0 {
			if err := tx.WithContext(ctx).Create(p1).Error; err != nil {
				return nil, fmt.Errorf("添加参与者1到会话 %d 失败: %w", newConversation.ID, err)
			}
		}

		// 检查并添加参与者2
		if err := tx.WithContext(ctx).Model(&models.ConversationParticipant{}).
			Where("conversation_id = ? AND user_id = ?", newConversation.ID, userID2).
			Count(&count).Error; err != nil {
			return nil, fmt.Errorf("检查参与者2是否已存在失败: %w", err)
		}

		if count == 0 {
			if err := tx.WithContext(ctx).Create(p2).Error; err != nil {
				return nil, fmt.Errorf("添加参与者2到会话 %d 失败: %w", newConversation.ID, err)
			}
		}

		return newConversation, nil
	}

	return &conversation, nil
}

// GetDB 返回底层数据库连接，用于事务操作
func (r *gormConversationRepository) GetDB() *gorm.DB {
	return r.db
}
