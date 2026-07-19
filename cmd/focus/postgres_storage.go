package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func newID() string {
	return uuid.NewString()
}

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

func loadSessionsByPeriodFromDB(ctx context.Context, db *sql.DB, from, to time.Time) ([]Session, error) {
	const query = `
		SELECT
			id,
			start_time,
			end_time,
			duration_seconds
		FROM sessions
		WHERE start_time < $2
			AND end_time > $1
		ORDER BY start_time;
	`

	rows, err := db.QueryContext(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("query sessions by period: %w", err)
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
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}

	return sessions, nil
}

func getSessionByIDFromDB(ctx context.Context, executor DBExecutor, id string) (Session, error) {
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
	err := executor.QueryRowContext(ctx, query, id).Scan(
		&session.ID,
		&session.Start,
		&session.End,
		&session.DurationSeconds,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Session{}, ErrSessionNotFound
		}

		return Session{}, fmt.Errorf("query sessions by id: %w", err)
	}

	return session, nil
}

func updateSessionDuration(ctx context.Context, executor DBExecutor, id string, duration time.Duration) (Session, error) {
	const query = `
		UPDATE sessions
		SET duration_seconds = $1
		WHERE id = $2
		RETURNING
			id,
			start_time,
			end_time,
			duration_seconds;
	`
	var session Session

	err := executor.QueryRowContext(ctx, query, int(duration.Seconds()), id).Scan(
		&session.ID,
		&session.Start,
		&session.End,
		&session.DurationSeconds,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Session{}, ErrSessionNotFound
		}

		return Session{}, fmt.Errorf("update session duration: %w", err)
	}

	return session, nil
}

func deleteSession(ctx context.Context, executor DBExecutor, id string) (Session, error) {
	const query = `
		DELETE FROM sessions
		WHERE id = $1
		RETURNING
			id,
			start_time,
			end_time,
			duration_seconds;
	`

	var session Session

	err := executor.QueryRowContext(ctx, query, id).Scan(
		&session.ID,
		&session.Start,
		&session.End,
		&session.DurationSeconds,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Session{}, ErrSessionNotFound
		}

		return Session{}, fmt.Errorf("delete session: %w", err)
	}

	return session, nil
}

func createSession(ctx context.Context, executor DBExecutor, session Session) error {
	const query = `
		INSERT INTO sessions (
			id,
			start_time,
			end_time,
			duration_seconds
		)
		VALUES ($1, $2, $3, $4);
	`

	_, err := executor.ExecContext(ctx, query, session.ID, session.Start, session.End, session.DurationSeconds)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

//active sesions

func createActiveSession(ctx context.Context, executor DBExecutor, activeSession ActiveSession) error {
	const query = `
		INSERT INTO active_sessions (
			id,
			start_time,
			last_resume,
			focused_seconds,
			is_paused
		)
		VALUES (1, $1, $2, $3, $4);
	`

	_, err := executor.ExecContext(
		ctx,
		query,
		activeSession.Start,
		activeSession.LastResume,
		activeSession.FocusedSeconds,
		activeSession.IsPaused,
	)
	if err != nil {
		return fmt.Errorf("create active session: %w", err)
	}

	return nil
}

func loadActiveSession(ctx context.Context, executor DBExecutor) (ActiveSession, error) {
	const query = `
		SELECT
			start_time,
			last_resume,
			focused_seconds,
			is_paused
		FROM active_sessions
		WHERE id = 1;
	`
	var activeSession ActiveSession

	err := executor.QueryRowContext(ctx, query).Scan(
		&activeSession.Start,
		&activeSession.LastResume,
		&activeSession.FocusedSeconds,
		&activeSession.IsPaused,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ActiveSession{}, ErrNoActiveSession
		}

		return ActiveSession{}, fmt.Errorf("load active session: %w", err)
	}

	return activeSession, nil
}

func updateActiveSession(ctx context.Context, executor DBExecutor, activeSession ActiveSession) error {
	const query = `
		UPDATE active_sessions
		SET
			start_time = $1,
			last_resume = $2,
			focused_seconds = $3,
			is_paused = $4
		WHERE id = 1;
	`

	result, err := executor.ExecContext(
		ctx,
		query,
		activeSession.Start,
		activeSession.LastResume,
		activeSession.FocusedSeconds,
		activeSession.IsPaused,
	)

	if err != nil {
		return fmt.Errorf("update active session: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get affected rows: %w", err)
	}

	if affected == 0 {
		return ErrNoActiveSession
	}
	return nil
}

func deleteActiveSessionFromDB(ctx context.Context, executor DBExecutor) error {
	const query = `
		DELETE FROM active_sessions
		WHERE id = 1;
	`

	result, err := executor.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("delete active session: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get affected rows: %w", err)
	}

	if affected == 0 {
		return ErrNoActiveSession
	}
	return nil
}
