package vpp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/stretchr/testify/require"
)

func setupFakeServer(t *testing.T, handler http.HandlerFunc) {
	server := httptest.NewServer(handler)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", server.URL, t)
	t.Cleanup(server.Close)
}

func TestGetConfig(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		handler        http.HandlerFunc
		wantName       string
		expectedErrMsg string
	}{
		{
			name:  "valid token",
			token: "valid_token",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"locationName": "Test Location"}`)
			},
			wantName:       "Test Location",
			expectedErrMsg: "",
		},
		{
			name:  "invalid token",
			token: "invalid_token",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprintln(w, `{"errorNumber": 9622}`)
			},
			wantName:       "",
			expectedErrMsg: "making request to Apple VPP endpoint: Apple VPP endpoint returned error:  (error number: 9622)",
		},
		{
			name:  "server error",
			token: "valid_token",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, `Internal Server Error`)
			},
			wantName:       "",
			expectedErrMsg: "calling Apple VPP endpoint failed with status 500: Internal Server Error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupFakeServer(t, tt.handler)

			name, err := GetConfig(tt.token)
			if tt.expectedErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.wantName, name)
		})
	}
}

func TestAssociateAssets(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		params         *AssociateAssetsRequest
		handler        http.HandlerFunc
		expectedErrMsg string
	}{
		{
			name:  "valid request",
			token: "valid_token",
			params: &AssociateAssetsRequest{
				Assets:        []Asset{{AdamID: "12345", PricingParam: "STDQ"}},
				SerialNumbers: []string{"SN12345"},
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "/assets/associate", r.URL.Path)
				require.Equal(t, "Bearer valid_token", r.Header.Get("Authorization"))

				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)

				var reqParams AssociateAssetsRequest
				err = json.Unmarshal(body, &reqParams)
				require.NoError(t, err)

				require.Equal(t, []Asset{{AdamID: "12345", PricingParam: "STDQ"}}, reqParams.Assets)
				require.Equal(t, []string{"SN12345"}, reqParams.SerialNumbers)

				_, _ = w.Write([]byte(`{"eventId": "123"}`))
			},
			expectedErrMsg: "",
		},
		{
			name:  "server error",
			token: "valid_token",
			params: &AssociateAssetsRequest{
				Assets:        []Asset{{AdamID: "12345", PricingParam: "STDQ"}},
				SerialNumbers: []string{"SN12345"},
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, `Internal Server Error`)
			},
			expectedErrMsg: "calling Apple VPP endpoint failed with status 500: Internal Server Error\n",
		},
		{
			name:  "client error",
			token: "valid_token",
			params: &AssociateAssetsRequest{
				Assets:        []Asset{{AdamID: "12345", PricingParam: "STDQ"}},
				SerialNumbers: []string{"SN12345"},
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintln(w, `{"errorInfo":{},"errorMessage":"Bad Request","errorNumber":400}`)
			},
			expectedErrMsg: "making request to Apple VPP endpoint: Apple VPP endpoint returned error: Bad Request (error number: 400)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupFakeServer(t, tt.handler)

			_, err := AssociateAssets(t.Context(), tt.token, tt.params)
			if tt.expectedErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetAssets(t *testing.T) {
	originalClient := client
	client = fleethttp.NewClient(fleethttp.WithTimeout(time.Second))
	t.Cleanup(func() {
		client = originalClient
	})

	var requestCount atomic.Int64

	tests := []struct {
		name             string
		token            string
		filter           *AssetFilter
		handler          http.HandlerFunc
		expectedAssets   []Asset
		expectedErrMsg   string
		expectedRequests int
	}{
		{
			name:  "valid token and filters",
			token: "valid_token",
			filter: &AssetFilter{
				AdamID: "12345",
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodGet, r.Method)
				require.Equal(t, "/assets", r.URL.Path)
				require.Equal(t, "Bearer valid_token", r.Header.Get("Authorization"))

				query := r.URL.Query()
				require.Equal(t, "12345", query.Get("adamId"))

				type resp struct {
					Assets []Asset `json:"assets"`
				}
				assets := resp{
					Assets: []Asset{
						{AdamID: "12345", PricingParam: "STDQ"},
						{AdamID: "67890", PricingParam: "PLUS"},
					},
				}
				w.WriteHeader(http.StatusOK)
				require.NoError(t, json.NewEncoder(w).Encode(assets))
			},
			expectedAssets: []Asset{
				{AdamID: "12345", PricingParam: "STDQ"},
				{AdamID: "67890", PricingParam: "PLUS"},
			},
			expectedErrMsg:   "",
			expectedRequests: 1,
		},
		{
			name:   "server error",
			token:  "valid_token",
			filter: nil,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, `Internal Server Error`)
			},
			expectedAssets:   nil,
			expectedErrMsg:   "calling Apple VPP endpoint failed with status 500: Internal Server Error\n",
			expectedRequests: 1,
		},
		{
			name:   "client error",
			token:  "valid_token",
			filter: nil,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintln(w, `{"errorInfo":{},"errorMessage":"Bad Request","errorNumber":400}`)
			},
			expectedAssets:   nil,
			expectedErrMsg:   "retrieving assets: Apple VPP endpoint returned error: Bad Request (error number: 400)",
			expectedRequests: 1,
		},
		{
			name:   "always times out",
			token:  "valid_token",
			filter: nil,
			handler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(time.Second + 500*time.Millisecond) // longer than the 1s client timeout
				type resp struct {
					Assets []Asset `json:"assets"`
				}
				assets := resp{
					Assets: []Asset{
						{AdamID: "12345", PricingParam: "STDQ"},
						{AdamID: "67890", PricingParam: "PLUS"},
					},
				}
				w.WriteHeader(http.StatusOK)
				require.NoError(t, json.NewEncoder(w).Encode(assets))
			},
			expectedAssets:   nil,
			expectedErrMsg:   "exceeded",
			expectedRequests: 3,
		},
		{
			name:   "times out then valid",
			token:  "valid_token",
			filter: nil,
			handler: func(w http.ResponseWriter, r *http.Request) {
				if requestCount.Load() < 2 {
					time.Sleep(time.Second + 500*time.Millisecond) // longer than the 1s client timeout
				}

				type resp struct {
					Assets []Asset `json:"assets"`
				}
				assets := resp{
					Assets: []Asset{
						{AdamID: "12345", PricingParam: "STDQ"},
						{AdamID: "67890", PricingParam: "PLUS"},
					},
				}
				w.WriteHeader(http.StatusOK)
				require.NoError(t, json.NewEncoder(w).Encode(assets))
			},
			expectedAssets: []Asset{
				{AdamID: "12345", PricingParam: "STDQ"},
				{AdamID: "67890", PricingParam: "PLUS"},
			},
			expectedErrMsg:   "",
			expectedRequests: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestCount.Store(0)

			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount.Add(1)
				tt.handler(w, r)
			})
			setupFakeServer(t, h)

			assets, err := GetAssets(t.Context(), tt.token, tt.filter)
			if tt.expectedErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedAssets, assets)
			}
			require.EqualValues(t, tt.expectedRequests, requestCount.Load())
		})
	}
}

func TestDoRetryAfter(t *testing.T) {
	tests := []struct {
		name      string
		handler   http.HandlerFunc
		wantCalls int
		wantErr   bool
	}{
		{
			name: "no retry-after header",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, err := w.Write([]byte("{}"))
				require.NoError(t, err)
			},
			wantCalls: 1,
			wantErr:   true,
		},
		{
			name: "invalid retry-after header",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("Retry-After", "foo")
				w.WriteHeader(http.StatusInternalServerError)
				_, err := w.Write([]byte("{}"))
				require.NoError(t, err)
			},
			wantCalls: 1,
			wantErr:   true,
		},
		{
			name: "three retries",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("Retry-After", "1")
				w.WriteHeader(http.StatusInternalServerError)
				_, err := w.Write([]byte("{}"))
				require.NoError(t, err)
			},
			wantCalls: 3,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls int
			setupFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if calls < tt.wantCalls {
					tt.handler(w, r)
					return
				}
			})

			start := time.Now()
			req, err := http.NewRequest(http.MethodGet, dev_mode.Env("FLEET_DEV_VPP_URL"), nil)
			require.NoError(t, err)
			err = do[any](req, "test-token", nil)
			require.NoError(t, err)
			require.Equal(t, tt.wantCalls, calls)
			require.WithinRange(t, time.Now(), start, start.Add(time.Duration(tt.wantCalls)*time.Second))
		})
	}
}

func TestDoRetry(t *testing.T) {
	t.Run("retries after 500 with Retry-After", func(t *testing.T) {
		var calls int
		setupFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			calls++

			// Verify Authorization header appears exactly once
			authHeaders := r.Header.Values("Authorization")
			require.Len(t, authHeaders, 1,
				"expected exactly 1 Authorization header on attempt %d, got %d: %v",
				calls, len(authHeaders), authHeaders)
			require.Equal(t, "Bearer test-token", authHeaders[0])

			// Verify POST body is intact
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.NotEmpty(t, body, "request body should not be empty on attempt %d", calls)

			var reqParams AssociateAssetsRequest
			err = json.Unmarshal(body, &reqParams)
			require.NoError(t, err, "request body should be valid JSON on attempt %d, got: %q", calls, string(body))
			require.Equal(t, "462054704", reqParams.Assets[0].AdamID)
			require.Equal(t, "GXH409KH7X", reqParams.SerialNumbers[0])

			if calls == 1 {
				// First call: return 500 with Retry-After to trigger retry
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("{}"))
				return
			}

			// Second call: success
			_, _ = w.Write([]byte(`{"eventId": "evt-123"}`))
		})

		eventID, err := AssociateAssets(t.Context(), "test-token", &AssociateAssetsRequest{
			Assets:        []Asset{{AdamID: "462054704", PricingParam: "STDQ"}},
			SerialNumbers: []string{"GXH409KH7X"},
		})
		require.NoError(t, err)
		require.Equal(t, "evt-123", eventID)
		require.Equal(t, 2, calls)
	})

	t.Run("retries after error 9646", func(t *testing.T) {
		var calls int
		setupFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			calls++

			// Verify Authorization header appears exactly once
			authHeaders := r.Header.Values("Authorization")
			require.Len(t, authHeaders, 1,
				"expected exactly 1 Authorization header on attempt %d, got %d: %v",
				calls, len(authHeaders), authHeaders)

			// Verify POST body is intact
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.NotEmpty(t, body, "request body should not be empty on attempt %d", calls)

			var reqParams AssociateAssetsRequest
			err = json.Unmarshal(body, &reqParams)
			require.NoError(t, err, "request body should be valid JSON on attempt %d, got: %q", calls, string(body))
			require.Equal(t, "462054704", reqParams.Assets[0].AdamID)

			if calls == 1 {
				// First call: return rate-limit error 9646
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"errorMessage":"Too many requests","errorNumber":9646}`))
				return
			}

			// Second call: success
			_, _ = w.Write([]byte(`{"eventId": "evt-456"}`))
		})

		eventID, err := AssociateAssets(t.Context(), "test-token", &AssociateAssetsRequest{
			Assets:        []Asset{{AdamID: "462054704", PricingParam: "STDQ"}},
			SerialNumbers: []string{"GXH409KH7X"},
		})
		require.NoError(t, err)
		require.Equal(t, "evt-456", eventID)
		require.GreaterOrEqual(t, calls, 2)
	})
}

