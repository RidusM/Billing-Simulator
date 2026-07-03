package kafka

import "errors"

type UnretryableError struct {
	Err error
}

func (e UnretryableError) Error() string { return e.Err.Error() }
func (e UnretryableError) Unwrap() error { return e.Err }

func IsUnretryable(err error) bool {
	var target UnretryableError
	return errors.As(err, &target)
}
