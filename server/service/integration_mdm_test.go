package service

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	_ "embed"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/WatchBeam/clock"

	"github.com/MicahParks/jwkset"
	"github.com/davecgh/go-spew/spew"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/testhelpers"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	android_service "github.com/fleetdm/fleet/v4/server/mdm/android/service"
	"github.com/fleetdm/fleet/v4/server/pubsub"

	svc_scep "github.com/fleetdm/fleet/v4/ee/server/service/scep"
	"github.com/fleetdm/fleet/v4/pkg/fleetdbase"
	shared_mdm "github.com/fleetdm/fleet/v4/pkg/mdm"
	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/filesystem"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/datastore/s3/s3test"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/live_query/live_query_mock"
	servermdm "github.com/fleetdm/fleet/v4/server/mdm"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/vpp"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	nanodep_client "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	nanodep_storage "github.com/fleetdm/fleet/v4/server/mdm/nanodep/storage"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/tokenpki"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	nanomdm_pushsvc "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/service"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/mock"
	"github.com/fleetdm/fleet/v4/server/service/redis_key_value"
	"github.com/fleetdm/fleet/v4/server/service/schedule"
	"github.com/fleetdm/fleet/v4/server/worker"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"github.com/micromdm/plist"
	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestIntegrationsMDM(t *testing.T) {
	testingSuite := new(integrationMDMTestSuite)
	testingSuite.withServer.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

type integrationMDMTestSuite struct {
	suite.Suite
	withServer
	fleetCfg                     config.FleetConfig
	fleetDMNextCSRStatus         atomic.Value
	pushProvider                 *mock.APNSPushProvider
	depStorage                   nanodep_storage.AllDEPStorage
	profileSchedule              *schedule.Schedule
	androidProfileSchedule       *schedule.Schedule
	integrationsSchedule         *schedule.Schedule
	cleanupsSchedule             *schedule.Schedule
	appleMDMWorkerSchedule       *schedule.Schedule
	appleOSUpdatesSchedule       *schedule.Schedule
	onProfileJobDone             func() // function called when profileSchedule.Trigger() job completed
	onAndroidProfileJobDone      func() // function called when androidProfileSchedule.Trigger() job completed
	onIntegrationsScheduleDone   func() // function called when integrationsSchedule.Trigger() job completed
	onCleanupScheduleDone        func() // function called when cleanupsSchedule.Trigger() job completed
	onAppleMDMWorkerScheduleDone func() // function called when appleMDMWorkerSchedule.Trigger() job completed
	onAppleOSUpdatesScheduleDone func() // function called when appleOSUpdatesSchedule.Trigger() job completed
	mdmStorage                   *mysql.NanoMDMStorage
	worker                       *worker.Worker
	appleMDMWorker               *worker.Worker
	// Flag to skip jobs processing by worker
	skipWorkerJobs atomic.Bool
	mdmCommander   *apple_mdm.MDMAppleCommander
	logger         *slog.Logger
	scepChallenge  string

	acmeCertCA  *x509.Certificate
	acmeCertKey *ecdsa.PrivateKey

	appleVPPConfigSrv         *httptest.Server
	appleVPPConfigSrvConfig   *appleVPPConfigSrvConf
	appleVPPProxySrv          *httptest.Server
	appleVPPProxySrvData      map[string]string
	appleGDMFSrv              *httptest.Server
	mockedDownloadFleetdmMeta fleetdbase.Metadata
	scepConfig                *svc_scep.SCEPConfigService
	androidAPIClient          *android_mock.Client
	androidSvc                android.Service
	proxyCallbackURL          string
	jwtSigningKey             *rsa.PrivateKey
	softwareInstallerStore    fleet.SoftwareInstallerStore
	stagedUploadStore         fleet.StagedUploadStore
	acmeSvc                   fleet.ACMEWriteService
	keyValueStore             fleet.AdvancedKeyValueStore
}

// appleVPPConfigSrvConf is used to configure the mock server that mocks Apple's VPP endpoints.
type appleVPPConfigSrvConf struct {
	Assets        []vpp.Asset
	SerialNumbers []string
	Location      string
	// Disassociations records every /assets/disassociate request the mock
	// server received, in order. Used by tests that assert Fleet released a
	// previously-reserved VPP seat (e.g. on install failure or cancel).
	Disassociations []vpp.DisassociateAssetsRequest
}

var defaultVPPAssetList = []vpp.Asset{
	{
		AdamID:         "1",
		PricingParam:   "STDQ",
		AvailableCount: 12,
	},
	{
		AdamID:         "2",
		PricingParam:   "STDQ",
		AvailableCount: 3,
	},
	{
		AdamID:         "3",
		PricingParam:   "STDQ",
		AvailableCount: 1,
	},
}

const defaultFakeJWTKeyID = "fleetdefaultkeyid"

// requireTrackedRefetchCommands asserts the host's tracking rows cover exactly
// the given refetch command types, each recording the full prefixed UUID of
// the command it tracks (the exact UUIDs are random).
func requireTrackedRefetchCommands(t *testing.T, commands []fleet.HostMDMCommand, hostID uint, commandTypes ...string) {
	gotTypes := make([]string, 0, len(commands))
	for _, c := range commands {
		assert.Equal(t, hostID, c.HostID)
		assert.Truef(t, strings.HasPrefix(c.CommandUUID, c.CommandType),
			"tracking row for %s must record its full prefixed command UUID, got %q", c.CommandType, c.CommandUUID)
		gotTypes = append(gotTypes, c.CommandType)
	}
	assert.ElementsMatch(t, commandTypes, gotTypes)
}

func (s *integrationMDMTestSuite) SetupSuite() {
	s.withDS.SetupSuite("integrationMDMTestSuite")

	appConf, err := s.ds.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.MDM.EnabledAndConfigured = true
	appConf.MDM.WindowsEnabledAndConfigured = true
	appConf.MDM.AppleBMEnabledAndConfigured = true
	appConf.MDM.AndroidEnabledAndConfigured = true
	appConf.MDM.MacOSSetup.BootstrapPackage.Set = false
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)

	fleetCfg := config.TestConfig()
	fleetCfg.MDM.AppleConnectJWT = "fake-token" // skip as we test VPP auth elsewhere
	// Custom activations are opt-in on the server (#50764); the DDM suite covers
	// them, so this suite runs with them on.
	fleetCfg.MDM.AllowCustomActivations = true
	testCert, testKey, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(s.T(), err)
	testCertPEM := tokenpki.PEMCertificate(testCert.Raw)
	testKeyPEM := tokenpki.PEMRSAPrivateKey(testKey)
	config.SetTestMDMConfig(s.T(), &fleetCfg, testCertPEM, testKeyPEM, "../../server/service/testdata")
	fleetCfg.Osquery.EnrollCooldown = 0
	fleetCfg.Server.VPPVerifyRequestDelay = 0 * time.Second // Remove the delay to speed up tests
	fleetCfg.Server.VPPVerifyTimeout = fleet.DefaultVPPInstallVerifyTimeout

	mdmStorage, err := s.ds.NewMDMAppleMDMStorage()
	require.NoError(s.T(), err)
	mdmStorage.SetNewActivityFunc(func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return s.fleetSvc.NewActivity(ctx, user, activity)
	})
	depStorage, err := s.ds.NewMDMAppleDEPStorage()
	require.NoError(s.T(), err)
	scepStorage, err := s.ds.NewSCEPDepot()
	require.NoError(s.T(), err)
	conditionalAccessSCEPStorage, err := s.ds.NewConditionalAccessSCEPDepot(slog.New(slog.DiscardHandler), &fleetCfg)
	require.NoError(s.T(), err)

	pushLog := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if os.Getenv("FLEET_INTEGRATION_TESTS_DISABLE_LOG") != "" {
		pushLog = slog.New(slog.DiscardHandler)
	}
	pushFactory, pushProvider := newMockAPNSPushProviderFactory()
	mdmPushService := nanomdm_pushsvc.New(
		mdmStorage,
		mdmStorage,
		pushFactory,
		NewNanoMDMLogger(pushLog),
	)
	mdmCommander := apple_mdm.NewMDMAppleCommander(mdmStorage, mdmPushService)
	s.redisPool = redistest.SetupRedis(s.T(), "zz", false, false, false)
	s.withServer.lq = live_query_mock.New(s.T())

	wlog := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if os.Getenv("FLEET_INTEGRATION_TESTS_DISABLE_LOG") != "" {
		wlog = slog.New(slog.DiscardHandler)
	}

	androidMockClient := &android_mock.Client{}
	androidMockClient.SetAuthenticationSecretFunc = func(secret string) error {
		return nil
	}
	androidSvc, err := android_service.NewServiceWithClient(wlog, s.ds, androidMockClient, "test-private-key", s.ds,
		func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			return s.fleetSvc.NewActivity(ctx, user, activity)
		}, config.AndroidAgentConfig{
			Package:       "com.fleetdm.agent",
			SigningSHA256: "abc123def456",
		})
	if err != nil {
		panic(err)
	}
	androidSvc.(*android_service.Service).AllowLocalhostServerURL = true
	s.androidSvc = androidSvc

	macosJob := &worker.MacosSetupAssistant{
		Datastore:  s.ds,
		Log:        wlog,
		DEPService: apple_mdm.NewDEPService(s.ds, depStorage, wlog),
		DEPClient:  apple_mdm.NewDEPClient(depStorage, s.ds, wlog),
	}
	appleMDMJob := &worker.AppleMDM{
		Datastore: s.ds,
		Log:       wlog,
		Commander: mdmCommander,
	}
	vppVerifyJob := &worker.AppleSoftware{
		Datastore: s.ds,
		Log:       wlog,
		Commander: mdmCommander,
	}
	softwareWorker := &worker.SoftwareWorker{
		Datastore:     s.ds,
		Log:           wlog,
		AndroidModule: androidSvc,
	}
	workr := worker.NewWorker(s.ds, wlog)
	workr.TestIgnoreUnknownJobs = true
	workr.Register(macosJob, vppVerifyJob, softwareWorker)

	s.worker = workr

	appleMDMWorker := worker.NewWorker(s.ds, wlog)
	appleMDMWorker.TestIgnoreUnknownJobs = true
	appleMDMWorker.Register(appleMDMJob)

	s.appleMDMWorker = appleMDMWorker

	// clear the jobs queue of any pending jobs generated via DB migrations
	mysqltest.ExecAdhocSQL(s.T(), s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(context.Background(), "DELETE FROM jobs")
		return err
	})

	var integrationsSchedule *schedule.Schedule
	var profileSchedule *schedule.Schedule
	var cleanupsSchedule *schedule.Schedule
	var androidProfileSchedule *schedule.Schedule
	var appleMDMWorkerSchedule *schedule.Schedule
	var appleOSUpdatesSchedule *schedule.Schedule
	cronLog := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if os.Getenv("FLEET_INTEGRATION_TESTS_DISABLE_LOG") != "" {
		cronLog = slog.New(slog.DiscardHandler)
	}
	serverLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if os.Getenv("FLEET_INTEGRATION_TESTS_DISABLE_LOG") != "" {
		serverLogger = slog.New(slog.DiscardHandler)
	}

	var softwareInstallerStore fleet.SoftwareInstallerStore
	var bootstrapPackageStore fleet.MDMBootstrapPackageStore
	_, localS3Enabled := os.LookupEnv("S3_STORAGE_TEST")
	if wantStore := os.Getenv("FLEET_INTEGRATION_TESTS_SOFTWARE_INSTALLER_STORE"); localS3Enabled &&
		(wantStore == "s3" || (wantStore == "" && time.Now().UnixNano()%2 == 0)) {

		s.T().Log(">>> using S3-compatible software installer store")
		softwareInstallerStore = s3test.SetupSoftwareInstallerStore(s.T(), "integration-tests", "")
		bootstrapPackageStore = s3test.SetupBootstrapPackageStore(s.T(), "integration-tests", "")
	}
	s.softwareInstallerStore = softwareInstallerStore
	if localS3Enabled {
		s.stagedUploadStore = s3test.SetupStagedUploadStore(s.T(), "integration-tests", "")
	}
	scepTimeout := ptr.Duration(10 * time.Second)
	s.scepConfig = svc_scep.NewSCEPConfigService(serverLogger, scepTimeout).(*svc_scep.SCEPConfigService)

	// Create a software title icon store
	iconDir := s.T().TempDir()
	softwareTitleIconStore, err := filesystem.NewSoftwareTitleIconStore(iconDir)
	require.NoError(s.T(), err)

	keyValueStore := redis_key_value.New(s.redisPool)

	serverConfig := TestServerOpts{
		License: &fleet.LicenseInfo{
			Tier: fleet.TierPremium,
		},
		DBConns:                s.dbConns,
		Logger:                 serverLogger,
		FleetConfig:            &fleetCfg,
		MDMStorage:             mdmStorage,
		DEPStorage:             depStorage,
		SCEPStorage:            scepStorage,
		ConditionalAccess:      &ConditionalAccess{SCEPStorage: conditionalAccessSCEPStorage},
		MDMPusher:              mdmPushService,
		Pool:                   s.redisPool,
		Rs:                     pubsub.NewInmemQueryResults(),
		Lq:                     s.lq,
		SoftwareInstallStore:   s.softwareInstallerStore,
		SoftwareTitleIconStore: softwareTitleIconStore,
		BootstrapPackageStore:  bootstrapPackageStore,
		StagedUploadStore:      s.stagedUploadStore,
		AndroidMockClient:      androidMockClient,
		AndroidModule:          androidSvc,
		KeyValueStore:          keyValueStore,
		StartCronSchedules: []TestNewScheduleFunc{
			func(ctx context.Context, ds fleet.Datastore) fleet.NewCronScheduleFunc {
				return func() (fleet.CronSchedule, error) {
					const name = string(fleet.CronMDMAppleProfileManager)
					logger := cronLog
					profileSchedule = schedule.New(
						ctx, name, s.T().Name(), 1*time.Hour, ds, ds,
						schedule.WithLogger(logger),
						schedule.WithJob("manage_apple_profiles", func(ctx context.Context) error {
							logger.InfoContext(ctx, "Starting manage_apple_profiles job", "test", s.T().Name(), "time", time.Now().Format(time.RFC3339))
							if s.onProfileJobDone != nil {
								defer func() {
									logger.InfoContext(ctx, "Completing manage_apple_profiles job", "test", s.T().Name(), "time",
										time.Now().Format(time.RFC3339))
									s.onProfileJobDone()
								}()
							}
							err = ReconcileAppleProfilesBatched(ctx, ds, mdmCommander, keyValueStore, logger, 0, false)
							require.NoError(s.T(), err)
							return err
						}),
						schedule.WithJob("manage_apple_declarations", func(ctx context.Context) error {
							logger.InfoContext(ctx, "Starting manage_apple_declarations job", "test", s.T().Name(), "time", time.Now().Format(time.RFC3339))
							if s.onProfileJobDone != nil {
								defer func() {
									logger.InfoContext(ctx, "Completing manage_apple_declarations job", "test", s.T().Name(), "time",
										time.Now().Format(time.RFC3339))
									s.onProfileJobDone()
								}()
							}
							err = ReconcileAppleDeclarationsBatched(ctx, ds, mdmCommander, logger)
							require.NoError(s.T(), err)
							return err
						}),
						schedule.WithJob("manage_windows_profiles", func(ctx context.Context) error {
							logger.InfoContext(ctx, "Starting manage_windows_profiles job", "test", s.T().Name(), "time", time.Now().Format(time.RFC3339))
							if s.onProfileJobDone != nil {
								defer func() {
									logger.InfoContext(ctx, "Completing manage_windows_profiles job", "test", s.T().Name(), "time",
										time.Now().Format(time.RFC3339))
									s.onProfileJobDone()
								}()
							}
							err := ReconcileWindowsProfiles(ctx, ds, logger, false)
							require.NoError(s.T(), err)
							return err
						}),
					)
					return profileSchedule, nil
				}
			},
			func(ctx context.Context, ds fleet.Datastore) fleet.NewCronScheduleFunc {
				return func() (fleet.CronSchedule, error) {
					const name = string(fleet.CronAppleMDMWorker)
					logger := cronLog
					appleMDMWorkerSchedule = schedule.New(
						ctx, name, s.T().Name(), 1*time.Hour, ds, ds,
						schedule.WithLogger(logger),
						schedule.WithJob("apple_mdm_worker", func(ctx context.Context) error {
							if s.onAppleMDMWorkerScheduleDone != nil {
								defer s.onAppleMDMWorkerScheduleDone()
							}

							return s.appleMDMWorker.ProcessJobs(ctx)
						}),
					)
					return appleMDMWorkerSchedule, nil
				}
			},
			func(ctx context.Context, ds fleet.Datastore) fleet.NewCronScheduleFunc {
				return func() (fleet.CronSchedule, error) {
					const name = string(fleet.CronAppleMDMOSUpdatesSchedule)
					logger := cronLog
					appleOSUpdatesSchedule = schedule.New(
						ctx, name, s.T().Name(), 1*time.Hour, ds, ds,
						schedule.WithLogger(logger),
						schedule.WithJob("apple_os_updates", func(ctx context.Context) error {
							if s.onAppleOSUpdatesScheduleDone != nil {
								defer s.onAppleOSUpdatesScheduleDone()
							}
							return apple_mdm.HandleAppleMDMOSUpdates(ctx, ds, logger)
						}),
					)
					return appleOSUpdatesSchedule, nil
				}
			},
			func(ctx context.Context, ds fleet.Datastore) fleet.NewCronScheduleFunc {
				return func() (fleet.CronSchedule, error) {
					const name = string(fleet.CronWorkerIntegrations)
					logger := cronLog
					integrationsSchedule = schedule.New(
						ctx, name, s.T().Name(), 1*time.Minute, ds, ds,
						schedule.WithLogger(logger),
						schedule.WithJob("integrations_worker", func(ctx context.Context) error {
							if s.skipWorkerJobs.Load() {
								return nil
							}
							return s.worker.ProcessJobs(ctx)
						}),
						schedule.WithJob("dep_cooldowns", func(ctx context.Context) error {
							if s.skipWorkerJobs.Load() {
								return nil
							}
							if s.onIntegrationsScheduleDone != nil {
								defer s.onIntegrationsScheduleDone()
							}

							return worker.ProcessDEPCooldowns(ctx, ds, logger)
						}),
					)
					return integrationsSchedule, nil
				}
			},
			func(ctx context.Context, ds fleet.Datastore) fleet.NewCronScheduleFunc {
				return func() (fleet.CronSchedule, error) {
					const name = string(fleet.CronCleanupsThenAggregation)
					logger := cronLog
					cleanupsSchedule = schedule.New(
						ctx, name, s.T().Name(), 1*time.Hour, ds, ds,
						schedule.WithLogger(logger),
						schedule.WithJob("cleanup_android_enterprise", func(ctx context.Context) error {
							return androidSvc.VerifyExistingEnterpriseIfAny(ctx)
						}),
						schedule.WithJob("renew_android_certificate_templates", func(ctx context.Context) error {
							if s.onCleanupScheduleDone != nil {
								defer s.onCleanupScheduleDone()
							}
							return android_service.RenewCertificateTemplates(ctx, ds, logger)
						}),
					)
					return cleanupsSchedule, nil
				}
			},
			func(ctx context.Context, ds fleet.Datastore) fleet.NewCronScheduleFunc {
				return func() (fleet.CronSchedule, error) {
					const name = string(fleet.CronMDMAndroidProfileManager)
					logger := cronLog
					androidProfileSchedule = schedule.New(
						ctx, name, s.T().Name(), 1*time.Hour, ds, ds,
						schedule.WithLogger(logger),
						schedule.WithJob("manage_android_profiles", func(ctx context.Context) error {
							logger.InfoContext(ctx, "Starting manage_android_profiles job", "test", s.T().Name(), "time", time.Now().Format(time.RFC3339))
							if s.onAndroidProfileJobDone != nil {
								defer func() {
									logger.InfoContext(ctx, "Completing manage_android_profiles job", "test", s.T().Name(), "time",
										time.Now().Format(time.RFC3339))
									s.onAndroidProfileJobDone()
								}()
							}
							err := android_service.ReconcileProfilesWithClient(ctx, ds, logger, "", androidMockClient, config.AndroidAgentConfig{
								Package:       "com.fleetdm.agent",
								SigningSHA256: "abc123def456",
							}, 0)
							require.NoError(s.T(), err)
							return err
						}),
					)
					return androidProfileSchedule, nil
				}
			},
			func(ctx context.Context, ds fleet.Datastore) fleet.NewCronScheduleFunc {
				return func() (fleet.CronSchedule, error) {
					const name = string(fleet.CronAppleMDMIPhoneIPadRefetcher)
					logger := cronLog
					refetcherSchedule := schedule.New(
						ctx, name, s.T().Name(), 1*time.Hour, ds, ds,
						schedule.WithLogger(logger),
						schedule.WithJob("cron_iphone_ipad_refetcher", func(ctx context.Context) error {
							return apple_mdm.IOSiPadOSRefetch(ctx, ds, mdmCommander, logger, s.fleetSvc.NewActivity)
						}),
					)
					return refetcherSchedule, nil
				}
			},
		},
		APNSTopic:         "com.apple.mgmt.External.10ac3ce5-4668-4e58-b69a-b2b5ce667589",
		EnableSCEPProxy:   true,
		WithDEPWebview:    true,
		SCEPConfigService: s.scepConfig,
		EnableSCIM:        true,
	}

	// generate ACME cert and key for testing ACME flows
	acmeCert, acmeKey, err := testhelpers.GenerateTestAttestationCA()
	require.NoError(s.T(), err)
	serverConfig.ACMECertCA = acmeCert
	serverConfig.ACMECertKey = acmeKey
	s.acmeCertCA = acmeCert
	s.acmeCertKey = acmeKey

	// ensure all our tests support challenges with invalid XML characters
	s.scepChallenge = "scepcha/><llenge"
	err = s.ds.InsertMDMConfigAssets(context.Background(), []fleet.MDMConfigAsset{
		{Name: fleet.MDMAssetSCEPChallenge, Value: []byte(s.scepChallenge)},
	}, nil)
	require.NoError(s.T(), err)

	svc, ctx := NewTestService(s.T(), s.ds, fleetCfg, &serverConfig)
	// This is a bit of a code smell but I don't see a better way to initialize this for the tests. The
	// initialization pattern works fine in our normal fleet server setup
	appleMDMJob.VPPInstaller = svc
	appleMDMJob.InHouseAppInstaller = svc
	softwareWorker.VPPInstaller = svc

	users, server := RunServerForTestsWithServiceWithDS(s.T(), ctx, s.ds, svc, &serverConfig)

	s.server = server
	s.users = users
	s.fleetSvc = svc
	s.token = s.getTestAdminToken()
	s.cachedAdminToken = s.token
	s.fleetCfg = fleetCfg
	s.pushProvider = pushProvider
	s.depStorage = depStorage
	s.integrationsSchedule = integrationsSchedule
	s.profileSchedule = profileSchedule
	s.cleanupsSchedule = cleanupsSchedule
	s.androidProfileSchedule = androidProfileSchedule
	s.appleMDMWorkerSchedule = appleMDMWorkerSchedule
	s.appleOSUpdatesSchedule = appleOSUpdatesSchedule
	s.mdmStorage = mdmStorage
	s.mdmCommander = mdmCommander
	s.logger = serverLogger
	s.androidAPIClient = androidMockClient
	s.keyValueStore = keyValueStore

	// FIXME: Sort out dependency injection between Fleet service and ACME service, especially around
	// testing cron jobs. For now, this works because the ACME service is embedded in the Fleet
	// service, but it feels backwards in comparison to how things are injected by the Fleet
	// service into the cron. Something smells wrong here.
	acmeSvc, ok := svc.(fleet.ACMEWriteService)
	require.True(s.T(), ok, "fleet service should be ACMEWriteService")
	s.acmeSvc = acmeSvc

	fleetdmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := s.fleetDMNextCSRStatus.Swap(http.StatusOK)
		statusCode := status.(int)
		// Set x-exit header for invalid email domain errors (400 or 422 status)
		if statusCode == http.StatusBadRequest || statusCode == http.StatusUnprocessableEntity {
			w.Header().Set("x-exit", "invalidEmailDomain")
		}
		w.WriteHeader(statusCode)
		resp := fmt.Appendf(nil, "status: %d", status)
		if statusCode == http.StatusOK && strings.Contains(r.URL.RawQuery, "deliveryMethod=json") {
			rawBody, err := io.ReadAll(r.Body)
			require.NoError(s.T(), err)
			var req struct {
				UnsignedCSRData []byte `json:"unsignedCsrData"`
			}
			err = json.Unmarshal(rawBody, &req)
			require.NoError(s.T(), err)

			resp = []byte(
				fmt.Sprintf(
					`{"csr": %q}`,
					base64.StdEncoding.EncodeToString(req.UnsignedCSRData),
				),
			)
		}
		_, _ = w.Write(resp)
	}))

	if s.appleVPPConfigSrvConfig == nil {
		s.appleVPPConfigSrvConfig = &appleVPPConfigSrvConf{
			Assets:        defaultVPPAssetList,
			SerialNumbers: []string{"123", "456"},
			Location:      "Fleet Location One",
		}
	}

	s.appleVPPConfigSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle /assets/disassociate — must come before the /associate
		// catch-all below (HasSuffix prevents the substring "associate" from
		// swallowing this case). Records every call so tests can assert Fleet
		// released a reserved seat.
		if strings.HasSuffix(r.URL.Path, "/disassociate") {
			var req vpp.DisassociateAssetsRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			s.appleVPPConfigSrvConfig.Disassociations = append(s.appleVPPConfigSrvConfig.Disassociations, req)
			_, _ = w.Write([]byte(`{"eventId": "disassociate-evt"}`))
			return
		}
		// Handle /associate
		if strings.Contains(r.URL.Path, "associate") {
			var associations vpp.AssociateAssetsRequest

			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(&associations); err != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}

			fmt.Printf("Mock VPP Server: Trying to associate %v with %v\n", associations.SerialNumbers, associations.Assets)

			if len(associations.Assets) == 0 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				res := vpp.ErrorResponse{
					ErrorNumber:  9718,
					ErrorMessage: "This request doesn't contain an asset, which is a required argument. Change the request to provide an asset.",
				}
				if err := json.NewEncoder(w).Encode(res); err != nil {
					panic(err)
				}
				return
			}

			if len(associations.SerialNumbers) == 0 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				res := vpp.ErrorResponse{
					ErrorNumber:  9719,
					ErrorMessage: "Either clientUserIds or serialNumbers are required arguments. Change the request to provide assignable users and devices.",
				}
				if err := json.NewEncoder(w).Encode(res); err != nil {
					panic(err)
				}
				return
			}

			var badAssets []vpp.Asset
			for _, reqAsset := range associations.Assets {
				var found bool
				for _, goodAsset := range s.appleVPPConfigSrvConfig.Assets {
					if reqAsset == goodAsset {
						found = true
					}
				}
				if !found {
					badAssets = append(badAssets, reqAsset)
				}
			}

			var badSerials []string
			for _, reqSerial := range associations.SerialNumbers {
				var found bool
				for _, goodSerial := range s.appleVPPConfigSrvConfig.SerialNumbers {
					if reqSerial == goodSerial {
						found = true
					}
				}
				if !found {
					badSerials = append(badSerials, reqSerial)
				}
			}

			if len(badAssets) != 0 || len(badSerials) != 0 {
				errMsg := "error associating assets."
				if len(badAssets) > 0 {
					var badAdamIds []string
					for _, asset := range badAssets {
						badAdamIds = append(badAdamIds, asset.AdamID)
					}
					errMsg += fmt.Sprintf(" assets don't exist on account: %s.", strings.Join(badAdamIds, ", "))
				}
				if len(badSerials) > 0 {
					errMsg += fmt.Sprintf(" bad serials: %s.", strings.Join(badSerials, ", "))
				}
				res := vpp.ErrorResponse{
					ErrorInfo: vpp.ResponseErrorInfo{
						Assets:        badAssets,
						ClientUserIds: []string{"something"},
						SerialNumbers: badSerials,
					},
					// Not sure what error should be returned on each
					// error type
					ErrorNumber:  1,
					ErrorMessage: errMsg,
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				if err := json.NewEncoder(w).Encode(res); err != nil {
					panic(err)
				}
			}
			_, _ = w.Write([]byte(`{"eventId": "123-345"}`))
			return
		}

		// Handle /assets
		if strings.Contains(r.URL.Path, "assets") {
			w.Header().Set("Content-Type", "application/json")
			adamID := r.URL.Query().Get("adamId")
			var assets []vpp.Asset
			switch adamID {
			case "":
				assets = s.appleVPPConfigSrvConfig.Assets
			default:
				for _, a := range s.appleVPPConfigSrvConfig.Assets {
					if a.AdamID == adamID {
						assets = []vpp.Asset{a}
					}
				}
			}
			encoder := json.NewEncoder(w)
			err := encoder.Encode(map[string][]vpp.Asset{"assets": assets})
			if err != nil {
				panic(err)
			}
			return
		}

		// Handle /client/config
		resp := fmt.Appendf(nil, `{"locationName": "%s", "countryISO2ACode": "US"}`, s.appleVPPConfigSrvConfig.Location)
		if strings.Contains(r.URL.RawQuery, "invalidToken") {
			// This replicates the response sent back from Apple's VPP endpoints when an invalid
			// token is passed. For more details see:
			// https://developer.apple.com/documentation/devicemanagement/app_and_book_management/app_and_book_management_legacy/interpreting_error_codes
			// https://developer.apple.com/documentation/devicemanagement/client_config
			// https://developer.apple.com/documentation/devicemanagement/errorresponse
			// Note that the Apple server returns 200 in this case.
			resp = []byte(`{"errorNumber": 9622,"errorMessage": "Invalid authentication token"}`)
		}

		if strings.Contains(r.URL.RawQuery, "serverError") {
			resp = []byte(`{"errorNumber": 9603,"errorMessage": "Internal server error"}`)
			w.WriteHeader(http.StatusInternalServerError)
		}

		_, _ = w.Write(resp)
	}))

	// deviceFamilies: "mac" -> osx platform, "iphone" -> ios platform, "ipad" -> ios platform
	s.appleVPPProxySrvData = map[string]string{
		// macOS app
		"1": `{"id": "1", "attributes": {"name": "App 1", "platformAttributes": {"osx": {"bundleId": "a-1", "artwork": {"url": "https://example.com/images/1/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "1.0.0"}}}, "deviceFamilies": ["mac"]}}`,
		// macOS, iOS, iPadOS app
		"2": `{"id": "2", "attributes": {"name": "App 2", "platformAttributes": {"osx": {"bundleId": "b-2", "artwork": {"url": "https://example.com/images/2/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "2.0.1"}}, "ios": {"bundleId": "b-2", "artwork": {"url": "https://example.com/images/2/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "2.0.0"}}}, "deviceFamilies": ["mac", "iphone", "ipad"]}}`,
		// iPadOS app
		"3": `{"id": "3", "attributes": {"name": "App 3", "platformAttributes": {"ios": {"bundleId": "c-3", "artwork": {"url": "https://example.com/images/3/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "3.0.0"}}}, "deviceFamilies": ["ipad"]}}`,
		// macOS app
		"4": `{"id": "4", "attributes": {"name": "App 4", "platformAttributes": {"osx": {"bundleId": "d-4", "artwork": {"url": "https://example.com/images/4/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "4.0.0"}}}, "deviceFamilies": ["mac"]}}`,
		// App with 0 licenses - macOS app
		"5": `{"id": "5", "attributes": {"name": "App 5", "platformAttributes": {"osx": {"bundleId": "e-5", "artwork": {"url": "https://example.com/images/5/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "5.0.0"}}}, "deviceFamilies": ["mac"]}}`,
	}

	s.appleVPPProxySrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		adamIDString := r.URL.Query().Get("ids")
		adamIDs := strings.Split(adamIDString, ",")

		var objs []string
		for _, a := range adamIDs {
			data, ok := s.appleVPPProxySrvData[a]
			if !ok {
				continue
			}
			objs = append(objs, data)
		}

		_, _ = w.Write(fmt.Appendf(nil, `{"data": [%s]}`, strings.Join(objs, ",")))
	}))

	s.appleGDMFSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// load the test data from the file
		b, err := os.ReadFile("../mdm/apple/gdmf/testdata/gdmf.json")
		require.NoError(s.T(), err)
		_, err = w.Write(b)
		require.NoError(s.T(), err)
	}))
	dev_mode.SetOverride("FLEET_DEV_GDMF_URL", s.appleGDMFSrv.URL, s.T())

	s.T().Setenv("TEST_FLEETDM_API_URL", fleetdmSrv.URL)
	s.T().Setenv("FLEET_DEV_STOKEN_AUTHENTICATED_APPS_URL", s.appleVPPProxySrv.URL)

	s.mockedDownloadFleetdmMeta = fleetdbase.Metadata{
		MSIURL:           fmt.Sprintf("https://download-testing.fleetdm.com/archive/stable/%s/fleetd-base.msi", uuid.NewString()),
		MSISha256:        uuid.NewString(),
		PKGURL:           fmt.Sprintf("https://download-testing.fleetdm.com/archive/stable/%s/fleetd-base.pkg", uuid.NewString()),
		PKGSha256:        uuid.NewString(),
		ManifestPlistURL: fmt.Sprintf("https://download-testing.fleetdm.com/archive/stable/%s/fleetd-base-manifest.plist", uuid.NewString()),
		Version:          "2024-06-25_03-01-17",
	}
	downloadFleetdmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/stable/meta.json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			require.NoError(s.T(), json.NewEncoder(w).Encode(s.mockedDownloadFleetdmMeta))
		}
	}))
	s.T().Setenv("FLEET_DEV_DOWNLOAD_FLEETDM_URL", downloadFleetdmSrv.URL)
	s.T().Cleanup(downloadFleetdmSrv.Close)

	appConf, err = s.ds.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.ServerSettings.ServerURL = server.URL
	appConf.Features.EnableSoftwareInventory = true
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)

	// enable MDM flows
	s.appleCoreCertsSetup()

	// create a global enroll secret
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: "global-secret"}},
		},
	}, http.StatusOK, &applyResp)

	s.T().Cleanup(fleetdmSrv.Close)
	s.T().Cleanup(s.appleVPPConfigSrv.Close)
	s.T().Cleanup(s.appleVPPProxySrv.Close)
	s.T().Cleanup(s.appleGDMFSrv.Close)

	// Setup fake JWKs server for Azure JWT verification
	metadata := jwkset.JWKMetadataOptions{
		KID: defaultFakeJWTKeyID,
	}
	options := jwkset.JWKOptions{
		Metadata: metadata,
	}

	// Create the JWK from the key and options.
	jwk, err := jwkset.NewJWKFromKey(testKey, options)
	require.NoError(s.T(), err)

	jwkSet := jwkset.NewMemoryStorage()

	err = jwkSet.KeyWrite(ctx, jwk)
	require.NoError(s.T(), err)

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/jwks.json" {
			response, err := jwkSet.JSONPublic(r.Context())
			require.NoError(s.T(), err)

			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write(response)
			require.NoError(s.T(), err)
		}
	}))
	s.T().Cleanup(jwksServer.Close)
	s.jwtSigningKey = testKey
	s.T().Setenv("FLEET_DEV_AZURE_JWT_JWKS_URI", jwksServer.URL+"/jwks.json")

	dev_mode.SetOverride("FLEET_DEV_BATCH_RETRY_INTERVAL", "1s")
}

