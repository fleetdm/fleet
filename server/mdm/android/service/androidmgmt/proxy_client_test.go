package androidmgmt

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
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
	assert.Equal(t, []time.Duration{60 * time.Second, 120 * time.Second, 240 * time.Second}, client.delays)
}

func TestProxyClientEnterprisesPoliciesPatchUpdateMask(t *testing.T) {
	tests := []struct {
		name     string
		opts     PoliciesPatchOpts
		wantMask string
	}{
		{name: "no options", opts: PoliciesPatchOpts{}, wantMask: ""},
		{name: "exclude apps", opts: PoliciesPatchOpts{ExcludeApps: true}, wantMask: policyFieldMask},
		{name: "only apps", opts: PoliciesPatchOpts{OnlyUpdateApps: true}, wantMask: "applications"},
		{name: "only status reporting", opts: PoliciesPatchOpts{OnlyUpdateStatusReporting: true}, wantMask: "statusReportingSettings"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMask string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMask = r.URL.Query().Get("updateMask")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"name":"enterprises/e/policies/p","version":"2"}`))
			}))
			defer srv.Close()

			client := NewProxyClient(t.Context(), slog.New(slog.DiscardHandler), "license", func(name string) string {
				if name == "FLEET_DEV_ANDROID_PROXY_ENDPOINT" {
					return srv.URL + "/"
				}
				return ""
			})
			require.NotNil(t, client)

			_, err := client.EnterprisesPoliciesPatch(t.Context(), "enterprises/e/policies/p", &androidmanagement.Policy{}, tt.opts)
			require.NoError(t, err)
			assert.Equal(t, tt.wantMask, gotMask)
		})
	}
}
