CREATE TABLE products (
    id UUID PRIMARY KEY,
    public_id VARCHAR(64) UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    active BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB NOT NULL DEFAULT '{}',
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX idx_products_active ON products(active) WHERE active = true AND deleted_at IS NULL;