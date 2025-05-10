package models

import "time"

// User 代表系统中的用户。
type User struct {
	BaseModel
	Username     string     `gorm:"type:varchar(100);uniqueIndex;not null" json:"username"`
	PasswordHash string     `gorm:"type:varchar(255);not null" json:"-"` // 不暴露密码哈希
	Email        string     `gorm:"type:varchar(100);uniqueIndex" json:"email,omitempty"`
	Nickname     string     `gorm:"type:varchar(100)" json:"nickname,omitempty"`
	AvatarURL    string     `gorm:"type:varchar(255)" json:"avatarUrl,omitempty"`
	Status       string     `gorm:"type:varchar(20);default:'offline'" json:"status,omitempty"` // 例如：online, offline, busy
	LastSeenAt   *time.Time `json:"lastSeenAt,omitempty"`
	Bio          string     `gorm:"type:text" json:"bio,omitempty"`

	// 关联关系
	Messages      []Message       `gorm:"foreignKey:SenderID" json:"messages,omitempty"`                       // 用户发送的消息
	Conversations []*Conversation `gorm:"many2many:conversation_participants;" json:"conversations,omitempty"` // 用户参与的会话
	// Groups        []*GroupMember  `gorm:"foreignKey:UserID" json:"group_memberships,omitempty"` // 用户的群组成员身份记录
	// 或者直接关联到 Group，如果只想获取群组列表而不是成员详情：
	UserGroups []*Group `gorm:"many2many:group_members;" json:"user_groups,omitempty"` // 用户所属的群组
}

// UserBasicInfo holds minimal public information about a user.
// Used for scenarios like displaying requester info in friend requests.
type UserBasicInfo struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname,omitempty"`
	AvatarURL string `json:"avatarUrl,omitempty"`
}

// TableName 指定 User 模型的表名。
func (User) TableName() string {
	return "users"
}
