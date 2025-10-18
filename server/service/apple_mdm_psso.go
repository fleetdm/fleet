package service

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log/level"
)

func (svc *Service) CheckMDMAppleEnrollmentWithPSSO(ctx context.Context, m *fleet.MDMAppleMachineInfo) (*fleet.MDMApplePSSORequired, error) {
	// // TODO(pssopoc): confirm how we want to handle authz here
	// skipauth: The enroll profile endpoint is unauthenticated.
	svc.authz.SkipAuthorization(ctx)

	if m == nil {
		// TODO(pssopoc): do we instead want to always fail here if we don't have machine info?
		level.Debug(svc.logger).Log("msg", "no machine info, skipping psso check")
		return nil, nil
	}

	level.Debug(svc.logger).Log("msg", "checking psso", "serial", m.Serial, "current_version", m.OSVersion)

	if !m.MDMCanRequestPSSOConfig {
		level.Debug(svc.logger).Log("msg", "mdm cannot request psso config, skipping psso check", "serial", m.Serial)
		return nil, nil
	}

	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "psso: fetching app config")
	}
	serverURL := strings.TrimSuffix(appCfg.ServerSettings.ServerURL, "/") // TODO(pssopoc): confirm how this should work with general server url prefix setting and/or custom MDM URL

	return &fleet.MDMApplePSSORequired{
		Code: fleet.MDMApplePSSORequiredCode,
		Details: fleet.MDMApplePSSORequiredDetails{
			AuthURL:    "https://login.microsoft.com/YOUR-TENANT-UUID", // TODO(pssopoc): make configurable and replace with real idp url
			ProfileURL: serverURL + "/api/latest/fleet/mdm/apple/psso_installer/profile",
			Package: fleet.MDMApplePSSORequiredPackage{
				ManifestURL: serverURL + "/api/latest/fleet/mdm/apple/psso_installer/manifest",
			},
		},
	}, nil
}

type mdmApplePSSOManifestResponse struct {
	Manifest string `json:"manifest"`
	Err      error  `json:"error,omitempty"`
}

func (r mdmApplePSSOManifestResponse) Error() error { return r.Err }

func (r mdmApplePSSOManifestResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(r.Manifest)), 10))
	w.Header().Set("Content-Type", "application/xml")

	// OK to just log the error here as writing anything on
	// `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the
	// header provided.
	if n, err := w.Write([]byte(r.Manifest)); err != nil {
		logging.WithExtras(ctx, "err", err, "written", n)
	}
}

func mdmApplePSSOManifestEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	m, err := svc.GetMDMApplePSSOManifest(ctx, request) // TODO(pssopoc): replace with real service method, for PoC just call this to enforce the skipauth
	if err != nil {
		return mdmApplePSSOManifestResponse{Err: err}, nil
	}

	return mdmApplePSSOManifestResponse{Manifest: m}, nil
}

func (svc *Service) GetMDMApplePSSOManifest(ctx context.Context, request interface{}) (string, error) {
	// // TODO(pssopoc): replace with real service method, for PoC just call this to enforce the skipauth
	// skipauth: proof of concept endpoint to serve the company portal installer
	svc.authz.SkipAuthorization(ctx)

	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "fetching app config")
	}
	serverURL := strings.TrimSuffix(ac.ServerSettings.ServerURL, "/") // TODO(pssopoc): confirm how this should work with general server url prefix setting and/or custom MDM URL

	level.Debug(svc.logger).Log("msg", "generating psso manifest", "server_url", serverURL)

	return generatePSSOManifest(companyPortalHash, serverURL+"/api/latest/fleet/mdm/apple/psso_installer"), nil
}

type mdmApplePSSOProfileResponse struct {
	Profile string `json:"profile"`
	Err     error  `json:"error,omitempty"`
}

func (r mdmApplePSSOProfileResponse) Error() error { return r.Err }

func (r mdmApplePSSOProfileResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(r.Profile)), 10))
	w.Header().Set("Content-Type", "application/x-apple-aspen-config")

	// OK to just log the error here as writing anything on
	// `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the
	// header provided.
	if n, err := w.Write([]byte(r.Profile)); err != nil {
		logging.WithExtras(ctx, "err", err, "written", n)
	}
}

func mdmApplePSSOProfileEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	p, err := svc.GetMDMApplePSSOProfile(ctx, request) // TODO(pssopoc): replace with real service method, for PoC just call this to enforce the skipauth
	if err != nil {
		return mdmApplePSSOProfileResponse{Err: err}, nil
	}

	return mdmApplePSSOProfileResponse{Profile: p}, nil
}

func (svc *Service) GetMDMApplePSSOProfile(ctx context.Context, request interface{}) (string, error) {
	// // TODO(pssopoc): replace with real service method, for PoC just call this to enforce the skipauth
	// skipauth: proof of concept endpoint to serve the company portal installer
	svc.authz.SkipAuthorization(ctx)

	level.Debug(svc.logger).Log("msg", "serving psso profile")

	// TODO(pssopoc): replace with a method to generate the profile based on admin-configured values
	return pssoProfile, nil
}

func mdmApplePSSOInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	installer, name, err := svc.GetMDMApplePSSOInstaller(ctx, request) // TODO(pssopoc): replace with real service method, for PoC just to call this to enforce the skipauth
	if err != nil {
		return mdmAppleGetInstallerResponse{Err: err}, nil
	}

	return mdmAppleGetInstallerResponse{
		head:      false,
		name:      name,
		size:      int64(len(installer)),
		installer: installer,
	}, nil
}

