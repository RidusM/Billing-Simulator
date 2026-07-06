CREATE TABLE virtual_clock (
    id INT PRIMARY KEY DEFAULT 1,
    current_time TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT single_row_clock CHECK (id = 1)
);