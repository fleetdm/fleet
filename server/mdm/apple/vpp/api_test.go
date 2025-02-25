package vpp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func setupFakeServer(t *testing.T, handler http.HandlerFunc) {
	server := httptest.NewServer(handler)
	os.Setenv("FLEET_DEV_VPP_URL", server.URL)
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
	tests := []struct {
		name           string
		token          string
		filter         *AssetFilter
		handler        http.HandlerFunc
		expectedAssets []Asset
		expectedErrMsg string
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
			expectedErrMsg: "",
		},
		{
			name:   "server error",
			token:  "valid_token",
			filter: nil,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, `Internal Server Error`)
			},
			expectedAssets: nil,
			expectedErrMsg: "calling Apple VPP endpoint failed with status 500: Internal Server Error\n",
		},
		{
			name:   "client error",
			token:  "valid_token",
			filter: nil,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintln(w, `{"errorInfo":{},"errorMessage":"Bad Request","errorNumber":400}`)
			},
			expectedAssets: nil,
			expectedErrMsg: "retrieving assets: Apple VPP endpoint returned error: Bad Request (error number: 400)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupFakeServer(t, tt.handler)

			assets, err := GetAssets(tt.token, tt.filter)
			if tt.expectedErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedAssets, assets)
			}
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
			req, err := http.NewRequest(http.MethodGet, os.Getenv("FLEET_DEV_VPP_URL"), nil)
			require.NoError(t, err)
			err = do[any](req, "test-token", nil)
			require.NoError(t, err)
			require.Equal(t, tt.wantCalls, calls)
			require.WithinRange(t, time.Now(), start, start.Add(time.Duration(tt.wantCalls)*time.Second))
		})
	}
}

func TestGetBaseURL(t *testing.T) {
	t.Run("Default URL", func(t *testing.T) {
		os.Setenv("FLEET_DEV_VPP_URL", "")
		require.Equal(t, "https://vpp.itunes.apple.com/mdm/v2", getBaseURL())
	})

	t.Run("Custom URL", func(t *testing.T) {
		customURL := "http://localhost:8000"
		os.Setenv("FLEET_DEV_VPP_URL", customURL)
		require.Equal(t, customURL, getBaseURL())
	})
}
