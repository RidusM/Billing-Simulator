CREATE TABLE outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    event_type VARCHAR(64) NOT NULL,
    aggregate_id UUID NOT NULL,
    payload JSONB NOT NULL,
    
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    processed BOOLEAN NOT NULL DEFAULT false,
    processed_at TIMESTAMPTZ,
    error TEXT,
    
    CONSTRAINT chk_outbox_events_payload CHECK (jsonb_typeof(payload) = 'object')
);

CREATE INDEX idx_outbox_events_unprocessed 
    ON outbox_events(created_at ASC) 
    WHERE processed = false;

CREATE INDEX idx_outbox_events_processed_cleanup 
    ON outbox_events(processed_at ASC) 
    WHERE processed = true;

CREATE INDEX idx_outbox_events_event_type 
    ON outbox_events(event_type);

CREATE INDEX idx_outbox_events_aggregate_id 
    ON outbox_events(aggregate_id);