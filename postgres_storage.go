package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

func insertSession(ctx context.Context, db *sql.DB, session Session) error {
	const query = `
		INSERT INTO sessions (
			id,
			start_time,
			end_time,
			duration_seconds
		)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO NOTHING;
	`

	_, err := db.ExecContext(
		ctx,
		query,
		session.ID,
		session.Start,
		session.End,
		session.DurationSeconds,
	)
	if err != nil {
		return fmt.Errorf("insert session %s: %w", session.ID, err)
	}
	return nil
}

func importSessions(ctx context.Context, db *sql.DB, sessions []Session) error {
	for _, session := range sessions {
		if err := insertSession(ctx, db, session); err != nil {
			return err
		}
	}
	return nil
}

func loadSessionsFromDB(ctx context.Context, db *sql.DB) ([]Session, error) {
	const query = `
		SELECT
			id,
			start_time,
			end_time,
			duration_seconds
		FROM sessions
		ORDER BY start_time;
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	sessions := make([]Session, 0)

	for rows.Next() {
		var session Session

		if err := rows.Scan(
			&session.ID,
			&session.Start,
			&session.End,
			&session.DurationSeconds,
		); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read session rows: %w", err)
	}

	return sessions, nil
}

func getSessionByIDFromDB(ctx context.Context, db *sql.DB, id string) (Session, error) {
	const query = `
		SELECT
			id,
			start_time,
			end_time,
			duration_seconds
		FROM sessions
		WHERE id = $1;
	`

	var session Session
	err := db.QueryRowContext(ctx, query, id).Scan(
		&session.ID,
		&session.Start,
		&session.End,
		&session.DurationSeconds,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Session{}, fmt.Errorf("session not found")
		}

		return Session{}, fmt.Errorf("query sessions by id: %w", err)
	}

	return session, nil
}
