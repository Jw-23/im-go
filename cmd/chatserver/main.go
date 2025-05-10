package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"im-go/internal/imtypes"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	confluentKafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"

	"im-go/internal/config"
	"im-go/internal/handlers/chatserver"
	appKafka "im-go/internal/kafka"
	"im-go/internal/services"
	"im-go/internal/storage"
	"im-go/internal/websocket"
)

// Kafka配置
const (
	kafkaBroker = "localhost:9092"
	kafkaTopic  = "chat-messages"
)

func main() {
	// 1. 加载配置
	cfg, err := config.LoadConfig("")
	if err != nil {
		log.Fatalf("无法加载配置: %v", err)
	}
	log.Println("Chat 服务器配置加载成功。")

	// 2. 初始化数据库连接
	db, err := storage.InitDB(cfg.Database)
	if err != nil {
		log.Fatalf("无法初始化数据库: %v", err)
	}
	log.Println("Chat 服务器数据库连接成功。")

	// 3. 自动迁移数据库表结构 (通常一个服务实例负责即可)
	if err := storage.AutoMigrateTables(db); err != nil {
		log.Fatalf("无法迁移数据库表: %v", err)
	}
	log.Println("Chat 服务器数据库表迁移成功。")

	// 4. 初始化 Kafka Producer
	kfkProducer, err := appKafka.NewConfluentKafkaProducer(cfg.Kafka)

	if err != nil {
		log.Fatalf("无法创建 Kafka 生产者: %v", err)
	}
	defer kfkProducer.Close()
	log.Println("Kafka 生产者初始化成功 (ChatServer)。")

	// 5. 初始化 Repositories (MessageService 需要)
	msgRepo := storage.NewGormMessageRepository(db)
	convoRepo := storage.NewGormConversationRepository(db)
	userRepo := storage.NewGormUserRepository(db) // UserService 可能被 WebSocketHandler 使用

	// 6. 初始化 Services
	// ChatServer 主要关注 MessageService，其他服务按需添加
	messageService := services.NewMessageService(msgRepo, convoRepo, kfkProducer, cfg)
	userService := services.NewUserService(userRepo) // WebSocketHandler 可能用它来获取用户信息

	// 7. 初始化 WebSocket Hub
	hub := websocket.NewHub()
	go hub.Run() // 在 goroutine 中运行 Hub
	log.Println("WebSocket Hub 已启动。")

	// 8. 初始化 WebSocket Handler
	wsHandler := chatserver.NewWebSocketHandler(hub, messageService, userService, cfg)

	// 9. 初始化 Kafka 消费者 (用于处理入站消息)
	inboundConsumer, err := appKafka.NewConfluentKafkaConsumer(cfg.Kafka)
	if err != nil {
		log.Fatalf("无法创建入站 Kafka 消费者: %v", err)
	}
	defer inboundConsumer.Close()

	// 9.1 初始化 Kafka 消费者 (用于处理出站 WebSocket 消息)
	outboundConsumer, err := appKafka.NewConfluentKafkaConsumer(cfg.Kafka)
	if err != nil {
		log.Fatalf("无法创建出站 Kafka 消费者: %v", err)
	}
	defer outboundConsumer.Close()

	// 为 Kafka 消费者创建可以取消的上下文
	consumerCtx, cancelConsumers := context.WithCancel(context.Background())
	defer cancelConsumers()

	// 9.2 启动入站消息消费者 Goroutine
	go func() {
		log.Printf("Kafka 入站消费者 goroutine 启动，监听 topic: %s", cfg.Kafka.MessagesTopic)
		topicsToConsume := []string{cfg.Kafka.MessagesTopic}
		if err := inboundConsumer.Consume(consumerCtx, topicsToConsume, cfg.Kafka.ConsumerGroup,
			func(ctx context.Context, kafkaMsg *confluentKafka.Message) error {
				// 这个回调调用 MessageService 处理并可能产生出站消息
				return messageService.ProcessKafkaMessage(ctx, kafkaMsg)
			}); err != nil {
			// Don't Fatal here, allow graceful shutdown
			log.Printf("Kafka 入站消费者错误: %v", err)
		}
		log.Println("Kafka 入站消费者 goroutine 已停止。")
	}()

	// 9.3 启动出站消息消费者 Goroutine
	go func() {
		log.Printf("Kafka 出站消费者 goroutine 启动，监听 topic: %s", cfg.Kafka.WebSocketOutgoingTopic)
		topicsToConsume := []string{cfg.Kafka.WebSocketOutgoingTopic}
		// 注意：这里的 Consumer Group 可以与入站的不同，如果需要确保每个 ChatServer 实例都收到所有出站消息
		// 但当前配置共用 ConsumerGroup，意味着消息会被分发到组内的一个消费者实例
		if err := outboundConsumer.Consume(consumerCtx, topicsToConsume, cfg.Kafka.ConsumerGroup,
			func(ctx context.Context, kafkaMsg *confluentKafka.Message) error {
				// 这个回调直接将消息发送给 Hub
				var wsMsg imtypes.Message
				if err := json.Unmarshal(kafkaMsg.Value, &wsMsg); err != nil {
					log.Printf("错误: 无法从 Kafka 反序列化出站 WebSocket 消息: %v, 原始值: %s", err, string(kafkaMsg.Value))
					return nil // Don't stop consumer for one bad message
				}
				// 调用 Hub 的新方法来投递消息
				hub.DeliverDirectMessage(&wsMsg)
				return nil // Assume successful delivery attempt for Kafka commit
			}); err != nil {
			log.Printf("Kafka 出站消费者错误: %v", err)
		}
		log.Println("Kafka 出站消费者 goroutine 已停止。")
	}()

	// 10. 配置 HTTP 服务器路由
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/chat", wsHandler.ServeWS)
	mux.HandleFunc("/ws/chat/", wsHandler.ServeWS)

	// 11. 启动 HTTP 服务器
	serverAddr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	httpServer := &http.Server{Addr: serverAddr, Handler: mux}

	go func() {
		log.Printf("Chat HTTP 服务器启动于 %s, WebSocket 路径: %s", serverAddr, cfg.Server.WebSocketPath)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Chat 服务器启动失败: %v", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Chat 服务器准备关闭...")

	cancelConsumers() // 通知 Kafka 消费者停止
	log.Println("正在等待 Kafka 消费者停止...")
	// TODO: Maybe add a WaitGroup or check channel closure to ensure consumers actually stopped before closing HTTP server?

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := httpServer.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Chat 服务器关闭失败: %v", err)
	}
	log.Println("Chat 服务器已优雅关闭。")
}