// associateAssetsParams is a small valid request used by the retry tests below.
func associateAssetsParams() *AssociateAssetsRequest {
	return &AssociateAssetsRequest{
		Assets:        []Asset{{AdamID: "1", PricingParam: "STDQ"}},
		SerialNumbers: []string{"SN1"},
	}
}

// TestDoRetryIsBoundedAndNonRecursive verifies that when Apple persistently
// returns a retryable condition, do() retries a BOUNDED number of times and
// returns — it must never recurse (which previously stacked open response
// bodies / cancel-watcher goroutines / spans / timers per level and OOM'd the
// server). It also verifies a sustained Retry-After stays bounded and that
// context cancellation aborts the backoff promptly.
// See https://github.com/fleetdm/fleet/issues/46656.
func TestDoRetryIsBoundedAndNonRecursive(t *testing.T) {
	// Shrink the retry knobs so the bounded loop runs fast.
	origAttempts, origBackoff, origInterval, origMult := vppMaxAttempts, maxVPPBackoff, vppRateLimitInterval, vppRateLimitBackoffMultiplier
	t.Cleanup(func() {
		vppMaxAttempts, maxVPPBackoff, vppRateLimitInterval, vppRateLimitBackoffMultiplier = origAttempts, origBackoff, origInterval, origMult
	})
	vppMaxAttempts = 4
	vppRateLimitInterval = 1 * time.Millisecond
	maxVPPBackoff = 5 * time.Millisecond
	vppRateLimitBackoffMultiplier = 2

	t.Run("rate-limited (too many requests) retries a bounded number of times then fails", func(t *testing.T) {
		var calls atomic.Int32
		setupFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"errorMessage":"Too many requests","errorNumber":9646}`))
		})

		ctx := t.Context()
		done := make(chan error, 1)
		go func() {
			_, err := AssociateAssets(ctx, "tok", associateAssetsParams())
			done <- err
		}()

		select {
		case err := <-done:
			require.Error(t, err)
			require.Contains(t, err.Error(), "rate limited")
		case <-time.After(5 * time.Second):
			t.Fatal("AssociateAssets did not return — the retry loop is not bounded")
		}

		require.EqualValues(t, vppMaxAttempts, calls.Load(),
			"expected exactly vppMaxAttempts requests; more means the retries are nesting/recursing")
	})

	t.Run("HTTP 500 + Retry-After is honored but capped and bounded", func(t *testing.T) {
		var calls atomic.Int32
		setupFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			w.Header().Set("Retry-After", "600") // Apple asks for 10 minutes
			w.WriteHeader(http.StatusInternalServerError)
		})

		start := time.Now()
		_, err := AssociateAssets(t.Context(), "tok", associateAssetsParams())
		require.Error(t, err)
		// Bounded to vppMaxAttempts — a sustained Retry-After must NOT loop forever.
		require.EqualValues(t, vppMaxAttempts, calls.Load())
		// Retry-After is honored but capped at maxVPPBackoff (5ms here), so the
		// call finishes far under the 600s Apple requested — a multi-minute value
		// can't pin a synchronous request open.
		require.Less(t, time.Since(start), 2*time.Second)
	})

	t.Run("context cancellation aborts the backoff promptly", func(t *testing.T) {
		// Use a long backoff so that, without ctx cancellation, the call would block.
		vppRateLimitInterval = 30 * time.Second
		maxVPPBackoff = 30 * time.Second
		t.Cleanup(func() {
			vppRateLimitInterval = 1 * time.Millisecond
			maxVPPBackoff = 5 * time.Millisecond
		})

		setupFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"errorMessage":"Too many requests","errorNumber":9646}`))
		})

		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()

		start := time.Now()
		_, err := AssociateAssets(ctx, "tok", associateAssetsParams())
		require.Error(t, err)
		require.ErrorContains(t, err, "context")
		require.Less(t, time.Since(start), 2*time.Second, "ctx cancellation should abort the backoff sleep")
	})

	t.Run("applies a growing backoff between retries", func(t *testing.T) {
		// Override for measurable, non-flaky spacing; restore afterward.
		oa, oi, om, ob := vppMaxAttempts, vppRateLimitInterval, vppRateLimitBackoffMultiplier, maxVPPBackoff
		t.Cleanup(func() {
			vppMaxAttempts, vppRateLimitInterval, vppRateLimitBackoffMultiplier, maxVPPBackoff = oa, oi, om, ob
		})
		vppMaxAttempts = 3
		vppRateLimitInterval = 30 * time.Millisecond
		vppRateLimitBackoffMultiplier = 2
		maxVPPBackoff = time.Second // generous, so capping doesn't interfere here

		var mu sync.Mutex
		var times []time.Time
		setupFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			times = append(times, time.Now())
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"errorMessage":"Too many requests","errorNumber":9646}`))
		})

		_, err := AssociateAssets(t.Context(), "tok", associateAssetsParams())
		require.Error(t, err)

		mu.Lock()
		defer mu.Unlock()
		require.Len(t, times, 3)
		// The backoff is actually applied between attempts (not skipped) and
		// grows (30ms, then 60ms). Timers fire at-or-after their interval, so
		// these lower bounds are not flaky.
		require.GreaterOrEqual(t, times[1].Sub(times[0]), 30*time.Millisecond)
		require.GreaterOrEqual(t, times[2].Sub(times[1]), 60*time.Millisecond)
	})
}

// TestDoVPPAttemptClampsRetryAfter verifies that an absurdly large Retry-After
// value is clamped to the backoff cap before being scaled to a time.Duration,
// rather than overflowing the int64 nanosecond math (which could wrap negative
// and bypass the cap). Uses the default maxVPPBackoff and calls doVPPAttempt
// directly (it doesn't sleep), so the test is instant.
func TestDoVPPAttemptClampsRetryAfter(t *testing.T) {
	setupFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
		// ~1e14 seconds: seconds * 1e9 ns overflows int64 if not clamped first.
		w.Header().Set("Retry-After", "99999999999999")
		w.WriteHeader(http.StatusInternalServerError)
	})

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, dev_mode.Env("FLEET_DEV_VPP_URL"), nil)
	require.NoError(t, err)

	done, retryAfter, err := doVPPAttempt[any](req, nil)
	require.NoError(t, err)
	require.False(t, done, "a 500 + Retry-After should be retryable, not terminal")
	require.Greater(t, retryAfter, time.Duration(0), "clamped Retry-After must stay positive (no overflow to negative)")
	require.Equal(t, maxVPPBackoff, retryAfter, "an over-cap Retry-After should clamp to the backoff cap")
}


func TestGetBaseURL(t *testing.T) {
	t.Run("Default URL", func(t *testing.T) {
		require.Equal(t, "https://vpp.itunes.apple.com/mdm/v2", getBaseURL())
	})

	t.Run("Custom URL", func(t *testing.T) {
		customURL := "http://localhost:8000"
		dev_mode.SetOverride("FLEET_DEV_VPP_URL", customURL, t)
		require.Equal(t, customURL, getBaseURL())
	})
}
