package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func startSession(ctx context.Context, db *sql.DB) error {
	now := time.Now()

	activeSession := ActiveSession{
		Start:          now,
		LastResume:     now,
		FocusedSeconds: 0,
		IsPaused:       false,
	}

	if err := createActiveSession(ctx, db, activeSession); err != nil {
		return fmt.Errorf("start session: %w", err)
	}

	return nil
}

func pauseSession(ctx context.Context, db *sql.DB) error {
	activeSession, err := loadActiveSession(ctx, db)
	if err != nil {
		return err
	}

	if activeSession.IsPaused {
		return fmt.Errorf("session is already paused")
	}

	now := time.Now()

	activeDuration := now.Sub(activeSession.LastResume)
	activeSession.FocusedSeconds += int(activeDuration.Seconds())

	activeSession.IsPaused = true

	return updateActiveSession(ctx, db, activeSession)
}

func stopSession(ctx context.Context, db *sql.DB) (StopSessionResult, error) {

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return StopSessionResult{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	activeSession, err := loadActiveSession(ctx, tx)
	if err != nil {
		return StopSessionResult{}, err
	}

	now := time.Now()
	totalSeconds := activeSession.FocusedSeconds

	if !activeSession.IsPaused {
		activeDuration := now.Sub(activeSession.LastResume)
		totalSeconds += int(activeDuration.Seconds())
	}

	currentSession := Session{
		ID:              newSessionID(activeSession.Start),
		Start:           activeSession.Start,
		End:             now,
		DurationSeconds: totalSeconds,
	}

	saved := totalSeconds >= 60

	if saved {
		if err := createSession(ctx, tx, currentSession); err != nil {
			return StopSessionResult{}, err
		}
	}

	if err := deleteActiveSessionFromDB(ctx, tx); err != nil {
		return StopSessionResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return StopSessionResult{}, fmt.Errorf("commit transaction: %w", err)
	}

	return StopSessionResult{
		Session: currentSession,
		Saved:   saved,
	}, nil
}

func resumeSession(ctx context.Context, db *sql.DB) error {
	activeSession, err := loadActiveSession(ctx, db)
	if err != nil {
		return err
	}

	if !activeSession.IsPaused {
		return fmt.Errorf("session is not paused")
	}

	activeSession.LastResume = time.Now()
	activeSession.IsPaused = false

	return updateActiveSession(ctx, db, activeSession)
}

func getTotalFocusTimeAll(sessions []Session) int {
	total := 0
	for _, session := range sessions {
		total += session.DurationSeconds
	}
	return total

}

func countFocusDays(from, to time.Time) int {
	days := 0

	for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
		days++
	}
	return days
}

func averageFocusPerDay(total time.Duration, from, to time.Time) time.Duration {
	days := countFocusDays(from, to)
	if days == 0 {
		return 0
	}
	return total / time.Duration(days)
}
