package httptrt

import (
	"context"
	"errors"
	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"net/http"
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
	Username string `json:"login" form:"login" binding:"required,min=1,max=15"`
	Password string `json:"password" form:"password" binding:"required,min=6,max=255"`
}

type UserRegisterResponse struct {
	ID       int64  `json:"id"`
	Username string `json:"login"`
}

// Register POST APIRouteGroup + APIRegisterRoute. Регистрирует пользователя и аутентифицирует его.
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
