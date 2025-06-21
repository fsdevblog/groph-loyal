package svsuser

//go:generate mockgen -source=interfaces.go -destination=mocks/mocks.go -package=mocks

import (
	"context"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/repository/repoargs"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user repoargs.CreateUser) (*domain.User, error)
	FindUserByUsername(ctx context.Context, username string) (*domain.User, error)
}

type PasswordHasher interface {
	HashPassword(password string) (string, error)
	ComparePassword(password string, hashedPassword string) bool
}
