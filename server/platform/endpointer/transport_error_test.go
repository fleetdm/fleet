package endpointer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			"context canceled",
			context.Canceled,
			499,
		},
		{
			"wrapped context canceled",
			fmt.Errorf("db query: %w", context.Canceled),
			499,
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

// TestSafeReason pins which error detail may reach a client. Storage errors carry
// schema identifiers and query structure and must not; everything else is either
// intentional or describes the caller's own payload, and must survive.
func TestSafeReason(t *testing.T) {
	t.Parallel()

	fkErr := &mysql.MySQLError{
		Number: 1452,
		Message: "Cannot add or update a child row: a foreign key constraint fails " +
			"(`fleet`.`queries`, CONSTRAINT `queries_ibfk_2` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`))",
	}

	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{
			"driver error is replaced",
			fkErr,
			platform_http.GenericErrorMessage,
		},
		{
			"wrapped driver error is replaced",
			fmt.Errorf("creating new Query: %w", fkErr),
			platform_http.GenericErrorMessage,
		},
		{
			"database/sql error is replaced",
			errors.New("sql: expected 2 arguments, got 3"),
			platform_http.GenericErrorMessage,
		},
		{
			// The whole point of ErrWithInternal is that Error() is already the safe
			// message and the detail lives in Internal(). Replacing it would punish
			// the call sites that did this correctly.
			"curated message wrapping a driver error survives",
			&platform_http.BadRequestError{
				Message:     "Couldn't add. Fleet doesn't exist.",
				InternalErr: fkErr,
			},
			"Couldn't add. Fleet doesn't exist.",
		},
		{
			// UserMessageError embeds the error interface, which promotes Error() but
			// not Unwrap(); without the Unwrap it defines, errors.As stops at the
			// wrapper and the driver text underneath is returned verbatim.
			"driver error behind a UserMessageError is replaced",
			platform_http.NewUserMessageError(fkErr, 400),
			platform_http.GenericErrorMessage,
		},
		{
			"validation message survives",
			errors.New("query payload verification: report name cannot be empty"),
			"query payload verification: report name cannot be empty",
		},
		{
			"auth message survives",
			errors.New("Authorization header required"),
			"Authorization header required",
		},
		{
			// Describes the caller's own request body, not the server.
			"parser error survives",
			errors.New("failed to parse multipart form: multipart: NextPart: EOF"),
			"failed to parse multipart form: multipart: NextPart: EOF",
		},
		{
			// Fleet manages devices; a host named like a database is ordinary and its
			// own error text must not be mistaken for storage detail.
			"host named after a database survives",
			errors.New(`Host "mysql-01" doesn't exist`),
			`Host "mysql-01" doesn't exist`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, safeReason(tc.err, tc.err.Error()))
		})
	}
}

// TestEncodeErrorKeepsCuratedStatus pins the interaction between Unwrap and the
// ctxerr.Cause call at the top of EncodeError: a single-error Unwrap lets Cause
// walk past UserMessageError, losing both its status code and its message.
func TestEncodeErrorKeepsCuratedStatus(t *testing.T) {
	t.Parallel()

	inner := errors.New("conditional access bypass disabled")
	err := ctxerr.Wrap(t.Context(), platform_http.NewUserMessageError(inner, http.StatusForbidden))

	recorder := httptest.NewRecorder()
	EncodeError(t.Context(), err, recorder, nil)

	require.Equal(t, http.StatusForbidden, recorder.Code)

	var got JsonError
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &got))
	require.Equal(t, http.StatusText(http.StatusForbidden), got.Message)
	// UserMessageError exists to put this text in front of the caller; what must
	// not happen is losing the status and falling through to the 500 branch.
	require.Equal(t, inner.Error(), got.Errors[0]["reason"])
}

// TestEncodeErrorSanitizesBody drives the real encoder, which is what the switch
// in EncodeError has to keep doing; safeReason being correct is not enough if a
// branch stops calling it.
func TestEncodeErrorSanitizesBody(t *testing.T) {
	t.Parallel()

	fkErr := &mysql.MySQLError{
		Number: 1452,
		Message: "Cannot add or update a child row: a foreign key constraint fails " +
			"(`fleet`.`queries`, CONSTRAINT `queries_ibfk_2` FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`))",
	}

	ctx := logging.NewContext(t.Context(), &logging.LoggingContext{})
	logCtx, ok := logging.FromContext(ctx)
	require.True(t, ok)

	recorder := httptest.NewRecorder()
	EncodeError(ctx, fmt.Errorf("creating new Query: %w", fkErr), recorder, nil)

	body := recorder.Body.String()
	for _, leaked := range []string{"queries_ibfk_2", "REFERENCES", "fleet`.`queries", "1452"} {
		require.NotContains(t, body, leaked)
	}

	var got JsonError
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &got))
	require.Equal(t, platform_http.GenericErrorMessage, got.Errors[0]["reason"])

	// The response is only traceable if this is the same id the log line carries.
	require.NotEmpty(t, got.UUID)
	require.Equal(t, logCtx.RequestID, got.UUID)
}
