package svsbt

//go:generate mockgen -source=interfaces.go -destination=mocks/mocks.go -package=mocks

import (
	"context"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/repository/repoargs"
)

type BalanceTransactionRepository interface {
	GetUserBalance(ctx context.Context, userID int64) (*repoargs.BalanceAggregation, error)
	Create(ctx context.Context, transaction repoargs.BalanceTransactionCreate) (*domain.BalanceTransaction, error)
	GetByDirection(
		ctx context.Context,
		userID int64,
		direction domain.DirectionType,
	) ([]domain.BalanceTransaction, error)
}

type OrderRepository interface {
	CreateOrder(ctx context.Context, userID int64, orderCode string) (*domain.Order, error)
}
