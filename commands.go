package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"
)

func handleStart(db *sql.DB) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := startSession(ctx, db)
	if err != nil {
		fmt.Println("Failed to start", err)
		return
	}
	fmt.Println("Focus session started")
}

func handleStop(db *sql.DB) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := connectDB()
	if err != nil {
		fmt.Println("Failed to connect to PostgreSQL:", err)
		return
	}
	defer db.Close()

	result, err := stopSession(ctx, db)
	if err != nil {
		fmt.Println("Failed to stop session", err)
		return
	}
	fmt.Println("Focus session stopped")
	fmt.Println("Focused for:", formatDuration(result.Session.DurationSeconds))

	if !result.Saved {
		fmt.Println("Session was not saved because it was shorter than 1 minute")
	}
}

func handleStats(fs *FocusService, args []string, db *sql.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sessions, err := loadSessionsFromDB(ctx, db)
	if err != nil {
		fmt.Println("Failed to load sessions:", err)
		return
	}

	if len(sessions) == 0 {
		fmt.Println("There are no sessions yet")
		return
	}

	if len(args) < 3 {
		printStatsUsage()
		return
	}

	stats, err := fs.StatsForCommand(sessions, os.Args[2], time.Now())
	if err != nil {
		fmt.Println("Failed to get stats:", err)
		printStatsUsage()
		return
	}

	if stats.Period == StatsTotal {
		fmt.Println("Total Focused:", formatDuration(int(stats.Total.Seconds())))
		fmt.Println("Avg. time spent per day:", formatDuration(int(stats.Avg.Seconds())))
		return
	}

	if stats.Period == StatsDay {
		fmt.Println("Period:", stats.Period)
		fmt.Println("From:", stats.From.Format("2006-01-02 15:04"))
		fmt.Println("To:", stats.To.Format("2006-01-02 15:04"))
		fmt.Println("Focused:", formatDuration(int(stats.Total.Seconds())))
		return
	}

	fmt.Println("Period:", stats.Period)
	fmt.Println("From:", stats.From.Format("2006-01-02 15:04"))
	fmt.Println("To:", stats.To.Format("2006-01-02 15:04"))
	fmt.Println("Focused:", formatDuration(int(stats.Total.Seconds())))
	fmt.Println("Avg. time spent per day:", formatDuration(int(stats.Avg.Seconds())))
}

func handlePause(db *sql.DB) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pauseSession(ctx, db); err != nil {
		fmt.Println("Failed to pause session:", err)
		return
	}
	fmt.Println("Focus session paused")
}

func handleResume(db *sql.DB) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := resumeSession(ctx, db); err != nil {
		fmt.Println("Failed to resume session:", err)
		return
	}
	fmt.Println("Focus session resumed")
}

func handleGoal(fs *FocusService, db *sql.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sessions, err := loadSessionsFromDB(ctx, db)
	if err != nil {
		fmt.Println("Failed to load sessions:", err)
		return
	}
	if len(sessions) == 0 {
		fmt.Println("There are no sessions yet")
		return
	}
	focusDay := fs.FocusDay(time.Now())
	progress := fs.DailyProgress(sessions)

	fmt.Println("Focus day:", focusDay.Format("2006-01-02"))
	fmt.Println("Focused:", formatDuration(int(progress.Total.Seconds())))
	fmt.Println("Goal:", formatDuration(int(progress.Goal.Seconds())))
	fmt.Println("Remaining:", formatDuration(int(progress.Remaining.Seconds())))
	fmt.Printf("Progress: %.0f%%\n", progress.Percent)

	if progress.IsCompleted {
		fmt.Println("Goal completed!")
	}
}

func handleHistory(db *sql.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sessions, err := loadSessionsFromDB(ctx, db)
	if err != nil {
		fmt.Println("Failed to load sessions:", err)
		return
	}
	if len(sessions) == 0 {
		fmt.Println("There are no sessions yet")
		return
	}

	for i, session := range sessions {
		fmt.Printf("%d. [%s] %s → %s | %s\n",
			i+1,
			session.ID,
			timeFormat(session.Start),
			timeFormat(session.End),
			formatDuration(session.DurationSeconds),
		)
	}
}

func handleEdit(db *sql.DB, args []string) {
	if len(args) < 4 {
		printEditUsage()
		return
	}

	id := args[2]
	durationText := args[3]

	duration, err := time.ParseDuration(durationText)
	if err != nil {
		fmt.Println("Invalid duration:", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	updatedSession, err := updateSessionDuration(ctx, db, id, duration)
	if err != nil {
		switch {
		case errors.Is(err, ErrSessionNotFound):
			fmt.Println("Session not found:", id)
		default:
			fmt.Println("Failed to edit session:", err)
		}
		return
	}

	fmt.Println("Session updated")
	fmt.Println("ID:", updatedSession.ID)
	fmt.Println("New duration:", formatDuration(updatedSession.DurationSeconds))
}

func handleServe(fs *FocusService) {

	db, err := connectDB()
	if err != nil {
		fmt.Println("Failed to connect to PostgreSQL:", err)
		return
	}
	defer db.Close()

	err = runApiServer(fs, db)
	if err != nil {
		fmt.Println("Failed to start API server:", err)
		return
	}
}
func connectDB() (*sql.DB, error) {

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set")
	}

	db, err := openDatabase(dsn)
	if err != nil {
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	return db, nil
}

func dbList() {
	db, err := connectDB()
	if err != nil {
		fmt.Println("Failed to connect to PostgreSQL:", err)
		return
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sessions, err := loadSessionsFromDB(ctx, db)
	if err != nil {
		fmt.Println("Failed to load sessions from PostgreSQL:", err)
		return
	}

	for _, session := range sessions {
		fmt.Printf("%s | %s → %s | %s\n", session.ID, timeFormat(session.Start), timeFormat(session.End), formatDuration(session.DurationSeconds))
	}
}

func dbGetSession(args []string) {

	if len(os.Args) < 3 {
		fmt.Println("Usage: focus db-get <session-id>")
		return
	}

	id := args[0]

	db, err := connectDB()
	if err != nil {
		fmt.Println("Failed to connect to PostgreSQL:", err)
		return
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := getSessionByIDFromDB(ctx, db, id)
	if err != nil {
		fmt.Println("Failed to get session:", err)
		return
	}
	fmt.Printf(
		"%s | %s → %s | %s\n",
		session.ID,
		timeFormat(session.Start),
		timeFormat(session.End),
		formatDuration(session.DurationSeconds),
	)
}
