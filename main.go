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

	if len(os.Args) < 2 {
		printUsage()
		return
	}
	switch os.Args[1] {
	case "start":
		handleStart()

	case "pause":
		handlePause()

	case "stop":
		handleStop()

	case "resume":
		handleResume()

	case "stats":
		handleStats(focusService, os.Args)

	case "goal":
		handleGoal(focusService)

	case "history":
		handleHistory()

	case "edit":
		handleEdit(os.Args)

	case "help", "--help", "-h":
		printUsage()

	case "serve":
		handleServe(focusService)

	case "db-import":
		dbImport()

	case "db-list":
		dbList()

	default:
		fmt.Println("Unknown command")
		printUsage()

	}
}
