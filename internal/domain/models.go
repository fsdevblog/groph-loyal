package domain

import "time"

type User struct {
	ID        int64
	CreatedAt time.Time
	UpdatedAt time.Time
	Username  string
	Password  string
}

type OrderStatus string

const (
	OrderStatusNew        OrderStatus = "NEW"
	OrderStatusRegistered OrderStatus = "REGISTERED"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusProcessed  OrderStatus = "PROCESSED"
	OrderStatusInvalid    OrderStatus = "INVALID"
)

type Order struct {
	ID        int64
	CreatedAt time.Time
	UpdatedAt time.Time
	UserID    int64
	OrderCode string
	Status    OrderStatus
	Accrual   uint
}
