package api

//go:generate mockgen -source=interfaces.go -destination=mocks/mocks.go -package=mocks

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/service"
)

// UserServicer интерфейс исключительно для моков.
type UserServicer interface {
	Register(ctx context.Context, args service.RegisterUserArgs) (*domain.User, string, error)
	Login(ctx context.Context, args service.LoginUserArgs) (*domain.User, string, error)
}

type OrderServicer interface {
	Create(ctx context.Context, userID int64, orderCode string) (*domain.Order, error)
	GetByUserID(ctx context.Context, userID int64) ([]domain.Order, error)
}

type BalanceServicer interface {
	GetUserBalance(ctx context.Context, userID int64) (*service.UserBalance, error)
	Withdraw(
		ctx context.Context,
		userID int64,
		orderCode string,
		amount decimal.Decimal,
	) (*domain.BalanceTransaction, error)
	GetByDirection(
		ctx context.Context,
		userID int64,
		direction domain.DirectionType,
	) ([]domain.BalanceTransaction, error)
}
