CREATE TABLE webhook_endpoints (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    url VARCHAR(512) NOT NULL,
    secret VARCHAR(128) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_webhook_endpoints_customer_id ON webhook_endpoints(customer_id);
CREATE INDEX idx_webhook_endpoints_enabled ON webhook_endpoints(enabled) WHERE enabled = true;