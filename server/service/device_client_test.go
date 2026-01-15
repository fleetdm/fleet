package service

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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

	t.Run("alternative browser URL gets set from server response", func(t *testing.T) {
		mockRequestDoer.statusCode = http.StatusOK
		mockRequestDoer.resBody = `{"alternative_browser_host": "gogetit.com:6969"}`
		_, err := client.DesktopSummary(token)
		require.NoError(t, err)
		require.EqualValues(t, "gogetit.com:6969", client.fleetAlternativeBrowserHostFromServer)
	})
}

func TestApplyAlternativeBrowserHostSetting(t *testing.T) {
	tests := []struct {
		name          string
		serverSetting string
		envSetting    string
		initialURL    string
		expectedURL   string
	}{
		{
			name:          "server setting with path prepends to existing path",
			serverSetting: "https://proxy.example.com/prefix",
			envSetting:    "",
			initialURL:    "https://fleet.example.com/api/v1/device/token/ping",
			expectedURL:   "https://proxy.example.com/prefix/api/v1/device/token/ping",
		},
		{
			name:          "server setting without path only changes host",
			serverSetting: "https://proxy.example.com",
			envSetting:    "",
			initialURL:    "https://fleet.example.com/api/v1/device/token/ping",
			expectedURL:   "https://proxy.example.com/api/v1/device/token/ping",
		},
		{
			name:          "client setting used as fallback for host only",
			serverSetting: "",
			envSetting:    "fallback.example.com",
			initialURL:    "https://fleet.example.com/api/v1/device/token/ping",
			expectedURL:   "https://fallback.example.com/api/v1/device/token/ping",
		},
		{
			name:          "server setting takes precedence over client setting",
			serverSetting: "https://server.example.com/path",
			envSetting:    "client.example.com",
			initialURL:    "https://fleet.example.com/ping",
			expectedURL:   "https://server.example.com/path/ping",
		},
		{
			name:          "no settings does not change URL",
			serverSetting: "",
			envSetting:    "",
			initialURL:    "https://fleet.example.com/ping",
			expectedURL:   "https://fleet.example.com/ping",
		},
		{
			name:          "server setting with complex path trims slashes correctly",
			serverSetting: "https://proxy.com/a/b/",
			envSetting:    "",
			initialURL:    "https://fleet.com/c/d",
			expectedURL:   "https://proxy.com/a/b/c/d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc := &DeviceClient{
				fleetAlternativeBrowserHostFromServer: tt.serverSetting,
				fleetAlternativeBrowserHost:           tt.envSetting,
			}

			u, err := url.Parse(tt.initialURL)
			require.NoError(t, err)

			dc.applyAlternativeBrowserHostSetting(u)
			require.Equal(t, tt.expectedURL, u.String())
		})
	}
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
