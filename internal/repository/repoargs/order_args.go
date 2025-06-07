package repoargs

import (
	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/shopspring/decimal"
)

type BatchUpdateWithAccrualData struct {
	ID      int64
	Status  domain.OrderStatusType
	Accrual decimal.Decimal
}

type OrderBatchQueryRow func(i int, t *domain.Order, err error)
