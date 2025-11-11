package service

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"text/template"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
)

const appleProfileTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
        <!-- Trusted CA Certificate -->
        <dict>
            <key>PayloadCertificateFileName</key>
            <string>conditional_access_ca.der</string>
            <key>PayloadContent</key>
            <data>{{.CACertBase64}}</data>
            <key>PayloadDescription</key>
            <string>Fleet conditional access CA certificate</string>
            <key>PayloadDisplayName</key>
            <string>Fleet conditional access CA</string>
            <key>PayloadIdentifier</key>
            <string>com.fleetdm.conditional-access-ca</string>
            <key>PayloadType</key>
            <string>com.apple.security.root</string>
            <key>PayloadUUID</key>
            <string>{{.CACertUUID}}</string>
            <key>PayloadVersion</key>
            <integer>1</integer>
        </dict>
		<!-- SCEP Configuration -->
		<dict>
			<key>PayloadContent</key>
			<dict>
				<key>URL</key>
				<string>{{.SCEPURL}}</string>
				<key>Challenge</key>
				<string>{{.Challenge}}</string>
				<key>Keysize</key>
				<integer>2048</integer>
				<key>Key Type</key>
				<string>RSA</string>
				<key>Key Usage</key>
				<integer>5</integer>
                <key>ExtendedKeyUsage</key>
                <array>
                    <string>1.3.6.1.5.5.7.3.2</string>
                </array>
				<key>Subject</key>
				<array>
					<array>
						<array>
							<string>CN</string>
							<string>{{.CertificateCN}}</string>
						</array>
					</array>
				</array>
				<key>SubjectAltName</key>
				<dict>
					<key>uniformResourceIdentifier</key>
					<array>
						<string>urn:device:apple:uuid:%HardwareUUID%</string>
					</array>
				</dict>
				<key>Retries</key>
				<integer>3</integer>
				<key>RetryDelay</key>
				<integer>10</integer>
                <!-- ACL for browser access -->
                <key>AllowAllAppsAccess</key>
                <true/>
                <!-- Set true for Safari access. Set false if Safari support not needed. -->
                <key>KeyIsExtractable</key>
                <false/>
			</dict>
			<key>PayloadDescription</key>
			<string>Configures SCEP for Fleet conditional access for Okta certificate</string>
			<key>PayloadDisplayName</key>
			<string>Fleet conditional access SCEP</string>
			<key>PayloadIdentifier</key>
			<string>com.fleetdm.conditional-access-scep</string>
			<key>PayloadType</key>
			<string>com.apple.security.scep</string>
			<key>PayloadUUID</key>
			<string>{{.SCEPPayloadUUID}}</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
		</dict>
        <!-- Identity Preference for mTLS endpoint -->
        <dict>
            <key>Name</key>
            <string>{{.MTLSURL}}</string>
            <key>PayloadCertificateUUID</key>
            <string>{{.SCEPPayloadUUID}}</string>
            <key>PayloadDescription</key>
            <string>Identity preference for mTLS endpoints</string>
            <key>PayloadDisplayName</key>
            <string>Fleet mTLS identity preference</string>
            <key>PayloadIdentifier</key>
            <string>com.fleetdm.conditional-access-preference</string>
            <key>PayloadType</key>
            <string>com.apple.security.identitypreference</string>
            <key>PayloadUUID</key>
            <string>{{.IdentityPrefUUID}}</string>
            <key>PayloadVersion</key>
            <integer>1</integer>
        </dict>
        <dict>
            <key>PayloadType</key>
            <string>com.apple.ManagedClient.preferences</string>
            <key>PayloadVersion</key>
            <integer>1</integer>
            <key>PayloadIdentifier</key>
            <string>com.fleetdm.chrome.certs</string>
            <key>PayloadUUID</key>
            <string>{{.ChromeConfigUUID}}</string>
            <key>PayloadDisplayName</key>
            <string>Chrome mTLS auto-select</string>
            <key>PayloadContent</key>
            <dict>
                <key>com.google.Chrome</key>
                <dict>
                    <key>Forced</key>
                    <array>
                        <dict>
                            <key>mcx_preference_settings</key>
                            <dict>
                                <key>AllowPolicyInIncognito</key>
                                <true/>
                                <key>AutoSelectCertificateForUrls</key>
                                <array>
                                    <!-- MUST be stringified JSON -->
                                    <string>{"pattern":"{{.MTLSURL}}","filter":{"SUBJECT":{"CN":"{{.CertificateCN}}"}}}</string>
                                </array>
                            </dict>
                        </dict>
                    </array>
                </dict>
            </dict>
        </dict>
	</array>
	<key>PayloadDescription</key>
	<string>Configures SCEP enrollment for Okta conditional access</string>
	<key>PayloadDisplayName</key>
	<string>Fleet conditional access for Okta</string>
	<key>PayloadIdentifier</key>
	<string>com.fleetdm.conditional-access-okta</string>
	<key>PayloadOrganization</key>
	<string>Fleet Device Management</string>
	<key>PayloadRemovalDisallowed</key>
	<false/>
	<key>PayloadScope</key>
	<string>User</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>{{.RootPayloadUUID}}</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>
