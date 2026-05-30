package main

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/buford"
	nanomdm_pushsvc "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/service"
	scepdepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	"github.com/fleetdm/fleet/v4/server/service"
)

// initAppleMDMStorages constructs the three NanoMDM-backed storages used by
// the Apple MDM stack: MDM, DEP, and SCEP. Each call delegates to the
// MySQL datastore; failures go through initFatal. Returns nil values on
// the failure path so the function is safe when initFatal does not
// terminate (e.g., tests using a recorder).
func initAppleMDMStorages(mds *mysql.Datastore, initFatal func(err error, msg string)) (
	*mysql.NanoMDMStorage,
	*mysql.NanoDEPStorage,
	scepdepot.Depot,
) {
	mdmStorage, err := mds.NewMDMAppleMDMStorage()
	if err != nil {
		initFatal(err, "initialize mdm apple MySQL storage")
		return nil, nil, nil
	}

	depStorage, err := mds.NewMDMAppleDEPStorage()
	if err != nil {
		initFatal(err, "initialize Apple BM DEP storage")
		return nil, nil, nil
	}

	scepStorage, err := mds.NewSCEPDepot()
	if err != nil {
		initFatal(err, "initialize mdm apple scep storage")
		return nil, nil, nil
	}

	return mdmStorage, depStorage, scepStorage
}

// initAppleMDMPushService chooses the push service implementation: a no-op
// pusher when FLEET_DEV_MDM_APPLE_DISABLE_PUSH=1 (development mode), or a
// real APNs pusher built around the NanoMDM storage in all other cases.
func initAppleMDMPushService(mdmStorage *mysql.NanoMDMStorage, logger *slog.Logger) push.Pusher {
	if dev_mode.Env("FLEET_DEV_MDM_APPLE_DISABLE_PUSH") == "1" {
		return nopPusher{}
	}
	nanoMDMLogger := service.NewNanoMDMLogger(logger.With("component", "apple-mdm-push"))
	pushProviderFactory := buford.NewPushProviderFactory(buford.WithNewClient(func(cert *tls.Certificate) (*http.Client, error) {
		return fleethttp.NewClient(fleethttp.WithTLSClientConfig(&tls.Config{
			Certificates: []tls.Certificate{*cert},
			MinVersion:   tls.VersionTLS12, // Apple APNs requires TLS 1.2+
		})), nil
	}))
	return nanomdm_pushsvc.New(mdmStorage, mdmStorage, pushProviderFactory, nanoMDMLogger)
}

