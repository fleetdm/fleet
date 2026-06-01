package vpp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/stretchr/testify/assert"
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
		wantCountry    string
		expectedErrMsg string
		expectMinCalls int
		expectMaxCalls int
	}{
		{
			name:  "valid token US",
			token: "valid_token",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"locationName": "Test Location", "countryISO2ACode": "US"}`)
			},
			wantName:       "Test Location",
			wantCountry:    "us",
			expectedErrMsg: "",
			expectMinCalls: 1,
			expectMaxCalls: 1,
		},
		{
			name:  "valid token DE lowercased",
			token: "valid_token",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"locationName": "DE Org", "countryISO2ACode": "DE"}`)
			},
			wantName:       "DE Org",
			wantCountry:    "de",
			expectedErrMsg: "",
			expectMinCalls: 1,
			expectMaxCalls: 1,
		},
		{
			name:  "invalid token",
			token: "invalid_token",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprintln(w, `{"errorNumber": 9622}`)
			},
			wantName:       "",
			wantCountry:    "",
			expectedErrMsg: "making request to Apple VPP endpoint: Apple VPP endpoint returned error:  (error number: 9622)",
			// Apple application errors should not be retried.
			expectMinCalls: 1,
			expectMaxCalls: 1,
		},
		{
			name:  "server error retries up to 3 times",
			token: "valid_token",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, `Internal Server Error`)
			},
			wantName:       "",
			wantCountry:    "",
			expectedErrMsg: "calling Apple VPP endpoint failed with status 500: Internal Server Error\n",
			expectMinCalls: 3,
			expectMaxCalls: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls int
			setupFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				tt.handler(w, r)
			})

			cfg, err := GetConfig(t.Context(), tt.token)
			if tt.expectedErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.wantName, cfg.LocationName)
			require.Equal(t, tt.wantCountry, cfg.CountryCode)
			if tt.expectMinCalls > 0 {
				require.GreaterOrEqual(t, calls, tt.expectMinCalls)
				require.LessOrEqual(t, calls, tt.expectMaxCalls)
			}
		})
	}

	t.Run("transient failure then success", func(t *testing.T) {
		var calls int
		setupFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			if calls < 2 {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, `Internal Server Error`)
				return
			}
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"locationName": "Recovered", "countryISO2ACode": "FR"}`)
		})

		cfg, err := GetConfig(t.Context(), "token")
		require.NoError(t, err)
		require.Equal(t, "Recovered", cfg.LocationName)
		require.Equal(t, "fr", cfg.CountryCode)
		require.Equal(t, 2, calls)
	})
}

func TestAssociateAssetsRequestValidate(t *testing.T) {
	t.Run("serial numbers only is valid", func(t *testing.T) {
		req := &AssociateAssetsRequest{SerialNumbers: []string{"SN1"}}
		require.NoError(t, req.Validate())
	})
	t.Run("client user ids only is valid", func(t *testing.T) {
		req := &AssociateAssetsRequest{ClientUserIds: []string{"user-1"}}
		require.NoError(t, req.Validate())
	})
	t.Run("both populated is rejected", func(t *testing.T) {
		req := &AssociateAssetsRequest{
			SerialNumbers: []string{"SN1"},
			ClientUserIds: []string{"user-1"},
		}
		err := req.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "mutually exclusive")
	})
	t.Run("neither populated is rejected", func(t *testing.T) {
		req := &AssociateAssetsRequest{}
		err := req.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "required")
	})
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
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/assets/associate", r.URL.Path)
				assert.Equal(t, "Bearer valid_token", r.Header.Get("Authorization"))

				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)

				var reqParams AssociateAssetsRequest
				err = json.Unmarshal(body, &reqParams)
				assert.NoError(t, err)

				assert.Equal(t, []Asset{{AdamID: "12345", PricingParam: "STDQ"}}, reqParams.Assets)
				assert.Equal(t, []string{"SN12345"}, reqParams.SerialNumbers)
				assert.Empty(t, reqParams.ClientUserIds)

				// Verify omitempty: clientUserIds key should not appear in the wire payload.
				assert.NotContains(t, string(body), "clientUserIds")

				_, _ = w.Write([]byte(`{"eventId": "123"}`))
			},
			expectedErrMsg: "",
		},
		{
			name:  "valid request with client user ids",
			token: "valid_token",
			params: &AssociateAssetsRequest{
				Assets:        []Asset{{AdamID: "12345", PricingParam: "STDQ"}},
				ClientUserIds: []string{"user-uuid-1", "user-uuid-2"},
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/assets/associate", r.URL.Path)
				assert.Equal(t, "Bearer valid_token", r.Header.Get("Authorization"))

				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)

				var reqParams AssociateAssetsRequest
				err = json.Unmarshal(body, &reqParams)
				assert.NoError(t, err)

				assert.Equal(t, []Asset{{AdamID: "12345", PricingParam: "STDQ"}}, reqParams.Assets)
				assert.Empty(t, reqParams.SerialNumbers)
				assert.Equal(t, []string{"user-uuid-1", "user-uuid-2"}, reqParams.ClientUserIds)

				// Verify omitempty: serialNumbers key should not appear in the wire payload.
				assert.NotContains(t, string(body), "serialNumbers")

				_, _ = w.Write([]byte(`{"eventId": "456"}`))
			},
			expectedErrMsg: "",
		},
		{
			name:  "rejects both serials and client user ids before HTTP",
			token: "valid_token",
			params: &AssociateAssetsRequest{
				Assets:        []Asset{{AdamID: "12345", PricingParam: "STDQ"}},
				SerialNumbers: []string{"SN12345"},
				ClientUserIds: []string{"user-uuid-1"},
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("HTTP request must not be made when validation fails")
			},
			expectedErrMsg: "mutually exclusive",
		},
		{
			name:  "rejects neither serials nor client user ids before HTTP",
			token: "valid_token",
			params: &AssociateAssetsRequest{
				Assets: []Asset{{AdamID: "12345", PricingParam: "STDQ"}},
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("HTTP request must not be made when validation fails")
			},
			expectedErrMsg: "required",
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

			_, err := AssociateAssets(tt.token, tt.params)
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

		eventID, err := AssociateAssets("test-token", &AssociateAssetsRequest{
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

		eventID, err := AssociateAssets("test-token", &AssociateAssetsRequest{
			Assets:        []Asset{{AdamID: "462054704", PricingParam: "STDQ"}},
			SerialNumbers: []string{"GXH409KH7X"},
		})
		require.NoError(t, err)
		require.Equal(t, "evt-456", eventID)
		require.GreaterOrEqual(t, calls, 2)
	})
}

func TestRegisterUser(t *testing.T) {
	t.Run("rejects empty client user id or managed apple id", func(t *testing.T) {
		_, err := RegisterUser("tok", "", "user@example.com")
		require.Error(t, err)

		_, err = RegisterUser("tok", "uuid-1", "")
		require.Error(t, err)
	})

	t.Run("success returns apple userId synchronously", func(t *testing.T) {
		setupFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/registerVPPUserSrv", r.URL.Path)
			// v1 carries the token in the body, not the Authorization header.
			assert.Empty(t, r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err)

			var got struct {
				SToken            string `json:"sToken"`
				ClientUserIDStr   string `json:"clientUserIdStr"`
				ManagedAppleIDStr string `json:"managedAppleIDStr"`
				Email             string `json:"email"`
			}
			assert.NoError(t, json.Unmarshal(body, &got))
			assert.Equal(t, "valid_token", got.SToken)
			assert.Equal(t, "uuid-1", got.ClientUserIDStr)
			assert.Equal(t, "user1@example.com", got.ManagedAppleIDStr)
			// Apple keys on email — Fleet sends the Managed Apple ID for both.
			assert.Equal(t, "user1@example.com", got.Email)

			_, _ = w.Write([]byte(`{
				"status": 0,
				"user": {
					"userId": 12345,
					"status": "Registered",
					"clientUserIdStr": "uuid-1",
					"managedAppleIDStr": "user1@example.com"
				}
			}`))
		})

		appleUserID, err := RegisterUser("valid_token", "uuid-1", "user1@example.com")
		require.NoError(t, err)
		require.Equal(t, "12345", appleUserID)
	})

	t.Run("apple application error surfaces as ErrorResponse", func(t *testing.T) {
		setupFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			// v1 application errors come back as HTTP 200 with status=-1.
			_, _ = w.Write([]byte(`{
				"status": -1,
				"errorNumber": 9637,
				"errorMessage": "Managed Apple ID not found"
			}`))
		})

		_, err := RegisterUser("valid_token", "uuid-1", "missing@example.com")
		require.Error(t, err)

		var appleErr *ErrorResponse
		require.ErrorAs(t, err, &appleErr)
		require.EqualValues(t, 9637, appleErr.ErrorNumber)
		require.Equal(t, "Managed Apple ID not found", appleErr.ErrorMessage)
	})

	t.Run("apple transport-level error surfaces as ErrorResponse", func(t *testing.T) {
		setupFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"errorMessage":"Bad Request","errorNumber":400}`))
		})

		_, err := RegisterUser("valid_token", "uuid-1", "user1@example.com")
		require.Error(t, err)
		require.Contains(t, err.Error(), "error number: 400")
	})

	t.Run("success without user object is treated as error", func(t *testing.T) {
		setupFakeServer(t, func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"status": 0}`))
		})

		_, err := RegisterUser("valid_token", "uuid-1", "user1@example.com")
		require.Error(t, err)
		require.Contains(t, err.Error(), "no user record")
	})
}

func TestIsMaxDevicesPerUserError(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-VPP error",
			err:      errors.New("network down"),
			expected: false,
		},
		{
			name:     "canonical numeric code 9622",
			err:      &ErrorResponse{ErrorMessage: "License count exceeded", ErrorNumber: 9622},
			expected: true,
		},
		{
			name:     "matched by case-insensitive message",
			err:      &ErrorResponse{ErrorMessage: "User has reached the Maximum Number of Devices for this license", ErrorNumber: 99999},
			expected: true,
		},
		{
			name:     "matched by 'device limit' phrasing",
			err:      &ErrorResponse{ErrorMessage: "Device limit exceeded for this client user.", ErrorNumber: 0},
			expected: true,
		},
		{
			name:     "unrelated VPP error 9610",
			err:      &ErrorResponse{ErrorMessage: "Cannot establish a connection.", ErrorNumber: 9610},
			expected: false,
		},
		{
			name:     "wrapped via fmt.Errorf %w still detected",
			err:      fmt.Errorf("calling vpp: %w", &ErrorResponse{ErrorMessage: "User has reached the maximum number of devices.", ErrorNumber: 9622}),
			expected: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, IsMaxDevicesPerUserError(tt.err))
		})
	}
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
