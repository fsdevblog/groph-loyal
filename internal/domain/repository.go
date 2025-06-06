package domain

import "context"

//go:generate mockgen -source=repository.go -destination=mocks/mocks.go -package=mocks
type RepositoryName string

const (
	UserRepoName               RepositoryName = "user"
	OrderRepoName              RepositoryName = "order"
	BalanceTransactionRepoName RepositoryName = "balance_transaction"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user User) (*User, error)
	FindUserByUsername(ctx context.Context, username string) (*User, error)
}

type OrderRepository interface {
	CreateOrder(ctx context.Context, userID int64, orderCode string) (*Order, error)
	FindByOrderCode(ctx context.Context, orderCode string) (*Order, error)
	GetByUserID(ctx context.Context, userID int64) ([]Order, error)
	GetByStatuses(ctx context.Context, limit uint, statuses []OrderStatusType) ([]Order, error)
	BatchUpdateWithAccrualData(ctx context.Context, updates []OrderAccrualUpdateDTO, fn OrderBatchQueryRowDTO)
}

type BalanceTransactionRepository interface {
	BatchCreate(ctx context.Context, transactions []BalanceTransactionCreateDTO, fn BalanceTransBatchQueryRowDTO)
	GetUserBalance(ctx context.Context, userID int64) (*UserBalanceSumDTO, error)
	Create(ctx context.Context, transaction BalanceTransactionCreateDTO) (*BalanceTransaction, error)
}
