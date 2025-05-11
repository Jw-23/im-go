package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

// APIServerConfig 保存 API 服务器特有的配置。
type APIServerConfig struct {
	Host string     `mapstructure:"HOST"`
	Port string     `mapstructure:"PORT"`
	CORS CORSConfig `mapstructure:"CORS"` // ADDED: CORS configuration
}

// ADDED: CORSConfig holds configuration for CORS.
type CORSConfig struct {
	AllowedOrigins   []string `mapstructure:"ALLOWED_ORIGINS"`
	AllowedMethods   []string `mapstructure:"ALLOWED_METHODS"`
	AllowedHeaders   []string `mapstructure:"ALLOWED_HEADERS"`
	ExposedHeaders   []string `mapstructure:"EXPOSED_HEADERS"`
	AllowCredentials bool     `mapstructure:"ALLOW_CREDENTIALS"`
	MaxAge           int      `mapstructure:"MAX_AGE"`
}

// ADDED: RedisConfig holds configuration for Redis.
type RedisConfig struct {
	Addr     string `mapstructure:"ADDR"`
	Password string `mapstructure:"PASSWORD"`
	DB       int    `mapstructure:"DB"`
}

// Config holds all configuration for the application.
// The values are read by viper from a config file or environment variables.
type Config struct {
	AppName    string          `mapstructure:"APP_NAME"`
	AppVersion string          `mapstructure:"APP_VERSION"`
	LogLevel   string          `mapstructure:"LOG_LEVEL"`
	Server     ServerConfig    `mapstructure:"SERVER"`     // 这个是 ChatServer 的配置
	APIServer  APIServerConfig `mapstructure:"API_SERVER"` // 新增 API 服务器配置
	Kafka      KafkaConfig     `mapstructure:"KAFKA"`
	Database   DatabaseConfig  `mapstructure:"DATABASE"`
	Storage    StorageConfig   `mapstructure:"STORAGE"`
	Auth       AuthConfig      `mapstructure:"AUTH"`
	WebSocket  WebSocketConfig `mapstructure:"WEBSOCKET"`
	Redis      RedisConfig     `mapstructure:"REDIS"` // ADDED RedisConfig
}

// ServerConfig holds configuration for the HTTP server.
// We might have multiple servers (e.g., ChatServer, APIServer)
// For now, one generic server config.
// We can also split ChatServer specific config later.
type ServerConfig struct {
	Host           string        `mapstructure:"HOST"`
	Port           string        `mapstructure:"PORT"`
	WebSocketPath  string        `mapstructure:"WEBSOCKET_PATH"`
	ReadTimeout    time.Duration `mapstructure:"READ_TIMEOUT"`
	WriteTimeout   time.Duration `mapstructure:"WRITE_TIMEOUT"`
	MaxHeaderBytes int           `mapstructure:"MAX_HEADER_BYTES"`
}

// KafkaConfig holds configuration for Kafka.
type KafkaConfig struct {
	Brokers                []string `mapstructure:"BROKERS"`
	ClientID               string   `mapstructure:"CLIENT_ID"`
	MessagesTopic          string   `mapstructure:"MESSAGES_TOPIC"`           // 用于从客户端到服务端的原始消息
	GroupTopic             string   `mapstructure:"GROUP_TOPIC"`              // (可选) 如果群聊消息单独处理
	NotificationsTopic     string   `mapstructure:"NOTIFICATIONS_TOPIC"`      // (可选) 用于其他系统通知
	WebSocketOutgoingTopic string   `mapstructure:"WEBSOCKET_OUTGOING_TOPIC"` // 新增：用于服务端推向客户端的消息
	ConsumerGroup          string   `mapstructure:"CONSUMER_GROUP"`           // ChatServer 主消费者组
	FriendRequestTopic     string   `mapstructure:"FRIEND_REQUEST_TOPIC"`     // ADDED
	Protocol               string   `mapstructure:"PROTOCOL"`
	// 可以为 WebSocketOutgoingTopic 定义一个单独的消费者组，如果需要每个 ChatServer 实例都收到所有出站消息
	// OutgoingConsumerGroup string   `mapstructure:"OUTGOING_CONSUMER_GROUP"`
}

// DatabaseConfig holds configuration for the database.
type DatabaseConfig struct {
	Type     string `mapstructure:"TYPE"`
	Host     string `mapstructure:"HOST"`
	Port     int    `mapstructure:"PORT"`
	User     string `mapstructure:"USER"`
	Password string `mapstructure:"PASSWORD"`
	DBName   string `mapstructure:"DB_NAME"`
	SSLMode  string `mapstructure:"SSL_MODE"`
}

// StorageConfig holds configuration for file storage.
type StorageConfig struct {
	Type          string   `mapstructure:"TYPE"` // "local", "s3", "gcs"
	LocalPath     string   `mapstructure:"LOCAL_PATH"`
	MaxFileSizeMB int64    `mapstructure:"MAX_FILE_SIZE_MB"`
	S3            S3Config `mapstructure:"S3"`
}

// S3Config holds configuration for AWS S3.
type S3Config struct {
	BucketName      string `mapstructure:"BUCKET_NAME"`
	Region          string `mapstructure:"REGION"`
	AccessKeyID     string `mapstructure:"ACCESS_KEY_ID"`
	SecretAccessKey string `mapstructure:"SECRET_ACCESS_KEY"`
	Endpoint        string `mapstructure:"ENDPOINT"` // For S3 compatible storage like MinIO
}

