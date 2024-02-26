package service

import (
	"bytes"
	"context"
	"crypto/md5" // nolint:gosec // used only for tests
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
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

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	mdm_types "github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/live_query/live_query_mock"
	servermdm "github.com/fleetdm/fleet/v4/server/mdm"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	nanomdm_pushsvc "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/service"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/mock"
	"github.com/fleetdm/fleet/v4/server/service/osquery_utils"
	"github.com/fleetdm/fleet/v4/server/service/schedule"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/fleetdm/fleet/v4/server/worker"
	"github.com/go-kit/kit/log"
	kitlog "github.com/go-kit/kit/log"
	"github.com/google/uuid"
	"github.com/groob/plist"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	nanodep_client "github.com/micromdm/nanodep/client"
	"github.com/micromdm/nanodep/godep"
	nanodep_storage "github.com/micromdm/nanodep/storage"
	"github.com/micromdm/nanodep/tokenpki"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.mozilla.org/pkcs7"
)

func TestIntegrationsMDM(t *testing.T) {
	testingSuite := new(integrationMDMTestSuite)
	testingSuite.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

type integrationMDMTestSuite struct {
	suite.Suite
	withServer
	fleetCfg             config.FleetConfig
	fleetDMNextCSRStatus atomic.Value
	pushProvider         *mock.APNSPushProvider
	depStorage           nanodep_storage.AllStorage
	depSchedule          *schedule.Schedule
	profileSchedule      *schedule.Schedule
	onProfileJobDone     func() // function called when profileSchedule.Trigger() job completed
	onDEPScheduleDone    func() // function called when depSchedule.Trigger() job completed
	mdmStorage           *mysql.NanoMDMStorage
	worker               *worker.Worker
	mdmCommander         *apple_mdm.MDMAppleCommander
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

	testCert, testKey, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(s.T(), err)
	testCertPEM := tokenpki.PEMCertificate(testCert.Raw)
	testKeyPEM := tokenpki.PEMRSAPrivateKey(testKey)

	fleetCfg := config.TestConfig()
	config.SetTestMDMConfig(s.T(), &fleetCfg, testCertPEM, testKeyPEM, testBMToken, "")
	fleetCfg.Osquery.EnrollCooldown = 0

	mdmStorage, err := s.ds.NewMDMAppleMDMStorage(testCertPEM, testKeyPEM)
	require.NoError(s.T(), err)
	depStorage, err := s.ds.NewMDMAppleDEPStorage(*testBMToken)
	require.NoError(s.T(), err)
	scepStorage, err := s.ds.NewSCEPDepot(testCertPEM, testKeyPEM)
	require.NoError(s.T(), err)

	pushFactory, pushProvider := newMockAPNSPushProviderFactory()
	mdmPushService := nanomdm_pushsvc.New(
		mdmStorage,
		mdmStorage,
		pushFactory,
		NewNanoMDMLogger(kitlog.NewJSONLogger(os.Stdout)),
	)
	mdmCommander := apple_mdm.NewMDMAppleCommander(mdmStorage, mdmPushService)
	redisPool := redistest.SetupRedis(s.T(), "zz", false, false, false)
	s.withServer.lq = live_query_mock.New(s.T())

	var depSchedule *schedule.Schedule
	var profileSchedule *schedule.Schedule
	config := TestServerOpts{
		License: &fleet.LicenseInfo{
			Tier: fleet.TierPremium,
		},
		FleetConfig: &fleetCfg,
		MDMStorage:  mdmStorage,
		DEPStorage:  depStorage,
		SCEPStorage: scepStorage,
		MDMPusher:   mdmPushService,
		Pool:        redisPool,
		Lq:          s.lq,
		StartCronSchedules: []TestNewScheduleFunc{
			func(ctx context.Context, ds fleet.Datastore) fleet.NewCronScheduleFunc {
				return func() (fleet.CronSchedule, error) {
					const name = string(fleet.CronAppleMDMDEPProfileAssigner)
					logger := kitlog.NewJSONLogger(os.Stdout)
					fleetSyncer := apple_mdm.NewDEPService(ds, depStorage, logger)
					depSchedule = schedule.New(
						ctx, name, s.T().Name(), 1*time.Hour, ds, ds,
						schedule.WithLogger(logger),
						schedule.WithJob("dep_syncer", func(ctx context.Context) error {
							if s.onDEPScheduleDone != nil {
								defer s.onDEPScheduleDone()
							}
							err := fleetSyncer.RunAssigner(ctx)
							require.NoError(s.T(), err)
							return err
						}),
					)
					return depSchedule, nil
				}
			},
			func(ctx context.Context, ds fleet.Datastore) fleet.NewCronScheduleFunc {
				return func() (fleet.CronSchedule, error) {
					const name = string(fleet.CronMDMAppleProfileManager)
					logger := kitlog.NewJSONLogger(os.Stdout)
					profileSchedule = schedule.New(
						ctx, name, s.T().Name(), 1*time.Hour, ds, ds,
						schedule.WithLogger(logger),
						schedule.WithJob("manage_apple_profiles", func(ctx context.Context) error {
							if s.onProfileJobDone != nil {
								s.onProfileJobDone()
							}
							err := ReconcileAppleProfiles(ctx, ds, mdmCommander, logger)
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
		},
		APNSTopic: "com.apple.mgmt.External.10ac3ce5-4668-4e58-b69a-b2b5ce667589",
	}
	users, server := RunServerForTestsWithDS(s.T(), s.ds, &config)
	s.server = server
	s.users = users
	s.token = s.getTestAdminToken()
	s.cachedAdminToken = s.token
	s.fleetCfg = fleetCfg
	s.pushProvider = pushProvider
	s.depStorage = depStorage
	s.depSchedule = depSchedule
	s.profileSchedule = profileSchedule
	s.mdmStorage = mdmStorage
	s.mdmCommander = mdmCommander

	macosJob := &worker.MacosSetupAssistant{
		Datastore:  s.ds,
		Log:        kitlog.NewJSONLogger(os.Stdout),
		DEPService: apple_mdm.NewDEPService(s.ds, depStorage, kitlog.NewJSONLogger(os.Stdout)),
		DEPClient:  apple_mdm.NewDEPClient(depStorage, s.ds, kitlog.NewJSONLogger(os.Stdout)),
	}
	appleMDMJob := &worker.AppleMDM{
		Datastore: s.ds,
		Log:       kitlog.NewJSONLogger(os.Stdout),
		Commander: mdmCommander,
	}
	workr := worker.NewWorker(s.ds, kitlog.NewJSONLogger(os.Stdout))
	workr.TestIgnoreUnknownJobs = true
	workr.Register(macosJob, appleMDMJob)
	s.worker = workr

	fleetdmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := s.fleetDMNextCSRStatus.Swap(http.StatusOK)
		w.WriteHeader(status.(int))
		_, _ = w.Write([]byte(fmt.Sprintf("status: %d", status)))
	}))
	s.T().Setenv("TEST_FLEETDM_API_URL", fleetdmSrv.URL)

	appConf, err = s.ds.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.ServerSettings.ServerURL = server.URL
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)

	s.T().Cleanup(fleetdmSrv.Close)
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
	// ensure global Windows OS updates are always disabled for the next test
	appCfg.MDM.WindowsUpdates = mdm_types.WindowsUpdates{}
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
	// clear any pending worker job
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM jobs")
		return err
	})

	// clear any mdm windows enrollments
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM mdm_windows_enrollments")
		return err
	})
}

func (s *integrationMDMTestSuite) mockDEPResponse(handler http.Handler) {
	t := s.T()
	srv := httptest.NewServer(handler)
	err := s.depStorage.StoreConfig(context.Background(), apple_mdm.DEPName, &nanodep_client.Config{BaseURL: srv.URL})
	require.NoError(t, err)
	t.Cleanup(func() {
		srv.Close()
		err := s.depStorage.StoreConfig(context.Background(), apple_mdm.DEPName, &nanodep_client.Config{BaseURL: nanodep_client.DefaultBaseURL})
		require.NoError(t, err)
	})
}

func (s *integrationMDMTestSuite) awaitTriggerProfileSchedule(t *testing.T) {
	// two jobs running sequentially (macOS then Windows) on the same schedule
	var wg sync.WaitGroup
	wg.Add(2)
	s.onProfileJobDone = wg.Done
	_, err := s.profileSchedule.Trigger()
	require.NoError(t, err)
	wg.Wait()
}

