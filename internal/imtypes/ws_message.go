package imtypes

import "time"

// MessageType defines the type of a message.
type MessageType string

const (
	TextMessageType   MessageType = "text"
	ImageMessageType  MessageType = "image"
	FileMessageType   MessageType = "file"
	EmojiMessageType  MessageType = "emoji"
	SystemMessageType MessageType = "system" // For system notifications, e.g., user joined/left
)

// Message defines the structure for messages exchanged over WebSocket or to be sent to clients.
type Message struct {
	ID             string      `json:"id"`
	Type           MessageType `json:"type"`
	Content        string      `json:"content"`
	SenderID       string      `json:"senderId"`
	ReceiverID     string      `json:"receiverId"`
	Timestamp      time.Time   `json:"timestamp"`
	FileName       string      `json:"fileName,omitempty"`
	FileSize       int64       `json:"fileSize,omitempty"`
	ConversationID string      `json:"conversationId,omitempty"`
}
