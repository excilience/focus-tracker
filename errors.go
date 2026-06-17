package main

import "errors"

var (
	ErrSessionNotFound       = errors.New("session not found")
	ErrNoActiveSession       = errors.New("no active session")
	ErrSessionAlreadyActive  = errors.New("session already active")
	ErrSessionAlreadyPaused  = errors.New("session is alredy paused")
	ErrSessionAlreadyRunning = errors.New("session is already running")
)
