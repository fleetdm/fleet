package condaccess

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/crewjam/saml"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/middleware/otel"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	dsig "github.com/russellhaering/goxmldsig"
)

const (
	idpMetadataPath    = "/api/fleet/conditional_access/idp/metadata"
	idpSSOPath         = "/api/fleet/conditional_access/idp/sso"
	idpSigningCertPath = "/api/fleet/conditional_access/idp/signing_cert"
	idpSSOPrefix       = "okta."
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
	ctx := r.Context()

	// Load AppConfig to get Okta settings
	appConfig, err := s.ds.AppConfig(ctx)
	if err != nil {
		level.Error(s.logger).Log("msg", "failed to load app config", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Get Fleet server URL from config
	serverURL := appConfig.ServerSettings.ServerURL
	if serverURL == "" {
		level.Error(s.logger).Log("msg", "server URL not configured")
		http.Error(w, "Server URL not configured", http.StatusInternalServerError)
		return
	}

	// Build IdP
	idp, err := s.buildIdentityProvider(ctx, appConfig, serverURL)
	if err != nil {
		level.Error(s.logger).Log("msg", "failed to build identity provider", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// ServeMetadata writes XML directly to the response
	idp.ServeMetadata(w, r)
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

// buildIdentityProvider creates a SAML IdentityProvider from AppConfig.
func (s *idpService) buildIdentityProvider(ctx context.Context, appConfig *fleet.AppConfig, serverURL string) (*saml.IdentityProvider, error) {
	// Load Fleet's IdP certificate and key from mdm_config_assets
	assets, err := s.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{
		fleet.MDMAssetConditionalAccessIDPCert,
		fleet.MDMAssetConditionalAccessIDPKey,
	}, nil)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load conditional access idp assets")
	}

	certAsset, certOK := assets[fleet.MDMAssetConditionalAccessIDPCert]
	keyAsset, keyOK := assets[fleet.MDMAssetConditionalAccessIDPKey]
	if !certOK || !keyOK {
		return nil, ctxerr.New(ctx, "conditional access idp certificate or key not found in mdm_config_assets")
	}

	// Parse certificate and key
	cert, key, err := parseCertAndKeyBytes(certAsset.Value, keyAsset.Value)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parse okta idp certificate")
	}

	// Build metadata URL
	metadataURL, err := url.Parse(serverURL)
	if err != nil {
		return nil, ctxerr.Wrap(context.Background(), err, "parse server URL for metadata")
	}
	metadataURL = metadataURL.JoinPath(idpMetadataPath)

	// Build SSO URL (uses okta.* subdomain or dev override)
	ssoServerURL, err := buildSSOServerURL(serverURL)
	if err != nil {
		return nil, ctxerr.Wrap(context.Background(), err, "build SSO server URL")
	}
	ssoURL, err := url.Parse(ssoServerURL)
	if err != nil {
		return nil, ctxerr.Wrap(context.Background(), err, "parse SSO server URL")
	}
	ssoURL = ssoURL.JoinPath(idpSSOPath)

	// Create kitlog adapter for SAML library
	samlLogger := &kitlogAdapter{logger: kitlog.With(s.logger, "component", "saml-idp")}

	// Build IdentityProvider
	idp := &saml.IdentityProvider{
		Key:             key,
		SignatureMethod: dsig.RSASHA256SignatureMethod,
		Logger:          samlLogger,
		Certificate:     cert,
		MetadataURL:     *metadataURL,
		SSOURL:          *ssoURL,
		// SessionProvider and ServiceProviderProvider will be added in Phase 5
	}

	return idp, nil
}

// buildSSOServerURL builds the SSO server base URL.
// It checks for FLEET_DEV_OKTA_SSO_SERVER_URL environment variable first.
// If not set, it transforms the serverURL by prepending "okta." to the hostname.
// Examples:
//   - https://bozo.example.com -> https://okta.bozo.example.com
//   - https://bozo.example.com:8080 -> https://okta.bozo.example.com:8080
func buildSSOServerURL(serverURL string) (string, error) {
	// Check for dev override
	if devURL := os.Getenv("FLEET_DEV_OKTA_SSO_SERVER_URL"); devURL != "" {
		return devURL, nil
	}

	// Parse the server URL
	u, err := url.Parse(serverURL)
	if err != nil {
		return "", fmt.Errorf("parse server URL: %w", err)
	}

	// Prepend "okta." to the hostname
	if u.Hostname() != "" {
		// Reconstruct host with port if present
		newHost := idpSSOPrefix + u.Hostname()
		if port := u.Port(); port != "" {
			newHost = newHost + ":" + port
		}
		u.Host = newHost
	}

	return u.String(), nil
}

// parseCertAndKeyBytes parses PEM-encoded certificate and private key from separate byte slices.
func parseCertAndKeyBytes(certPEM, keyPEM []byte) (*x509.Certificate, crypto.PrivateKey, error) {
	var cert *x509.Certificate
	var key crypto.PrivateKey

	// Parse certificate
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, nil, errors.New("failed to decode certificate PEM")
	}
	var err error
	cert, err = x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parse certificate: %w", err)
	}

	// Parse private key (we always generate RSA PRIVATE KEY format via certificate.EncodePrivateKeyPEM)
	block, _ = pem.Decode(keyPEM)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, nil, errors.New("failed to decode RSA private key PEM")
	}

	key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parse RSA private key: %w", err)
	}

	return cert, key, nil
}
