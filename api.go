package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"
)

func runApiServer(fs *FocusService) error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /sessions", getSessionsHandler)
	mux.HandleFunc("GET /stats", func(w http.ResponseWriter, r *http.Request) {
		getStatsHandler(w, r, fs)
	})
	mux.HandleFunc("GET /goal", func(w http.ResponseWriter, r *http.Request) {
		getGoalHandler(w, r, fs)
	})
	mux.HandleFunc("GET /sessions/active", getActiveSessionHandler)
	mux.HandleFunc("GET /sessions/{id}", getSessionByIDHandler)

	mux.HandleFunc("PATCH /sessions/{id}", updateSessionHandler)

	mux.HandleFunc("POST /sessions/start", startSessionHandler)
	mux.HandleFunc("POST /sessions/stop", stopSessionHandler)
	mux.HandleFunc("POST /sessions/pause", pauseSessionHandler)
	mux.HandleFunc("POST /sessions/resume", resumeSessionHandler)

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

func stopSessionHandler(w http.ResponseWriter, _ *http.Request) {
	result, err := stopSession()
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
		"message": "focus sesion stopped and saved",
		"saved":   true,
		"session": result.Session,
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

func getStatsHandler(w http.ResponseWriter, r *http.Request, fs *FocusService) {
	sessions, err := loadSessions()
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

func getSessionsHandler(w http.ResponseWriter, r *http.Request) {
	sessions, err := loadSessions()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to load sessions",
		})
		return
	}
	writeJSON(w, http.StatusOK, sessions)
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

func getGoalHandler(w http.ResponseWriter, _ *http.Request, fs *FocusService) {
	sessions, err := loadSessions()
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

func updateSessionHandler(w http.ResponseWriter, r *http.Request) {
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
	updatedSession, err := editSessionDuration(id, duration)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, toSessionResponse(updatedSession))
}

func toSessionResponse(session Session) SessionResponse {
	return SessionResponse{
		ID:              session.ID,
		Start:           timeFormat(session.Start),
		End:             timeFormat(session.End),
		DurationSeconds: session.DurationSeconds,
		DurationHuman:   formatDuration(session.DurationSeconds),
	}
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

func getSessionByIDHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "session id is required",
		})
		return
	}

	session, err := getSessionByID(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, toSessionResponse(session))
}
