package client

import (
	"fmt"
	"time"
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

type TooManyRequestError struct {
	RetryAfter time.Duration
}

func NewTooManyRequestError(retryAfter time.Duration) *TooManyRequestError {
	return &TooManyRequestError{RetryAfter: retryAfter}
}

func (e *TooManyRequestError) Error() string {
	return fmt.Sprintf("Too many requests. Need retry after %.f seconds", e.RetryAfter.Seconds())
}
