package service

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	stdlog "log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	configpkg "github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/scep/scep_mysql"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/uuid"
	"github.com/micromdm/micromdm/mdm/appmanifest"
	"github.com/micromdm/nanodep/client"
	"github.com/micromdm/nanodep/godep"
	nanodep_stdlogfmt "github.com/micromdm/nanodep/log/stdlogfmt"
	"github.com/micromdm/nanodep/proxy"
	nanodep_mysql "github.com/micromdm/nanodep/storage/mysql"
	depsync "github.com/micromdm/nanodep/sync"
	"github.com/micromdm/nanomdm/certverify"
	nanomdm_httpapi "github.com/micromdm/nanomdm/http/api"
	httpmdm "github.com/micromdm/nanomdm/http/mdm"
	nanomdm_stdlogfmt "github.com/micromdm/nanomdm/log/stdlogfmt"
	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/push/buford"
	nanomdm_pushsvc "github.com/micromdm/nanomdm/push/service"
	nanomdm_service "github.com/micromdm/nanomdm/service"
	"github.com/micromdm/nanomdm/service/certauth"
	"github.com/micromdm/nanomdm/service/nanomdm"
	scep_depot "github.com/micromdm/scep/v2/depot"
	scepserver "github.com/micromdm/scep/v2/server"
	_ "go.elastic.co/apm/module/apmsql"
	_ "go.elastic.co/apm/module/apmsql/mysql"
	"howett.net/plist"
)

type SetupConfig struct {
	MDMConfig    configpkg.MDMAppleConfig
	Logger       kitlog.Logger
	MDMStorage   *mysql.NanoMDMStorage
	SCEPStorage  *scep_mysql.MySQLDepot
	DEPStorage   *nanodep_mysql.MySQLStorage
	LoggingDebug bool
}

// Setup registers all MDM services and starts all MDM functionality.
// It registers all the services on the given mux.
func Setup(ctx context.Context, mux *http.ServeMux, config SetupConfig) error {
	if err := registerServices(ctx, mux, config); err != nil {
		return fmt.Errorf("register services: %w", err)
	}
	if err := startDEPRoutine(ctx, config); err != nil {
		return fmt.Errorf("start DEP routine: %w", err)
	}
	if err := startMunkiRepoServer(ctx, mux, config.MDMConfig.Munki); err != nil {
		return fmt.Errorf("start munki server: %w", err)
	}
	if err := startMunkiPkgServer(ctx, mux, config.MDMConfig.Munki.MunkiPkg, config.Logger); err != nil {
		return fmt.Errorf("start munki pkg server: %w", err)
	}
	return nil
}

// TODO(lucas): None of the API endpoints have authentication yet.
// We should use Fleet admin bearer token authentication.
func registerServices(ctx context.Context, mux *http.ServeMux, config SetupConfig) error {
	scepCACrt, err := registerSCEP(mux, config)
	if err != nil {
		return fmt.Errorf("SCEP: %w", err)
	}
	if err := registerMDM(mux, config, scepCACrt); err != nil {
		return fmt.Errorf("MDM: %w", err)
	}
	if err := registerEnroll(ctx, mux, config); err != nil {
		return fmt.Errorf("enroll endpoint: %w", err)
	}
	registerDEPProxy(mux, config)
	return nil
}

