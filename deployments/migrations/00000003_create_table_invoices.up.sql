CREATE TYPE invoice_status AS ENUM ('draft', 'open', 'paid', 'uncollectible', 'void');

CREATE TABLE invoices (
    id UUID PRIMARY KEY,
    public_id VARCHAR(64) UNIQUE,
    subscription_id UUID REFERENCES subscriptions(id) ON DELETE SET NULL,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    amount BIGINT NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    status invoice_status NOT NULL DEFAULT 'open',
    attempt_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_invoices_customer_id ON invoices(customer_id);
CREATE INDEX idx_invoices_subscription_id ON invoices(subscription_id);