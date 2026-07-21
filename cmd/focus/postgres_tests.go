package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

const (
	testSessionID1             = "11111111-1111-1111-1111-111111111111"
	testSessionIDDelete        = "22222222-2222-2222-2222-222222222222"
	testSessionIDUpdate        = "33333333-3333-3333-3333-333333333333"
	testSessionIDBeforePeriod  = "44444444-4444-4444-4444-444444444444"
	testSessionIDOverlapsStart = "55555555-5555-5555-5555-555555555555"
	testSessionIDInsidePeriod  = "66666666-6666-6666-6666-666666666666"
	testSessionIDOverlapsEnd   = "77777777-7777-7777-7777-777777777777"
	testSessionIDAfterPeriod   = "88888888-8888-8888-8888-888888888888"
	testMissingSessionID       = "99999999-9999-9999-9999-999999999999"
)

func cleanTestDB(t *testing.T, db *sql.DB) {
	t.Helper()

	ctx := context.Background()

	_, err := db.ExecContext(ctx, `DELETE FROM active_sessions`)
	if err != nil {
		t.Fatalf("clean active_sessions: %v", err)
	}

	_, err = db.ExecContext(ctx, `DELETE FROM sessions`)
	if err != nil {
		t.Fatalf("clean sessions: %v", err)
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}

	if dsn == "" {
		t.Skip("TEST_DATABASE_URL or DATABASE_URL is not set")
	}

	db, err := openDatabase(dsn)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})
	return db

}

func TestCreateSessionInDBAndGetByID(t *testing.T) {
	db := openTestDB(t)
	cleanTestDB(t, db)

	ctx := context.Background()
	location := time.FixedZone("UTC+3", 3*60*60)

	expected := Session{
		ID:              testSessionID1,
		Start:           time.Date(2026, time.June, 14, 10, 0, 0, 0, location),
		End:             time.Date(2026, time.June, 14, 11, 0, 0, 0, location),
		DurationSeconds: int((1 * time.Hour).Seconds()),
	}

	err := createSession(ctx, db, expected)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	got, err := getSessionByIDFromDB(ctx, db, expected.ID)
	if err != nil {
		t.Fatalf("get session by id: %v", err)
	}

	if got.ID != expected.ID {
		t.Errorf("ID = %q; expected %q", got.ID, expected.ID)
	}

	if !got.Start.Equal(expected.Start) {
		t.Errorf("Start = %s; expected %s", got.Start, expected.Start)
	}

	if !got.End.Equal(expected.End) {
		t.Errorf("End = %s; expected %s", got.End, expected.End)
	}

	if got.DurationSeconds != expected.DurationSeconds {
		t.Errorf(
			"DurationSeconds = %d; expected %d",
			got.DurationSeconds,
			expected.DurationSeconds,
		)
	}
}

func TestGetSessionByIDFromDBNotFound(t *testing.T) {
	db := openTestDB(t)
	cleanTestDB(t, db)

	ctx := context.Background()

	_, err := getSessionByIDFromDB(ctx, db, testMissingSessionID)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound; got %v", err)
	}
}

func TestDeleteSession(t *testing.T) {
	db := openTestDB(t)
	cleanTestDB(t, db)

	ctx := context.Background()
	location := time.FixedZone("UTC+3", 3*60*60)

	session := Session{
		ID:              testSessionIDDelete,
		Start:           time.Date(2026, time.June, 14, 10, 0, 0, 0, location),
		End:             time.Date(2026, time.June, 14, 11, 0, 0, 0, location),
		DurationSeconds: int((1 * time.Hour).Seconds()),
	}

	err := createSession(ctx, db, session)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	deleteSession, err := deleteSession(ctx, db, session.ID)
	if err != nil {
		t.Fatalf("delete session: %v", err)
	}

	if deleteSession.ID != session.ID {
		t.Errorf("deleted ID = %q; expected %q", deleteSession.ID, session.ID)
	}

	_, err = getSessionByIDFromDB(ctx, db, session.ID)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound after delete; got %v", err)
	}
}

func TestUpdateSessionDuration(t *testing.T) {

	db := openTestDB(t)
	cleanTestDB(t, db)

	ctx := context.Background()
	location := time.FixedZone("UTC+3", 3*60*60)

	session := Session{
		ID:              testSessionIDUpdate,
		Start:           time.Date(2026, time.June, 14, 10, 0, 0, 0, location),
		End:             time.Date(2026, time.June, 14, 11, 0, 0, 0, location),
		DurationSeconds: int((1 * time.Hour).Seconds()),
	}

	err := createSession(ctx, db, session)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	newDuration := 45 * time.Minute

	updatedSession, err := updateSessionDuration(ctx, db, session.ID, newDuration)
	if err != nil {
		t.Fatalf("update session duration: %v", err)
	}

	expectedSeconds := int(newDuration.Seconds())

	if updatedSession.ID != session.ID {
		t.Errorf("ID = %q; expected %q", updatedSession.ID, session.ID)
	}

	if updatedSession.DurationSeconds != expectedSeconds {
		t.Errorf("DurationSeconds = %d; expecteds %d", updatedSession.DurationSeconds, expectedSeconds)
	}

	got, err := getSessionByIDFromDB(ctx, db, session.ID)
	if err != nil {
		t.Fatalf("get session by id: %v", err)
	}

	if got.DurationSeconds != expectedSeconds {
		t.Errorf("stored DurationSeconds = %d; expected %d", got.DurationSeconds, expectedSeconds)
	}
}

