package main

import (
	"errors"
	"net/http"
)

var (
	ErrSessionNotFound       = errors.New("session not found")
	ErrNoActiveSession       = errors.New("no active session")
	ErrSessionAlreadyActive  = errors.New("session already active")
	ErrSessionAlreadyPaused  = errors.New("session is alredy paused")
	ErrSessionAlreadyRunning = errors.New("session is already running")
)

const (
	ErrorCodeInvalidRequest        = "invalid_request"
	ErrorCodeSessionNotFound       = "session_not_found"
	ErrorCodeNoActiveSession       = "no_active_session"
	ErrorCodeSessionAlreadyActive  = "session_already_active"
	ErrorCodeSessionAlreadyPaused  = "session_already_paused"
	ErrorCodeSessionAlreadyRunning = "session_already_running"
	ErrorCodeInternalError         = "internal_error"
)

type APIErrorResponse struct {
	Error APIError `json:"error"`
}

type APIError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
}

func writeAPIError(w http.ResponseWriter, status int, code string, message string) {
	writeAPIErrorWithDetails(w, status, code, message, nil)
}

func writeAPIErrorWithDetails(w http.ResponseWriter, status int, code string, message string, details map[string]any) {
	writeJSON(w, status, APIErrorResponse{
		Error: APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}
