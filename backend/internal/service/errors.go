package service

import (
	"errors"
	"net/http"
)

// AuthError carries an HTTP status for auth service failures.
type AuthError struct {
	Status int
	Msg    string
}

func (e *AuthError) Error() string { return e.Msg }

func authUnauthorized(msg string) error { return &AuthError{Status: http.StatusUnauthorized, Msg: msg} }
func authInternal(msg string) error {
	return &AuthError{Status: http.StatusInternalServerError, Msg: msg}
}

// HTTPStatus extracts the status code from service errors.
func HTTPStatus(err error) int {
	var ue *UserError
	if errors.As(err, &ue) {
		return ue.Status
	}
	var ae *AuthError
	if errors.As(err, &ae) {
		return ae.Status
	}
	return http.StatusInternalServerError
}

// UserError carries an HTTP status so the handler can map service failures to
// the right response code without leaking implementation details.
type UserError struct {
	Status int
	Msg    string
}

func (e *UserError) Error() string { return e.Msg }

func badRequest(msg string) error { return &UserError{Status: http.StatusBadRequest, Msg: msg} }
func forbidden(msg string) error  { return &UserError{Status: http.StatusForbidden, Msg: msg} }
func notFound(msg string) error   { return &UserError{Status: http.StatusNotFound, Msg: msg} }
func conflict(msg string) error   { return &UserError{Status: http.StatusConflict, Msg: msg} }
