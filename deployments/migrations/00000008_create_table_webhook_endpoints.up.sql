CREATE TABLE webhook_endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    public_id VARCHAR(64) UNIQUE NOT NULL,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    
    url VARCHAR(512) NOT NULL,
    description VARCHAR(255),
    
    secret_prefix VARCHAR(16) NOT NULL,
    secret_encrypted TEXT NOT NULL,
    
    enabled_events TEXT[] NOT NULL DEFAULT ARRAY['*'],
    
    enabled BOOLEAN NOT NULL DEFAULT true,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_endpoints_customer_id ON webhook_endpoints(customer_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_webhook_endpoints_enabled ON webhook_endpoints(enabled) WHERE enabled = true AND deleted_at IS NULL;
CREATE INDEX idx_webhook_endpoints_enabled_events ON webhook_endpoints USING GIN(enabled_events);