// AuthConfig holds configuration for authentication (e.g., JWT).
type AuthConfig struct {
	JWTSecretKey string        `mapstructure:"JWT_SECRET_KEY"`
	JWTExpiry    time.Duration `mapstructure:"JWT_EXPIRY"`
}

// WebSocketConfig holds configuration for WebSocket connections.
type WebSocketConfig struct {
	WriteWaitSeconds    int `mapstructure:"WRITE_WAIT_SECONDS"`
	PongWaitSeconds     int `mapstructure:"PONG_WAIT_SECONDS"`
	PingPeriodSeconds   int `mapstructure:"PING_PERIOD_SECONDS"`
	MaxMessageSizeBytes int `mapstructure:"MAX_MESSAGE_SIZE_BYTES"`
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(path string) (config Config, err error) {
	v := viper.New()

	v.SetDefault("APP_NAME", "IM-Go")
	v.SetDefault("APP_VERSION", "0.0.1")
	v.SetDefault("LOG_LEVEL", "info")

	// Server Defaults (ChatServer)
	v.SetDefault("SERVER.HOST", "0.0.0.0")
	v.SetDefault("SERVER.PORT", "8080")
	v.SetDefault("SERVER.WEBSOCKET_PATH", "/ws")
	v.SetDefault("SERVER.READ_TIMEOUT", 30*time.Second)
	v.SetDefault("SERVER.WRITE_TIMEOUT", 30*time.Second)
	v.SetDefault("SERVER.MAX_HEADER_BYTES", 1<<20) // 1 MB

	// APIServer Defaults (新增)
	v.SetDefault("API_SERVER.HOST", "0.0.0.0")
	v.SetDefault("API_SERVER.PORT", "8081") // 为 API 服务器设置不同端口
	// ADDED: APIServer CORS Defaults
	v.SetDefault("API_SERVER.CORS.ALLOWED_ORIGINS", []string{"http://localhost:5173"}) // Adjust for your frontend URL
	v.SetDefault("API_SERVER.CORS.ALLOWED_METHODS", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	v.SetDefault("API_SERVER.CORS.ALLOWED_HEADERS", []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"})
	v.SetDefault("API_SERVER.CORS.EXPOSED_HEADERS", []string{"Content-Length"})
	v.SetDefault("API_SERVER.CORS.ALLOW_CREDENTIALS", true)
	v.SetDefault("API_SERVER.CORS.MAX_AGE", 300) // 5 minutes

	// Kafka Defaults
	v.SetDefault("KAFKA.BROKERS", []string{"localhost:9092"})
	v.SetDefault("KAFKA.CLIENT_ID", "im-go-client")
	v.SetDefault("KAFKA.MESSAGES_TOPIC", "im-messages")
	v.SetDefault("KAFKA.GROUP_TOPIC", "im-group-messages")
	v.SetDefault("KAFKA.NOTIFICATIONS_TOPIC", "im-notifications")
	v.SetDefault("KAFKA.WEBSOCKET_OUTGOING_TOPIC", "im-websocket-outgoing")
	v.SetDefault("KAFKA.CONSUMER_GROUP", "im-chat-server-group")
	v.SetDefault("KAFKA.FRIEND_REQUEST_TOPIC", "im-friend-request")

	// Database Defaults (Example for PostgreSQL)
	v.SetDefault("DATABASE.TYPE", "postgres")
	v.SetDefault("DATABASE.HOST", "localhost")
	v.SetDefault("DATABASE.PORT", 5432)
	v.SetDefault("DATABASE.USER", "postgres")
	v.SetDefault("DATABASE.PASSWORD", "password")
	v.SetDefault("DATABASE.DB_NAME", "im_go_db")
	v.SetDefault("DATABASE.SSL_MODE", "disable")

	// Storage Defaults
	v.SetDefault("STORAGE.TYPE", "local")
	v.SetDefault("STORAGE.LOCAL_PATH", "./uploads")
	v.SetDefault("STORAGE.MAX_FILE_SIZE_MB", 100) // 100 MB

	// Auth Defaults
	v.SetDefault("AUTH.JWT_SECRET_KEY", "a_very_secret_key_that_should_be_changed")
	v.SetDefault("AUTH.JWT_EXPIRY", 15*time.Minute) // 15 minutes

	// ADDED: Redis Defaults
	v.SetDefault("REDIS.ADDR", "localhost:6379")
	v.SetDefault("REDIS.PASSWORD", "")
	v.SetDefault("REDIS.DB", 0)

	// WebSocket Defaults (values similar to existing constants)
	v.SetDefault("WEBSOCKET.WRITE_WAIT_SECONDS", 10)
	v.SetDefault("WEBSOCKET.PONG_WAIT_SECONDS", 60)
	v.SetDefault("WEBSOCKET.PING_PERIOD_SECONDS", 54) // (60 * 9) / 10
	v.SetDefault("WEBSOCKET.MAX_MESSAGE_SIZE_BYTES", 512)

	if path != "" {
		v.SetConfigFile(path) // Path to look for the config file in.
	} else {
		v.AddConfigPath("./config") // Path to look for the config file in.
		v.AddConfigPath(".")        // Optionally look for config in the working directory.
		v.SetConfigName("config")   // Name of config file (without extension).
		v.SetConfigType("yaml")     // REQUIRED if the config file does not have the extension in the name
	}

	v.AutomaticEnv() // Read in environment variables that match
	// Example: SERVER_PORT will override Server.Port
	// For nested structs, viper uses underscore: SERVER_WEBSOCKET_PATH
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err = v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error was produced
			return
		}
		// Config file not found; ignore error if desired
		// We have defaults, so this might be acceptable
	}

	err = v.Unmarshal(&config)
	return
}
