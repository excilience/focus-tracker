package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func createActivity(ctx context.Context, db *sql.DB, title string) (Activity, error) {
	normalizedTitle := strings.TrimSpace(title)

	if normalizedTitle == "" {
		return Activity{}, fmt.Errorf("activity title is required")
	}

	now := time.Now().UTC()

	activity := Activity{
		ID:         newID(),
		Title:      normalizedTitle,
		IsArchived: false,
		CreatedAt:  now,
	}

	err := db.QueryRowContext(ctx, `
		INSERT INTO activities (
			id,
			title,
			is_archived,
			created_at
		)
		VALUES ($1, $2, $3, $4)
		RETURNING
			id,
			title,
			is_archived,
			created_at
	`, activity.ID, activity.Title, activity.IsArchived, activity.CreatedAt).Scan(
		&activity.ID,
		&activity.Title,
		&activity.IsArchived,
		&activity.CreatedAt,
	)

	if err != nil {
		return Activity{}, fmt.Errorf("create activity: %w", err)
	}

	return activity, nil
}

func loadActivities(ctx context.Context, db *sql.DB) ([]Activity, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT
			id,
			title,
			is_archived,
			created_at
		FROM activities
		WHERE is_archived = FALSE
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("load activities: %w", err)
	}
	defer rows.Close()

	activities := make([]Activity, 0)

	for rows.Next() {
		var activity Activity

		if err := rows.Scan(
			&activity.ID,
			&activity.Title,
			&activity.IsArchived,
			&activity.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan activity: %w", err)
		}

		activities = append(activities, activity)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate activities: %w", err)
	}

	return activities, nil
}
