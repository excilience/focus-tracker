package main

import (
	"fmt"
	"time"
)

func NewFocusService(settings Settings) (*FocusService, error) {

	if settings.DayStartHour < 0 || settings.DayStartHour > 23 {
		return nil, fmt.Errorf("day start hour must be between 0 and 23")
	}

	if settings.DailyGoal < 0 {
		return nil, fmt.Errorf("daily goal minutes cannot be negative")
	}

	loc := time.Local

	if settings.Timezone != "" {
		loadedLoc, err := time.LoadLocation(settings.Timezone)
		if err != nil {
			return nil, fmt.Errorf("load timezone: %w", err)
		}

		loc = loadedLoc
	}

	return &FocusService{
		settings: settings,
		location: loc,
	}, nil
}

func (fs *FocusService) StartOfFocusDay(t time.Time) time.Time {
	focusDay := fs.FocusDay(t)
	y, m, d := focusDay.Date()

	return time.Date(y, m, d, fs.settings.DayStartHour, 0, 0, 0, fs.location)
}

func (fs *FocusService) StartOfWeek(t time.Time) time.Time {
	focusDay := fs.FocusDay(t)

	weekday := int(focusDay.Weekday())
	if weekday == 0 {
		weekday = 7 //Sunday
	}
	weekStartDay := focusDay.AddDate(0, 0, -(weekday - 1))
	y, m, d := weekStartDay.Date()
	return time.Date(y, m, d, fs.settings.DayStartHour, 0, 0, 0, fs.location)

}
func (fs *FocusService) StartOfMonth(t time.Time) time.Time {
	focusDay := fs.FocusDay(t)
	y, m, _ := focusDay.Date()

	return time.Date(y, m, 1, fs.settings.DayStartHour, 0, 0, 0, fs.location)
}

func (fs *FocusService) StartOfYear(t time.Time) time.Time {
	focusDay := fs.FocusDay(t)
	y, _, _ := focusDay.Date()

	return time.Date(y, time.January, 1, fs.settings.DayStartHour, 0, 0, 0, fs.location)
}
func (fs *FocusService) TotalFocusForPeriod(sessions []Session, from, to time.Time) time.Duration {
	total := time.Duration(0)

	for _, s := range sessions {
		start := s.Start.In(fs.location)
		end := s.End.In(fs.location)

		if !start.Before(to) || !end.After(from) {
			continue
		}

		overlapStart := maxTime(start, from)
		overlapEnd := minTime(end, to)

		if !overlapEnd.After(overlapStart) {
			continue
		}

		fullSessionDuration := end.Sub(start)

		if fullSessionDuration <= 0 {
			continue
		}

		overlapDuration := overlapEnd.Sub(overlapStart)

		focusedPartSeconds := float64(s.DurationSeconds) *
			float64(overlapDuration) /
			float64(fullSessionDuration)

		total += time.Duration(focusedPartSeconds) * time.Second
	}

	return total
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func (fs *FocusService) StatsForPeriod(sessions []Session, period StatsPeriod, now time.Time) (FocusStats, error) {
	now = now.In(fs.location)

	var from time.Time
	var to time.Time

	switch period {
	case StatsDay:
		from = fs.StartOfFocusDay(now)
		to = from.AddDate(0, 0, 1)

	case StatsWeek:
		from = fs.StartOfWeek(now)
		to = from.AddDate(0, 0, 7)

	case StatsMonth:
		from = fs.StartOfMonth(now)
		to = from.AddDate(0, 1, 0)

	case StatsYear:
		from = fs.StartOfYear(now)
		to = from.AddDate(1, 0, 0)

	default:
		return FocusStats{}, fmt.Errorf("unknown stats period: %s", period)
	}

	total := fs.TotalFocusForPeriod(sessions, from, to)

	avg := time.Duration(0)

	switch period {
	case StatsWeek, StatsMonth, StatsYear:
		avg = averageFocusPerDay(total, from, to)
	}

	return FocusStats{
		Period: period,
		From:   from,
		To:     to,
		Total:  total,
		Avg:    avg,
	}, nil

}

func (fs *FocusService) TotalFocusDayRange(sessions []Session) (time.Time, time.Time, bool) {
	if len(sessions) == 0 {
		return time.Time{}, time.Time{}, false
	}

	first := sessions[0].Start.In(fs.location)
	last := sessions[0].End.In(fs.location)

	for _, s := range sessions {
		start := s.Start.In(fs.location)
		end := s.End.In(fs.location)

		if start.Before(first) {
			first = start
		}

		if end.After(last) {
			last = end
		}
	}

	from := fs.StartOfFocusDay(first)
	to := fs.StartOfFocusDay(last).AddDate(0, 0, 1)

	return from, to, true
}

func (fs *FocusService) StatsForCommand(sessions []Session, arg string, now time.Time) (FocusStats, error) {

	period := StatsPeriod(arg)

	switch period {
	case StatsTotal:
		total := time.Duration(getTotalFocusTimeAll(sessions)) * time.Second

		from, to, ok := fs.TotalFocusDayRange(sessions)

		avg := time.Duration(0)
		if ok {
			avg = averageFocusPerDay(total, from, to)
		}

		return FocusStats{
			Period: StatsTotal,
			Total:  total,
			Avg:    avg,
		}, nil
	case StatsDay, StatsWeek, StatsMonth, StatsYear:
		return fs.StatsForPeriod(sessions, period, now)

	default:
		return FocusStats{}, fmt.Errorf("unknown stats period: %s", period)
	}
}

func (fs *FocusService) FocusDay(t time.Time) time.Time {
	dayStart := time.Duration(fs.settings.DayStartHour) * time.Hour

	local := t.In(fs.location)
	shifted := local.Add(-dayStart)

	y, m, d := shifted.Date()

	return time.Date(y, m, d, 0, 0, 0, 0, fs.location)
}

func (fs *FocusService) TotalFocusForDay(sessions []Session, t time.Time) time.Duration {
	from := fs.StartOfFocusDay(t)
	to := from.AddDate(0, 0, 1)

	return fs.TotalFocusForPeriod(sessions, from, to)
}

func (fs *FocusService) DailyGoal() time.Duration {
	return time.Duration(fs.settings.DailyGoal) * time.Minute
}

func (fs *FocusService) DailyProgress(sessions []Session) FocusProgress {
	total := fs.TotalFocusToday(sessions)
	goal := fs.DailyGoal()

	progress := FocusProgress{
		Total: total,
		Goal:  goal,
	}

	if goal > 0 {
		progress.Percent = float64(total) / float64(goal) * 100
		progress.IsCompleted = total >= goal
	}

	if total < goal {
		progress.Remaining = goal - total
	}

	return progress
}

func (fs *FocusService) TotalFocusToday(sessions []Session) time.Duration {
	return fs.TotalFocusForDay(sessions, time.Now().In(fs.location))
}
