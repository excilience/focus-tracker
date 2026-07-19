package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFocusDay(t *testing.T) {
	location := time.FixedZone("UTC+3", 3*60*60)

	fs := &FocusService{
		settings: Settings{
			DayStartHour: 4,
		},
		location: location,
	}

	tests := []struct {
		name     string
		input    time.Time
		expected time.Time
	}{
		{
			name: "time before day start belongs to previous focus day",
			input: time.Date(
				2026, time.June, 14,
				2, 18, 0, 0,
				location,
			),
			expected: time.Date(
				2026, time.June, 13,
				0, 0, 0, 0,
				location,
			),
		},
		{
			name: "time exactly at day start belongs to current focus day",
			input: time.Date(
				2026, time.June, 14,
				4, 0, 0, 0,
				location,
			),
			expected: time.Date(
				2026, time.June, 14,
				0, 0, 0, 0,
				location,
			),
		},
		{
			name: "time after day start belongs to current focus day",
			input: time.Date(
				2026, time.June, 14,
				15, 30, 0, 0,
				location,
			),
			expected: time.Date(
				2026, time.June, 14,
				0, 0, 0, 0,
				location,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fs.FocusDay(tt.input)

			if !result.Equal(tt.expected) {
				t.Errorf(
					"FocusDay(%s) = %s; expected %s",
					tt.input.Format(time.RFC3339),
					result.Format(time.RFC3339),
					tt.expected.Format(time.RFC3339),
				)
			}
		})
	}
}

func TestStartOfWeek(t *testing.T) {
	location := time.FixedZone("UTC+3", 3*60*60)

	fs := &FocusService{
		settings: Settings{
			DayStartHour: 4,
		},
		location: location,
	}

	tests := []struct {
		name     string
		input    time.Time
		expected time.Time
	}{
		{
			name: "given any non-monday time, returns monday of current focus week",
			input: time.Date(
				2026, time.June, 12,
				2, 18, 0, 0,
				location,
			),
			expected: time.Date(
				2026, time.June, 8,
				4, 0, 0, 0,
				location,
			),
		},
		{
			name: "given monday after DayStartHour, returns same monday",
			input: time.Date(
				2026, time.June, 15,
				10, 0, 0, 0,
				location,
			),
			expected: time.Date(
				2026, time.June, 15,
				4, 0, 0, 0,
				location,
			),
		},
		{
			name: "given monday before DayStartHour, returns previous monday",
			input: time.Date(
				2026, time.June, 15,
				2, 0, 0, 0,
				location,
			),
			expected: time.Date(
				2026, time.June, 8,
				4, 0, 0, 0,
				location,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fs.StartOfWeek(tt.input)

			if !result.Equal(tt.expected) {
				t.Errorf(
					"FocusDay(%s) = %s; expected %s",
					tt.input.Format(time.RFC3339),
					result.Format(time.RFC3339),
					tt.expected.Format(time.RFC3339),
				)

			}
		})
	}

}

func TestTotalFocusForPeriod(t *testing.T) {
	location := time.FixedZone("UTC+3", 3*60*60)

	fs := &FocusService{
		settings: Settings{
			DayStartHour: 4,
		},
		location: location,
	}

	tests := []struct {
		name     string
		input    time.Time
		sessions []Session
		from     time.Time
		to       time.Time
		expected time.Duration
	}{
		{
			name: "session fully inside period",
			sessions: []Session{
				{
					ID:              "1",
					Start:           time.Date(2026, time.June, 14, 10, 0, 0, 0, location),
					End:             time.Date(2026, time.June, 14, 12, 0, 0, 0, location),
					DurationSeconds: int((2 * time.Hour).Seconds()),
				},
			},
			from:     time.Date(2026, time.June, 14, 4, 0, 0, 0, location),
			to:       time.Date(2026, time.June, 15, 4, 0, 0, 0, location),
			expected: 2 * time.Hour,
		},
		{
			name: "session partially overlaps period",
			sessions: []Session{
				{
					ID:              "1",
					Start:           time.Date(2026, time.June, 14, 10, 0, 0, 0, location),
					End:             time.Date(2026, time.June, 14, 12, 0, 0, 0, location),
					DurationSeconds: int((2 * time.Hour).Seconds()),
				},
			},
			from:     time.Date(2026, time.June, 14, 11, 0, 0, 0, location),
			to:       time.Date(2026, time.June, 14, 13, 0, 0, 0, location),
			expected: 1 * time.Hour,
		},
		{
			name: "session outside period",
			sessions: []Session{
				{
					ID:              "1",
					Start:           time.Date(2026, time.June, 14, 10, 0, 0, 0, location),
					End:             time.Date(2026, time.June, 14, 12, 0, 0, 0, location),
					DurationSeconds: int((2 * time.Hour).Seconds()),
				},
			},
			from:     time.Date(2026, time.June, 15, 4, 0, 0, 0, location),
			to:       time.Date(2026, time.June, 16, 4, 0, 0, 0, location),
			expected: 0,
		},
		{
			name: "paused session is prorated by real focused duration",
			sessions: []Session{
				{
					ID:    "1",
					Start: time.Date(2026, time.June, 14, 10, 0, 0, 0, location),
					End:   time.Date(2026, time.June, 14, 12, 0, 0, 0, location),

					// Wall time is 2h, but actual focus is only 1h.
					DurationSeconds: int((1 * time.Hour).Seconds()),
				},
			},
			from:     time.Date(2026, time.June, 14, 11, 0, 0, 0, location),
			to:       time.Date(2026, time.June, 14, 12, 0, 0, 0, location),
			expected: 30 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			total := fs.TotalFocusForPeriod(tt.sessions, tt.from, tt.to)

			if total != tt.expected {
				t.Errorf("TotalFocusFoPeriod() = %v; expected %v", total, tt.expected)
			}
		})
	}
}