`

var appleProfileTemplateParsed = template.Must(template.New("appleProfile").Parse(appleProfileTemplate))

// fleetConditionalAccessNamespace is a custom UUID namespace for Fleet Okta conditional access profiles.
// Generated using: uuid.NewSHA1(uuid.NameSpaceURL, []byte("https://fleetdm.com/learn-more-about/okta-conditional-access"))
// This ensures UUIDs are unique to Fleet's Okta conditional access feature and won't collide with other systems.
var fleetConditionalAccessNamespace = uuid.Must(uuid.Parse("fe5c0046-e83e-5a1d-9693-ace1348d34ec"))

// generateDeterministicUUID generates a UUID v5 based on the server URL and a component name.
// This ensures the same server always generates the same UUIDs for profile components.
func generateDeterministicUUID(serverURL, component string) string {
	// Use Fleet's conditional access namespace to avoid collisions
	// Create a deterministic UUID based on serverURL + component
	name := fmt.Sprintf("%s:%s", serverURL, component)
	return uuid.NewSHA1(fleetConditionalAccessNamespace, []byte(name)).String()
}

type appleProfileTemplateData struct {
	CACertBase64     string
	SCEPURL          string
	Challenge        string
	CertificateCN    string
	MTLSURL          string
	CACertUUID       string
	SCEPPayloadUUID  string
	IdentityPrefUUID string
	ChromeConfigUUID string
	RootPayloadUUID  string
}

type conditionalAccessGetIdPSigningCertRequest struct{}

type conditionalAccessGetIdPSigningCertResponse struct {
	CertPEM []byte
	Err     error `json:"error,omitempty"`
}

func (r conditionalAccessGetIdPSigningCertResponse) Error() error { return r.Err }

func (r conditionalAccessGetIdPSigningCertResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(r.CertPEM)), 10))
	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", "attachment; filename=\"fleet-idp-signing-cert.pem\"")

	// OK to just log the error here as writing anything on `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the header provided
	n, err := w.Write(r.CertPEM)
	if err != nil {
		logging.WithExtras(ctx, "err", err, "bytes_written", n)
	}
}

func conditionalAccessGetIdPSigningCertEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	certPEM, err := svc.ConditionalAccessGetIdPSigningCert(ctx)
	if err != nil {
		return conditionalAccessGetIdPSigningCertResponse{Err: err}, nil
	}
	return conditionalAccessGetIdPSigningCertResponse{
		CertPEM: certPEM,
	}, nil
}

func (svc *Service) ConditionalAccessGetIdPSigningCert(ctx context.Context) (certPEM []byte, err error) {
	// Check user is authorized to read conditional access Okta IdP certificate
	if err := svc.authz.Authorize(ctx, &fleet.ConditionalAccessIDPAssets{}, fleet.ActionRead); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to authorize")
	}

	// Load IdP certificate from mdm_config_assets
	assets, err := svc.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{
		fleet.MDMAssetConditionalAccessIDPCert,
	}, nil)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to load IdP certificate")
	}

	certAsset, ok := assets[fleet.MDMAssetConditionalAccessIDPCert]
	if !ok {
		return nil, ctxerr.New(ctx, "IdP certificate not configured")
	}

	return certAsset.Value, nil
}

type conditionalAccessGetIdPAppleProfileRequest struct{}

