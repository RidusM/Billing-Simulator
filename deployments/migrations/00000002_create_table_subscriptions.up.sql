CREATE TYPE subscription_status AS ENUM ('active', 'past_due', 'canceled', 'unpaid');

CREATE TABLE subscriptions (
    id UUID PRIMARY KEY,
    public_id VARCHAR(64) UNIQUE,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    status subscription_status NOT NULL DEFAULT 'active',
    price_id VARCHAR(64) NOT NULL,
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end TIMESTAMPTZ NOT NULL,
    next_billing_at TIMESTAMPTZ NOT NULL,
    canceled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_subscriptions_next_billing ON subscriptions(next_billing_at) WHERE status IN ('active', 'past_due');