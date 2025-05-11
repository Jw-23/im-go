package services

import (
	"context"
	"errors"
	"fmt"

	"im-go/internal/auth"
	"im-go/internal/config"
	"im-go/internal/models"
	"im-go/internal/storage"

	"gorm.io/gorm"
)

var (
	ErrUserAlreadyExists  = errors.New("用户名或邮箱已存在")
	ErrInvalidCredentials = errors.New("无效的用户名或密码")
	ErrUserNotFound       = errors.New("用户未找到")
)

// AuthService 定义了用户认证服务的接口。
type AuthService interface {
	Register(ctx context.Context, username, nickname, email, password string) (*models.User, error)
	Login(ctx context.Context, usernameOrEmail, password string) (token string, user *models.User, err error)
}

// authService 是 AuthService 的实现。
type authService struct {
	userRepo storage.UserRepository
	cfg      config.Config // 包含 AuthConfig
}

// NewAuthService 创建一个新的 AuthService 实例。
func NewAuthService(userRepo storage.UserRepository, cfg config.Config) AuthService {
	return &authService{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

// Register 处理用户注册逻辑。
func (s *authService) Register(ctx context.Context, username, nickname, email, password string) (*models.User, error) {
	// 检查用户名是否存在
	_, err := s.userRepo.GetByUsername(ctx, username)
	if err == nil {
		return nil, ErrUserAlreadyExists // 用户名已存在
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("检查用户名时出错: %w", err)
	}

	// 检查邮箱是否存在 (如果邮箱是必须的且唯一的)
	if email != "" {
		_, err = s.userRepo.GetByEmail(ctx, email)
		if err == nil {
			return nil, ErrUserAlreadyExists // 邮箱已存在
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("检查邮箱时出错: %w", err)
		}
	}

	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("密码哈希失败: %w", err)
	}

	newUser := &models.User{
		Username:     username,
		Nickname:     nickname,
		Email:        email,
		PasswordHash: hashedPassword,
		// 其他默认字段可以在模型或数据库层面设置
	}

	if err := s.userRepo.Create(ctx, newUser); err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	return newUser, nil
}

// Login 处理用户登录逻辑。
func (s *authService) Login(ctx context.Context, usernameOrEmail, password string) (string, *models.User, error) {
	var user *models.User
	var err error

	// 尝试通过用户名查找用户
	user, err = s.userRepo.GetByUsername(ctx, usernameOrEmail)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 如果用户名未找到，尝试通过邮箱查找 (如果 email 字段被用于登录)
		user, err = s.userRepo.GetByEmail(ctx, usernameOrEmail)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, ErrUserNotFound
		} else if err != nil {
			return "", nil, fmt.Errorf("通过邮箱查找用户失败: %w", err)
		}
	} else if err != nil {
		return "", nil, fmt.Errorf("通过用户名查找用户失败: %w", err)
	}

	if !auth.CheckPasswordHash(password, user.PasswordHash) {
		return "", nil, ErrInvalidCredentials
	}

	token, err := auth.GenerateToken(user.ID, user.Username, s.cfg.Auth)
	if err != nil {
		return "", nil, fmt.Errorf("生成令牌失败: %w", err)
	}

	return token, user, nil
}
