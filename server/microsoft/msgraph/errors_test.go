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
