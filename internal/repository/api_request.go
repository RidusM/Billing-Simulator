package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bill-stripe-sim/internal/entity"
	"bill-stripe-sim/pkg/storage/postgres"
	"bill-stripe-sim/pkg/storage/postgres/transaction"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type APIRequestRepository struct {
	storage *postgres.Postgres
}

func NewAPIRequestRepository(storage *postgres.Postgres) *APIRequestRepository {
	return &APIRequestRepository{storage: storage}
}

func (r *APIRequestRepository) Create(ctx context.Context, req *entity.APIRequest) error {
	const op = "repository.api_request.Create"
	sql, args, err := r.storage.Builder.
		Insert("api_requests").
		Columns("id", "trace_id", "method", "path", "query_params", "request_body", "headers", "response_status", "response_body", "ip_address", "user_agent", "duration_ms", "customer_id", "created_at").
		Values(req.ID, req.TraceID, req.Method, req.Path, req.QueryParams, req.RequestBody, req.Headers, req.ResponseStatus, req.ResponseBody, req.IPAddress, req.UserAgent, req.DurationMs, req.CustomerID, req.CreatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_, err = r.executor(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (r *APIRequestRepository) GetByTraceID(ctx context.Context, traceID uuid.UUID) (*entity.APIRequest, error) {
	const op = "repository.api_request.GetByTraceID"
	sql, args, err := r.storage.
		Select("id", "trace_id", "method", "path", "query_params", "request_body", "headers", "response_status", "response_body", "ip_address", "user_agent", "duration_ms", "customer_id", "created_at").
		From("api_requests").
		Where(squirrel.Eq{"trace_id": traceID}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var req entity.APIRequest
	err = r.executor(ctx).QueryRow(ctx, sql, args...).Scan(
		&req.ID, &req.TraceID, &req.Method, &req.Path, &req.QueryParams, &req.RequestBody, &req.Headers,
		&req.ResponseStatus, &req.ResponseBody, &req.IPAddress, &req.UserAgent, &req.DurationMs, &req.CustomerID, &req.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, entity.ErrDataNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &req, nil
}

func (r *APIRequestRepository) GetRecent(ctx context.Context, limit, offset int) ([]*entity.APIRequest, error) {
	const op = "repository.api_request.GetRecent"
	sql, args, err := r.storage.
		Select("id", "trace_id", "method", "path", "query_params", "request_body", "headers", "response_status", "response_body", "ip_address", "user_agent", "duration_ms", "customer_id", "created_at").
		From("api_requests").
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := r.executor(ctx).Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var requests []*entity.APIRequest
	for rows.Next() {
		var req entity.APIRequest
		err = rows.Scan(
			&req.ID, &req.TraceID, &req.Method, &req.Path, &req.QueryParams, &req.RequestBody, &req.Headers,
			&req.ResponseStatus, &req.ResponseBody, &req.IPAddress, &req.UserAgent, &req.DurationMs, &req.CustomerID, &req.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		requests = append(requests, &req)
	}
	return requests, nil
}

func (r *APIRequestRepository) DeleteOldRequests(ctx context.Context, olderThan time.Duration) (int64, error) {
	const op = "repository.api_request.DeleteOldRequests"
	sql, args, err := r.storage.Builder.
		Delete("api_requests").
		Where(squirrel.Expr("created_at <= NOW() - ? * INTERVAL '1 SECOND'", olderThan.Seconds())).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	tag, err := r.executor(ctx).Exec(ctx, sql, args...)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	return tag.RowsAffected(), nil
}

func (r *APIRequestRepository) executor(ctx context.Context) postgres.QueryExecuter {
	if qe, ok := transaction.TxFromCtx(ctx); ok {
		return qe
	}
	return r.storage
}
