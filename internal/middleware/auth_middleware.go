package middleware

import (
	"context"
	"net/http"
	"strings"

	"im-go/internal/auth"
	// "im-go/internal/config" // 假设 jwtSecret 会直接传入
	// "encoding/json" // 如果需要 writeJSONError
	// "fmt"           // 如果需要 writeJSONError
)

// contextKey 是用于在 context.Context 中存储值的自定义类型，以避免键冲突。
type contextKey string

// UserIDKey 是用于在上下文中存储用户ID的键。
const UserIDKey contextKey = "userID"

// UsernameKey 是用于在上下文中存储用户名的键。
const UsernameKey contextKey = "username"

// JTIKey 是用于在上下文中存储 JTI (JWT ID) 的键。
const JTIKey contextKey = "jti"

// ClaimsKey 是用于在上下文中存储完整 Claims 对象的键。
const ClaimsKey contextKey = "claims"

// AuthMiddleware 创建一个用于 gorilla/mux 的 HTTP 中间件，用于 JWT 认证。
// jwtSecret 是用于验证 Token 的密钥。
// blacklist 是 TokenBlacklist 接口的实例。
func AuthMiddleware(jwtSecret string, blacklist auth.TokenBlacklist) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeJSONError(w, "请求未包含认证 Token", http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if !(len(parts) == 2 && parts[0] == "Bearer") {
				writeJSONError(w, "认证 Token 格式错误", http.StatusUnauthorized)
				return
			}
			tokenString := parts[1]

			claims, err := auth.ValidateToken(r.Context(), tokenString, jwtSecret, blacklist)
			if err != nil {
				writeJSONError(w, "Token 无效、已过期或已被吊销: "+err.Error(), http.StatusUnauthorized)
				return
			}

			// 将用户信息和 JTI 存入请求上下文
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UsernameKey, claims.Username)
			ctx = context.WithValue(ctx, JTIKey, claims.ID) // 存储 JTI
			ctx = context.WithValue(ctx, ClaimsKey, claims) // 存储完整的 Claims

			// 调用链中的下一个处理器，使用更新后的上下文
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserIDFromContext 从上下文中获取用户ID。
// 如果用户ID不存在或类型不正确，返回0和false。
func GetUserIDFromContext(ctx context.Context) (uint, bool) {
	userID, ok := ctx.Value(UserIDKey).(uint)
	return userID, ok
}

// GetUsernameFromContext 从上下文中获取用户名。
// 如果用户名不存在或类型不正确，返回空字符串和false。
func GetUsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(UsernameKey).(string)
	return username, ok
}

// GetJTIFromContext 从上下文中获取 JTI。
// 如果 JTI 不存在或类型不正确，返回空字符串和false。
func GetJTIFromContext(ctx context.Context) (string, bool) {
	jti, ok := ctx.Value(JTIKey).(string)
	return jti, ok
}

// GetClaimsFromContext 从上下文中获取完整的 Claims 对象。
// 如果 Claims 不存在或类型不正确，返回 nil 和 false。
func GetClaimsFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(ClaimsKey).(*auth.Claims)
	return claims, ok
}

// writeJSONError 是一个辅助函数，用于发送 JSON 格式的错误响应。
// 注意：为了简单起见，这里没有导入 "encoding/json"。如果需要，请取消注释并导入。
func writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff") // 安全头部
	w.WriteHeader(statusCode)
	// 为了避免导入 encoding/json，这里手动构建一个简单的 JSON 字符串
	// 在实际应用中，使用 json.NewEncoder(w).Encode(...) 会更好
	jsonResponse := `{"error":"` + strings.ReplaceAll(message, `"`, `\"`) + `"}`
	w.Write([]byte(jsonResponse))
}
