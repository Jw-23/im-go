package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"im-go/internal/models"
	"im-go/internal/storage"

	"gorm.io/gorm"
	// "errors"
	// "gorm.io/gorm"
)

// GroupService 定义了群组相关服务的接口。
type GroupService interface {
	CreateGroup(ctx context.Context, ownerID uint, name, description, avatarURL string, isPublic bool, joinCondition string) (*models.Group, error)
	GetGroupDetails(ctx context.Context, groupID uint, userID uint) (*models.Group, error) // userID 用于检查成员资格和权限
	UpdateGroupInfo(ctx context.Context, userID, groupID uint, name, description, avatarURL string, isPublic bool, joinCondition string) (*models.Group, error)
	// DeleteGroup(ctx context.Context, userID, groupID uint) error // 需要权限检查
	SearchPublicGroups(ctx context.Context, query string, limit, offset int) ([]*models.Group, error)

	JoinGroup(ctx context.Context, userID, groupID uint) (*models.GroupMember, error)
	LeaveGroup(ctx context.Context, userID, groupID uint) error
	// InviteUserToGroup(ctx context.Context, inviterID, groupID, inviteeID uint) error
	// ApproveJoinRequest(ctx context.Context, adminID, groupID, userID uint) error
	// KickMember(ctx context.Context, adminID, groupID, memberID uint) error
	GetGroupMembers(ctx context.Context, groupID uint, limit, offset int) ([]*models.GroupMember, error)
	UpdateMemberRole(ctx context.Context, adminID, groupID, memberID uint, newRole models.GroupMemberRole) (*models.GroupMember, error)
	GetUserGroups(ctx context.Context, userID uint, limit, offset int) ([]*models.Group, error)
	GetGroupDetailsByID(ctx context.Context, groupID uint) (*models.Group, error)
}

// groupService 是 GroupService 的实现。
type groupService struct {
	groupRepo storage.GroupRepository
	userRepo  storage.UserRepository
	convoRepo storage.ConversationRepository // 用于在创建群组时，可能需要创建关联的群聊会话
}

// NewGroupService 创建一个新的 GroupService 实例。
func NewGroupService(groupRepo storage.GroupRepository, userRepo storage.UserRepository, convoRepo storage.ConversationRepository) GroupService {
	return &groupService{groupRepo: groupRepo, userRepo: userRepo, convoRepo: convoRepo}
}

// CreateGroup 创建一个新的群组。
func (s *groupService) CreateGroup(ctx context.Context, ownerID uint, name, description, avatarURL string, isPublic bool, joinCondition string) (*models.Group, error) {
	// 1. 验证输入 (例如，名称不能为空)
	if name == "" {
		return nil, fmt.Errorf("群组名称不能为空")
	}

	// 2. 检查群组名称是否已存在 (如果需要唯一性)
	// existingGroup, _ := s.groupRepo.GetGroupByName(ctx, name)
	// if existingGroup != nil {
	// 	return nil, fmt.Errorf("群组名称 '%s' 已存在", name)
	// }

	newGroup := &models.Group{
		OwnerID:       ownerID,
		Name:          name,
		Description:   description,
		AvatarURL:     avatarURL,
		IsPublic:      isPublic,
		JoinCondition: joinCondition,
		MemberCount:   1, // 创建者是第一个成员
	}

	if err := s.groupRepo.CreateGroup(ctx, newGroup); err != nil {
		return nil, fmt.Errorf("创建群组失败: %w", err)
	}

	// 3. 将创建者添加为群管理员
	ownerMember := &models.GroupMember{
		GroupID:  newGroup.ID,
		UserID:   ownerID,
		Role:     models.AdminRole,
		JoinedAt: time.Now(),
	}
	if err := s.groupRepo.AddMember(ctx, ownerMember); err != nil {
		// 注意：这里可能需要事务回滚，如果创建群组成功但添加成员失败
		return newGroup, fmt.Errorf("将群主 %d 添加到群组 %d 失败: %w", ownerID, newGroup.ID, err)
	}

	// 4. (可选) 为新群组创建一个关联的群聊会话
	groupConversation := &models.Conversation{
		Type:     models.GroupConversation,
		TargetID: newGroup.ID, // 将群组ID关联到会话
	}
	if err := s.convoRepo.CreateConversation(ctx, groupConversation); err != nil {
		// 这个错误可能不应该阻止群组创建成功，但需要记录日志
		// log.Printf("为群组 %d 创建关联会话失败: %v", newGroup.ID, err)
	}

	return newGroup, nil
}

