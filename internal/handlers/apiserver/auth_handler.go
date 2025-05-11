package apiserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"im-go/internal/auth"       // Import for TokenBlacklist interface
	"im-go/internal/middleware" // Import for GetClaimsFromContext
	"im-go/internal/models"     // 用于 LoginResponse 中的 User
	"im-go/internal/services"
)

// AuthHandler 封装了认证相关的 HTTP 处理器方法。
type AuthHandler struct {
	AuthService    services.AuthService
	TokenBlacklist auth.TokenBlacklist // Added TokenBlacklist service
}

// NewAuthHandler 创建一个新的 AuthHandler 实例。
func NewAuthHandler(authService services.AuthService, tokenBlacklist auth.TokenBlacklist) *AuthHandler {
	return &AuthHandler{
		AuthService:    authService,
		TokenBlacklist: tokenBlacklist, // Store the injected service
	}
}

// RegisterRequest 是用户注册请求的结构体。
type RegisterRequest struct {
	Username string `json:"username"`
	Nickname string `json:"nickname"`        // 昵称必填
	Email    string `json:"email,omitempty"` // 邮箱可选
	Password string `json:"password"`
}

// LoginRequest 是用户登录请求的结构体。
type LoginRequest struct {
	UsernameOrEmail string `json:"username"` // 可以是用户名或邮箱
	Password        string `json:"password"`
}

// LoginResponse 是成功登录后返回的结构体。
type LoginResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"` // 返回一些用户信息，注意过滤敏感数据
}

// ErrorResponse 是 API 错误响应的通用结构体。
type ErrorResponse struct {
	Error string `json:"error"`
}

// Register 处理用户注册请求。
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "请求体无效", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.Username == "" || req.Password == "" || req.Nickname == "" {
		http.Error(w, "用户名、昵称和密码不能为空", http.StatusBadRequest)
		return
	}

	// TODO: 添加更严格的输入验证 (例如密码强度，用户名/邮箱格式)

	user, err := h.AuthService.Register(r.Context(), req.Username, req.Nickname, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, services.ErrUserAlreadyExists) {
			writeJSONError(w, err.Error(), http.StatusConflict)
		} else {
			// 对于其他内部错误，可能不想暴露详细信息给客户端
			writeJSONError(w, "注册失败", http.StatusInternalServerError)
		}
		return
	}

	// 注册成功，可以考虑直接登录并返回token，或者仅返回成功信息
	// 这里我们返回创建的用户信息（不含密码）
	user.PasswordHash = "" // 清除敏感信息
	writeJSONResponse(w, http.StatusCreated, user)
}

// Login 处理用户登录请求。
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "请求体无效", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.UsernameOrEmail == "" || req.Password == "" {
		http.Error(w, "用户名/邮箱和密码不能为空", http.StatusBadRequest)
		return
	}

	token, user, err := h.AuthService.Login(r.Context(), req.UsernameOrEmail, req.Password)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) || errors.Is(err, services.ErrInvalidCredentials) {
			writeJSONError(w, "用户名或密码错误", http.StatusUnauthorized)
		} else {
			writeJSONError(w, "登录失败", http.StatusInternalServerError)
		}
		return
	}

	user.PasswordHash = "" // 清除敏感信息
	writeJSONResponse(w, http.StatusOK, LoginResponse{Token: token, User: user})
}

// LogoutHandler 处理用户登出请求，将当前 Token 加入黑名单。
func (h *AuthHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		writeJSONError(w, "用户未认证或无法解析用户声明", http.StatusUnauthorized)
		return
	}

	if claims.ID == "" { // JTI (claims.ID)
		writeJSONError(w, "Token 缺少 JTI，无法执行登出", http.StatusInternalServerError)
		return
	}
	if claims.ExpiresAt == nil {
		writeJSONError(w, "Token 缺少过期时间，无法执行登出", http.StatusInternalServerError)
		return
	}

	originalExpiryTime := claims.ExpiresAt.Time
	err := h.TokenBlacklist.Add(r.Context(), claims.ID, originalExpiryTime)
	if err != nil {
		// log.Printf("将 Token 加入黑名单失败: %v", err) // Consider logging this error
		writeJSONError(w, "登出过程中发生内部错误", http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]string{"message": "登出成功"})
}

// writeJSONResponse 是一个辅助函数，用于发送 JSON 响应。
func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			// 如果编码失败，记录错误，但可能已经发送了头部，所以不能再写入 http.Error
			// 这种情况比较少见，除非 data 结构有问题
			// log.Printf("无法编码 JSON 响应: %v", err)
			return // 尝试返回前确保日志库已初始化
		}
	}
}

// writeJSONError 是一个辅助函数，用于发送 JSON 格式的错误响应。
func writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	writeJSONResponse(w, statusCode, ErrorResponse{Error: message})
}