type conditionalAccessGetIdPAppleProfileResponse struct {
	ProfileData []byte
	Err         error `json:"error,omitempty"`
}

func (r conditionalAccessGetIdPAppleProfileResponse) Error() error { return r.Err }

func (r conditionalAccessGetIdPAppleProfileResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(r.ProfileData)), 10))
	w.Header().Set("Content-Type", "application/x-apple-aspen-config")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", "attachment; filename=\"fleet-conditional-access.mobileconfig\"")

	// OK to just log the error here as writing anything on `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the header provided
	n, err := w.Write(r.ProfileData)
	if err != nil {
		logging.WithExtras(ctx, "err", err, "bytes_written", n)
	}
}

func conditionalAccessGetIdPAppleProfileEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	profileData, err := svc.ConditionalAccessGetIdPAppleProfile(ctx)
	if err != nil {
		return conditionalAccessGetIdPAppleProfileResponse{Err: err}, nil
	}
	return conditionalAccessGetIdPAppleProfileResponse{
		ProfileData: profileData,
	}, nil
}

func (svc *Service) ConditionalAccessGetIdPAppleProfile(ctx context.Context) (profileData []byte, err error) {
	// Check user is authorized to read conditional access Apple profile
	if err := svc.authz.Authorize(ctx, &fleet.ConditionalAccessIDPAssets{}, fleet.ActionRead); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to authorize")
	}

	// Load CA certificate for SCEP from mdm_config_assets
	assets, err := svc.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{
		fleet.MDMAssetConditionalAccessCACert,
	}, nil)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to load conditional access CA certificate")
	}

	caCertAsset, ok := assets[fleet.MDMAssetConditionalAccessCACert]
	if !ok {
		return nil, ctxerr.New(ctx, "conditional access CA certificate not configured")
	}

	// Parse PEM certificate
	block, _ := pem.Decode(caCertAsset.Value)
	if block == nil {
		return nil, ctxerr.New(ctx, "failed to decode CA certificate PEM")
	}

	// Parse DER certificate
	_, err = x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to parse CA certificate")
	}

	// Base64 encode the DER certificate for the profile
	caCertBase64 := base64.StdEncoding.EncodeToString(block.Bytes)

	// Get app config for server URL
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to load app config")
	}

	// Construct SCEP URL
	scepURL := fmt.Sprintf("%s/api/fleet/conditional_access/scep", appConfig.ServerSettings.ServerURL)

	// Get global enroll secrets
	secrets, err := svc.ds.GetEnrollSecrets(ctx, nil)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to get enroll secrets")
	}
	if len(secrets) == 0 {
		return nil, ctxerr.Wrap(ctx, newNotFoundError(), "enroll_secret")
	}

	// Use the first global enroll secret as the challenge
	challenge := secrets[0].Secret

	// Get mTLS URL using ConditionalAccessIdPSSOURL
	mtlsURL, err := appConfig.ConditionalAccessIdPSSOURL(os.Getenv)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to get mTLS URL")
	}

	// Generate deterministic UUIDs based on server URL
	serverURL := appConfig.ServerSettings.ServerURL
	caCertUUID := generateDeterministicUUID(serverURL, "conditional-access-ca-cert")
	scepPayloadUUID := generateDeterministicUUID(serverURL, "conditional-access-scep")
	identityPrefUUID := generateDeterministicUUID(serverURL, "conditional-access-identity-pref")
	chromeConfigUUID := generateDeterministicUUID(serverURL, "conditional-access-chrome-config")
	rootPayloadUUID := generateDeterministicUUID(serverURL, "conditional-access-root-payload")

	// Execute template
	var buf bytes.Buffer
	if err := appleProfileTemplateParsed.Execute(&buf, appleProfileTemplateData{
		CACertBase64:     caCertBase64,
		SCEPURL:          scepURL,
		Challenge:        challenge,
		CertificateCN:    "Fleet conditional access for Okta",
		MTLSURL:          mtlsURL,
		CACertUUID:       caCertUUID,
		SCEPPayloadUUID:  scepPayloadUUID,
		IdentityPrefUUID: identityPrefUUID,
		ChromeConfigUUID: chromeConfigUUID,
		RootPayloadUUID:  rootPayloadUUID,
	}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to execute profile template")
	}

	return buf.Bytes(), nil
}