// GetGroupDetails 获取群组的详细信息。
func (s *groupService) GetGroupDetails(ctx context.Context, groupID uint, userID uint) (*models.Group, error) {
	group, err := s.groupRepo.GetGroupByID(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("获取群组 %d 详情失败: %w", groupID, err)
	}

	// TODO: 根据 userID 检查用户是否是群成员，如果群组是私有的，非成员可能无权查看
	// if !group.IsPublic {
	// 	_, memberErr := s.groupRepo.GetMember(ctx, groupID, userID)
	// 	if memberErr != nil { // 如果不是成员
	// 		 return nil, fmt.Errorf("无权查看私有群组 %d 的详情", groupID)
	// 	}
	// }

	return group, nil
}

// UpdateGroupInfo 更新群组信息 (需要权限检查，例如只有群主或管理员可以更新)。
func (s *groupService) UpdateGroupInfo(ctx context.Context, userID, groupID uint, name, description, avatarURL string, isPublic bool, joinCondition string) (*models.Group, error) {
	group, err := s.groupRepo.GetGroupByID(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("更新群组信息失败，群组 %d 未找到: %w", groupID, err)
	}

	// 权限检查：例如，只有群主可以修改
	if group.OwnerID != userID {
		// 或者检查是否为管理员
		// member, _ := s.groupRepo.GetMember(ctx, groupID, userID)
		// if member == nil || member.Role != models.AdminRole {
		// 	 return nil, fmt.Errorf("用户 %d 无权修改群组 %d 的信息", userID, groupID)
		// }
		return nil, fmt.Errorf("用户 %d 不是群主，无权修改群组 %d 的信息", userID, groupID) // 简化权限
	}

	updated := false
	if name != "" && group.Name != name {
		group.Name = name
		updated = true
	}
	if description != "" && group.Description != description {
		group.Description = description
		updated = true
	}
	// ... 其他字段更新逻辑 ...
	group.IsPublic = isPublic // isPublic 通常可以直接赋值
	updated = true

	if !updated {
		return group, nil
	}

	if err := s.groupRepo.UpdateGroup(ctx, group); err != nil {
		return nil, fmt.Errorf("更新群组 %d 信息失败: %w", groupID, err)
	}
	return group, nil
}

// SearchPublicGroups 搜索公开群组。
func (s *groupService) SearchPublicGroups(ctx context.Context, query string, limit, offset int) ([]*models.Group, error) {
	// TODO: 可以在 Repository 层增加一个只搜索 IsPublic=true 的方法
	return s.groupRepo.SearchGroups(ctx, query, limit, offset) // 当前 SearchGroups 未区分公开/私有
}

// JoinGroup 用户加入公开群组或处理邀请/申请（后者逻辑更复杂）。
func (s *groupService) JoinGroup(ctx context.Context, userID, groupID uint) (*models.GroupMember, error) {
	group, err := s.groupRepo.GetGroupByID(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("加入群组失败，群组 %d 未找到: %w", groupID, err)
	}

	// 检查是否已是成员
	existingMember, _ := s.groupRepo.GetMember(ctx, groupID, userID)
	if existingMember != nil {
		return existingMember, fmt.Errorf("用户 %d 已经是群组 %d 的成员", userID, groupID)
	}

	// TODO: 根据 group.JoinCondition 处理加入逻辑 (直接加入、需要审批、仅邀请)
	// 这里简化为直接加入
	if !group.IsPublic && group.JoinCondition != "direct_join" { // 只是一个示例判断
		// return nil, fmt.Errorf("此群组 %d 不允许直接加入", groupID)
	}

	newMember := &models.GroupMember{
		GroupID:  groupID,
		UserID:   userID,
		Role:     models.MemberRole,
		JoinedAt: time.Now(),
	}
	if err := s.groupRepo.AddMember(ctx, newMember); err != nil {
		return nil, fmt.Errorf("用户 %d 加入群组 %d 失败: %w", userID, groupID, err)
	}

	// 更新群组成员数
	group.MemberCount++
	if err := s.groupRepo.UpdateGroup(ctx, group); err != nil {
		// 这个错误不应阻塞加入成功，但需要记录
		// log.Printf("更新群组 %d 成员数失败: %v", groupID, err)
	}

	return newMember, nil
}

