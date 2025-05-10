package middleware

import (
	"context"
	"net/http"
	"strings"

	"im-go/internal/auth"
	"im-go/internal/config"
	// "im-go/internal/handlers/apiserver" // 用于 writeJSONError，或者在中间件中定义类似的辅助函数
)

// contextKey 是用于在 context.Context 中存储值的自定义类型，以避免键冲突。
type contextKey string

// UserIDKey 是用于在上下文中存储用户ID的键。
const UserIDKey contextKey = "userID"

// UsernameKey 是用于在上下文中存储用户名的键。
const UsernameKey contextKey = "username"

// AuthMiddleware 是一个 HTTP 中间件，用于验证 JWT 并将用户信息添加到上下文中。
func AuthMiddleware(next http.Handler, authCfg config.AuthConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// writeJSONError(w, "请求未包含授权令牌", http.StatusUnauthorized)
			http.Error(w, "请求未包含授权令牌", http.StatusUnauthorized) // 简化版错误响应
			return
		}

		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) != 2 || strings.ToLower(headerParts[0]) != "bearer" {
			// writeJSONError(w, "授权头部格式无效，应为 Bearer {token}", http.StatusUnauthorized)
			http.Error(w, "授权头部格式无效", http.StatusUnauthorized) // 简化版错误响应
			return
		}

		tokenString := headerParts[1]
		claims, err := auth.ValidateToken(tokenString, authCfg.JWTSecretKey)
		if err != nil {
			// writeJSONError(w, fmt.Sprintf("令牌无效: %v", err), http.StatusUnauthorized)
			http.Error(w, "令牌无效", http.StatusUnauthorized) // 简化版错误响应
			return
		}

		// 将用户信息存入请求上下文
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UsernameKey, claims.Username)

		// 调用链中的下一个处理器，使用更新后的上下文
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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

/*
// writeJSONError (可以从 apiserver handler 复制或重新实现一个简版)
func writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
*/
