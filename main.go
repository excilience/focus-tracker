package main

import (
	"fmt"
	"os"
)

func main() {

	settings := Settings{
		DayStartHour: 4,
		Timezone:     "Europe/Istanbul",
		DailyGoal:    120,
	}

	focusService, err := NewFocusService(settings)
	if err != nil {
		fmt.Println("Failed to create focus service:", err)
		return
	}

	db, err := connectDB()
	if err != nil {
		fmt.Println("Failed to connect to PostgreSQL:", err)
		return
	}
	defer db.Close()

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
		handleHistory(focusService, db)

	case "edit":
		handleEdit(db, os.Args)

	case "help", "--help", "-h":
		printUsage()

	case "api":
		handleServe(focusService)

	case "db-list":
		dbList()

	case "db-get":
		dbGetSession(os.Args[2:])

	default:
		fmt.Println("Unknown command")
		printUsage()

	}
}
