package condaccess

import (
	"errors"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/middleware/otel"
	kitlog "github.com/go-kit/log"
)

const (
	idpMetadataPath    = "/api/fleet/conditional_access/idp/metadata"
	idpSSOPath         = "/api/fleet/conditional_access/idp/sso"
	idpSigningCertPath = "/api/fleet/conditional_access/idp/signing_cert"
)

// idpService implements the Okta conditional access IdP functionality.
type idpService struct {
	ds     fleet.Datastore
	logger kitlog.Logger
}

// RegisterIdP registers the HTTP handlers for Okta conditional access IdP endpoints.
func RegisterIdP(
	mux *http.ServeMux,
	ds fleet.Datastore,
	logger kitlog.Logger,
	fleetConfig *config.FleetConfig,
) error {
	if fleetConfig == nil {
		return errors.New("fleet config is nil")
	}

	svc := &idpService{
		ds:     ds,
		logger: kitlog.With(logger, "component", "conditional-access-idp"),
	}

	// Register handlers with OpenTelemetry middleware
	metadataHandler := otel.WrapHandler(http.HandlerFunc(svc.serveMetadata), idpMetadataPath, *fleetConfig)
	ssoHandler := otel.WrapHandler(http.HandlerFunc(svc.serveSSO), idpSSOPath, *fleetConfig)
	signingCertHandler := otel.WrapHandler(http.HandlerFunc(svc.serveSigningCert), idpSigningCertPath, *fleetConfig)

	mux.Handle(idpMetadataPath, metadataHandler)
	mux.Handle(idpSSOPath, ssoHandler)
	mux.Handle(idpSigningCertPath, signingCertHandler)

	return nil
}

// serveMetadata handles GET /api/fleet/conditional_access/idp/metadata
// Returns SAML IdP metadata for Okta to consume.
func (s *idpService) serveMetadata(w http.ResponseWriter, r *http.Request) {
	s.logger.Log("msg", "metadata endpoint called (not yet implemented)")
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// serveSSO handles POST /api/fleet/conditional_access/idp/sso
// Handles SAML AuthnRequest from Okta, verifies device certificate and health.
func (s *idpService) serveSSO(w http.ResponseWriter, r *http.Request) {
	s.logger.Log("msg", "SSO endpoint called (not yet implemented)")
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// serveSigningCert handles GET /api/fleet/conditional_access/idp/signing_cert
// Returns the public signing certificate for Okta to verify SAML assertions.
func (s *idpService) serveSigningCert(w http.ResponseWriter, r *http.Request) {
	s.logger.Log("msg", "signing cert endpoint called (not yet implemented)")
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}
