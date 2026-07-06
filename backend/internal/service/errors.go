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

// SafeError returns the HTTP status and a client-safe message for err. Known
// UserError/AuthError values carry curated, safe messages and are preserved;
// any other error (an unexpected internal failure) is reduced to a generic
// message so infrastructure details such as decrypt/DB/upstream text are not
// leaked to clients.
func SafeError(err error) (int, string) {
	if err == nil {
		return http.StatusOK, ""
	}
	var ue *UserError
	if errors.As(err, &ue) {
		return ue.Status, ue.Msg
	}
	var ae *AuthError
	if errors.As(err, &ae) {
		if ae.Status >= 500 {
			return ae.Status, "internal server error"
		}
		return ae.Status, ae.Msg
	}
	return http.StatusInternalServerError, "internal server error"
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
