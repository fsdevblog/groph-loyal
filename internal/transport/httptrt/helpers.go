package httptrt

import (
	"unicode"

	"github.com/fsdevblog/groph-loyal/internal/transport/httptrt/middlewares"
	"github.com/gin-gonic/gin"
)

// getUserIDFromContext берет из контекста gin ID текущего юзера. ID устанавливается в
// middlewares.AuthRequiredMiddleware. В случае, если значения в контексте нет или ошибка утверждения типа -
// вернется 0.
func getUserIDFromContext(c *gin.Context) int64 {
	userIDStr, exist := c.Get(middlewares.CurrentUserIDKey)
	if !exist {
		return 0
	}
	userID, ok := userIDStr.(int64)
	if !ok {
		return 0
	}
	return userID
}

// isValidLuhn проверяет корректность строки по алгоритму Луна.
func isValidLuhn(code string) bool {
	var sum int
	maxDigit := 9
	double := false

	for i := len(code) - 1; i >= 0; i-- {
		char := code[i]

		if !unicode.IsDigit(rune(char)) {
			return false
		}

		digit := int(char - '0')

		if double {
			digit *= 2
			if digit > maxDigit {
				digit -= maxDigit
			}
		}
		sum += digit
		double = !double
	}

	// Код считается валидным, если сумма кратна 10
	return sum%10 == 0
}
