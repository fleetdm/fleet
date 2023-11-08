package service

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

type mockHttpClient struct {
	resBody    string
	statusCode int
	err        error
}

func (m *mockHttpClient) Do(req *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}

	res := &http.Response{
		StatusCode: m.statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(m.resBody)),
	}

	return res, nil
}

func TestDeviceClientGetDesktopPayload(t *testing.T) {
	client, err := NewDeviceClient("https://test.com", true, "", nil, "")
	token := "test_token"
	require.NoError(t, err)

	mockRequestDoer := &mockHttpClient{}
	client.http = mockRequestDoer

	t.Run("with wrong license", func(t *testing.T) {
		mockRequestDoer.statusCode = http.StatusPaymentRequired
		_, err = client.DesktopSummary(token)
		require.ErrorIs(t, err, ErrMissingLicense)
	})

	t.Run("with no failing policies", func(t *testing.T) {
		mockRequestDoer.statusCode = http.StatusOK
		mockRequestDoer.resBody = `{}`
		result, err := client.DesktopSummary(token)
		require.NoError(t, err)
		require.EqualValues(t, 0, *result.FailingPolicies)
		require.False(t, result.Notifications.NeedsMDMMigration)
	})

	t.Run("with failing policies", func(t *testing.T) {
		mockRequestDoer.statusCode = http.StatusOK
		mockRequestDoer.resBody = `{"failing_policies_count": 1}`
		result, err := client.DesktopSummary(token)
		require.NoError(t, err)
		require.EqualValues(t, 1, *result.FailingPolicies)
		require.False(t, result.Notifications.NeedsMDMMigration)
	})

	t.Run("with flag to enable MDM migration", func(t *testing.T) {
		mockRequestDoer.statusCode = http.StatusOK
		mockRequestDoer.resBody = `{"failing_policies_count": 15, "notifications": {"needs_mdm_migration": true}}`
		result, err := client.DesktopSummary(token)
		require.NoError(t, err)
		require.EqualValues(t, 15, *result.FailingPolicies)
		require.True(t, result.Notifications.NeedsMDMMigration)
	})
}

func TestDeviceClientRetryInvalidToken(t *testing.T) {
	var callCounts atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCounts.Add(1)

		parts := strings.Split(r.URL.Path, "/")
		token := parts[len(parts)-2] // last parts are /.../{token}/desktop
		require.NotEmpty(t, token)
		if token == "good_token" {
			fmt.Fprint(w, `{"failing_policies_count": 1}`)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	t.Run("no retry, bad token", func(t *testing.T) {
		t.Cleanup(func() { callCounts.Store(0) })

		client, err := NewDeviceClient(srv.URL, true, "", nil, "")
		require.NoError(t, err)

		_, err = client.DesktopSummary("bad")
		require.Error(t, err)
		require.ErrorIs(t, err, ErrUnauthenticated)
		require.Equal(t, int64(1), callCounts.Load())
	})

	t.Run("no retry, good token", func(t *testing.T) {
		t.Cleanup(func() { callCounts.Store(0) })

		client, err := NewDeviceClient(srv.URL, true, "", nil, "")
		require.NoError(t, err)

		_, err = client.DesktopSummary("good_token")
		require.NoError(t, err)
		require.Equal(t, int64(1), callCounts.Load())
	})

	t.Run("with retry, good after retry", func(t *testing.T) {
		t.Cleanup(func() { callCounts.Store(0) })

		client, err := NewDeviceClient(srv.URL, true, "", nil, "")
		require.NoError(t, err)
		client.WithInvalidTokenRetry(func() string {
			return "good_token"
		})

		_, err = client.DesktopSummary("bad")
		require.NoError(t, err)
		require.Equal(t, int64(2), callCounts.Load())
	})

	t.Run("with retry, good after 2 retries", func(t *testing.T) {
		t.Cleanup(func() { callCounts.Store(0) })

		client, err := NewDeviceClient(srv.URL, true, "", nil, "")
		require.NoError(t, err)

		var newToken string
		client.WithInvalidTokenRetry(func() string {
			switch newToken {
			case "":
				newToken = "bad"
			default:
				newToken = "good_token"
			}
			return newToken
		})

		_, err = client.DesktopSummary("bad")
		require.NoError(t, err)
		require.Equal(t, int64(3), callCounts.Load())
	})

	t.Run("with retry, always bad", func(t *testing.T) {
		t.Cleanup(func() { callCounts.Store(0) })

		client, err := NewDeviceClient(srv.URL, true, "", nil, "")
		require.NoError(t, err)

		client.WithInvalidTokenRetry(func() string {
			return "still-bad"
		})

		_, err = client.DesktopSummary("bad")
		require.Error(t, err)
		require.ErrorIs(t, err, ErrUnauthenticated)
		require.Equal(t, int64(4), callCounts.Load())
	})
}