func (s *integrationMDMTestSuite) TestGetBootstrapToken() {
	// see https://developer.apple.com/documentation/devicemanagement/get_bootstrap_token
	t := s.T()
	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.fleetCfg.MDM.AppleSCEPChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	})
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
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple", nil, http.StatusOK, &mdmResp)
	// returned values are dummy, this is a test certificate
	require.Equal(t, "FleetDM", mdmResp.Issuer)
	require.NotZero(t, mdmResp.SerialNumber)
	require.Equal(t, "FleetDM", mdmResp.CommonName)
	require.NotZero(t, mdmResp.RenewDate)

	s.mockDEPResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
		case "/account":
			_, _ = w.Write([]byte(`{"admin_id": "abc", "org_name": "test_org"}`))
		}
	}))
	var getAppleBMResp getAppleBMResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple_bm", nil, http.StatusOK, &getAppleBMResp)
	require.NoError(t, getAppleBMResp.Err)
	require.Equal(t, "abc", getAppleBMResp.AppleID)
	require.Equal(t, "test_org", getAppleBMResp.OrgName)
	require.Equal(t, s.server.URL+"/mdm/apple/mdm", getAppleBMResp.MDMServerURL)
	require.Empty(t, getAppleBMResp.DefaultTeam)

	// create a new team
	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name(),
		Description: "desc",
	})
	require.NoError(t, err)
	// set the default bm assignment to that team
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": {
			"apple_bm_default_team": %q
		}
	}`, tm.Name)), http.StatusOK, &acResp)

	// try again, this time we get a default team in the response
	getAppleBMResp = getAppleBMResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple_bm", nil, http.StatusOK, &getAppleBMResp)
	require.NoError(t, getAppleBMResp.Err)
	require.Equal(t, "abc", getAppleBMResp.AppleID)
	require.Equal(t, "test_org", getAppleBMResp.OrgName)
	require.Equal(t, s.server.URL+"/mdm/apple/mdm", getAppleBMResp.MDMServerURL)
	require.Equal(t, tm.Name, getAppleBMResp.DefaultTeam)
}

func (s *integrationMDMTestSuite) TestABMExpiredToken() {
	t := s.T()
	var returnType string
	s.mockDEPResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	// not signed error flips the AppleBMTermsExpired flag
	returnType = "not_signed"
	res := s.DoRaw("GET", "/api/latest/fleet/mdm/apple_bm", nil, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "DEP auth error: 403 Forbidden")

	config = s.getConfig()
	require.True(t, config.MDM.AppleBMTermsExpired)

	// a successful call clears it
	returnType = "success"
	s.DoRaw("GET", "/api/latest/fleet/mdm/apple_bm", nil, http.StatusOK)

	config = s.getConfig()
	require.False(t, config.MDM.AppleBMTermsExpired)

	// an unauthorized call returns 400 but does not flip the terms expired flag
	returnType = "unauthorized"
	res = s.DoRaw("GET", "/api/latest/fleet/mdm/apple_bm", nil, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Apple Business Manager certificate or server token is invalid")

	config = s.getConfig()
	require.False(t, config.MDM.AppleBMTermsExpired)
}

func (s *integrationMDMTestSuite) TestAppleProfileManagement() {
	t := s.T()
	ctx := context.Background()

	err := s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: t.Name()}})
	require.NoError(t, err)

	globalProfiles := [][]byte{
		mobileconfigForTest("N1", "I1"),
		mobileconfigForTest("N2", "I2"),
	}
	wantGlobalProfiles := append(globalProfiles, setupExpectedFleetdProfile(t, s.server.URL, t.Name(), nil))

	// add global profiles
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: globalProfiles}, http.StatusNoContent)

	// create a new team
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "batch_set_mdm_profiles"})
	require.NoError(t, err)

	// add an enroll secret so the fleetd profiles differ
	var teamResp teamEnrollSecretsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", tm.ID),
		modifyTeamEnrollSecretsRequest{
			Secrets: []fleet.EnrollSecret{{Secret: "team1_enroll_sec"}},
		}, http.StatusOK, &teamResp)

	teamProfiles := [][]byte{
		mobileconfigForTest("N3", "I3"),
	}
	wantTeamProfiles := append(teamProfiles, setupExpectedFleetdProfile(t, s.server.URL, "team1_enroll_sec", &tm.ID))
	// add profiles to the team
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: teamProfiles}, http.StatusNoContent, "team_id", strconv.Itoa(int(tm.ID)))

	// create a non-macOS host
	_, err = s.ds.NewHost(context.Background(), &fleet.Host{
		ID:            1,
		OsqueryHostID: ptr.String("non-macos-host"),
		NodeKey:       ptr.String("non-macos-host"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.non.macos", t.Name()),
		Platform:      "windows",
	})
	require.NoError(t, err)

	// create a host that's not enrolled into MDM
	_, err = s.ds.NewHost(context.Background(), &fleet.Host{
		ID:            2,
		OsqueryHostID: ptr.String("not-mdm-enrolled"),
		NodeKey:       ptr.String("not-mdm-enrolled"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.not.enrolled", t.Name()),
		Platform:      "darwin",
	})
	require.NoError(t, err)

	// Create a host and then enroll to MDM.
	host, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setupPusher(s, t, mdmDevice)

	// trigger a profile sync
	s.awaitTriggerProfileSchedule(t)
	installs, removes := checkNextPayloads(t, mdmDevice, false)
	// verify that we received all profiles
	require.ElementsMatch(t, wantGlobalProfiles, installs)
	require.Empty(t, removes)

	expectedNoTeamSummary := fleet.MDMProfilesSummary{
		Pending:   0,
		Failed:    0,
		Verifying: 1,
		Verified:  0,
	}
	expectedTeamSummary := fleet.MDMProfilesSummary{}
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamSummary, &expectedNoTeamSummary)
	s.checkMDMProfilesSummaries(t, &tm.ID, expectedTeamSummary, &expectedTeamSummary) // empty because no hosts in team

	// add the host to a team
	err = s.ds.AddHostsToTeam(ctx, &tm.ID, []uint{host.ID})
	require.NoError(t, err)

	// trigger a profile sync
	s.awaitTriggerProfileSchedule(t)
	installs, removes = checkNextPayloads(t, mdmDevice, false)
	// verify that we should install the team profile
	require.ElementsMatch(t, wantTeamProfiles, installs)
	// verify that we should delete both profiles
	require.ElementsMatch(t, []string{"I1", "I2"}, removes)

	expectedNoTeamSummary = fleet.MDMProfilesSummary{}
	expectedTeamSummary = fleet.MDMProfilesSummary{
		Pending:   0,
		Failed:    0,
		Verifying: 1,
		Verified:  0,
	}
	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamSummary, &expectedNoTeamSummary) // empty because host was transferred
	s.checkMDMProfilesSummaries(t, &tm.ID, expectedTeamSummary, &expectedTeamSummary)  // host now verifying team profiles

	// set new team profiles (delete + addition)
	teamProfiles = [][]byte{
		mobileconfigForTest("N4", "I4"),
		mobileconfigForTest("N5", "I5"),
	}
	wantTeamProfiles = teamProfiles
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: teamProfiles}, http.StatusNoContent, "team_id", strconv.Itoa(int(tm.ID)))

	// trigger a profile sync
	s.awaitTriggerProfileSchedule(t)
	installs, removes = checkNextPayloads(t, mdmDevice, false)
	// verify that we should install the team profiles
	require.ElementsMatch(t, wantTeamProfiles, installs)
	// verify that we should delete the old team profiles
	require.ElementsMatch(t, []string{"I3"}, removes)

	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamSummary, &expectedNoTeamSummary) // empty because host was transferred
	s.checkMDMProfilesSummaries(t, &tm.ID, expectedTeamSummary, &expectedTeamSummary)  // host still verifying team profiles

	// with no changes
	s.awaitTriggerProfileSchedule(t)
	installs, removes = checkNextPayloads(t, mdmDevice, false)
	require.Empty(t, installs)
	require.Empty(t, removes)

	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", host.ID), getHostRequest{}, http.StatusOK, &hostResp)
	require.NotEmpty(t, hostResp.Host.MDM.Profiles)
	resProfiles := *hostResp.Host.MDM.Profiles
	// one extra profile for the fleetd config
	require.Len(t, resProfiles, len(wantTeamProfiles)+1)

	s.checkMDMProfilesSummaries(t, nil, expectedNoTeamSummary, &expectedNoTeamSummary) // empty because host was transferred
	s.checkMDMProfilesSummaries(t, &tm.ID, expectedTeamSummary, &expectedTeamSummary)  // host still verifying team profiles
}

func (s *integrationMDMTestSuite) TestAppleProfileRetries() {
	t := s.T()
	ctx := context.Background()

	enrollSecret := "test-profile-retries-secret"
	err := s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: enrollSecret}})
	require.NoError(t, err)

	testProfiles := [][]byte{
		mobileconfigForTest("N1", "I1"),
		mobileconfigForTest("N2", "I2"),
	}
	initialExpectedProfiles := append(testProfiles, setupExpectedFleetdProfile(t, s.server.URL, enrollSecret, nil))

	h, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setupPusher(s, t, mdmDevice)

	expectedProfileStatuses := map[string]fleet.MDMDeliveryStatus{
		"I1": fleet.MDMDeliveryVerifying,
		"I2": fleet.MDMDeliveryVerifying,
		mobileconfig.FleetdConfigPayloadIdentifier: fleet.MDMDeliveryVerifying,
	}
	checkProfilesStatus := func(t *testing.T) {
		storedProfs, err := s.ds.GetHostMDMAppleProfiles(ctx, h.UUID)
		require.NoError(t, err)
		require.Len(t, storedProfs, len(expectedProfileStatuses))
		for _, p := range storedProfs {
			want, ok := expectedProfileStatuses[p.Identifier]
			require.True(t, ok, "unexpected profile: %s", p.Identifier)
			require.Equal(t, want, *p.Status, "expected status %s but got %s for profile: %s", want, *p.Status, p.Identifier)
		}
	}

	expectedRetryCounts := map[string]uint{
		"I1": 0,
		"I2": 0,
		mobileconfig.FleetdConfigPayloadIdentifier: 0,
	}
	checkRetryCounts := func(t *testing.T) {
		counts, err := s.ds.GetHostMDMProfilesRetryCounts(ctx, h)
		require.NoError(t, err)
		require.Len(t, counts, len(expectedRetryCounts))
		for _, c := range counts {
			want, ok := expectedRetryCounts[c.ProfileIdentifier]
			require.True(t, ok, "unexpected profile: %s", c.ProfileIdentifier)
			require.Equal(t, want, c.Retries, "expected retry count %d but got %d for profile: %s", want, c.Retries, c.ProfileIdentifier)
		}
	}

	hostProfsByIdent := map[string]*fleet.HostMacOSProfile{
		"I1": {
			Identifier:  "I1",
			DisplayName: "N1",
			InstallDate: time.Now().Add(15 * time.Minute),
		},
		"I2": {
			Identifier:  "I2",
			DisplayName: "N2",
			InstallDate: time.Now().Add(15 * time.Minute),
		},
		mobileconfig.FleetdConfigPayloadIdentifier: {
			Identifier:  mobileconfig.FleetdConfigPayloadIdentifier,
			DisplayName: "Fleetd configuration",
			InstallDate: time.Now().Add(15 * time.Minute),
		},
	}
	reportHostProfs := func(t *testing.T, identifiers ...string) {
		report := make(map[string]*fleet.HostMacOSProfile, len(hostProfsByIdent))
		for _, ident := range identifiers {
			report[ident] = hostProfsByIdent[ident]
		}
		require.NoError(t, apple_mdm.VerifyHostMDMProfiles(ctx, s.ds, h, report))
	}

	setProfileUploadedAt := func(t *testing.T, uploadedAt time.Time, identifiers ...interface{}) {
		bindVars := strings.TrimSuffix(strings.Repeat("?, ", len(identifiers)), ", ")
		stmt := fmt.Sprintf("UPDATE mdm_apple_configuration_profiles SET uploaded_at = ? WHERE identifier IN(%s)", bindVars)
		args := append([]interface{}{uploadedAt}, identifiers...)
		mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
			_, err := tx.ExecContext(ctx, stmt, args...)
			return err
		})
	}

	t.Run("retry after verifying", func(t *testing.T) {
		// upload test profiles then simulate expired grace period by setting updated_at timestamp of profiles back by 48 hours
		s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
		setProfileUploadedAt(t, time.Now().Add(-48*time.Hour), "I1", "I2", mobileconfig.FleetdConfigPayloadIdentifier)

		// trigger initial profile sync and confirm that we received all profiles
		s.awaitTriggerProfileSchedule(t)
		installs, removes := checkNextPayloads(t, mdmDevice, false)
		require.ElementsMatch(t, initialExpectedProfiles, installs)
		require.Empty(t, removes)

		checkProfilesStatus(t) // all profiles verifying
		checkRetryCounts(t)    // no retries yet

		// report osquery results with I2 missing and confirm I2 marked as pending and other profiles are marked as verified
		reportHostProfs(t, "I1", mobileconfig.FleetdConfigPayloadIdentifier)
		expectedProfileStatuses["I2"] = fleet.MDMDeliveryPending
		expectedProfileStatuses["I1"] = fleet.MDMDeliveryVerified
		expectedProfileStatuses[mobileconfig.FleetdConfigPayloadIdentifier] = fleet.MDMDeliveryVerified
		checkProfilesStatus(t)
		expectedRetryCounts["I2"] = 1
		checkRetryCounts(t)

		// trigger a profile sync and confirm that the install profile command for I2 was resent
		s.awaitTriggerProfileSchedule(t)
		installs, removes = checkNextPayloads(t, mdmDevice, false)
		require.ElementsMatch(t, [][]byte{initialExpectedProfiles[1]}, installs)
		require.Empty(t, removes)

		// report osquery results with I2 present and confirm that all profiles are verified
		reportHostProfs(t, "I1", "I2", mobileconfig.FleetdConfigPayloadIdentifier)
		expectedProfileStatuses["I2"] = fleet.MDMDeliveryVerified
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// trigger a profile sync and confirm that no profiles were sent
		s.awaitTriggerProfileSchedule(t)
		installs, removes = checkNextPayloads(t, mdmDevice, false)
		require.Empty(t, installs)
		require.Empty(t, removes)
	})

	t.Run("retry after verification", func(t *testing.T) {
		// report osquery results with I1 missing and confirm that the I1 marked as pending (initial retry)
		reportHostProfs(t, "I2", mobileconfig.FleetdConfigPayloadIdentifier)
		expectedProfileStatuses["I1"] = fleet.MDMDeliveryPending
		checkProfilesStatus(t)
		expectedRetryCounts["I1"] = 1
		checkRetryCounts(t)

		// trigger a profile sync and confirm that the install profile command for I1 was resent
		s.awaitTriggerProfileSchedule(t)
		installs, removes := checkNextPayloads(t, mdmDevice, false)
		require.ElementsMatch(t, [][]byte{initialExpectedProfiles[0]}, installs)
		require.Empty(t, removes)

		// report osquery results with I1 missing again and confirm that the I1 marked as failed (max retries exceeded)
		reportHostProfs(t, "I2", mobileconfig.FleetdConfigPayloadIdentifier)
		expectedProfileStatuses["I1"] = fleet.MDMDeliveryFailed
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// trigger a profile sync and confirm that the install profile command for I1 was not resent
		s.awaitTriggerProfileSchedule(t)
		installs, removes = checkNextPayloads(t, mdmDevice, false)
		require.Empty(t, installs)
		require.Empty(t, removes)
	})

	t.Run("retry after device error", func(t *testing.T) {
		// add another profile and set the updated_at timestamp back by 48 hours
		newProfile := mobileconfigForTest("N3", "I3")
		testProfiles = append(testProfiles, newProfile)
		s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
		setProfileUploadedAt(t, time.Now().Add(-48*time.Hour), "I1", "I2", mobileconfig.FleetdConfigPayloadIdentifier, "I3")

		// trigger a profile sync and confirm that the install profile command for I3 was sent and
		// simulate a device error
		s.awaitTriggerProfileSchedule(t)
		installs, removes := checkNextPayloads(t, mdmDevice, true)
		require.ElementsMatch(t, [][]byte{newProfile}, installs)
		require.Empty(t, removes)
		expectedProfileStatuses["I3"] = fleet.MDMDeliveryPending
		checkProfilesStatus(t)
		expectedRetryCounts["I3"] = 1
		checkRetryCounts(t)

		// trigger a profile sync and confirm that the install profile command for I3 was sent and
		// simulate a device ack
		s.awaitTriggerProfileSchedule(t)
		installs, removes = checkNextPayloads(t, mdmDevice, false)
		require.ElementsMatch(t, [][]byte{newProfile}, installs)
		require.Empty(t, removes)
		expectedProfileStatuses["I3"] = fleet.MDMDeliveryVerifying
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// report osquery results with I3 missing and confirm that the I3 marked as failed (max
		// retries exceeded)
		reportHostProfs(t, "I2", mobileconfig.FleetdConfigPayloadIdentifier)
		expectedProfileStatuses["I3"] = fleet.MDMDeliveryFailed
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// trigger a profile sync and confirm that the install profile command for I3 was not resent
		s.awaitTriggerProfileSchedule(t)
		installs, removes = checkNextPayloads(t, mdmDevice, false)
		require.Empty(t, installs)
		require.Empty(t, removes)
	})

	t.Run("repeated device error", func(t *testing.T) {
		// add another profile and set the updated_at timestamp back by 48 hours
		newProfile := mobileconfigForTest("N4", "I4")
		testProfiles = append(testProfiles, newProfile)
		s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
		setProfileUploadedAt(t, time.Now().Add(-48*time.Hour), "I1", "I2", mobileconfig.FleetdConfigPayloadIdentifier, "I3", "I4")

		// trigger a profile sync and confirm that the install profile command for I3 was sent and
		// simulate a device error
		s.awaitTriggerProfileSchedule(t)
		installs, removes := checkNextPayloads(t, mdmDevice, true)
		require.ElementsMatch(t, [][]byte{newProfile}, installs)
		require.Empty(t, removes)
		expectedProfileStatuses["I4"] = fleet.MDMDeliveryPending
		checkProfilesStatus(t)
		expectedRetryCounts["I4"] = 1
		checkRetryCounts(t)

		// trigger a profile sync and confirm that the install profile command for I4 was sent and
		// simulate a second device error
		s.awaitTriggerProfileSchedule(t)
		installs, removes = checkNextPayloads(t, mdmDevice, true)
		require.ElementsMatch(t, [][]byte{newProfile}, installs)
		require.Empty(t, removes)
		expectedProfileStatuses["I4"] = fleet.MDMDeliveryFailed
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// trigger a profile sync and confirm that the install profile command for I3 was not resent
		s.awaitTriggerProfileSchedule(t)
		installs, removes = checkNextPayloads(t, mdmDevice, false)
		require.Empty(t, installs)
		require.Empty(t, removes)
	})

	t.Run("retry count does not reset", func(t *testing.T) {
		// add another profile and set the updated_at timestamp back by 48 hours
		newProfile := mobileconfigForTest("N5", "I5")
		testProfiles = append(testProfiles, newProfile)
		hostProfsByIdent["I5"] = &fleet.HostMacOSProfile{Identifier: "I5", DisplayName: "N5", InstallDate: time.Now()}
		s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
		setProfileUploadedAt(t, time.Now().Add(-48*time.Hour), "I1", "I2", mobileconfig.FleetdConfigPayloadIdentifier, "I3", "I4", "I5")

		// trigger a profile sync and confirm that the install profile command for I3 was sent and
		// simulate a device error
		s.awaitTriggerProfileSchedule(t)
		installs, removes := checkNextPayloads(t, mdmDevice, true)
		require.ElementsMatch(t, [][]byte{newProfile}, installs)
		require.Empty(t, removes)
		expectedProfileStatuses["I5"] = fleet.MDMDeliveryPending
		checkProfilesStatus(t)
		expectedRetryCounts["I5"] = 1
		checkRetryCounts(t)

		// trigger a profile sync and confirm that the install profile command for I5 was sent and
		// simulate a device ack
		s.awaitTriggerProfileSchedule(t)
		installs, removes = checkNextPayloads(t, mdmDevice, false)
		require.ElementsMatch(t, [][]byte{newProfile}, installs)
		require.Empty(t, removes)
		expectedProfileStatuses["I5"] = fleet.MDMDeliveryVerifying
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// report osquery results with I5 found and confirm that the I5 marked as verified
		reportHostProfs(t, "I2", mobileconfig.FleetdConfigPayloadIdentifier, "I5")
		expectedProfileStatuses["I5"] = fleet.MDMDeliveryVerified
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// trigger a profile sync and confirm that the install profile command for I5 was not resent
		s.awaitTriggerProfileSchedule(t)
		installs, removes = checkNextPayloads(t, mdmDevice, false)
		require.Empty(t, installs)
		require.Empty(t, removes)

		// report osquery results again, this time I5 is missing and confirm that the I5 marked as
		// failed (max retries exceeded)
		reportHostProfs(t, "I2", mobileconfig.FleetdConfigPayloadIdentifier)
		expectedProfileStatuses["I5"] = fleet.MDMDeliveryFailed
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// trigger a profile sync and confirm that the install profile command for I5 was not resent
		s.awaitTriggerProfileSchedule(t)
		installs, removes = checkNextPayloads(t, mdmDevice, false)
		require.Empty(t, installs)
		require.Empty(t, removes)
	})
}

func (s *integrationMDMTestSuite) TestWindowsProfileRetries() {
	t := s.T()
	ctx := context.Background()

	testProfiles := []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: syncml.ForTestWithData(map[string]string{"L1": "D1"})},
		{Name: "N2", Contents: syncml.ForTestWithData(map[string]string{"L2": "D2", "L3": "D3"})},
	}

	h, mdmDevice := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	expectedProfileStatuses := map[string]fleet.MDMDeliveryStatus{
		"N1": fleet.MDMDeliveryVerifying,
		"N2": fleet.MDMDeliveryVerifying,
	}
	checkProfilesStatus := func(t *testing.T) {
		storedProfs, err := s.ds.GetHostMDMWindowsProfiles(ctx, h.UUID)
		require.NoError(t, err)
		require.Len(t, storedProfs, len(expectedProfileStatuses))
		for _, p := range storedProfs {
			want, ok := expectedProfileStatuses[p.Name]
			require.True(t, ok, "unexpected profile: %s", p.Name)
			require.Equal(t, want, *p.Status, "expected status %s but got %s for profile: %s", want, *p.Status, p.Name)
		}
	}

	expectedRetryCounts := map[string]uint{
		"N1": 0,
		"N2": 0,
	}
	checkRetryCounts := func(t *testing.T) {
		counts, err := s.ds.GetHostMDMProfilesRetryCounts(ctx, h)
		require.NoError(t, err)
		require.Len(t, counts, len(expectedRetryCounts))
		for _, c := range counts {
			want, ok := expectedRetryCounts[c.ProfileName]
			require.True(t, ok, "unexpected profile: %s", c.ProfileName)
			require.Equal(t, want, c.Retries, "expected retry count %d but got %d for profile: %s", want, c.Retries, c.ProfileName)
		}
	}

	type profileData struct {
		Status string
		LocURI string
		Data   string
	}
	hostProfileReports := map[string][]profileData{
		"N1": {{"200", "L1", "D1"}},
		"N2": {{"200", "L2", "D2"}, {"200", "L3", "D3"}},
	}
	reportHostProfs := func(t *testing.T, profileNames ...string) {
		var responseOps []*mdm_types.SyncMLCmd
		for _, profileName := range profileNames {
			report, ok := hostProfileReports[profileName]
			require.True(t, ok)

			for _, p := range report {
				ref := microsoft_mdm.HashLocURI(profileName, p.LocURI)
				responseOps = append(responseOps, &mdm_types.SyncMLCmd{
					XMLName: xml.Name{Local: mdm_types.CmdStatus},
					CmdID:   fleet.CmdID{Value: uuid.NewString()},
					CmdRef:  &ref,
					Data:    ptr.String(p.Status),
				})

				// the protocol can respond with only a `Status`
				// command if the status failed
				if p.Status != "200" || p.Data != "" {
					responseOps = append(responseOps, &mdm_types.SyncMLCmd{
						XMLName: xml.Name{Local: mdm_types.CmdResults},
						CmdID:   fleet.CmdID{Value: uuid.NewString()},
						CmdRef:  &ref,
						Items: []mdm_types.CmdItem{
							{Target: ptr.String(p.LocURI), Data: &fleet.RawXmlData{Content: p.Data}},
						},
					})
				}
			}
		}

		msg, err := createSyncMLMessage("2", "2", "foo", "bar", responseOps)
		require.NoError(t, err)
		out, err := xml.Marshal(msg)
		require.NoError(t, err)
		require.NoError(t, microsoft_mdm.VerifyHostMDMProfiles(ctx, s.ds, h, out))
	}

	verifyCommands := func(wantProfileInstalls int, status string) {
		s.awaitTriggerProfileSchedule(t)
		cmds, err := mdmDevice.StartManagementSession()
		require.NoError(t, err)
		// profile installs + 2 protocol commands acks
		require.Len(t, cmds, wantProfileInstalls+2)
		msgID, err := mdmDevice.GetCurrentMsgID()
		require.NoError(t, err)
		atomicCmds := 0
		for _, c := range cmds {
			if c.Verb == "Atomic" {
				atomicCmds++
			}
			mdmDevice.AppendResponse(fleet.SyncMLCmd{
				XMLName: xml.Name{Local: mdm_types.CmdStatus},
				MsgRef:  &msgID,
				CmdRef:  ptr.String(c.Cmd.CmdID.Value),
				Cmd:     ptr.String(c.Verb),
				Data:    ptr.String(status),
				Items:   nil,
				CmdID:   fleet.CmdID{Value: uuid.NewString()},
			})
		}
		require.Equal(t, wantProfileInstalls, atomicCmds)
		cmds, err = mdmDevice.SendResponse()
		require.NoError(t, err)
		// the ack of the message should be the only returned command
		require.Len(t, cmds, 1)
	}

	t.Run("retry after verifying", func(t *testing.T) {
		// upload test profiles then simulate expired grace period by setting updated_at timestamp of profiles back by 48 hours
		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
		// profiles to install + 2 boilerplate <Status>
		verifyCommands(len(testProfiles), syncml.CmdStatusOK)
		checkProfilesStatus(t) // all profiles verifying
		checkRetryCounts(t)    // no retries yet

		// report osquery results with N2 missing and confirm N2 marked
		// as verifying and other profiles are marked as verified
		reportHostProfs(t, "N1")
		expectedProfileStatuses["N2"] = fleet.MDMDeliveryPending
		expectedProfileStatuses["N1"] = fleet.MDMDeliveryVerified
		checkProfilesStatus(t)
		expectedRetryCounts["N2"] = 1
		checkRetryCounts(t)

		// report osquery results with N2 present and confirm that all profiles are verified
		verifyCommands(1, syncml.CmdStatusOK)
		reportHostProfs(t, "N1", "N2")
		expectedProfileStatuses["N2"] = fleet.MDMDeliveryVerified
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// trigger a profile sync and confirm that no profiles were sent
		verifyCommands(0, syncml.CmdStatusOK)
	})

	t.Run("retry after verification", func(t *testing.T) {
		// report osquery results with N1 missing and confirm that the N1 marked as pending (initial retry)
		reportHostProfs(t, "N2")
		expectedProfileStatuses["N1"] = fleet.MDMDeliveryPending
		checkProfilesStatus(t)
		expectedRetryCounts["N1"] = 1
		checkRetryCounts(t)

		// trigger a profile sync and confirm that the install profile command for N1 was resent
		verifyCommands(1, syncml.CmdStatusOK)

		// report osquery results with N1 missing again and confirm that the N1 marked as failed (max retries exceeded)
		reportHostProfs(t, "N2")
		expectedProfileStatuses["N1"] = fleet.MDMDeliveryFailed
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// trigger a profile sync and confirm that the install profile command for N1 was not resent
		verifyCommands(0, syncml.CmdStatusOK)
	})

	t.Run("retry after device error", func(t *testing.T) {
		// add another profile
		newProfile := syncml.ForTestWithData(map[string]string{"L3": "D3"})
		testProfiles = append(testProfiles, fleet.MDMProfileBatchPayload{
			Name:     "N3",
			Contents: newProfile,
		})
		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
		// trigger a profile sync and confirm that the install profile command for N3 was sent and
		// simulate a device error
		verifyCommands(1, syncml.CmdStatusAtomicFailed)
		expectedProfileStatuses["N3"] = fleet.MDMDeliveryPending
		checkProfilesStatus(t)
		expectedRetryCounts["N3"] = 1
		checkRetryCounts(t)

		// trigger a profile sync and confirm that the install profile command for N3 was sent and
		// simulate a device ack
		verifyCommands(1, syncml.CmdStatusOK)
		expectedProfileStatuses["N3"] = fleet.MDMDeliveryVerifying
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// report osquery results with N3 missing and confirm that the N3 marked as failed (max
		// retries exceeded)
		reportHostProfs(t, "N2")
		expectedProfileStatuses["N3"] = fleet.MDMDeliveryFailed
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// trigger a profile sync and confirm that the install profile command for N3 was not resent
		verifyCommands(0, syncml.CmdStatusOK)
	})

	t.Run("repeated device error", func(t *testing.T) {
		// add another profile
		testProfiles = append(testProfiles, fleet.MDMProfileBatchPayload{
			Name:     "N4",
			Contents: syncml.ForTestWithData(map[string]string{"L4": "D4"}),
		})
		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
		// trigger a profile sync and confirm that the install profile command for N4 was sent and
		// simulate a device error
		verifyCommands(1, syncml.CmdStatusAtomicFailed)
		expectedProfileStatuses["N4"] = fleet.MDMDeliveryPending
		checkProfilesStatus(t)
		expectedRetryCounts["N4"] = 1
		checkRetryCounts(t)

		// trigger a profile sync and confirm that the install profile
		// command for N4 was sent and simulate a second device error
		verifyCommands(1, syncml.CmdStatusAtomicFailed)
		expectedProfileStatuses["N4"] = fleet.MDMDeliveryFailed
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// trigger a profile sync and confirm that the install profile
		// command for N4 was not resent
		verifyCommands(0, syncml.CmdStatusOK)
	})

	t.Run("retry count does not reset", func(t *testing.T) {
		// add another profile
		testProfiles = append(testProfiles, fleet.MDMProfileBatchPayload{
			Name:     "N5",
			Contents: syncml.ForTestWithData(map[string]string{"L5": "D5"}),
		})
		// hostProfsByIdent["N5"] = &fleet.HostMacOSProfile{Identifier: "N5", DisplayName: "N5", InstallDate: time.Now()}
		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: testProfiles}, http.StatusNoContent)
		// trigger a profile sync and confirm that the install profile
		// command for N5 was sent and simulate a device error
		verifyCommands(1, syncml.CmdStatusAtomicFailed)
		expectedProfileStatuses["N5"] = fleet.MDMDeliveryPending
		checkProfilesStatus(t)
		expectedRetryCounts["N5"] = 1
		checkRetryCounts(t)

		// trigger a profile sync and confirm that the install profile
		// command for N5 was sent and simulate a device ack
		verifyCommands(1, syncml.CmdStatusOK)
		expectedProfileStatuses["N5"] = fleet.MDMDeliveryVerifying
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// report osquery results with N5 found and confirm that the N5 marked as verified
		hostProfileReports["N5"] = []profileData{{"200", "L5", "D5"}}
		reportHostProfs(t, "N2", "N5")
		expectedProfileStatuses["N5"] = fleet.MDMDeliveryVerified
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// trigger a profile sync and confirm that the install profile command for N5 was not resent
		verifyCommands(0, syncml.CmdStatusOK)

		// report osquery results again, this time N5 is missing and confirm that the N5 marked as
		// failed (max retries exceeded)
		reportHostProfs(t, "N2")
		expectedProfileStatuses["N5"] = fleet.MDMDeliveryFailed
		checkProfilesStatus(t)
		checkRetryCounts(t) // unchanged

		// trigger a profile sync and confirm that the install profile command for N5 was not resent
		verifyCommands(0, syncml.CmdStatusOK)
	})
}

func checkNextPayloads(t *testing.T, mdmDevice *mdmtest.TestAppleMDMClient, forceDeviceErr bool) ([][]byte, []string) {
	var cmd *micromdm.CommandPayload
	var err error
	installs := [][]byte{}
	removes := []string{}

	// on the first run, cmd will be nil and we need to
	// ping the server via idle
	// if after idle or acknowledge cmd is still nil, it
	// means there aren't any commands left to run
	for {
		if cmd == nil {
			cmd, err = mdmDevice.Idle()
		} else {
			if forceDeviceErr {
				cmd, err = mdmDevice.Err(cmd.CommandUUID, []mdm.ErrorChain{})
			} else {
				cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			}
		}
		require.NoError(t, err)

		if cmd == nil {
			break
		}

		switch cmd.Command.RequestType {
		case "InstallProfile":
			installs = append(installs, cmd.Command.InstallProfile.Payload)
		case "RemoveProfile":
			removes = append(removes, cmd.Command.RemoveProfile.Identifier)

		}
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

func setupPusher(s *integrationMDMTestSuite, t *testing.T, mdmDevice *mdmtest.TestAppleMDMClient) {
	origPush := s.pushProvider.PushFunc
	s.pushProvider.PushFunc = func(pushes []*mdm.Push) (map[string]*push.Response, error) {
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

func (s *integrationMDMTestSuite) TestPuppetMatchPreassignProfiles() {
	ctx := context.Background()
	t := s.T()

	// create a host enrolled in fleet
	mdmHost, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	s.runWorker()

	// create a host that's not enrolled into MDM
	nonMDMHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		OsqueryHostID: ptr.String("not-mdm-enrolled"),
		NodeKey:       ptr.String("not-mdm-enrolled"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.not.enrolled", t.Name()),
		Platform:      "darwin",
	})
	require.NoError(t, err)

	// create a setup assistant for no team, for this we need to:
	// 1. mock the ABM API, as it gets called to set the profile
	// 2. run the DEP schedule, as this registers the default profile
	s.mockDEPResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
	}))
	s.runDEPSchedule()
	noTeamProf := `{"x": 1}`
	var globalAsstResp createMDMAppleSetupAssistantResponse
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/enrollment_profile", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "no-team",
		EnrollmentProfile: json.RawMessage(noTeamProf),
	}, http.StatusOK, &globalAsstResp)

	// preassign an empty profile, fails
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{MDMApplePreassignProfilePayload: fleet.MDMApplePreassignProfilePayload{ExternalHostIdentifier: "empty", HostUUID: nonMDMHost.UUID, Profile: nil}}, http.StatusUnprocessableEntity)

	// preassign a valid profile to the MDM host
	prof1 := mobileconfigForTest("n1", "i1")
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{MDMApplePreassignProfilePayload: fleet.MDMApplePreassignProfilePayload{ExternalHostIdentifier: "mdm1", HostUUID: mdmHost.UUID, Profile: prof1}}, http.StatusNoContent)

	// preassign another valid profile to the MDM host
	prof2 := mobileconfigForTest("n2", "i2")
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{MDMApplePreassignProfilePayload: fleet.MDMApplePreassignProfilePayload{ExternalHostIdentifier: "mdm1", HostUUID: mdmHost.UUID, Profile: prof2, Group: "g1"}}, http.StatusNoContent)

	// preassign a valid profile to the non-MDM host, still works as the host is not validated in this call
	prof3 := mobileconfigForTest("n3", "i3")
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{MDMApplePreassignProfilePayload: fleet.MDMApplePreassignProfilePayload{ExternalHostIdentifier: "non-mdm", HostUUID: nonMDMHost.UUID, Profile: prof3, Group: "g2"}}, http.StatusNoContent)

	// match with an invalid external host id, succeeds as it is the same as if
	// there was no matching to do (no preassignment was done)
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/match", matchMDMApplePreassignmentRequest{ExternalHostIdentifier: "no-such-id"}, http.StatusNoContent)

	// match with the non-mdm host fails
	res := s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/match", matchMDMApplePreassignmentRequest{ExternalHostIdentifier: "non-mdm"}, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "host is not enrolled in Fleet MDM")

	// match with the mdm host succeeds and creates a team based on the group labels
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/match", matchMDMApplePreassignmentRequest{ExternalHostIdentifier: "mdm1"}, http.StatusNoContent)

	// the host is now part of that team
	h, err := s.ds.Host(ctx, mdmHost.ID)
	require.NoError(t, err)
	require.NotNil(t, h.TeamID)
	tm1, err := s.ds.Team(ctx, *h.TeamID)
	require.NoError(t, err)
	require.Equal(t, "g1", tm1.Name)

	// it create activities for the new team, the profiles assigned to it,
	// the host moved to it, and setup assistant
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeCreatedTeam{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q}`, tm1.ID, tm1.Name),
		0)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedMacosProfile{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q}`, tm1.ID, tm1.Name),
		0)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeTransferredHostsToTeam{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q, "host_ids": [%d], "host_display_names": [%q]}`,
			tm1.ID, tm1.Name, h.ID, h.DisplayName()),
		0)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "name": %q, "team_name": %q}`,
			tm1.ID, globalAsstResp.Name, tm1.Name),
		0)

	// and the team has the expected profiles
	profs, err := s.ds.ListMDMAppleConfigProfiles(ctx, &tm1.ID)
	require.NoError(t, err)
	require.Len(t, profs, 2)
	// order is guaranteed by profile name
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	// filevault is enabled by default
	require.True(t, tm1.Config.MDM.EnableDiskEncryption)
	// setup assistant settings are copyied from "no team"
	teamAsst, err := s.ds.GetMDMAppleSetupAssistant(ctx, &tm1.ID)
	require.NoError(t, err)
	require.Equal(t, globalAsstResp.Name, teamAsst.Name)
	require.JSONEq(t, string(globalAsstResp.Profile), string(teamAsst.Profile))

	// create a team and set profiles to it
	tm2, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name: "g1 - g4",
	})
	require.NoError(t, err)
	prof4 := mobileconfigForTest("n4", "i4")
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{
		prof1, prof4,
	}}, http.StatusNoContent, "team_id", fmt.Sprint(tm2.ID))

	// create another team with a superset of profiles
	tm3, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name: "team3_" + t.Name(),
	})
	require.NoError(t, err)
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{
		prof1, prof2, prof4,
	}}, http.StatusNoContent, "team_id", fmt.Sprint(tm3.ID))

	// and yet another team with the same profiles as tm3
	tm4, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name: "team4_" + t.Name(),
	})
	require.NoError(t, err)
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{
		prof1, prof2, prof4,
	}}, http.StatusNoContent, "team_id", fmt.Sprint(tm4.ID))

	// trigger the schedule so profiles are set in their state
	s.awaitTriggerProfileSchedule(t)

	// preassign the MDM host to prof1 and prof4, should match existing team tm2
	//
	// additionally, use external host identifiers with different
	// suffixes to simulate real world distributed scenarios where more
	// than one puppet server might be running at the time.
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{MDMApplePreassignProfilePayload: fleet.MDMApplePreassignProfilePayload{ExternalHostIdentifier: "6f36ab2c-1a40-429b-9c9d-07c9029f4aa8-puppetcompiler06.test.example.com", HostUUID: mdmHost.UUID, Profile: prof1, Group: "g1"}}, http.StatusNoContent)
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{MDMApplePreassignProfilePayload: fleet.MDMApplePreassignProfilePayload{ExternalHostIdentifier: "6f36ab2c-1a40-429b-9c9d-07c9029f4aa8-puppetcompiler01.test.example.com", HostUUID: mdmHost.UUID, Profile: prof4, Group: "g4"}}, http.StatusNoContent)

	// match with the mdm host succeeds and assigns it to tm2
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/match", matchMDMApplePreassignmentRequest{ExternalHostIdentifier: "6f36ab2c-1a40-429b-9c9d-07c9029f4aa8-puppetcompiler03.test.example.com"}, http.StatusNoContent)

	// the host is now part of that team
	h, err = s.ds.Host(ctx, mdmHost.ID)
	require.NoError(t, err)
	require.NotNil(t, h.TeamID)
	require.Equal(t, tm2.ID, *h.TeamID)

	// the host's profiles are:
	// - the same as the team's and are pending
	// - prof2 + old filevault are pending removal
	// - fleetd config being reinstalled (to update the enroll secret)
	s.awaitTriggerProfileSchedule(t)
	hostProfs, err := s.ds.GetHostMDMAppleProfiles(ctx, mdmHost.UUID)
	require.NoError(t, err)
	require.Len(t, hostProfs, 5)

	sort.Slice(hostProfs, func(i, j int) bool {
		l, r := hostProfs[i], hostProfs[j]
		return l.Name < r.Name
	})
	require.Equal(t, "Disk encryption", hostProfs[0].Name)
	require.NotNil(t, hostProfs[0].Status)
	require.Equal(t, fleet.MDMDeliveryPending, *hostProfs[0].Status)
	require.Equal(t, fleet.MDMOperationTypeRemove, hostProfs[0].OperationType)
	require.Equal(t, "Fleetd configuration", hostProfs[1].Name)
	require.NotNil(t, hostProfs[1].Status)
	require.Equal(t, fleet.MDMDeliveryPending, *hostProfs[1].Status)
	require.Equal(t, fleet.MDMOperationTypeInstall, hostProfs[1].OperationType)
	require.Equal(t, "n1", hostProfs[2].Name)
	require.NotNil(t, hostProfs[2].Status)
	require.Equal(t, fleet.MDMDeliveryPending, *hostProfs[2].Status)
	require.Equal(t, fleet.MDMOperationTypeInstall, hostProfs[2].OperationType)
	require.Equal(t, "n2", hostProfs[3].Name)
	require.NotNil(t, hostProfs[3].Status)
	require.Equal(t, fleet.MDMDeliveryPending, *hostProfs[3].Status)
	require.Equal(t, fleet.MDMOperationTypeRemove, hostProfs[3].OperationType)
	require.Equal(t, "n4", hostProfs[4].Name)
	require.NotNil(t, hostProfs[4].Status)
	require.Equal(t, fleet.MDMDeliveryPending, *hostProfs[4].Status)
	require.Equal(t, fleet.MDMOperationTypeInstall, hostProfs[4].OperationType)

	// create a new mdm host enrolled in fleet
	mdmHost2, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	s.runWorker()
	// make it part of team 2
	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &tm2.ID, HostIDs: []uint{mdmHost2.ID}}, http.StatusOK)

	// simulate having its profiles installed
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE host_mdm_apple_profiles SET status = ? WHERE host_uuid = ?`, fleet.OSSettingsVerifying, mdmHost2.UUID)
		return err
	})

	// preassign the MDM host using "g1" and "g4", should match existing
	// team tm2, and nothing be done since the host is already in tm2
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{MDMApplePreassignProfilePayload: fleet.MDMApplePreassignProfilePayload{ExternalHostIdentifier: "mdm2", HostUUID: mdmHost2.UUID, Profile: prof1, Group: "g1"}}, http.StatusNoContent)
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{MDMApplePreassignProfilePayload: fleet.MDMApplePreassignProfilePayload{ExternalHostIdentifier: "mdm2", HostUUID: mdmHost2.UUID, Profile: prof4, Group: "g4"}}, http.StatusNoContent)
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/match", matchMDMApplePreassignmentRequest{ExternalHostIdentifier: "mdm2"}, http.StatusNoContent)

	// the host is still part of tm2
	h, err = s.ds.Host(ctx, mdmHost2.ID)
	require.NoError(t, err)
	require.NotNil(t, h.TeamID)
	require.Equal(t, tm2.ID, *h.TeamID)

	// and its profiles have been left untouched
	s.awaitTriggerProfileSchedule(t)
	hostProfs, err = s.ds.GetHostMDMAppleProfiles(ctx, mdmHost2.UUID)
	require.NoError(t, err)
	require.Len(t, hostProfs, 3)

	sort.Slice(hostProfs, func(i, j int) bool {
		l, r := hostProfs[i], hostProfs[j]
		return l.Name < r.Name
	})
	require.Equal(t, "Fleetd configuration", hostProfs[0].Name)
	require.NotNil(t, hostProfs[0].Status)
	require.Equal(t, fleet.MDMDeliveryVerifying, *hostProfs[0].Status)
	require.Equal(t, "n1", hostProfs[1].Name)
	require.NotNil(t, hostProfs[1].Status)
	require.Equal(t, fleet.MDMDeliveryVerifying, *hostProfs[1].Status)
	require.Equal(t, "n4", hostProfs[2].Name)
	require.NotNil(t, hostProfs[2].Status)
	require.Equal(t, fleet.MDMDeliveryVerifying, *hostProfs[2].Status)
}

