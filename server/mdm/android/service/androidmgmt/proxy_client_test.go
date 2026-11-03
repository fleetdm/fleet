package androidmgmt

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/googleapi"
)

func TestProxyClientEnterprisesCreateStatusError(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		wantQuotaErr  bool
		wantErrSubstr string
	}{
		{name: "too many requests", status: http.StatusTooManyRequests, wantQuotaErr: true, wantErrSubstr: "unexpected status code: 429"},
		{name: "server error", status: http.StatusInternalServerError, wantErrSubstr: "unexpected status code: 500"},
		{name: "not modified", status: http.StatusNotModified, wantErrSubstr: "was already created"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/enterprises", r.URL.Path)
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()

			client := NewProxyClient(t.Context(), slog.New(slog.DiscardHandler), "license", func(name string) string {
				if name == "FLEET_DEV_ANDROID_PROXY_ENDPOINT" {
					return srv.URL + "/"
				}
				return ""
			})
			require.NotNil(t, client)

			_, err := client.EnterprisesCreate(t.Context(), EnterprisesCreateRequest{SignupURLName: "signup"})
			require.ErrorContains(t, err, tt.wantErrSubstr)
			assert.Equal(t, tt.wantQuotaErr, IsTooManyRequestsError(err))
			if tt.status != http.StatusNotModified {
				var ae *googleapi.Error
				require.ErrorAs(t, err, &ae)
				assert.Equal(t, tt.status, ae.Code)
			}
		})
	}
}

func TestNewRetryClientDefaultDelays(t *testing.T) {
	client, ok := NewRetryClient(&ProxyClient{}, slog.New(slog.DiscardHandler)).(*retryClient)
	require.True(t, ok)
	assert.Equal(t, []time.Duration{60 * time.Second}, client.delays)
	assert.Equal(t, 75*time.Second, client.budget)
}

func TestRetryWait(t *testing.T) {
	tests := []struct {
		name      string
		delay     time.Duration
		remaining time.Duration
		wantOK    bool
		wantMax   time.Duration
	}{
		{name: "plenty of budget allows 50% jitter", delay: 60 * time.Second, remaining: 10 * time.Minute, wantOK: true, wantMax: 90 * time.Second},
		{name: "jitter is clamped to the budget", delay: 60 * time.Second, remaining: 70 * time.Second, wantOK: true, wantMax: 70 * time.Second},
		{name: "delay equal to the budget has no jitter", delay: 60 * time.Second, remaining: 60 * time.Second, wantOK: true, wantMax: 60 * time.Second},
		{name: "delay past the budget is not retried", delay: 60 * time.Second, remaining: 59 * time.Second, wantOK: false},
		{name: "exhausted budget is not retried", delay: 60 * time.Second, remaining: -time.Second, wantOK: false},
		{name: "zero delay", delay: 0, remaining: time.Second, wantOK: true, wantMax: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for range 1000 {
				wait, ok := retryWait(tt.delay, tt.remaining)
				require.Equal(t, tt.wantOK, ok)
				if !ok {
					require.Zero(t, wait)
					continue
				}
				require.GreaterOrEqual(t, wait, tt.delay)
				require.LessOrEqual(t, wait, tt.wantMax)
			}
		})
	}
}
