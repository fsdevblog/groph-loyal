package trhttp

import (
	"errors"
	"net/http"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/service"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	userService UserServicer
}

func NewAuthHandler(userService UserServicer) *AuthHandler {
	return &AuthHandler{
		userService: userService,
	}
}

type RegisterParams struct {
	Username string `json:"login"`
	Password string `json:"password"`
}

// Register POST /api/user/register. Регистрирует пользователя и аутентифицирует его.
func (h *AuthHandler) Register(c *gin.Context) {
	var params RegisterParams
	if bindErr := c.ShouldBindJSON(&params); bindErr != nil {
		_ = c.AbortWithError(http.StatusBadRequest, bindErr)
		return
	}

	user, createErr := h.userService.Register(c, service.RegisterUserArgs{
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

	// TODO: Аутентификация после регистрации.
	c.JSON(http.StatusOK, gin.H{"user": user})
}
