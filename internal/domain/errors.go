package domain

import "errors"

var (
	ErrRecordNotFound    = errors.New("record not found")
	ErrPasswordMissMatch = errors.New("password mismatch")
	ErrDuplicateKey      = errors.New("duplicate key")
	ErrUnknown           = errors.New("unknown error")
)
