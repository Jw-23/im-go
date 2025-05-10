package kafkahandlers

import (
	"context"
	"encoding/json"
	"im-go/internal/services" // Assuming FriendRequestService is in this package or a subpackage
	"log"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// FriendRequestMessage defines the structure of a friend request message from Kafka.
// This should align with what the producer sends.
type FriendRequestMessage struct {
	RequesterID uint `json:"requesterId"`
	RecipientID uint `json:"recipientId"`
	// Potentially other fields like Message string, Timestamp int64
}

// FriendRequestConsumerLogic encapsulates the business logic for handling friend requests from Kafka.
// It depends on the FriendRequestService to interact with the underlying data store or other services.
type FriendRequestConsumerLogic struct {
	friendService services.FriendRequestService
}

// NewFriendRequestConsumerLogic creates a new instance of FriendRequestConsumerLogic.
func NewFriendRequestConsumerLogic(fs services.FriendRequestService) *FriendRequestConsumerLogic {
	if fs == nil {
		// Or handle this more gracefully, perhaps return an error
		log.Panic("FriendRequestService cannot be nil")
	}
	return &FriendRequestConsumerLogic{friendService: fs}
}

// HandleFriendRequest is the actual MessageHandler function that will be passed to the Kafka consumer.
// It processes a single Kafka message representing a friend request.
// The msg parameter is now *kafka.Message from confluent-kafka-go.
func (h *FriendRequestConsumerLogic) HandleFriendRequest(ctx context.Context, msg *kafka.Message) error {
	log.Printf("Kafka Consumer: Received message for Topic %s, Partition %d, Offset %d, Key: %s\n",
		*msg.TopicPartition.Topic, msg.TopicPartition.Partition, msg.TopicPartition.Offset, string(msg.Key))

	var frMsg FriendRequestMessage
	err := json.Unmarshal(msg.Value, &frMsg)
	if err != nil {
		log.Printf("Error unmarshalling friend request message (Value: '%s'): %v. This message will be skipped.", string(msg.Value), err)
		// Returning nil means the message is considered 'processed' (or skipped) and won't be retried for this reason.
		// If unmarshalling is crucial and retriable, an error could be returned.
		return nil
	}

	log.Printf("Successfully unmarshalled friend request: %+v", frMsg)

	// Here, you would call the method on your friendService to actually process the request.
	// For example, if your service has a method like: ProcessFriendRequest(ctx context.Context, requesterID, recipientID uint) error
	// The call would be:
	// err = h.friendService.ProcessFriendRequest(ctx, frMsg.RequesterID, frMsg.RecipientID)

	// Placeholder for actual service call:
	log.Printf("Simulating processing of friend request for RequesterID: %d, RecipientID: %d", frMsg.RequesterID, frMsg.RecipientID)
	// Simulate some work or a call that might fail
	// if frMsg.RecipientID == 0 { // Example error condition
	// 	err = errors.New("simulated processing error: recipient ID is zero")
	// }

	if err != nil { // This err is from the simulated processing above, ensure it's correctly scoped
		log.Printf("Error processing friend request (RequesterID: %d, RecipientID: %d) after unmarshalling: %v", frMsg.RequesterID, frMsg.RecipientID, err)
		return err
	}

	log.Printf("Successfully processed friend request for RequesterID: %d, RecipientID: %d", frMsg.RequesterID, frMsg.RecipientID)
	// If processing is successful, return nil so the Kafka consumer can commit the offset.
	return nil
}
