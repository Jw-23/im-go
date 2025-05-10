package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"im-go/internal/config"
	"im-go/internal/kafka"
	"im-go/internal/models"
	"im-go/internal/storage"
	"log"
	"time"

	confluentKafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"gorm.io/gorm" // Import gorm for error checking
)

var (
	ErrFriendRequestSelf     = errors.New("不能添加自己为好友")
	ErrFriendRequestExists   = errors.New("已存在待处理的好友请求")
	ErrRecipientNotFound     = errors.New("接收用户不存在")
	ErrAlreadyFriends        = errors.New("你们已经是好友了")
	ErrFriendRequestInvalid  = errors.New("无效的好友请求")
	ErrFriendRequestNotFound = errors.New("好友请求不存在")
	ErrNotRecipientOfRequest = errors.New("您不是此好友请求的接收者")
	ErrRequestNotPending     = errors.New("该好友请求不是待处理状态")
	ErrFriendshipExists      = errors.New("好友关系已存在")
)

// FriendRequestEvent defines the structure for Kafka messages related to friend requests.
type FriendRequestEvent struct {
	RequesterUserID uint      `json:"requesterUserId"`
	RecipientUserID uint      `json:"recipientUserId"`
	Timestamp       time.Time `json:"timestamp"`
	// 可以添加 RequestMessage 等其他字段
}

// FriendRequestService defines the interface for friend request operations.
type FriendRequestService interface {
	SendFriendRequest(ctx context.Context, requesterID, recipientID uint) error
	ProcessFriendRequest(ctx context.Context, kafkaMsg *confluentKafka.Message) error
	AcceptFriendRequest(ctx context.Context, recipientUserID uint, requestID uint) error
	RejectFriendRequest(ctx context.Context, recipientUserID uint, requestID uint) error
	ListPendingRequests(ctx context.Context, userID uint) ([]*models.FriendRequestWithRequester, error)
	GetFriendsList(ctx context.Context, userID uint) ([]*models.UserBasicInfo, error)
	// TODO: Add RejectFriendRequest, ListPendingRequests etc.
}

// FriendRequestWithRequester is a DTO that includes friend request details along with requester info.
// This is useful for API responses.
type FriendRequestWithRequester struct {
	models.FriendRequest
	Requester *models.UserBasicInfo `json:"requester"`
}

type friendRequestService struct {
	db             *gorm.DB // Added for transaction support
	userRepo       storage.UserRepository
	friendRepo     storage.FriendRequestRepository
	friendshipRepo storage.FriendshipRepository // Added
	producer       kafka.MessageProducer
	kafkaConfig    config.KafkaConfig
}

// NewFriendRequestService creates a new FriendRequestService instance.
func NewFriendRequestService(
	db *gorm.DB, // Added
	userRepo storage.UserRepository,
	friendRepo storage.FriendRequestRepository,
	friendshipRepo storage.FriendshipRepository, // Added
	producer kafka.MessageProducer,
	cfg config.KafkaConfig,
) FriendRequestService {
	return &friendRequestService{
		db:             db,
		userRepo:       userRepo,
		friendRepo:     friendRepo,
		friendshipRepo: friendshipRepo,
		producer:       producer,
		kafkaConfig:    cfg,
	}
}

// SendFriendRequest validates the request and publishes an event to Kafka.
func (s *friendRequestService) SendFriendRequest(ctx context.Context, requesterID, recipientID uint) error {
	if requesterID == recipientID {
		return ErrFriendRequestSelf
	}

	// 1. Check if recipient exists
	_, err := s.userRepo.GetByID(ctx, recipientID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRecipientNotFound
		}
		log.Printf("Error checking recipient user %d: %v", recipientID, err)
		return fmt.Errorf("检查接收用户时出错: %w", err)
	}

	// 2. Check if users are already friends
	areFriends, err := s.friendshipRepo.AreUsersFriends(ctx, requesterID, recipientID)
	if err != nil {
		log.Printf("Error checking if users %d and %d are already friends: %v", requesterID, recipientID, err)
		return fmt.Errorf("检查好友关系时出错: %w", err)
	}
	if areFriends {
		return ErrAlreadyFriends
	}

	// 3. Check if a pending request already exists (in either direction)
	existingRequest, err := s.friendRepo.FindPendingRequest(ctx, requesterID, recipientID)
	if err != nil {
		log.Printf("Error checking existing friend request between %d and %d: %v", requesterID, recipientID, err)
		return fmt.Errorf("检查现有请求时出错: %w", err)
	}
	if existingRequest != nil {
		return ErrFriendRequestExists
	}

	// 4. Prepare Kafka message event and payload
	event := FriendRequestEvent{
		RequesterUserID: requesterID,
		RecipientUserID: recipientID,
		Timestamp:       time.Now(),
	}
	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshalling friend request event for Kafka: %v", err)
		return fmt.Errorf("序列化好友请求事件失败: %w", err)
	}

	// 5. Publish to Kafka
	topic := s.kafkaConfig.FriendRequestTopic
	key := []byte(fmt.Sprintf("%d-%d", requesterID, recipientID))

	err = s.producer.SendMessage(ctx, topic, key, payload)
	if err != nil {
		log.Printf("Error producing friend request event to Kafka topic %s: %v", topic, err)
		return fmt.Errorf("发送好友请求到处理队列失败: %w", err)
	}

	log.Printf("Friend request event published to topic %s for %d -> %d", topic, requesterID, recipientID)
	return nil
}

