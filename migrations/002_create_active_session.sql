CREATE TABLE IF NOT EXISTS active_sessions (
    id SMALLINT PRIMARY KEY DEFAULT 1
    CHECK (id = 1),

    start_time TIMESTAMPTZ NOT NULL,
    last_resume TIMESTAMPTZ NOT NULL,
    
    focused_seconds INTEGER NOT NULL 
        CHECK (focused_seconds >= 0),
    
    is_paused BOOLEAN NOT NULL DEFAULT FALSE
);