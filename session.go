package main

import (
	"fmt"
	"os"
	"time"
)

func startSession() error {
	if _, err := os.Stat(activeSessionFile); err == nil {
		return fmt.Errorf("session already started")
	} else if !os.IsNotExist(err) {
		return err
	}

	now := time.Now()

	activeSession := ActiveSession{
		Start:          now,
		LastResume:     now,
		FocusedSeconds: 0,
		IsPaused:       false,
	}
	return saveActiveSession(activeSession)
}

func pauseSession() error {
	activeSession, err := loadActiveSession()
	if err != nil {
		return err
	}

	if activeSession.IsPaused {
		return fmt.Errorf("session is already paused")
	}

	now := time.Now()

	activeDuration := now.Sub(activeSession.LastResume)
	activeSession.FocusedSeconds += int(activeDuration.Seconds())

	activeSession.IsPaused = true

	return saveActiveSession(activeSession)
}

func stopSession() (Session, error) {
	activeSession, err := loadActiveSession()
	if err != nil {
		return Session{}, err
	}

	now := time.Now()
	totalSeconds := activeSession.FocusedSeconds

	if !activeSession.IsPaused {
		activeDuration := now.Sub(activeSession.LastResume)
		totalSeconds += int(activeDuration.Seconds())
	}

	currentSession := Session{
		ID:              newSessionID(activeSession.Start),
		Start:           activeSession.Start,
		End:             now,
		DurationSeconds: totalSeconds,
	}

	if totalSeconds < 60 {
		if err := os.Remove(activeSessionFile); err != nil {
			return Session{}, err
		}

		fmt.Printf("---------------\nSession not saved, it was shorter than 1 minute.\n---------------\n")
		return currentSession, nil
	}

	if err := saveSession(currentSession); err != nil {
		return Session{}, err
	}
	if err = os.Remove(activeSessionFile); err != nil {
		return Session{}, err
	}

	return currentSession, nil
}
func resumeSession() error {
	activeSession, err := loadActiveSession()
	if err != nil {
		return err
	}

	if !activeSession.IsPaused {
		return fmt.Errorf("session is not paused")
	}

	activeSession.LastResume = time.Now()
	activeSession.IsPaused = false

	return saveActiveSession(activeSession)
}

func saveSession(newSession Session) error {
	allSessions, err := loadSessions()
	if err != nil {
		return err
	}

	allSessions = append(allSessions, newSession)

	return saveSessions(allSessions)
}

func editSessionDuration(id string, newDuration time.Duration) (Session, error) {
	if newDuration < 0 {
		return Session{}, fmt.Errorf("duration can't be less than 0")
	}

	sessions, err := loadSessions()
	if err != nil {
		return Session{}, err
	}

	for i, session := range sessions {
		if session.ID == id {
			sessions[i].DurationSeconds = int(newDuration.Seconds())

			if err := saveSessions(sessions); err != nil {
				return Session{}, err
			}

			return sessions[i], nil
		}
	}

	return Session{}, fmt.Errorf("session not found")
}

func newSessionID(start time.Time) string {
	return start.Format("20060102-150405")
}

func getTotalFocusTimeAll(sessions []Session) int {
	total := 0
	for _, session := range sessions {
		total += session.DurationSeconds
	}
	return total

}

func countFocusDays(from, to time.Time) int {
	days := 0

	for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
		days++
	}
	return days
}

func averageFocusPerDay(total time.Duration, from, to time.Time) time.Duration {
	days := countFocusDays(from, to)
	if days == 0 {
		return 0
	}
	return total / time.Duration(days)
}
