package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
)

func runApiServer(fs *FocusService, db *sql.DB) error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /sessions", func(w http.ResponseWriter, r *http.Request) {
		getSessionsHandler(w, r, fs, db)
	})
	mux.HandleFunc("GET /sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		getSessionByIDHandler(w, r, db, fs)
	})
	mux.HandleFunc("GET /sessions/active", func(w http.ResponseWriter, r *http.Request) {
		getActiveSessionHandler(w, r, db)
	})
	mux.HandleFunc("GET /stats", func(w http.ResponseWriter, r *http.Request) {
		getStatsHandler(w, r, fs, db)
	})
	mux.HandleFunc("GET /goal", func(w http.ResponseWriter, r *http.Request) {
		getGoalHandler(w, r, fs, db)
	})
	mux.HandleFunc("GET /settings", func(w http.ResponseWriter, r *http.Request) {
		getSettingsHandler(w, r, fs)
	})
	mux.HandleFunc("GET /activities", func(w http.ResponseWriter, r *http.Request) {
		getActivitiesHandler(w, r, db)
	})
	mux.HandleFunc("GET /activities/stats", func(w http.ResponseWriter, r *http.Request) {
		getActivityStatsHandler(w, r, db)
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
		stopSessionHandler(w, r, db, fs)
	})
	mux.HandleFunc("POST /activities", func(w http.ResponseWriter, r *http.Request) {
		createActivityHandler(w, r, db)
	})

	mux.HandleFunc("PATCH /sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		updateSessionHandler(w, r, db, fs)
	})
	mux.HandleFunc("PATCH /goal", func(w http.ResponseWriter, r *http.Request) {
		updateGoalHandler(w, r, fs, db)
	})
	mux.HandleFunc("PATCH /settings", func(w http.ResponseWriter, r *http.Request) {
		updateSettingsHandler(w, r, fs, db)
	})
	mux.HandleFunc("PATCH /activities/{id}", func(w http.ResponseWriter, r *http.Request) {
		updateActivityHandler(w, r, db)
	})

	mux.HandleFunc("DELETE /sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		deleteSessionHandler(w, r, db, fs)
	})

	addr := ":8080"

	handler := devCORSMiddleware(mux)
	handler = loggingMiddleware(handler)

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}
	serverErr := make(chan error, 1)

	go func() {
		fmt.Println("API server started on http://localhost" + addr)

		if err := server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}

		serverErr <- nil
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)

	select {
	case <-stop:
		fmt.Println("\nShutting down API server...")

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}

		return nil

	case err := <-serverErr:
		return err
	}
}

func devCORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin == "http://localhost:5173" || origin == "http://127.0.0.1:5173" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		recorder := &statusRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(recorder, r)

		fmt.Printf(
			"%s %s %d %s\n",
			r.Method,
			r.URL.RequestURI(),
			recorder.status,
			time.Since(start),
		)
	})
}

func startSessionHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var request startSessionRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
		writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, "invalid JSON body")
		return
	}

	activityID := request.ActivityID

	if activityID != nil {
		if _, err := uuid.Parse(*activityID); err != nil {
			writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, "invalid activity_id")
			return
		}
	}

	if activityID != nil {
		activity, err := getActivityByID(ctx, db, *activityID)
		if err != nil {
			if errors.Is(err, ErrActivityNotFound) {
				writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, "activity not found")
				return
			}

			writeAPIError(w, http.StatusInternalServerError, ErrorCodeInternalError, "failed to validate activity")
			return
		}

		if activity.IsArchived {
			writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, "activity is archived")
			return
		}
	}

	if err := startSession(ctx, db, activityID); err != nil {
		writeDomainError(w, err, "failed to start session")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"message": "focus session started",
	})
}

func stopSessionHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, fs *FocusService) {

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	result, err := stopSession(ctx, db)
	if err != nil {
		writeDomainError(w, err, "failed to stop session")
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
		"session": toSessionResponse(result.Session, fs),
	})
}

func pauseSessionHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := pauseSession(ctx, db); err != nil {
		writeDomainError(w, err, "failed to pause session")
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
		writeDomainError(w, err, "failed to resume session")
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
		writeDomainError(w, err, "failed to load active session")
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
		Activity:       toActivitySummary(activeSession.ActivityID, activeSession.ActivityTitle),
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
		writeAPIError(w, http.StatusInternalServerError, ErrorCodeInternalError, "failed to load sessions")
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "day"
	}

	stats, err := fs.StatsForCommand(sessions, period, time.Now())
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, err.Error())
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
			writeAPIError(w, http.StatusInternalServerError, ErrorCodeInternalError, "failed to load sessions")
			return
		}
		writeJSON(w, http.StatusOK, toSessionResponses(sessions, fs))
		return
	}

	if fromText == "" || toText == "" {
		writeAPIErrorWithDetails(w, http.StatusBadRequest, ErrorCodeInvalidRequest, "both 'from' and 'to' are required",
			map[string]any{
				"example": "/sessions?from=YYYY-MM-DD&to=YYYY-MM-DD",
			},
		)
		return
	}

	fromDate, err := time.ParseInLocation(dateLayout, fromText, fs.location)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, "invalid from date, expected YYYY-MM-DD")
		return
	}

	toDate, err := time.ParseInLocation(dateLayout, toText, fs.location)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, "invalid to date, expected YYYY-MM-DD")
		return
	}

	from := fs.StartOfFocusDayFromDate(fromDate)
	to := fs.StartOfFocusDayFromDate(toDate).AddDate(0, 0, 1)

	if !to.After(from) {
		writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, "'to' date must be the same as or later than 'from' date")
		return
	}

	sessions, err := loadSessionsByPeriodFromDB(ctx, db, from, to)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, ErrorCodeInternalError, "failed to load sessions by period")
		return
	}

	writeJSON(w, http.StatusOK, toSessionResponses(sessions, fs))
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
		writeAPIError(w, http.StatusInternalServerError, ErrorCodeInternalError, "failed to load sessions")
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

func updateGoalHandler(w http.ResponseWriter, r *http.Request, fs *FocusService, db *sql.DB) {
	var request updateGoalRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, "invalid JSON body")
		return
	}

	if request.DailyGoal == "" {
		writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, "daily_goal is required")
		return
	}

	dailyGoal, err := time.ParseDuration(request.DailyGoal)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, "invalid daily_goal format")
		return
	}

	if dailyGoal <= 0 {
		writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, "daily_goal must be higher than zero")
		return
	}

	dailyGoalMinutes := int(dailyGoal.Minutes())

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := updateDailyGoalMinutes(ctx, db, dailyGoalMinutes); err != nil {
		writeAPIError(w, http.StatusInternalServerError, ErrorCodeInternalError, "failed to update goal")
		return
	}

	fs.settings.DailyGoal = dailyGoalMinutes

	sessions, err := loadSessionsFromDB(ctx, db)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, ErrorCodeInternalError, "failed to load sessions")
		return
	}

	progress := fs.DailyProgress(sessions)

	writeJSON(w, http.StatusOK, toGoalResponse(fs, progress, time.Now()))

}

func updateSessionHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, fs *FocusService) {
	id := r.PathValue("id")
	if id == "" {
		writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, "session id is required")
		return
	}

	var request updateSessionRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, "invalid JSON body")
		return
	}

	if request.Duration == "" {
		writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, "duration is required")
		return
	}

	duration, err := time.ParseDuration(request.Duration)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, "invalid duration format")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	updatedSession, err := updateSessionDuration(ctx, db, id, duration)
	if err != nil {
		writeDomainError(w, err, "failed to update session")
		return
	}

	writeJSON(w, http.StatusOK, toSessionResponse(updatedSession, fs))
}

func deleteSessionHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, fs *FocusService) {
	id := r.PathValue("id")
	if id == "" {
		writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, "session id is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	deletedSession, err := deleteSession(ctx, db, id)
	if err != nil {
		writeDomainError(w, err, "failed to delete session")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "session deleted",
		"session": toSessionResponse(deletedSession, fs),
	})
}

func toSessionResponse(session Session, fs *FocusService) SessionResponse {
	start := session.Start.In(fs.location)
	end := session.End.In(fs.location)
	focusDay := fs.StartOfFocusDay(start)

	return SessionResponse{
		ID:              session.ID,
		Start:           timeFormat(start),
		End:             timeFormat(end),
		FocusDay:        focusDay.Format(dateLayout),
		DurationSeconds: session.DurationSeconds,
		DurationHuman:   formatDuration(session.DurationSeconds),
		Activity:        toActivitySummary(session.ActivityID, session.ActivityTitle),
	}
}

func toSessionResponses(sessions []Session, fs *FocusService) []SessionResponse {
	response := make([]SessionResponse, 0, len(sessions))

	for _, session := range sessions {
		response = append(response, toSessionResponse(session, fs))
	}

	return response
}