func (s *integrationMDMTestSuite) TearDownSuite() {
	appConf, err := s.ds.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.MDM.EnabledAndConfigured = false
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)

	dev_mode.ClearOverride("FLEET_DEV_BATCH_RETRY_INTERVAL")
}

func (s *integrationMDMTestSuite) FailNextCSRRequestWith(status int) {
	s.fleetDMNextCSRStatus.Store(status)
}

func (s *integrationMDMTestSuite) SucceedNextCSRRequest() {
	s.fleetDMNextCSRStatus.Store(http.StatusOK)
}

func (s *integrationMDMTestSuite) TearDownTest() {
	t := s.T()
	ctx := context.Background()

	s.token = s.getTestAdminToken()
	appCfg := s.getConfig()
	// ensure windows mdm is always enabled for the next test
	appCfg.MDM.WindowsEnabledAndConfigured = true
	// ensure global disk encryption is disabled on exit. The flat toggle is
	// virtual (recomputed as the AND of the per-platform settings on save), so
	// the per-platform settings are what must be reset.
	appCfg.MDM.EnableDiskEncryption = optjson.SetBool(false)
	appCfg.MDM.MacOSSettings.EnableDiskEncryption = optjson.SetBool(false)
	appCfg.MDM.MacOSSettings.EnableEscrowDiskEncryptionKey = optjson.SetBool(false)
	appCfg.MDM.WindowsSettings.EnableDiskEncryption = optjson.SetBool(false)
	appCfg.MDM.WindowsSettings.RequireBitLockerPIN = optjson.SetBool(false)
	appCfg.MDM.RequireBitLockerPIN = optjson.SetBool(false)
	appCfg.MDM.LinuxSettings.EnableEscrowDiskEncryptionKey = optjson.SetBool(false)
	// ensure enable release manually is false
	appCfg.MDM.MacOSSetup.EnableReleaseDeviceManually = optjson.SetBool(false)
	// ensure global Windows OS updates are always disabled for the next test
	appCfg.MDM.WindowsUpdates = fleet.WindowsUpdates{}
	appCfg.MDM.EnabledAndConfigured = true
	appCfg.MDM.AppleBMEnabledAndConfigured = true
	appCfg.MDM.AndroidEnabledAndConfigured = true

	appCfg.MDM.EndUserAuthentication = fleet.MDMEndUserAuthentication{} // Reset end user auth

	// abm_tokens is truncated below, so the app config entries referencing them
	// must go too, otherwise the next test's config PATCH fails validation with
	// "token with organization name X doesn't exist"
	appCfg.MDM.AppleBusinessManager = optjson.Slice[fleet.MDMAppleABMAssignmentInfo]{}

	// ensure the server URL is constant
	appCfg.ServerSettings.ServerURL = s.server.URL
	err := s.ds.SaveAppConfig(ctx, &appCfg.AppConfig)
	require.NoError(t, err)

	s.withServer.commonTearDownTest(t)

	// use a sql statement to delete all profiles, since the datastore prevents
	// deleting the fleet-specific ones.
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM mdm_apple_configuration_profiles")
		return err
	})
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM mdm_windows_configuration_profiles")
		return err
	})
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM mdm_apple_bootstrap_packages")
		return err
	})
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM mdm_android_configuration_profiles")
		return err
	})

	// Delete certificate templates
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM host_certificate_templates")
		return err
	})
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM certificate_templates")
		return err
	})

	// Delete any CAs
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM certificate_authorities")
		return err
	})

	// clear any pending worker job
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM jobs")
		return err
	})

	// clear any host dep assignments
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM host_dep_assignments")
		return err
	})

	// clear any mdm windows enrollments
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM mdm_windows_enrollments")
		return err
	})

	// clear any lingering declarations
	mysqltest.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM mdm_apple_declarations")
		return err
	})
	mysqltest.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM host_mdm_apple_declarations")
		return err
	})
	mysqltest.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM mobile_device_management_solutions;")
		return err
	})
	mysqltest.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM host_mdm;")
		return err
	})
	mysqltest.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM abm_tokens;")
		return err
	})
	mysqltest.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM vpp_tokens;")
		return err
	})
	mysqltest.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM setup_experience_status_results;")
		return err
	})
	mysqltest.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM setup_experience_scripts;")
		return err
	})

	mysqltest.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM vpp_apps;")
		return err
	})

	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM nano_enrollment_queue")
		return err
	})
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM nano_command_results")
		return err
	})
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM nano_cert_auth_associations")
		return err
	})
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM nano_commands")
		return err
	})
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM nano_enrollments")
		return err
	})
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM nano_seen_times")
		return err
	})
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM nano_users")
		return err
	})
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM nano_devices")
		return err
	})

	err = s.ds.DeleteAllEnterprises(ctx)
	require.NoError(t, err)

	s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked = false
	s.androidAPIClient.EnterprisesPoliciesPatchFuncInvoked = false
	s.androidAPIClient.EnterprisesDevicesPatchFuncInvoked = false
}

