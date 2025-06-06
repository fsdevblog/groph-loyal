package dto

import (
	"github.com/fsdevblog/groph-loyal/internal/domain"
	"github.com/shopspring/decimal"
)

type OrderAccrualResponse struct {
	OrderCode string                 `json:"order"`
	Status    domain.OrderStatusType `json:"status"`
	Accrual   decimal.Decimal        `json:"accrual,omitempty"`
}
