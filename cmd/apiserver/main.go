package main

import (
	"context"
	"errors"
	"fmt"
	"im-go/internal/imtypes" // Import for StorageService interface
	appKafka "im-go/internal/kafka"
	"im-go/internal/models"
	"log"
	"net/http"
	"os"
	"os/signal" // Added for stripping prefix
	"path/filepath"
	"strings" // Added for stripping prefix
	"syscall"
	"time"

	"im-go/internal/config"
	"im-go/internal/handlers/apiserver"
	"im-go/internal/middleware" // Needed for FriendRequest
	"im-go/internal/services"
	"im-go/internal/storage"

	appRedis "im-go/internal/redis" // Alias for your internal redis package

	"github.com/gorilla/handlers" // ADDED import
	"github.com/gorilla/mux"
	redisDriver "github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func main() {
	// 1. 加载配置
	cfg, err := config.LoadConfig("")
	if err != nil {
		log.Fatalf("无法加载配置: %v", err)
	}
	log.Println("API 服务器配置加载成功。")

	// 2. 初始化数据库连接
	db, err := storage.InitDB(cfg.Database)
	if err != nil {
		log.Fatalf("无法初始化数据库: %v", err)
	}
	log.Println("API 服务器数据库连接成功。")

	// (可选) 表结构迁移
	if err := storage.AutoMigrateTables(db); err != nil {
		log.Printf("警告：API 服务器数据库表迁移可能失败: %v", err)
	} else {
		log.Println("API 服务器数据库表迁移成功 (如果执行)。")
	}

	// 3. 初始化 Redis Client
	redisClient := redisDriver.NewClient(&redisDriver.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	_, err = redisClient.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("无法连接到 Redis: %v", err)
	}
	log.Println("成功连接到 Redis")

	// 4. 初始化 TokenBlacklist 服务
	tokenBlacklistService := appRedis.NewRedisTokenBlacklist(redisClient)

	// 5. 初始化 Repositories
	userRepo := storage.NewGormUserRepository(db)
	convoRepo := storage.NewGormConversationRepository(db)
	groupRepo := storage.NewGormGroupRepository(db)
	msgRepo := storage.NewGormMessageRepository(db)
	friendReqRepo := storage.NewGormFriendRequestRepository(db)
	friendshipRepo := storage.NewGormFriendshipRepository(db)

	// 6. 初始化 Kafka Producer
	log.Printf("DEBUG [Kafka Init]: Brokers from config: %v", cfg.Kafka.Brokers)
	log.Printf("DEBUG [Kafka Init]: Friend Request Topic from config: %s", cfg.Kafka.FriendRequestTopic)

	kfkProducer, err := appKafka.NewConfluentKafkaProducer(cfg.Kafka)
	if err != nil {
		log.Fatalf("无法创建 Kafka 生产者: %v", err)
	}
	defer kfkProducer.Close()
	log.Println("Kafka 生产者初始化成功 (API Server)。")

	// 7. 初始化 Services
	authService := services.NewAuthService(userRepo, cfg)
	userService := services.NewUserService(userRepo)
	messageService := services.NewMessageService(msgRepo, convoRepo, kfkProducer, cfg)
	conversationService := services.NewConversationService(convoRepo, userRepo)
	groupService := services.NewGroupService(groupRepo, userRepo, convoRepo)
	friendReqService := services.NewFriendRequestService(db, userRepo, friendReqRepo, friendshipRepo, kfkProducer, cfg.Kafka)

	// 7.1 初始化存储服务 (New)
	var storageService imtypes.StorageService // Use interface type from imtypes
	storageBaseURL := "/uploads"              // Base URL for accessing uploaded files
	if cfg.Storage.Type == "local" {
		storageService, err = storage.NewLocalStorageService(cfg.Storage, storageBaseURL)
		if err != nil {
			log.Fatalf("无法初始化本地存储服务: %v", err)
		}
		log.Println("本地存储服务初始化成功。")
	} else if cfg.Storage.Type == "s3" {
		// TODO: Initialize S3 storage service
		log.Fatalf("S3 存储服务尚未实现")
	} else {
		log.Fatalf("不支持的存储类型: %s", cfg.Storage.Type)
	}

	// 8. 初始化 Handlers
	authHandler := apiserver.NewAuthHandler(authService, tokenBlacklistService)
	userHandler := apiserver.NewUserHandler(userService)
	convoHandler := apiserver.NewConversationHandler(conversationService, messageService, groupService)
	groupHandler := apiserver.NewGroupHandler(groupService, conversationService)
	uploadHandler := apiserver.NewUploadHandler(storageService, cfg.Storage)
	friendReqHandler := apiserver.NewFriendRequestHandler(friendReqService)

	// 9. 设置 HTTP 路由
	r := mux.NewRouter()

	// 9.1 认证路由
	authRouter := r.PathPrefix("/auth").Subrouter()
	authRouter.HandleFunc("/register", authHandler.Register).Methods(http.MethodPost)
	authRouter.HandleFunc("/login", authHandler.Login).Methods(http.MethodPost)

	// 创建 AuthMiddleware 实例
	authMW := middleware.AuthMiddleware(cfg.Auth.JWTSecretKey, tokenBlacklistService)

	// 7.2 API 子路由 (需要认证)
	apiRouter := r.PathPrefix("/api/v1").Subrouter()
	apiRouter.Use(authMW) // 应用创建的认证中间件

	// 将登出路由移到受保护的 apiRouter 下，因为它需要认证来获取 JTI
	// 确保 authHandler 已经初始化并且 LogoutHandler 可以工作
	// 如果 AuthHandler 自身需要 tokenBlacklistService (例如在 NewAuthHandler 中注入)，请确保已完成
	apiRouter.HandleFunc("/auth/logout", authHandler.LogoutHandler).Methods(http.MethodPost)

	// 用户路由
	apiRouter.HandleFunc("/users/me", userHandler.GetMyProfileHandler).Methods(http.MethodGet)
	apiRouter.HandleFunc("/users/me", userHandler.UpdateMyProfileHandler).Methods(http.MethodPut)
	apiRouter.HandleFunc("/users/search", userHandler.SearchUsersHandler).Methods(http.MethodGet)
	// 联系人/好友路由 (ADDED)
	apiRouter.HandleFunc("/friends", friendReqHandler.ListFriendsHandler).Methods(http.MethodGet)
	// 会话路由
	apiRouter.HandleFunc("/conversations", convoHandler.GetUserConversationsHandler).Methods(http.MethodGet)
	apiRouter.HandleFunc("/conversations/private", convoHandler.CreateOrGetPrivateConversationHandler).Methods(http.MethodPost)
	apiRouter.HandleFunc("/conversations/{conversationID:[0-9]+}/messages", convoHandler.GetConversationMessagesHandler).Methods(http.MethodGet)
	// 群组路由
	apiRouter.HandleFunc("/groups", groupHandler.CreateGroupHandler).Methods(http.MethodPost)
	apiRouter.HandleFunc("/groups/{groupID:[0-9]+}/join", groupHandler.JoinGroupHandler).Methods(http.MethodPost)
	apiRouter.HandleFunc("/groups/{groupID:[0-9]+}/leave", groupHandler.LeaveGroupHandler).Methods(http.MethodPost)
	apiRouter.HandleFunc("/groups/{groupID:[0-9]+}/members", groupHandler.GetGroupMembersHandler).Methods(http.MethodGet)
	apiRouter.HandleFunc("/groups/{id:[0-9]+}/fix-participants", groupHandler.FixGroupConversationParticipants).Methods(http.MethodPost)
	// 文件上传路由 (New)
	apiRouter.HandleFunc("/upload", uploadHandler.UploadFileHandler).Methods(http.MethodPost)

	// 好友请求路由
	friendRequestRouter := apiRouter.PathPrefix("/friend-requests").Subrouter() // Create subrouter for friend requests
	friendRequestRouter.HandleFunc("", friendReqHandler.SendFriendRequestHandler).Methods(http.MethodPost)
	friendRequestRouter.HandleFunc("/pending", friendReqHandler.ListPendingRequestsHandler).Methods(http.MethodGet)
	friendRequestRouter.HandleFunc("/{requestID:[0-9]+}/accept", friendReqHandler.AcceptFriendRequestHandler).Methods(http.MethodPost)
	friendRequestRouter.HandleFunc("/{requestID:[0-9]+}/reject", friendReqHandler.RejectFriendRequestHandler).Methods(http.MethodPost)

	// 7.3 公开路由 (不需要认证)
	// 获取其他用户公开信息
	r.HandleFunc("/users/{userID:[0-9]+}", userHandler.GetUserProfileHandler).Methods(http.MethodGet)
	// 搜索公开群组
	r.HandleFunc("/groups/search", groupHandler.SearchPublicGroupsHandler).Methods(http.MethodGet)
	// 获取公开群组详情
	r.HandleFunc("/groups/{groupID:[0-9]+}", groupHandler.GetGroupDetailsHandler).Methods(http.MethodGet)

	// 7.4 静态文件服务路由 (New) - 用于访问上传的文件
	if cfg.Storage.Type == "local" {
		staticPath := strings.TrimSuffix(storageBaseURL, "/") + "/"
		localDir := http.Dir(cfg.Storage.LocalPath)
		r.PathPrefix(staticPath).Handler(http.StripPrefix(staticPath, http.FileServer(localDir)))
		log.Printf("提供静态文件服务于 %s -> %s", staticPath, cfg.Storage.LocalPath)
	}

	// 7.5 静态文件服务路由 (前端应用) - ADDED
	frontendBuildPath := "./site/dist" // ADJUST THIS to where your frontend build output is
	frontendFileServer := http.FileServer(http.Dir(frontendBuildPath))
	// Serve static files like CSS, JS, images etc. from /app/static/
	// The URL path /app/static/ will be stripped, and the rest will be looked up in frontendBuildPath.
	// Example: /app/static/css/main.css -> frontendBuildPath/css/main.css
	// You might need to adjust your frontend's base path or asset paths if they don't align.
	// A common pattern is to serve assets directly from paths like /assets/ or /static/ relative to index.html
	// For now, let's assume assets are referenced relative to index.html in the frontendBuildPath.
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", frontendFileServer)) // Or another path like /assets/

	// Serve index.html for the root path and any other unhandled paths (for SPA routing)
	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the requested file exists in the static directory
		// This prevents serving index.html for actual static files that might be requested without the /static/ prefix
		potentialFilePath := filepath.Join(frontendBuildPath, r.URL.Path)
		// Ensure the path is clean and doesn't try to escape the frontendBuildPath (though http.Dir should handle this)
		potentialFilePath = filepath.Clean(potentialFilePath)
		if strings.HasPrefix(potentialFilePath, filepath.Clean(frontendBuildPath)) {
			fileInfo, err := os.Stat(potentialFilePath)
			if err == nil && !fileInfo.IsDir() {
				http.ServeFile(w, r, potentialFilePath)
				return
			}
		}
		// If not an existing file, or it's a directory, or any other case, serve index.html
		http.ServeFile(w, r, filepath.Join(frontendBuildPath, "index.html"))
	})
	log.Printf("为前端应用提供静态文件服务，根路径指向 %s/index.html", frontendBuildPath)

	// 8. 初始化并启动 Kafka 消费者 (用于处理 FriendRequest)
	friendReqConsumer, err := appKafka.NewConfluentKafkaConsumer(cfg.Kafka)
	if err != nil {
		log.Fatalf("无法创建好友请求 Kafka 消费者: %v", err)
	}
	defer friendReqConsumer.Close()

	consumerCtx, cancelConsumers := context.WithCancel(context.Background())
	defer cancelConsumers()

	go func() {
		topics := []string{cfg.Kafka.FriendRequestTopic}
		log.Printf("Kafka 好友请求消费者启动，监听 topic: %s, GroupID: %s", cfg.Kafka.FriendRequestTopic, cfg.Kafka.ConsumerGroup)
		err := friendReqConsumer.Consume(consumerCtx, topics, cfg.Kafka.ConsumerGroup, friendReqService.ProcessFriendRequest)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("Kafka 好友请求消费者错误: %v", err)
		}
		log.Println("Kafka 好友请求消费者 goroutine 已停止。")
	}()

	// 添加修复命令
	if len(os.Args) > 1 && os.Args[1] == "fix-group-conversations" {
		fixGroupConversations(db)
		return
	}

	// 9. 启动 HTTP 服务器并实现优雅关闭
	serverAddr := fmt.Sprintf("%s:%s", cfg.APIServer.Host, cfg.APIServer.Port)

	// 定义 CORS 选项，从配置中读取 (MODIFIED)
	allowedOrigins := handlers.AllowedOrigins(cfg.APIServer.CORS.AllowedOrigins)
	allowedMethods := handlers.AllowedMethods(cfg.APIServer.CORS.AllowedMethods)
	allowedHeaders := handlers.AllowedHeaders(cfg.APIServer.CORS.AllowedHeaders)
	exposedHeaders := handlers.ExposedHeaders(cfg.APIServer.CORS.ExposedHeaders)
	maxAge := handlers.MaxAge(cfg.APIServer.CORS.MaxAge)

	// 构建CORS选项列表
	corsOptions := []handlers.CORSOption{
		allowedOrigins,
		allowedMethods,
		allowedHeaders,
		exposedHeaders,
		maxAge,
	}
	if cfg.APIServer.CORS.AllowCredentials {
		corsOptions = append(corsOptions, handlers.AllowCredentials())
	}

	// 将主路由器 r 包装在 CORS 中间件中
	corsHandler := handlers.CORS(corsOptions...)(r)

	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      corsHandler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  time.Second * 60,
	}

	go func() {
		log.Printf("API 服务器启动于 %s", serverAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("API 服务器启动失败: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("收到关闭信号，正在关闭 API 服务器...")

	cancelConsumers() // Signal Kafka consumer to stop
	log.Println("正在等待 Kafka 消费者停止...")
	// Ideally, wait for consumer goroutine to finish here, e.g., using a WaitGroup

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second) // Increased timeout slightly
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("API 服务器强制关闭: %v", err)
	}

	log.Println("API 服务器已成功关闭")
}

