package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/fsdevblog/groph-loyal/internal/service/svsuser"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type AuthHandler struct {
	userService UserServicer
}

func NewAuthHandler(userService UserServicer) *AuthHandler {
	return &AuthHandler{
		userService: userService,
	}
}

// UserRegisterRequest запрос на регистрацию юзера.
type UserRegisterRequest struct {
	Username string `binding:"required,min=1,max=15"       json:"login"`
	Password string `binding:"required,min=6,max_bytes=72" json:"password"`
}

// Register POST RouteGroup + RegisterRoute. Регистрирует пользователя и аутентифицирует его.
func (h *AuthHandler) Register(c *gin.Context) {
	var params UserRegisterRequest
	if errBind := c.ShouldBindJSON(&params); errBind != nil {
		var errValidator validator.ValidationErrors
		if errors.As(errBind, &errValidator) {
			_ = c.AbortWithError(http.StatusUnprocessableEntity, errBind).
				SetType(gin.ErrorTypeBind)
			return
		}
		_ = c.AbortWithError(http.StatusBadRequest, errors.New("invalid request body")).
			SetType(gin.ErrorTypePublic)
		return
	}

	ctx, cancel := context.WithTimeout(c, DefaultServiceTimeout)
	defer cancel()

	_, jwtToken, errCreate := h.userService.Register(ctx, svsuser.RegisterUserArgs{
		Username: params.Username,
		Password: params.Password,
	})
	if errCreate != nil {
		if errors.Is(errCreate, domain.ErrDuplicateKey) {
			_ = c.AbortWithError(http.StatusConflict, errors.New("user with this login already exists")).
				SetType(gin.ErrorTypePublic)
			return
		}
		_ = c.AbortWithError(http.StatusInternalServerError, errCreate).
			SetType(gin.ErrorTypePrivate)
		return
	}

	c.Header("Authorization", "Bearer "+jwtToken)
	c.AbortWithStatus(http.StatusOK)
}

// UserLoginParams запрос на аутентификацию юзера.
type UserLoginParams struct {
	Username string `binding:"required,min=1,max=15"       json:"login"`
	Password string `binding:"required,min=6,max_bytes=72" json:"password"`
}

// UserResponse ответ при успешной аутентификации.
type UserResponse struct {
	ID        int64     `json:"ID"`
	Username  string    `json:"login"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Login POST RouteGroup + LoginRoute. Аутентификация по паре логин/пароль.
func (h *AuthHandler) Login(c *gin.Context) {
	var params UserLoginParams
	if errBind := c.ShouldBindJSON(&params); errBind != nil {
		_ = c.AbortWithError(http.StatusBadRequest, errBind).
			SetType(gin.ErrorTypeBind)
		return
	}

	ctx, cancel := context.WithTimeout(c, DefaultServiceTimeout)
	defer cancel()

	user, token, errLogin := h.userService.Login(ctx, svsuser.LoginUserArgs{
		Username: params.Username,
		Password: params.Password,
	})

	if errLogin != nil {
		if errors.Is(errLogin, domain.ErrRecordNotFound) || errors.Is(errLogin, domain.ErrPasswordMissMatch) {
			_ = c.AbortWithError(http.StatusUnauthorized, errors.New("invalid credentials")).
				SetType(gin.ErrorTypePublic)
			return
		}
		_ = c.AbortWithError(http.StatusInternalServerError, errLogin).SetType(gin.ErrorTypePrivate)
		return
	}
	c.Header("Authorization", "Bearer "+token)

	c.JSON(http.StatusOK, gin.H{"user": UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}})
}
