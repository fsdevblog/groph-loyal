package testutils

import "strings"

// GenerateOverBytesUnderRunes генерирует строку, длина которой в рунах будет всегда меньше длины в байтах.
func GenerateOverBytesUnderRunes(count int) string {
	symbol := "😁" // 4 байта, 1 руна
	return strings.Repeat(symbol, count)
}
