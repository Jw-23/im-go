package kafka

import (
	"context"
	"fmt"
	"im-go/internal/config"
	"log"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// MessageHandler is a function type for processing consumed Kafka messages.
// It now uses kafka.Message from confluent-kafka-go.
type MessageHandler func(ctx context.Context, msg *kafka.Message) error

// MessageConsumer defines the interface for a Kafka message consumer.
type MessageConsumer interface {
	Consume(ctx context.Context, topics []string, groupID string, handler MessageHandler) error
	Close()
}

// confluentKafkaConsumer is an implementation of MessageConsumer using confluent-kafka-go.
type confluentKafkaConsumer struct {
	consumer *kafka.Consumer
	cfg      config.KafkaConfig
	groupID  string // Store groupID for logging and potential re-use
}

// NewConfluentKafkaConsumer creates a new Kafka consumer instance using confluent-kafka-go.
func NewConfluentKafkaConsumer(cfg config.KafkaConfig) (MessageConsumer, error) {
	// Note: GroupID will be set in the Consume method as per the original design,
	// or it can be part of the initial configuration if it's static.
	// For now, we prepare a nil consumer, to be configured in Consume.
	// Alternatively, if groupID is known at construction, consumer could be created here.
	return &confluentKafkaConsumer{cfg: cfg}, nil
}

// Consume starts consuming messages from the specified topics and group.
// This method will block until the context is canceled or a fatal error occurs.
func (c *confluentKafkaConsumer) Consume(ctx context.Context, topics []string, groupID string, handler MessageHandler) error {
	if len(topics) == 0 {
		return fmt.Errorf("kafka consumer: no topics specified")
	}
	c.groupID = groupID

	configMap := &kafka.ConfigMap{
		"bootstrap.servers":  strings.Join(c.cfg.Brokers, ","),
		"group.id":           c.groupID,
		"auto.offset.reset":  "earliest", // Process messages from the beginning if no offset is stored
		"enable.auto.commit": "false",    // We will commit manually after processing
		"security.protocol":  c.cfg.Protocol,
		// Add other necessary configurations from c.cfg if needed
		// e.g., security.protocol, sasl.mechanisms, sasl.username, sasl.password
		// "debug": "consumer,cgrp,topic,fetch", // Uncomment for verbose debugging from librdkafka
	}
	if c.cfg.ClientID != "" {
		_ = configMap.SetKey("client.id", c.cfg.ClientID)
	}

	consumer, err := kafka.NewConsumer(configMap)
	if err != nil {
		return fmt.Errorf("failed to create Kafka consumer for group %s: %w", groupID, err)
	}
	c.consumer = consumer
	// defer c.Close() // Close should be called by the owner of the consumer instance, typically where it's created.

	err = c.consumer.SubscribeTopics(topics, nil)
	if err != nil {
		// If subscribe fails, close the consumer before returning the error.
		_ = c.consumer.Close() // Best effort close
		return fmt.Errorf("failed to subscribe to topics %v for group %s: %w", topics, groupID, err)
	}

	log.Printf("Kafka consumer started for GroupID: %s, subscribed to Topics: %v. Waiting for messages...", groupID, topics)

	run := true
	for run {
		select {
		case <-ctx.Done(): // Context cancellation
			log.Printf("Context canceled for consumer group %s. Shutting down.", groupID)
			run = false
		default:
			ev := c.consumer.Poll(1000) // Poll for 1 second
			if ev == nil {
				continue // Timeout, poll again
			}

			switch e := ev.(type) {
			case *kafka.Message:
				if err := handler(ctx, e); err != nil {
					log.Printf("Error processing Kafka message for group %s (Topic: %s, Offset: %v): %v",
						groupID, *e.TopicPartition.Topic, e.TopicPartition.Offset, err)
				} else {
					if _, err := c.consumer.CommitMessage(e); err != nil {
						log.Printf("Failed to commit offset for group %s (Topic: %s, Offset: %v): %v",
							groupID, *e.TopicPartition.Topic, e.TopicPartition.Offset, err)
					}
				}
			case kafka.Error:
				log.Printf("Kafka consumer error for group %s: %v (Code: %d, Fatal: %t, Retriable: %t, TxnRequiresAbort: %t)", groupID, e, e.Code(), e.IsFatal(), e.IsRetriable(), e.TxnRequiresAbort())
				if e.IsFatal() {
					log.Printf("FATAL Kafka error for group %s: %v. Shutting down consumer loop.", groupID, e)
					run = false
					return e
				}
			case kafka.PartitionEOF:
				// log.Printf("Reached EOF for %s [%d] at offset %d for group %s\n", *e.Topic, e.Partition, e.Offset, groupID)
			case kafka.AssignedPartitions:
				log.Printf("Partitions assigned for group %s: %v", groupID, e.Partitions)
				c.consumer.Assign(e.Partitions)
			case kafka.RevokedPartitions:
				log.Printf("Partitions revoked for group %s: %v", groupID, e.Partitions)
				c.consumer.Unassign()
			default:
				// log.Printf("Ignored Kafka event for group %s: %v\n", groupID, e)
			}
		}
	}
	log.Printf("Kafka consumer loop for group %s finished.", groupID)
	return nil
}

// Close closes the Kafka consumer.
func (c *confluentKafkaConsumer) Close() {
	if c.consumer != nil {
		log.Printf("Closing Kafka consumer for group %s...", c.groupID)
		if err := c.consumer.Close(); err != nil {
			log.Printf("Error closing Kafka consumer for group %s: %v", c.groupID, err)
		} else {
			log.Printf("Kafka consumer for group %s closed.", c.groupID)
		}
		c.consumer = nil
	}
}
