package imtypes

import "time"

// RawMessageInput 定义了一个通用的消息传输结构。
// 这可以是一个独立的 DTO (Data Transfer Object)
type RawMessageInput struct {
	ID         string    `json:"id,omitempty"`       // 客户端生成的消息ID (可选)
	Type       string    `json:"type"`               // 消息类型 (例如: text, image, file), 可以是 ws_message.go 中定义的 MessageType
	Content    []byte    `json:"content"`            // 原始消息内容
	SenderID   string    `json:"senderId"`           // 发送者ID
	ReceiverID string    `json:"receiverId"`         // 接收者ID (用户ID或群组ID)
	Timestamp  time.Time `json:"timestamp"`          // 时间戳
	FileName   string    `json:"fileName,omitempty"` // 文件名 (如果适用)
	FileSize   int64     `json:"fileSize,omitempty"` // 文件大小 (如果适用)
}
