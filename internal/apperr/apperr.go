// Package apperr defines application errors that carry an HTTP status and a
// stable machine-readable code.
package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

// Error is a user-facing application error.
type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

// New constructs an application error.
func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

// BadRequest reports invalid input.
func BadRequest(message string) *Error {
	return New(http.StatusBadRequest, "BAD_REQUEST", message)
}

// Unauthorized reports a missing or expired session.
func Unauthorized(message string) *Error {
	return New(http.StatusUnauthorized, "UNAUTHORIZED", message)
}

// Forbidden reports an authenticated user without enough permission.
func Forbidden(message string) *Error {
	return New(http.StatusForbidden, "FORBIDDEN", message)
}

// NotFound reports a missing resource.
func NotFound(message string) *Error {
	return New(http.StatusNotFound, "NOT_FOUND", message)
}

// Conflict reports a uniqueness or state conflict.
func Conflict(message string) *Error {
	return New(http.StatusConflict, "CONFLICT", message)
}

// TooLarge reports an oversized request or attachment.
func TooLarge(message string) *Error {
	return New(http.StatusRequestEntityTooLarge, "CONTENT_TOO_LARGE", message)
}

// Internal reports an unexpected failure without leaking details.
func Internal() *Error {
	return New(http.StatusInternalServerError, "INTERNAL", "服务器内部错误")
}

// From converts any error into an application error.
func From(err error) *Error {
	var app *Error
	if errors.As(err, &app) {
		return app
	}
	return Internal()
}

// Wrap adds context while preserving the original message.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}
