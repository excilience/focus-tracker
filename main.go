package main

import (
	"context"
	"fmt"
	"os"
	"time"
)

func main() {

	defaultSettings := Settings{
		DayStartHour: 0,
		Timezone:     "",
		DailyGoal:    60,
	}

	db, err := connectDB()
	if err != nil {
		fmt.Println("Failed to connect to PostgreSQL:", err)
		return
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ensureAppSettings(ctx, db, defaultSettings); err != nil {
		fmt.Println("Failed to ensure app settings:", err)
		return
	}

	dbSettings, err := loadAppSettings(ctx, db)
	if err != nil {
		fmt.Println("Failed to load app settings:", err)
		return
	}

	runtimeSettings := defaultSettings
	runtimeSettings.DailyGoal = dbSettings.DailyGoalMinutes
	runtimeSettings.DayStartHour = dbSettings.DayStartHour

	focusService, err := NewFocusService(runtimeSettings)
	if err != nil {
		fmt.Println("Failed to create focus service:", err)
		return
	}

	if len(os.Args) < 2 {
		printUsage()
		return
	}
	switch os.Args[1] {
	case "start":
		handleStart(db)

	case "pause":
		handlePause(db)

	case "stop":
		handleStop(db)

	case "resume":
		handleResume(db)

	case "stats":
		handleStats(focusService, os.Args, db)

	case "goal":
		handleGoal(focusService, db)

	case "history":
		handleHistory(db)

	case "edit":
		handleEdit(db, os.Args)

	case "help", "--help", "-h":
		printUsage()

	case "api":
		handleServe(focusService, db)

	case "db-list":
		dbList(db)

	case "db-get":
		dbGetSession(db, os.Args[2:])

	default:
		fmt.Println("Unknown command")
		printUsage()

	}
}
