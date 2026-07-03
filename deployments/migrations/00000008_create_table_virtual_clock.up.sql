CREATE TABLE virtual_clock (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    current_time TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);