// ProcessFriendRequest handles incoming friend request events from Kafka.
func (s *friendRequestService) ProcessFriendRequest(ctx context.Context, kafkaMsg *confluentKafka.Message) error {
	log.Printf("Processing friend request event from Kafka, offset %d", kafkaMsg.TopicPartition.Offset)
	var event FriendRequestEvent
	if err := json.Unmarshal(kafkaMsg.Value, &event); err != nil {
		log.Printf("Error unmarshalling friend request event from Kafka: %v, value: %s", err, string(kafkaMsg.Value))
		return nil // Commit offset for bad message
	}

	// Check if users are already friends before creating request (idempotency for retries)
	areFriends, err := s.friendshipRepo.AreUsersFriends(ctx, event.RequesterUserID, event.RecipientUserID)
	if err != nil {
		log.Printf("Error checking friendship status for %d and %d in ProcessFriendRequest: %v", event.RequesterUserID, event.RecipientUserID, err)
		return err // Retryable
	}
	if areFriends {
		log.Printf("Users %d and %d are already friends, skipping friend request creation.", event.RequesterUserID, event.RecipientUserID)
		return nil // Commit offset
	}

	// Double-check if request already exists
	existing, err := s.friendRepo.FindPendingRequest(ctx, event.RequesterUserID, event.RecipientUserID)
	if err != nil {
		log.Printf("Error re-checking friend request before creation (%d -> %d): %v", event.RequesterUserID, event.RecipientUserID, err)
		return err // Retryable
	}
	if existing != nil {
		log.Printf("Friend request already processed or exists (%d -> %d), skipping creation.", event.RequesterUserID, event.RecipientUserID)
		return nil // Commit offset
	}

	request := models.FriendRequest{
		RequesterUserID: event.RequesterUserID,
		RecipientUserID: event.RecipientUserID,
		Status:          models.FriendRequestStatusPending,
	}

	if err := s.friendRepo.Create(ctx, &request); err != nil {
		log.Printf("Error saving friend request (%d -> %d) to database: %v", event.RequesterUserID, event.RecipientUserID, err)
		return err // Retryable
	}

	log.Printf("Friend request from %d to %d saved successfully with ID %d", event.RequesterUserID, event.RecipientUserID, request.ID)
	// TODO: Trigger real-time notification to recipientUserID about the new request
	return nil
}

// AcceptFriendRequest processes the acceptance of a friend request.
func (s *friendRequestService) AcceptFriendRequest(ctx context.Context, recipientUserID uint, requestID uint) error {
	// Use a transaction to ensure atomicity
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get the specific friend request repository instance for this transaction
		txFriendRepo := storage.NewGormFriendRequestRepository(tx)
		txFriendshipRepo := storage.NewGormFriendshipRepository(tx)

		// 1. Retrieve the friend request
		request, err := txFriendRepo.GetRequestByID(ctx, requestID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrFriendRequestNotFound
			}
			log.Printf("Error retrieving friend request %d: %v", requestID, err)
			return fmt.Errorf("检索好友请求失败: %w", err)
		}

		// 2. Validate the request
		if request.RecipientUserID != recipientUserID {
			return ErrNotRecipientOfRequest
		}
		if request.Status != models.FriendRequestStatusPending {
			return ErrRequestNotPending
		}

		// 3. Check if they are already friends (should not happen if logic is correct, but good check)
		areFriends, err := txFriendshipRepo.AreUsersFriends(ctx, request.RequesterUserID, request.RecipientUserID)
		if err != nil {
			log.Printf("Error checking friendship in AcceptFriendRequest for users %d, %d: %v", request.RequesterUserID, request.RecipientUserID, err)
			return fmt.Errorf("检查好友关系时出错: %w", err)
		}
		if areFriends {
			// If already friends, perhaps just mark request as accepted and log warning
			log.Printf("Warning: Users %d and %d are already friends, but request %d was pending. Marking as accepted.", request.RequesterUserID, request.RecipientUserID, requestID)
			// Still proceed to update status for idempotency, but don't create friendship again.
			// return ErrFriendshipExists // Or just continue to update status
		}

		// 4. Update friend request status to accepted
		if err := txFriendRepo.UpdateRequestStatus(ctx, requestID, models.FriendRequestStatusAccepted); err != nil {
			log.Printf("Error updating friend request %d status to accepted: %v", requestID, err)
			return fmt.Errorf("更新好友请求状态失败: %w", err)
		}

		// 5. Create friendship record (only if not already friends)
		if !areFriends {
			friendship := &models.Friendship{
				UserID1: request.RequesterUserID,
				UserID2: request.RecipientUserID,
			}
			friendship.EnsureCanonicalOrder() // Ensure UserID1 < UserID2
			if err := txFriendshipRepo.Create(ctx, friendship); err != nil {
				log.Printf("Error creating friendship for users %d and %d: %v", request.RequesterUserID, request.RecipientUserID, err)
				return fmt.Errorf("创建好友关系失败: %w", err)
			}
			log.Printf("Friendship created between %d and %d from request %d", request.RequesterUserID, request.RecipientUserID, requestID)
		} else {
			log.Printf("Skipped creating friendship for request %d as it already exists between %d and %d", requestID, request.RequesterUserID, request.RecipientUserID)
		}

		// TODO: Create a new private conversation for these users if one doesn't exist.
		// TODO: Send real-time notification to the requesterUserID that their request was accepted.

		return nil // Commit transaction
	})

	if txErr != nil {
		return txErr // Return the error from the transaction
	}

	log.Printf("Friend request %d accepted successfully by user %d for requester %d.", requestID, recipientUserID, 0) // RequesterID not directly available here without re-fetch
	return nil
}