// checkMDMAssetsExist reports whether the named MDM config assets exist in
// the datastore. Returns false (without error) when assets are missing
// entirely or partially — both are valid "not configured yet" states for
// the caller. Other datastore errors are surfaced.
func checkMDMAssetsExist(ctx context.Context, ds fleet.Datastore, names []fleet.MDMAssetName) (bool, error) {
	_, err := ds.GetAllMDMConfigAssetsByName(ctx, names, nil)
	if err != nil {
		if fleet.IsNotFound(err) || errors.Is(err, mysql.ErrPartialResult) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// reconcileAppleMDMAPNsAndSCEPAssets reconciles APNs and SCEP cert/key
// assets between the server configuration and the datastore. If either
// APNs or SCEP is configured, the function requires a server private key,
// inserts the configured certs and keys when the datastore has none, and
// warns when the datastore already has them. Validates that both APNs and
// SCEP are provided together when insert is required. No-op when neither
// is configured.
func reconcileAppleMDMAPNsAndSCEPAssets(
	ctx context.Context,
	cfg config.FleetConfig,
	ds fleet.Datastore,
	logger *slog.Logger,
	initFatal func(err error, msg string),
) {
	if !cfg.MDM.IsAppleAPNsSet() && !cfg.MDM.IsAppleSCEPSet() {
		return
	}
	if len(cfg.Server.PrivateKey) == 0 {
		initFatal(errors.New("inserting MDM APNs and SCEP assets"),
			"missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key")
		return
	}

	// First check if the APNs and SCEP assets are already in the database
	// and only insert config values if they're not already present.
	toInsert := make(map[fleet.MDMAssetName]struct{}, 4)

	// APNs assets.
	found, err := checkMDMAssetsExist(ctx, ds, []fleet.MDMAssetName{fleet.MDMAssetAPNSCert, fleet.MDMAssetAPNSKey})
	switch {
	case err != nil:
		initFatal(err, "reading APNs assets from database")
		return
	case !found:
		toInsert[fleet.MDMAssetAPNSCert] = struct{}{}
		toInsert[fleet.MDMAssetAPNSKey] = struct{}{}
	default:
		logger.WarnContext(ctx,
			"Your server already has stored APNs certificates. Fleet will ignore any certificates provided via environment variables when this happens.")
	}

	// SCEP assets.
	found, err = checkMDMAssetsExist(ctx, ds, []fleet.MDMAssetName{fleet.MDMAssetCACert, fleet.MDMAssetCAKey})
	switch {
	case err != nil:
		initFatal(err, "reading SCEP assets from database")
		return
	case !found:
		toInsert[fleet.MDMAssetCACert] = struct{}{}
		toInsert[fleet.MDMAssetCAKey] = struct{}{}
	default:
		logger.WarnContext(ctx,
			"Your server already has stored SCEP certificates. Fleet will ignore any certificates provided via environment variables when this happens.")
	}

	if len(toInsert) == 0 {
		return
	}

	cfg.MDM.ValidateAppleAPNSAndSCEPPair(initFatal)

	_, apnsCertPEM, apnsKeyPEM, err := cfg.MDM.AppleAPNs()
	if err != nil {
		initFatal(err, "parse Apple APNs certificate and key from config")
		return
	}
	_, appleSCEPCertPEM, appleSCEPKeyPEM, err := cfg.MDM.AppleSCEP()
	if err != nil {
		initFatal(err, "load Apple SCEP certificate and key from config")
		return
	}

	var args []fleet.MDMConfigAsset
	for name := range toInsert {
		switch name {
		case fleet.MDMAssetAPNSCert:
			args = append(args, fleet.MDMConfigAsset{Name: name, Value: apnsCertPEM})
		case fleet.MDMAssetAPNSKey:
			args = append(args, fleet.MDMConfigAsset{Name: name, Value: apnsKeyPEM})
		case fleet.MDMAssetCACert:
			args = append(args, fleet.MDMConfigAsset{Name: name, Value: appleSCEPCertPEM})
		case fleet.MDMAssetCAKey:
			args = append(args, fleet.MDMConfigAsset{Name: name, Value: appleSCEPKeyPEM})
		}
	}

	if err := ds.InsertMDMConfigAssets(ctx, args, nil); err != nil {
		if mysql.IsDuplicate(err) {
			// We already checked for existing assets so we should never hit a
			// duplicate key error here; log just in case.
			logger.DebugContext(ctx, "unexpected duplicate key error inserting MDM APNs and SCEP assets")
			return
		}
		initFatal(err, "inserting MDM APNs and SCEP assets")
		return
	}
}

// reconcileAppleMDMABMAssets reconciles Apple Business Manager (ABM)
// assets between the server configuration and the datastore. Similar shape
// to reconcileAppleMDMAPNsAndSCEPAssets but also inserts a freshly-created
// ABM token row on first setup so the apple_mdm_dep_profile_assigner cron
// can backfill it. No-op when ABM is not configured.
func reconcileAppleMDMABMAssets(
	ctx context.Context,
	cfg config.FleetConfig,
	ds fleet.Datastore,
	logger *slog.Logger,
	initFatal func(err error, msg string),
) {
	if !cfg.MDM.IsAppleBMSet() {
		return
	}
	if len(cfg.Server.PrivateKey) == 0 {
		initFatal(errors.New("inserting MDM ABM assets"),
			"missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key")
		return
	}

	appleBM, err := cfg.MDM.AppleBM()
	if err != nil {
		initFatal(err, "parse Apple BM token, certificate and key from config")
		return
	}

	toInsert := make([]fleet.MDMConfigAsset, 0, 2)

	found, err := checkMDMAssetsExist(ctx, ds, []fleet.MDMAssetName{fleet.MDMAssetABMKey, fleet.MDMAssetABMCert})
	switch {
	case err != nil:
		initFatal(err, "reading ABM assets from database")
		return
	case !found:
		toInsert = append(toInsert,
			fleet.MDMConfigAsset{Name: fleet.MDMAssetABMKey, Value: appleBM.KeyPEM},
			fleet.MDMConfigAsset{Name: fleet.MDMAssetABMCert, Value: appleBM.CertPEM},
		)
	default:
		logger.WarnContext(ctx,
			"Your server already has stored ABM certificates and token. Fleet will ignore any certificates provided via environment variables when this happens.")
	}

	if len(toInsert) == 0 {
		return
	}

	err = ds.InsertMDMConfigAssets(ctx, toInsert, nil)
	switch {
	case err != nil && mysql.IsDuplicate(err):
		// We already checked for existing assets so we should never hit a
		// duplicate key error here; log just in case.
		logger.DebugContext(ctx, "unexpected duplicate key error inserting ABM assets")
		return
	case err != nil:
		initFatal(err, "inserting ABM assets")
		return
	}

	// Insert the ABM token without any metadata; it'll be picked up by the
	// apple_mdm_dep_profile_assigner cron and backfilled. 2000-01-01 is our
	// "zero value" for time.
	if _, err := ds.InsertABMToken(ctx, &fleet.ABMToken{
		EncryptedToken: appleBM.EncryptedToken,
		RenewAt:        time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
	}); err != nil {
		initFatal(err, "save ABM token")
	}
}
