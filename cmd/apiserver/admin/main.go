package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"im-go/internal/models"
	"im-go/internal/storage"
)

func main() {
	// 简单命令行参数解析
	if len(os.Args) < 2 {
		fmt.Println("使用方法:")
		fmt.Println("  ./admin list-participants <conversationID> - 列出会话的所有参与者")
		fmt.Println("  ./admin show-group <groupID> - 显示群组信息")
		fmt.Println("  ./admin show-conversation <conversationID> - 显示会话信息")
		os.Exit(1)
	}

	// 数据库连接
	dsn := "host=localhost port=5432 user=postgres dbname=im_chat_prod password=postgres sslmode=disable"
	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer sqlDB.Close()

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // 慢SQL阈值
			LogLevel:                  logger.Info, // 日志级别
			IgnoreRecordNotFoundError: true,        // 忽略ErrRecordNotFound
			Colorful:                  true,        // 禁用彩色
		},
	)

	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		log.Fatalf("Failed to create GORM instance: %v", err)
	}

	// 创建存储层实例
	convoRepo := storage.NewGormConversationRepository(db)

	// 执行指定的命令
	switch os.Args[1] {
	case "list-participants":
		if len(os.Args) < 3 {
			log.Fatalf("需要指定会话ID")
		}
		conversationID, err := strconv.ParseUint(os.Args[2], 10, 64)
		if err != nil {
			log.Fatalf("无效的会话ID: %v", err)
		}
		listParticipants(convoRepo, uint(conversationID))

	case "show-group":
		if len(os.Args) < 3 {
			log.Fatalf("需要指定群组ID")
		}
		groupID, err := strconv.ParseUint(os.Args[2], 10, 64)
		if err != nil {
			log.Fatalf("无效的群组ID: %v", err)
		}
		showGroup(db, uint(groupID))

	case "show-conversation":
		if len(os.Args) < 3 {
			log.Fatalf("需要指定会话ID")
		}
		conversationID, err := strconv.ParseUint(os.Args[2], 10, 64)
		if err != nil {
			log.Fatalf("无效的会话ID: %v", err)
		}
		showConversation(convoRepo, uint(conversationID))

	default:
		log.Fatalf("未知命令: %s", os.Args[1])
	}
}

func listParticipants(repo storage.ConversationRepository, conversationID uint) {
	participants, err := repo.GetConversationParticipants(context.Background(), conversationID)
	if err != nil {
		log.Fatalf("获取参与者失败: %v", err)
	}

	fmt.Printf("会话 %d 的参与者 (%d 人):\n", conversationID, len(participants))
	fmt.Println("--------------------------------------")
	for i, p := range participants {
		fmt.Printf("#%d ID: %d, 用户ID: %d, 加入时间: %s, 是否管理员: %v\n",
			i+1, p.ID, p.UserID, p.JoinedAt.Format("2006-01-02 15:04:05"), p.IsAdmin)
	}
}

func showGroup(db *gorm.DB, groupID uint) {
	var group models.Group
	err := db.First(&group, groupID).Error
	if err != nil {
		log.Fatalf("查找群组失败: %v", err)
	}

	fmt.Printf("群组 %d 信息:\n", groupID)
	fmt.Println("--------------------------------------")
	fmt.Printf("名称: %s\n", group.Name)
	fmt.Printf("描述: %s\n", group.Description)
	fmt.Printf("创建者: %d\n", group.CreatorID)
	fmt.Printf("创建时间: %s\n", group.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("是否公开: %v\n", group.IsPublic)

	// 查找关联的会话
	var conversation models.Conversation
	err = db.Where("type = ? AND target_id = ?", models.GroupConversation, groupID).First(&conversation).Error
	if err != nil {
		fmt.Printf("未找到关联的会话: %v\n", err)
	} else {
		fmt.Printf("关联的会话ID: %d\n", conversation.ID)
	}
}

func showConversation(repo storage.ConversationRepository, conversationID uint) {
	conversation, err := repo.GetConversationByID(context.Background(), conversationID)
	if err != nil {
		log.Fatalf("获取会话失败: %v", err)
	}

	fmt.Printf("会话 %d 信息:\n", conversationID)
	fmt.Println("--------------------------------------")
	fmt.Printf("类型: %s\n", conversation.Type)
	fmt.Printf("目标ID: %d\n", conversation.TargetID)
	fmt.Printf("创建时间: %s\n", conversation.CreatedAt.Format("2006-01-02 15:04:05"))

	// 获取参与者
	participants, err := repo.GetConversationParticipants(context.Background(), conversationID)
	if err != nil {
		fmt.Printf("获取参与者失败: %v\n", err)
	} else {
		fmt.Printf("参与者数量: %d\n", len(participants))
	}
}