// 修复群组会话参与者
func fixGroupConversations(db *gorm.DB) {
	ctx := context.Background()

	// 初始化存储库
	userRepo := storage.NewGormUserRepository(db)
	convoRepo := storage.NewGormConversationRepository(db)

	// 初始化服务
	convoService := services.NewConversationService(convoRepo, userRepo)

	// 获取所有群组
	var groups []models.Group
	if err := db.Find(&groups).Error; err != nil {
		fmt.Printf("获取群组失败: %v\n", err)
		return
	}

	fmt.Printf("找到 %d 个群组，开始修复会话参与者...\n", len(groups))

	// 针对每个群组修复会话参与者
	for _, group := range groups {
		fmt.Printf("处理群组: ID=%d, 名称=%s\n", group.ID, group.Name)

		// 尝试修复
		if convoService, ok := convoService.(interface {
			FixGroupConversationParticipants(context.Context, uint) error
		}); ok {
			if err := convoService.FixGroupConversationParticipants(ctx, group.ID); err != nil {
				fmt.Printf("修复群组 %d 的会话参与者失败: %v\n", group.ID, err)
			}
		} else {
			fmt.Println("转换ConversationService失败，无法调用FixGroupConversationParticipants")
		}
	}

	fmt.Println("修复流程完成")
}
