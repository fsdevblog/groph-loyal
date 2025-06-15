package tokens

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserClaims struct {
	jwt.RegisteredClaims
	ID int64
}

func GenerateUserJWT(id int64, expire time.Duration, key []byte) (string, error) {
	userClaims := UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expire)),
		},
		ID: id,
	}
	token, err := generateJWT(userClaims, key)
	if err != nil {
		return "", fmt.Errorf("generating user jwt token: %s", err.Error())
	}
	return token, nil
}

func ValidateUserJWT(tokenString string, key []byte) (*jwt.Token, error) {
	token, err := validateJWT(tokenString, new(UserClaims), key)
	if err != nil {
		return nil, fmt.Errorf("validating user jwt token: %w", err)
	}

	_, ok := token.Claims.(*UserClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return token, nil
}

func generateJWT(claims jwt.Claims, key []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("generating jwt token: %s", err.Error())
	}

	return tokenString, nil
}

func validateJWT(tokenString string, claims jwt.Claims, key []byte) (*jwt.Token, error) {
	token, err := jwt.ParseWithClaims(tokenString, claims, func(_ *jwt.Token) (any, error) {
		return key, nil
	}, jwt.WithValidMethods([]string{"HS256"}))

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("parsing jwt token `%s`: %w", tokenString, err)
	}

	return token, nil
}
