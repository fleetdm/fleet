package digicert

import (
	"context"
	"encoding/json"
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

	config := fleet.DigiCertIntegration{
		URL:       mockDigiCertServer.URL,
		APIToken:  "api_token",
		ProfileID: profileID,
	}
	s := NewService(WithTimeout(1 * time.Millisecond))
	err := s.VerifyProfileID(context.Background(), config)
	assert.ErrorContains(t, err, "deadline exceeded")
}
