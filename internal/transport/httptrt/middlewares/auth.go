package middlewares

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/fsdevblog/groph-loyal/internal/transport/httptrt/tokens"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var ErrTokenNotExist = errors.New("token not exist")

const CurrentUserIDKey = "currentUserID"

// checkAuthorization извлекает токен из заголовка Authorization и проверяет его. Если токен не передан, вернется ошибка
// ErrTokenNotExist.
func checkAuthorization(c *gin.Context, jwtTokenSecret []byte) (*jwt.Token, error) {
	tokenHeader := c.GetHeader("Authorization")
	bearer := "Bearer "

	if len(tokenHeader) < len(bearer) || tokenHeader[:len(bearer)] != bearer {
		return nil, ErrTokenNotExist
	}

	tokenStr := tokenHeader[len(bearer):]
	token, err := tokens.ValidateUserJWT(tokenStr, jwtTokenSecret)
	if err != nil {
		return nil, fmt.Errorf("check authorization: %w", err)
	}
	return token, nil
}

// AuthRequiredMiddleware проверяет, что запрос авторизован. Записывает в контекст (поле CurrentUserIDKey)
// id юзера.
func AuthRequiredMiddleware(jwtTokenSecret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := checkAuthorization(c, jwtTokenSecret)
		if err != nil {
			_ = c.AbortWithError(http.StatusUnauthorized, errors.New("auth required")).
				SetType(gin.ErrorTypePublic)
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
			_ = c.AbortWithError(http.StatusUnauthorized, errors.New("you are already logged in")).
				SetType(gin.ErrorTypePublic)
			return
		}

		c.Next()
	}
}
