package auth

import (
	"fmt"
	"time"

	"im-go/internal/config" // 用于 JWT 配置常量，或者直接传递配置值

	"github.com/golang-jwt/jwt/v5"
)

// Claims 是 JWT 中的自定义声明，嵌入了 jwt.RegisteredClaims。
// RegisteredClaims 包含标准的声明如 Issuer, Subject, Audience, ExpiresAt, NotBefore, IssuedAt, JWT ID。
type Claims struct {
	UserID   uint   `json:"userId"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken 为指定用户生成一个新的 JWT。
// jwtKey 是用于签发令牌的密钥。
// expiry 是令牌的有效期。
func GenerateToken(userID uint, username string, authCfg config.AuthConfig) (string, error) {
	expirationTime := time.Now().Add(authCfg.JWTExpiry)
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "im-go-server", // 可以从配置中读取或硬编码
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(authCfg.JWTSecretKey))
	if err != nil {
		return "", fmt.Errorf("生成 JWT 失败: %w", err)
	}
	return tokenString, nil
}

// ValidateToken 验证给定的 JWT 字符串的有效性。
// 如果令牌有效，它会返回 Claims。否则返回错误。
// jwtKey 是用于验证令牌签名的密钥。
func ValidateToken(tokenString string, jwtKey string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// 确保签名算法是我们期望的
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("非预期的签名算法: %v", token.Header["alg"])
		}
		return []byte(jwtKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("解析或验证 JWT 失败: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("JWT 无效")
	}

	return claims, nil
}
