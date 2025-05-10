package services

import (
	"context"
	"fmt"

	"im-go/internal/models"
	"im-go/internal/storage"
	// "im-go/internal/auth" // 如果需要密码更新等操作
)

// UserService 定义了用户相关服务的接口。
type UserService interface {
	GetUserProfile(ctx context.Context, userID uint) (*models.User, error)
	UpdateUserProfile(ctx context.Context, userID uint, nickname, avatarURL, bio string) (*models.User, error)
	// UpdateUserStatus(ctx context.Context, userID uint, status string) error
	// AddContact(ctx context.Context, userID, contactID uint) error
	// RemoveContact(ctx context.Context, userID, contactID uint) error
	// GetContacts(ctx context.Context, userID uint) ([]*models.User, error)
	SearchUsers(ctx context.Context, query string, currentUserID uint) ([]models.User, error)
}

// userService 是 UserService 的实现。
type userService struct {
	userRepo storage.UserRepository
	// 可能需要其他 repository，例如 ContactRepository
}

// NewUserService 创建一个新的 UserService 实例。
func NewUserService(userRepo storage.UserRepository) UserService {
	return &userService{userRepo: userRepo}
}

// GetUserProfile 获取用户公开的个人资料。
func (s *userService) GetUserProfile(ctx context.Context, userID uint) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		// 根据实际的错误类型（例如 gorm.ErrRecordNotFound）返回更具体的业务错误
		return nil, fmt.Errorf("获取用户 %d 失败: %w", userID, err)
	}
	// 清理敏感信息，例如密码哈希，即使它在 JSON 中通常被忽略
	user.PasswordHash = ""
	return user, nil
}

// UpdateUserProfile 更新用户的个人资料。
func (s *userService) UpdateUserProfile(ctx context.Context, userID uint, nickname, avatarURL, bio string) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("更新用户资料失败，用户 %d 未找到: %w", userID, err)
	}

	// 按需更新字段
	updated := false
	if nickname != "" && user.Nickname != nickname {
		user.Nickname = nickname
		updated = true
	}
	if avatarURL != "" && user.AvatarURL != avatarURL {
		user.AvatarURL = avatarURL
		updated = true
	}
	if bio != "" && user.Bio != bio { // 假设 bio 也可以清空
		user.Bio = bio
		updated = true
	}

	if !updated {
		user.PasswordHash = "" // 确保返回前清理
		return user, nil       // 没有字段被更新
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("更新用户 %d 资料失败: %w", userID, err)
	}
	user.PasswordHash = "" // 确保返回前清理
	return user, nil
}

// SearchUsers 实现 SearchUsers 方法
func (s *userService) SearchUsers(ctx context.Context, query string, currentUserID uint) ([]models.User, error) {
	// Basic validation for query can be added here if desired, e.g., min length.
	// For now, directly pass to repository.
	return s.userRepo.SearchUsers(ctx, query, currentUserID)
}
