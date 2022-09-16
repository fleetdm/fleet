package service

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	stdlog "log"
	"net/http"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	configpkg "github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/scep/scep_ca"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/scep/scep_mysql"
	kitlog "github.com/go-kit/kit/log"
	"github.com/micromdm/nanodep/godep"
	nanodep_stdlogfmt "github.com/micromdm/nanodep/log/stdlogfmt"
	depsync "github.com/micromdm/nanodep/sync"
	"github.com/micromdm/nanomdm/certverify"
	httpmdm "github.com/micromdm/nanomdm/http/mdm"
	nanomdm_stdlogfmt "github.com/micromdm/nanomdm/log/stdlogfmt"
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
	// TODO(lucas): Move from here.
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

	nanomdmService := nanomdm.New(config.MDMStorage, nanomdm.WithLogger(mdmLogger))
	var mdmService nanomdm_service.CheckinAndCommandService = nanomdmService
	mdmService = certauth.New(mdmService, config.MDMStorage)
	var mdmHandler http.Handler
	mdmHandler = httpmdm.CheckinAndCommandHandler(mdmService, mdmLogger.With("handler", "checkin-command"))
	mdmHandler = httpmdm.CertVerifyMiddleware(mdmHandler, certVerifier, mdmLogger.With("handler", "cert-verify"))
	mdmHandler = httpmdm.CertExtractMdmSignatureMiddleware(mdmHandler, mdmLogger.With("handler", "cert-extract"))
	mux.Handle("/mdm/apple/mdm", mdmHandler)

	return nil
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