// while s.TestPuppetMatchPreassignProfiles focuses on many edge cases/extra
// checks around profile assignment, this test is mainly focused on
// simulating a few puppet runs in scenarios we want to support, and ensuring that:
//
// - different hosts end up in the right teams
// - teams get edited as expected
// - commands to add/remove profiles are issued adequately
func (s *integrationMDMTestSuite) TestPuppetRun() {
	t := s.T()
	ctx := context.Background()

	// define a few profiles
	prof1, prof2, prof3, prof4 := mobileconfigForTest("n1", "i1"),
		mobileconfigForTest("n2", "i2"),
		mobileconfigForTest("n3", "i3"),
		mobileconfigForTest("n4", "i4")

	// create three hosts
	host1, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	host2, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	host3, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	s.runWorker()

	// preassignAndMatch simulates the puppet module doing all the
	// preassign/match calls for a given set of profiles.
	preassignAndMatch := func(profs []fleet.MDMApplePreassignProfilePayload) {
		require.NotEmpty(t, profs)
		for _, prof := range profs {
			s.Do(
				"POST",
				"/api/latest/fleet/mdm/apple/profiles/preassign",
				preassignMDMAppleProfileRequest{MDMApplePreassignProfilePayload: prof},
				http.StatusNoContent,
			)
		}
		s.Do(
			"POST",
			"/api/latest/fleet/mdm/apple/profiles/match",
			matchMDMApplePreassignmentRequest{ExternalHostIdentifier: profs[0].ExternalHostIdentifier},
			http.StatusNoContent,
		)
	}

	// node default {
	//   fleetdm::profile { 'n1':
	//     template => template('n1.mobileconfig.erb'),
	//     group    => 'base',
	//   }
	//
	//   fleetdm::profile { 'n2':
	//     template => template('n2.mobileconfig.erb'),
	//     group    => 'workstations',
	//   }
	//
	//   fleetdm::profile { 'n3':
	//     template => template('n3.mobileconfig.erb'),
	//     group    => 'workstations',
	//   }
	//
	//   if $facts['system_profiler']['hardware_uuid'] == 'host_2_uuid' {
	//       fleetdm::profile { 'n4':
	//         template => template('fleetdm/n4.mobileconfig.erb'),
	//         group    => 'kiosks',
	//       }
	//   }
	puppetRun := func(host *fleet.Host) {
		payload := []fleet.MDMApplePreassignProfilePayload{
			{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof1,
				Group:                  "base",
			},
			{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof2,
				Group:                  "workstations",
			},
			{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof3,
				Group:                  "workstations",
			},
		}

		if host.UUID == host2.UUID {
			payload = append(payload, fleet.MDMApplePreassignProfilePayload{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof4,
				Group:                  "kiosks",
			})
		}

		preassignAndMatch(payload)
	}

	// host1 checks in
	puppetRun(host1)

	// the host now belongs to a team
	h1, err := s.ds.Host(ctx, host1.ID)
	require.NoError(t, err)
	require.NotNil(t, h1.TeamID)

	// the team has the right name
	tm1, err := s.ds.Team(ctx, *h1.TeamID)
	require.NoError(t, err)
	require.Equal(t, "base - workstations", tm1.Name)
	// and the right profiles
	profs, err := s.ds.ListMDMAppleConfigProfiles(ctx, &tm1.ID)
	require.NoError(t, err)
	require.Len(t, profs, 3)
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	require.Equal(t, prof3, []byte(profs[2].Mobileconfig))
	require.True(t, tm1.Config.MDM.EnableDiskEncryption)

	// host2 checks in
	puppetRun(host2)
	// a new team is created
	h2, err := s.ds.Host(ctx, host2.ID)
	require.NoError(t, err)
	require.NotNil(t, h2.TeamID)

	// the team has the right name
	tm2, err := s.ds.Team(ctx, *h2.TeamID)
	require.NoError(t, err)
	require.Equal(t, "base - kiosks - workstations", tm2.Name)
	// and the right profiles
	profs, err = s.ds.ListMDMAppleConfigProfiles(ctx, &tm2.ID)
	require.NoError(t, err)
	require.Len(t, profs, 4)
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	require.Equal(t, prof3, []byte(profs[2].Mobileconfig))
	require.Equal(t, prof4, []byte(profs[3].Mobileconfig))
	require.True(t, tm2.Config.MDM.EnableDiskEncryption)

	// host3 checks in
	puppetRun(host3)
	// it belongs to the same team as host1
	h3, err := s.ds.Host(ctx, host3.ID)
	require.NoError(t, err)
	require.Equal(t, h1.TeamID, h3.TeamID)

	// prof2 is edited
	oldProf2 := prof2
	prof2 = mobileconfigForTest("n2", "i2-v2")
	// host3 checks in again
	puppetRun(host3)
	// still belongs to the same team
	h3, err = s.ds.Host(ctx, host3.ID)
	require.NoError(t, err)
	require.Equal(t, tm1.ID, *h3.TeamID)

	// but the team has prof2 updated
	profs, err = s.ds.ListMDMAppleConfigProfiles(ctx, &tm1.ID)
	require.NoError(t, err)
	require.Len(t, profs, 3)
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	require.Equal(t, prof3, []byte(profs[2].Mobileconfig))
	require.NotEqual(t, oldProf2, []byte(profs[1].Mobileconfig))
	require.True(t, tm1.Config.MDM.EnableDiskEncryption)

	// host2 checks in, still belongs to the same team
	puppetRun(host2)
	h2, err = s.ds.Host(ctx, host2.ID)
	require.NoError(t, err)
	require.Equal(t, tm2.ID, *h2.TeamID)

	// but the team has prof2 updated as well
	profs, err = s.ds.ListMDMAppleConfigProfiles(ctx, &tm2.ID)
	require.NoError(t, err)
	require.Len(t, profs, 4)
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	require.Equal(t, prof3, []byte(profs[2].Mobileconfig))
	require.Equal(t, prof4, []byte(profs[3].Mobileconfig))
	require.NotEqual(t, oldProf2, []byte(profs[1].Mobileconfig))
	require.True(t, tm1.Config.MDM.EnableDiskEncryption)

	// the puppet manifest is changed, and prof3 is removed
	// node default {
	//   fleetdm::profile { 'n1':
	//     template => template('n1.mobileconfig.erb'),
	//     group    => 'base',
	//   }
	//
	//   fleetdm::profile { 'n2':
	//     template => template('n2.mobileconfig.erb'),
	//     group    => 'workstations',
	//   }
	//
	//   if $facts['system_profiler']['hardware_uuid'] == 'host_2_uuid' {
	//       fleetdm::profile { 'n4':
	//         template => template('fleetdm/n4.mobileconfig.erb'),
	//         group    => 'kiosks',
	//       }
	//   }
	puppetRun = func(host *fleet.Host) {
		payload := []fleet.MDMApplePreassignProfilePayload{
			{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof1,
				Group:                  "base",
			},
			{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof2,
				Group:                  "workstations",
			},
		}

		if host.UUID == host2.UUID {
			payload = append(payload, fleet.MDMApplePreassignProfilePayload{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof4,
				Group:                  "kiosks",
			})
		}

		preassignAndMatch(payload)
	}

	// host1 checks in again
	puppetRun(host1)
	// still belongs to the same team
	h1, err = s.ds.Host(ctx, host1.ID)
	require.NoError(t, err)
	require.Equal(t, tm1.ID, *h1.TeamID)

	// but the team doesn't have prof3 anymore
	profs, err = s.ds.ListMDMAppleConfigProfiles(ctx, &tm1.ID)
	require.NoError(t, err)
	require.Len(t, profs, 2)
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	require.True(t, tm1.Config.MDM.EnableDiskEncryption)

	// same for host2
	puppetRun(host2)
	h2, err = s.ds.Host(ctx, host2.ID)
	require.NoError(t, err)
	require.Equal(t, tm2.ID, *h2.TeamID)
	profs, err = s.ds.ListMDMAppleConfigProfiles(ctx, &tm2.ID)
	require.NoError(t, err)
	require.Len(t, profs, 3)
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	require.Equal(t, prof4, []byte(profs[2].Mobileconfig))
	require.True(t, tm1.Config.MDM.EnableDiskEncryption)

	// The puppet manifest is drastically updated, this time to use exclusions on host3:
	//
	// node default {
	//   fleetdm::profile { 'n1':
	//     template => template('n1.mobileconfig.erb'),
	//     group    => 'base',
	//   }
	//
	//   fleetdm::profile { 'n2':
	//     template => template('n2.mobileconfig.erb'),
	//     group    => 'workstations',
	//   }
	//
	//   if $facts['system_profiler']['hardware_uuid'] == 'host_3_uuid' {
	//       fleetdm::profile { 'n3':
	//         template => template('fleetdm/n3.mobileconfig.erb'),
	//         group    => 'no-nudge',
	//       }
	//   } else {
	//       fleetdm::profile { 'n3':
	//         ensure => absent,
	//         template => template('fleetdm/n3.mobileconfig.erb'),
	//         group    => 'workstations',
	//       }
	//   }
	// }
	puppetRun = func(host *fleet.Host) {
		manifest := []fleet.MDMApplePreassignProfilePayload{
			{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof1,
				Group:                  "base",
			},
			{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof2,
				Group:                  "workstations",
			},
		}

		if host.UUID == host3.UUID {
			manifest = append(manifest, fleet.MDMApplePreassignProfilePayload{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof3,
				Group:                  "no-nudge",
				Exclude:                true,
			})
		} else {
			manifest = append(manifest, fleet.MDMApplePreassignProfilePayload{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof3,
				Group:                  "workstations",
			})
		}

		preassignAndMatch(manifest)
	}

	// host1 checks in
	puppetRun(host1)

	// the host belongs to the same team
	h1, err = s.ds.Host(ctx, host1.ID)
	require.NoError(t, err)
	require.Equal(t, tm1.ID, *h1.TeamID)

	// the team has the right profiles
	profs, err = s.ds.ListMDMAppleConfigProfiles(ctx, &tm1.ID)
	require.NoError(t, err)
	require.Len(t, profs, 3)
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	require.Equal(t, prof3, []byte(profs[2].Mobileconfig))
	require.True(t, tm1.Config.MDM.EnableDiskEncryption)

	// host2 checks in
	puppetRun(host2)
	// it is assigned to tm1
	h2, err = s.ds.Host(ctx, host2.ID)
	require.NoError(t, err)
	require.Equal(t, tm1.ID, *h2.TeamID)

	// host3 checks in
	puppetRun(host3)

	// it is assigned to a new team
	h3, err = s.ds.Host(ctx, host3.ID)
	require.NoError(t, err)
	require.NotNil(t, h3.TeamID)
	require.NotEqual(t, tm1.ID, *h3.TeamID)
	require.NotEqual(t, tm2.ID, *h3.TeamID)

	// a new team is created
	tm3, err := s.ds.Team(ctx, *h3.TeamID)
	require.NoError(t, err)
	require.Equal(t, "base - no-nudge - workstations", tm3.Name)
	// and the right profiles
	profs, err = s.ds.ListMDMAppleConfigProfiles(ctx, &tm3.ID)
	require.NoError(t, err)
	require.Len(t, profs, 2)
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	require.True(t, tm3.Config.MDM.EnableDiskEncryption)
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

func createWindowsHostThenEnrollMDM(ds fleet.Datastore, fleetServerURL string, t *testing.T) (*fleet.Host, *mdmtest.TestWindowsMDMClient) {
	host := createOrbitEnrolledHost(t, "windows", "h1", ds)
	mdmDevice := mdmtest.NewTestMDMClientWindowsProgramatic(fleetServerURL, *host.OrbitNodeKey)
	err := mdmDevice.Enroll()
	require.NoError(t, err)
	err = ds.UpdateMDMWindowsEnrollmentsHostUUID(context.Background(), host.UUID, mdmDevice.DeviceID)
	require.NoError(t, err)
	return host, mdmDevice
}

func (s *integrationMDMTestSuite) TestDEPProfileAssignment() {
	t := s.T()

	ctx := context.Background()
	devices := []godep.Device{
		{SerialNumber: uuid.New().String(), Model: "MacBook Pro", OS: "osx", OpType: "added"},
		{SerialNumber: uuid.New().String(), Model: "MacBook Mini", OS: "osx", OpType: "added"},
		{SerialNumber: uuid.New().String(), Model: "MacBook Mini", OS: "osx", OpType: ""},
		{SerialNumber: uuid.New().String(), Model: "MacBook Mini", OS: "osx", OpType: "modified"},
	}

	type profileAssignmentReq struct {
		ProfileUUID string   `json:"profile_uuid"`
		Devices     []string `json:"devices"`
	}
	profileAssignmentReqs := []profileAssignmentReq{}

	// add global profiles
	globalProfile := mobileconfigForTest("N1", "I1")
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{globalProfile}}, http.StatusNoContent)

	checkPostEnrollmentCommands := func(mdmDevice *mdmtest.TestAppleMDMClient, shouldReceive bool) {
		// run the worker to process the DEP enroll request
		s.runWorker()
		// run the worker to assign configuration profiles
		s.awaitTriggerProfileSchedule(t)

		var fleetdCmd, installProfileCmd *micromdm.CommandPayload
		cmd, err := mdmDevice.Idle()
		require.NoError(t, err)
		for cmd != nil {
			if cmd.Command.RequestType == "InstallEnterpriseApplication" &&
				cmd.Command.InstallEnterpriseApplication.ManifestURL != nil &&
				strings.Contains(*cmd.Command.InstallEnterpriseApplication.ManifestURL, apple_mdm.FleetdPublicManifestURL) {
				fleetdCmd = cmd
			} else if cmd.Command.RequestType == "InstallProfile" {
				installProfileCmd = cmd
			}
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}

		if shouldReceive {
			// received request to install fleetd
			require.NotNil(t, fleetdCmd, "host didn't get a command to install fleetd")
			require.NotNil(t, fleetdCmd.Command, "host didn't get a command to install fleetd")

			// received request to install the global configuration profile
			require.NotNil(t, installProfileCmd, "host didn't get a command to install profiles")
			require.NotNil(t, installProfileCmd.Command, "host didn't get a command to install profiles")
		} else {
			require.Nil(t, fleetdCmd, "host got a command to install fleetd")
			require.Nil(t, installProfileCmd, "host got a command to install profiles")
		}
	}

	s.mockDEPResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			err := encoder.Encode(map[string]string{"auth_session_token": "xyz"})
			require.NoError(t, err)
		case "/profile":
			err := encoder.Encode(godep.ProfileResponse{ProfileUUID: uuid.New().String()})
			require.NoError(t, err)
		case "/server/devices":
			// This endpoint  is used to get an initial list of
			// devices, return a single device
			err := encoder.Encode(godep.DeviceResponse{Devices: devices[:1]})
			require.NoError(t, err)
		case "/devices/sync":
			// This endpoint is polled over time to sync devices from
			// ABM, send a repeated serial and a new one
			err := encoder.Encode(godep.DeviceResponse{Devices: devices, Cursor: "foo"})
			require.NoError(t, err)
		case "/profile/devices":
			b, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var prof profileAssignmentReq
			require.NoError(t, json.Unmarshal(b, &prof))
			profileAssignmentReqs = append(profileAssignmentReqs, prof)
			_, _ = w.Write([]byte(`{}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))

	// query all hosts
	listHostsRes := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Empty(t, listHostsRes.Hosts)

	// trigger a profile sync
	s.runDEPSchedule()

	// all hosts should be returned from the hosts endpoint
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, len(devices))
	var wantSerials []string
	var gotSerials []string
	for i, device := range devices {
		wantSerials = append(wantSerials, device.SerialNumber)
		gotSerials = append(gotSerials, listHostsRes.Hosts[i].HardwareSerial)
		// entries for all hosts should be created in the host_dep_assignments table
		_, err := s.ds.GetHostDEPAssignment(ctx, listHostsRes.Hosts[i].ID)
		require.NoError(t, err)
	}
	require.ElementsMatch(t, wantSerials, gotSerials)
	// called two times:
	// - one when we get the initial list of devices (/server/devices)
	// - one when we do the device sync (/device/sync)
	require.Len(t, profileAssignmentReqs, 2)
	require.Len(t, profileAssignmentReqs[0].Devices, 1)
	require.Len(t, profileAssignmentReqs[1].Devices, len(devices))

	// create a new host
	nonDEPHost := createHostAndDeviceToken(t, s.ds, "not-dep")
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, len(devices)+1)

	// filtering by MDM status works
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts?mdm_enrollment_status=pending", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, len(devices))

	// searching by display name works
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts?query=%s", url.QueryEscape("MacBook Mini")), nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 3)
	for _, host := range listHostsRes.Hosts {
		require.Equal(t, "MacBook Mini", host.HardwareModel)
		require.Equal(t, host.DisplayName, fmt.Sprintf("MacBook Mini (%s)", host.HardwareSerial))
	}

	s.pushProvider.PushFunc = func(pushes []*mdm.Push) (map[string]*push.Response, error) {
		return map[string]*push.Response{}, nil
	}

	// Enroll one of the hosts
	depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
	mdmDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
	mdmDevice.SerialNumber = devices[0].SerialNumber
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	// make sure the host gets post enrollment requests
	checkPostEnrollmentCommands(mdmDevice, true)

	// only one shows up as pending
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts?mdm_enrollment_status=pending", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, len(devices)-1)

	activities := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activities, "order_key", "created_at")
	found := false
	for _, activity := range activities.Activities {
		if activity.Type == "mdm_enrolled" &&
			strings.Contains(string(*activity.Details), devices[0].SerialNumber) {
			found = true
			require.Nil(t, activity.ActorID)
			require.Nil(t, activity.ActorFullName)
			require.JSONEq(
				t,
				fmt.Sprintf(
					`{"host_serial": "%s", "host_display_name": "%s (%s)", "installed_from_dep": true, "mdm_platform": "apple"}`,
					devices[0].SerialNumber, devices[0].Model, devices[0].SerialNumber,
				),
				string(*activity.Details),
			)
		}
	}
	require.True(t, found)

	// add devices[1].SerialNumber to a team
	teamName := t.Name() + "team1"
	team := &fleet.Team{
		Name:        teamName,
		Description: "desc team1",
	}
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &createTeamResp)
	require.NotZero(t, createTeamResp.Team.ID)
	team = createTeamResp.Team
	for _, h := range listHostsRes.Hosts {
		if h.HardwareSerial == devices[1].SerialNumber {
			err = s.ds.AddHostsToTeam(ctx, &team.ID, []uint{h.ID})
			require.NoError(t, err)
		}
	}

	// modify the response and trigger another sync to include:
	//
	// 1. A repeated device with "added"
	// 2. A repeated device with "modified"
	// 3. A device with "deleted"
	// 4. A new device
	deletedSerial := devices[2].SerialNumber
	addedSerial := uuid.New().String()
	devices = []godep.Device{
		{SerialNumber: devices[0].SerialNumber, Model: "MacBook Pro", OS: "osx", OpType: "added"},
		{SerialNumber: devices[1].SerialNumber, Model: "MacBook Mini", OS: "osx", OpType: "modified"},
		{SerialNumber: deletedSerial, Model: "MacBook Mini", OS: "osx", OpType: "deleted"},
		{SerialNumber: addedSerial, Model: "MacBook Mini", OS: "osx", OpType: "added"},
	}
	profileAssignmentReqs = []profileAssignmentReq{}
	s.runDEPSchedule()

	// all hosts should be returned from the hosts endpoint
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	// all previous devices + the manually added host + the new `addedSerial`
	wantSerials = append(wantSerials, devices[3].SerialNumber, nonDEPHost.HardwareSerial)
	require.Len(t, listHostsRes.Hosts, len(wantSerials))
	gotSerials = []string{}
	var deletedHostID uint
	var addedHostID uint
	var mdmDeviceID uint
	for _, device := range listHostsRes.Hosts {
		gotSerials = append(gotSerials, device.HardwareSerial)
		switch device.HardwareSerial {
		case deletedSerial:
			deletedHostID = device.ID
		case addedSerial:
			addedHostID = device.ID
		case mdmDevice.SerialNumber:
			mdmDeviceID = device.ID
		}
	}
	require.ElementsMatch(t, wantSerials, gotSerials)
	require.Len(t, profileAssignmentReqs, 3)

	// first request to get a list of profiles
	// TODO: seems like we're doing this request on each loop?
	require.Len(t, profileAssignmentReqs[0].Devices, 1)
	require.Equal(t, devices[0].SerialNumber, profileAssignmentReqs[0].Devices[0])

	// profileAssignmentReqs[1] and [2] can be in any order
	ix2Devices, ix1Device := 1, 2
	if len(profileAssignmentReqs[1].Devices) == 1 {
		ix2Devices, ix1Device = ix1Device, ix2Devices
	}

	// - existing device with "added"
	// - new device with "added"
	require.Len(t, profileAssignmentReqs[ix2Devices].Devices, 2, "%#+v", profileAssignmentReqs)
	require.Equal(t, devices[0].SerialNumber, profileAssignmentReqs[ix2Devices].Devices[0])
	require.Equal(t, addedSerial, profileAssignmentReqs[ix2Devices].Devices[1])

	// - existing device with "modified" and a different team (thus different profile request)
	require.Len(t, profileAssignmentReqs[ix1Device].Devices, 1)
	require.Equal(t, devices[1].SerialNumber, profileAssignmentReqs[ix1Device].Devices[0])

	// entries for all hosts except for the one with OpType = "deleted"
	assignment, err := s.ds.GetHostDEPAssignment(ctx, deletedHostID)
	require.NoError(t, err)
	require.NotZero(t, assignment.DeletedAt)

	_, err = s.ds.GetHostDEPAssignment(ctx, addedHostID)
	require.NoError(t, err)

	// send a TokenUpdate command, it shouldn't re-send the post-enrollment commands
	err = mdmDevice.TokenUpdate()
	require.NoError(t, err)
	checkPostEnrollmentCommands(mdmDevice, false)

	// enroll the device again, it should get the post-enrollment commands
	err = mdmDevice.Enroll()
	require.NoError(t, err)
	checkPostEnrollmentCommands(mdmDevice, true)

	// delete the device from Fleet
	var delResp deleteHostResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d", mdmDeviceID), nil, http.StatusOK, &delResp)

	// the device comes back as pending
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts?query=%s", mdmDevice.UUID), nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 1)
	require.Equal(t, mdmDevice.SerialNumber, listHostsRes.Hosts[0].HardwareSerial)

	// we assign a DEP profile to the device
	profileAssignmentReqs = []profileAssignmentReq{}
	s.runWorker()
	require.Equal(t, mdmDevice.SerialNumber, profileAssignmentReqs[0].Devices[0])

	// it should get the post-enrollment commands
	require.NoError(t, mdmDevice.Enroll())
	checkPostEnrollmentCommands(mdmDevice, true)

	// delete all MDM info
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM host_mdm WHERE host_id = ?`, listHostsRes.Hosts[0].ID)
		return err
	})

	// it should still get the post-enrollment commands
	require.NoError(t, mdmDevice.Enroll())
	checkPostEnrollmentCommands(mdmDevice, true)

	// The user unenrolls from Fleet (e.g. was DEP enrolled but with `is_mdm_removable: true`
	// so the user removes the enrollment profile).
	err = mdmDevice.Checkout()
	require.NoError(t, err)

	// Simulate a refetch where we clean up the MDM data since the host is not enrolled anymore
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM host_mdm WHERE host_id = ?`, mdmDeviceID)
		return err
	})

	// Simulate fleetd re-enrolling automatically.
	err = mdmDevice.Enroll()
	require.NoError(t, err)

	// The last activity should have `installed_from_dep=true`.
	s.lastActivityMatches(
		"mdm_enrolled",
		fmt.Sprintf(
			`{"host_serial": "%s", "host_display_name": "%s (%s)", "installed_from_dep": true, "mdm_platform": "apple"}`,
			mdmDevice.SerialNumber, mdmDevice.Model, mdmDevice.SerialNumber,
		),
		0,
	)

	// enroll a host into Fleet
	eHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		ID:             1,
		OsqueryHostID:  ptr.String("Desktop-ABCQWE"),
		NodeKey:        ptr.String("Desktop-ABCQWE"),
		UUID:           uuid.New().String(),
		Hostname:       fmt.Sprintf("%sfoo.local", s.T().Name()),
		Platform:       "darwin",
		HardwareSerial: uuid.New().String(),
	})
	require.NoError(t, err)

	// on team transfer, we don't assign a DEP profile to the device
	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &team.ID, HostIDs: []uint{eHost.ID}}, http.StatusOK)
	profileAssignmentReqs = []profileAssignmentReq{}
	s.runWorker()
	require.Empty(t, profileAssignmentReqs)

	// assign the host in ABM
	devices = []godep.Device{
		{SerialNumber: eHost.HardwareSerial, Model: "MacBook Pro", OS: "osx", OpType: "modified"},
	}
	profileAssignmentReqs = []profileAssignmentReq{}
	s.runDEPSchedule()
	require.NotEmpty(t, profileAssignmentReqs)
	require.Equal(t, eHost.HardwareSerial, profileAssignmentReqs[0].Devices[0])

	// transfer to "no team", we assign a DEP profile to the device
	profileAssignmentReqs = []profileAssignmentReq{}
	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: nil, HostIDs: []uint{eHost.ID}}, http.StatusOK)
	s.runWorker()
	require.NotEmpty(t, profileAssignmentReqs)
	require.Equal(t, eHost.HardwareSerial, profileAssignmentReqs[0].Devices[0])

	// transfer to the team back again, we assign a DEP profile to the device again
	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &team.ID, HostIDs: []uint{eHost.ID}}, http.StatusOK)
	profileAssignmentReqs = []profileAssignmentReq{}
	s.runWorker()
	require.NotEmpty(t, profileAssignmentReqs)
	require.Equal(t, eHost.HardwareSerial, profileAssignmentReqs[0].Devices[0])
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
	s.downloadAndVerifyEnrollmentProfile("/api/latest/fleet/device/" + token + "/mdm/apple/manual_enrollment_profile")
}

func (s *integrationMDMTestSuite) TestAppleMDMDeviceEnrollment() {
	t := s.T()

	// Enroll two devices into MDM
	mdmEnrollInfo := mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.fleetCfg.MDM.AppleSCEPChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}
	mdmDeviceA := mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo)
	err := mdmDeviceA.Enroll()
	require.NoError(t, err)
	mdmDeviceB := mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo)
	err = mdmDeviceB.Enroll()
	require.NoError(t, err)

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

	// Activities are generated for each device
	activities := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activities, "order_key", "created_at")
	require.GreaterOrEqual(t, len(activities.Activities), 2)

	details := []*json.RawMessage{}
	for _, activity := range activities.Activities {
		if activity.Type == "mdm_enrolled" {
			require.Nil(t, activity.ActorID)
			require.Nil(t, activity.ActorFullName)
			details = append(details, activity.Details)
		}
	}
	require.Len(t, details, 2)
	require.JSONEq(t, fmt.Sprintf(`{"host_serial": "%s", "host_display_name": "%s (%s)", "installed_from_dep": false, "mdm_platform": "apple"}`, mdmDeviceA.SerialNumber, mdmDeviceA.Model, mdmDeviceA.SerialNumber), string(*details[len(details)-2]))
	require.JSONEq(t, fmt.Sprintf(`{"host_serial": "%s", "host_display_name": "%s (%s)", "installed_from_dep": false, "mdm_platform": "apple"}`, mdmDeviceB.SerialNumber, mdmDeviceB.Model, mdmDeviceB.SerialNumber), string(*details[len(details)-1]))

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
	activities = listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activities)

	found := false
	for _, activity := range activities.Activities {
		if activity.Type == "mdm_unenrolled" {
			found = true
			require.Nil(t, activity.ActorID)
			require.Nil(t, activity.ActorFullName)
			details = append(details, activity.Details)
			require.JSONEq(t, fmt.Sprintf(`{"host_serial": "%s", "host_display_name": "%s (%s)", "installed_from_dep": false}`, mdmDeviceA.SerialNumber, mdmDeviceA.Model, mdmDeviceA.SerialNumber), string(*activity.Details))
		}
	}
	require.True(t, found)
}

func (s *integrationMDMTestSuite) TestDeviceMultipleAuthMessages() {
	t := s.T()

	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.fleetCfg.MDM.AppleSCEPChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	})
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

func (s *integrationMDMTestSuite) TestMDMAppleUnenroll() {
	t := s.T()

	// Enroll a device into MDM.
	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.fleetCfg.MDM.AppleSCEPChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	})
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
	// 3 profiles added + 1 profile with fleetd configuration
	require.Len(t, *hostResp.Host.MDM.Profiles, 4)

	// try to unenroll the host, fails since the host doesn't respond
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/mdm/hosts/%d/unenroll", h.ID), nil, http.StatusGatewayTimeout)

	// we're going to modify this mock, make sure we restore its default
	originalPushMock := s.pushProvider.PushFunc
	defer func() { s.pushProvider.PushFunc = originalPushMock }()

	// if there's an error coming from APNs servers
	s.pushProvider.PushFunc = func(pushes []*mdm.Push) (map[string]*push.Response, error) {
		return map[string]*push.Response{
			pushes[0].Token.String(): {
				Id:  uuid.New().String(),
				Err: errors.New("test"),
			},
		}, nil
	}
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/mdm/hosts/%d/unenroll", h.ID), nil, http.StatusBadGateway)

	// if there was an error unrelated to APNs
	s.pushProvider.PushFunc = func(pushes []*mdm.Push) (map[string]*push.Response, error) {
		res := map[string]*push.Response{
			pushes[0].Token.String(): {
				Id:  uuid.New().String(),
				Err: nil,
			},
		}
		return res, errors.New("baz")
	}
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/mdm/hosts/%d/unenroll", h.ID), nil, http.StatusInternalServerError)

	// try again, but this time the host is online and answers
	var checkoutErr error
	s.pushProvider.PushFunc = func(pushes []*mdm.Push) (map[string]*push.Response, error) {
		res, err := mockSuccessfulPush(pushes)
		checkoutErr = mdmDevice.Checkout()
		return res, err
	}
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/mdm/hosts/%d/unenroll", h.ID), nil, http.StatusOK)

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

	checkConfigSetErrors := func() {
		// try to set app config
		res := s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusUnprocessableEntity)
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, "Couldn't edit enable_disk_encryption. Neither macOS MDM nor Windows is turned on. Visit https://fleetdm.com/docs/using-fleet to learn how to turn on MDM.")

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
		res = s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusUnprocessableEntity)
		errMsg = extractServerErrorText(res.Body)
		require.Contains(t, errMsg, "Couldn't edit enable_disk_encryption. Neither macOS MDM nor Windows is turned on. Visit https://fleetdm.com/docs/using-fleet to learn how to turn on MDM.")

		// try to edit the existing team using specs
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
		res = s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusUnprocessableEntity)
		errMsg = extractServerErrorText(res.Body)
		require.Contains(t, errMsg, "Couldn't edit enable_disk_encryption. Neither macOS MDM nor Windows is turned on. Visit https://fleetdm.com/docs/using-fleet to learn how to turn on MDM.")
	}

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

	// disable both windows and mac mdm
	// we should get an error
	setMDMEnabled(false, false)
	checkConfigSetErrors()

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
	require.Equal(t, fleet.ActionRequiredLogOut, *getHostResp.Host.MDM.MacOSSettings.ActionRequired)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, fleet.DiskEncryptionActionRequired, *getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, "", getHostResp.Host.MDM.OSSettings.DiskEncryption.Detail)

	// add an encryption key for the host
	cert, _, _, err := s.fleetCfg.MDM.AppleSCEP()
	require.NoError(t, err)
	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	require.NoError(t, err)
	recoveryKey := "AAA-BBB-CCC"
	encryptedKey, err := pkcs7.Encrypt([]byte(recoveryKey), []*x509.Certificate{parsed})
	require.NoError(t, err)
	base64EncryptedKey := base64.StdEncoding.EncodeToString(encryptedKey)

	err = s.ds.SetOrUpdateHostDiskEncryptionKey(ctx, host.ID, base64EncryptedKey, "", nil)
	require.NoError(t, err)

	// get that host - it has an encryption key with unknown decryptability, so
	// it should report "enforcing" disk encryption.
	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Equal(t, fleet.DiskEncryptionEnforcing, *getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Nil(t, getHostResp.Host.MDM.MacOSSettings.ActionRequired)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings)
	require.NotNil(t, getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, fleet.DiskEncryptionEnforcing, *getHostResp.Host.MDM.OSSettings.DiskEncryption.Status)
	require.Equal(t, "", getHostResp.Host.MDM.OSSettings.DiskEncryption.Detail)

	// request with no token
	res := s.DoRawNoAuth("GET", fmt.Sprintf("/api/latest/fleet/mdm/hosts/%d/encryption_key", host.ID), nil, http.StatusUnauthorized)
	res.Body.Close()

	// encryption key not processed yet
	resp := getHostEncryptionKeyResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/mdm/hosts/%d/encryption_key", host.ID), nil, http.StatusNotFound, &resp)

	// unable to decrypt encryption key
	err = s.ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{host.ID}, false, time.Now())
	require.NoError(t, err)
	resp = getHostEncryptionKeyResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/mdm/hosts/%d/encryption_key", host.ID), nil, http.StatusNotFound, &resp)

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
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/mdm/hosts/%d/encryption_key", host.ID), nil, http.StatusOK, &resp)
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
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/mdm/hosts/%d/encryption_key", host.ID), nil, http.StatusForbidden, &resp)
}

func (s *integrationMDMTestSuite) TestWindowsMDMGetEncryptionKey() {
	t := s.T()
	ctx := context.Background()

	// create a host and enroll it in Fleet
	host := createOrbitEnrolledHost(t, "windows", "h1", s.ds)
	err := s.ds.SetOrUpdateMDMData(ctx, host.ID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, "")
	require.NoError(t, err)

	// request encryption key with no auth token
	res := s.DoRawNoAuth("GET", fmt.Sprintf("/api/latest/fleet/mdm/hosts/%d/encryption_key", host.ID), nil, http.StatusUnauthorized)
	res.Body.Close()

	// no encryption key
	resp := getHostEncryptionKeyResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/mdm/hosts/%d/encryption_key", host.ID), nil, http.StatusNotFound, &resp)

	// invalid host id
	resp = getHostEncryptionKeyResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/mdm/hosts/%d/encryption_key", host.ID+999), nil, http.StatusNotFound, &resp)

	// add an encryption key for the host
	cert, _, _, err := s.fleetCfg.MDM.MicrosoftWSTEP()
	require.NoError(t, err)
	recoveryKey := "AAA-BBB-CCC"
	encryptedKey, err := microsoft_mdm.Encrypt(recoveryKey, cert.Leaf)
	require.NoError(t, err)

	err = s.ds.SetOrUpdateHostDiskEncryptionKey(ctx, host.ID, encryptedKey, "", ptr.Bool(true))
	require.NoError(t, err)

	resp = getHostEncryptionKeyResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/mdm/hosts/%d/encryption_key", host.ID), nil, http.StatusOK, &resp)
	require.Equal(t, host.ID, resp.HostID)
	require.Equal(t, recoveryKey, resp.EncryptionKey.DecryptedValue)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeReadHostDiskEncryptionKey{}.ActivityName(),
		fmt.Sprintf(`{"host_display_name": "%s", "host_id": %d}`, host.DisplayName(), host.ID), 0)

	// update the key to blank with a client error
	err = s.ds.SetOrUpdateHostDiskEncryptionKey(ctx, host.ID, "", "failed", nil)
	require.NoError(t, err)

	resp = getHostEncryptionKeyResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/mdm/hosts/%d/encryption_key", host.ID), nil, http.StatusNotFound, &resp)
}

func (s *integrationMDMTestSuite) TestMDMAppleListConfigProfiles() {
	t := s.T()
	ctx := context.Background()

	testTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "TestTeam"})
	require.NoError(t, err)

	mdmHost, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	s.runWorker()

	t.Run("no profiles", func(t *testing.T) {
		var listResp listMDMAppleConfigProfilesResponse
		s.DoJSON("GET", "/api/v1/fleet/mdm/apple/profiles", nil, http.StatusOK, &listResp)
		require.NotNil(t, listResp.ConfigProfiles) // expect empty slice instead of nil
		require.Len(t, listResp.ConfigProfiles, 0)

		listResp = listMDMAppleConfigProfilesResponse{}
		s.DoJSON("GET", fmt.Sprintf(`/api/v1/fleet/mdm/apple/profiles?team_id=%d`, testTeam.ID), nil, http.StatusOK, &listResp)
		require.NotNil(t, listResp.ConfigProfiles) // expect empty slice instead of nil
		require.Len(t, listResp.ConfigProfiles, 0)

		var hostProfilesResp getHostProfilesResponse
		s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/mdm/hosts/%d/profiles", mdmHost.ID), nil, http.StatusOK, &hostProfilesResp)
		require.NotNil(t, hostProfilesResp.Profiles) // expect empty slice instead of nil
		require.Len(t, hostProfilesResp.Profiles, 0)
		require.EqualValues(t, mdmHost.ID, hostProfilesResp.HostID)
	})

	t.Run("with profiles", func(t *testing.T) {
		p1, err := fleet.NewMDMAppleConfigProfile(mcBytesForTest("p1", "p1.identifier", "p1.uuid"), nil)
		require.NoError(t, err)
		_, err = s.ds.NewMDMAppleConfigProfile(ctx, *p1)
		require.NoError(t, err)

		p2, err := fleet.NewMDMAppleConfigProfile(mcBytesForTest("p2", "p2.identifier", "p2.uuid"), &testTeam.ID)
		require.NoError(t, err)
		_, err = s.ds.NewMDMAppleConfigProfile(ctx, *p2)
		require.NoError(t, err)

		var resp listMDMAppleConfigProfilesResponse
		s.DoJSON("GET", "/api/latest/fleet/mdm/apple/profiles", listMDMAppleConfigProfilesRequest{TeamID: 0}, http.StatusOK, &resp)
		require.NotNil(t, resp.ConfigProfiles)
		require.Len(t, resp.ConfigProfiles, 1)
		require.Equal(t, p1.Name, resp.ConfigProfiles[0].Name)
		require.Equal(t, p1.Identifier, resp.ConfigProfiles[0].Identifier)

		resp = listMDMAppleConfigProfilesResponse{}
		s.DoJSON("GET", fmt.Sprintf(`/api/v1/fleet/mdm/apple/profiles?team_id=%d`, testTeam.ID), nil, http.StatusOK, &resp)
		require.NotNil(t, resp.ConfigProfiles)
		require.Len(t, resp.ConfigProfiles, 1)
		require.Equal(t, p2.Name, resp.ConfigProfiles[0].Name)
		require.Equal(t, p2.Identifier, resp.ConfigProfiles[0].Identifier)

		p3, err := fleet.NewMDMAppleConfigProfile(mcBytesForTest("p3", "p3.identifier", "p3.uuid"), &testTeam.ID)
		require.NoError(t, err)
		_, err = s.ds.NewMDMAppleConfigProfile(ctx, *p3)
		require.NoError(t, err)

		resp = listMDMAppleConfigProfilesResponse{}
		s.DoJSON("GET", fmt.Sprintf(`/api/v1/fleet/mdm/apple/profiles?team_id=%d`, testTeam.ID), nil, http.StatusOK, &resp)
		require.NotNil(t, resp.ConfigProfiles)
		require.Len(t, resp.ConfigProfiles, 2)
		for _, p := range resp.ConfigProfiles {
			if p.Name == p2.Name {
				require.Equal(t, p2.Identifier, p.Identifier)
			} else if p.Name == p3.Name {
				require.Equal(t, p3.Identifier, p.Identifier)
			} else {
				require.Fail(t, "unexpected profile name")
			}
		}

		var hostProfilesResp getHostProfilesResponse
		s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/mdm/hosts/%d/profiles", mdmHost.ID), nil, http.StatusOK, &hostProfilesResp)
		require.NotNil(t, hostProfilesResp.Profiles)
		require.Len(t, hostProfilesResp.Profiles, 1)
		require.Equal(t, p1.Name, hostProfilesResp.Profiles[0].Name)
		require.Equal(t, p1.Identifier, hostProfilesResp.Profiles[0].Identifier)
		require.EqualValues(t, mdmHost.ID, hostProfilesResp.HostID)

		// add the host to a team
		err = s.ds.AddHostsToTeam(ctx, &testTeam.ID, []uint{mdmHost.ID})
		require.NoError(t, err)

		hostProfilesResp = getHostProfilesResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/mdm/hosts/%d/profiles", mdmHost.ID), nil, http.StatusOK, &hostProfilesResp)
		require.NotNil(t, hostProfilesResp.Profiles)
		require.Len(t, hostProfilesResp.Profiles, 2)
		require.EqualValues(t, mdmHost.ID, hostProfilesResp.HostID)
		for _, p := range resp.ConfigProfiles {
			if p.Name == p2.Name {
				require.Equal(t, p2.Identifier, p.Identifier)
			} else if p.Name == p3.Name {
				require.Equal(t, p3.Identifier, p.Identifier)
			} else {
				require.Fail(t, "unexpected profile name")
			}
		}
	})
}

func (s *integrationMDMTestSuite) TestMDMAppleConfigProfileCRUD() {
	t := s.T()
	ctx := context.Background()

	testTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "TestTeam"})
	require.NoError(t, err)

	testProfiles := make(map[string]fleet.MDMAppleConfigProfile)
	generateTestProfile := func(name string, identifier string) {
		i := identifier
		if i == "" {
			i = fmt.Sprintf("%s.SomeIdentifier", name)
		}
		cp := fleet.MDMAppleConfigProfile{
			Name:       name,
			Identifier: i,
		}
		cp.Mobileconfig = mcBytesForTest(cp.Name, cp.Identifier, fmt.Sprintf("%s.UUID", name))
		testProfiles[name] = cp
	}
	setTestProfileID := func(name string, id uint) {
		tp := testProfiles[name]
		tp.ProfileID = id
		testProfiles[name] = tp
	}

	generateNewReq := func(name string, teamID *uint) (*bytes.Buffer, map[string]string) {
		args := map[string][]string{}
		if teamID != nil {
			args["team_id"] = []string{fmt.Sprintf("%d", *teamID)}
		}
		return generateNewProfileMultipartRequest(t, "some_filename", testProfiles[name].Mobileconfig, s.token, args)
	}

	checkGetResponse := func(resp *http.Response, expected fleet.MDMAppleConfigProfile) {
		// check expected headers
		require.Contains(t, resp.Header["Content-Type"], "application/x-apple-aspen-config")
		require.Contains(t, resp.Header["Content-Disposition"], fmt.Sprintf(`attachment;filename="%s_%s.%s"`, time.Now().Format("2006-01-02"), strings.ReplaceAll(expected.Name, " ", "_"), "mobileconfig"))
		// check expected body
		var bb bytes.Buffer
		_, err = io.Copy(&bb, resp.Body)
		require.NoError(t, err)
		require.Equal(t, []byte(expected.Mobileconfig), bb.Bytes())
	}

	checkConfigProfile := func(expected fleet.MDMAppleConfigProfile, actual fleet.MDMAppleConfigProfile) {
		require.Equal(t, expected.Name, actual.Name)
		require.Equal(t, expected.Identifier, actual.Identifier)
	}

	// create new profile (no team)
	generateTestProfile("TestNoTeam", "")
	body, headers := generateNewReq("TestNoTeam", nil)
	newResp := s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusOK, headers)
	var newCP fleet.MDMAppleConfigProfile
	err = json.NewDecoder(newResp.Body).Decode(&newCP)
	require.NoError(t, err)
	require.NotEmpty(t, newCP.ProfileID)
	setTestProfileID("TestNoTeam", newCP.ProfileID)

	// create new profile (with team id)
	generateTestProfile("TestWithTeamID", "")
	body, headers = generateNewReq("TestWithTeamID", &testTeam.ID)
	newResp = s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusOK, headers)
	err = json.NewDecoder(newResp.Body).Decode(&newCP)
	require.NoError(t, err)
	require.NotEmpty(t, newCP.ProfileID)
	setTestProfileID("TestWithTeamID", newCP.ProfileID)

	// list profiles (no team)
	expectedCP := testProfiles["TestNoTeam"]
	var listResp listMDMAppleConfigProfilesResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/profiles", nil, http.StatusOK, &listResp)
	require.Len(t, listResp.ConfigProfiles, 1)
	respCP := listResp.ConfigProfiles[0]
	require.Equal(t, expectedCP.Name, respCP.Name)
	checkConfigProfile(expectedCP, *respCP)
	require.Empty(t, respCP.Mobileconfig) // list profiles endpoint shouldn't include mobileconfig bytes
	require.Empty(t, respCP.TeamID)       // zero means no team

	// list profiles (team 1)
	expectedCP = testProfiles["TestWithTeamID"]
	listResp = listMDMAppleConfigProfilesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/profiles", listMDMAppleConfigProfilesRequest{TeamID: testTeam.ID}, http.StatusOK, &listResp)
	require.Len(t, listResp.ConfigProfiles, 1)
	respCP = listResp.ConfigProfiles[0]
	require.Equal(t, expectedCP.Name, respCP.Name)
	checkConfigProfile(expectedCP, *respCP)
	require.Empty(t, respCP.Mobileconfig)         // list profiles endpoint shouldn't include mobileconfig bytes
	require.Equal(t, testTeam.ID, *respCP.TeamID) // team 1

	// get profile (no team)
	expectedCP = testProfiles["TestNoTeam"]
	getPath := fmt.Sprintf("/api/latest/fleet/mdm/apple/profiles/%d", expectedCP.ProfileID)
	getResp := s.DoRawWithHeaders("GET", getPath, nil, http.StatusOK, map[string]string{"Authorization": fmt.Sprintf("Bearer %s", s.token)})
	checkGetResponse(getResp, expectedCP)

	// get profile (team 1)
	expectedCP = testProfiles["TestWithTeamID"]
	getPath = fmt.Sprintf("/api/latest/fleet/mdm/apple/profiles/%d", expectedCP.ProfileID)
	getResp = s.DoRawWithHeaders("GET", getPath, nil, http.StatusOK, map[string]string{"Authorization": fmt.Sprintf("Bearer %s", s.token)})
	checkGetResponse(getResp, expectedCP)

	// delete profile (no team)
	deletedCP := testProfiles["TestNoTeam"]
	deletePath := fmt.Sprintf("/api/latest/fleet/mdm/apple/profiles/%d", deletedCP.ProfileID)
	var deleteResp deleteMDMAppleConfigProfileResponse
	s.DoJSON("DELETE", deletePath, nil, http.StatusOK, &deleteResp)
	// confirm deleted
	listResp = listMDMAppleConfigProfilesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/profiles", listMDMAppleConfigProfilesRequest{}, http.StatusOK, &listResp)
	require.Len(t, listResp.ConfigProfiles, 0)
	getPath = fmt.Sprintf("/api/latest/fleet/mdm/apple/profiles/%d", deletedCP.ProfileID)
	_ = s.DoRawWithHeaders("GET", getPath, nil, http.StatusNotFound, map[string]string{"Authorization": fmt.Sprintf("Bearer %s", s.token)})

	// delete profile (team 1)
	deletedCP = testProfiles["TestWithTeamID"]
	deletePath = fmt.Sprintf("/api/latest/fleet/mdm/apple/profiles/%d", deletedCP.ProfileID)
	deleteResp = deleteMDMAppleConfigProfileResponse{}
	s.DoJSON("DELETE", deletePath, nil, http.StatusOK, &deleteResp)
	// confirm deleted
	listResp = listMDMAppleConfigProfilesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/profiles", listMDMAppleConfigProfilesRequest{TeamID: testTeam.ID}, http.StatusOK, &listResp)
	require.Len(t, listResp.ConfigProfiles, 0)
	getPath = fmt.Sprintf("/api/latest/fleet/mdm/apple/profiles/%d", deletedCP.ProfileID)
	_ = s.DoRawWithHeaders("GET", getPath, nil, http.StatusNotFound, map[string]string{"Authorization": fmt.Sprintf("Bearer %s", s.token)})

	// trying to add/delete profiles with identifiers managed by Fleet fails
	for p := range mobileconfig.FleetPayloadIdentifiers() {
		generateTestProfile("TestNoTeam", p)
		body, headers := generateNewReq("TestNoTeam", nil)
		s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusBadRequest, headers)

		generateTestProfile("TestWithTeamID", p)
		body, headers = generateNewReq("TestWithTeamID", nil)
		s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusBadRequest, headers)
		cp, err := fleet.NewMDMAppleConfigProfile(mobileconfigForTestWithContent("N1", "I1", p, "random", ""), nil)
		require.NoError(t, err)
		testProfiles["WithContent"] = *cp
		body, headers = generateNewReq("WithContent", nil)
		s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusBadRequest, headers)
	}

	// trying to add profiles with identifiers managed by Fleet fails
	for p := range mobileconfig.FleetPayloadIdentifiers() {
		generateTestProfile("TestNoTeam", p)
		body, headers := generateNewReq("TestNoTeam", nil)
		s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusBadRequest, headers)

		generateTestProfile("TestWithTeamID", p)
		body, headers = generateNewReq("TestWithTeamID", nil)
		s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusBadRequest, headers)
		cp, err := fleet.NewMDMAppleConfigProfile(mobileconfigForTestWithContent("N1", "I1", p, "random", ""), nil)
		require.NoError(t, err)
		testProfiles["WithContent"] = *cp
		body, headers = generateNewReq("WithContent", nil)
		s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusBadRequest, headers)
	}

	// trying to add profiles with names reserved by Fleet fails
	for name := range servermdm.FleetReservedProfileNames() {
		cp := &fleet.MDMAppleConfigProfile{
			Name:         name,
			Identifier:   "valid.identifier",
			Mobileconfig: mcBytesForTest(name, "valid.identifier", "some-uuid"),
		}
		body, headers := generateNewProfileMultipartRequest(t, "some_filename", cp.Mobileconfig, s.token, nil)
		s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusBadRequest, headers)

		body, headers = generateNewProfileMultipartRequest(t, "some_filename", cp.Mobileconfig, s.token, map[string][]string{
			"team_id": {fmt.Sprintf("%d", testTeam.ID)},
		})
		s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusBadRequest, headers)

		cp, err := fleet.NewMDMAppleConfigProfile(mobileconfigForTestWithContent(
			"valid outer name",
			"valid.outer.identifier",
			"valid.inner.identifer",
			"some-uuid",
			name,
		), nil)
		require.NoError(t, err)
		body, headers = generateNewProfileMultipartRequest(t, "some_filename", cp.Mobileconfig, s.token, nil)
		s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusBadRequest, headers)

		cp.TeamID = &testTeam.ID
		body, headers = generateNewProfileMultipartRequest(t, "some_filename", cp.Mobileconfig, s.token, map[string][]string{
			"team_id": {fmt.Sprintf("%d", testTeam.ID)},
		})

		s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusBadRequest, headers)
	}

	// make fleet add a FileVault profile
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	profile := s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// try to delete the profile
	deletePath = fmt.Sprintf("/api/latest/fleet/mdm/apple/profiles/%d", profile.ProfileID)
	deleteResp = deleteMDMAppleConfigProfileResponse{}
	s.DoJSON("DELETE", deletePath, nil, http.StatusBadRequest, &deleteResp)
}

func (s *integrationMDMTestSuite) TestAppConfigMDMAppleProfiles() {
	t := s.T()

	// set the macos custom settings fields
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
		  "macos_settings": {
		    "custom_settings": [
		        {"path": "foo", "labels": ["baz"]},
			{"path": "bar"}
		    ]
		  }
		}
  }`), http.StatusOK, &acResp)
	assert.Equal(t, []fleet.MDMProfileSpec{{Path: "foo", Labels: []string{"baz"}}, {Path: "bar"}}, acResp.MDM.MacOSSettings.CustomSettings)

	// check that they are returned by a GET /config
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.Equal(t, []fleet.MDMProfileSpec{{Path: "foo", Labels: []string{"baz"}}, {Path: "bar"}}, acResp.MDM.MacOSSettings.CustomSettings)

	// patch without specifying the macos custom settings fields and an unrelated
	// field, should not remove them
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusOK, &acResp)
	assert.Equal(t, []fleet.MDMProfileSpec{{Path: "foo", Labels: []string{"baz"}}, {Path: "bar"}}, acResp.MDM.MacOSSettings.CustomSettings)

	// patch with explicitly empty macos custom settings fields, would remove
	// them but this is a dry-run
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "macos_settings": { "custom_settings": null } }
  }`), http.StatusOK, &acResp, "dry_run", "true")
	assert.Equal(t, []fleet.MDMProfileSpec{{Path: "foo", Labels: []string{"baz"}}, {Path: "bar"}}, acResp.MDM.MacOSSettings.CustomSettings)

	// patch with explicitly empty macos custom settings fields, removes them
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "macos_settings": { "custom_settings": null } }
  }`), http.StatusOK, &acResp)
	assert.Empty(t, acResp.MDM.MacOSSettings.CustomSettings)
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

	// use the MDM settings endpoint to set it to true
	s.Do("PATCH", "/api/latest/fleet/mdm/apple/settings",
		fleet.MDMAppleSettingsPayload{EnableDiskEncryption: ptr.Bool(true)}, http.StatusNoContent)
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
	s.Do("PATCH", "/api/latest/fleet/mdm/apple/settings",
		fleet.MDMAppleSettingsPayload{EnableDiskEncryption: ptr.Bool(false)}, http.StatusNoContent)
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.EnableDiskEncryption.Value)

	s.Do("PATCH", "/api/latest/fleet/mdm/apple/settings",
		fleet.MDMAppleSettingsPayload{EnableDiskEncryption: ptr.Bool(true)}, http.StatusNoContent)
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

