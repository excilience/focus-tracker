package main

import (
	"fmt"
	"os"
	"time"
)

func handleStart() {
	err := startSession()
	if err != nil {
		fmt.Println("Failed to start", err)
		return
	}
	fmt.Println("Focus session started")
}

func handleStop() {
	currentSession, err := stopSession()
	if err != nil {
		fmt.Println("Failed to stop session", err)
		return
	}
	fmt.Println("Focus session stopped")
	fmt.Println("Focused for:", formatDuration(currentSession.DurationSeconds))
}

func handleStats(fs *FocusService, args []string) {
	sessions, err := loadSessions()
	if err != nil {
		fmt.Println("Failed to load sessions:", err)
		return
	}

	if len(sessions) == 0 {
		fmt.Println("There are no sessions yet")
		return
	}

	if len(os.Args) < 3 {
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

func handlePause() {
	err := pauseSession()
	if err != nil {
		fmt.Println("Failed to pause session:", err)
		return
	}
	fmt.Println("Focus session paused")
}

func handleResume() {
	err := resumeSession()
	if err != nil {
		fmt.Println("Failed to resume session:", err)
		return
	}
	fmt.Println("Focus session resumed")
}

func handleGoal(fs *FocusService) {
	sessions, err := loadSessions()
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

func handleHistory() {
	sessions, err := loadSessions()
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

func handleEdit(args []string) {
	if len(os.Args) < 4 {
		printEditUsage()
		return
	}

	id := os.Args[2]
	durationText := os.Args[3]

	duration, err := time.ParseDuration(durationText)
	if err != nil {
		fmt.Println("Invalid duration:", err)
		return
	}

	updatedSession, err := editSessionDuration(id, duration)
	if err != nil {
		fmt.Println("Failed to edit session:", err)
		return
	}

	fmt.Println("Session updated")
	fmt.Println("ID:", updatedSession.ID)
	fmt.Println("New duration:", formatDuration(updatedSession.DurationSeconds))
}