// withDEPDeviceDetails answers Apple's "Get Device Details" endpoint, which host
// deletion consults to tell a device released from Apple Business apart from one
// Fleet still owns. Every requested serial is reported as still assigned, which is
// the state these suites set up; without this the lookup fails and deletes are
// refused. Tests needing a different answer should serve /devices themselves and
// bypass this wrapper.
func withDEPDeviceDetails(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/devices" {
			handler.ServeHTTP(w, r)
			return
		}

		var req struct {
			Devices []string `json:"devices"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// Fail loudly rather than answering an unreadable request, so a client
			// regression shows up here instead of as a puzzling downstream result.
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		devices := make(map[string]any, len(req.Devices))
		for _, serial := range req.Devices {
			devices[serial] = map[string]any{"serial_number": serial, "response_status": "SUCCESS"}
		}
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]any{"devices": devices}); err != nil {
			panic(err)
		}
	})
}

func (s *integrationMDMTestSuite) mockDEPResponse(orgName string, handler http.Handler) {
	t := s.T()
	srv := httptest.NewServer(withDEPDeviceDetails(handler))
	err := s.depStorage.StoreConfig(context.Background(), orgName, &nanodep_client.Config{BaseURL: srv.URL})
	depSvc := apple_mdm.NewDEPService(s.ds, s.depStorage, s.logger)
	require.NoError(t, depSvc.CreateDefaultAutomaticProfile(context.Background()))
	require.NoError(t, err)
	t.Cleanup(func() {
		srv.Close()
		err := s.depStorage.StoreConfig(context.Background(), orgName, &nanodep_client.Config{BaseURL: nanodep_client.DefaultBaseURL})
		require.NoError(t, err)
	})
}

func (s *integrationMDMTestSuite) awaitTriggerProfileSchedule(t *testing.T) {
	// three jobs running sequentially (macOS profiles and declarations, then Windows) on the same schedule
	t.Logf("[awaitTriggerProfileSchedule] Starting profile schedule trigger at %s", time.Now().Format(time.RFC3339))
	var wg sync.WaitGroup
	wg.Add(3)
	s.onProfileJobDone = wg.Done
	t.Logf("[awaitTriggerProfileSchedule] Triggering profile schedule...")

	var (
		didTrigger bool
		err        error
	)
	for range 10 {
		_, didTrigger, err = s.profileSchedule.Trigger(t.Context())
		require.NoError(t, err)
		if didTrigger {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	require.True(t, didTrigger, "profile schedule did not trigger after 1 second of retries")

	t.Logf("[awaitTriggerProfileSchedule] Waiting for 3 jobs to complete...")
	// Add timeout detection
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Logf("[awaitTriggerProfileSchedule] All jobs completed successfully at %s", time.Now().Format(time.RFC3339))
	case <-time.After(5 * time.Minute):
		t.Fatalf("Profile schedule jobs timed out after 5 minutes")
	}
}

func (s *integrationMDMTestSuite) awaitTriggerAndroidProfileSchedule(t *testing.T) {
	t.Logf("[awaitTriggerAndroidProfileSchedule] Starting Android profile schedule trigger at %s", time.Now().Format(time.RFC3339))
	var wg sync.WaitGroup
	wg.Add(1)
	s.onAndroidProfileJobDone = wg.Done
	t.Logf("[awaitTriggerAndroidProfileSchedule] Triggering Android profile schedule...")

	var (
		didTrigger bool
		err        error
	)
	for range 10 {
		_, didTrigger, err = s.androidProfileSchedule.Trigger(t.Context())
		require.NoError(t, err)
		if didTrigger {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	require.True(t, didTrigger, "android profile schedule did not trigger after 1 second of retries")

	t.Logf("[awaitTriggerAndroidProfileSchedule] Waiting for job to complete...")
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Logf("[awaitTriggerAndroidProfileSchedule] Job completed successfully at %s", time.Now().Format(time.RFC3339))
	case <-time.After(5 * time.Minute):
		t.Fatalf("Android profile schedule job timed out after 5 minutes")
	}
}

func checkNextPayloads(t *testing.T, mdmDevice *mdmtest.TestAppleMDMClient, forceDeviceErr bool) ([][]byte, []string) {
	installs := [][]byte{}
	removes := []string{}

	// on the first run, cmd will be nil and we need to
	// ping the server via idle
	// if after idle or acknowledge cmd is still nil, it
	// means there aren't any commands left to run
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		var fullCmd micromdm.CommandPayload
		switch cmd.Command.RequestType {
		case "InstallProfile":
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
			installs = append(installs, fullCmd.Command.InstallProfile.Payload)
		case "RemoveProfile":
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
			removes = append(removes, fullCmd.Command.RemoveProfile.Identifier)
		}

		if forceDeviceErr {
			cmd, err = mdmDevice.Err(cmd.CommandUUID, []mdm.ErrorChain{})
		} else {
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		}

		require.NoError(t, err)
	}

	return installs, removes
}

func setupExpectedFleetdProfile(t *testing.T, serverURL string, enrollSecret string, teamID *uint) []byte {
	var b bytes.Buffer
	params := mobileconfig.FleetdProfileOptions{
		EnrollSecret: enrollSecret,
		ServerURL:    serverURL,
		PayloadType:  mobileconfig.FleetdConfigPayloadIdentifier,
		PayloadName:  servermdm.FleetdConfigProfileName,
	}
	err := mobileconfig.FleetdProfileTemplate.Execute(&b, params)
	require.NoError(t, err)
	return b.Bytes()
}

func setupExpectedCAProfile(t *testing.T, ds *mysql.Datastore) []byte {
	assets, err := ds.GetAllMDMConfigAssetsByName(context.Background(), []fleet.MDMAssetName{
		fleet.MDMAssetCACert,
	}, nil)
	require.NoError(t, err)

	block, _ := pem.Decode(assets[fleet.MDMAssetCACert].Value)
	require.NotNil(t, block)
	require.Equal(t, "CERTIFICATE", block.Type)

	var b bytes.Buffer
	params := mobileconfig.FleetCARootTemplateOptions{
		PayloadName:       servermdm.FleetCAConfigProfileName,
		PayloadIdentifier: mobileconfig.FleetCARootConfigPayloadIdentifier,
		Certificate:       base64.StdEncoding.EncodeToString(block.Bytes),
	}
	err = mobileconfig.FleetCARootTemplate.Execute(&b, params)
	require.NoError(t, err)
	return b.Bytes()
}

func setupPusher(s *integrationMDMTestSuite, t *testing.T, mdmDevice *mdmtest.TestAppleMDMClient) {
	origPush := s.pushProvider.PushFunc
	s.pushProvider.PushFunc = func(_ context.Context, pushes []*mdm.Push) (map[string]*push.Response, error) {
		require.Len(t, pushes, 1)
		require.Equal(t, "pushmagic"+mdmDevice.SerialNumber, pushes[0].PushMagic)
		res := map[string]*push.Response{
			pushes[0].Token.String(): {
				Id:  uuid.New().String(),
				Err: nil,
			},
		}
		return res, nil
	}
	t.Cleanup(func() { s.pushProvider.PushFunc = origPush })
}

func createHostThenEnrollMDM(ds fleet.Datastore, fleetServerURL string, t *testing.T) (*fleet.Host, *mdmtest.TestAppleMDMClient) {
	fleetHost, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name() + uuid.New().String()),
		NodeKey:         ptr.String(t.Name() + uuid.New().String()),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
		HardwareModel:   "MacBookPro16,1",

		UUID:           strings.ToUpper(uuid.NewString()),
		HardwareSerial: mdmtest.RandSerialNumber(),
	})
	require.NoError(t, err)
	mdmDevice := enrollMacOSHostInMDMManually(t, fleetHost, ds, fleetServerURL)

	return fleetHost, mdmDevice
}

func enrollMacOSHostInMDMManually(t *testing.T, host *fleet.Host, ds fleet.Datastore, fleetServerURL string) *mdmtest.TestAppleMDMClient {
	desktopToken := uuid.New().String()
	mdmDevice := mdmtest.NewTestMDMClientAppleDesktopManual(fleetServerURL, desktopToken)
	mdmDevice.UUID = host.UUID
	mdmDevice.SerialNumber = host.HardwareSerial

	err := ds.SetOrUpdateDeviceAuthToken(context.Background(), host.ID, desktopToken)
	require.NoError(t, err)

	err = mdmDevice.Enroll()
	require.NoError(t, err)
	return mdmDevice
}

func (s *integrationMDMTestSuite) createAppleMobileHostThenDEPEnrollMDM(platform string, serial string) (*fleet.Host, *mdmtest.TestAppleMDMClient) {
	ctx := context.Background()
	t := s.T()

	model := "iPhone14,6"
	if platform == "ipados" {
		model = "iPad13,18"
	}

	// create a host with minimal information and the serial, no uuid/osquery id
	// (as when created via DEP sync).
	dbZeroTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	fleetHost, err := s.ds.NewHost(ctx, &fleet.Host{
		HardwareSerial:   serial,
		HardwareModel:    model,
		Platform:         platform,
		LastEnrolledAt:   dbZeroTime,
		DetailUpdatedAt:  time.Now(), // so that we don't trigger a cron detail update
		RefetchRequested: true,
	})
	require.NoError(t, err)
	require.Equal(t, dbZeroTime, fleetHost.LastEnrolledAt)

	s.createDEPAssignmentForHost(t, fleetHost)

	depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
	mdmDevice := mdmtest.NewTestMDMClientAppleDEPFromDevice(s.server.URL, depURLToken, serial, model)
	mdmDevice.SerialNumber = serial
	err = mdmDevice.Enroll()
	require.NoError(t, err)

	return fleetHost, mdmDevice
}

// createDEPAssignmentForHost records the host as DEP-assigned to Fleet, as the
// DEP sync would, so it can enroll through the automatic enrollment endpoint.
func (s *integrationMDMTestSuite) createDEPAssignmentForHost(t *testing.T, host *fleet.Host) {
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(t.Context(), `
			INSERT INTO host_dep_assignments (host_id, hardware_serial)
			VALUES (?, ?)
			ON DUPLICATE KEY UPDATE deleted_at = NULL, hardware_serial = VALUES(hardware_serial)`,
			host.ID, host.HardwareSerial)
		return err
	})
}

func (s *integrationMDMTestSuite) createAppleMobileHostThenEnrollMDM(platform string) (*fleet.Host, *mdmtest.TestAppleMDMClient) {
	ctx := context.Background()
	t := s.T()

	model := "iPhone14,6"
	if platform == "ipados" {
		model = "iPad13,18"
	}

	// create a host with minimal information and the serial, no uuid/osquery id
	// (as when created via DEP sync).
	dbZeroTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	serialNumber := mdmtest.RandSerialNumber()
	fleetHost, err := s.ds.NewHost(ctx, &fleet.Host{
		UUID:             uuid.NewString(),
		HardwareSerial:   serialNumber,
		HardwareModel:    model,
		Platform:         platform,
		LastEnrolledAt:   dbZeroTime,
		DetailUpdatedAt:  time.Now(), // so that we don't trigger a cron detail update
		RefetchRequested: true,
	})
	require.NoError(t, err)
	require.Equal(t, dbZeroTime, fleetHost.LastEnrolledAt)

	// Perform the MDM enrollment.
	mdmEnrollInfo := mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}

	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, model)
	mdmDevice.SerialNumber = serialNumber
	mdmDevice.UUID = fleetHost.UUID
	err = mdmDevice.Enroll()
	require.NoError(t, err)

	return fleetHost, mdmDevice
}

// Simulates an iPod Touch MDM enrollment. We made this a separate method
// becuase this is not a device type we are fully supporting in fleet,
// so we want to keep it isolated from the iPhone/iPad host creation methods.
// Functionally it is the same as createAppleMobileHostThenEnrollMDM with
// an "ios" platform argument, and we expect the device to behave the same way
// as an iPhone device.
func (s *integrationMDMTestSuite) createIpodHostThenEnrollMDM() (*fleet.Host, *mdmtest.TestAppleMDMClient) {
	ctx := context.Background()
	t := s.T()

	// for some reason this is the model name apple uses for an iPod Touch 7th gen
	model := "iPod9,1"

	// create a host with minimal information and the serial, no uuid/osquery id
	// (as when created via DEP sync).
	dbZeroTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	serialNumber := mdmtest.RandSerialNumber()
	fleetHost, err := s.ds.NewHost(ctx, &fleet.Host{
		UUID:           uuid.NewString(),
		HardwareSerial: serialNumber,
		HardwareModel:  model, Platform: "ios",
		LastEnrolledAt:   dbZeroTime,
		DetailUpdatedAt:  time.Now(), // so that we don't trigger a cron detail update
		RefetchRequested: true,
	})
	require.NoError(t, err)
	require.Equal(t, dbZeroTime, fleetHost.LastEnrolledAt)

	// Perform the MDM enrollment.
	mdmEnrollInfo := mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}

	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, model)
	mdmDevice.SerialNumber = serialNumber
	mdmDevice.UUID = fleetHost.UUID
	err = mdmDevice.Enroll()
	require.NoError(t, err)

	return fleetHost, mdmDevice
}

func createWindowsHostThenEnrollMDM(ds fleet.Datastore, fleetServerURL string, t *testing.T) (*fleet.Host, *mdmtest.TestWindowsMDMClient) {
	host := createOrbitEnrolledHost(t, "windows", uuid.NewString(), ds)
	mdmDevice := enrollWindowsHostInMDMViaOrbit(t, host, ds, fleetServerURL)
	return host, mdmDevice
}

func enrollWindowsHostInMDMViaOrbit(t *testing.T, host *fleet.Host, ds fleet.Datastore, fleetServerURL string) *mdmtest.TestWindowsMDMClient {
	mdmDevice := mdmtest.NewTestMDMClientWindowsProgramatic(fleetServerURL, *host.OrbitNodeKey)
	err := mdmDevice.Enroll()
	require.NoError(t, err)
	_, err = ds.UpdateMDMWindowsEnrollmentsHostUUID(context.Background(), host.UUID, mdmDevice.DeviceID)
	require.NoError(t, err)
	err = ds.SetOrUpdateMDMData(context.Background(), host.ID, false, true, fleetServerURL, false, fleet.WellKnownMDMFleet, "", fleet.PersonalEnrollmentTypeNone)
	require.NoError(t, err)
	return mdmDevice
}

// Note that this method only creates the MDM Enrollment but it will still need to be linked to the host record either
// via DS methods or by simualting a refetch.
func (s *integrationMDMTestSuite) enrollWindowsMDMViaSettingsApp(fleetServerURL, email, tenantID string, opts ...mdmtest.TestWindowsMDMClientOption) *mdmtest.TestWindowsMDMClient {
	opts = append([]mdmtest.TestWindowsMDMClientOption{
		mdmtest.TestWindowsMDMClientNotInOOBE(),
		mdmtest.TestWindowsMDMClientWithSigningKeyAndTenantID(s.jwtSigningKey, defaultFakeJWTKeyID, tenantID),
	}, opts...)
	mdmDevice := mdmtest.NewTestMDMClientWindowsAutomatic(fleetServerURL, email, opts...)
	err := mdmDevice.Enroll()
	require.NoError(s.T(), err)
	return mdmDevice
}

func (s *integrationMDMTestSuite) enrollWindowsHostInMDMViaAutopilot(fleetServerURL, email, tenantID string, opts ...mdmtest.TestWindowsMDMClientOption) *mdmtest.TestWindowsMDMClient {
	opts = append([]mdmtest.TestWindowsMDMClientOption{
		mdmtest.TestWindowsMDMClientWithSigningKeyAndTenantID(s.jwtSigningKey, defaultFakeJWTKeyID, tenantID),
	}, opts...)
	mdmDevice := mdmtest.NewTestMDMClientWindowsAutomatic(fleetServerURL, email, opts...)
	err := mdmDevice.Enroll()
	require.NoError(s.T(), err)
	return mdmDevice
}

func loadEnrollmentProfileDEPToken(t *testing.T, ds *mysql.Datastore) string {
	var token string
	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &token,
			`SELECT token FROM mdm_apple_enrollment_profiles`)
	})
	return token
}

func (s *integrationMDMTestSuite) uploadDataViaForm(endpoint, fieldName, fileName string, data []byte, expectedStatus int, wantErr string, response any) {
	s.uploadDataViaFormWithVerb(endpoint, "POST", fieldName, fileName, data, expectedStatus, wantErr, response)
}

func (s *integrationMDMTestSuite) uploadDataViaFormWithVerb(endpoint, verb, fieldName, fileName string, data []byte, expectedStatus int, wantErr string, response any) {
	t := s.T()

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// add the package field
	fw, err := w.CreateFormFile(fieldName, fileName)
	require.NoError(t, err)
	_, err = io.Copy(fw, bytes.NewBuffer(data))
	require.NoError(t, err)

	w.Close()

	headers := map[string]string{
		"Content-Type":  w.FormDataContentType(),
		"Accept":        "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", s.token),
	}

	res := s.DoRawWithHeaders(verb, endpoint, b.Bytes(), expectedStatus, headers)
	if wantErr != "" {
		errMsg := extractServerErrorText(res.Body)
		assert.Contains(t, errMsg, wantErr)
	}

	if response != nil {
		err := json.NewDecoder(res.Body).Decode(&response)
		require.NoError(t, err)
	}
}

// setSkipWorkerJobs sets the skipWorkerJobs flag to true for the duration of the test.
// This avoids running into timing issues with the test.
// We can manually run the jobs if needed with s.runWorker().
func (s *integrationMDMTestSuite) setSkipWorkerJobs(t *testing.T) {
	s.skipWorkerJobs.Store(true)
	t.Cleanup(func() { s.skipWorkerJobs.Store(false) })
}

// only asserts the profile identifier, status and operation (per host)
func (s *integrationMDMTestSuite) assertHostAppleConfigProfiles(want map[*fleet.Host][]fleet.HostMDMAppleProfile) {
	t := s.T()
	ds := s.ds
	ctx := context.Background()

	for h, wantProfs := range want {
		gotProfs, err := ds.GetHostMDMAppleProfiles(ctx, h.UUID)
		require.NoError(t, err)
		idents := make([]string, 0, len(gotProfs))
		for _, gp := range gotProfs {
			idents = append(idents, gp.Identifier)
		}
		require.Len(t, gotProfs, len(wantProfs), "apple host uuid: %s, profiles: %v - want %v", h.UUID, idents, wantProfs)

		sort.Slice(gotProfs, func(i, j int) bool {
			l, r := gotProfs[i], gotProfs[j]
			return l.Identifier < r.Identifier
		})
		sort.Slice(wantProfs, func(i, j int) bool {
			l, r := wantProfs[i], wantProfs[j]
			return l.Identifier < r.Identifier
		})
		for i, wp := range wantProfs {
			gp := gotProfs[i] // nolint:nilaway // require.Len above guarantees the index exists
			require.Equal(t, wp.Identifier, gp.Identifier, "host uuid: %s, prof id: %s", h.UUID, gp.Identifier)
			require.Equal(t, wp.OperationType, gp.OperationType, "host uuid: %s, prof id: %s", h.UUID, gp.Identifier)
			require.Equal(t, *wp.Status, *gp.Status, "host uuid: %s, prof id: %s", h.UUID, gp.Identifier)
		}
	}
}

// only asserts the profile name, status and operation (per host)
func (s *integrationMDMTestSuite) assertHostWindowsConfigProfiles(want map[*fleet.Host][]fleet.HostMDMWindowsProfile) {
	t := s.T()
	ds := s.ds
	ctx := context.Background()

	for h, wantProfs := range want {
		gotProfs, err := ds.GetHostMDMWindowsProfiles(ctx, h.UUID)
		require.NoError(t, err)
		require.Len(t, gotProfs, len(wantProfs), "host uuid: %s", h.UUID)

		sort.Slice(gotProfs, func(i, j int) bool {
			l, r := gotProfs[i], gotProfs[j]
			return l.Name < r.Name
		})
		sort.Slice(wantProfs, func(i, j int) bool {
			l, r := wantProfs[i], wantProfs[j]
			return l.Name < r.Name
		})
		for i, wp := range wantProfs {
			gp := gotProfs[i] // nolint:nilaway // require.Len above guarantees the index exists
			require.Equal(t, wp.Name, gp.Name, "host uuid: %s, prof id: %s", h.UUID, gp.Name)
			require.Equal(t, wp.OperationType, gp.OperationType, "host uuid: %s, prof id: %s", h.UUID, gp.Name)
			require.Equal(t, wp.Status, gp.Status, "host uuid: %s, prof id: %s", h.UUID, gp.Name)
		}
	}
}

func (s *integrationMDMTestSuite) assertConfigProfilesByIdentifier(teamID *uint, profileIdent string, exists bool) (profile *fleet.MDMAppleConfigProfile) {
	t := s.T()
	if teamID == nil {
		teamID = new(uint(0))
	}
	var cfgProfs []*fleet.MDMAppleConfigProfile
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(context.Background(), q, &cfgProfs, `SELECT * FROM mdm_apple_configuration_profiles WHERE team_id = ?`, teamID)
	})

	label := "exist"
	if !exists {
		label = "not exist"
	}
	require.Condition(t, func() bool {
		for _, p := range cfgProfs {
			if p.Identifier == profileIdent {
				profile = p
				return exists // success if we want it to exist, failure if we don't
			}
		}
		return !exists
	}, "a config profile must %s with identifier: %s", label, profileIdent)

	return profile
}

func (s *integrationMDMTestSuite) assertMacOSConfigProfilesByName(teamID *uint, profileName string, exists bool) {
	t := s.T()
	if teamID == nil {
		teamID = new(uint(0))
	}
	var cfgProfs []*fleet.MDMAppleConfigProfile
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(context.Background(), q, &cfgProfs, `SELECT name FROM mdm_apple_configuration_profiles WHERE team_id = ?`, teamID)
	})

	label := "exist"
	if !exists {
		label = "not exist"
	}
	require.Condition(t, func() bool {
		for _, p := range cfgProfs {
			if p.Name == profileName {
				return exists // success if we want it to exist, failure if we don't
			}
		}
		return !exists
	}, "a config profile must %s with name: %s", label, profileName)
}

func (s *integrationMDMTestSuite) assertMacOSDeclarationsByName(teamID *uint, declarationName string, exists bool) {
	t := s.T()
	if teamID == nil {
		teamID = new(uint(0))
	}
	var cfgProfs []*fleet.MDMAppleConfigProfile
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(context.Background(), q, &cfgProfs, `SELECT name FROM mdm_apple_declarations WHERE team_id = ?`, teamID)
	})

	label := "exist"
	if !exists {
		label = "not exist"
	}
	require.Condition(t, func() bool {
		for _, p := range cfgProfs {
			if p.Name == declarationName {
				return exists // success if we want it to exist, failure if we don't
			}
		}
		return !exists
	}, "a config profile must %s with name: %s", label, declarationName)
}

func (s *integrationMDMTestSuite) assertWindowsConfigProfilesByName(teamID *uint, profileName string, exists bool) {
	t := s.T()
	if teamID == nil {
		teamID = new(uint(0))
	}
	var cfgProfs []*fleet.MDMWindowsConfigProfile
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(context.Background(), q, &cfgProfs,
			`SELECT profile_uuid, team_id, name, syncml, created_at, uploaded_at FROM mdm_windows_configuration_profiles WHERE team_id = ?`,
			teamID)
	})

	label := "exist"
	if !exists {
		label = "not exist"
	}
	require.Condition(t, func() bool {
		for _, p := range cfgProfs {
			if p.Name == profileName {
				return exists // success if we want it to exist, failure if we don't
			}
		}
		return !exists
	}, "a config profile must %s with name: %s", label, profileName)
}

// generates the body and headers part of a multipart request ready to be
// used via s.DoRawWithHeaders to POST /api/_version_/fleet/mdm/apple/profiles.
func generateNewProfileMultipartRequest(t *testing.T,
	fileName string, fileContent []byte, token string, extraFields map[string][]string,
) (*bytes.Buffer, map[string]string) {
	return generateMultipartRequest(t, "profile", fileName, fileContent, token, extraFields)
}

func generateMultipartRequest(t *testing.T,
	uploadFileField, fileName string, fileContent []byte, token string,
	extraFields map[string][]string,
) (*bytes.Buffer, map[string]string) {
	return generateMultipartRequestWithFiles(t, uploadFileField, fileName, fileContent, token, extraFields, nil)
}

// multipartFile is an additional file part for endpoints that accept more than
// one, such as a configuration profile uploaded together with its custom DDM
// activation.
type multipartFile struct {
	fileName string
	content  []byte
}

// generateMultipartRequestWithFiles builds a multipart body with a primary file
// part plus any number of additional file parts, keyed by form field name.
// generateMultipartRequest delegates here so single-file callers are unchanged.
func generateMultipartRequestWithFiles(t *testing.T,
	uploadFileField, fileName string, fileContent []byte, token string,
	extraFields map[string][]string, extraFiles map[string]multipartFile,
) (*bytes.Buffer, map[string]string) {
	var body bytes.Buffer

	writer := multipart.NewWriter(&body)

	// add file content
	if fileName != "" || len(fileContent) > 0 {
		ff, err := writer.CreateFormFile(uploadFileField, fileName)
		require.NoError(t, err)
		_, err = io.Copy(ff, bytes.NewReader(fileContent))
		require.NoError(t, err)
	}

	// add any additional file parts
	for field, f := range extraFiles {
		ff, err := writer.CreateFormFile(field, f.fileName)
		require.NoError(t, err)
		_, err = io.Copy(ff, bytes.NewReader(f.content))
		require.NoError(t, err)
	}

	// add extra fields
	for key, values := range extraFields {
		for _, value := range values {
			err := writer.WriteField(key, value)
			require.NoError(t, err)
		}
	}

	err := writer.Close()
	require.NoError(t, err)

	headers := map[string]string{
		"Content-Type":  writer.FormDataContentType(),
		"Accept":        "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", token),
	}
	return &body, headers
}

func (s *integrationMDMTestSuite) uploadBootstrapPackage(
	pkg *fleet.MDMAppleBootstrapPackage,
	expectedStatus int,
	wantErr string,
	dryRun bool,
) {
	t := s.T()

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// add the package field
	fw, err := w.CreateFormFile("package", pkg.Name)
	require.NoError(t, err)
	_, err = io.Copy(fw, bytes.NewBuffer(pkg.Bytes))
	require.NoError(t, err)

	// add the team_id field
	err = w.WriteField("team_id", fmt.Sprint(pkg.TeamID))
	require.NoError(t, err)

	w.Close()

	headers := map[string]string{
		"Content-Type":  w.FormDataContentType(),
		"Accept":        "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", s.token),
	}

	res := s.DoRawWithHeaders("POST", "/api/latest/fleet/bootstrap", b.Bytes(), expectedStatus, headers, "dry_run", strconv.FormatBool(dryRun))

	if wantErr != "" {
		errMsg := extractServerErrorText(res.Body)
		assert.Contains(t, errMsg, wantErr)
	}
}

func (s *integrationMDMTestSuite) uploadEULA(eula *fleet.MDMEULA, expectedStatus int, wantErr string) {
	s.uploadEULAFile("", "", eula, expectedStatus, wantErr)
}

// uploadEULAFile posts eula with platform in the form, left out when empty. query is appended to the path.
func (s *integrationMDMTestSuite) uploadEULAFile(platform, query string, eula *fleet.MDMEULA, expectedStatus int, wantErr string) {
	t := s.T()

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	if platform != "" {
		require.NoError(t, w.WriteField("platform", platform))
	}
	fw, err := w.CreateFormFile("eula", eula.Name)
	require.NoError(t, err)
	_, err = io.Copy(fw, bytes.NewBuffer(eula.Bytes))
	require.NoError(t, err)
	w.Close()

	headers := map[string]string{
		"Content-Type":  w.FormDataContentType(),
		"Accept":        "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", s.token),
	}

	res := s.DoRawWithHeaders("POST", "/api/latest/fleet/setup_experience/eula"+query, b.Bytes(), expectedStatus, headers)

	if wantErr != "" {
		errMsg := extractServerErrorText(res.Body)
		assert.Contains(t, errMsg, wantErr)
	}
}

func (s *integrationMDMTestSuite) setTokenForTest(t *testing.T, email, password string) {
	oldToken := s.token
	t.Cleanup(func() {
		s.token = oldToken
	})

	s.token = s.getCachedUserToken(email, password)
}

// Note if you call this function it will change your server URL which may cause certain downstream
// things in your test to fail. Either change the SSO Server URL back after doing SSO or make changes
// in your test to account for the hardcoded, wrong server URL of https://localhost:8080 being used
// in places like enrollment profiles
func (s *integrationMDMTestSuite) setUpMDMSSO(t *testing.T, lockEndUserInfo bool) appConfigResponse {
	// MDM SSO fields are empty by default
	acResp := appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.Empty(t, acResp.MDM.EndUserAuthentication.SSOProviderSettings)

	// set the SSO fields
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
			"server_settings": {
				"server_url": "https://localhost:8080"
			},
			"mdm": {
				"end_user_authentication": {
					"entity_id": "mdm.test.com",
					"idp_name": "SimpleSAML",
					"metadata_url": "%s"
				},
				"macos_setup": {
					"enable_end_user_authentication": true,
					"lock_end_user_info": %t
				}
			}
		}`, testSAMLIDPMetadataURL, lockEndUserInfo)), http.StatusOK, &acResp)
	wantSettings := fleet.SSOProviderSettings{
		EntityID:    "mdm.test.com",
		IDPName:     "SimpleSAML",
		MetadataURL: testSAMLIDPMetadataURL,
	}
	assert.Equal(t, wantSettings, acResp.MDM.EndUserAuthentication.SSOProviderSettings)

	t.Cleanup(func() {
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
				"server_settings": {
					"server_url": "`+s.server.URL+`"
				},
				"mdm": {
					"end_user_authentication": {
						"entity_id": "",
						"idp_name": "",
						"metadata_url": ""
					},
					"macos_setup": {
						"enable_end_user_authentication": false,
						"lock_end_user_info": false
					}
				}
			}`), http.StatusOK, &acResp)
	})

	// check that they are returned by a GET /config
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.Equal(t, wantSettings, acResp.MDM.EndUserAuthentication.SSOProviderSettings)
	return acResp
}

type scepPayload struct {
	Challenge string
	URL       string
}

type enrollmentPayload struct {
	PayloadType            string
	ServerURL              string      // used by the enrollment payload
	PayloadContent         scepPayload // scep contains a nested payload content dict
	AssignedManagedAppleID string      // used by account driven enrollment. Not always present
}

type enrollmentProfile struct {
	PayloadIdentifier string
	PayloadContent    []enrollmentPayload
}

type optsDownloadEnrollProf struct {
	basePath string
	token    string
	diParam  string
	diHeader string
}

func (s *integrationMDMTestSuite) downloadAndVerifyEnrollmentProfile(t *testing.T, opts optsDownloadEnrollProf) *enrollmentProfile {
	var headers map[string]string
	if opts.diHeader != "" {
		headers = map[string]string{"X-apple-aspen-deviceinfo": opts.diHeader}
	}

	path := opts.basePath
	if path == "" {
		path = "/api/mdm/apple/enroll"
	}

	var queryParams []string
	if opts.token != "" {
		queryParams = append(queryParams, "token", opts.token)
	}
	if opts.diParam != "" {
		queryParams = append(queryParams, "deviceinfo", opts.diParam)
	}

	resp := s.DoRawWithHeaders("GET", path, nil, http.StatusOK, headers, queryParams...)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, resp.Header, "Content-Disposition")
	require.Contains(t, resp.Header, "Content-Type")
	require.Contains(t, resp.Header, "X-Content-Type-Options")
	require.Contains(t, resp.Header.Get("Content-Disposition"), "attachment;")
	require.Contains(t, resp.Header.Get("Content-Type"), "application/x-apple-aspen-config")
	require.Contains(t, resp.Header.Get("X-Content-Type-Options"), "nosniff")
	headerLen, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	require.NoError(t, err)
	require.Equal(t, len(body), headerLen)

	return s.verifyEnrollmentProfile(body, "", "")
}

func (s *integrationMDMTestSuite) verifyEnrollmentProfile(rawProfile []byte, enrollmentRef, managedAppleAccount string) *enrollmentProfile {
	t := s.T()
	var profile enrollmentProfile

	if !bytes.HasPrefix(bytes.TrimSpace(rawProfile), []byte("<?xml")) {
		p7, err := pkcs7.Parse(rawProfile)
		require.NoError(t, err)
		rootCA := x509.NewCertPool()

		assets, err := s.ds.GetAllMDMConfigAssetsByName(context.Background(), []fleet.MDMAssetName{
			fleet.MDMAssetCACert,
		}, nil)
		require.NoError(t, err)

		require.True(t, rootCA.AppendCertsFromPEM(assets[fleet.MDMAssetCACert].Value))
		require.NoError(t, p7.VerifyWithChain(rootCA))
		rawProfile = p7.Content
	}

	require.NoError(t, plist.Unmarshal(rawProfile, &profile))

	for _, p := range profile.PayloadContent {
		switch p.PayloadType {
		case "com.apple.security.scep":
			require.Equal(t, s.getConfig().ServerSettings.ServerURL+apple_mdm.SCEPPath, p.PayloadContent.URL)
			// the profile carries either the static challenge or one minted for it
			if p.PayloadContent.Challenge != s.scepChallenge {
				var minted bool
				mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
					return sqlx.GetContext(t.Context(), q, &minted,
						`SELECT EXISTS(SELECT 1 FROM mdm_apple_scep_challenges WHERE challenge = ?)`, p.PayloadContent.Challenge)
				})
				require.True(t, minted, "unexpected SCEP challenge %q", p.PayloadContent.Challenge)
			}
		case "com.apple.mdm":
			require.Contains(t, p.ServerURL, s.getConfig().ServerSettings.ServerURL+apple_mdm.MDMPath)
			if enrollmentRef != "" {
				require.Contains(t, p.ServerURL, enrollmentRef)
			}
			if managedAppleAccount != "" {
				require.Equal(t, p.AssignedManagedAppleID, managedAppleAccount)
			}
		default:
			require.Failf(t, "unrecognized payload type in enrollment profile: %s", p.PayloadType)
		}
	}
	return &profile
}

func (s *integrationMDMTestSuite) runWorker() {
	err := s.worker.ProcessJobs(context.Background())
	require.NoError(s.T(), err)
	pending, err := s.ds.GetQueuedJobs(context.Background(), 1, time.Time{})
	require.NoError(s.T(), err)
	s.Require().Empty(pending)
}

// runWorkerUntilDone runs queued jobs until there are no more pending jobs
// this method handles the case when jobs could be queued asynchronously, such as during MDM enrollment
func (s *integrationMDMTestSuite) runWorkerUntilDone() {
	s.runWorkerUntilDoneWithChecks(false)
}

func (s *integrationMDMTestSuite) runWorkerUntilDoneWithChecks(failIfFailedJobs bool) {
	err := s.worker.ProcessJobs(s.T().Context())
	s.Require().NoError(err)

	// Retry checking for pending jobs
	s.Require().Eventually(func() bool {
		pending, err := s.ds.GetQueuedJobs(s.T().Context(), 1, time.Time{})
		s.Require().NoError(err)
		if len(pending) == 0 {
			return true
		}
		// Process any newly queued jobs
		s.T().Logf("Pending job: %s", pending[0].Name)
		err = s.worker.ProcessJobs(s.T().Context())
		return err == nil && len(pending) == 0
	}, 5*time.Second, 100*time.Millisecond, "jobs still pending after processing")

	if failIfFailedJobs {
		var failedJobs []*fleet.Job
		t := s.T()
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(t.Context(), q, &failedJobs, "SELECT name, args, state, error FROM jobs WHERE state NOT IN ('queued', 'success')")
		})
		s.Require().Empty(failedJobs, "some jobs have failed: %s", spew.Sdump(failedJobs))
	}
}

func (s *integrationMDMTestSuite) runDEPSchedule() {
	ctx := context.Background()
	fleetSyncer := apple_mdm.NewDEPService(s.ds, s.depStorage, s.logger)
	err := fleetSyncer.RunAssigner(ctx)
	s.Require().NoError(err)
}

func (s *integrationMDMTestSuite) runAppleOSUpdatesSchedule() {
	ch := make(chan bool)
	var once sync.Once
	s.onAppleOSUpdatesScheduleDone = func() {
		once.Do(func() { close(ch) })
	}

	var (
		didTrigger bool
		err        error
	)
	for range 10 {
		_, didTrigger, err = s.appleOSUpdatesSchedule.Trigger(s.T().Context())
		s.Require().NoError(err)
		if didTrigger {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	s.Require().True(didTrigger, "apple os updates schedule did not trigger after 1 second of retries")

	select {
	case <-ch:
	case <-time.After(30 * time.Second):
		s.T().Fatal("apple os updates schedule did not complete")
	}
}

func (s *integrationMDMTestSuite) runIntegrationsSchedule() {
	ch := make(chan bool)
	var once sync.Once
	s.onIntegrationsScheduleDone = func() {
		once.Do(func() { close(ch) })
	}

	var (
		didTrigger bool
		err        error
	)
	for range 10 {
		_, didTrigger, err = s.integrationsSchedule.Trigger(s.T().Context())
		s.Require().NoError(err)
		if didTrigger {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	s.Require().True(didTrigger, "integrations schedule did not trigger after 1 second of retries")

	<-ch
}

func (s *integrationMDMTestSuite) awaitRunAppleMDMWorkerSchedule() {
	var wg sync.WaitGroup
	wg.Add(1)
	s.onAppleMDMWorkerScheduleDone = wg.Done

	var (
		didTrigger bool
		err        error
	)
	for range 10 {
		_, didTrigger, err = s.appleMDMWorkerSchedule.Trigger(s.T().Context())
		s.Require().NoError(err)
		if didTrigger {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	s.Require().True(didTrigger, "apple MDM worker schedule did not trigger after 1 second of retries")

	// Add timeout detection
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Completed successfully
	case <-time.After(5 * time.Minute):
		s.T().Fatalf("Apple MDM worker schedule jobs timed out after 5 minutes")
	}
}

func (s *integrationMDMTestSuite) awaitRunCleanupSchedule() {
	var wg sync.WaitGroup
	wg.Add(1)
	s.onCleanupScheduleDone = wg.Done

	var (
		didTrigger bool
		err        error
	)
	for range 10 {
		_, didTrigger, err = s.cleanupsSchedule.Trigger(s.T().Context())
		require.NoError(s.T(), err)
		if didTrigger {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	s.Require().True(didTrigger, "cleanups schedule did not trigger after 1 second of retries")

	// Add timeout detection
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Completed successfully
	case <-time.After(5 * time.Minute):
		s.T().Fatalf("Cleanup schedule jobs timed out after 5 minutes")
	}
}

// enrollRequestHWDevID is the hardware ID every newSecurityTokenMsg request presents. A test enrolling a new host with it
// must first delete the enrollment a previous test left, which Fleet refuses to hand over to a different host.
const enrollRequestHWDevID = "CF1D12AA5AE42E47D52465E9A71316CAF3AFCC1D3088F230F4D50B371FB2256F"

func (s *integrationMDMTestSuite) newSecurityTokenMsg(encodedBinToken string, deviceToken bool, missingContextItem bool) ([]byte, error) {
	if len(encodedBinToken) == 0 {
		return nil, errors.New("encodedBinToken is empty")
	}

	var reqSecTokenContextItemDeviceType []byte
	if !missingContextItem {
		reqSecTokenContextItemDeviceType = []byte(
			`<ac:ContextItem Name="DeviceType">
			 <ac:Value>CIMClient_Windows</ac:Value>
			 </ac:ContextItem>`,
		)
	}

	// JWT token by default
	tokType := syncml.BinarySecurityAzureEnroll
	if deviceToken {
		tokType = syncml.BinarySecurityDeviceEnroll
	}

	// Preparing the RequestSecurityToken Request message
	requestBytes := []byte(
		`<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://www.w3.org/2005/08/addressing" xmlns:u="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd" xmlns:wsse="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd" xmlns:wst="http://docs.oasis-open.org/ws-sx/ws-trust/200512" xmlns:ac="http://schemas.xmlsoap.org/ws/2006/12/authorization">
			<s:Header>
				<a:Action s:mustUnderstand="1">http://schemas.microsoft.com/windows/pki/2009/01/enrollment/RST/wstep</a:Action>
				<a:MessageID>urn:uuid:0d5a1441-5891-453b-becf-a2e5f6ea3749</a:MessageID>
				<a:ReplyTo>
				<a:Address>http://www.w3.org/2005/08/addressing/anonymous</a:Address>
				</a:ReplyTo>
				<a:To s:mustUnderstand="1">https://mdmwindows.com/EnrollmentServer/Enrollment.svc</a:To>
				<wsse:Security s:mustUnderstand="1">
				<wsse:BinarySecurityToken ValueType="` + tokType + `" EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd#base64binary">` + encodedBinToken + `</wsse:BinarySecurityToken>
				</wsse:Security>
			</s:Header>
			<s:Body>
				<wst:RequestSecurityToken>
				<wst:TokenType>http://schemas.microsoft.com/5.0.0.0/ConfigurationManager/Enrollment/DeviceEnrollmentToken</wst:TokenType>
				<wst:RequestType>http://docs.oasis-open.org/ws-sx/ws-trust/200512/Issue</wst:RequestType>
				<wsse:BinarySecurityToken ValueType="http://schemas.microsoft.com/windows/pki/2009/01/enrollment#PKCS10" EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd#base64binary">MIICzjCCAboCAQAwSzFJMEcGA1UEAxNAMkI5QjUyQUMtREYzOC00MTYxLTgxNDItRjRCMUUwIURCMjU3QzNBMDg3NzhGNEZCNjFFMjc0OTA2NkMxRjI3ADCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAKogsEpbKL8fuXpTNAE5RTZim8JO5CCpxj3z+SuWabs/s9Zse6RziKr12R4BXPiYE1zb8god4kXxet8x3ilGqAOoXKkdFTdNkdVa23PEMrIZSX5MuQ7mwGtctayARxmDvsWRF/icxJbqSO+bYIKvuifesOCHW2cJ1K+JSKijTMik1N8NFbLi5fg1J+xImT9dW1z2fLhQ7SNEMLosUPHsbU9WKoDBfnPsLHzmhM2IMw+5dICZRoxHZalh70FefBk0XoT8b6w4TIvc8572TyPvvdwhc5o/dvyR3nAwTmJpjBs1YhJfSdP+EBN1IC2T/i/mLNUuzUSC2OwiHPbZ6MMr/hUCAwEAAaBCMEAGCSqGSIb3DQEJDjEzMDEwLwYKKwYBBAGCN0IBAAQhREIyNTdDM0EwODc3OEY0RkI2MUUyNzQ5MDY2QzFGMjcAMAkGBSsOAwIdBQADggEBACQtxyy74sCQjZglwdh/Ggs6ofMvnWLMq9A9rGZyxAni66XqDUoOg5PzRtSt+Gv5vdLQyjsBYVzo42W2HCXLD2sErXWwh/w0k4H7vcRKgEqv6VYzpZ/YRVaewLYPcqo4g9NoXnbW345OPLwT3wFvVR5v7HnD8LB2wHcnMu0fAQORgafCRWJL1lgw8VZRaGw9BwQXCF/OrBNJP1ivgqtRdbSoH9TD4zivlFFa+8VDz76y2mpfo0NbbD+P0mh4r0FOJan3X9bLswOLFD6oTiyXHgcVSzLN0bQ6aQo0qKp3yFZYc8W4SgGdEl07IqNquKqJ/1fvmWxnXEbl3jXwb1efhbM=</wsse:BinarySecurityToken>
				<ac:AdditionalContext xmlns="http://schemas.xmlsoap.org/ws/2006/12/authorization">
					<ac:ContextItem Name="UXInitiated">
					<ac:Value>false</ac:Value>
					</ac:ContextItem>
					<ac:ContextItem Name="HWDevID">
					<ac:Value>` + enrollRequestHWDevID + `</ac:Value>
					</ac:ContextItem>
					<ac:ContextItem Name="Locale">
					<ac:Value>en-US</ac:Value>
					</ac:ContextItem>
					<ac:ContextItem Name="TargetedUserLoggedIn">
					<ac:Value>true</ac:Value>
					</ac:ContextItem>
					<ac:ContextItem Name="OSEdition">
					<ac:Value>48</ac:Value>
					</ac:ContextItem>
					<ac:ContextItem Name="DeviceName">
					<ac:Value>DESKTOP-0C89RC0</ac:Value>
					</ac:ContextItem>
					<ac:ContextItem Name="MAC">
					<ac:Value>01-1C-29-7B-3E-1C</ac:Value>
					</ac:ContextItem>
					<ac:ContextItem Name="MAC">
					<ac:Value>01-0C-21-7B-3E-52</ac:Value>
					</ac:ContextItem>
					<ac:ContextItem Name="DeviceID">
					<ac:Value>AB157C3A18778F4FB21E2739066C1F27</ac:Value>
					</ac:ContextItem>
					<ac:ContextItem Name="EnrollmentType">
					<ac:Value>Full</ac:Value>
					</ac:ContextItem>
					` + string(reqSecTokenContextItemDeviceType) + `
					<ac:ContextItem Name="OSVersion">
					<ac:Value>10.0.19045.2965</ac:Value>
					</ac:ContextItem>
					<ac:ContextItem Name="ApplicationVersion">
					<ac:Value>10.0.19045.1965</ac:Value>
					</ac:ContextItem>
					<ac:ContextItem Name="NotInOobe">
					<ac:Value>false</ac:Value>
					</ac:ContextItem>
					<ac:ContextItem Name="RequestVersion">
					<ac:Value>5.0</ac:Value>
					</ac:ContextItem>
				</ac:AdditionalContext>
				</wst:RequestSecurityToken>
			</s:Body>
			</s:Envelope>
		`,
	)

	return requestBytes, nil
}

func (s *integrationMDMTestSuite) checkMDMProfilesSummaries(t *testing.T, teamID *uint, expectedSummary fleet.MDMProfilesSummary, expectedAppleSummary *fleet.MDMProfilesSummary) {
	// The Windows profiles summary reads the maintained host_mdm_windows_profiles_status rollup. Some tests simulate device reports
	// by writing host_mdm_windows_profiles directly, bypassing the write paths that maintain the rollup, so reconcile it before
	// reading.
	require.NoError(t, s.ds.ReconcileWindowsProfilesStatus(t.Context()))

	var queryParams []string
	if teamID != nil {
		queryParams = append(queryParams, "team_id", fmt.Sprintf("%d", *teamID))
	}

	if expectedAppleSummary != nil {
		var apple getMDMAppleProfilesSummaryResponse
		s.DoJSON("GET", "/api/v1/fleet/mdm/apple/profiles/summary", getMDMAppleProfilesSummaryRequest{}, http.StatusOK, &apple, queryParams...)
		require.Equal(t, expectedSummary.Failed, apple.Failed, "failed summary count doesn't match")
		require.Equal(t, expectedSummary.Pending, apple.Pending, "pending summary count doesn't match")
		require.Equal(t, expectedSummary.Verifying, apple.Verifying, "verifying summary count doesn't match")
		require.Equal(t, expectedSummary.Verified, apple.Verified, "verified summary count doesn't match")
	}

	var combined getMDMProfilesSummaryResponse
	s.DoJSON("GET", "/api/v1/fleet/configuration_profiles/summary", getMDMProfilesSummaryRequest{}, http.StatusOK, &combined, queryParams...)
	require.Equal(t, expectedSummary.Failed, combined.Failed, "failed summary count doesn't match")
	require.Equal(t, expectedSummary.Pending, combined.Pending, "pending summary count doesn't match")
	require.Equal(t, expectedSummary.Verifying, combined.Verifying, "verifying summary count doesn't match")
	require.Equal(t, expectedSummary.Verified, combined.Verified, "verified summary count doesn't match")
}

func (s *integrationMDMTestSuite) enableABM(orgName string) *fleet.ABMToken {
	t := s.T()
	var abmResp generateABMKeyPairResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/ab_public_key", nil, http.StatusOK, &abmResp)
	require.NoError(t, abmResp.Err)
	require.NotEmpty(t, abmResp.PublicKey)
	block, _ := pem.Decode(abmResp.PublicKey)
	require.NotNil(t, block)
	require.Equal(t, "CERTIFICATE", block.Type)

	// try to upload an invalid token
	s.uploadABMToken([]byte("foo"), http.StatusBadRequest, "Invalid token. Please provide a valid token from Apple Business.")

	// generate a mock token and encrypt it using the public key
	testBMToken := &nanodep_client.OAuth1Tokens{
		ConsumerKey:    "test_consumer",
		ConsumerSecret: "test_secret",
		AccessToken:    "test_access_token",
		AccessSecret:   "test_access_secret",
		// Use 2037 to avoid Y2K38 issues with 32-bit timestamps and potential MySQL date validation issues
		AccessTokenExpiry: time.Date(2037, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	rawToken, err := json.Marshal(testBMToken) // #nosec G117 -- fake ABM token fixture, not a real credential
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	smimeToken := fmt.Sprintf(
		"Content-Type: text/plain;charset=UTF-8\r\n"+
			"Content-Transfer-Encoding: 7bit\r\n"+
			"\r\n%s", rawToken,
	)

	encryptedToken, err := pkcs7.Encrypt([]byte(smimeToken), []*x509.Certificate{cert})
	require.NoError(t, err)

	s.mockDEPResponse(apple_mdm.UnsavedABMTokenOrgName, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
		case "/account":
			_, _ = w.Write([]byte(fmt.Sprintf(`{"admin_id": "abc", "org_name": %q}`, orgName)))
		}
	}))

	// upload the encrypted token
	smimeMessage := fmt.Sprintf(
		"Content-Type: application/pkcs7-mime; name=\"smime.p7m\"; smime-type=enveloped-data\r\n"+
			"Content-Transfer-Encoding: base64\r\n"+
			"Content-Disposition: attachment; filename=\"smime.p7m\"\r\n"+
			"Content-Description: S/MIME Encrypted Message\r\n"+
			"\r\n%s", base64.StdEncoding.EncodeToString(encryptedToken),
	)
	s.uploadABMToken([]byte(smimeMessage), http.StatusOK, "")

	// verify that all the secrets are in the db
	ctx := context.Background()
	assets, err := s.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{
		fleet.MDMAssetABMCert,
		fleet.MDMAssetABMKey,
	}, nil)
	require.NoError(t, err)
	require.Len(t, assets, 2)
	require.Equal(t, abmResp.PublicKey, assets[fleet.MDMAssetABMCert].Value)

	tok, err := s.ds.GetABMTokenByOrgName(ctx, orgName)
	require.NoError(t, err)
	require.Equal(t, orgName, tok.OrganizationName)

	// do a dummy call so the nanodep client updates the org name in
	// nano_dep_names, and leave the mock set with a dummy response
	s.mockDEPResponse(orgName, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
		case "/account":
			_, _ = w.Write([]byte(fmt.Sprintf(`{"admin_id": "abc", "org_name": %q}`, orgName)))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	depClient := apple_mdm.NewDEPClient(s.depStorage, s.ds, s.logger)
	_, err = depClient.AccountDetail(ctx, orgName)
	require.NoError(t, err)
	return tok
}

func (s *integrationMDMTestSuite) appleCoreCertsSetup() {
	t := s.T()
	ctx := context.Background()

	// Successful request
	resp := getMDMAppleCSRResponse{}
	s.SucceedNextCSRRequest()
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/request_csr", getMDMAppleCSRRequest{}, http.StatusOK, &resp)
	require.NotNil(t, resp.CSR)
	block, _ := pem.Decode(resp.CSR)
	require.NotNil(t, block)
	require.Equal(t, "CERTIFICATE REQUEST", block.Type)

	// Check that we created the right assets
	originalAssets, err := s.ds.GetAllMDMConfigAssetsByName(ctx,
		[]fleet.MDMAssetName{fleet.MDMAssetCACert, fleet.MDMAssetCAKey, fleet.MDMAssetAPNSKey}, nil)
	require.NoError(t, err)
	require.Len(t, originalAssets, 3)

	resp = getMDMAppleCSRResponse{}
	s.SucceedNextCSRRequest()
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/request_csr", getMDMAppleCSRRequest{}, http.StatusOK, &resp)
	require.NotNil(t, resp.CSR)
	block, _ = pem.Decode(resp.CSR)
	require.NotNil(t, block)
	require.Equal(t, "CERTIFICATE REQUEST", block.Type)

	// Check that the assets stayed the same in the subsequent call
	assets, err := s.ds.GetAllMDMConfigAssetsByName(ctx,
		[]fleet.MDMAssetName{fleet.MDMAssetCACert, fleet.MDMAssetCAKey, fleet.MDMAssetAPNSKey}, nil)
	require.NoError(t, err)
	require.Equal(t, originalAssets, assets)

	// Successfully upload an APNS cert
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	require.NoError(t, err)

	certTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(12345678),
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		Subject: pkix.Name{
			CommonName: "Fleet",
			ExtraNames: []pkix.AttributeTypeAndValue{
				{
					Type:  asn1.ObjectIdentifier{0, 9, 2342, 19200300, 100, 1, 1},
					Value: "com.apple.mgmt.Example",
				},
			},
		},
	}

	testCert, testKey, err := apple_mdm.NewSCEPCACertKey()
	s.Require().NoError(err)
	certDER, err := x509.CreateCertificate(rand.Reader, certTemplate, testCert, csr.PublicKey, testKey)
	require.NoError(t, err)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	s.uploadDataViaForm("/api/latest/fleet/mdm/apple/apns_certificate", "certificate", "certificate.pem", certPEM, http.StatusAccepted, "", nil)

	assets, err = s.ds.GetAllMDMConfigAssetsByName(ctx,
		[]fleet.MDMAssetName{fleet.MDMAssetCACert, fleet.MDMAssetCAKey, fleet.MDMAssetAPNSKey, fleet.MDMAssetAPNSCert}, nil)
	require.NoError(t, err)
	require.Len(t, assets, 4)
}

func (s *integrationMDMTestSuite) uploadABMToken(encryptedToken []byte, expectedStatus int, wantErr string) {
	t := s.T()

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// add the package field
	fw, err := w.CreateFormFile("token", "token.tok")
	require.NoError(t, err)
	_, err = io.Copy(fw, bytes.NewBuffer(encryptedToken))
	require.NoError(t, err)

	w.Close()

	headers := map[string]string{
		"Content-Type":  w.FormDataContentType(),
		"Accept":        "application/json",
		"Authorization": fmt.Sprintf("Bearer %s", s.token),
	}

	res := s.DoRawWithHeaders("POST", "/api/latest/fleet/abm_tokens", b.Bytes(), expectedStatus, headers)
	if wantErr != "" {
		errMsg := extractServerErrorText(res.Body)
		assert.Contains(t, errMsg, wantErr)
	}
}

func checkInstallFleetdCommandSent(t *testing.T, mdmDevice *mdmtest.TestAppleMDMClient, wantCommand bool) {
	foundInstallFleetdCommand := false
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		if cmd.Command.RequestType == "DeclarativeManagement" {
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
			continue // Do not add to commands as it's not a XML file, so we use a bool to see it once.
		}

		var fullCmd micromdm.CommandPayload
		require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
		if fullCmd.Command.RequestType != "InstallEnterpriseApplication" {
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
			continue
		}

		if manifest := fullCmd.Command.InstallEnterpriseApplication.ManifestURL; manifest != nil {
			foundInstallFleetdCommand = true
			require.Equal(t, "InstallEnterpriseApplication", cmd.Command.RequestType)
			require.Contains(t, *fullCmd.Command.InstallEnterpriseApplication.ManifestURL, fleetdbase.GetPKGManifestURL())
		}
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}
	require.Equal(t, wantCommand, foundInstallFleetdCommand)
}

func (s *integrationMDMTestSuite) getSignedOTAEnrollmentBody(t *testing.T, reqBody []byte) ([]byte, *x509.Certificate, *rsa.PrivateKey) {
	t.Helper()

	// Setup signed req body, but also verify we can sign it.
	cert, key, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(t, err)
	signedData, err := pkcs7.NewSignedData(reqBody)
	require.NoError(t, err)
	require.NoError(t, signedData.AddSigner(cert, key, pkcs7.SignerInfoConfig{}))
	signedReqBody, err := signedData.Finish()
	require.NoError(t, err)

	return signedReqBody, cert, key
}

type mockDigiCertServer struct {
	server    *httptest.Server
	certReq   certificateReq
	certReqMu *sync.Mutex
	notAfter  time.Time
}

type certificateReq struct {
	CSR            string            `json:"csr"`
	Profile        map[string]string `json:"profile"`
	Seat           map[string]string `json:"seat"`
	DeliveryFormat string            `json:"delivery_format"`
	Attributes     map[string]any    `json:"attributes"`
}

// registerResetVPPProxyData resets the VPP proxy data after tests in `t` complete.
func (s *integrationMDMTestSuite) registerResetVPPProxyData(t *testing.T) {
	oldApps := s.appleVPPProxySrvData
	t.Cleanup(func() { s.appleVPPProxySrvData = oldApps })
}

func pluckHostIDsFromHostResponse(hosts []fleet.HostResponse) []uint {
	if len(hosts) == 0 {
		return nil
	}

	ids := make([]uint, len(hosts))
	for i, h := range hosts {
		ids[i] = h.ID
	}
	return ids
}

// mustBYODIdPSession mints the session the IdP callback would have, so a test can
// present a valid BYOD cookie without driving the SAML flow.
func (s *integrationMDMTestSuite) mustBYODIdPSession(t *testing.T, idpAccountUUID string) string {
	sessionID, err := shared_mdm.CreateBYODIdPSession(t.Context(), redis_key_value.New(s.redisPool), clock.C, idpAccountUUID)
	require.NoError(t, err)
	return sessionID
}
