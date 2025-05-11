package auth

import (
	"context"
	"time"
)

// TokenBlacklist 定义了 Token 黑名单的存储操作接口
type TokenBlacklist interface {
	// Add 将 jti 加入黑名单，并使其在 Token 的原始过期时间点之后自动从黑名单中移除。
	Add(ctx context.Context, jti string, originalTokenExpTime time.Time) error
	// IsBlacklisted 检查 jti 是否存在于黑名单中。
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
}
