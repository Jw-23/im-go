package models

import "time"

// ConversationType 定义了会话的类型。
type ConversationType string

const (
	PrivateConversation ConversationType = "private" // 一对一聊天
	GroupConversation   ConversationType = "group"   // 群组聊天
)

// Conversation 代表一个聊天会话（一对一或群组）。
type Conversation struct {
	BaseModel
	Type ConversationType `gorm:"type:varchar(20);not null;index" json:"type"`

	// 对于群组会话，这是群组的 ID。
	// 对于私聊会话，这可以是 null 或为该用户对生成的唯一ID。
	// 或者，私聊会话可能不需要在这里有直接的 GroupID 等价物，可以通过涉及的用户对来识别。
	// 我们将其用作通用引用。如果 Type 是 "group"，则这是 Group.ID。
	TargetID uint `gorm:"index" json:"targetId,omitempty"` // 例如，如果 Type 是 group，则为 GroupID

	// LastMessageID 可用于快速获取最后一条消息以供显示。
	// 可为空，因为新会话可能还没有消息。
	LastMessageID *uint `gorm:"index" json:"lastMessageId,omitempty"`

	// 关联关系 (用于预加载或直接查询，实际成员关系由 ConversationParticipant 管理)
	Users []*User `gorm:"many2many:conversation_participants;" json:"users,omitempty"` // 参与此会话的用户
	// LastMessage  *Message                  `gorm:"foreignKey:LastMessageID" json:"lastMessage,omitempty"`       // 此会话的最后一条消息 // 暂时注释掉
	Messages     []Message                 `gorm:"foreignKey:ConversationID" json:"messages,omitempty"`     // 此会话的所有消息 (按需加载)
	Participants []ConversationParticipant `gorm:"foreignKey:ConversationID" json:"participants,omitempty"` // 此会话的所有参与者记录 (按需加载)

	// 未读计数可以在这里管理，或者如果变得复杂（每个用户每个会话），则在单独的表中管理。
	// 为简单起见，我们最初可能在客户端或通过消息状态处理未读状态。

	// 会话的附加元数据，例如用户为私聊设置的自定义名称。
	// 对于群聊，群名在 Group 模型中。
	// Name          string `gorm:"type:varchar(255)" json:"name,omitempty"` // 例如，用户为私聊定义的名称
	// AvatarURL     string `gorm:"type:varchar(255)" json:"avatarUrl,omitempty"` // 用户为私聊定义的头像
}

// TableName 指定 Conversation 模型的表名。
func (Conversation) TableName() string {
	return "conversations"
}

// ConversationParticipant 将用户链接到会话。
// 此表对于私聊（2个参与者）和群聊（多个参与者）都至关重要。
type ConversationParticipant struct {
	BaseModel                 // 或者如果对于连接表更喜欢，可以仅用 ID, CreatedAt, UpdatedAt
	ConversationID uint       `gorm:"primaryKey;autoIncrement:false" json:"conversationId"`
	UserID         uint       `gorm:"primaryKey;autoIncrement:false" json:"userId"`
	JoinedAt       time.Time  `json:"joinedAt"`
	LastReadAt     *time.Time `json:"lastReadAt,omitempty"`                   // 跟踪用户在此会话中最后阅读消息的时间
	IsAdmin        bool       `gorm:"default:false" json:"isAdmin,omitempty"` // 与群组会话相关

	// 关联关系
	User         User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Conversation Conversation `gorm:"foreignKey:ConversationID" json:"conversation,omitempty"`
}

// TableName 指定 ConversationParticipant 模型的表名。
func (ConversationParticipant) TableName() string {
	return "conversation_participants"
}
