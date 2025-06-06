package accrual

import (
	"errors"
	"fmt"
)

var (
	ErrNoOrders = errors.New("no orders")
)

type StatusCodeError struct {
	Code int
}

func NewStatusCodeError(code int) *StatusCodeError {
	return &StatusCodeError{Code: code}
}

func (e *StatusCodeError) Error() string {
	return fmt.Sprintf("Unexpected status code %d", e.Code)
}
