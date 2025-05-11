package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"im-go/internal/models"
	"im-go/internal/storage"

	"gorm.io/gorm"
)

// ConversationService 定义了会话相关服务的接口。
type ConversationService interface {
	// GetOrCreatePrivateConversation 获取或创建两个用户之间的私聊会话。
	// 返回会话对象以及一个布尔值，指示会话是否是新创建的。
	GetOrCreatePrivateConversation(ctx context.Context, userID1, userID2 uint) (*models.Conversation, bool, error)
	GetUserConversations(ctx context.Context, userID uint, limit, offset int) ([]*models.Conversation, error)
	GetConversationDetails(ctx context.Context, conversationID uint, userID uint) (*models.Conversation, error) // userID 用于权限检查或个性化信息
	GetConversationParticipants(ctx context.Context, conversationID uint) ([]*models.ConversationParticipant, error)
	GetUserByID(ctx context.Context, userID uint) (*models.User, error)
	// 添加创建群聊会话的方法
	CreateGroupConversation(ctx context.Context, groupID uint, creatorID uint, memberIDs []uint) (*models.Conversation, error)
	// 添加修复群聊会话参与者的方法
	FixGroupConversationParticipants(ctx context.Context, groupID uint) error
	// UpdateConversationSettings(ctx context.Context, userID, conversationID uint, settings map[string]interface{}) error
	// LeaveConversation(ctx context.Context, userID, conversationID uint) error
}

// conversationService 是 ConversationService 的实现。
type conversationService struct {
	convoRepo storage.ConversationRepository
	userRepo  storage.UserRepository // 可能需要用于获取参与者信息
}

// NewConversationService 创建一个新的 ConversationService 实例。
func NewConversationService(convoRepo storage.ConversationRepository, userRepo storage.UserRepository) ConversationService {
	return &conversationService{convoRepo: convoRepo, userRepo: userRepo}
}

// GetOrCreatePrivateConversation 获取或创建两个用户之间的私聊会话。
func (s *conversationService) GetOrCreatePrivateConversation(ctx context.Context, userID1, userID2 uint) (*models.Conversation, bool, error) {
	if userID1 == userID2 {
		return nil, false, fmt.Errorf("不能与自己创建私聊会话")
	}

	// 确保 userID1 < userID2，以使查找具有确定性，避免重复会话
	if userID1 > userID2 {
		userID1, userID2 = userID2, userID1
	}

	conversation, err := s.convoRepo.FindPrivateConversationByUsers(ctx, userID1, userID2)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) { // 如果是 gorm.ErrRecordNotFound 以外的错误
		return nil, false, fmt.Errorf("查找私聊会话失败: %w", err)
	}

	if conversation != nil {
		return conversation, false, nil // 会话已存在
	}

	// 会话不存在，创建新的私聊会话
	newConversation := &models.Conversation{
		Type: models.PrivateConversation,
		// TargetID 对于私聊可以不设置，或根据需要定义其含义
	}
	if err := s.convoRepo.CreateConversation(ctx, newConversation); err != nil {
		return nil, true, fmt.Errorf("创建新会话失败: %w", err)
	}

	// 添加参与者
	p1 := &models.ConversationParticipant{ConversationID: newConversation.ID, UserID: userID1, JoinedAt: time.Now()}
	p2 := &models.ConversationParticipant{ConversationID: newConversation.ID, UserID: userID2, JoinedAt: time.Now()}

	if err := s.convoRepo.AddParticipant(ctx, p1); err != nil {
		// 注意：这里可能需要事务回滚，如果创建会话成功但添加参与者失败
		return newConversation, true, fmt.Errorf("为会话 %d 添加参与者 %d 失败: %w", newConversation.ID, userID1, err)
	}
	if err := s.convoRepo.AddParticipant(ctx, p2); err != nil {
		return newConversation, true, fmt.Errorf("为会话 %d 添加参与者 %d 失败: %w", newConversation.ID, userID2, err)
	}

	return newConversation, true, nil
}

