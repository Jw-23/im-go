package storage

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"im-go/internal/config"
	"im-go/internal/models"
)

// DB is the global database connection pool.
var DB *gorm.DB

// InitDB initializes the database connection using the provided configuration.
func InitDB(cfg config.DatabaseConfig) (*gorm.DB, error) {
	var dsn string
	var dialector gorm.Dialector

	log.Printf("DEBUG [InitDB]: Received full cfg for database: %+v", cfg)

	switch cfg.Type {
	case "postgres":
		var dsnParts []string
		dsnParts = append(dsnParts, fmt.Sprintf("host=%s", cfg.Host))
		dsnParts = append(dsnParts, fmt.Sprintf("port=%d", cfg.Port))
		dsnParts = append(dsnParts, fmt.Sprintf("user=%s", cfg.User))
		dsnParts = append(dsnParts, fmt.Sprintf("dbname=%s", cfg.DBName))

		if cfg.Password != "" {
			dsnParts = append(dsnParts, fmt.Sprintf("password=%s", cfg.Password))
		}

		dsnParts = append(dsnParts, fmt.Sprintf("sslmode=%s", cfg.SSLMode))

		dsn = strings.Join(dsnParts, " ")
		log.Printf("DEBUG [InitDB]: Constructed DSN: %s", dsn)

		dialector = postgres.Open(dsn)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}

	// Configure GORM logger
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Info, // Log level (Silent, Error, Warn, Info)
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			Colorful:                  true,        // Disable color
		},
	)

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: newLogger,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	DB = db // Assign to global DB variable for convenience, or manage it via dependency injection
	return db, nil
}

// AutoMigrateTables runs GORM's auto-migration feature for all defined models.
func AutoMigrateTables(db *gorm.DB) error {
	log.Println("开始数据库表结构迁移...")
	err := db.AutoMigrate(
		&models.User{},
		&models.Message{},
		&models.Conversation{},
		&models.ConversationParticipant{},
		&models.Group{},
		&models.GroupMember{},
		&models.FriendRequest{},
		&models.Friendship{},
	)
	if err != nil {
		log.Printf("数据库迁移失败: %v", err)
		return fmt.Errorf("数据库迁移失败: %w", err)
	}
	log.Println("数据库迁移完成。")
	return nil
}