// LeaveGroup 用户离开群组。
func (s *groupService) LeaveGroup(ctx context.Context, userID, groupID uint) error {
	// 检查用户是否是群成员。 GetMember 会在找不到成员时返回错误。
	_, err := s.groupRepo.GetMember(ctx, groupID, userID)
	if err != nil {
		return fmt.Errorf("离开群组失败，用户 %d 可能不是群组 %d 的成员: %w", userID, groupID, err)
	}

	group, err := s.groupRepo.GetGroupByID(ctx, groupID)
	if err != nil {
		return fmt.Errorf("获取群组 %d 信息失败: %w", groupID, err)
	}

	if group.OwnerID == userID {
		// 如果是群主
		if group.MemberCount <= 1 {
			// 是最后一个成员，理论上可以解散群组，但这里简单处理为不允许离开
			return fmt.Errorf("群主是最后一个成员，不能离开。请先解散群组或转让群主身份。")
		} else {
			return fmt.Errorf("群主需要先转让群主身份才能离开群组")
		}
	}

	if err := s.groupRepo.RemoveMember(ctx, groupID, userID); err != nil {
		return fmt.Errorf("从群组 %d 移除用户 %d 失败: %w", groupID, userID, err)
	}

	// 更新群组成员数
	// 确保 group.MemberCount 不会变为负数，尽管 RemoveMember 应该已经处理了不存在的情况。
	if group.MemberCount > 0 {
		group.MemberCount--
		if err := s.groupRepo.UpdateGroup(ctx, group); err != nil {
			// log.Printf("更新群组 %d 成员数失败: %v", groupID, err) // 记录日志，但不应阻塞用户离开操作
		}
	}
	return nil
}

// GetGroupMembers 获取群组成员列表。
func (s *groupService) GetGroupMembers(ctx context.Context, groupID uint, limit, offset int) ([]*models.GroupMember, error) {
	return s.groupRepo.GetGroupMembers(ctx, groupID, limit, offset)
}

// UpdateMemberRole 更新群组成员的角色 (通常由管理员或群主操作)。
func (s *groupService) UpdateMemberRole(ctx context.Context, adminID, groupID, memberID uint, newRole models.GroupMemberRole) (*models.GroupMember, error) {
	// 1. 验证操作者 (adminID) 是否有权限
	adminMember, err := s.groupRepo.GetMember(ctx, groupID, adminID)
	if err != nil || adminMember.Role != models.AdminRole {
		return nil, fmt.Errorf("用户 %d 无权在群组 %d 中修改成员角色", adminID, groupID)
	}

	// 2. 获取目标成员
	targetMember, err := s.groupRepo.GetMember(ctx, groupID, memberID)
	if err != nil {
		return nil, fmt.Errorf("修改角色失败，成员 %d 未在群组 %d 中找到: %w", memberID, groupID, err)
	}

	// 3. 更新角色
	// TODO: 检查是否允许将自己降级，或将最后一个管理员降级等逻辑
	targetMember.Role = newRole
	if err := s.groupRepo.UpdateMember(ctx, targetMember); err != nil {
		return nil, fmt.Errorf("更新成员 %d 在群组 %d 中的角色失败: %w", memberID, groupID, err)
	}
	return targetMember, nil
}

// GetUserGroups 获取用户加入的所有群组列表。
func (s *groupService) GetUserGroups(ctx context.Context, userID uint, limit, offset int) ([]*models.Group, error) {
	return s.groupRepo.GetUserGroups(ctx, userID, limit, offset)
}

// GetGroupDetailsByID retrieves group details by its ID.
// This is a new method we are adding.
func (s *groupService) GetGroupDetailsByID(ctx context.Context, groupID uint) (*models.Group, error) {
	if s.groupRepo == nil {
		return nil, errors.New("groupRepository is not initialized in groupService")
	}
	// We assume groupRepo will have a FindByID method.
	// You will need to add this method to your storage.GroupRepository interface
	// and implement it in its GORM (or other) implementation.
	group, err := s.groupRepo.GetGroupByID(ctx, groupID) // ASSUMING groupRepo.FindByID
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("group with ID %d not found", groupID) // More specific error
		}
		return nil, fmt.Errorf("failed to get group %d: %w", groupID, err)
	}
	return group, nil
}
