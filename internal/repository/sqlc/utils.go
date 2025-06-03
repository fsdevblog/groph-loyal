package sqlc

import "errors"

// safeConvertInt32ToUint безопасно конвертирует int32 в uint. Если int32 выходит
// за рамки диапазона uint - выбрасывает ошибку.
func safeConvertInt32ToUint(value int32) (uint, error) {
	if value < 0 {
		return 0, errors.New("value is negative: cannot convert to uint")
	}
	return uint(value), nil
}
