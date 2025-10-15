package okta

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/config"
	otelmw "github.com/fleetdm/fleet/v4/server/service/middleware/otel"
	"github.com/gorilla/mux"
)

type ctxKey int

const (
	// clientCertSerialKey is the context key for storing client certificate serial number
	clientCertSerialKey ctxKey = iota
)

// MakeHandler creates an HTTP handler for Okta device health endpoints with OTEL middleware
func (s *Service) MakeHandler(baseMetadataURL, baseSSOURL string, config config.FleetConfig) http.Handler {
	r := mux.NewRouter()
	r.Handle("/api/v1/fleet/okta/device_health/metadata", s.makeOktaDeviceHealthMetadataHandler(baseMetadataURL, baseSSOURL))
	r.Handle("/api/v1/fleet/okta/device_health/sso", s.makeOktaDeviceHealthSSOHandler(baseMetadataURL, baseSSOURL))

	// Wrap with OTEL middleware
	return otelmw.WrapHandlerDynamic(r, config)
}

// //////////////////////////////////////////////////////////////////////////////
// GET /api/v1/fleet/okta/device_health/metadata
// //////////////////////////////////////////////////////////////////////////////

func (s *Service) makeOktaDeviceHealthMetadataHandler(baseMetadataURL, baseSSOURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idp, err := s.getOktaDeviceHealthIDP(baseMetadataURL, baseSSOURL)
		if err != nil {
			s.logger.Log("err", "error creating IdP", "details", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// ServeMetadata writes XML directly to the response
		idp.ServeMetadata(w, r)
	}
}

// //////////////////////////////////////////////////////////////////////////////
// POST /api/v1/fleet/okta/device_health/sso
// //////////////////////////////////////////////////////////////////////////////

func (s *Service) makeOktaDeviceHealthSSOHandler(baseMetadataURL, baseSSOURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract client certificate from mTLS proxy header (DER base64 encoded)
		clientCertB64 := r.Header.Get("X-Client-Cert")
		if clientCertB64 != "" {
			// Decode base64 DER certificate
			certDER, err := base64.StdEncoding.DecodeString(clientCertB64)
			if err != nil {
				s.logger.Log("err", "failed to decode client certificate", "details", err)
			} else {
				// Parse certificate
				cert, err := x509.ParseCertificate(certDER)
				if err != nil {
					s.logger.Log("err", "failed to parse client certificate", "details", err)
				} else {
					// Extract serial number for host identity lookup
					serialNumber := cert.SerialNumber.Uint64()

					s.logger.Log("msg", "extracted client certificate", "serial", serialNumber, "subject", cert.Subject.String())

					// Add serial number to request context so GetSession can access it
					ctx := context.WithValue(r.Context(), clientCertSerialKey, serialNumber)
					r = r.WithContext(ctx)
				}
			}
		} else {
			s.logger.Log("msg", "no client certificate in request")
		}

		idp, err := s.getOktaDeviceHealthIDP(baseMetadataURL, baseSSOURL)
		if err != nil {
			s.logger.Log("err", "error creating IdP", "details", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// ServeSSO handles the SAML AuthnRequest and sends back a response
		idp.ServeSSO(w, r)
	}
}
