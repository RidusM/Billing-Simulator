CREATE TABLE customers (
    id UUID PRIMARY KEY,
    public_id VARCHAR(64) UNIQUE,
    email VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);