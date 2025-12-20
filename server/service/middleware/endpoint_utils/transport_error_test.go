package endpoint_utils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/stretchr/testify/assert"
)

type foreignKeyError struct{}

func (foreignKeyError) IsForeignKey() bool { return true }
func (foreignKeyError) Error() string      { return "" }

type alreadyExists struct{}

func (alreadyExists) IsExists() bool { return false }
func (alreadyExists) Error() string  { return "" }

type newAndExciting struct{}

func (newAndExciting) Error() string { return "" }

type notFoundError struct {
	platform_http.ErrorWithUUID
}

func (e *notFoundError) Error() string {
	return "not found"
}

func (e *notFoundError) IsNotFound() bool {
	return true
}

// validationError is a test implementation of validationErrorInterface.
type validationError struct {
	errors []map[string]string
}

func (e validationError) Error() string {
	return "validation failed"
}

func (e validationError) Invalid() []map[string]string {
	return e.errors
}

// permissionError is a test implementation of permissionErrorInterface.
type permissionError struct {
	message string
}

func (e permissionError) Error() string {
	return e.message
}

func (e permissionError) PermissionError() []map[string]string {
	return nil
}

func TestHandlesErrorsCode(t *testing.T) {
	errorTests := []struct {
		name string
		err  error
		code int
	}{
		{
			"validation",
			validationError{errors: []map[string]string{{"name": "a", "reason": "b"}}},
			http.StatusUnprocessableEntity,
		},
		{
			"permission",
			permissionError{message: "a"},
			http.StatusForbidden,
		},
		{
			"foreign key",
			foreignKeyError{},
			http.StatusUnprocessableEntity,
		},
		{
			"data not found",
			&notFoundError{},
			http.StatusNotFound,
		},
		{
			"already exists",
			alreadyExists{},
			http.StatusConflict,
		},
		{
			"status coder",
			platform_http.NewAuthFailedError(""),
			http.StatusUnauthorized,
		},
		{
			"default",
			newAndExciting{},
			http.StatusInternalServerError,
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			EncodeError(context.Background(), tt.err, recorder, nil)
			assert.Equal(t, recorder.Code, tt.code)
		})
	}
}
