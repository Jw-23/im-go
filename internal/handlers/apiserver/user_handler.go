package apiserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"im-go/internal/middleware"
	"im-go/internal/services"

	"github.com/gorilla/mux" // 用于从路径参数中提取 userID
)

// UserHandler 封装了用户相关的 HTTP 处理器方法。
type UserHandler struct {
	userService services.UserService
}

// NewUserHandler 创建一个新的 UserHandler 实例。
func NewUserHandler(userService services.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// GetMyProfileHandler 处理获取当前登录用户信息的请求。
func (h *UserHandler) GetMyProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, "无法从上下文中获取用户ID，请确保请求已通过认证", http.StatusInternalServerError)
		return
	}

	user, err := h.userService.GetUserProfile(r.Context(), userID)
	if err != nil {
		// 假设 GetUserProfile 内部会处理 ErrRecordNotFound 并返回合适的错误
		writeJSONError(w, fmt.Sprintf("获取用户信息失败: %v", err), http.StatusNotFound)
		return
	}
	writeJSONResponse(w, http.StatusOK, user)
}

// UpdateMyProfileRequest 是更新用户信息的请求结构体。
type UpdateMyProfileRequest struct {
	Nickname  string `json:"nickname,omitempty"`
	AvatarURL string `json:"avatarUrl,omitempty"`
	Bio       string `json:"bio,omitempty"`
}

// UpdateMyProfileHandler 处理更新当前登录用户信息的请求。
func (h *UserHandler) UpdateMyProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, "无法从上下文中获取用户ID", http.StatusInternalServerError)
		return
	}

	var req UpdateMyProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "请求体无效", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	user, err := h.userService.UpdateUserProfile(r.Context(), userID, req.Nickname, req.AvatarURL, req.Bio)
	if err != nil {
		writeJSONError(w, fmt.Sprintf("更新用户信息失败: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSONResponse(w, http.StatusOK, user)
}

// GetUserProfileHandler 处理获取指定用户公开信息的请求。
func (h *UserHandler) GetUserProfileHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userIDStr, ok := vars["userID"]
	if !ok {
		writeJSONError(w, "请求路径中缺少 userID", http.StatusBadRequest)
		return
	}
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		writeJSONError(w, "无效的用户ID格式", http.StatusBadRequest)
		return
	}

	user, err := h.userService.GetUserProfile(r.Context(), uint(userID))
	if err != nil {
		writeJSONError(w, fmt.Sprintf("获取用户信息失败: %v", err), http.StatusNotFound)
		return
	}
	// 确保不返回敏感信息，GetUserProfile 内部应该已经处理
	writeJSONResponse(w, http.StatusOK, user)
}

// SearchUsersHandler 处理搜索用户的请求。
func (h *UserHandler) SearchUsersHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, "无法从上下文中获取用户ID", http.StatusInternalServerError)
		return
	}

	query := r.URL.Query().Get("query")
	if strings.TrimSpace(query) == "" {
		writeJSONError(w, "搜索查询不能为空", http.StatusBadRequest)
		return
	}

	if len(query) < 2 { // Example: minimum query length
		writeJSONError(w, "搜索查询太短 (至少2个字符)", http.StatusBadRequest)
		return
	}

	users, err := h.userService.SearchUsers(r.Context(), query, userID)
	if err != nil {
		log.Printf("Error in SearchUsersHandler while calling service: %v", err)
		writeJSONError(w, "搜索用户时出错", http.StatusInternalServerError)
		return
	}

	// Define a DTO for search results to control returned fields
	type UserSearchResultDTO struct {
		ID        uint   `json:"id"`
		Username  string `json:"username"`
		Nickname  string `json:"nickname,omitempty"`
		AvatarURL string `json:"avatarUrl,omitempty"`
	}

	resultsDTO := make([]UserSearchResultDTO, len(users))
	for i, u := range users {
		resultsDTO[i] = UserSearchResultDTO{
			ID:        u.ID,
			Username:  u.Username,
			Nickname:  u.Nickname,
			AvatarURL: u.AvatarURL,
		}
	}

	writeJSONResponse(w, http.StatusOK, resultsDTO)
}
