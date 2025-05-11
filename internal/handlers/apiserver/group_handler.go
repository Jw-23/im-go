package apiserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"im-go/internal/middleware"
	"im-go/internal/models"
	"im-go/internal/services"

	"github.com/gorilla/mux"
)

// GroupHandler 封装了群组相关的 HTTP 处理器方法。
type GroupHandler struct {
	groupService   services.GroupService
	convoService   services.ConversationService
	requestLock    sync.Mutex
	recentRequests map[string]time.Time
}

// NewGroupHandler 创建一个新的 GroupHandler 实例。
func NewGroupHandler(groupService services.GroupService, convoService services.ConversationService) *GroupHandler {
	return &GroupHandler{
		groupService:   groupService,
		convoService:   convoService,
		recentRequests: make(map[string]time.Time),
	}
}

// 检查是否是重复请求
func (h *GroupHandler) isRecentRequest(key string) bool {
	h.requestLock.Lock()
	defer h.requestLock.Unlock()

	// 清理过期的请求记录
	now := time.Now()
	for k, t := range h.recentRequests {
		if now.Sub(t) > 1*time.Minute {
			delete(h.recentRequests, k)
		}
	}

	// 检查请求是否已存在
	if _, exists := h.recentRequests[key]; exists {
		return true
	}

	// 记录新请求
	h.recentRequests[key] = now
	return false
}

// CreateGroupRequest 是创建群组的请求结构体。
type CreateGroupRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	AvatarURL     string `json:"avatarUrl,omitempty"`
	IsPublic      bool   `json:"isPublic"`
	JoinCondition string `json:"joinCondition,omitempty"` // 例如 models.DirectJoin, models.ApprovalRequired
	MemberIds     []uint `json:"memberIds,omitempty"`     // 初始成员ID列表
}

// CreateGroupHandler 处理创建新群组的请求。
func (h *GroupHandler) CreateGroupHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, "用户未认证", http.StatusUnauthorized)
		return
	}

	var req CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "请求体无效", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 添加更详细的日志
	fmt.Printf("收到创建群组请求: 名称=%s, 成员数量=%d, 创建者ID=%d\n", req.Name, len(req.MemberIds), userID)
	fmt.Printf("成员IDs原始数据: %#v，类型: %T\n", req.MemberIds, req.MemberIds)

	// 确保MemberIds是有效的数字
	var validMembers []uint
	for i, id := range req.MemberIds {
		if id > 0 {
			validMembers = append(validMembers, id)
			fmt.Printf("有效成员 #%d: ID=%d\n", i+1, id)
		} else {
			fmt.Printf("忽略无效成员ID: %d 在位置 %d\n", id, i)
		}
	}

	// 验证创建者自己是否也在成员列表中
	creatorInList := false
	for _, id := range validMembers {
		if id == userID {
			creatorInList = true
			break
		}
	}
	if !creatorInList {
		fmt.Printf("创建者 (ID=%d) 不在成员列表中，将自动添加\n", userID)
		validMembers = append(validMembers, userID)
	}

	// 更新请求中的成员ID列表
	req.MemberIds = validMembers
	fmt.Printf("处理后的有效成员IDs: %v, 总数: %d\n", req.MemberIds, len(req.MemberIds))

	// 生成请求唯一标识
	requestKey := fmt.Sprintf("create_group:%d:%s:%v", userID, req.Name, req.MemberIds)
	if h.isRecentRequest(requestKey) {
		fmt.Printf("检测到重复请求: %s\n", requestKey)
		writeJSONError(w, "请求过于频繁，请稍后再试", http.StatusTooManyRequests)
		return
	}

	if req.Name == "" {
		writeJSONError(w, "群组名称不能为空", http.StatusBadRequest)
		return
	}

	group, err := h.groupService.CreateGroup(r.Context(), userID, req.Name, req.Description, req.AvatarURL, req.IsPublic, req.JoinCondition)
	if err != nil {
		writeJSONError(w, fmt.Sprintf("创建群组失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 创建群聊会话
	fmt.Printf("开始为群组 %d 创建会话, 创建者ID: %d, 有效成员数: %d\n", group.ID, userID, len(req.MemberIds))
	fmt.Printf("传递给CreateGroupConversation的成员IDs: %v\n", req.MemberIds)

	convo, err := h.convoService.CreateGroupConversation(r.Context(), group.ID, userID, req.MemberIds)
	if err != nil {
		// 记录错误但不影响群组创建成功的响应
		fmt.Printf("为群组 %d 创建会话失败: %v\n", group.ID, err)
		// 也可以考虑在这里回滚群组创建
	} else {
		fmt.Printf("成功为群组 %d 创建会话 %d\n", group.ID, convo.ID)

		// 额外验证：检查参与者是否真的被添加
		participants, participantsErr := h.convoService.GetConversationParticipants(r.Context(), convo.ID)
		if participantsErr != nil {
			fmt.Printf("获取会话 %d 参与者失败: %v\n", convo.ID, participantsErr)
		} else {
			fmt.Printf("会话 %d 有 %d 名参与者:\n", convo.ID, len(participants))
			for i, p := range participants {
				fmt.Printf("参与者 #%d: UserID=%d, IsAdmin=%v, JoinedAt=%v\n",
					i+1, p.UserID, p.IsAdmin, p.JoinedAt)
			}
		}
	}

	writeJSONResponse(w, http.StatusCreated, group)
}

// GetGroupDetailsHandler 获取群组详情。
func (h *GroupHandler) GetGroupDetailsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupIDStr, _ := vars["groupID"]
	groupID, err := strconv.ParseUint(groupIDStr, 10, 32)
	if err != nil {
		writeJSONError(w, "无效的群组ID格式", http.StatusBadRequest)
		return
	}

	// userID 用于权限检查 (例如，私有群组是否是成员)
	// 对于公开群组，userID 可能不是必需的，或者用于个性化视图
	userID, _ := middleware.GetUserIDFromContext(r.Context()) // 可选认证

	group, err := h.groupService.GetGroupDetails(r.Context(), uint(groupID), userID)
	if err != nil {
		writeJSONError(w, fmt.Sprintf("获取群组详情失败: %v", err), http.StatusNotFound)
		return
	}
	writeJSONResponse(w, http.StatusOK, group)
}

// JoinGroupHandler 处理用户加入群组的请求。
func (h *GroupHandler) JoinGroupHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, "用户未认证", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	groupIDStr, _ := vars["groupID"]
	groupID, err := strconv.ParseUint(groupIDStr, 10, 32)
	if err != nil {
		writeJSONError(w, "无效的群组ID格式", http.StatusBadRequest)
		return
	}

	member, err := h.groupService.JoinGroup(r.Context(), userID, uint(groupID))
	if err != nil {
		// 根据错误类型返回不同状态码，例如冲突 (已是成员) 或禁止 (不允许加入)
		writeJSONError(w, fmt.Sprintf("加入群组失败: %v", err), http.StatusInternalServerError) // 简化处理
		return
	}
	writeJSONResponse(w, http.StatusOK, member)
}

