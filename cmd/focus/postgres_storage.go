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
			duration_seconds,
			activity_id
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO NOTHING;
	`

	_, err := db.ExecContext(
		ctx,
		query,
		session.ID,
		session.Start,
		session.End,
		session.DurationSeconds,
		session.ActivityID,
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
			s.id,
			s.start_time,
			s.end_time,
			s.duration_seconds,
			s.activity_id,
			a.title
		FROM sessions s
		LEFT JOIN activities a ON a.id = s.activity_id
		ORDER BY s.end_time DESC, s.start_time DESC;
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	sessions := make([]Session, 0)

	for rows.Next() {
		var session Session
		var activityID sql.NullString
		var activityTitle sql.NullString

		if err := rows.Scan(
			&session.ID,
			&session.Start,
			&session.End,
			&session.DurationSeconds,
			&activityID,
			&activityTitle,
		); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}

		session.ActivityID = nullableStringPtr(activityID)
		session.ActivityTitle = nullableStringPtr(activityTitle)

		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read session rows: %w", err)
	}

	return sessions, nil
}

func nullableStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}

	return &value.String
}

func loadSessionsByPeriodFromDB(ctx context.Context, db *sql.DB, from, to time.Time) ([]Session, error) {
	const query = `
		SELECT
			s.id,
			s.start_time,
			s.end_time,
			s.duration_seconds,
			s.activity_id,
			a.title
		FROM sessions s
		LEFT JOIN activities a ON a.id = s.activity_id
		WHERE s.start_time < $2
			AND s.end_time > $1
		ORDER BY s.start_time;
	`

	rows, err := db.QueryContext(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("query sessions by period: %w", err)
	}
	defer rows.Close()

	sessions := make([]Session, 0)

	for rows.Next() {
		var session Session
		var activityID sql.NullString
		var activityTitle sql.NullString

		if err := rows.Scan(
			&session.ID,
			&session.Start,
			&session.End,
			&session.DurationSeconds,
			&activityID,
			&activityTitle,
		); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}

		session.ActivityID = nullableStringPtr(activityID)
		session.ActivityTitle = nullableStringPtr(activityTitle)

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
			s.id,
			s.start_time,
			s.end_time,
			s.duration_seconds,
			s.activity_id,
			a.title
		FROM sessions s
		LEFT JOIN activities a ON a.id = s.activity_id
		WHERE s.id = $1;
	`

	var session Session
	var activityID sql.NullString
	var activityTitle sql.NullString

	err := executor.QueryRowContext(ctx, query, id).Scan(
		&session.ID,
		&session.Start,
		&session.End,
		&session.DurationSeconds,
		&activityID,
		&activityTitle,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Session{}, ErrSessionNotFound
		}

		return Session{}, fmt.Errorf("query sessions by id: %w", err)
	}

	session.ActivityID = nullableStringPtr(activityID)
	session.ActivityTitle = nullableStringPtr(activityTitle)

	return session, nil
}

func updateSessionDuration(ctx context.Context, executor DBExecutor, id string, duration time.Duration) (Session, error) {
	const query = `
		WITH updated_session AS (
			UPDATE sessions
			SET duration_seconds = $1
			WHERE id = $2
			RETURNING
				id,
				start_time,
				end_time,
				duration_seconds,
				activity_id
		)
		SELECT
			s.id,
			s.start_time,
			s.end_time,
			s.duration_seconds,
			s.activity_id,
			a.title
		FROM updated_session s
		LEFT JOIN activities a ON a.id = s.activity_id;
	`

	var session Session
	var activityID sql.NullString
	var activityTitle sql.NullString

	err := executor.QueryRowContext(ctx, query, int(duration.Seconds()), id).Scan(
		&session.ID,
		&session.Start,
		&session.End,
		&session.DurationSeconds,
		&activityID,
		&activityTitle,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Session{}, ErrSessionNotFound
		}

		return Session{}, fmt.Errorf("update session duration: %w", err)
	}

	session.ActivityID = nullableStringPtr(activityID)
	session.ActivityTitle = nullableStringPtr(activityTitle)

	return session, nil
}

func deleteSession(ctx context.Context, executor DBExecutor, id string) (Session, error) {
	const query = `
		WITH deleted_session AS (
			DELETE FROM sessions
			WHERE id = $1
			RETURNING
				id,
				start_time,
				end_time,
				duration_seconds,
				activity_id
		)
		SELECT
			s.id,
			s.start_time,
			s.end_time,
			s.duration_seconds,
			s.activity_id,
			a.title
		FROM deleted_session s
		LEFT JOIN activities a ON a.id = s.activity_id;
	`

	var session Session
	var activityID sql.NullString
	var activityTitle sql.NullString

	err := executor.QueryRowContext(ctx, query, id).Scan(
		&session.ID,
		&session.Start,
		&session.End,
		&session.DurationSeconds,
		&activityID,
		&activityTitle,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Session{}, ErrSessionNotFound
		}

		return Session{}, fmt.Errorf("delete session: %w", err)
	}

	session.ActivityID = nullableStringPtr(activityID)
	session.ActivityTitle = nullableStringPtr(activityTitle)

	return session, nil
}

func createSession(ctx context.Context, executor DBExecutor, session Session) error {
	const query = `
		INSERT INTO sessions (
			id,
			start_time,
			end_time,
			duration_seconds,
			activity_id
		)
		VALUES ($1, $2, $3, $4, $5);
	`

	_, err := executor.ExecContext(
		ctx,
		query,
		session.ID,
		session.Start,
		session.End,
		session.DurationSeconds,
		session.ActivityID,
	)

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
			is_paused,
			activity_id
		)
		VALUES (1, $1, $2, $3, $4, $5);
	`

	_, err := executor.ExecContext(
		ctx,
		query,
		activeSession.Start,
		activeSession.LastResume,
		activeSession.FocusedSeconds,
		activeSession.IsPaused,
		activeSession.ActivityID,
	)
	if err != nil {
		return fmt.Errorf("create active session: %w", err)
	}

	return nil
}

func loadActiveSession(ctx context.Context, executor DBExecutor) (ActiveSession, error) {
	const query = `
		SELECT
			active.start_time,
			active.last_resume,
			active.focused_seconds,
			active.is_paused,
			active.activity_id,
			activity.title
		FROM active_sessions active
		LEFT JOIN activities activity ON activity.id = active.activity_id
		WHERE active.id = 1;
	`

	var activeSession ActiveSession
	var activityID sql.NullString
	var activityTitle sql.NullString

	err := executor.QueryRowContext(ctx, query).Scan(
		&activeSession.Start,
		&activeSession.LastResume,
		&activeSession.FocusedSeconds,
		&activeSession.IsPaused,
		&activityID,
		&activityTitle,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ActiveSession{}, ErrNoActiveSession
		}

		return ActiveSession{}, fmt.Errorf("load active session: %w", err)
	}

	activeSession.ActivityID = nullableStringPtr(activityID)
	activeSession.ActivityTitle = nullableStringPtr(activityTitle)

	return activeSession, nil
}

func updateActiveSession(ctx context.Context, executor DBExecutor, activeSession ActiveSession) error {
	const query = `
		UPDATE active_sessions
		SET
			start_time = $1,
			last_resume = $2,
			focused_seconds = $3,
			is_paused = $4,
			activity_id = $5
		WHERE id = 1;
	`

	result, err := executor.ExecContext(
		ctx,
		query,
		activeSession.Start,
		activeSession.LastResume,
		activeSession.FocusedSeconds,
		activeSession.IsPaused,
		activeSession.ActivityID,
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
