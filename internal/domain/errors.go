package domain

import "errors"

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrDuplicateKey   = errors.New("duplicate key")
	ErrUnknown        = errors.New("unknown error")
)
