package storage

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"im-go/internal/models"
)

// UserRepository defines the interface for user data operations.
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uint) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	SearchUsers(ctx context.Context, query string, currentUserID uint) ([]models.User, error)
	GetBasicInfoByID(ctx context.Context, id uint) (*models.UserBasicInfo, error)
	GetMultipleBasicInfoByIDs(ctx context.Context, userIDs []uint) ([]*models.UserBasicInfo, error)
	GetDB() *gorm.DB
	// Delete(ctx context.Context, id uint) error // Depending on soft delete or hard delete preference
	// List(ctx context.Context, offset, limit int) ([]*models.User, error)
}

// gormUserRepository implements UserRepository using GORM.
type gormUserRepository struct {
	db *gorm.DB
}

// NewGormUserRepository creates a new GORM-based UserRepository.
func NewGormUserRepository(db *gorm.DB) UserRepository {
	return &gormUserRepository{db: db}
}

// Create creates a new user record in the database.
func (r *gormUserRepository) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// GetByID retrieves a user by their ID.
func (r *gormUserRepository) GetByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		return nil, err // Handles gorm.ErrRecordNotFound as well
	}
	return &user, nil
}

// GetByUsername retrieves a user by their username.
func (r *gormUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail retrieves a user by their email.
func (r *gormUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update updates an existing user record in the database.
func (r *gormUserRepository) Update(ctx context.Context, user *models.User) error {
	// Ensure ID is present for updates
	if user.ID == 0 {
		return gorm.ErrMissingWhereClause // Or a custom error
	}
	// Use Save to update all fields, or Updates to update non-zero fields / specified fields
	// For full control over what's updated, especially with potential zero values for fields
	// that should be updated, consider using a map with Updates or select specific fields.
	return r.db.WithContext(ctx).Save(user).Error
}

// SearchUsers implements the SearchUsers method for the UserRepository interface.
func (r *gormUserRepository) SearchUsers(ctx context.Context, query string, currentUserID uint) ([]models.User, error) {
	var users []models.User
	// 使用 strings.ToLower 来准备大小写不敏感的搜索词
	searchTerm := "%" + strings.ToLower(query) + "%"

	// 执行查询
	err := r.db.WithContext(ctx).
		// 在 username 和 nickname 字段上进行大小写不敏感的模糊匹配
		// 并排除当前用户自己
		Where("(LOWER(username) LIKE ? OR LOWER(nickname) LIKE ?) AND id != ?", searchTerm, searchTerm, currentUserID).
		//明确选择需要的字段，避免泄露敏感信息，同时提高查询效率
		Select("id", "username", "nickname", "avatar_url").
		Limit(10). // 限制返回结果的数量，例如最多10条
		Find(&users).Error

	if err != nil {
		// 如果是 gorm.ErrRecordNotFound，说明没有找到匹配的用户，这不是一个需要中断操作的错误
		// 对于搜索功能，返回空的用户列表是正常行为
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return users, nil // 返回空的 users 切片和 nil 错误
		}
		return nil, err // 对于其他数据库错误，返回错误
	}
	return users, nil
}

// GetBasicInfoByID retrieves minimal public user info by ID.
func (r *gormUserRepository) GetBasicInfoByID(ctx context.Context, id uint) (*models.UserBasicInfo, error) {
	var basicInfo models.UserBasicInfo
	err := r.db.WithContext(ctx).
		Model(&models.User{}).
		Select("id", "username", "nickname", "avatar_url").
		Where("id = ?", id).
		First(&basicInfo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &basicInfo, nil
}

// GetMultipleBasicInfoByIDs retrieves minimal public user info for a list of user IDs.
func (r *gormUserRepository) GetMultipleBasicInfoByIDs(ctx context.Context, userIDs []uint) ([]*models.UserBasicInfo, error) {
	var basicInfos []*models.UserBasicInfo
	if len(userIDs) == 0 {
		return basicInfos, nil // Return empty slice if no IDs are provided
	}

	err := r.db.WithContext(ctx).
		Model(&models.User{}).
		Select("id", "username", "nickname", "avatar_url").
		Where("id IN ?", userIDs).
		Find(&basicInfos).Error

	if err != nil {
		// Don't return ErrRecordNotFound for batch fetches, just return potentially empty slice
		return nil, err
	}
	return basicInfos, nil
}

// GetDB returns the underlying gorm.DB instance
func (r *gormUserRepository) GetDB() *gorm.DB {
	return r.db
}
