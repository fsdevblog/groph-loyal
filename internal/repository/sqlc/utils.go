package sqlc

import (
	"fmt"
	"math"
)

// safeConvertUintToInt32 безопасно конвертирует uint в int32. В случае выхода значения за рамки диапазона
// выбрасывает ошибку.
func safeConvertUintToInt32(val uint) (int32, error) {
	if val > uint(math.MaxInt32) {
		return 0, fmt.Errorf("value is out of range: %d", val)
	}
	return int32(val), nil
}

// safeConvertInt32ToUint безопасно конвертирует int32 в uint. В случае отрицательного значения выбрасывает ошибку.
func safeConvertInt32ToUint(val int32) (uint, error) {
	if val < 0 {
		return 0, fmt.Errorf("value is negative: %d", val)
	}
	return uint(val), nil
}
