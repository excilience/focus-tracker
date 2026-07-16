package main

import (
	"context"
	"database/sql"
	"fmt"
)

type AppSettings struct {
	DailyGoalMinutes int
	DayStartHour     int
}

func ensureAppSettings(ctx context.Context, db *sql.DB, defaults Settings) error {
	_, err := db.ExecContext(ctx, `
			INSERT INTO app_settings (
			id,
			daily_goal_minutes,
			day_start_hour
		)
		VALUES (
			TRUE,
			$1,
			$2
		)
		ON CONFLICT (id) DO NOTHING
		`, defaults.DailyGoal, defaults.DayStartHour)

	if err != nil {
		return fmt.Errorf("ensuere app settings: %w", err)
	}

	return nil
}

func loadAppSettings(ctx context.Context, db *sql.DB) (AppSettings, error) {
	var settings AppSettings

	err := db.QueryRowContext(ctx, `
			SELECT
			daily_goal_minutes,
			day_start_hour
		FROM app_settings
		WHERE id = TRUE
		`).Scan(
		&settings.DailyGoalMinutes,
		&settings.DayStartHour,
	)

	if err != nil {
		return AppSettings{}, fmt.Errorf("load app settings: %w", err)
	}

	return settings, nil
}

func updateDailyGoalMinutes(ctx context.Context, db *sql.DB, minutes int) error {
	result, err := db.ExecContext(ctx, `
			UPDATE app_settings
		SET
			daily_goal_minutes = $1,
			updated_at = now()
		WHERE id = TRUE
	`, minutes)

	if err != nil {
		return fmt.Errorf("update daily goal minutes: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update daily goal minutes rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func updateDayStartHour(ctx context.Context, db *sql.DB, hour int) error {
	result, err := db.ExecContext(ctx, `
		UPDATE app_settings
		SET
			day_start_hour = $1,
			updated_at = now()
		WHERE id = TRUE
	`, hour)

	if err != nil {
		return fmt.Errorf("update day start hour: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update day start hour rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil

}
