package entity

import "errors"

var (
	ErrDataNotFound    = errors.New("data not found")
	ErrConflictingData = errors.New("conflicting data")
	ErrInvalidData     = errors.New("invalid data")
)
