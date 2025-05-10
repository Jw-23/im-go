package models

import (
	"encoding/json"
	"time"
)

// MessageTypeDB 定义了存储在数据库中的消息类型。
// 重命名以避免在没有别名的情况下与 websocket.MessageType 发生冲突。
type MessageTypeDB string

const (
	TextMessageTypeDB   MessageTypeDB = "text"
	ImageMessageTypeDB  MessageTypeDB = "image"
	FileMessageTypeDB   MessageTypeDB = "file"
	EmojiMessageTypeDB  MessageTypeDB = "emoji"
	SystemMessageTypeDB MessageTypeDB = "system" // 用于系统通知
	AudioMessageTypeDB  MessageTypeDB = "audio"
	VideoMessageTypeDB  MessageTypeDB = "video"
)

// Message 代表存储在数据库中的聊天消息。
type Message struct {
	BaseModel
	ConversationID uint          `gorm:"index;not null" json:"conversationId"` // 指向 Conversation 模型的外键
	SenderID       uint          `gorm:"index;not null" json:"senderId"`       // 指向 User 模型（发送者）的外键
	Type           MessageTypeDB `gorm:"type:varchar(20);not null" json:"type"`
	Content        string        `gorm:"type:text" json:"content"` // 文本消息内容或文件/图片的URL

	// Metadata 可以存储附加信息，例如文件名、大小、图片尺寸等。
	// 在数据库中以 JSONB 或 TEXT 类型存储。
	MetadataRaw json.RawMessage `gorm:"type:jsonb" json:"metadata,omitempty"`

	Status      string     `gorm:"type:varchar(20);default:'sent'" json:"status,omitempty"` // 例如: sent, delivered, read
	SentAt      time.Time  `gorm:"not null" json:"sentAt"`
	DeliveredAt *time.Time `json:"deliveredAt,omitempty"`
	ReadAt      *time.Time `json:"readAt,omitempty"`

	// 关联关系
	Sender       User         `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
	Conversation Conversation `gorm:"foreignKey:ConversationID" json:"conversation,omitempty"` // 此消息所属的会话
}

// TableName 指定 Message 模型的表名。
func (Message) TableName() string {
	return "messages"
}

// FileMetadata stores metadata for file messages.
// This can be marshaled into Message.MetadataRaw.
type FileMetadata struct {
	FileName string `json:"fileName"`
	FileSize int64  `json:"fileSize"`
	MimeType string `json:"mimeType"`
	URL      string `json:"url"` // URL to access the file
}

// ImageMetadata stores metadata for image messages.
type ImageMetadata struct {
	FileName     string `json:"fileName"`
	FileSize     int64  `json:"fileSize"`
	MimeType     string `json:"mimeType"`
	URL          string `json:"url"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	ThumbnailURL string `json:"thumbnailUrl,omitempty"`
}

// SetMetadata helper to set metadata
func (m *Message) SetMetadata(data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	m.MetadataRaw = jsonData
	return nil
}

// GetFileMetadata helper to get file metadata
func (m *Message) GetFileMetadata() (*FileMetadata, error) {
	if m.Type != FileMessageTypeDB || m.MetadataRaw == nil {
		return nil, nil // Or an error indicating wrong type/no metadata
	}
	var metadata FileMetadata
	err := json.Unmarshal(m.MetadataRaw, &metadata)
	if err != nil {
		return nil, err
	}
	return &metadata, nil
}

// GetImageMetadata helper to get image metadata
func (m *Message) GetImageMetadata() (*ImageMetadata, error) {
	if m.Type != ImageMessageTypeDB || m.MetadataRaw == nil {
		return nil, nil // Or an error indicating wrong type/no metadata
	}
	var metadata ImageMetadata
	err := json.Unmarshal(m.MetadataRaw, &metadata)
	if err != nil {
		return nil, err
	}
	return &metadata, nil
}
