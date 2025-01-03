package service

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

var (
	ErrUnauthenticated = errors.New("unauthenticated, or invalid token")
	ErrMissingLicense  = errors.New("missing or invalid license")
)

type SetupAlreadyErr interface {
	SetupAlready() bool
	Error() string
}

type setupAlreadyErr struct{}

func (e setupAlreadyErr) Error() string {
	return "Fleet has already been setup"
}

func (e setupAlreadyErr) SetupAlready() bool {
	return true
}

type NotSetupErr interface {
	NotSetup() bool
	Error() string
}

type notSetupErr struct{}

func (e notSetupErr) Error() string {
	return "The Fleet instance is not set up yet"
}

func (e notSetupErr) NotSetup() bool {
	return true
}

// TODO: we have a similar but different interface in the fleet package,
// fleet.NotFoundError - at the very least, the NotFound method should be the
// same in both (the other is currently IsNotFound), and ideally we'd just have
// one of those interfaces.
type NotFoundErr interface {
	NotFound() bool
	Error() string
}

type notFoundErr struct {
	msg string

	fleet.ErrorWithUUID
}

func (e notFoundErr) Error() string {
	if e.msg != "" {
		return e.msg
	}
	return "The resource was not found"
}

func (e notFoundErr) NotFound() bool {
	return true
}

// Implement Is so that errors.Is(err, sql.ErrNoRows) returns true for an
// error of type *notFoundError, without having to wrap sql.ErrNoRows
// explicitly.
func (e notFoundErr) Is(other error) bool {
	return other == sql.ErrNoRows
}

type ConflictErr interface {
	Conflict() bool
	Error() string
}

type conflictErr struct {
	msg string
}

func (e conflictErr) Error() string {
	return e.msg
}

func (e conflictErr) Conflict() bool {
	return true
}

type serverError struct {
	Message string `json:"message"`
	Errors  []struct {
		Name   string `json:"name"`
		Reason string `json:"reason"`
	} `json:"errors"`
}

func extractServerErrorText(body io.Reader) string {
	_, reason := extractServerErrorNameReason(body)
	return reason
}

func extractServerErrorNameReason(body io.Reader) (string, string) {
	var serverErr serverError
	if err := json.NewDecoder(body).Decode(&serverErr); err != nil {
		return "", "unknown"
	}

	errName := ""
	errReason := serverErr.Message
	if len(serverErr.Errors) > 0 {
		errReason += ": " + serverErr.Errors[0].Reason
		errName = serverErr.Errors[0].Name
	}

	return errName, errReason
}

func extractServerErrorNameReasons(body io.Reader) ([]string, []string) {
	var serverErr serverError
	if err := json.NewDecoder(body).Decode(&serverErr); err != nil {
		return []string{""}, []string{"unknown"}
	}

	var errName []string
	var errReason []string
	for _, err := range serverErr.Errors {
		errName = append(errName, err.Name)
		errReason = append(errReason, err.Reason)
	}

	return errName, errReason
}

type statusCodeErr struct {
	code int
	body string
}

func (e *statusCodeErr) Error() string {
	return fmt.Sprintf("%d %s", e.code, e.body)
}

func (e *statusCodeErr) StatusCode() int {
	return e.code
}
