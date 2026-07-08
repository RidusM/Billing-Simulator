CREATE TABLE virtual_clock (
    id INT PRIMARY KEY DEFAULT 1,
    virtual_time TIMESTAMPTZ NOT NULL,
    offset_seconds BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ,
    updated_by VARCHAR(128),
    CONSTRAINT single_row_clock CHECK (id = 1)
);
