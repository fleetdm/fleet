package vpp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func setupFakeServer(t *testing.T, handler http.HandlerFunc) {
	server := httptest.NewServer(handler)
	os.Setenv("FLEET_DEV_VPP_URL", server.URL)
	t.Cleanup(server.Close)
}

func TestGetConfig(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		handler   http.HandlerFunc
		wantName  string
		expectErr bool
	}{
		{
			name:  "valid token",
			token: "valid_token",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"locationName": "Test Location"}`)
			},
			wantName:  "Test Location",
			expectErr: false,
		},
		{
			name:  "invalid token",
			token: "invalid_token",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprintln(w, `{"errorNumber": 9622}`)
			},
			wantName:  "",
			expectErr: true,
		},
		{
			name:  "server error",
			token: "valid_token",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantName:  "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupFakeServer(t, tt.handler)

			name, err := GetConfig(tt.token)
			if tt.expectErr {
				require.Error(t, err)
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
		expectErr      bool
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

				w.WriteHeader(http.StatusOK)
			},
			expectErr: false,
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
				fmt.Fprintln(w, `{"errorInfo":{},"errorMessage":"Internal Server Error","errorNumber":500}`)
			},
			expectErr:      true,
			expectedErrMsg: "Apple VPP endpoint returned error: Internal Server Error (error number: 500)",
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
			expectErr:      true,
			expectedErrMsg: "Apple VPP endpoint returned error: Bad Request (error number: 400)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupFakeServer(t, tt.handler)

			err := AssociateAssets(tt.token, tt.params)
			if tt.expectErr {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.expectedErrMsg)
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
		expectErr      bool
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
			expectErr: false,
		},
		{
			name:   "server error",
			token:  "valid_token",
			filter: nil,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, `{"errorInfo":{},"errorMessage":"Internal Server Error","errorNumber":500}`)
			},
			expectedAssets: nil,
			expectErr:      true,
			expectedErrMsg: "Apple VPP endpoint returned error: Internal Server Error (error number: 500)",
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
			expectErr:      true,
			expectedErrMsg: "Apple VPP endpoint returned error: Bad Request (error number: 400)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupFakeServer(t, tt.handler)

			assets, err := GetAssets(tt.token, tt.filter)
			if tt.expectErr {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.expectedErrMsg)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedAssets, assets)
			}
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
