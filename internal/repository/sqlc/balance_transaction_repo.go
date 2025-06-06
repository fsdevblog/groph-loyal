package sqlc

import (
	"context"

	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/fsdevblog/groph-loyal/internal/repository/sqlc/sqlcgen"
	"github.com/jackc/pgx/v5/pgtype"
)

type balanceTransactionRepository struct {
	q *sqlcgen.Queries
}

func NewBalanceTransactionRepository(conn sqlcgen.DBTX) domain.BalanceTransactionRepository {
	return &balanceTransactionRepository{q: sqlcgen.New(conn)}
}

func (b *balanceTransactionRepository) BatchCreate(
	ctx context.Context,
	transactions []domain.BalanceTransactionCreateDTO,
	fn domain.BalanceTransBatchQueryRowDTO,
) {
	var params = make([]sqlcgen.BalanceTransaction_CreateParams, len(transactions))
	for i, transaction := range transactions {
		var orderID int64
		if transaction.OrderID != nil {
			orderID = *transaction.OrderID
		}
		params[i] = sqlcgen.BalanceTransaction_CreateParams{
			UserID: transaction.UserID,
			OrderID: pgtype.Int8{
				Int64: orderID,
				Valid: transaction.OrderID != nil,
			},
			Amount: transaction.Amount,
		}
	}
	r := b.q.BalanceTransaction_Create(ctx, params)
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
