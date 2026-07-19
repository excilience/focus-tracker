CREATE TABLE activities (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL,
    is_archived BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT activities_title_not_empty CHECK (length(trim(title)) > 0)
);

ALTER TABLE sessions
ADD COLUMN activity_id UUID REFERENCES activities(id);

ALTER TABLE active_sessions
ADD COLUMN activity_id UUID REFERENCES activities(id);