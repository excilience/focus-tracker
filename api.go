package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"
)

func runApiServer(fs *FocusService, db *sql.DB) error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /sessions", func(w http.ResponseWriter, r *http.Request) {
		getSessionsHandler(w, r, fs, db)
	})
	mux.HandleFunc("GET /sessions/{id}", func(w http.ResponseWriter, r *http.Request) { getSessionByIDHandler(w, r, db, fs.location) })
	mux.HandleFunc("GET /sessions/active", func(w http.ResponseWriter, r *http.Request) {
		getActiveSessionHandler(w, r, db)
	})
	mux.HandleFunc("GET /stats", func(w http.ResponseWriter, r *http.Request) {
		getStatsHandler(w, r, fs, db)
	})
	mux.HandleFunc("GET /goal", func(w http.ResponseWriter, r *http.Request) {
		getGoalHandler(w, r, fs, db)
	})

	mux.HandleFunc("POST /sessions/start", func(w http.ResponseWriter, r *http.Request) {
		startSessionHandler(w, r, db)
	})
	mux.HandleFunc("POST /sessions/pause", func(w http.ResponseWriter, r *http.Request) {
		pauseSessionHandler(w, r, db)
	})
	mux.HandleFunc("POST /sessions/resume", func(w http.ResponseWriter, r *http.Request) {
		resumeSessionHandler(w, r, db)
	})
	mux.HandleFunc("POST /sessions/stop", func(w http.ResponseWriter, r *http.Request) {
		stopSessionHandler(w, r, db, fs.location)
	})

	mux.HandleFunc("PATCH /sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		updateSessionHandler(w, r, db, fs.location)
	})
	mux.HandleFunc("DELETE /sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		deleteSessionHandler(w, r, db, fs.location)
	})

	addr := ":8080"

	fmt.Println("API server started on http://localhost" + addr)

	return http.ListenAndServe(addr, mux)
}

func startSessionHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := startSession(ctx, db); err != nil {
		switch {
		case errors.Is(err, ErrSessionAlreadyActive):
			writeJSON(w, http.StatusConflict, map[string]string{
				"error": err.Error(),
			})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "failed to start session",
			})
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"message": "focus session started",
	})
}

func stopSessionHandler(w http.ResponseWriter, _ *http.Request, db *sql.DB, location *time.Location) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := stopSession(ctx, db)
	if err != nil {
		switch {
		case errors.Is(err, ErrNoActiveSession):
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": err.Error(),
			})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "failed to stop session",
			})
		}

		return
	}

	if !result.Saved {
		writeJSON(w, http.StatusOK, map[string]any{
			"message": "focus session stopped but not saved because it was shorter than 1 minute",
			"saved":   false,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "focus session stopped and saved",
		"saved":   true,
		"session": toSessionResponse(result.Session, location),
	})
}

func pauseSessionHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := pauseSession(ctx, db); err != nil {
		switch {
		case errors.Is(err, ErrNoActiveSession):
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": err.Error(),
			})

		case errors.Is(err, ErrSessionAlreadyPaused):
			writeJSON(w, http.StatusConflict, map[string]string{
				"error": err.Error(),
			})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "failed to pause session",
			})
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "focus session paused",
	})
}

func resumeSessionHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := resumeSession(ctx, db); err != nil {
		switch {
		case errors.Is(err, ErrNoActiveSession):
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": err.Error(),
			})
		case errors.Is(err, ErrSessionAlreadyRunning):
			writeJSON(w, http.StatusConflict, map[string]string{
				"error": err.Error(),
			})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "failed to resume session",
			})
		}

		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "focus session resumed",
	})
}

func getActiveSessionHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	activeSession, err := loadActiveSession(ctx, db)
	if err != nil {
		switch {
		case errors.Is(err, ErrNoActiveSession):
			writeJSON(w, http.StatusNotFound, map[string]any{
				"error": err.Error(),
			})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "failed to load active session",
			})
		}
		return
	}
	focusedSeconds := activeSession.FocusedSeconds

	if !activeSession.IsPaused && !activeSession.LastResume.IsZero() {
		activeDuration := time.Since(activeSession.LastResume)
		focusedSeconds += int(activeDuration.Seconds())
	}

	response := ActiveSessionResponse{
		Start:          timeFormat(activeSession.Start),
		FocusedSeconds: focusedSeconds,
		FocusedHuman:   formatDuration(focusedSeconds),
		IsPaused:       activeSession.IsPaused,
	}

	if !activeSession.LastResume.IsZero() {
		response.LastResume = timeFormat(activeSession.LastResume)
	}

	writeJSON(w, http.StatusOK, response)
}

