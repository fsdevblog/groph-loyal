package sqlc

import (
	"context"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/repository/repoargs"
	"github.com/fsdevblog/groph-loyal/internal/repository/sqlc/sqlcgen"
)

type BalanceTransRepository struct {
	q *sqlcgen.Queries
}

func NewBalanceTransactionRepository(conn sqlcgen.DBTX) *BalanceTransRepository {
	return &BalanceTransRepository{q: sqlcgen.New(conn)}
}

func (b *BalanceTransRepository) Create(
	ctx context.Context,
	transaction repoargs.BalanceTransactionCreate,
) (*domain.BalanceTransaction, error) {
	dbTrans, err := b.q.BalanceTransaction_Create(ctx, sqlcgen.BalanceTransaction_CreateParams{
		UserID:    transaction.UserID,
		OrderID:   transaction.OrderID,
		Amount:    transaction.Amount,
		OrderCode: transaction.OrderCode,
		Direction: sqlcgen.BalanceTransactionType(transaction.Direction),
	})

	if err != nil {
		return nil, convertErr(err, "creating balance transaction")
	}
	return convertBalanceTransactionModel(dbTrans), nil
}

func (b *BalanceTransRepository) BatchCreate(
	ctx context.Context,
	transactions []repoargs.BalanceTransactionCreate,
	fn repoargs.BalanceTransBatchQueryRow,
) {
	var params = make([]sqlcgen.BalanceTransaction_CreateBatchParams, len(transactions))
	for i, transaction := range transactions {
		params[i] = sqlcgen.BalanceTransaction_CreateBatchParams{
			UserID:    transaction.UserID,
			OrderID:   transaction.OrderID,
			OrderCode: transaction.OrderCode,
			Amount:    transaction.Amount,
			Direction: sqlcgen.BalanceTransactionType(transaction.Direction),
		}
	}
	r := b.q.BalanceTransaction_CreateBatch(ctx, params)
	r.Exec(func(i int, err error) {
		fn(i, convertErr(err, "creating balance transaction"))
	})
}

func (b *BalanceTransRepository) GetUserBalance(
	ctx context.Context,
	userID int64,
) (*repoargs.BalanceSum, error) {
	stats, err := b.q.BalanceTransaction_SumByUserID(ctx, userID)
	if err != nil {
		return nil, convertErr(err, "getting balance sum by userID %d", userID)
	}
	var sum = new(repoargs.BalanceSum)
	for _, row := range stats {
		if row.Direction == sqlcgen.BalanceTransactionTypeCredit {
			sum.CreditAmount = row.Sum
		} else {
			sum.DebitAmount = row.Sum
		}
	}
	return sum, nil
}

func (b *BalanceTransRepository) GetByDirection(
	ctx context.Context,
	userID int64,
	direction domain.DirectionType,
) ([]domain.BalanceTransaction, error) {
	dbTransactions, err := b.q.BalanceTransaction_GetByDirection(ctx, sqlcgen.BalanceTransaction_GetByDirectionParams{
		UserID:    userID,
		Direction: sqlcgen.BalanceTransactionType(direction),
	})
	if err != nil {
		return nil, convertErr(err, "balance transactions")
	}
	var transactions = make([]domain.BalanceTransaction, len(dbTransactions))
	for i, transaction := range dbTransactions {
		transactions[i] = *convertBalanceTransactionModel(transaction)
	}
	return transactions, nil
}

func convertBalanceTransactionModel(model sqlcgen.BalanceTransaction) *domain.BalanceTransaction {
	return &domain.BalanceTransaction{
		ID:        model.ID,
		CreatedAt: model.CreatedAt.Time,
		UpdatedAt: model.UpdatedAt.Time,
		UserID:    model.UserID,
		OrderID:   model.OrderID,
		OrderCode: model.OrderCode,
		Amount:    model.Amount,
	}
}
