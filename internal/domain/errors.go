package domain

import (
	"errors"
	"fmt"
)

var (
	ErrRecordNotFound    = errors.New("record not found")
	ErrPasswordMissMatch = errors.New("password mismatch")
	ErrDuplicateKey      = errors.New("duplicate key")
	ErrUnknown           = errors.New("unknown error")

	ErrNotEnoughBalance = errors.New("not enough balance")
	ErrOwnerConflict    = errors.New("owner conflict")
)

type DuplicateOrderError struct {
	Order *Order
}

func NewDuplicateOrderError(order *Order) error {
	return &DuplicateOrderError{Order: order}
}

func (e *DuplicateOrderError) Error() string {
	return fmt.Sprintf(
		"order with code %s already exists for user with id %d",
		e.Order.OrderCode,
		e.Order.UserID,
	)
}
