ALTER TABLE active_sessions
DROP COLUMN activity_id;

ALTER TABLE sessions
DROP COLUMN activity_id;

DROP TABLE activities;s