CREATE TYPE payment_intent_status AS ENUM (
    'requires_payment_method',
    'requires_confirmation',
    'requires_action',
    'processing',
    'succeeded',
    'canceled',
    'requires_capture'
);

CREATE TABLE payment_intents (
    id UUID PRIMARY KEY,
    public_id VARCHAR(64) UNIQUE NOT NULL,         -- pi_xxx
    invoice_id UUID REFERENCES invoices(id) ON DELETE SET NULL,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    
    amount BIGINT NOT NULL,
    amount_captured BIGINT NOT NULL DEFAULT 0,
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    status payment_intent_status NOT NULL DEFAULT 'requires_payment_method',
    
    last_payment_error JSONB,
    
    payment_method_id VARCHAR(64),
    payment_method_type VARCHAR(32) DEFAULT 'card',
    
    metadata JSONB NOT NULL DEFAULT '{}',
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    
    CONSTRAINT chk_pi_amount_positive CHECK (amount > 0)
);

CREATE INDEX idx_payment_intents_invoice_id ON payment_intents(invoice_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_payment_intents_customer_id ON payment_intents(customer_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_payment_intents_status ON payment_intents(status) WHERE status NOT IN ('succeeded', 'canceled') AND deleted_at IS NULL;
