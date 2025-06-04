package domain

import "context"

//go:generate mockgen -source=repository.go -destination=mocks/mocks.go -package=mocks
type RepositoryName string

const (
	UserRepoName  RepositoryName = "user"
	OrderRepoName RepositoryName = "order"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user User) (*User, error)
	FindUserByUsername(ctx context.Context, username string) (*User, error)
}

type OrderRepository interface {
	CreateOrder(ctx context.Context, userID int64, orderCode string) (*Order, error)
	FindByOrderCode(ctx context.Context, orderCode string) (*Order, error)
	GetByUserID(ctx context.Context, userID int64) ([]Order, error)
}
