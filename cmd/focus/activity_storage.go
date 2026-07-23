package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
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
		if isActivityTitleConflict(err) {
			return Activity{}, ErrActivityTitleAlreadyExists
		}

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

func getActivityByID(ctx context.Context, db *sql.DB, id string) (Activity, error) {
	const query = `
		SELECT
			id,
			title,
			is_archived,
			created_at
		FROM activities
		WHERE id = $1;
	`

	var activity Activity

	err := db.QueryRowContext(ctx, query, id).Scan(
		&activity.ID,
		&activity.Title,
		&activity.IsArchived,
		&activity.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Activity{}, ErrActivityNotFound
		}

		return Activity{}, fmt.Errorf("get activity by id: %w", err)
	}

	return activity, nil
}

func updateActivity(ctx context.Context, db *sql.DB, id string, title *string, isArchived *bool) (Activity, error) {
	activity, err := getActivityByID(ctx, db, id)
	if err != nil {
		return Activity{}, err
	}

	if title != nil {
		normalizedTitle := strings.TrimSpace(*title)
		if normalizedTitle == "" {
			return Activity{}, fmt.Errorf("activity title is required")
		}

		activity.Title = normalizedTitle
	}

	if isArchived != nil {
		activity.IsArchived = *isArchived
	}

	const query = `
		UPDATE activities
		SET
			title = $1,
			is_archived = $2
		WHERE id = $3
		RETURNING
			id,
			title,
			is_archived,
			created_at;
	`

	err = db.QueryRowContext(ctx, query, activity.Title, activity.IsArchived, id).Scan(
		&activity.ID,
		&activity.Title,
		&activity.IsArchived,
		&activity.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Activity{}, ErrActivityNotFound
		}

		if isActivityTitleConflict(err) {
			return Activity{}, ErrActivityTitleAlreadyExists
		}

		return Activity{}, fmt.Errorf("update activity: %w", err)
	}

	return activity, nil
}

func isActivityTitleConflict(err error) bool {
	var pgErr *pgconn.PgError

	return errors.As(err, &pgErr) &&
		pgErr.Code == "23505" &&
		pgErr.ConstraintName == "activities_active_title_unique_idx"
}
