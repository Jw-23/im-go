package redis

import (
	"context"
	"fmt"
	"time"

	"im-go/internal/auth" // 确保路径正确，引用 TokenBlacklist 接口

	"github.com/redis/go-redis/v9"
)

// redisTokenBlacklist 是 auth.TokenBlacklist 接口的 Redis 实现
type redisTokenBlacklist struct {
	client *redis.Client
}

// NewRedisTokenBlacklist 创建一个新的 redisTokenBlacklist 实例。
func NewRedisTokenBlacklist(client *redis.Client) auth.TokenBlacklist {
	return &redisTokenBlacklist{client: client}
}

const blacklistKeyPrefix = "bl:jti:"

// Add 将 jti 加入黑名单，并设置其在 Redis 中的过期时间为 Token 的原始过期时间点。
func (r *redisTokenBlacklist) Add(ctx context.Context, jti string, originalTokenExpTime time.Time) error {
	duration := time.Until(originalTokenExpTime)

	if duration <= 0 {
		// Token 已经过期，理论上不需要加入黑名单。
		// 返回 nil 表示操作完成，因为 JWT 验证本身会处理已过期的 Token。
		return nil
	}

	key := blacklistKeyPrefix + jti
	// 在 Redis 中存储一个简单的值 (例如 "1" 或 "revoked")，并设置过期时间。
	err := r.client.Set(ctx, key, "revoked", duration).Err()
	if err != nil {
		return fmt.Errorf("添加到 Redis 黑名单失败 for JTI %s: %w", jti, err)
	}
	return nil
}

// IsBlacklisted 检查 jti 是否在黑名单中。
func (r *redisTokenBlacklist) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	key := blacklistKeyPrefix + jti
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil // Key 不存在，因此不在黑名单中
	}
	if err != nil {
		return false, fmt.Errorf("从 Redis 黑名单检查失败 for JTI %s: %w", jti, err)
	}
	// 如果键存在并且有值 (例如我们设置的 "revoked")，则视为在黑名单中。
	// 也可以简单地通过检查 err == nil (即键存在) 来判断。
	return val == "revoked", nil // 或者 return true, nil 如果仅检查存在性
}
