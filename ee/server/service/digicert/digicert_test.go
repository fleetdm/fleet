package digicert

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeout(t *testing.T) {
	const profileID = "e1cda713-1c92-4475-8d60-99228cdc4d04"

	mockDigiCertServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		resp := map[string]string{
			"id":     profileID,
			"name":   "Test CA",
			"status": "Active",
		}
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))
	defer mockDigiCertServer.Close()

	config := fleet.DigiCertCA{
		URL:       mockDigiCertServer.URL,
		APIToken:  "api_token",
		ProfileID: profileID,
	}
	s := NewService(WithTimeout(1 * time.Millisecond))
	err := s.VerifyProfileID(context.Background(), config)
	assert.ErrorContains(t, err, "exceeded")
}

func TestGetCertificateTransientErrors(t *testing.T) {
	// A status of 0 is a server that isn't running.
	for status, wantTransient := range map[int]bool{
		http.StatusServiceUnavailable: true, http.StatusBadGateway: true, http.StatusTooManyRequests: true, 0: true,
		http.StatusUnauthorized: false, http.StatusForbidden: false, http.StatusBadRequest: false,
	} {
		t.Run(fmt.Sprintf("status %d", status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }))
			defer server.Close()
			if status == 0 {
				server.Close()
			}
			config := fleet.DigiCertCA{URL: server.URL, APIToken: "token", ProfileID: "profile", CertificateCommonName: "cn", CertificateSeatID: "seat"}
			_, err := NewService().GetCertificate(t.Context(), config)
			require.Error(t, err)
			_, isTransient := errors.AsType[fleet.CertificateAuthorityTransientError](err)
			assert.Equal(t, wantTransient, isTransient)
		})
	}
}