func registerSCEP(mux *http.ServeMux, config SetupConfig) (*x509.Certificate, error) {
	scepCAKeyPassphrase := []byte(config.MDMConfig.SCEP.CA.Passphrase)
	if len(scepCAKeyPassphrase) == 0 {
		return nil, errors.New("missing passphrase for SCEP CA private key")
	}
	scepCACrt, scepCAKey, err := config.SCEPStorage.LoadCA(scepCAKeyPassphrase)
	if err != nil {
		return nil, fmt.Errorf("load SCEP CA: %w", err)
	}
	var signer scepserver.CSRSigner = scep_depot.NewSigner(
		config.SCEPStorage,
		scep_depot.WithCAPass(string(scepCAKeyPassphrase)),
		scep_depot.WithValidityDays(config.MDMConfig.SCEP.Signer.ValidityDays),
		scep_depot.WithAllowRenewalDays(config.MDMConfig.SCEP.Signer.AllowRenewalDays),
	)
	scepChallenge := config.MDMConfig.SCEP.Challenge
	if scepChallenge == "" {
		return nil, errors.New("missing SCEP challenge")
	}
	signer = scepserver.ChallengeMiddleware(scepChallenge, signer)
	scepService, err := scepserver.NewService(scepCACrt, scepCAKey, signer,
		scepserver.WithLogger(kitlog.With(config.Logger, "component", "mdm-apple-scep")),
	)
	if err != nil {
		return nil, fmt.Errorf("initialize SCEP service: %w", err)
	}
	scepLogger := kitlog.With(config.Logger, "component", "http-mdm-apple-scep")
	e := scepserver.MakeServerEndpoints(scepService)
	e.GetEndpoint = scepserver.EndpointLoggingMiddleware(scepLogger)(e.GetEndpoint)
	e.PostEndpoint = scepserver.EndpointLoggingMiddleware(scepLogger)(e.PostEndpoint)
	scepHandler := scepserver.MakeHTTPHandler(e, scepService, scepLogger)
	mux.Handle("/mdm/apple/scep", scepHandler)
	return scepCACrt, nil
}

func registerMDM(mux *http.ServeMux, config SetupConfig, scepCACrt *x509.Certificate) error {
	const (
		endpointAPIPushCert = "/mdm/apple/mdm/api/v1/pushcert"
		endpointAPIPush     = "/mdm/apple/mdm/api/v1/push/"
		endpointAPIEnqueue  = "/mdm/apple/mdm/api/v1/enqueue/"
	)
	// TODO(lucas): Using bare minimum to run MDM in Fleet. Revisit
	// https://github.com/micromdm/nanomdm/blob/92c977e42859ba56e73d1fc2377732a9ab6e5e01/cmd/nanomdm/main.go
	// to allow for more configuration/options.
	scepCAPEMBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: scepCACrt.Raw,
	}
	scepCAPEM := pem.EncodeToMemory(scepCAPEMBlock)
	certVerifier, err := certverify.NewPoolVerifier(scepCAPEM, x509.ExtKeyUsageClientAuth)
	if err != nil {
		return fmt.Errorf("certificate pool verifier: %w", err)
	}
	mdmLogger := nanomdm_stdlogfmt.New(
		nanomdm_stdlogfmt.WithLogger(
			stdlog.New(
				kitlog.NewStdlibAdapter(
					kitlog.With(config.Logger, "component", "http-mdm-apple-mdm")),
				"", stdlog.LstdFlags,
			),
		),
		nanomdm_stdlogfmt.WithDebugFlag(config.LoggingDebug),
	)

	pushProviderFactory := buford.NewPushProviderFactory()
	pushService := nanomdm_pushsvc.New(config.MDMStorage, config.MDMStorage, pushProviderFactory, mdmLogger.With("service", "push"))
	pushCertHandler := nanomdm_httpapi.StorePushCertHandler(config.MDMStorage, mdmLogger.With("handler", "store-cert"))
	mux.Handle(endpointAPIPushCert, pushCertHandler)
	var pushHandler http.Handler
	pushHandler = nanomdm_httpapi.PushHandler(pushService, mdmLogger.With("handler", "push"))
	pushHandler = http.StripPrefix(endpointAPIPush, pushHandler)
	mux.Handle(endpointAPIPush, pushHandler)

	nanomdmService := nanomdm.New(config.MDMStorage, nanomdm.WithLogger(mdmLogger))
	var mdmService nanomdm_service.CheckinAndCommandService = nanomdmService
	mdmService = wrappedNanoMDMService{
		CheckinAndCommandService: mdmService,
		logger:                   kitlog.With(config.Logger, "component", "wrapped-nanomdm-service"),
		cmdPusher: bootstrapCommandPusher{
			mdmStorage:         config.MDMStorage,
			serverURL:          config.MDMConfig.Munki.MunkiPkg.ServerURL,
			munkiRepoBasicAuth: config.MDMConfig.Munki.HTTPBasicAuth,
			pushSvc:            pushService,
		},
	}
	mdmService = certauth.New(mdmService, config.MDMStorage)
	var mdmHandler http.Handler
	mdmHandler = httpmdm.CheckinAndCommandHandler(mdmService, mdmLogger.With("handler", "checkin-command"))
	mdmHandler = httpmdm.CertVerifyMiddleware(mdmHandler, certVerifier, mdmLogger.With("handler", "cert-verify"))
	mdmHandler = httpmdm.CertExtractMdmSignatureMiddleware(mdmHandler, mdmLogger.With("handler", "cert-extract"))
	mux.Handle("/mdm/apple/mdm", mdmHandler)

	var enqueueHandler http.Handler
	enqueueHandler = nanomdm_httpapi.RawCommandEnqueueHandler(config.MDMStorage, pushService, mdmLogger.With("handler", "enqueue"))
	enqueueHandler = http.StripPrefix(endpointAPIEnqueue, enqueueHandler)
	mux.Handle(endpointAPIEnqueue, enqueueHandler)
	return nil
}

