CREATE TYPE subscription_status AS ENUM (
    'active',
    'past_due',
    'canceled',
    'unpaid',
    'trialing',
    'incomplete'
);

CREATE TABLE subscriptions (
    id UUID PRIMARY KEY,
    public_id VARCHAR(64) UNIQUE NOT NULL,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    price_id UUID NOT NULL REFERENCES prices(id) ON DELETE RESTRICT,
    status subscription_status NOT NULL DEFAULT 'active',
    
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end TIMESTAMPTZ NOT NULL,
    next_billing_at TIMESTAMPTZ NOT NULL,
    
    trial_start TIMESTAMPTZ,
    trial_end TIMESTAMPTZ,

    canceled_at TIMESTAMPTZ,
    cancel_at_period_end BOOLEAN NOT NULL DEFAULT false,
    cancellation_details JSONB,
    
    metadata JSONB NOT NULL DEFAULT '{}',
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    
    CONSTRAINT chk_subscriptions_period CHECK (current_period_end >= current_period_start)
);

CREATE INDEX idx_subscriptions_next_billing 
    ON subscriptions(next_billing_at) 
    WHERE status IN ('active', 'past_due', 'trialing') AND deleted_at IS NULL;

CREATE INDEX idx_subscriptions_customer_id 
    ON subscriptions(customer_id) 
    WHERE deleted_at IS NULL;

CREATE INDEX idx_subscriptions_status 
    ON subscriptions(status) 
    WHERE deleted_at IS NULL;