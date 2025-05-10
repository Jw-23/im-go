package storage

import (
	"strconv"
)

// StrToUint 将字符串转换为 uint。
// 如果转换失败，它会返回 0 和错误。
// 在服务层中，你可能需要更健壮的错误处理或根据业务逻辑返回默认值。
func StrToUint(s string) (uint, error) {
	val, err := strconv.ParseUint(s, 10, 32) // 32 表示结果适合 uint32，uint 在 Go 中通常是 32 或 64位
	if err != nil {
		return 0, err
	}
	return uint(val), nil
}