// LeaveGroupHandler 处理用户离开群组的请求。
func (h *GroupHandler) LeaveGroupHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, "用户未认证", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	groupIDStr, _ := vars["groupID"]
	groupID, err := strconv.ParseUint(groupIDStr, 10, 32)
	if err != nil {
		writeJSONError(w, "无效的群组ID格式", http.StatusBadRequest)
		return
	}

	if err := h.groupService.LeaveGroup(r.Context(), userID, uint(groupID)); err != nil {
		writeJSONError(w, fmt.Sprintf("离开群组失败: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSONResponse(w, http.StatusOK, map[string]string{"message": "成功离开群组"})
}

// GetGroupMembersHandler 获取群组成员列表。
func (h *GroupHandler) GetGroupMembersHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupIDStr, _ := vars["groupID"]
	groupID, err := strconv.ParseUint(groupIDStr, 10, 32)
	if err != nil {
		writeJSONError(w, "无效的群组ID格式", http.StatusBadRequest)
		return
	}
	// TODO: 分页参数
	limit := 100
	offset := 0
	members, err := h.groupService.GetGroupMembers(r.Context(), uint(groupID), limit, offset)
	if err != nil {
		writeJSONError(w, fmt.Sprintf("获取群组成员失败: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSONResponse(w, http.StatusOK, members)
}

// SearchPublicGroupsHandler 处理搜索公开群组的请求。
func (h *GroupHandler) SearchPublicGroupsHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q") // 从查询参数获取搜索词
	// TODO: 从查询参数获取分页信息 (limit, offset)
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)
	if limit <= 0 || limit > 100 { // 设置默认和最大限制
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	groups, err := h.groupService.SearchPublicGroups(r.Context(), query, limit, offset)
	if err != nil {
		writeJSONError(w, fmt.Sprintf("搜索群组失败: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSONResponse(w, http.StatusOK, groups)
}

// FixGroupConversationParticipants 处理修复群组会话参与者的请求
func (h *GroupHandler) FixGroupConversationParticipants(w http.ResponseWriter, r *http.Request) {
	// 从URL中获取群组ID
	vars := mux.Vars(r)
	groupIDStr, ok := vars["id"]
	if !ok {
		RespondWithError(w, http.StatusBadRequest, "缺少群组ID")
		return
	}

	// 转换群组ID为uint
	groupID, err := strconv.ParseUint(groupIDStr, 10, 64)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "无效的群组ID")
		return
	}

	// 从上下文中获取当前用户ID
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		RespondWithError(w, http.StatusUnauthorized, "未授权的请求")
		return
	}

	// 检查用户是否有权限修复群组会话
	group, err := h.groupService.GetGroupDetails(r.Context(), uint(groupID), userID)
	if err != nil {
		RespondWithError(w, http.StatusNotFound, "找不到群组")
		return
	}

	// 检查用户是否是群主或管理员
	member, err := h.groupService.(interface {
		GetMember(context.Context, uint, uint) (*models.GroupMember, error)
	}).GetMember(r.Context(), uint(groupID), userID)
	if err != nil || (member.Role != models.AdminRole) {
		RespondWithError(w, http.StatusForbidden, "您没有权限修复此群组会话")
		return
	}

	// 调用修复服务
	err = h.convoService.FixGroupConversationParticipants(r.Context(), uint(groupID))
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "修复群组会话失败: "+err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message":   "成功修复群组会话参与者",
		"groupID":   groupID,
		"groupName": group.Name,
	})
}

// RespondWithError 返回错误响应
func RespondWithError(w http.ResponseWriter, statusCode int, message string) {
	writeJSONResponse(w, statusCode, map[string]string{"error": message})
}

// TODO: 添加其他群组管理相关的 Handler，如 UpdateGroupInfo, UpdateMemberRole 等。
