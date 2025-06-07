package repoargs

import (
	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/shopspring/decimal"
)

type BalanceTransactionCreate struct {
	UserID    int64
	OrderID   int64
	Direction domain.DirectionType
	Amount    decimal.Decimal
}
type BalanceTransBatchQueryRow func(i int, err error)
