package condaccess

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

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
	idpMetadataPath = "/api/fleet/conditional_access/idp/metadata"
	idpSSOPath      = "/api/fleet/conditional_access/idp/sso"
	idpSSOPrefix    = "okta."

	// URL to redirect users when they have failing conditional access policies and need to remediate their device
	remediateURL = "https://fleetdm.com/remediate"
	// URL to redirect users when there's a certificate error during Okta conditional access authentication
	certificateErrorURL = "https://fleetdm.com/okta-conditional-access-error"

	// SAML constants
	samlBindingHTTPPost = "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
	samlKeyUseSigning   = "signing"

	// Policy response values
	policyResponseFail = "fail"
)

// notFoundError implements fleet.NotFoundError interface for conditional access IdP errors.
type notFoundError struct {
	msg string
}

func (e *notFoundError) Error() string {
	return e.msg
}

func (e *notFoundError) IsNotFound() bool {
	return true
}

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

	mux.Handle(idpMetadataPath, metadataHandler)
	mux.Handle(idpSSOPath, ssoHandler)

	return nil
}

// handleInternalServerError logs the error, records it in context, and returns HTTP 500.
// This function should be used whenever returning StatusInternalServerError to ensure
// consistent error handling across the IdP service.
// Additional key-value pairs can be passed for logging context (e.g., "host_id", hostID).
func handleInternalServerError(ctx context.Context, w http.ResponseWriter, logger kitlog.Logger, msg string, err error, keyvals ...any) {
	// Build the log keyvals starting with msg and err
	logKeyvals := []any{"msg", msg, "err", err}
	logKeyvals = append(logKeyvals, keyvals...)

	level.Error(logger).Log(logKeyvals...)
	ctxerr.Handle(ctx, err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// serveMetadata handles GET /api/fleet/conditional_access/idp/metadata
// Returns SAML IdP metadata for Okta to consume.
func (s *idpService) serveMetadata(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Load AppConfig to get Okta settings
	appConfig, err := s.ds.AppConfig(ctx)
	if err != nil {
		handleInternalServerError(ctx, w, s.logger, "failed to load app config", err)
		return
	}

	// Get Fleet server URL from config
	serverURL := appConfig.ServerSettings.ServerURL
	if serverURL == "" {
		level.Error(s.logger).Log("msg", "server URL not configured")
		http.Error(w, "Server URL not configured", http.StatusNotFound)
		return
	}

	// Build IdP
	idp, err := s.buildIdentityProvider(ctx, serverURL)
	if err != nil {
		if fleet.IsNotFound(err) {
			level.Error(s.logger).Log("msg", "IdP certificate or key not found", "err", err)
			http.Error(w, "IdP not configured", http.StatusNotFound)
			return
		}
		handleInternalServerError(ctx, w, s.logger, "failed to build identity provider", err)
		return
	}

	// ServeMetadata writes XML directly to the response
	idp.ServeMetadata(w, r)
}

// serveSSO handles POST /api/fleet/conditional_access/idp/sso
// Handles SAML AuthnRequest from Okta, verifies device certificate and health.
func (s *idpService) serveSSO(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	level.Info(s.logger).Log(
		"msg", "received SSO request",
		"method", r.Method,
		"remote_addr", r.RemoteAddr,
	)

	// Extract certificate serial number from header (set by load balancer)
	serialStr := r.Header.Get("X-Client-Cert-Serial")
	if serialStr == "" {
		level.Error(s.logger).Log("msg", "missing client certificate serial", "remote_addr", r.RemoteAddr)
		http.Redirect(w, r, certificateErrorURL, http.StatusSeeOther)
		return
	}

	// Parse serial number (hex string to uint64)
	serial, err := parseSerialNumber(serialStr)
	if err != nil {
		level.Error(s.logger).Log("msg", "invalid certificate serial format", "serial", serialStr, "err", err)
		http.Redirect(w, r, certificateErrorURL, http.StatusSeeOther)
		return
	}

	// Look up host by certificate serial number
	hostID, err := s.ds.GetConditionalAccessCertHostIDBySerialNumber(ctx, serial)
	if err != nil {
		if fleet.IsNotFound(err) {
			level.Error(s.logger).Log("msg", "certificate not recognized", "serial", serial, "err", err)
			http.Redirect(w, r, certificateErrorURL, http.StatusSeeOther)
			return
		}
		handleInternalServerError(ctx, w, s.logger, "failed to lookup host by certificate serial", err, "serial", serial)
		return
	}

	level.Debug(s.logger).Log("msg", "found host for certificate", "host_id", hostID, "serial", serial)

	// Load AppConfig for IdP configuration
	appConfig, err := s.ds.AppConfig(ctx)
	if err != nil {
		handleInternalServerError(ctx, w, s.logger, "failed to load app config", err)
		return
	}

	// Get Fleet server URL from config
	serverURL := appConfig.ServerSettings.ServerURL
	if serverURL == "" {
		handleInternalServerError(ctx, w, s.logger, "server URL not configured", errors.New("server URL not configured"))
		return
	}

	// Build IdP
	idp, err := s.buildIdentityProvider(ctx, serverURL)
	if err != nil {
		if fleet.IsNotFound(err) {
			level.Error(s.logger).Log("msg", "IdP certificate or key not found", "err", err)
			http.Redirect(w, r, certificateErrorURL, http.StatusSeeOther)
			return
		}
		handleInternalServerError(ctx, w, s.logger, "failed to build identity provider", err)
		return
	}

	// Set session provider to handle device health checks
	idp.SessionProvider = &deviceHealthSessionProvider{
		ds:     s.ds,
		logger: s.logger,
		hostID: hostID,
	}

	// ServeSSO handles SAML AuthnRequest parsing, generates assertion, and returns response
	level.Debug(s.logger).Log("msg", "calling SAML IdP ServeSSO", "host_id", hostID)
	idp.ServeSSO(w, r)
}

// parseSerialNumber parses a certificate serial number from hex string to uint64.
// The serial number is provided by the load balancer in the X-Client-Cert-Serial header.
//
// SECURITY NOTE: This function only supports certificate serial numbers up to uint64 max
// (18,446,744,073,709,551,615). While X.509 allows serial numbers up to 160 bits, this
// limitation is acceptable because Fleet controls the Certificate Authority and generates
// all certificates via SCEP
func parseSerialNumber(serialStr string) (uint64, error) {
	// Remove any colons or spaces that might be in the serial number
	serialStr = strings.ReplaceAll(serialStr, ":", "")
	serialStr = strings.ReplaceAll(serialStr, " ", "")

	// Parse as hex (base 16) to uint64
	serial, err := strconv.ParseUint(serialStr, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("parse serial number: %w", err)
	}

	return serial, nil
}

// extractNameID extracts the NameID (user identifier) from a SAML AuthnRequest.
// Okta sends the NameID in the request to identify which user is authenticating.
// Returns empty string if the NameID cannot be extracted.
func extractNameID(req *saml.IdpAuthnRequest) string {
	if req != nil && req.Request.Subject != nil && req.Request.Subject.NameID != nil {
		return req.Request.Subject.NameID.Value
	}
	return ""
}

// deviceHealthSessionProvider implements saml.SessionProvider interface to handle
// device health verification during SAML SSO flow.
type deviceHealthSessionProvider struct {
	ds     fleet.Datastore
	logger kitlog.Logger
	hostID uint
}

// GetSession is called by the SAML library to get session information for the SAML assertion.
// It performs device health checks and returns appropriate session data or error.
func (p *deviceHealthSessionProvider) GetSession(w http.ResponseWriter, r *http.Request, req *saml.IdpAuthnRequest) *saml.Session {
	ctx := r.Context()

	// Extract NameID (email/username) from the SAML AuthnRequest
	// Okta sends this to identify which user is authenticating
	nameID := extractNameID(req)

	level.Debug(p.logger).Log("msg", "processing SAML session", "host_id", p.hostID)

	// Load host to get team ID
	hostLite, err := p.ds.HostLite(ctx, p.hostID)
	if err != nil {
		if fleet.IsNotFound(err) {
			level.Error(p.logger).Log("msg", "host not found", "host_id", p.hostID, "err", err)
			http.Redirect(w, r, certificateErrorURL, http.StatusSeeOther)
			return nil
		}
		handleInternalServerError(ctx, w, p.logger, "failed to load host", err, "host_id", p.hostID)
		return nil
	}

	// Get policies configured for conditional access
	teamID := uint(0)
	if hostLite.TeamID != nil {
		teamID = *hostLite.TeamID
	}
	conditionalAccessPolicyIDs, err := p.ds.GetPoliciesForConditionalAccess(ctx, teamID)
	if err != nil {
		handleInternalServerError(ctx, w, p.logger, "failed to get conditional access policies", err, "host_id", p.hostID)
		return nil
	}

	// Create a set of conditional access policy IDs for fast lookup
	conditionalAccessPolicyIDsSet := make(map[uint]struct{}, len(conditionalAccessPolicyIDs))
	for _, policyID := range conditionalAccessPolicyIDs {
		conditionalAccessPolicyIDsSet[policyID] = struct{}{}
	}

	// Create a minimal Host for ListPoliciesForHost
	// Platform is required for policy filtering
	host := &fleet.Host{
		ID:       p.hostID,
		Platform: hostLite.Platform,
	}

	// Get all policies for the host
	policies, err := p.ds.ListPoliciesForHost(ctx, host)
	if err != nil {
		handleInternalServerError(ctx, w, p.logger, "failed to list policies for host", err, "host_id", p.hostID)
		return nil
	}

	// Check if device has failing conditional access policies
	failingConditionalAccessCount := 0
	for _, policy := range policies {
		// Only check policies that are marked for conditional access
		if _, isConditionalAccessPolicy := conditionalAccessPolicyIDsSet[policy.ID]; !isConditionalAccessPolicy {
			continue
		}
		// Check if policy is failing
		if policy.Response == policyResponseFail {
			failingConditionalAccessCount++
		}
	}

	if failingConditionalAccessCount > 0 {
		level.Debug(p.logger).Log(
			"msg", "device has failing conditional access policies",
			"host_id", p.hostID,
			"failing_conditional_access_policies_count", failingConditionalAccessCount,
		)
		http.Redirect(w, r, remediateURL, http.StatusSeeOther)
		return nil
	}

	// Device is compliant - return session for SAML assertion
	// The NameID must match what Okta sent in the AuthnRequest (typically user email)
	// If no NameID was provided in the request, fall back to host-based identifier
	if nameID == "" {
		nameID = fmt.Sprintf("host-%d", p.hostID)
		level.Debug(p.logger).Log("msg", "no NameID in request, using host-based identifier", "name_id", nameID)
	}

	level.Info(p.logger).Log(
		"msg", "device is compliant, generating SAML assertion",
		"host_id", p.hostID,
	)

	return &saml.Session{
		NameID: nameID,
	}
}

// oktaServiceProviderProvider implements saml.ServiceProviderProvider to provide
// Okta service provider metadata to the IdP.
type oktaServiceProviderProvider struct {
	ds     fleet.Datastore
	logger kitlog.Logger
}

// GetServiceProvider returns the Okta service provider metadata.
// The serviceProviderID parameter is the entityID from the SAML AuthnRequest,
// which should match the Okta Audience URI from the configuration.
func (p *oktaServiceProviderProvider) GetServiceProvider(r *http.Request, serviceProviderID string) (*saml.EntityDescriptor, error) {
	ctx := r.Context()

	// Load AppConfig to get Okta settings
	appConfig, err := p.ds.AppConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load app config: %w", err)
	}

	// Validate Okta configuration exists
	if appConfig.ConditionalAccess == nil ||
		appConfig.ConditionalAccess.OktaAssertionConsumerServiceURL.Value == "" ||
		appConfig.ConditionalAccess.OktaAudienceURI.Value == "" {
		return nil, os.ErrNotExist
	}

	// Check if the requested service provider ID (entityID) matches our configured Okta Audience URI
	if serviceProviderID != appConfig.ConditionalAccess.OktaAudienceURI.Value {
		level.Debug(p.logger).Log("msg", "service provider ID mismatch",
			"requested", serviceProviderID,
			"configured", appConfig.ConditionalAccess.OktaAudienceURI.Value)
		return nil, os.ErrNotExist
	}

	// Build EntityDescriptor for Okta service provider
	acsURL, err := url.Parse(appConfig.ConditionalAccess.OktaAssertionConsumerServiceURL.Value)
	if err != nil {
		return nil, fmt.Errorf("parse assertion consumer service URL: %w", err)
	}

	descriptor := saml.SPSSODescriptor{
		AssertionConsumerServices: []saml.IndexedEndpoint{
			{
				Binding:  samlBindingHTTPPost,
				Location: acsURL.String(),
				Index:    0,
			},
		},
	}

	// Parse Okta's certificate if provided (for validating signed AuthnRequests)
	if appConfig.ConditionalAccess.OktaCertificate.Value != "" {
		oktaCert, err := parseCertificateBytes([]byte(appConfig.ConditionalAccess.OktaCertificate.Value))
		if err != nil {
			return nil, fmt.Errorf("parse okta certificate: %w", err)
		}

		descriptor.SSODescriptor.RoleDescriptor.KeyDescriptors = []saml.KeyDescriptor{
			{
				Use: samlKeyUseSigning,
				KeyInfo: saml.KeyInfo{
					X509Data: saml.X509Data{
						X509Certificates: []saml.X509Certificate{
							{Data: base64.StdEncoding.EncodeToString(oktaCert.Raw)},
						},
					},
				},
			},
		}
	}

	entityDescriptor := &saml.EntityDescriptor{
		EntityID:         appConfig.ConditionalAccess.OktaAudienceURI.Value,
		SPSSODescriptors: []saml.SPSSODescriptor{descriptor},
	}

	return entityDescriptor, nil
}

// buildIdentityProvider creates a SAML IdentityProvider using the Fleet server URL.
func (s *idpService) buildIdentityProvider(ctx context.Context, serverURL string) (*saml.IdentityProvider, error) {
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
		// Return NotFoundError so it can be properly handled as a configuration issue
		// (redirect to error page) rather than an infrastructure error (500)
		return nil, &notFoundError{msg: "conditional access idp certificate or key not found in mdm_config_assets"}
	}

	// Parse certificate and key
	cert, key, err := parseCertAndKeyBytes(certAsset.Value, keyAsset.Value)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parse okta idp certificate")
	}

	// Build metadata URL
	metadataURL, err := url.Parse(serverURL)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parse server URL for metadata")
	}
	metadataURL = metadataURL.JoinPath(idpMetadataPath)

	// Build SSO URL (uses okta.* subdomain or dev override)
	ssoServerURL, err := s.buildSSOServerURL(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build SSO server URL")
	}
	ssoURL, err := url.Parse(ssoServerURL)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parse SSO server URL")
	}
	ssoURL = ssoURL.JoinPath(idpSSOPath)

	// Create kitlog adapter for SAML library
	samlLogger := &kitlogAdapter{logger: kitlog.With(s.logger, "component", "saml-idp")}

	// Build IdentityProvider
	// Note: SessionProvider is set dynamically in serveSSO based on the authenticated device
	idp := &saml.IdentityProvider{
		Key:                     key,
		SignatureMethod:         dsig.RSASHA256SignatureMethod,
		Logger:                  samlLogger,
		Certificate:             cert,
		MetadataURL:             *metadataURL,
		SSOURL:                  *ssoURL,
		ServiceProviderProvider: &oktaServiceProviderProvider{ds: s.ds, logger: s.logger},
	}

	return idp, nil
}

// buildSSOServerURL builds the SSO server base URL from the app config.
// It delegates to AppConfig.ConditionalAccessIdPSSOURL() for the URL construction logic.
func (s *idpService) buildSSOServerURL(ctx context.Context) (string, error) {
	// Load app config
	appConfig, err := s.ds.AppConfig(ctx)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "load app config")
	}

	// Use the AppConfig method to build the SSO URL
	ssoURL, err := appConfig.ConditionalAccessIdPSSOURL(os.Getenv)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "build conditional access SSO URL")
	}

	return ssoURL, nil
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

// parseCertificateBytes parses a PEM-encoded certificate.
func parseCertificateBytes(certPEM []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("failed to decode certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}
	return cert, nil
}
