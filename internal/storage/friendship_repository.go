package storage

import (
	"context"
	"im-go/internal/models"

	"gorm.io/gorm"
)

// FriendshipRepository defines the interface for friendship data operations.
type FriendshipRepository interface {
	Create(ctx context.Context, friendship *models.Friendship) error
	AreUsersFriends(ctx context.Context, userID1, userID2 uint) (bool, error)
	GetFriendIDs(ctx context.Context, userID uint) ([]uint, error)
	// TODO: Add other methods like ListFriends(userID uint), RemoveFriendship(userID1, userID2 uint) if needed
}

type gormFriendshipRepository struct {
	db *gorm.DB
}

// NewGormFriendshipRepository creates a new GormFriendshipRepository.
func NewGormFriendshipRepository(db *gorm.DB) FriendshipRepository {
	return &gormFriendshipRepository{db: db}
}

// Create creates a new friendship record in the database.
// It assumes that friendship.EnsureCanonicalOrder() has been called before.
func (r *gormFriendshipRepository) Create(ctx context.Context, friendship *models.Friendship) error {
	return r.db.WithContext(ctx).Create(friendship).Error
}

// AreUsersFriends checks if two users are already friends.
func (r *gormFriendshipRepository) AreUsersFriends(ctx context.Context, userID1, userID2 uint) (bool, error) {
	u1, u2 := userID1, userID2
	if u1 > u2 {
		u1, u2 = u2, u1 // Ensure canonical order for query
	}
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Friendship{}).Where("user_id1 = ? AND user_id2 = ?", u1, u2).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetFriendIDs retrieves a list of user IDs who are friends with the given userID.
func (r *gormFriendshipRepository) GetFriendIDs(ctx context.Context, userID uint) ([]uint, error) {
	var friendIDs []uint
	// We need to find friendships where the given userID is either user_id1 or user_id2
	// and extract the *other* user's ID.
	// This requires two queries or a more complex UNION/subquery.

	// Query 1: Find IDs where the user is UserID1
	var idsPart1 []uint
	err := r.db.WithContext(ctx).Model(&models.Friendship{}).
		Where("user_id1 = ?", userID).
		Pluck("user_id2", &idsPart1).Error
	if err != nil {
		return nil, err
	}

	// Query 2: Find IDs where the user is UserID2
	var idsPart2 []uint
	err = r.db.WithContext(ctx).Model(&models.Friendship{}).
		Where("user_id2 = ?", userID).
		Pluck("user_id1", &idsPart2).Error
	if err != nil {
		return nil, err
	}

	// Combine the results
	friendIDs = append(idsPart1, idsPart2...)

	// Optional: Remove duplicates if somehow possible (shouldn't be with canonical order)
	// Optional: Check if friendIDs is empty

	return friendIDs, nil
}
