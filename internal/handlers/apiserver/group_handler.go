package apiserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"im-go/internal/middleware"
	"im-go/internal/services"

	"github.com/gorilla/mux"
)

// GroupHandler 封装了群组相关的 HTTP 处理器方法。
type GroupHandler struct {
	groupService services.GroupService
}

// NewGroupHandler 创建一个新的 GroupHandler 实例。
func NewGroupHandler(groupService services.GroupService) *GroupHandler {
	return &GroupHandler{groupService: groupService}
}

// CreateGroupRequest 是创建群组的请求结构体。
type CreateGroupRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	AvatarURL     string `json:"avatarUrl,omitempty"`
	IsPublic      bool   `json:"isPublic"`
	JoinCondition string `json:"joinCondition,omitempty"` // 例如 models.DirectJoin, models.ApprovalRequired
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

	if req.Name == "" {
		writeJSONError(w, "群组名称不能为空", http.StatusBadRequest)
		return
	}

	group, err := h.groupService.CreateGroup(r.Context(), userID, req.Name, req.Description, req.AvatarURL, req.IsPublic, req.JoinCondition)
	if err != nil {
		writeJSONError(w, fmt.Sprintf("创建群组失败: %v", err), http.StatusInternalServerError)
		return
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

// TODO: 添加其他群组管理相关的 Handler，如 UpdateGroupInfo, UpdateMemberRole 等。
