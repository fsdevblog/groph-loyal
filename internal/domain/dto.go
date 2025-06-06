package domain

import "github.com/shopspring/decimal"

type OrderStatusType string

const (
	OrderStatusNew        OrderStatusType = "NEW"
	OrderStatusProcessing OrderStatusType = "PROCESSING"
	OrderStatusProcessed  OrderStatusType = "PROCESSED"
	OrderStatusInvalid    OrderStatusType = "INVALID"
)

type OrderAccrualUpdateDTO struct {
	ID      int64
	Status  OrderStatusType
	Accrual decimal.Decimal
}

type OrderBatchQueryRowDTO func(i int, t *Order, err error)

type DirectionType string

const (
	DirectionDebit  DirectionType = "debit"
	DirectionCredit DirectionType = "credit"
)

type BalanceTransactionCreateDTO struct {
	UserID    int64
	OrderID   int64
	Direction DirectionType
	Amount    decimal.Decimal
}
type BalanceTransBatchQueryRowDTO func(i int, err error)

type UserBalanceSumDTO struct {
	DebitAmount  decimal.Decimal `json:"current"`
	CreditAmount decimal.Decimal `json:"withdrawn"`
}
