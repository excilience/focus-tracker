package main

import (
	"fmt"
	"time"
)

func timeFormat(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func formatDuration(totalSeconds int) string {
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60

	if hours > 0 {
		return fmt.Sprintf("%dh %02dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
