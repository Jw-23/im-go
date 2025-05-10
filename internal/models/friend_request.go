package models

// FriendRequestStatus 定义好友请求的状态
type FriendRequestStatus string

const (
	FriendRequestStatusPending   FriendRequestStatus = "pending"
	FriendRequestStatusAccepted  FriendRequestStatus = "accepted"
	FriendRequestStatusRejected  FriendRequestStatus = "rejected"
	FriendRequestStatusCancelled FriendRequestStatus = "cancelled" // If sender cancels
)

// FriendRequest 代表一个好友请求记录
type FriendRequest struct {
	BaseModel                           // Embed BaseModel
	RequesterUserID uint                `gorm:"not null;index:idx_friend_request_users"`     // 请求发送者
	RecipientUserID uint                `gorm:"not null;index:idx_friend_request_users"`     // 请求接收者
	Status          FriendRequestStatus `gorm:"type:varchar(20);not null;default:'pending'"` // 请求状态
	RequestMessage  string              `gorm:"type:text"`                                   // 可选的请求消息

	// Optional: Define relationships if needed for easier loading
	// Requester User `gorm:"foreignKey:RequesterUserID"`
	// Recipient User `gorm:"foreignKey:RecipientUserID"`
}

// FriendRequestWithRequester is a DTO that includes friend request details
// along with basic information about the user who sent the request.
// Useful for API responses for listing pending requests.
type FriendRequestWithRequester struct {
	FriendRequest                // Embed the core FriendRequest data
	Requester     *UserBasicInfo `json:"requester"` // Embed basic requester info
}
