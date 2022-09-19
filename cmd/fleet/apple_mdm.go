package main

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	stdlog "log"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/scep/scep_ca"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/scep/scep_mysql"
	kitlog "github.com/go-kit/kit/log"
	"github.com/micromdm/nanomdm/certverify"
	httpmdm "github.com/micromdm/nanomdm/http/mdm"
	nanomdm_stdlogfmt "github.com/micromdm/nanomdm/log/stdlogfmt"
	nanomdm_service "github.com/micromdm/nanomdm/service"
	"github.com/micromdm/nanomdm/service/certauth"
	"github.com/micromdm/nanomdm/service/nanomdm"
	scep_depot "github.com/micromdm/scep/v2/depot"
	scepserver "github.com/micromdm/scep/v2/server"
)

func registerAppleMDMProtocolServices(
	mux *http.ServeMux,
	config config.MDMAppleConfig,
	mdmStorage *mysql.NanoMDMStorage,
	scepStorage *scep_mysql.MySQLDepot,
	logger kitlog.Logger,
	loggingDebug bool,
) error {
	scepCACrt, err := registerSCEP(mux, config, scepStorage, logger)
	if err != nil {
		return fmt.Errorf("scep: %w", err)
	}
	if err := registerMDM(mux, config, scepCACrt, mdmStorage, logger, loggingDebug); err != nil {
		return fmt.Errorf("mdm: %w", err)
	}
	return nil
}

func registerSCEP(mux *http.ServeMux, config config.MDMAppleConfig, scepStorage *scep_mysql.MySQLDepot, logger kitlog.Logger) (*x509.Certificate, error) {
	scepCACrt, scepCAKey, err := scep_ca.Load(
		config.SCEP.CA.PEMCert,
		config.SCEP.CA.PEMKey,
	)
	if err != nil {
		return nil, fmt.Errorf("load SCEP CA: %w", err)
	}
	var signer scepserver.CSRSigner = scep_depot.NewSigner(
		scepStorage,
		scep_depot.WithValidityDays(config.SCEP.Signer.ValidityDays),
		scep_depot.WithAllowRenewalDays(config.SCEP.Signer.AllowRenewalDays),
	)
	scepChallenge := config.SCEP.Challenge
	if scepChallenge == "" {
		return nil, errors.New("missing SCEP challenge")
	}
	signer = scepserver.ChallengeMiddleware(scepChallenge, signer)
	scepService, err := scepserver.NewService(scepCACrt, scepCAKey, signer,
		scepserver.WithLogger(kitlog.With(logger, "component", "mdm-apple-scep")),
	)
	if err != nil {
		return nil, fmt.Errorf("initialize SCEP service: %w", err)
	}
	scepLogger := kitlog.With(logger, "component", "http-mdm-apple-scep")
	e := scepserver.MakeServerEndpoints(scepService)
	e.GetEndpoint = scepserver.EndpointLoggingMiddleware(scepLogger)(e.GetEndpoint)
	e.PostEndpoint = scepserver.EndpointLoggingMiddleware(scepLogger)(e.PostEndpoint)
	scepHandler := scepserver.MakeHTTPHandler(e, scepService, scepLogger)
	mux.Handle(apple_mdm.SCEPPath, scepHandler)
	return scepCACrt, nil
}

func registerMDM(mux *http.ServeMux, config config.MDMAppleConfig, scepCACrt *x509.Certificate, mdmStorage *mysql.NanoMDMStorage, logger kitlog.Logger, loggingDebug bool) error {
	scepCAPEMBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: scepCACrt.Raw,
	}
	scepCAPEM := pem.EncodeToMemory(scepCAPEMBlock)
	certVerifier, err := certverify.NewPoolVerifier(scepCAPEM, nil, x509.ExtKeyUsageClientAuth)
	if err != nil {
		return fmt.Errorf("certificate pool verifier: %w", err)
	}
	mdmLogger := nanomdm_stdlogfmt.New(
		nanomdm_stdlogfmt.WithLogger(
			stdlog.New(
				kitlog.NewStdlibAdapter(
					kitlog.With(logger, "component", "http-mdm-apple-mdm")),
				"", stdlog.LstdFlags,
			),
		),
		nanomdm_stdlogfmt.WithDebugFlag(loggingDebug),
	)

	nanomdmService := nanomdm.New(mdmStorage, nanomdm.WithLogger(mdmLogger))
	var mdmService nanomdm_service.CheckinAndCommandService = nanomdmService
	mdmService = certauth.New(mdmService, mdmStorage)
	var mdmHandler http.Handler
	mdmHandler = httpmdm.CheckinAndCommandHandler(mdmService, mdmLogger.With("handler", "checkin-command"))
	mdmHandler = httpmdm.CertVerifyMiddleware(mdmHandler, certVerifier, mdmLogger.With("handler", "cert-verify"))
	mdmHandler = httpmdm.CertExtractMdmSignatureMiddleware(mdmHandler, mdmLogger.With("handler", "cert-extract"))
	mux.Handle(apple_mdm.MDMPath, mdmHandler)
	return nil
}