func getStatsHandler(w http.ResponseWriter, r *http.Request, fs *FocusService, db *sql.DB) {

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	sessions, err := loadSessionsFromDB(ctx, db)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to load sessions",
		})
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "day"
	}

	stats, err := fs.StatsForCommand(sessions, period, time.Now())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, toStatsResponse(stats))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func getSessionsHandler(w http.ResponseWriter, r *http.Request, fs *FocusService, db *sql.DB) {

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	fromText := r.URL.Query().Get("from")
	toText := r.URL.Query().Get("to")

	if fromText == "" && toText == "" {
		sessions, err := loadSessionsFromDB(ctx, db)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "failed to load sessions",
			})
			return
		}
		writeJSON(w, http.StatusOK, toSessionResponses(sessions, fs.location))
		return
	}

	if fromText == "" || toText == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "both 'from' and 'to' are required",
			"example": "/sessions?from=YYYY-MM-DD&to=YYYY-MM-DD",
		})
		return
	}

	fromDate, err := time.ParseInLocation(dateLayout, fromText, fs.location)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid from date, expected YYYY-MM-DD",
		})
		return
	}

	toDate, err := time.ParseInLocation(dateLayout, toText, fs.location)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid to date, expected YYYY-MM-DD",
		})
		return
	}

	from := fs.StartOfFocusDayFromDate(fromDate)
	to := fs.StartOfFocusDayFromDate(toDate).AddDate(0, 0, 1)

	if !to.After(from) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "'to' date must be the same as or later than 'from' date",
		})
		return
	}

	sessions, err := loadSessionsByPeriodFromDB(ctx, db, from, to)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to load sessions by period",
		})
		return
	}

	writeJSON(w, http.StatusOK, toSessionResponses(sessions, fs.location))
}

func toStatsResponse(stats FocusStats) StatsResponse {
	totalSeconds := int(stats.Total.Seconds())
	avgSeconds := int(stats.Avg.Seconds())

	response := StatsResponse{
		Period:       stats.Period,
		TotalSeconds: totalSeconds,
		TotalHuman:   formatDuration(totalSeconds),
	}

	if !stats.From.IsZero() {
		response.From = timeFormat(stats.From)
	}

	if !stats.To.IsZero() {
		response.To = timeFormat(stats.To)
	}

	if avgSeconds > 0 {
		response.AvgSeconds = avgSeconds
		response.AvgHuman = formatDuration(avgSeconds)
	}
	return response
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Println("failed to write JSON response:", err)
	}
}

func getGoalHandler(w http.ResponseWriter, r *http.Request, fs *FocusService, db *sql.DB) {

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	sessions, err := loadSessionsFromDB(ctx, db)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to load sessions",
		})
		return
	}

	progress := fs.DailyProgress(sessions)

	writeJSON(w, http.StatusOK, toGoalResponse(fs, progress, time.Now()))
}

func toGoalResponse(fs *FocusService, progress FocusProgress, now time.Time) GoalResponse {
	totalSeconds := int(progress.Total.Seconds())
	goalSeconds := int(progress.Goal.Seconds())
	remainingSeconds := int(progress.Remaining.Seconds())

	response := GoalResponse{
		FocusDay:         timeFormat(fs.StartOfFocusDay(now)),
		FocusedSeconds:   totalSeconds,
		FocusedHuman:     formatDuration(totalSeconds),
		GoalSeconds:      goalSeconds,
		GoalHuman:        formatDuration(goalSeconds),
		RemainingSeconds: remainingSeconds,
		RemainingHuman:   formatDuration(remainingSeconds),
		Percent:          int(math.Round(progress.Percent)),
		IsCompleted:      progress.IsCompleted,
	}

	return response

}

func updateSessionHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, location *time.Location) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "session id is required",
		})
		return
	}

	var request updateSessionRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid JSON body",
		})
		return
	}

	if request.Duration == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "duration is required",
		})
		return
	}

	duration, err := time.ParseDuration(request.Duration)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid duration format",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	updatedSession, err := updateSessionDuration(ctx, db, id, duration)
	if err != nil {
		switch {
		case errors.Is(err, ErrSessionNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": err.Error(),
			})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "failed to update session",
			})
		}

		return
	}

	writeJSON(w, http.StatusOK, toSessionResponse(updatedSession, location))
}

func deleteSessionHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, location *time.Location) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "session id is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	deletedSession, err := deleteSession(ctx, db, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrSessionNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": err.Error(),
			})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "failed to delete session",
			})
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "session deleted",
		"session": toSessionResponse(deletedSession, location),
	})
}

func toSessionResponse(session Session, location *time.Location) SessionResponse {
	return SessionResponse{
		ID:              session.ID,
		Start:           timeFormat(session.Start.In(location)),
		End:             timeFormat(session.End.In(location)),
		DurationSeconds: session.DurationSeconds,
		DurationHuman:   formatDuration(session.DurationSeconds),
	}
}

func toSessionResponses(sessions []Session, location *time.Location) []SessionResponse {
	response := make([]SessionResponse, 0, len(sessions))

	for _, session := range sessions {
		response = append(response, toSessionResponse(session, location))
	}

	return response
}

func getSessionByIDHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, location *time.Location) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "session id is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	session, err := getSessionByIDFromDB(ctx, db, id)
	if err != nil {
		switch {
		case errors.Is(err, ErrSessionNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": err.Error(),
			})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "failed to get session",
			})
		}
		return
	}

	writeJSON(w, http.StatusOK, toSessionResponse(session, location))
}
