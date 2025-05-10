package kafka

import (
	"context"
	"fmt"
	"im-go/internal/config"
	"log"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// MessageProducer defines the interface for a Kafka message producer.
// SendMessage now takes a topic, key, and payload directly for simplicity.
// Context is included for potential future use (e.g., tracing).
type MessageProducer interface {
	SendMessage(ctx context.Context, topic string, key []byte, payload []byte) error
	Close()
}

// confluentKafkaProducer is an implementation of MessageProducer using confluent-kafka-go.
type confluentKafkaProducer struct {
	producer *kafka.Producer
	cfg      config.KafkaConfig
}

// NewConfluentKafkaProducer creates a new Kafka producer instance using confluent-kafka-go.
func NewConfluentKafkaProducer(cfg config.KafkaConfig) (MessageProducer, error) {
	configMap := &kafka.ConfigMap{
		"bootstrap.servers": strings.Join(cfg.Brokers, ","),
		"security.protocol": cfg.Protocol,
		// "acks": "all", // Example: For stronger delivery guarantees
		// Add other necessary configurations from cfg if needed
		// e.g., security.protocol, sasl.mechanisms, etc.
		// "debug": "producer,msg,broker", // Uncomment for verbose debugging
	}
	if cfg.ClientID != "" {
		_ = configMap.SetKey("client.id", cfg.ClientID)
	}

	p, err := kafka.NewProducer(configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	// Optional: Start a goroutine to handle delivery reports if you want to process them globally.
	// For this example, SendMessage will handle delivery reports synchronously.
	// go func() {
	// 	for e := range p.Events() {
	// 		switch ev := e.(type) {
	// 		case *kafka.Message:
	// 			if ev.TopicPartition.Error != nil {
	// 				log.Printf("Producer: Delivery failed for message to %s: %v", *ev.TopicPartition.Topic, ev.TopicPartition.Error)
	// 			} else {
	// 				// log.Printf("Producer: Delivered message to %s [%d] at offset %v",
	// 				// 	*ev.TopicPartition.Topic, ev.TopicPartition.Partition, ev.TopicPartition.Offset)
	// 			}
	// 		case kafka.Error:
	// 			log.Printf("Producer: Kafka error: %v", ev)
	// 		default:
	// 			// log.Printf("Producer: Ignored event: %s", ev)
	// 		}
	// 	}
	// 	log.Println("Producer event handler goroutine stopped.")
	// }()

	return &confluentKafkaProducer{producer: p, cfg: cfg}, nil
}

// SendMessage sends a single message to the specified Kafka topic.
// This implementation waits for the delivery report synchronously.
func (p *confluentKafkaProducer) SendMessage(ctx context.Context, topic string, key []byte, payload []byte) error {
	deliveryChan := make(chan kafka.Event, 1) // Buffered channel to avoid blocking if main goroutine isn't reading fast enough
	defer close(deliveryChan)

	kafkaMsg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Key:            key,
		Value:          payload,
		Timestamp:      time.Now(),
	}

	err := p.producer.Produce(kafkaMsg, deliveryChan)
	if err != nil {
		// This error is typically for local issues like the producer queue being full.
		// Actual delivery errors are reported via the deliveryChan.
		return fmt.Errorf("kafka producer failed to enqueue message for topic %s: %w", topic, err)
	}

	// Wait for the delivery report
	// A timeout can be added here using a select with ctx.Done() or time.After()
	select {
	case e := <-deliveryChan:
		m, ok := e.(*kafka.Message)
		if !ok {
			// This shouldn't happen if Produce was successful and an event was received.
			// It could be another type of event (e.g., kafka.Error) if not filtered by Produce itself.
			return fmt.Errorf("kafka producer: unexpected event type received on delivery channel: %T %v", e, e)
		}
		if m.TopicPartition.Error != nil {
			return fmt.Errorf("kafka producer: delivery failed for topic %s: %w", topic, m.TopicPartition.Error)
		}
		// log.Printf("Producer: Successfully delivered message to topic %s, partition %d, offset %v",
		// 	topic, m.TopicPartition.Partition, m.TopicPartition.Offset)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("kafka producer: context canceled while waiting for delivery report for topic %s: %w", topic, ctx.Err())
		// Consider what to do if context is cancelled: the message might still be delivered or not.
		// The Produce call is async; this timeout is only for the delivery report.
	}
}

// Close flushes any outstanding messages and closes the Kafka producer.
func (p *confluentKafkaProducer) Close() {
	if p.producer != nil {
		log.Println("Closing Kafka producer...")
		// producer.Flush is used to wait for all outstanding messages to be delivered.
		// The timeout specifies how long to wait.
		// A positive value means it will block until all messages are sent or the timeout is reached.
		// A value of 0 means non-blocking check.
		// A negative value means block indefinitely.
		remaining := p.producer.Flush(15 * 1000) // 15 second timeout
		if remaining > 0 {
			log.Printf("Warning: %d messages still outstanding after flush, producer closing.", remaining)
		} else {
			log.Println("All messages flushed successfully.")
		}
		p.producer.Close() // This closes the producer instance and releases resources.
		log.Println("Kafka producer closed.")
	}
}