func (svc *Service) GetMDMApplePSSOInstaller(ctx context.Context, request interface{}) ([]byte, string, error) {
	// // TODO(pssopoc): replace with real service method, for PoC just call this to enforce the skipauth
	// skipauth: proof of concept endpoint to serve the company portal installer
	svc.authz.SkipAuthorization(ctx)
	return CompanyPortal, "CompanyPortal-Installer.pkg", nil
}

// // TODO(pssopoc): replace with real service method that allows for uploading the desired PSSO app, for PoC just to call this to enforce the skipauth
// Embed the company portal app for PoC
//
//go:embed testdata/software-installers/CompanyPortal-Installer.pkg
var CompanyPortal []byte

const companyPortalHash = "2cf89bb6c2af88b0ed44defdcf21a9c023fa6948e4da61e16bec71545a46c329" // TODO(pssopoc): replace with method to obtain hash of the uploaded app

// TODO(pssopoc): replace with a method to retrieve a pre-computed manifest, stored when the
// installer is uploaded, probably using existing `createManifest` as a starting point
func generatePSSOManifest(hash string, url string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>items</key>
  <array>
    <dict>
      <key>assets</key>
      <array>
        <dict>
          <key>kind</key>
          <string>software-package</string>
          <key>sha256-size</key>
          <integer>32</integer>
          <key>sha256s</key>
          <array>
            <string>%s</string>
          </array>
          <key>url</key>
          <string>%s</string>
        </dict>
      </array>
    </dict>
  </array>
</dict>
</plist>`, hash, url)
}

// TODO(pssopoc): replace this with a method to generate the profile based on admin-configured values
const pssoProfile = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>ExtensionIdentifier</key>
			<string>com.microsoft.CompanyPortalMac.ssoextension</string>
			<key>PayloadDisplayName</key>
			<string>Extensible Single Sign-On</string>
			<key>PayloadIdentifier</key>
			<string>com.apple.extensiblesso.4D68D4CF-1250-4FF4-AFFB-1176DB539C49</string>
			<key>PayloadType</key>
			<string>com.apple.extensiblesso</string>
			<key>PayloadUUID</key>
			<string>4D68D4CF-1250-4FF4-AFFB-1176DB539C49</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
			<key>PlatformSSO</key>
			<dict>
				<key>AuthenticationMethod</key>
				<string>Password</string>
				<key>TokenToUserMapping</key>
				<dict>
					<key>AccountName</key>
					<string>preferred_username</string>
					<key>FullName</key>
					<string>name</string>
				</dict>
				<key>UseSharedDeviceKeys</key>
				<true/>
				<key>EnableRegistrationDuringSetup</key>
				<true/>
				<key>EnableCreateFirstUserDuringSetup</key>
				<true/>
			</dict>
			<key>RegistrationToken</key>
			<string>{{DEVICEREGISTRATION}}</string>
			<key>ScreenLockedBehavior</key>
			<string>DoNotHandle</string>
			<key>TeamIdentifier</key>
			<string>UBF8T346G9</string>
			<key>Type</key>
			<string>Redirect</string>
			<key>URLs</key>
			<array>
				<string>https://login.microsoftonline.com</string>
				<string>https://login.microsoft.com</string>
				<string>https://sts.windows.net</string>
				<string>https://login-us.microsoftonline.com</string>
			</array>
		</dict>
	</array>
	<key>PayloadDisplayName</key>
	<string>PlatformSSO</string>
	<key>PayloadIdentifier</key>
	<string>com.fleetdm.platformsso.652B07D0-2E08-45CE-9423-1FCAFFAEC390</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>652B07D0-2E08-45CE-9423-1FCAFFAEC390</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>`

// // TODO(pssopoc): PoC currently assumes that the AuthURL is hosted by the IdP rather than Fleet, but
// // In the Apple docs example, the request is to `Host: idp.example.com`, but the flow is ambiguous.
// //
// // They say the device creates an ASWebAuthenticationSession using AuthURL and a callback scheme
// // that it sets to apple-remotemanagement-user-login (step 10). This starts an authentication flow
// // with the organization’s identity provider. But then they say the ASWebAuthenticationSession web
// // flow completes when the device management service returns an HTTP 308 permanent redirect
// // response to the device.
// //
// // This is a very rough sketch of how we might try to implement a Fleet-hosted AuthURL similar to
// // OTA or account-driven enrollment where Fleet intermediates so that it can populate the
// // apple-remotemanagement-user-login callback scheme. It would encompass the following steps:
// // that initiates the following steps:
// // https://developer.apple.com/documentation/devicemanagement/implementing-platform-sso-during-device-enrollment#Authenticate-the-user
// // https://developer.apple.com/documentation/devicemanagement/implementing-platform-sso-during-device-enrollment#Process-the-user-authentication-result
// func ServePSSOAuth(
// 	svc fleet.Service,
// 	urlPrefix string,
// 	ds fleet.Datastore,
// 	logger log.Logger,
// ) http.Handler {
// 	herr := func(w http.ResponseWriter, err string) {
// 		logger.Log("err", err)
// 		http.Error(w, err, http.StatusInternalServerError)
// 	}

// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		endpoint_utils.WriteBrowserSecurityHeaders(w)

// 		// TODO(pssopoc): validate things like required Fleet setup, valid enroll secret, etc

// 		// TODO(pssopoc): initiate IdP auth if needed, for PoC we skip this and assume auth is always successful

// 		// TODO(pssopoc): if we get here, IdP SSO authentication is either not required, or has
// 		// been successfully completed (e.g., we have received the IdP access token by some means TBD)
// 		w.Header().Set("Location", "apple-remotemanagement-user-login://authentication-results?access-token=dXNlci1pZGVudGl0eQ")
// 		w.WriteHeader(http.StatusPermanentRedirect)
// 	})
// }