func TestStatsForPeriodMonth(t *testing.T) {
	location := time.FixedZone("UTC+3", 3*60*60)

	fs := &FocusService{
		settings: Settings{
			DayStartHour: 4,
		},
		location: location,
	}

	now := time.Date(
		2026, time.June, 14,
		15, 0, 0, 0,
		location,
	)

	sessions := []Session{
		{
			ID:              "1",
			Start:           time.Date(2026, time.June, 14, 10, 0, 0, 0, location),
			End:             time.Date(2026, time.June, 14, 11, 0, 0, 0, location),
			DurationSeconds: int((1 * time.Hour).Seconds()),
		},
		{
			ID:              "2",
			Start:           time.Date(2026, time.June, 13, 10, 0, 0, 0, location),
			End:             time.Date(2026, time.June, 13, 11, 0, 0, 0, location),
			DurationSeconds: int((1 * time.Hour).Seconds()),
		},
		{
			ID:              "3",
			Start:           time.Date(2026, time.June, 22, 10, 0, 0, 0, location),
			End:             time.Date(2026, time.June, 22, 11, 0, 0, 0, location),
			DurationSeconds: int((5 * time.Hour).Seconds()),
		},
	}

	stats, _ := fs.StatsForPeriod(
		sessions,
		StatsMonth,
		now,
	)

	expectedFrom := time.Date(
		2026, time.June, 1,
		4, 0, 0, 0,
		location,
	)

	expectedTo := time.Date(
		2026, time.July, 1,
		4, 0, 0, 0,
		location,
	)

	if stats.Period != StatsMonth {
		t.Errorf("Period = %q; expected %q", stats.Period, StatsMonth)
	}

	if !stats.From.Equal(expectedFrom) {
		t.Errorf("From = %s; expected %s", stats.From, expectedFrom)
	}

	if !stats.To.Equal(expectedTo) {
		t.Errorf("To = %s; expected %s", stats.To, expectedTo)
	}

	if stats.Total != 7*time.Hour {
		t.Errorf("Total = %v; expected %v", stats.Total, 7*time.Hour)
	}
}

func TestUpdateSessionHandlerInvalidDuration(t *testing.T) {
	fs := &FocusService{
		settings: Settings{
			DayStartHour: 4,
			Timezone:     "UTC+3",
			DailyGoal:    3600,
		},
		location: time.FixedZone("UTC+3", 3*60*60),
	}

	body := strings.NewReader(`{
		"duration": "wrong-duration"
	}`)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/sessions/test-id",
		body,
	)

	req.SetPathValue("id", "test-id")

	rr := httptest.NewRecorder()

	updateSessionHandler(
		rr,
		req,
		nil,
		fs,
	)

	if rr.Code != http.StatusBadRequest {
		t.Errorf(
			"status = %d; expected %d", rr.Code, http.StatusBadRequest)
	}
	t.Logf("status = %d", rr.Code)
	t.Logf("body = %s", rr.Body.String())
}

func TestUpdateSessionHandlerInvalidJSON(t *testing.T) {
	fs := &FocusService{
		settings: Settings{
			DayStartHour: 4,
			Timezone:     "UTC+3",
			DailyGoal:    3600,
		},
		location: time.FixedZone("UTC+3", 3*60*60),
	}

	body := strings.NewReader(`{	JSON: "Bad JSON"`)

	req := httptest.NewRequest(http.MethodPatch, "/sessions/test-id", body)

	req.SetPathValue("id", "test-id")

	rr := httptest.NewRecorder()

	updateSessionHandler(rr, req, nil, fs)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d; expected %d", rr.Code, http.StatusBadRequest)
	}
	t.Logf("status = %d", rr.Code)
	t.Logf("body = %s", rr.Body.String())

}
