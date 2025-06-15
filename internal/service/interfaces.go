package service

import (
	"context"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/repository/repoargs"
)

//go:generate mockgen -source=interfaces.go -destination=mocks/mocks.go -package=mocks

type PasswordHasher interface {
	HashPassword(password string) (string, error)
	ComparePassword(password string, hashedPassword string) bool
}

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

type UserRepository interface {
	CreateUser(ctx context.Context, user repoargs.CreateUser) (*domain.User, error)
	FindUserByUsername(ctx context.Context, username string) (*domain.User, error)
}

type BalanceTransactionRepository interface {
	BatchCreate(
		ctx context.Context,
		transactions []repoargs.BalanceTransactionCreate,
		fn repoargs.BatchExecQueryRow,
	)
	GetUserBalance(ctx context.Context, userID int64) (*repoargs.BalanceAggregation, error)
	Create(ctx context.Context, transaction repoargs.BalanceTransactionCreate) (*domain.BalanceTransaction, error)
	GetByDirection(
		ctx context.Context,
		userID int64,
		direction domain.DirectionType,
	) ([]domain.BalanceTransaction, error)
}
