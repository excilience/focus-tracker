CREATE TABLE app_settings (
    id BOOLEAN PRIMARY KEY DEFAULT TRUE,
    daily_goal_minutes INTEGER NOT NULL,
    day_start_hour INTEGER NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT app_settings_single_row CHECK (id = TRUE),
    CONSTRAINT daily_goal_minutes_positive CHECK (daily_goal_minutes > 0),
    CONSTRAINT day_start_hour_range CHECK (day_start_hour >= 0 AND day_start_hour <= 23)
);