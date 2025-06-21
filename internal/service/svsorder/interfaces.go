package svsorder

//go:generate mockgen -source=interfaces.go -destination=mocks/mocks.go -package=mocks

import (
	"context"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/repository/repoargs"
)

type OrderRepository interface {
	CreateOrder(ctx context.Context, userID int64, orderCode string) (*domain.Order, error)
	FindByOrderCode(ctx context.Context, orderCode string) (*domain.Order, error)
	GetByUserID(ctx context.Context, userID int64) ([]domain.Order, error)
	GetForMonitoring(ctx context.Context, limit uint) ([]domain.Order, error)
	BatchUpdateWithAccrualData(
		ctx context.Context,
		updates []repoargs.BatchUpdateWithAccrualData,
		fn repoargs.OrderBatchQueryRow,
	)
	IncrementErrAttempts(
		ctx context.Context,
		data []repoargs.OrderBatchIncrementAttempts,
		fn repoargs.BatchExecQueryRow,
	)
}

type BalanceTransactionRepository interface {
	BatchCreate(
		ctx context.Context,
		transactions []repoargs.BalanceTransactionCreate,
		fn repoargs.BatchExecQueryRow,
	)
}
