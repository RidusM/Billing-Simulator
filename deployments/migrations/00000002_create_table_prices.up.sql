CREATE TYPE price_interval AS ENUM ('day', 'week', 'month', 'year');

CREATE TABLE prices (
    id UUID PRIMARY KEY,
    public_id VARCHAR(64) UNIQUE,
    product_id VARCHAR(64) NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    amount BIGINT NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    interval price_interval NOT NULL,
    interval_count INT NOT NULL DEFAULT 1,
    active BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB NOT NULL DEFAULT '{}',
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_prices_amount_positive CHECK (amount >= 0),
    CONSTRAINT chk_prices_interval_count_positive CHECK (interval_count > 0)
);

CREATE INDEX idx_prices_product_id ON prices(product_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_prices_active ON prices(active) WHERE active = true AND deleted_at IS NULL;