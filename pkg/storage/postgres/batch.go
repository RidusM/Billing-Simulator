package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

var ErrInvalidTableName = errors.New("invalid table name type")

func BatchInsert(ctx context.Context, qe QueryExecuter, sql string, rows [][]any) error {
	const op = "storage.postgres.BatchInsert"

	batch := &pgx.Batch{}
	for _, row := range rows {
		batch.Queue(sql, row...)
	}

	results := qe.SendBatch(ctx, batch)
	defer results.Close()

	for i := range rows {
		_, err := results.Exec()
		if err != nil {
			return fmt.Errorf("%s: executing statement at index %d: %w", op, i, err)
		}
	}

	return nil
}

func BulkInsert(ctx context.Context, qe QueryExecuter, tableName any, columns []string, data [][]any) (int64, error) {
	const op = "storage.postgres.BulkInsert"

	var ident pgx.Identifier
	switch t := tableName.(type) {
	case string:
		ident = pgx.Identifier{t}
	case []string:
		ident = pgx.Identifier(t)
	case pgx.Identifier:
		ident = t
	default:
		return 0, fmt.Errorf("%w", ErrInvalidTableName)
	}

	count, err := qe.CopyFrom(ctx, ident, columns, pgx.CopyFromRows(data))
	if err != nil {
		return 0, fmt.Errorf("%s: copy from: %w", op, err)
	}

	return count, nil
}
