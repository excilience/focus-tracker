package main

import (
	"errors"
	"net/http"
)

var (
	ErrSessionNotFound       = errors.New("session not found")
	ErrNoActiveSession       = errors.New("no active session")
	ErrSessionAlreadyActive  = errors.New("session already active")
	ErrSessionAlreadyPaused  = errors.New("session is already paused")
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
	Details map[string]any `json:"details,omitempty"`
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

func writeDomainError(w http.ResponseWriter, err error, defaultMessage string) {
	switch {
	case errors.Is(err, ErrSessionNotFound):
		writeAPIError(w, http.StatusNotFound, ErrorCodeSessionNotFound, err.Error())

	case errors.Is(err, ErrNoActiveSession):
		writeAPIError(w, http.StatusNotFound, ErrorCodeNoActiveSession, err.Error())

	case errors.Is(err, ErrSessionAlreadyActive):
		writeAPIError(w, http.StatusConflict, ErrorCodeSessionAlreadyActive, err.Error())

	case errors.Is(err, ErrSessionAlreadyPaused):
		writeAPIError(w, http.StatusConflict, ErrorCodeSessionAlreadyPaused, err.Error())

	case errors.Is(err, ErrSessionAlreadyRunning):
		writeAPIError(w, http.StatusConflict, ErrorCodeSessionAlreadyRunning, err.Error())

	default:
		writeAPIError(w, http.StatusInternalServerError, ErrorCodeInternalError, defaultMessage)

	}
}