func (s *integrationMDMTestSuite) TestApplyTeamsMDMAppleProfiles() {
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

	// apply with custom macos settings
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSSettings: map[string]interface{}{
				"custom_settings": []map[string]interface{}{{"path": "foo"}, {"path": "bar"}},
			},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	// retrieving the team returns the custom macos settings
	var teamResp getTeamResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, []fleet.MDMProfileSpec{{Path: "foo"}, {Path: "bar"}}, teamResp.Team.Config.MDM.MacOSSettings.CustomSettings)

	// apply with invalid macos settings subfield should fail
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSSettings: map[string]interface{}{"foo_bar": 123},
		},
	}}}
	res := s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, `unsupported key provided: "foo_bar"`)

	// apply with some good and some bad macos settings subfield should fail
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSSettings: map[string]interface{}{"custom_settings": []interface{}{"A", true}},
		},
	}}}
	res = s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, `invalid value type at 'macos_settings.custom_settings': expected array of MDMProfileSpecs but got bool`)

	// apply without custom macos settings specified and unrelated field, should
	// not replace existing settings
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			EnableDiskEncryption: optjson.SetBool(false),
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, []fleet.MDMProfileSpec{{Path: "foo"}, {Path: "bar"}}, teamResp.Team.Config.MDM.MacOSSettings.CustomSettings)

	// apply with explicitly empty custom macos settings would clear the existing
	// settings, but dry-run
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSSettings: map[string]interface{}{"custom_settings": []map[string]interface{}{}},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, "dry_run", "true")
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, []fleet.MDMProfileSpec{{Path: "foo"}, {Path: "bar"}}, teamResp.Team.Config.MDM.MacOSSettings.CustomSettings)

	// apply with explicitly empty custom macos settings clears the existing settings
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSSettings: map[string]interface{}{"custom_settings": []map[string]interface{}{}},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, []fleet.MDMProfileSpec{}, teamResp.Team.Config.MDM.MacOSSettings.CustomSettings)
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
		http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(team.ID)), "team_name", team.Name)

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
			MacOSSettings:        &fleet.MacOSSettings{},
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
	s.Do("PATCH", "/api/latest/fleet/mdm/apple/settings",
		fleet.MDMAppleSettingsPayload{TeamID: ptr.Uint(team.ID), EnableDiskEncryption: ptr.Bool(true)}, http.StatusNoContent)
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
	s.Do("PATCH", "/api/latest/fleet/mdm/apple/settings",
		fleet.MDMAppleSettingsPayload{TeamID: ptr.Uint(9999)}, http.StatusNotFound)

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
	s.Do("PATCH", "/api/latest/fleet/mdm/apple/settings",
		fleet.MDMAppleSettingsPayload{TeamID: ptr.Uint(team.ID), EnableDiskEncryption: ptr.Bool(false)}, http.StatusNoContent)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.False(t, teamResp.Team.Config.MDM.EnableDiskEncryption)

	s.Do("PATCH", "/api/latest/fleet/mdm/apple/settings",
		fleet.MDMAppleSettingsPayload{TeamID: ptr.Uint(team.ID), EnableDiskEncryption: ptr.Bool(true)}, http.StatusNoContent)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.True(t, teamResp.Team.Config.MDM.EnableDiskEncryption)
}

