package main

import (
	"fmt"
	"time"
)

const humanTimeLayout = "02.01.2006 15:04:05"

func timeFormat(t time.Time) string {
	return t.Format(humanTimeLayout)
}

func formatDuration(totalSeconds int) string {
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60

	if hours > 0 {
		return fmt.Sprintf("%dh %02dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
