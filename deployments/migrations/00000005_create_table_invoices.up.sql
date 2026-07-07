CREATE TYPE invoice_status AS ENUM (
    'draft',
    'open',
    'paid',
    'uncollectible',
    'void'
);

CREATE TABLE invoices (
    id UUID PRIMARY KEY,
    public_id VARCHAR(64) UNIQUE NOT NULL,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    subscription_id UUID REFERENCES subscriptions(id) ON DELETE SET NULL,
    status invoice_status NOT NULL DEFAULT 'draft',
    
    amount BIGINT NOT NULL,
    amount_paid BIGINT NOT NULL DEFAULT 0,
    amount_remaining BIGINT NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'USD',

    period_start TIMESTAMPTZ,
    period_end TIMESTAMPTZ,
    due_date TIMESTAMPTZ,
    
    attempt_count INT NOT NULL DEFAULT 0,
    attempted_at TIMESTAMPTZ,
    
    hosted_invoice_url VARCHAR(512),
    invoice_pdf_url VARCHAR(512),
    
    metadata JSONB NOT NULL DEFAULT '{}',
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_invoices_amount_positive CHECK (amount >= 0),
    CONSTRAINT chk_invoices_amount_paid CHECK (amount_paid >= 0),
    CONSTRAINT chk_invoices_attempt_count CHECK (attempt_count >= 0)
);

CREATE INDEX idx_invoices_customer_id ON invoices(customer_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_invoices_subscription_id ON invoices(subscription_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_invoices_status ON invoices(status) WHERE status IN ('open', 'draft') AND deleted_at IS NULL;
CREATE INDEX idx_invoices_due_date ON invoices(due_date) WHERE status = 'open' AND deleted_at IS NULL;