func (s *integrationMDMTestSuite) TestBatchSetMDMAppleProfiles() {
	t := s.T()
	ctx := context.Background()

	// create a new team
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "batch_set_mdm_profiles"})
	require.NoError(t, err)

	// apply an empty set to no-team
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: nil}, http.StatusNoContent)
	s.lastActivityMatches(
		fleet.ActivityTypeEditedMacosProfile{}.ActivityName(),
		`{"team_id": null, "team_name": null}`,
		0,
	)

	// apply to both team id and name
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: nil},
		http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)), "team_name", tm.Name)

	// invalid team name
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: nil},
		http.StatusNotFound, "team_name", uuid.New().String())

	// duplicate profile names
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{
		mobileconfigForTest("N1", "I1"),
		mobileconfigForTest("N1", "I2"),
	}}, http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)))

	// profiles with reserved identifiers
	for p := range mobileconfig.FleetPayloadIdentifiers() {
		res := s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{
			mobileconfigForTest("N1", "I1"),
			mobileconfigForTest(p, p),
		}}, http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)))
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, fmt.Sprintf("Validation Failed: payload identifier %s is not allowed", p))
	}

	// payloads with reserved types
	for p := range mobileconfig.FleetPayloadTypes() {
		res := s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{
			mobileconfigForTestWithContent("N1", "I1", "II1", p, ""),
		}}, http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)))
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, fmt.Sprintf("Validation Failed: unsupported PayloadType(s): %s", p))
	}

	// payloads with reserved identifiers
	for p := range mobileconfig.FleetPayloadIdentifiers() {
		res := s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{
			mobileconfigForTestWithContent("N1", "I1", p, "random", ""),
		}}, http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)))
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, fmt.Sprintf("Validation Failed: unsupported PayloadIdentifier(s): %s", p))
	}

	// successfully apply a profile for the team
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{
		mobileconfigForTest("N1", "I1"),
	}}, http.StatusNoContent, "team_id", strconv.Itoa(int(tm.ID)))
	s.lastActivityMatches(
		fleet.ActivityTypeEditedMacosProfile{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q}`, tm.ID, tm.Name),
		0,
	)
}

func (s *integrationMDMTestSuite) TestEnrollOrbitAfterDEPSync() {
	t := s.T()
	ctx := context.Background()

	// create a host with minimal information and the serial, no uuid/osquery id
	// (as when created via DEP sync). Platform must be "darwin" as this is the
	// only supported OS with DEP.
	dbZeroTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	h, err := s.ds.NewHost(ctx, &fleet.Host{
		HardwareSerial:   uuid.New().String(),
		Platform:         "darwin",
		LastEnrolledAt:   dbZeroTime,
		DetailUpdatedAt:  dbZeroTime,
		RefetchRequested: true,
	})
	require.NoError(t, err)

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
	s.DoJSON("POST", "/api/fleet/orbit/enroll", EnrollOrbitRequest{
		EnrollSecret:   secret,
		HardwareUUID:   hostUUID, // will not match any existing host
		HardwareSerial: h.HardwareSerial,
	}, http.StatusOK, &resp)
	require.NotEmpty(t, resp.OrbitNodeKey)

	// fetch the host, it will match the one created above
	// (NOTE: cannot check the returned OrbitNodeKey, this field is not part of the response)
	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", h.ID), nil, http.StatusOK, &hostResp)
	require.Equal(t, h.ID, hostResp.Host.ID)

	got, err := s.ds.LoadHostByOrbitNodeKey(ctx, resp.OrbitNodeKey)
	require.NoError(t, err)
	require.Equal(t, h.ID, got.ID)

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

func (s *integrationMDMTestSuite) TestDiskEncryptionRotation() {
	t := s.T()
	h := createOrbitEnrolledHost(t, "darwin", "h", s.ds)

	// false by default
	resp := orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h.OrbitNodeKey)), http.StatusOK, &resp)
	require.False(t, resp.Notifications.RotateDiskEncryptionKey)

	// create an auth token for h
	token := "much_valid"
	mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(context.Background(), `INSERT INTO host_device_auth (host_id, token) VALUES (?, ?)`, h.ID, token)
		return err
	})

	tokRes := s.DoRawNoAuth("POST", "/api/latest/fleet/device/"+token+"/rotate_encryption_key", nil, http.StatusOK)
	tokRes.Body.Close()

	// true after the POST request
	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h.OrbitNodeKey)), http.StatusOK, &resp)
	require.True(t, resp.Notifications.RotateDiskEncryptionKey)

	// false on following requests
	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h.OrbitNodeKey)), http.StatusOK, &resp)
	require.False(t, resp.Notifications.RotateDiskEncryptionKey)
}

func (s *integrationMDMTestSuite) TestHostMDMAppleProfilesStatus() {
	t := s.T()
	ctx := context.Background()

	createManualMDMEnrollWithOrbit := func(secret string) *fleet.Host {
		// orbit enrollment happens before mdm enrollment, otherwise the host would
		// always receive the "no team" profiles on mdm enrollment since it would
		// not be part of any team yet (team assignment is done when it enrolls
		// with orbit).
		mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
			SCEPChallenge: s.fleetCfg.MDM.AppleSCEPChallenge,
			SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
			MDMURL:        s.server.URL + apple_mdm.MDMPath,
		})

		// enroll the device with orbit
		var resp EnrollOrbitResponse
		s.DoJSON("POST", "/api/fleet/orbit/enroll", EnrollOrbitRequest{
			EnrollSecret:   secret,
			HardwareUUID:   mdmDevice.UUID, // will not match any existing host
			HardwareSerial: mdmDevice.SerialNumber,
		}, http.StatusOK, &resp)
		require.NotEmpty(t, resp.OrbitNodeKey)
		orbitNodeKey := resp.OrbitNodeKey
		h, err := s.ds.LoadHostByOrbitNodeKey(ctx, orbitNodeKey)
		require.NoError(t, err)
		h.OrbitNodeKey = &orbitNodeKey
		h.Platform = "darwin"

		err = mdmDevice.Enroll()
		require.NoError(t, err)

		return h
	}

	triggerReconcileProfiles := func() {
		s.awaitTriggerProfileSchedule(t)
		// this will only mark them as "pending", as the response to confirm
		// profile deployment is asynchronous, so we simulate it here by
		// updating any "pending" (not NULL) profiles to "verifying"
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `UPDATE host_mdm_apple_profiles SET status = ? WHERE status = ?`, fleet.OSSettingsVerifying, fleet.OSSettingsPending)
			return err
		})
	}

	assignHostToTeam := func(h *mdm_types.Host, teamID *uint) {
		var moveHostResp addHostsToTeamResponse
		s.DoJSON("POST", "/api/v1/fleet/hosts/transfer",
			addHostsToTeamRequest{TeamID: teamID, HostIDs: []uint{h.ID}}, http.StatusOK, &moveHostResp)

		h.TeamID = teamID
	}

	// add a couple global profiles
	globalProfiles := [][]byte{
		mobileconfigForTest("G1", "G1"),
		mobileconfigForTest("G2", "G2"),
	}
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch",
		batchSetMDMAppleProfilesRequest{Profiles: globalProfiles}, http.StatusNoContent)
	// create the no-team enroll secret
	var applyResp applyEnrollSecretSpecResponse
	globalEnrollSec := "global_enroll_sec"
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret",
		applyEnrollSecretSpecRequest{
			Spec: &fleet.EnrollSecretSpec{
				Secrets: []*fleet.EnrollSecret{{Secret: globalEnrollSec}},
			},
		}, http.StatusOK, &applyResp)

	// create a team with a couple profiles
	tm1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team_profiles_status_1"})
	require.NoError(t, err)
	tm1Profiles := [][]byte{
		mobileconfigForTest("T1.1", "T1.1"),
		mobileconfigForTest("T1.2", "T1.2"),
	}
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch",
		batchSetMDMAppleProfilesRequest{Profiles: tm1Profiles}, http.StatusNoContent,
		"team_id", strconv.Itoa(int(tm1.ID)))
	// create the team 1 enroll secret
	var teamResp teamEnrollSecretsResponse
	tm1EnrollSec := "team1_enroll_sec"
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", tm1.ID),
		modifyTeamEnrollSecretsRequest{
			Secrets: []fleet.EnrollSecret{{Secret: tm1EnrollSec}},
		}, http.StatusOK, &teamResp)

	// create another team with different profiles
	tm2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team_profiles_status_2"})
	require.NoError(t, err)
	tm2Profiles := [][]byte{
		mobileconfigForTest("T2.1", "T2.1"),
	}
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch",
		batchSetMDMAppleProfilesRequest{Profiles: tm2Profiles}, http.StatusNoContent,
		"team_id", strconv.Itoa(int(tm2.ID)))

	// enroll a couple hosts in no team
	h1 := createManualMDMEnrollWithOrbit(globalEnrollSec)
	require.Nil(t, h1.TeamID)
	h2 := createManualMDMEnrollWithOrbit(globalEnrollSec)
	require.Nil(t, h2.TeamID)
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: "G1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
		h2: {
			{Identifier: "G1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})

	// enroll a couple hosts in team 1
	h3 := createManualMDMEnrollWithOrbit(tm1EnrollSec)
	require.NotNil(t, h3.TeamID)
	require.Equal(t, tm1.ID, *h3.TeamID)
	h4 := createManualMDMEnrollWithOrbit(tm1EnrollSec)
	require.NotNil(t, h4.TeamID)
	require.Equal(t, tm1.ID, *h4.TeamID)
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h3: {
			{Identifier: "T1.1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T1.2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
		h4: {
			{Identifier: "T1.1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T1.2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})

	// apply the pending profiles
	triggerReconcileProfiles()

	// switch a no team host (h1) to a team (tm2)
	var moveHostResp addHostsToTeamResponse
	s.DoJSON("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &tm2.ID, HostIDs: []uint{h1.ID}}, http.StatusOK, &moveHostResp)
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: "G1", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G2", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T2.1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h2: {
			{Identifier: "G1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// switch a team host (h3) to another team (tm2)
	s.DoJSON("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &tm2.ID, HostIDs: []uint{h3.ID}}, http.StatusOK, &moveHostResp)
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h3: {
			{Identifier: "T1.1", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T1.2", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T2.1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
		h4: {
			{Identifier: "T1.1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "T1.2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// switch a team host (h4) to no team
	s.DoJSON("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: nil, HostIDs: []uint{h4.ID}}, http.StatusOK, &moveHostResp)
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h3: {
			{Identifier: "T1.1", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T1.2", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T2.1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
		h4: {
			{Identifier: "T1.1", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T1.2", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})

	// apply the pending profiles
	triggerReconcileProfiles()

	// add a profile to no team (h2 and h4 are now part of no team)
	body, headers := generateNewProfileMultipartRequest(t,
		"some_name", mobileconfigForTest("G3", "G3"), s.token, nil)
	s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusOK, headers)
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h2: {
			{Identifier: "G1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
		h4: {
			{Identifier: "G1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// add a profile to team 2 (h1 and h3 are now part of team 2)
	body, headers = generateNewProfileMultipartRequest(t,
		"some_name", mobileconfigForTest("T2.2", "T2.2"), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", tm2.ID)}})
	s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusOK, headers)
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: "T2.1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "T2.2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h3: {
			{Identifier: "T2.1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "T2.2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// apply the pending profiles
	triggerReconcileProfiles()

	// delete a no team profile
	noTeamProfs, err := s.ds.ListMDMAppleConfigProfiles(ctx, nil)
	require.NoError(t, err)
	var g1ProfID uint
	for _, p := range noTeamProfs {
		if p.Identifier == "G1" {
			g1ProfID = p.ProfileID
			break
		}
	}
	require.NotZero(t, g1ProfID)
	var delProfResp deleteMDMAppleConfigProfileResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/mdm/apple/profiles/%d", g1ProfID),
		deleteMDMAppleConfigProfileRequest{}, http.StatusOK, &delProfResp)
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h2: {
			{Identifier: "G1", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h4: {
			{Identifier: "G1", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// delete a team profile
	tm2Profs, err := s.ds.ListMDMAppleConfigProfiles(ctx, &tm2.ID)
	require.NoError(t, err)
	var tm21ProfID uint
	for _, p := range tm2Profs {
		if p.Identifier == "T2.1" {
			tm21ProfID = p.ProfileID
			break
		}
	}
	require.NotZero(t, tm21ProfID)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/mdm/apple/profiles/%d", tm21ProfID),
		deleteMDMAppleConfigProfileRequest{}, http.StatusOK, &delProfResp)
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: "T2.1", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T2.2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h3: {
			{Identifier: "T2.1", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T2.2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// apply the pending profiles
	triggerReconcileProfiles()

	// bulk-set profiles for no team, with add/delete/edit
	g2Edited := mobileconfigForTest("G2b", "G2b")
	g4Content := mobileconfigForTest("G4", "G4")
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/batch",
		batchSetMDMAppleProfilesRequest{
			Profiles: [][]byte{
				g2Edited,
				// G3 is deleted
				g4Content,
			},
		}, http.StatusNoContent)

	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h2: {
			{Identifier: "G2", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G3", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h4: {
			{Identifier: "G2", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G3", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// bulk-set profiles for a team, with add/delete/edit
	t22Edited := mobileconfigForTest("T2.2b", "T2.2b")
	t23Content := mobileconfigForTest("T2.3", "T2.3")
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/batch",
		batchSetMDMAppleProfilesRequest{
			Profiles: [][]byte{
				t22Edited,
				t23Content,
			},
		}, http.StatusNoContent, "team_id", fmt.Sprint(tm2.ID))
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: "T2.2", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T2.2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T2.3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h3: {
			{Identifier: "T2.2", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T2.2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T2.3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// apply the pending profiles
	triggerReconcileProfiles()

	// bulk-set profiles for no team and team 2, without changes, and team 1 added (but no host affected)
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/batch",
		batchSetMDMAppleProfilesRequest{
			Profiles: [][]byte{
				g2Edited,
				g4Content,
			},
		}, http.StatusNoContent)
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/batch",
		batchSetMDMAppleProfilesRequest{
			Profiles: [][]byte{
				t22Edited,
				t23Content,
			},
		}, http.StatusNoContent, "team_id", fmt.Sprint(tm2.ID))
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/batch",
		batchSetMDMAppleProfilesRequest{
			Profiles: [][]byte{
				mobileconfigForTest("T1.3", "T1.3"),
			},
		}, http.StatusNoContent, "team_id", fmt.Sprint(tm1.ID))
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: "T2.2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "T2.3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h2: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h3: {
			{Identifier: "T2.2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "T2.3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h4: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// delete team 2 (h1 and h3 are part of that team)
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d", tm2.ID), nil, http.StatusOK)
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: "T2.2b", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T2.3", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h3: {
			{Identifier: "T2.2b", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "T2.3", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// apply the pending profiles
	triggerReconcileProfiles()

	// all profiles now verifying
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h2: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h3: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h4: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// h1 verified one of the profiles
	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(context.Background(), s.ds, h1, map[string]*fleet.HostMacOSProfile{
		"G2b": {Identifier: "G2b", DisplayName: "G2b", InstallDate: time.Now()},
	}))
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerified},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h2: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h3: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h4: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// switch a team host (h1) to another team (tm1)
	assignHostToTeam(h1, &tm1.ID)

	// Create a new profile that will be labeled
	body, headers = generateNewProfileMultipartRequest(
		t,
		"label_prof",
		mobileconfigForTest("label_prof", "label_prof"),
		s.token,
		map[string][]string{"team_id": {fmt.Sprintf("%d", tm1.ID)}},
	)
	s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusOK, headers)

	var uid string
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &uid, `SELECT profile_uuid FROM mdm_apple_configuration_profiles WHERE identifier = ?`, "label_prof")
	})

	label, err := s.ds.NewLabel(ctx, &mdm_types.Label{Name: "test label 1", Query: "select 1;"})
	require.NoError(t, err)

	// Update label with host membership
	mysql.ExecAdhocSQL(
		t, s.ds, func(db sqlx.ExtContext) error {
			_, err := db.ExecContext(
				context.Background(),
				"INSERT IGNORE INTO label_membership (host_id, label_id) VALUES (?, ?)",
				h1.ID,
				label.ID,
			)
			return err
		},
	)

	// Update profile <-> label mapping
	mysql.ExecAdhocSQL(
		t, s.ds, func(db sqlx.ExtContext) error {
			_, err := db.ExecContext(
				context.Background(),
				"INSERT INTO mdm_configuration_profile_labels (apple_profile_uuid, label_name, label_id) VALUES (?, ?, ?)",
				uid,
				label.Name,
				label.ID,
			)
			return err
		},
	)

	triggerReconcileProfiles()

	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "T1.3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "label_prof", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h2: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h3: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h4: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(context.Background(), s.ds, h1, map[string]*fleet.HostMacOSProfile{
		"label_prof": {Identifier: "label_prof", DisplayName: "label_prof", InstallDate: time.Now()},
	}))

	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "T1.3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "label_prof", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerified},
		},
		h2: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h3: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		h4: {
			{Identifier: "G2b", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})
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

	// set the default bm assignment to that team
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": {
			"apple_bm_default_team": %q
		}
	}`, tm.Name)), http.StatusOK, &acResp)

	// the team doesn't have any enroll secrets yet, a profile is created using the global enroll secret
	s.awaitTriggerProfileSchedule(t)
	p := s.assertConfigProfilesByIdentifier(&tm.ID, mobileconfig.FleetdConfigPayloadIdentifier, true)
	require.Contains(t, string(p.Mobileconfig), t.Name())

	// create an enroll secret for the team
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name:    tm.Name,
		Secrets: []fleet.EnrollSecret{{Secret: t.Name() + "team-secret"}},
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

	// Create host enrolled via osquery, but not enrolled in MDM.
	unenrolledHost := createHostAndDeviceToken(t, s.ds, "unused")

	// Create device enrolled in MDM but not enrolled via osquery.
	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.fleetCfg.MDM.AppleSCEPChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	})
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
	s.DoJSON("GET", "/api/latest/fleet/mdm/commandresults", nil, http.StatusNotFound, &cmdResResp, "command_uuid", uuid1)

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
	s.DoJSON("GET", "/api/latest/fleet/mdm/commandresults", nil, http.StatusOK, &getMDMCmdResp, "command_uuid", uuid2)
	require.Len(t, getMDMCmdResp.Results, 0)

	// simulate a result and call again
	err = s.mdmStorage.StoreCommandReport(&mdm.Request{
		EnrollID: &mdm.EnrollID{ID: mdmDevice.UUID},
		Context:  ctx,
	}, &mdm.CommandResults{
		CommandUUID: uuid2,
		Status:      "Acknowledged",
		RequestType: "ProfileList",
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

	s.DoJSON("GET", "/api/latest/fleet/mdm/commandresults", nil, http.StatusOK, &getMDMCmdResp, "command_uuid", uuid2)
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
	require.Len(t, listCmdResp.Results, 1)
	require.NotZero(t, listCmdResp.Results[0].UpdatedAt)
	listCmdResp.Results[0].UpdatedAt = time.Time{}
	require.Equal(t, &fleet.MDMAppleCommand{
		DeviceID:    mdmDevice.UUID,
		CommandUUID: uuid2,
		Status:      "Acknowledged",
		RequestType: "ProfileList",
		Hostname:    "test-host",
	}, listCmdResp.Results[0])
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
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/mdm/commandresults?command_uuid=%s", cmdUUID), nil, http.StatusOK, &resp)
	require.Len(t, resp.Results, 1)
	require.Equal(t, dev.HostUUID, resp.Results[0].HostUUID)
	require.Equal(t, cmdUUID, resp.Results[0].CommandUUID)
	require.Equal(t, rawResponse, resp.Results[0].Result)
	require.Equal(t, cmdTarget, resp.Results[0].RequestType)
	require.Equal(t, statusCode, resp.Results[0].Status)
	require.Equal(t, h.Hostname, resp.Results[0].Hostname)

	resp = getMDMCommandResultsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/mdm/commandresults?command_uuid=%s", uuid.New().String()), nil, http.StatusNotFound, &resp)
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
	s.DoJSON("GET", "/api/latest/fleet/mdm/bootstrap/0/metadata", nil, http.StatusOK, &metadataResp)
	require.Equal(t, metadataResp.MDMAppleBootstrapPackage.Name, "pkg.pkg")
	require.NotEmpty(t, metadataResp.MDMAppleBootstrapPackage.Sha256, "")
	require.NotEmpty(t, metadataResp.MDMAppleBootstrapPackage.Token)

	// download a package, wrong token
	var downloadResp downloadBootstrapPackageResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/bootstrap?token=bad", nil, http.StatusNotFound, &downloadResp)

	resp := s.DoRaw("GET", fmt.Sprintf("/api/latest/fleet/mdm/bootstrap?token=%s", metadataResp.MDMAppleBootstrapPackage.Token), nil, http.StatusOK)
	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.EqualValues(t, signedPkg, respBytes)

	// missing package
	metadataResp = bootstrapPackageMetadataResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/bootstrap/1/metadata", nil, http.StatusNotFound, &metadataResp)

	// delete package
	var deleteResp deleteBootstrapPackageResponse
	s.DoJSON("DELETE", "/api/latest/fleet/mdm/bootstrap/0", nil, http.StatusOK, &deleteResp)
	// check the activity log
	s.lastActivityMatches(
		fleet.ActivityTypeDeletedBootstrapPackage{}.ActivityName(),
		`{"bootstrap_package_name": "pkg.pkg", "team_id": null, "team_name": null}`,
		0,
	)

	metadataResp = bootstrapPackageMetadataResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/bootstrap/0/metadata", nil, http.StatusNotFound, &metadataResp)
	// trying to delete again is a bad request
	s.DoJSON("DELETE", "/api/latest/fleet/mdm/bootstrap/0", nil, http.StatusNotFound, &deleteResp)
}

func (s *integrationMDMTestSuite) TestBootstrapPackageStatus() {
	t := s.T()
	pkg, err := os.ReadFile(filepath.Join("testdata", "bootstrap-packages", "signed.pkg"))
	require.NoError(t, err)

	// upload a bootstrap package for "no team"
	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Bytes: pkg, Name: "pkg.pkg", TeamID: 0}, http.StatusOK, "")

	// get package metadata
	var metadataResp bootstrapPackageMetadataResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/bootstrap/0/metadata", nil, http.StatusOK, &metadataResp)
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
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/mdm/bootstrap/%d/metadata", team.ID), nil, http.StatusOK, &metadataResp)
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
		SCEPChallenge: s.fleetCfg.MDM.AppleSCEPChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}
	noTeamDevices := []deviceWithResponse{
		{"Acknowledge", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo)},
		{"Acknowledge", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo)},
		{"Acknowledge", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo)},
		{"Error", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo)},
		{"Offline", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo)},
		{"Offline", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo)},
		{"Pending", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo)},
		{"Pending", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo)},
	}

	teamDevices := []deviceWithResponse{
		{"Acknowledge", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo)},
		{"Acknowledge", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo)},
		{"Error", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo)},
		{"Error", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo)},
		{"Error", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo)},
		{"Offline", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo)},
		{"Pending", mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo)},
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
	_, err = s.ds.NewHost(context.Background(), &fleet.Host{
		OsqueryHostID: ptr.String("non-macos-host"),
		NodeKey:       ptr.String("non-macos-host"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.non.macos", t.Name()),
		Platform:      "windows",
	})
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

	ch := make(chan bool)
	mockRespDevices := noTeamDevices
	s.mockDEPResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			ch <- true
			_, _ = w.Write([]byte(`{}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))

	// trigger a dep sync
	_, err = s.depSchedule.Trigger()
	require.NoError(t, err)
	<-ch

	var summaryResp getMDMAppleBootstrapPackageSummaryResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/bootstrap/summary", nil, http.StatusOK, &summaryResp)
	require.Equal(t, fleet.MDMAppleBootstrapPackageSummary{Pending: uint(len(noTeamDevices))}, summaryResp.MDMAppleBootstrapPackageSummary)

	// set the default bm assignment to `team`
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": {
			"apple_bm_default_team": %q
		}
	}`, team.Name)), http.StatusOK, &acResp)

	// trigger a dep sync
	mockRespDevices = teamDevices
	_, err = s.depSchedule.Trigger()
	require.NoError(t, err)
	<-ch

	summaryResp = getMDMAppleBootstrapPackageSummaryResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/mdm/bootstrap/summary?team_id=%d", team.ID), nil, http.StatusOK, &summaryResp)
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
			// if the command is to install the bootstrap package
			if manifest := cmd.Command.InstallEnterpriseApplication.Manifest; manifest != nil {
				require.Equal(t, "InstallEnterpriseApplication", cmd.Command.RequestType)
				require.Equal(t, "software-package", (*manifest).ManifestItems[0].Assets[0].Kind)
				wantURL, err := bp.URL(s.server.URL)
				require.NoError(t, err)
				require.Equal(t, wantURL, (*manifest).ManifestItems[0].Assets[0].URL)

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
	s.DoJSON("GET", "/api/latest/fleet/mdm/bootstrap/summary", nil, http.StatusOK, &summaryResp)
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
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/mdm/bootstrap/summary?team_id=%d", team.ID), nil, http.StatusOK, &summaryResp)
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
	s.DoJSON("GET", "/api/latest/fleet/mdm/setup/eula/metadata", nil, http.StatusNotFound, &metadataResp)

	// trying to upload a file that is not a PDF fails
	s.uploadEULA(&fleet.MDMEULA{Bytes: []byte("should-fail"), Name: "should-fail.pdf"}, http.StatusBadRequest, "invalid file type")
	// trying to upload an empty file fails
	s.uploadEULA(&fleet.MDMEULA{Bytes: []byte{}, Name: "should-fail.pdf"}, http.StatusBadRequest, "invalid file type")

	// admin is able to upload a new EULA
	s.uploadEULA(&fleet.MDMEULA{Bytes: pdfBytes, Name: pdfName}, http.StatusOK, "")

	// get EULA metadata
	metadataResp = getMDMEULAMetadataResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/setup/eula/metadata", nil, http.StatusOK, &metadataResp)
	require.NotEmpty(t, metadataResp.MDMEULA.Token)
	require.NotEmpty(t, metadataResp.MDMEULA.CreatedAt)
	require.Equal(t, pdfName, metadataResp.MDMEULA.Name)
	eulaToken := metadataResp.Token

	// download EULA
	resp := s.DoRaw("GET", fmt.Sprintf("/api/latest/fleet/mdm/setup/eula/%s", eulaToken), nil, http.StatusOK)
	require.EqualValues(t, len(pdfBytes), resp.ContentLength)
	require.Equal(t, "application/pdf", resp.Header.Get("content-type"))
	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.EqualValues(t, pdfBytes, respBytes)

	// try to download EULA with a bad token
	var downloadResp downloadBootstrapPackageResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/setup/eula/bad-token", nil, http.StatusNotFound, &downloadResp)

	// trying to upload any EULA without deleting the previous one first results in an error
	s.uploadEULA(&fleet.MDMEULA{Bytes: pdfBytes, Name: "should-fail.pdf"}, http.StatusConflict, "")

	// delete EULA
	var deleteResp deleteMDMEULAResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/mdm/setup/eula/%s", eulaToken), nil, http.StatusOK, &deleteResp)
	metadataResp = getMDMEULAMetadataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/mdm/setup/eula/%s", eulaToken), nil, http.StatusNotFound, &metadataResp)
	// trying to delete again is a bad request
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/mdm/setup/eula/%s", eulaToken), nil, http.StatusNotFound, &deleteResp)
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

	// host is not DEP so migration is not allowed
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), h.ID, !isServer, enrolled, mdmURL, !installedFromDEP, mdmName, ""))
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

	// simulate that the device is assigned to Fleet in ABM
	s.mockDEPResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func (s *integrationMDMTestSuite) TestMDMMacOSSetup() {
	t := s.T()

	s.mockDEPResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	cases := []struct {
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

	t.Run("UpdateAppConfig", func(t *testing.T) {
		acResp := appConfigResponse{}
		path := "/api/latest/fleet/config"
		fmtJSON := func(s string) json.RawMessage {
			return json.RawMessage(fmt.Sprintf(`{
				%s
			}`, s))
		}

		// get the initial appconfig; enable end user authentication default is false
		s.DoJSON("GET", path, nil, http.StatusOK, &acResp)
		require.False(t, acResp.MDM.MacOSSetup.EnableEndUserAuthentication)

		for i, c := range cases {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				acResp = appConfigResponse{}
				s.DoJSON("PATCH", path, fmtJSON(c.raw), http.StatusOK, &acResp)
				require.Equal(t, c.expected, acResp.MDM.MacOSSetup.EnableEndUserAuthentication)

				acResp = appConfigResponse{}
				s.DoJSON("GET", path, nil, http.StatusOK, &acResp)
				require.Equal(t, c.expected, acResp.MDM.MacOSSetup.EnableEndUserAuthentication)
			})
		}
	})

	t.Run("UpdateTeamConfig", func(t *testing.T) {
		path := fmt.Sprintf("/api/latest/fleet/teams/%d", tm.ID)
		fmtJSON := `{
			"name": %q,
			%s
		}`

		// get the initial team config; enable end user authentication default is false
		teamResp := teamResponse{}
		s.DoJSON("GET", path, nil, http.StatusOK, &teamResp)
		require.False(t, teamResp.Team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)

		for i, c := range cases {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				teamResp = teamResponse{}
				s.DoJSON("PATCH", path, json.RawMessage(fmt.Sprintf(fmtJSON, tm.Name, c.raw)), http.StatusOK, &teamResp)
				require.Equal(t, c.expected, teamResp.Team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)

				teamResp = teamResponse{}
				s.DoJSON("GET", path, nil, http.StatusOK, &teamResp)
				require.Equal(t, c.expected, teamResp.Team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
			})
		}
	})

	t.Run("TestMDMAppleSetupEndpoint", func(t *testing.T) {
		t.Run("TestNoTeam", func(t *testing.T) {
			var acResp appConfigResponse
			s.Do("PATCH", "/api/latest/fleet/mdm/apple/setup",
				fleet.MDMAppleSetupPayload{TeamID: ptr.Uint(0), EnableEndUserAuthentication: ptr.Bool(true)}, http.StatusNoContent)
			acResp = appConfigResponse{}
			s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
			require.True(t, acResp.MDM.MacOSSetup.EnableEndUserAuthentication)
			lastActivityID := s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledMacosSetupEndUserAuth{}.ActivityName(),
				`{"team_id": null, "team_name": null}`, 0)

			s.Do("PATCH", "/api/latest/fleet/mdm/apple/setup",
				fleet.MDMAppleSetupPayload{TeamID: ptr.Uint(0), EnableEndUserAuthentication: ptr.Bool(true)}, http.StatusNoContent)
			acResp = appConfigResponse{}
			s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
			require.True(t, acResp.MDM.MacOSSetup.EnableEndUserAuthentication)
			s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledMacosSetupEndUserAuth{}.ActivityName(),
				``, lastActivityID) // no new activity

			s.Do("PATCH", "/api/latest/fleet/mdm/apple/setup",
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
			s.Do("PATCH", "/api/latest/fleet/mdm/apple/setup",
				fleet.MDMAppleSetupPayload{TeamID: &tm.ID, EnableEndUserAuthentication: ptr.Bool(true)}, http.StatusNoContent)
			tmResp = teamResponse{}
			s.DoJSON("GET", tmConfigPath, nil, http.StatusOK, &tmResp)
			require.True(t, tmResp.Team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
			lastActivityID := s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledMacosSetupEndUserAuth{}.ActivityName(),
				expectedActivityDetail, 0)

			s.Do("PATCH", "/api/latest/fleet/mdm/apple/setup",
				fleet.MDMAppleSetupPayload{TeamID: &tm.ID, EnableEndUserAuthentication: ptr.Bool(true)}, http.StatusNoContent)
			tmResp = teamResponse{}
			s.DoJSON("GET", tmConfigPath, nil, http.StatusOK, &tmResp)
			require.True(t, tmResp.Team.Config.MDM.MacOSSetup.EnableEndUserAuthentication)
			s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledMacosSetupEndUserAuth{}.ActivityName(),
				``, lastActivityID) // no new activity

			s.Do("PATCH", "/api/latest/fleet/mdm/apple/setup",
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
		s.Do("PATCH", "/api/latest/fleet/mdm/apple/setup",
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
		s.Do("PATCH", "/api/latest/fleet/mdm/apple/setup",
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

	// get for no team returns 404
	var getResp getMDMAppleSetupAssistantResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/enrollment_profile", nil, http.StatusNotFound, &getResp)
	// get for non-existing team returns 404
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/enrollment_profile", nil, http.StatusNotFound, &getResp, "team_id", "123")

	// create a setup assistant for no team
	noTeamProf := `{"x": 1}`
	var createResp createMDMAppleSetupAssistantResponse
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/enrollment_profile", createMDMAppleSetupAssistantRequest{
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
	tmProf := `{"y": 1}`
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/enrollment_profile", createMDMAppleSetupAssistantRequest{
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
	noTeamProf = `{"x": 2}`
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/enrollment_profile", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "no-team2",
		EnrollmentProfile: json.RawMessage(noTeamProf),
	}, http.StatusOK, &createResp)
	s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		`{"name": "no-team2", "team_id": null, "team_name": null}`, 0)

	// update team
	tmProf = `{"y": 2}`
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/enrollment_profile", createMDMAppleSetupAssistantRequest{
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
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/enrollment_profile", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "no-team2",
		EnrollmentProfile: json.RawMessage(noTeamProf),
	}, http.StatusOK, &createResp)
	// the last activity is that of the team (i.e. no new activity was created for no-team)
	s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "team2", "team_id": %d, "team_name": %q}`, tm.ID, tm.Name), lastChangedActID)

	// no change to team
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/enrollment_profile", createMDMAppleSetupAssistantRequest{
		TeamID:            &tm.ID,
		Name:              "team2",
		EnrollmentProfile: json.RawMessage(tmProf),
	}, http.StatusOK, &createResp)
	s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "team2", "team_id": %d, "team_name": %q}`, tm.ID, tm.Name), lastChangedActID)

	// update team with only a setup assistant JSON change, should detect it
	// and create a new activity (name is the same)
	tmProf = `{"y": 3}`
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/enrollment_profile", createMDMAppleSetupAssistantRequest{
		TeamID:            &tm.ID,
		Name:              "team2",
		EnrollmentProfile: json.RawMessage(tmProf),
	}, http.StatusOK, &createResp)
	latestChangedActID := s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "team2", "team_id": %d, "team_name": %q}`, tm.ID, tm.Name), 0)
	require.Greater(t, latestChangedActID, lastChangedActID)

	// get no team
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/enrollment_profile", nil, http.StatusOK, &getResp)
	require.Nil(t, getResp.TeamID)
	require.NotZero(t, getResp.UploadedAt)
	require.Equal(t, "no-team2", getResp.Name)
	require.JSONEq(t, noTeamProf, string(getResp.Profile))

	// get team
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/enrollment_profile", nil, http.StatusOK, &getResp, "team_id", fmt.Sprint(tm.ID))
	require.NotNil(t, getResp.TeamID)
	require.Equal(t, tm.ID, *getResp.TeamID)
	require.NotZero(t, getResp.UploadedAt)
	require.Equal(t, "team2", getResp.Name)
	require.JSONEq(t, tmProf, string(getResp.Profile))

	// try to set the url
	tmProf = `{"url": "https://example.com"}`
	res := s.Do("POST", "/api/latest/fleet/mdm/apple/enrollment_profile", createMDMAppleSetupAssistantRequest{
		TeamID:            &tm.ID,
		Name:              "team5",
		EnrollmentProfile: json.RawMessage(tmProf),
	}, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `The automatic enrollment profile can’t include url.`)
	s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "team2", "team_id": %d, "team_name": %q}`, tm.ID, tm.Name), latestChangedActID)

	// try to set a non-object json value
	tmProf = `true`
	res = s.Do("POST", "/api/latest/fleet/mdm/apple/enrollment_profile", createMDMAppleSetupAssistantRequest{
		TeamID:            &tm.ID,
		Name:              "team6",
		EnrollmentProfile: json.RawMessage(tmProf),
	}, http.StatusInternalServerError) // TODO: that should be a 4xx error, see #4406
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `cannot unmarshal bool into Go value of type map[string]interface`)
	s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "team2", "team_id": %d, "team_name": %q}`, tm.ID, tm.Name), latestChangedActID)

	// delete the no-team setup assistant
	s.Do("DELETE", "/api/latest/fleet/mdm/apple/enrollment_profile", nil, http.StatusNoContent)
	latestChangedActID = s.lastActivityMatches(fleet.ActivityTypeDeletedMacosSetupAssistant{}.ActivityName(),
		`{"name": "no-team2", "team_id": null, "team_name": null}`, 0)

	// get for no team returns 404
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/enrollment_profile", nil, http.StatusNotFound, &getResp)

	// delete the team (not the assistant), this also deletes the assistant
	err = s.ds.DeleteTeam(ctx, tm.ID)
	require.NoError(t, err)

	// get for team returns 404
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/enrollment_profile", nil, http.StatusNotFound, &getResp, "team_id", fmt.Sprint(tm.ID))

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
	tm2Prof := `{"z": 1}`
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/enrollment_profile", createMDMAppleSetupAssistantRequest{
		TeamID:            &tm2.ID,
		Name:              "teamB",
		EnrollmentProfile: json.RawMessage(tm2Prof),
	}, http.StatusOK, &createResp)
	s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "teamB", "team_id": %d, "team_name": %q}`, tm2.ID, tm2.Name), 0)

	// delete that team's setup assistant
	s.Do("DELETE", "/api/latest/fleet/mdm/apple/enrollment_profile", nil, http.StatusNoContent, "team_id", fmt.Sprint(tm2.ID))
	s.lastActivityMatches(fleet.ActivityTypeDeletedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "teamB", "team_id": %d, "team_name": %q}`, tm2.ID, tm2.Name), 0)
}

// only asserts the profile identifier, status and operation (per host)
func (s *integrationMDMTestSuite) assertHostConfigProfiles(want map[*fleet.Host][]fleet.HostMDMAppleProfile) {
	t := s.T()
	ds := s.ds
	ctx := context.Background()

	for h, wantProfs := range want {
		gotProfs, err := ds.GetHostMDMAppleProfiles(ctx, h.UUID)
		require.NoError(t, err)
		require.Equal(t, len(wantProfs), len(gotProfs), "host uuid: %s", h.UUID)

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

func (s *integrationMDMTestSuite) assertWindowsConfigProfilesByName(teamID *uint, profileName string, exists bool) {
	t := s.T()
	if teamID == nil {
		teamID = ptr.Uint(0)
	}
	var cfgProfs []*fleet.MDMWindowsConfigProfile
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(context.Background(), q, &cfgProfs, `SELECT * FROM mdm_windows_configuration_profiles WHERE team_id = ?`, teamID)
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
	ff, err := writer.CreateFormFile(uploadFileField, fileName)
	require.NoError(t, err)
	_, err = io.Copy(ff, bytes.NewReader(fileContent))
	require.NoError(t, err)

	// add extra fields
	for key, values := range extraFields {
		for _, value := range values {
			err := writer.WriteField(key, value)
			require.NoError(t, err)
		}
	}

	err = writer.Close()
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

	res := s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/bootstrap", b.Bytes(), expectedStatus, headers)

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

	res := s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/setup/eula", b.Bytes(), expectedStatus, headers)

	if wantErr != "" {
		errMsg := extractServerErrorText(res.Body)
		assert.Contains(t, errMsg, wantErr)
	}
}

var testBMToken = &nanodep_client.OAuth1Tokens{
	ConsumerKey:       "test_consumer",
	ConsumerSecret:    "test_secret",
	AccessToken:       "test_access_token",
	AccessSecret:      "test_access_secret",
	AccessTokenExpiry: time.Date(2999, 1, 1, 0, 0, 0, 0, time.UTC),
}

// TestGitOpsUserActions tests the MDM permissions listed in ../../docs/Using-Fleet/Permissions.md.
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

	// Attempt to edit global MDM settings, should allow.
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)

	// Attempt to setup Apple MDM, will fail but the important thing is that it
	// fails with 422 (cannot enable end user auth because no IdP is configured)
	// and not 403 forbidden.
	s.Do("PATCH", "/api/latest/fleet/mdm/apple/setup",
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
	}, http.StatusNoContent, "team_id", strconv.Itoa(int(t1.ID)))

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
	}, http.StatusNoContent, "team_id", strconv.Itoa(int(t1.ID)))

	// Attempt to set profile batch for team t2, should not allow.
	teamProfiles = [][]byte{
		mobileconfigForTest("N7", "I7"),
		mobileconfigForTest("N8", "I8"),
	}
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{
		Profiles: teamProfiles,
	}, http.StatusForbidden, "team_id", strconv.Itoa(int(t2.ID)))
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
		SCEPChallenge: s.fleetCfg.MDM.AppleSCEPChallenge,
	})
	var lastSubmittedProfile *godep.Profile
	s.mockDEPResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	resp := s.DoRaw("GET", "/api/latest/fleet/mdm/setup/eula/"+q.Get("eula_token"), nil, http.StatusOK)
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
	var accCmd *micromdm.CommandPayload
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
	require.True(t, accCmd.Command.AccountConfiguration.LockPrimaryAccountInfo)
	require.Equal(t, "SSO User 1", accCmd.Command.AccountConfiguration.PrimaryAccountFullName)
	require.Equal(t, "sso_user", accCmd.Command.AccountConfiguration.PrimaryAccountUserName)

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
		log.NewNopLogger(),
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
		log.NewNopLogger(),
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
		log.NewNopLogger(),
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
	resp = s.DoRaw("GET", "/api/latest/fleet/mdm/setup/eula/"+q.Get("eula_token"), nil, http.StatusOK)
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

func (s *integrationMDMTestSuite) verifyEnrollmentProfile(rawProfile []byte, enrollmentRef string) *enrollmentProfile {
	t := s.T()
	var profile enrollmentProfile
	require.NoError(t, plist.Unmarshal(rawProfile, &profile))

	for _, p := range profile.PayloadContent {
		switch p.PayloadType {
		case "com.apple.security.scep":
			require.Equal(t, s.getConfig().ServerSettings.ServerURL+apple_mdm.SCEPPath, p.PayloadContent.URL)
			require.Equal(t, s.fleetCfg.MDM.AppleSCEPChallenge, p.PayloadContent.Challenge)
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
		s.mockDEPResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			}
		}))
		s.runDEPSchedule()

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
	}

	token := "token_test_migration"
	host := createOrbitEnrolledHost(t, "darwin", "h", s.ds)
	createDeviceTokenForHost(t, s.ds, host.ID, token)
	checkMigrationResponses(host, token)

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

	// the feature flag is enabled for the MDM test suite
	var acResp appConfigResponse
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.WindowsEnabledAndConfigured)

	// create a couple teams
	tm1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "1"})
	require.NoError(t, err)
	tm2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "2"})
	require.NoError(t, err)

	// create some hosts - a Windows workstation in each team and no-team,
	// Windows server in no team, Windows workstation enrolled in a 3rd-party in
	// team 2, Windows workstation already enrolled in Fleet in no team, and a
	// macOS host in no team.
	metadataHosts := []struct {
		os           string
		suffix       string
		isServer     bool
		teamID       *uint
		enrolledName string
		shouldEnroll bool
	}{
		{"windows", "win-no-team", false, nil, "", true},
		{"windows", "win-team-1", false, &tm1.ID, "", true},
		{"windows", "win-team-2", false, &tm2.ID, "", true},
		{"windows", "win-server", true, nil, "", false},                                    // is a server
		{"windows", "win-third-party", false, &tm2.ID, fleet.WellKnownMDMSimpleMDM, false}, // is enrolled in 3rd-party
		{"windows", "win-fleet", false, nil, fleet.WellKnownMDMFleet, false},               // is already Fleet-enrolled
		{"darwin", "macos-no-team", false, nil, "", false},                                 // is not Windows
	}
	hostsBySuffix := make(map[string]*fleet.Host, len(metadataHosts))
	for _, meta := range metadataHosts {
		h := createOrbitEnrolledHost(t, meta.os, meta.suffix, s.ds)
		createDeviceTokenForHost(t, s.ds, h.ID, meta.suffix)
		err := s.ds.SetOrUpdateMDMData(ctx, h.ID, meta.isServer, meta.enrolledName != "", "https://example.com", false, meta.enrolledName, "")
		require.NoError(t, err)
		if meta.teamID != nil {
			err = s.ds.AddHostsToTeam(ctx, meta.teamID, []uint{h.ID})
			require.NoError(t, err)
		}
		hostsBySuffix[meta.suffix] = h
	}

	// enable Windows MDM
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "windows_enabled_and_configured": true }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.WindowsEnabledAndConfigured)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledWindowsMDM{}.ActivityName(), `{}`, 0)

	// get the orbit config for each host, verify that only the expected ones
	// receive the "needs enrollment to Windows MDM" notification.
	for _, meta := range metadataHosts {
		var resp orbitGetConfigResponse
		s.DoJSON("POST", "/api/fleet/orbit/config",
			json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hostsBySuffix[meta.suffix].OrbitNodeKey)),
			http.StatusOK, &resp)
		require.Equal(t, meta.shouldEnroll, resp.Notifications.NeedsProgrammaticWindowsMDMEnrollment)
		require.False(t, resp.Notifications.NeedsProgrammaticWindowsMDMUnenrollment)
		if meta.shouldEnroll {
			require.Contains(t, resp.Notifications.WindowsMDMDiscoveryEndpoint, microsoft_mdm.MDE2DiscoveryPath)
		} else {
			require.Empty(t, resp.Notifications.WindowsMDMDiscoveryEndpoint)
		}
	}

	// disable Microsoft MDM
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "windows_enabled_and_configured": false }
  }`), http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.WindowsEnabledAndConfigured)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeDisabledWindowsMDM{}.ActivityName(), `{}`, 0)

	// set the win-no-team host as enrolled in Windows MDM
	noTeamHost := hostsBySuffix["win-no-team"]
	err = s.ds.SetOrUpdateMDMData(ctx, noTeamHost.ID, false, true, "https://example.com", false, fleet.WellKnownMDMFleet, "")
	require.NoError(t, err)

	// get the orbit config for win-no-team should return true for the
	// unenrollment notification
	var resp orbitGetConfigResponse
	s.DoJSON("POST", "/api/fleet/orbit/config",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *noTeamHost.OrbitNodeKey)),
		http.StatusOK, &resp)
	require.True(t, resp.Notifications.NeedsProgrammaticWindowsMDMUnenrollment)
	require.False(t, resp.Notifications.NeedsProgrammaticWindowsMDMEnrollment)
	require.Empty(t, resp.Notifications.WindowsMDMDiscoveryEndpoint)
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
		SCEPChallenge: s.fleetCfg.MDM.AppleSCEPChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	})
	mdmDevice.SerialNumber = h.HardwareSerial
	mdmDevice.UUID = h.UUID
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h.OrbitNodeKey)), http.StatusOK, &resp)
	wantCfg, err := fleet.NewNudgeConfig(fleet.MacOSUpdates{Deadline: optjson.SetString("2022-01-04"), MinimumVersion: optjson.SetString("12.1.3")})
	require.NoError(t, err)
	require.Equal(t, wantCfg, resp.NudgeConfig)
	require.Equal(t, wantCfg.OSVersionRequirements[0].RequiredInstallationDate.String(), "2022-01-04 04:00:00 +0000 UTC")

	// create a team with an empty macos_updates config
	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          4827,
		Name:        "team1_" + t.Name(),
		Description: "desc team1_" + t.Name(),
	})
	require.NoError(t, err)

	// add the host to the team
	err = s.ds.AddHostsToTeam(context.Background(), &team.ID, []uint{h.ID})
	require.NoError(t, err)

	// NudgeConfig should be empty
	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h.OrbitNodeKey)), http.StatusOK, &resp)
	require.Empty(t, resp.NudgeConfig)
	require.Equal(t, wantCfg.OSVersionRequirements[0].RequiredInstallationDate.String(), "2022-01-04 04:00:00 +0000 UTC")

	// modify the team config, add macos_updates config
	var tmResp teamResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		MDM: &fleet.TeamPayloadMDM{
			MacOSUpdates: &fleet.MacOSUpdates{
				Deadline:       optjson.SetString("1992-01-01"),
				MinimumVersion: optjson.SetString("13.1.1"),
			},
		},
	}, http.StatusOK, &tmResp)

	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h.OrbitNodeKey)), http.StatusOK, &resp)
	wantCfg, err = fleet.NewNudgeConfig(fleet.MacOSUpdates{Deadline: optjson.SetString("1992-01-01"), MinimumVersion: optjson.SetString("13.1.1")})
	require.NoError(t, err)
	require.Equal(t, wantCfg, resp.NudgeConfig)
	require.Equal(t, wantCfg.OSVersionRequirements[0].RequiredInstallationDate.String(), "1992-01-01 04:00:00 +0000 UTC")

	// create a new host, still receives the global config
	h2 := createOrbitEnrolledHost(t, "darwin", "h2", s.ds)
	mdmDevice = mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.fleetCfg.MDM.AppleSCEPChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	})
	mdmDevice.SerialNumber = h2.HardwareSerial
	mdmDevice.UUID = h2.UUID
	err = mdmDevice.Enroll()
	require.NoError(t, err)
	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *h2.OrbitNodeKey)), http.StatusOK, &resp)
	wantCfg, err = fleet.NewNudgeConfig(fleet.MacOSUpdates{Deadline: optjson.SetString("2022-01-04"), MinimumVersion: optjson.SetString("12.1.3")})
	require.NoError(t, err)
	require.Equal(t, wantCfg, resp.NudgeConfig)
	require.Equal(t, wantCfg.OSVersionRequirements[0].RequiredInstallationDate.String(), "2022-01-04 04:00:00 +0000 UTC")
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
		`{
			"mdm_platform": "microsoft",
			"host_serial": "",
			"installed_from_dep": false,
			"host_display_name": "DESKTOP-0C89RC0"
		 }`,
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
		XMLName: xml.Name{Local: mdm_types.CmdStatus},
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

	cmds, err = d.StartManagementSession()
	require.NoError(t, err)
	// two status + the two commands we enqueued
	require.Len(t, cmds, 4)
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

	// status 200 for command Two  (Get)
	d.AppendResponse(fleet.SyncMLCmd{
		XMLName: xml.Name{Local: mdm_types.CmdStatus},
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
		XMLName: xml.Name{Local: mdm_types.CmdResults},
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
		XMLName: xml.Name{Local: mdm_types.CmdStatus},
		MsgRef:  &msgID,
		CmdRef:  &cmdThreeUUID,
		Cmd:     ptr.String("Replace"),
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
	s.DoJSON("GET", "/api/latest/fleet/mdm/commandresults", nil, http.StatusOK, &getMDMCmdResp, "command_uuid", cmdOneUUID)
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

	s.DoJSON("GET", "/api/latest/fleet/mdm/commandresults", nil, http.StatusOK, &getMDMCmdResp, "command_uuid", cmdTwoUUID)
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

	s.DoJSON("GET", "/api/latest/fleet/mdm/commandresults", nil, http.StatusOK, &getMDMCmdResp, "command_uuid", cmdThreeUUID)
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

	cmds, err := d.StartManagementSession()
	require.NoError(t, err)

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
	ctx := context.Background()

	// create a Windows host enrolled in MDM
	enrolledWindows := createOrbitEnrolledHost(t, "windows", "h1", s.ds)
	deviceID := "DB257C3A08778F4FB61E2749066C1F27"
	enrolledDevice := &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            deviceID,
		MDMHardwareID:          uuid.New().String() + uuid.New().String(),
		MDMDeviceState:         uuid.New().String(),
		MDMDeviceType:          "CIMClient_Windows",
		MDMDeviceName:          "DESKTOP-1C3ARC1",
		MDMEnrollType:          "ProgrammaticEnrollment",
		MDMEnrollUserID:        "",
		MDMEnrollProtoVersion:  "5.0",
		MDMEnrollClientVersion: "10.0.19045.2965",
		MDMNotInOOBE:           false,
		HostUUID:               enrolledWindows.UUID,
	}
	err := s.ds.SetOrUpdateMDMData(ctx, enrolledWindows.ID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, "")
	require.NoError(t, err)

	err = s.ds.MDMWindowsInsertEnrolledDevice(context.Background(), enrolledDevice)
	require.NoError(t, err)
	err = s.ds.UpdateMDMWindowsEnrollmentsHostUUID(context.Background(), enrolledDevice.HostUUID, enrolledDevice.MDMDeviceID)
	require.NoError(t, err)

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
	s.DoJSON("POST", "/api/latest/fleet/mdm/commands/run", &runMDMCommandRequest{
		Command: base64.StdEncoding.EncodeToString([]byte(macRawCmd)),
	}, http.StatusNotFound, &runResp)

	// mix of mdm and non-mdm hosts
	s.DoJSON("POST", "/api/latest/fleet/mdm/commands/run", &runMDMCommandRequest{
		Command:   base64.StdEncoding.EncodeToString([]byte(macRawCmd)),
		HostUUIDs: []string{enrolledMac.UUID, unenrolledMac.UUID},
	}, http.StatusPreconditionFailed, &runResp)
	s.DoJSON("POST", "/api/latest/fleet/mdm/commands/run", &runMDMCommandRequest{
		Command:   base64.StdEncoding.EncodeToString([]byte(winRawCmd)),
		HostUUIDs: []string{enrolledWindows.UUID, unenrolledWindows.UUID},
	}, http.StatusPreconditionFailed, &runResp)

	// mix of windows and macos hosts
	s.DoJSON("POST", "/api/latest/fleet/mdm/commands/run", &runMDMCommandRequest{
		Command:   base64.StdEncoding.EncodeToString([]byte(macRawCmd)),
		HostUUIDs: []string{enrolledMac.UUID, enrolledWindows.UUID},
	}, http.StatusUnprocessableEntity, &runResp)

	// windows only, invalid command
	res := s.Do("POST", "/api/latest/fleet/mdm/commands/run", &runMDMCommandRequest{
		Command:   base64.StdEncoding.EncodeToString([]byte(macRawCmd)),
		HostUUIDs: []string{enrolledWindows.UUID},
	}, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "You can run only <Exec> command type")

	// macOS only, invalid command
	res = s.Do("POST", "/api/latest/fleet/mdm/commands/run", &runMDMCommandRequest{
		Command:   base64.StdEncoding.EncodeToString([]byte(winRawCmd)),
		HostUUIDs: []string{enrolledMac.UUID},
	}, http.StatusUnsupportedMediaType)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "unable to decode plist command")

	// valid windows
	runResp = runMDMCommandResponse{}
	s.DoJSON("POST", "/api/latest/fleet/mdm/commands/run", &runMDMCommandRequest{
		Command:   base64.StdEncoding.EncodeToString([]byte(winRawCmd)),
		HostUUIDs: []string{enrolledWindows.UUID},
	}, http.StatusOK, &runResp)
	require.NotEmpty(t, runResp.CommandUUID)
	require.Equal(t, "windows", runResp.Platform)
	require.Equal(t, "./SetValues", runResp.RequestType)

	// valid macOS
	runResp = runMDMCommandResponse{}
	s.DoJSON("POST", "/api/latest/fleet/mdm/commands/run", &runMDMCommandRequest{
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

	// mark it as enrolled in Fleet
	err := s.ds.SetOrUpdateMDMData(ctx, host.ID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, "")
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

func (s *integrationMDMTestSuite) TestMDMConfigProfileCRUD() {
	t := s.T()
	ctx := context.Background()

	testTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "TestTeam"})
	require.NoError(t, err)

	assertAppleProfile := func(filename, name, ident string, teamID uint, labelNames []string, wantStatus int, wantErrMsg string) string {
		fields := map[string][]string{
			"labels": labelNames,
		}
		if teamID > 0 {
			fields["team_id"] = []string{fmt.Sprintf("%d", teamID)}
		}
		body, headers := generateNewProfileMultipartRequest(
			t, filename, mobileconfigForTest(name, ident), s.token, fields,
		)
		res := s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/profiles", body.Bytes(), wantStatus, headers)

		if wantErrMsg != "" {
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, wantErrMsg)
			return ""
		}

		var resp newMDMConfigProfileResponse
		err := json.NewDecoder(res.Body).Decode(&resp)
		require.NoError(t, err)
		require.NotEmpty(t, resp.ProfileUUID)
		require.Equal(t, "a", string(resp.ProfileUUID[0]))
		return resp.ProfileUUID
	}
	createAppleProfile := func(name, ident string, teamID uint, labelNames []string) string {
		uid := assertAppleProfile(name+".mobileconfig", name, ident, teamID, labelNames, http.StatusOK, "")

		var wantJSON string
		if teamID == 0 {
			wantJSON = fmt.Sprintf(`{"team_id": null, "team_name": null, "profile_name": %q, "profile_identifier": %q}`, name, ident)
		} else {
			wantJSON = fmt.Sprintf(`{"team_id": %d, "team_name": %q, "profile_name": %q, "profile_identifier": %q}`, teamID, testTeam.Name, name, ident)
		}
		s.lastActivityOfTypeMatches(fleet.ActivityTypeCreatedMacosProfile{}.ActivityName(), wantJSON, 0)

		return uid
	}

	assertWindowsProfile := func(filename, locURI string, teamID uint, labelNames []string, wantStatus int, wantErrMsg string) string {
		fields := map[string][]string{
			"labels": labelNames,
		}
		if teamID > 0 {
			fields["team_id"] = []string{fmt.Sprintf("%d", teamID)}
		}
		body, headers := generateNewProfileMultipartRequest(
			t,
			filename,
			[]byte(fmt.Sprintf(`<Replace><Item><Target><LocURI>%s</LocURI></Target></Item></Replace>`, locURI)),
			s.token,
			fields,
		)
		res := s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/profiles", body.Bytes(), wantStatus, headers)

		if wantErrMsg != "" {
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, wantErrMsg)
			return ""
		}

		var resp newMDMConfigProfileResponse
		err := json.NewDecoder(res.Body).Decode(&resp)
		require.NoError(t, err)
		require.NotEmpty(t, resp.ProfileUUID)
		require.Equal(t, "w", string(resp.ProfileUUID[0]))
		return resp.ProfileUUID
	}
	createWindowsProfile := func(name string, teamID uint, labels []string) string {
		uid := assertWindowsProfile(name+".xml", "./Test", teamID, labels, http.StatusOK, "")

		var wantJSON string
		if teamID == 0 {
			wantJSON = fmt.Sprintf(`{"team_id": null, "team_name": null, "profile_name": %q}`, name)
		} else {
			wantJSON = fmt.Sprintf(`{"team_id": %d, "team_name": %q, "profile_name": %q}`, teamID, testTeam.Name, name)
		}
		s.lastActivityOfTypeMatches(fleet.ActivityTypeCreatedWindowsProfile{}.ActivityName(), wantJSON, 0)

		return uid
	}

	// create a couple Apple profiles for no-team and team
	noTeamAppleProfUUID := createAppleProfile("apple-global-profile", "test-global-ident", 0, nil)
	teamAppleProfUUID := createAppleProfile("apple-team-profile", "test-team-ident", testTeam.ID, nil)
	// create a couple Windows profiles for no-team and team
	noTeamWinProfUUID := createWindowsProfile("win-global-profile", 0, nil)
	teamWinProfUUID := createWindowsProfile("win-team-profile", testTeam.ID, nil)

	// Windows profile name conflicts with Apple's for no team
	assertWindowsProfile("apple-global-profile.xml", "./Test", 0, nil, http.StatusConflict, "Couldn't upload. A configuration profile with this name already exists.")
	// but no conflict for team 1
	assertWindowsProfile("apple-global-profile.xml", "./Test", testTeam.ID, nil, http.StatusOK, "")
	// Apple profile name conflicts with Windows' for no team
	assertAppleProfile("win-global-profile.mobileconfig", "win-global-profile", "test-global-ident-2", 0, nil, http.StatusConflict, "Couldn't upload. A configuration profile with this name already exists.")
	// but no conflict for team 1
	assertAppleProfile("win-global-profile.mobileconfig", "win-global-profile", "test-global-ident-2", testTeam.ID, nil, http.StatusOK, "")
	// Windows profile name conflicts with Apple's for team 1
	assertWindowsProfile("apple-team-profile.xml", "./Test", testTeam.ID, nil, http.StatusConflict, "Couldn't upload. A configuration profile with this name already exists.")
	// but no conflict for no-team
	assertWindowsProfile("apple-team-profile.xml", "./Test", 0, nil, http.StatusOK, "")
	// Apple profile name conflicts with Windows' for team 1
	assertAppleProfile("win-team-profile.mobileconfig", "win-team-profile", "test-team-ident-2", testTeam.ID, nil, http.StatusConflict, "Couldn't upload. A configuration profile with this name already exists.")
	// but no conflict for no-team
	assertAppleProfile("win-team-profile.mobileconfig", "win-team-profile", "test-team-ident-2", 0, nil, http.StatusOK, "")

	// not an xml nor mobileconfig file
	assertWindowsProfile("foo.txt", "./Test", 0, nil, http.StatusBadRequest, "Couldn't upload. The file should be a .mobileconfig or .xml file.")
	assertAppleProfile("foo.txt", "foo", "foo-ident", 0, nil, http.StatusBadRequest, "Couldn't upload. The file should be a .mobileconfig or .xml file.")

	// Windows-reserved LocURI
	assertWindowsProfile("bitlocker.xml", syncml.FleetBitLockerTargetLocURI, 0, nil, http.StatusBadRequest, "Couldn't upload. Custom configuration profiles can't include BitLocker settings.")
	assertWindowsProfile("updates.xml", syncml.FleetOSUpdateTargetLocURI, testTeam.ID, nil, http.StatusBadRequest, "Couldn't upload. Custom configuration profiles can't include Windows updates settings.")

	// Fleet-reserved profiles
	for name := range servermdm.FleetReservedProfileNames() {
		assertAppleProfile(name+".mobileconfig", name, name+"-ident", 0, nil, http.StatusBadRequest, fmt.Sprintf(`name %s is not allowed`, name))
		assertWindowsProfile(name+".xml", "./Test", 0, nil, http.StatusBadRequest, fmt.Sprintf(`Couldn't upload. Profile name %q is not allowed.`, name))
	}

	// profiles with non-existent labels
	assertAppleProfile("apple-profile-with-labels.mobileconfig", "apple-profile-with-labels", "ident-with-labels", 0, []string{"does-not-exist"}, http.StatusBadRequest, "some or all the labels provided don't exist")
	assertWindowsProfile("win-profile-with-labels.xml", "./Test", 0, []string{"does-not-exist"}, http.StatusBadRequest, "some or all the labels provided don't exist")

	// create a couple of labels
	labelFoo := &fleet.Label{Name: "foo", Query: "select * from foo;"}
	labelFoo, err = s.ds.NewLabel(context.Background(), labelFoo)
	require.NoError(t, err)
	labelBar := &fleet.Label{Name: "bar", Query: "select * from bar;"}
	labelBar, err = s.ds.NewLabel(context.Background(), labelBar)
	require.NoError(t, err)

	// profiles mixing existent and non-existent labels
	assertAppleProfile("apple-profile-with-labels.mobileconfig", "apple-profile-with-labels", "ident-with-labels", 0, []string{"does-not-exist", "foo"}, http.StatusBadRequest, "some or all the labels provided don't exist")
	assertWindowsProfile("win-profile-with-labels.xml", "./Test", 0, []string{"does-not-exist", "bar"}, http.StatusBadRequest, "some or all the labels provided don't exist")

	// profiles with valid labels
	uuidAppleWithLabel := assertAppleProfile("apple-profile-with-labels.mobileconfig", "apple-profile-with-labels", "ident-with-labels", 0, []string{"foo"}, http.StatusOK, "")
	uuidWindowsWithLabel := assertWindowsProfile("win-profile-with-labels.xml", "./Test", 0, []string{"foo", "bar"}, http.StatusOK, "")

	// verify that the label associations have been created
	// TODO: update when we have datastore methods to get this data
	var profileLabels []fleet.ConfigurationProfileLabel
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `SELECT COALESCE(apple_profile_uuid, windows_profile_uuid) as profile_uuid, label_name, label_id FROM mdm_configuration_profile_labels`
		return sqlx.SelectContext(context.Background(), q, &profileLabels, stmt)
	})

	require.NotEmpty(t, profileLabels)
	require.Len(t, profileLabels, 3)
	require.ElementsMatch(
		t,
		[]fleet.ConfigurationProfileLabel{
			{ProfileUUID: uuidAppleWithLabel, LabelName: labelFoo.Name, LabelID: labelFoo.ID},
			{ProfileUUID: uuidWindowsWithLabel, LabelName: labelFoo.Name, LabelID: labelFoo.ID},
			{ProfileUUID: uuidWindowsWithLabel, LabelName: labelBar.Name, LabelID: labelBar.ID},
		},
		profileLabels,
	)

	// Windows invalid content
	body, headers := generateNewProfileMultipartRequest(t, "win.xml", []byte("\x00\x01\x02"), s.token, nil)
	res := s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/profiles", body.Bytes(), http.StatusBadRequest, headers)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't upload. The file should include valid XML:")

	// Apple invalid content
	body, headers = generateNewProfileMultipartRequest(t,
		"apple.mobileconfig", []byte("\x00\x01\x02"), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/profiles", body.Bytes(), http.StatusBadRequest, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "mobileconfig is not XML nor PKCS7 parseable")

	// get the existing profiles work
	expectedProfiles := []fleet.MDMConfigProfilePayload{
		{ProfileUUID: noTeamAppleProfUUID, Platform: "darwin", Name: "apple-global-profile", Identifier: "test-global-ident", TeamID: nil},
		{ProfileUUID: teamAppleProfUUID, Platform: "darwin", Name: "apple-team-profile", Identifier: "test-team-ident", TeamID: &testTeam.ID},
		{ProfileUUID: noTeamWinProfUUID, Platform: "windows", Name: "win-global-profile", TeamID: nil},
		{ProfileUUID: teamWinProfUUID, Platform: "windows", Name: "win-team-profile", TeamID: &testTeam.ID},
	}
	for _, prof := range expectedProfiles {
		var getResp getMDMConfigProfileResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/mdm/profiles/%s", prof.ProfileUUID), nil, http.StatusOK, &getResp)
		require.NotZero(t, getResp.CreatedAt)
		require.NotZero(t, getResp.UploadedAt)
		if getResp.Platform == "darwin" {
			require.Len(t, getResp.Checksum, 16)
		} else {
			require.Empty(t, getResp.Checksum)
		}
		getResp.CreatedAt, getResp.UploadedAt = time.Time{}, time.Time{}
		getResp.Checksum = nil
		require.Equal(t, prof, *getResp.MDMConfigProfilePayload)

		resp := s.Do("GET", fmt.Sprintf("/api/latest/fleet/mdm/profiles/%s", prof.ProfileUUID), nil, http.StatusOK, "alt", "media")
		require.NotZero(t, resp.ContentLength)
		require.Contains(t, resp.Header.Get("Content-Disposition"), "attachment;")
		if getResp.Platform == "darwin" {
			require.Contains(t, resp.Header.Get("Content-Type"), "application/x-apple-aspen-config")
		} else {
			require.Contains(t, resp.Header.Get("Content-Type"), "application/octet-stream")
		}
		require.Contains(t, resp.Header.Get("X-Content-Type-Options"), "nosniff")

		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, resp.ContentLength, int64(len(b)))
	}

	var getResp getMDMConfigProfileResponse
	// get an unknown Apple profile
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/mdm/profiles/%s", "ano-such-profile"), nil, http.StatusNotFound, &getResp)
	s.Do("GET", fmt.Sprintf("/api/latest/fleet/mdm/profiles/%s", "ano-such-profile"), nil, http.StatusNotFound, "alt", "media")
	// get an unknown Windows profile
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/mdm/profiles/%s", "wno-such-profile"), nil, http.StatusNotFound, &getResp)
	s.Do("GET", fmt.Sprintf("/api/latest/fleet/mdm/profiles/%s", "wno-such-profile"), nil, http.StatusNotFound, "alt", "media")

	var deleteResp deleteMDMConfigProfileResponse
	// delete existing Apple profiles
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/mdm/profiles/%s", noTeamAppleProfUUID), nil, http.StatusOK, &deleteResp)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/mdm/profiles/%s", teamAppleProfUUID), nil, http.StatusOK, &deleteResp)
	// delete non-existing Apple profile
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/mdm/profiles/%s", "ano-such-profile"), nil, http.StatusNotFound, &deleteResp)
	// delete existing Windows profiles
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/mdm/profiles/%s", noTeamWinProfUUID), nil, http.StatusOK, &deleteResp)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/mdm/profiles/%s", teamWinProfUUID), nil, http.StatusOK, &deleteResp)
	// delete non-existing Windows profile
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/mdm/profiles/%s", "wno-such-profile"), nil, http.StatusNotFound, &deleteResp)

	// trying to create/delete profiles managed by Fleet fails
	for p := range mobileconfig.FleetPayloadIdentifiers() {
		assertAppleProfile("foo.mobileconfig", p, p, 0, nil, http.StatusBadRequest, fmt.Sprintf("payload identifier %s is not allowed", p))

		// create it directly in the DB to test deletion
		uid := "a" + uuid.NewString()
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			mc := mcBytesForTest(p, p, uuid.New().String())
			_, err := q.ExecContext(ctx,
				"INSERT INTO mdm_apple_configuration_profiles (profile_uuid, identifier, name, mobileconfig, checksum, team_id, uploaded_at) VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP())",
				uid, p, p, mc, "1234", 0)
			return err
		})

		var deleteResp deleteMDMConfigProfileResponse
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/mdm/profiles/%s", uid), nil, http.StatusBadRequest, &deleteResp)

		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				"DELETE FROM mdm_apple_configuration_profiles WHERE profile_uuid = ?",
				uid)
			return err
		})
	}

	// make fleet add a FileVault profile
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	profile := s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// try to delete the profile
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/mdm/profiles/%s", profile.ProfileUUID), nil, http.StatusBadRequest, &deleteResp)

	// make fleet add a Windows OS Updates profile
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "windows_updates": {"deadline_days": 1, "grace_period_days": 1} }
  }`), http.StatusOK, &acResp)
	profUUID := checkWindowsOSUpdatesProfile(t, s.ds, nil, &fleet.WindowsUpdates{DeadlineDays: optjson.SetInt(1), GracePeriodDays: optjson.SetInt(1)})

	// try to delete the profile
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/mdm/profiles/%s", profUUID), nil, http.StatusBadRequest, &deleteResp)
}

func (s *integrationMDMTestSuite) TestListMDMConfigProfiles() {
	t := s.T()
	ctx := context.Background()

	// create some teams
	tm1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	tm2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	tm3, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team3"})
	require.NoError(t, err)

	// create 5 profiles for no team and team 1, names are A, B, C ... for global and
	// tA, tB, tC ... for team 1. Alternate macOS and Windows profiles.
	for i := 0; i < 5; i++ {
		name := string('A' + byte(i))
		if i%2 == 0 {
			prof, err := fleet.NewMDMAppleConfigProfile(mcBytesForTest(name, name+".identifier", name+".uuid"), nil)
			require.NoError(t, err)
			_, err = s.ds.NewMDMAppleConfigProfile(ctx, *prof)
			require.NoError(t, err)

			tprof, err := fleet.NewMDMAppleConfigProfile(mcBytesForTest("t"+name, "t"+name+".identifier", "t"+name+".uuid"), nil)
			require.NoError(t, err)
			tprof.TeamID = &tm1.ID
			_, err = s.ds.NewMDMAppleConfigProfile(ctx, *tprof)
			require.NoError(t, err)
		} else {
			_, err = s.ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: name, SyncML: []byte(`<Replace></Replace>`)})
			require.NoError(t, err)
			_, err = s.ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: "t" + name, TeamID: &tm1.ID, SyncML: []byte(`<Replace></Replace>`)})
			require.NoError(t, err)
		}
	}

	// create a couple profiles (Win and mac) for team 2, and none for team 3
	tprof, err := fleet.NewMDMAppleConfigProfile(mcBytesForTest("tF", "tF.identifier", "tF.uuid"), nil)
	require.NoError(t, err)
	tprof.TeamID = &tm2.ID
	tm2ProfF, err := s.ds.NewMDMAppleConfigProfile(ctx, *tprof)
	require.NoError(t, err)
	// checksum is not returned by New..., so compute it manually
	checkSum := md5.Sum(tm2ProfF.Mobileconfig) // nolint:gosec // used only for test
	tm2ProfF.Checksum = checkSum[:]

	// make tm2ProfG a label-based profile
	lblFoo, err := s.ds.NewLabel(ctx, &fleet.Label{Name: "foo", Query: "select 1"})
	require.NoError(t, err)
	lblBar, err := s.ds.NewLabel(ctx, &fleet.Label{Name: "bar", Query: "select 1"})
	require.NoError(t, err)

	tm2ProfG, err := s.ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{
		Name:   "tG",
		TeamID: &tm2.ID,
		SyncML: []byte(`<Replace></Replace>`),
		Labels: []mdm_types.ConfigurationProfileLabel{
			{LabelID: lblFoo.ID, LabelName: lblFoo.Name},
			{LabelID: lblBar.ID, LabelName: lblBar.Name},
		},
	})
	require.NoError(t, err)
	// break lblFoo by deleting it
	require.NoError(t, s.ds.DeleteLabel(ctx, lblFoo.Name))

	// test that all fields are correctly returned with team 2
	var listResp listMDMConfigProfilesResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/profiles", nil, http.StatusOK, &listResp, "team_id", fmt.Sprint(tm2.ID))
	require.Len(t, listResp.Profiles, 2)
	require.NotZero(t, listResp.Profiles[0].CreatedAt)
	require.NotZero(t, listResp.Profiles[0].UploadedAt)
	require.NotZero(t, listResp.Profiles[1].CreatedAt)
	require.NotZero(t, listResp.Profiles[1].UploadedAt)
	listResp.Profiles[0].CreatedAt, listResp.Profiles[0].UploadedAt = time.Time{}, time.Time{}
	listResp.Profiles[1].CreatedAt, listResp.Profiles[1].UploadedAt = time.Time{}, time.Time{}
	require.Equal(t, &fleet.MDMConfigProfilePayload{
		ProfileUUID: tm2ProfF.ProfileUUID,
		TeamID:      tm2ProfF.TeamID,
		Name:        tm2ProfF.Name,
		Platform:    "darwin",
		Identifier:  tm2ProfF.Identifier,
		Checksum:    tm2ProfF.Checksum,
		Labels:      nil,
	}, listResp.Profiles[0])
	require.Equal(t, &fleet.MDMConfigProfilePayload{
		ProfileUUID: tm2ProfG.ProfileUUID,
		TeamID:      tm2ProfG.TeamID,
		Name:        tm2ProfG.Name,
		Platform:    "windows",
		// labels are ordered by name
		Labels: []mdm_types.ConfigurationProfileLabel{
			{LabelID: lblBar.ID, LabelName: lblBar.Name},
			{LabelID: 0, LabelName: lblFoo.Name, Broken: true},
		},
	}, listResp.Profiles[1])

	// get the specific label-based profile returns the information
	var getProfResp getMDMConfigProfileResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/profiles/"+tm2ProfG.ProfileUUID, nil, http.StatusOK, &getProfResp)
	getProfResp.CreatedAt, getProfResp.UploadedAt = time.Time{}, time.Time{}
	require.Equal(t, &fleet.MDMConfigProfilePayload{
		ProfileUUID: tm2ProfG.ProfileUUID,
		TeamID:      tm2ProfG.TeamID,
		Name:        tm2ProfG.Name,
		Platform:    "windows",
		// labels are ordered by name
		Labels: []mdm_types.ConfigurationProfileLabel{
			{LabelID: lblBar.ID, LabelName: lblBar.Name},
			{LabelID: 0, LabelName: lblFoo.Name, Broken: true},
		},
	}, getProfResp.MDMConfigProfilePayload)

	// get the non label-based profile returns no labels
	getProfResp = getMDMConfigProfileResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/profiles/"+tm2ProfF.ProfileUUID, nil, http.StatusOK, &getProfResp)
	getProfResp.CreatedAt, getProfResp.UploadedAt = time.Time{}, time.Time{}
	require.Equal(t, &fleet.MDMConfigProfilePayload{
		ProfileUUID: tm2ProfF.ProfileUUID,
		TeamID:      tm2ProfF.TeamID,
		Name:        tm2ProfF.Name,
		Platform:    "darwin",
		Identifier:  tm2ProfF.Identifier,
		Checksum:    tm2ProfF.Checksum,
		Labels:      nil,
	}, getProfResp.MDMConfigProfilePayload)

	// list for a non-existing team returns 404
	s.DoJSON("GET", "/api/latest/fleet/mdm/profiles", nil, http.StatusNotFound, &listResp, "team_id", "99999")

	cases := []struct {
		queries   []string // alternate query name and value
		teamID    *uint
		wantNames []string
		wantMeta  *fleet.PaginationMetadata
	}{
		{
			wantNames: []string{"A", "B", "C", "D", "E"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false},
		},
		{
			queries:   []string{"per_page", "2"},
			wantNames: []string{"A", "B"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false},
		},
		{
			queries:   []string{"per_page", "2", "page", "1"},
			wantNames: []string{"C", "D"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "2", "page", "2"},
			wantNames: []string{"E"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "3"},
			teamID:    &tm1.ID,
			wantNames: []string{"tA", "tB", "tC"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false},
		},
		{
			queries:   []string{"per_page", "3", "page", "1"},
			teamID:    &tm1.ID,
			wantNames: []string{"tD", "tE"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "3", "page", "2"},
			teamID:    &tm1.ID,
			wantNames: nil,
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "3"},
			teamID:    &tm2.ID,
			wantNames: []string{"tF", "tG"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false},
		},
		{
			queries:   []string{"per_page", "2"},
			teamID:    &tm3.ID,
			wantNames: nil,
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false},
		},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%v: %#v", c.teamID, c.queries), func(t *testing.T) {
			var listResp listMDMConfigProfilesResponse
			queryArgs := c.queries
			if c.teamID != nil {
				queryArgs = append(queryArgs, "team_id", fmt.Sprint(*c.teamID))
			}
			s.DoJSON("GET", "/api/latest/fleet/mdm/profiles", nil, http.StatusOK, &listResp, queryArgs...)

			require.Equal(t, len(c.wantNames), len(listResp.Profiles))
			require.Equal(t, c.wantMeta, listResp.Meta)

			var gotNames []string
			if len(listResp.Profiles) > 0 {
				gotNames = make([]string, len(listResp.Profiles))
				for i, p := range listResp.Profiles {
					gotNames[i] = p.Name
					if p.Name == "tG" {
						require.Len(t, p.Labels, 2)
					} else {
						require.Nil(t, p.Labels)
					}
					if c.teamID == nil {
						// we set it to 0 for global
						require.NotNil(t, p.TeamID)
						require.Zero(t, *p.TeamID)
					} else {
						require.NotNil(t, p.TeamID)
						require.Equal(t, *c.teamID, *p.TeamID)
					}
					require.NotEmpty(t, p.Platform)
				}
			}
			require.Equal(t, c.wantNames, gotNames)
		})
	}
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
		}))

		// add new setup assistant
		_, err := s.ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{
			TeamID:      teamID,
			Name:        "bar",
			ProfileUUID: uuid.New().String(),
			Profile:     []byte("{}"),
		})
		require.NoError(t, err)
	}

	// TODO: Some global MDM config settings don't have MDMEnabledAndConfigured or
	// WindowsMDMEnabledAndConfigured validations currently. Either add validations
	// and test them or test abscence of validation.
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

			// disabling disk encryption doesn't cause validation error because Windows is still enabled
			ac.MDM.EnableDiskEncryption = optjson.SetBool(false)
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, true)                           // only windows mdm enabled
			require.False(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value) // disabled
			require.Equal(t, "f1337", acResp.AppConfig.OrgInfo.OrgName)

			// enabling disk encryption doesn't cause validation error because Windows is still enabled
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

			// changing unrelated config doesn't cause validation error
			ac.OrgInfo.OrgName = "f1338"
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, false)                         // both mac and windows mdm disabled
			require.True(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value) // no change
			require.Equal(t, "f1338", acResp.AppConfig.OrgInfo.OrgName)

			// changing MDM config doesn't cause validation error when switching to default values
			ac.MDM.EnableDiskEncryption = optjson.SetBool(false)
			// TODO: Should it be ok to disable disk encryption when MDM is disabled?
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusOK, &acResp)
			acResp = checkAppConfig(t, false, false)                          // both mac and windows mdm disabled
			require.False(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value) // changed to disabled

			// changing MDM config does cause validation error when switching to non-default vailes
			ac.MDM.EnableDiskEncryption = optjson.SetBool(true)
			s.DoJSON("PATCH", "/api/latest/fleet/config", ac, http.StatusUnprocessableEntity, &acResp)
			acResp = checkAppConfig(t, false, false)                          // both mac and windows mdm disabled
			require.False(t, acResp.AppConfig.MDM.EnableDiskEncryption.Value) // still disabled
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
			var wantDiskEncryption bool
			var wantMacOSSetup fleet.MacOSSetup
			if checkMDM != nil {
				wantMacOSSetup = checkMDM.MacOSSetup
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
				// disk encryption requires mdm enabled and configured
				http.StatusUnprocessableEntity,
			},
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
	pending, err := s.ds.GetQueuedJobs(context.Background(), 1)
	require.NoError(s.T(), err)
	require.Empty(s.T(), pending)
}

func (s *integrationMDMTestSuite) runDEPSchedule() {
	ch := make(chan bool)
	s.onDEPScheduleDone = func() { close(ch) }
	_, err := s.depSchedule.Trigger()
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
		require.Equal(t, expectedSummary.Failed, apple.Failed)
		require.Equal(t, expectedSummary.Pending, apple.Pending)
		require.Equal(t, expectedSummary.Verifying, apple.Verifying)
		require.Equal(t, expectedSummary.Verified, apple.Verified)
	}

	var combined getMDMProfilesSummaryResponse
	s.DoJSON("GET", "/api/v1/fleet/mdm/profiles/summary", getMDMProfilesSummaryRequest{}, http.StatusOK, &combined, queryParams...)
	require.Equal(t, expectedSummary.Failed, combined.Failed)
	require.Equal(t, expectedSummary.Pending, combined.Pending)
	require.Equal(t, expectedSummary.Verifying, combined.Verifying)
	require.Equal(t, expectedSummary.Verified, combined.Verified)
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
	s.DoJSON("GET", "/api/v1/fleet/mdm/disk_encryption/summary", getMDMProfilesSummaryRequest{}, http.StatusOK, &combined, queryParams...)
	require.Equal(t, expectedSummary.Failed, combined.Failed)
	require.Equal(t, expectedSummary.Enforcing, combined.Enforcing)
	require.Equal(t, expectedSummary.ActionRequired, combined.ActionRequired)
	require.Equal(t, expectedSummary.Verifying, combined.Verifying)
	require.Equal(t, expectedSummary.Verified, combined.Verified)
	require.Equal(t, expectedSummary.RemovingEnforcement, combined.RemovingEnforcement)
}

func (s *integrationMDMTestSuite) TestWindowsProfileManagement() {
	t := s.T()
	ctx := context.Background()

	err := s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: t.Name()}})
	require.NoError(t, err)

	globalProfiles := []string{
		mysql.InsertWindowsProfileForTest(t, s.ds, 0),
		mysql.InsertWindowsProfileForTest(t, s.ds, 0),
		mysql.InsertWindowsProfileForTest(t, s.ds, 0),
	}

	// create a new team
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "batch_set_mdm_profiles"})
	require.NoError(t, err)
	teamProfiles := []string{
		mysql.InsertWindowsProfileForTest(t, s.ds, tm.ID),
		mysql.InsertWindowsProfileForTest(t, s.ds, tm.ID),
	}

	// create a non-Windows host
	_, err = s.ds.NewHost(context.Background(), &fleet.Host{
		ID:            1,
		OsqueryHostID: ptr.String("non-windows-host"),
		NodeKey:       ptr.String("non-windows-host"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.non.windows", t.Name()),
		Platform:      "darwin",
	})
	require.NoError(t, err)

	// create a Windows host that's not enrolled into MDM
	_, err = s.ds.NewHost(context.Background(), &fleet.Host{
		ID:            2,
		OsqueryHostID: ptr.String("not-mdm-enrolled"),
		NodeKey:       ptr.String("not-mdm-enrolled"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.not.enrolled", t.Name()),
		Platform:      "windows",
	})
	require.NoError(t, err)

	verifyHostProfileStatus := func(cmds []fleet.ProtoCmdOperation, wantStatus string) {
		for _, cmd := range cmds {
			var gotProfile struct {
				Status  string `db:"status"`
				Retries int    `db:"retries"`
			}
			mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				stmt := `
				SELECT COALESCE(status, 'pending') as status, retries
				FROM host_mdm_windows_profiles
				WHERE command_uuid = ?`
				return sqlx.GetContext(context.Background(), q, &gotProfile, stmt, cmd.Cmd.CmdID.Value)
			})

			wantDeliveryStatus := fleet.WindowsResponseToDeliveryStatus(wantStatus)
			if gotProfile.Retries <= servermdm.MaxProfileRetries && wantDeliveryStatus == mdm_types.MDMDeliveryFailed {
				require.EqualValues(t, "pending", gotProfile.Status, "command_uuid", cmd.Cmd.CmdID.Value)
			} else {
				require.EqualValues(t, wantDeliveryStatus, gotProfile.Status, "command_uuid", cmd.Cmd.CmdID.Value)
			}
		}
	}

	verifyProfiles := func(device *mdmtest.TestWindowsMDMClient, n int, fail bool) {
		mdmResponseStatus := syncml.CmdStatusOK
		if fail {
			mdmResponseStatus = syncml.CmdStatusAtomicFailed
		}
		s.awaitTriggerProfileSchedule(t)
		cmds, err := device.StartManagementSession()
		require.NoError(t, err)
		// 2 Status + n profiles
		require.Len(t, cmds, n+2)

		var atomicCmds []fleet.ProtoCmdOperation
		msgID, err := device.GetCurrentMsgID()
		require.NoError(t, err)
		for _, c := range cmds {
			cmdID := c.Cmd.CmdID
			status := syncml.CmdStatusOK
			if c.Verb == "Atomic" {
				atomicCmds = append(atomicCmds, c)
				status = mdmResponseStatus
				require.NotEmpty(t, c.Cmd.ReplaceCommands)
				for _, rc := range c.Cmd.ReplaceCommands {
					require.NotEmpty(t, rc.CmdID)
				}
			}
			device.AppendResponse(fleet.SyncMLCmd{
				XMLName: xml.Name{Local: mdm_types.CmdStatus},
				MsgRef:  &msgID,
				CmdRef:  &cmdID.Value,
				Cmd:     ptr.String(c.Verb),
				Data:    &status,
				Items:   nil,
				CmdID:   fleet.CmdID{Value: uuid.NewString()},
			})
		}
		// TODO: verify profile contents as well
		require.Len(t, atomicCmds, n)

		// before we send the response, commands should be "pending"
		verifyHostProfileStatus(atomicCmds, "")

		cmds, err = device.SendResponse()
		require.NoError(t, err)
		// the ack of the message should be the only returned command
		require.Len(t, cmds, 1)

		// verify that we updated status in the db
		verifyHostProfileStatus(atomicCmds, mdmResponseStatus)
	}

	checkHostsProfilesMatch := func(host *fleet.Host, wantUUIDs []string) {
		var gotUUIDs []string
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			stmt := `SELECT profile_uuid FROM host_mdm_windows_profiles WHERE host_uuid = ?`
			return sqlx.SelectContext(context.Background(), q, &gotUUIDs, stmt, host.UUID)
		})
		require.ElementsMatch(t, wantUUIDs, gotUUIDs)
	}

	checkHostDetails := func(t *testing.T, host *fleet.Host, wantProfs []string, wantStatus fleet.MDMDeliveryStatus) {
		var gotHostResp getHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", host.ID), nil, http.StatusOK, &gotHostResp)
		require.NotNil(t, gotHostResp.Host.MDM.Profiles)
		var gotProfs []string
		require.Len(t, *gotHostResp.Host.MDM.Profiles, len(wantProfs))
		for _, p := range *gotHostResp.Host.MDM.Profiles {
			gotProfs = append(gotProfs, strings.Replace(p.Name, "name-", "", 1))
			require.NotNil(t, p.Status)
			require.Equal(t, wantStatus, *p.Status, "profile", p.Name)
			require.Equal(t, "windows", p.Platform)
			// Fleet reserved profiles (e.g., OS updates) should be screened from the host details response
			require.NotContains(t, servermdm.ListFleetReservedWindowsProfileNames(), p.Name)
		}
		require.ElementsMatch(t, wantProfs, gotProfs)
	}

	checkHostsFilteredByOSSettingsStatus := func(t *testing.T, wantHosts []string, wantStatus fleet.MDMDeliveryStatus, teamID *uint, labels ...*fleet.Label) {
		var teamFilter string
		if teamID != nil {
			teamFilter = fmt.Sprintf("&team_id=%d", *teamID)
		}
		var gotHostsResp listHostsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts?os_settings=%s%s", wantStatus, teamFilter), nil, http.StatusOK, &gotHostsResp)
		require.NotNil(t, gotHostsResp.Hosts)
		var gotHosts []string
		for _, h := range gotHostsResp.Hosts {
			gotHosts = append(gotHosts, h.Hostname)
		}
		require.ElementsMatch(t, wantHosts, gotHosts)

		var countHostsResp countHostsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/count?os_settings=%s%s", wantStatus, teamFilter), nil, http.StatusOK, &countHostsResp)
		require.Equal(t, len(wantHosts), countHostsResp.Count)

		for _, l := range labels {
			gotHostsResp = listHostsResponse{}
			s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/labels/%d/hosts?os_settings=%s%s", l.ID, wantStatus, teamFilter), nil, http.StatusOK, &gotHostsResp)
			require.NotNil(t, gotHostsResp.Hosts)
			gotHosts = []string{}
			for _, h := range gotHostsResp.Hosts {
				gotHosts = append(gotHosts, h.Hostname)
			}
			require.ElementsMatch(t, wantHosts, gotHosts, "label", l.Name)

			countHostsResp = countHostsResponse{}
			s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/count?label_id=%d&os_settings=%s%s", l.ID, wantStatus, teamFilter), nil, http.StatusOK, &countHostsResp)
		}
	}

	getProfileUUID := func(t *testing.T, profName string, teamID *uint) string {
		var profUUID string
		mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
			var globalOrTeamID uint
			if teamID != nil {
				globalOrTeamID = *teamID
			}
			return sqlx.GetContext(ctx, tx, &profUUID, `SELECT profile_uuid FROM mdm_windows_configuration_profiles WHERE team_id = ? AND name = ?`, globalOrTeamID, profName)
		})
		require.NotNil(t, profUUID)
		return profUUID
	}

	checkHostProfileStatus := func(t *testing.T, hostUUID string, profUUID string, wantStatus fleet.MDMDeliveryStatus) {
		var gotStatus fleet.MDMDeliveryStatus
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			stmt := `SELECT status FROM host_mdm_windows_profiles WHERE host_uuid = ? AND profile_uuid = ?`
			err := sqlx.GetContext(context.Background(), q, &gotStatus, stmt, hostUUID, profUUID)
			return err
		})
		require.Equal(t, wantStatus, gotStatus)
	}

	// Create a host and then enroll to MDM.
	host, mdmDevice := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
	// trigger a profile sync
	verifyProfiles(mdmDevice, 3, false)
	checkHostsProfilesMatch(host, globalProfiles)
	checkHostDetails(t, host, globalProfiles, fleet.MDMDeliveryVerifying)

	// create new label that includes host
	label := &fleet.Label{
		Name:  t.Name() + "foo",
		Query: "select * from foo;",
	}
	label, err = s.ds.NewLabel(context.Background(), label)
	require.NoError(t, err)
	require.NoError(t, s.ds.RecordLabelQueryExecutions(ctx, host, map[uint]*bool{label.ID: ptr.Bool(true)}, time.Now(), false))

	// simulate osquery reporting host mdm details (host_mdm.enrolled = 1 is condition for
	// hosts filtering by os settings status and generating mdm profiles summaries)
	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, host.ID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, ""))
	checkHostsFilteredByOSSettingsStatus(t, []string{host.Hostname}, fleet.MDMDeliveryVerifying, nil, label)
	s.checkMDMProfilesSummaries(t, nil, fleet.MDMProfilesSummary{
		Verifying: 1,
	}, nil)

	// another sync shouldn't return profiles
	verifyProfiles(mdmDevice, 0, false)

	// make fleet add a Windows OS Updates profile
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"mdm": { "windows_updates": {"deadline_days": 1, "grace_period_days": 1} }}`), http.StatusOK, &acResp)
	osUpdatesProf := getProfileUUID(t, servermdm.FleetWindowsOSUpdatesProfileName, nil)

	// os updates is sent via a profiles commands
	verifyProfiles(mdmDevice, 1, false)
	checkHostsProfilesMatch(host, append(globalProfiles, osUpdatesProf))
	// but is hidden from host details response
	checkHostDetails(t, host, globalProfiles, fleet.MDMDeliveryVerifying)

	// os updates profile status doesn't matter for filtered hosts results or summaries
	checkHostProfileStatus(t, host.UUID, osUpdatesProf, fleet.MDMDeliveryVerifying)
	checkHostsFilteredByOSSettingsStatus(t, []string{host.Hostname}, fleet.MDMDeliveryVerifying, nil, label)
	s.checkMDMProfilesSummaries(t, nil, fleet.MDMProfilesSummary{
		Verifying: 1,
	}, nil)
	// force os updates profile to failed, doesn't impact filtered hosts results or summaries
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `UPDATE host_mdm_windows_profiles SET status = 'failed' WHERE profile_uuid = ?`
		_, err := q.ExecContext(context.Background(), stmt, osUpdatesProf)
		return err
	})
	checkHostProfileStatus(t, host.UUID, osUpdatesProf, fleet.MDMDeliveryFailed)
	checkHostsFilteredByOSSettingsStatus(t, []string{host.Hostname}, fleet.MDMDeliveryVerifying, nil, label)
	s.checkMDMProfilesSummaries(t, nil, fleet.MDMProfilesSummary{
		Verifying: 1,
	}, nil)
	// force another profile to failed, does impact filtered hosts results and summaries
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `UPDATE host_mdm_windows_profiles SET status = 'failed' WHERE profile_uuid = ?`
		_, err := q.ExecContext(context.Background(), stmt, globalProfiles[0])
		return err
	})
	checkHostProfileStatus(t, host.UUID, globalProfiles[0], fleet.MDMDeliveryFailed)
	checkHostsFilteredByOSSettingsStatus(t, []string{}, fleet.MDMDeliveryVerifying, nil, label)           // expect no hosts
	checkHostsFilteredByOSSettingsStatus(t, []string{host.Hostname}, fleet.MDMDeliveryFailed, nil, label) // expect host
	s.checkMDMProfilesSummaries(t, nil, fleet.MDMProfilesSummary{
		Failed:    1,
		Verifying: 0,
	}, nil)

	// add the host to a team
	err = s.ds.AddHostsToTeam(ctx, &tm.ID, []uint{host.ID})
	require.NoError(t, err)

	// trigger a profile sync, device gets the team profile
	verifyProfiles(mdmDevice, 2, false)
	checkHostsProfilesMatch(host, teamProfiles)
	checkHostDetails(t, host, teamProfiles, fleet.MDMDeliveryVerifying)

	// set new team profiles (delete + addition)
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `DELETE FROM mdm_windows_configuration_profiles WHERE profile_uuid = ?`
		_, err := q.ExecContext(context.Background(), stmt, teamProfiles[1])
		return err
	})
	teamProfiles = []string{
		teamProfiles[0],
		mysql.InsertWindowsProfileForTest(t, s.ds, tm.ID),
	}

	// trigger a profile sync, device gets the team profile
	verifyProfiles(mdmDevice, 1, false)

	// check that we deleted the old profile in the DB
	checkHostsProfilesMatch(host, teamProfiles)
	checkHostDetails(t, host, teamProfiles, fleet.MDMDeliveryVerifying)

	// another sync shouldn't return profiles
	verifyProfiles(mdmDevice, 0, false)

	// set new team profiles (delete + addition)
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `DELETE FROM mdm_windows_configuration_profiles WHERE profile_uuid = ?`
		_, err := q.ExecContext(context.Background(), stmt, teamProfiles[1])
		return err
	})
	teamProfiles = []string{
		teamProfiles[0],
		mysql.InsertWindowsProfileForTest(t, s.ds, tm.ID),
	}
	// trigger a profile sync, this time fail the delivery
	verifyProfiles(mdmDevice, 1, true)

	// check that we deleted the old profile in the DB
	checkHostsProfilesMatch(host, teamProfiles)

	// a second sync gets the profile again, because of delivery retries.
	// Succeed that one
	verifyProfiles(mdmDevice, 1, false)

	// another sync shouldn't return profiles
	verifyProfiles(mdmDevice, 0, false)

	// make fleet add a Windows OS Updates profile
	tmResp := teamResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm.ID), json.RawMessage(`{"mdm": { "windows_updates": {"deadline_days": 1, "grace_period_days": 1} }}`), http.StatusOK, &tmResp)
	osUpdatesProf = getProfileUUID(t, servermdm.FleetWindowsOSUpdatesProfileName, &tm.ID)

	// os updates is sent via a profiles commands
	verifyProfiles(mdmDevice, 1, false)
	checkHostsProfilesMatch(host, append(teamProfiles, osUpdatesProf))
	// but is hidden from host details response
	checkHostDetails(t, host, teamProfiles, fleet.MDMDeliveryVerifying)

	// os updates profile status doesn't matter for filtered hosts results or summaries
	checkHostProfileStatus(t, host.UUID, osUpdatesProf, fleet.MDMDeliveryVerifying)
	checkHostsFilteredByOSSettingsStatus(t, []string{host.Hostname}, fleet.MDMDeliveryVerifying, &tm.ID, label)
	s.checkMDMProfilesSummaries(t, &tm.ID, fleet.MDMProfilesSummary{
		Verifying: 1,
	}, nil)
	// force os updates profile to failed, doesn't impact filtered hosts results or summaries
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `UPDATE host_mdm_windows_profiles SET status = 'failed' WHERE profile_uuid = ?`
		_, err := q.ExecContext(context.Background(), stmt, osUpdatesProf)
		return err
	})
	checkHostProfileStatus(t, host.UUID, osUpdatesProf, fleet.MDMDeliveryFailed)
	checkHostsFilteredByOSSettingsStatus(t, []string{host.Hostname}, fleet.MDMDeliveryVerifying, &tm.ID, label)
	s.checkMDMProfilesSummaries(t, &tm.ID, fleet.MDMProfilesSummary{
		Verifying: 1,
	}, nil)
	// force another profile to failed, does impact filtered hosts results and summaries
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `UPDATE host_mdm_windows_profiles SET status = 'failed' WHERE profile_uuid = ?`
		_, err := q.ExecContext(context.Background(), stmt, teamProfiles[0])
		return err
	})
	checkHostProfileStatus(t, host.UUID, teamProfiles[0], fleet.MDMDeliveryFailed)
	checkHostsFilteredByOSSettingsStatus(t, []string{}, fleet.MDMDeliveryVerifying, &tm.ID, label)           // expect no hosts
	checkHostsFilteredByOSSettingsStatus(t, []string{host.Hostname}, fleet.MDMDeliveryFailed, &tm.ID, label) // expect host
	s.checkMDMProfilesSummaries(t, &tm.ID, fleet.MDMProfilesSummary{
		Failed:    1,
		Verifying: 0,
	}, nil)
}

