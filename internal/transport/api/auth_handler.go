package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/service"
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

type UserRegisterParams struct {
	Username string `binding:"required,min=1,max=15"  form:"login"    json:"login"`
	Password string `binding:"required,min=6,max=255" form:"password" json:"password"`
}

type UserRegisterResponse struct {
	ID       int64  `json:"id"`
	Username string `json:"login"`
}

// Register POST RouteGroup + RegisterRoute. Регистрирует пользователя и аутентифицирует его.
func (h *AuthHandler) Register(c *gin.Context) {
	var params UserRegisterParams
	if bindErr := c.ShouldBindJSON(&params); bindErr != nil {
		var valErrs validator.ValidationErrors
		if errors.As(bindErr, &valErrs) {
			c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": valErrs})
			return
		}
		_ = c.AbortWithError(http.StatusBadRequest, bindErr).
			SetType(gin.ErrorTypeBind)
		return
	}

	ctx, cancel := context.WithTimeout(c, DefaultServiceTimeout)
	defer cancel()

	user, jwtToken, createErr := h.userService.Register(ctx, service.RegisterUserArgs{
		Username: params.Username,
		Password: params.Password,
	})
	if createErr != nil {
		if errors.Is(createErr, domain.ErrDuplicateKey) {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "user already exists"})
			return
		}
		_ = c.AbortWithError(http.StatusInternalServerError, createErr)
		return
	}

	response := UserRegisterResponse{
		ID:       user.ID,
		Username: user.Username,
	}
	c.Header("Authorization", "Bearer "+jwtToken)
	c.JSON(http.StatusOK, gin.H{"user": response})
}

type UserLoginParams struct {
	Username string `form:"login"    json:"login"`
	Password string `form:"password" json:"password"`
}

type UserResponse struct {
	ID        int64     `json:"ID"`
	Username  string    `json:"login"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Login POST RouteGroup + LoginRoute. Аутентификация по паре логин/пароль.
func (h *AuthHandler) Login(c *gin.Context) {
	var params UserLoginParams
	if bindErr := c.ShouldBindJSON(&params); bindErr != nil {
		_ = c.AbortWithError(http.StatusBadRequest, bindErr).
			SetType(gin.ErrorTypeBind)
		return
	}

	ctx, cancel := context.WithTimeout(c, DefaultServiceTimeout)
	defer cancel()

	user, token, err := h.userService.Login(ctx, service.LoginUserArgs{
		Username: params.Username,
		Password: params.Password,
	})

	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) || errors.Is(err, domain.ErrPasswordMissMatch) {
			_ = c.Error(err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		_ = c.AbortWithError(http.StatusInternalServerError, err).SetType(gin.ErrorTypePublic)
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
