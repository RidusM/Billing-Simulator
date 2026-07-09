CREATE TYPE webhook_delivery_status AS ENUM ('pending', 'delivered', 'failed');

CREATE TABLE webhook_logs (
    id UUID PRIMARY KEY,
    public_id VARCHAR(64) UNIQUE NOT NULL,
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    endpoint_id UUID NOT NULL REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
    
    trace_id UUID NOT NULL,
    
    event_type VARCHAR(64) NOT NULL,
    payload JSONB NOT NULL,
    target_url VARCHAR(512) NOT NULL,
    
    status webhook_delivery_status NOT NULL DEFAULT 'pending',
    response_code INT,
    response_body TEXT,
    
    attempt INT NOT NULL DEFAULT 1,
    max_attempts INT NOT NULL DEFAULT 5,
    error_message TEXT,
    next_attempt_at TIMESTAMPTZ NOT NULL,
    delivered_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    
    CONSTRAINT chk_webhook_logs_attempt CHECK (attempt >= 1)
);

CREATE INDEX idx_webhook_logs_retry_queue 
    ON webhook_logs(next_attempt_at ASC) 
    WHERE status = 'pending' AND next_attempt_at IS NOT NULL;

CREATE INDEX idx_webhook_logs_trace_id ON webhook_logs(trace_id);
CREATE INDEX idx_webhook_logs_endpoint_id ON webhook_logs(endpoint_id);
CREATE INDEX idx_webhook_logs_event_id ON webhook_logs(event_id);
CREATE INDEX idx_webhook_logs_status ON webhook_logs(status);
CREATE INDEX idx_webhook_logs_dashboard
ON webhook_logs(created_at DESC)
WHERE deleted_at IS NULL;
