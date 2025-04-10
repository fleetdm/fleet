package endpoint_utils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
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
	fleet.ErrorWithUUID
}

func (e *notFoundError) Error() string {
	return "not found"
}

func (e *notFoundError) IsNotFound() bool {
	return true
}

func TestHandlesErrorsCode(t *testing.T) {
	errorTests := []struct {
		name string
		err  error
		code int
	}{
		{
			"validation",
			fleet.NewInvalidArgumentError("a", "b"),
			http.StatusUnprocessableEntity,
		},
		{
			"permission",
			fleet.NewPermissionError("a"),
			http.StatusForbidden,
		},
		{
			"foreign key",
			foreignKeyError{},
			http.StatusUnprocessableEntity,
		},
		{
			"mail error",
			MailError{},
			http.StatusInternalServerError,
		},
		{
			"osquery error - invalid node",
			&OsqueryError{nodeInvalid: true},
			http.StatusUnauthorized,
		},
		{
			"osquery error - valid node",
			&OsqueryError{},
			http.StatusInternalServerError,
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
			fleet.NewAuthFailedError(""),
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
			EncodeError(context.Background(), tt.err, recorder)
			assert.Equal(t, recorder.Code, tt.code)
		})
	}
}
