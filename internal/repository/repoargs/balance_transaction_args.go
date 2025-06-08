package repoargs

import (
	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/shopspring/decimal"
)

type BalanceTransactionCreate struct {
	UserID    int64
	OrderID   int64
	OrderCode string
	Direction domain.DirectionType
	Amount    decimal.Decimal
}
type BatchExecQueryRow func(i int, err error)
