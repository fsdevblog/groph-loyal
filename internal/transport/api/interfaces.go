package api

//go:generate mockgen -source=interfaces.go -destination=mocks/mocks.go -package=mocks

import (
	"context"

	"github.com/fsdevblog/groph-loyal/internal/service/svsbt"
	"github.com/fsdevblog/groph-loyal/internal/service/svsuser"

	"github.com/shopspring/decimal"

	"github.com/fsdevblog/groph-loyal/internal/domain"
)

// UserServicer интерфейс исключительно для моков.
type UserServicer interface {
	Register(ctx context.Context, args svsuser.RegisterUserArgs) (*domain.User, string, error)
	Login(ctx context.Context, args svsuser.LoginUserArgs) (*domain.User, string, error)
}

type OrderServicer interface {
	Create(ctx context.Context, userID int64, orderCode string) (*domain.Order, error)
	GetByUserID(ctx context.Context, userID int64) ([]domain.Order, error)
}

type BalanceServicer interface {
	GetUserBalance(ctx context.Context, userID int64) (*svsbt.UserBalance, error)
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
