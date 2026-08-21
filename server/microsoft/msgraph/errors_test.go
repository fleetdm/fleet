package msgraph

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestErrorUnwrapsUnderlyingCause(t *testing.T) {
	t.Parallel()
	// A token-endpoint failure must keep the oauth2 error reachable.
	retrieveErr := &oauth2.RetrieveError{
		ErrorCode:        "invalid_client",
		ErrorDescription: "AADSTS7000215: Invalid client secret provided.",
	}
	wrapped := fmt.Errorf("outer: %w", newTokenError(retrieveErr))

	graphErr, ok := errors.AsType[*Error](wrapped)
	require.True(t, ok)
	assert.True(t, graphErr.IsAuthError())

	var gotRetrieve *oauth2.RetrieveError
	require.ErrorAs(t, wrapped, &gotRetrieve, "the oauth2 cause must survive wrapping")
	require.NotNil(t, gotRetrieve)
	assert.Equal(t, "invalid_client", gotRetrieve.ErrorCode)
}

func TestAsError(t *testing.T) {
	t.Parallel()
	_, ok := AsError(nil)
	assert.False(t, ok, "a nil error is not a Graph error")

	_, ok = AsError(errors.New("dial tcp: timeout"))
	assert.False(t, ok, "a transport failure is not a Graph error")

	graphErr, ok := AsError(fmt.Errorf("outer: %w", &Error{StatusCode: http.StatusForbidden, Code: "Forbidden"}))
	require.True(t, ok, "a wrapped Graph error must still be reachable")
	assert.Equal(t, http.StatusForbidden, graphErr.StatusCode)
	assert.Equal(t, "Forbidden", graphErr.Code)
}

func TestCredentialRejected(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"no error", nil, false},
		{"401 rejects the credential", &Error{StatusCode: http.StatusUnauthorized}, true},
		{"403 rejects the credential", &Error{StatusCode: http.StatusForbidden}, true},
		{"429 is transient", &Error{StatusCode: http.StatusTooManyRequests}, false},
		{"500 is transient", &Error{StatusCode: http.StatusInternalServerError}, false},
		{"404 is neither", &Error{StatusCode: http.StatusNotFound}, false},
		{"a non-graph error", errors.New("dial tcp: timeout"), false},
		{"a wrapped 401", fmt.Errorf("outer: %w", &Error{StatusCode: http.StatusUnauthorized}), true},
		{"a wrapped non-graph error", fmt.Errorf("outer: %w", errors.New("dial tcp: timeout")), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, CredentialRejected(tc.err))
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		header string
		want   bool // whether a positive duration is expected
	}{
		{"delta seconds", "42", true},
		{"zero", "0", false},
		{"negative is ignored", "-5", false},
		{"absent", "", false},
		{"garbage", "soon", false},
		// RFC 7231 permits an HTTP-date. Reading it as zero would mean no backoff at all.
		{"http date in the future", time.Now().Add(90 * time.Second).UTC().Format(http.TimeFormat), true},
		{"http date in the past", time.Now().Add(-90 * time.Second).UTC().Format(http.TimeFormat), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := parseRetryAfter(tc.header)
			assert.Equal(t, tc.want, got > 0, "got %v", got)
		})
	}
	assert.Equal(t, 42*time.Second, parseRetryAfter("42"))
}

func TestGraphErrorBodyIsBounded(t *testing.T) {
	t.Parallel()
	// An edge proxy can return a large HTML page on 5xx; this string lands in logs and in the admin-visible sync error.
	huge := strings.Repeat("x", 10_000)
	gs := newGraphServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(huge))
	})

	_, err := gs.client(t).ListWindowsAutopilotDevices(t.Context())
	require.Error(t, err)
	graphErr, ok := errors.AsType[*Error](err)
	require.True(t, ok)
	assert.LessOrEqual(t, len(graphErr.Message), maxErrorBodyBytes+len("... (truncated)"))
	assert.Contains(t, graphErr.Message, "truncated")
}

func TestUserFacingMessage(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		err      error
		contains string
	}{
		{"permission", &Error{StatusCode: http.StatusForbidden}, "DeviceManagementServiceConfig.Read.All"},
		{"auth", &Error{StatusCode: http.StatusUnauthorized}, "rejected the credential"},
		{"transient", &Error{StatusCode: http.StatusBadGateway}, "temporarily unavailable"},
		{"other graph error", &Error{StatusCode: http.StatusBadRequest, Message: "AADSTS700016: app not found"}, "AADSTS700016"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Contains(t, UserFacingMessage(tc.err), tc.contains)
		})
	}

	t.Run("non-graph error yields no message", func(t *testing.T) {
		// The caller describes a failure that never reached Graph, since only it knows what it was attempting.
		assert.Empty(t, UserFacingMessage(errors.New("dial tcp: timeout")))
	})

	t.Run("wrapped graph error is still classified", func(t *testing.T) {
		wrapped := fmt.Errorf("list windows autopilot devices: %w", &Error{StatusCode: http.StatusUnauthorized})
		msg := UserFacingMessage(wrapped)
		assert.Contains(t, msg, "rejected the credential")
		assert.NotContains(t, msg, "list windows autopilot devices", "the wrap chain must never reach the UI")
	})
}