// FixGroupConversationParticipants 为现有群组会话添加缺失的参与者
func (s *conversationService) FixGroupConversationParticipants(ctx context.Context, groupID uint) error {
	// 查找群组会话
	conversation, err := s.findGroupConversationByGroupID(ctx, groupID)
	if err != nil {
		return fmt.Errorf("获取群组会话失败: %w", err)
	}

	// 验证现有参与者
	existingParticipants, err := s.convoRepo.GetConversationParticipants(ctx, conversation.ID)
	if err != nil {
		return fmt.Errorf("获取现有参与者失败: %w", err)
	}

	fmt.Printf("修复会话参与者: 群组ID=%d, 会话ID=%d, 现有参与者数量=%d\n", groupID, conversation.ID, len(existingParticipants))

	// 如果已经有参与者，跳过处理
	if len(existingParticipants) > 0 {
		fmt.Printf("会话 %d 已存在 %d 个参与者，无需修复\n", conversation.ID, len(existingParticipants))
		return nil
	}

	// 获取群组成员
	groupMembers, err := s.userRepo.GetDB().WithContext(ctx).
		Table("group_members").
		Where("group_id = ? AND deleted_at IS NULL", groupID).
		Find(&[]struct{}{}).
		Rows()
	if err != nil {
		return fmt.Errorf("获取群组成员失败: %w", err)
	}
	defer groupMembers.Close()

	// 添加所有成员作为会话参与者
	memberCount := 0
	now := time.Now()

	for groupMembers.Next() {
		var member struct {
			UserID uint
			Role   string
		}
		if err := s.userRepo.GetDB().ScanRows(groupMembers, &member); err != nil {
			fmt.Printf("扫描群组成员行失败: %v\n", err)
			continue
		}

		// 添加该成员为会话参与者
		isAdmin := member.Role == "admin" || member.Role == "owner"
		participant := &models.ConversationParticipant{
			ConversationID: conversation.ID,
			UserID:         member.UserID,
			JoinedAt:       now,
			IsAdmin:        isAdmin,
		}

		if err := s.convoRepo.AddParticipant(ctx, participant); err != nil {
			fmt.Printf("添加参与者失败: UserID=%d, 错误: %v\n", member.UserID, err)
		} else {
			fmt.Printf("成功添加参与者: UserID=%d\n", member.UserID)
			memberCount++
		}
	}

	fmt.Printf("修复完成: 为会话 %d 添加了 %d 名参与者\n", conversation.ID, memberCount)
	return nil
}

