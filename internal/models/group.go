package models

import "time"

// Group 代表一个聊天群组。
type Group struct {
	BaseModel
	Name        string `gorm:"type:varchar(100);not null" json:"name"`
	Description string `gorm:"type:text" json:"description,omitempty"`
	AvatarURL   string `gorm:"type:varchar(255)" json:"avatarUrl,omitempty"`
	OwnerID     uint   `gorm:"not null" json:"ownerId"` // 指向 User 模型的外键
	MemberCount int    `gorm:"default:0" json:"memberCount"`

	// 群组设置，例如：公开/私有，加入权限
	IsPublic      bool   `gorm:"default:true" json:"isPublic"`                                          // 如果为 true，则任何人都可以查找并加入（或请求加入）
	JoinCondition string `gorm:"type:varchar(50);default:'direct_join'" json:"joinCondition,omitempty"` // 例如：direct_join, approval_required, invite_only

	// 关联关系
	Owner   User          `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Members []GroupMember `gorm:"foreignKey:GroupID" json:"members,omitempty"` // 群组成员列表
	// 一个 Group 逻辑上对应一个 Conversation (Type="group", TargetID=Group.ID)
	// 这个关联通常在服务层或查询时处理，而不是直接在模型中定义一个 `Conversation` 字段，以避免复杂性。
}

// TableName 指定 Group 模型的表名。
func (Group) TableName() string {
	return "groups"
}

// GroupMemberRole 定义了用户在群组中的角色。
type GroupMemberRole string

const (
	AdminRole  GroupMemberRole = "admin"
	MemberRole GroupMemberRole = "member"
	// 可以添加其他角色，如 Moderator
)

// GroupMember 将用户链接到群组并定义其角色。
type GroupMember struct {
	BaseModel                 // 或者如果对于连接表更喜欢，可以仅用 ID, CreatedAt, UpdatedAt
	GroupID   uint            `gorm:"primaryKey;autoIncrement:false" json:"groupId"`
	UserID    uint            `gorm:"primaryKey;autoIncrement:false" json:"userId"`
	Role      GroupMemberRole `gorm:"type:varchar(20);default:'member'" json:"role"`
	JoinedAt  time.Time       `json:"joinedAt"`
	Alias     string          `gorm:"type:varchar(100)" json:"alias,omitempty"` // 在此特定群组中的昵称

	// 关联关系
	User  User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Group Group `gorm:"foreignKey:GroupID" json:"group,omitempty"`
}

// TableName 指定 GroupMember 模型的表名。
func (GroupMember) TableName() string {
	return "group_members"
}
