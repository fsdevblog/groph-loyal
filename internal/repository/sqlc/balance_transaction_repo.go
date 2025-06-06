package sqlc

import (
	"context"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/repository/sqlc/sqlcgen"
)

type balanceTransactionRepository struct {
	q *sqlcgen.Queries
}

func NewBalanceTransactionRepository(conn sqlcgen.DBTX) domain.BalanceTransactionRepository {
	return &balanceTransactionRepository{q: sqlcgen.New(conn)}
}

func (b *balanceTransactionRepository) Create(
	ctx context.Context,
	transaction domain.BalanceTransactionCreateDTO,
) (*domain.BalanceTransaction, error) {

	dbTrans, err := b.q.BalanceTransaction_Create(ctx, sqlcgen.BalanceTransaction_CreateParams{
		UserID:    transaction.UserID,
		OrderID:   transaction.OrderID,
		Amount:    transaction.Amount,
		Direction: sqlcgen.BalanceTransactionType(transaction.Direction),
	})

	if err != nil {
		return nil, convertErr(err, "creating balance transaction")
	}
	return convertBalanceTransactionModel(dbTrans), nil
}

func (b *balanceTransactionRepository) BatchCreate(
	ctx context.Context,
	transactions []domain.BalanceTransactionCreateDTO,
	fn domain.BalanceTransBatchQueryRowDTO,
) {
	var params = make([]sqlcgen.BalanceTransaction_CreateBatchParams, len(transactions))
	for i, transaction := range transactions {

		params[i] = sqlcgen.BalanceTransaction_CreateBatchParams{
			UserID:  transaction.UserID,
			OrderID: transaction.OrderID,
			Amount:  transaction.Amount,
		}
	}
	r := b.q.BalanceTransaction_CreateBatch(ctx, params)
	r.Exec(func(i int, err error) {
		fn(i, convertErr(err, "creating balance transaction"))
	})
}

func (b *balanceTransactionRepository) GetUserBalance(
	ctx context.Context,
	userID int64,
) (*domain.UserBalanceSumDTO, error) {
	stats, err := b.q.BalanceTransaction_SumByUserID(ctx, userID)
	if err != nil {
		return nil, convertErr(err, "getting balance sum by userID %d", userID)
	}
	var sum = new(domain.UserBalanceSumDTO)
	for _, row := range stats {
		if row.Direction == sqlcgen.BalanceTransactionTypeCredit {
			sum.CreditAmount = row.Sum
		} else {
			sum.DebitAmount = row.Sum
		}
	}
	return sum, nil
}

func convertBalanceTransactionModel(model sqlcgen.BalanceTransaction) *domain.BalanceTransaction {
	return &domain.BalanceTransaction{
		ID:        model.ID,
		CreatedAt: model.CreatedAt.Time,
		UpdatedAt: model.UpdatedAt.Time,
		UserID:    model.UserID,
		OrderID:   model.OrderID,
		Amount:    model.Amount,
	}
}
