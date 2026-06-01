package svctest

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	eeservice "github.com/fleetdm/fleet/v4/ee/server/service"
	"github.com/fleetdm/fleet/v4/ee/server/service/digicert"
	"github.com/fleetdm/fleet/v4/ee/server/service/est"
	"github.com/fleetdm/fleet/v4/ee/server/service/scep"
	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/datastore/filesystem"
	"github.com/fleetdm/fleet/v4/server/errorstore"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/logging"
	"github.com/fleetdm/fleet/v4/server/mail"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	nanodep_storage "github.com/fleetdm/fleet/v4/server/mdm/nanodep/storage"
	nanomdm_push "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	fleet_mock "github.com/fleetdm/fleet/v4/server/mock"
	nanodep_mock "github.com/fleetdm/fleet/v4/server/mock/nanodep"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/service/async"
	"github.com/fleetdm/fleet/v4/server/service/redis_key_value"
	"github.com/fleetdm/fleet/v4/server/service/redis_lock"
	"github.com/fleetdm/fleet/v4/server/sso"
	"github.com/stretchr/testify/require"
)

// NewTestService builds a fleet.Service backed by ds for use in tests. It
// honors the relevant fields on opts (License, Logger, Pool, etc.).
func NewTestService(t *testing.T, ds fleet.Datastore, cfg config.FleetConfig, opts ...*service.TestServerOpts) (fleet.Service, context.Context) {
	var rs fleet.QueryResultStore
	if len(opts) > 0 && opts[0].Rs != nil {
		rs = opts[0].Rs
	}
	var lq fleet.LiveQueryStore
	if len(opts) > 0 && opts[0].Lq != nil {
		lq = opts[0].Lq
	}
	return newTestServiceWithConfig(t, ds, cfg, rs, lq, opts...)
}

