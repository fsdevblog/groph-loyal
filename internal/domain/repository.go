package domain

import "context"

//go:generate mockgen -source=repository.go -destination=mocks/mocks.go -package=mocks
type RepositoryName string

const (
	UserRepoName RepositoryName = "user"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user User) (*User, error)
	FindUserByUsername(ctx context.Context, username string) (*User, error)
}
