package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword 使用 bcrypt 对密码进行哈希处理。
// cost 参数可以调整哈希的计算成本，默认通常是 10 或 12。
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash 验证提供的密码是否与其 bcrypt 哈希值匹配。
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
