package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"
)

func runApiServer(fs *FocusService, db *sql.DB) error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /sessions", func(w http.ResponseWriter, r *http.Request) {
		getSessionsHandler(w, r, fs, db, fs.location)
	})
	mux.HandleFunc("GET /sessions/{id}", func(w http.ResponseWriter, r *http.Request) { getSessionByIDHandler(w, r, db, fs.location) })
	mux.HandleFunc("GET /sessions/active", getActiveSessionHandler)
	mux.HandleFunc("GET /stats", func(w http.ResponseWriter, r *http.Request) {
		getStatsHandler(w, r, fs, db)
	})
	mux.HandleFunc("GET /goal", func(w http.ResponseWriter, r *http.Request) {
		getGoalHandler(w, r, fs, db)
	})

	mux.HandleFunc("POST /sessions/start", startSessionHandler)
	mux.HandleFunc("POST /sessions/pause", pauseSessionHandler)
	mux.HandleFunc("POST /sessions/resume", resumeSessionHandler)
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

func startSessionHandler(w http.ResponseWriter, _ *http.Request) {
	if err := startSession(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"message": "focus session started",
	})
}

func stopSessionHandler(w http.ResponseWriter, _ *http.Request, db *sql.DB, location *time.Location) {
	result, err := stopSession(db)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
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

func pauseSessionHandler(w http.ResponseWriter, _ *http.Request) {
	if err := pauseSession(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "focus session paused",
	})
}

func resumeSessionHandler(w http.ResponseWriter, _ *http.Request) {
	if err := resumeSession(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "focus session resumed",
	})
}

func getActiveSessionHandler(w http.ResponseWriter, _ *http.Request) {
	activeSession, err := loadActiveSession()
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error": err.Error(),
		})
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

func getSessionsHandler(w http.ResponseWriter, r *http.Request, fs *FocusService, db *sql.DB, location *time.Location) {

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	sessions, err := loadSessionsFromDB(ctx, db)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to load sessions",
		})
		return
	}

	fromText := r.URL.Query().Get("from")
	toText := r.URL.Query().Get("to")

	if fromText == "" && toText == "" {
		writeJSON(w, http.StatusOK, toSessionResponses(sessions, location))
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
	to := fs.StartOfFocusDayFromDate(toDate)

	if !to.After(from) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "to date must be after from date",
		})
		return
	}

	filtered := filterSessionsByPeriod(sessions, from, to)

	writeJSON(w, http.StatusOK, toSessionResponses(filtered, location))
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

	updatedSession, err := updateSessionDurationInDB(ctx, db, id, duration)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
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

	deletedSession, err := deleteSessionFromDB(ctx, db, id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
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

func getSessionByID(id string) (Session, error) {
	sessions, err := loadSessions()
	if err != nil {
		return Session{}, err
	}

	for _, session := range sessions {
		if session.ID == id {
			return session, nil
		}
	}

	return Session{}, fmt.Errorf("session not found")

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
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, toSessionResponse(session, location))
}