type wrappedNanoMDMService struct {
	nanomdm_service.CheckinAndCommandService
	cmdPusher bootstrapCommandPusher
	logger    kitlog.Logger
}

type bootstrapCommandPusher struct {
	mdmStorage         *mysql.NanoMDMStorage
	serverURL          string
	munkiRepoBasicAuth configpkg.HTTPBasicAuthConfig
	pushSvc            *nanomdm_pushsvc.PushService
}

// TokenUpdate partially implements nanomdm_service.CheckinAndCommandService.
//
// TODO(lucas): Am using TokenUpdate as the indicator that the enrollment for a device is complete.
// Check if there's a better way to determine when a device fully enrolled.
//
// Once enrolled, we send two "bootstrap" commands to setup Munki on the device.
func (w wrappedNanoMDMService) TokenUpdate(r *mdm.Request, t *mdm.TokenUpdate) error {
	err := w.CheckinAndCommandService.TokenUpdate(r, t)
	if err != nil {
		return err
	}

	if err := w.cmdPusher.enqueueBootstrapCommands(r.Context, r.ID); err != nil {
		w.logger.Log("msg", "failed to enqueue bootstrap commands", "err", err)
	}
	return nil
}

func (c bootstrapCommandPusher) enqueueBootstrapCommands(ctx context.Context, deviceID string) error {
	// TODO(lucas): Does the Munki PKG server need to be instantiated sooner?
	munkiManifestURL, err := url.Parse(c.serverURL)
	if err != nil {
		return fmt.Errorf("parse server url: %w", err)
	}
	munkiManifestURL.Path += munkiManifestPath
	munkiManifestURL.Scheme = "https"

	installAppCommandUUID, err := uuid.NewRandom()
	if err != nil {
		return fmt.Errorf("failed to generate install app command uuid: %w", err)
	}

	installAppCmdPlist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>Command</key>
        <dict>
                <key>ManifestURL</key>
                <string>%s</string>
                <key>RequestType</key>
                <string>InstallApplication</string>
        </dict>
        <key>CommandUUID</key>
        <string>%s</string>
</dict>
</plist>`, munkiManifestURL, installAppCommandUUID)
	installAppCmd, err := mdm.DecodeCommand([]byte(installAppCmdPlist))
	if err != nil {
		return fmt.Errorf("failed to decode install app command: %w", err)
	}
	softwareRepoURL, err := url.Parse(c.serverURL)
	if err != nil {
		return fmt.Errorf("parse server url: %w", err)
	}
	softwareRepoURL.Path += strings.TrimSuffix(munkiRepoPath, "/")
	softwareRepoURL.Scheme = "https"

	// TODO(lucas): Currently HTTP basic auth is stored in `/Library/Preferences/ManagedInstalls`
	// which is world-readable. We should store under root preferences, see:
	// https://github.com/munki/munki/wiki/Using-Basic-Authentication#protecting-the-password-from-local-users
	munkiPreferencesMobileConfig := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>PayloadContent</key>
			<dict>
				<key>ManagedInstalls</key>
				<dict>
					<key>Forced</key>
					<array>
						<dict>
							<key>mcx_preference_settings</key>
							<dict>
								<key>AdditionalHttpHeaders</key>
								<array>
									<string>Authorization: Basic %s</string>
								</array>
								<key>AppleSoftwareUpdatesOnly</key>
								<false/>
								<key>ClientIdentifier</key>
								<string></string>
								<key>IgnoreSystemProxies</key>
								<false/>
								<key>InstallAppleSoftwareUpdates</key>
								<false/>
								<key>LogFile</key>
								<string>/Library/Managed Installs/Logs/ManagedSoftwareUpdate.log</string>
								<key>LogToSyslog</key>
								<false/>
								<key>LoggingLevel</key>
								<integer>1</integer>
								<key>ManagedInstallDir</key>
								<string>/Library/Managed Installs</string>
								<key>PackageVerificationMode</key>
								<string>hash</string>
								<key>SoftwareRepoURL</key>
								<string>%s</string>
								<key>SuppressStopButtonOnInstall</key>
								<false/>
							</dict>
						</dict>
					</array>
				</dict>
			</dict>
			<key>PayloadEnabled</key>
			<true/>
			<key>PayloadIdentifier</key>
			<string>MCXToProfile.67fdc08f-0f1b-4fdb-b203-4948b0b8574c.alacarte.customsettings.c21ebab8-b2be-459c-b6c8-c7908d88e2e7</string>
			<key>PayloadType</key>
			<string>com.apple.ManagedClient.preferences</string>
			<key>PayloadUUID</key>
			<string>c21ebab8-b2be-459c-b6c8-c7908d88e2e7</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
		</dict>
	</array>
	<key>PayloadDescription</key>
	<string>Included custom settings:
ManagedInstalls
</string>
	<key>PayloadDisplayName</key>
	<string>MCXToProfile: ManagedInstalls</string>
	<key>PayloadIdentifier</key>
	<string>MunkiPreferences</string>
	<key>PayloadOrganization</key>
	<string></string>
	<key>PayloadRemovalDisallowed</key>
	<true/>
	<key>PayloadScope</key>
	<string>System</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>67fdc08f-0f1b-4fdb-b203-4948b0b8574c</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>`, c.munkiRepoBasicAuth.Encoded(), softwareRepoURL.String())
	installProfileCommandUUID, err := uuid.NewRandom()
	if err != nil {
		return fmt.Errorf("failed to generate install profile command uuid: %w", err)
	}
	installProfileCmdPlist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>Command</key>
        <dict>
                <key>Payload</key>
                <data>
				%s
				</data>
                <key>RequestType</key>
                <string>InstallProfile</string>
        </dict>
        <key>CommandUUID</key>
        <string>%s</string>
