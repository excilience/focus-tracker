CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    duration_seconds INTEGER NOT NULL CHECK (duration_seconds >= 0)
);