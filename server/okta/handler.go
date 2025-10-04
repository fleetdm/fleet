package okta

import (
	"net/http"

	"github.com/fleetdm/fleet/v4/server/config"
	otelmw "github.com/fleetdm/fleet/v4/server/service/middleware/otel"
	"github.com/go-kit/log/level"
	"github.com/gorilla/mux"
)

// MakeHandler creates an HTTP handler for Okta device health endpoints with OTEL middleware
func (s *Service) MakeHandler(baseURL string, config config.FleetConfig) http.Handler {
	if baseURL == "" {
		level.Error(s.logger).Log("msg", "FLEET_CONDITIONAL_ACCESS_URL is not set; disabling Okta device health endpoints")
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		})
	}

	r := mux.NewRouter()
	r.Handle("/api/v1/fleet/okta/device_health/metadata", s.makeOktaDeviceHealthMetadataHandler(baseURL))
	r.Handle("/api/v1/fleet/okta/device_health/sso", s.makeOktaDeviceHealthSSOHandler(baseURL))

	// Wrap with OTEL middleware
	return otelmw.WrapHandlerDynamic(r, config)
}

// //////////////////////////////////////////////////////////////////////////////
// GET /api/v1/fleet/okta/device_health/metadata
// //////////////////////////////////////////////////////////////////////////////

func (s *Service) makeOktaDeviceHealthMetadataHandler(baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idp, err := s.getOktaDeviceHealthIDP(baseURL)
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

func (s *Service) makeOktaDeviceHealthSSOHandler(baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idp, err := s.getOktaDeviceHealthIDP(baseURL)
		if err != nil {
			s.logger.Log("err", "error creating IdP", "details", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// ServeSSO handles the SAML AuthnRequest and sends back a response
		idp.ServeSSO(w, r)
	}
}
