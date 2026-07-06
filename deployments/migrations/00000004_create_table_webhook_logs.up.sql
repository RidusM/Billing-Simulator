CREATE TYPE webhook_status AS ENUM ('pending', 'delivered', 'failed');

CREATE TABLE webhook_logs (
    id UUID PRIMARY KEY,
    trace_id UUID NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    payload JSONB NOT NULL,
    target_url VARCHAR(512) NOT NULL,
    status webhook_status NOT NULL DEFAULT 'pending',
    response_code INT,
    attempt INT NOT NULL DEFAULT 1,
    error_message TEXT,
    next_attempt_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_webhook_logs_status ON webhook_logs(status);
CREATE INDEX idx_webhook_logs_retry_manager 
ON webhook_logs(next_attempt_at) 
WHERE status = 'pending';

CREATE INDEX idx_webhook_logs_trace_id ON webhook_logs(trace_id);