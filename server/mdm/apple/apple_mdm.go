package apple_mdm

import (
	"context"
	"fmt"
	"net/url"
	"path"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/logging"
	"github.com/getsentry/sentry-go"
	"github.com/go-kit/log/level"
	"github.com/micromdm/nanodep/godep"

	kitlog "github.com/go-kit/kit/log"
	nanodep_storage "github.com/micromdm/nanodep/storage"
	depsync "github.com/micromdm/nanodep/sync"
)

// DEPName is the identifier/name used in nanodep MySQL storage which
// holds the DEP configuration.
//
// Fleet uses only one DEP configuration set for the whole deployment.
const DEPName = "fleet"

const (
	// SCEPPath is Fleet's HTTP path for the SCEP service.
	SCEPPath = "/mdm/apple/scep"
	// MDMPath is Fleet's HTTP path for the core MDM service.
	MDMPath = "/mdm/apple/mdm"

	// EnrollPath is the HTTP path that serves the mobile profile to devices when enrolling.
	EnrollPath = "/api/mdm/apple/enroll"
	// InstallerPath is the HTTP path that serves installers to Apple devices.
	InstallerPath = "/api/mdm/apple/installer"

	// FleetPayloadIdentifier is the value for the "<key>PayloadIdentifier</key>"
	// used by Fleet MDM on the enrollment profile.
	FleetPayloadIdentifier = "com.fleetdm.fleet.mdm.apple"

	// FleetdConfigPayloadIdentifier is the value for the PayloadIdentifier used
	// by fleetd to read configuration values from the system.
	FleetdConfigPayloadIdentifier = "com.fleetdm.fleetd.config"
)

func ResolveAppleMDMURL(serverURL string) (string, error) {
	return resolveURL(serverURL, MDMPath)
}

func ResolveAppleEnrollMDMURL(serverURL string) (string, error) {
	return resolveURL(serverURL, EnrollPath)
}

func ResolveAppleSCEPURL(serverURL string) (string, error) {
	return resolveURL(serverURL, SCEPPath)
}

func resolveURL(serverURL, relPath string) (string, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, relPath)
	return u.String(), nil
}

type DEPSyncer struct {
	depStorage nanodep_storage.AllStorage
	syncer     *depsync.Syncer
	logger     kitlog.Logger
}

func (d *DEPSyncer) Run(ctx context.Context) error {
	profileUUID, profileModTime, err := d.depStorage.RetrieveAssignerProfile(ctx, DEPName)
	if err != nil {
		return err
	}
	if profileUUID == "" {
		d.logger.Log("msg", "DEP profile not set, nothing to do")
		return nil
	}
	cursor, cursorModTime, err := d.depStorage.RetrieveCursor(ctx, DEPName)
	if err != nil {
		return err
	}
	// If the DEP Profile was changed since last sync then we clear
	// the cursor and perform a full sync of all devices and profile assigning.
	if cursor != "" && profileModTime.After(cursorModTime) {
		d.logger.Log("msg", "clearing device syncer cursor")
		if err := d.depStorage.StoreCursor(ctx, DEPName, ""); err != nil {
			return err
		}
	}
	return d.syncer.Run(ctx)
}

func NewDEPSyncer(
	ds fleet.Datastore,
	depStorage nanodep_storage.AllStorage,
	logger kitlog.Logger,
	loggingDebug bool,
) *DEPSyncer {
	depClient := NewDEPClient(depStorage, ds, logger)
	assignerOpts := []depsync.AssignerOption{
		depsync.WithAssignerLogger(logging.NewNanoDEPLogger(kitlog.With(logger, "component", "nanodep-assigner"))),
	}
	if loggingDebug {
		assignerOpts = append(assignerOpts, depsync.WithDebug())
	}
	assigner := depsync.NewAssigner(
		depClient,
		DEPName,
		depStorage,
		assignerOpts...,
	)

	syncer := depsync.NewSyncer(
		depClient,
		DEPName,
		depStorage,
		depsync.WithLogger(logging.NewNanoDEPLogger(kitlog.With(logger, "component", "nanodep-syncer"))),
		depsync.WithCallback(func(ctx context.Context, isFetch bool, resp *godep.DeviceResponse) error {
			n, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, resp.Devices)
			switch {
			case err != nil:
				level.Error(kitlog.With(logger)).Log("err", err)
				sentry.CaptureException(err)
			case n > 0:
				level.Info(kitlog.With(logger)).Log("msg", fmt.Sprintf("added %d new mdm device(s) to pending hosts", n))
			case n == 0:
				level.Info(kitlog.With(logger)).Log("msg", "no DEP hosts to add")
			}

			return assigner.ProcessDeviceResponse(ctx, resp)
		}),
	)

	return &DEPSyncer{
		syncer:     syncer,
		depStorage: depStorage,
		logger:     logger,
	}
}

// NewDEPClient creates an Apple DEP API HTTP client based on the provided
// storage that will flag the AppConfig's AppleBMTermsExpired field
// whenever the status of the terms changes.
func NewDEPClient(storage godep.ClientStorage, appCfgUpdater fleet.AppConfigUpdater, logger kitlog.Logger) *godep.Client {
	return godep.NewClient(storage, fleethttp.NewClient(), godep.WithAfterHook(func(ctx context.Context, reqErr error) error {
		// if the request failed due to terms not signed, or if it succeeded,
		// update the app config flag accordingly. If it failed for any other
		// reason, do not update the flag.
		termsExpired := reqErr != nil && godep.IsTermsNotSigned(reqErr)
		if reqErr == nil || termsExpired {
			appCfg, err := appCfgUpdater.AppConfig(ctx)
			if err != nil {
				level.Error(logger).Log("msg", "Apple DEP client: failed to get app config", "err", err)
				return reqErr
			}

			var mustSaveAppCfg bool
			if termsExpired && !appCfg.MDM.AppleBMTermsExpired {
				// flag the AppConfig that the terms have changed and must be accepted
				appCfg.MDM.AppleBMTermsExpired = true
				mustSaveAppCfg = true
			} else if reqErr == nil && appCfg.MDM.AppleBMTermsExpired {
				// flag the AppConfig that the terms have been accepted
				appCfg.MDM.AppleBMTermsExpired = false
				mustSaveAppCfg = true
			}

			if mustSaveAppCfg {
				if err := appCfgUpdater.SaveAppConfig(ctx, appCfg); err != nil {
					level.Error(logger).Log("msg", "Apple DEP client: failed to save app config", "err", err)
				}
				level.Debug(logger).Log("msg", "Apple DEP client: updated app config Terms Expired flag",
					"apple_bm_terms_expired", appCfg.MDM.AppleBMTermsExpired)
			}
		}
		return reqErr
	}))
}
