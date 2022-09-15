package service

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	stdlog "log"
	"net/http"
	"strconv"
	"text/template"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	configpkg "github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/scep/scep_ca"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/scep/scep_mysql"
	kitlog "github.com/go-kit/kit/log"
	"github.com/micromdm/nanodep/client"
	"github.com/micromdm/nanodep/godep"
	nanodep_stdlogfmt "github.com/micromdm/nanodep/log/stdlogfmt"
	"github.com/micromdm/nanodep/proxy"
	depsync "github.com/micromdm/nanodep/sync"
	"github.com/micromdm/nanomdm/certverify"
	"github.com/micromdm/nanomdm/cryptoutil"
	nanomdm_httpapi "github.com/micromdm/nanomdm/http/api"
	httpmdm "github.com/micromdm/nanomdm/http/mdm"
	nanomdm_stdlogfmt "github.com/micromdm/nanomdm/log/stdlogfmt"
	"github.com/micromdm/nanomdm/push/buford"
	nanomdm_pushsvc "github.com/micromdm/nanomdm/push/service"
	nanomdm_service "github.com/micromdm/nanomdm/service"
	"github.com/micromdm/nanomdm/service/certauth"
	"github.com/micromdm/nanomdm/service/nanomdm"
	scep_depot "github.com/micromdm/scep/v2/depot"
	scepserver "github.com/micromdm/scep/v2/server"
	_ "go.elastic.co/apm/module/apmsql"
	_ "go.elastic.co/apm/module/apmsql/mysql"
)

type SetupConfig struct {
	MDMConfig    configpkg.MDMAppleConfig
	Logger       kitlog.Logger
	MDMStorage   *mysql.NanoMDMStorage
	SCEPStorage  *scep_mysql.MySQLDepot
	DEPStorage   *mysql.NanoDEPStorage
	Datastore    fleet.Datastore
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
	return nil
}

// TODO(lucas): None of the API endpoints have authentication yet.
// We should use Fleet admin bearer token authentication.
func registerServices(ctx context.Context, mux *http.ServeMux, config SetupConfig) error {
	scepCACrt, err := registerSCEP(mux, config)
	if err != nil {
		return fmt.Errorf("scep: %w", err)
	}
	if err := registerMDM(mux, config, scepCACrt); err != nil {
		return fmt.Errorf("mdm: %w", err)
	}
	if err := registerEnroll(ctx, mux, config); err != nil {
		return fmt.Errorf("enroll endpoint: %w", err)
	}
	if err := registerInstaller(ctx, mux, config); err != nil {
		return fmt.Errorf("installer endpoint: %w", err)
	}
	registerDEPProxy(mux, config)
	return nil
}

func registerSCEP(mux *http.ServeMux, config SetupConfig) (*x509.Certificate, error) {
	scepCACrt, scepCAKey, err := scep_ca.Load(
		config.MDMConfig.SCEP.CA.PEMCert,
		config.MDMConfig.SCEP.CA.PEMKey,
	)
	if err != nil {
		return nil, fmt.Errorf("load SCEP CA: %w", err)
	}
	var signer scepserver.CSRSigner = scep_depot.NewSigner(
		config.SCEPStorage,
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

// TODO(lucas): The enroll profile must be protected by SSO. Currently the endpoint is unauthenticated.
func registerEnroll(ctx context.Context, mux *http.ServeMux, config SetupConfig) error {
	topic, err := cryptoutil.TopicFromPEMCert(config.MDMConfig.MDM.PushCert.PEMCert)
	if err != nil {
		return fmt.Errorf("extract push certificate topic: %w", err)
	}
	enrollLogger := kitlog.With(config.Logger, "handler", "enroll-profile")
	mux.HandleFunc("/mdm/apple/api/enroll", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			enrollLogger.Log("err", "invalid method")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		values := r.URL.Query()
		id, ok := values["id"]
		if !ok || len(id) == 0 {
			enrollLogger.Log("err", "missing enrollment id")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		enrollmentID, err := strconv.ParseUint(id[0], 10, 64)
		if err != nil {
			enrollLogger.Log("err", "invalid enrollment id")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		enrollment, err := config.Datastore.MDMAppleEnrollment(ctx, uint(enrollmentID))
		if err != nil {
			enrollLogger.Log("err", err, "enrollmentID", enrollmentID)
			status := http.StatusInternalServerError
			if fleet.IsNotFound(err) {
				status = http.StatusNotFound
			}
			w.WriteHeader(status)
			return
		}
		// TODO(lucas): Actually use enrollment.Config.
		_ = enrollment
		mobileConfig, err := generateMobileConfig(
			"https://"+config.MDMConfig.ServerAddress+"/mdm/apple/scep",
			"https://"+config.MDMConfig.ServerAddress+"/mdm/apple/mdm",
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

func registerInstaller(ctx context.Context, mux *http.ServeMux, config SetupConfig) error {
	installerLogger := kitlog.With(config.Logger, "handler", "enroll-profile")
	mux.HandleFunc("/mdm/apple/installer", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			installerLogger.Log("err", "invalid method")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		values := r.URL.Query()
		tv, ok := values["token"]
		if !ok || len(tv) != 1 || tv[0] == "" {
			installerLogger.Log("err", "invalid token", "value", tv)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		token := tv[0]
		installer, err := config.Datastore.MDMAppleInstaller(ctx, token)
		if err != nil {
			installerLogger.Log("err", err, "token", token)
			status := http.StatusInternalServerError
			if fleet.IsNotFound(err) {
				status = http.StatusNotFound
			}
			w.WriteHeader(status)
			return
		}
		w.Header().Set("Content-Length", strconv.FormatInt(installer.Size, 10))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, installer.Name))

		// OK to just log the error here as writing anything on
		// `http.ResponseWriter` sets the status code to 200 (and it can't be
		// changed.) Clients should rely on matching content-length with the
		// header provided
		if n, err := w.Write(installer.Installer); err != nil {
			logging.WithExtras(ctx, "err", err, "bytes_copied", n)
		}
	})
	return nil
}