// ListPendingRequests retrieves all pending friend requests for a given user.
func (s *friendRequestService) ListPendingRequests(ctx context.Context, userID uint) ([]*models.FriendRequestWithRequester, error) {
	pendingRequests, err := s.friendRepo.GetPendingRequestsForUser(ctx, userID)
	if err != nil {
		log.Printf("Error fetching pending friend requests for user %d: %v", userID, err)
		return nil, fmt.Errorf("获取待处理好友请求失败: %w", err)
	}

	if len(pendingRequests) == 0 {
		return []*models.FriendRequestWithRequester{}, nil
	}

	// Enrich with requester info
	var resultDTOs []*models.FriendRequestWithRequester
	for _, req := range pendingRequests {
		requester, err := s.userRepo.GetBasicInfoByID(ctx, req.RequesterUserID)
		if err != nil {
			log.Printf("Error fetching requester info for user %d (request %d): %v", req.RequesterUserID, req.ID, err)
			continue
		}
		resultDTOs = append(resultDTOs, &models.FriendRequestWithRequester{
			FriendRequest: req,
			Requester:     requester,
		})
	}
	return resultDTOs, nil
}

// RejectFriendRequest processes the rejection of a friend request.
func (s *friendRequestService) RejectFriendRequest(ctx context.Context, recipientUserID uint, requestID uint) error {
	// 1. Retrieve the friend request
	request, err := s.friendRepo.GetRequestByID(ctx, requestID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrFriendRequestNotFound
		}
		log.Printf("Error retrieving friend request %d for rejection: %v", requestID, err)
		return fmt.Errorf("检索好友请求失败: %w", err)
	}

	// 2. Validate the request
	if request.RecipientUserID != recipientUserID {
		return ErrNotRecipientOfRequest
	}
	if request.Status != models.FriendRequestStatusPending {
		// Allow rejecting already accepted/rejected requests? For now, only pending.
		return ErrRequestNotPending
	}

	// 3. Update friend request status to rejected
	if err := s.friendRepo.UpdateRequestStatus(ctx, requestID, models.FriendRequestStatusRejected); err != nil {
		log.Printf("Error updating friend request %d status to rejected: %v", requestID, err)
		return fmt.Errorf("更新好友请求状态为已拒绝失败: %w", err)
	}

	log.Printf("Friend request %d rejected by user %d.", requestID, recipientUserID)
	// TODO: Send real-time notification to the requesterUserID that their request was rejected.
	return nil
}

// GetFriendsList retrieves the basic info for all friends of the given user.
func (s *friendRequestService) GetFriendsList(ctx context.Context, userID uint) ([]*models.UserBasicInfo, error) {
	// 1. Get the IDs of all friends
	friendIDs, err := s.friendshipRepo.GetFriendIDs(ctx, userID)
	if err != nil {
		log.Printf("Error getting friend IDs for user %d: %v", userID, err)
		return nil, fmt.Errorf("获取好友列表失败: %w", err)
	}

	if len(friendIDs) == 0 {
		return []*models.UserBasicInfo{}, nil // Return empty list if no friends
	}

	// 2. Get the basic info for those friend IDs
	friendsInfo, err := s.userRepo.GetMultipleBasicInfoByIDs(ctx, friendIDs)
	if err != nil {
		log.Printf("Error getting basic info for friend IDs of user %d: %v", userID, err)
		return nil, fmt.Errorf("获取好友信息失败: %w", err)
	}

	return friendsInfo, nil
}
