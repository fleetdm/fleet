package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math/big"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	eeservice "github.com/fleetdm/fleet/v4/ee/server/service"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/fleetdbase"
	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/bindata"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/datastore/s3"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/live_query/live_query_mock"
	servermdm "github.com/fleetdm/fleet/v4/server/mdm"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/vpp"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	nanodep_client "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	nanodep_storage "github.com/fleetdm/fleet/v4/server/mdm/nanodep/storage"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/tokenpki"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	nanomdm_pushsvc "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/service"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	filedepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot/file"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/mock"
	"github.com/fleetdm/fleet/v4/server/service/osquery_utils"
	"github.com/fleetdm/fleet/v4/server/service/schedule"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/fleetdm/fleet/v4/server/worker"
	kitlog "github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/groob/plist"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"github.com/smallstep/pkcs7"
	"github.com/smallstep/scep"
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
	fleetCfg                   config.FleetConfig
	fleetDMNextCSRStatus       atomic.Value
	pushProvider               *mock.APNSPushProvider
	depStorage                 nanodep_storage.AllDEPStorage
	profileSchedule            *schedule.Schedule
	integrationsSchedule       *schedule.Schedule
	onProfileJobDone           func() // function called when profileSchedule.Trigger() job completed
	onIntegrationsScheduleDone func() // function called when integrationsSchedule.Trigger() job completed
	mdmStorage                 *mysql.NanoMDMStorage
	worker                     *worker.Worker
	// Flag to skip jobs processing by worker
	skipWorkerJobs            atomic.Bool
	mdmCommander              *apple_mdm.MDMAppleCommander
	logger                    kitlog.Logger
	scepChallenge             string
	appleVPPConfigSrv         *httptest.Server
	appleVPPConfigSrvConfig   *appleVPPConfigSrvConf
	appleITunesSrv            *httptest.Server
	appleGDMFSrv              *httptest.Server
	mockedDownloadFleetdmMeta fleetdbase.Metadata
}

// appleVPPConfigSrvConf is used to configure the mock server that mocks Apple's VPP endpoints.
type appleVPPConfigSrvConf struct {
	Assets        []vpp.Asset
	SerialNumbers []string
	Location      string
}

func (s *integrationMDMTestSuite) SetupSuite() {
	s.withDS.SetupSuite("integrationMDMTestSuite")

	appConf, err := s.ds.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.MDM.EnabledAndConfigured = true
	appConf.MDM.WindowsEnabledAndConfigured = true
	appConf.MDM.AppleBMEnabledAndConfigured = true
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)

	fleetCfg := config.TestConfig()
	testCert, testKey, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(s.T(), err)
	testCertPEM := tokenpki.PEMCertificate(testCert.Raw)
	testKeyPEM := tokenpki.PEMRSAPrivateKey(testKey)
	config.SetTestMDMConfig(s.T(), &fleetCfg, testCertPEM, testKeyPEM, "../../server/service/testdata")
	fleetCfg.Osquery.EnrollCooldown = 0

	mdmStorage, err := s.ds.NewMDMAppleMDMStorage()
	require.NoError(s.T(), err)
	depStorage, err := s.ds.NewMDMAppleDEPStorage()
	require.NoError(s.T(), err)
	scepStorage, err := s.ds.NewSCEPDepot()
	require.NoError(s.T(), err)

	pushLog := kitlog.NewJSONLogger(os.Stdout)
	if os.Getenv("FLEET_INTEGRATION_TESTS_DISABLE_LOG") != "" {
		pushLog = kitlog.NewNopLogger()
	}
	pushFactory, pushProvider := newMockAPNSPushProviderFactory()
	mdmPushService := nanomdm_pushsvc.New(
		mdmStorage,
		mdmStorage,
		pushFactory,
		NewNanoMDMLogger(pushLog),
	)
	mdmCommander := apple_mdm.NewMDMAppleCommander(mdmStorage, mdmPushService)
	redisPool := redistest.SetupRedis(s.T(), "zz", false, false, false)
	s.withServer.lq = live_query_mock.New(s.T())

	wlog := kitlog.NewJSONLogger(os.Stdout)
	if os.Getenv("FLEET_INTEGRATION_TESTS_DISABLE_LOG") != "" {
		wlog = kitlog.NewNopLogger()
	}
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
	workr := worker.NewWorker(s.ds, wlog)
	workr.TestIgnoreUnknownJobs = true
	workr.Register(macosJob, appleMDMJob)
	s.worker = workr

	// clear the jobs queue of any pending jobs generated via DB migrations
	mysql.ExecAdhocSQL(s.T(), s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(context.Background(), "DELETE FROM jobs")
		return err
	})

	var integrationsSchedule *schedule.Schedule
	var profileSchedule *schedule.Schedule
	cronLog := kitlog.NewJSONLogger(os.Stdout)
	if os.Getenv("FLEET_INTEGRATION_TESTS_DISABLE_LOG") != "" {
		cronLog = kitlog.NewNopLogger()
	}
	serverLogger := kitlog.NewJSONLogger(os.Stdout)
	if os.Getenv("FLEET_INTEGRATION_TESTS_DISABLE_LOG") != "" {
		serverLogger = kitlog.NewNopLogger()
	}

	var softwareInstallerStore fleet.SoftwareInstallerStore
	var bootstrapPackageStore fleet.MDMBootstrapPackageStore
	_, minioEnabled := os.LookupEnv("MINIO_STORAGE_TEST")
	if wantStore := os.Getenv("FLEET_INTEGRATION_TESTS_SOFTWARE_INSTALLER_STORE"); minioEnabled &&
		(wantStore == "s3" || (wantStore == "" && time.Now().UnixNano()%2 == 0)) {

		s.T().Log(">>> using S3/minio software installer store")
		softwareInstallerStore = s3.SetupTestSoftwareInstallerStore(s.T(), "integration-tests", "")
		bootstrapPackageStore = s3.SetupTestBootstrapPackageStore(s.T(), "integration-tests", "")
	}

	serverConfig := TestServerOpts{
		License: &fleet.LicenseInfo{
			Tier: fleet.TierPremium,
		},
		Logger:                serverLogger,
		FleetConfig:           &fleetCfg,
		MDMStorage:            mdmStorage,
		DEPStorage:            depStorage,
		SCEPStorage:           scepStorage,
		MDMPusher:             mdmPushService,
		Pool:                  redisPool,
		Lq:                    s.lq,
		SoftwareInstallStore:  softwareInstallerStore,
		BootstrapPackageStore: bootstrapPackageStore,
		StartCronSchedules: []TestNewScheduleFunc{
			func(ctx context.Context, ds fleet.Datastore) fleet.NewCronScheduleFunc {
				return func() (fleet.CronSchedule, error) {
					const name = string(fleet.CronMDMAppleProfileManager)
					logger := cronLog
					profileSchedule = schedule.New(
						ctx, name, s.T().Name(), 1*time.Hour, ds, ds,
						schedule.WithLogger(logger),
						schedule.WithJob("manage_apple_profiles", func(ctx context.Context) error {
							if s.onProfileJobDone != nil {
								defer s.onProfileJobDone()
							}
							err = ReconcileAppleProfiles(ctx, ds, mdmCommander, logger)
							require.NoError(s.T(), err)
							return err
						}),
						schedule.WithJob("manage_apple_declarations", func(ctx context.Context) error {
							if s.onProfileJobDone != nil {
								defer s.onProfileJobDone()
							}
							err = ReconcileAppleDeclarations(ctx, ds, mdmCommander, logger)
							require.NoError(s.T(), err)
							return err
						}),
						schedule.WithJob("manage_windows_profiles", func(ctx context.Context) error {
							if s.onProfileJobDone != nil {
								defer s.onProfileJobDone()
							}
							err := ReconcileWindowsProfiles(ctx, ds, logger)
							require.NoError(s.T(), err)
							return err
						}),
					)
					return profileSchedule, nil
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
					const name = string(fleet.CronAppleMDMIPhoneIPadRefetcher)
					logger := cronLog
					refetcherSchedule := schedule.New(
						ctx, name, s.T().Name(), 1*time.Hour, ds, ds,
						schedule.WithLogger(logger),
						schedule.WithJob("cron_iphone_ipad_refetcher", func(ctx context.Context) error {
							return apple_mdm.IOSiPadOSRefetch(ctx, ds, mdmCommander, logger)
						}),
					)
					return refetcherSchedule, nil
				}
			},
		},
		APNSTopic:       "com.apple.mgmt.External.10ac3ce5-4668-4e58-b69a-b2b5ce667589",
		EnableSCEPProxy: true,
		WithDEPWebview:  true,
	}

	// ensure all our tests support challenges with invalid XML characters
	s.scepChallenge = "scepcha/><llenge"
	err = s.ds.InsertMDMConfigAssets(context.Background(), []fleet.MDMConfigAsset{
		{Name: fleet.MDMAssetSCEPChallenge, Value: []byte(s.scepChallenge)},
	}, nil)
	require.NoError(s.T(), err)
	users, server := RunServerForTestsWithDS(s.T(), s.ds, &serverConfig)
	s.server = server
	s.users = users
	s.token = s.getTestAdminToken()
	s.cachedAdminToken = s.token
	s.fleetCfg = fleetCfg
	s.pushProvider = pushProvider
	s.depStorage = depStorage
	s.integrationsSchedule = integrationsSchedule
	s.profileSchedule = profileSchedule
	s.mdmStorage = mdmStorage
	s.mdmCommander = mdmCommander
	s.logger = serverLogger

	fleetdmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := s.fleetDMNextCSRStatus.Swap(http.StatusOK)
		w.WriteHeader(status.(int))
		resp := []byte(fmt.Sprintf("status: %d", status))
		if status == http.StatusOK && strings.Contains(r.URL.RawQuery, "deliveryMethod=json") {
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
			Assets: []vpp.Asset{
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
			},
			SerialNumbers: []string{"123", "456"},
			Location:      "Fleet Location One",
		}
	}

	s.appleVPPConfigSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			assets := s.appleVPPConfigSrvConfig.Assets
			if adamID := r.URL.Query().Get("adamId"); adamID != "" {
				for _, a := range assets {
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
		resp := []byte(fmt.Sprintf(`{"locationName": "%s"}`, s.appleVPPConfigSrvConfig.Location))
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

	s.appleITunesSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// a map of apps we can respond with
		db := map[string]string{
			// macos app
			"1": `{"bundleId": "a-1", "artworkUrl512": "https://example.com/images/1", "version": "1.0.0", "trackName": "App 1", "TrackID": 1}`,
			// macos, ios, ipados app
			"2": `{"bundleId": "b-2", "artworkUrl512": "https://example.com/images/2", "version": "2.0.0", "trackName": "App 2", "TrackID": 2,
				"supportedDevices": ["MacDesktop-MacDesktop", "iPhone5s-iPhone5s", "iPadAir-iPadAir"] }`,
			// ipados app
			"3": `{"bundleId": "c-3", "artworkUrl512": "https://example.com/images/3", "version": "3.0.0", "trackName": "App 3", "TrackID": 3,
				"supportedDevices": ["iPadAir-iPadAir"] }`,
		}

		adamIDString := r.URL.Query().Get("id")
		adamIDs := strings.Split(adamIDString, ",")

		var objs []string
		for _, a := range adamIDs {
			objs = append(objs, db[a])
		}

		_, _ = w.Write([]byte(fmt.Sprintf(`{"results": [%s]}`, strings.Join(objs, ","))))
	}))

	s.appleGDMFSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// load the test data from the file
		b, err := os.ReadFile("../mdm/apple/gdmf/testdata/gdmf.json")
		require.NoError(s.T(), err)
		_, err = w.Write(b)
		require.NoError(s.T(), err)
	}))

	s.T().Setenv("FLEET_DEV_GDMF_URL", s.appleGDMFSrv.URL)
	s.T().Setenv("TEST_FLEETDM_API_URL", fleetdmSrv.URL)
	s.T().Setenv("FLEET_DEV_ITUNES_URL", s.appleITunesSrv.URL)

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
	s.T().Cleanup(s.appleITunesSrv.Close)
	s.T().Cleanup(s.appleGDMFSrv.Close)
}

func (s *integrationMDMTestSuite) TearDownSuite() {
	appConf, err := s.ds.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.MDM.EnabledAndConfigured = false
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)
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
	// ensure global disk encryption is disabled on exit
	appCfg.MDM.EnableDiskEncryption = optjson.SetBool(false)
	// ensure enable release manually is false
	appCfg.MDM.MacOSSetup.EnableReleaseDeviceManually = optjson.SetBool(false)
	// ensure global Windows OS updates are always disabled for the next test
	appCfg.MDM.WindowsUpdates = fleet.WindowsUpdates{}
	// ensure the server URL is constant
	appCfg.ServerSettings.ServerURL = s.server.URL
	err := s.ds.SaveAppConfig(ctx, &appCfg.AppConfig)
	require.NoError(t, err)

	s.withServer.commonTearDownTest(t)

	// use a sql statement to delete all profiles, since the datastore prevents
	// deleting the fleet-specific ones.
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM mdm_apple_configuration_profiles")
		return err
	})
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM mdm_windows_configuration_profiles")
		return err
	})
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM mdm_apple_bootstrap_packages")
		return err
	})

	// clear any pending worker job
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM jobs")
		return err
	})

	// clear any host dep assignments
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM host_dep_assignments")
		return err
	})

	// clear any mdm windows enrollments
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM mdm_windows_enrollments")
		return err
	})

	// clear any lingering declarations
	mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM mdm_apple_declarations")
		return err
	})
	mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM host_mdm_apple_declarations")
		return err
	})
	mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM mobile_device_management_solutions;")
		return err
	})
	mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM host_mdm;")
		return err
	})
	mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM abm_tokens;")
		return err
	})
	mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM vpp_tokens;")
		return err
	})
	mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM setup_experience_status_results;")
		return err
	})
	mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM setup_experience_scripts;")
		return err
	})
}

func (s *integrationMDMTestSuite) mockDEPResponse(orgName string, handler http.Handler) {
	t := s.T()
	srv := httptest.NewServer(handler)
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
	var wg sync.WaitGroup
	wg.Add(3)
	s.onProfileJobDone = wg.Done
	_, err := s.profileSchedule.Trigger()
	require.NoError(t, err)
	wg.Wait()
}

func (s *integrationMDMTestSuite) TestGetBootstrapToken() {
	// see https://developer.apple.com/documentation/devicemanagement/get_bootstrap_token
	t := s.T()
	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "MacBookPro16,1")
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	checkStoredCertAuthAssociation := func(id string, expectedCount uint) {
		// confirm expected cert auth association
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			var ct uint
			// query duplicates the logic in nanomdm/storage/mysql/certauth.go
			if err := sqlx.GetContext(context.Background(), q, &ct, "SELECT COUNT(*) FROM nano_cert_auth_associations WHERE id = ?", mdmDevice.UUID); err != nil {
				return err
			}
			require.Equal(t, expectedCount, ct)
			return nil
		})
	}
	checkStoredCertAuthAssociation(mdmDevice.UUID, 1)

	checkStoredBootstrapToken := func(id string, expectedToken *string, expectedErr error) {
		// confirm expected bootstrap token
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			var tok *string
			err := sqlx.GetContext(context.Background(), q, &tok, "SELECT bootstrap_token_b64 FROM nano_devices WHERE id = ?", mdmDevice.UUID)
			if err != nil || expectedErr != nil {
				require.ErrorIs(t, err, expectedErr)
			} else {
				require.NoError(t, err)
			}

			if expectedToken != nil {
				require.NotEmpty(t, tok)
				decoded, err := base64.StdEncoding.DecodeString(*tok)
				require.NoError(t, err)
				require.Equal(t, *expectedToken, string(decoded))
			} else {
				require.Empty(t, tok)
			}
			return nil
		})
	}

	t.Run("bootstrap token not set", func(t *testing.T) {
		// device record exists, but bootstrap token not set
		checkStoredBootstrapToken(mdmDevice.UUID, nil, nil)

		// if token not set, server returns empty body and no error (see https://github.com/micromdm/nanomdm/pull/63)
		res, err := mdmDevice.GetBootstrapToken()
		require.NoError(t, err)
		require.Nil(t, res)
	})

	t.Run("bootstrap token set", func(t *testing.T) {
		// device record exists, set bootstrap token
		token := base64.StdEncoding.EncodeToString([]byte("testtoken"))
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(context.Background(), "UPDATE nano_devices SET bootstrap_token_b64 = ? WHERE id = ?", base64.StdEncoding.EncodeToString([]byte(token)), mdmDevice.UUID)
			require.NoError(t, err)
			return nil
		})
		checkStoredBootstrapToken(mdmDevice.UUID, &token, nil)

		// if token set, server returns token
		res, err := mdmDevice.GetBootstrapToken()
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, token, string(res))
	})

	t.Run("no device record", func(t *testing.T) {
		// delete the entire device record
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(context.Background(), "DELETE FROM nano_devices WHERE id = ?", mdmDevice.UUID)
			require.NoError(t, err)
			return nil
		})
		checkStoredBootstrapToken(mdmDevice.UUID, nil, sql.ErrNoRows)

		// if not found, server returns empty body and no error (see https://github.com/fleetdm/nanomdm/pull/8)
		res, err := mdmDevice.GetBootstrapToken()
		require.NoError(t, err)
		require.Nil(t, res)
	})

	t.Run("no cert auth association", func(t *testing.T) {
		// on mdm checkout, nano soft deletes by calling storage.Disable, which leaves the cert auth
		// association in place, so what if we hard delete instead?
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(context.Background(), "DELETE FROM nano_cert_auth_associations WHERE id = ?", mdmDevice.UUID)
			require.NoError(t, err)
			return nil
		})
		checkStoredCertAuthAssociation(mdmDevice.UUID, 0)

		// TODO: server returns 500 on account of cert auth but what is the expected behavior?
		res, err := mdmDevice.GetBootstrapToken()
		require.ErrorContains(t, err, "500") // getbootstraptoken service: cert auth: existing enrollment: enrollment not associated with cert
		require.Nil(t, res)
	})
}

func (s *integrationMDMTestSuite) TestAppleGetAppleMDM() {
	t := s.T()

	var mdmResp getAppleMDMResponse
	s.DoJSON("GET", "/api/latest/fleet/apns", nil, http.StatusOK, &mdmResp)
	// returned values are dummy, this is a test certificate
	require.Equal(t, "Fleet", mdmResp.Issuer)
	require.NotZero(t, mdmResp.SerialNumber)
	require.Equal(t, "Fleet", mdmResp.CommonName)
	require.NotZero(t, mdmResp.RenewDate)

	var countTokensResp countABMTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/abm_tokens/count", nil, http.StatusOK, &countTokensResp)
	assert.EqualValues(t, 0, countTokensResp.Count)

	// set up multiple ABM tokens with different org names
	defaultOrgName := "fleet_test"
	s.enableABM(defaultOrgName)
	tmOrgName := t.Name()
	s.enableABM(tmOrgName)

	var tokensResp listABMTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/abm_tokens", nil, http.StatusOK, &tokensResp)

	// for t.Name()
	tok := s.getABMTokenByName(defaultOrgName, tokensResp.Tokens)
	require.NotNil(t, tok)
	require.False(t, tok.TermsExpired)
	require.Equal(t, "abc", tok.AppleID)
	require.Equal(t, defaultOrgName, tok.OrganizationName)
	require.Equal(t, s.server.URL+"/mdm/apple/mdm", tok.MDMServerURL)
	require.Equal(t, fleet.TeamNameNoTeam, tok.MacOSTeam.Name)
	require.Equal(t, fleet.TeamNameNoTeam, tok.IOSTeam.Name)
	require.Equal(t, fleet.TeamNameNoTeam, tok.IPadOSTeam.Name)

	// for tmOrgName
	tok = s.getABMTokenByName(tmOrgName, tokensResp.Tokens)
	require.NotNil(t, tok)
	require.False(t, tok.TermsExpired)
	require.Equal(t, "abc", tok.AppleID)
	require.Equal(t, tmOrgName, tok.OrganizationName)
	require.Equal(t, s.server.URL+"/mdm/apple/mdm", tok.MDMServerURL)
	require.Equal(t, fleet.TeamNameNoTeam, tok.MacOSTeam.Name)
	require.Equal(t, fleet.TeamNameNoTeam, tok.IOSTeam.Name)
	require.Equal(t, fleet.TeamNameNoTeam, tok.IPadOSTeam.Name)

	s.DoJSON("GET", "/api/latest/fleet/abm_tokens/count", nil, http.StatusOK, &countTokensResp)
	assert.EqualValues(t, 2, countTokensResp.Count)

	// create a new team
	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name(),
		Description: "desc",
	})
	require.NoError(t, err)
	// set the default bm assignment for that token to that team
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": {
			"apple_business_manager": [{
			  "organization_name": %q,
			  "macos_team": %q,
			  "ios_team": %q,
			  "ipados_team": %q
			}]
		}
	}`, tmOrgName, tm.Name, tm.Name, tm.Name)), http.StatusOK, &acResp)
	t.Cleanup(func() {
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"apple_business_manager": []
			}
		}`), http.StatusOK, &acResp)
	})

	// try again, this time we get team assignments in the response
	tokensResp = listABMTokensResponse{}
	s.DoJSON("GET", "/api/latest/fleet/abm_tokens", nil, http.StatusOK, &tokensResp)

	tok = s.getABMTokenByName(tmOrgName, tokensResp.Tokens)
	require.NotNil(t, tok)
	require.False(t, tok.TermsExpired)
	require.Equal(t, "abc", tok.AppleID)
	require.Equal(t, tmOrgName, tok.OrganizationName)
	require.Equal(t, s.server.URL+"/mdm/apple/mdm", tok.MDMServerURL)
	require.Equal(t, tm.Name, tok.MacOSTeam.Name)
	require.Equal(t, tm.Name, tok.IOSTeam.Name)
	require.Equal(t, tm.Name, tok.IPadOSTeam.Name)

	// Reset the teams via app config
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"apple_business_manager": []
			}
		}`), http.StatusOK, &acResp)

	tokensResp = listABMTokensResponse{}
	s.DoJSON("GET", "/api/latest/fleet/abm_tokens", nil, http.StatusOK, &tokensResp)
	tok = s.getABMTokenByName(tmOrgName, tokensResp.Tokens)
	require.NotNil(t, tok)
	require.False(t, tok.TermsExpired)
	require.Equal(t, "abc", tok.AppleID)
	require.Equal(t, tmOrgName, tok.OrganizationName)
	require.Equal(t, s.server.URL+"/mdm/apple/mdm", tok.MDMServerURL)
	require.Equal(t, fleet.TeamNameNoTeam, tok.MacOSTeam.Name)
	require.Equal(t, fleet.TeamNameNoTeam, tok.IOSTeam.Name)
	require.Equal(t, fleet.TeamNameNoTeam, tok.IPadOSTeam.Name)
}

func (s *integrationMDMTestSuite) getABMTokenByName(orgName string, tokens []*fleet.ABMToken) *fleet.ABMToken {
	for _, tok := range tokens {
		if tok.OrganizationName == orgName {
			return tok
		}
	}

	return nil
}

func (s *integrationMDMTestSuite) TestABMExpiredToken() {
	t := s.T()

	s.enableABM(t.Name())

	var returnType string
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch returnType {
		case "not_signed":
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"code": "T_C_NOT_SIGNED"}`))
		case "unauthorized":
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{}`))
		case "success":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"auth_session_token": "abcd"}`))
		default:
			require.Fail(t, "unexpected return type: %s", returnType)
		}
	}))

	config := s.getConfig()
	require.False(t, config.MDM.AppleBMTermsExpired)

	ctx := context.Background()
	fleetSyncer := apple_mdm.NewDEPService(s.ds, s.depStorage, s.logger)

	// not signed error flips the AppleBMTermsExpired flag
	returnType = "not_signed"
	err := fleetSyncer.RunAssigner(ctx)
	require.ErrorContains(t, err, "T_C_NOT_SIGNED")
	var tokensResp listABMTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/abm_tokens", nil, http.StatusOK, &tokensResp)
	tok := s.getABMTokenByName(t.Name(), tokensResp.Tokens)
	require.NotNil(t, tok)
	require.True(t, tok.TermsExpired)
	config = s.getConfig()
	require.True(t, config.MDM.AppleBMTermsExpired)

	// a successful call clears it
	returnType = "success"
	err = fleetSyncer.RunAssigner(ctx)
	require.NoError(t, err)
	tokensResp = listABMTokensResponse{}
	s.DoJSON("GET", "/api/latest/fleet/abm_tokens", nil, http.StatusOK, &tokensResp)
	tok = s.getABMTokenByName(t.Name(), tokensResp.Tokens)
	require.NotNil(t, tok)
	require.False(t, tok.TermsExpired)

	config = s.getConfig()
	require.False(t, config.MDM.AppleBMTermsExpired)

	// an unauthorized call does not flip the terms expired flag
	returnType = "unauthorized"
	err = fleetSyncer.RunAssigner(ctx)
	require.ErrorContains(t, err, "DEP auth error")
	tokensResp = listABMTokensResponse{}
	s.DoJSON("GET", "/api/latest/fleet/abm_tokens", nil, http.StatusOK, &tokensResp)
	tok = s.getABMTokenByName(t.Name(), tokensResp.Tokens)
	require.NotNil(t, tok)
	require.False(t, tok.TermsExpired)

	config = s.getConfig()
	require.False(t, config.MDM.AppleBMTermsExpired)
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
		require.Equal(t, pushes[0].PushMagic, "pushmagic"+mdmDevice.SerialNumber)
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
	desktopToken := uuid.New().String()
	mdmDevice := mdmtest.NewTestMDMClientAppleDesktopManual(fleetServerURL, desktopToken)
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

		UUID:           mdmDevice.UUID,
		HardwareSerial: mdmDevice.SerialNumber,
	})
	require.NoError(t, err)

	err = ds.SetOrUpdateDeviceAuthToken(context.Background(), fleetHost.ID, desktopToken)
	require.NoError(t, err)

	err = mdmDevice.Enroll()
	require.NoError(t, err)

	return fleetHost, mdmDevice
}

func (s *integrationMDMTestSuite) createAppleMobileHostThenEnrollMDM(platform string) (*fleet.Host, *mdmtest.TestAppleMDMClient) {
	ctx := context.Background()
	t := s.T()

	// create a host with minimal information and the serial, no uuid/osquery id
	// (as when created via DEP sync).
	dbZeroTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	serialNumber := mdmtest.RandSerialNumber()
	fleetHost, err := s.ds.NewHost(ctx, &fleet.Host{
		HardwareSerial:   serialNumber,
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
	model := "iPhone14,6"
	if platform == "ipados" {
		model = "iPad13,18"
	}
	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, model)
	mdmDevice.SerialNumber = serialNumber
	err = mdmDevice.Enroll()
	require.NoError(t, err)

	return fleetHost, mdmDevice
}

func createWindowsHostThenEnrollMDM(ds fleet.Datastore, fleetServerURL string, t *testing.T) (*fleet.Host, *mdmtest.TestWindowsMDMClient) {
	host := createOrbitEnrolledHost(t, "windows", "h1", ds)
	mdmDevice := mdmtest.NewTestMDMClientWindowsProgramatic(fleetServerURL, *host.OrbitNodeKey)
	err := mdmDevice.Enroll()
	require.NoError(t, err)
	err = ds.UpdateMDMWindowsEnrollmentsHostUUID(context.Background(), host.UUID, mdmDevice.DeviceID)
	require.NoError(t, err)
	err = ds.SetOrUpdateMDMData(context.Background(), host.ID, false, true, fleetServerURL, false, fleet.WellKnownMDMFleet, "")
	require.NoError(t, err)
	return host, mdmDevice
}

func loadEnrollmentProfileDEPToken(t *testing.T, ds *mysql.Datastore) string {
	var token string
	mysql.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &token,
			`SELECT token FROM mdm_apple_enrollment_profiles`)
	})
	return token
}

func (s *integrationMDMTestSuite) TestDeviceMDMManualEnroll() {
	t := s.T()

	token := "token_test_manual_enroll"
	createHostAndDeviceToken(t, s.ds, token)

	// invalid token fails
	s.DoRaw("GET", "/api/latest/fleet/device/invalid_token/mdm/apple/manual_enrollment_profile", nil, http.StatusUnauthorized)

	// valid token downloads the profile
	s.downloadAndVerifyOTAEnrollmentProfile("/api/latest/fleet/device/" + token + "/mdm/apple/manual_enrollment_profile")
}

func (s *integrationMDMTestSuite) TestAppleMDMDeviceEnrollment() {
	t := s.T()

	// Enroll two devices into MDM
	mdmEnrollInfo := mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}
	mdmDeviceA := mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")
	err := mdmDeviceA.Enroll()
	require.NoError(t, err)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeMDMEnrolled{}.ActivityName(),
		fmt.Sprintf(`{"host_serial": "%s", "host_display_name": "%s (%s)", "installed_from_dep": false, "mdm_platform": "apple"}`, mdmDeviceA.SerialNumber, mdmDeviceA.Model, mdmDeviceA.SerialNumber), 0)

	mdmDeviceB := mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")
	err = mdmDeviceB.Enroll()
	require.NoError(t, err)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeMDMEnrolled{}.ActivityName(),
		fmt.Sprintf(`{"host_serial": "%s", "host_display_name": "%s (%s)", "installed_from_dep": false, "mdm_platform": "apple"}`, mdmDeviceB.SerialNumber, mdmDeviceB.Model, mdmDeviceB.SerialNumber), 0)

	// Find the ID of Fleet's MDM solution
	var mdmID uint
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &mdmID,
			`SELECT id FROM mobile_device_management_solutions WHERE name = ?`, fleet.WellKnownMDMFleet)
	})

	// Check that both devices are returned by the /hosts endpoint
	listHostsRes := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes, "mdm_id", fmt.Sprint(mdmID))
	require.Len(t, listHostsRes.Hosts, 2)
	require.EqualValues(
		t,
		[]string{mdmDeviceA.UUID, mdmDeviceB.UUID},
		[]string{listHostsRes.Hosts[0].UUID, listHostsRes.Hosts[1].UUID},
	)

	var targetHostID uint
	var lastEnroll time.Time
	for _, host := range listHostsRes.Hosts {
		if host.UUID == mdmDeviceA.UUID {
			targetHostID = host.ID
			lastEnroll = host.LastEnrolledAt
			break
		}
	}

	// set an enroll secret
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: t.Name()}},
		},
	}, http.StatusOK, &applyResp)

	// simulate a matching host enrolling via osquery
	j, err := json.Marshal(&enrollAgentRequest{
		EnrollSecret:   t.Name(),
		HostIdentifier: mdmDeviceA.UUID,
	})
	require.NoError(t, err)
	var enrollResp enrollAgentResponse
	hres := s.DoRawNoAuth("POST", "/api/osquery/enroll", j, http.StatusOK)
	defer hres.Body.Close()
	require.NoError(t, json.NewDecoder(hres.Body).Decode(&enrollResp))
	require.NotEmpty(t, enrollResp.NodeKey)

	// query all hosts
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	// we still have only two hosts
	require.Len(t, listHostsRes.Hosts, 2)

	// LastEnrolledAt should have been updated
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", targetHostID), nil, http.StatusOK, &getHostResp)
	require.Greater(t, getHostResp.Host.LastEnrolledAt, lastEnroll)

	// Unenroll a device
	err = mdmDeviceA.Checkout()
	require.NoError(t, err)

	// An activity is created
	activities := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activities)

	found := false
	for _, activity := range activities.Activities {
		if activity.Type == "mdm_unenrolled" {
			found = true
			require.Nil(t, activity.ActorID)
			require.Nil(t, activity.ActorFullName)
			require.JSONEq(t, fmt.Sprintf(`{"host_serial": "%s", "host_display_name": "%s (%s)", "installed_from_dep": false}`, mdmDeviceA.SerialNumber, mdmDeviceA.Model, mdmDeviceA.SerialNumber), string(*activity.Details))
		}
	}
	require.True(t, found)
}

func (s *integrationMDMTestSuite) TestDeviceMultipleAuthMessages() {
	t := s.T()

	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "MacBookPro16,1")
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	listHostsRes := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(s.T(), listHostsRes.Hosts, 1)

	// send the auth message again, we still have only one host
	err = mdmDevice.Authenticate()
	require.NoError(t, err)
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(s.T(), listHostsRes.Hosts, 1)
}

func (s *integrationMDMTestSuite) TestAppleMDMCSRRequest() {
	t := s.T()

	var errResp validationErrResp
	// missing arguments
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/request_csr", requestMDMAppleCSRRequest{}, http.StatusUnprocessableEntity, &errResp)
	require.Len(t, errResp.Errors, 1)
	require.Equal(t, errResp.Errors[0].Name, "email_address")

	// invalid email address
	errResp = validationErrResp{}
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/request_csr", requestMDMAppleCSRRequest{EmailAddress: "abc", Organization: "def"}, http.StatusUnprocessableEntity, &errResp)
	require.Len(t, errResp.Errors, 1)
	require.Equal(t, errResp.Errors[0].Name, "email_address")

	// missing organization
	errResp = validationErrResp{}
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/request_csr", requestMDMAppleCSRRequest{EmailAddress: "a@b.c", Organization: ""}, http.StatusUnprocessableEntity, &errResp)
	require.Len(t, errResp.Errors, 1)
	require.Equal(t, errResp.Errors[0].Name, "organization")

	// fleetdm CSR request failed
	s.FailNextCSRRequestWith(http.StatusBadRequest)
	errResp = validationErrResp{}
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/request_csr", requestMDMAppleCSRRequest{EmailAddress: "a@b.c", Organization: "test"}, http.StatusUnprocessableEntity, &errResp)
	require.Len(t, errResp.Errors, 1)
	require.Contains(t, errResp.Errors[0].Reason, "this email address is not valid")

	s.FailNextCSRRequestWith(http.StatusInternalServerError)
	errResp = validationErrResp{}
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/request_csr", requestMDMAppleCSRRequest{EmailAddress: "a@b.c", Organization: "test"}, http.StatusBadGateway, &errResp)
	require.Len(t, errResp.Errors, 1)
	require.Contains(t, errResp.Errors[0].Reason, "FleetDM CSR request failed")

	var reqCSRResp requestMDMAppleCSRResponse
	// fleetdm CSR request succeeds
	s.SucceedNextCSRRequest()
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/request_csr", requestMDMAppleCSRRequest{EmailAddress: "a@b.c", Organization: "test"}, http.StatusOK, &reqCSRResp)
	require.Contains(t, string(reqCSRResp.APNsKey), "-----BEGIN RSA PRIVATE KEY-----\n")
	require.Contains(t, string(reqCSRResp.SCEPCert), "-----BEGIN CERTIFICATE-----\n")
	require.Contains(t, string(reqCSRResp.SCEPKey), "-----BEGIN RSA PRIVATE KEY-----\n")
}

func (s *integrationMDMTestSuite) TestGetMDMCSR() {
	t := s.T()
	ctx := context.Background()

	// Validate errors if no private key is set
	testSetEmptyPrivateKey = true
	t.Cleanup(func() { testSetEmptyPrivateKey = false })
	s.uploadDataViaForm("/api/latest/fleet/mdm/apple/apns_certificate", "certificate", "certificate.pem", []byte("-----BEGIN CERTIFICATE-----\nZm9vCg==\n-----END CERTIFICATE-----"), http.StatusInternalServerError, "Couldn't upload APNs certificate. Missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key", nil)

	r := s.Do("GET", "/api/latest/fleet/mdm/apple/request_csr", getMDMAppleCSRRequest{}, http.StatusInternalServerError)
	require.Contains(t, extractServerErrorText(r.Body), "Couldn't download signed CSR. Missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key")
	testSetEmptyPrivateKey = false

	// ensure we leave everything in a clean state for other tests
	t.Cleanup(s.appleCoreCertsSetup)

	// Delete APNS cert, should soft delete all certs and keys created in this test
	s.Do("DELETE", "/api/latest/fleet/mdm/apple/apns_certificate", nil, http.StatusOK)

	assets, err := s.ds.GetAllMDMConfigAssetsByName(ctx,
		[]fleet.MDMAssetName{fleet.MDMAssetCACert, fleet.MDMAssetCAKey, fleet.MDMAssetAPNSKey, fleet.MDMAssetAPNSCert}, nil)
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, assets)

	// trying to upload a certificate without generating a private key first is not allowed
	s.uploadDataViaForm("/api/latest/fleet/mdm/apple/apns_certificate", "certificate", "certificate.pem", []byte("-----BEGIN CERTIFICATE-----\nZm9vCg==\n-----END CERTIFICATE-----"), http.StatusBadRequest, "Please generate a private key first.", nil)

	// Check that we return bad gateway if the website API errors
	s.FailNextCSRRequestWith(http.StatusInternalServerError)
	errResp := validationErrResp{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/request_csr", getMDMAppleCSRRequest{}, http.StatusBadGateway, &errResp)
	require.Len(t, errResp.Errors, 1)
	require.Contains(t, errResp.Errors[0].Reason, "FleetDM CSR request failed")

	// Check that we return bad request if the website API does (it will do this in case of an
	// invalid email address
	s.FailNextCSRRequestWith(http.StatusUnprocessableEntity)
	errResp = validationErrResp{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/request_csr", getMDMAppleCSRRequest{}, http.StatusUnprocessableEntity, &errResp)
	require.Len(t, errResp.Errors, 1)
	require.Contains(t, errResp.Errors[0].Reason, "this email address is not valid")

	// Invalid APNS cert upload attempt
	s.uploadDataViaForm("/api/latest/fleet/mdm/apple/apns_certificate", "certificate", "certificate.pem", []byte("invalid-cert"), http.StatusUnprocessableEntity, "Invalid certificate. Please provide a valid certificate from Apple Push Certificate Portal.", nil)

	// simulate a renew flow
	s.appleCoreCertsSetup()
}

func (s *integrationMDMTestSuite) uploadDataViaForm(endpoint, fieldName, fileName string, data []byte, expectedStatus int, wantErr string, response any) {
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

	res := s.DoRawWithHeaders("POST", endpoint, b.Bytes(), expectedStatus, headers)
	if wantErr != "" {
		errMsg := extractServerErrorText(res.Body)
		assert.Contains(t, errMsg, wantErr)
	}

	if response != nil {
		err := json.NewDecoder(res.Body).Decode(&response)
		require.NoError(t, err)
	}
}

func (s *integrationMDMTestSuite) TestMDMAppleUnenroll() {
	t := s.T()

	// Enroll a device into MDM.
	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "MacBookPro16,1")
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	// set an enroll secret
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: t.Name()}},
		},
	}, http.StatusOK, &applyResp)

	// simulate a matching host enrolling via osquery
	j, err := json.Marshal(&enrollAgentRequest{
		EnrollSecret:   t.Name(),
		HostIdentifier: mdmDevice.UUID,
	})
	require.NoError(t, err)
	var enrollResp enrollAgentResponse
	hres := s.DoRawNoAuth("POST", "/api/osquery/enroll", j, http.StatusOK)
	defer hres.Body.Close()
	require.NoError(t, json.NewDecoder(hres.Body).Decode(&enrollResp))
	require.NotEmpty(t, enrollResp.NodeKey)

	listHostsRes := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 1)
	h := listHostsRes.Hosts[0]

	// assign profiles to the host
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{
		mobileconfigForTest("N1", "I1"),
		mobileconfigForTest("N2", "I2"),
		mobileconfigForTest("N3", "I3"),
	}}, http.StatusNoContent)

	// trigger a sync and verify that there are profiles assigned to the host
	s.awaitTriggerProfileSchedule(t)

	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", h.ID), getHostRequest{}, http.StatusOK, &hostResp)
	// 3 profiles added + 1 profile with fleetd configuration + 1 root CA config
	require.Len(t, *hostResp.Host.MDM.Profiles, 5)

	// returns success, but this is effectively a no-op because the host isn't enrolled yet.
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", h.ID), nil, http.StatusOK)

	// we're going to modify this mock, make sure we restore its default
	originalPushMock := s.pushProvider.PushFunc
	defer func() { s.pushProvider.PushFunc = originalPushMock }()

	// if there's an error coming from APNs servers
	s.pushProvider.PushFunc = func(_ context.Context, pushes []*mdm.Push) (map[string]*push.Response, error) {
		return map[string]*push.Response{
			pushes[0].Token.String(): {
				Id:  uuid.New().String(),
				Err: errors.New("test"),
			},
		}, nil
	}
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", h.ID), nil, http.StatusBadGateway)

	// if there was an error unrelated to APNs
	s.pushProvider.PushFunc = func(_ context.Context, pushes []*mdm.Push) (map[string]*push.Response, error) {
		res := map[string]*push.Response{
			pushes[0].Token.String(): {
				Id:  uuid.New().String(),
				Err: nil,
			},
		}
		return res, errors.New("baz")
	}
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", h.ID), nil, http.StatusInternalServerError)

	// try again, but this time the host is online and answers
	var checkoutErr error
	s.pushProvider.PushFunc = func(ctx context.Context, pushes []*mdm.Push) (map[string]*push.Response, error) {
		res, err := mockSuccessfulPush(ctx, pushes)
		checkoutErr = mdmDevice.Checkout()
		return res, err
	}
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", h.ID), nil, http.StatusOK)
	// trying again fails with 409 as it is alreayd unenrolled
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", h.ID), nil, http.StatusConflict)

	require.NoError(t, checkoutErr)

	// profiles are removed and the host is no longer enrolled
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", h.ID), getHostRequest{}, http.StatusOK, &hostResp)
	require.Nil(t, hostResp.Host.MDM.Profiles)
	require.Equal(t, "", hostResp.Host.MDM.Name)
}

func (s *integrationMDMTestSuite) TestMDMDiskEncryptionSettingBackwardsCompat() {
	t := s.T()

	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": false }
  }`), http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableDiskEncryption.Value)

	// new config takes precedence over old config
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
	  "mdm": { "enable_disk_encryption": false, "macos_settings": {"enable_disk_encryption": true} }
  }`), http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableDiskEncryption.Value)

	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, false)

	// if new config is not present, old config is applied
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
	  "mdm": { "macos_settings": {"enable_disk_encryption": true} }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// new config takes precedence over old config again
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
	  "mdm": { "enable_disk_encryption": false, "macos_settings": {"enable_disk_encryption": true} }
  }`), http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableDiskEncryption.Value)
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, false)

	// unrelated change doesn't affect the disk encryption setting
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
	  "mdm": { "macos_settings": {"custom_settings": ["test.mobileconfig"]} }
  }`), http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableDiskEncryption.Value)

	// Same tests, but for teams
	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        "team1_" + t.Name(),
		Description: "desc team1_" + t.Name(),
	})
	require.NoError(t, err)

	checkTeamDiskEncryption := func(wantSetting bool) {
		var teamResp getTeamResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
		require.Equal(t, wantSetting, teamResp.Team.Config.MDM.EnableDiskEncryption)
	}

	// after creation, disk encryption is off
	checkTeamDiskEncryption(false)

	// new config takes precedence over old config
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: team.Name,
		MDM: fleet.TeamSpecMDM{
			EnableDiskEncryption: optjson.SetBool(false),
			MacOSSettings:        map[string]interface{}{"enable_disk_encryption": true},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	checkTeamDiskEncryption(false)
	s.assertConfigProfilesByIdentifier(ptr.Uint(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, false)

	// if new config is not present, old config is applied
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: team.Name,
		MDM: fleet.TeamSpecMDM{
			MacOSSettings: map[string]interface{}{"enable_disk_encryption": true},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	checkTeamDiskEncryption(true)
	s.assertConfigProfilesByIdentifier(ptr.Uint(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// new config takes precedence over old config again
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: team.Name,
		MDM: fleet.TeamSpecMDM{
			EnableDiskEncryption: optjson.SetBool(false),
			MacOSSettings:        map[string]interface{}{"enable_disk_encryption": true},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	checkTeamDiskEncryption(false)
	s.assertConfigProfilesByIdentifier(ptr.Uint(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, false)

	// unrelated change doesn't affect the disk encryption setting
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: team.Name,
		MDM: fleet.TeamSpecMDM{
			EnableDiskEncryption: optjson.SetBool(false),
			MacOSSettings:        map[string]interface{}{"custom_settings": []interface{}{"A", "B"}},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	checkTeamDiskEncryption(false)
	s.assertConfigProfilesByIdentifier(ptr.Uint(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, false)
}

func (s *integrationMDMTestSuite) TestDiskEncryptionSharedSetting() {
	t := s.T()

	// create a team
	teamName := t.Name()
	team := &fleet.Team{
		Name:        teamName,
		Description: "desc " + teamName,
	}
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &createTeamResp)
	require.NotZero(t, createTeamResp.Team.ID)

	setMDMEnabled := func(macMDM, windowsMDM bool) {
		appConf, err := s.ds.AppConfig(context.Background())
		require.NoError(s.T(), err)
		appConf.MDM.WindowsEnabledAndConfigured = windowsMDM
		appConf.MDM.EnabledAndConfigured = macMDM
		err = s.ds.SaveAppConfig(context.Background(), appConf)
		require.NoError(s.T(), err)
	}

	// before doing any modifications, grab the current values and make
	// sure they're set to the same ones on cleanup to not interfere with
	// other tests.
	origAppConf, err := s.ds.AppConfig(context.Background())
	require.NoError(s.T(), err)
	t.Cleanup(func() {
		err := s.ds.SaveAppConfig(context.Background(), origAppConf)
		require.NoError(s.T(), err)
	})

	checkConfigSetSucceeds := func() {
		res := s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusOK)
		errMsg := extractServerErrorText(res.Body)
		require.Empty(t, errMsg)

		// try to create a new team using specs
		teamSpecs := map[string]any{
			"specs": []any{
				map[string]any{
					"name": teamName + uuid.NewString(),
					"mdm": map[string]any{
						"enable_disk_encryption": true,
					},
				},
			},
		}
		res = s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
		errMsg = extractServerErrorText(res.Body)
		require.Empty(t, errMsg)

		// edit the existing team using specs
		teamSpecs = map[string]any{
			"specs": []any{
				map[string]any{
					"name": teamName,
					"mdm": map[string]any{
						"enable_disk_encryption": true,
					},
				},
			},
		}
		res = s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
		errMsg = extractServerErrorText(res.Body)
		require.Empty(t, errMsg)

		// always try to set the value to `false` so we start fresh
		s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": false }
  }`), http.StatusOK)
		teamSpecs = map[string]any{
			"specs": []any{
				map[string]any{
					"name": teamName,
					"mdm": map[string]any{
						"enable_disk_encryption": false,
					},
				},
			},
		}
		s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	}

	// MDM config succeeds because we have a private key baked into default suite config
	setMDMEnabled(false, false)
	checkConfigSetSucceeds()

	// enable windows mdm, no errors
	setMDMEnabled(false, true)
	checkConfigSetSucceeds()

	// enable mac mdm, no errors
	setMDMEnabled(true, true)
	checkConfigSetSucceeds()

	// only macos mdm enabled, no errors
	setMDMEnabled(true, false)
	checkConfigSetSucceeds()
}

func (s *integrationMDMTestSuite) TestEscrowBuddyBackwardsCompat() {
	t := s.T()
	ctx := context.Background()

	// create a host
	host, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	orbitKey := setOrbitEnrollment(t, host, s.ds)
	host.OrbitNodeKey = &orbitKey

	// install a filevault profile for that host
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)

	// set the status as non-decryptable so a notification should be sent
	err := s.ds.SetOrUpdateHostDiskEncryptionKey(ctx, host.ID, "", "", ptr.Bool(false))
	require.NoError(t, err)

	// notification is false because the escrow buddy capability is not set
	orbitConfigResp := orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &orbitConfigResp)
	require.False(t, orbitConfigResp.Notifications.RotateDiskEncryptionKey)

	// send the request again, this time with the right header
	orbitConfigResp = orbitGetConfigResponse{}
	res := s.DoRawWithHeaders("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, map[string]string{
		"Authorization":          fmt.Sprintf("Bearer %s", s.token),
		fleet.CapabilitiesHeader: string(fleet.CapabilityEscrowBuddy),
	})
	err = json.NewDecoder(res.Body).Decode(&orbitConfigResp)
	require.NoError(t, err)
	require.True(t, orbitConfigResp.Notifications.RotateDiskEncryptionKey)
}

func (s *integrationMDMTestSuite) TestMDMAppleHostDiskEncryption() {
	t := s.T()
	ctx := context.Background()

	// create a host
	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name()),
		NodeKey:         ptr.String(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	// install a filevault profile for that host

	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	fileVaultProf := s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)
	hostCmdUUID := uuid.New().String()
	err = s.ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
		{
			ProfileUUID:       fileVaultProf.ProfileUUID,
			ProfileIdentifier: fileVaultProf.Identifier,
			HostUUID:          host.UUID,
			CommandUUID:       hostCmdUUID,
			OperationType:     fleet.MDMOperationTypeInstall,
			Status:            &fleet.MDMDeliveryPending,
			Checksum:          []byte("csum"),
		},
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		err := s.ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
			HostUUID:      host.UUID,
			CommandUUID:   hostCmdUUID,
			ProfileUUID:   fileVaultProf.ProfileUUID,
			Status:        &fleet.MDMDeliveryVerifying,
			OperationType: fleet.MDMOperationTypeRemove,
		})
		require.NoError(t, err)
		// not an error if the profile does not exist
		_ = s.ds.DeleteMDMAppleConfigProfile(ctx, fileVaultProf.ProfileUUID)
	})

	// get that host - it should
	// report "enforcing" disk encryption
	getHostResp := getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Equal(t, fleet.DiskEncryptionEnforcing, *getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Nil(t, getHostResp.Host.MDM.MacOSSettings.ActionRequired)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, fleet.DiskEncryptionEnforcing, *getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, "", getHostResp.Host.MDM.OSSettings.DiskEncryption.Detail)

	// report a profile install error
	err = s.ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
		HostUUID:      host.UUID,
		CommandUUID:   hostCmdUUID,
		ProfileUUID:   fileVaultProf.ProfileUUID,
		Status:        &fleet.MDMDeliveryFailed,
		OperationType: fleet.MDMOperationTypeInstall,
		Detail:        "test error",
	})
	require.NoError(t, err)

	// get that host - it should report "failed" disk encryption and include the error message detail
	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Equal(t, fleet.DiskEncryptionFailed, *getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Nil(t, getHostResp.Host.MDM.MacOSSettings.ActionRequired)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, fleet.DiskEncryptionFailed, *getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, "test error", getHostResp.Host.MDM.OSSettings.DiskEncryption.Detail)

	// report that the profile was installed and verified
	err = s.ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
		HostUUID:      host.UUID,
		CommandUUID:   hostCmdUUID,
		ProfileUUID:   fileVaultProf.ProfileUUID,
		Status:        &fleet.MDMDeliveryVerified,
		OperationType: fleet.MDMOperationTypeInstall,
		Detail:        "",
	})
	require.NoError(t, err)

	// get that host - it has no encryption key at this point, so it should
	// report "action_required" disk encryption and "log_out" action.
	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Equal(t, fleet.DiskEncryptionActionRequired, *getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.NotNil(t, getHostResp.Host.MDM.MacOSSettings.ActionRequired)
	require.Equal(t, fleet.ActionRequiredRotateKey, *getHostResp.Host.MDM.MacOSSettings.ActionRequired)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, fleet.DiskEncryptionActionRequired, *getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, "", getHostResp.Host.MDM.OSSettings.DiskEncryption.Detail)

	// add an encryption key for the host
	assets, err := s.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{
		fleet.MDMAssetCACert,
	}, nil)
	require.NoError(t, err)

	block, _ := pem.Decode(assets[fleet.MDMAssetCACert].Value)
	require.NotNil(t, block)

	parsed, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	recoveryKey := "AAA-BBB-CCC"
	encryptedKey, err := pkcs7.Encrypt([]byte(recoveryKey), []*x509.Certificate{parsed})
	require.NoError(t, err)
	base64EncryptedKey := base64.StdEncoding.EncodeToString(encryptedKey)

	err = s.ds.SetOrUpdateHostDiskEncryptionKey(ctx, host.ID, base64EncryptedKey, "", nil)
	require.NoError(t, err)

	// get that host - it has an encryption key with unknown decryptability, so
	// it should report "verifying" disk encryption.
	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Equal(t, fleet.DiskEncryptionVerifying, *getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Nil(t, getHostResp.Host.MDM.MacOSSettings.ActionRequired)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, fleet.DiskEncryptionVerifying, *getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, "", getHostResp.Host.MDM.OSSettings.DiskEncryption.Detail)

	// request with no token
	res := s.DoRawNoAuth("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID), nil, http.StatusUnauthorized)
	res.Body.Close()

	// encryption key not processed yet
	resp := getHostEncryptionKeyResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID), nil, http.StatusNotFound, &resp)

	// unable to decrypt encryption key
	err = s.ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{host.ID}, false, time.Now())
	require.NoError(t, err)
	resp = getHostEncryptionKeyResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID), nil, http.StatusNotFound, &resp)

	// get that host - it has an encryption key that is un-decryptable, so it
	// should report "action_required" disk encryption and "rotate_key" action.
	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Equal(t, fleet.DiskEncryptionActionRequired, *getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.NotNil(t, getHostResp.Host.MDM.MacOSSettings.ActionRequired)
	require.Equal(t, fleet.ActionRequiredRotateKey, *getHostResp.Host.MDM.MacOSSettings.ActionRequired)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, fleet.DiskEncryptionActionRequired, *getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, "", getHostResp.Host.MDM.OSSettings.DiskEncryption.Detail)

	// no activities created so far
	activities := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activities)
	found := false
	for _, activity := range activities.Activities {
		if activity.Type == "read_host_disk_encryption_key" {
			found = true
		}
	}
	require.False(t, found)

	// decryptable key
	checkDecryptableKey := func(u fleet.User) {
		err = s.ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{host.ID}, true, time.Now())
		require.NoError(t, err)
		resp = getHostEncryptionKeyResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID), nil, http.StatusOK, &resp)
		require.Equal(t, recoveryKey, resp.EncryptionKey.DecryptedValue)

		// use the admin token to get the activities
		currToken := s.token
		defer func() { s.token = currToken }()
		s.token = s.getTestAdminToken()
		s.lastActivityMatches(
			"read_host_disk_encryption_key",
			fmt.Sprintf(`{"host_display_name": "%s", "host_id": %d}`, host.DisplayName(), host.ID),
			0,
		)
	}

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          4827,
		Name:        "team1_" + t.Name(),
		Description: "desc team1_" + t.Name(),
	})
	require.NoError(t, err)

	// enable disk encryption on the team so the key is not deleted when the host is added
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: "team1_" + t.Name(),
		MDM: fleet.TeamSpecMDM{
			EnableDiskEncryption: optjson.SetBool(true),
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	// we're about to mess up with the token, make sure to set it to the
	// default value when the test ends
	currToken := s.token
	t.Cleanup(func() { s.token = currToken })

	// admins are able to see the host encryption key
	s.token = s.getTestAdminToken()
	checkDecryptableKey(s.users["admin1@example.com"])

	// get that host - it has an encryption key that is decryptable, so it
	// should report "verified" disk encryption.
	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Equal(t, fleet.DiskEncryptionVerified, *getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Nil(t, getHostResp.Host.MDM.MacOSSettings.ActionRequired)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, fleet.DiskEncryptionVerified, *getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, "", getHostResp.Host.MDM.OSSettings.DiskEncryption.Detail)

	// maintainers are able to see the token
	u := s.users["user1@example.com"]
	s.token = s.getTestToken(u.Email, test.GoodPassword)
	checkDecryptableKey(u)

	// observers are able to see the token
	u = s.users["user2@example.com"]
	s.token = s.getTestToken(u.Email, test.GoodPassword)
	checkDecryptableKey(u)

	// add the host to a team
	err = s.ds.AddHostsToTeam(ctx, &team.ID, []uint{host.ID})
	require.NoError(t, err)

	// admins are still able to see the token
	s.token = s.getTestAdminToken()
	checkDecryptableKey(s.users["admin1@example.com"])

	// maintainers are still able to see the token
	u = s.users["user1@example.com"]
	s.token = s.getTestToken(u.Email, test.GoodPassword)
	checkDecryptableKey(u)

	// observers are still able to see the token
	u = s.users["user2@example.com"]
	s.token = s.getTestToken(u.Email, test.GoodPassword)
	checkDecryptableKey(u)

	// add a team member
	u = fleet.User{
		Name:       "test team user",
		Email:      "user1+team@example.com",
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *team,
				Role: fleet.RoleMaintainer,
			},
		},
	}
	require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(ctx, &u)
	require.NoError(t, err)

	// members are able to see the token
	s.token = s.getTestToken(u.Email, test.GoodPassword)
	checkDecryptableKey(u)

	// create a separate team
	team2, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          4828,
		Name:        "team2_" + t.Name(),
		Description: "desc team2_" + t.Name(),
	})
	require.NoError(t, err)
	// add a team member
	u = fleet.User{
		Name:       "test team user",
		Email:      "user1+team2@example.com",
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *team2,
				Role: fleet.RoleMaintainer,
			},
		},
	}
	require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(ctx, &u)
	require.NoError(t, err)

	// non-members aren't able to see the token
	s.token = s.getTestToken(u.Email, test.GoodPassword)
	resp = getHostEncryptionKeyResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID), nil, http.StatusForbidden, &resp)
}

func (s *integrationMDMTestSuite) TestWindowsMDMGetEncryptionKey() {
	t := s.T()
	ctx := context.Background()

	// create a host and enroll it in Fleet
	host := createOrbitEnrolledHost(t, "windows", "h1", s.ds)
	err := s.ds.SetOrUpdateMDMData(ctx, host.ID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, "")
	require.NoError(t, err)

	// request encryption key with no auth token
	res := s.DoRawNoAuth("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID), nil, http.StatusUnauthorized)
	res.Body.Close()

	// no encryption key
	resp := getHostEncryptionKeyResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID), nil, http.StatusNotFound, &resp)

	// invalid host id
	resp = getHostEncryptionKeyResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID+999), nil, http.StatusNotFound, &resp)

	// add an encryption key for the host
	cert, _, _, err := s.fleetCfg.MDM.MicrosoftWSTEP()
	require.NoError(t, err)
	recoveryKey := "AAA-BBB-CCC"
	encryptedKey, err := microsoft_mdm.Encrypt(recoveryKey, cert.Leaf)
	require.NoError(t, err)

	err = s.ds.SetOrUpdateHostDiskEncryptionKey(ctx, host.ID, encryptedKey, "", ptr.Bool(true))
	require.NoError(t, err)

	resp = getHostEncryptionKeyResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID), nil, http.StatusOK, &resp)
	require.Equal(t, host.ID, resp.HostID)
	require.Equal(t, recoveryKey, resp.EncryptionKey.DecryptedValue)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeReadHostDiskEncryptionKey{}.ActivityName(),
		fmt.Sprintf(`{"host_display_name": "%s", "host_id": %d}`, host.DisplayName(), host.ID), 0)

	// update the key to blank with a client error
	err = s.ds.SetOrUpdateHostDiskEncryptionKey(ctx, host.ID, "", "failed", nil)
	require.NoError(t, err)

	resp = getHostEncryptionKeyResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/encryption_key", host.ID), nil, http.StatusNotFound, &resp)
}

func (s *integrationMDMTestSuite) TestAppConfigMDMAppleDiskEncryption() {
	t := s.T()

	// set the macos disk encryption field
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	enabledDiskActID := s.lastActivityMatches(fleet.ActivityTypeEnabledMacosDiskEncryption{}.ActivityName(),
		`{"team_id": null, "team_name": null}`, 0)

	// will have generated the macos config profile
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// check that they are returned by a GET /config
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)

	// patch without specifying the macos disk encryption and an unrelated field,
	// should not alter it
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
                     "mdm": { "macos_settings": {"custom_settings": [{"path": "a"}]} }
		}`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	assert.Equal(t, []fleet.MDMProfileSpec{{Path: "a"}}, acResp.MDM.MacOSSettings.CustomSettings)
	s.lastActivityMatches(fleet.ActivityTypeEnabledMacosDiskEncryption{}.ActivityName(),
		``, enabledDiskActID)

	// patch with false, would reset it but this is a dry-run
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
				"mdm": { "enable_disk_encryption": false }
		  }`), http.StatusOK, &acResp, "dry_run", "true")
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	assert.Equal(t, []fleet.MDMProfileSpec{{Path: "a"}}, acResp.MDM.MacOSSettings.CustomSettings)
	s.lastActivityMatches(fleet.ActivityTypeEnabledMacosDiskEncryption{}.ActivityName(),
		``, enabledDiskActID)

	// patch with false, resets it
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
	    "mdm": { "enable_disk_encryption": false, "macos_settings": { "custom_settings": [{"path":"b"}] } }
		  }`), http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableDiskEncryption.Value)
	assert.Equal(t, []fleet.MDMProfileSpec{{Path: "b"}}, acResp.MDM.MacOSSettings.CustomSettings)
	s.lastActivityMatches(fleet.ActivityTypeDisabledMacosDiskEncryption{}.ActivityName(),
		`{"team_id": null, "team_name": null}`, 0)

	// will have deleted the macos config profile
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, false)

	// use the MDM disk encryption endpoint to set it to true
	s.Do("POST", "/api/latest/fleet/disk_encryption",
		updateDiskEncryptionRequest{EnableDiskEncryption: true}, http.StatusNoContent)
	enabledDiskActID = s.lastActivityMatches(fleet.ActivityTypeEnabledMacosDiskEncryption{}.ActivityName(),
		`{"team_id": null, "team_name": null}`, 0)

	// will have created the macos config profile
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)

	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	assert.Equal(t, []fleet.MDMProfileSpec{{Path: "b"}}, acResp.MDM.MacOSSettings.CustomSettings)

	// call update endpoint with no changes
	s.Do("PATCH", "/api/latest/fleet/mdm/apple/settings",
		fleet.MDMAppleSettingsPayload{}, http.StatusNoContent)
	s.lastActivityMatches(fleet.ActivityTypeEnabledMacosDiskEncryption{}.ActivityName(),
		``, enabledDiskActID)

	// the macos config profile still exists
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)

	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	assert.Equal(t, []fleet.MDMProfileSpec{{Path: "b"}}, acResp.MDM.MacOSSettings.CustomSettings)

	// mdm/apple/settings works for windows as well as it's being used by
	// clients (UI) this way
	appConf, err := s.ds.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.MDM.EnabledAndConfigured = false
	appConf.MDM.WindowsEnabledAndConfigured = true
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)
	defer func() {
		appConf, err := s.ds.AppConfig(context.Background())
		require.NoError(s.T(), err)
		appConf.MDM.EnabledAndConfigured = true
		appConf.MDM.WindowsEnabledAndConfigured = true
		err = s.ds.SaveAppConfig(context.Background(), appConf)
		require.NoError(s.T(), err)
	}()

	// flip and verify the value
	s.Do("POST", "/api/latest/fleet/disk_encryption",
		updateDiskEncryptionRequest{EnableDiskEncryption: false}, http.StatusNoContent)
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableDiskEncryption.Value)

	s.Do("POST", "/api/latest/fleet/disk_encryption",
		updateDiskEncryptionRequest{EnableDiskEncryption: true}, http.StatusNoContent)
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
}

func (s *integrationMDMTestSuite) TestMDMAppleDiskEncryptionAggregate() {
	t := s.T()
	ctx := context.Background()

	// no hosts with any disk encryption status's
	expectedNoTeamDiskEncryptionSummary := fleet.MDMDiskEncryptionSummary{}
	s.checkMDMDiskEncryptionSummaries(t, nil, expectedNoTeamDiskEncryptionSummary, true)
	expectedNoTeamProfilesSummary := fleet.MDMProfilesSummary{}
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamProfilesSummary, &expectedNoTeamProfilesSummary)

	// 10 new hosts
	var hosts []*fleet.Host
	for i := 0; i < 10; i++ {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-1 * time.Minute),
			OsqueryHostID:   ptr.String(fmt.Sprintf("%s-%d", t.Name(), i)),
			NodeKey:         ptr.String(fmt.Sprintf("%s-%d", t.Name(), i)),
			UUID:            fmt.Sprintf("%d-%s", i, uuid.New().String()),
			Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
			Platform:        "darwin",
		})
		require.NoError(t, err)
		hosts = append(hosts, h)
	}

	// no team tests ====

	// new filevault profile with no team
	prof, err := fleet.NewMDMAppleConfigProfile(mobileconfigForTest("filevault-1", mobileconfig.FleetFileVaultPayloadIdentifier), ptr.Uint(0))
	require.NoError(t, err)

	// generates a disk encryption aggregate value based on the arguments passed in
	generateAggregateValue := func(
		hosts []*fleet.Host,
		operationType fleet.MDMOperationType,
		status *fleet.MDMDeliveryStatus,
		decryptable bool,
	) {
		for _, host := range hosts {
			hostCmdUUID := uuid.New().String()
			err := s.ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
				{
					ProfileUUID:       prof.ProfileUUID,
					ProfileIdentifier: prof.Identifier,
					HostUUID:          host.UUID,
					CommandUUID:       hostCmdUUID,
					OperationType:     operationType,
					Status:            status,
					Checksum:          []byte("csum"),
				},
			})
			require.NoError(t, err)
			oneMinuteAfterThreshold := time.Now().Add(+1 * time.Minute)
			err = s.ds.SetOrUpdateHostDiskEncryptionKey(ctx, host.ID, "test-key", "", nil)
			require.NoError(t, err)
			err = s.ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{host.ID}, decryptable, oneMinuteAfterThreshold)
			require.NoError(t, err)
		}
	}

	// hosts 1,2 have disk encryption "applied" status
	generateAggregateValue(hosts[0:2], fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerifying, true)
	expectedNoTeamDiskEncryptionSummary.Verifying.MacOS = 2
	s.checkMDMDiskEncryptionSummaries(t, nil, expectedNoTeamDiskEncryptionSummary, true)
	expectedNoTeamProfilesSummary.Verifying = 2
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamProfilesSummary, &expectedNoTeamProfilesSummary)

	// hosts 3,4 have disk encryption "action required" status
	generateAggregateValue(hosts[2:4], fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerifying, false)
	expectedNoTeamDiskEncryptionSummary.ActionRequired.MacOS = 2
	s.checkMDMDiskEncryptionSummaries(t, nil, expectedNoTeamDiskEncryptionSummary, true)
	expectedNoTeamProfilesSummary.Pending = 2
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamProfilesSummary, &expectedNoTeamProfilesSummary)

	// hosts 5,6 have disk encryption "enforcing" status

	// host profiles status are `pending`
	generateAggregateValue(hosts[4:6], fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryPending, true)
	expectedNoTeamDiskEncryptionSummary.Enforcing.MacOS = 2
	s.checkMDMDiskEncryptionSummaries(t, nil, expectedNoTeamDiskEncryptionSummary, true)
	expectedNoTeamProfilesSummary.Pending = 4
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamProfilesSummary, &expectedNoTeamProfilesSummary)

	// host profiles status dont exist
	generateAggregateValue(hosts[4:6], fleet.MDMOperationTypeInstall, nil, true)
	s.checkMDMDiskEncryptionSummaries(t, nil, expectedNoTeamDiskEncryptionSummary, true)               // no change
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamProfilesSummary, &expectedNoTeamProfilesSummary) // no change

	// host profile is applied but decryptable key does not exist
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(
			context.Background(),
			"UPDATE host_disk_encryption_keys SET decryptable = NULL WHERE host_id IN (?, ?)",
			hosts[5].ID,
			hosts[6].ID,
		)
		require.NoError(t, err)
		return err
	})
	s.checkMDMDiskEncryptionSummaries(t, nil, expectedNoTeamDiskEncryptionSummary, true)               // no change
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamProfilesSummary, &expectedNoTeamProfilesSummary) // no change

	// hosts 7,8 have disk encryption "failed" status
	generateAggregateValue(hosts[6:8], fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryFailed, true)
	expectedNoTeamDiskEncryptionSummary.Failed.MacOS = 2
	s.checkMDMDiskEncryptionSummaries(t, nil, expectedNoTeamDiskEncryptionSummary, true)
	expectedNoTeamProfilesSummary.Failed = 2
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamProfilesSummary, &expectedNoTeamProfilesSummary)

	// hosts 9,10 have disk encryption "removing enforcement" status
	generateAggregateValue(hosts[8:10], fleet.MDMOperationTypeRemove, &fleet.MDMDeliveryPending, true)
	expectedNoTeamDiskEncryptionSummary.RemovingEnforcement.MacOS = 2
	s.checkMDMDiskEncryptionSummaries(t, nil, expectedNoTeamDiskEncryptionSummary, true)
	expectedNoTeamProfilesSummary.Pending = 6
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamProfilesSummary, &expectedNoTeamProfilesSummary)

	// team tests ====

	// host 1,2 added to team 1
	tm, _ := s.ds.NewTeam(ctx, &fleet.Team{Name: "team-1"})
	err = s.ds.AddHostsToTeam(ctx, &tm.ID, []uint{hosts[0].ID, hosts[1].ID})
	require.NoError(t, err)

	// new filevault profile for team 1
	prof, err = fleet.NewMDMAppleConfigProfile(mobileconfigForTest("filevault-1", mobileconfig.FleetFileVaultPayloadIdentifier), ptr.Uint(1))
	require.NoError(t, err)
	prof.TeamID = &tm.ID
	require.NoError(t, err)

	// filtering by the "team_id" query param
	generateAggregateValue(hosts[0:2], fleet.MDMOperationTypeInstall, &fleet.MDMDeliveryVerifying, true)

	var expectedTeamDiskEncryptionSummary fleet.MDMDiskEncryptionSummary
	expectedTeamDiskEncryptionSummary.Verifying.MacOS = 2
	s.checkMDMDiskEncryptionSummaries(t, &tm.ID, expectedTeamDiskEncryptionSummary, true)

	expectedNoTeamDiskEncryptionSummary.Verifying.MacOS = 0 // now 0 because hosts 1,2 were added to team 1
	s.checkMDMDiskEncryptionSummaries(t, nil, expectedNoTeamDiskEncryptionSummary, true)

	expectedTeamProfilesSummary := fleet.MDMProfilesSummary{Verifying: 2}
	s.checkMDMProfilesSummaries(t, &tm.ID, expectedTeamProfilesSummary, &expectedTeamProfilesSummary)

	expectedNoTeamProfilesSummary = fleet.MDMProfilesSummary{
		Verifying: 0, // now 0 because hosts 1,2 were added to team 1
		Pending:   6,
		Failed:    2,
	}
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamProfilesSummary, &expectedNoTeamProfilesSummary)

	// verified status for host 1
	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, s.ds, hosts[0], map[string]*fleet.HostMacOSProfile{prof.Identifier: {Identifier: prof.Identifier, DisplayName: prof.Name, InstallDate: time.Now()}}))
	// TODO: Why is there no change to the verification status of host 1 reflected in the summaries?
	s.checkMDMDiskEncryptionSummaries(t, &tm.ID, expectedTeamDiskEncryptionSummary, true)              // no change
	s.checkMDMDiskEncryptionSummaries(t, nil, expectedNoTeamDiskEncryptionSummary, true)               // no change
	s.checkMDMProfilesSummaries(t, &tm.ID, expectedTeamProfilesSummary, &expectedTeamProfilesSummary)  // no change
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamProfilesSummary, &expectedNoTeamProfilesSummary) // no change
}

func (s *integrationMDMTestSuite) TestTeamsMDMAppleDiskEncryption() {
	t := s.T()

	// create a team through the service so it initializes the agent ops
	teamName := t.Name() + "team1"
	team := &fleet.Team{
		Name:        teamName,
		Description: "desc team1",
	}
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &createTeamResp)
	require.NotZero(t, createTeamResp.Team.ID)
	team = createTeamResp.Team

	// no macos config profile yet
	s.assertConfigProfilesByIdentifier(ptr.Uint(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, false)

	// apply with disk encryption
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			EnableDiskEncryption: optjson.SetBool(true),
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	lastDiskActID := s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledMacosDiskEncryption{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q}`, team.ID, teamName), 0)

	// macos config profile created
	s.assertConfigProfilesByIdentifier(ptr.Uint(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// retrieving the team returns the disk encryption setting
	var teamResp getTeamResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.True(t, teamResp.Team.Config.MDM.EnableDiskEncryption)

	// apply with invalid disk encryption value should fail
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSSettings: map[string]interface{}{"enable_disk_encryption": 123},
		},
	}}}
	res := s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, `invalid value type at 'macos_settings.enable_disk_encryption': expected bool but got float64`)

	// apply an empty set of batch profiles to the team
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: nil},
		http.StatusUnprocessableEntity, "team_id", fmt.Sprint(team.ID), "team_name", team.Name)

	// the configuration profile is still there
	s.assertConfigProfilesByIdentifier(ptr.Uint(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// apply without disk encryption settings specified and unrelated field,
	// should not replace existing disk encryption
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSSettings: map[string]interface{}{
				"custom_settings": []map[string]interface{}{
					{"path": "a"},
				},
			},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.True(t, teamResp.Team.Config.MDM.EnableDiskEncryption)
	require.Equal(t, []fleet.MDMProfileSpec{{Path: "a"}}, teamResp.Team.Config.MDM.MacOSSettings.CustomSettings)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledMacosDiskEncryption{}.ActivityName(),
		``, lastDiskActID)

	// apply with false would clear the existing setting, but dry-run
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			EnableDiskEncryption: optjson.SetBool(false),
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, "dry_run", "true")
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.True(t, teamResp.Team.Config.MDM.EnableDiskEncryption)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledMacosDiskEncryption{}.ActivityName(),
		``, lastDiskActID)

	// apply with false clears the existing setting
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSSettings: map[string]interface{}{"enable_disk_encryption": false},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.False(t, teamResp.Team.Config.MDM.EnableDiskEncryption)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeDisabledMacosDiskEncryption{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q}`, team.ID, teamName), 0)

	// macos config profile deleted
	s.assertConfigProfilesByIdentifier(ptr.Uint(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, false)

	// modify team's disk encryption via ModifyTeam endpoint
	var modResp teamResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		MDM: &fleet.TeamPayloadMDM{
			EnableDiskEncryption: optjson.SetBool(true),
		},
	}, http.StatusOK, &modResp)
	require.True(t, modResp.Team.Config.MDM.EnableDiskEncryption)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledMacosDiskEncryption{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q}`, team.ID, teamName), 0)

	// macos config profile created
	s.assertConfigProfilesByIdentifier(ptr.Uint(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// modify team's disk encryption and description via ModifyTeam endpoint
	modResp = teamResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Description: ptr.String("foobar"),
		MDM: &fleet.TeamPayloadMDM{
			EnableDiskEncryption: optjson.SetBool(false),
		},
	}, http.StatusOK, &modResp)
	require.False(t, modResp.Team.Config.MDM.EnableDiskEncryption)
	require.Equal(t, "foobar", modResp.Team.Description)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeDisabledMacosDiskEncryption{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q}`, team.ID, teamName), 0)

	// macos config profile deleted
	s.assertConfigProfilesByIdentifier(ptr.Uint(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, false)

	// use the MDM settings endpoint to set it to true
	s.Do("POST", "/api/latest/fleet/disk_encryption",
		updateDiskEncryptionRequest{TeamID: ptr.Uint(team.ID), EnableDiskEncryption: true}, http.StatusNoContent)
	lastDiskActID = s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledMacosDiskEncryption{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q}`, team.ID, teamName), 0)

	// macos config profile created
	s.assertConfigProfilesByIdentifier(ptr.Uint(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, true)

	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.True(t, teamResp.Team.Config.MDM.EnableDiskEncryption)

	// use the MDM settings endpoint with no changes
	s.Do("PATCH", "/api/latest/fleet/mdm/apple/settings",
		fleet.MDMAppleSettingsPayload{TeamID: ptr.Uint(team.ID)}, http.StatusNoContent)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledMacosDiskEncryption{}.ActivityName(),
		``, lastDiskActID)

	// macos config profile still exists
	s.assertConfigProfilesByIdentifier(ptr.Uint(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, true)

	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.True(t, teamResp.Team.Config.MDM.EnableDiskEncryption)

	// use the MDM settings endpoint with an unknown team id
	s.Do("POST", "/api/latest/fleet/disk_encryption",
		updateDiskEncryptionRequest{TeamID: ptr.Uint(9999), EnableDiskEncryption: true}, http.StatusNotFound)

	// mdm/apple/settings works for windows as well as it's being used by
	// clients (UI) this way
	appConf, err := s.ds.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.MDM.EnabledAndConfigured = false
	appConf.MDM.WindowsEnabledAndConfigured = true
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)
	defer func() {
		appConf, err := s.ds.AppConfig(context.Background())
		require.NoError(s.T(), err)
		appConf.MDM.EnabledAndConfigured = true
		appConf.MDM.WindowsEnabledAndConfigured = true
		err = s.ds.SaveAppConfig(context.Background(), appConf)
		require.NoError(s.T(), err)
	}()

	// flip and verify the value
	s.Do("POST", "/api/latest/fleet/disk_encryption",
		updateDiskEncryptionRequest{TeamID: ptr.Uint(team.ID), EnableDiskEncryption: false}, http.StatusNoContent)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.False(t, teamResp.Team.Config.MDM.EnableDiskEncryption)

	s.Do("POST", "/api/latest/fleet/disk_encryption",
		updateDiskEncryptionRequest{TeamID: ptr.Uint(team.ID), EnableDiskEncryption: true}, http.StatusNoContent)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.True(t, teamResp.Team.Config.MDM.EnableDiskEncryption)
}

func (s *integrationMDMTestSuite) TestEnrollOrbitAfterDEPSync() {
	t := s.T()
	ctx := context.Background()

	// create a host with minimal information and the serial, no uuid/osquery id
	// (as when created via DEP sync).
	dbZeroTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	h, err := s.ds.NewHost(ctx, &fleet.Host{
		HardwareSerial:   uuid.New().String(),
		Platform:         "darwin",
		LastEnrolledAt:   dbZeroTime,
		DetailUpdatedAt:  dbZeroTime,
		RefetchRequested: true,
	})
	require.NoError(t, err)
	require.Equal(t, dbZeroTime, h.LastEnrolledAt)

	// create an enroll secret
	secret := uuid.New().String()
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: secret}},
		},
	}, http.StatusOK, &applyResp)

	// enroll the host from orbit, it should match the host above via the serial
	var resp EnrollOrbitResponse
	hostUUID := uuid.New().String()
	h.ComputerName = "My Mac"
	h.HardwareModel = "MacBook Pro"
	s.DoJSON("POST", "/api/fleet/orbit/enroll", EnrollOrbitRequest{
		EnrollSecret:   secret,
		HardwareUUID:   hostUUID, // will not match any existing host
		HardwareSerial: h.HardwareSerial,
		ComputerName:   h.ComputerName,
		HardwareModel:  h.HardwareModel,
	}, http.StatusOK, &resp)
	require.NotEmpty(t, resp.OrbitNodeKey)

	// fetch the host, it will match the one created above
	// (NOTE: cannot check the returned OrbitNodeKey, this field is not part of the response)
	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", h.ID), nil, http.StatusOK, &hostResp)
	require.Equal(t, h.ID, hostResp.Host.ID)
	require.NotEqual(t, dbZeroTime, hostResp.Host.LastEnrolledAt)
	assert.Equal(t, h.ComputerName, hostResp.Host.ComputerName)
	assert.Equal(t, h.HardwareModel, hostResp.Host.HardwareModel)
	assert.Equal(t, h.HardwareSerial, hostResp.Host.HardwareSerial)
	assert.Equal(t, h.DisplayName(), hostResp.Host.DisplayName)

	got, err := s.ds.LoadHostByOrbitNodeKey(ctx, resp.OrbitNodeKey)
	require.NoError(t, err)
	require.Equal(t, h.ID, got.ID)

	s.lastActivityMatches(
		"fleet_enrolled",
		fmt.Sprintf(`{"host_display_name": "%s", "host_serial": "%s"}`, h.DisplayName(), h.HardwareSerial),
		0,
	)

	// enroll the host from osquery, it should match the same host
	var osqueryResp enrollAgentResponse
	osqueryID := uuid.New().String()
	s.DoJSON("POST", "/api/osquery/enroll", enrollAgentRequest{
		EnrollSecret:   secret,
		HostIdentifier: osqueryID, // osquery host_identifier may not be the same as the host UUID, simulate that here
		HostDetails: map[string]map[string]string{
			"system_info": {
				"uuid":            hostUUID,
				"hardware_serial": h.HardwareSerial,
			},
		},
	}, http.StatusOK, &osqueryResp)
	require.NotEmpty(t, osqueryResp.NodeKey)

	// load the host by osquery node key, should match the initial host
	got, err = s.ds.LoadHostByNodeKey(ctx, osqueryResp.NodeKey)
	require.NoError(t, err)
	require.Equal(t, h.ID, got.ID)
}

func (s *integrationMDMTestSuite) TestFleetdConfiguration() {
	t := s.T()
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetdConfigPayloadIdentifier, false)

	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: t.Name()}},
		},
	}, http.StatusOK, &applyResp)

	// a new fleetd configuration profile for "no team" is created
	s.awaitTriggerProfileSchedule(t)
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetdConfigPayloadIdentifier, true)

	// create a new team
	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name(),
		Description: "desc",
	})
	require.NoError(t, err)
	s.assertConfigProfilesByIdentifier(&tm.ID, mobileconfig.FleetdConfigPayloadIdentifier, false)

	// upload an ABM token
	s.enableABM(t.Name())

	// set the default bm assignment to that team
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": {
			"apple_business_manager": [{
			  "organization_name": %q,
			  "macos_team": %q,
			  "ios_team": %q,
			  "ipados_team": %q
			}]
		}
	}`, t.Name(), tm.Name, tm.Name, tm.Name)), http.StatusOK, &acResp)
	t.Cleanup(func() {
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"apple_business_manager": []
			}
		}`), http.StatusOK, &acResp)
	})

	// the team doesn't have any enroll secrets yet, a profile is created using the global enroll secret
	s.awaitTriggerProfileSchedule(t)
	p := s.assertConfigProfilesByIdentifier(&tm.ID, mobileconfig.FleetdConfigPayloadIdentifier, true)
	require.Contains(t, string(p.Mobileconfig), t.Name())

	// create an enroll secret for the team
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name:    tm.Name,
		Secrets: &[]fleet.EnrollSecret{{Secret: t.Name() + "team-secret"}},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	// a new fleetd configuration profile for that team is created
	s.awaitTriggerProfileSchedule(t)
	p = s.assertConfigProfilesByIdentifier(&tm.ID, mobileconfig.FleetdConfigPayloadIdentifier, true)
	require.Contains(t, string(p.Mobileconfig), t.Name()+"team-secret")

	// the old configuration profile is kept
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetdConfigPayloadIdentifier, true)
}

func (s *integrationMDMTestSuite) TestEnqueueMDMCommand() {
	ctx := context.Background()
	t := s.T()
	s.setSkipWorkerJobs(t)

	// list commands should return all the commands we sent
	var listCmdResp0 listMDMAppleCommandsResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/commands", nil, http.StatusOK, &listCmdResp0)
	require.Empty(t, listCmdResp0.Results)

	// Create host enrolled via osquery, but not enrolled in MDM.
	unenrolledHost := createHostAndDeviceToken(t, s.ds, "unused")

	// Create device enrolled in MDM but not enrolled via osquery.
	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "MacBookPro16,1")
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	base64Cmd := func(rawCmd string) string {
		return base64.RawStdEncoding.EncodeToString([]byte(rawCmd))
	}

	newRawCmd := func(cmdUUID string) string {
		return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Command</key>
    <dict>
        <key>ManagedOnly</key>
        <false/>
        <key>RequestType</key>
        <string>ProfileList</string>
    </dict>
    <key>CommandUUID</key>
    <string>%s</string>
</dict>
</plist>`, cmdUUID)
	}

	// call with unknown host UUID
	uuid1 := uuid.New().String()
	s.Do("POST", "/api/latest/fleet/mdm/apple/enqueue",
		enqueueMDMAppleCommandRequest{
			// explicitly use standard encoding to make sure it also works
			// see #11384
			Command:   base64.StdEncoding.EncodeToString([]byte(newRawCmd(uuid1))),
			DeviceIDs: []string{"no-such-host"},
		}, http.StatusNotFound)

	// get command results returns 404, that command does not exist
	var cmdResResp getMDMAppleCommandResultsResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/commandresults", nil, http.StatusNotFound, &cmdResResp, "command_uuid", uuid1)
	var getMDMCmdResp getMDMCommandResultsResponse
	s.DoJSON("GET", "/api/latest/fleet/commands/results", nil, http.StatusNotFound, &cmdResResp, "command_uuid", uuid1)

	// list commands returns empty set
	var listCmdResp listMDMAppleCommandsResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/commands", nil, http.StatusOK, &listCmdResp)
	require.Empty(t, listCmdResp.Results)

	// call with unenrolled host UUID
	res := s.Do("POST", "/api/latest/fleet/mdm/apple/enqueue",
		enqueueMDMAppleCommandRequest{
			Command:   base64Cmd(newRawCmd(uuid.New().String())),
			DeviceIDs: []string{unenrolledHost.UUID},
		}, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "at least one of the hosts is not enrolled in MDM")

	// create a new Host to get the UUID on the DB
	linuxHost := createOrbitEnrolledHost(t, "linux", "h1", s.ds)
	windowsHost := createOrbitEnrolledHost(t, "windows", "h2", s.ds)
	// call with unenrolled host UUID
	res = s.Do("POST", "/api/latest/fleet/mdm/apple/enqueue",
		enqueueMDMAppleCommandRequest{
			Command:   base64Cmd(newRawCmd(uuid.New().String())),
			DeviceIDs: []string{linuxHost.UUID, windowsHost.UUID},
		}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "at least one of the hosts is not enrolled in MDM or is not an elegible device")

	// call with payload that is not a valid, plist-encoded MDM command
	res = s.Do("POST", "/api/latest/fleet/mdm/apple/enqueue",
		enqueueMDMAppleCommandRequest{
			Command:   base64Cmd(string(mobileconfigForTest("test config profile", uuid.New().String()))),
			DeviceIDs: []string{mdmDevice.UUID},
		}, http.StatusUnsupportedMediaType)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "unable to decode plist command")

	// call with enrolled host UUID
	uuid2 := uuid.New().String()
	rawCmd := newRawCmd(uuid2)
	var resp enqueueMDMAppleCommandResponse
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/enqueue",
		enqueueMDMAppleCommandRequest{
			Command:   base64Cmd(rawCmd),
			DeviceIDs: []string{mdmDevice.UUID},
		}, http.StatusOK, &resp)
	require.NotEmpty(t, resp.CommandUUID)
	require.Contains(t, rawCmd, resp.CommandUUID)
	require.Equal(t, resp.Platform, "darwin")
	require.Empty(t, resp.FailedUUIDs)
	require.Equal(t, "ProfileList", resp.RequestType)

	// the command exists but no results yet
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/commandresults", nil, http.StatusOK, &cmdResResp, "command_uuid", uuid2)
	require.Len(t, cmdResResp.Results, 0)
	s.DoJSON("GET", "/api/latest/fleet/commands/results", nil, http.StatusOK, &getMDMCmdResp, "command_uuid", uuid2)
	require.Len(t, getMDMCmdResp.Results, 0)

	// simulate a result and call again
	err = s.mdmStorage.StoreCommandReport(&mdm.Request{
		EnrollID: &mdm.EnrollID{ID: mdmDevice.UUID},
		Context:  ctx,
	}, &mdm.CommandResults{
		CommandUUID: uuid2,
		Status:      "Acknowledged",
		Raw:         []byte(rawCmd),
	})
	require.NoError(t, err)

	h, err := s.ds.HostByIdentifier(ctx, mdmDevice.UUID)
	require.NoError(t, err)
	h.Hostname = "test-host"
	err = s.ds.UpdateHost(ctx, h)
	require.NoError(t, err)

	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/commandresults", nil, http.StatusOK, &cmdResResp, "command_uuid", uuid2)
	require.Len(t, cmdResResp.Results, 1)
	require.NotZero(t, cmdResResp.Results[0].UpdatedAt)
	cmdResResp.Results[0].UpdatedAt = time.Time{}
	require.Equal(t, &fleet.MDMCommandResult{
		HostUUID:    mdmDevice.UUID,
		CommandUUID: uuid2,
		Status:      "Acknowledged",
		RequestType: "ProfileList",
		Result:      []byte(rawCmd),
		Payload:     []byte(rawCmd),
		Hostname:    "test-host",
	}, cmdResResp.Results[0])

	s.DoJSON("GET", "/api/latest/fleet/commands/results", nil, http.StatusOK, &getMDMCmdResp, "command_uuid", uuid2)
	require.Len(t, getMDMCmdResp.Results, 1)
	require.NotZero(t, getMDMCmdResp.Results[0].UpdatedAt)
	getMDMCmdResp.Results[0].UpdatedAt = time.Time{}
	require.Equal(t, &fleet.MDMCommandResult{
		HostUUID:    mdmDevice.UUID,
		CommandUUID: uuid2,
		Status:      "Acknowledged",
		RequestType: "ProfileList",
		Result:      []byte(rawCmd),
		Payload:     []byte(rawCmd),
		Hostname:    "test-host",
	}, getMDMCmdResp.Results[0])

	// list commands returns that command
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/commands", nil, http.StatusOK, &listCmdResp)
	results, err := json.Marshal(listCmdResp.Results)
	require.NoError(t, err)
	t.Logf("GET /api/latest/fleet/mdm/apple/commands response:\n%s", results)

	// filter to the expected command.
	// there may be other commands due to the bootstrap packages uploaded in prior tests.
	// TODO: decouple these tests so we don't have to do this -- perhaps delete bootstrap packages?
	var profileListCommands []*fleet.MDMAppleCommand
	for _, result := range listCmdResp.Results {
		if result.RequestType == "ProfileList" {
			profileListCommands = append(profileListCommands, result)
		}
	}

	require.Len(t, profileListCommands, 1)
	require.NotZero(t, profileListCommands[0].UpdatedAt)
	profileListCommands[0].UpdatedAt = time.Time{}
	require.Equal(t, &fleet.MDMAppleCommand{
		DeviceID:    mdmDevice.UUID,
		CommandUUID: uuid2,
		Status:      "Acknowledged",
		RequestType: "ProfileList",
		Hostname:    "test-host",
	}, profileListCommands[0])
}

// setSkipWorkerJobs sets the skipWorkerJobs flag to true for the duration of the test.
// This avoids running into timing issues with the test.
// We can manually run the jobs if needed with s.runWorker().
func (s *integrationMDMTestSuite) setSkipWorkerJobs(t *testing.T) {
	s.skipWorkerJobs.Store(true)
	t.Cleanup(func() { s.skipWorkerJobs.Store(false) })
}

func (s *integrationMDMTestSuite) TestEnqueueMDMCommandWithSecret() {
	t := s.T()
	s.setSkipWorkerJobs(t)

	_, mdmClient := createHostThenEnrollMDM(s.ds, s.server.URL, t)

	// list commands should return all the commands we sent
	var listCmdResp listMDMAppleCommandsResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/commands", nil, http.StatusOK, &listCmdResp, "host_identifier", mdmClient.UUID)
	require.Empty(t, listCmdResp.Results)

	base64Cmd := func(rawCmd string) string {
		return base64.RawStdEncoding.EncodeToString([]byte(rawCmd))
	}

	newRawCmd := func(cmdUUID string) string {
		return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Command</key>
    <dict>
        <key>ManagedOnly</key>
        <false/>
        <key>RequestType</key>
        <string>ProfileList</string>
		<key>SecretValue</key>
		<string>$FLEET_SECRET_VALUE</string>
    </dict>
    <key>CommandUUID</key>
    <string>%s</string>
</dict>
</plist>`, cmdUUID)
	}

	// Load secret(s)
	secretValue := "*abc123*"
	req := secretVariablesRequest{
		SecretVariables: []fleet.SecretVariable{
			{
				Name:  "FLEET_SECRET_VALUE",
				Value: secretValue,
			},
		},
	}
	secretResp := secretVariablesResponse{}
	s.DoJSON("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusOK, &secretResp)

	// call with enrolled host UUID
	uuid2 := uuid.New().String()
	rawCmd := newRawCmd(uuid2)
	var resp enqueueMDMAppleCommandResponse
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/enqueue",
		enqueueMDMAppleCommandRequest{
			Command:   base64Cmd(rawCmd),
			DeviceIDs: []string{mdmClient.UUID},
		}, http.StatusOK, &resp)
	require.NotEmpty(t, resp.CommandUUID)
	require.Contains(t, rawCmd, resp.CommandUUID)
	require.Equal(t, resp.Platform, "darwin")
	require.Empty(t, resp.FailedUUIDs)
	require.Equal(t, "ProfileList", resp.RequestType)

	// 1 command queued up
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/commands", nil, http.StatusOK, &listCmdResp, "host_identifier", mdmClient.UUID)
	require.Len(t, listCmdResp.Results, 1)

	cmd, err := mdmClient.Idle()
	require.NoError(t, err)
	assert.Contains(t, string(cmd.Raw), secretValue)
	assert.NotContains(t, string(cmd.Raw), "FLEET_SECRET_VALUE")
}

func (s *integrationMDMTestSuite) TestMDMWindowsCommandResults() {
	ctx := context.Background()
	t := s.T()

	h, err := s.ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-win-host-name",
		OsqueryHostID: ptr.String("1337"),
		NodeKey:       ptr.String("1337"),
		UUID:          "test-win-host-uuid",
		Platform:      "windows",
	})
	require.NoError(t, err)

	dev := &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            "test-device-id",
		MDMHardwareID:          "test-hardware-id",
		MDMDeviceState:         "ds",
		MDMDeviceType:          "dt",
		MDMDeviceName:          "dn",
		MDMEnrollType:          "et",
		MDMEnrollUserID:        "euid",
		MDMEnrollProtoVersion:  "epv",
		MDMEnrollClientVersion: "ecv",
		MDMNotInOOBE:           false,
		HostUUID:               h.UUID,
	}

	require.NoError(t, s.ds.MDMWindowsInsertEnrolledDevice(ctx, dev))
	var enrollmentID uint

	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &enrollmentID, `SELECT id FROM mdm_windows_enrollments WHERE mdm_device_id = ?`, dev.MDMDeviceID)
	})

	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`UPDATE mdm_windows_enrollments SET host_uuid = ? WHERE id = ?`, dev.HostUUID, enrollmentID)
		return err
	})

	rawCmd := "some-command"
	cmdUUID := "some-uuid"
	cmdTarget := "some-target-loc-uri"

	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO windows_mdm_commands (command_uuid, raw_command, target_loc_uri) VALUES (?, ?, ?)`, cmdUUID, rawCmd, cmdTarget)
		return err
	})

	var responseID int64
	rawResponse := []byte("some-response")
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		res, err := q.ExecContext(ctx, `INSERT INTO windows_mdm_responses (enrollment_id, raw_response) VALUES (?, ?)`, enrollmentID, rawResponse)
		if err != nil {
			return err
		}
		responseID, err = res.LastInsertId()
		return err
	})

	rawResult := []byte("some-result")
	statusCode := "200"
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO windows_mdm_command_results (enrollment_id, command_uuid, raw_result, response_id, status_code) VALUES (?, ?, ?, ?, ?)`, enrollmentID, cmdUUID, rawResult, responseID, statusCode)
		return err
	})

	var resp getMDMCommandResultsResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/commands/results?command_uuid=%s", cmdUUID), nil, http.StatusOK, &resp)
	require.Len(t, resp.Results, 1)
	require.Equal(t, dev.HostUUID, resp.Results[0].HostUUID)
	require.Equal(t, cmdUUID, resp.Results[0].CommandUUID)
	require.Equal(t, rawResponse, resp.Results[0].Result)
	require.Equal(t, cmdTarget, resp.Results[0].RequestType)
	require.Equal(t, statusCode, resp.Results[0].Status)
	require.Equal(t, h.Hostname, resp.Results[0].Hostname)

	resp = getMDMCommandResultsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/commands/results?command_uuid=%s", uuid.New().String()), nil, http.StatusNotFound, &resp)
	require.Empty(t, resp.Results)
}

func (s *integrationMDMTestSuite) TestAppConfigMDMMacOSMigration() {
	t := s.T()

	checkDefaultAppConfig := func() {
		var ac appConfigResponse
		s.DoJSON("GET", "/api/v1/fleet/config", nil, http.StatusOK, &ac)
		require.False(t, ac.MDM.MacOSMigration.Enable)
		require.Empty(t, ac.MDM.MacOSMigration.Mode)
		require.Empty(t, ac.MDM.MacOSMigration.WebhookURL)
	}
	checkDefaultAppConfig()

	var acResp appConfigResponse
	// missing webhook_url
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"mdm": { "macos_migration": { "enable": true, "mode": "voluntary", "webhook_url": "" } }
  	}`), http.StatusUnprocessableEntity, &acResp)
	checkDefaultAppConfig()

	// invalid url scheme for webhook_url
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"mdm": { "macos_migration": { "enable": true, "mode": "voluntary", "webhook_url": "ftp://example.com" } }
	}`), http.StatusUnprocessableEntity, &acResp)
	checkDefaultAppConfig()

	// invalid mode
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"mdm": { "macos_migration": { "enable": true, "mode": "foobar", "webhook_url": "https://example.com" } }
  	}`), http.StatusUnprocessableEntity, &acResp)
	checkDefaultAppConfig()

	// valid request
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"mdm": { "macos_migration": { "enable": true, "mode": "voluntary", "webhook_url": "https://example.com" } }
	}`), http.StatusOK, &acResp)

	// confirm new app config
	s.DoJSON("GET", "/api/v1/fleet/config", nil, http.StatusOK, &acResp)
	require.True(t, acResp.MDM.MacOSMigration.Enable)
	require.Equal(t, fleet.MacOSMigrationModeVoluntary, acResp.MDM.MacOSMigration.Mode)
	require.Equal(t, "https://example.com", acResp.MDM.MacOSMigration.WebhookURL)
}

func (s *integrationMDMTestSuite) TestBootstrapPackage() {
	t := s.T()

	read := func(name string) []byte {
		b, err := os.ReadFile(filepath.Join("testdata", "bootstrap-packages", name))
		require.NoError(t, err)
		return b
	}
	invalidPkg := read("invalid.tar.gz")
	unsignedPkg := read("unsigned.pkg")
	wrongTOCPkg := read("wrong-toc.pkg")
	signedPkg := read("signed.pkg")

	// empty bootstrap package
	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{}, http.StatusBadRequest, "package multipart field is required")
	// no name
	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Bytes: signedPkg}, http.StatusBadRequest, "package multipart field is required")
	// invalid
	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Bytes: invalidPkg, Name: "invalid.tar.gz"}, http.StatusBadRequest, "invalid file type")
	// invalid names
	for _, char := range file.InvalidMacOSChars {
		s.uploadBootstrapPackage(
			&fleet.MDMAppleBootstrapPackage{
				Bytes: signedPkg,
				Name:  fmt.Sprintf("invalid_%c_name.pkg", char),
			}, http.StatusBadRequest, "")
	}
	// unsigned
	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Bytes: unsignedPkg, Name: "pkg.pkg"}, http.StatusBadRequest, "file is not signed")
	// wrong TOC
	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Bytes: wrongTOCPkg, Name: "pkg.pkg"}, http.StatusBadRequest, "invalid package")
	// successfully upload a package
	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Bytes: signedPkg, Name: "pkg.pkg", TeamID: 0}, http.StatusOK, "")
	// check the activity log
	s.lastActivityMatches(
		fleet.ActivityTypeAddedBootstrapPackage{}.ActivityName(),
		`{"bootstrap_package_name": "pkg.pkg", "team_id": null, "team_name": null}`,
		0,
	)

	// get package metadata
	var metadataResp bootstrapPackageMetadataResponse
	s.DoJSON("GET", "/api/latest/fleet/bootstrap/0/metadata", nil, http.StatusOK, &metadataResp)
	require.Equal(t, metadataResp.MDMAppleBootstrapPackage.Name, "pkg.pkg")
	require.NotEmpty(t, metadataResp.MDMAppleBootstrapPackage.Sha256, "")
	require.NotEmpty(t, metadataResp.MDMAppleBootstrapPackage.Token)

	// download a package, wrong token
	var downloadResp downloadBootstrapPackageResponse
	s.DoJSON("GET", "/api/latest/fleet/bootstrap?token=bad", nil, http.StatusNotFound, &downloadResp)

	resp := s.DoRaw("GET", fmt.Sprintf("/api/latest/fleet/bootstrap?token=%s", metadataResp.MDMAppleBootstrapPackage.Token), nil, http.StatusOK)
	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.EqualValues(t, signedPkg, respBytes)

	// missing package
	metadataResp = bootstrapPackageMetadataResponse{}
	s.DoJSON("GET", "/api/latest/fleet/bootstrap/1/metadata", nil, http.StatusNotFound, &metadataResp)

	// delete package
	var deleteResp deleteBootstrapPackageResponse
	s.DoJSON("DELETE", "/api/latest/fleet/bootstrap/0", nil, http.StatusOK, &deleteResp)
	// check the activity log
	s.lastActivityMatches(
		fleet.ActivityTypeDeletedBootstrapPackage{}.ActivityName(),
		`{"bootstrap_package_name": "pkg.pkg", "team_id": null, "team_name": null}`,
		0,
	)

	metadataResp = bootstrapPackageMetadataResponse{}
	s.DoJSON("GET", "/api/latest/fleet/bootstrap/0/metadata", nil, http.StatusNotFound, &metadataResp)
	// trying to delete again is a bad request
	s.DoJSON("DELETE", "/api/latest/fleet/bootstrap/0", nil, http.StatusNotFound, &deleteResp)
}

func (s *integrationMDMTestSuite) TestBootstrapPackageStatus() {
	t := s.T()

	abmOrgName := "abm_org"
	s.enableABM(abmOrgName)

	pkg, err := os.ReadFile(filepath.Join("testdata", "bootstrap-packages", "signed.pkg"))
	require.NoError(t, err)

	// upload a bootstrap package for "no team"
	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Bytes: pkg, Name: "pkg.pkg", TeamID: 0}, http.StatusOK, "")

	// get package metadata
	var metadataResp bootstrapPackageMetadataResponse
	s.DoJSON("GET", "/api/latest/fleet/bootstrap/0/metadata", nil, http.StatusOK, &metadataResp)
	globalBootstrapPackage := metadataResp.MDMAppleBootstrapPackage

	// create a team and upload a bootstrap package for that team.
	teamName := t.Name() + "team1"
	team := &fleet.Team{
		Name:        teamName,
		Description: "desc team1",
	}
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &createTeamResp)
	require.NotZero(t, createTeamResp.Team.ID)
	team = createTeamResp.Team

	// upload a bootstrap package for the team
	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Bytes: pkg, Name: "pkg.pkg", TeamID: team.ID}, http.StatusOK, "")

	// get package metadata
	metadataResp = bootstrapPackageMetadataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/bootstrap/%d/metadata", team.ID), nil, http.StatusOK, &metadataResp)
	teamBootstrapPackage := metadataResp.MDMAppleBootstrapPackage

	type deviceWithResponse struct {
		bootstrapResponse string
		device            *mdmtest.TestAppleMDMClient
	}

	// Note: The responses specified here are not a 1:1 mapping of the possible responses specified
	// by Apple. Instead `enrollAndCheckBootstrapPackage` below uses them to simulate scenarios in
	// which a device may or may not send a response. For example, "Offline" means that no response
	// will be sent by the device, which should in turn be interpreted by Fleet as "Pending"). See
	// https://developer.apple.com/documentation/devicemanagement/installenterpriseapplicationresponse
	//
	// Below:
	// - Acknowledge means the device will enroll and acknowledge the request to install the bp
	// - Error means that the device will enroll and fail to install the bp
	// - Offline means that the device will enroll but won't acknowledge nor fail the bp request
	// - Pending means that the device won't enroll at all
	mdmEnrollInfo := mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}
	noTeamDevices := []deviceWithResponse{
		{"Acknowledge", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Acknowledge", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Acknowledge", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Error", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Offline", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Offline", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Pending", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Pending", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
	}

	teamDevices := []deviceWithResponse{
		{"Acknowledge", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Acknowledge", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Error", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Error", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Error", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Offline", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
		{"Pending", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, "MacBookPro16,1")},
	}

	expectedSerialsByTeamAndStatus := make(map[uint]map[fleet.MDMBootstrapPackageStatus][]string)
	expectedSerialsByTeamAndStatus[0] = map[fleet.MDMBootstrapPackageStatus][]string{
		fleet.MDMBootstrapPackageInstalled: {noTeamDevices[0].device.SerialNumber, noTeamDevices[1].device.SerialNumber, noTeamDevices[2].device.SerialNumber},
		fleet.MDMBootstrapPackageFailed:    {noTeamDevices[3].device.SerialNumber},
		fleet.MDMBootstrapPackagePending:   {noTeamDevices[4].device.SerialNumber, noTeamDevices[5].device.SerialNumber, noTeamDevices[6].device.SerialNumber, noTeamDevices[7].device.SerialNumber},
	}
	expectedSerialsByTeamAndStatus[team.ID] = map[fleet.MDMBootstrapPackageStatus][]string{
		fleet.MDMBootstrapPackageInstalled: {teamDevices[0].device.SerialNumber, teamDevices[1].device.SerialNumber},
		fleet.MDMBootstrapPackageFailed:    {teamDevices[2].device.SerialNumber, teamDevices[3].device.SerialNumber, teamDevices[4].device.SerialNumber},
		fleet.MDMBootstrapPackagePending:   {teamDevices[5].device.SerialNumber, teamDevices[6].device.SerialNumber},
	}

	// for good measure, add a couple of manually enrolled hosts
	createHostThenEnrollMDM(s.ds, s.server.URL, t)
	createHostThenEnrollMDM(s.ds, s.server.URL, t)

	// create a non-macOS host
	winHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		OsqueryHostID: ptr.String("non-macos-host"),
		NodeKey:       ptr.String("non-macos-host"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.non.macos", t.Name()),
		Platform:      "windows",
		MDM:           fleet.MDMHostData{},
	})
	require.NoError(t, err)

	err = s.ds.SetOrUpdateMDMData(context.Background(), winHost.ID, false, true, s.server.URL+apple_mdm.MDMPath, true, fleet.WellKnownMDMFleet, "")
	require.NoError(t, err)

	// create a host that's not enrolled into MDM
	_, err = s.ds.NewHost(context.Background(), &fleet.Host{
		OsqueryHostID: ptr.String("not-mdm-enrolled"),
		NodeKey:       ptr.String("not-mdm-enrolled"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.not.enrolled", t.Name()),
		Platform:      "darwin",
	})
	require.NoError(t, err)

	mockRespDevices := noTeamDevices
	s.mockDEPResponse(abmOrgName, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			err := encoder.Encode(map[string]string{"auth_session_token": "xyz"})
			require.NoError(t, err)
		case "/profile":
			err := encoder.Encode(godep.ProfileResponse{ProfileUUID: "abc"})
			require.NoError(t, err)
		case "/server/devices":
			err := encoder.Encode(godep.DeviceResponse{})
			require.NoError(t, err)
		case "/devices/sync":
			depResp := []godep.Device{}
			for _, gd := range mockRespDevices {
				depResp = append(depResp, godep.Device{SerialNumber: gd.device.SerialNumber})
			}
			err := encoder.Encode(godep.DeviceResponse{Devices: depResp})
			require.NoError(t, err)
		case "/profile/devices":
			_, _ = w.Write([]byte(`{}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))

	// trigger a dep sync
	s.runDEPSchedule()

	var summaryResp getMDMAppleBootstrapPackageSummaryResponse
	s.DoJSON("GET", "/api/latest/fleet/bootstrap/summary", nil, http.StatusOK, &summaryResp)
	require.Equal(t, fleet.MDMAppleBootstrapPackageSummary{Pending: uint(len(noTeamDevices))}, summaryResp.MDMAppleBootstrapPackageSummary)

	var lhr listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &lhr, "team_id", "0", "bootstrap_package", "pending")
	require.Len(t, lhr.Hosts, len(noTeamDevices))

	// set the default bm assignment to `team`
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": {
			"apple_business_manager": [{
			  "organization_name": %q,
			  "macos_team": %q,
			  "ios_team": %q,
			  "ipados_team": %q
			}]
		}
	}`, abmOrgName, team.Name, team.Name, team.Name)), http.StatusOK, &acResp)
	t.Cleanup(func() {
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"apple_business_manager": []
			}
		}`), http.StatusOK, &acResp)
	})

	// trigger a dep sync
	mockRespDevices = teamDevices
	s.runDEPSchedule()

	summaryResp = getMDMAppleBootstrapPackageSummaryResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/bootstrap/summary?team_id=%d", team.ID), nil, http.StatusOK, &summaryResp)
	require.Equal(t, fleet.MDMAppleBootstrapPackageSummary{Pending: uint(len(teamDevices))}, summaryResp.MDMAppleBootstrapPackageSummary)

	mockErrorChain := []mdm.ErrorChain{
		{ErrorCode: 12021, ErrorDomain: "MCMDMErrorDomain", LocalizedDescription: "Unknown command", USEnglishDescription: "Unknown command"},
	}

	// devices send their responses
	enrollAndCheckBootstrapPackage := func(d *deviceWithResponse, bp *fleet.MDMAppleBootstrapPackage) {
		err := d.device.Enroll() // queues DEP post-enrollment worker job
		require.NoError(t, err)

		// process worker jobs
		s.runWorker()

		cmd, err := d.device.Idle()
		require.NoError(t, err)
		for cmd != nil {
			var fullCmd micromdm.CommandPayload
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))

			// if the command is to install the bootstrap package
			if manifest := fullCmd.Command.InstallEnterpriseApplication.Manifest; manifest != nil {
				require.Equal(t, "InstallEnterpriseApplication", cmd.Command.RequestType)
				require.NotNil(t, manifest)
				require.Equal(t, "software-package", manifest.ManifestItems[0].Assets[0].Kind)
				wantURL, err := bp.URL(s.server.URL)
				require.NoError(t, err)
				require.Equal(t, wantURL, manifest.ManifestItems[0].Assets[0].URL)

				// respond to the command accordingly
				switch d.bootstrapResponse {
				case "Acknowledge":
					cmd, err = d.device.Acknowledge(cmd.CommandUUID)
					require.NoError(t, err)
					continue
				case "Error":
					cmd, err = d.device.Err(cmd.CommandUUID, mockErrorChain)
					require.NoError(t, err)
					continue
				case "Offline":
					// host is offline, can't process any more commands
					cmd = nil
					continue
				}
			}
			cmd, err = d.device.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}
	}

	for _, d := range noTeamDevices {
		dd := d
		if dd.bootstrapResponse != "Pending" {
			enrollAndCheckBootstrapPackage(&dd, globalBootstrapPackage)
		}
	}

	for _, d := range teamDevices {
		dd := d
		if dd.bootstrapResponse != "Pending" {
			enrollAndCheckBootstrapPackage(&dd, teamBootstrapPackage)
		}
	}

	checkHostDetails := func(t *testing.T, hostID uint, hostUUID string, expectedStatus fleet.MDMBootstrapPackageStatus) {
		var hostResp getHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostID), nil, http.StatusOK, &hostResp)
		require.NotNil(t, hostResp.Host)
		require.NotNil(t, hostResp.Host.MDM.MacOSSetup)
		require.Equal(t, hostResp.Host.MDM.MacOSSetup.BootstrapPackageName, "pkg.pkg")
		require.Equal(t, hostResp.Host.MDM.MacOSSetup.BootstrapPackageStatus, expectedStatus)
		if expectedStatus == fleet.MDMBootstrapPackageFailed {
			require.Equal(t, hostResp.Host.MDM.MacOSSetup.Detail, apple_mdm.FmtErrorChain(mockErrorChain))
		} else {
			require.Empty(t, hostResp.Host.MDM.MacOSSetup.Detail)
		}
		require.Nil(t, hostResp.Host.MDM.MacOSSetup.Result)

		var hostByIdentifierResp getHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/identifier/%s", hostUUID), nil, http.StatusOK, &hostByIdentifierResp)
		require.NotNil(t, hostByIdentifierResp.Host)
		require.NotNil(t, hostByIdentifierResp.Host.MDM.MacOSSetup)
		require.Equal(t, hostByIdentifierResp.Host.MDM.MacOSSetup.BootstrapPackageStatus, expectedStatus)
		if expectedStatus == fleet.MDMBootstrapPackageFailed {
			require.Equal(t, hostResp.Host.MDM.MacOSSetup.Detail, apple_mdm.FmtErrorChain(mockErrorChain))
		} else {
			require.Empty(t, hostResp.Host.MDM.MacOSSetup.Detail)
		}
		require.Nil(t, hostResp.Host.MDM.MacOSSetup.Result)
	}

	checkHostAPIs := func(t *testing.T, status fleet.MDMBootstrapPackageStatus, teamID *uint) {
		var expectedSerials []string
		if teamID == nil {
			expectedSerials = expectedSerialsByTeamAndStatus[0][status]
		} else {
			expectedSerials = expectedSerialsByTeamAndStatus[*teamID][status]
		}

		listHostsPath := fmt.Sprintf("/api/latest/fleet/hosts?bootstrap_package=%s", status)
		if teamID != nil {
			listHostsPath += fmt.Sprintf("&team_id=%d", *teamID)
		}
		var listHostsResp listHostsResponse
		s.DoJSON("GET", listHostsPath, nil, http.StatusOK, &listHostsResp)
		require.NotNil(t, listHostsResp.Hosts)
		require.Len(t, listHostsResp.Hosts, len(expectedSerials))

		gotHostsBySerial := make(map[string]fleet.HostResponse)
		for _, h := range listHostsResp.Hosts {
			gotHostsBySerial[h.HardwareSerial] = h
		}
		require.Len(t, gotHostsBySerial, len(expectedSerials))

		for _, serial := range expectedSerials {
			require.Contains(t, gotHostsBySerial, serial)
			h := gotHostsBySerial[serial]

			// pending hosts don't have an UUID yet.
			if h.UUID != "" {
				checkHostDetails(t, h.ID, h.UUID, status)
			}
		}

		countPath := fmt.Sprintf("/api/latest/fleet/hosts/count?bootstrap_package=%s", status)
		if teamID != nil {
			countPath += fmt.Sprintf("&team_id=%d", *teamID)
		}
		var countResp countHostsResponse
		s.DoJSON("GET", countPath, nil, http.StatusOK, &countResp)
		require.Equal(t, countResp.Count, len(expectedSerials))
	}

	// check summary no team hosts
	summaryResp = getMDMAppleBootstrapPackageSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/bootstrap/summary", nil, http.StatusOK, &summaryResp)
	require.Equal(t, fleet.MDMAppleBootstrapPackageSummary{
		Installed: uint(3),
		Pending:   uint(4),
		Failed:    uint(1),
	}, summaryResp.MDMAppleBootstrapPackageSummary)

	checkHostAPIs(t, fleet.MDMBootstrapPackageInstalled, nil)
	checkHostAPIs(t, fleet.MDMBootstrapPackagePending, nil)
	checkHostAPIs(t, fleet.MDMBootstrapPackageFailed, nil)

	// check team summary
	summaryResp = getMDMAppleBootstrapPackageSummaryResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/bootstrap/summary?team_id=%d", team.ID), nil, http.StatusOK, &summaryResp)
	require.Equal(t, fleet.MDMAppleBootstrapPackageSummary{
		Installed: uint(2),
		Pending:   uint(2),
		Failed:    uint(3),
	}, summaryResp.MDMAppleBootstrapPackageSummary)

	checkHostAPIs(t, fleet.MDMBootstrapPackageInstalled, &team.ID)
	checkHostAPIs(t, fleet.MDMBootstrapPackagePending, &team.ID)
	checkHostAPIs(t, fleet.MDMBootstrapPackageFailed, &team.ID)
}

func (s *integrationMDMTestSuite) TestEULA() {
	t := s.T()
	pdfBytes := []byte("%PDF-1.pdf-contents")
	pdfName := "eula.pdf"

	// trying to get metadata about an EULA that hasn't been uploaded yet is an error
	metadataResp := getMDMEULAMetadataResponse{}
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/eula/metadata", nil, http.StatusNotFound, &metadataResp)

	// trying to upload a file that is not a PDF fails
	s.uploadEULA(&fleet.MDMEULA{Bytes: []byte("should-fail"), Name: "should-fail.pdf"}, http.StatusBadRequest, "invalid file type")
	// trying to upload an empty file fails
	s.uploadEULA(&fleet.MDMEULA{Bytes: []byte{}, Name: "should-fail.pdf"}, http.StatusBadRequest, "invalid file type")

	// admin is able to upload a new EULA
	s.uploadEULA(&fleet.MDMEULA{Bytes: pdfBytes, Name: pdfName}, http.StatusOK, "")

	// get EULA metadata
	metadataResp = getMDMEULAMetadataResponse{}
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/eula/metadata", nil, http.StatusOK, &metadataResp)
	require.NotEmpty(t, metadataResp.MDMEULA.Token)
	require.NotEmpty(t, metadataResp.MDMEULA.CreatedAt)
	require.Equal(t, pdfName, metadataResp.MDMEULA.Name)
	eulaToken := metadataResp.Token

	// download EULA
	resp := s.DoRaw("GET", fmt.Sprintf("/api/latest/fleet/setup_experience/eula/%s", eulaToken), nil, http.StatusOK)
	require.EqualValues(t, len(pdfBytes), resp.ContentLength)
	require.Equal(t, "application/pdf", resp.Header.Get("content-type"))
	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.EqualValues(t, pdfBytes, respBytes)

	// try to download EULA with a bad token
	var downloadResp downloadBootstrapPackageResponse
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/eula/bad-token", nil, http.StatusNotFound, &downloadResp)

	// trying to upload any EULA without deleting the previous one first results in an error
	s.uploadEULA(&fleet.MDMEULA{Bytes: pdfBytes, Name: "should-fail.pdf"}, http.StatusConflict, "")

	// delete EULA
	var deleteResp deleteMDMEULAResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/setup_experience/eula/%s", eulaToken), nil, http.StatusOK, &deleteResp)
	metadataResp = getMDMEULAMetadataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/setup_experience/eula/%s", eulaToken), nil, http.StatusNotFound, &metadataResp)
	// trying to delete again is a bad request
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/setup_experience/eula/%s", eulaToken), nil, http.StatusNotFound, &deleteResp)
}

func (s *integrationMDMTestSuite) TestMigrateMDMDeviceWebhook() {
	t := s.T()

	h := createHostAndDeviceToken(t, s.ds, "good-token")

	var webhookCalled bool
	webhookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webhookCalled = true
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/test_mdm_migration":
			var payload fleet.MigrateMDMDeviceWebhookPayload
			b, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			err = json.Unmarshal(b, &payload)
			require.NoError(t, err)

			require.Equal(t, h.ID, payload.Host.ID)
			require.Equal(t, h.UUID, payload.Host.UUID)
			require.Equal(t, h.HardwareSerial, payload.Host.HardwareSerial)

		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
		}
	}))
	defer webhookSrv.Close()

	// patch app config with webhook url
	acResp := fleet.AppConfig{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": {
			"macos_migration": {
				"enable": true,
				"mode": "voluntary",
				"webhook_url": "%s/test_mdm_migration"
			}
		}
	}`, webhookSrv.URL)), http.StatusOK, &acResp)
	require.True(t, acResp.MDM.MacOSMigration.Enable)

	// expect errors when host is not eligible for migration
	isServer, enrolled, installedFromDEP := true, true, true
	mdmName := "ExampleMDM"
	mdmURL := "https://mdm.example.com"

	// host is a server so migration is not allowed
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), h.ID, isServer, enrolled, mdmURL, installedFromDEP, mdmName, ""))
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusBadRequest)
	require.False(t, webhookCalled)

	// host is not enrolled to MDM so migration is not allowed
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), h.ID, !isServer, !enrolled, mdmURL, installedFromDEP, mdmName, ""))
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusBadRequest)
	require.False(t, webhookCalled)

	// host is already enrolled to Fleet MDM so migration is not allowed
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), h.ID, !isServer, enrolled, mdmURL, installedFromDEP, fleet.WellKnownMDMFleet, ""))
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusBadRequest)
	require.False(t, webhookCalled)

	// up to this point, the refetch critical queries timestamp has not been set
	// on the host.
	h, err := s.ds.Host(context.Background(), h.ID)
	require.NoError(t, err)
	require.Nil(t, h.RefetchCriticalQueriesUntil)

	// host is enrolled to a third-party MDM but hasn't been assigned in
	// ABM yet, so migration is not allowed
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), h.ID, !isServer, enrolled, mdmURL, installedFromDEP, mdmName, ""))
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusBadRequest)
	require.False(t, webhookCalled)

	s.enableABM(t.Name())

	// simulate that the device is assigned to Fleet in ABM
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
		case "/profile":
			encoder := json.NewEncoder(w)
			err := encoder.Encode(godep.ProfileResponse{ProfileUUID: "abc"})
			require.NoError(t, err)
		case "/server/devices", "/devices/sync":
			encoder := json.NewEncoder(w)
			err := encoder.Encode(godep.DeviceResponse{
				Devices: []godep.Device{
					{
						SerialNumber: h.HardwareSerial,
						Model:        "Mac Mini",
						OS:           "osx",
						OpType:       "added",
					},
				},
			})
			require.NoError(t, err)

		case "/profile/devices":
			b, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var prof profileAssignmentReq
			require.NoError(t, json.Unmarshal(b, &prof))
			var resp godep.ProfileResponse
			resp.ProfileUUID = prof.ProfileUUID
			resp.Devices = map[string]string{
				prof.Devices[0]: string(fleet.DEPAssignProfileResponseSuccess),
			}
			encoder := json.NewEncoder(w)
			err = encoder.Encode(resp)
			require.NoError(t, err)
		}
	}))
	s.runDEPSchedule()

	// hosts meets all requirements, webhook is run
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusNoContent)
	require.True(t, webhookCalled)
	webhookCalled = false

	// the refetch critical queries timestamp has been set in the future
	h, err = s.ds.Host(context.Background(), h.ID)
	require.NoError(t, err)
	require.NotNil(t, h.RefetchCriticalQueriesUntil)
	require.True(t, h.RefetchCriticalQueriesUntil.After(time.Now()))

	// calling again works but does not trigger the webhook, as it was called recently
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusNoContent)
	require.False(t, webhookCalled)

	// setting the refetch critical queries timestamp in the past triggers the webhook again
	h.RefetchCriticalQueriesUntil = ptr.Time(time.Now().Add(-1 * time.Minute))
	err = s.ds.UpdateHost(context.Background(), h)
	require.NoError(t, err)

	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusNoContent)
	require.True(t, webhookCalled)
	webhookCalled = false

	// host is manually enrolled, which is allowed
	h.RefetchCriticalQueriesUntil = ptr.Time(time.Now().Add(-1 * time.Minute))
	err = s.ds.UpdateHost(context.Background(), h)
	require.NoError(t, err)

	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), h.ID, !isServer, enrolled, mdmURL, !installedFromDEP, mdmName, ""))
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusNoContent)
	require.True(t, webhookCalled)
	webhookCalled = false

	// the refetch critical queries timestamp has been updated to the future
	h, err = s.ds.Host(context.Background(), h.ID)
	require.NoError(t, err)
	require.NotNil(t, h.RefetchCriticalQueriesUntil)
	require.True(t, h.RefetchCriticalQueriesUntil.After(time.Now()))

	require.NoError(t, s.ds.UpdateHostRefetchCriticalQueriesUntil(context.Background(), h.ID, nil))

	// bad token
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "bad-token"), nil, http.StatusUnauthorized)
	require.False(t, webhookCalled)

	// disable macos migration
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"macos_migration": {
				"enable": false,
				"mode": "voluntary",
				"webhook_url": ""
		      }
		}
	}`), http.StatusOK, &acResp)
	require.False(t, acResp.MDM.MacOSMigration.Enable)

	// expect error if macos migration is not configured
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusBadRequest)
	require.False(t, webhookCalled)
}

func (s *integrationMDMTestSuite) TestMigrateMDMDeviceWebhookErrors() {
	t := s.T()

	h := createHostAndDeviceToken(t, s.ds, "good-token")

	var webhookCalled bool
	webhookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webhookCalled = true
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer webhookSrv.Close()

	// patch app config with webhook url
	acResp := fleet.AppConfig{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": {
			"macos_migration": {
				"enable": true,
				"mode": "voluntary",
				"webhook_url": "%s/test_mdm_migration"
			}
		}
	}`, webhookSrv.URL)), http.StatusOK, &acResp)
	require.True(t, acResp.MDM.MacOSMigration.Enable)

	isServer, enrolled, installedFromDEP := true, true, true
	mdmName := "ExampleMDM"
	mdmURL := "https://mdm.example.com"

	// host is enrolled to a third-party MDM but hasn't been assigned in
	// ABM yet, so migration is not allowed
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), h.ID, !isServer, enrolled, mdmURL, installedFromDEP, mdmName, ""))
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusBadRequest)
	require.False(t, webhookCalled)

	s.enableABM(t.Name())
	// simulate that the device is assigned to Fleet in ABM
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
		case "/profile":
			encoder := json.NewEncoder(w)
			err := encoder.Encode(godep.ProfileResponse{ProfileUUID: "abc"})
			require.NoError(t, err)
		case "/server/devices", "/devices/sync":
			encoder := json.NewEncoder(w)
			err := encoder.Encode(godep.DeviceResponse{
				Devices: []godep.Device{
					{
						SerialNumber: h.HardwareSerial,
						Model:        "Mac Mini",
						OS:           "osx",
						OpType:       "added",
					},
				},
			})
			require.NoError(t, err)
		case "/profile/devices":
			b, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var prof profileAssignmentReq
			require.NoError(t, json.Unmarshal(b, &prof))
			var resp godep.ProfileResponse
			resp.ProfileUUID = prof.ProfileUUID
			resp.Devices = map[string]string{
				prof.Devices[0]: string(fleet.DEPAssignProfileResponseSuccess),
			}
			encoder := json.NewEncoder(w)
			err = encoder.Encode(resp)
			require.NoError(t, err)
		}
	}))
	s.runDEPSchedule()

	// hosts meets all requirements, webhook is run but returns an error, server should respond with
	// the same status code
	require.False(t, webhookCalled)
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusBadRequest)
	require.True(t, webhookCalled)
}

func (s *integrationMDMTestSuite) TestMDMMacOSSetup() {
	t := s.T()

	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			err := encoder.Encode(map[string]string{"auth_session_token": "xyz"})
			require.NoError(t, err)
		case "/profile":
			err := encoder.Encode(godep.ProfileResponse{ProfileUUID: "abc"})
			require.NoError(t, err)
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))

	// setup test data
	var acResp appConfigResponse
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"end_user_authentication": {
				"entity_id": "https://localhost:8080",
				"issuer_uri": "http://localhost:8080/simplesaml/saml2/idp/SSOService.php",
				"idp_name": "SimpleSAML",
				"metadata_url": "http://localhost:9080/simplesaml/saml2/idp/metadata.php"
		      }
		}
	}`), http.StatusOK, &acResp)
	require.NotEmpty(t, acResp.MDM.EndUserAuthentication)

	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	endUserAuthCases := []struct {
		raw      string
		expected bool
	}{
		{
			raw:      `"mdm": {}`,
			expected: false,
		},
		{
			raw: `"mdm": {
				"macos_setup": {}
			}`,
			expected: false,
		},
		{
			raw: `"mdm": {
				"macos_setup": {
					"enable_end_user_authentication": true
				}
			}`,
			expected: true,
		},
		{
			raw: `"mdm": {
				"macos_setup": {
					"enable_end_user_authentication": false
				}
			}`,
			expected: false,
		},
	}

	writeTmpJSON := func(t *testing.T, v any) string {
		tmpFile, err := os.CreateTemp(t.TempDir(), "*.json")
		require.NoError(t, err)
		err = json.NewEncoder(tmpFile).Encode(v)
		require.NoError(t, err)
		return tmpFile.Name()
	}

	mustReadFile := func(t *testing.T, path string) string {
		b, err := os.ReadFile(path)
		require.NoError(t, err)
		return string(b)
	}

	asstOk := writeTmpJSON(t, map[string]any{"ok": true})
	asstURL := writeTmpJSON(t, map[string]any{"url": "https://example.com"})
	asstAwait := writeTmpJSON(t, map[string]any{"await_device_configured": true})
	asstsByName := map[string]string{
		asstOk:    mustReadFile(t, asstOk),
		asstURL:   mustReadFile(t, asstURL),
		asstAwait: mustReadFile(t, asstAwait),
	}

	enableReleaseDeviceCases := []struct {
		enableRelease     *bool
		setupAssistant    string
		expectedRelease   bool
		expectedAssistant string
		expectedStatus    int
	}{
		{
			enableRelease:     nil,
			setupAssistant:    "",
			expectedRelease:   false,
			expectedAssistant: "",
			expectedStatus:    http.StatusOK,
		},
		{
			enableRelease:     ptr.Bool(true),
			setupAssistant:    "",
			expectedRelease:   true,
			expectedAssistant: "",
			expectedStatus:    http.StatusOK,
		},
		{
			enableRelease:     ptr.Bool(false),
			setupAssistant:    "",
			expectedRelease:   false,
			expectedAssistant: "",
			expectedStatus:    http.StatusOK,
		},
		{
			enableRelease:     ptr.Bool(false),
			setupAssistant:    asstURL,
			expectedRelease:   false,
			expectedAssistant: "",
			expectedStatus:    http.StatusUnprocessableEntity,
		},
		{
			enableRelease:     ptr.Bool(true),
			setupAssistant:    asstAwait,
			expectedRelease:   false,
			expectedAssistant: "",
			expectedStatus:    http.StatusUnprocessableEntity,
		},
	}

	t.Run("UpdateAppConfig", func(t *testing.T) {
		acResp := appConfigResponse{}
		path := "/api/latest/fleet/config"
		fmtJSON := func(s string) json.RawMessage {
			return json.RawMessage(fmt.Sprintf(`{
				%s
			}`, s))
		}

		// get the initial appconfig; enable end user authentication and release
		// device default is false
		s.DoJSON("GET", path, nil, http.StatusOK, &acResp)
		require.False(t, acResp.MDM.MacOSSetup.EnableEndUserAuthentication)
		require.False(t, acResp.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)

		for i, c := range endUserAuthCases {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				acResp = appConfigResponse{}
				s.DoJSON("PATCH", path, fmtJSON(c.raw), http.StatusOK, &acResp)
				require.Equal(t, c.expected, acResp.MDM.MacOSSetup.EnableEndUserAuthentication)

				acResp = appConfigResponse{}
				s.DoJSON("GET", path, nil, http.StatusOK, &acResp)
				require.Equal(t, c.expected, acResp.MDM.MacOSSetup.EnableEndUserAuthentication)
			})
		}

		for i, c := range enableReleaseDeviceCases {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				macSetup := map[string]any{}
				if c.enableRelease != nil {
					macSetup["enable_release_device_manually"] = *c.enableRelease
				}
				if c.setupAssistant != "" {
					macSetup["macos_setup_assistant"] = c.setupAssistant
				}

				uploadSucceeded := true
				if c.setupAssistant != "" {
					s.Do("POST", "/api/v1/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
						Name:              c.setupAssistant,
						EnrollmentProfile: json.RawMessage(asstsByName[c.setupAssistant]),
					}, c.expectedStatus)
					if c.expectedStatus >= 300 {
						uploadSucceeded = false
					}
				}

				if uploadSucceeded {
					acResp = appConfigResponse{}
					s.DoJSON("PATCH", path,
						json.RawMessage(jsonMustMarshal(t, map[string]any{"mdm": map[string]any{"macos_setup": macSetup}})),
						c.expectedStatus, &acResp)
					require.Equal(t, c.expectedRelease, acResp.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)
					require.Equal(t, c.expectedAssistant, acResp.MDM.MacOSSetup.MacOSSetupAssistant.Value)
				}

				acResp = appConfigResponse{}
				s.DoJSON("GET", path, nil, http.StatusOK, &acResp)
				require.Equal(t, c.expectedRelease, acResp.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)
				require.Equal(t, c.expectedAssistant, acResp.MDM.MacOSSetup.MacOSSetupAssistant.Value)
			})
		}
	})

	t.Run("UpdateTeamConfig", func(t *testing.T) {
		path := fmt.Sprintf("/api/latest/fleet/teams/%d", tm.ID)
		fmtJSON := `{
			"name": %q,
			%s
		}`

		// get the initial team config; enable end user authentication and release
		// device default is false
		teamResp := teamResponse{}
		s.DoJSON("GET", path, nil, http.StatusOK, &teamResp)
		require.False(t, teamResp.Team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
		require.False(t, teamResp.Team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)

		for i, c := range endUserAuthCases {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				teamResp = teamResponse{}
				s.DoJSON("PATCH", path, json.RawMessage(fmt.Sprintf(fmtJSON, tm.Name, c.raw)), http.StatusOK, &teamResp)
				require.Equal(t, c.expected, teamResp.Team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)

				teamResp = teamResponse{}
				s.DoJSON("GET", path, nil, http.StatusOK, &teamResp)
				require.Equal(t, c.expected, teamResp.Team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
			})
		}

		for i, c := range enableReleaseDeviceCases {
			expectedPatchStatus := c.expectedStatus
			if expectedPatchStatus == http.StatusOK {
				expectedPatchStatus = http.StatusNoContent
			}

			t.Run(strconv.Itoa(i), func(t *testing.T) {
				if c.setupAssistant != "" {
					s.Do("POST", "/api/v1/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
						TeamID:            &tm.ID,
						Name:              c.setupAssistant,
						EnrollmentProfile: json.RawMessage(asstsByName[c.setupAssistant]),
					}, c.expectedStatus)
					uploadSucceeded := c.expectedStatus < 300

					if uploadSucceeded {
						// use the apply team specs to set both the setup assistant and the
						// enable release at once
						macSetup := fleet.MacOSSetup{
							MacOSSetupAssistant: optjson.SetString(c.setupAssistant),
						}
						if c.enableRelease != nil {
							macSetup.EnableReleaseDeviceManually = optjson.SetBool(*c.enableRelease)
						}
						teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
							Name: tm.Name,
							MDM:  fleet.TeamSpecMDM{MacOSSetup: macSetup},
						}}}
						s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
					}
				} else {
					// no setup assistant, use the PATCH /setup_experience endpoint
					payload := map[string]any{
						"team_id": tm.ID,
					}
					if c.enableRelease != nil {
						payload["enable_release_device_manually"] = *c.enableRelease
					}
					s.Do("PATCH", "/api/latest/fleet/setup_experience", json.RawMessage(jsonMustMarshal(t, payload)), expectedPatchStatus)
				}

				teamResp = teamResponse{}
				s.DoJSON("GET", path, nil, http.StatusOK, &teamResp)
				require.Equal(t, c.expectedRelease, teamResp.Team.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)
				require.Equal(t, c.expectedAssistant, teamResp.Team.Config.MDM.MacOSSetup.MacOSSetupAssistant.Value)
			})
		}
	})

	t.Run("TestMDMAppleSetupEndpoint", func(t *testing.T) {
		t.Run("TestNoTeam", func(t *testing.T) {
			var acResp appConfigResponse
			s.Do("PATCH", "/api/latest/fleet/setup_experience",
				fleet.MDMAppleSetupPayload{TeamID: ptr.Uint(0), EnableEndUserAuthentication: ptr.Bool(true)}, http.StatusNoContent)
			acResp = appConfigResponse{}
			s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
			require.True(t, acResp.MDM.MacOSSetup.EnableEndUserAuthentication)
			lastActivityID := s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledMacosSetupEndUserAuth{}.ActivityName(),
				`{"team_id": null, "team_name": null}`, 0)

			s.Do("PATCH", "/api/latest/fleet/setup_experience",
				fleet.MDMAppleSetupPayload{TeamID: ptr.Uint(0), EnableEndUserAuthentication: ptr.Bool(true)}, http.StatusNoContent)
			acResp = appConfigResponse{}
			s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
			require.True(t, acResp.MDM.MacOSSetup.EnableEndUserAuthentication)
			s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledMacosSetupEndUserAuth{}.ActivityName(),
				``, lastActivityID) // no new activity

			s.Do("PATCH", "/api/latest/fleet/setup_experience",
				fleet.MDMAppleSetupPayload{TeamID: ptr.Uint(0), EnableEndUserAuthentication: ptr.Bool(false)}, http.StatusNoContent)
			acResp = appConfigResponse{}
			s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
			require.False(t, acResp.MDM.MacOSSetup.EnableEndUserAuthentication)
			require.Greater(t, s.lastActivityOfTypeMatches(fleet.ActivityTypeDisabledMacosSetupEndUserAuth{}.ActivityName(),
				`{"team_id": null, "team_name": null}`, 0), lastActivityID)
		})

		t.Run("TestTeam", func(t *testing.T) {
			tmConfigPath := fmt.Sprintf("/api/latest/fleet/teams/%d", tm.ID)
			expectedActivityDetail := fmt.Sprintf(`{"team_id": %d, "team_name": %q}`, tm.ID, tm.Name)
			var tmResp teamResponse
			s.Do("PATCH", "/api/latest/fleet/setup_experience",
				fleet.MDMAppleSetupPayload{TeamID: &tm.ID, EnableEndUserAuthentication: ptr.Bool(true)}, http.StatusNoContent)
			tmResp = teamResponse{}
			s.DoJSON("GET", tmConfigPath, nil, http.StatusOK, &tmResp)
			require.True(t, tmResp.Team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
			lastActivityID := s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledMacosSetupEndUserAuth{}.ActivityName(),
				expectedActivityDetail, 0)

			s.Do("PATCH", "/api/latest/fleet/setup_experience",
				fleet.MDMAppleSetupPayload{TeamID: &tm.ID, EnableEndUserAuthentication: ptr.Bool(true)}, http.StatusNoContent)
			tmResp = teamResponse{}
			s.DoJSON("GET", tmConfigPath, nil, http.StatusOK, &tmResp)
			require.True(t, tmResp.Team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
			s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledMacosSetupEndUserAuth{}.ActivityName(),
				``, lastActivityID) // no new activity

			s.Do("PATCH", "/api/latest/fleet/setup_experience",
				fleet.MDMAppleSetupPayload{TeamID: &tm.ID, EnableEndUserAuthentication: ptr.Bool(false)}, http.StatusNoContent)
			tmResp = teamResponse{}
			s.DoJSON("GET", tmConfigPath, nil, http.StatusOK, &tmResp)
			require.False(t, tmResp.Team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
			require.Greater(t, s.lastActivityOfTypeMatches(fleet.ActivityTypeDisabledMacosSetupEndUserAuth{}.ActivityName(),
				expectedActivityDetail, 0), lastActivityID)
		})
	})

	t.Run("ValidateEnableEndUserAuthentication", func(t *testing.T) {
		// ensure the test is setup correctly
		var acResp appConfigResponse
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"end_user_authentication": {
					"entity_id": "https://localhost:8080",
					"issuer_uri": "http://localhost:8080/simplesaml/saml2/idp/SSOService.php",
					"idp_name": "SimpleSAML",
					"metadata_url": "http://localhost:9080/simplesaml/saml2/idp/metadata.php"
				},
				"macos_setup": {
					"enable_end_user_authentication": true
				}
			}
		}`), http.StatusOK, &acResp)
		require.NotEmpty(t, acResp.MDM.EndUserAuthentication)

		// ok to disable end user authentication without a configured IdP
		acResp = appConfigResponse{}
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"end_user_authentication": {
					"entity_id": "",
					"issuer_uri": "",
					"idp_name": "",
					"metadata_url": ""
				},
				"macos_setup": {
					"enable_end_user_authentication": false
				}
			}
		}`), http.StatusOK, &acResp)
		require.Equal(t, acResp.MDM.MacOSSetup.EnableEndUserAuthentication, false)
		require.True(t, acResp.MDM.EndUserAuthentication.IsEmpty())

		// can't enable end user authentication without a configured IdP
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"end_user_authentication": {
					"entity_id": "",
					"issuer_uri": "",
					"idp_name": "",
					"metadata_url": ""
				},
				"macos_setup": {
					"enable_end_user_authentication": true
				}
			}
		}`), http.StatusUnprocessableEntity, &acResp)

		// can't use setup endpoint to enable end user authentication on no team without a configured IdP
		s.Do("PATCH", "/api/latest/fleet/setup_experience",
			fleet.MDMAppleSetupPayload{TeamID: ptr.Uint(0), EnableEndUserAuthentication: ptr.Bool(true)}, http.StatusUnprocessableEntity)

		// can't enable end user authentication on team config without a configured IdP already on app config
		var teamResp teamResponse
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm.ID), json.RawMessage(fmt.Sprintf(`{
			"name": %q,
			"mdm": {
				"macos_setup": {
					"enable_end_user_authentication": true
				}
			}
		}`, tm.Name)), http.StatusUnprocessableEntity, &teamResp)

		// can't use setup endpoint to enable end user authentication on team without a configured IdP
		s.Do("PATCH", "/api/latest/fleet/setup_experience",
			fleet.MDMAppleSetupPayload{TeamID: &tm.ID, EnableEndUserAuthentication: ptr.Bool(true)}, http.StatusUnprocessableEntity)

		// ensure IdP is empty for the rest of the tests
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"end_user_authentication": {
					"entity_id": "",
					"issuer_uri": "",
					"idp_name": "",
					"metadata_url": ""
				}
			}
		}`), http.StatusOK, &acResp)
		require.Empty(t, acResp.MDM.EndUserAuthentication)
	})
}

func (s *integrationMDMTestSuite) TestMacosSetupAssistant() {
	ctx := context.Background()
	t := s.T()

	const defaultProf = `{
	"profile_name": "%s",
	"allow_pairing": true,
	"is_mdm_removable": true,
	"org_magic": "1",
	"language": "en",
	"region": "US",
	"skip_setup_items": [
		"Accessibility",
		"Appearance",
		"AppleID",
		"AppStore",
		"Biometric",
		"Diagnostics",
		"FileVault",
		"iCloudDiagnostics",
		"iCloudStorage",
		"Location",
		"Payment",
		"Privacy",
		"Restore",
		"ScreenTime",
		"Siri",
		"TermsOfAddress",
		"TOS",
		"UnlockWithWatch"
	]
}
`

	// Associate the token with the team
	s.enableABM(t.Name())
	// start a server that will mock the Apple DEP API
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "session123"}`))
		case "/account":
			_, _ = w.Write([]byte(fmt.Sprintf(`{"admin_id": "admin123", "org_name": "%s"}`, "foo")))
		case "/profile":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var prof godep.Profile
			require.NoError(t, json.Unmarshal(body, &prof))
			switch {
			case len(prof.ProfileName) > 125:
				w.WriteHeader(http.StatusBadRequest)
				require.NoError(t, encoder.Encode(map[string]any{"error": "CONFIG_NAME_INVALID"}))
			case prof.ProfileName == "":
				w.WriteHeader(http.StatusBadRequest)
				require.NoError(t, encoder.Encode(map[string]any{"error": "CONFIG_NAME_REQUIRED"}))
			case len(prof.ConfigurationWebURL) > 125:
				w.WriteHeader(http.StatusBadRequest)
				require.NoError(t, encoder.Encode(map[string]any{"error": "CONFIG_URL_INVALID"}))
			case len(prof.Department) > 125:
				w.WriteHeader(http.StatusBadRequest)
				require.NoError(t, encoder.Encode(map[string]any{"error": "DEPARTMENT_INVALID"}))
			case len(prof.SupportPhoneNumber) > 50:
				w.WriteHeader(http.StatusBadRequest)
				require.NoError(t, encoder.Encode(map[string]any{"error": "SUPPORT_PHONE_INVALID"}))
			case len(prof.SupportEmailAddress) > 250:
				w.WriteHeader(http.StatusBadRequest)
				require.NoError(t, encoder.Encode(map[string]any{"error": "SUPPORT_EMAIL_INVALID"}))
			case len(prof.OrgMagic) > 256:
				w.WriteHeader(http.StatusBadRequest)
				require.NoError(t, encoder.Encode(map[string]any{"error": "MAGIC_INVALID"}))
			case !prof.IsMDMRemovable && !prof.IsSupervised:
				w.WriteHeader(http.StatusBadRequest)
				require.NoError(t, encoder.Encode(map[string]any{"error": "FLAGS_INVALID"}))
			default:
				w.WriteHeader(http.StatusOK)
				require.NoError(t, encoder.Encode(godep.ProfileResponse{ProfileUUID: "profile123"}))
			}
		}
	}))

	// get for no team returns 404
	var getResp getMDMAppleSetupAssistantResponse
	s.DoJSON("GET", "/api/latest/fleet/enrollment_profiles/automatic", nil, http.StatusNotFound, &getResp)
	// get for non-existing team returns 404
	s.DoJSON("GET", "/api/latest/fleet/enrollment_profiles/automatic", nil, http.StatusNotFound, &getResp, "team_id", "123")

	// Profile name too long
	var createResp createMDMAppleSetupAssistantResponse
	r := s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "profile_name_too_long",
		EnrollmentProfile: json.RawMessage(fmt.Sprintf(`{"profile_name": "%s"}`, strings.Repeat("a", 126))),
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), "CONFIG_NAME_INVALID")

	// Profile name missing
	r = s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "profile_name_missing",
		EnrollmentProfile: json.RawMessage(`{"profile_name": ""}`),
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), "CONFIG_NAME_REQUIRED")

	// Config URL invalid
	r = s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "profile_name_missing",
		EnrollmentProfile: json.RawMessage(fmt.Sprintf(`{"profile_name": "prof_name", "configuration_web_url": "%s"}`, strings.Repeat("a", 126))),
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), "CONFIG_URL_INVALID")

	// Department invalid
	r = s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "profile_name_missing",
		EnrollmentProfile: json.RawMessage(fmt.Sprintf(`{"profile_name": "prof_name", "configuration_web_url": "https://example.com", "department": "%s"}`, strings.Repeat("a", 126))),
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), "DEPARTMENT_INVALID")

	// Invalid support phone
	r = s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "profile_name_missing",
		EnrollmentProfile: json.RawMessage(fmt.Sprintf(`{"profile_name": "prof_name", "configuration_web_url": "https://example.com", "department": "foo", "support_phone_number": "%s"}`, strings.Repeat("1", 51))),
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), "SUPPORT_PHONE_INVALID")

	// Invalid support email
	r = s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "profile_name_missing",
		EnrollmentProfile: json.RawMessage(fmt.Sprintf(`{"profile_name": "prof_name", "configuration_web_url": "https://example.com", "department": "foo", "support_phone_number": "555-123-4567", "support_email_address": "%s"}`, strings.Repeat("1", 251))),
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), "SUPPORT_EMAIL_INVALID")

	// Invalid magic
	r = s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "profile_name_missing",
		EnrollmentProfile: json.RawMessage(fmt.Sprintf(`{"profile_name": "prof_name", "configuration_web_url": "https://example.com", "department": "foo", "support_phone_number": "555-123-4567", "support_email_address": "support@example.com", "org_magic": "%s"}`, strings.Repeat("1", 257))),
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), "MAGIC_INVALID")

	// Invalid flag combo
	r = s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "profile_name_missing",
		EnrollmentProfile: json.RawMessage(`{"profile_name": "prof_name", "configuration_web_url": "https://example.com", "department": "foo", "support_phone_number": "555-123-4567", "support_email_address": "support@example.com", "org_magic": "1", "is_mdm_removable": false, "is_supervised": false}`),
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), "FLAGS_INVALID")

	// create a setup assistant for no team
	noTeamProf := fmt.Sprintf(defaultProf, "no-team")
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "no-team",
		EnrollmentProfile: json.RawMessage(noTeamProf),
	}, http.StatusOK, &createResp)
	noTeamAsst := createResp.MDMAppleSetupAssistant
	require.Nil(t, noTeamAsst.TeamID)
	require.NotZero(t, noTeamAsst.UploadedAt)
	require.Equal(t, "no-team", noTeamAsst.Name)
	require.JSONEq(t, noTeamProf, string(noTeamAsst.Profile))
	s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		`{"name": "no-team", "team_id": null, "team_name": null}`, 0)

	// create a team and a setup assistant for that team
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name:        t.Name(),
		Description: "desc",
	})
	require.NoError(t, err)
	tmProf := fmt.Sprintf(defaultProf, "team1")
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            &tm.ID,
		Name:              "team1",
		EnrollmentProfile: json.RawMessage(tmProf),
	}, http.StatusOK, &createResp)
	tmAsst := createResp.MDMAppleSetupAssistant
	require.NotNil(t, tmAsst.TeamID)
	require.Equal(t, tm.ID, *tmAsst.TeamID)
	require.NotZero(t, tmAsst.UploadedAt)
	require.Equal(t, "team1", tmAsst.Name)
	require.JSONEq(t, tmProf, string(tmAsst.Profile))
	s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "team1", "team_id": %d, "team_name": %q}`, tm.ID, tm.Name), 0)

	// update no-team
	noTeamProf = fmt.Sprintf(defaultProf, "no-team2")
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "no-team2",
		EnrollmentProfile: json.RawMessage(noTeamProf),
	}, http.StatusOK, &createResp)
	s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		`{"name": "no-team2", "team_id": null, "team_name": null}`, 0)

	// update team
	tmProf = fmt.Sprintf(defaultProf, "team2")
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            &tm.ID,
		Name:              "team2",
		EnrollmentProfile: json.RawMessage(tmProf),
	}, http.StatusOK, &createResp)
	lastChangedActID := s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "team2", "team_id": %d, "team_name": %q}`, tm.ID, tm.Name), 0)

	// sleep a second so the uploaded-at timestamp would change if there were
	// changes, then update again no team/team but without any change, doesn't
	// create a changed activity.
	time.Sleep(time.Second)

	// no change to no-team
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "no-team2",
		EnrollmentProfile: json.RawMessage(noTeamProf),
	}, http.StatusOK, &createResp)
	// the last activity is that of the team (i.e. no new activity was created for no-team)
	s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "team2", "team_id": %d, "team_name": %q}`, tm.ID, tm.Name), lastChangedActID)

	// no change to team
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            &tm.ID,
		Name:              "team2",
		EnrollmentProfile: json.RawMessage(tmProf),
	}, http.StatusOK, &createResp)
	s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "team2", "team_id": %d, "team_name": %q}`, tm.ID, tm.Name), lastChangedActID)

	// update team with only a setup assistant JSON change, should detect it
	// and create a new activity (name is the same)
	tmProf = fmt.Sprintf(defaultProf, "update")
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            &tm.ID,
		Name:              "team2",
		EnrollmentProfile: json.RawMessage(tmProf),
	}, http.StatusOK, &createResp)
	latestChangedActID := s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "team2", "team_id": %d, "team_name": %q}`, tm.ID, tm.Name), 0)
	require.Greater(t, latestChangedActID, lastChangedActID)

	// get no team
	s.DoJSON("GET", "/api/latest/fleet/enrollment_profiles/automatic", nil, http.StatusOK, &getResp)
	require.Nil(t, getResp.TeamID)
	require.NotZero(t, getResp.UploadedAt)
	require.Equal(t, "no-team2", getResp.Name)
	require.JSONEq(t, noTeamProf, string(getResp.Profile))

	// get team
	s.DoJSON("GET", "/api/latest/fleet/enrollment_profiles/automatic", nil, http.StatusOK, &getResp, "team_id", fmt.Sprint(tm.ID))
	require.NotNil(t, getResp.TeamID)
	require.Equal(t, tm.ID, *getResp.TeamID)
	require.NotZero(t, getResp.UploadedAt)
	require.Equal(t, "team2", getResp.Name)
	require.JSONEq(t, tmProf, string(getResp.Profile))

	// try to set the url
	tmProf = `{"url": "https://example.com"}`
	res := s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            &tm.ID,
		Name:              "team5",
		EnrollmentProfile: json.RawMessage(tmProf),
	}, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `The automatic enrollment profile can't include url.`)
	s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "team2", "team_id": %d, "team_name": %q}`, tm.ID, tm.Name), latestChangedActID)

	// try to set a non-object json value
	tmProf = `true`
	res = s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            &tm.ID,
		Name:              "team6",
		EnrollmentProfile: json.RawMessage(tmProf),
	}, http.StatusInternalServerError) // TODO: that should be a 4xx error, see #4406
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `cannot unmarshal bool into Go value of type map[string]interface`)
	s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "team2", "team_id": %d, "team_name": %q}`, tm.ID, tm.Name), latestChangedActID)

	// delete the no-team setup assistant
	s.Do("DELETE", "/api/latest/fleet/enrollment_profiles/automatic", nil, http.StatusNoContent)
	latestChangedActID = s.lastActivityMatches(fleet.ActivityTypeDeletedMacosSetupAssistant{}.ActivityName(),
		`{"name": "no-team2", "team_id": null, "team_name": null}`, 0)

	// get for no team returns 404
	s.DoJSON("GET", "/api/latest/fleet/enrollment_profiles/automatic", nil, http.StatusNotFound, &getResp)

	// delete the team (not the assistant), this also deletes the assistant
	err = s.ds.DeleteTeam(ctx, tm.ID)
	require.NoError(t, err)

	// get for team returns 404
	s.DoJSON("GET", "/api/latest/fleet/enrollment_profiles/automatic", nil, http.StatusNotFound, &getResp, "team_id", fmt.Sprint(tm.ID))

	// no deleted activity was created for the team as the whole team was deleted
	// (a deleted team activity would exist if that was done via the API and not
	// directly with the datastore)
	s.lastActivityMatches(fleet.ActivityTypeDeletedMacosSetupAssistant{}.ActivityName(),
		`{"name": "no-team2", "team_id": null, "team_name": null}`, latestChangedActID)

	// create another team and a setup assistant for that team
	tm2, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name:        t.Name() + "2",
		Description: "desc2",
	})
	require.NoError(t, err)
	tm2Prof := fmt.Sprintf(defaultProf, "teamB")
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            &tm2.ID,
		Name:              "teamB",
		EnrollmentProfile: json.RawMessage(tm2Prof),
	}, http.StatusOK, &createResp)
	s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "teamB", "team_id": %d, "team_name": %q}`, tm2.ID, tm2.Name), 0)

	// delete that team's setup assistant
	s.Do("DELETE", "/api/latest/fleet/enrollment_profiles/automatic", nil, http.StatusNoContent, "team_id", fmt.Sprint(tm2.ID))
	s.lastActivityMatches(fleet.ActivityTypeDeletedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "teamB", "team_id": %d, "team_name": %q}`, tm2.ID, tm2.Name), 0)

	// Try with a team that has no relevant ABM tokens
	teamNoABM, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name:        t.Name() + "no_abm",
		Description: "no abm",
	})
	require.NoError(t, err)
	// Adding another, unrelated token to the DB means that this team (which has no hosts and is not
	// a default team for any token) will not have any relevant tokens and thus we don't know which
	// token to use to hit the Apple APIs.
	otherOrg := t.Name() + "some_other_org"
	s.enableABM(otherOrg)
	// mysql.CreateABMKeyCertIfNotExists(t, s.ds)
	// mysql.CreateAndSetABMToken(t, s.ds, "nurv")
	// err = s.depStorage.StoreConfig(ctx, "nurv", &nanodep_client.Config{BaseURL: srv.URL})
	s.mockDEPResponse(otherOrg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "session123"}`))
		case "/account":
			_, _ = w.Write([]byte(fmt.Sprintf(`{"admin_id": "admin123", "org_name": "%s"}`, "foo")))
		case "/profile":
			w.WriteHeader(http.StatusOK)
			require.NoError(t, encoder.Encode(godep.ProfileResponse{ProfileUUID: "profile123"}))
		}
	}))
	require.NoError(t, err)
	r = s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            &teamNoABM.ID,
		Name:              "profile_name_missing",
		EnrollmentProfile: json.RawMessage(fmt.Sprintf(defaultProf, "no_abm")),
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(r.Body), "No relevant ABM tokens found. Please set this team as a default team for an ABM token.")
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
		require.Equal(t, len(wantProfs), len(gotProfs), "apple host uuid: %s, profiles: %v", h.UUID, idents)

		sort.Slice(gotProfs, func(i, j int) bool {
			l, r := gotProfs[i], gotProfs[j]
			return l.Identifier < r.Identifier
		})
		sort.Slice(wantProfs, func(i, j int) bool {
			l, r := wantProfs[i], wantProfs[j]
			return l.Identifier < r.Identifier
		})
		for i, wp := range wantProfs {
			gp := gotProfs[i]
			require.Equal(t, wp.Identifier, gp.Identifier, "host uuid: %s, prof id: %s", h.UUID, gp.Identifier)
			require.Equal(t, wp.OperationType, gp.OperationType, "host uuid: %s, prof id: %s", h.UUID, gp.Identifier)
			require.Equal(t, wp.Status, gp.Status, "host uuid: %s, prof id: %s", h.UUID, gp.Identifier)
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
		require.Equal(t, len(wantProfs), len(gotProfs), "host uuid: %s", h.UUID)

		sort.Slice(gotProfs, func(i, j int) bool {
			l, r := gotProfs[i], gotProfs[j]
			return l.Name < r.Name
		})
		sort.Slice(wantProfs, func(i, j int) bool {
			l, r := wantProfs[i], wantProfs[j]
			return l.Name < r.Name
		})
		for i, wp := range wantProfs {
			gp := gotProfs[i]
			require.Equal(t, wp.Name, gp.Name, "host uuid: %s, prof id: %s", h.UUID, gp.Name)
			require.Equal(t, wp.OperationType, gp.OperationType, "host uuid: %s, prof id: %s", h.UUID, gp.Name)
			require.Equal(t, wp.Status, gp.Status, "host uuid: %s, prof id: %s", h.UUID, gp.Name)
		}
	}
}

func (s *integrationMDMTestSuite) assertConfigProfilesByIdentifier(teamID *uint, profileIdent string, exists bool) (profile *fleet.MDMAppleConfigProfile) {
	t := s.T()
	if teamID == nil {
		teamID = ptr.Uint(0)
	}
	var cfgProfs []*fleet.MDMAppleConfigProfile
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
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
		teamID = ptr.Uint(0)
	}
	var cfgProfs []*fleet.MDMAppleConfigProfile
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
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
		teamID = ptr.Uint(0)
	}
	var cfgProfs []*fleet.MDMAppleConfigProfile
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
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
		teamID = ptr.Uint(0)
	}
	var cfgProfs []*fleet.MDMWindowsConfigProfile
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
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
	var body bytes.Buffer

	writer := multipart.NewWriter(&body)

	// add file content
	if fileName != "" || len(fileContent) > 0 {
		ff, err := writer.CreateFormFile(uploadFileField, fileName)
		require.NoError(t, err)
		_, err = io.Copy(ff, bytes.NewReader(fileContent))
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

	res := s.DoRawWithHeaders("POST", "/api/latest/fleet/bootstrap", b.Bytes(), expectedStatus, headers)

	if wantErr != "" {
		errMsg := extractServerErrorText(res.Body)
		assert.Contains(t, errMsg, wantErr)
	}
}

func (s *integrationMDMTestSuite) uploadEULA(
	eula *fleet.MDMEULA,
	expectedStatus int,
	wantErr string,
) {
	t := s.T()

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// add the eula field
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

	res := s.DoRawWithHeaders("POST", "/api/latest/fleet/setup_experience/eula", b.Bytes(), expectedStatus, headers)

	if wantErr != "" {
		errMsg := extractServerErrorText(res.Body)
		assert.Contains(t, errMsg, wantErr)
	}
}

// TestGitOpsUserActions tests the MDM permissions listed in ../../docs/Using\ Fleet/manage-access.md
func (s *integrationMDMTestSuite) TestGitOpsUserActions() {
	t := s.T()
	ctx := context.Background()

	//
	// Setup test data.
	// All setup actions are authored by a global admin.
	//

	t1, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "Foo",
	})
	require.NoError(t, err)
	t2, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "Bar",
	})
	require.NoError(t, err)
	t3, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "Zoo",
	})
	require.NoError(t, err)
	// Create the global GitOps user we'll use in tests.
	u := &fleet.User{
		Name:       "GitOps",
		Email:      "gitops1-mdm@example.com",
		GlobalRole: ptr.String(fleet.RoleGitOps),
	}
	require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(context.Background(), u)
	require.NoError(t, err)
	// Create a GitOps user for team t1 we'll use in tests.
	u2 := &fleet.User{
		Name:       "GitOps 2",
		Email:      "gitops2-mdm@example.com",
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *t1,
				Role: fleet.RoleGitOps,
			},
			{
				Team: *t3,
				Role: fleet.RoleGitOps,
			},
		},
	}
	require.NoError(t, u2.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(context.Background(), u2)
	require.NoError(t, err)

	//
	// Start running permission tests with user gitops1-mdm.
	//
	s.setTokenForTest(t, "gitops1-mdm@example.com", test.GoodPassword)

	// Attempt to edit global MDM settings, should allow (also ensure the IdP settings are cleared).
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"macos_setup": {
				"enable_end_user_authentication": false
			},
			"enable_disk_encryption": true,
			"end_user_authentication": {
				"entity_id": "",
				"issuer_uri": "",
				"idp_name": "",
				"metadata_url": ""
			}
		}
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)

	// Attempt to setup Apple MDM, will fail but the important thing is that it
	// fails with 422 (cannot enable end user auth because no IdP is configured)
	// and not 403 forbidden.
	s.Do("PATCH", "/api/latest/fleet/setup_experience",
		fleet.MDMAppleSetupPayload{TeamID: ptr.Uint(0), EnableEndUserAuthentication: ptr.Bool(true)}, http.StatusUnprocessableEntity)

	// Attempt to update the Apple MDM settings but with no change, just to
	// validate the access.
	s.Do("PATCH", "/api/latest/fleet/mdm/apple/settings",
		fleet.MDMAppleSettingsPayload{}, http.StatusNoContent)

	// Attempt to set profile batch globally, should allow.
	globalProfiles := [][]byte{
		mobileconfigForTest("N1", "I1"),
		mobileconfigForTest("N2", "I2"),
	}
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: globalProfiles}, http.StatusNoContent)

	// Attempt to edit team MDM settings, should allow.
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: t1.Name,
		MDM: fleet.TeamSpecMDM{
			EnableDiskEncryption: optjson.SetBool(true),
			MacOSSettings: map[string]interface{}{
				"custom_settings": []interface{}{"foo", "bar"},
			},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	// Attempt to set profile batch for team t1, should allow.
	teamProfiles := [][]byte{
		mobileconfigForTest("N3", "I3"),
		mobileconfigForTest("N4", "I4"),
	}
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{
		Profiles: teamProfiles,
	}, http.StatusNoContent, "team_id", fmt.Sprint(t1.ID))

	//
	// Start running permission tests with user gitops2-mdm,
	// which is GitOps for teams t1 and t3.
	//
	s.setTokenForTest(t, "gitops2-mdm@example.com", test.GoodPassword)

	// Attempt to edit team t1 MDM settings, should allow.
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: t1.Name,
		MDM: fleet.TeamSpecMDM{
			EnableDiskEncryption: optjson.SetBool(true),
			MacOSSettings: map[string]interface{}{
				"custom_settings": []interface{}{"foo", "bar"},
			},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	// Attempt to set profile batch for team t1, should allow.
	teamProfiles = [][]byte{
		mobileconfigForTest("N5", "I5"),
		mobileconfigForTest("N6", "I6"),
	}
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{
		Profiles: teamProfiles,
	}, http.StatusNoContent, "team_id", fmt.Sprint(t1.ID))

	// Attempt to set profile batch for team t2, should not allow.
	teamProfiles = [][]byte{
		mobileconfigForTest("N7", "I7"),
		mobileconfigForTest("N8", "I8"),
	}
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{
		Profiles: teamProfiles,
	}, http.StatusForbidden, "team_id", fmt.Sprint(t2.ID))

	// Attempt to retrieve host profiles fails if the host doesn't belong to the team
	h1, err := s.ds.NewHost(ctx, &fleet.Host{
		NodeKey:  ptr.String(t.Name() + "1"),
		UUID:     t.Name() + "1",
		Hostname: t.Name() + "foo.local",
	})
	require.NoError(t, err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/mdm/hosts/%d/profiles", h1.ID), getHostRequest{}, http.StatusForbidden, &getHostResponse{})

	err = s.ds.AddHostsToTeam(ctx, &t1.ID, []uint{h1.ID})
	require.NoError(t, err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/mdm/hosts/%d/profiles", h1.ID), getHostRequest{}, http.StatusOK, &getHostResponse{})
}

func (s *integrationMDMTestSuite) TestOrgLogo() {
	t := s.T()

	// change org logo urls
	var acResp appConfigResponse
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"org_info": {
			"org_logo_url": "http://test-image.com",
			"org_logo_url_light_background": "http://test-image-light.com"
		}
	}`), http.StatusOK, &acResp)

	// enroll a host
	token := "token_test_migration"
	host := createOrbitEnrolledHost(t, "darwin", "h", s.ds)
	createDeviceTokenForHost(t, s.ds, host.ID, token)

	// check icon urls are correct
	getDesktopResp := fleetDesktopResponse{}
	res := s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
	require.NoError(t, json.NewDecoder(res.Body).Decode(&getDesktopResp))
	require.NoError(t, res.Body.Close())
	require.NoError(t, getDesktopResp.Err)
	require.Equal(t, acResp.OrgInfo.OrgLogoURL, getDesktopResp.Config.OrgInfo.OrgLogoURL)
	require.Equal(t, acResp.OrgInfo.OrgLogoURLLightBackground, getDesktopResp.Config.OrgInfo.OrgLogoURLLightBackground)
}

func (s *integrationMDMTestSuite) setTokenForTest(t *testing.T, email, password string) {
	oldToken := s.token
	t.Cleanup(func() {
		s.token = oldToken
	})

	s.token = s.getCachedUserToken(email, password)
}

func (s *integrationMDMTestSuite) TestSSO() {
	t := s.T()

	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
	}, "MacBookPro16,1")
	s.enableABM(t.Name())
	var lastSubmittedProfile *godep.Profile
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
		case "/profile":
			lastSubmittedProfile = &godep.Profile{}
			rawProfile, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			err = json.Unmarshal(rawProfile, lastSubmittedProfile)
			require.NoError(t, err)
			encoder := json.NewEncoder(w)
			err = encoder.Encode(godep.ProfileResponse{ProfileUUID: "abc"})
			require.NoError(t, err)
		case "/profile/devices":
			encoder := json.NewEncoder(w)
			err := encoder.Encode(godep.ProfileResponse{
				ProfileUUID: "abc",
				Devices:     map[string]string{},
			})
			require.NoError(t, err)
		case "/server/devices", "/devices/sync":
			// This endpoint  is used to get an initial list of
			// devices, return a single device
			encoder := json.NewEncoder(w)
			err := encoder.Encode(godep.DeviceResponse{
				Devices: []godep.Device{
					{
						SerialNumber: mdmDevice.SerialNumber,
						Model:        mdmDevice.Model,
						OS:           "osx",
						OpType:       "added",
					},
				},
			})
			require.NoError(t, err)
		}
	}))

	// sync the list of ABM devices
	s.runDEPSchedule()

	// MDM SSO fields are empty by default
	acResp := appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.Empty(t, acResp.MDM.EndUserAuthentication.SSOProviderSettings)

	// set the SSO fields
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"end_user_authentication": {
				"entity_id": "https://localhost:8080",
				"issuer_uri": "http://localhost:8080/simplesaml/saml2/idp/SSOService.php",
				"idp_name": "SimpleSAML",
				"metadata_url": "http://localhost:9080/simplesaml/saml2/idp/metadata.php"
			},
			"macos_setup": {
				"enable_end_user_authentication": true
			}
		}
	}`), http.StatusOK, &acResp)
	wantSettings := fleet.SSOProviderSettings{
		EntityID:    "https://localhost:8080",
		IssuerURI:   "http://localhost:8080/simplesaml/saml2/idp/SSOService.php",
		IDPName:     "SimpleSAML",
		MetadataURL: "http://localhost:9080/simplesaml/saml2/idp/metadata.php",
	}
	assert.Equal(t, wantSettings, acResp.MDM.EndUserAuthentication.SSOProviderSettings)

	// check that they are returned by a GET /config
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.Equal(t, wantSettings, acResp.MDM.EndUserAuthentication.SSOProviderSettings)

	// trigger the worker to process the job and wait for result before continuing.
	s.runWorker()

	// check that the last submitted DEP profile has been updated accordingly
	require.Contains(t, lastSubmittedProfile.URL, acResp.ServerSettings.ServerURL+"/mdm/sso")
	require.Equal(t, acResp.ServerSettings.ServerURL+"/mdm/sso", lastSubmittedProfile.ConfigurationWebURL)

	// patch without specifying the mdm sso settings fields and an unrelated
	// field, should not remove them
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusOK, &acResp)
	assert.Equal(t, wantSettings, acResp.MDM.EndUserAuthentication.SSOProviderSettings)

	s.runWorker()

	// patch with explicitly empty mdm sso settings fields, would remove
	// them but this is a dry-run
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"end_user_authentication": {
				"entity_id": "",
				"issuer_uri": "",
				"idp_name": "",
				"metadata_url": ""
			},
			"macos_setup": {
				"enable_end_user_authentication": false
			}
		}
	}`), http.StatusOK, &acResp, "dry_run", "true")
	assert.Equal(t, wantSettings, acResp.MDM.EndUserAuthentication.SSOProviderSettings)

	s.runWorker()

	// patch with explicitly empty mdm sso settings fields, fails because end user auth is still enabled
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"end_user_authentication": {
				"entity_id": "",
				"issuer_uri": "",
				"idp_name": "",
				"metadata_url": ""
			}
		}
	}`), http.StatusUnprocessableEntity, &acResp)

	// patch with explicitly empty mdm sso settings fields and disabled end user auth, removes them
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"end_user_authentication": {
				"entity_id": "",
				"issuer_uri": "",
				"idp_name": "",
				"metadata_url": ""
			},
			"macos_setup": {
				"enable_end_user_authentication": false
			}
		}
	}`), http.StatusOK, &acResp)
	assert.Empty(t, acResp.MDM.EndUserAuthentication.SSOProviderSettings)

	s.runWorker()
	require.Equal(t, lastSubmittedProfile.ConfigurationWebURL, lastSubmittedProfile.URL)

	checkStoredIdPInfo := func(uuid, username, fullname, email string) {
		acc, err := s.ds.GetMDMIdPAccountByUUID(context.Background(), uuid)
		require.NoError(t, err)
		require.Equal(t, username, acc.Username)
		require.Equal(t, fullname, acc.Fullname)
		require.Equal(t, email, acc.Email)
	}

	// test basic authentication for each supported config flow.
	//
	// IT admins can set up SSO as part of the same entity or as a completely
	// separate entity.
	//
	// Configs supporting each flow are defined in `tools/saml/config.php`
	configFlows := []string{
		"mdm.test.com",           // independent, mdm-sso only app
		"https://localhost:8080", // app that supports both MDM and Fleet UI SSO
	}
	for _, entityID := range configFlows {
		acResp = appConfigResponse{}
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"server_settings": {"server_url": "https://localhost:8080"},
		"mdm": {
			"end_user_authentication": {
				"entity_id": "%s",
				"issuer_uri": "http://localhost:8080/simplesaml/saml2/idp/SSOService.php",
				"idp_name": "SimpleSAML",
				"metadata_url": "http://localhost:9080/simplesaml/saml2/idp/metadata.php"
			},
			"macos_setup": {
				"enable_end_user_authentication": true
			}
		}
	}`, entityID)), http.StatusOK, &acResp)

		s.runWorker()
		require.Contains(t, lastSubmittedProfile.URL, acResp.ServerSettings.ServerURL+"/mdm/sso")
		require.Equal(t, acResp.ServerSettings.ServerURL+"/mdm/sso", lastSubmittedProfile.ConfigurationWebURL)

		res := s.LoginMDMSSOUser("sso_user", "user123#")
		require.NotEmpty(t, res.Header.Get("Location"))
		require.Equal(t, http.StatusSeeOther, res.StatusCode)

		u, err := url.Parse(res.Header.Get("Location"))
		require.NoError(t, err)
		q := u.Query()
		user1EnrollRef := q.Get("enrollment_reference")
		// without an EULA uploaded
		require.False(t, q.Has("eula_token"))
		require.True(t, q.Has("profile_token"))
		require.True(t, q.Has("enrollment_reference"))
		require.False(t, q.Has("error"))
		// the url retrieves a valid profile
		s.downloadAndVerifyEnrollmentProfile(
			fmt.Sprintf(
				"/api/mdm/apple/enroll?token=%s&enrollment_reference=%s",
				q.Get("profile_token"),
				user1EnrollRef,
			),
		)

		// IdP info stored is accurate for the account
		checkStoredIdPInfo(user1EnrollRef, "sso_user", "SSO User 1", "sso_user@example.com")
	}

	res := s.LoginMDMSSOUser("sso_user", "user123#")
	require.NotEmpty(t, res.Header.Get("Location"))
	require.Equal(t, http.StatusSeeOther, res.StatusCode)

	u, err := url.Parse(res.Header.Get("Location"))
	require.NoError(t, err)
	q := u.Query()
	user1EnrollRef := q.Get("enrollment_reference")

	// upload an EULA
	pdfBytes := []byte("%PDF-1.pdf-contents")
	pdfName := "eula.pdf"
	s.uploadEULA(&fleet.MDMEULA{Bytes: pdfBytes, Name: pdfName}, http.StatusOK, "")

	res = s.LoginMDMSSOUser("sso_user", "user123#")
	require.NotEmpty(t, res.Header.Get("Location"))
	require.Equal(t, http.StatusSeeOther, res.StatusCode)
	u, err = url.Parse(res.Header.Get("Location"))
	require.NoError(t, err)
	q = u.Query()
	// with an EULA uploaded, all values are present
	require.True(t, q.Has("eula_token"))
	require.True(t, q.Has("profile_token"))
	require.True(t, q.Has("enrollment_reference"))
	require.False(t, q.Has("error"))
	// the enrollment reference is the same for the same user
	require.Equal(t, user1EnrollRef, q.Get("enrollment_reference"))
	// the url retrieves a valid profile
	prof := s.downloadAndVerifyEnrollmentProfile(
		fmt.Sprintf(
			"/api/mdm/apple/enroll?token=%s&enrollment_reference=%s",
			q.Get("profile_token"),
			user1EnrollRef,
		),
	)
	// the url retrieves a valid EULA
	resp := s.DoRaw("GET", "/api/latest/fleet/setup_experience/eula/"+q.Get("eula_token"), nil, http.StatusOK)
	require.EqualValues(t, len(pdfBytes), resp.ContentLength)
	require.Equal(t, "application/pdf", resp.Header.Get("content-type"))
	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.EqualValues(t, pdfBytes, respBytes)

	// IdP info stored is accurate for the account
	checkStoredIdPInfo(user1EnrollRef, "sso_user", "SSO User 1", "sso_user@example.com")

	enrollURL := ""
	scepURL := ""
	for _, p := range prof.PayloadContent {
		switch p.PayloadType {
		case "com.apple.security.scep":
			scepURL = p.PayloadContent.URL
		case "com.apple.mdm":
			enrollURL = p.ServerURL
		}
	}
	require.NotEmpty(t, enrollURL)
	require.NotEmpty(t, scepURL)

	// enroll the device using the provided profile
	// we're using localhost for SSO because that's how the local
	// SimpleSAML server is configured, and s.server.URL changes between
	// test runs.
	mdmDevice.EnrollInfo.MDMURL = strings.Replace(enrollURL, "https://localhost:8080", s.server.URL, 1)
	mdmDevice.EnrollInfo.SCEPURL = strings.Replace(scepURL, "https://localhost:8080", s.server.URL, 1)
	err = mdmDevice.Enroll()
	require.NoError(t, err)

	// Enroll generated the TokenUpdate request to Fleet and enqueued the
	// Post-DEP enrollment job, it needs to be processed.
	s.runWorker()

	// ask for commands and verify that we get AccountConfiguration
	var accCmd *mdm.Command
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		if cmd.Command.RequestType == "AccountConfiguration" {
			accCmd = cmd
		}
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}
	require.NotNil(t, accCmd)
	require.NotNil(t, accCmd.Command)

	var fullAccCmd *micromdm.CommandPayload
	require.NoError(t, plist.Unmarshal(accCmd.Raw, &fullAccCmd))
	require.True(t, fullAccCmd.Command.AccountConfiguration.LockPrimaryAccountInfo)
	require.Equal(t, "SSO User 1", fullAccCmd.Command.AccountConfiguration.PrimaryAccountFullName)
	require.Equal(t, "sso_user", fullAccCmd.Command.AccountConfiguration.PrimaryAccountUserName)

	// report host details for the device
	var hostResp getHostResponse
	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/"+mdmDevice.UUID, nil, http.StatusOK, &hostResp)

	ac, err := s.ds.AppConfig(context.Background())
	require.NoError(t, err)

	detailQueries := osquery_utils.GetDetailQueries(context.Background(), config.FleetConfig{}, ac, &ac.Features)

	// simulate osquery reporting mdm information
	rows := []map[string]string{
		{
			"enrolled":           "true",
			"installed_from_dep": "true",
			"server_url":         "https://test.example.com?enrollment_reference=" + user1EnrollRef,
			"payload_identifier": apple_mdm.FleetPayloadIdentifier,
		},
	}
	err = detailQueries["mdm"].DirectIngestFunc(
		context.Background(),
		kitlog.NewNopLogger(),
		&fleet.Host{ID: hostResp.Host.ID},
		s.ds,
		rows,
	)
	require.NoError(t, err)

	// sumulate osquery reporting chrome extension information
	rows = []map[string]string{
		{"email": "g1@example.com"},
		{"email": "g2@example.com"},
	}
	err = detailQueries["google_chrome_profiles"].DirectIngestFunc(
		context.Background(),
		kitlog.NewNopLogger(),
		&fleet.Host{ID: hostResp.Host.ID},
		s.ds,
		rows,
	)
	require.NoError(t, err)

	// host device mapping includes the SSO user and the chrome extension users
	var dmResp listHostDeviceMappingResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d/device_mapping", hostResp.Host.ID), nil, http.StatusOK, &dmResp)
	require.Len(t, dmResp.DeviceMapping, 3)
	sourceByEmail := make(map[string]string, 3)
	for _, dm := range dmResp.DeviceMapping {
		sourceByEmail[dm.Email] = dm.Source
	}
	source, ok := sourceByEmail["sso_user@example.com"]
	require.True(t, ok)
	require.Equal(t, fleet.DeviceMappingMDMIdpAccounts, source)
	source, ok = sourceByEmail["g1@example.com"]
	require.True(t, ok)
	require.Equal(t, "google_chrome_profiles", source)
	source, ok = sourceByEmail["g2@example.com"]
	require.True(t, ok)
	require.Equal(t, "google_chrome_profiles", source)

	// list hosts can filter on mdm idp email
	var hostsResp listHostsResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts?query=%s&device_mapping=true", url.QueryEscape("sso_user@example.com")), nil, http.StatusOK, &hostsResp)
	require.Len(t, hostsResp.Hosts, 1)
	gotHost := hostsResp.Hosts[0]
	require.Equal(t, hostResp.Host.ID, gotHost.ID)
	require.NotNil(t, gotHost.DeviceMapping)
	var dm []fleet.HostDeviceMapping
	require.NoError(t, json.Unmarshal(*gotHost.DeviceMapping, &dm))
	require.Len(t, dm, 3)

	// reporting google chrome profiles only clears chrome profiles from device mapping
	err = detailQueries["google_chrome_profiles"].DirectIngestFunc(
		context.Background(),
		kitlog.NewNopLogger(),
		&fleet.Host{ID: hostResp.Host.ID},
		s.ds,
		[]map[string]string{},
	)
	require.NoError(t, err)
	dmResp = listHostDeviceMappingResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d/device_mapping", hostResp.Host.ID), nil, http.StatusOK, &dmResp)
	require.Len(t, dmResp.DeviceMapping, 1)
	require.Equal(t, "sso_user@example.com", dmResp.DeviceMapping[0].Email)
	require.Equal(t, fleet.DeviceMappingMDMIdpAccounts, dmResp.DeviceMapping[0].Source)
	hostsResp = listHostsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts?query=%s&device_mapping=true", url.QueryEscape("sso_user@example.com")), nil, http.StatusOK, &hostsResp)
	require.Len(t, hostsResp.Hosts, 1)
	gotHost = hostsResp.Hosts[0]
	require.Equal(t, hostResp.Host.ID, gotHost.ID)
	require.NotNil(t, gotHost.DeviceMapping)
	dm = []fleet.HostDeviceMapping{}
	require.NoError(t, json.Unmarshal(*gotHost.DeviceMapping, &dm))
	require.Len(t, dm, 1)
	require.Equal(t, "sso_user@example.com", dm[0].Email)
	require.Equal(t, fleet.DeviceMappingMDMIdpAccounts, dm[0].Source)

	// enrolling a different user works without problems
	res = s.LoginMDMSSOUser("sso_user2", "user123#")
	require.NotEmpty(t, res.Header.Get("Location"))
	require.Equal(t, http.StatusSeeOther, res.StatusCode)
	u, err = url.Parse(res.Header.Get("Location"))
	require.NoError(t, err)
	q = u.Query()
	user2EnrollRef := q.Get("enrollment_reference")
	require.True(t, q.Has("eula_token"))
	require.True(t, q.Has("profile_token"))
	require.True(t, q.Has("enrollment_reference"))
	require.False(t, q.Has("error"))
	// the enrollment reference is different to the one used for the previous user
	require.NotEqual(t, user1EnrollRef, user2EnrollRef)
	// the url retrieves a valid profile
	s.downloadAndVerifyEnrollmentProfile(
		fmt.Sprintf(
			"/api/mdm/apple/enroll?token=%s&enrollment_reference=%s",
			q.Get("profile_token"),
			user2EnrollRef,
		),
	)
	// the url retrieves a valid EULA
	resp = s.DoRaw("GET", "/api/latest/fleet/setup_experience/eula/"+q.Get("eula_token"), nil, http.StatusOK)
	require.EqualValues(t, len(pdfBytes), resp.ContentLength)
	require.Equal(t, "application/pdf", resp.Header.Get("content-type"))
	respBytes, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.EqualValues(t, pdfBytes, respBytes)

	// IdP info stored is accurate for the account
	checkStoredIdPInfo(user2EnrollRef, "sso_user2", "SSO User 2", "sso_user2@example.com")

	// changing the server URL also updates the remote DEP profile
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
                "server_settings": {"server_url": "https://example.com"}
	}`), http.StatusOK, &acResp)

	s.runWorker()
	require.Contains(t, lastSubmittedProfile.URL, "https://example.com/mdm/sso")
	require.Equal(t, "https://example.com/mdm/sso", lastSubmittedProfile.ConfigurationWebURL)

	// hitting the callback with an invalid session id redirects the user to the UI
	rawSSOResp := base64.StdEncoding.EncodeToString([]byte(`<samlp:Response ID="_7822b394622740aa92878ca6c7d1a28c53e80ec5ef"></samlp:Response>`))
	res = s.DoRawNoAuth("POST", "/api/v1/fleet/mdm/sso/callback?SAMLResponse="+url.QueryEscape(rawSSOResp), nil, http.StatusSeeOther)
	require.NotEmpty(t, res.Header.Get("Location"))
	u, err = url.Parse(res.Header.Get("Location"))
	require.NoError(t, err)
	q = u.Query()
	require.False(t, q.Has("eula_token"))
	require.False(t, q.Has("profile_token"))
	require.False(t, q.Has("enrollment_reference"))
	require.True(t, q.Has("error"))
}

type scepPayload struct {
	Challenge string
	URL       string
}

type enrollmentPayload struct {
	PayloadType    string
	ServerURL      string      // used by the enrollment payload
	PayloadContent scepPayload // scep contains a nested payload content dict
}

type enrollmentProfile struct {
	PayloadIdentifier string
	PayloadContent    []enrollmentPayload
}

func (s *integrationMDMTestSuite) downloadAndVerifyEnrollmentProfile(path string) *enrollmentProfile {
	t := s.T()

	resp := s.DoRaw("GET", path, nil, http.StatusOK)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
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

	return s.verifyEnrollmentProfile(body, "")
}

func (s *integrationMDMTestSuite) downloadAndVerifyOTAEnrollmentProfile(path string) {
	t := s.T()

	resp := s.DoRaw("GET", path, nil, http.StatusOK)
	rawProfile, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	require.Contains(t, resp.Header, "Content-Disposition")
	require.Contains(t, resp.Header, "Content-Type")
	require.Contains(t, resp.Header, "X-Content-Type-Options")
	require.Contains(t, resp.Header.Get("Content-Disposition"), "attachment;")
	require.Contains(t, resp.Header.Get("Content-Type"), "application/x-apple-aspen-config")
	require.Contains(t, resp.Header.Get("X-Content-Type-Options"), "nosniff")
	headerLen, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	require.NoError(t, err)
	require.Equal(t, len(rawProfile), headerLen)

	p7, err := pkcs7.Parse(rawProfile)
	require.NoError(t, err)
	rootCA := x509.NewCertPool()

	assets, err := s.ds.GetAllMDMConfigAssetsByName(context.Background(), []fleet.MDMAssetName{
		fleet.MDMAssetCACert,
	}, nil)
	require.NoError(t, err)

	require.True(t, rootCA.AppendCertsFromPEM(assets[fleet.MDMAssetCACert].Value))
	require.NoError(t, p7.VerifyWithChain(rootCA))

	var otaEnrollmentProfile struct {
		PayloadContent struct {
			URL string `plist:"URL"`
		} `plist:"PayloadContent"`
	}
	err = plist.Unmarshal(p7.Content, &otaEnrollmentProfile)
	require.NoError(t, err)
	require.Contains(t, otaEnrollmentProfile.PayloadContent.URL, s.getConfig().ServerSettings.ServerURL+"/api/v1/fleet/ota_enrollment")
}

func (s *integrationMDMTestSuite) verifyEnrollmentProfile(rawProfile []byte, enrollmentRef string) *enrollmentProfile {
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
			require.Equal(t, s.scepChallenge, p.PayloadContent.Challenge)
		case "com.apple.mdm":
			require.Contains(t, p.ServerURL, s.getConfig().ServerSettings.ServerURL+apple_mdm.MDMPath)
			if enrollmentRef != "" {
				require.Contains(t, p.ServerURL, enrollmentRef)
			}
		default:
			require.Failf(t, "unrecognized payload type in enrollment profile: %s", p.PayloadType)
		}
	}
	return &profile
}

func (s *integrationMDMTestSuite) TestMDMMigration() {
	t := s.T()
	ctx := context.Background()

	// enable migration
	var acResp appConfigResponse
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"mdm": { "macos_migration": { "enable": true, "mode": "voluntary", "webhook_url": "https://example.com" } }
	}`), http.StatusOK, &acResp)

	abmToken := s.enableABM(t.Name())

	checkMigrationResponses := func(host *fleet.Host, token string) {
		getDesktopResp := fleetDesktopResponse{}
		res := s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
		require.NoError(t, json.NewDecoder(res.Body).Decode(&getDesktopResp))
		require.NoError(t, res.Body.Close())
		require.NoError(t, getDesktopResp.Err)
		require.Zero(t, *getDesktopResp.FailingPolicies)
		require.False(t, getDesktopResp.Notifications.NeedsMDMMigration)
		require.False(t, getDesktopResp.Notifications.RenewEnrollmentProfile)
		require.Equal(t, acResp.OrgInfo.OrgLogoURL, getDesktopResp.Config.OrgInfo.OrgLogoURL)
		require.Equal(t, acResp.OrgInfo.OrgLogoURLLightBackground, getDesktopResp.Config.OrgInfo.OrgLogoURLLightBackground)
		require.Equal(t, acResp.OrgInfo.ContactURL, getDesktopResp.Config.OrgInfo.ContactURL)
		require.Equal(t, acResp.OrgInfo.OrgName, getDesktopResp.Config.OrgInfo.OrgName)
		require.Equal(t, acResp.MDM.MacOSMigration.Mode, getDesktopResp.Config.MDM.MacOSMigration.Mode)

		orbitConfigResp := orbitGetConfigResponse{}
		s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &orbitConfigResp)
		require.False(t, orbitConfigResp.Notifications.NeedsMDMMigration)
		require.False(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)

		// simulate that the device is assigned to Fleet in ABM
		profileAssignmentStatusResponse := fleet.DEPAssignProfileResponseSuccess
		s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			switch r.URL.Path {
			case "/session":
				_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
			case "/profile":
				encoder := json.NewEncoder(w)
				err := encoder.Encode(godep.ProfileResponse{ProfileUUID: "abc"})
				require.NoError(t, err)
			case "/server/devices", "/devices/sync":
				encoder := json.NewEncoder(w)
				err := encoder.Encode(godep.DeviceResponse{
					Devices: []godep.Device{
						{
							SerialNumber: host.HardwareSerial,
							Model:        "Mac Mini",
							OS:           "osx",
							OpType:       "added",
						},
					},
				})
				require.NoError(t, err)
			case "/profile/devices":
				b, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				var prof profileAssignmentReq
				require.NoError(t, json.Unmarshal(b, &prof))
				var resp godep.ProfileResponse
				resp.ProfileUUID = prof.ProfileUUID
				resp.Devices = map[string]string{
					prof.Devices[0]: string(profileAssignmentStatusResponse),
				}
				encoder := json.NewEncoder(w)
				err = encoder.Encode(resp)
				require.NoError(t, err)
			}
		}))

		cleanAssignmentStatus := func() {
			mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				stmt := `UPDATE host_dep_assignments
					 SET assign_profile_response = NULL,
					     response_updated_at = NULL,
					     profile_uuid = NULL
					 WHERE host_id = ?`
				_, err := q.ExecContext(context.Background(), stmt, host.ID)
				return err
			})
		}

		// simulate that the device is enrolled in a third-party MDM and DEP capable
		err := s.ds.SetOrUpdateMDMData(
			ctx,
			host.ID,
			false,
			true,
			"https://simplemdm.com",
			true,
			fleet.WellKnownMDMSimpleMDM,
			"",
		)
		require.NoError(t, err)

		// simulate a response before we have the chance to assign the profile
		getDesktopResp = fleetDesktopResponse{}
		res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
		require.NoError(t, json.NewDecoder(res.Body).Decode(&getDesktopResp))
		require.NoError(t, res.Body.Close())
		require.NoError(t, getDesktopResp.Err)
		require.Zero(t, *getDesktopResp.FailingPolicies)
		require.False(t, getDesktopResp.Notifications.NeedsMDMMigration)
		require.False(t, getDesktopResp.Notifications.RenewEnrollmentProfile)
		require.Equal(t, acResp.OrgInfo.OrgLogoURL, getDesktopResp.Config.OrgInfo.OrgLogoURL)
		require.Equal(t, acResp.OrgInfo.OrgLogoURLLightBackground, getDesktopResp.Config.OrgInfo.OrgLogoURLLightBackground)
		require.Equal(t, acResp.OrgInfo.ContactURL, getDesktopResp.Config.OrgInfo.ContactURL)
		require.Equal(t, acResp.OrgInfo.OrgName, getDesktopResp.Config.OrgInfo.OrgName)
		require.Equal(t, acResp.MDM.MacOSMigration.Mode, getDesktopResp.Config.MDM.MacOSMigration.Mode)
		orbitConfigResp = orbitGetConfigResponse{}
		s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &orbitConfigResp)
		require.False(t, orbitConfigResp.Notifications.NeedsMDMMigration)
		require.False(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)
		cleanAssignmentStatus()

		// simulate a "FAILED" JSON profile assignment
		profileAssignmentStatusResponse = fleet.DEPAssignProfileResponseFailed
		s.runDEPSchedule()
		getDesktopResp = fleetDesktopResponse{}
		res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
		require.NoError(t, json.NewDecoder(res.Body).Decode(&getDesktopResp))
		require.NoError(t, res.Body.Close())
		require.False(t, getDesktopResp.Notifications.NeedsMDMMigration)
		require.False(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)
		orbitConfigResp = orbitGetConfigResponse{}
		s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &orbitConfigResp)
		require.False(t, orbitConfigResp.Notifications.NeedsMDMMigration)
		require.False(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)
		require.NoError(t, s.ds.DeleteHostDEPAssignments(ctx, abmToken.ID, []string{host.HardwareSerial}))
		cleanAssignmentStatus()

		// simulate a "NOT_ACCESSIBLE" JSON profile assignment
		profileAssignmentStatusResponse = fleet.DEPAssignProfileResponseNotAccessible
		s.runDEPSchedule()
		getDesktopResp = fleetDesktopResponse{}
		res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
		require.NoError(t, json.NewDecoder(res.Body).Decode(&getDesktopResp))
		require.NoError(t, res.Body.Close())
		require.False(t, getDesktopResp.Notifications.NeedsMDMMigration)
		require.False(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)
		s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &orbitConfigResp)
		require.False(t, orbitConfigResp.Notifications.NeedsMDMMigration)
		require.False(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)
		require.NoError(t, s.ds.DeleteHostDEPAssignments(ctx, abmToken.ID, []string{host.HardwareSerial}))
		cleanAssignmentStatus()

		// simulate a "SUCCESS" JSON profile assignment
		profileAssignmentStatusResponse = fleet.DEPAssignProfileResponseSuccess
		s.runDEPSchedule()
		getDesktopResp = fleetDesktopResponse{}
		res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
		require.NoError(t, json.NewDecoder(res.Body).Decode(&getDesktopResp))
		require.NoError(t, res.Body.Close())
		require.True(t, getDesktopResp.Notifications.NeedsMDMMigration)
		require.False(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)
		s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &orbitConfigResp)
		require.True(t, orbitConfigResp.Notifications.NeedsMDMMigration)
		require.False(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)

		// simulate that the device needs to be enrolled in fleet, DEP capable
		err = s.ds.SetOrUpdateMDMData(
			ctx,
			host.ID,
			false,
			false,
			s.server.URL,
			true,
			fleet.WellKnownMDMFleet,
			"",
		)
		require.NoError(t, err)

		getDesktopResp = fleetDesktopResponse{}
		res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
		require.NoError(t, json.NewDecoder(res.Body).Decode(&getDesktopResp))
		require.NoError(t, res.Body.Close())
		require.NoError(t, getDesktopResp.Err)
		require.Zero(t, *getDesktopResp.FailingPolicies)
		require.False(t, getDesktopResp.Notifications.NeedsMDMMigration)
		require.True(t, getDesktopResp.Notifications.RenewEnrollmentProfile)
		require.Equal(t, acResp.OrgInfo.OrgLogoURL, getDesktopResp.Config.OrgInfo.OrgLogoURL)
		require.Equal(t, acResp.OrgInfo.OrgLogoURLLightBackground, getDesktopResp.Config.OrgInfo.OrgLogoURLLightBackground)
		require.Equal(t, acResp.OrgInfo.ContactURL, getDesktopResp.Config.OrgInfo.ContactURL)
		require.Equal(t, acResp.OrgInfo.OrgName, getDesktopResp.Config.OrgInfo.OrgName)
		require.Equal(t, acResp.MDM.MacOSMigration.Mode, getDesktopResp.Config.MDM.MacOSMigration.Mode)

		orbitConfigResp = orbitGetConfigResponse{}
		s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &orbitConfigResp)
		require.False(t, orbitConfigResp.Notifications.NeedsMDMMigration)
		require.True(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)

		// simulate that the device is manually enrolled into fleet, but DEP capable
		err = s.ds.SetOrUpdateMDMData(
			ctx,
			host.ID,
			false,
			true,
			s.server.URL,
			false,
			fleet.WellKnownMDMFleet,
			"",
		)
		require.NoError(t, err)
		mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
			SCEPChallenge: s.scepChallenge,
			SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
			MDMURL:        s.server.URL + apple_mdm.MDMPath,
		}, "MacBookPro16,1")
		mdmDevice.SerialNumber = host.HardwareSerial
		mdmDevice.UUID = host.UUID
		err = mdmDevice.Enroll()
		require.NoError(t, err)
		require.NoError(t, err)
		getDesktopResp = fleetDesktopResponse{}
		res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
		require.NoError(t, json.NewDecoder(res.Body).Decode(&getDesktopResp))
		require.NoError(t, res.Body.Close())
		require.NoError(t, getDesktopResp.Err)
		require.Zero(t, *getDesktopResp.FailingPolicies)
		require.False(t, getDesktopResp.Notifications.NeedsMDMMigration)
		require.False(t, getDesktopResp.Notifications.RenewEnrollmentProfile)
		require.Equal(t, acResp.OrgInfo.OrgLogoURL, getDesktopResp.Config.OrgInfo.OrgLogoURL)
		require.Equal(t, acResp.OrgInfo.OrgLogoURLLightBackground, getDesktopResp.Config.OrgInfo.OrgLogoURLLightBackground)
		require.Equal(t, acResp.OrgInfo.ContactURL, getDesktopResp.Config.OrgInfo.ContactURL)
		require.Equal(t, acResp.OrgInfo.OrgName, getDesktopResp.Config.OrgInfo.OrgName)
		require.Equal(t, acResp.MDM.MacOSMigration.Mode, getDesktopResp.Config.MDM.MacOSMigration.Mode)

		orbitConfigResp = orbitGetConfigResponse{}
		s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &orbitConfigResp)
		require.False(t, orbitConfigResp.Notifications.NeedsMDMMigration)
		require.False(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)

		// simulate a host that was reset to factory, but fleet still has an mdm record active
		err = s.ds.SetOrUpdateMDMData(
			ctx,
			host.ID,
			false,
			true,
			s.server.URL,
			false,
			fleet.WellKnownMDMSimpleMDM,
			"",
		)
		require.NoError(t, err)
		getDesktopResp = fleetDesktopResponse{}
		res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
		require.NoError(t, json.NewDecoder(res.Body).Decode(&getDesktopResp))
		require.NoError(t, res.Body.Close())
		require.NoError(t, getDesktopResp.Err)
		require.Zero(t, *getDesktopResp.FailingPolicies)
		require.True(t, getDesktopResp.Notifications.NeedsMDMMigration)
		require.False(t, getDesktopResp.Notifications.RenewEnrollmentProfile)
		require.Equal(t, acResp.OrgInfo.OrgLogoURL, getDesktopResp.Config.OrgInfo.OrgLogoURL)
		require.Equal(t, acResp.OrgInfo.OrgLogoURLLightBackground, getDesktopResp.Config.OrgInfo.OrgLogoURLLightBackground)
		require.Equal(t, acResp.OrgInfo.ContactURL, getDesktopResp.Config.OrgInfo.ContactURL)
		require.Equal(t, acResp.OrgInfo.OrgName, getDesktopResp.Config.OrgInfo.OrgName)
		require.Equal(t, acResp.MDM.MacOSMigration.Mode, getDesktopResp.Config.MDM.MacOSMigration.Mode)

		orbitConfigResp = orbitGetConfigResponse{}
		s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &orbitConfigResp)
		require.True(t, orbitConfigResp.Notifications.NeedsMDMMigration)
		require.False(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)

		// simulate a device that is manually enrolled to 3rd party
		err = s.ds.SetOrUpdateMDMData(
			ctx,
			host.ID,
			false,
			true,
			"https://simplemdm.com",
			false,
			fleet.WellKnownMDMSimpleMDM,
			"",
		)
		require.NoError(t, err)
		getDesktopResp = fleetDesktopResponse{}
		res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
		require.NoError(t, json.NewDecoder(res.Body).Decode(&getDesktopResp))
		require.NoError(t, res.Body.Close())
		require.NoError(t, getDesktopResp.Err)
		require.Zero(t, *getDesktopResp.FailingPolicies)
		require.True(t, getDesktopResp.Notifications.NeedsMDMMigration)
		require.False(t, getDesktopResp.Notifications.RenewEnrollmentProfile)
		require.Equal(t, acResp.OrgInfo.OrgLogoURL, getDesktopResp.Config.OrgInfo.OrgLogoURL)
		require.Equal(t, acResp.OrgInfo.OrgLogoURLLightBackground, getDesktopResp.Config.OrgInfo.OrgLogoURLLightBackground)
		require.Equal(t, acResp.OrgInfo.ContactURL, getDesktopResp.Config.OrgInfo.ContactURL)
		require.Equal(t, acResp.OrgInfo.OrgName, getDesktopResp.Config.OrgInfo.OrgName)
		require.Equal(t, acResp.MDM.MacOSMigration.Mode, getDesktopResp.Config.MDM.MacOSMigration.Mode)

		orbitConfigResp = orbitGetConfigResponse{}
		s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &orbitConfigResp)
		require.True(t, orbitConfigResp.Notifications.NeedsMDMMigration)
		require.False(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)

		// clean up nano tables
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(context.Background(), `
			DELETE FROM nano_enrollments WHERE id = ?
			`, host.UUID)
			return err
		})
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(context.Background(), `
			DELETE FROM nano_devices WHERE id = ?
			`, host.UUID)
			return err
		})
	}

	token := "token_test_migration"
	host := createOrbitEnrolledHost(t, "darwin", "h", s.ds)
	createDeviceTokenForHost(t, s.ds, host.ID, token)
	checkMigrationResponses(host, token)
	require.NoError(t, s.ds.DeleteHostDEPAssignments(ctx, abmToken.ID, []string{host.HardwareSerial}))

	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team-1"})
	require.NoError(t, err)
	err = s.ds.AddHostsToTeam(ctx, &tm.ID, []uint{host.ID})
	require.NoError(t, err)
	checkMigrationResponses(host, token)
}

// ///////////////////////////////////////////////////////////////////////////
// Windows MDM tests

func (s *integrationMDMTestSuite) TestAppConfigWindowsMDM() {
	ctx := context.Background()
	t := s.T()

	appConf, err := s.ds.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.MDM.WindowsEnabledAndConfigured = false
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)

	var acResp appConfigResponse
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.WindowsEnabledAndConfigured)

	// create a couple teams
	tm1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "1"})
	require.NoError(t, err)
	tm2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "2"})
	require.NoError(t, err)

	// enable Windows MDM
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "windows_enabled_and_configured": true }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.WindowsEnabledAndConfigured)
	assert.False(t, acResp.MDM.WindowsMigrationEnabled)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledWindowsMDM{}.ActivityName(), `{}`, 0)

	// create some hosts - a Windows workstation in each team and no-team,
	// Windows server in no team, Windows workstation enrolled in a 3rd-party in
	// team 2, Windows workstation already enrolled in Fleet in no team, and a
	// macOS host in no team.
	metadataHosts := []struct {
		os            string
		suffix        string
		isServer      bool
		teamID        *uint
		enrolledName  string
		shouldEnroll  bool
		shouldMigrate bool
	}{
		{"windows", "win-no-team", false, nil, "", true, false},
		{"windows", "win-team-1", false, &tm1.ID, "", true, false},
		{"windows", "win-team-2", false, &tm2.ID, "", true, false},
		{"windows", "win-server", true, nil, "", false, false},                                      // is a server
		{"windows", "win-third-party", false, &tm2.ID, fleet.WellKnownMDMSimpleMDM, false, true},    // is enrolled in 3rd-party
		{"windows", "win-fleet", false, nil, fleet.WellKnownMDMFleet, false, false},                 // is already Fleet-enrolled
		{"darwin", "macos-no-team", false, nil, "", false, false},                                   // is not Windows
		{"windows", "win-server-third-party", true, nil, fleet.WellKnownMDMSimpleMDM, false, false}, // is enrolled in 3rd-party, but is a server
	}
	hostsBySuffix := make(map[string]*fleet.Host, len(metadataHosts))
	for _, meta := range metadataHosts {
		var host *fleet.Host
		if meta.os == "windows" && meta.enrolledName == fleet.WellKnownMDMFleet {
			// special-case to create a properly MDM-enrolled into Fleet host
			host = createOrbitEnrolledHost(t, meta.os, meta.suffix, s.ds)
			mdmDevice := mdmtest.NewTestMDMClientWindowsProgramatic(s.server.URL, *host.OrbitNodeKey)
			err := mdmDevice.Enroll()
			require.NoError(t, err)
			err = s.ds.UpdateMDMWindowsEnrollmentsHostUUID(ctx, host.UUID, mdmDevice.DeviceID)
			require.NoError(t, err)
			err = s.ds.SetOrUpdateMDMData(ctx, host.ID, meta.isServer, true, s.server.URL, false, fleet.WellKnownMDMFleet, "")
			require.NoError(t, err)
		} else {
			host = createOrbitEnrolledHost(t, meta.os, meta.suffix, s.ds)
			createDeviceTokenForHost(t, s.ds, host.ID, meta.suffix)

			serverURL := "https://example.com"
			err := s.ds.SetOrUpdateMDMData(ctx, host.ID, meta.isServer, meta.enrolledName != "", serverURL, false, meta.enrolledName, "")
			require.NoError(t, err)
		}

		if meta.teamID != nil {
			err = s.ds.AddHostsToTeam(ctx, meta.teamID, []uint{host.ID})
			require.NoError(t, err)
		}
		hostsBySuffix[meta.suffix] = host
	}

	// get the orbit config for each host, verify that only the expected ones
	// receive the "needs enrollment to Windows MDM" notification.
	for _, meta := range metadataHosts {
		var resp orbitGetConfigResponse
		s.DoJSON("POST", "/api/fleet/orbit/config",
			json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hostsBySuffix[meta.suffix].OrbitNodeKey)),
			http.StatusOK, &resp)
		require.Equal(t, meta.shouldEnroll, resp.Notifications.NeedsProgrammaticWindowsMDMEnrollment)
		require.False(t, resp.Notifications.NeedsProgrammaticWindowsMDMUnenrollment)
		require.False(t, resp.Notifications.NeedsMDMMigration)
		if meta.shouldEnroll {
			require.Contains(t, resp.Notifications.WindowsMDMDiscoveryEndpoint, microsoft_mdm.MDE2DiscoveryPath)
		} else {
			require.Empty(t, resp.Notifications.WindowsMDMDiscoveryEndpoint)
		}
	}

	// enable Windows MDM migration
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "windows_migration_enabled": true }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.WindowsEnabledAndConfigured)
	assert.True(t, acResp.MDM.WindowsMigrationEnabled)
	s.lastActivityMatches(fleet.ActivityTypeEnabledWindowsMDMMigration{}.ActivityName(), `{}`, 0)

	// get the orbit config for each host, verify that only the expected ones
	// receive the "needs enrollment to Windows MDM" and "needs migration" notifications.
	// They still get enrollment notifications as we have not proceeded with enrollment.
	for _, meta := range metadataHosts {
		var resp orbitGetConfigResponse
		s.DoJSON("POST", "/api/fleet/orbit/config",
			json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hostsBySuffix[meta.suffix].OrbitNodeKey)),
			http.StatusOK, &resp)
		require.Equal(t, meta.shouldEnroll, resp.Notifications.NeedsProgrammaticWindowsMDMEnrollment)
		require.Equal(t, meta.shouldMigrate, resp.Notifications.NeedsMDMMigration)
		require.False(t, resp.Notifications.NeedsProgrammaticWindowsMDMUnenrollment)
		if meta.shouldEnroll {
			require.Contains(t, resp.Notifications.WindowsMDMDiscoveryEndpoint, microsoft_mdm.MDE2DiscoveryPath)
		} else {
			require.Empty(t, resp.Notifications.WindowsMDMDiscoveryEndpoint)
		}
	}

	// turn on MDM for another host
	orbitHost, _ := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	// disable Microsoft MDM
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "windows_enabled_and_configured": false }
  }`), http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.WindowsEnabledAndConfigured)
	assert.False(t, acResp.MDM.WindowsMigrationEnabled)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeDisabledWindowsMDM{}.ActivityName(), `{}`, 0)

	// get the orbit config for that MDM-enrolled host returns true for the
	// unenrollment notification
	var resp orbitGetConfigResponse
	s.DoJSON("POST", "/api/fleet/orbit/config",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitHost.OrbitNodeKey)),
		http.StatusOK, &resp)
	require.True(t, resp.Notifications.NeedsProgrammaticWindowsMDMUnenrollment)
	require.False(t, resp.Notifications.NeedsProgrammaticWindowsMDMEnrollment)
	require.False(t, resp.Notifications.NeedsMDMMigration)
	require.Empty(t, resp.Notifications.WindowsMDMDiscoveryEndpoint)

	// get the orbit config for each host, only the fleet-enrolled ones get the unenrollment,
	// and none get enrollment/migration (because MDM is now off).
	for _, meta := range metadataHosts {
		var resp orbitGetConfigResponse
		s.DoJSON("POST", "/api/fleet/orbit/config",
			json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hostsBySuffix[meta.suffix].OrbitNodeKey)),
			http.StatusOK, &resp)
		require.False(t, resp.Notifications.NeedsProgrammaticWindowsMDMEnrollment)
		require.False(t, resp.Notifications.NeedsMDMMigration)
		if meta.enrolledName == fleet.WellKnownMDMFleet {
			require.True(t, resp.Notifications.NeedsProgrammaticWindowsMDMUnenrollment)
		} else {
			require.False(t, resp.Notifications.NeedsProgrammaticWindowsMDMUnenrollment)
		}
		require.Empty(t, resp.Notifications.WindowsMDMDiscoveryEndpoint)
	}
}

func (s *integrationMDMTestSuite) TestOrbitConfigNudgeSettings() {
	t := s.T()

	// ensure the config is empty before starting
	s.applyConfig([]byte(`
  mdm:
    macos_updates:
      deadline: ""
      minimum_version: ""
 `))

	var resp orbitGetConfigResponse
	// missing orbit key
	s.DoJSON("POST", "/api/fleet/orbit/config", nil, http.StatusUnauthorized, &resp)

	// nudge config is empty if macos_updates is not set, and Windows MDM notifications are unset
	h := createOrbitEnrolledHost(t, "darwin", "h", s.ds)

	err := s.ds.UpdateHostOperatingSystem(context.Background(), h.ID, fleet.OperatingSystem{Platform: "darwin", Version: "12.0"})
	require.NoError(t, err)

	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h.OrbitNodeKey)), http.StatusOK, &resp)
	require.Empty(t, resp.NudgeConfig)
	require.False(t, resp.Notifications.NeedsProgrammaticWindowsMDMEnrollment)
	require.Empty(t, resp.Notifications.WindowsMDMDiscoveryEndpoint)
	require.False(t, resp.Notifications.NeedsProgrammaticWindowsMDMUnenrollment)

	// set macos_updates
	s.applyConfig([]byte(`
  mdm:
    macos_updates:
      deadline: 2022-01-04
      minimum_version: 12.1.3
 `))

	// still empty if MDM is turned off for the host
	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h.OrbitNodeKey)), http.StatusOK, &resp)
	require.Empty(t, resp.NudgeConfig)

	// turn on MDM features
	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "MacBookPro16,1")
	mdmDevice.SerialNumber = h.HardwareSerial
	mdmDevice.UUID = h.UUID
	err = mdmDevice.Enroll()
	require.NoError(t, err)

	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h.OrbitNodeKey)), http.StatusOK, &resp)
	wantCfg, err := fleet.NewNudgeConfig(fleet.AppleOSUpdateSettings{Deadline: optjson.SetString("2022-01-04"), MinimumVersion: optjson.SetString("12.1.3")})
	require.NoError(t, err)
	require.Equal(t, wantCfg, resp.NudgeConfig)
	require.Equal(t, wantCfg.OSVersionRequirements[0].RequiredInstallationDate.String(), "2022-01-04 20:00:00 +0000 UTC")

	// create a team with an empty macos_updates config
	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          4827,
		Name:        "team1_" + t.Name(),
		Description: "desc team1_" + t.Name(),
	})
	require.NoError(t, err)
	s.assertMacOSDeclarationsByName(&team.ID, servermdm.FleetMacOSUpdatesProfileName, false)

	// add the host to the team
	err = s.ds.AddHostsToTeam(context.Background(), &team.ID, []uint{h.ID})
	require.NoError(t, err)

	// NudgeConfig should be empty
	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h.OrbitNodeKey)), http.StatusOK, &resp)
	require.Empty(t, resp.NudgeConfig)
	require.Equal(t, wantCfg.OSVersionRequirements[0].RequiredInstallationDate.String(), "2022-01-04 20:00:00 +0000 UTC")

	// modify the team config, add macos_updates config
	var tmResp teamResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		MDM: &fleet.TeamPayloadMDM{
			MacOSUpdates: &fleet.AppleOSUpdateSettings{
				Deadline:       optjson.SetString("1992-01-01"),
				MinimumVersion: optjson.SetString("13.1.1"),
			},
		},
	}, http.StatusOK, &tmResp)
	s.assertMacOSDeclarationsByName(&team.ID, servermdm.FleetMacOSUpdatesProfileName, true)

	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h.OrbitNodeKey)), http.StatusOK, &resp)
	wantCfg, err = fleet.NewNudgeConfig(fleet.AppleOSUpdateSettings{Deadline: optjson.SetString("1992-01-01"), MinimumVersion: optjson.SetString("13.1.1")})
	require.NoError(t, err)
	require.Equal(t, wantCfg, resp.NudgeConfig)
	require.Equal(t, wantCfg.OSVersionRequirements[0].RequiredInstallationDate.String(), "1992-01-01 20:00:00 +0000 UTC")

	// create a new host, still receives the global config
	h2 := createOrbitEnrolledHost(t, "darwin", "h2", s.ds)
	mdmDevice = mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "MacBookPro16,1")
	mdmDevice.SerialNumber = h2.HardwareSerial
	mdmDevice.UUID = h2.UUID
	err = mdmDevice.Enroll()
	require.NoError(t, err)

	err = s.ds.UpdateHostOperatingSystem(context.Background(), h2.ID, fleet.OperatingSystem{Platform: "darwin", Version: "12.0"})
	require.NoError(t, err)

	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h2.OrbitNodeKey)), http.StatusOK, &resp)
	wantCfg, err = fleet.NewNudgeConfig(fleet.AppleOSUpdateSettings{Deadline: optjson.SetString("2022-01-04"), MinimumVersion: optjson.SetString("12.1.3")})
	require.NoError(t, err)
	require.Equal(t, wantCfg, resp.NudgeConfig)
	require.Equal(t, wantCfg.OSVersionRequirements[0].RequiredInstallationDate.String(), "2022-01-04 20:00:00 +0000 UTC")

	// host on macos > 14, shouldn't be receiving nudge configs
	h3 := createOrbitEnrolledHost(t, "darwin", "h3", s.ds)

	mdmDevice = mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "MacBookPro16,1")
	mdmDevice.SerialNumber = h3.HardwareSerial
	mdmDevice.UUID = h3.UUID
	err = mdmDevice.Enroll()
	require.NoError(t, err)

	err = s.ds.UpdateHostOperatingSystem(context.Background(), h3.ID, fleet.OperatingSystem{Platform: "darwin", Version: "14.1"})
	require.NoError(t, err)

	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h3.OrbitNodeKey)), http.StatusOK, &resp)
	require.Nil(t, resp.NudgeConfig)

	// host is available for nudge, but has not had details query run
	// yet, so we don't know the os version.
	h4 := createOrbitEnrolledHost(t, "darwin", "h4", s.ds)

	mdmDevice = mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "MacBookPro16,1")
	mdmDevice.SerialNumber = h4.HardwareSerial
	mdmDevice.UUID = h4.UUID
	err = mdmDevice.Enroll()
	require.NoError(t, err)

	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h4.OrbitNodeKey)), http.StatusOK, &resp)
	require.Nil(t, resp.NudgeConfig)
}

func (s *integrationMDMTestSuite) TestValidDiscoveryRequest() {
	t := s.T()

	// Preparing the Discovery Request message
	requestBytes := []byte(`
		 <s:Envelope xmlns:a="http://www.w3.org/2005/08/addressing" xmlns:s="http://www.w3.org/2003/05/soap-envelope">
		   <s:Header>
		     <a:Action s:mustUnderstand="1">http://schemas.microsoft.com/windows/management/2012/01/enrollment/IDiscoveryService/Discover</a:Action>
		     <a:MessageID>urn:uuid:148132ec-a575-4322-b01b-6172a9cf8478</a:MessageID>
		     <a:ReplyTo>
		       <a:Address>http://www.w3.org/2005/08/addressing/anonymous</a:Address>
		     </a:ReplyTo>
		     <a:To s:mustUnderstand="1">https://mdmwindows.com:443/EnrollmentServer/Discovery.svc</a:To>
		   </s:Header>
		   <s:Body>
		     <Discover xmlns="http://schemas.microsoft.com/windows/management/2012/01/enrollment">
		       <request xmlns:i="http://www.w3.org/2001/XMLSchema-instance">
		         <EmailAddress>demo@mdmwindows.com</EmailAddress>
		         <RequestVersion>5.0</RequestVersion>
		         <DeviceType>CIMClient_Windows</DeviceType>
		         <ApplicationVersion>6.2.9200.2965</ApplicationVersion>
		         <OSEdition>48</OSEdition>
		         <AuthPolicies>
		           <AuthPolicy>OnPremise</AuthPolicy>
		           <AuthPolicy>Federated</AuthPolicy>
		         </AuthPolicies>
		       </request>
		     </Discover>
		   </s:Body>
		 </s:Envelope>`)

	resp := s.DoRaw("POST", microsoft_mdm.MDE2DiscoveryPath, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], syncml.SoapContentType)

	// Checking if SOAP response can be unmarshalled to an golang type
	var xmlType interface{}
	err = xml.Unmarshal(resBytes, &xmlType)
	require.NoError(t, err)

	// Checking if SOAP response contains a valid DiscoveryResponse message
	resSoapMsg := string(resBytes)
	require.True(t, s.isXMLTagPresent("DiscoverResult", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("AuthPolicy", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("EnrollmentVersion", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("EnrollmentPolicyServiceUrl", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("EnrollmentServiceUrl", resSoapMsg))
}

func (s *integrationMDMTestSuite) TestInvalidDiscoveryRequest() {
	t := s.T()

	// Preparing the Discovery Request message
	requestBytes := []byte(`
		 <s:Envelope xmlns:a="http://www.w3.org/2005/08/addressing" xmlns:s="http://www.w3.org/2003/05/soap-envelope">
		   <s:Header>
		     <a:Action s:mustUnderstand="1">http://schemas.microsoft.com/windows/management/2012/01/enrollment/IDiscoveryService/Discover</a:Action>
		     <a:ReplyTo>
		       <a:Address>http://www.w3.org/2005/08/addressing/anonymous</a:Address>
		     </a:ReplyTo>
		     <a:To s:mustUnderstand="1">https://mdmwindows.com:443/EnrollmentServer/Discovery.svc</a:To>
		   </s:Header>
		   <s:Body>
		     <Discover xmlns="http://schemas.microsoft.com/windows/management/2012/01/enrollment">
		       <request xmlns:i="http://www.w3.org/2001/XMLSchema-instance">
		         <EmailAddress>demo@mdmwindows.com</EmailAddress>
		         <RequestVersion>5.0</RequestVersion>
		         <DeviceType>CIMClient_Windows</DeviceType>
		         <ApplicationVersion>6.2.9200.2965</ApplicationVersion>
		         <OSEdition>48</OSEdition>
		         <AuthPolicies>
		           <AuthPolicy>OnPremise</AuthPolicy>
		           <AuthPolicy>Federated</AuthPolicy>
		         </AuthPolicies>
		       </request>
		     </Discover>
		   </s:Body>
		 </s:Envelope>`)

	resp := s.DoRaw("POST", microsoft_mdm.MDE2DiscoveryPath, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], syncml.SoapContentType)

	// Checking if response can be unmarshalled to an golang type
	var xmlType interface{}
	err = xml.Unmarshal(resBytes, &xmlType)
	require.NoError(t, err)

	// Checking if SOAP response contains a valid SoapFault message
	resSoapMsg := string(resBytes)

	require.True(t, s.isXMLTagPresent("s:fault", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("s:value", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("s:text", resSoapMsg))
	require.True(t, s.checkIfXMLTagContains("s:text", "invalid SOAP header: Header.MessageID", resSoapMsg))
}

func (s *integrationMDMTestSuite) TestNoEmailDiscoveryRequest() {
	t := s.T()

	// Preparing the Discovery Request message
	requestBytes := []byte(`
		 <s:Envelope xmlns:a="http://www.w3.org/2005/08/addressing" xmlns:s="http://www.w3.org/2003/05/soap-envelope">
		   <s:Header>
		     <a:Action s:mustUnderstand="1">http://schemas.microsoft.com/windows/management/2012/01/enrollment/IDiscoveryService/Discover</a:Action>
		     <a:MessageID>urn:uuid:148132ec-a575-4322-b01b-6172a9cf8478</a:MessageID>
		     <a:ReplyTo>
		       <a:Address>http://www.w3.org/2005/08/addressing/anonymous</a:Address>
		     </a:ReplyTo>
		     <a:To s:mustUnderstand="1">https://mdmwindows.com:443/EnrollmentServer/Discovery.svc</a:To>
		   </s:Header>
		   <s:Body>
		     <Discover xmlns="http://schemas.microsoft.com/windows/management/2012/01/enrollment">
		       <request xmlns:i="http://www.w3.org/2001/XMLSchema-instance">
		         <EmailAddress></EmailAddress>
		         <RequestVersion>5.0</RequestVersion>
		         <DeviceType>CIMClient_Windows</DeviceType>
		         <ApplicationVersion>6.2.9200.2965</ApplicationVersion>
		         <OSEdition>48</OSEdition>
		         <AuthPolicies>
		           <AuthPolicy>OnPremise</AuthPolicy>
		           <AuthPolicy>Federated</AuthPolicy>
		         </AuthPolicies>
		       </request>
		     </Discover>
		   </s:Body>
		 </s:Envelope>`)

	resp := s.DoRaw("POST", microsoft_mdm.MDE2DiscoveryPath, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], syncml.SoapContentType)

	// Checking if SOAP response can be unmarshalled to an golang type
	var xmlType interface{}
	err = xml.Unmarshal(resBytes, &xmlType)
	require.NoError(t, err)

	// Checking if SOAP response contains a valid DiscoveryResponse message
	resSoapMsg := string(resBytes)
	require.True(t, s.isXMLTagPresent("DiscoverResult", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("AuthPolicy", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("EnrollmentVersion", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("EnrollmentPolicyServiceUrl", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("EnrollmentServiceUrl", resSoapMsg))
	require.True(t, !s.isXMLTagContentPresent("AuthenticationServiceUrl", resSoapMsg))
}

func (s *integrationMDMTestSuite) TestValidGetPoliciesRequestWithDeviceToken() {
	t := s.T()

	// create a new Host to get the UUID on the DB
	windowsHost := createOrbitEnrolledHost(t, "windows", "h1", s.ds)

	// Preparing the GetPolicies Request message
	encodedBinToken, err := fleet.GetEncodedBinarySecurityToken(fleet.WindowsMDMProgrammaticEnrollmentType, *windowsHost.OrbitNodeKey)
	require.NoError(t, err)

	requestBytes, err := s.newGetPoliciesMsg(true, encodedBinToken)
	require.NoError(t, err)

	resp := s.DoRaw("POST", microsoft_mdm.MDE2PolicyPath, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], syncml.SoapContentType)

	// Checking if SOAP response can be unmarshalled to an golang type
	var xmlType interface{}
	err = xml.Unmarshal(resBytes, &xmlType)
	require.NoError(t, err)

	// Checking if SOAP response contains a valid GetPoliciesResponse message
	resSoapMsg := string(resBytes)
	require.True(t, s.isXMLTagPresent("GetPoliciesResponse", resSoapMsg))
	require.True(t, s.isXMLTagPresent("policyOIDReference", resSoapMsg))
	require.True(t, s.isXMLTagPresent("oIDReferenceID", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("validityPeriodSeconds", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("renewalPeriodSeconds", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("minimalKeyLength", resSoapMsg))
}

func (s *integrationMDMTestSuite) TestValidGetPoliciesRequestWithAzureToken() {
	t := s.T()

	// Preparing the GetPolicies Request message with Azure JWT token
	azureADTok := "ZXlKMGVYQWlPaUpLVjFRaUxDSmhiR2NpT2lKU1V6STFOaUlzSW5nMWRDSTZJaTFMU1ROUk9XNU9VamRpVW05bWVHMWxXbTlZY1dKSVdrZGxkeUlzSW10cFpDSTZJaTFMU1ROUk9XNU9VamRpVW05bWVHMWxXbTlZY1dKSVdrZGxkeUo5LmV5SmhkV1FpT2lKb2RIUndjem92TDIxaGNtTnZjMnhoWW5NdWIzSm5MeUlzSW1semN5STZJbWgwZEhCek9pOHZjM1J6TG5kcGJtUnZkM011Ym1WMEwyWmhaVFZqTkdZekxXWXpNVGd0TkRRNE15MWlZelptTFRjMU9UVTFaalJoTUdFM01pOGlMQ0pwWVhRaU9qRTJPRGt4TnpBNE5UZ3NJbTVpWmlJNk1UWTRPVEUzTURnMU9Dd2laWGh3SWpveE5qZzVNVGMxTmpZeExDSmhZM0lpT2lJeElpd2lZV2x2SWpvaVFWUlJRWGt2T0ZSQlFVRkJOV2gwUTNFMGRERjNjbHBwUTIxQmVEQlpWaTloZGpGTVMwRkRPRXM1Vm10SGVtNUdXVGxzTUZoYWVrZHVha2N6VVRaMWVIUldNR3QxT1hCeFJXdFRZeUlzSW1GdGNpSTZXeUp3ZDJRaUxDSnljMkVpWFN3aVlYQndhV1FpT2lJeU9XUTVaV1E1T0MxaE5EWTVMVFExTXpZdFlXUmxNaTFtT1RneFltTXhaRFl3TldVaUxDSmhjSEJwWkdGamNpSTZJakFpTENKa1pYWnBZMlZwWkNJNkltRXhNMlkzWVdVd0xURXpPR0V0TkdKaU1pMDVNalF5TFRka09USXlaVGRqTkdGak15SXNJbWx3WVdSa2NpSTZJakU0Tmk0eE1pNHhPRGN1TWpZaUxDSnVZVzFsSWpvaVZHVnpkRTFoY21OdmMweGhZbk1pTENKdmFXUWlPaUpsTTJNMU5XVmtZeTFqTXpRNExUUTBNVFl0T0dZd05TMHlOVFJtWmpNd05qVmpOV1VpTENKd2QyUmZkWEpzSWpvaWFIUjBjSE02THk5d2IzSjBZV3d1YldsamNtOXpiMlowYjI1c2FXNWxMbU52YlM5RGFHRnVaMlZRWVhOemQyOXlaQzVoYzNCNElpd2ljbWdpT2lJd0xrRldTVUU0T0ZSc0xXaHFlbWN3VXpoaU0xZFdXREJ2UzJOdFZGRXpTbHB1ZUUxa1QzQTNUbVZVVm5OV2FYVkhOa0ZRYnk0aUxDSnpZM0FpT2lKdFpHMWZaR1ZzWldkaGRHbHZiaUlzSW5OMVlpSTZJa1pTUTJ4RldURk9ObXR2ZEdWblMzcFplV0pFTjJkdFdGbGxhVTVIUkZrd05FSjJOV3R6ZDJGeGJVRWlMQ0owYVdRaU9pSm1ZV1UxWXpSbU15MW1NekU0TFRRME9ETXRZbU0yWmkwM05UazFOV1kwWVRCaE56SWlMQ0oxYm1seGRXVmZibUZ0WlNJNkluUmxjM1JBYldGeVkyOXpiR0ZpY3k1dmNtY2lMQ0oxY0c0aU9pSjBaWE4wUUcxaGNtTnZjMnhoWW5NdWIzSm5JaXdpZFhScElqb2lNVGg2WkVWSU5UZFRSWFZyYWpseGJqRm9aMlJCUVNJc0luWmxjaUk2SWpFdU1DSjkuVG1FUlRsZktBdWo5bTVvQUc2UTBRblV4VEFEaTNFamtlNHZ3VXo3UTdqUUFVZVZGZzl1U0pzUXNjU2hFTXVxUmQzN1R2VlpQanljdEVoRFgwLVpQcEVVYUlSempuRVEyTWxvc21SZURYZzhrYkhNZVliWi1jb0ZucDEyQkVpQnpJWFBGZnBpaU1GRnNZZ0hSSF9tSWxwYlBlRzJuQ2p0LTZSOHgzYVA5QS1tM0J3eV91dnV0WDFNVEVZRmFsekhGa04wNWkzbjZRcjhURnlJQ1ZUYW5OanlkMjBBZFRMbHJpTVk0RVBmZzRaLThVVTctZkcteElycWVPUmVWTnYwOUFHV192MDd6UkVaNmgxVk9tNl9nelRGcElVVURuZFdabnFLTHlySDlkdkF3WnFFSG1HUmlTNElNWnRFdDJNTkVZSnhDWHhlSi1VbWZJdV9tUVhKMW9R"
	requestBytes, err := s.newGetPoliciesMsg(false, azureADTok)
	require.NoError(t, err)

	resp := s.DoRaw("POST", microsoft_mdm.MDE2PolicyPath, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], syncml.SoapContentType)

	// Checking if SOAP response can be unmarshalled to an golang type
	var xmlType interface{}
	err = xml.Unmarshal(resBytes, &xmlType)
	require.NoError(t, err)

	// Checking if SOAP response contains a valid GetPoliciesResponse message
	resSoapMsg := string(resBytes)
	require.True(t, s.isXMLTagPresent("GetPoliciesResponse", resSoapMsg))
	require.True(t, s.isXMLTagPresent("policyOIDReference", resSoapMsg))
	require.True(t, s.isXMLTagPresent("oIDReferenceID", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("validityPeriodSeconds", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("renewalPeriodSeconds", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("minimalKeyLength", resSoapMsg))
}

func (s *integrationMDMTestSuite) TestGetPoliciesRequestWithInvalidUUID() {
	t := s.T()

	// create a new Host to get the UUID on the DB
	_, err := s.ds.NewHost(context.Background(), &fleet.Host{
		ID:            1,
		OsqueryHostID: ptr.String("Desktop-ABCQWE"),
		NodeKey:       ptr.String("Desktop-ABCQWE"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.not.enrolled", s.T().Name()),
		Platform:      "windows",
	})
	require.NoError(t, err)

	// Preparing the GetPolicies Request message
	encodedBinToken, err := fleet.GetEncodedBinarySecurityToken(fleet.WindowsMDMProgrammaticEnrollmentType, "not_exists")
	require.NoError(t, err)

	requestBytes, err := s.newGetPoliciesMsg(true, encodedBinToken)
	require.NoError(t, err)

	resp := s.DoRaw("POST", microsoft_mdm.MDE2PolicyPath, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], syncml.SoapContentType)

	// Checking if SOAP response can be unmarshalled to an golang type
	var xmlType interface{}
	err = xml.Unmarshal(resBytes, &xmlType)
	require.NoError(t, err)

	// Checking if SOAP response contains a valid SoapFault message
	resSoapMsg := string(resBytes)
	require.True(t, s.isXMLTagPresent("s:fault", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("s:value", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("s:text", resSoapMsg))
	require.True(t, s.checkIfXMLTagContains("s:text", "host data cannot be found", resSoapMsg))
}

func (s *integrationMDMTestSuite) TestGetPoliciesRequestWithNotElegibleHost() {
	t := s.T()

	// create a new Host to get the UUID on the DB
	linuxHost := createOrbitEnrolledHost(t, "linux", "h1", s.ds)

	// Preparing the GetPolicies Request message
	encodedBinToken, err := fleet.GetEncodedBinarySecurityToken(fleet.WindowsMDMProgrammaticEnrollmentType, *linuxHost.OrbitNodeKey)
	require.NoError(t, err)

	requestBytes, err := s.newGetPoliciesMsg(true, encodedBinToken)
	require.NoError(t, err)

	resp := s.DoRaw("POST", microsoft_mdm.MDE2PolicyPath, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], syncml.SoapContentType)

	// Checking if SOAP response can be unmarshalled to an golang type
	var xmlType interface{}
	err = xml.Unmarshal(resBytes, &xmlType)
	require.NoError(t, err)

	// Checking if SOAP response contains a valid SoapFault message
	resSoapMsg := string(resBytes)
	require.True(t, s.isXMLTagPresent("s:fault", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("s:value", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("s:text", resSoapMsg))
	require.True(t, s.checkIfXMLTagContains("s:text", "host is not elegible for Windows MDM enrollment", resSoapMsg))
}

func (s *integrationMDMTestSuite) TestValidRequestSecurityTokenRequestWithDeviceToken() {
	t := s.T()
	windowsHost := createOrbitEnrolledHost(t, "windows", "h1", s.ds)

	// Delete the host from the list of MDM enrolled devices if present
	_ = s.ds.MDMWindowsDeleteEnrolledDevice(context.Background(), windowsHost.UUID)

	// Preparing the RequestSecurityToken Request message
	encodedBinToken, err := fleet.GetEncodedBinarySecurityToken(fleet.WindowsMDMProgrammaticEnrollmentType, *windowsHost.OrbitNodeKey)
	require.NoError(t, err)

	requestBytes, err := s.newSecurityTokenMsg(encodedBinToken, true, false)
	require.NoError(t, err)

	resp := s.DoRaw("POST", microsoft_mdm.MDE2EnrollPath, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], syncml.SoapContentType)

	// Checking if SOAP response can be unmarshalled to an golang type
	var xmlType interface{}
	err = xml.Unmarshal(resBytes, &xmlType)
	require.NoError(t, err)

	// Checking if SOAP response contains a valid RequestSecurityTokenResponseCollection message
	resSoapMsg := string(resBytes)

	require.True(t, s.isXMLTagPresent("RequestSecurityTokenResponseCollection", resSoapMsg))
	require.True(t, s.isXMLTagPresent("DispositionMessage", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("TokenType", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("RequestID", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("BinarySecurityToken", resSoapMsg))

	// Checking if an activity was created for the enrollment
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeMDMEnrolled{}.ActivityName(),
		fmt.Sprintf(`{
			"mdm_platform": "microsoft",
			"host_serial": "%s",
			"installed_from_dep": false,
			"host_display_name": "%s"
		 }`, windowsHost.HardwareSerial, windowsHost.DisplayName()),
		0)

	expectedDeviceID := "AB157C3A18778F4FB21E2739066C1F27" // TODO: make the hard-coded deviceID in `s.newSecurityTokenMsg` configurable

	// Checking if the host uuid was set on mdm windows enrollments
	d, err := s.ds.MDMWindowsGetEnrolledDeviceWithDeviceID(context.Background(), expectedDeviceID)
	require.NoError(t, err)
	require.NotEmpty(t, d.HostUUID)
	require.Equal(t, windowsHost.UUID, d.HostUUID)
}

// TODO: Do we need integration tests for WindowsMDMAutomaticEnrollmentType flows?

func (s *integrationMDMTestSuite) TestValidRequestSecurityTokenRequestWithAzureToken() {
	t := s.T()

	// Preparing the SecurityToken Request message with Azure JWT token
	azureADTok := "ZXlKMGVYQWlPaUpLVjFRaUxDSmhiR2NpT2lKU1V6STFOaUlzSW5nMWRDSTZJaTFMU1ROUk9XNU9VamRpVW05bWVHMWxXbTlZY1dKSVdrZGxkeUlzSW10cFpDSTZJaTFMU1ROUk9XNU9VamRpVW05bWVHMWxXbTlZY1dKSVdrZGxkeUo5LmV5SmhkV1FpT2lKb2RIUndjem92TDIxaGNtTnZjMnhoWW5NdWIzSm5MeUlzSW1semN5STZJbWgwZEhCek9pOHZjM1J6TG5kcGJtUnZkM011Ym1WMEwyWmhaVFZqTkdZekxXWXpNVGd0TkRRNE15MWlZelptTFRjMU9UVTFaalJoTUdFM01pOGlMQ0pwWVhRaU9qRTJPRGt4TnpBNE5UZ3NJbTVpWmlJNk1UWTRPVEUzTURnMU9Dd2laWGh3SWpveE5qZzVNVGMxTmpZeExDSmhZM0lpT2lJeElpd2lZV2x2SWpvaVFWUlJRWGt2T0ZSQlFVRkJOV2gwUTNFMGRERjNjbHBwUTIxQmVEQlpWaTloZGpGTVMwRkRPRXM1Vm10SGVtNUdXVGxzTUZoYWVrZHVha2N6VVRaMWVIUldNR3QxT1hCeFJXdFRZeUlzSW1GdGNpSTZXeUp3ZDJRaUxDSnljMkVpWFN3aVlYQndhV1FpT2lJeU9XUTVaV1E1T0MxaE5EWTVMVFExTXpZdFlXUmxNaTFtT1RneFltTXhaRFl3TldVaUxDSmhjSEJwWkdGamNpSTZJakFpTENKa1pYWnBZMlZwWkNJNkltRXhNMlkzWVdVd0xURXpPR0V0TkdKaU1pMDVNalF5TFRka09USXlaVGRqTkdGak15SXNJbWx3WVdSa2NpSTZJakU0Tmk0eE1pNHhPRGN1TWpZaUxDSnVZVzFsSWpvaVZHVnpkRTFoY21OdmMweGhZbk1pTENKdmFXUWlPaUpsTTJNMU5XVmtZeTFqTXpRNExUUTBNVFl0T0dZd05TMHlOVFJtWmpNd05qVmpOV1VpTENKd2QyUmZkWEpzSWpvaWFIUjBjSE02THk5d2IzSjBZV3d1YldsamNtOXpiMlowYjI1c2FXNWxMbU52YlM5RGFHRnVaMlZRWVhOemQyOXlaQzVoYzNCNElpd2ljbWdpT2lJd0xrRldTVUU0T0ZSc0xXaHFlbWN3VXpoaU0xZFdXREJ2UzJOdFZGRXpTbHB1ZUUxa1QzQTNUbVZVVm5OV2FYVkhOa0ZRYnk0aUxDSnpZM0FpT2lKdFpHMWZaR1ZzWldkaGRHbHZiaUlzSW5OMVlpSTZJa1pTUTJ4RldURk9ObXR2ZEdWblMzcFplV0pFTjJkdFdGbGxhVTVIUkZrd05FSjJOV3R6ZDJGeGJVRWlMQ0owYVdRaU9pSm1ZV1UxWXpSbU15MW1NekU0TFRRME9ETXRZbU0yWmkwM05UazFOV1kwWVRCaE56SWlMQ0oxYm1seGRXVmZibUZ0WlNJNkluUmxjM1JBYldGeVkyOXpiR0ZpY3k1dmNtY2lMQ0oxY0c0aU9pSjBaWE4wUUcxaGNtTnZjMnhoWW5NdWIzSm5JaXdpZFhScElqb2lNVGg2WkVWSU5UZFRSWFZyYWpseGJqRm9aMlJCUVNJc0luWmxjaUk2SWpFdU1DSjkuVG1FUlRsZktBdWo5bTVvQUc2UTBRblV4VEFEaTNFamtlNHZ3VXo3UTdqUUFVZVZGZzl1U0pzUXNjU2hFTXVxUmQzN1R2VlpQanljdEVoRFgwLVpQcEVVYUlSempuRVEyTWxvc21SZURYZzhrYkhNZVliWi1jb0ZucDEyQkVpQnpJWFBGZnBpaU1GRnNZZ0hSSF9tSWxwYlBlRzJuQ2p0LTZSOHgzYVA5QS1tM0J3eV91dnV0WDFNVEVZRmFsekhGa04wNWkzbjZRcjhURnlJQ1ZUYW5OanlkMjBBZFRMbHJpTVk0RVBmZzRaLThVVTctZkcteElycWVPUmVWTnYwOUFHV192MDd6UkVaNmgxVk9tNl9nelRGcElVVURuZFdabnFLTHlySDlkdkF3WnFFSG1HUmlTNElNWnRFdDJNTkVZSnhDWHhlSi1VbWZJdV9tUVhKMW9R"
	requestBytes, err := s.newSecurityTokenMsg(azureADTok, false, false)
	require.NoError(t, err)

	resp := s.DoRaw("POST", microsoft_mdm.MDE2EnrollPath, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], syncml.SoapContentType)

	// Checking if SOAP response can be unmarshalled to an golang type
	var xmlType interface{}
	err = xml.Unmarshal(resBytes, &xmlType)
	require.NoError(t, err)

	// Checking if SOAP response contains a valid RequestSecurityTokenResponseCollection message
	resSoapMsg := string(resBytes)
	require.True(t, s.isXMLTagPresent("RequestSecurityTokenResponseCollection", resSoapMsg))
	require.True(t, s.isXMLTagPresent("DispositionMessage", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("TokenType", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("RequestID", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("BinarySecurityToken", resSoapMsg))

	// Checking if an activity was created for the enrollment
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeMDMEnrolled{}.ActivityName(),
		`{
			"mdm_platform": "microsoft",
			"host_serial": "",
			"installed_from_dep": false,
			"host_display_name": "DESKTOP-0C89RC0"
		 }`,
		0)

	expectedDeviceID := "AB157C3A18778F4FB21E2739066C1F27" // TODO: make the hard-coded deviceID in `s.newSecurityTokenMsg` configurable

	// Checking the host uuid was not set on mdm windows enrollments
	d, err := s.ds.MDMWindowsGetEnrolledDeviceWithDeviceID(context.Background(), expectedDeviceID)
	require.NoError(t, err)
	require.Empty(t, d.HostUUID)
}

func (s *integrationMDMTestSuite) TestInvalidRequestSecurityTokenRequestWithMissingAdditionalContext() {
	t := s.T()

	// create a new Host to get the UUID on the DB
	windowsHost := createOrbitEnrolledHost(t, "windows", "h1", s.ds)

	// Preparing the RequestSecurityToken Request message
	encodedBinToken, err := fleet.GetEncodedBinarySecurityToken(fleet.WindowsMDMProgrammaticEnrollmentType, *windowsHost.OrbitNodeKey)
	require.NoError(t, err)

	requestBytes, err := s.newSecurityTokenMsg(encodedBinToken, true, true)
	require.NoError(t, err)

	resp := s.DoRaw("POST", microsoft_mdm.MDE2EnrollPath, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], syncml.SoapContentType)

	// Checking if SOAP response can be unmarshalled to an golang type
	var xmlType interface{}
	err = xml.Unmarshal(resBytes, &xmlType)
	require.NoError(t, err)

	// Checking if SOAP response contains a valid SoapFault message
	resSoapMsg := string(resBytes)
	require.True(t, s.isXMLTagPresent("s:fault", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("s:value", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("s:text", resSoapMsg))
	require.True(t, s.checkIfXMLTagContains("s:text", "ContextItem item DeviceType is not present", resSoapMsg))
}

func (s *integrationMDMTestSuite) TestValidGetAuthRequest() {
	t := s.T()

	// Target Endpoint url with query params
	targetEndpointURL := microsoft_mdm.MDE2AuthPath + "?appru=ms-app%3A%2F%2Fwindows.immersivecontrolpanel&login_hint=demo%40mdmwindows.com"
	resp := s.DoRaw("GET", targetEndpointURL, nil, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, resp.Header["Content-Type"], "text/html; charset=UTF-8")
	require.NotEmpty(t, resBytes)

	// Checking response content
	resContent := string(resBytes)
	require.Contains(t, resContent, "inputToken.name = 'wresult'")
	require.Contains(t, resContent, "form.action = \"ms-app://windows.immersivecontrolpanel\"")
	require.Contains(t, resContent, "performPost()")

	// Getting token content
	encodedToken := s.getRawTokenValue(resContent)
	require.NotEmpty(t, encodedToken)
}

func (s *integrationMDMTestSuite) TestInvalidGetAuthRequest() {
	t := s.T()

	// Target Endpoint url with no login_hit query param
	targetEndpointURL := microsoft_mdm.MDE2AuthPath + "?appru=ms-app%3A%2F%2Fwindows.immersivecontrolpanel"
	resp := s.DoRaw("GET", targetEndpointURL, nil, http.StatusInternalServerError)

	resBytes, err := io.ReadAll(resp.Body)
	resContent := string(resBytes)
	require.NoError(t, err)
	require.NotEmpty(t, resBytes)
	require.Contains(t, resContent, "forbidden")
}

func (s *integrationMDMTestSuite) TestValidGetTOC() {
	t := s.T()

	// hacky check to make sure the assets were built (they require `-tags full`
	// when building the tests otherwise this test will always fail due to a
	// panic in server/bindata package)
	defer func() {
		if panicVal := recover(); panicVal != nil {
			s, ok := panicVal.(string)
			if ok && strings.Contains(s, "Assets may not be used when running Fleet as a library") {
				t.Skip("skipping, test will fail due to assets not built (requires '-tags full')")
			}
		}
	}()
	_, _ = bindata.Asset("check if assets are build")

	resp := s.DoRaw("GET", microsoft_mdm.MDE2TOSPath+"?api-version=1.0&redirect_uri=ms-appx-web%3a%2f%2fMicrosoft.AAD.BrokerPlugin&client-request-id=f2cf3127-1e80-4d73-965d-42a3b84bdb40", nil, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], syncml.WebContainerContentType)

	resTOCcontent := string(resBytes)
	require.Contains(t, resTOCcontent, "Microsoft.AAD.BrokerPlugin")
	require.Contains(t, resTOCcontent, "IsAccepted=true")
	require.Contains(t, resTOCcontent, "OpaqueBlob=")
}

func (s *integrationMDMTestSuite) TestWindowsMDM() {
	t := s.T()
	orbitHost, d := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	cmdOneUUID := uuid.New().String()
	commandOne := &fleet.MDMWindowsCommand{
		CommandUUID: cmdOneUUID,
		RawCommand: []byte(fmt.Sprintf(`
                     <Exec>
                       <CmdID>%s</CmdID>
                       <Item>
                         <Target>
                           <LocURI>./Device/Vendor/MSFT/Reboot/RebootNow</LocURI>
                         </Target>
                         <Meta>
                           <Format xmlns="syncml:metinf">null</Format>
                           <Type>text/plain</Type>
                         </Meta>
                         <Data></Data>
                       </Item>
                     </Exec>
		`, cmdOneUUID)),
		TargetLocURI: "./Device/Vendor/MSFT/Reboot/RebootNow",
	}
	err := s.ds.MDMWindowsInsertCommandForHosts(context.Background(), []string{orbitHost.UUID}, commandOne)
	require.NoError(t, err)

	cmds, err := d.StartManagementSession()
	require.NoError(t, err)
	// 2 Status + 1 Exec
	require.Len(t, cmds, 3)
	receivedCmd := cmds[cmdOneUUID]
	require.NotNil(t, receivedCmd)
	require.Equal(t, receivedCmd.Verb, fleet.CmdExec)
	require.Len(t, receivedCmd.Cmd.Items, 1)
	require.EqualValues(t, "./Device/Vendor/MSFT/Reboot/RebootNow", *receivedCmd.Cmd.Items[0].Target)

	msgID, err := d.GetCurrentMsgID()
	require.NoError(t, err)

	d.AppendResponse(fleet.SyncMLCmd{
		XMLName: xml.Name{Local: fleet.CmdStatus},
		MsgRef:  &msgID,
		CmdRef:  &cmdOneUUID,
		Cmd:     ptr.String("Exec"),
		Data:    ptr.String("200"),
		Items:   nil,
		CmdID:   fleet.CmdID{Value: uuid.NewString()},
	})
	cmds, err = d.SendResponse()
	require.NoError(t, err)
	// the ack of the message should be the only returned command
	require.Len(t, cmds, 1)

	cmdTwoUUID := uuid.New().String()
	commandTwo := &fleet.MDMWindowsCommand{
		CommandUUID: cmdTwoUUID,
		RawCommand: []byte(fmt.Sprintf(`
                    <Get>
                      <CmdID>%s</CmdID>
                      <Item>
                        <Target>
                          <LocURI>./Device/Vendor/MSFT/DMClient/Provider/DEMO%%20MDM/SignedEntDMID</LocURI>
                        </Target>
                      </Item>
                    </Get>
		`, cmdTwoUUID)),
		TargetLocURI: "./Device/Vendor/MSFT/DMClient/Provider/DEMO%%20MDM/SignedEntDMID",
	}
	err = s.ds.MDMWindowsInsertCommandForHosts(context.Background(), []string{orbitHost.UUID}, commandTwo)
	require.NoError(t, err)

	cmdThreeUUID := uuid.New().String()
	commandThree := &fleet.MDMWindowsCommand{
		CommandUUID: cmdThreeUUID,
		RawCommand: []byte(fmt.Sprintf(`
                    <Replace>
                       <CmdID>%s</CmdID>
                       <Item>
                         <Target>
                           <LocURI>./Device/Vendor/MSFT/DMClient/Provider/DEMO%%20MDM/SignedEntDMID</LocURI>
                         </Target>
                         <Meta>
                           <Type xmlns="syncml:metinf">text/plain</Type>
                           <Format xmlns="syncml:metinf">chr</Format>
                         </Meta>
                         <Data>1</Data>
                       </Item>
                    </Replace>
		`, cmdThreeUUID)),
		TargetLocURI: "./Device/Vendor/MSFT/DMClient/Provider/DEMO%%20MDM/SignedEntDMID",
	}
	err = s.ds.MDMWindowsInsertCommandForHosts(context.Background(), []string{orbitHost.UUID}, commandThree)
	require.NoError(t, err)

	cmdFourUUID := uuid.New().String()
	commandFour := &fleet.MDMWindowsCommand{
		CommandUUID: cmdFourUUID,
		RawCommand: []byte(fmt.Sprintf(`
                    <Add>
                       <CmdID>%s</CmdID>
                       <Item>
                         <Target>
                           <LocURI>./Vendor/MSFT/WiFi/Profile/MyNetwork/WlanXml</LocURI>
                         </Target>
                         <Meta>
                           <Type xmlns="syncml:metinf">text/plain</Type>
                           <Format xmlns="syncml:metinf">chr</Format>
                         </Meta>
                         <Data>
						   &lt;?xml version=&quot;1.0&quot;?&gt;&lt;WLANProfile
						   xmlns=&quot;http://contoso.com/provisioning/EapHostConfig&quot;&gt;&lt;EapMethod&gt;&lt;Type
						 </Data>
                       </Item>
                    </Add>
		`, cmdFourUUID)),
		TargetLocURI: "./Vendor/MSFT/WiFi/Profile/MyNetwork/WlanXml",
	}
	err = s.ds.MDMWindowsInsertCommandForHosts(context.Background(), []string{orbitHost.UUID}, commandFour)
	require.NoError(t, err)

	cmds, err = d.StartManagementSession()
	require.NoError(t, err)
	// two status + the three commands we enqueued
	require.Len(t, cmds, 5)
	receivedCmdTwo := cmds[cmdTwoUUID]
	require.NotNil(t, receivedCmdTwo)
	require.Equal(t, receivedCmdTwo.Verb, fleet.CmdGet)
	require.Len(t, receivedCmdTwo.Cmd.Items, 1)
	require.EqualValues(t, "./Device/Vendor/MSFT/DMClient/Provider/DEMO%20MDM/SignedEntDMID", *receivedCmdTwo.Cmd.Items[0].Target)

	receivedCmdThree := cmds[cmdThreeUUID]
	require.NotNil(t, receivedCmdThree)
	require.Equal(t, receivedCmdThree.Verb, fleet.CmdReplace)
	require.Len(t, receivedCmdThree.Cmd.Items, 1)
	require.EqualValues(t, "./Device/Vendor/MSFT/DMClient/Provider/DEMO%20MDM/SignedEntDMID", *receivedCmdThree.Cmd.Items[0].Target)

	receivedCmdFour := cmds[cmdFourUUID]
	require.NotNil(t, receivedCmdFour)
	require.Equal(t, receivedCmdFour.Verb, fleet.CmdAdd)
	require.Len(t, receivedCmdFour.Cmd.Items, 1)
	require.EqualValues(t, "./Vendor/MSFT/WiFi/Profile/MyNetwork/WlanXml", *receivedCmdFour.Cmd.Items[0].Target)

	// status 200 for command Two  (Get)
	d.AppendResponse(fleet.SyncMLCmd{
		XMLName: xml.Name{Local: fleet.CmdStatus},
		MsgRef:  &msgID,
		CmdRef:  &cmdTwoUUID,
		Cmd:     ptr.String("Get"),
		Data:    ptr.String("200"),
		Items:   nil,
		CmdID:   fleet.CmdID{Value: uuid.NewString()},
	})
	// results for command two (Get)
	cmdTwoRespUUID := uuid.NewString()
	d.AppendResponse(fleet.SyncMLCmd{
		XMLName: xml.Name{Local: fleet.CmdResults},
		MsgRef:  &msgID,
		CmdRef:  &cmdTwoUUID,
		Cmd:     ptr.String("Replace"),
		Data:    ptr.String("200"),
		Items: []fleet.CmdItem{
			{
				Source: ptr.String("./Device/Vendor/MSFT/DMClient/Provider/DEMO%20MDM/SignedEntDMID"),
				Data:   &fleet.RawXmlData{Content: "0"},
			},
		},
		CmdID: fleet.CmdID{Value: cmdTwoRespUUID},
	})
	// status 200 for command Three (Replace)
	d.AppendResponse(fleet.SyncMLCmd{
		XMLName: xml.Name{Local: fleet.CmdStatus},
		MsgRef:  &msgID,
		CmdRef:  &cmdThreeUUID,
		Cmd:     ptr.String("Replace"),
		Data:    ptr.String("200"),
		Items:   nil,
		CmdID:   fleet.CmdID{Value: uuid.NewString()},
	})
	// status 200 for command Four (Add)
	d.AppendResponse(fleet.SyncMLCmd{
		XMLName: xml.Name{Local: fleet.CmdStatus},
		MsgRef:  &msgID,
		CmdRef:  &cmdFourUUID,
		Cmd:     ptr.String("Add"),
		Data:    ptr.String("200"),
		Items:   nil,
		CmdID:   fleet.CmdID{Value: uuid.NewString()},
	})
	cmds, err = d.SendResponse()
	require.NoError(t, err)

	// the ack of the message should be the only returned command
	require.Len(t, cmds, 1)

	// check command results

	getCommandFullResult := func(cmdUUID string) []byte {
		var fullResult []byte
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(context.Background(), q, &fullResult, `
			SELECT raw_response
			FROM windows_mdm_responses wmr
			JOIN windows_mdm_command_results wmcr ON wmcr.response_id = wmr.id
			WHERE command_uuid = ?
			`, cmdUUID)
		})
		return fullResult
	}

	var getMDMCmdResp getMDMCommandResultsResponse
	s.DoJSON("GET", "/api/latest/fleet/commands/results", nil, http.StatusOK, &getMDMCmdResp, "command_uuid", cmdOneUUID)
	require.Len(t, getMDMCmdResp.Results, 1)
	require.NotZero(t, getMDMCmdResp.Results[0].UpdatedAt)
	getMDMCmdResp.Results[0].UpdatedAt = time.Time{}
	require.Equal(t, &fleet.MDMCommandResult{
		HostUUID:    orbitHost.UUID,
		CommandUUID: cmdOneUUID,
		Status:      "200",
		RequestType: "./Device/Vendor/MSFT/Reboot/RebootNow",
		Result:      getCommandFullResult(cmdOneUUID),
		Payload:     commandOne.RawCommand,
		Hostname:    "TestIntegrationsMDM/TestWindowsMDMh1.local",
	}, getMDMCmdResp.Results[0])

	s.DoJSON("GET", "/api/latest/fleet/commands/results", nil, http.StatusOK, &getMDMCmdResp, "command_uuid", cmdTwoUUID)
	require.Len(t, getMDMCmdResp.Results, 1)
	require.NotZero(t, getMDMCmdResp.Results[0].UpdatedAt)
	getMDMCmdResp.Results[0].UpdatedAt = time.Time{}
	require.Equal(t, &fleet.MDMCommandResult{
		HostUUID:    orbitHost.UUID,
		CommandUUID: cmdTwoUUID,
		Status:      "200",
		RequestType: "./Device/Vendor/MSFT/DMClient/Provider/DEMO%%20MDM/SignedEntDMID",
		Result:      getCommandFullResult(cmdTwoUUID),
		Payload:     commandTwo.RawCommand,
		Hostname:    "TestIntegrationsMDM/TestWindowsMDMh1.local",
	}, getMDMCmdResp.Results[0])

	s.DoJSON("GET", "/api/latest/fleet/commands/results", nil, http.StatusOK, &getMDMCmdResp, "command_uuid", cmdThreeUUID)
	require.Len(t, getMDMCmdResp.Results, 1)
	require.NotZero(t, getMDMCmdResp.Results[0].UpdatedAt)
	getMDMCmdResp.Results[0].UpdatedAt = time.Time{}
	require.Equal(t, &fleet.MDMCommandResult{
		HostUUID:    orbitHost.UUID,
		CommandUUID: cmdThreeUUID,
		Status:      "200",
		RequestType: "./Device/Vendor/MSFT/DMClient/Provider/DEMO%%20MDM/SignedEntDMID",
		Result:      getCommandFullResult(cmdThreeUUID),
		Hostname:    "TestIntegrationsMDM/TestWindowsMDMh1.local",
		Payload:     commandThree.RawCommand,
	}, getMDMCmdResp.Results[0])

	s.DoJSON("GET", "/api/latest/fleet/commands/results", nil, http.StatusOK, &getMDMCmdResp, "command_uuid", cmdFourUUID)
	require.Len(t, getMDMCmdResp.Results, 1)
	require.NotZero(t, getMDMCmdResp.Results[0].UpdatedAt)
	getMDMCmdResp.Results[0].UpdatedAt = time.Time{}
	require.Equal(t, &fleet.MDMCommandResult{
		HostUUID:    orbitHost.UUID,
		CommandUUID: cmdFourUUID,
		Status:      "200",
		RequestType: "./Vendor/MSFT/WiFi/Profile/MyNetwork/WlanXml",
		Result:      getCommandFullResult(cmdFourUUID),
		Hostname:    "TestIntegrationsMDM/TestWindowsMDMh1.local",
		Payload:     commandFour.RawCommand,
	}, getMDMCmdResp.Results[0])
}

func (s *integrationMDMTestSuite) TestWindowsMDMCommandWithSecret() {
	t := s.T()
	orbitHost, d := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	secretValue := "abcd1234"
	req := secretVariablesRequest{
		SecretVariables: []fleet.SecretVariable{
			{
				Name:  "FLEET_SECRET_DATA",
				Value: secretValue,
			},
		},
	}
	secretResp := secretVariablesResponse{}
	s.DoJSON("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusOK, &secretResp)

	cmdOneUUID := uuid.New().String()
	commandOne := &fleet.MDMWindowsCommand{
		CommandUUID: cmdOneUUID,
		RawCommand: []byte(fmt.Sprintf(`
                     <Exec>
                       <CmdID>%s</CmdID>
                       <Item>
                         <Target>
                           <LocURI>./Device/Vendor/MSFT/Reboot/RebootNow</LocURI>
                         </Target>
                         <Meta>
                           <Format xmlns="syncml:metinf">null</Format>
                           <Type>text/plain</Type>
                         </Meta>
                         <Data>$FLEET_SECRET_DATA</Data>
                       </Item>
                     </Exec>
		`, cmdOneUUID)),
		TargetLocURI: "./Device/Vendor/MSFT/Reboot/RebootNow",
	}
	err := s.ds.MDMWindowsInsertCommandForHosts(context.Background(), []string{orbitHost.UUID}, commandOne)
	require.NoError(t, err)

	cmds, err := d.StartManagementSession()
	require.NoError(t, err)
	// 2 Status + 1 Exec
	require.Len(t, cmds, 3)
	receivedCmd := cmds[cmdOneUUID]
	require.NotNil(t, receivedCmd)
	require.Equal(t, receivedCmd.Verb, fleet.CmdExec)
	require.Len(t, receivedCmd.Cmd.Items, 1)
	require.EqualValues(t, "./Device/Vendor/MSFT/Reboot/RebootNow", *receivedCmd.Cmd.Items[0].Target)
	assert.EqualValues(t, secretValue, receivedCmd.Cmd.Items[0].Data.Content)

	msgID, err := d.GetCurrentMsgID()
	require.NoError(t, err)

	d.AppendResponse(fleet.SyncMLCmd{
		XMLName: xml.Name{Local: fleet.CmdStatus},
		MsgRef:  &msgID,
		CmdRef:  &cmdOneUUID,
		Cmd:     ptr.String("Exec"),
		Data:    ptr.String("200"),
		Items:   nil,
		CmdID:   fleet.CmdID{Value: uuid.NewString()},
	})
	cmds, err = d.SendResponse()
	require.NoError(t, err)
	// the ack of the message should be the only returned command
	require.Len(t, cmds, 1)

	var getMDMCmdResp getMDMCommandResultsResponse
	s.DoJSON("GET", "/api/latest/fleet/commands/results", nil, http.StatusOK, &getMDMCmdResp, "command_uuid", cmdOneUUID)
	require.Len(t, getMDMCmdResp.Results, 1)
	// The secret value should not be exposed via the regular API.
	assert.NotContains(t, string(getMDMCmdResp.Results[0].Payload), secretValue)
	assert.Contains(t, string(getMDMCmdResp.Results[0].Payload), "$FLEET_SECRET_DATA")
}

func (s *integrationMDMTestSuite) TestWindowsAutomaticEnrollmentCommands() {
	t := s.T()
	ctx := context.Background()

	// define a global enroll secret
	err := s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: t.Name()}})
	require.NoError(t, err)

	azureMail := "foo.bar.baz@example.com"
	d := mdmtest.NewTestMDMClientWindowsAutomatic(s.server.URL, azureMail)
	require.NoError(t, d.Enroll())

	checkinAndAck := func(expectFleetdCmds bool) {
		cmds, err := d.StartManagementSession()
		require.NoError(t, err)

		if !expectFleetdCmds {
			// receives only the 2 status commands
			require.Len(t, cmds, 2)
			for _, c := range cmds {
				require.Equal(t, "Status", c.Verb, c)
			}
			return
		}

		// 2 status + 2 commands to install fleetd
		require.Len(t, cmds, 4)
		var fleetdAddCmd, fleetdExecCmd fleet.ProtoCmdOperation
		for _, c := range cmds {
			switch c.Verb {
			case "Add":
				fleetdAddCmd = c
			case "Exec":
				fleetdExecCmd = c
			}
		}
		require.Equal(t, syncml.FleetdWindowsInstallerGUID, fleetdAddCmd.Cmd.GetTargetURI())
		require.Equal(t, syncml.FleetdWindowsInstallerGUID, fleetdExecCmd.Cmd.GetTargetURI())
		require.Len(t, fleetdExecCmd.Cmd.Items, 1)

		var installJob struct {
			Product struct {
				ContentURL string `xml:"Download>ContentURLList>ContentURL"`
				FileHash   string `xml:"Validation>FileHash"`
			} `xml:"Product"`
		}
		err = xml.Unmarshal([]byte(fleetdExecCmd.Cmd.Items[0].Data.Content), &installJob)
		require.NoError(t, err)
		require.Equal(t, s.mockedDownloadFleetdmMeta.MSIURL, installJob.Product.ContentURL)
		require.Equal(t, s.mockedDownloadFleetdmMeta.MSISha256, installJob.Product.FileHash)

		// reply with success for both commands
		msgID, err := d.GetCurrentMsgID()
		require.NoError(t, err)

		d.AppendResponse(fleet.SyncMLCmd{
			XMLName: xml.Name{Local: fleet.CmdStatus},
			MsgRef:  &msgID,
			CmdRef:  &fleetdAddCmd.Cmd.CmdID.Value,
			Cmd:     &fleetdAddCmd.Verb,
			Data:    ptr.String("200"),
			Items:   nil,
			CmdID:   fleet.CmdID{Value: uuid.NewString()},
		})
		d.AppendResponse(fleet.SyncMLCmd{
			XMLName: xml.Name{Local: fleet.CmdStatus},
			MsgRef:  &msgID,
			CmdRef:  &fleetdExecCmd.Cmd.CmdID.Value,
			Cmd:     &fleetdExecCmd.Verb,
			Data:    ptr.String("200"),
			Items:   nil,
			CmdID:   fleet.CmdID{Value: uuid.NewString()},
		})
		cmds, err = d.SendResponse()
		require.NoError(t, err)

		// the ack of the message should be the only returned command
		require.Len(t, cmds, 1)
	}

	// start a management session, will receive the install fleetd commands
	checkinAndAck(true)

	// start a new management session again, Fleetd is not reported as installed
	// so it receives the commands again
	checkinAndAck(true)

	// simulate fleetd installed and enrolled
	host := createOrbitEnrolledHost(t, "windows", "h1", s.ds)
	err = s.ds.UpdateMDMWindowsEnrollmentsHostUUID(ctx, host.UUID, d.DeviceID)
	require.NoError(t, err)
	err = s.ds.SetOrUpdateHostOrbitInfo(ctx, host.ID, "1.23", sql.NullString{}, sql.NullBool{})
	require.NoError(t, err)

	// start a new management session again, Fleetd is reported as installed so
	// it does not receive the commands
	checkinAndAck(false)
}

func (s *integrationMDMTestSuite) TestValidManagementUnenrollRequest() {
	t := s.T()

	// Target Endpoint URL for the management endpoint
	targetEndpointURL := microsoft_mdm.MDE2ManagementPath

	// Target DeviceID to use
	deviceID := "DB257C3A08778F4FB61E2749066C1F27"

	// Inserting new device
	enrolledDevice := &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            deviceID,
		MDMHardwareID:          uuid.New().String() + uuid.New().String(),
		MDMDeviceState:         uuid.New().String(),
		MDMDeviceType:          "CIMClient_Windows",
		MDMDeviceName:          "DESKTOP-1C3ARC1",
		MDMEnrollType:          "ProgrammaticEnrollment",
		MDMEnrollUserID:        "upn@domain.com",
		MDMEnrollProtoVersion:  "5.0",
		MDMEnrollClientVersion: "10.0.19045.2965",
		MDMNotInOOBE:           false,
	}

	err := s.ds.MDMWindowsInsertEnrolledDevice(context.Background(), enrolledDevice)
	require.NoError(t, err)

	// Checking if device was enrolled
	_, err = s.ds.MDMWindowsGetEnrolledDeviceWithDeviceID(context.Background(), deviceID)
	require.NoError(t, err)

	// Preparing the SyncML unenroll request
	requestBytes, err := s.newSyncMLUnenrollMsg(deviceID, targetEndpointURL)
	require.NoError(t, err)

	resp := s.DoRaw("POST", targetEndpointURL, requestBytes, http.StatusOK)

	// Checking that Command error code was updated

	// Checking response headers
	require.Contains(t, resp.Header["Content-Type"], syncml.SyncMLContentType)

	// Read response data
	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Checking if response can be unmarshalled to an golang type
	var xmlType interface{}
	err = xml.Unmarshal(resBytes, &xmlType)
	require.NoError(t, err)

	// Checking if device was unenrolled
	_, err = s.ds.MDMWindowsGetEnrolledDeviceWithDeviceID(context.Background(), deviceID)
	require.True(t, fleet.IsNotFound(err))
}

func (s *integrationMDMTestSuite) TestRunMDMCommands() {
	t := s.T()

	// create a Windows host enrolled in MDM
	enrolledWindows, _ := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	// create an unenrolled Windows host
	unenrolledWindows := createOrbitEnrolledHost(t, "windows", "h2", s.ds)

	// create an enrolled and unenrolled macOS host
	enrolledMac, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	unenrolledMac := createOrbitEnrolledHost(t, "darwin", "h4", s.ds)

	macRawCmd := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Command</key>
    <dict>
        <key>RequestType</key>
        <string>ShutDownDevice</string>
    </dict>
    <key>CommandUUID</key>
    <string>0001_ShutDownDevice</string>
</dict>
</plist>`

	winRawCmd := `
<Exec>
	<CmdID>11</CmdID>
	<Item>
		<Target>
			<LocURI>./SetValues</LocURI>
		</Target>
		<Meta>
			<Format xmlns="syncml:metinf">chr</Format>
			<Type xmlns="syncml:metinf">text/plain</Type>
		</Meta>
		<Data>NamedValuesList=MinPasswordLength,8;</Data>
	</Item>
</Exec>
`

	var runResp runMDMCommandResponse

	// no host provided
	s.DoJSON("POST", "/api/latest/fleet/commands/run", &runMDMCommandRequest{
		Command: base64.StdEncoding.EncodeToString([]byte(macRawCmd)),
	}, http.StatusNotFound, &runResp)

	// mix of mdm and non-mdm hosts
	s.DoJSON("POST", "/api/latest/fleet/commands/run", &runMDMCommandRequest{
		Command:   base64.StdEncoding.EncodeToString([]byte(macRawCmd)),
		HostUUIDs: []string{enrolledMac.UUID, unenrolledMac.UUID},
	}, http.StatusPreconditionFailed, &runResp)
	s.DoJSON("POST", "/api/latest/fleet/commands/run", &runMDMCommandRequest{
		Command:   base64.StdEncoding.EncodeToString([]byte(winRawCmd)),
		HostUUIDs: []string{enrolledWindows.UUID, unenrolledWindows.UUID},
	}, http.StatusPreconditionFailed, &runResp)

	// mix of windows and macos hosts
	s.DoJSON("POST", "/api/latest/fleet/commands/run", &runMDMCommandRequest{
		Command:   base64.StdEncoding.EncodeToString([]byte(macRawCmd)),
		HostUUIDs: []string{enrolledMac.UUID, enrolledWindows.UUID},
	}, http.StatusUnprocessableEntity, &runResp)

	// windows only, invalid command
	res := s.Do("POST", "/api/latest/fleet/commands/run", &runMDMCommandRequest{
		Command:   base64.StdEncoding.EncodeToString([]byte(macRawCmd)),
		HostUUIDs: []string{enrolledWindows.UUID},
	}, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "You can run only <Exec> command type")

	// macOS only, invalid command
	res = s.Do("POST", "/api/latest/fleet/commands/run", &runMDMCommandRequest{
		Command:   base64.StdEncoding.EncodeToString([]byte(winRawCmd)),
		HostUUIDs: []string{enrolledMac.UUID},
	}, http.StatusUnsupportedMediaType)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "unable to decode plist command")

	// valid windows
	runResp = runMDMCommandResponse{}
	s.DoJSON("POST", "/api/latest/fleet/commands/run", &runMDMCommandRequest{
		Command:   base64.StdEncoding.EncodeToString([]byte(winRawCmd)),
		HostUUIDs: []string{enrolledWindows.UUID},
	}, http.StatusOK, &runResp)
	require.NotEmpty(t, runResp.CommandUUID)
	require.Equal(t, "windows", runResp.Platform)
	require.Equal(t, "./SetValues", runResp.RequestType)

	// valid macOS
	runResp = runMDMCommandResponse{}
	s.DoJSON("POST", "/api/latest/fleet/commands/run", &runMDMCommandRequest{
		Command:   base64.StdEncoding.EncodeToString([]byte(macRawCmd)),
		HostUUIDs: []string{enrolledMac.UUID},
	}, http.StatusOK, &runResp)
	require.NotEmpty(t, runResp.CommandUUID)
	require.Equal(t, "darwin", runResp.Platform)
	require.Equal(t, "ShutDownDevice", runResp.RequestType)
}

func (s *integrationMDMTestSuite) TestUpdateMDMWindowsEnrollmentsHostUUID() {
	ctx := context.Background()
	t := s.T()

	// simulate device that is MDM enrolled before fleetd is installed
	d := fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            "test-device-id",
		MDMHardwareID:          "test-hardware-id",
		MDMDeviceState:         "ds",
		MDMDeviceType:          "dt",
		MDMDeviceName:          "dn",
		MDMEnrollType:          "et",
		MDMEnrollUserID:        "euid",
		MDMEnrollProtoVersion:  "epv",
		MDMEnrollClientVersion: "ecv",
		MDMNotInOOBE:           false,
		HostUUID:               "", // empty host uuid when created
	}
	require.NoError(t, s.ds.MDMWindowsInsertEnrolledDevice(ctx, &d))

	gotDevice, err := s.ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, d.MDMDeviceID)
	require.NoError(t, err)
	require.Empty(t, gotDevice.HostUUID)

	// create an enroll secret
	secret := uuid.New().String()
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: secret}},
		},
	}, http.StatusOK, &applyResp)

	// simulate fleetd installed and enrolled
	var resp EnrollOrbitResponse
	hostUUID := uuid.New().String()
	hostSerial := "test-host-serial"
	s.DoJSON("POST", "/api/fleet/orbit/enroll", EnrollOrbitRequest{
		EnrollSecret:   secret,
		HardwareUUID:   hostUUID,
		HardwareSerial: hostSerial,
		Platform:       "windows",
	}, http.StatusOK, &resp)
	require.NotEmpty(t, resp.OrbitNodeKey)

	gotDevice, err = s.ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, d.MDMDeviceID)
	require.NoError(t, err)
	require.Empty(t, gotDevice.HostUUID)

	// simulate first report osquery host details
	require.NoError(t, s.ds.UpdateMDMWindowsEnrollmentsHostUUID(ctx, hostUUID, d.MDMDeviceID))

	// check that the host uuid was updated
	gotDevice, err = s.ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, d.MDMDeviceID)
	require.NoError(t, err)
	require.NotEmpty(t, gotDevice.HostUUID)
	require.Equal(t, hostUUID, gotDevice.HostUUID)
}

func (s *integrationMDMTestSuite) TestBitLockerEnforcementNotifications() {
	t := s.T()
	ctx := context.Background()
	windowsHost := createOrbitEnrolledHost(t, "windows", t.Name(), s.ds)

	checkNotification := func(want bool) {
		resp := orbitGetConfigResponse{}
		s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *windowsHost.OrbitNodeKey)), http.StatusOK, &resp)
		require.Equal(t, want, resp.Notifications.EnforceBitLockerEncryption)
	}

	// notification is false by default
	checkNotification(false)

	// enroll the host into Fleet MDM
	encodedBinToken, err := fleet.GetEncodedBinarySecurityToken(fleet.WindowsMDMProgrammaticEnrollmentType, *windowsHost.OrbitNodeKey)
	require.NoError(t, err)
	requestBytes, err := s.newSecurityTokenMsg(encodedBinToken, true, false)
	require.NoError(t, err)
	s.DoRaw("POST", microsoft_mdm.MDE2EnrollPath, requestBytes, http.StatusOK)

	// simulate osquery checking in and updating this info
	// TODO: should we automatically fill these fields on MDM enrollment?
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), windowsHost.ID, false, true, "https://example.com", true, fleet.WellKnownMDMFleet, ""))

	// notification is still false
	checkNotification(false)

	// configure disk encryption for the global team
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "mdm": { "macos_settings": { "enable_disk_encryption": true } } }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)

	// host still doesn't get the notification because we don't have disk
	// encryption information yet.
	checkNotification(false)

	// host has disk encryption off, gets the notification
	require.NoError(t, s.ds.SetOrUpdateHostDisksEncryption(context.Background(), windowsHost.ID, false))
	checkNotification(true)

	// host has disk encryption on, we don't have disk encryption info. Gets the notification
	require.NoError(t, s.ds.SetOrUpdateHostDisksEncryption(context.Background(), windowsHost.ID, true))
	checkNotification(true)

	// host has disk encryption on, we don't know if the key is decriptable. Gets the notification
	err = s.ds.SetOrUpdateHostDiskEncryptionKey(ctx, windowsHost.ID, "test-key", "", nil)
	require.NoError(t, err)
	checkNotification(true)

	// host has disk encryption on, the key is not decryptable by fleet. Gets the notification
	err = s.ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{windowsHost.ID}, false, time.Now())
	require.NoError(t, err)
	checkNotification(true)

	// host has disk encryption on, the disk was encrypted by fleet. Doesn't get the notification
	err = s.ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{windowsHost.ID}, true, time.Now())
	require.NoError(t, err)
	checkNotification(false)

	// create a new team
	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name(),
		Description: "desc",
	})
	require.NoError(t, err)
	// add the host to the team
	err = s.ds.AddHostsToTeam(context.Background(), &tm.ID, []uint{windowsHost.ID})
	require.NoError(t, err)

	// notification is false now since the team doesn't have disk encryption enabled
	checkNotification(false)

	// enable disk encryption on the team
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: tm.Name,
		MDM: fleet.TeamSpecMDM{
			EnableDiskEncryption: optjson.SetBool(true),
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	// host gets the notification
	checkNotification(true)

	// host has disk encryption off, gets the notification
	require.NoError(t, s.ds.SetOrUpdateHostDisksEncryption(context.Background(), windowsHost.ID, false))
	checkNotification(true)

	// host has disk encryption on, we don't have disk encryption info. Gets the notification
	require.NoError(t, s.ds.SetOrUpdateHostDisksEncryption(context.Background(), windowsHost.ID, true))
	checkNotification(true)

	// host has disk encryption on, we don't know if the key is decriptable. Gets the notification
	err = s.ds.SetOrUpdateHostDiskEncryptionKey(ctx, windowsHost.ID, "test-key", "", nil)
	require.NoError(t, err)
	checkNotification(true)

	// host has disk encryption on, the key is not decryptable by fleet. Gets the notification
	err = s.ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{windowsHost.ID}, false, time.Now())
	require.NoError(t, err)
	checkNotification(true)

	// host has disk encryption on, the disk was encrypted by fleet. Doesn't get the notification
	err = s.ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{windowsHost.ID}, true, time.Now())
	require.NoError(t, err)
	checkNotification(false)
}

func (s *integrationMDMTestSuite) TestHostDiskEncryptionKey() {
	t := s.T()
	ctx := context.Background()

	host := createOrbitEnrolledHost(t, "windows", "h1", s.ds)

	// turn on disk encryption for the global team
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "mdm": { "enable_disk_encryption": true } }`), http.StatusOK, &acResp)
	assert.True(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value)

	// try to call the endpoint while the host is not MDM-enrolled
	res := s.Do("POST", "/api/fleet/orbit/disk_encryption_key", orbitPostDiskEncryptionKeyRequest{
		OrbitNodeKey:  *host.OrbitNodeKey,
		EncryptionKey: []byte("WILL-FAIL"),
	}, http.StatusBadRequest)
	msg := extractServerErrorText(res.Body)
	require.Contains(t, msg, "host is not enrolled with fleet")

	// enroll it in fleet
	mdmDevice := mdmtest.NewTestMDMClientWindowsProgramatic(s.server.URL, *host.OrbitNodeKey)
	err := mdmDevice.Enroll()
	require.NoError(t, err)
	err = s.ds.SetOrUpdateMDMData(ctx, host.ID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, "")
	require.NoError(t, err)

	// set its encryption key
	s.Do("POST", "/api/fleet/orbit/disk_encryption_key", orbitPostDiskEncryptionKeyRequest{
		OrbitNodeKey:  *host.OrbitNodeKey,
		EncryptionKey: []byte("ABC"),
	}, http.StatusNoContent)

	hdek, err := s.ds.GetHostDiskEncryptionKey(ctx, host.ID)
	require.NoError(t, err)
	require.NotNil(t, hdek.Decryptable)
	require.True(t, *hdek.Decryptable)

	// mark it as non-server
	err = s.ds.SetOrUpdateMDMData(ctx, host.ID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, "")
	require.NoError(t, err)

	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.Nil(t, hostResp.Host.DiskEncryptionEnabled) // the disk encryption status of the host is not set by the orbit request
	require.NotNil(t, hostResp.Host.MDM.OSSettings)
	require.NotNil(t, hostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, fleet.DiskEncryptionEnforcing, *hostResp.Host.MDM.OSSettings.DiskEncryption.Status) // still pending because disk encryption status is not set
	require.Equal(t, "", hostResp.Host.MDM.OSSettings.DiskEncryption.Detail)

	// the key is encrypted the same way as the macOS keys (except with the WSTEP
	// certificate), so it can be decrypted using the same decryption function.
	wstepCert, _, _, err := s.fleetCfg.MDM.MicrosoftWSTEP()
	require.NoError(t, err)
	decrypted, err := servermdm.DecryptBase64CMS(hdek.Base64Encrypted, wstepCert.Leaf, wstepCert.PrivateKey)
	require.NoError(t, err)
	require.Equal(t, "ABC", string(decrypted))

	// set it with a client error
	s.Do("POST", "/api/fleet/orbit/disk_encryption_key", orbitPostDiskEncryptionKeyRequest{
		OrbitNodeKey: *host.OrbitNodeKey,
		ClientError:  "fail",
	}, http.StatusNoContent)

	hdek, err = s.ds.GetHostDiskEncryptionKey(ctx, host.ID)
	require.NoError(t, err)
	require.Nil(t, hdek.Decryptable)
	require.Empty(t, hdek.Base64Encrypted)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.Nil(t, hostResp.Host.DiskEncryptionEnabled) // the disk encryption status of the host is not set by the orbit request
	require.NotNil(t, hostResp.Host.MDM.OSSettings)
	require.NotNil(t, hostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, fleet.DiskEncryptionFailed, *hostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, "fail", hostResp.Host.MDM.OSSettings.DiskEncryption.Detail)

	// set a different key
	s.Do("POST", "/api/fleet/orbit/disk_encryption_key", orbitPostDiskEncryptionKeyRequest{
		OrbitNodeKey:  *host.OrbitNodeKey,
		EncryptionKey: []byte("DEF"),
	}, http.StatusNoContent)

	hdek, err = s.ds.GetHostDiskEncryptionKey(ctx, host.ID)
	require.NoError(t, err)
	require.NotNil(t, hdek.Decryptable)
	require.True(t, *hdek.Decryptable)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.Nil(t, hostResp.Host.DiskEncryptionEnabled) // the disk encryption status of the host is not set by the orbit request
	require.NotNil(t, hostResp.Host.MDM.OSSettings)
	require.NotNil(t, hostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, fleet.DiskEncryptionEnforcing, *hostResp.Host.MDM.OSSettings.DiskEncryption.Status) // still pending because disk encryption status is not set
	require.Equal(t, "", hostResp.Host.MDM.OSSettings.DiskEncryption.Detail)

	decrypted, err = servermdm.DecryptBase64CMS(hdek.Base64Encrypted, wstepCert.Leaf, wstepCert.PrivateKey)
	require.NoError(t, err)
	require.Equal(t, "DEF", string(decrypted))

	// report host disks as encrypted
	err = s.ds.SetOrUpdateHostDisksEncryption(ctx, host.ID, true)
	require.NoError(t, err)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.True(t, *hostResp.Host.DiskEncryptionEnabled)
	require.NotNil(t, hostResp.Host.MDM.OSSettings)
	require.NotNil(t, hostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, fleet.DiskEncryptionVerified, *hostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, "", hostResp.Host.MDM.OSSettings.DiskEncryption.Detail)
}

// ///////////////////////////////////////////////////////////////////////////
// Common MDM config test

func (s *integrationMDMTestSuite) TestMDMEnabledAndConfigured() {
	t := s.T()
	ctx := context.Background()

	appConfig, err := s.ds.AppConfig(ctx)
	originalCopy := appConfig.Copy()
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, s.ds.SaveAppConfig(ctx, originalCopy))
	})

	checkAppConfig := func(t *testing.T, mdmEnabled, winEnabled bool) appConfigResponse {
		acResp := appConfigResponse{}
		s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
		require.True(t, acResp.AppConfig.MDM.AppleBMEnabledAndConfigured)
		require.Equal(t, mdmEnabled, acResp.AppConfig.MDM.EnabledAndConfigured)
		require.Equal(t, winEnabled, acResp.AppConfig.MDM.WindowsEnabledAndConfigured)
		return acResp
	}

	compareMacOSSetupValues := (func(t *testing.T, got fleet.MacOSSetup, want fleet.MacOSSetup) {
		require.Equal(t, want.BootstrapPackage.Value, got.BootstrapPackage.Value)
		require.Equal(t, want.MacOSSetupAssistant.Value, got.MacOSSetupAssistant.Value)
		require.Equal(t, want.EnableEndUserAuthentication, got.EnableEndUserAuthentication)
	})

	insertBootstrapPackageAndSetupAssistant := func(t *testing.T, teamID *uint) {
		var tmID uint
		if teamID != nil {
			tmID = *teamID
		}

		// cleanup any residual bootstrap package
		_ = s.ds.DeleteMDMAppleBootstrapPackage(ctx, tmID)

		// add new bootstrap package
		require.NoError(t, s.ds.InsertMDMAppleBootstrapPackage(ctx, &fleet.MDMAppleBootstrapPackage{
			TeamID: tmID,
			Name:   "foo",
			Token:  uuid.New().String(),
			Bytes:  []byte("foo"),
			Sha256: []byte("foo-sha256"),
		}, nil))

		// add new setup assistant
		_, err := s.ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{
			TeamID:  teamID,
			Name:    "bar",
			Profile: []byte("{}"),
		})
		require.NoError(t, err)
	}

	// TODO: Some global MDM config settings don't have MDMEnabledAndConfigured or
	// WindowsMDMEnabledAndConfigured validations currently. Either add validations
	// and test them or test absence of validation.
	t.Run("apply app config spec", func(t *testing.T) {
		t.Run("disk encryption", func(t *testing.T) {
			t.Cleanup(func() {
				require.NoError(t, s.ds.SaveAppConfig(ctx, appConfig))
			})

			acResp := checkAppConfig(t, true, true)
			require.False(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value) // disabled by default

			// initialize our test app config
			ac := appConfig.Copy()
			ac.AgentOptions = nil

			// enable disk encryption
			ac.MDM.EnableDiskEncryption = optjson.SetBool(true)
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, true, true)                           // both mac and windows mdm enabled
			require.True(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value) // enabled

			// directly set MDM.EnabledAndConfigured to false
			ac.MDM.EnabledAndConfigured = false
			require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
			acResp = checkAppConfig(t, false, true)                          // only windows mdm enabled
			require.True(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value) // disabling mdm doesn't change disk encryption

			// making an unrelated change should not cause validation error
			ac.OrgInfo.OrgName = "f1337"
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, true)                          // only windows mdm enabled
			require.True(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value) // no change
			require.Equal(t, "f1337", acResp.AppConfig.OrgInfo.OrgName)

			// disabling disk encryption doesn't cause validation error
			ac.MDM.EnableDiskEncryption = optjson.SetBool(false)
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, true)                           // only windows mdm enabled
			require.False(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value) // disabled
			require.Equal(t, "f1337", acResp.AppConfig.OrgInfo.OrgName)

			// enabling disk encryption doesn't cause validation error
			ac.MDM.EnableDiskEncryption = optjson.SetBool(true)
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, true)                          // only windows mdm enabled
			require.True(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value) // enabled

			// directly set MDM.WindowsEnabledAndConfigured to false
			ac.MDM.WindowsEnabledAndConfigured = false
			require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
			acResp = checkAppConfig(t, false, false)                         // both mac and windows mdm disabled
			require.True(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value) // disabling mdm doesn't change disk encryption

			// disabling disk encryption doesn't cause validation error
			ac.MDM.EnableDiskEncryption = optjson.SetBool(false)
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, false)                          // no MDM enabled
			require.False(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value) // disabled
			require.Equal(t, "f1337", acResp.AppConfig.OrgInfo.OrgName)

			// enabling disk encryption doesn't cause validation error
			ac.MDM.EnableDiskEncryption = optjson.SetBool(true)
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, false)                         // no MDM enabled
			require.True(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value) // enabled

			// changing unrelated config doesn't cause validation error
			ac.OrgInfo.OrgName = "f1338"
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, false)                         // both mac and windows mdm disabled
			require.True(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value) // no change
			require.Equal(t, "f1338", acResp.AppConfig.OrgInfo.OrgName)
		})

		t.Run("macos setup", func(t *testing.T) {
			t.Cleanup(func() {
				require.NoError(t, s.ds.SaveAppConfig(ctx, appConfig))
			})

			acResp := checkAppConfig(t, true, true)
			compareMacOSSetupValues(t, fleet.MacOSSetup{}, acResp.AppConfig.MDM.MacOSSetup) // disabled by default

			// initialize our test app config
			ac := appConfig.Copy()
			ac.AgentOptions = nil
			ac.MDM.EndUserAuthentication = fleet.MDMEndUserAuthentication{
				SSOProviderSettings: fleet.SSOProviderSettings{
					EntityID:    "sso-provider",
					IDPName:     "sso-provider",
					MetadataURL: "https://sso-provider.example.com/metadata",
				},
			}

			// add db records for bootstrap package and setup assistant
			insertBootstrapPackageAndSetupAssistant(t, nil)

			// enable MacOSSetup options
			ac.MDM.MacOSSetup = fleet.MacOSSetup{
				BootstrapPackage:            optjson.SetString("foo"),
				EnableEndUserAuthentication: true,
				MacOSSetupAssistant:         optjson.SetString("bar"),
			}
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, true, true)                               // both mac and windows mdm enabled
			compareMacOSSetupValues(t, acResp.MDM.MacOSSetup, ac.MDM.MacOSSetup) // applied

			// directly set MDM.EnabledAndConfigured to false
			ac.MDM.EnabledAndConfigured = false
			require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
			acResp = checkAppConfig(t, false, true)                              // only windows mdm enabled
			compareMacOSSetupValues(t, acResp.MDM.MacOSSetup, ac.MDM.MacOSSetup) // still applied

			// making an unrelated change should not cause validation error
			ac.OrgInfo.OrgName = "f1337"
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, true)                              // only windows mdm enabled
			compareMacOSSetupValues(t, acResp.MDM.MacOSSetup, ac.MDM.MacOSSetup) // still applied
			require.Equal(t, "f1337", acResp.AppConfig.OrgInfo.OrgName)

			// disabling doesn't cause validation error
			ac.MDM.MacOSSetup = fleet.MacOSSetup{
				BootstrapPackage:            optjson.SetString(""),
				EnableEndUserAuthentication: false,
				MacOSSetupAssistant:         optjson.SetString(""),
			}
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, true)                              // only windows mdm enabled
			compareMacOSSetupValues(t, acResp.MDM.MacOSSetup, ac.MDM.MacOSSetup) // applied
			require.Equal(t, "f1337", acResp.AppConfig.OrgInfo.OrgName)

			// bootstrap package and setup assistant were removed so reinsert records for next test
			insertBootstrapPackageAndSetupAssistant(t, nil)

			// enable MacOSSetup options fails because only Windows is enabled.
			ac.MDM.MacOSSetup = fleet.MacOSSetup{
				BootstrapPackage:            optjson.SetString("foo"),
				EnableEndUserAuthentication: true,
				MacOSSetupAssistant:         optjson.SetString("bar"),
			}
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusUnprocessableEntity, &acResp)
			acResp = checkAppConfig(t, false, true) // only windows enabled

			// directly set MDM.EnabledAndConfigured to true and windows to false
			ac.MDM.EnabledAndConfigured = true
			ac.MDM.WindowsEnabledAndConfigured = false
			require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
			acResp = checkAppConfig(t, true, false)                              // mac enabled, windows disabled
			compareMacOSSetupValues(t, acResp.MDM.MacOSSetup, ac.MDM.MacOSSetup) // directly applied

			// changing unrelated config doesn't cause validation error
			ac.OrgInfo.OrgName = "f1338"
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, true, false)                              // mac enabled, windows disabled
			compareMacOSSetupValues(t, acResp.MDM.MacOSSetup, ac.MDM.MacOSSetup) // no change
			require.Equal(t, "f1338", acResp.AppConfig.OrgInfo.OrgName)

			// disabling doesn't cause validation error
			ac.MDM.MacOSSetup = fleet.MacOSSetup{
				BootstrapPackage:            optjson.SetString(""),
				EnableEndUserAuthentication: false,
				MacOSSetupAssistant:         optjson.SetString(""),
			}
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, true, false)                              // only windows mdm enabled
			compareMacOSSetupValues(t, acResp.MDM.MacOSSetup, ac.MDM.MacOSSetup) // applied

			// bootstrap package and setup assistant were removed so reinsert records for next test
			insertBootstrapPackageAndSetupAssistant(t, nil)

			// enable MacOSSetup options succeeds because only Windows is disabled
			ac.MDM.MacOSSetup = fleet.MacOSSetup{
				BootstrapPackage:            optjson.SetString("foo"),
				EnableEndUserAuthentication: true,
				MacOSSetupAssistant:         optjson.SetString("bar"),
			}
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, true, false)                              // only windows enabled
			compareMacOSSetupValues(t, acResp.MDM.MacOSSetup, ac.MDM.MacOSSetup) // applied

			// directly set MDM.EnabledAndConfigured to false
			ac.MDM.EnabledAndConfigured = false
			require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
			acResp = checkAppConfig(t, false, false)                             // both mac and windows mdm disabled
			compareMacOSSetupValues(t, acResp.MDM.MacOSSetup, ac.MDM.MacOSSetup) // still applied

			// changing unrelated config doesn't cause validation error
			ac.OrgInfo.OrgName = "f1339"
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, false)                             // both disabled
			compareMacOSSetupValues(t, acResp.MDM.MacOSSetup, ac.MDM.MacOSSetup) // no change
			require.Equal(t, "f1339", acResp.AppConfig.OrgInfo.OrgName)

			// setting macos setup empty values doesn't cause validation error when mdm is disabled
			ac.MDM.MacOSSetup = fleet.MacOSSetup{
				BootstrapPackage:            optjson.SetString(""),
				EnableEndUserAuthentication: false,
				MacOSSetupAssistant:         optjson.SetString(""),
			}
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, false)                             // both disabled
			compareMacOSSetupValues(t, acResp.MDM.MacOSSetup, ac.MDM.MacOSSetup) // applied

			// setting macos setup to non-empty values fails because mdm disabled
			ac.MDM.MacOSSetup = fleet.MacOSSetup{
				BootstrapPackage:            optjson.SetString("foo"),
				EnableEndUserAuthentication: true,
				MacOSSetupAssistant:         optjson.SetString("bar"),
			}
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusUnprocessableEntity, &acResp)
			acResp = checkAppConfig(t, false, false) // both disabled
		})

		t.Run("custom settings", func(t *testing.T) {
			t.Cleanup(func() {
				require.NoError(t, s.ds.SaveAppConfig(ctx, appConfig))
			})

			// initialize our test app config
			ac := appConfig.Copy()
			ac.AgentOptions = nil
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{}
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{})
			require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
			acResp := checkAppConfig(t, true, true)
			require.Empty(t, acResp.MDM.MacOSSettings.CustomSettings)
			require.Empty(t, acResp.MDM.WindowsSettings.CustomSettings.Value)

			// add custom settings
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{{Path: "foo"}, {Path: "bar"}}
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "baz"}, {Path: "zab"}})
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, true, true)                                                                                 // both mac and windows mdm enabled
			require.ElementsMatch(t, acResp.MDM.MacOSSettings.CustomSettings, ac.MDM.MacOSSettings.CustomSettings)                 // applied
			require.ElementsMatch(t, acResp.MDM.WindowsSettings.CustomSettings.Value, ac.MDM.WindowsSettings.CustomSettings.Value) // applied

			// directly set MDM.EnabledAndConfigured to false
			ac.MDM.EnabledAndConfigured = false
			require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
			acResp = checkAppConfig(t, false, true)                                                                                // only windows mdm enabled
			require.ElementsMatch(t, acResp.MDM.MacOSSettings.CustomSettings, ac.MDM.MacOSSettings.CustomSettings)                 // still applied
			require.ElementsMatch(t, acResp.MDM.WindowsSettings.CustomSettings.Value, ac.MDM.WindowsSettings.CustomSettings.Value) // still applied

			// making an unrelated change should not cause validation error
			ac.OrgInfo.OrgName = "f1337"
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, true)                                                                                // only windows mdm enabled
			require.ElementsMatch(t, acResp.MDM.MacOSSettings.CustomSettings, ac.MDM.MacOSSettings.CustomSettings)                 // still applied
			require.ElementsMatch(t, acResp.MDM.WindowsSettings.CustomSettings.Value, ac.MDM.WindowsSettings.CustomSettings.Value) // still applied
			require.Equal(t, "f1337", acResp.AppConfig.OrgInfo.OrgName)

			// remove custom settings
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{}
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{})
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, true) // only windows mdm enabled
			require.Empty(t, acResp.MDM.MacOSSettings.CustomSettings)
			require.Empty(t, acResp.MDM.WindowsSettings.CustomSettings.Value)

			// add custom macOS settings fails because only windows is enabled
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{{Path: "foo"}, {Path: "bar"}}
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusUnprocessableEntity, &acResp)
			acResp = checkAppConfig(t, false, true) // only windows enabled
			require.Empty(t, acResp.MDM.MacOSSettings.CustomSettings)
			require.Empty(t, acResp.MDM.WindowsSettings.CustomSettings.Value)

			// add custom Windows settings suceeds because only macOS is disabled
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{}
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "baz"}, {Path: "zab"}})
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, true)                                                                                // only windows mdm enabled
			require.ElementsMatch(t, acResp.MDM.WindowsSettings.CustomSettings.Value, ac.MDM.WindowsSettings.CustomSettings.Value) // applied
			require.Empty(t, acResp.MDM.MacOSSettings.CustomSettings)                                                              // no change

			// cleanup Windows settings
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{}
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{})
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, true) // only windows mdm enabled
			require.Empty(t, acResp.MDM.MacOSSettings.CustomSettings)
			require.Empty(t, acResp.MDM.WindowsSettings.CustomSettings.Value)

			// directly set MDM.EnabledAndConfigured to true and windows to false
			ac.MDM.EnabledAndConfigured = true
			ac.MDM.WindowsEnabledAndConfigured = false
			require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
			acResp = checkAppConfig(t, true, false)                                                                // mac enabled, windows disabled
			require.ElementsMatch(t, acResp.MDM.MacOSSettings.CustomSettings, ac.MDM.MacOSSettings.CustomSettings) // directly applied
			require.Empty(t, acResp.MDM.WindowsSettings.CustomSettings.Value)                                      // still empty

			// add custom windows settings fails because only mac is enabled
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "baz"}, {Path: "zab"}})
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusUnprocessableEntity, &acResp)
			acResp = checkAppConfig(t, true, false) // only mac enabled
			require.Empty(t, acResp.MDM.MacOSSettings.CustomSettings)
			require.Empty(t, acResp.MDM.WindowsSettings.CustomSettings.Value)
			// set this value to empty again so we can test other assertions assuming we're not setting it
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{})

			// changing unrelated config doesn't cause validation error
			ac.OrgInfo.OrgName = "f1338"
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, true, false)                                                                // mac enabled, windows disabled
			require.ElementsMatch(t, acResp.MDM.MacOSSettings.CustomSettings, ac.MDM.MacOSSettings.CustomSettings) // no change
			require.Empty(t, acResp.MDM.WindowsSettings.CustomSettings.Value)                                      // no change
			require.Equal(t, "f1338", acResp.AppConfig.OrgInfo.OrgName)

			// remove custom settings doesn't cause validation error
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{}
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, true, false) // only mac enabled
			require.Empty(t, acResp.MDM.MacOSSettings.CustomSettings)

			// add custom macOS settings suceeds because only Windows is disabled
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{{Path: "foo"}, {Path: "bar"}}
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, true, false)                                                                // mac enabled, windows disabled
			require.ElementsMatch(t, acResp.MDM.MacOSSettings.CustomSettings, ac.MDM.MacOSSettings.CustomSettings) // applied
			require.Empty(t, acResp.MDM.WindowsSettings.CustomSettings.Value)                                      // no change

			// temporarily enable and add custom settings for both platforms
			ac.MDM.EnabledAndConfigured = true
			ac.MDM.WindowsEnabledAndConfigured = true
			require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
			acResp = checkAppConfig(t, true, true) // both mac and windows mdm enabled
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{{Path: "foo"}, {Path: "bar"}}
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "baz"}, {Path: "zab"}})
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, true, true)                                                                                 // both mac and windows mdm enabled
			require.ElementsMatch(t, acResp.MDM.MacOSSettings.CustomSettings, ac.MDM.MacOSSettings.CustomSettings)                 // applied
			require.ElementsMatch(t, acResp.MDM.WindowsSettings.CustomSettings.Value, ac.MDM.WindowsSettings.CustomSettings.Value) // applied

			// directly set both configs to false
			ac.MDM.EnabledAndConfigured = false
			ac.MDM.WindowsEnabledAndConfigured = false
			require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
			acResp = checkAppConfig(t, false, false)                                                                               // both mac and windows mdm disabled
			require.ElementsMatch(t, acResp.MDM.MacOSSettings.CustomSettings, ac.MDM.MacOSSettings.CustomSettings)                 // no change
			require.ElementsMatch(t, acResp.MDM.WindowsSettings.CustomSettings.Value, ac.MDM.WindowsSettings.CustomSettings.Value) // no change

			// changing unrelated config doesn't cause validation error
			ac.OrgInfo.OrgName = "f1339"
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, false)                                                                               // both disabled
			require.ElementsMatch(t, acResp.MDM.MacOSSettings.CustomSettings, ac.MDM.MacOSSettings.CustomSettings)                 // no change
			require.ElementsMatch(t, acResp.MDM.WindowsSettings.CustomSettings.Value, ac.MDM.WindowsSettings.CustomSettings.Value) // no change
			require.Equal(t, "f1339", acResp.AppConfig.OrgInfo.OrgName)

			// setting the same values is ok even if mdm is disabled
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{{Path: "foo"}, {Path: "bar"}}
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "baz"}, {Path: "zab"}})
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, false)                                                                               // both disabled
			require.ElementsMatch(t, acResp.MDM.MacOSSettings.CustomSettings, ac.MDM.MacOSSettings.CustomSettings)                 // no change
			require.ElementsMatch(t, acResp.MDM.WindowsSettings.CustomSettings.Value, ac.MDM.WindowsSettings.CustomSettings.Value) // no change

			// setting different values fail even if mdm is disabled, and only some of the profiles have changed
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{{Path: "oof"}, {Path: "bar"}}
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "foo"}, {Path: "zab"}})
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusUnprocessableEntity, &acResp)
			acResp = checkAppConfig(t, false, false) // both disabled
			// set the values back so we can compare them
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{{Path: "foo"}, {Path: "bar"}}
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "baz"}, {Path: "zab"}})
			require.ElementsMatch(t, acResp.MDM.MacOSSettings.CustomSettings, ac.MDM.MacOSSettings.CustomSettings)                 // no change
			require.ElementsMatch(t, acResp.MDM.WindowsSettings.CustomSettings.Value, ac.MDM.WindowsSettings.CustomSettings.Value) // no change

			// setting empty values doesn't cause validation error when mdm is disabled
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{}
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{})
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, false) // both disabled
			require.Empty(t, acResp.MDM.MacOSSettings.CustomSettings)
			require.Empty(t, acResp.MDM.WindowsSettings.CustomSettings.Value)

			// setting non-empty values fails because mdm disabled
			ac.MDM.MacOSSettings.CustomSettings = []fleet.MDMProfileSpec{{Path: "foo"}, {Path: "bar"}}
			ac.MDM.WindowsSettings.CustomSettings = optjson.SetSlice([]fleet.MDMProfileSpec{{Path: "baz"}, {Path: "zab"}})
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusUnprocessableEntity, &acResp)
			acResp = checkAppConfig(t, false, false) // both disabled
			require.Empty(t, acResp.MDM.MacOSSettings.CustomSettings)
			require.Empty(t, acResp.MDM.WindowsSettings.CustomSettings.Value)
		})
	})

	// TODO: Improve validations and related test coverage of team MDM config.
	// Some settings don't have MDMEnabledAndConfigured or WindowsMDMEnabledAndConfigured
	// validations currently. Either add vailidations and test them or test abscence
	// of validation. Also, the tests below only cover a limited set of permutations
	// compared to the app config tests above and should be expanded accordingly.
	t.Run("modify team", func(t *testing.T) {
		t.Cleanup(func() {
			require.NoError(t, s.ds.SaveAppConfig(ctx, appConfig))
		})

		checkTeam := func(t *testing.T, team *fleet.Team, checkMDM *fleet.TeamPayloadMDM) teamResponse {
			var wantDiskEncryption bool
			var wantMacOSSetup fleet.MacOSSetup
			if checkMDM != nil {
				if checkMDM.MacOSSetup != nil {
					wantMacOSSetup = *checkMDM.MacOSSetup
					// bootstrap package always ignored by modify team endpoint so expect original value
					wantMacOSSetup.BootstrapPackage = team.Config.MDM.MacOSSetup.BootstrapPackage
					// setup assistant always ignored by modify team endpoint so expect original value
					wantMacOSSetup.MacOSSetupAssistant = team.Config.MDM.MacOSSetup.MacOSSetupAssistant
				}
				wantDiskEncryption = checkMDM.EnableDiskEncryption.Value
			}

			var resp teamResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &resp)
			require.Equal(t, team.Name, resp.Team.Name)
			require.Equal(t, wantDiskEncryption, resp.Team.Config.MDM.EnableDiskEncryption)
			require.Equal(t, wantMacOSSetup.BootstrapPackage.Value, resp.Team.Config.MDM.MacOSSetup.BootstrapPackage.Value)
			require.Equal(t, wantMacOSSetup.MacOSSetupAssistant.Value, resp.Team.Config.MDM.MacOSSetup.MacOSSetupAssistant.Value)
			require.Equal(t, wantMacOSSetup.EnableEndUserAuthentication, resp.Team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)

			return resp
		}

		// initialize our test app config
		ac := appConfig.Copy()
		ac.AgentOptions = nil
		ac.MDM.EnabledAndConfigured = false
		ac.MDM.WindowsEnabledAndConfigured = false
		require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
		checkAppConfig(t, false, false) // both mac and windows mdm disabled

		var createTeamResp teamResponse
		s.DoJSON("POST", "/api/latest/fleet/teams", createTeamRequest{fleet.TeamPayload{
			Name: ptr.String("Ninjas"),
			MDM:  &fleet.TeamPayloadMDM{EnableDiskEncryption: optjson.SetBool(true)}, // mdm is ignored by the create team endpoint
		}}, http.StatusOK, &createTeamResp)
		team := createTeamResp.Team
		getTeamResp := checkTeam(t, team, nil) // newly created team has empty mdm config

		t.Cleanup(func() {
			require.NoError(t, s.ds.DeleteTeam(ctx, team.ID))
		})

		// TODO: Add cases for other team MDM config (e.g., macos settings, macos updates,
		// migration) and for other permutations of starting values (see app config tests above).
		cases := []struct {
			name           string
			mdm            *fleet.TeamPayloadMDM
			expectedStatus int
		}{
			{
				"mdm empty",
				&fleet.TeamPayloadMDM{},
				http.StatusOK,
			},
			{
				"mdm all zero values",
				&fleet.TeamPayloadMDM{
					EnableDiskEncryption: optjson.SetBool(false),
					MacOSSetup: &fleet.MacOSSetup{
						BootstrapPackage:            optjson.SetString(""),
						EnableEndUserAuthentication: false,
						MacOSSetupAssistant:         optjson.SetString(""),
					},
				},
				http.StatusOK,
			},
			{
				"bootstrap package",
				&fleet.TeamPayloadMDM{
					MacOSSetup: &fleet.MacOSSetup{
						BootstrapPackage: optjson.SetString("some-package"),
					},
				},
				// bootstrap package is always ignored by the modify team endpoint
				http.StatusOK,
			},
			{
				"setup assistant",
				&fleet.TeamPayloadMDM{
					MacOSSetup: &fleet.MacOSSetup{
						MacOSSetupAssistant: optjson.SetString("some-setup-assistant"),
					},
				},
				// setup assistant is always ignored by the modify team endpoint
				http.StatusOK,
			},
			{
				"enable disk encryption",
				&fleet.TeamPayloadMDM{
					EnableDiskEncryption: optjson.SetBool(true),
				},
				// disk encryption requires mdm enabled and configured
				http.StatusUnprocessableEntity,
			},
			{
				"enable end user auth",
				&fleet.TeamPayloadMDM{
					MacOSSetup: &fleet.MacOSSetup{
						EnableEndUserAuthentication: true,
					},
				},
				// disk encryption requires mdm enabled and configured
				http.StatusUnprocessableEntity,
			},
		}

		for _, c := range cases {
			// TODO: Add tests for other combinations of mac and windows mdm enabled/disabled
			t.Run(c.name, func(t *testing.T) {
				checkAppConfig(t, false, false) // both mac and windows mdm disabled

				s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
					Name:        &team.Name,
					Description: ptr.String(c.name),
					MDM:         c.mdm,
				}, c.expectedStatus, &getTeamResp)

				if c.expectedStatus == http.StatusOK {
					getTeamResp = checkTeam(t, team, c.mdm)
					require.Equal(t, c.name, getTeamResp.Team.Description)
				} else {
					checkTeam(t, team, nil)
				}
			})
		}
	})

	// TODO: Improve validations and related test coverage of team MDM config.
	// Some settings don't have MDMEnabledAndConfigured or WindowsMDMEnabledAndConfigured
	// validations currently. Either add vailidations and test them or test abscence
	// of validation. Also, the tests below only cover a limited set of permutations
	// compared to the app config tests above and should be expanded accordingly.
	t.Run("edit team spec", func(t *testing.T) {
		t.Cleanup(func() {
			require.NoError(t, s.ds.SaveAppConfig(ctx, appConfig))
		})

		checkTeam := func(t *testing.T, team *fleet.Team, checkMDM *fleet.TeamSpecMDM) teamResponse {
			// TODO - remove check of disk encryption from this function entirely?
			// var wantDiskEncryption bool
			var wantMacOSSetup fleet.MacOSSetup
			if checkMDM != nil {
				wantMacOSSetup = checkMDM.MacOSSetup
				// wantDiskEncryption = checkMDM.EnableDiskEncryption.Value
			}

			var resp teamResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &resp)
			require.Equal(t, team.Name, resp.Team.Name)
			// require.Equal(t, wantDiskEncryption, resp.Team.Config.MDM.EnableDiskEncryption)
			require.Equal(t, wantMacOSSetup.BootstrapPackage.Value, resp.Team.Config.MDM.MacOSSetup.BootstrapPackage.Value)
			require.Equal(t, wantMacOSSetup.MacOSSetupAssistant.Value, resp.Team.Config.MDM.MacOSSetup.MacOSSetupAssistant.Value)
			require.Equal(t, wantMacOSSetup.EnableEndUserAuthentication, resp.Team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)

			return resp
		}

		// initialize our test app config
		ac := appConfig.Copy()
		ac.AgentOptions = nil
		ac.MDM.EnabledAndConfigured = false
		ac.MDM.WindowsEnabledAndConfigured = false
		require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
		checkAppConfig(t, false, false) // both mac and windows mdm disabled

		// create a team from spec
		tmSpecReq := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: "Pirates"}}}
		var tmSpecResp applyTeamSpecsResponse
		s.DoJSON("POST", "/api/latest/fleet/spec/teams", tmSpecReq, http.StatusOK, &tmSpecResp)
		teamID, ok := tmSpecResp.TeamIDsByName["Pirates"]
		require.True(t, ok)
		team := fleet.Team{ID: teamID, Name: "Pirates"}
		checkTeam(t, &team, nil) // newly created team has empty mdm config

		t.Cleanup(func() {
			require.NoError(t, s.ds.DeleteTeam(ctx, team.ID))
		})

		// TODO: Add cases for other team MDM config (e.g., macos settings, macos updates,
		// migration) and for other permutations of starting values (see app config tests above).
		cases := []struct {
			name           string
			mdm            *fleet.TeamSpecMDM
			expectedStatus int
		}{
			{
				"mdm empty",
				&fleet.TeamSpecMDM{},
				http.StatusOK,
			},
			{
				"mdm all zero values",
				&fleet.TeamSpecMDM{
					EnableDiskEncryption: optjson.SetBool(false),
					MacOSSetup: fleet.MacOSSetup{
						BootstrapPackage:            optjson.SetString(""),
						EnableEndUserAuthentication: false,
						MacOSSetupAssistant:         optjson.SetString(""),
					},
				},
				http.StatusOK,
			},
			{
				"bootstrap package",
				&fleet.TeamSpecMDM{
					MacOSSetup: fleet.MacOSSetup{
						BootstrapPackage: optjson.SetString("some-package"),
					},
				},
				// bootstrap package requires mdm enabled and configured
				http.StatusUnprocessableEntity,
			},
			{
				"setup assistant",
				&fleet.TeamSpecMDM{
					MacOSSetup: fleet.MacOSSetup{
						MacOSSetupAssistant: optjson.SetString("some-setup-assistant"),
					},
				},
				// setup assistant requires mdm enabled and configured
				http.StatusUnprocessableEntity,
			},
			{
				"enable disk encryption",
				&fleet.TeamSpecMDM{
					EnableDiskEncryption: optjson.SetBool(true),
				},
				// disk encryption does not require mdm enabled and configured
				http.StatusOK,
			},
			// Ian - this test still passes, that is, returns 4xx – perhaps related to one of the endpoints we still need to update
			{
				"enable end user auth",
				&fleet.TeamSpecMDM{
					MacOSSetup: fleet.MacOSSetup{
						EnableEndUserAuthentication: true,
					},
				},
				// disk encryption requires mdm enabled and configured
				http.StatusUnprocessableEntity,
			},
		}

		for _, c := range cases {
			// TODO: Add tests for other combinations of mac and windows mdm enabled/disabled
			t.Run(c.name, func(t *testing.T) {
				checkAppConfig(t, false, false) // both mac and windows mdm disabled

				tmSpecReq = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
					Name: team.Name,
					MDM:  *c.mdm,
				}}}
				s.DoJSON("POST", "/api/latest/fleet/spec/teams", tmSpecReq, c.expectedStatus, &tmSpecResp)

				if c.expectedStatus == http.StatusOK {
					checkTeam(t, &team, c.mdm)
				} else {
					checkTeam(t, &team, nil)
				}
			})
		}
	})
}

// ///////////////////////////////////////////////////////////////////////////
// Common helpers

func (s *integrationMDMTestSuite) runWorker() {
	err := s.worker.ProcessJobs(context.Background())
	require.NoError(s.T(), err)
	pending, err := s.ds.GetQueuedJobs(context.Background(), 1, time.Time{})
	require.NoError(s.T(), err)
	require.Empty(s.T(), pending)
}

func (s *integrationMDMTestSuite) runDEPSchedule() {
	ctx := context.Background()
	fleetSyncer := apple_mdm.NewDEPService(s.ds, s.depStorage, s.logger)
	err := fleetSyncer.RunAssigner(ctx)
	require.NoError(s.T(), err)
}

func (s *integrationMDMTestSuite) runIntegrationsSchedule() {
	// FIXME: This pattern (which is being used in testing other schedules as well) seems cause issues
	// where a subsequent call attempts to trigger when the schedule's trigger channel is full and
	// schedule ignored the subsquent call (which is the documented behavior of the trigger).
	// In testing, this can cause the test to hang until the next scheduled run. It isn't a very
	// noticeable issue here since the intervals for these schedules are short.
	ch := make(chan bool)
	var once sync.Once
	s.onIntegrationsScheduleDone = func() {
		once.Do(func() { close(ch) })
	}
	_, err := s.integrationsSchedule.Trigger()
	require.NoError(s.T(), err)
	<-ch
}

func (s *integrationMDMTestSuite) getRawTokenValue(content string) string {
	// Create a regex object with the defined pattern
	pattern := `inputToken.value\s*=\s*'([^']*)'`
	regex := regexp.MustCompile(pattern)

	// Find the submatch using the regex pattern
	submatches := regex.FindStringSubmatch(content)

	if len(submatches) >= 2 {
		// Extract the content from the submatch
		encodedToken := submatches[1]

		return encodedToken
	}

	return ""
}

func (s *integrationMDMTestSuite) isXMLTagPresent(xmlTag string, payload string) bool {
	regex := fmt.Sprintf("<%s.*>", xmlTag)
	matched, err := regexp.MatchString(regex, payload)
	if err != nil {
		return false
	}

	return matched
}

func (s *integrationMDMTestSuite) isXMLTagContentPresent(xmlTag string, payload string) bool {
	regex := fmt.Sprintf("<%s.*>(.+)</%s.*>", xmlTag, xmlTag)
	matched, err := regexp.MatchString(regex, payload)
	if err != nil {
		return false
	}

	return matched
}

func (s *integrationMDMTestSuite) checkIfXMLTagContains(xmlTag string, xmlContent string, payload string) bool {
	regex := fmt.Sprintf("<%s.*>.*%s.*</%s.*>", xmlTag, xmlContent, xmlTag)

	matched, err := regexp.MatchString(regex, payload)
	if err != nil || !matched {
		return false
	}

	return true
}

func (s *integrationMDMTestSuite) newGetPoliciesMsg(deviceToken bool, encodedBinToken string) ([]byte, error) {
	if len(encodedBinToken) == 0 {
		return nil, errors.New("encodedBinToken is empty")
	}

	// JWT token by default
	tokType := syncml.BinarySecurityAzureEnroll
	if deviceToken {
		tokType = syncml.BinarySecurityDeviceEnroll
	}

	return []byte(`
			<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://www.w3.org/2005/08/addressing" xmlns:u="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd" xmlns:wsse="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd" xmlns:wst="http://docs.oasis-open.org/ws-sx/ws-trust/200512" xmlns:ac="http://schemas.xmlsoap.org/ws/2006/12/authorization">
			<s:Header>
				<a:Action s:mustUnderstand="1">http://schemas.microsoft.com/windows/pki/2009/01/enrollmentpolicy/IPolicy/GetPolicies</a:Action>
				<a:MessageID>urn:uuid:148132ec-a575-4322-b01b-6172a9cf8478</a:MessageID>
				<a:ReplyTo>
				<a:Address>http://www.w3.org/2005/08/addressing/anonymous</a:Address>
				</a:ReplyTo>
				<a:To s:mustUnderstand="1">https://mdmwindows.com/EnrollmentServer/Policy.svc</a:To>
				<wsse:Security s:mustUnderstand="1">
				<wsse:BinarySecurityToken ValueType="` + tokType + `" EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd#base64binary">` + encodedBinToken + `</wsse:BinarySecurityToken>
				</wsse:Security>
			</s:Header>
			<s:Body xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema">
				<GetPolicies xmlns="http://schemas.microsoft.com/windows/pki/2009/01/enrollmentpolicy">
				<client>
					<lastUpdate xsi:nil="true"/>
					<preferredLanguage xsi:nil="true"/>
				</client>
				<requestFilter xsi:nil="true"/>
				</GetPolicies>
			</s:Body>
			</s:Envelope>`), nil
}

func (s *integrationMDMTestSuite) newSecurityTokenMsg(encodedBinToken string, deviceToken bool, missingContextItem bool) ([]byte, error) {
	if len(encodedBinToken) == 0 {
		return nil, errors.New("encodedBinToken is empty")
	}

	var reqSecTokenContextItemDeviceType []byte
	if !missingContextItem {
		reqSecTokenContextItemDeviceType = []byte(
			`<ac:ContextItem Name="DeviceType">
			 <ac:Value>CIMClient_Windows</ac:Value>
			 </ac:ContextItem>`)
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
					<ac:Value>CF1D12AA5AE42E47D52465E9A71316CAF3AFCC1D3088F230F4D50B371FB2256F</ac:Value>
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
		`)

	return requestBytes, nil
}

func (s *integrationMDMTestSuite) newSyncMLUnenrollMsg(deviceID string, managementUrl string) ([]byte, error) {
	if len(managementUrl) == 0 {
		return nil, errors.New("managementUrl is empty")
	}

	return []byte(`
			 <SyncML xmlns="SYNCML:SYNCML1.2">
			<SyncHdr>
				<VerDTD>1.2</VerDTD>
				<VerProto>DM/1.2</VerProto>
				<SessionID>2</SessionID>
				<MsgID>1</MsgID>
				<Target>
				<LocURI>` + managementUrl + `</LocURI>
				</Target>
				<Source>
				<LocURI>` + deviceID + `</LocURI>
				</Source>
			</SyncHdr>
			<SyncBody>
				<Alert>
				<CmdID>2</CmdID>
				<Data>1201</Data>
				</Alert>
				<Alert>
				<CmdID>3</CmdID>
				<Data>1224</Data>
				<Item>
					<Meta>
					<Type xmlns="syncml:metinf">com.microsoft/MDM/LoginStatus</Type>
					</Meta>
					<Data>user</Data>
				</Item>
				</Alert>
				<Alert>
				<CmdID>4</CmdID>
				<Data>1226</Data>
				<Item>
					<Meta>
					<Type xmlns="syncml:metinf">com.microsoft:mdm.unenrollment.userrequest</Type>
					<Format xmlns="syncml:metinf">int</Format>
					</Meta>
					<Data>1</Data>
				</Item>
				</Alert>
				<Final/>
			</SyncBody>
			</SyncML>`), nil
}

func (s *integrationMDMTestSuite) checkMDMProfilesSummaries(t *testing.T, teamID *uint, expectedSummary fleet.MDMProfilesSummary, expectedAppleSummary *fleet.MDMProfilesSummary) {
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

func (s *integrationMDMTestSuite) checkMDMDiskEncryptionSummaries(t *testing.T, teamID *uint, expectedSummary fleet.MDMDiskEncryptionSummary, checkFileVaultSummary bool) {
	var queryParams []string
	if teamID != nil {
		queryParams = append(queryParams, "team_id", fmt.Sprintf("%d", *teamID))
	}

	if checkFileVaultSummary {
		var fileVault getMDMAppleFileVaultSummaryResponse
		s.DoJSON("GET", "/api/v1/fleet/mdm/apple/filevault/summary", getMDMProfilesSummaryRequest{}, http.StatusOK, &fileVault, queryParams...)
		require.Equal(t, expectedSummary.Failed.MacOS, fileVault.Failed)
		require.Equal(t, expectedSummary.Enforcing.MacOS, fileVault.Enforcing)
		require.Equal(t, expectedSummary.ActionRequired.MacOS, fileVault.ActionRequired)
		require.Equal(t, expectedSummary.Verifying.MacOS, fileVault.Verifying)
		require.Equal(t, expectedSummary.Verified.MacOS, fileVault.Verified)
		require.Equal(t, expectedSummary.RemovingEnforcement.MacOS, fileVault.RemovingEnforcement)
	}

	var combined getMDMDiskEncryptionSummaryResponse
	s.DoJSON("GET", "/api/v1/fleet/disk_encryption", getMDMProfilesSummaryRequest{}, http.StatusOK, &combined, queryParams...)
	require.Equal(t, expectedSummary.Failed, combined.Failed)
	require.Equal(t, expectedSummary.Enforcing, combined.Enforcing)
	require.Equal(t, expectedSummary.ActionRequired, combined.ActionRequired)
	require.Equal(t, expectedSummary.Verifying, combined.Verifying)
	require.Equal(t, expectedSummary.Verified, combined.Verified)
	require.Equal(t, expectedSummary.RemovingEnforcement, combined.RemovingEnforcement)
}

func (s *integrationMDMTestSuite) TestWindowsFreshEnrollEmptyQuery() {
	t := s.T()
	host, _ := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	// make sure we don't have any profiles
	s.Do(
		"POST",
		"/api/v1/fleet/mdm/profiles/batch",
		batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{}},
		http.StatusNoContent,
	)

	// Ensure we can read distributed queries for the host.
	err := s.ds.UpdateHostRefetchRequested(context.Background(), host.ID, true)
	require.NoError(t, err)

	s.lq.On("QueriesForHost", host.ID).Return(map[string]string{fmt.Sprintf("%d", host.ID): "SELECT 1 FROM osquery;"}, nil)

	req := getDistributedQueriesRequest{NodeKey: *host.NodeKey}
	var dqResp getDistributedQueriesResponse
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	require.NotContains(t, dqResp.Queries, "fleet_detail_query_mdm_config_profiles_windows")

	// add two profiles
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
		{Name: "N2", Contents: syncMLForTest("./Foo/Bar")},
	}}, http.StatusNoContent)

	req = getDistributedQueriesRequest{NodeKey: *host.NodeKey}
	dqResp = getDistributedQueriesResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	require.Contains(t, dqResp.Queries, "fleet_detail_query_mdm_config_profiles_windows")
	require.NotEmpty(t, dqResp.Queries, "fleet_detail_query_mdm_config_profiles_windows")
}

func (s *integrationMDMTestSuite) TestManualEnrollmentCommands() {
	t := s.T()

	checkInstallFleetdCommandSent := func(mdmDevice *mdmtest.TestAppleMDMClient, wantCommand bool) {
		foundInstallFleetdCommand := false
		cmd, err := mdmDevice.Idle()
		require.NoError(t, err)
		for cmd != nil {
			var fullCmd micromdm.CommandPayload
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
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

	// create a device that's not enrolled into Fleet, it should get a command to
	// install fleetd
	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "MacBookPro16,1")
	err := mdmDevice.Enroll()
	require.NoError(t, err)
	s.runWorker()
	checkInstallFleetdCommandSent(mdmDevice, true)

	// create a device that's enrolled into Fleet before turning on MDM features,
	// it should still get the command to install fleetd if turns on MDM.
	desktopToken := uuid.New().String()
	host := createOrbitEnrolledHost(t, "darwin", "h1", s.ds)
	err = s.ds.SetOrUpdateDeviceAuthToken(context.Background(), host.ID, desktopToken)
	require.NoError(t, err)
	mdmDevice = mdmtest.NewTestMDMClientAppleDesktopManual(s.server.URL, desktopToken)
	mdmDevice.UUID = host.UUID
	err = mdmDevice.Enroll()
	require.NoError(t, err)
	s.runWorker()
	checkInstallFleetdCommandSent(mdmDevice, true)
}

func (s *integrationMDMTestSuite) TestLockUnlockWipeWindowsLinux() {
	t := s.T()
	ctx := context.Background()

	// create an MDM-enrolled Windows host
	winHost, winMDMClient := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
	// set its MDM data so it shows as MDM-enrolled in the backend
	err := s.ds.SetOrUpdateMDMData(ctx, winHost.ID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, "")
	require.NoError(t, err)
	linuxHost := createOrbitEnrolledHost(t, "linux", "lock_unlock_linux", s.ds)

	for _, host := range []*fleet.Host{winHost, linuxHost} {
		t.Run(host.FleetPlatform(), func(t *testing.T) {
			// get the host's information
			var getHostResp getHostResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, "unlocked", *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

			// try to unlock the host (which is already its status)
			var unlockResp unlockHostResponse
			s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", host.ID), nil, http.StatusConflict, &unlockResp)

			// lock the host
			s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", host.ID), nil, http.StatusNoContent)

			// refresh the host's status, it is now pending lock
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, "unlocked", *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, "lock", *getHostResp.Host.MDM.PendingAction)

			// try locking the host while it is pending lock fails
			res := s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", host.ID), nil, http.StatusUnprocessableEntity)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "Host has pending lock request.")

			// simulate a successful script result for the lock command
			status, err := s.ds.GetHostLockWipeStatus(ctx, host)
			require.NoError(t, err)

			var orbitScriptResp orbitPostScriptResultResponse
			s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
				json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *host.OrbitNodeKey, status.LockScript.ExecutionID)),
				http.StatusOK, &orbitScriptResp)

			// refresh the host's status, it is now locked
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, "locked", *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

			// try to lock the host again
			s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", host.ID), nil, http.StatusConflict)
			// try to wipe a locked host
			res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID), nil, http.StatusUnprocessableEntity)
			errMsg = extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "Host cannot be wiped until it is unlocked.")

			// unlock the host
			s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", host.ID), nil, http.StatusNoContent)

			// refresh the host's status, it is locked pending unlock
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, "locked", *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, "unlock", *getHostResp.Host.MDM.PendingAction)

			// try unlocking the host while it is pending unlock fails
			res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", host.ID), nil, http.StatusUnprocessableEntity)
			errMsg = extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "Host has pending unlock request.")

			// simulate a failed script result for the unlock command
			status, err = s.ds.GetHostLockWipeStatus(ctx, host)
			require.NoError(t, err)

			s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
				json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": -1, "output": "fail"}`, *host.OrbitNodeKey, status.UnlockScript.ExecutionID)),
				http.StatusOK, &orbitScriptResp)

			// refresh the host's status, it is still locked, no pending action
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, "locked", *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

			// unlock the host, simulate success
			s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", host.ID), nil, http.StatusNoContent)
			status, err = s.ds.GetHostLockWipeStatus(ctx, host)
			require.NoError(t, err)
			s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
				json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *host.OrbitNodeKey, status.UnlockScript.ExecutionID)),
				http.StatusOK, &orbitScriptResp)

			// refresh the host's status, it is unlocked, no pending action
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, "unlocked", *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

			// wipe the host
			s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID), nil, http.StatusNoContent)
			wipeActID := s.lastActivityOfTypeMatches(fleet.ActivityTypeWipedHost{}.ActivityName(), fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, host.ID, host.DisplayName()), 0)

			// try to wipe the host again, already have it pending
			res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID), nil, http.StatusUnprocessableEntity)
			errMsg = extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "Host has pending wipe request.")
			// no activity created
			s.lastActivityOfTypeMatches(fleet.ActivityTypeWipedHost{}.ActivityName(), fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, host.ID, host.DisplayName()), wipeActID)

			// refresh the host's status, it is unlocked, pending wipe
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, "unlocked", *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, "wipe", *getHostResp.Host.MDM.PendingAction)

			status, err = s.ds.GetHostLockWipeStatus(ctx, host)
			require.NoError(t, err)
			if host.FleetPlatform() == "linux" {
				// simulate a successful wipe for the Linux host's script response
				s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
					json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *host.OrbitNodeKey, status.WipeScript.ExecutionID)),
					http.StatusOK, &orbitScriptResp)
			} else {
				// simulate a successful wipe from the Windows device's MDM response
				cmds, err := winMDMClient.StartManagementSession()
				require.NoError(t, err)

				// two status + the wipe command we enqueued
				require.Len(t, cmds, 3)
				wipeCmd := cmds[status.WipeMDMCommand.CommandUUID]
				require.NotNil(t, wipeCmd)
				require.Equal(t, wipeCmd.Verb, fleet.CmdExec)
				require.Len(t, wipeCmd.Cmd.Items, 1)
				require.EqualValues(t, "./Device/Vendor/MSFT/RemoteWipe/doWipeProtected", *wipeCmd.Cmd.Items[0].Target)

				msgID, err := winMDMClient.GetCurrentMsgID()
				require.NoError(t, err)

				winMDMClient.AppendResponse(fleet.SyncMLCmd{
					XMLName: xml.Name{Local: fleet.CmdStatus},
					MsgRef:  &msgID,
					CmdRef:  &status.WipeMDMCommand.CommandUUID,
					Cmd:     ptr.String("Exec"),
					Data:    ptr.String("200"),
					Items:   nil,
					CmdID:   fleet.CmdID{Value: uuid.NewString()},
				})
				cmds, err = winMDMClient.SendResponse()
				require.NoError(t, err)
				// the ack of the message should be the only returned command
				require.Len(t, cmds, 1)
			}

			// refresh the host's status, it is wiped
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, "wiped", *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

			// try to lock/unlock the host fails
			res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", host.ID), nil, http.StatusUnprocessableEntity)
			errMsg = extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "Cannot process lock requests once host is wiped.")
			res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", host.ID), nil, http.StatusUnprocessableEntity)
			errMsg = extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "Cannot process unlock requests once host is wiped.")

			// try to wipe the host again, conflict (already wiped)
			s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID), nil, http.StatusConflict)
			// no activity created
			s.lastActivityOfTypeMatches(fleet.ActivityTypeWipedHost{}.ActivityName(), fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, host.ID, host.DisplayName()), wipeActID)

			// re-enroll the host, simulating that another user received the wiped host
			newOrbitKey := uuid.New().String()
			newHost, err := s.ds.EnrollOrbit(ctx, true, fleet.OrbitHostInfo{
				HardwareUUID:   *host.OsqueryHostID,
				HardwareSerial: host.HardwareSerial,
			}, newOrbitKey, nil)
			require.NoError(t, err)
			// it re-enrolled using the same host record
			require.Equal(t, host.ID, newHost.ID)

			// refresh the host's status, it is back to unlocked
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, "unlocked", *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)
		})
	}
}

func (s *integrationMDMTestSuite) TestLockUnlockWipeMacOS() {
	t := s.T()
	host, mdmClient := createHostThenEnrollMDM(s.ds, s.server.URL, t)

	// get the host's information
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, "unlocked", *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

	// try to unlock the host (which is already its status)
	var unlockResp unlockHostResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", host.ID), nil, http.StatusConflict, &unlockResp)

	// lock the host
	var lockResp lockHostResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", host.ID), nil, http.StatusOK, &lockResp, "view_pin", "true")
	assert.Len(t, lockResp.UnlockPIN, 6)

	// refresh the host's status, it is now pending lock
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, "unlocked", *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, "lock", *getHostResp.Host.MDM.PendingAction)

	// try locking the host while it is pending lock fails
	res := s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", host.ID), nil, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Host has pending lock request.")

	// simulate a successful MDM result for the lock command
	cmd, err := mdmClient.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "DeviceLock", cmd.Command.RequestType)
	_, err = mdmClient.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)

	// refresh the host's status, it is now locked
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, "locked", *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

	// try to lock the host again
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", host.ID), nil, http.StatusConflict)
	// try to wipe a locked host
	res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID), nil, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Host cannot be wiped until it is unlocked.")

	// unlock the host
	unlockResp = unlockHostResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", host.ID), nil, http.StatusOK, &unlockResp)
	require.NotNil(t, unlockResp.HostID)
	require.Equal(t, host.ID, *unlockResp.HostID)
	require.Len(t, unlockResp.UnlockPIN, 6)
	unlockPIN := unlockResp.UnlockPIN
	unlockActID := s.lastActivityOfTypeMatches(fleet.ActivityTypeUnlockedHost{}.ActivityName(),
		fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "host_platform": %q}`, host.ID, host.DisplayName(), host.FleetPlatform()), 0)

	// refresh the host's status, it is still locked
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, "locked", *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	assert.Empty(t, *getHostResp.Host.MDM.PendingAction)

	// try unlocking the host again simply returns the PIN again
	unlockResp = unlockHostResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", host.ID), nil, http.StatusOK, &unlockResp)
	require.Equal(t, unlockPIN, unlockResp.UnlockPIN)
	// a new unlock host activity is created every time the unlock PIN is viewed
	newUnlockActID := s.lastActivityOfTypeMatches(fleet.ActivityTypeUnlockedHost{}.ActivityName(),
		fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "host_platform": %q}`, host.ID, host.DisplayName(), host.FleetPlatform()), 0)
	require.NotEqual(t, unlockActID, newUnlockActID)

	// as soon as the host sends an Idle MDM request, it is maked as unlocked
	cmd, err = mdmClient.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	// refresh the host's status, it is unlocked
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, "unlocked", *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

	// wipe the host
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID), nil, http.StatusNoContent)
	wipeActID := s.lastActivityOfTypeMatches(fleet.ActivityTypeWipedHost{}.ActivityName(), fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, host.ID, host.DisplayName()), 0)

	// try to wipe the host again, already have it pending
	res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID), nil, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Host has pending wipe request.")
	// no activity created
	s.lastActivityOfTypeMatches(fleet.ActivityTypeWipedHost{}.ActivityName(), fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, host.ID, host.DisplayName()), wipeActID)

	// refresh the host's status, it is unlocked, pending wipe
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, "unlocked", *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, "wipe", *getHostResp.Host.MDM.PendingAction)

	// simulate a successful MDM result for the wipe command
	cmd, err = mdmClient.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "EraseDevice", cmd.Command.RequestType)
	_, err = mdmClient.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)

	// refresh the host's status, it is wiped
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, "wiped", *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

	// try to lock/unlock the host fails
	res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", host.ID), nil, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Cannot process lock requests once host is wiped.")
	res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", host.ID), nil, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Cannot process unlock requests once host is wiped.")

	// try to wipe the host again, conflict (already wiped)
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID), nil, http.StatusConflict)
	// no activity created
	s.lastActivityOfTypeMatches(fleet.ActivityTypeWipedHost{}.ActivityName(), fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, host.ID, host.DisplayName()), wipeActID)

	// re-enroll the host, simulating that another user received the wiped host
	err = mdmClient.Enroll()
	require.NoError(t, err)

	// refresh the host's status, it is back to unlocked
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, "unlocked", *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

	// lock the host without viewing the PIN
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", host.ID), nil, http.StatusNoContent)
}

func (s *integrationMDMTestSuite) TestCustomConfigurationWebURL() {
	t := s.T()

	acResp := appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)

	s.enableABM(t.Name())
	var lastSubmittedProfile *godep.Profile
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)

		switch r.URL.Path {
		case "/server/devices", "/devices/sync":
			encoder := json.NewEncoder(w)
			err := encoder.Encode(godep.DeviceResponse{
				Devices: []godep.Device{
					{
						SerialNumber: "FAKE-1",
						Model:        "Mac Mini",
						OS:           "osx",
						OpType:       "added",
					},
					{
						SerialNumber: "FAKE-2",
						Model:        "Mac Mini",
						OS:           "osx",
						OpType:       "added",
					},
				},
			})
			require.NoError(t, err)
		case "/profile":
			lastSubmittedProfile = &godep.Profile{}
			rawProfile, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			err = json.Unmarshal(rawProfile, lastSubmittedProfile)
			require.NoError(t, err)

			// check that the urls are not empty and equal
			require.NotEmpty(t, lastSubmittedProfile.URL)
			require.NotEmpty(t, lastSubmittedProfile.ConfigurationWebURL)
			require.Equal(t, lastSubmittedProfile.URL, lastSubmittedProfile.ConfigurationWebURL)
			err = encoder.Encode(godep.ProfileResponse{ProfileUUID: uuid.New().String()})
			require.NoError(t, err)
		default:
			_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
		}
	}))

	// run once to ingest the devices
	s.runDEPSchedule()

	// disable first to make sure we start in the desired state
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"macos_setup": {
				"enable_end_user_authentication": false
			}
		}
	}`), http.StatusOK, &acResp)

	// configure end-user authentication globally
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"end_user_authentication": {
				"entity_id": "https://localhost:8080",
				"issuer_uri": "http://localhost:8080/simplesaml/saml2/idp/SSOService.php",
				"idp_name": "SimpleSAML",
				"metadata_url": "http://localhost:9080/simplesaml/saml2/idp/metadata.php"
			},
			"macos_setup": {
				"enable_end_user_authentication": true
			}
		}
	}`), http.StatusOK, &acResp)

	// assign the DEP profile and assert that contains the right values for the URL
	s.runWorker()
	require.NotNil(t, lastSubmittedProfile)
	require.Contains(t, lastSubmittedProfile.ConfigurationWebURL, acResp.ServerSettings.ServerURL+"/mdm/sso")

	// trying to set a custom configuration_web_url fails because end user authentication is enabled
	customSetupAsst := `{"configuration_web_url": "https://foo.example.com"}`
	var globalAsstResp createMDMAppleSetupAssistantResponse
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "no-team",
		EnrollmentProfile: json.RawMessage(customSetupAsst),
	}, http.StatusUnprocessableEntity, &globalAsstResp)

	// disable end user authentication
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"end_user_authentication": {
				"entity_id": "",
				"issuer_uri": "",
				"idp_name": "",
				"metadata_url": ""
			},
			"macos_setup": {
				"enable_end_user_authentication": false
			}
		}
	}`), http.StatusOK, &acResp)

	// assign the DEP profile and assert that contains the right values for the URL
	s.runWorker()
	require.NotNil(t, lastSubmittedProfile)
	require.Contains(t, lastSubmittedProfile.ConfigurationWebURL, acResp.ServerSettings.ServerURL+"/api/mdm/apple/enroll?token=")

	// setting a custom configuration_web_url succeeds because user authentication is disabled
	globalAsstResp = createMDMAppleSetupAssistantResponse{}
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "no-team",
		EnrollmentProfile: json.RawMessage(customSetupAsst),
	}, http.StatusOK, &globalAsstResp)

	// assign the DEP profile and assert that contains the right values for the URL
	s.runWorker()
	require.NotNil(t, lastSubmittedProfile)
	require.Contains(t, lastSubmittedProfile.ConfigurationWebURL, "https://foo.example.com")

	// try to enable end user auth again, it fails because configuration_web_url is set
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"end_user_authentication": {
				"entity_id": "https://localhost:8080",
				"issuer_uri": "http://localhost:8080/simplesaml/saml2/idp/SSOService.php",
				"idp_name": "SimpleSAML",
				"metadata_url": "http://localhost:9080/simplesaml/saml2/idp/metadata.php"
			},
			"macos_setup": {
				"enable_end_user_authentication": true
			}
		}
	}`), http.StatusUnprocessableEntity, &acResp)

	// create a team via spec
	teamSpecs := map[string]any{
		"specs": []any{
			map[string]any{
				"name": t.Name(),
				"mdm": map[string]any{
					"macos_setup": map[string]any{
						"enable_end_user_authentication": false,
					},
				},
			},
		},
	}
	var applyResp applyTeamSpecsResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)
	teamID := applyResp.TeamIDsByName[t.Name()]

	// transfer a host to the team to ensure all ABM calls are made
	h, err := s.ds.HostByIdentifier(context.Background(), "FAKE-1")
	require.NoError(t, err)
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{TeamID: &teamID, HostIDs: []uint{h.ID}}, http.StatusOK, &addHostsToTeamResponse{})

	// re-set the global state to configure MDM SSO
	err = s.ds.DeleteMDMAppleSetupAssistant(context.Background(), nil)
	require.NoError(t, err)
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"end_user_authentication": {
				"entity_id": "https://localhost:8080",
				"issuer_uri": "http://localhost:8080/simplesaml/saml2/idp/SSOService.php",
				"idp_name": "SimpleSAML",
				"metadata_url": "http://localhost:9080/simplesaml/saml2/idp/metadata.php"
			},
			"macos_setup": {
				"enable_end_user_authentication": true
			}
		}
	}`), http.StatusOK, &acResp)

	// enable end user auth
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": t.Name(),
				"mdm": map[string]any{
					"macos_setup": map[string]any{
						"enable_end_user_authentication": true,
					},
				},
			},
		},
	}
	applyResp = applyTeamSpecsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)
	// assign the DEP profile and assert that contains the right values for the URL
	s.runWorker()
	require.Contains(t, lastSubmittedProfile.ConfigurationWebURL, acResp.ServerSettings.ServerURL+"/mdm/sso")

	// trying to set a custom configuration_web_url fails because end user authentication is enabled
	var tmAsstResp createMDMAppleSetupAssistantResponse
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            &teamID,
		Name:              t.Name(),
		EnrollmentProfile: json.RawMessage(customSetupAsst),
	}, http.StatusUnprocessableEntity, &tmAsstResp)

	// disable end user auth
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": t.Name(),
				"mdm": map[string]any{
					"macos_setup": map[string]any{
						"enable_end_user_authentication": false,
					},
				},
			},
		},
	}
	applyResp = applyTeamSpecsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)

	// assign the DEP profile and assert that contains the right values for the URL
	s.runWorker()
	require.Contains(t, lastSubmittedProfile.ConfigurationWebURL, acResp.ServerSettings.ServerURL+"/api/mdm/apple/enroll?token=")

	// setting configuration_web_url succeeds because end user authentication is disabled
	tmAsstResp = createMDMAppleSetupAssistantResponse{}
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            &teamID,
		Name:              t.Name(),
		EnrollmentProfile: json.RawMessage(customSetupAsst),
	}, http.StatusOK, &tmAsstResp)

	// assign the DEP profile and assert that contains the right values for the URL
	s.runWorker()
	require.Contains(t, lastSubmittedProfile.ConfigurationWebURL, "https://foo.example.com")

	// try to enable end user auth again, it fails because configuration_web_url is set
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": t.Name(),
				"mdm": map[string]any{
					"macos_setup": map[string]any{
						"enable_end_user_authentication": true,
					},
				},
			},
		},
	}
	applyResp = applyTeamSpecsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusUnprocessableEntity, &applyResp)
}

func (s *integrationMDMTestSuite) TestDontIgnoreAnyProfileErrors() {
	t := s.T()
	ctx := context.Background()

	// Create a host and a couple of profiles
	host, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)

	globalProfiles := [][]byte{
		mobileconfigForTest("N1", "I1"),
		mobileconfigForTest("N2", "I2"),
	}

	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: globalProfiles}, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)

	// The profiles should be associated with the host we made + the standard fleet configs
	profs, err := s.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
	require.NoError(t, err)
	require.Len(t, profs, 4)

	// Acknowledge the profiles so we can mark them as verified
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(context.Background(), s.ds, host, map[string]*fleet.HostMacOSProfile{
		"I1": {Identifier: "I1", DisplayName: "I1", InstallDate: time.Now()},
		"I2": {Identifier: "I2", DisplayName: "I2", InstallDate: time.Now()},
		mobileconfig.FleetdConfigPayloadIdentifier:      {Identifier: mobileconfig.FleetdConfigPayloadIdentifier, DisplayName: "I2", InstallDate: time.Now()},
		mobileconfig.FleetCARootConfigPayloadIdentifier: {Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, DisplayName: "I2", InstallDate: time.Now()},
	}))

	// Check that the profile is marked as verified when fetching the host
	getHostResp := getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.Profiles)
	for _, hm := range *getHostResp.Host.MDM.Profiles {
		require.Equal(t, fleet.MDMDeliveryVerified, *hm.Status)
	}

	// remove the profiles
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{}, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)

	// On the host side, return errors for the two profile removal actions
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		if cmd.Command.RequestType == "RemoveProfile" {
			var errChain []mdm.ErrorChain
			var fullCmd micromdm.CommandPayload
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))

			if fullCmd.Command.RemoveProfile.Identifier == "I1" {
				errChain = append(errChain, mdm.ErrorChain{ErrorCode: 89, ErrorDomain: "MDMClientError", USEnglishDescription: "Profile with identifier 'I1' not found."})
			} else {
				errChain = append(errChain, mdm.ErrorChain{ErrorCode: 96, ErrorDomain: "MDMClientError", USEnglishDescription: "Cannot replace profile 'I2' because it was not installed by the MDM server."})
			}
			cmd, err = mdmDevice.Err(cmd.CommandUUID, errChain)
			require.NoError(t, err)
			continue
		}
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	// get that host - it should report "failed" for the profiles and include the error message detail.
	expectedErrs := map[string]string{
		"N1": "Failed to remove: MDMClientError (89): Profile with identifier 'I1' not found.\n",
		"N2": "Failed to remove: MDMClientError (96): Cannot replace profile 'I2' because it was not installed by the MDM server.\n",
	}
	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	for _, hm := range *getHostResp.Host.MDM.Profiles {
		if wantErr, ok := expectedErrs[hm.Name]; ok {
			require.Equal(t, fleet.MDMDeliveryFailed, *hm.Status)
			require.Equal(t, wantErr, hm.Detail)
			continue
		}
		require.Equal(t, fleet.MDMDeliveryVerified, *hm.Status)
	}
}

func (s *integrationMDMTestSuite) TestMDMDiskEncryptionIssue16636() {
	// see https://github.com/fleetdm/fleet/issues/16636

	t := s.T()

	// send an empty patch object to ensure it's ok if not provided
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{}`), http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableDiskEncryption.Value)
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, false)

	// set it to true
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// call PATCH with a null value, leaves it untouched
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": null }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// send an empty patch object to ensure it doesn't alter the value
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{}`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// setting it to false works
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": false }
  }`), http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableDiskEncryption.Value)
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, false)

	// send a null value again
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": null }
  }`), http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableDiskEncryption.Value)
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, false)

	// confirm via GET
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableDiskEncryption.Value)
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, false)
}

func (s *integrationMDMTestSuite) TestIsServerBitlockerStatus() {
	t := s.T()
	ctx := context.Background()

	// create a server host that is not enrolled in MDM
	host := createOrbitEnrolledHost(t, "windows", "server-host", s.ds)
	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, host.ID, true, false, "", false, "", ""))

	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)

	var hr getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hr)
	require.Nil(t, hr.Host.MDM.OSSettings.DiskEncryption.Status)

	// create a non-server host that is enrolled in MDM
	host2 := createOrbitEnrolledHost(t, "windows", "non-server-host", s.ds)
	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, host2.ID, false, true, "http://example.com", false, fleet.WellKnownMDMFleet, ""))

	hr = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host2.ID), nil, http.StatusOK, &hr)
	require.NotNil(t, hr.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, fleet.DiskEncryptionEnforcing, *hr.Host.MDM.OSSettings.DiskEncryption.Status)
}

func (s *integrationMDMTestSuite) TestRemoveFailedProfiles() {
	t := s.T()

	teamName := t.Name()
	team := &fleet.Team{
		Name:        teamName,
		Description: "desc " + teamName,
	}

	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &createTeamResp)
	require.NotZero(t, createTeamResp.Team.ID)
	team.ID = createTeamResp.Team.ID

	host, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)

	ident := uuid.NewString()

	mdmDeviceRespond := func(device *mdmtest.TestAppleMDMClient) {
		cmd, err := device.Idle()
		require.NoError(t, err)
		for cmd != nil {
			if cmd.Command.RequestType == "InstallProfile" {
				var fullCmd micromdm.CommandPayload
				require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))

				if strings.Contains(string(fullCmd.Command.InstallProfile.Payload), ident) {
					var errChain []mdm.ErrorChain
					errChain = append(errChain, mdm.ErrorChain{ErrorCode: -102, ErrorDomain: "CPProfile", USEnglishDescription: "The profile is either missing some required information, or contains information in an invalid format."})
					cmd, err = device.Err(cmd.CommandUUID, errChain)
					require.NoError(t, err)
					continue
				}
			}
			cmd, err = device.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}
	}

	globalProfiles := [][]byte{
		mobileconfigForTest("N1", ident),
		mobileconfigForTest("N2", "I2"),
	}
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: globalProfiles}, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)
	mdmDeviceRespond(mdmDevice)
	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(context.Background(), s.ds, host, map[string]*fleet.HostMacOSProfile{
		"I2": {Identifier: "I2", DisplayName: "I2", InstallDate: time.Now()},
		"I1": {Identifier: "I1", DisplayName: "I1", InstallDate: time.Now()},
	}))

	// Do another trigger + command fetching cycle, since we retry when a profile fails on install.
	s.awaitTriggerProfileSchedule(t)
	mdmDeviceRespond(mdmDevice)

	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(context.Background(), s.ds, host, map[string]*fleet.HostMacOSProfile{
		"I1": {Identifier: "I1", DisplayName: "I1", InstallDate: time.Now()},
		mobileconfig.FleetdConfigPayloadIdentifier:      {Identifier: mobileconfig.FleetdConfigPayloadIdentifier, DisplayName: "dn1", InstallDate: time.Now()},
		mobileconfig.FleetCARootConfigPayloadIdentifier: {Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, DisplayName: "dn2", InstallDate: time.Now()},
	}))

	// Check that the profile is marked as failed when fetching the host
	getHostResp := getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.Profiles)
	require.Len(t, *getHostResp.Host.MDM.Profiles, 4)
	for _, hm := range *getHostResp.Host.MDM.Profiles {
		if hm.Name == "N1" {
			require.Equal(t, fleet.MDMDeliveryFailed, *hm.Status)
			continue
		}

		require.Equal(t, fleet.MDMDeliveryVerified, *hm.Status)
	}

	// transfer host to a team without the failed profile

	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{TeamID: &team.ID, HostIDs: []uint{host.ID}}, http.StatusOK, &addHostsToTeamResponse{})
	s.awaitTriggerProfileSchedule(t)

	// confirm that we remove the failed profile
	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.Profiles)
	require.Len(t, *getHostResp.Host.MDM.Profiles, 3) // This would be 4 if we hadn't deleted the profile that failed to install.
	for _, hm := range *getHostResp.Host.MDM.Profiles {
		require.NotEqual(t, "N1", hm.Name)
	}

	// Test case where the profile never makes it to the host at all
	host, _ = createHostThenEnrollMDM(s.ds, s.server.URL, t)
	ident = uuid.NewString()

	globalProfiles = [][]byte{
		mobileconfigForTest("N3", ident),
	}
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: globalProfiles}, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.Profiles)
	require.Len(t, *getHostResp.Host.MDM.Profiles, 3)
	var profUUID string
	for _, hm := range *getHostResp.Host.MDM.Profiles {
		require.Equal(t, fleet.MDMDeliveryPending, *hm.Status)
		if hm.Name == "N3" {
			profUUID = hm.ProfileUUID
		}
	}

	// delete the custom profile
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/mdm/profiles/%s", profUUID), &deleteMDMAppleConfigProfileRequest{}, http.StatusOK)
	s.awaitTriggerProfileSchedule(t)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.Profiles)
	// Since Fleet doesn't know for sure whether profile was installed or not, it sends a remove command just in case.
	require.Len(t, *getHostResp.Host.MDM.Profiles, 3)
	for _, hm := range *getHostResp.Host.MDM.Profiles {
		require.Equal(t, fleet.MDMDeliveryPending, *hm.Status)
		if hm.Name == "N3" {
			assert.Equal(t, fleet.MDMOperationTypeRemove, hm.OperationType)
		}
	}
}

func (s *integrationMDMTestSuite) TestABMAssetManagement() {
	t := s.T()
	ctx := context.Background()

	s.enableABM(t.Name())

	// Validate error when server private key not set
	testSetEmptyPrivateKey = true
	t.Cleanup(func() { testSetEmptyPrivateKey = false })

	r := s.Do("GET", "/api/latest/fleet/mdm/apple/abm_public_key", generateABMKeyPairResponse{}, http.StatusInternalServerError)
	require.Contains(t, extractServerErrorText(r.Body), "Couldn't download public key. Missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key")
	testSetEmptyPrivateKey = false

	// grab the current public key
	var abmResp generateABMKeyPairResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/abm_public_key", nil, http.StatusOK, &abmResp)
	require.Nil(t, abmResp.Err)
	require.NotEmpty(t, abmResp.PublicKey)

	var tokensResp listABMTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/abm_tokens", nil, http.StatusOK, &tokensResp)
	tok := s.getABMTokenByName(t.Name(), tokensResp.Tokens)

	// disable ABM
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/abm_tokens/%d", tok.ID), nil, http.StatusNoContent)
	tok, err := s.ds.GetABMTokenByOrgName(ctx, t.Name())
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)
	require.Nil(t, tok)

	// try to upload an invalid token
	s.uploadABMToken([]byte("foo"), http.StatusBadRequest, "Please provide a valid token from Apple Business Manager")

	// enable ABM again
	var newABMResp generateABMKeyPairResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/abm_public_key", nil, http.StatusOK, &newABMResp)
	require.Nil(t, newABMResp.Err)
	require.NotEmpty(t, newABMResp.PublicKey)
	block, _ := pem.Decode(newABMResp.PublicKey)
	require.NotNil(t, block)
	require.Equal(t, "CERTIFICATE", block.Type)

	// we should always return the same values to support renewing the token
	var renewABMResp generateABMKeyPairResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/abm_public_key", nil, http.StatusOK, &renewABMResp)
	require.Nil(t, renewABMResp.Err)
	require.NotEmpty(t, renewABMResp.PublicKey)
	require.Equal(t, renewABMResp.PublicKey, newABMResp.PublicKey)

	// simulate a renew flow
	s.enableABM(t.Name())
}

func (s *integrationMDMTestSuite) enableABM(orgName string) *fleet.ABMToken {
	t := s.T()
	var abmResp generateABMKeyPairResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/abm_public_key", nil, http.StatusOK, &abmResp)
	require.Nil(t, abmResp.Err)
	require.NotEmpty(t, abmResp.PublicKey)
	block, _ := pem.Decode(abmResp.PublicKey)
	require.NotNil(t, block)
	require.Equal(t, "CERTIFICATE", block.Type)

	// try to upload an invalid token
	s.uploadABMToken([]byte("foo"), http.StatusBadRequest, "Invalid token. Please provide a valid token from Apple Business Manager.")

	// generate a mock token and encrypt it using the public key
	testBMToken := &nanodep_client.OAuth1Tokens{
		ConsumerKey:       "test_consumer",
		ConsumerSecret:    "test_secret",
		AccessToken:       "test_access_token",
		AccessSecret:      "test_access_secret",
		AccessTokenExpiry: time.Date(2999, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	rawToken, err := json.Marshal(testBMToken)
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
			"\r\n%s", base64.StdEncoding.EncodeToString(encryptedToken))
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
	require.NoError(s.T(), err)
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

func (s *integrationMDMTestSuite) TestSilentMigrationGotchas() {
	t := s.T()
	ctx := context.Background()

	host := createOrbitEnrolledHost(t, "darwin", t.Name(), s.ds)
	// set the host as enrolled in a third-party MDM
	err := s.ds.SetOrUpdateMDMData(ctx, host.ID, true, true, "https://foo.com", false, fleet.WellKnownMDMSimpleMDM, "")
	require.NoError(t, err)

	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.NotNil(t, hostResp.Host)
	require.NotNil(t, hostResp.Host.MDM.ConnectedToFleet)
	require.False(t, *hostResp.Host.MDM.ConnectedToFleet)

	// simulate that the device is assigned to Fleet in ABM
	s.enableABM(t.Name())
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
		case "/profile":
			encoder := json.NewEncoder(w)
			err := encoder.Encode(godep.ProfileResponse{ProfileUUID: "abc"})
			require.NoError(t, err)
		case "/server/devices", "/devices/sync":
			encoder := json.NewEncoder(w)
			err := encoder.Encode(godep.DeviceResponse{
				Devices: []godep.Device{
					{
						SerialNumber: host.HardwareSerial,
						Model:        "Mac Mini",
						OS:           "osx",
						OpType:       "added",
					},
				},
			})
			require.NoError(t, err)

		case "/profile/devices":
			b, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var prof profileAssignmentReq
			require.NoError(t, json.Unmarshal(b, &prof))
			var resp godep.ProfileResponse
			resp.ProfileUUID = prof.ProfileUUID
			resp.Devices = map[string]string{
				prof.Devices[0]: string(fleet.DEPAssignProfileResponseSuccess),
			}
			encoder := json.NewEncoder(w)
			err = encoder.Encode(resp)
			require.NoError(t, err)
		}
	}))
	s.runDEPSchedule()

	// enable migrations
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"mdm": { "macos_migration": { "enable": true, "mode": "voluntary", "webhook_url": "https://example.com" } }
	}`), http.StatusOK, &acResp)

	// orbit config asks for a migration but not to renew enrollment profile
	resp := orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &resp)
	require.False(t, resp.Notifications.RenewEnrollmentProfile)
	require.True(t, resp.Notifications.NeedsMDMMigration)

	// simulate that's actually enrolled to Fleet under the hood
	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "MacBookPro16,1")
	// by default the mdm test client will have a random uuid and serial, but we want
	// it to match with the previously created host
	mdmDevice.SerialNumber = host.HardwareSerial
	mdmDevice.UUID = host.UUID
	err = mdmDevice.Enroll()
	require.NoError(t, err)

	// host response says that it's connected to Fleet (this just checks that the
	// device exists and is enabled in nano_enrollments, which it is thanks to
	// the mdmDevice.Enroll call above - the row is created during TokenUpdate).
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.NotNil(t, hostResp.Host)
	require.NotNil(t, hostResp.Host.MDM.ConnectedToFleet)
	require.True(t, *hostResp.Host.MDM.ConnectedToFleet)

	// orbit config asks for a migration because user migrations are enabled, but no ask to renew the enrollment profile.
	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &resp)
	require.False(t, resp.Notifications.RenewEnrollmentProfile)
	require.True(t, resp.Notifications.NeedsMDMMigration)

	// set an enroll secret so the fleetd profile is delivered
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: t.Name()}},
		},
	}, http.StatusOK, &applyResp)

	// trigger the profile cron
	s.awaitTriggerProfileSchedule(t)

	// explicitly run the worker, which will send the install fleetd command
	// because the device is ADE-enrolled (due to the simulation of it being
	// ingested from ABM)
	s.runWorker()

	var installEnterpriseCount int
	installs := [][]byte{}
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)

	for cmd != nil {
		if cmd.Command.RequestType == "InstallEnterpriseApplication" {
			installEnterpriseCount++
		} else {
			require.Equal(t, "InstallProfile", cmd.Command.RequestType)
			installs = append(installs, cmd.Raw)
		}
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	require.Equal(t, 1, installEnterpriseCount)
	require.Len(t, installs, 2) // fleetd profile and root cert profile

	// trigger the scep renewals cron
	cert, key, err := generateCertWithAPNsTopic()
	require.NoError(t, err)
	fleetCfg := config.TestConfig()
	config.SetTestMDMConfig(s.T(), &fleetCfg, cert, key, "")
	logger := kitlog.NewJSONLogger(os.Stdout)
	err = RenewSCEPCertificates(ctx, logger, s.ds, &fleetCfg, s.mdmCommander)
	require.NoError(t, err)

	// no new commands were enqueued
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	// set the host as completely unenrolled
	err = s.ds.SetOrUpdateMDMData(ctx, host.ID, false, false, "", false, "", "")
	require.NoError(t, err)
	// orbit config asks to renew the enrollment profile, migration is not needed anymore so it's false
	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &resp)
	require.True(t, resp.Notifications.RenewEnrollmentProfile)
	require.False(t, resp.Notifications.NeedsMDMMigration)

	// with migrations disabled, it still asks to renew the enrollment profile
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"mdm": { "macos_migration": { "enable": false } }
	}`), http.StatusOK, &acResp)
	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &resp)
	require.True(t, resp.Notifications.RenewEnrollmentProfile)
	require.False(t, resp.Notifications.NeedsMDMMigration)
}

func (s *integrationMDMTestSuite) TestAPNsPushCron() {
	t := s.T()
	ctx := context.Background()

	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
		{Name: "N2", Contents: syncMLForTest("./Foo/Bar")},
		{Name: "N4", Contents: declarationForTest("D1")},
	}}, http.StatusNoContent)

	// macOS host, MDM on
	_, macDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	// windows host, MDM on
	createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
	// linux and darwin, MDM off
	createOrbitEnrolledHost(t, "linux", "linux_host", s.ds)
	createOrbitEnrolledHost(t, "darwin", "mac_not_enrolled", s.ds)

	// we're going to modify this mock, make sure we restore its default
	originalPushMock := s.pushProvider.PushFunc
	defer func() { s.pushProvider.PushFunc = originalPushMock }()

	var recordedPushes []*mdm.Push
	var mu sync.Mutex
	s.pushProvider.PushFunc = func(ctx context.Context, pushes []*mdm.Push) (map[string]*push.Response, error) {
		mu.Lock()
		defer mu.Unlock()
		recordedPushes = pushes
		return mockSuccessfulPush(ctx, pushes)
	}

	// trigger the reconciliation schedule
	err := ReconcileAppleProfiles(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)
	require.Len(t, recordedPushes, 1)
	recordedPushes = nil

	// triggering the schedule again doesn't send any more pushes
	err = ReconcileAppleProfiles(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)
	require.Len(t, recordedPushes, 0)
	recordedPushes = nil

	// the cron to trigger pushes sends a new push request each time it
	// runs if there are pending commands
	for i := 0; i < 3; i++ {
		err := SendPushesToPendingDevices(ctx, s.ds, s.mdmCommander, s.logger)
		require.NoError(t, err)
		require.Len(t, recordedPushes, 1)
		recordedPushes = nil
	}

	// device acknowledges the commands
	cmd, err := macDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	for cmd != nil {
		cmd, err = macDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	// no more pushes are enqueued
	err = SendPushesToPendingDevices(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)
	require.Len(t, recordedPushes, 0)
}

func (s *integrationMDMTestSuite) TestAPNsPushWithNotNow() {
	t := s.T()
	ctx := context.Background()

	// macOS host, MDM on
	_, macDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	// windows host, MDM on
	createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
	// linux and darwin, MDM off
	createOrbitEnrolledHost(t, "linux", "linux_host", s.ds)
	createOrbitEnrolledHost(t, "darwin", "mac_not_enrolled", s.ds)

	// we're going to modify this mock, make sure we restore its default
	originalPushMock := s.pushProvider.PushFunc
	defer func() { s.pushProvider.PushFunc = originalPushMock }()

	var recordedPushes []*mdm.Push
	var mu sync.Mutex
	s.pushProvider.PushFunc = func(ctx context.Context, pushes []*mdm.Push) (map[string]*push.Response, error) {
		mu.Lock()
		defer mu.Unlock()
		recordedPushes = pushes
		return mockSuccessfulPush(ctx, pushes)
	}

	// trigger the reconciliation schedule
	err := ReconcileAppleProfiles(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)
	require.Len(t, recordedPushes, 1)
	recordedPushes = nil

	// Flush any existing profiles.
	cmd, err := macDevice.Idle()
	require.NoError(t, err)
	for {
		if cmd == nil {
			break
		}
		t.Logf("Received: %s %s", cmd.CommandUUID, cmd.Command.RequestType)
		cmd, err = macDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	// Load new profiles
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
		{Name: "N2", Contents: syncMLForTest("./Foo/Bar")},
	}}, http.StatusNoContent)

	// trigger the reconciliation schedule
	err = ReconcileAppleProfiles(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)
	require.Len(t, recordedPushes, 1)
	recordedPushes = nil

	// The cron to trigger pushes sends a new push request each time it runs.
	err = SendPushesToPendingDevices(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)
	require.Len(t, recordedPushes, 1)
	recordedPushes = nil

	// device sends 'NotNow'
	cmd, err = macDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	cmd, err = macDevice.NotNow(cmd.CommandUUID)
	require.NoError(t, err)
	assert.Nil(t, cmd)

	// A 'NotNow' command will not trigger a new push. Device is expected to check in again when conditions change.
	err = ReconcileAppleProfiles(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)
	require.Len(t, recordedPushes, 0)
	recordedPushes = nil

	// device acknowledges the commands
	cmd, err = macDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	cmd, err = macDevice.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)
	assert.Nil(t, cmd)

	// no more pushes are enqueued
	err = SendPushesToPendingDevices(ctx, s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)
	assert.Zero(t, recordedPushes)
}

func (s *integrationMDMTestSuite) TestMDMRequestWithoutCerts() {
	t := s.T()
	res := s.DoRawNoAuth("PUT", "/mdm/apple/mdm", nil, http.StatusBadRequest)
	require.NoError(t, res.Body.Close())
}

func (s *integrationMDMTestSuite) TestMDMEnrollDoesntClearLastEnrolledAtForMacOS() {
	t := s.T()

	// Enroll to Fleet with fleetd first.
	desktopToken := uuid.New().String()
	lastEnrolledAt := time.Now().UTC()
	mdmDevice := mdmtest.NewTestMDMClientAppleDesktopManual(s.server.URL, desktopToken)
	fleetHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		LastEnrolledAt:  lastEnrolledAt,
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name() + uuid.New().String()),
		NodeKey:         ptr.String(t.Name() + uuid.New().String()),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",

		UUID:           mdmDevice.UUID,
		HardwareSerial: mdmDevice.SerialNumber,
	})
	require.NoError(t, err)
	err = s.ds.SetOrUpdateDeviceAuthToken(context.Background(), fleetHost.ID, desktopToken)
	require.NoError(t, err)

	// Enroll with MDM manually after.
	err = mdmDevice.Enroll()
	require.NoError(t, err)

	hostResp := getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", fleetHost.ID), nil, http.StatusOK, &hostResp)
	require.NotNil(t, hostResp.Host)
	require.Equal(t, lastEnrolledAt.Truncate(1*time.Second).UTC(), hostResp.Host.LastEnrolledAt.Truncate(1*time.Second).UTC())
}

func (s *integrationMDMTestSuite) TestConnectedToFleetWithoutCheckout() {
	t := s.T()
	ctx := context.Background()

	host := createOrbitEnrolledHost(t, "darwin", t.Name(), s.ds)

	// simulate that's actually enrolled to Fleet under the hood
	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "MacBookPro16,1")
	mdmDevice.UUID = host.UUID
	mdmDevice.SerialNumber = host.HardwareSerial
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	// but, set the host as enrolled in a third-party MDM
	err = s.ds.SetOrUpdateMDMData(ctx, host.ID, true, true, "https://foo.com", false, fleet.WellKnownMDMSimpleMDM, "")
	require.NoError(t, err)

	// host is connected to Fleet
	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.NotNil(t, hostResp.Host)
	require.NotNil(t, hostResp.Host.MDM.ConnectedToFleet)
	require.True(t, *hostResp.Host.MDM.ConnectedToFleet)

	// now simulate an un-enrollment without checkout, in this case, osquery reports the host as not-enrolled
	err = s.ds.SetOrUpdateMDMData(ctx, host.ID, false, false, "", false, "", "")
	require.NoError(t, err)

	// host is not connected to Fleet anymore
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.NotNil(t, hostResp.Host)
	require.NotNil(t, hostResp.Host.MDM.ConnectedToFleet)
	require.False(t, *hostResp.Host.MDM.ConnectedToFleet)
}

func (s *integrationMDMTestSuite) TestBatchAssociateAppStoreApps() {
	t := s.T()
	batchURL := "/api/latest/fleet/software/app_store_apps/batch"

	// non-existent team
	s.Do("POST", batchURL, batchAssociateAppStoreAppsRequest{}, http.StatusNotFound, "team_name", "foo")

	// create a team
	tmGood, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name() + " good",
		Description: "desc",
	})
	require.NoError(t, err)

	// create a team
	tmEmpty, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name() + " empty",
		Description: "desc",
	})
	require.NoError(t, err)

	// No vpp token set, but request is empty so it succeeds (clears VPP apps for the team).
	s.Do("POST", batchURL, batchAssociateAppStoreAppsRequest{}, http.StatusNoContent, "team_name", tmGood.Name)

	// No vpp token set, try association
	// FIXME
	// s.Do("POST", batchURL, batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{{AppStoreID: s.appleVPPConfigSrvConfig.Assets[0].AdamID}}}, http.StatusUnprocessableEntity, "team_name", tmGood.Name)

	// Valid token
	orgName := "Fleet Device Management Inc."
	token := "mycooltoken"
	expDate := "2025-06-24T15:50:50+0000"
	tokenJSON := fmt.Sprintf(`{"expDate":"%s","token":"%s","orgName":"%s"}`, expDate, token, orgName)
	t.Setenv("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL)
	var vppRes uploadVPPTokenResponse
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken", []byte(base64.StdEncoding.EncodeToString([]byte(tokenJSON))), http.StatusAccepted, "", &vppRes)

	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", vppRes.Token.ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{}}, http.StatusOK, &resPatchVPP)

	// Remove all vpp associations from team with no members
	s.Do("POST", batchURL, batchAssociateAppStoreAppsRequest{}, http.StatusNoContent, "team_name", tmGood.Name)

	// host with valid serial number
	hValid, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		HardwareSerial:  "123",
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name() + uuid.New().String()),
		NodeKey:         ptr.String(t.Name() + uuid.New().String()),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)
	err = s.ds.AddHostsToTeam(context.Background(), &tmGood.ID, []uint{hValid.ID})
	require.NoError(t, err)

	ctx := context.Background()

	assoc, err := s.ds.GetAssignedVPPApps(ctx, &tmGood.ID)
	require.NoError(t, err)
	require.Len(t, assoc, 0)

	// Remove all vpp associations from team with no members
	s.Do("POST", batchURL, batchAssociateAppStoreAppsRequest{}, http.StatusNoContent, "team_name", tmGood.Name)

	// Incorrect type check
	incorrectTypes := struct {
		Apps []struct {
			AppStoreID  int  `json:"app_store_id"`
			SelfService bool `json:"self_service"`
		} `json:"app_store_apps"`
	}{
		Apps: []struct {
			AppStoreID  int  `json:"app_store_id"`
			SelfService bool `json:"self_service"`
		}{
			{
				AppStoreID: 1,
			},
		},
	}
	badTypeReq := s.Do("POST", batchURL, incorrectTypes, http.StatusBadRequest, "team_name", tmGood.Name)
	badTypeBody := extractServerErrorText(badTypeReq.Body)
	assert.Contains(t, badTypeBody, "must be a string")

	// Associating an app we don't own
	s.Do("POST", batchURL, batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{{AppStoreID: "fake-app"}}}, http.StatusUnprocessableEntity, "team_name", tmGood.Name)

	assoc, err = s.ds.GetAssignedVPPApps(ctx, &tmGood.ID)
	require.NoError(t, err)
	require.Len(t, assoc, 0)

	// Associating an app we own
	s.Do("POST", batchURL, batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{{AppStoreID: s.appleVPPConfigSrvConfig.Assets[0].AdamID}}}, http.StatusNoContent, "team_name", tmGood.Name)

	assoc, err = s.ds.GetAssignedVPPApps(ctx, &tmGood.ID)
	require.NoError(t, err)
	require.Len(t, assoc, 1)
	assert.Contains(t, assoc, fleet.VPPAppID{AdamID: s.appleVPPConfigSrvConfig.Assets[0].AdamID, Platform: fleet.MacOSPlatform})

	// Associating one good and one bad app
	s.Do("POST",
		batchURL,
		batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{
			{AppStoreID: s.appleVPPConfigSrvConfig.Assets[0].AdamID},
			{AppStoreID: "fake-app"},
		}}, http.StatusUnprocessableEntity, "team_name", tmGood.Name,
	)

	assoc, err = s.ds.GetAssignedVPPApps(ctx, &tmGood.ID)
	require.NoError(t, err)
	require.Len(t, assoc, 1)

	// Associating two apps we own
	s.Do("POST",
		batchURL,
		batchAssociateAppStoreAppsRequest{
			Apps: []fleet.VPPBatchPayload{
				{AppStoreID: s.appleVPPConfigSrvConfig.Assets[0].AdamID},
				{AppStoreID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, SelfService: true},
			},
		}, http.StatusNoContent, "team_name", tmGood.Name,
	)
	assoc, err = s.ds.GetAssignedVPPApps(ctx, &tmGood.ID)
	require.NoError(t, err)
	require.Len(t, assoc, 4)
	assert.Contains(t, assoc, fleet.VPPAppID{AdamID: s.appleVPPConfigSrvConfig.Assets[0].AdamID, Platform: fleet.MacOSPlatform})
	assert.Contains(t, assoc, fleet.VPPAppID{AdamID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, Platform: fleet.IOSPlatform})
	assert.Contains(t, assoc, fleet.VPPAppID{AdamID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, Platform: fleet.IPadOSPlatform})
	// Only macOS version should be self-service
	assert.Equal(t, fleet.VPPAppTeam{
		VPPAppID:           fleet.VPPAppID{AdamID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, Platform: fleet.MacOSPlatform},
		SelfService:        true,
		InstallDuringSetup: ptr.Bool(false),
	},
		assoc[fleet.VPPAppID{AdamID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, Platform: fleet.MacOSPlatform}])

	// Reverse self-service associations
	// Associating two apps we own
	s.Do("POST",
		batchURL,
		batchAssociateAppStoreAppsRequest{
			Apps: []fleet.VPPBatchPayload{
				{AppStoreID: s.appleVPPConfigSrvConfig.Assets[0].AdamID, SelfService: true},
				{AppStoreID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, SelfService: false},
			},
		}, http.StatusNoContent, "team_name", tmGood.Name,
	)
	assoc, err = s.ds.GetAssignedVPPApps(ctx, &tmGood.ID)
	require.NoError(t, err)
	require.Len(t, assoc, 4)
	assert.Equal(t, fleet.VPPAppTeam{
		VPPAppID:           fleet.VPPAppID{AdamID: s.appleVPPConfigSrvConfig.Assets[0].AdamID, Platform: fleet.MacOSPlatform},
		SelfService:        true,
		InstallDuringSetup: ptr.Bool(false),
	}, assoc[fleet.VPPAppID{AdamID: s.appleVPPConfigSrvConfig.Assets[0].AdamID, Platform: fleet.MacOSPlatform}])
	assert.Equal(t, fleet.VPPAppTeam{
		VPPAppID:           fleet.VPPAppID{AdamID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, Platform: fleet.IOSPlatform},
		InstallDuringSetup: ptr.Bool(false),
	}, assoc[fleet.VPPAppID{AdamID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, Platform: fleet.IOSPlatform}])
	assert.Equal(t, fleet.VPPAppTeam{
		VPPAppID:           fleet.VPPAppID{AdamID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, Platform: fleet.IPadOSPlatform},
		InstallDuringSetup: ptr.Bool(false),
	}, assoc[fleet.VPPAppID{AdamID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, Platform: fleet.IPadOSPlatform}])
	assert.Equal(t, fleet.VPPAppTeam{
		VPPAppID:           fleet.VPPAppID{AdamID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, Platform: fleet.MacOSPlatform},
		InstallDuringSetup: ptr.Bool(false),
	}, assoc[fleet.VPPAppID{AdamID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, Platform: fleet.MacOSPlatform}])

	// Associate an app with a team with no team members
	s.Do("POST", batchURL, batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{{AppStoreID: s.appleVPPConfigSrvConfig.Assets[0].AdamID}}}, http.StatusNoContent, "team_name", tmEmpty.Name)

	// Remove all vpp associations
	s.Do("POST", batchURL, batchAssociateAppStoreAppsRequest{}, http.StatusNoContent, "team_name", tmGood.Name)

	assoc, err = s.ds.GetAssignedVPPApps(ctx, &tmGood.ID)
	require.NoError(t, err)
	require.Len(t, assoc, 0)
}

func (s *integrationMDMTestSuite) TestInvalidCommandUUID() {
	t := s.T()
	_, device := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	s.runWorker()
	cmd, err := device.Acknowledge("foo")
	require.NoError(t, err)
	require.NotNil(t, cmd)
}

func (s *integrationMDMTestSuite) TestEnrollAfterDEPSyncIOSIPadOS() {
	t := s.T()

	h, _ := s.createAppleMobileHostThenEnrollMDM("ios")

	// fetch the host, it will match the one created above
	// (NOTE: cannot check the returned OrbitNodeKey, this field is not part of the response)
	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", h.ID), nil, http.StatusOK, &hostResp)
	require.Equal(t, h.ID, hostResp.Host.ID)
	require.NotEqual(t, h.LastEnrolledAt, hostResp.Host.LastEnrolledAt)

	h, _ = s.createAppleMobileHostThenEnrollMDM("ipados")

	// fetch the host, it will match the one created above
	// (NOTE: cannot check the returned OrbitNodeKey, this field is not part of the response)
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", h.ID), nil, http.StatusOK, &hostResp)
	require.Equal(t, h.ID, hostResp.Host.ID)
	require.NotEqual(t, h.LastEnrolledAt, hostResp.Host.LastEnrolledAt)

	// list commands returns empty set -- no MDM commands should be queued at this point
	var listCmdResp listMDMAppleCommandsResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/commands", nil, http.StatusOK, &listCmdResp)
	require.Empty(t, listCmdResp.Results)
}

func (s *integrationMDMTestSuite) TestRefetchIOSIPadOS() {
	t := s.T()

	// Try to refetch host that is not MDM enrolled
	serialNumber := mdmtest.RandSerialNumber()
	fleetHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		HardwareSerial:   serialNumber,
		Platform:         "ipados",
		LastEnrolledAt:   time.Now(),
		DetailUpdatedAt:  time.Now(),
		RefetchRequested: true,
	})
	require.NoError(t, err)
	r := s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/refetch", fleetHost.ID), nil, http.StatusUnprocessableEntity, "error",
		"host is not enrolled in MDM")
	assert.Contains(t, extractServerErrorText(r.Body), "Host does not have MDM turned on")

	// Enroll host
	host, mdmClient := s.createAppleMobileHostThenEnrollMDM("ios")
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), host.ID, false, true, "https://foo.com", true, "", ""))

	// Refetch host
	_ = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/refetch", host.ID), nil, http.StatusOK)
	const commandsSentPerRefetch = 2
	commandsSent := commandsSentPerRefetch

	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	assert.Equal(t, host.ID, hostResp.Host.ID)
	assert.True(t, hostResp.Host.RefetchRequested)

	commands, err := s.ds.GetHostMDMCommands(context.Background(), host.ID)
	require.NoError(t, err)
	require.Len(t, commands, commandsSent)
	assert.ElementsMatch(t, []fleet.HostMDMCommand{
		{HostID: host.ID, CommandType: fleet.RefetchAppsCommandUUIDPrefix},
		{HostID: host.ID, CommandType: fleet.RefetchDeviceCommandUUIDPrefix},
	}, commands)

	// Since refetch is already queued up, doing another refetch is a no-op and will not add more MDM commands
	_ = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/refetch", host.ID), nil, http.StatusOK)

	// Check the MDM commands and send response
	cmd, err := mdmClient.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)

	const deviceName = "My iPhone"
	expectedSoftware := []fleet.HostSoftwareEntry{
		{
			Software: fleet.Software{
				BundleIdentifier: "com.evernote.iPhone.Evernote",
				Name:             "Evernote",
				Version:          "10.98.0",
				Source:           "ios_apps",
			},
		},
	}
	require.Equal(t, "InstalledApplicationList", cmd.Command.RequestType)
	cmd, err = mdmClient.AcknowledgeInstalledApplicationList(mdmClient.UUID, cmd.CommandUUID,
		[]fleet.Software{expectedSoftware[0].Software})
	require.NoError(t, err)
	require.Equal(t, "DeviceInformation", cmd.Command.RequestType)
	_, err = mdmClient.AcknowledgeDeviceInformation(mdmClient.UUID, cmd.CommandUUID, deviceName, "iPhone SE")
	require.NoError(t, err)

	commands, err = s.ds.GetHostMDMCommands(context.Background(), host.ID)
	require.NoError(t, err)
	require.Empty(t, commands)

	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	assert.Equal(t, host.ID, hostResp.Host.ID)
	assert.False(t, hostResp.Host.RefetchRequested)
	assert.Equal(t, deviceName, hostResp.Host.ComputerName)

	for index := range hostResp.Host.Software {
		hostResp.Host.Software[index].ID = 0
	}
	assert.ElementsMatch(t, expectedSoftware, hostResp.Host.Software)

	// Install the same app for iPadOS
	hostIPad, mdmClientIPad := s.createAppleMobileHostThenEnrollMDM("ipados")

	// Refetch host
	_ = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/refetch", hostIPad.ID), nil, http.StatusOK)
	commandsSent += commandsSentPerRefetch

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostIPad.ID), nil, http.StatusOK, &hostResp)
	assert.Equal(t, hostIPad.ID, hostResp.Host.ID)
	assert.True(t, hostResp.Host.RefetchRequested)

	// Check the MDM commands and send response
	cmd, err = mdmClientIPad.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)

	const deviceNameIPad = "My iPad"
	expectedSoftware = []fleet.HostSoftwareEntry{
		{
			Software: fleet.Software{
				BundleIdentifier: "com.evernote.iPhone.Evernote",
				Name:             "Evernote",
				Version:          "10.98.0",
				Source:           "ipados_apps",
			},
		},
	}
	require.Equal(t, "InstalledApplicationList", cmd.Command.RequestType)
	cmd, err = mdmClientIPad.AcknowledgeInstalledApplicationList(mdmClientIPad.UUID, cmd.CommandUUID,
		[]fleet.Software{expectedSoftware[0].Software})
	require.NoError(t, err)
	require.Equal(t, "DeviceInformation", cmd.Command.RequestType)
	cmd, err = mdmClientIPad.AcknowledgeDeviceInformation(mdmClientIPad.UUID, cmd.CommandUUID, deviceNameIPad, "iPad 10")
	require.NoError(t, err)
	require.Nil(t, cmd)

	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostIPad.ID), nil, http.StatusOK, &hostResp)
	assert.Equal(t, hostIPad.ID, hostResp.Host.ID)
	assert.False(t, hostResp.Host.RefetchRequested)
	assert.Equal(t, deviceNameIPad, hostResp.Host.ComputerName)

	for index := range hostResp.Host.Software {
		hostResp.Host.Software[index].ID = 0
	}
	assert.ElementsMatch(t, expectedSoftware, hostResp.Host.Software)

	hostsCountTs := time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(context.Background(), hostsCountTs))
	ctx := context.Background()
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, hostsCountTs))

	// Check that we have the correct software titles
	var resp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &resp, "query", "Evernote")
	expectedTitles := []fleet.SoftwareTitleListResult{
		{
			BundleIdentifier: ptr.String("com.evernote.iPhone.Evernote"),
			Name:             "Evernote",
			Source:           "ios_apps",
			HostsCount:       1,
			VersionsCount:    1,
		},
		{
			BundleIdentifier: ptr.String("com.evernote.iPhone.Evernote"),
			Name:             "Evernote",
			Source:           "ipados_apps",
			HostsCount:       1,
			VersionsCount:    1,
		},
	}
	require.Len(t, resp.SoftwareTitles, 2)
	// Cleaning up the response to make it easier to compare using ElementsMatch
	for index := range resp.SoftwareTitles {
		resp.SoftwareTitles[index].ID = 0
		assert.Len(t, resp.SoftwareTitles[index].Versions, 1)
		resp.SoftwareTitles[index].Versions = nil
	}
	assert.ElementsMatch(t, expectedTitles, resp.SoftwareTitles)

	// Delete from software titles table and make sure titles are recreated
	mysql.ExecAdhocSQL(s.T(), s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(context.Background(), "DELETE FROM software_titles")
		return err
	})
	hostsCountTs = time.Now().UTC()
	require.NoError(t, s.ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, hostsCountTs))
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &resp, "query", "Evernote")
	require.Len(t, resp.SoftwareTitles, 2)
	// Cleaning up the response to make it easier to compare using ElementsMatch
	for index := range resp.SoftwareTitles {
		resp.SoftwareTitles[index].ID = 0
		assert.Len(t, resp.SoftwareTitles[index].Versions, 1)
		resp.SoftwareTitles[index].Versions = nil
	}
	assert.ElementsMatch(t, expectedTitles, resp.SoftwareTitles)

	// Test that software was deleted from device
	_ = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/refetch", host.ID), nil, http.StatusOK)
	commandsSent += commandsSentPerRefetch

	// Check the MDM commands and send response
	cmd, err = mdmClient.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "InstalledApplicationList", cmd.Command.RequestType)
	cmd, err = mdmClient.AcknowledgeInstalledApplicationList(mdmClient.UUID, cmd.CommandUUID, []fleet.Software{})
	require.NoError(t, err)
	require.Equal(t, "DeviceInformation", cmd.Command.RequestType)
	const deviceNameRenamed = "My new iPhone"
	cmd, err = mdmClient.AcknowledgeDeviceInformation(mdmClient.UUID, cmd.CommandUUID, deviceNameRenamed, "iPhone SE")
	require.NoError(t, err)
	require.Nil(t, cmd)

	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	assert.Equal(t, host.ID, hostResp.Host.ID)
	assert.False(t, hostResp.Host.RefetchRequested)
	assert.Equal(t, deviceNameRenamed, hostResp.Host.ComputerName)
	assert.Empty(t, hostResp.Host.Software)

	// Mark host as unenrolled and refetch.
	require.NoError(t, s.ds.UpdateMDMData(ctx, host.ID, false))
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.NoError(t, err)
	require.NotNil(t, hostResp.Host.MDM.EnrollmentStatus)
	assert.Equal(t, "Pending", *hostResp.Host.MDM.EnrollmentStatus)

	// Set iOS detail_updated_at as 2 hours in the past.
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE hosts SET detail_updated_at = DATE_SUB(NOW(), INTERVAL 2 HOUR) WHERE id = ?`, host.ID)
		return err
	})
	trigger := triggerRequest{
		Name: string(fleet.CronAppleMDMIPhoneIPadRefetcher),
	}
	_ = s.Do("POST", "/api/latest/fleet/trigger", trigger, http.StatusOK)
	commandsSent += commandsSentPerRefetch

	// Wait until MDM commands are set up
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			commands, err = s.ds.GetHostMDMCommands(context.Background(), host.ID)
			require.NoError(t, err)
			if len(commands) == commandsSentPerRefetch {
				done <- struct{}{}
				return
			}
		}
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Error("Timeout: MDM commands not queued up")
	}

	// Check the MDM commands and send response
	cmd, err = mdmClient.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "InstalledApplicationList", cmd.Command.RequestType)
	cmd, err = mdmClient.AcknowledgeInstalledApplicationList(mdmClient.UUID, cmd.CommandUUID, []fleet.Software{})
	require.NoError(t, err)
	require.Equal(t, "DeviceInformation", cmd.Command.RequestType)
	cmd, err = mdmClient.AcknowledgeDeviceInformation(mdmClient.UUID, cmd.CommandUUID, deviceNameRenamed, "iPhone SE")
	require.NoError(t, err)
	require.Nil(t, cmd)

	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	assert.Equal(t, host.ID, hostResp.Host.ID)
	assert.False(t, hostResp.Host.RefetchRequested)
	require.NotNil(t, hostResp.Host.MDM.EnrollmentStatus)
	assert.Equal(t, "On (automatic)", *hostResp.Host.MDM.EnrollmentStatus)

	// list commands should return all the commands we sent
	var listCmdResp listMDMAppleCommandsResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/commands", nil, http.StatusOK, &listCmdResp)
	require.Len(t, listCmdResp.Results, commandsSent)
}

func (s *integrationMDMTestSuite) TestVPPApps() {
	t := s.T()
	s.setSkipWorkerJobs(t)

	// Invalid token
	t.Setenv("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL+"?invalidToken")
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken", []byte("foobar"), http.StatusUnprocessableEntity, "Invalid token. Please provide a valid content token from Apple Business Manager.", nil)

	// Simulate a server error from the Apple API
	t.Setenv("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL+"?serverError")
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken", []byte("foobar"), http.StatusInternalServerError, "Apple VPP endpoint returned error: Internal server error (error number: 9603)", nil)

	// Valid token
	orgName := "Fleet Device Management Inc."
	location := "Fleet Location One"
	token := "mycooltoken"
	expTime := time.Now().Add(200 * time.Hour).UTC().Round(time.Second)
	expDate := expTime.Format(fleet.VPPTimeFormat)
	tokenJSON := fmt.Sprintf(`{"expDate":"%s","token":"%s","orgName":"%s"}`, expDate, token, orgName)
	t.Setenv("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL)
	var validToken uploadVPPTokenResponse
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken", []byte(base64.StdEncoding.EncodeToString([]byte(tokenJSON))), http.StatusAccepted, "", &validToken)

	s.lastActivityMatches(fleet.ActivityEnabledVPP{}.ActivityName(), "", 0)

	// Get the token
	var resp getVPPTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &resp)
	require.NoError(t, resp.Err)
	require.Len(t, resp.Tokens, 1)
	require.Equal(t, orgName, resp.Tokens[0].OrgName)
	require.Equal(t, location, resp.Tokens[0].Location)
	require.Equal(t, expTime, resp.Tokens[0].RenewDate)

	// Create a team
	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: ptr.String("Team 1")}}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

	// Associate team to the VPP token.
	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", resp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID}}, http.StatusOK, &resPatchVPP)

	// A PATCH endpoint omitting mdm.volume_purchasing_program should not remove the VPP token association.
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": {
			"config": {
				"options": {
					"pack_delimiter": "/",
					"logger_tls_period": 10,
					"distributed_plugin": "tls",
					"disable_distributed": false,
					"logger_tls_endpoint": "/api/osquery/log",
					"distributed_interval": 10,
					"distributed_tls_max_attempts": 3
				}
			}
		}
  }`), http.StatusOK, &acResp)

	// Check that the VPP token is still associated to the team.
	resp = getVPPTokensResponse{}
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &resp)
	require.NoError(t, resp.Err)
	require.Len(t, resp.Tokens, 1)
	require.Equal(t, orgName, resp.Tokens[0].OrgName)
	require.Equal(t, location, resp.Tokens[0].Location)
	require.Equal(t, expTime, resp.Tokens[0].RenewDate)
	require.Len(t, resp.Tokens[0].Teams, 1)
	require.Equal(t, team.ID, resp.Tokens[0].Teams[0].ID)
	require.Equal(t, team.Name, resp.Tokens[0].Teams[0].Name)

	// Reset the token's teams by omitting the token from app config
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "volume_purchasing_program": [] }
  }`), http.StatusOK, &acResp)

	resp = getVPPTokensResponse{}
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &resp)
	require.NoError(t, resp.Err)
	require.Len(t, resp.Tokens, 1)
	require.Equal(t, orgName, resp.Tokens[0].OrgName)
	require.Equal(t, location, resp.Tokens[0].Location)
	require.Equal(t, expTime, resp.Tokens[0].RenewDate)
	require.Empty(t, resp.Tokens[0].Teams)

	// Add the team back using the PATCH /api/latest/fleet/config endpoint now (what GitOps uses).
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": { "volume_purchasing_program": [ {"location": "%s", "teams": [ "%s" ]} ] }
  }`, location, team.Name)), http.StatusOK, &acResp)

	// Check again that the VPP token is associated to the team.
	resp = getVPPTokensResponse{}
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &resp)
	require.NoError(t, resp.Err)
	require.Len(t, resp.Tokens, 1)
	require.Equal(t, orgName, resp.Tokens[0].OrgName)
	require.Equal(t, location, resp.Tokens[0].Location)
	require.Equal(t, expTime, resp.Tokens[0].RenewDate)
	require.Len(t, resp.Tokens[0].Teams, 1)
	require.Equal(t, team.ID, resp.Tokens[0].Teams[0].ID)
	require.Equal(t, team.Name, resp.Tokens[0].Teams[0].Name)

	// Get list of VPP apps from "Apple"
	// We're passing team 1 here, but we haven't added any app store apps to that team, so we get
	// back all available apps in our VPP location.
	var appResp getAppStoreAppsResponse
	s.DoJSON("GET", "/api/latest/fleet/software/app_store_apps", &getAppStoreAppsRequest{}, http.StatusOK, &appResp, "team_id",
		fmt.Sprint(team.ID))
	require.NoError(t, appResp.Err)

	macOSApp := fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "1",
				Platform: fleet.MacOSPlatform,
			},
		},
		Name:             "App 1",
		BundleIdentifier: "a-1",
		IconURL:          "https://example.com/images/1",
		LatestVersion:    "1.0.0",
	}
	iPadOSApp := fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "2",
				Platform: fleet.IPadOSPlatform,
			},
		},
		Name:             "App 2",
		BundleIdentifier: "b-2",
		IconURL:          "https://example.com/images/2",
		LatestVersion:    "2.0.0",
	}
	iOSApp := fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "2",
				Platform: fleet.IOSPlatform,
			},
		},
		Name:             "App 2",
		BundleIdentifier: "b-2",
		IconURL:          "https://example.com/images/2",
		LatestVersion:    "2.0.0",
	}
	expectedApps := []*fleet.VPPApp{
		&macOSApp,
		&iPadOSApp,
		&iOSApp,
		{
			VPPAppTeam: fleet.VPPAppTeam{
				VPPAppID: fleet.VPPAppID{
					AdamID:   "2",
					Platform: fleet.MacOSPlatform,
				},
			},
			Name:             "App 2",
			BundleIdentifier: "b-2",
			IconURL:          "https://example.com/images/2",
			LatestVersion:    "2.0.0",
		},
		{
			VPPAppTeam: fleet.VPPAppTeam{
				VPPAppID: fleet.VPPAppID{
					AdamID:   "3",
					Platform: fleet.IPadOSPlatform,
				},
			},
			Name:             "App 3",
			BundleIdentifier: "c-3",
			IconURL:          "https://example.com/images/3",
			LatestVersion:    "3.0.0",
		},
	}
	assert.ElementsMatch(t, expectedApps, appResp.AppStoreApps)

	getSoftwareTitleIDFromApp := func(app *fleet.VPPApp) uint {
		var titleID uint
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			ctx := context.Background()
			return sqlx.GetContext(ctx, q, &titleID, `SELECT title_id FROM vpp_apps WHERE adam_id = ? AND platform = ?;`, app.AdamID, app.Platform)
		})

		return titleID
	}

	// Insert/deletion flow for macOS app
	// Add an app store app to team 1
	addedApp := expectedApps[0]
	var addAppResp addAppStoreAppResponse
	// Add an app store app to non-existent team
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: ptr.Uint(9999), AppStoreID: addedApp.AdamID}, http.StatusNotFound, &addAppResp)

	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: addedApp.AdamID, SelfService: true}, http.StatusOK, &addAppResp)
	s.lastActivityMatches(fleet.ActivityAddedAppStoreApp{}.ActivityName(),
		fmt.Sprintf(`{"team_name": "%s", "software_title": "%s", "software_title_id": %d, "app_store_id": "%s", "team_id": %d, "platform": "%s", "self_service": true}`, team.Name,
			addedApp.Name, getSoftwareTitleIDFromApp(addedApp), addedApp.AdamID, team.ID, addedApp.Platform), 0)

	// Now we should be filtering out the app we added to team 1
	appResp = getAppStoreAppsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/app_store_apps", &getAppStoreAppsRequest{}, http.StatusOK, &appResp, "team_id",
		fmt.Sprint(team.ID))
	require.NoError(t, appResp.Err)
	assert.ElementsMatch(t, expectedApps[1:], appResp.AppStoreApps)

	// list the software titles for that team, to get the title id of the VPP app
	var listSw listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID), "available_for_install", "true")
	require.Len(t, listSw.SoftwareTitles, 1)
	require.True(t, *listSw.SoftwareTitles[0].AppStoreApp.SelfService)
	macOSTitleID := listSw.SoftwareTitles[0].ID

	// listing with the self-service filter also returns it
	listSw = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID), "self_service", "true")
	require.Len(t, listSw.SoftwareTitles, 1)
	require.Equal(t, macOSTitleID, listSw.SoftwareTitles[0].ID)

	// delete the app store app for team 1
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", macOSTitleID), nil, http.StatusNoContent,
		"team_id", fmt.Sprint(team.ID))
	s.lastActivityMatches(fleet.ActivityDeletedAppStoreApp{}.ActivityName(),
		fmt.Sprintf(`{"team_name": "%s", "software_title": "%s", "app_store_id": "%s", "team_id": %d, "platform": "%s"}`, team.Name,
			addedApp.Name, addedApp.AdamID, team.ID, addedApp.Platform), 0)

	// deleting it again fails, not found
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", macOSTitleID), nil, http.StatusNotFound,
		"team_id", fmt.Sprint(team.ID))

	// get the list of available apps, returns all apps now
	appResp = getAppStoreAppsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/app_store_apps", nil, http.StatusOK, &appResp, "team_id", fmt.Sprint(team.ID))
	require.NoError(t, appResp.Err)
	assert.ElementsMatch(t, expectedApps, appResp.AppStoreApps)

	// Insert/deletion flow for iPadOS app
	addedApp = expectedApps[1]
	addAppResp = addAppStoreAppResponse{}
	// No self-service for iPadOS
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps",
		&addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: addedApp.AdamID, Platform: addedApp.Platform, SelfService: true},
		http.StatusBadRequest, &addAppResp)
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps",
		&addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: addedApp.AdamID, Platform: addedApp.Platform},
		http.StatusOK, &addAppResp)
	s.lastActivityMatches(fleet.ActivityAddedAppStoreApp{}.ActivityName(),
		fmt.Sprintf(`{"team_name": "%s", "software_title": "%s", "software_title_id": %d, "app_store_id": "%s", "team_id": %d, "platform": "%s", "self_service": false}`, team.Name,
			addedApp.Name, getSoftwareTitleIDFromApp(addedApp), addedApp.AdamID, team.ID, addedApp.Platform), 0)

	// Now we should be filtering out the app we added to team 1
	appResp = getAppStoreAppsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/app_store_apps", &getAppStoreAppsRequest{}, http.StatusOK, &appResp, "team_id",
		fmt.Sprint(team.ID))
	require.NoError(t, appResp.Err)
	assert.ElementsMatch(t, append(expectedApps[2:], expectedApps[0]), appResp.AppStoreApps)

	// list the software titles for that team, to get the title id of the VPP app
	listSw = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID),
		"available_for_install", "true")
	require.Len(t, listSw.SoftwareTitles, 1)
	macOSTitleID = listSw.SoftwareTitles[0].ID
	assert.Equal(t, "ipados_apps", listSw.SoftwareTitles[0].Source)

	// filtering by self-service returns nothing
	listSw = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID),
		"self_service", "true")
	require.Len(t, listSw.SoftwareTitles, 0)

	// delete the app store app for team 1
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", macOSTitleID), nil, http.StatusNoContent,
		"team_id",
		fmt.Sprint(team.ID))
	s.lastActivityMatches(fleet.ActivityDeletedAppStoreApp{}.ActivityName(),
		fmt.Sprintf(`{"team_name": "%s", "software_title": "%s", "app_store_id": "%s", "team_id": %d, "platform": "%s"}`, team.Name,
			addedApp.Name, addedApp.AdamID, team.ID, addedApp.Platform), 0)

	// deleting it again fails, not found
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", macOSTitleID), nil, http.StatusNotFound,
		"team_id",
		fmt.Sprint(team.ID))

	// get the list of available apps, returns all apps now
	appResp = getAppStoreAppsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/app_store_apps", nil, http.StatusOK, &appResp, "team_id", fmt.Sprint(team.ID))
	require.NoError(t, appResp.Err)
	assert.ElementsMatch(t, expectedApps, appResp.AppStoreApps)

	// Installation flow

	// Create a couple of hosts
	orbitHost := createOrbitEnrolledHost(t, "darwin", "nonmdm", s.ds)
	mdmHost, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setOrbitEnrollment(t, mdmHost, s.ds)
	selfServiceHost, selfServiceDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setOrbitEnrollment(t, selfServiceHost, s.ds)
	selfServiceToken := "selfservicetoken"
	updateDeviceTokenForHost(t, s.ds, selfServiceHost.ID, selfServiceToken)
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, selfServiceDevice.SerialNumber)
	iOSHost, iOSMdmClient := s.createAppleMobileHostThenEnrollMDM("ios")
	iPadOSHost, iPadOSMdmClient := s.createAppleMobileHostThenEnrollMDM("ipados")

	// Add serial number to our fake Apple server
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, mdmHost.HardwareSerial,
		iOSHost.HardwareSerial, iPadOSHost.HardwareSerial)
	s.Do("POST", "/api/latest/fleet/hosts/transfer",
		&addHostsToTeamRequest{HostIDs: []uint{mdmHost.ID, orbitHost.ID, iOSHost.ID, iPadOSHost.ID, selfServiceHost.ID}, TeamID: &team.ID}, http.StatusOK)

	// Add all apps to the team
	addedApp = expectedApps[0]
	errApp := expectedApps[3]
	appSelfService := expectedApps[0]
	// Add app 1 as self-service
	addAppResp = addAppStoreAppResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps",
		&addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: appSelfService.AdamID, Platform: appSelfService.Platform, SelfService: true},
		http.StatusOK, &addAppResp)
	s.lastActivityMatches(
		fleet.ActivityAddedAppStoreApp{}.ActivityName(),
		fmt.Sprintf(`{"team_name": "%s", "software_title": "%s", "software_title_id": %d, "app_store_id": "%s", "team_id": %d, "platform": "%s", "self_service": true}`, team.Name,
			appSelfService.Name, getSoftwareTitleIDFromApp(appSelfService), appSelfService.AdamID, team.ID, appSelfService.Platform),
		0,
	)
	listSw = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID),
		"available_for_install", "true")
	// Add remaining as non-self-service
	for _, app := range expectedApps[1:] {
		addAppResp = addAppStoreAppResponse{}
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps",
			&addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: app.AdamID, Platform: app.Platform},
			http.StatusOK, &addAppResp)
		s.lastActivityMatches(
			fleet.ActivityAddedAppStoreApp{}.ActivityName(),
			fmt.Sprintf(`{"team_name": "%s", "software_title": "%s", "software_title_id": %d, "app_store_id": "%s", "team_id": %d, "platform": "%s", "self_service": false}`, team.Name,
				app.Name, getSoftwareTitleIDFromApp(app), app.AdamID, team.ID, app.Platform),
			0,
		)
		listSw = listSoftwareTitlesResponse{}
		s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID),
			"available_for_install", "true")
	}

	listSw = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID), "available_for_install", "true")
	require.Len(t, listSw.SoftwareTitles, len(expectedApps))
	macOSTitleID = 99999
	errTitleID := uint(99999)
	iOSTitleID := uint(99999)
	iPadOSTitleID := uint(99999)
	for _, sw := range listSw.SoftwareTitles {
		assert.NotNil(t, sw.AppStoreApp)
		switch {
		case sw.Name == addedApp.Name && sw.Source == "apps":
			macOSTitleID = sw.ID
		case sw.Name == errApp.Name && sw.Source == "apps":
			errTitleID = sw.ID
		case sw.Name == iOSApp.Name && sw.Source == "ios_apps":
			iOSTitleID = sw.ID
		case sw.Name == iPadOSApp.Name && sw.Source == "ipados_apps":
			iPadOSTitleID = sw.ID
		}
	}

	// attempt to install a VPP app on the non-MDM enrolled host
	r := s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", orbitHost.ID, macOSTitleID), &installSoftwareRequest{}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(r.Body), "Error: Couldn't install. To install App Store app, turn on MDM for this host.")

	// Disable all teams token
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", validToken.Token.ID), patchVPPTokensTeamsRequest{}, http.StatusOK, &resPatchVPP)

	// Spoof an expired VPP token and attempt to install VPP app
	tokenJSONBad := fmt.Sprintf(`{"expDate":"%s","token":"%s","orgName":"%s"}`, "2099-06-24T15:50:50+0000", "badtoken", "Evil Fleet")
	s.appleVPPConfigSrvConfig.Location = "Spooky Haunted House"
	var vppRes uploadVPPTokenResponse
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken", []byte(base64.StdEncoding.EncodeToString([]byte(tokenJSONBad))), http.StatusAccepted, "", &vppRes)

	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", vppRes.Token.ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID, 99}}, http.StatusUnprocessableEntity, &resPatchVPP)

	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", vppRes.Token.ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID}}, http.StatusOK, &resPatchVPP)

	// Disable the token
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", vppRes.Token.ID), patchVPPTokensTeamsRequest{}, http.StatusOK, &resPatchVPP)

	// Enable all teams token
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", validToken.Token.ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{}}, http.StatusOK, &resPatchVPP)

	// Attempt to install non-existent app
	r = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost.ID, 99999), &installSoftwareRequest{},
		http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(r.Body), "Couldn't install software. Software title is not available for install. Please add software package or App Store app to install.")

	// Add app 1 as self-service
	addAppResp = addAppStoreAppResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps",
		&addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: errApp.AdamID, Platform: errApp.Platform, SelfService: true},
		http.StatusOK, &addAppResp)

	// Add remaining apps without self-service
	for _, app := range expectedApps {
		addAppResp = addAppStoreAppResponse{}
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps",
			&addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: app.AdamID, Platform: app.Platform, SelfService: app.AdamID == macOSApp.AdamID},
			http.StatusOK, &addAppResp)
	}

	// Trigger install to the host
	installResp := installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost.ID, errTitleID), &installSoftwareRequest{},
		http.StatusAccepted, &installResp)

	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d", vppRes.Token.ID), &deleteVPPTokenRequest{}, http.StatusNoContent)

	// Check if the host is listed as pending
	var listResp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", "pending", "team_id", fmt.Sprint(team.ID),
		"software_title_id", fmt.Sprint(errTitleID))
	require.Len(t, listResp.Hosts, 1)
	require.Equal(t, listResp.Hosts[0].ID, mdmHost.ID)
	var countResp countHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(errTitleID))
	require.Equal(t, 1, countResp.Count)

	// Simulate failed installation on the host
	cmd, err := mdmDevice.Idle()
	var failedCmdUUID string
	require.NoError(t, err)
	assert.NotNil(t, cmd)
	for cmd != nil {
		var fullCmd micromdm.CommandPayload
		switch cmd.Command.RequestType { //nolint:gocritic // ignore singleCaseSwitch
		case "InstallApplication":
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
			failedCmdUUID = cmd.CommandUUID
			t.Logf("Failed command UUID: %s", failedCmdUUID)
			cmd, err = mdmDevice.Err(cmd.CommandUUID, []mdm.ErrorChain{{ErrorCode: 1234}})
			require.NoError(t, err)
		}
	}

	listResp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", "failed", "team_id", fmt.Sprint(team.ID),
		"software_title_id", fmt.Sprint(errTitleID))
	require.Len(t, listResp.Hosts, 1)
	require.Equal(t, listResp.Hosts[0].ID, mdmHost.ID)
	countResp = countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "failed", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(errTitleID))
	require.Equal(t, 1, countResp.Count)

	s.lastActivityMatches(
		fleet.ActivityInstalledAppStoreApp{}.ActivityName(),
		fmt.Sprintf(
			`{"host_id": %d, "host_display_name": "%s", "software_title": "%s", "app_store_id": "%s", "command_uuid": "%s", "status": "%s", "self_service": false, "policy_id": null, "policy_name": null}`,
			mdmHost.ID,
			mdmHost.DisplayName(),
			errApp.Name,
			errApp.AdamID,
			failedCmdUUID,
			fleet.SoftwareInstallFailed,
		),
		0,
	)

	// Trigger install to the host
	installResp = installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost.ID, macOSTitleID), &installSoftwareRequest{},
		http.StatusAccepted, &installResp)
	countResp = countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 1, countResp.Count)

	// Simulate successful installation on the host
	var cmdUUID string
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		var fullCmd micromdm.CommandPayload
		switch cmd.Command.RequestType { //nolint:gocritic // ignore singleCaseSwitch
		case "InstallApplication":
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
			cmdUUID = cmd.CommandUUID
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}
	}

	listResp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", "installed", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Len(t, listResp.Hosts, 1)
	countResp = countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "installed", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 1, countResp.Count)

	s.lastActivityMatches(
		fleet.ActivityInstalledAppStoreApp{}.ActivityName(),
		fmt.Sprintf(
			`{"host_id": %d, "host_display_name": "%s", "software_title": "%s", "app_store_id": "%s", "command_uuid": "%s", "status": "%s", "self_service": false, "policy_id": null, "policy_name": null}`,
			mdmHost.ID,
			mdmHost.DisplayName(),
			addedApp.Name,
			addedApp.AdamID,
			cmdUUID,
			fleet.SoftwareInstalled,
		),
		0,
	)

	// Check list host software

	getHostSw := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", mdmHost.ID), nil, http.StatusOK, &getHostSw)
	gotSW := getHostSw.Software
	require.Len(t, gotSW, 2) // App 1 and App 2
	got1, got2 := gotSW[0], gotSW[1]
	require.Equal(t, got1.Name, "App 1")
	require.NotNil(t, got1.AppStoreApp)
	require.Equal(t, got1.AppStoreApp.AppStoreID, addedApp.AdamID)
	require.Equal(t, got1.AppStoreApp.IconURL, ptr.String(addedApp.IconURL))
	require.Empty(t, got1.AppStoreApp.Name) // Name is only present for installer packages
	require.Equal(t, got1.AppStoreApp.Version, addedApp.LatestVersion)
	require.NotNil(t, got1.Status)
	require.Equal(t, *got1.Status, fleet.SoftwareInstalled)
	require.Equal(t, got1.AppStoreApp.LastInstall.CommandUUID, cmdUUID)
	require.NotNil(t, got1.AppStoreApp.LastInstall.InstalledAt)
	require.Equal(t, got2.Name, "App 2")
	require.NotNil(t, got2.Status)
	require.Equal(t, *got2.Status, fleet.SoftwareInstallFailed)
	require.NotNil(t, got2.AppStoreApp)
	require.Equal(t, got2.AppStoreApp.AppStoreID, errApp.AdamID)
	require.Equal(t, got2.AppStoreApp.IconURL, ptr.String(errApp.IconURL))
	require.Empty(t, got2.AppStoreApp.Name)
	require.Equal(t, got2.AppStoreApp.Version, errApp.LatestVersion)
	require.Equal(t, got2.AppStoreApp.LastInstall.CommandUUID, failedCmdUUID)
	require.NotNil(t, got2.AppStoreApp.LastInstall.InstalledAt)

	// Check with a query
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", mdmHost.ID), nil, http.StatusOK, &getHostSw, "query", "App 1")
	require.Len(t, getHostSw.Software, 1) // App 1 only
	got1 = getHostSw.Software[0]
	require.Equal(t, got1.Name, "App 1")
	require.NotNil(t, got1.AppStoreApp)
	require.NotNil(t, got1.AppStoreApp)
	require.Equal(t, got1.AppStoreApp.AppStoreID, addedApp.AdamID)
	require.Equal(t, got1.AppStoreApp.IconURL, ptr.String(addedApp.IconURL))
	require.Empty(t, got1.AppStoreApp.Name)
	require.Equal(t, got1.AppStoreApp.Version, addedApp.LatestVersion)
	require.NotNil(t, got1.Status)
	require.Equal(t, *got1.Status, fleet.SoftwareInstalled)
	require.Equal(t, got1.AppStoreApp.LastInstall.CommandUUID, cmdUUID)
	require.NotNil(t, got1.AppStoreApp.LastInstall.InstalledAt)

	// Filter the self-service apps for that host
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", mdmHost.ID), nil, http.StatusOK, &getHostSw, "self_service", "true")
	require.Len(t, getHostSw.Software, 1)
	require.Equal(t, appSelfService.Name, got1.Name)

	// Return installed app with software detail query
	distributedReq := submitDistributedQueryResultsRequestShim{
		NodeKey: *mdmHost.NodeKey,
		Results: map[string]json.RawMessage{
			hostDetailQueryPrefix + "software_macos": json.RawMessage(fmt.Sprintf(
				`[{"name": "%s", "version": "%s", "type": "Application (macOS)",
					"bundle_identifier": "%s", "source": "apps", "last_opened_at": "",
					"installed_path": "/Applications/a.app"}]`, addedApp.Name, addedApp.LatestVersion, addedApp.BundleIdentifier)),
		},
		Statuses: map[string]interface{}{
			hostDistributedQueryPrefix + "software_macos": 0,
		},
		Messages: map[string]string{},
		Stats:    map[string]*fleet.Stats{},
	}
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)

	// Remove the installed app by not returning it
	distributedReq = submitDistributedQueryResultsRequestShim{
		NodeKey: *mdmHost.NodeKey,
		Results: map[string]json.RawMessage{
			hostDetailQueryPrefix + "software_macos": json.RawMessage(`[]`),
		},
		Statuses: map[string]interface{}{
			hostDistributedQueryPrefix + "software_macos": 0,
		},
		Messages: map[string]string{},
		Stats:    map[string]*fleet.Stats{},
	}
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)

	// Check list host software
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", mdmHost.ID), nil, http.StatusOK, &getHostSw)
	gotSW = getHostSw.Software
	require.Len(t, gotSW, 2) // App 1 and App 2
	got1 = gotSW[0]
	require.Equal(t, got1.Name, "App 1")
	require.NotNil(t, got1.AppStoreApp)
	require.Equal(t, got1.AppStoreApp.AppStoreID, addedApp.AdamID)
	require.Equal(t, got1.AppStoreApp.IconURL, ptr.String(addedApp.IconURL))
	require.Empty(t, got1.AppStoreApp.Name) // Name is only present for installer packages
	require.Equal(t, got1.AppStoreApp.Version, addedApp.LatestVersion)
	assert.Nil(t, got1.Status)
	assert.Nil(t, got1.AppStoreApp.LastInstall)

	// Install on iOS and iPadOS devices
	installs := map[string]struct {
		installHost    *fleet.Host
		titleID        uint
		mdmClient      *mdmtest.TestAppleMDMClient
		app            fleet.VPPApp
		extraAvailable int
		hostCount      int
		deviceToken    string
	}{
		"iOS app install": {installHost: iOSHost, titleID: iOSTitleID, mdmClient: iOSMdmClient, app: iOSApp, hostCount: 1},
		"iPadOS app install": {
			installHost: iPadOSHost, titleID: iPadOSTitleID, mdmClient: iPadOSMdmClient, app: iPadOSApp,
			extraAvailable: 1, hostCount: 1,
		},
		"macOS app install": {
			installHost: selfServiceHost, titleID: macOSTitleID, mdmClient: selfServiceDevice, app: macOSApp,
			hostCount: 2, deviceToken: selfServiceToken,
		},
	}

	for name, install := range installs {
		t.Run(name, func(t *testing.T) {
			installHost := install.installHost
			titleID := install.titleID
			mdmClient := install.mdmClient
			app := install.app

			// Self-service install
			if install.deviceToken != "" {
				var ssInstallResp submitSelfServiceSoftwareInstallResponse
				s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/device/%s/software/install/%d", install.deviceToken, install.titleID),
					&fleetSelfServiceSoftwareInstallRequest{}, http.StatusAccepted, &ssInstallResp)
			} else {
				installResp = installSoftwareResponse{}
				s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", installHost.ID, titleID),
					&installSoftwareRequest{}, http.StatusAccepted, &installResp)
			}
			countResp = countHostsResponse{}
			s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
				fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(titleID))
			require.Equal(t, 1, countResp.Count)

			// send an idle request to grab the command uuid
			cmd, err = mdmClient.Idle()
			require.NoError(t, err)
			var fullCmd micromdm.CommandPayload
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
			cmdUUID = cmd.CommandUUID

			// Get pending activity
			var hostActivitiesResp listHostUpcomingActivitiesResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", installHost.ID),
				nil, http.StatusOK, &hostActivitiesResp)
			activitiesToString := func(activities []*fleet.Activity) []string {
				var res []string
				for _, activity := range activities {
					res = append(res, fmt.Sprintf("%+v", activity))
				}
				return res
			}
			require.Len(t, hostActivitiesResp.Activities, 1, "got activities: %v", activitiesToString(hostActivitiesResp.Activities))
			assert.Equal(t, hostActivitiesResp.Activities[0].Type, fleet.ActivityInstalledAppStoreApp{}.ActivityName())
			assert.EqualValues(t, 1, hostActivitiesResp.Count)
			assert.JSONEq(
				t,
				fmt.Sprintf(
					`{"host_id": %d, "host_display_name": "%s", "software_title": "%s", "app_store_id": "%s", "command_uuid": "%s", "status": "%s", "self_service": %v}`,
					installHost.ID,
					installHost.DisplayName(),
					app.Name,
					app.AdamID,
					cmdUUID,
					fleet.SoftwareInstallPending,
					install.deviceToken != "",
				),
				string(*hostActivitiesResp.Activities[0].Details),
			)

			// Simulate successful installation on the host
			cmd, err = mdmClient.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
			// No further commands expected
			assert.Nil(t, cmd)

			listResp = listHostsResponse{}
			s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", "installed", "team_id",
				fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(titleID))
			assert.Len(t, listResp.Hosts, install.hostCount)
			countResp = countHostsResponse{}
			s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "installed", "team_id",
				fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(titleID))
			assert.Equal(t, install.hostCount, countResp.Count)

			s.lastActivityMatches(
				fleet.ActivityInstalledAppStoreApp{}.ActivityName(),
				fmt.Sprintf(
					`{"host_id": %d, "host_display_name": "%s", "software_title": "%s", "app_store_id": "%s", "command_uuid": "%s", "status": "%s", "self_service": %v, "policy_id": null, "policy_name": null}`,
					installHost.ID,
					installHost.DisplayName(),
					app.Name,
					app.AdamID,
					cmdUUID,
					fleet.SoftwareInstalled,
					install.deviceToken != "",
				),
				0,
			)

			// Check list host software
			getHostSw = getHostSoftwareResponse{}
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", installHost.ID), nil, http.StatusOK, &getHostSw)
			require.Len(t, getHostSw.Software, install.hostCount+install.extraAvailable)
			var foundInstalledApp bool
			for index := range getHostSw.Software {
				got1 = getHostSw.Software[index]
				if got1.Status != nil {
					require.Equal(t, got1.Name, app.Name)
					require.NotNil(t, got1.AppStoreApp)
					require.Equal(t, got1.AppStoreApp.AppStoreID, app.AdamID)
					require.Equal(t, got1.AppStoreApp.IconURL, ptr.String(app.IconURL))
					require.Empty(t, got1.AppStoreApp.Name) // Name is only present for installer packages
					require.Equal(t, got1.AppStoreApp.Version, app.LatestVersion)
					require.Equal(t, *got1.Status, fleet.SoftwareInstalled)
					require.Equal(t, got1.AppStoreApp.LastInstall.CommandUUID, cmdUUID)
					require.NotNil(t, got1.AppStoreApp.LastInstall.InstalledAt)
					foundInstalledApp = true
				}
			}
			assert.True(t, foundInstalledApp)
		})
	}

	// Attempt (and fail) to self-service install iPad and iOS titles
	var ssInstallResp submitSelfServiceSoftwareInstallResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/device/%s/software/install/%d", selfServiceToken, iPadOSTitleID), &fleetSelfServiceSoftwareInstallRequest{},
		http.StatusBadRequest, &ssInstallResp)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/device/%s/software/install/%d", selfServiceToken, iOSTitleID), &fleetSelfServiceSoftwareInstallRequest{},
		http.StatusBadRequest, &ssInstallResp)

	// Delete VPP token and check that it's not appearing anymore

	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d", validToken.Token.ID), &deleteVPPTokenResponse{}, http.StatusNoContent)
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &resp)
}

func (s *integrationMDMTestSuite) TestVPPAppPolicyAutomation() {
	t := s.T()
	ctx := context.Background()

	// Set up VPP token
	orgName := "Fleet Device Management Inc."
	token := "mycooltoken"
	expTime := time.Now().Add(200 * time.Hour).UTC().Round(time.Second)
	expDate := expTime.Format(fleet.VPPTimeFormat)
	tokenJSON := fmt.Sprintf(`{"expDate":"%s","token":"%s","orgName":"%s"}`, expDate, token, orgName)
	t.Setenv("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL)
	var validToken uploadVPPTokenResponse
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken", []byte(base64.StdEncoding.EncodeToString([]byte(tokenJSON))), http.StatusAccepted, "", &validToken)

	// Get the token
	var resp getVPPTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &resp)
	require.NoError(t, resp.Err)

	// Create a team
	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: ptr.String("Team 1")}}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

	// Associate team to the VPP token.
	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", resp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID}}, http.StatusOK, &resPatchVPP)

	// Get list of VPP apps from "Apple"
	// We're passing team 1 here, but we haven't added any app store apps to that team, so we get
	// back all available apps in our VPP location.
	var appResp getAppStoreAppsResponse
	s.DoJSON("GET", "/api/latest/fleet/software/app_store_apps", &getAppStoreAppsRequest{}, http.StatusOK, &appResp, "team_id",
		fmt.Sprint(team.ID))
	require.NoError(t, appResp.Err)

	macOSApp := fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "1",
				Platform: fleet.MacOSPlatform,
			},
		},
		Name:             "App 1",
		BundleIdentifier: "a-1",
		IconURL:          "https://example.com/images/1",
		LatestVersion:    "1.0.0",
	}
	iPadOSApp := fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "2",
				Platform: fleet.IPadOSPlatform,
			},
		},
		Name:             "App 2",
		BundleIdentifier: "b-2",
		IconURL:          "https://example.com/images/2",
		LatestVersion:    "2.0.0",
	}
	iOSApp := fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "2",
				Platform: fleet.IOSPlatform,
			},
		},
		Name:             "App 2",
		BundleIdentifier: "b-2",
		IconURL:          "https://example.com/images/2",
		LatestVersion:    "2.0.0",
	}
	expectedApps := []*fleet.VPPApp{
		&macOSApp,
		&iPadOSApp,
		&iOSApp,
		{
			VPPAppTeam: fleet.VPPAppTeam{
				VPPAppID: fleet.VPPAppID{
					AdamID:   "2",
					Platform: fleet.MacOSPlatform,
				},
			},
			Name:             "App 2",
			BundleIdentifier: "b-2",
			IconURL:          "https://example.com/images/2",
			LatestVersion:    "2.0.0",
		},
		{
			VPPAppTeam: fleet.VPPAppTeam{
				VPPAppID: fleet.VPPAppID{
					AdamID:   "3",
					Platform: fleet.IPadOSPlatform,
				},
			},
			Name:             "App 3",
			BundleIdentifier: "c-3",
			IconURL:          "https://example.com/images/3",
			LatestVersion:    "3.0.0",
		},
	}
	assert.ElementsMatch(t, expectedApps, appResp.AppStoreApps)

	getSoftwareTitleIDFromApp := func(app *fleet.VPPApp) uint {
		var titleID uint
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &titleID, `SELECT title_id FROM vpp_apps WHERE adam_id = ? AND platform = ?;`, app.AdamID, app.Platform)
		})

		return titleID
	}

	// Insert/deletion flow for macOS app
	// Add an app store app to team 1
	addedApp := expectedApps[0]
	var addedMacOSApp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: addedApp.AdamID, SelfService: true}, http.StatusOK, &addedMacOSApp)

	// list the software titles for that team, to get the title id of the VPP app
	var listSw listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID), "available_for_install", "true")
	require.Len(t, listSw.SoftwareTitles, 1)
	require.True(t, *listSw.SoftwareTitles[0].AppStoreApp.SelfService)
	macOSTitleID := listSw.SoftwareTitles[0].ID

	// Insert iPadOS app
	addedApp = expectedApps[1]
	addedIOSApp := addAppStoreAppResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps",
		&addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: addedApp.AdamID, Platform: addedApp.Platform},
		http.StatusOK, &addedIOSApp)

	// Create a couple of hosts
	orbitHost := createOrbitEnrolledHost(t, "darwin", "nonmdm", s.ds)
	mdmHost, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setOrbitEnrollment(t, mdmHost, s.ds)
	selfServiceHost, selfServiceDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setOrbitEnrollment(t, selfServiceHost, s.ds)
	selfServiceToken := "selfservicetoken"
	updateDeviceTokenForHost(t, s.ds, selfServiceHost.ID, selfServiceToken)
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, selfServiceDevice.SerialNumber)

	// Add serial number to our fake Apple server
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, mdmHost.HardwareSerial)
	s.Do("POST", "/api/latest/fleet/hosts/transfer",
		&addHostsToTeamRequest{HostIDs: []uint{mdmHost.ID, orbitHost.ID, selfServiceHost.ID}, TeamID: &team.ID}, http.StatusOK)

	// Add all apps to the team
	appSelfService := expectedApps[0]
	// Add app 1 as self-service
	addedMacOSApp = addAppStoreAppResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps",
		&addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: appSelfService.AdamID, Platform: appSelfService.Platform, SelfService: true},
		http.StatusOK, &addedMacOSApp)
	// Add remaining as non-self-service
	for _, app := range expectedApps[1:] {
		addedMacOSApp = addAppStoreAppResponse{}
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps",
			&addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: app.AdamID, Platform: app.Platform},
			http.StatusOK, &addedMacOSApp)
		s.lastActivityMatches(
			fleet.ActivityAddedAppStoreApp{}.ActivityName(),
			fmt.Sprintf(`{"team_name": "%s", "software_title": "%s", "software_title_id": %d, "app_store_id": "%s", "team_id": %d, "platform": "%s", "self_service": false}`, team.Name,
				app.Name, getSoftwareTitleIDFromApp(app), app.AdamID, team.ID, app.Platform),
			0,
		)
		listSw = listSoftwareTitlesResponse{}
		s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID),
			"available_for_install", "true")
	}

	listSw = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID), "available_for_install", "true")
	require.Len(t, listSw.SoftwareTitles, len(expectedApps))
	iOSTitleID := uint(99999)

	policy0Team1, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:     "policy0Team1",
		Query:    "SELECT 1;",
		Platform: "darwin",
	})
	require.NoError(t, err)
	policy1Team1, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:     "policy1Team1",
		Query:    "SELECT 1;",
		Platform: "darwin",
	})
	require.NoError(t, err)
	policy2Team1, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{ // will be set up with same VPP app
		Name:     "policy2Team1",
		Query:    "SELECT 1;",
		Platform: "darwin",
	})
	require.NoError(t, err)

	mtplr := modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team.ID, policy1Team1.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: iOSTitleID},
		},
	}, http.StatusBadRequest, &mtplr)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team.ID, policy1Team1.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: macOSTitleID},
		},
	}, http.StatusOK, &mtplr)

	titleResponse := getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", macOSTitleID), getSoftwareTitleRequest{
		TeamID: &team.ID,
	}, http.StatusOK, &titleResponse)
	require.Len(t, titleResponse.SoftwareTitle.AppStoreApp.AutomaticInstallPolicies, 1)
	require.Equal(t, titleResponse.SoftwareTitle.AppStoreApp.AutomaticInstallPolicies[0].ID, policy1Team1.ID)

	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team.ID, policy2Team1.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: macOSTitleID},
		},
	}, http.StatusOK, &mtplr)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", macOSTitleID), getSoftwareTitleRequest{
		TeamID: &team.ID,
	}, http.StatusOK, &titleResponse)
	require.Len(t, titleResponse.SoftwareTitle.AppStoreApp.AutomaticInstallPolicies, 2)

	// add a non-macOS host
	newHost := func(name string, teamID *uint, platform string) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-1 * time.Minute),
			OsqueryHostID:   ptr.String(t.Name() + name),
			NodeKey:         ptr.String(t.Name() + name),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%s.%s.local", name, t.Name()),
			Platform:        platform,
			TeamID:          teamID,
		})
		require.NoError(t, err)
		return h
	}
	newFleetdHost := func(name string, teamID *uint, platform string) *fleet.Host {
		h := newHost(name, teamID, platform)
		orbitKey := setOrbitEnrollment(t, h, s.ds)
		h.OrbitNodeKey = &orbitKey
		return h
	}

	host3 := newFleetdHost("host3", &team.ID, "ubuntu")

	// setting/unsetting is covered by the other software automation integration test

	// We use DoJSONWithoutAuth for distributed/write because we want the requests to not have the
	// current user's "Authorization: Bearer <API_TOKEN>" header.

	// host3 failure should not queue an install due to mismatched platform
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host3,
		map[uint]*bool{
			policy1Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)
	var countResp countHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 0, countResp.Count)

	// non-MDM host failure should not queue an install due to lack of MDM
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		orbitHost,
		map[uint]*bool{
			policy1Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 0, countResp.Count)

	// MDM host passing policy should not queue install, nor should failing non-VPP policy
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		mdmHost,
		map[uint]*bool{
			policy0Team1.ID: ptr.Bool(false),
			policy1Team1.ID: ptr.Bool(true),
		},
	), http.StatusOK, &distributedResp)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 0, countResp.Count)

	// MDM host failing policy should queue install
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		mdmHost,
		map[uint]*bool{
			policy1Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 1, countResp.Count)

	// MDM host failing policy should not queue another install while install is pending
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		mdmHost,
		map[uint]*bool{
			policy1Team1.ID: ptr.Bool(false),
			policy2Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)
	var countPendingInstalls uint
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &countPendingInstalls, `SELECT COUNT(*)
			FROM host_vpp_software_installs hvsi
			JOIN nano_view_queue nvq ON nvq.command_uuid = hvsi.command_uuid AND nvq.status IS NULL
			WHERE hvsi.host_id = ?`, mdmHost.ID)
	})
	require.Equal(t, uint(1), countPendingInstalls)

	// send an idle request to grab the command uuid
	var cmdUUID string
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	var fullCmd micromdm.CommandPayload
	require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
	cmdUUID = cmd.CommandUUID

	// Get pending activity, confirm one pending activity
	var hostActivitiesResp listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", mdmHost.ID),
		nil, http.StatusOK, &hostActivitiesResp)
	activitiesToString := func(activities []*fleet.Activity) []string {
		var res []string
		for _, activity := range activities {
			res = append(res, fmt.Sprintf("%+v", activity))
		}
		return res
	}
	require.Len(t, hostActivitiesResp.Activities, 1, "got activities: %v", activitiesToString(hostActivitiesResp.Activities))
	assert.Equal(t, hostActivitiesResp.Activities[0].Type, fleet.ActivityInstalledAppStoreApp{}.ActivityName())
	assert.EqualValues(t, 1, hostActivitiesResp.Count)
	assert.JSONEq(
		t,
		fmt.Sprintf(
			`{"host_id": %d, "host_display_name": "%s", "software_title": "%s", "app_store_id": "%s", "command_uuid": "%s", "status": "%s", "self_service": %v}`,
			mdmHost.ID,
			mdmHost.DisplayName(),
			macOSApp.Name,
			macOSApp.AdamID,
			cmdUUID,
			fleet.SoftwareInstallPending,
			false,
		),
		string(*hostActivitiesResp.Activities[0].Details),
	)

	_, err = mdmDevice.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)

	s.lastActivityMatches(
		fleet.ActivityInstalledAppStoreApp{}.ActivityName(),
		fmt.Sprintf(
			`{"host_id": %d, "host_display_name": "%s", "software_title": "%s", "app_store_id": "%s", "command_uuid": "%s", "status": "%s", "self_service": %v, "policy_id": %d, "policy_name": "%s"}`,
			mdmHost.ID,
			mdmHost.DisplayName(),
			macOSApp.Name,
			macOSApp.AdamID,
			cmdUUID,
			fleet.SoftwareInstalled,
			false,
			policy1Team1.ID,
			policy1Team1.Name,
		),
		0,
	)

	// MDM host failing already-failing policies should not trigger any installs
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		mdmHost,
		map[uint]*bool{
			policy1Team1.ID: ptr.Bool(false),
			policy2Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 0, countResp.Count)
}

func (s *integrationMDMTestSuite) TestEnrollmentProfilesWithSpecialChars() {
	t := s.T()
	ctx := context.Background()

	initialConfig, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	initialSecrets, err := s.ds.GetEnrollSecrets(ctx, nil)
	require.NoError(t, err)

	nameWithInvalidChars := "Fleet & Device <3 Management"
	/* #nosec G101 -- this is a made up value for tests */
	enrollSecretWithInvalidChars := "1<2>3&4&/"

	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"org_info": {
			"org_name": %q
		}
	}`, nameWithInvalidChars)), http.StatusOK, &acResp)
	enrollSecretResp := applyEnrollSecretSpecResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: enrollSecretWithInvalidChars}},
		},
	}, http.StatusOK, &enrollSecretResp)
	t.Cleanup(func() {
		acResp := appConfigResponse{}
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		  "org_info": {
			  "org_name": %q
		  }
		}`, initialConfig.OrgInfo.OrgName)), http.StatusOK, &acResp)
		s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
			Spec: &fleet.EnrollSecretSpec{
				Secrets: initialSecrets,
			},
		}, http.StatusOK, &enrollSecretResp)
	})

	// manual enrollment from My Device
	token := "token_test_manual_enroll"
	createHostAndDeviceToken(t, s.ds, token)
	s.downloadAndVerifyOTAEnrollmentProfile("/api/latest/fleet/device/" + token + "/mdm/apple/manual_enrollment_profile")

	// automatic enrollment by token
	rawMsg := json.RawMessage(`{"allow_pairing": true}`)
	_, err = s.ds.NewMDMAppleEnrollmentProfile(ctx, fleet.MDMAppleEnrollmentProfilePayload{
		Type:       "automatic",
		DEPProfile: &rawMsg,
		Token:      "abcd",
	})
	require.NoError(t, err)
	s.downloadAndVerifyEnrollmentProfile(apple_mdm.EnrollPath + "?token=abcd")

	// unsigned manual enrollment profile for IT admins
	s.downloadAndVerifyEnrollmentProfile("/api/latest/fleet/enrollment_profiles/manual")

	// ensure the fleetd profile sends a good enroll secret too
	s.awaitTriggerProfileSchedule(t)
	prof := s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetdConfigPayloadIdentifier, true)

	type fleetdPlist struct {
		PayloadContent []struct {
			EnrollSecret string `plist:"EnrollSecret"`
		} `plist:"PayloadContent"`
	}

	// Parse the plist data
	var parsedData fleetdPlist
	err = plist.NewDecoder(bytes.NewReader(prof.Mobileconfig)).Decode(&parsedData)
	require.NoError(t, err)
	require.Equal(t, enrollSecretWithInvalidChars, parsedData.PayloadContent[0].EnrollSecret)
}

func (s *integrationMDMTestSuite) TestOTAEnrollment() {
	t := s.T()

	// create a global enroll secret
	globalSecret := "global_secret"
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: globalSecret}},
		},
	}, http.StatusOK, &applyResp)

	reqBody := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PRODUCT</key>
	<string></string>
	<key>SERIAL</key>
	<string>foo</string>
	<key>UDID</key>
	<string></string>
	<key>VERSION</key>
	<string></string>
</dict>
</plist>`)

	// request with no enroll secret
	httpResp := s.DoRawNoAuth("POST", "/api/latest/fleet/ota_enrollment", reqBody, http.StatusForbidden)
	errMsg := extractServerErrorText(httpResp.Body)
	require.Contains(t, errMsg, "Couldn't install the profile. Invalid enroll secret. Please contact your IT admin.")
	require.NoError(t, httpResp.Body.Close())

	// request with no body
	httpResp = s.DoRawNoAuth("POST", "/api/latest/fleet/ota_enrollment?enroll_secret=foo", nil, http.StatusBadRequest)
	errMsg = extractServerErrorText(httpResp.Body)
	require.Contains(t, errMsg, "invalid request body")
	require.NoError(t, httpResp.Body.Close())

	// request with unsigned body
	httpResp = s.DoRawNoAuth("POST", "/api/latest/fleet/ota_enrollment?enroll_secret=foo", reqBody, http.StatusBadRequest)
	errMsg = extractServerErrorText(httpResp.Body)
	require.Contains(t, errMsg, "invalid request body")
	require.NoError(t, httpResp.Body.Close())

	cert, key, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(t, err)
	signedData, err := pkcs7.NewSignedData(reqBody)
	require.NoError(t, err)
	require.NoError(t, signedData.AddSigner(cert, key, pkcs7.SignerInfoConfig{}))
	signedReqBody, err := signedData.Finish()
	require.NoError(t, err)

	// request with invalid apple signature
	httpResp = s.DoRawNoAuth("POST", "/api/latest/fleet/ota_enrollment?enroll_secret=foo", signedReqBody, http.StatusForbidden)
	errMsg = extractServerErrorText(httpResp.Body)
	require.Contains(t, errMsg, "Couldn't install the profile. Invalid enroll secret. Please contact your IT admin.")
	require.NoError(t, httpResp.Body.Close())

	// request with invalid device signature
	os.Setenv("FLEET_DEV_MDM_APPLE_DISABLE_DEVICE_INFO_CERT_VERIFY", "1")
	httpResp = s.DoRawNoAuth("POST", "/api/latest/fleet/ota_enrollment?enroll_secret=foo", signedReqBody, http.StatusForbidden)
	errMsg = extractServerErrorText(httpResp.Body)
	require.Contains(t, errMsg, "Couldn't install the profile. Invalid enroll secret. Please contact your IT admin.")
	require.NoError(t, httpResp.Body.Close())

	// request without serial number
	signedData, err = pkcs7.NewSignedData([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>SERIAL</key>
	<string></string>
</dict>
</plist>`))
	require.NoError(t, err)
	require.NoError(t, signedData.AddSigner(cert, key, pkcs7.SignerInfoConfig{}))
	signedReqBody, err = signedData.Finish()
	require.NoError(t, err)
	httpResp = s.DoRawNoAuth("POST", "/api/latest/fleet/ota_enrollment?enroll_secret=foo", signedReqBody, http.StatusBadRequest)
	errMsg = extractServerErrorText(httpResp.Body)
	require.Contains(t, errMsg, "SERIAL is required")
	require.NoError(t, httpResp.Body.Close())

	checkInstallFleetdCommandSent := func(mdmDevice *mdmtest.TestAppleMDMClient, wantCommand bool) {
		foundInstallFleetdCommand := false
		cmd, err := mdmDevice.Idle()
		require.NoError(t, err)
		for cmd != nil {
			var fullCmd micromdm.CommandPayload
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
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

	hwModel := "MacBookPro16,1"
	mdmDevice := mdmtest.NewTestMDMClientAppleOTA(
		s.server.URL,
		globalSecret,
		hwModel,
	)
	require.NoError(t, mdmDevice.Enroll())
	s.runWorker()
	checkInstallFleetdCommandSent(mdmDevice, true)

	var hostByIdentifierResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/identifier/%s", mdmDevice.UUID), nil, http.StatusOK, &hostByIdentifierResp)
	require.Equal(t, hwModel, hostByIdentifierResp.Host.HardwareModel)
	require.Equal(t, "darwin", hostByIdentifierResp.Host.Platform)
	require.Nil(t, hostByIdentifierResp.Host.TeamID)

	// create a team with a different enroll secret
	var specResp applyTeamSpecsResponse
	teamSecret := "team_secret"
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: "newteam", Secrets: &[]fleet.EnrollSecret{{Secret: teamSecret}}}}}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, &specResp)

	hwModel = "iPad13,16"
	mdmDevice = mdmtest.NewTestMDMClientAppleOTA(
		s.server.URL,
		teamSecret,
		hwModel,
	)
	require.NoError(t, mdmDevice.Enroll())
	s.runWorker()
	checkInstallFleetdCommandSent(mdmDevice, false)

	hostByIdentifierResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/identifier/%s", mdmDevice.UUID), nil, http.StatusOK, &hostByIdentifierResp)
	require.Equal(t, hwModel, hostByIdentifierResp.Host.HardwareModel)
	require.Equal(t, "ipados", hostByIdentifierResp.Host.Platform)
	require.NotNil(t, hostByIdentifierResp.Host.TeamID)
	require.Equal(t, specResp.TeamIDsByName["newteam"], *hostByIdentifierResp.Host.TeamID)
}

func (s *integrationMDMTestSuite) TestSCEPProxy() {
	t := s.T()
	ctx := context.Background()

	data, err := os.ReadFile("./testdata/PKCSReq.der")
	require.NoError(t, err)
	message := base64.StdEncoding.EncodeToString(data)

	// NDES not configured
	res := s.DoRawNoAuth("GET", apple_mdm.SCEPProxyPath+"1%2C1", nil, http.StatusBadRequest)
	errBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "missing operation")
	// Provide SCEP operation (GetCACaps)
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+"1%2C1", nil, http.StatusBadRequest, nil, "operation", "GetCACaps")
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), eeservice.MessageSCEPProxyNotConfigured)
	// Provide SCEP operation (GetCACerts)
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+"1%2C1", nil, http.StatusBadRequest, nil, "operation", "GetCACert")
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), eeservice.MessageSCEPProxyNotConfigured)
	// Provide SCEP operation (PKIOperation)
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+"1%2C1", nil, http.StatusBadRequest, nil, "operation", "PKIOperation",
		"message", message)
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), eeservice.MessageSCEPProxyNotConfigured)
	// Provide SCEP operation (GetNextCACert)
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+"1%2C1", nil, http.StatusBadRequest, nil, "operation", "GetNextCACert")
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "not implemented")

	// Add an MDM profile
	globalProfiles := [][]byte{
		mobileconfigForTest("N1", "Good1"),
		mobileconfigForTest("N2", "Bad1"),
	}
	// add global profile
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: globalProfiles}, http.StatusNoContent)

	// Create a host and then enroll to MDM.
	host, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setupPusher(s, t, mdmDevice)
	// trigger a profile sync
	s.awaitTriggerProfileSchedule(t)
	profiles, err := s.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(profiles), 2)
	var profileUUID string
	var badProfile fleet.HostMDMAppleProfile
	for _, profile := range profiles {
		switch profile.Identifier {
		case "Good1":
			profileUUID = profile.ProfileUUID
		case "Bad1":
			profile.Status = &fleet.MDMDeliveryFailed
			badProfile = profile
		}
	}
	err = s.ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{{
		ProfileUUID:       badProfile.ProfileUUID,
		ProfileIdentifier: badProfile.Identifier,
		ProfileName:       badProfile.Name,
		HostUUID:          host.UUID,
		CommandUUID:       badProfile.CommandUUID,
		OperationType:     badProfile.OperationType,
		Status:            badProfile.Status,
		Detail:            badProfile.Detail,
		Checksum:          []byte("checksum"),
	}})
	require.NoError(t, err)
	identifier := url.PathEscape(host.UUID + "," + profileUUID)
	badIdentifier := url.PathEscape(host.UUID + "," + badProfile.ProfileUUID)

	// Configure a bad SCEP URL
	appConf, err := s.ds.AppConfig(context.Background())
	require.NoError(t, err)
	appConf.Integrations.NDESSCEPProxy.Valid = true
	appConf.Integrations.NDESSCEPProxy.Value.URL = "https://httpstat.us/410"
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(t, err)

	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier, nil, http.StatusInternalServerError, nil, "operation", "GetCACaps")
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "Could not GetCACaps from SCEP server")
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier, nil, http.StatusInternalServerError, nil, "operation", "GetCACert")
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "Could not GetCACert from SCEP server")
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier, nil, http.StatusInternalServerError, nil, "operation",
		"PKIOperation", "message", message)
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "Could not do PKIOperation on SCEP server")

	// Test timeout error
	ndesTimeoutServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(1 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	origNDESTimeout := eeservice.NDESTimeout
	eeservice.NDESTimeout = ptr.Duration(1 * time.Microsecond)
	t.Cleanup(func() {
		eeservice.NDESTimeout = origNDESTimeout
		ndesTimeoutServer.Close()
	})
	appConf.Integrations.NDESSCEPProxy.Value.URL = ndesTimeoutServer.URL
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(t, err)
	// GetCACaps
	_ = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier, nil, http.StatusRequestTimeout, nil, "operation", "GetCACaps")
	// GetCACert
	_ = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier, nil, http.StatusRequestTimeout, nil, "operation", "GetCACert")
	// PKIOperation
	_ = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier, nil, http.StatusRequestTimeout, nil, "operation",
		"PKIOperation", "message", message)
	eeservice.NDESTimeout = origNDESTimeout

	// Spin up an "external" SCEP server, which Fleet server will proxy
	newSCEPServer := func(t *testing.T, opts ...scepserver.ServiceOption) *httptest.Server {
		var server *httptest.Server
		teardown := func() {
			if server != nil {
				server.Close()
			}
			os.Remove("./testdata/externalCA/serial")
			os.Remove("./testdata/externalCA/index.txt")
		}
		t.Cleanup(teardown)

		var err error
		var certDepot depot.Depot // cert storage
		certDepot, err = filedepot.NewFileDepot("./testdata/externalCA")
		if err != nil {
			t.Fatal(err)
		}
		certDepot = &noopCertDepot{certDepot}
		crt, key, err := certDepot.CA([]byte{})
		if err != nil {
			t.Fatal(err)
		}

		var svc scepserver.Service // scep service
		svc, err = scepserver.NewService(crt[0], key, scepserver.NopCSRSigner())
		if err != nil {
			t.Fatal(err)
		}
		logger := kitlog.NewNopLogger()
		e := scepserver.MakeServerEndpoints(svc)
		scepHandler := scepserver.MakeHTTPHandler(e, svc, logger)
		r := mux.NewRouter()
		r.Handle("/scep", scepHandler)
		server = httptest.NewServer(r)
		return server
	}
	scepServer := newSCEPServer(t)

	appConf.Integrations.NDESSCEPProxy.Value.URL = scepServer.URL + "/scep"
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(t, err)

	// GetCACaps
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier, nil, http.StatusOK, nil, "operation", "GetCACaps")
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, scepserver.DefaultCACaps, string(body))

	// GetCACert
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier, nil, http.StatusOK, nil, "operation", "GetCACert")
	body, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	certs, err := x509.ParseCertificates(body)
	require.NoError(t, err)
	assert.Len(t, certs, 1)

	// PKIOperation
	// Invalid identifier format
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+"%2Cbozo", nil, http.StatusBadRequest, nil, "operation",
		"PKIOperation", "message", message)
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "invalid identifier")
	// Non-Apple config profile (missing leading 'a')
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+"bozoHost%2CwbozoProfile", nil, http.StatusBadRequest, nil, "operation",
		"PKIOperation", "message", message)
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "invalid profile UUID")
	// Unknown host/profile
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+"bozoHost%2CabozoProfile", nil, http.StatusBadRequest, nil, "operation",
		"PKIOperation", "message", message)
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "unknown identifier")
	// Profile which is not pending
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+badIdentifier, nil, http.StatusBadRequest, nil, "operation",
		"PKIOperation", "message", message)
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "profile status")

	// "Good" request without one-time challenge.
	// Note, a cert is not returned here because the message is not a fully valid SCEP request. However, building a valid SCEP request is a bit involved,
	// and this request is sufficient for testing our proxy functionality.
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier, nil, http.StatusOK, nil, "operation",
		"PKIOperation", "message", message)
	body, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	pkiMessage, err := scep.ParsePKIMessage(body, scep.WithCACerts(certs))
	require.NoError(t, err)
	assert.Equal(t, scep.CertRep, pkiMessage.MessageType)

	// Expired challenge
	err = s.ds.BulkUpsertMDMManagedCertificates(ctx, []*fleet.MDMBulkUpsertManagedCertificatePayload{
		{
			HostUUID:             host.UUID,
			ProfileUUID:          profileUUID,
			ChallengeRetrievedAt: ptr.Time(time.Now().Add(-eeservice.NDESChallengeInvalidAfter)),
		},
	})
	require.NoError(t, err)
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier, nil, http.StatusBadRequest, nil, "operation",
		"PKIOperation", "message", message)
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "challenge password")

	// Non-expired challenge
	err = s.ds.BulkUpsertMDMManagedCertificates(ctx, []*fleet.MDMBulkUpsertManagedCertificatePayload{
		{
			HostUUID:             host.UUID,
			ProfileUUID:          profileUUID,
			ChallengeRetrievedAt: ptr.Time(time.Now().Add(-eeservice.NDESChallengeInvalidAfter + time.Minute)),
		},
	})
	require.NoError(t, err)

	// Profile status is not yet "pending" (should be null) until profile sync
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+badIdentifier, nil, http.StatusBadRequest, nil, "operation",
		"PKIOperation", "message", message)
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "profile status")

	// trigger a profile sync
	s.awaitTriggerProfileSchedule(t)
	// Good request with non-expired challenge
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier, nil, http.StatusOK, nil, "operation",
		"PKIOperation", "message", message)
	body, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	pkiMessage, err = scep.ParsePKIMessage(body, scep.WithCACerts(certs))
	require.NoError(t, err)
	assert.Equal(t, scep.CertRep, pkiMessage.MessageType)
}

type noopCertDepot struct{ depot.Depot }

func (d *noopCertDepot) Put(_ string, _ *x509.Certificate) error {
	return nil
}

func (s *integrationMDMTestSuite) TestVPPAppsMDMFiltering() {
	t := s.T()

	ctx := context.Background()

	// Create hosts
	orbitHost := createOrbitEnrolledHost(t, "darwin", "nonmdm", s.ds)
	mdmHost, mdmClient := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	_, _ = mdmHost, mdmClient

	test.CreateInsertGlobalVPPToken(t, s.ds)

	// Create team and add hosts to team
	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: ptr.String("Team 1")}}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

	s.Do("POST", "/api/latest/fleet/hosts/transfer", &addHostsToTeamRequest{HostIDs: []uint{orbitHost.ID, mdmHost.ID}, TeamID: &team.ID}, http.StatusOK)

	// Add an app so that we don't get a not found error
	_, err := s.ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name:             "App " + t.Name(),
		BundleIdentifier: "bid_" + t.Name(),
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "adam_" + t.Name(),
				Platform: fleet.MacOSPlatform,
			},
		},
	}, &team.ID)
	require.NoError(t, err)

	resp := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", orbitHost.ID), getHostSoftwareRequest{}, http.StatusOK, &resp)
	assert.Len(t, resp.Software, 0)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", mdmHost.ID), getHostSoftwareRequest{}, http.StatusOK, &resp)
	assert.Len(t, resp.Software, 1)
}

func (s *integrationMDMTestSuite) TestSetupExperience() {
	t := s.T()
	ds := s.ds
	ctx := context.Background()

	test.CreateInsertGlobalVPPToken(t, ds)
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	tfr1, err := fleet.NewTempFileReader(strings.NewReader("hello"), t.TempDir)
	require.NoError(t, err)
	installerID1, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:     "hello",
		PreInstallQuery:   "SELECT 1",
		PostInstallScript: "world",
		UninstallScript:   "goodbye",
		InstallerFile:     tfr1,
		StorageID:         "storage1",
		Filename:          "file1",
		Title:             "file1",
		Version:           "1.0",
		Source:            "apps",
		UserID:            user1.ID,
		TeamID:            &team1.ID,
		Platform:          string(fleet.MacOSPlatform),
		ValidatedLabels:   &fleet.LabelIdentsWithScope{},
	})
	_ = installerID1
	require.NoError(t, err)

	app1 := &fleet.VPPApp{Name: "vpp_app_1", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: "1", Platform: fleet.MacOSPlatform}}, BundleIdentifier: "b1"}
	_, err = ds.InsertVPPAppWithTeam(ctx, app1, &team1.ID)
	require.NoError(t, err)

	var respListTitles listSoftwareTitlesResponse
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &respListTitles,
		"team_id", fmt.Sprint(team1.ID),
	)
	require.Len(t, respListTitles.SoftwareTitles, 2)

	var respGetSetupExperience getSetupExperienceSoftwareResponse
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/software", getSetupExperienceSoftwareRequest{}, http.StatusOK, &respGetSetupExperience, "team_id", fmt.Sprint(team1.ID))
	require.Len(t, respGetSetupExperience.SoftwareTitles, 2)

	var respPutSetupExperience putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/latest/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{TeamID: team1.ID, TitleIDs: []uint{respListTitles.SoftwareTitles[0].ID, respListTitles.SoftwareTitles[1].ID}}, http.StatusOK, &respPutSetupExperience)
	require.Nil(t, respPutSetupExperience.error())

	s.DoJSON("GET", "/api/latest/fleet/setup_experience/software", getSetupExperienceSoftwareRequest{}, http.StatusOK, &respGetSetupExperience, "team_id", fmt.Sprint(team1.ID))
	require.Len(t, respGetSetupExperience.SoftwareTitles, 2)

	desktopToken := uuid.New().String()
	mdmDevice := mdmtest.NewTestMDMClientAppleDesktopManual(s.server.URL, desktopToken)
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
		TeamID:          &team1.ID,

		UUID:           mdmDevice.UUID,
		HardwareSerial: mdmDevice.SerialNumber,
	})
	require.NoError(t, err)

	err = ds.SetOrUpdateDeviceAuthToken(context.Background(), fleetHost.ID, desktopToken)
	require.NoError(t, err)

	orbitNodeKey := setOrbitEnrollment(t, fleetHost, ds)

	err = mdmDevice.Enroll()
	require.NoError(t, err)

	var orbitRes getOrbitSetupExperienceStatusResponse
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", getOrbitSetupExperienceStatusRequest{OrbitNodeKey: orbitNodeKey}, http.StatusOK, &orbitRes)

	require.Len(t, orbitRes.Results.Software, 2)

	var vppFound, softwareFound bool
	for _, res := range orbitRes.Results.Software {

		if res.Name == "file1" {
			softwareFound = true
		}
		if res.Name == "vpp_app_1" {
			vppFound = true
		}
	}

	require.True(t, vppFound, "vpp app not found in status results")
	require.True(t, softwareFound, "software installer app not found in status results")

	awaitingConfig, err := s.ds.GetHostAwaitingConfiguration(ctx, fleetHost.UUID)
	require.NoError(t, err)
	require.True(t, awaitingConfig)
}

func (s *integrationMDMTestSuite) TestWindowsMigrationEnabled() {
	t := s.T()

	var acResp appConfigResponse
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.True(t, acResp.MDM.WindowsEnabledAndConfigured)
	require.False(t, acResp.MDM.WindowsMigrationEnabled)

	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"windows_migration_enabled": true
		}
	}`), http.StatusOK, &acResp)
	require.True(t, acResp.MDM.WindowsEnabledAndConfigured)
	require.True(t, acResp.MDM.WindowsMigrationEnabled)
	s.lastActivityMatches(fleet.ActivityTypeEnabledWindowsMDMMigration{}.ActivityName(), "", 0)

	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"windows_migration_enabled": false
		}
	}`), http.StatusOK, &acResp)
	require.True(t, acResp.MDM.WindowsEnabledAndConfigured)
	require.False(t, acResp.MDM.WindowsMigrationEnabled)
	s.lastActivityMatches(fleet.ActivityTypeDisabledWindowsMDMMigration{}.ActivityName(), "", 0)

	// set migrations back to true to see if they turn false when turning MDM off
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"windows_migration_enabled": true
		}
	}`), http.StatusOK, &acResp)
	require.True(t, acResp.MDM.WindowsEnabledAndConfigured)
	require.True(t, acResp.MDM.WindowsMigrationEnabled)
	lastEnabledID := s.lastActivityMatches(fleet.ActivityTypeEnabledWindowsMDMMigration{}.ActivityName(), "", 0)

	// not providing any mdm update should leave the current values
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {}
	}`), http.StatusOK, &acResp)
	require.True(t, acResp.MDM.WindowsEnabledAndConfigured)
	require.True(t, acResp.MDM.WindowsMigrationEnabled)
	// no new activity was created
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledWindowsMDMMigration{}.ActivityName(), "", lastEnabledID)

	// set to true again does not generate a new activity, was already true
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"windows_migration_enabled": true
		}
	}`), http.StatusOK, &acResp)
	require.True(t, acResp.MDM.WindowsEnabledAndConfigured)
	require.True(t, acResp.MDM.WindowsMigrationEnabled)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledWindowsMDMMigration{}.ActivityName(), "", lastEnabledID)

	res := s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"windows_enabled_and_configured": false,
			"windows_migration_enabled": true
		}
	}`), http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Windows MDM is not enabled")

	// turn off Windows MDM and try to enable migrations in a distinct call
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"windows_enabled_and_configured": false
		}
	}`), http.StatusOK, &acResp)
	require.False(t, acResp.MDM.WindowsEnabledAndConfigured)
	require.False(t, acResp.MDM.WindowsMigrationEnabled)

	res = s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"windows_migration_enabled": true
		}
	}`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Windows MDM is not enabled")
}

func (s *integrationMDMTestSuite) TestHostsCantTurnMDMOff() {
	t := s.T()
	iOSHost, _ := s.createAppleMobileHostThenEnrollMDM("ios")
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), iOSHost.ID, false, true, "https://foo.com", true, "", ""))

	r := s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", iOSHost.ID), nil, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(r.Body), fleet.CantTurnOffMDMForIOSOrIPadOSMessage)

	iPadOSHost, _ := s.createAppleMobileHostThenEnrollMDM("ipados")
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), iPadOSHost.ID, false, true, "https://foo.com", true, "", ""))

	r = s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", iPadOSHost.ID), nil, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(r.Body), fleet.CantTurnOffMDMForIOSOrIPadOSMessage)

	winHost, _ := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
	r = s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", winHost.ID), nil, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(r.Body), fleet.CantTurnOffMDMForWindowsHostsMessage)
}