</dict>
</plist>`, base64.StdEncoding.EncodeToString([]byte(munkiPreferencesMobileConfig)), installProfileCommandUUID)
	installProfileCmd, err := mdm.DecodeCommand([]byte(installProfileCmdPlist))
	if err != nil {
		return fmt.Errorf("failed to decode install profile command: %w", err)
	}

	if _, err := c.mdmStorage.EnqueueCommand(ctx, []string{deviceID}, installAppCmd); err != nil {
		return fmt.Errorf("failed to enqueue install app command: %w", err)
	}
	if _, err := c.mdmStorage.EnqueueCommand(ctx, []string{deviceID}, installProfileCmd); err != nil {
		return fmt.Errorf("failed to enqueue install profile command: %w", err)
	}

	if _, err := c.pushSvc.Push(ctx, []string{deviceID}); err != nil {
		return fmt.Errorf("failed to push device: %w", err)
	}
	return nil
}

func registerEnroll(ctx context.Context, mux *http.ServeMux, config SetupConfig) error {
	// TODO(lucas): The enroll profile must be protected by SSO. Currently the endpoint is unauthenticated.
	topic, err := config.MDMStorage.CurrentTopic(ctx)
	if err != nil {
		return fmt.Errorf("load push certificate topic: %w", err)
	}
	enrollLogger := kitlog.With(config.Logger, "handler", "enroll-profile")
	mux.HandleFunc("/mdm/apple/api/enroll", func(w http.ResponseWriter, r *http.Request) {
		mobileConfig, err := generateMobileConfig(
			"https://"+config.MDMConfig.DEP.ServerURL+"/mdm/apple/scep",
			"https://"+config.MDMConfig.DEP.ServerURL+"/mdm/apple/mdm",
			config.MDMConfig.SCEP.Challenge,
			topic,
		)
		if err != nil {
			enrollLogger.Log("err", err)
		}
		w.Header().Add("Content-Type", "application/x-apple-aspen-config")
		if _, err := w.Write(mobileConfig); err != nil {
			enrollLogger.Log("err", err)
		}
	})
	return nil
}

func registerDEPProxy(mux *http.ServeMux, config SetupConfig) {
	stdLogger := stdlog.New(
		kitlog.NewStdlibAdapter(
			kitlog.With(config.Logger, "component", "http-mdm-apple-dep")),
		"", stdlog.LstdFlags,
	)
	depLogger := nanodep_stdlogfmt.New(stdLogger, config.LoggingDebug)
	p := proxy.New(
		client.NewTransport(http.DefaultTransport, http.DefaultClient, config.DEPStorage, nil),
		config.DEPStorage,
		depLogger.With("component", "proxy"),
	)
	var proxyHandler http.Handler = proxy.ProxyDEPNameHandler(p, depLogger.With("handler", "proxy"))
	proxyHandler = http.StripPrefix("/mdm/apple/proxy/", proxyHandler)
	proxyHandler = delHeaderMiddleware(proxyHandler, "Authorization")
	mux.Handle("/mdm/apple/proxy/", proxyHandler)
}

func startDEPRoutine(ctx context.Context, config SetupConfig) error {
	stdLogger := stdlog.New(
		kitlog.NewStdlibAdapter(
			kitlog.With(config.Logger, "component", "mdm-apple-dep-routine")),
		"", stdlog.LstdFlags,
	)
	depLogger := nanodep_stdlogfmt.New(stdLogger, config.LoggingDebug)
	httpClient := fleethttp.NewClient()
	depClient := godep.NewClient(config.DEPStorage, httpClient)
	assignerOpts := []depsync.AssignerOption{
		depsync.WithAssignerLogger(depLogger.With("component", "assigner")),
	}
	if config.LoggingDebug {
		assignerOpts = append(assignerOpts, depsync.WithDebug())
	}
	assigner := depsync.NewAssigner(
		depClient,
		apple.DEPName,
		config.DEPStorage,
		assignerOpts...,
	)
	depSyncerCallback := func(ctx context.Context, isFetch bool, resp *godep.DeviceResponse) error {
		go func() {
			err := assigner.ProcessDeviceResponse(ctx, resp)
			if err != nil {
				depLogger.Info("msg", "assigner process device response", "err", err)
			}
		}()
		return nil
	}
	syncerLogger := depLogger.With("component", "syncer")
	// TODO(lucas): Expose syncNow.
	syncNow := make(chan struct{})
	syncerOpts := []depsync.SyncerOption{
		depsync.WithLogger(syncerLogger),
		depsync.WithSyncNow(syncNow),
		depsync.WithCallback(depSyncerCallback),
		depsync.WithDuration(config.MDMConfig.DEP.SyncPeriodicity),
		depsync.WithLimit(config.MDMConfig.DEP.SyncDeviceLimit),
	}
	syncer := depsync.NewSyncer(
		depClient,
		apple.DEPName,
		config.DEPStorage,
		syncerOpts...,
	)
	go func() {
		defer close(syncNow)

		if err := syncer.Run(ctx); err != nil {
			syncerLogger.Info("msg", "syncer run", "err", err)
		}
	}()
	return nil
}

// delHeaderMiddleware deletes header from the HTTP request headers before calling h.
func delHeaderMiddleware(h http.Handler, header string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del(header)
		h.ServeHTTP(w, r)
	}
}

// mobileConfigTemplate is the template Fleet uses to assemble a .mobileconfig enroll profile to serve to devices.
//
// TODO(lucas): Tweak the remaining configuration.
// Downloaded from:
// https://github.com/micromdm/nanomdm/blob/3b1eb0e4e6538b6644633b18dedc6d8645853cb9/docs/enroll.mobileconfig
//
// TODO(lucas): Support enroll profile signing?
var mobileConfigTemplate = template.Must(template.New(".mobileconfig").Parse(`
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>PayloadContent</key>
			<dict>
				<key>Key Type</key>
				<string>RSA</string>
				<key>Challenge</key>
				<string>{{ .SCEPChallenge }}</string>
				<key>Key Usage</key>
				<integer>5</integer>
				<key>Keysize</key>
				<integer>2048</integer>
				<key>URL</key>
				<string>{{ .SCEPServerURL }}</string>
			</dict>
			<key>PayloadIdentifier</key>
			<string>com.github.micromdm.scep</string>
			<key>PayloadType</key>
			<string>com.apple.security.scep</string>
			<key>PayloadUUID</key>
			<string>CB90E976-AD44-4B69-8108-8095E6260978</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
		</dict>
		<dict>
			<key>AccessRights</key>
			<integer>8191</integer>
			<key>CheckOutWhenRemoved</key>
			<true/>
			<key>IdentityCertificateUUID</key>
			<string>CB90E976-AD44-4B69-8108-8095E6260978</string>
			<key>PayloadIdentifier</key>
			<string>com.github.micromdm.nanomdm.mdm</string>
			<key>PayloadType</key>
			<string>com.apple.mdm</string>
			<key>PayloadUUID</key>
			<string>96B11019-B54C-49DC-9480-43525834DE7B</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
			<key>ServerCapabilities</key>
			<array>
				<string>com.apple.mdm.per-user-connections</string>
			</array>
			<key>ServerURL</key>
			<string>{{ .MDMServerURL }}</string>
			<key>SignMessage</key>
			<true/>
			<key>Topic</key>
			<string>{{ .Topic }}</string>
		</dict>
	</array>
	<key>PayloadDisplayName</key>
	<string>Enrollment Profile</string>
	<key>PayloadIdentifier</key>
	<string>com.github.micromdm.nanomdm</string>
	<key>PayloadScope</key>
	<string>System</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>F9760DD4-F2D1-4F29-8D2C-48D52DD0A9B3</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>`))

func generateMobileConfig(scepServerURL, mdmServerURL, scepChallenge, topic string) ([]byte, error) {
	var contents bytes.Buffer
	if err := mobileConfigTemplate.Execute(&contents, struct {
		SCEPServerURL string
		MDMServerURL  string
		SCEPChallenge string
		Topic         string
	}{
		SCEPServerURL: scepServerURL,
		MDMServerURL:  mdmServerURL,
		SCEPChallenge: scepChallenge,
		Topic:         topic,
	}); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	return contents.Bytes(), nil
}

const (
	munkiRepoPath     = "/mdm/apple/munki/repo/"
	munkiManifestPath = "/mdm/apple/munki/manifest"
)

func startMunkiRepoServer(ctx context.Context, mux *http.ServeMux, config configpkg.MDMMunkiConfig) error {
	if config.HTTPBasicAuth.Username == "" {
		return errors.New("basic auth username empty")
	}
	if config.HTTPBasicAuth.Password == "" {
		return errors.New("basic auth password empty")
	}
	if config.RepoPath == "" {
		return errors.New("repo path empty")
	}
	if _, err := os.Stat(config.RepoPath); err != nil {
		return fmt.Errorf("stat repo path: %w", err)
	}
	munkiFileServer := fleethttp.BasicAuthHandler(
		config.HTTPBasicAuth.Username,
		config.HTTPBasicAuth.Password,
		http.FileServer(
			http.Dir(config.RepoPath),
		),
	)
	mux.Handle(munkiRepoPath, http.StripPrefix(munkiRepoPath, munkiFileServer))
	return nil
}

func startMunkiPkgServer(ctx context.Context, mux *http.ServeMux, config configpkg.MunkiPkgConfig, logger kitlog.Logger) error {
	if config.FilePath == "" {
		return errors.New("pkg file path empty")
	}
	if _, err := os.Stat(config.FilePath); err != nil {
		return fmt.Errorf("stat pkg file path: %w", err)
	}
	pkgURL, err := url.Parse(config.ServerURL)
	if err != nil {
		return fmt.Errorf("parse manifest url: %w", err)
	}
	const munkiPkgPath = "/mdm/apple/munki/pkg"
	pkgURL.Path += munkiPkgPath
	pkgURL.Scheme = "https"
	manifest, err := createManifest(config.FilePath, pkgURL.String())
	if err != nil {
		return fmt.Errorf("create manifest: %w", err)
	}
	mux.HandleFunc(munkiPkgPath, func(w http.ResponseWriter, r *http.Request) {
		pkgFile, err := os.Open(config.FilePath)
		if err != nil {
			level.Error(logger).Log("msg", "munki package open", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer pkgFile.Close()

		w.Header().Set(
			"Content-Disposition",
			"attachment; filename="+filepath.Base(config.FilePath),
		)
		if _, err := io.Copy(w, pkgFile); err != nil {
			level.Error(logger).Log("msg", "munki package write response", "err", err)
		}
	})
	mux.HandleFunc(munkiManifestPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(
			"Content-Disposition",
			"attachment; filename="+strings.TrimSuffix(filepath.Base(config.FilePath), ".pkg")+".plist",
		)

		if _, err := w.Write(manifest); err != nil {
			level.Error(logger).Log("msg", "munki manifest write response", "err", err)
		}
	})
	return nil
}

func createManifest(pkgFilePath string, pkgURL string) ([]byte, error) {
	pkgFile, err := os.Open(pkgFilePath)
	if err != nil {
		return nil, fmt.Errorf("open pkg file: %w", err)
	}
	defer pkgFile.Close()
	manifest, err := appmanifest.Create(&fileWithSize{pkgFile}, pkgURL)
	if err != nil {
		return nil, fmt.Errorf("create manifest file: %w", err)
	}
	var buf bytes.Buffer
	enc := plist.NewEncoder(&buf)
	enc.Indent("  ")
	if err := enc.Encode(manifest); err != nil {
		return nil, fmt.Errorf("encode manifest: %w", err)
	}
	return buf.Bytes(), nil
}

type fileWithSize struct {
	*os.File
}

func (f *fileWithSize) Size() int64 {
	info, err := f.Stat()
	if err != nil {
		log.Fatal(err)
	}
	return info.Size()
}
