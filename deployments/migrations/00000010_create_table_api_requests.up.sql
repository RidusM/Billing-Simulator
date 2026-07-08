CREATE TABLE api_requests (
    id UUID PRIMARY KEY,
    
    trace_id UUID NOT NULL,
    
    method VARCHAR(10) NOT NULL,
    path VARCHAR(512) NOT NULL,
    query_params JSONB,
    request_body JSONB,
    headers JSONB,
    
    -- Ответ
    response_status INT NOT NULL,
    response_body JSONB,
    
    ip_address VARCHAR(45),
    user_agent VARCHAR(512),
    duration_ms INT,
    customer_id UUID,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_requests_trace_id ON api_requests(trace_id);
CREATE INDEX idx_api_requests_created_at ON api_requests(created_at DESC);
CREATE INDEX idx_api_requests_customer_id ON api_requests(customer_id) WHERE customer_id IS NOT NULL;
CREATE INDEX idx_api_requests_path ON api_requests(path);