func toActivitySummary(id *string, title *string) *ActivitySummaryResponse {
	if id == nil || title == nil {
		return nil
	}

	return &ActivitySummaryResponse{
		ID:    *id,
		Title: *title,
	}
}

func toActivityStatsResponses(stats []ActivityStat) []ActivityStatsResponse {
	response := make([]ActivityStatsResponse, 0, len(stats))

	for _, stat := range stats {
		response = append(response, ActivityStatsResponse{
			Activity:        toActivitySummary(stat.ActivityID, stat.ActivityTitle),
			DurationSeconds: stat.DurationSeconds,
			DurationHuman:   formatDuration(stat.DurationSeconds),
		})
	}

	return response
}

func getActivityStatsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	sessions, err := loadSessionsFromDB(ctx, db)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, ErrorCodeInternalError, "failed to load sessions")
		return
	}

	stats := calculateActivityStats(sessions)

	writeJSON(w, http.StatusOK, toActivityStatsResponses(stats))
}

func getSessionByIDHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, fs *FocusService) {
	id := r.PathValue("id")
	if id == "" {
		writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, "session id is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	session, err := getSessionByIDFromDB(ctx, db, id)
	if err != nil {
		writeDomainError(w, err, "failed to get session")
		return
	}

	writeJSON(w, http.StatusOK, toSessionResponse(session, fs))
}

func toSettingsResponse(fs *FocusService) SettingsResponse {
	return SettingsResponse{
		DayStartHour: fs.settings.DayStartHour,
	}
}

func getSettingsHandler(w http.ResponseWriter, _ *http.Request, fs *FocusService) {
	writeJSON(w, http.StatusOK, toSettingsResponse(fs))
}

func updateSettingsHandler(w http.ResponseWriter, r *http.Request, fs *FocusService, db *sql.DB) {
	var request updateSettingsRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, "invalid JSON body")
		return
	}

	if request.DayStartHour == nil {
		writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, "day_start_hour is required")
		return
	}

	dayStartHour := *request.DayStartHour

	if dayStartHour < 0 || dayStartHour > 23 {
		writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, "day_start_hour must be between 0 and 23")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := updateDayStartHour(ctx, db, dayStartHour); err != nil {
		writeAPIError(w, http.StatusInternalServerError, ErrorCodeInternalError, "failed to update day start hour")
		return
	}

	fs.settings.DayStartHour = dayStartHour

	writeJSON(w, http.StatusOK, toSettingsResponse(fs))
}

func toActivityResponse(activity Activity) ActivityResponse {
	return ActivityResponse{
		ID:         activity.ID,
		Title:      activity.Title,
		IsArchived: activity.IsArchived,
		CreatedAt:  timeFormat(activity.CreatedAt),
	}
}

func toActivityResponses(activities []Activity) []ActivityResponse {
	response := make([]ActivityResponse, 0, len(activities))

	for _, activity := range activities {
		response = append(response, toActivityResponse(activity))
	}

	return response
}

func getActivitiesHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	activities, err := loadActivities(ctx, db)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, ErrorCodeInternalError, "failed to load activities")
		return
	}

	writeJSON(w, http.StatusOK, toActivityResponses(activities))
}

func createActivityHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var request createActivityRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, "invalid JSON body")
		return
	}

	activity, err := createActivity(r.Context(), db, request.Title)
	if err != nil {
		if errors.Is(err, ErrActivityTitleAlreadyExists) {
			writeAPIError(w, http.StatusConflict, ErrorCodeActivityTitleAlreadyExists, err.Error())
			return
		}

		writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toActivityResponse(activity))
}

func updateActivityHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	id := r.PathValue("id")
	if strings.TrimSpace(id) == "" {
		writeAPIError(
			w,
			http.StatusBadRequest,
			ErrorCodeInvalidRequest,
			"activity id is required",
		)
		return
	}

	var request updateActivityRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, "invalid JSON body")
		return
	}

	if request.Title == nil && request.IsArchived == nil {
		writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, "nothing to update")
		return
	}

	activity, err := updateActivity(r.Context(), db, id, request.Title, request.IsArchived)
	if err != nil {
		if errors.Is(err, ErrActivityNotFound) {
			writeAPIError(w, http.StatusNotFound, ErrorCodeActivityNotFound, "activity not found")
			return
		}

		if errors.Is(err, ErrActivityTitleAlreadyExists) {
			writeAPIError(w, http.StatusConflict, ErrorCodeActivityTitleAlreadyExists, err.Error())
			return
		}

		writeAPIError(w, http.StatusBadRequest, ErrorCodeInvalidRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toActivityResponse(activity))
}
