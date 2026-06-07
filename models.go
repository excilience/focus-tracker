package main

import "time"

type StatsPeriod string

const (
	StatsDay   StatsPeriod = "day"
	StatsWeek  StatsPeriod = "week"
	StatsMonth StatsPeriod = "month"
	StatsYear  StatsPeriod = "year"
	StatsTotal StatsPeriod = "total"
)

type StatsResponse struct {
	Period       StatsPeriod `json:"period"`
	From         string      `json:"from,omitempty"`
	To           string      `json:"to,omitempty"`
	TotalSeconds int         `json:"total_seconds"`
	TotalHuman   string      `json:"total_human"`
	AvgSeconds   int         `json:"avg_seconds,omitempty"`
	AvgHuman     string      `json:"avg_human,omitempty"`
}

type FocusStats struct {
	Period StatsPeriod
	From   time.Time
	To     time.Time
	Total  time.Duration
	Avg    time.Duration
}

type Session struct {
	ID              string    `json:"id"`
	Start           time.Time `json:"start"`
	End             time.Time `json:"end"`
	DurationSeconds int       `json:"duration_seconds"`
}

type StopSessionResult struct {
	Session Session
	Saved   bool
}

type Settings struct {
	DayStartHour int
	Timezone     string
	DailyGoal    int
}

type FocusService struct {
	settings Settings
	location *time.Location
}

type GoalResponse struct {
	FocusDay         string `json:"focus_day"`
	FocusedSeconds   int    `json:"focused_seconds"`
	FocusedHuman     string `json:"focused_human"`
	GoalSeconds      int    `json:"goal_seconds"`
	GoalHuman        string `json:"goal_human"`
	RemainingSeconds int    `json:"remaining_seconds"`
	RemainingHuman   string `json:"remaining_human"`
	Percent          int    `json:"percent"`
	IsCompleted      bool   `json:"is_completed"`
}

type FocusProgress struct {
	Total       time.Duration
	Goal        time.Duration
	Remaining   time.Duration
	Percent     float64
	IsCompleted bool
}

type ActiveSession struct {
	Start          time.Time `json:"start"`
	LastResume     time.Time `json:"last_resume"`
	FocusedSeconds int       `json:"focused_seconds"`
	IsPaused       bool      `json:"is_paused"`
}
