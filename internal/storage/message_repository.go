package storage

import (
	"context"

	"gorm.io/gorm"

	"im-go/internal/models"
)

// MessageRepository 定义了消息数据操作的接口。
type MessageRepository interface {
	Create(ctx context.Context, message *models.Message) error
	GetByID(ctx context.Context, id uint) (*models.Message, error)
	GetByConversationID(ctx context.Context, conversationID uint, limit int, offset int) ([]*models.Message, error)
	// Update(ctx context.Context, message *models.Message) error // 一般消息创建后不直接更新内容，可能更新状态
	// Delete(ctx context.Context, id uint) error // 消息通常是软删除或逻辑删除，较少物理删除
	// UpdateStatus(ctx context.Context, messageIDs []uint, status string) error // 批量更新消息状态，例如已读
}

// gormMessageRepository 使用 GORM 实现 MessageRepository。
type gormMessageRepository struct {
	db *gorm.DB
}

// NewGormMessageRepository 创建一个新的基于 GORM 的 MessageRepository。
func NewGormMessageRepository(db *gorm.DB) MessageRepository {
	return &gormMessageRepository{db: db}
}

// Create 在数据库中创建一条新的消息记录。
func (r *gormMessageRepository) Create(ctx context.Context, message *models.Message) error {
	return r.db.WithContext(ctx).Create(message).Error
}

// GetByID 通过ID检索消息。
func (r *gormMessageRepository) GetByID(ctx context.Context, id uint) (*models.Message, error) {
	var message models.Message
	// Preload Sender to get user information along with the message
	err := r.db.WithContext(ctx).Preload("Sender").First(&message, id).Error
	if err != nil {
		return nil, err
	}
	return &message, nil
}

// GetByConversationID 通过会话ID检索消息列表，支持分页。
func (r *gormMessageRepository) GetByConversationID(ctx context.Context, conversationID uint, limit int, offset int) ([]*models.Message, error) {
	var messages []*models.Message
	query := r.db.WithContext(ctx).Where("conversation_id = ?", conversationID).Order("sent_at DESC") // 通常按发送时间倒序排列

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	// Preload Sender to get user information along with the messages
	err := query.Preload("Sender").Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}
