package domain

type OrderStatusType string

const (
	OrderStatusNew        OrderStatusType = "NEW"
	OrderStatusProcessing OrderStatusType = "PROCESSING"
	OrderStatusProcessed  OrderStatusType = "PROCESSED"
	OrderStatusInvalid    OrderStatusType = "INVALID"
)

type DirectionType string

const (
	DirectionDebit  DirectionType = "debit"
	DirectionCredit DirectionType = "credit"
)
