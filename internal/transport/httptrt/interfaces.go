package httptrt

//go:generate mockgen -source=interfaces.go -destination=mocks/mocks.go -package=mocks

import (
	"context"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/service"
)

// UserServicer интерфейс исключительно для моков.
type UserServicer interface {
	Register(ctx context.Context, args service.RegisterUserArgs) (*domain.User, string, error)
	Login(ctx context.Context, args service.LoginUserArgs) (*domain.User, string, error)
}
