package transaction

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
	ErrTransactionTimeout = errors.New("transaction timeout")
	ErrConflictingData    = errors.New("data conflicts with existing data in unique column")
	ErrInvalidData        = errors.New("invalid data")
)

func HandleError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("timeout: %w", ErrTransactionTimeout)
	}

	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("canceled: %w", err)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "40P01":
			return fmt.Errorf("deadlock: %w", err)
		case "40001":
			return fmt.Errorf("serialization failure: %w", err)
		case "57014":
			return fmt.Errorf("statement timeout: %w", err)
		case "55P03":
			return fmt.Errorf("lock timeout: %w", err)
		case "23505":
			return fmt.Errorf("unique constraint violation: %w", ErrConflictingData)
		case "23503":
			return fmt.Errorf("foreign key violation: %w", ErrInvalidData)
		}
	}

	return err
}
