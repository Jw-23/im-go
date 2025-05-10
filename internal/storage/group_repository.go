package storage

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"im-go/internal/models"
)

// GroupRepository 定义了群组数据操作的接口。
type GroupRepository interface {
	CreateGroup(ctx context.Context, group *models.Group) error
	GetGroupByID(ctx context.Context, id uint) (*models.Group, error)
	GetGroupByName(ctx context.Context, name string) (*models.Group, error)
	UpdateGroup(ctx context.Context, group *models.Group) error
	DeleteGroup(ctx context.Context, id uint) error // 软删除群组
	SearchGroups(ctx context.Context, query string, limit int, offset int) ([]*models.Group, error)

	AddMember(ctx context.Context, member *models.GroupMember) error
	GetMember(ctx context.Context, groupID uint, userID uint) (*models.GroupMember, error)
	UpdateMember(ctx context.Context, member *models.GroupMember) error
	RemoveMember(ctx context.Context, groupID uint, userID uint) error
	GetGroupMembers(ctx context.Context, groupID uint, limit int, offset int) ([]*models.GroupMember, error)
	GetUserGroups(ctx context.Context, userID uint, limit int, offset int) ([]*models.Group, error)
}

// gormGroupRepository 使用 GORM 实现 GroupRepository。
type gormGroupRepository struct {
	db *gorm.DB
}

// NewGormGroupRepository 创建一个新的基于 GORM 的 GroupRepository。
func NewGormGroupRepository(db *gorm.DB) GroupRepository {
	return &gormGroupRepository{db: db}
}

// CreateGroup 创建一个新的群组。
func (r *gormGroupRepository) CreateGroup(ctx context.Context, group *models.Group) error {
	return r.db.WithContext(ctx).Create(group).Error
}

// GetGroupByID 通过ID检索群组。
func (r *gormGroupRepository) GetGroupByID(ctx context.Context, id uint) (*models.Group, error) {
	var group models.Group
	// 预加载群主和成员信息
	err := r.db.WithContext(ctx).Preload("Owner").Preload("Members.User").First(&group, id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// GetGroupByName 通过名称检索群组。
func (r *gormGroupRepository) GetGroupByName(ctx context.Context, name string) (*models.Group, error) {
	var group models.Group
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&group).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// UpdateGroup 更新群组信息。
func (r *gormGroupRepository) UpdateGroup(ctx context.Context, group *models.Group) error {
	return r.db.WithContext(ctx).Save(group).Error
}

// DeleteGroup 软删除一个群组。
func (r *gormGroupRepository) DeleteGroup(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Group{}, id).Error
}

// SearchGroups 搜索群组。
func (r *gormGroupRepository) SearchGroups(ctx context.Context, query string, limit int, offset int) ([]*models.Group, error) {
	var groups []*models.Group
	dbQuery := r.db.WithContext(ctx).Model(&models.Group{}).
		Where("name LIKE ? OR description LIKE ?", "%"+query+"%", "%"+query+"%").
		Preload("Owner")

	if limit > 0 {
		dbQuery = dbQuery.Limit(limit)
	}
	if offset > 0 {
		dbQuery = dbQuery.Offset(offset)
	}

	err := dbQuery.Find(&groups).Error
	return groups, err
}

// AddMember向群组中添加成员。
// 如果成员已存在，可以考虑更新或返回错误。
func (r *gormGroupRepository) AddMember(ctx context.Context, member *models.GroupMember) error {
	// 使用 OnConflict 处理已存在成员的情况，例如更新角色或忽略
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(member).Error
}

// GetMember 获取群组中的特定成员信息。
func (r *gormGroupRepository) GetMember(ctx context.Context, groupID uint, userID uint) (*models.GroupMember, error) {
	var member models.GroupMember
	err := r.db.WithContext(ctx).Where("group_id = ? AND user_id = ?", groupID, userID).Preload("User").First(&member).Error
	return &member, err
}

// UpdateMember 更新群组成员信息，例如角色或别名。
func (r *gormGroupRepository) UpdateMember(ctx context.Context, member *models.GroupMember) error {
	return r.db.WithContext(ctx).Save(member).Error
}

// RemoveMember 从群组中移除成员。
func (r *gormGroupRepository) RemoveMember(ctx context.Context, groupID uint, userID uint) error {
	return r.db.WithContext(ctx).Where("group_id = ? AND user_id = ?", groupID, userID).Delete(&models.GroupMember{}).Error
}

// GetGroupMembers 获取群组的所有成员列表。
func (r *gormGroupRepository) GetGroupMembers(ctx context.Context, groupID uint, limit int, offset int) ([]*models.GroupMember, error) {
	var members []*models.GroupMember
	dbQuery := r.db.WithContext(ctx).Where("group_id = ?", groupID).Preload("User").Order("joined_at ASC")

	if limit > 0 {
		dbQuery = dbQuery.Limit(limit)
	}
	if offset > 0 {
		dbQuery = dbQuery.Offset(offset)
	}
	err := dbQuery.Find(&members).Error
	return members, err
}

// GetUserGroups 获取用户加入的所有群组列表。
func (r *gormGroupRepository) GetUserGroups(ctx context.Context, userID uint, limit int, offset int) ([]*models.Group, error) {
	var groups []*models.Group
	// 查询用户作为成员的所有群组
	dbQuery := r.db.WithContext(ctx).Joins("JOIN group_members gm ON gm.group_id = groups.id").
		Where("gm.user_id = ?", userID).Preload("Owner")

	if limit > 0 {
		dbQuery = dbQuery.Limit(limit)
	}
	if offset > 0 {
		dbQuery = dbQuery.Offset(offset)
	}

	err := dbQuery.Find(&groups).Error
	return groups, err
}
