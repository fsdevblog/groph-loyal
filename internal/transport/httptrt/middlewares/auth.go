package middlewares

import (
	"errors"
	"fmt"
	"github.com/fsdevblog/groph-loyal/internal/transport/httptrt/tokens"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
)

var ErrTokenNotExist = errors.New("token not exist")

const CurrentUserIDKey = "currentUserID"

// checkAuthorization извлекает токен из заголовка Authorization и проверяет его. Если токен не передан, вернется ошибка
// ErrTokenNotExist
func checkAuthorization(c *gin.Context, jwtTokenSecret []byte) (*jwt.Token, error) {
	tokenHeader := c.GetHeader("Authorization")
	bearer := "Bearer "

	if len(tokenHeader) < len(bearer) || tokenHeader[:len(bearer)] != bearer {
		return nil, ErrTokenNotExist
	}

	tokenStr := tokenHeader[len(bearer):]
	token, err := tokens.ValidateUserJWT(tokenStr, jwtTokenSecret)
	return token, fmt.Errorf("check authorization: %w", err)
}

// AuthRequiredMiddleware проверяет, что запрос авторизован. Записывает в контекст (поле CurrentUserIDKey)
// id юзера.
func AuthRequiredMiddleware(jwtTokenSecret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := checkAuthorization(c, jwtTokenSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			if !errors.Is(err, ErrTokenNotExist) {
				_ = c.Error(err).SetType(gin.ErrorTypePrivate)
			}
			return
		}
		userClaim, ok := token.Claims.(*tokens.UserClaims)
		if !ok {
			_ = c.AbortWithError(http.StatusInternalServerError, errors.New("invalid jwt claims type")).
				SetType(gin.ErrorTypePrivate)
			return
		}
		c.Set(CurrentUserIDKey, userClaim.ID)
		c.Next()
	}
}

// NonAuthRequiredMiddleware пропускает запросы без токена или с недействительным токеном.
func NonAuthRequiredMiddleware(jwtTokenSecret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, err := checkAuthorization(c, jwtTokenSecret)
		if err == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Already authorized"})
			return
		}

		c.Next()
	}
}