func TestLoadSessionsByPeriodFromDB(t *testing.T) {
	db := openTestDB(t)
	cleanTestDB(t, db)

	ctx := context.Background()
	location := time.FixedZone("UTC+3", 3*60*60)

	sessions := []Session{
		{
			ID:              testSessionIDBeforePeriod,
			Start:           time.Date(2026, time.June, 14, 8, 0, 0, 0, location),
			End:             time.Date(2026, time.June, 14, 9, 0, 0, 0, location),
			DurationSeconds: int((1 * time.Hour).Seconds()),
		},
		{
			ID:              testSessionIDOverlapsStart,
			Start:           time.Date(2026, time.June, 14, 9, 30, 0, 0, location),
			End:             time.Date(2026, time.June, 14, 10, 30, 0, 0, location),
			DurationSeconds: int((1 * time.Hour).Seconds()),
		},
		{
			ID:              testSessionIDInsidePeriod,
			Start:           time.Date(2026, time.June, 14, 11, 0, 0, 0, location),
			End:             time.Date(2026, time.June, 14, 12, 0, 0, 0, location),
			DurationSeconds: int((1 * time.Hour).Seconds()),
		},
		{
			ID:              testSessionIDOverlapsEnd,
			Start:           time.Date(2026, time.June, 14, 12, 30, 0, 0, location),
			End:             time.Date(2026, time.June, 14, 13, 30, 0, 0, location),
			DurationSeconds: int((1 * time.Hour).Seconds()),
		},
		{
			ID:              testSessionIDAfterPeriod,
			Start:           time.Date(2026, time.June, 14, 14, 0, 0, 0, location),
			End:             time.Date(2026, time.June, 14, 15, 0, 0, 0, location),
			DurationSeconds: int((1 * time.Hour).Seconds()),
		},
	}

	for _, session := range sessions {
		err := createSession(ctx, db, session)
		if err != nil {
			t.Fatalf("create session %q: %v", session.ID, err)
		}
	}

	from := time.Date(2026, time.June, 14, 10, 0, 0, 0, location)
	to := time.Date(2026, time.June, 14, 13, 0, 0, 0, location)

	got, err := loadSessionsByPeriodFromDB(ctx, db, from, to)
	if err != nil {
		t.Fatalf("load sessions by period: %v", err)
	}

	gotIDs := make([]string, 0, len(got))
	for _, session := range got {
		gotIDs = append(gotIDs, session.ID)
	}

	expectedIDs := []string{
		testSessionIDOverlapsStart,
		testSessionIDInsidePeriod,
		testSessionIDOverlapsEnd,
	}

	if !reflect.DeepEqual(gotIDs, expectedIDs) {
		t.Errorf("IDs = %v; expected %v", gotIDs, expectedIDs)
	}
}

func TestUpdateSessionHandlerInvalidDurationResponse(t *testing.T) {
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
		"/sessions/"+testMissingSessionID,
		body,
	)
	req.SetPathValue("id", testMissingSessionID)

	rr := httptest.NewRecorder()

	updateSessionHandler(
		rr,
		req,
		nil,
		fs,
	)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf(
			"status = %d; expected %d",
			rr.Code,
			http.StatusBadRequest,
		)
	}

	var response APIErrorResponse

	err := json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Error.Code != ErrorCodeInvalidRequest {
		t.Errorf(
			"error code = %q; expected %q",
			response.Error.Code,
			ErrorCodeInvalidRequest,
		)
	}

	if response.Error.Message != "invalid duration format" {
		t.Errorf(
			"error message = %q; expected %q",
			response.Error.Message,
			"invalid duration format",
		)
	}
}

func TestGetSessionsHandlerMissingDateRangeResponse(t *testing.T) {
	location := time.FixedZone("UTC+3", 3*60*60)

	fs := &FocusService{
		settings: Settings{
			DayStartHour: 4,
		},
		location: location,
	}

	req := httptest.NewRequest(http.MethodGet, "/sessions?from=2026-06-01", nil)

	rr := httptest.NewRecorder()

	getSessionsHandler(rr, req, fs, nil)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; expected %d", rr.Code, http.ErrBodyReadAfterClose)
	}

	var response APIErrorResponse

	err := json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Error.Code != ErrorCodeInvalidRequest {
		t.Errorf("error code = %q; expected %q", response.Error.Code, ErrorCodeInvalidRequest)
	}

	example, ok := response.Error.Details["example"].(string)
	if !ok {
		t.Fatalf("expected details.example to be a string")
	}

	expectedExample := "/sessions?from=YYYY-MM-DD&to=YYYY-MM-DD"

	if example != expectedExample {
		t.Errorf("details.example = %q; expected %q", example, expectedExample)
	}
}
