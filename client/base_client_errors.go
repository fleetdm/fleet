package client

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

var (
	ErrUnauthenticated       = errors.New("unauthenticated, or invalid token")
	ErrPasswordResetRequired = errors.New("Password reset required. Please sign into the Fleet UI to update your password, then log in again with: fleetctl login.")
	ErrMissingLicense        = errors.New("missing or invalid license")
	// ErrEndUserAuthRequired is returned when an action (such as enrolling a device)
	// requires end user authentication
	ErrEndUserAuthRequired = errors.New("end user authentication required")
)

type SetupAlreadyErr interface {
	SetupAlready() bool
	Error() string
}

type SetupAlreadyError struct{}

func (e SetupAlreadyError) Error() string {
	return "Fleet has already been setup"
}

func (e SetupAlreadyError) SetupAlready() bool {
	return true
}

type NotSetupErr interface {
	NotSetup() bool
	Error() string
}

type NotSetupError struct{}

func (e NotSetupError) Error() string {
	return "The Fleet instance is not set up yet"
}

func (e NotSetupError) NotSetup() bool {
	return true
}

// NotFoundErrIface is the interface for not-found errors.
// TODO: we have a similar but different interface in the fleet package,
// fleet.NotFoundError - at the very least, the NotFound method should be the
// same in both (the other is currently IsNotFound), and ideally we'd just have
// one of those interfaces.
type NotFoundErrIface interface {
	NotFound() bool
	Error() string
}

type NotFoundErr struct {
	Msg string

	fleet.ErrorWithUUID
}

func (e *NotFoundErr) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	return "The resource was not found"
}

func (e *NotFoundErr) NotFound() bool {
	return true
}

// Implement Is so that errors.Is(err, sql.ErrNoRows) returns true for an
// error of type *NotFoundErr, without having to wrap sql.ErrNoRows
// explicitly. It also matches other *NotFoundErr targets so that pointer-based
// comparison works (pointers to distinct structs are never == even if their
// contents are identical).
func (e *NotFoundErr) Is(other error) bool {
	if other == sql.ErrNoRows {
		return true
	}
	_, ok := other.(*NotFoundErr)
	return ok
}

// IsNotFoundErr reports whether err's chain contains a *NotFoundErr.
func IsNotFoundErr(err error) bool {
	var nfe *NotFoundErr
	return errors.As(err, &nfe)
}

type ConflictErr interface {
	Conflict() bool
	Error() string
}

type ConflictError struct {
	Msg string
}

func (e ConflictError) Error() string {
	return e.Msg
}

func (e ConflictError) Conflict() bool {
	return true
}

type serverError struct {
	Message string `json:"message"`
	Errors  []struct {
		Name   string `json:"name"`
		Reason string `json:"reason"`
	} `json:"errors"`
}

// TruncateAndDetectHTML truncates a response body to a reasonable length and
// detects if it's HTML content. Returns the truncated body and whether it's HTML.
func TruncateAndDetectHTML(body []byte, maxLen int) (truncated []byte, isHTML bool) {
	if len(body) > maxLen {
		// Use append which is more idiomatic and efficient
		truncated = append([]byte(nil), body[:maxLen]...)
		truncated = append(truncated, "..."...)
	} else {
		// For small bodies, we can return the slice directly since it will be
		// converted to string soon anyway and won't hold a large underlying array
		truncated = body
	}
	lowerPrefix := bytes.ToLower(truncated)
	isHTML = bytes.Contains(lowerPrefix, []byte("<html")) || bytes.Contains(lowerPrefix, []byte("<!doctype"))

	// Return truncated byte slice
	return truncated, isHTML
}

func ExtractServerErrorText(body io.Reader) string {
	_, reason := ExtractServerErrorNameReason(body)
	return reason
}

func ExtractServerErrorNameReason(body io.Reader) (string, string) {
	// Read the body first so we can try to parse it as JSON and fallback to text if needed
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return "", "failed to read response body"
	}

	// Try to parse as JSON first
	var serverErr serverError
	if err := json.Unmarshal(bodyBytes, &serverErr); err != nil {
		// If it's not JSON, it might be HTML or plain text error from a proxy/load balancer
		const maxLen = 200
		truncatedBytes, isHTML := TruncateAndDetectHTML(bodyBytes, maxLen)

		if isHTML {
			// Generic HTML response
			return "", fmt.Sprintf("server returned HTML instead of JSON response, body: %s", truncatedBytes)
		}

		// Return cleaned up text for non-HTML responses
		truncated := strings.TrimSpace(string(truncatedBytes))
		if truncated == "" {
			return "", "empty response body"
		}
		return "", truncated
	}

	errName := ""
	errReason := serverErr.Message
	if len(serverErr.Errors) > 0 {
		errReason += ": " + serverErr.Errors[0].Reason
		errName = serverErr.Errors[0].Name
	}

	return errName, errReason
}

func ExtractServerErrorNameReasons(body io.Reader) ([]string, []string) {
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

type StatusCodeErr struct {
	Code int
	Body string
}

func (e *StatusCodeErr) Error() string {
	return fmt.Sprintf("%d %s", e.Code, e.Body)
}

func (e *StatusCodeErr) StatusCode() int {
	return e.Code
}
