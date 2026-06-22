package main

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"
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
		ID:              "test-session-1",
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

	_, err := getSessionByIDFromDB(ctx, db, "missing-session-id")
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
		ID:              "test-session-delete",
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
		ID:              "test-session-update",
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
			ID:              "before-period",
			Start:           time.Date(2026, time.June, 14, 8, 0, 0, 0, location),
			End:             time.Date(2026, time.June, 14, 9, 0, 0, 0, location),
			DurationSeconds: int((1 * time.Hour).Seconds()),
		},
		{
			ID:              "overlaps-start",
			Start:           time.Date(2026, time.June, 14, 9, 30, 0, 0, location),
			End:             time.Date(2026, time.June, 14, 10, 30, 0, 0, location),
			DurationSeconds: int((1 * time.Hour).Seconds()),
		},
		{
			ID:              "inside-period",
			Start:           time.Date(2026, time.June, 14, 11, 0, 0, 0, location),
			End:             time.Date(2026, time.June, 14, 12, 0, 0, 0, location),
			DurationSeconds: int((1 * time.Hour).Seconds()),
		},
		{
			ID:              "overlaps-end",
			Start:           time.Date(2026, time.June, 14, 12, 30, 0, 0, location),
			End:             time.Date(2026, time.June, 14, 13, 30, 0, 0, location),
			DurationSeconds: int((1 * time.Hour).Seconds()),
		},
		{
			ID:              "after-period",
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
		"overlaps-start",
		"inside-period",
		"overlaps-end",
	}

	if !reflect.DeepEqual(gotIDs, expectedIDs) {
		t.Errorf("IDs = %v; expected %v", gotIDs, expectedIDs)
	}
}
