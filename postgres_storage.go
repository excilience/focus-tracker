package main

import (
	"context"
	"database/sql"
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
