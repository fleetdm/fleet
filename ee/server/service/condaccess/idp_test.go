package condaccess

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/mock"
	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestRegisterIdP(t *testing.T) {
	ds := new(mock.Store)
	logger := kitlog.NewNopLogger()
	cfg := &config.FleetConfig{}

	mux := http.NewServeMux()
	err := RegisterIdP(mux, ds, logger, cfg)
	require.NoError(t, err)

	// Verify all three endpoints are registered
	t.Run("metadata endpoint registered", func(t *testing.T) {
		req := httptest.NewRequest("GET", idpMetadataPath, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		// Should return 501 Not Implemented (handler stub)
		require.Equal(t, http.StatusNotImplemented, w.Code)
	})

	t.Run("SSO endpoint registered", func(t *testing.T) {
		req := httptest.NewRequest("POST", idpSSOPath, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		// Should return 501 Not Implemented (handler stub)
		require.Equal(t, http.StatusNotImplemented, w.Code)
	})

	t.Run("signing cert endpoint registered", func(t *testing.T) {
		req := httptest.NewRequest("GET", idpSigningCertPath, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		// Should return 501 Not Implemented (handler stub)
		require.Equal(t, http.StatusNotImplemented, w.Code)
	})
}

func TestRegisterIdP_NilConfig(t *testing.T) {
	ds := new(mock.Store)
	logger := kitlog.NewNopLogger()
	mux := http.NewServeMux()

	err := RegisterIdP(mux, ds, logger, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "fleet config is nil")
}
