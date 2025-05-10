package models

// Friendship represents a friendship relationship between two users.
// To avoid duplicates and simplify queries, UserID1 should always be less than UserID2.
type Friendship struct {
	BaseModel
	UserID1 uint `gorm:"not null;uniqueIndex:idx_friendship_users"` // ID of the first user
	User1   User `gorm:"foreignKey:UserID1"`
	UserID2 uint `gorm:"not null;uniqueIndex:idx_friendship_users"` // ID of the second user
	User2   User `gorm:"foreignKey:UserID2"`

	// Optional: Could store the timestamp when the friendship was established,
	// though BaseModel.CreatedAt can also serve this purpose if friendship is created upon acceptance.
	// AcceptedAt time.Time `gorm:"not null"`
}

// EnsureCanonicalOrder sets UserID1 to the smaller ID and UserID2 to the larger ID.
// This should be called before creating a Friendship record.
func (f *Friendship) EnsureCanonicalOrder() {
	if f.UserID1 > f.UserID2 {
		f.UserID1, f.UserID2 = f.UserID2, f.UserID1
	}
}