func newTestServiceWithConfig(t *testing.T, ds fleet.Datastore, fleetConfig config.FleetConfig, rs fleet.QueryResultStore, lq fleet.LiveQueryStore, opts ...*service.TestServerOpts) (fleet.Service, context.Context) {
	lic := &fleet.LicenseInfo{Tier: fleet.TierFree}
	logger := slog.New(slog.DiscardHandler)
	writer, err := logging.NewFilesystemLogWriter(t.Context(), fleetConfig.Filesystem.StatusLogFile, logger, fleetConfig.Filesystem.EnableLogRotation,
		fleetConfig.Filesystem.EnableLogCompression, 500, 28, 3)
	require.NoError(t, err)

	osqlogger := &service.OsqueryLogger{Status: writer, Result: writer}

	var (
		failingPolicySet                fleet.FailingPolicySet        = service.NewMemFailingPolicySet()
		enrollHostLimiter               fleet.EnrollHostLimiter       = nopEnrollHostLimiter{}
		depStorage                      nanodep_storage.AllDEPStorage = &nanodep_mock.Storage{}
		mailer                          fleet.MailService             = &mockMailService{SendEmailFn: func(e fleet.Email) error { return nil }}
		c                               clock.Clock                   = clock.C
		scepConfigService                                             = scep.NewSCEPConfigService(logger, nil)
		digiCertService                                               = digicert.NewService(digicert.WithLogger(logger))
		estCAService                                                  = est.NewService(est.WithLogger(logger))
		conditionalAccessMicrosoftProxy service.ConditionalAccessMicrosoftProxy

		mdmStorage             fleet.MDMAppleStore
		mdmPusher              nanomdm_push.Pusher
		ssoStore               sso.SessionStore
		profMatcher            fleet.ProfileMatcher
		softwareInstallStore   fleet.SoftwareInstallerStore
		bootstrapPackageStore  fleet.MDMBootstrapPackageStore
		softwareTitleIconStore fleet.SoftwareTitleIconStore
		distributedLock        fleet.Lock
		keyValueStore          fleet.KeyValueStore
		androidService         android.Service
	)
	if len(opts) > 0 {
		if opts[0].Clock != nil {
			c = opts[0].Clock
		}
	}

	if len(opts) > 0 && opts[0].KeyValueStore != nil {
		keyValueStore = opts[0].KeyValueStore
	}

	task := async.NewTask(ds, nil, c, nil)
	if len(opts) > 0 {
		if opts[0].Task != nil {
			task = opts[0].Task
		} else {
			opts[0].Task = task
		}
	}

	if len(opts) > 0 {
		if opts[0].Logger != nil {
			logger = opts[0].Logger
		}
		if opts[0].License != nil {
			lic = opts[0].License
		}
		if opts[0].Pool != nil {
			ssoStore = sso.NewSessionStore(opts[0].Pool)
			profMatcher = apple_mdm.NewProfileMatcher(opts[0].Pool)
			distributedLock = redis_lock.NewLock(opts[0].Pool)
			keyValueStore = redis_key_value.New(opts[0].Pool)
		}
		if opts[0].ProfileMatcher != nil {
			profMatcher = opts[0].ProfileMatcher
		}
		if opts[0].FailingPolicySet != nil {
			failingPolicySet = opts[0].FailingPolicySet
		}
		if opts[0].EnrollHostLimiter != nil {
			enrollHostLimiter = opts[0].EnrollHostLimiter
		}
		if opts[0].UseMailService {
			mailer, err = mail.NewService(config.TestConfig())
			require.NoError(t, err)
		}
		if opts[0].SoftwareInstallStore != nil {
			softwareInstallStore = opts[0].SoftwareInstallStore
		}
		if opts[0].BootstrapPackageStore != nil {
			bootstrapPackageStore = opts[0].BootstrapPackageStore
		}
		if opts[0].SoftwareTitleIconStore != nil {
			softwareTitleIconStore = opts[0].SoftwareTitleIconStore
		}

		// allow to explicitly set MDM storage to nil
		mdmStorage = opts[0].MDMStorage
		if opts[0].DEPStorage != nil {
			depStorage = opts[0].DEPStorage
		}
		// allow to explicitly set mdm pusher to nil
		mdmPusher = opts[0].MDMPusher
	}

	ctx := license.NewContext(context.Background(), lic)

	cronSchedulesService := fleet.NewCronSchedules()

	var eh *errorstore.Handler
	if len(opts) > 0 {
		if opts[0].Pool != nil {
			eh = errorstore.NewHandler(ctx, opts[0].Pool, logger, time.Minute*10)
			ctx = ctxerr.NewContext(ctx, eh)
		}
		if opts[0].StartCronSchedules != nil {
			for _, fn := range opts[0].StartCronSchedules {
				err = cronSchedulesService.StartCronSchedule(fn(ctx, ds))
				require.NoError(t, err)
			}
		}
	}
	if len(opts) > 0 && opts[0].SCEPConfigService != nil {
		scepConfigService = opts[0].SCEPConfigService
	}
	if len(opts) > 0 && opts[0].DigiCertService != nil {
		digiCertService = opts[0].DigiCertService
	}
	if len(opts) > 0 && opts[0].ConditionalAccessMicrosoftProxy != nil {
		conditionalAccessMicrosoftProxy = opts[0].ConditionalAccessMicrosoftProxy
		fleetConfig.MicrosoftCompliancePartner.ProxyAPIKey = "insecure" // setting this so the feature is "enabled".
	}

	if len(opts) > 0 && opts[0].AndroidModule != nil {
		androidService = opts[0].AndroidModule
	}

	var wstepManager microsoft_mdm.CertManager
	if fleetConfig.MDM.WindowsWSTEPIdentityCert != "" && fleetConfig.MDM.WindowsWSTEPIdentityKey != "" {
		rawCert, err := os.ReadFile(fleetConfig.MDM.WindowsWSTEPIdentityCert)
		require.NoError(t, err)
		rawKey, err := os.ReadFile(fleetConfig.MDM.WindowsWSTEPIdentityKey)
		require.NoError(t, err)

		wstepManager, err = microsoft_mdm.NewCertManager(ds, rawCert, rawKey)
		require.NoError(t, err)
	}

	orgLogoStore, err := filesystem.NewOrgLogoStore(t.TempDir())
	require.NoError(t, err)

	svc, err := service.NewService(
		ctx,
		ds,
		task,
		rs,
		logger,
		osqlogger,
		fleetConfig,
		mailer,
		c,
		ssoStore,
		lq,
		ds,
		failingPolicySet,
		&fleet.NoOpGeoIP{},
		enrollHostLimiter,
		depStorage,
		mdmStorage,
		mdmPusher,
		cronSchedulesService,
		wstepManager,
		scepConfigService,
		digiCertService,
		conditionalAccessMicrosoftProxy,
		keyValueStore,
		androidService,
		orgLogoStore,
	)
	if err != nil {
		panic(err)
	}
	if lic.IsPremium() {
		if softwareInstallStore == nil {
			// default to file-based
			dir := t.TempDir()
			store, err := filesystem.NewSoftwareInstallerStore(dir)
			if err != nil {
				panic(err)
			}
			softwareInstallStore = store
		}

		var androidModule android.Service
		if len(opts) > 0 {
			androidModule = opts[0].AndroidModule
		}

		svc, err = eeservice.NewService(
			svc,
			ds,
			logger,
			fleetConfig,
			mailer,
			c,
			depStorage,
			apple_mdm.NewMDMAppleCommander(mdmStorage, mdmPusher),
			ssoStore,
			profMatcher,
			softwareInstallStore,
			bootstrapPackageStore,
			softwareTitleIconStore,
			distributedLock,
			keyValueStore,
			scepConfigService,
			digiCertService,
			androidModule,
			estCAService,
		)
		if err != nil {
			panic(err)
		}
	}

	// Set up mock activity service for unit tests. When DBConns is provided,
	// RunServerForTestsWithServiceWithDS will overwrite this with the real bounded context.
	activityMock := &fleet_mock.MockActivityService{
		NewActivityFunc: func(_ context.Context, _ *activity_api.User, _ activity_api.ActivityDetails) error {
			return nil
		},
	}
	svc.SetActivityService(activityMock)
	if len(opts) > 0 {
		opts[0].ActivityMock = activityMock
	}

	// Set up mock ACME service for unit tests. When DBConns is provided,
	// RunServerForTestsWithServiceWithDS will overwrite this with the real service module.
	svc.SetACMEService(&fleet_mock.MockACMEService{})

	return svc, ctx
}
