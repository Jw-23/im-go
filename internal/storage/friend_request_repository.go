package storage

import (
	"context"
	"errors"
	"im-go/internal/models"

	"gorm.io/gorm"
)

// FriendRequestRepository defines the interface for friend request data operations.
type FriendRequestRepository interface {
	Create(ctx context.Context, request *models.FriendRequest) error
	FindPendingRequest(ctx context.Context, userID1, userID2 uint) (*models.FriendRequest, error)
	// FindExistingRelationship(ctx context.Context, userID1, userID2 uint) (*models.FriendRequest, error) // Checks pending OR accepted
	GetRequestByID(ctx context.Context, requestID uint) (*models.FriendRequest, error)
	UpdateRequestStatus(ctx context.Context, requestID uint, status models.FriendRequestStatus) error
	GetPendingRequestsForUser(ctx context.Context, recipientUserID uint) ([]models.FriendRequest, error)
}

type gormFriendRequestRepository struct {
	db *gorm.DB
}

func NewGormFriendRequestRepository(db *gorm.DB) FriendRequestRepository {
	return &gormFriendRequestRepository{db: db}
}

func (r *gormFriendRequestRepository) Create(ctx context.Context, request *models.FriendRequest) error {
	return r.db.WithContext(ctx).Create(request).Error
}

// FindPendingRequest checks if there is an existing pending request between two users (in either direction).
func (r *gormFriendRequestRepository) FindPendingRequest(ctx context.Context, userID1, userID2 uint) (*models.FriendRequest, error) {
	var request models.FriendRequest
	err := r.db.WithContext(ctx).
		Where("(requester_user_id = ? AND recipient_user_id = ?) OR (requester_user_id = ? AND recipient_user_id = ?)", userID1, userID2, userID2, userID1).
		Where("status = ?", models.FriendRequestStatusPending).
		First(&request).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No pending request found is not an error in this context
		}
		return nil, err // Other database error
	}
	return &request, nil
}

// TODO: Implement other methods as needed: GetRequestByID, UpdateRequestStatus, GetPendingRequestsForUser
func (r *gormFriendRequestRepository) GetRequestByID(ctx context.Context, requestID uint) (*models.FriendRequest, error) {
	var request models.FriendRequest
	err := r.db.WithContext(ctx).First(&request, requestID).Error
	return &request, err
}

func (r *gormFriendRequestRepository) UpdateRequestStatus(ctx context.Context, requestID uint, status models.FriendRequestStatus) error {
	return r.db.WithContext(ctx).Model(&models.FriendRequest{}).Where("id = ?", requestID).Update("status", status).Error
}

func (r *gormFriendRequestRepository) GetPendingRequestsForUser(ctx context.Context, recipientUserID uint) ([]models.FriendRequest, error) {
	var requests []models.FriendRequest
	err := r.db.WithContext(ctx).Where("recipient_user_id = ? AND status = ?", recipientUserID, models.FriendRequestStatusPending).Find(&requests).Error
	return requests, err
}