// CreateGroupConversation 为群组创建会话
func (s *conversationService) CreateGroupConversation(ctx context.Context, groupID uint, creatorID uint, memberIDs []uint) (*models.Conversation, error) {
	if groupID == 0 {
		return nil, fmt.Errorf("群组ID不能为空")
	}

	// 打印详细日志
	fmt.Printf("[CreateGroupConversation] 开始创建群聊会话, 群组ID: %d, 创建者ID: %d, 成员数量: %d\n", groupID, creatorID, len(memberIDs))
	fmt.Printf("[CreateGroupConversation] 请求中的成员IDs: %v\n", memberIDs)

	// 首先检查群组和创建者是否存在
	// 检查创建者是否存在
	creator, err := s.userRepo.GetByID(ctx, creatorID)
	if err != nil {
		return nil, fmt.Errorf("创建者(ID=%d)不存在: %w", creatorID, err)
	}
	fmt.Printf("[CreateGroupConversation] 创建者验证通过: ID=%d, 用户名=%s\n", creator.ID, creator.Username)

	// 首先检查是否已经存在关联到此群组的会话
	existingConvo, err := s.findGroupConversationByGroupID(ctx, groupID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("检查群组会话是否存在时出错: %w", err)
	}

	if existingConvo != nil {
		fmt.Printf("[CreateGroupConversation] 群组 %d 已有会话 %d，跳过创建\n", groupID, existingConvo.ID)

		// 检查现有会话的参与者
		existingParticipants, err := s.convoRepo.GetConversationParticipants(ctx, existingConvo.ID)
		if err != nil {
			fmt.Printf("[CreateGroupConversation] 查询现有会话 %d 的参与者失败: %v\n", existingConvo.ID, err)
		} else {
			fmt.Printf("[CreateGroupConversation] 现有会话 %d 已有 %d 名参与者\n", existingConvo.ID, len(existingParticipants))
			for _, p := range existingParticipants {
				fmt.Printf("[CreateGroupConversation] 现有参与者: UserID=%d, IsAdmin=%v\n", p.UserID, p.IsAdmin)
			}

			// 如果没有参与者，尝试修复
			if len(existingParticipants) == 0 {
				fmt.Printf("[CreateGroupConversation] 检测到现有会话没有参与者，尝试修复...\n")
				// 先验证用户是否存在
				validMemberIDs, err := s.validateUsers(ctx, append(memberIDs, creatorID))
				if err != nil {
					return existingConvo, fmt.Errorf("验证用户失败: %w", err)
				}

				// 使用事务处理参与者添加
				tx := s.convoRepo.GetDB().Begin()
				if tx.Error != nil {
					return existingConvo, fmt.Errorf("开始事务失败: %w", tx.Error)
				}

				addedCount := 0
				for _, memberID := range validMemberIDs {
					// 再次验证每个用户是否存在
					_, userErr := s.userRepo.GetByID(ctx, memberID)
					if userErr != nil {
						fmt.Printf("[CreateGroupConversation] 用户 %d 不存在，跳过添加: %v\n", memberID, userErr)
						continue
					}

					isAdmin := memberID == creatorID
					participant := &models.ConversationParticipant{
						ConversationID: existingConvo.ID,
						UserID:         memberID,
						JoinedAt:       time.Now(),
						IsAdmin:        isAdmin,
					}

					// 检查是否已经存在
					var count int64
					if err := tx.WithContext(ctx).Model(&models.ConversationParticipant{}).
						Where("conversation_id = ? AND user_id = ?", existingConvo.ID, memberID).
						Count(&count).Error; err != nil {
						fmt.Printf("[CreateGroupConversation] 检查参与者是否存在失败: %v\n", err)
						continue
					}

					if count > 0 {
						fmt.Printf("[CreateGroupConversation] 参与者已存在: UserID=%d\n", memberID)
						addedCount++
						continue
					}

					if err := tx.WithContext(ctx).Create(&participant).Error; err != nil {
						fmt.Printf("[CreateGroupConversation] 修复添加参与者失败: UserID=%d, 错误: %v\n", memberID, err)
					} else {
						addedCount++
						fmt.Printf("[CreateGroupConversation] 修复添加参与者成功: UserID=%d\n", memberID)
					}
				}

				if addedCount > 0 {
					if err := tx.Commit().Error; err != nil {
						tx.Rollback()
						fmt.Printf("[CreateGroupConversation] 提交事务失败: %v\n", err)
					} else {
						fmt.Printf("[CreateGroupConversation] 成功修复会话参与者，添加了 %d 名成员\n", addedCount)
					}
				} else {
					tx.Rollback()
					fmt.Printf("[CreateGroupConversation] 未添加任何参与者，已回滚\n")
				}
			}
		}

		return existingConvo, nil
	}

	// 确保memberIDs包含有效值，并检查用户是否存在
	validMemberIDsWithCreator, err := s.validateUsers(ctx, append(memberIDs, creatorID))
	if err != nil {
		return nil, fmt.Errorf("验证用户失败: %w", err)
	}

	if len(validMemberIDsWithCreator) == 0 {
		return nil, fmt.Errorf("没有有效的成员ID")
	}

	// 开始数据库事务
	db := s.convoRepo.GetDB()
	tx := db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("开始事务失败: %w", tx.Error)
	}

	// 确保事务最终提交或回滚
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			fmt.Printf("[CreateGroupConversation] 发生异常，已回滚: %v\n", r)
		}
	}()

	// 创建新的群聊会话
	newConversation := &models.Conversation{
		Type:     models.GroupConversation,
		TargetID: groupID, // 关联到群组ID
	}

	if err := tx.WithContext(ctx).Create(newConversation).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建群聊会话失败: %w", err)
	}
	fmt.Printf("[CreateGroupConversation] 成功创建群聊会话, 会话ID: %d\n", newConversation.ID)

	// 创建参与者记录
	now := time.Now()
	participantCount := 0

	// 确保创建者存在并被添加
	creatorFound := false
	for _, id := range validMemberIDsWithCreator {
		if id == creatorID {
			creatorFound = true
			break
		}
	}

	if !creatorFound {
		tx.Rollback()
		return nil, fmt.Errorf("创建者(ID=%d)不在有效用户列表中", creatorID)
	}

	// 先添加创建者，确保他一定成功添加
	creatorParticipant := models.ConversationParticipant{
		ConversationID: newConversation.ID,
		UserID:         creatorID,
		JoinedAt:       now,
		IsAdmin:        true,
	}

	if err := tx.WithContext(ctx).Create(&creatorParticipant).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("添加创建者(ID=%d)作为参与者失败: %w", creatorID, err)
	}
	fmt.Printf("[CreateGroupConversation] 成功添加创建者: UserID=%d\n", creatorID)
	participantCount++

	// 添加其他成员
	for _, memberID := range validMemberIDsWithCreator {
		// 跳过创建者，已经添加过了
		if memberID == creatorID {
			continue
		}

		// 再次验证用户存在
		_, userErr := s.userRepo.GetByID(ctx, memberID)
		if userErr != nil {
			fmt.Printf("[CreateGroupConversation] 跳过不存在的用户: ID=%d, 错误: %v\n", memberID, userErr)
			continue
		}

		participant := models.ConversationParticipant{
			ConversationID: newConversation.ID,
			UserID:         memberID,
			JoinedAt:       now,
			IsAdmin:        false,
		}

		fmt.Printf("[CreateGroupConversation] 尝试添加参与者: UserID=%d, ConversationID=%d\n", memberID, newConversation.ID)

		if err := tx.WithContext(ctx).Create(&participant).Error; err != nil {
			fmt.Printf("[CreateGroupConversation] 添加参与者失败: UserID=%d, 错误: %v\n", memberID, err)
		} else {
			fmt.Printf("[CreateGroupConversation] 成功添加参与者: UserID=%d\n", memberID)
			participantCount++
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	fmt.Printf("[CreateGroupConversation] 群聊会话创建完成: 成功添加 %d 名成员\n", participantCount)

	// 验证记录是否真的添加到数据库
	verifyParticipants, verifyErr := s.convoRepo.GetConversationParticipants(ctx, newConversation.ID)
	if verifyErr != nil {
		fmt.Printf("[CreateGroupConversation] 验证参与者时出错: %v\n", verifyErr)
	} else {
		fmt.Printf("[CreateGroupConversation] 验证结果: 会话 %d 有 %d 名参与者\n", newConversation.ID, len(verifyParticipants))
		for i, p := range verifyParticipants {
			fmt.Printf("[CreateGroupConversation] 验证的参与者 #%d: UserID=%d, IsAdmin=%v\n", i+1, p.UserID, p.IsAdmin)
		}
	}

	return newConversation, nil
}

// validateUsers 验证用户是否存在并返回有效的用户ID列表
func (s *conversationService) validateUsers(ctx context.Context, userIDs []uint) ([]uint, error) {
	if len(userIDs) == 0 {
		return []uint{}, nil
	}

	// 去重
	userIDMap := make(map[uint]bool)
	var uniqueIDs []uint
	for _, id := range userIDs {
		if id > 0 && !userIDMap[id] {
			userIDMap[id] = true
			uniqueIDs = append(uniqueIDs, id)
		}
	}

	if len(uniqueIDs) == 0 {
		return []uint{}, nil
	}

	// 批量验证用户存在性
	var validIDs []uint
	var users []*models.User
	if err := s.userRepo.GetDB().WithContext(ctx).
		Model(&models.User{}).
		Where("id IN ?", uniqueIDs).
		Find(&users).Error; err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	// 提取找到的有效用户ID
	for _, user := range users {
		validIDs = append(validIDs, user.ID)
	}

	fmt.Printf("[validateUsers] 验证结果: %d/%d 个用户有效\n", len(validIDs), len(uniqueIDs))
	return validIDs, nil
}

// findGroupConversationByGroupID 查找与特定群组关联的会话
func (s *conversationService) findGroupConversationByGroupID(ctx context.Context, groupID uint) (*models.Conversation, error) {
	var conversation models.Conversation
	err := s.convoRepo.GetDB().WithContext(ctx).
		Where("type = ? AND target_id = ?", models.GroupConversation, groupID).
		First(&conversation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("查询群组会话失败: %w", err)
	}
	return &conversation, nil
}

// GetUserConversations 获取用户参与的所有会话列表。
func (s *conversationService) GetUserConversations(ctx context.Context, userID uint, limit, offset int) ([]*models.Conversation, error) {
	// TODO: 在 Repository 或 Service 层填充会话的最后一条消息、未读计数、以及对方用户信息等
	// 例如，通过遍历 conversations，对每个 conversation 调用 convoRepo.GetConversationParticipants 和 msgRepo.GetByID(LastMessageID)
	// 并根据需要组合成更丰富的会话列表项返回给前端。
	return s.convoRepo.GetUserConversations(ctx, userID, limit, offset)
}

// GetConversationDetails 获取会话的详细信息，包括参与者等。
func (s *conversationService) GetConversationDetails(ctx context.Context, conversationID uint, userID uint) (*models.Conversation, error) {
	// 1. 检查用户是否有权限查看此会话
	_, err := s.convoRepo.GetParticipant(ctx, conversationID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("用户 %d 不是会话 %d 的成员，无权查看", userID, conversationID)
		}
		return nil, fmt.Errorf("检查用户 %d 在会话 %d 中的参与情况失败: %w", userID, conversationID, err)
	}

	conversation, err := s.convoRepo.GetConversationByID(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("获取会话 %d 详情失败: %w", conversationID, err)
	}

	// TODO: 预加载或手动加载会话的参与者信息 (User) 和最后一条消息 (Message)
	// participants, _ := s.convoRepo.GetConversationParticipants(ctx, conversationID)
	// conversation.Participants = participants // GORM 的 Preload 更佳

	return conversation, nil
}

func (s *conversationService) GetConversationParticipants(ctx context.Context, conversationID uint) ([]*models.ConversationParticipant, error) {
	return s.convoRepo.GetConversationParticipants(ctx, conversationID)
}

// GetUserByID retrieves a user by their ID using the user repository.
func (s *conversationService) GetUserByID(ctx context.Context, userID uint) (*models.User, error) {
	if s.userRepo == nil {
		return nil, errors.New("userRepository is not initialized in conversationService")
	}
	return s.userRepo.GetByID(ctx, userID)
}