func (s *integrationMDMTestSuite) TestAppConfigMDMWindowsProfiles() {
	t := s.T()

	// set the windows custom settings fields
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
		  "windows_settings": {
		    "custom_settings": [
		        {"path": "foo", "labels": ["baz"]},
			{"path": "bar"}
		    ]
		  }
		}
  }`), http.StatusOK, &acResp)
	assert.Equal(t, []fleet.MDMProfileSpec{{Path: "foo", Labels: []string{"baz"}}, {Path: "bar"}}, acResp.MDM.WindowsSettings.CustomSettings.Value)

	// check that they are returned by a GET /config
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.Equal(t, []fleet.MDMProfileSpec{{Path: "foo", Labels: []string{"baz"}}, {Path: "bar"}}, acResp.MDM.WindowsSettings.CustomSettings.Value)

	// patch without specifying the windows custom settings fields and an unrelated
	// field, should not remove them
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusOK, &acResp)
	assert.Equal(t, []fleet.MDMProfileSpec{{Path: "foo", Labels: []string{"baz"}}, {Path: "bar"}}, acResp.MDM.WindowsSettings.CustomSettings.Value)

	// patch with explicitly empty windows custom settings fields, would remove
	// them but this is a dry-run
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "windows_settings": { "custom_settings": null } }
  }`), http.StatusOK, &acResp, "dry_run", "true")
	assert.Equal(t, []fleet.MDMProfileSpec{{Path: "foo", Labels: []string{"baz"}}, {Path: "bar"}}, acResp.MDM.WindowsSettings.CustomSettings.Value)

	// patch with explicitly empty windows custom settings fields, removes them
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "windows_settings": { "custom_settings": null } }
  }`), http.StatusOK, &acResp)
	assert.Empty(t, acResp.MDM.WindowsSettings.CustomSettings.Value)
}

func (s *integrationMDMTestSuite) TestApplyTeamsMDMWindowsProfiles() {
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

	rawTeamSpec := func(mdmValue string) json.RawMessage {
		return json.RawMessage(fmt.Sprintf(`{ "specs": [{ "name": %q, "mdm": %s }] }`, team.Name, mdmValue))
	}

	// set the windows custom settings fields
	var applyResp applyTeamSpecsResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", rawTeamSpec(`
		{
		  "windows_settings": {
		    "custom_settings": [
		       {"path": "foo", "labels": ["baz"]},
		       {"path": "bar"}
		    ]
		  }
		}
	`), http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)

	// check that they are returned by a GET /config
	var teamResp getTeamResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.ElementsMatch(t, []fleet.MDMProfileSpec{{Path: "foo", Labels: []string{"baz"}}, {Path: "bar"}}, teamResp.Team.Config.MDM.WindowsSettings.CustomSettings.Value)

	// patch without specifying the windows custom settings fields and an unrelated
	// field, should not remove them
	applyResp = applyTeamSpecsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", rawTeamSpec(`{ "enable_disk_encryption": true }`), http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)

	// check that they are returned by a GET /config
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.ElementsMatch(t, []fleet.MDMProfileSpec{{Path: "foo", Labels: []string{"baz"}}, {Path: "bar"}}, teamResp.Team.Config.MDM.WindowsSettings.CustomSettings.Value)

	// patch with explicitly empty windows custom settings fields, would remove
	// them but this is a dry-run
	applyResp = applyTeamSpecsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", rawTeamSpec(`
		{ "windows_settings": { "custom_settings": null } }
  `), http.StatusOK, &applyResp, "dry_run", "true")
	assert.Equal(t, map[string]uint{team.Name: team.ID}, applyResp.TeamIDsByName)

	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.ElementsMatch(t, []fleet.MDMProfileSpec{{Path: "foo", Labels: []string{"baz"}}, {Path: "bar"}}, teamResp.Team.Config.MDM.WindowsSettings.CustomSettings.Value)

	// patch with explicitly empty windows custom settings fields, removes them
	applyResp = applyTeamSpecsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", rawTeamSpec(`
		{ "windows_settings": { "custom_settings": null } }
  `), http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)

	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Empty(t, teamResp.Team.Config.MDM.WindowsSettings.CustomSettings.Value)
}

func (s *integrationMDMTestSuite) TestBatchSetMDMProfiles() {
	t := s.T()
	ctx := context.Background()

	// create a new team
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "batch_set_mdm_profiles"})
	require.NoError(t, err)

	// apply an empty set to no-team
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: nil}, http.StatusNoContent)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedMacosProfile{}.ActivityName(),
		`{"team_id": null, "team_name": null}`,
		0,
	)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedWindowsProfile{}.ActivityName(),
		`{"team_id": null, "team_name": null}`,
		0,
	)

	// apply to both team id and name
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: nil},
		http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)), "team_name", tm.Name)

	// invalid team name
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: nil},
		http.StatusNotFound, "team_name", uuid.New().String())

	// duplicate PayloadDisplayName
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
		{Name: "N2", Contents: mobileconfigForTest("N1", "I2")},
		{Name: "N3", Contents: syncMLForTest("./Foo/Bar")},
	}}, http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)))

	// profiles with reserved macOS identifiers
	for p := range mobileconfig.FleetPayloadIdentifiers() {
		res := s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
			{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
			{Name: p, Contents: mobileconfigForTest(p, p)},
			{Name: "N3", Contents: syncMLForTest("./Foo/Bar")},
		}}, http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)))
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, fmt.Sprintf("Validation Failed: payload identifier %s is not allowed", p))
	}

	// payloads with reserved types
	for p := range mobileconfig.FleetPayloadTypes() {
		res := s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
			{Name: "N1", Contents: mobileconfigForTestWithContent("N1", "I1", "II1", p, "")},
			{Name: "N3", Contents: syncMLForTest("./Foo/Bar")},
		}}, http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)))
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, fmt.Sprintf("Validation Failed: unsupported PayloadType(s): %s", p))
	}

	// payloads with reserved identifiers
	for p := range mobileconfig.FleetPayloadIdentifiers() {
		res := s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
			{Name: "N1", Contents: mobileconfigForTestWithContent("N1", "I1", p, "random", "")},
			{Name: "N3", Contents: syncMLForTest("./Foo/Bar")},
		}}, http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)))
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, fmt.Sprintf("Validation Failed: unsupported PayloadIdentifier(s): %s", p))
	}

	// profiles with reserved Windows location URIs
	// bitlocker
	res := s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
		{Name: syncml.FleetBitLockerTargetLocURI, Contents: syncMLForTest(fmt.Sprintf("%s/Foo", syncml.FleetBitLockerTargetLocURI))},
		{Name: "N3", Contents: syncMLForTest("./Foo/Bar")},
	}}, http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)))
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Custom configuration profiles can't include BitLocker settings. To control these settings, use the mdm.enable_disk_encryption option.")

	// os updates
	res = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
		{Name: syncml.FleetOSUpdateTargetLocURI, Contents: syncMLForTest(fmt.Sprintf("%s/Foo", syncml.FleetOSUpdateTargetLocURI))},
		{Name: "N3", Contents: syncMLForTest("./Foo/Bar")},
	}}, http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)))
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Custom configuration profiles can't include Windows updates settings. To control these settings, use the mdm.windows_updates option.")

	// invalid windows tag
	res = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N3", Contents: []byte(`<Exec></Exec>`)},
	}}, http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)))
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Only <Replace> supported as a top level element")

	// invalid xml
	res = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N3", Contents: []byte(`foo`)},
	}}, http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)))
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Only <Replace> supported as a top level element")

	// successfully apply windows and macOS a profiles for the team, but it's a dry run
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
		{Name: "N2", Contents: syncMLForTest("./Foo/Bar")},
	}}, http.StatusNoContent, "team_id", strconv.Itoa(int(tm.ID)), "dry_run", "true")
	s.assertConfigProfilesByIdentifier(&tm.ID, "I1", false)
	s.assertWindowsConfigProfilesByName(&tm.ID, "N1", false)

	// successfully apply for a team and verify activities
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
		{Name: "N2", Contents: syncMLForTest("./Foo/Bar")},
	}}, http.StatusNoContent, "team_id", strconv.Itoa(int(tm.ID)))
	s.assertConfigProfilesByIdentifier(&tm.ID, "I1", true)
	s.assertWindowsConfigProfilesByName(&tm.ID, "N2", true)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedMacosProfile{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q}`, tm.ID, tm.Name),
		0,
	)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedWindowsProfile{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q}`, tm.ID, tm.Name),
		0,
	)
}

func (s *integrationMDMTestSuite) TestBatchSetMDMProfilesBackwardsCompat() {
	t := s.T()
	ctx := context.Background()

	// create a new team
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "batch_set_mdm_profiles"})
	require.NoError(t, err)

	// apply an empty set to no-team
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": nil}, http.StatusNoContent)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedMacosProfile{}.ActivityName(),
		`{"team_id": null, "team_name": null}`,
		0,
	)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedWindowsProfile{}.ActivityName(),
		`{"team_id": null, "team_name": null}`,
		0,
	)

	// apply to both team id and name
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": nil},
		http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)), "team_name", tm.Name)

	// invalid team name
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": nil},
		http.StatusNotFound, "team_name", uuid.New().String())

	// duplicate PayloadDisplayName
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": map[string][]byte{
		"N1": mobileconfigForTest("N1", "I1"),
		"N2": mobileconfigForTest("N1", "I2"),
		"N3": syncMLForTest("./Foo/Bar"),
	}}, http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)))

	// profiles with reserved macOS identifiers
	for p := range mobileconfig.FleetPayloadIdentifiers() {
		res := s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": map[string][]byte{
			"N1": mobileconfigForTest("N1", "I1"),
			p:    mobileconfigForTest(p, p),
			"N3": syncMLForTest("./Foo/Bar"),
		}}, http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)))
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, fmt.Sprintf("Validation Failed: payload identifier %s is not allowed", p))
	}

	// payloads with reserved types
	for p := range mobileconfig.FleetPayloadTypes() {
		res := s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": map[string][]byte{
			"N1": mobileconfigForTestWithContent("N1", "I1", "II1", p, ""),
			"N3": syncMLForTest("./Foo/Bar"),
		}}, http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)))
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, fmt.Sprintf("Validation Failed: unsupported PayloadType(s): %s", p))
	}

	// payloads with reserved identifiers
	for p := range mobileconfig.FleetPayloadIdentifiers() {
		res := s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": map[string][]byte{
			"N1": mobileconfigForTestWithContent("N1", "I1", p, "random", ""),
			"N3": syncMLForTest("./Foo/Bar"),
		}}, http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)))
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, fmt.Sprintf("Validation Failed: unsupported PayloadIdentifier(s): %s", p))
	}

	// profiles with reserved Windows location URIs
	// bitlocker
	res := s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": map[string][]byte{
		"N1":                              mobileconfigForTest("N1", "I1"),
		syncml.FleetBitLockerTargetLocURI: syncMLForTest(fmt.Sprintf("%s/Foo", syncml.FleetBitLockerTargetLocURI)),
		"N3":                              syncMLForTest("./Foo/Bar"),
	}}, http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)))
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Custom configuration profiles can't include BitLocker settings. To control these settings, use the mdm.enable_disk_encryption option.")

	// os updates
	res = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": map[string][]byte{
		"N1":                             mobileconfigForTest("N1", "I1"),
		syncml.FleetOSUpdateTargetLocURI: syncMLForTest(fmt.Sprintf("%s/Foo", syncml.FleetOSUpdateTargetLocURI)),
		"N3":                             syncMLForTest("./Foo/Bar"),
	}}, http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)))
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Custom configuration profiles can't include Windows updates settings. To control these settings, use the mdm.windows_updates option.")

	// invalid windows tag
	res = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": map[string][]byte{
		"N3": []byte(`<Exec></Exec>`),
	}}, http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)))
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Only <Replace> supported as a top level element")

	// invalid xml
	res = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": map[string][]byte{
		"N3": []byte(`foo`),
	}}, http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)))
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Only <Replace> supported as a top level element")

	// successfully apply windows and macOS a profiles for the team, but it's a dry run
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": map[string][]byte{
		"N1": mobileconfigForTest("N1", "I1"),
		"N2": syncMLForTest("./Foo/Bar"),
	}}, http.StatusNoContent, "team_id", strconv.Itoa(int(tm.ID)), "dry_run", "true")
	s.assertConfigProfilesByIdentifier(&tm.ID, "I1", false)
	s.assertWindowsConfigProfilesByName(&tm.ID, "N1", false)

	// successfully apply for a team and verify activities
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", map[string]any{"profiles": map[string][]byte{
		"N1": mobileconfigForTest("N1", "I1"),
		"N2": syncMLForTest("./Foo/Bar"),
	}}, http.StatusNoContent, "team_id", strconv.Itoa(int(tm.ID)))
	s.assertConfigProfilesByIdentifier(&tm.ID, "I1", true)
	s.assertWindowsConfigProfilesByName(&tm.ID, "N2", true)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedMacosProfile{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q}`, tm.ID, tm.Name),
		0,
	)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedWindowsProfile{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q}`, tm.ID, tm.Name),
		0,
	)
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
			if manifest := cmd.Command.InstallEnterpriseApplication.ManifestURL; manifest != nil {
				foundInstallFleetdCommand = true
				require.Equal(t, "InstallEnterpriseApplication", cmd.Command.RequestType)
				require.Contains(t, *cmd.Command.InstallEnterpriseApplication.ManifestURL, apple_mdm.FleetdPublicManifestURL)
			}
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}
		require.Equal(t, wantCommand, foundInstallFleetdCommand)
	}

	// create a device that's not enrolled into Fleet, it should get a command to
	// install fleetd
	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.fleetCfg.MDM.AppleSCEPChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	})
	err := mdmDevice.Enroll()
	require.NoError(t, err)
	s.runWorker()
	checkInstallFleetdCommandSent(mdmDevice, true)

	// create a device that's enrolled into Fleet before turning on MDM features,
	// it shouldn't get the command to install fleetd
	desktopToken := uuid.New().String()
	host := createOrbitEnrolledHost(t, "darwin", "h1", s.ds)
	err = s.ds.SetOrUpdateDeviceAuthToken(context.Background(), host.ID, desktopToken)
	require.NoError(t, err)
	mdmDevice = mdmtest.NewTestMDMClientAppleDesktopManual(s.server.URL, desktopToken)
	mdmDevice.UUID = host.UUID
	err = mdmDevice.Enroll()
	require.NoError(t, err)
	s.runWorker()
	checkInstallFleetdCommandSent(mdmDevice, false)
}

func (s *integrationMDMTestSuite) TestLockUnlockWindowsLinux() {
	t := s.T()
	ctx := context.Background()

	// create an MDM-enrolled Windows host
	winHost, _ := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
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
			status, err := s.ds.GetHostLockWipeStatus(ctx, host.ID, host.FleetPlatform())
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
			status, err = s.ds.GetHostLockWipeStatus(ctx, host.ID, host.FleetPlatform())
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
		})
	}
}

func (s *integrationMDMTestSuite) TestZCustomConfigurationWebURL() {
	t := s.T()

	acResp := appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)

	var lastSubmittedProfile *godep.Profile
	s.mockDEPResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)

		switch r.URL.Path {
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
	require.Contains(t, lastSubmittedProfile.ConfigurationWebURL, acResp.ServerSettings.ServerURL+"/mdm/sso")

	// trying to set a custom configuration_web_url fails because end user authentication is enabled
	customSetupAsst := `{"configuration_web_url": "https://foo.example.com"}`
	var globalAsstResp createMDMAppleSetupAssistantResponse
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/enrollment_profile", createMDMAppleSetupAssistantRequest{
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
	require.Contains(t, lastSubmittedProfile.ConfigurationWebURL, acResp.ServerSettings.ServerURL+"/api/mdm/apple/enroll?token=")

	// setting a custom configuration_web_url succeeds because user authentication is disabled
	globalAsstResp = createMDMAppleSetupAssistantResponse{}
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/enrollment_profile", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "no-team",
		EnrollmentProfile: json.RawMessage(customSetupAsst),
	}, http.StatusOK, &globalAsstResp)

	// assign the DEP profile and assert that contains the right values for the URL
	s.runWorker()
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

	// re-set the global state to configure MDM SSO
	err := s.ds.DeleteMDMAppleSetupAssistant(context.Background(), nil)
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
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/enrollment_profile", createMDMAppleSetupAssistantRequest{
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
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/enrollment_profile", createMDMAppleSetupAssistantRequest{
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

func (s *integrationMDMTestSuite) TestGetManualEnrollmentProfile() {
	s.downloadAndVerifyEnrollmentProfile("/api/latest/fleet/mdm/manual_enrollment_profile")
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

	// The profiles should be associated with the host we made + the standard fleet config
	profs, err := s.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
	require.NoError(t, err)
	require.Len(t, profs, 3)

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
		mobileconfig.FleetdConfigPayloadIdentifier: {Identifier: mobileconfig.FleetdConfigPayloadIdentifier, DisplayName: "I2", InstallDate: time.Now()},
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
			if cmd.Command.RemoveProfile.Identifier == "I1" {
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

	// get that host - it should report "failed" for the profiles and include the error message detail
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

func (s *integrationMDMTestSuite) TestSCEPCertExpiration() {
	t := s.T()
	ctx := context.Background()
	// ensure there's a token for automatic enrollments
	s.mockDEPResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
	}))
	s.runDEPSchedule()

	// add a device that's manually enrolled
	desktopToken := uuid.New().String()
	manualHost := createOrbitEnrolledHost(t, "darwin", "h1", s.ds)
	err := s.ds.SetOrUpdateDeviceAuthToken(context.Background(), manualHost.ID, desktopToken)
	require.NoError(t, err)
	manualEnrolledDevice := mdmtest.NewTestMDMClientAppleDesktopManual(s.server.URL, desktopToken)
	manualEnrolledDevice.UUID = manualHost.UUID
	err = manualEnrolledDevice.Enroll()
	require.NoError(t, err)

	// add a device that's automatically enrolled
	automaticHost := createOrbitEnrolledHost(t, "darwin", "h2", s.ds)
	depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
	automaticEnrolledDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
	automaticEnrolledDevice.UUID = automaticHost.UUID
	automaticEnrolledDevice.SerialNumber = automaticHost.HardwareSerial
	err = automaticEnrolledDevice.Enroll()
	require.NoError(t, err)

	// add a device that's automatically enrolled with a server ref
	automaticHostWithRef := createOrbitEnrolledHost(t, "darwin", "h3", s.ds)
	automaticEnrolledDeviceWithRef := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
	automaticEnrolledDeviceWithRef.UUID = automaticHostWithRef.UUID
	automaticEnrolledDeviceWithRef.SerialNumber = automaticHostWithRef.HardwareSerial
	err = automaticEnrolledDeviceWithRef.Enroll()
	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, automaticHostWithRef.ID, false, true, s.server.URL, true, fleet.WellKnownMDMFleet, "foo"))
	require.NoError(t, err)

	cert, key, err := generateCertWithAPNsTopic()
	require.NoError(t, err)
	fleetCfg := config.TestConfig()
	config.SetTestMDMConfig(s.T(), &fleetCfg, cert, key, testBMToken, "")
	logger := kitlog.NewJSONLogger(os.Stdout)

	// run without expired certs, no command enqueued
	err = RenewSCEPCertificates(ctx, logger, s.ds, &fleetCfg, s.mdmCommander)
	require.NoError(t, err)
	cmd, err := manualEnrolledDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	cmd, err = automaticEnrolledDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	cmd, err = automaticEnrolledDeviceWithRef.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	// expire all the certs we just created
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
                  UPDATE nano_cert_auth_associations
                  SET cert_not_valid_after = DATE_SUB(CURDATE(), INTERVAL 1 YEAR)
                  WHERE id IN (?, ?, ?)
		`, manualHost.UUID, automaticHost.UUID, automaticHostWithRef.UUID)
		return err
	})

	// generate a new config here so we can manipulate the certs.
	err = RenewSCEPCertificates(ctx, logger, s.ds, &fleetCfg, s.mdmCommander)
	require.NoError(t, err)

	checkRenewCertCommand := func(device *mdmtest.TestAppleMDMClient, enrollRef string) {
		var renewCmd *micromdm.CommandPayload
		cmd, err := device.Idle()
		require.NoError(t, err)
		for cmd != nil {
			if cmd.Command.RequestType == "InstallProfile" {
				renewCmd = cmd
			}
			cmd, err = device.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}
		require.NotNil(t, renewCmd)
		s.verifyEnrollmentProfile(renewCmd.Command.InstallProfile.Payload, enrollRef)
	}

	checkRenewCertCommand(manualEnrolledDevice, "")
	checkRenewCertCommand(automaticEnrolledDevice, "")
	checkRenewCertCommand(automaticEnrolledDeviceWithRef, "foo")

	// another cron run shouldn't enqueue more commands
	err = RenewSCEPCertificates(ctx, logger, s.ds, &fleetCfg, s.mdmCommander)
	require.NoError(t, err)

	cmd, err = manualEnrolledDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	cmd, err = automaticEnrolledDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	cmd, err = automaticEnrolledDeviceWithRef.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)
}
