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
		wantValid bool
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
			wantValid: true,
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
			wantValid: false,
			expectErr: false,
		},
		{
			name:  "server error",
			token: "valid_token",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantName:  "",
			wantValid: false,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupFakeServer(t, tt.handler)

			name, valid, err := GetConfig(tt.token)
			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.wantName, name)
			require.Equal(t, tt.wantValid, valid)
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
				require.Equal(t, tt.expectedErrMsg, err.Error())
			} else {
				require.NoError(t, err)
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
