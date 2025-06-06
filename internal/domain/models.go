package domain

import (
	"github.com/shopspring/decimal"

	"time"
)

type User struct {
	ID        int64
	CreatedAt time.Time
	UpdatedAt time.Time
	Username  string
	Password  string
}

type Order struct {
	ID        int64
	CreatedAt time.Time
	UpdatedAt time.Time
	UserID    int64
	OrderCode string
	Status    OrderStatusType
	Accrual   decimal.Decimal
}

type BalanceTransaction struct {
	ID        int64
	CreatedAt time.Time
	UpdatedAt time.Time
	UserID    int64
	OrderID   int64
	Amount    decimal.Decimal
}
