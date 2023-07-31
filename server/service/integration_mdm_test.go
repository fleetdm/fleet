package service

import (
	"bytes"
	"context"
	"crypto/x509"
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
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/mock"
	"github.com/fleetdm/fleet/v4/server/service/schedule"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/fleetdm/fleet/v4/server/worker"
	kitlog "github.com/go-kit/kit/log"
	"github.com/google/uuid"
	"github.com/groob/plist"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	nanodep_client "github.com/micromdm/nanodep/client"
	"github.com/micromdm/nanodep/godep"
	nanodep_storage "github.com/micromdm/nanodep/storage"
	"github.com/micromdm/nanodep/tokenpki"
	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/push"
	nanomdm_pushsvc "github.com/micromdm/nanomdm/push/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.mozilla.org/pkcs7"
)

func TestIntegrationsMDM(t *testing.T) {
	t.Setenv("FLEET_DEV_MDM_ENABLED", "1")

	testingSuite := new(integrationMDMTestSuite)
	testingSuite.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

type integrationMDMTestSuite struct {
	suite.Suite
	withServer
	fleetCfg              config.FleetConfig
	fleetDMNextCSRStatus  atomic.Value
	pushProvider          *mock.APNSPushProvider
	depStorage            nanodep_storage.AllStorage
	depSchedule           *schedule.Schedule
	profileSchedule       *schedule.Schedule
	onProfileScheduleDone func() // function called when profileSchedule.Trigger() job completed
	onDEPScheduleDone     func() // function called when depSchedule.Trigger() job completed
	mdmStorage            *mysql.NanoMDMStorage
	worker                *worker.Worker
}

func (s *integrationMDMTestSuite) SetupSuite() {
	s.withDS.SetupSuite("integrationMDMTestSuite")

	appConf, err := s.ds.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.MDM.EnabledAndConfigured = true
	appConf.MDM.WindowsEnabledAndConfigured = true
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)

	testCert, testKey, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(s.T(), err)
	testCertPEM := tokenpki.PEMCertificate(testCert.Raw)
	testKeyPEM := tokenpki.PEMRSAPrivateKey(testKey)

	fleetCfg := config.TestConfig()
	config.SetTestMDMConfig(s.T(), &fleetCfg, testCertPEM, testKeyPEM, testBMToken)
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
							return fleetSyncer.RunAssigner(ctx)
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
						schedule.WithJob("manage_profiles", func(ctx context.Context) error {
							if s.onProfileScheduleDone != nil {
								defer s.onProfileScheduleDone()
							}
							return ReconcileProfiles(ctx, ds, mdmCommander, logger)
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
	appCfg.MDM.MacOSSettings.EnableDiskEncryption = false
	err := s.ds.SaveAppConfig(ctx, &appCfg.AppConfig)
	require.NoError(t, err)

	s.withServer.commonTearDownTest(t)

	// use a sql statement to delete all profiles, since the datastore prevents
	// deleting the fleet-specific ones.
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM mdm_apple_configuration_profiles")
		return err
	})
	// clear any pending worker job
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM jobs")
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

func (s *integrationMDMTestSuite) TestProfileManagement() {
	t := s.T()
	ctx := context.Background()

	err := s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: t.Name()}})
	require.NoError(t, err)
	var globalFleetdProfile bytes.Buffer
	params := mobileconfig.FleetdProfileOptions{
		EnrollSecret: t.Name(),
		ServerURL:    s.server.URL,
		PayloadType:  mobileconfig.FleetdConfigPayloadIdentifier,
	}
	err = mobileconfig.FleetdProfileTemplate.Execute(&globalFleetdProfile, params)
	require.NoError(t, err)

	globalProfiles := [][]byte{
		mobileconfigForTest("N1", "I1"),
		mobileconfigForTest("N2", "I2"),
	}
	wantGlobalProfiles := append(globalProfiles, globalFleetdProfile.Bytes())

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
	var teamFleetdProfile bytes.Buffer
	params.EnrollSecret = "team1_enroll_sec"
	err = mobileconfig.FleetdProfileTemplate.Execute(&teamFleetdProfile, params)
	require.NoError(t, err)
	wantTeamProfiles := append(teamProfiles, teamFleetdProfile.Bytes())
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

	triggerSchedule := func() {
		ch := make(chan struct{})
		s.onProfileScheduleDone = func() {
			close(ch)
		}
		_, err := s.profileSchedule.Trigger()
		require.NoError(t, err)
		<-ch
	}

	origPush := s.pushProvider.PushFunc
	defer func() { s.pushProvider.PushFunc = origPush }()
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

	checkNextPayloads := func() ([][]byte, []string) {
		var cmd *micromdm.CommandPayload
		installs := [][]byte{}
		removes := []string{}

		for {
			// on the first run, cmd will be nil and we need to
			// ping the server via idle
			if cmd == nil {
				cmd, err = mdmDevice.Idle()
			} else {
				cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			}
			require.NoError(t, err)

			// if after idle or acknowledge cmd is still nil, it
			// means there aren't any commands left to run
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

	// trigger a profile sync
	triggerSchedule()

	time.Sleep(5 * time.Second)

	installs, removes := checkNextPayloads()

	// verify that we received all profiles
	require.ElementsMatch(t, wantGlobalProfiles, installs)
	require.Empty(t, removes)

	// add the host to a team
	err = s.ds.AddHostsToTeam(ctx, &tm.ID, []uint{host.ID})
	require.NoError(t, err)

	// trigger a profile sync
	triggerSchedule()

	installs, removes = checkNextPayloads()
	// verify that we should install the team profile
	require.ElementsMatch(t, wantTeamProfiles, installs)
	// verify that we should delete both profiles
	require.ElementsMatch(t, []string{"I1", "I2"}, removes)

	// set new team profiles (delete + addition)
	teamProfiles = [][]byte{
		mobileconfigForTest("N4", "I4"),
		mobileconfigForTest("N5", "I5"),
	}
	wantTeamProfiles = teamProfiles
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: teamProfiles}, http.StatusNoContent, "team_id", strconv.Itoa(int(tm.ID)))

	// trigger a profile sync
	triggerSchedule()
	installs, removes = checkNextPayloads()
	// verify that we should install the team profiles
	require.ElementsMatch(t, wantTeamProfiles, installs)
	// verify that we should delete the old team profiles
	require.ElementsMatch(t, []string{"I3"}, removes)

	// with no changes
	_, err = s.profileSchedule.Trigger()
	require.NoError(t, err)
	installs, removes = checkNextPayloads()
	require.Empty(t, installs)
	require.Empty(t, removes)

	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", host.ID), getHostRequest{}, http.StatusOK, &hostResp)
	require.NotEmpty(t, hostResp.Host.MDM.Profiles)
	resProfiles := *hostResp.Host.MDM.Profiles
	// one extra profile for the fleetd config
	require.Len(t, resProfiles, len(wantTeamProfiles)+1)

	var teamSummaryResp getMDMAppleProfilesSummaryResponse
	s.DoJSON("GET", "/api/v1/fleet/mdm/apple/profiles/summary", getMDMAppleProfilesSummaryRequest{TeamID: &tm.ID}, http.StatusOK, &teamSummaryResp)
	require.Equal(t, uint(0), teamSummaryResp.Pending)
	require.Equal(t, uint(0), teamSummaryResp.Failed)
	require.Equal(t, uint(1), teamSummaryResp.Verifying)
	require.Equal(t, uint(0), teamSummaryResp.Verified)

	var noTeamSummaryResp getMDMAppleProfilesSummaryResponse
	s.DoJSON("GET", "/api/v1/fleet/mdm/apple/profiles/summary", getMDMAppleProfilesSummaryRequest{}, http.StatusOK, &noTeamSummaryResp)
	require.Equal(t, uint(0), noTeamSummaryResp.Pending)
	require.Equal(t, uint(0), noTeamSummaryResp.Failed)
	require.Equal(t, uint(0), noTeamSummaryResp.Verifying)
	require.Equal(t, uint(0), noTeamSummaryResp.Verified)
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

	// it create activities for the new team, the profiles assigned to it, and
	// the host moved to it
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

	// and the team has the expected profiles
	profs, err := s.ds.ListMDMAppleConfigProfiles(ctx, &tm1.ID)
	require.NoError(t, err)
	require.Len(t, profs, 2)
	// order is guaranteed by profile name
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	// filevault is enabled by default
	require.True(t, tm1.Config.MDM.MacOSSettings.EnableDiskEncryption)

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

	// preassign the MDM host to prof1 and prof4, should match existing team tm2
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{MDMApplePreassignProfilePayload: fleet.MDMApplePreassignProfilePayload{ExternalHostIdentifier: "mdm1", HostUUID: mdmHost.UUID, Profile: prof1, Group: "g1"}}, http.StatusNoContent)
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{MDMApplePreassignProfilePayload: fleet.MDMApplePreassignProfilePayload{ExternalHostIdentifier: "mdm1", HostUUID: mdmHost.UUID, Profile: prof4, Group: "g4"}}, http.StatusNoContent)

	// match with the mdm host succeeds and assigns it to tm2
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/match", matchMDMApplePreassignmentRequest{ExternalHostIdentifier: "mdm1"}, http.StatusNoContent)

	// the host is now part of that team
	h, err = s.ds.Host(ctx, mdmHost.ID)
	require.NoError(t, err)
	require.NotNil(t, h.TeamID)
	require.Equal(t, tm2.ID, *h.TeamID)

	// the host's profiles are the same as the team's and are pending, and prof2 + old filevault are pending removal
	hostProfs, err := s.ds.GetHostMDMProfiles(ctx, mdmHost.UUID)
	require.NoError(t, err)
	require.Len(t, hostProfs, 4)

	sort.Slice(hostProfs, func(i, j int) bool {
		l, r := hostProfs[i], hostProfs[j]
		return l.Name < r.Name
	})
	require.Equal(t, "Disk encryption", hostProfs[0].Name)
	require.NotNil(t, hostProfs[0].Status)
	require.Equal(t, fleet.MDMAppleDeliveryPending, *hostProfs[0].Status)
	require.Equal(t, fleet.MDMAppleOperationTypeRemove, hostProfs[0].OperationType)
	require.Equal(t, "n1", hostProfs[1].Name)
	require.NotNil(t, hostProfs[1].Status)
	require.Equal(t, fleet.MDMAppleDeliveryPending, *hostProfs[1].Status)
	require.Equal(t, fleet.MDMAppleOperationTypeInstall, hostProfs[1].OperationType)
	require.Equal(t, "n2", hostProfs[2].Name)
	require.NotNil(t, hostProfs[2].Status)
	require.Equal(t, fleet.MDMAppleDeliveryPending, *hostProfs[2].Status)
	require.Equal(t, fleet.MDMAppleOperationTypeRemove, hostProfs[2].OperationType)
	require.Equal(t, "n4", hostProfs[3].Name)
	require.NotNil(t, hostProfs[3].Status)
	require.Equal(t, fleet.MDMAppleDeliveryPending, *hostProfs[3].Status)
	require.Equal(t, fleet.MDMAppleOperationTypeInstall, hostProfs[3].OperationType)

	// create a new mdm host enrolled in fleet
	mdmHost2, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	s.runWorker()
	// make it part of team 2
	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &tm2.ID, HostIDs: []uint{mdmHost2.ID}}, http.StatusOK)

	// simulate having its profiles installed
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE host_mdm_apple_profiles SET status = ? WHERE host_uuid = ?`, fleet.MacOSSettingsVerifying, mdmHost2.UUID)
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
	hostProfs, err = s.ds.GetHostMDMProfiles(ctx, mdmHost2.UUID)
	require.NoError(t, err)
	require.Len(t, hostProfs, 2)

	sort.Slice(hostProfs, func(i, j int) bool {
		l, r := hostProfs[i], hostProfs[j]
		return l.Name < r.Name
	})
	require.Equal(t, "n1", hostProfs[0].Name)
	require.NotNil(t, hostProfs[0].Status)
	require.Equal(t, fleet.MDMAppleDeliveryVerifying, *hostProfs[0].Status)
	require.Equal(t, "n4", hostProfs[1].Name)
	require.NotNil(t, hostProfs[1].Status)
	require.Equal(t, fleet.MDMAppleDeliveryVerifying, *hostProfs[1].Status)
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
	require.True(t, tm1.Config.MDM.MacOSSettings.EnableDiskEncryption)

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
	require.True(t, tm2.Config.MDM.MacOSSettings.EnableDiskEncryption)

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
	require.True(t, tm1.Config.MDM.MacOSSettings.EnableDiskEncryption)

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
	require.True(t, tm1.Config.MDM.MacOSSettings.EnableDiskEncryption)

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
	require.True(t, tm1.Config.MDM.MacOSSettings.EnableDiskEncryption)

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
	require.True(t, tm1.Config.MDM.MacOSSettings.EnableDiskEncryption)

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
	require.True(t, tm1.Config.MDM.MacOSSettings.EnableDiskEncryption)

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
	require.True(t, tm3.Config.MDM.MacOSSettings.EnableDiskEncryption)
}

func createHostThenEnrollMDM(ds fleet.Datastore, fleetServerURL string, t *testing.T) (*fleet.Host, *mdmtest.TestMDMClient) {
	desktopToken := uuid.New().String()
	mdmDevice := mdmtest.NewTestMDMClientDesktopManual(fleetServerURL, desktopToken)
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

func (s *integrationMDMTestSuite) TestDEPProfileAssignment() {
	t := s.T()
	ctx := context.Background()
	devices := []godep.Device{
		{SerialNumber: uuid.New().String(), Model: "MacBook Pro", OS: "osx", OpType: "added"},
		{SerialNumber: uuid.New().String(), Model: "MacBook Mini", OS: "osx", OpType: "added"},
		{SerialNumber: uuid.New().String(), Model: "MacBook Mini", OS: "osx", OpType: ""},
		{SerialNumber: uuid.New().String(), Model: "MacBook Mini", OS: "osx", OpType: "modified"},
	}

	runDEPSchedule := func() {
		ch := make(chan bool)
		s.onDEPScheduleDone = func() { close(ch) }
		_, err := s.depSchedule.Trigger()
		require.NoError(t, err)
		<-ch
	}

	// add global profiles
	globalProfile := mobileconfigForTest("N1", "I1")
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{globalProfile}}, http.StatusNoContent)

	checkPostEnrollmentCommands := func(mdmDevice *mdmtest.TestMDMClient, shouldReceive bool) {
		// run the worker to process the DEP enroll request
		s.runWorker()
		// run the worker to assign configuration profiles
		ch := make(chan bool)
		s.onProfileScheduleDone = func() { close(ch) }
		_, err := s.profileSchedule.Trigger()
		require.NoError(t, err)
		<-ch

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
			err := encoder.Encode(godep.ProfileResponse{ProfileUUID: "abc"})
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
	runDEPSchedule()

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

	// create a new host
	nonDEPHost := createHostAndDeviceToken(t, s.ds, "not-dep")
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, len(devices)+1)

	// filtering by MDM status works
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts?mdm_enrollment_status=pending", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, len(devices))

	s.pushProvider.PushFunc = func(pushes []*mdm.Push) (map[string]*push.Response, error) {
		return map[string]*push.Response{}, nil
	}

	// Enroll one of the hosts
	depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
	mdmDevice := mdmtest.NewTestMDMClientDEP(s.server.URL, depURLToken)
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
	runDEPSchedule()

	// all hosts should be returned from the hosts endpoint
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	// all previous devices + the manually added host + the new `addedSerial`
	wantSerials = append(wantSerials, devices[3].SerialNumber, nonDEPHost.HardwareSerial)
	require.Len(t, listHostsRes.Hosts, len(wantSerials))
	gotSerials = []string{}
	var deletedHostID uint
	var addedHostID uint
	for _, device := range listHostsRes.Hosts {
		gotSerials = append(gotSerials, device.HardwareSerial)
		switch device.HardwareSerial {
		case deletedSerial:
			deletedHostID = device.ID
		case addedSerial:
			addedHostID = device.ID
		}
	}
	require.ElementsMatch(t, wantSerials, gotSerials)

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
	mdmEnrollInfo := mdmtest.EnrollInfo{
		SCEPChallenge: s.fleetCfg.MDM.AppleSCEPChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}
	mdmDeviceA := mdmtest.NewTestMDMClientDirect(mdmEnrollInfo)
	err := mdmDeviceA.Enroll()
	require.NoError(t, err)
	mdmDeviceB := mdmtest.NewTestMDMClientDirect(mdmEnrollInfo)
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

	mdmDevice := mdmtest.NewTestMDMClientDirect(mdmtest.EnrollInfo{
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
	mdmDevice := mdmtest.NewTestMDMClientDirect(mdmtest.EnrollInfo{
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
	_, err = s.profileSchedule.Trigger()
	require.NoError(t, err)

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

func (s *integrationMDMTestSuite) TestMDMAppleGetEncryptionKey() {
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
		"mdm": { "macos_settings": { "enable_disk_encryption": true } }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.MacOSSettings.EnableDiskEncryption)
	fileVaultProf := s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)
	hostCmdUUID := uuid.New().String()
	err = s.ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
		{
			ProfileID:         fileVaultProf.ProfileID,
			ProfileIdentifier: fileVaultProf.Identifier,
			HostUUID:          host.UUID,
			CommandUUID:       hostCmdUUID,
			OperationType:     fleet.MDMAppleOperationTypeInstall,
			Status:            &fleet.MDMAppleDeliveryVerified,
			Checksum:          []byte("csum"),
		},
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		err := s.ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
			HostUUID:      host.UUID,
			CommandUUID:   hostCmdUUID,
			ProfileID:     fileVaultProf.ProfileID,
			Status:        &fleet.MDMAppleDeliveryVerifying,
			OperationType: fleet.MDMAppleOperationTypeRemove,
		})
		require.NoError(t, err)
		// not an error if the profile does not exist
		_ = s.ds.DeleteMDMAppleConfigProfile(ctx, fileVaultProf.ProfileID)
	})

	// get that host - it has no encryption key at this point, so it should
	// report "action_required" disk encryption and "log_out" action.
	getHostResp := getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Equal(t, fleet.DiskEncryptionActionRequired, *getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.NotNil(t, getHostResp.Host.MDM.MacOSSettings.ActionRequired)
	require.Equal(t, fleet.ActionRequiredLogOut, *getHostResp.Host.MDM.MacOSSettings.ActionRequired)

	// add an encryption key for the host
	cert, _, _, err := s.fleetCfg.MDM.AppleSCEP()
	require.NoError(t, err)
	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	require.NoError(t, err)
	recoveryKey := "AAA-BBB-CCC"
	encryptedKey, err := pkcs7.Encrypt([]byte(recoveryKey), []*x509.Certificate{parsed})
	require.NoError(t, err)
	base64EncryptedKey := base64.StdEncoding.EncodeToString(encryptedKey)

	err = s.ds.SetOrUpdateHostDiskEncryptionKey(ctx, host.ID, base64EncryptedKey)
	require.NoError(t, err)

	// get that host - it has an encryption key with unknown decryptability, so
	// it should report "enforcing" disk encryption.
	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Equal(t, fleet.DiskEncryptionEnforcing, *getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Nil(t, getHostResp.Host.MDM.MacOSSettings.ActionRequired)

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
			MacOSSettings: map[string]interface{}{"enable_disk_encryption": true},
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
		return generateNewProfileMultipartRequest(t, teamID, "some_filename", testProfiles[name].Mobileconfig, s.token)
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

	// trying to add/delete profiles managed by Fleet fails
	for p := range mobileconfig.FleetPayloadIdentifiers() {
		generateTestProfile("TestNoTeam", p)
		body, headers := generateNewReq("TestNoTeam", nil)
		s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusBadRequest, headers)

		generateTestProfile("TestWithTeamID", p)
		body, headers = generateNewReq("TestWithTeamID", nil)
		s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusBadRequest, headers)
		cp, err := fleet.NewMDMAppleConfigProfile(mobileconfigForTestWithContent("N1", "I1", p, "random"), nil)
		require.NoError(t, err)
		testProfiles["WithContent"] = *cp
		body, headers = generateNewReq("WithContent", nil)
		s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusBadRequest, headers)
	}

	// make fleet add a FileVault profile
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "macos_settings": { "enable_disk_encryption": true } }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.MacOSSettings.EnableDiskEncryption)
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
		"mdm": { "macos_settings": { "custom_settings": ["foo", "bar"] } }
  }`), http.StatusOK, &acResp)
	assert.Equal(t, []string{"foo", "bar"}, acResp.MDM.MacOSSettings.CustomSettings)

	// check that they are returned by a GET /config
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.Equal(t, []string{"foo", "bar"}, acResp.MDM.MacOSSettings.CustomSettings)

	// patch without specifying the macos custom settings fields and an unrelated
	// field, should not remove them
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "macos_settings": {"enable_disk_encryption": true} }
  }`), http.StatusOK, &acResp)
	assert.Equal(t, []string{"foo", "bar"}, acResp.MDM.MacOSSettings.CustomSettings)

	// patch with explicitly empty macos custom settings fields, would remove
	// them but this is a dry-run
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "macos_settings": { "custom_settings": null } }
  }`), http.StatusOK, &acResp, "dry_run", "true")
	assert.Equal(t, []string{"foo", "bar"}, acResp.MDM.MacOSSettings.CustomSettings)

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
		"mdm": { "macos_settings": { "enable_disk_encryption": true } }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.MacOSSettings.EnableDiskEncryption)
	enabledDiskActID := s.lastActivityMatches(fleet.ActivityTypeEnabledMacosDiskEncryption{}.ActivityName(),
		`{"team_id": null, "team_name": null}`, 0)

	// will have generated the macos config profile
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// check that they are returned by a GET /config
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.MacOSSettings.EnableDiskEncryption)

	// patch without specifying the macos disk encryption and an unrelated field,
	// should not alter it
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": { "macos_settings": {"custom_settings": ["a"]} }
		}`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.MacOSSettings.EnableDiskEncryption)
	assert.Equal(t, []string{"a"}, acResp.MDM.MacOSSettings.CustomSettings)
	s.lastActivityMatches(fleet.ActivityTypeEnabledMacosDiskEncryption{}.ActivityName(),
		``, enabledDiskActID)

	// patch with false, would reset it but this is a dry-run
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
				"mdm": { "macos_settings": { "enable_disk_encryption": false } }
		  }`), http.StatusOK, &acResp, "dry_run", "true")
	assert.True(t, acResp.MDM.MacOSSettings.EnableDiskEncryption)
	assert.Equal(t, []string{"a"}, acResp.MDM.MacOSSettings.CustomSettings)
	s.lastActivityMatches(fleet.ActivityTypeEnabledMacosDiskEncryption{}.ActivityName(),
		``, enabledDiskActID)

	// patch with false, resets it
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "macos_settings": { "enable_disk_encryption": false, "custom_settings": ["b"] } }
		  }`), http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.MacOSSettings.EnableDiskEncryption)
	assert.Equal(t, []string{"b"}, acResp.MDM.MacOSSettings.CustomSettings)
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
	assert.True(t, acResp.MDM.MacOSSettings.EnableDiskEncryption)
	assert.Equal(t, []string{"b"}, acResp.MDM.MacOSSettings.CustomSettings)

	// call update endpoint with no changes
	s.Do("PATCH", "/api/latest/fleet/mdm/apple/settings",
		fleet.MDMAppleSettingsPayload{}, http.StatusNoContent)
	s.lastActivityMatches(fleet.ActivityTypeEnabledMacosDiskEncryption{}.ActivityName(),
		``, enabledDiskActID)

	// the macos config profile still exists
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)

	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.MacOSSettings.EnableDiskEncryption)
	assert.Equal(t, []string{"b"}, acResp.MDM.MacOSSettings.CustomSettings)
}

func (s *integrationMDMTestSuite) TestMDMAppleDiskEncryptionAggregate() {
	t := s.T()
	ctx := context.Background()

	// no hosts with any disk encryption status's
	fvsResp := getMDMAppleFileVauleSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/filevault/summary", nil, http.StatusOK, &fvsResp)
	require.Equal(t, uint(0), fvsResp.Verifying)
	require.Equal(t, uint(0), fvsResp.Verified)
	require.Equal(t, uint(0), fvsResp.ActionRequired)
	require.Equal(t, uint(0), fvsResp.Enforcing)
	require.Equal(t, uint(0), fvsResp.Failed)
	require.Equal(t, uint(0), fvsResp.RemovingEnforcement)

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
		operationType fleet.MDMAppleOperationType,
		status *fleet.MDMAppleDeliveryStatus,
		decryptable bool,
	) {
		for _, host := range hosts {
			hostCmdUUID := uuid.New().String()
			err := s.ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
				{
					ProfileID:         prof.ProfileID,
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
			err = s.ds.SetOrUpdateHostDiskEncryptionKey(ctx, host.ID, "test-key")
			require.NoError(t, err)
			err = s.ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{host.ID}, decryptable, oneMinuteAfterThreshold)
			require.NoError(t, err)
		}
	}

	// hosts 1,2 have disk encryption "applied" status
	generateAggregateValue(hosts[0:2], fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryVerifying, true)
	fvsResp = getMDMAppleFileVauleSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/filevault/summary", nil, http.StatusOK, &fvsResp)
	require.Equal(t, uint(2), fvsResp.Verifying)
	require.Equal(t, uint(0), fvsResp.Verified)
	require.Equal(t, uint(0), fvsResp.ActionRequired)
	require.Equal(t, uint(0), fvsResp.Enforcing)
	require.Equal(t, uint(0), fvsResp.Failed)
	require.Equal(t, uint(0), fvsResp.RemovingEnforcement)

	// hosts 3,4 have disk encryption "action required" status
	generateAggregateValue(hosts[2:4], fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryVerifying, false)
	fvsResp = getMDMAppleFileVauleSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/filevault/summary", nil, http.StatusOK, &fvsResp)
	require.Equal(t, uint(2), fvsResp.Verifying)
	require.Equal(t, uint(0), fvsResp.Verified)
	require.Equal(t, uint(2), fvsResp.ActionRequired)
	require.Equal(t, uint(0), fvsResp.Enforcing)
	require.Equal(t, uint(0), fvsResp.Failed)
	require.Equal(t, uint(0), fvsResp.RemovingEnforcement)

	// hosts 5,6 have disk encryption "enforcing" status

	// host profiles status are `pending`
	generateAggregateValue(hosts[4:6], fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryPending, true)
	fvsResp = getMDMAppleFileVauleSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/filevault/summary", nil, http.StatusOK, &fvsResp)
	require.Equal(t, uint(2), fvsResp.Verifying)
	require.Equal(t, uint(0), fvsResp.Verified)
	require.Equal(t, uint(2), fvsResp.ActionRequired)
	require.Equal(t, uint(2), fvsResp.Enforcing)
	require.Equal(t, uint(0), fvsResp.Failed)
	require.Equal(t, uint(0), fvsResp.RemovingEnforcement)

	// host profiles status dont exist
	generateAggregateValue(hosts[4:6], fleet.MDMAppleOperationTypeInstall, nil, true)
	fvsResp = getMDMAppleFileVauleSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/filevault/summary", nil, http.StatusOK, &fvsResp)
	require.Equal(t, uint(2), fvsResp.Verifying)
	require.Equal(t, uint(0), fvsResp.Verified)
	require.Equal(t, uint(2), fvsResp.ActionRequired)
	require.Equal(t, uint(2), fvsResp.Enforcing)
	require.Equal(t, uint(0), fvsResp.Failed)
	require.Equal(t, uint(0), fvsResp.RemovingEnforcement)

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
	fvsResp = getMDMAppleFileVauleSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/filevault/summary", nil, http.StatusOK, &fvsResp)
	require.Equal(t, uint(2), fvsResp.Verifying)
	require.Equal(t, uint(0), fvsResp.Verified)
	require.Equal(t, uint(2), fvsResp.ActionRequired)
	require.Equal(t, uint(2), fvsResp.Enforcing)
	require.Equal(t, uint(0), fvsResp.Failed)
	require.Equal(t, uint(0), fvsResp.RemovingEnforcement)

	// hosts 7,8 have disk encryption "failed" status
	generateAggregateValue(hosts[6:8], fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryFailed, true)
	fvsResp = getMDMAppleFileVauleSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/filevault/summary", nil, http.StatusOK, &fvsResp)
	require.Equal(t, uint(2), fvsResp.Verifying)
	require.Equal(t, uint(0), fvsResp.Verified)

	require.Equal(t, uint(2), fvsResp.ActionRequired)
	require.Equal(t, uint(2), fvsResp.Enforcing)
	require.Equal(t, uint(2), fvsResp.Failed)
	require.Equal(t, uint(0), fvsResp.RemovingEnforcement)

	// hosts 9,10 have disk encryption "removing enforcement" status
	generateAggregateValue(hosts[8:10], fleet.MDMAppleOperationTypeRemove, &fleet.MDMAppleDeliveryPending, true)
	fvsResp = getMDMAppleFileVauleSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/filevault/summary", nil, http.StatusOK, &fvsResp)
	require.Equal(t, uint(2), fvsResp.Verifying)
	require.Equal(t, uint(0), fvsResp.Verified)
	require.Equal(t, uint(2), fvsResp.ActionRequired)
	require.Equal(t, uint(2), fvsResp.Enforcing)
	require.Equal(t, uint(2), fvsResp.Failed)
	require.Equal(t, uint(2), fvsResp.RemovingEnforcement)

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
	generateAggregateValue(hosts[0:2], fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryVerifying, true)
	fvsResp = getMDMAppleFileVauleSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/filevault/summary", nil, http.StatusOK, &fvsResp, "team_id", strconv.Itoa(int(tm.ID)))
	require.Equal(t, uint(2), fvsResp.Verifying)
	require.Equal(t, uint(0), fvsResp.Verified)
	require.Equal(t, uint(0), fvsResp.ActionRequired)
	require.Equal(t, uint(0), fvsResp.Enforcing)
	require.Equal(t, uint(0), fvsResp.Failed)
	require.Equal(t, uint(0), fvsResp.RemovingEnforcement)

	// verified status for host 1
	require.NoError(t, s.ds.UpdateVerificationHostMacOSProfiles(ctx, hosts[0], []*fleet.HostMacOSProfile{{Identifier: prof.Identifier, DisplayName: prof.Name, InstallDate: time.Now()}}))
	fvsResp = getMDMAppleFileVauleSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/filevault/summary", nil, http.StatusOK, &fvsResp, "team_id", strconv.Itoa(int(tm.ID)))
	require.Equal(t, uint(2), fvsResp.Verifying)
	require.Equal(t, uint(0), fvsResp.Verified)
	require.Equal(t, uint(0), fvsResp.ActionRequired)
	require.Equal(t, uint(0), fvsResp.Enforcing)
	require.Equal(t, uint(0), fvsResp.Failed)
	require.Equal(t, uint(0), fvsResp.RemovingEnforcement)
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
			MacOSSettings: map[string]interface{}{"custom_settings": []string{"foo", "bar"}},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	// retrieving the team returns the custom macos settings
	var teamResp getTeamResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, []string{"foo", "bar"}, teamResp.Team.Config.MDM.MacOSSettings.CustomSettings)

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
	assert.Contains(t, errMsg, `invalid value type at 'macos_settings.custom_settings': expected array of strings but got bool`)

	// apply without custom macos settings specified and unrelated field, should
	// not replace existing settings
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSSettings: map[string]interface{}{"enable_disk_encryption": false},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, []string{"foo", "bar"}, teamResp.Team.Config.MDM.MacOSSettings.CustomSettings)

	// apply with explicitly empty custom macos settings would clear the existing
	// settings, but dry-run
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSSettings: map[string]interface{}{"custom_settings": []string{}},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, "dry_run", "true")
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, []string{"foo", "bar"}, teamResp.Team.Config.MDM.MacOSSettings.CustomSettings)

	// apply with explicitly empty custom macos settings clears the existing settings
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSSettings: map[string]interface{}{"custom_settings": []string{}},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, []string{}, teamResp.Team.Config.MDM.MacOSSettings.CustomSettings)
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
			MacOSSettings: map[string]interface{}{"enable_disk_encryption": true},
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
	require.True(t, teamResp.Team.Config.MDM.MacOSSettings.EnableDiskEncryption)

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
			MacOSSettings: map[string]interface{}{"custom_settings": []string{"a"}},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.True(t, teamResp.Team.Config.MDM.MacOSSettings.EnableDiskEncryption)
	require.Equal(t, []string{"a"}, teamResp.Team.Config.MDM.MacOSSettings.CustomSettings)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledMacosDiskEncryption{}.ActivityName(),
		``, lastDiskActID)

	// apply with false would clear the existing setting, but dry-run
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: teamName,
		MDM: fleet.TeamSpecMDM{
			MacOSSettings: map[string]interface{}{"enable_disk_encryption": false},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, "dry_run", "true")
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.True(t, teamResp.Team.Config.MDM.MacOSSettings.EnableDiskEncryption)
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
	require.False(t, teamResp.Team.Config.MDM.MacOSSettings.EnableDiskEncryption)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeDisabledMacosDiskEncryption{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q}`, team.ID, teamName), 0)

	// macos config profile deleted
	s.assertConfigProfilesByIdentifier(ptr.Uint(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, false)

	// modify team's disk encryption via ModifyTeam endpoint
	var modResp teamResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		MDM: &fleet.TeamPayloadMDM{
			MacOSSettings: &fleet.MacOSSettings{EnableDiskEncryption: true},
		},
	}, http.StatusOK, &modResp)
	require.True(t, modResp.Team.Config.MDM.MacOSSettings.EnableDiskEncryption)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledMacosDiskEncryption{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q}`, team.ID, teamName), 0)

	// macos config profile created
	s.assertConfigProfilesByIdentifier(ptr.Uint(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// modify team's disk encryption and description via ModifyTeam endpoint
	modResp = teamResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Description: ptr.String("foobar"),
		MDM: &fleet.TeamPayloadMDM{
			MacOSSettings: &fleet.MacOSSettings{EnableDiskEncryption: false},
		},
	}, http.StatusOK, &modResp)
	require.False(t, modResp.Team.Config.MDM.MacOSSettings.EnableDiskEncryption)
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
	require.True(t, teamResp.Team.Config.MDM.MacOSSettings.EnableDiskEncryption)

	// use the MDM settings endpoint with no changes
	s.Do("PATCH", "/api/latest/fleet/mdm/apple/settings",
		fleet.MDMAppleSettingsPayload{TeamID: ptr.Uint(team.ID)}, http.StatusNoContent)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledMacosDiskEncryption{}.ActivityName(),
		``, lastDiskActID)

	// macos config profile still exists
	s.assertConfigProfilesByIdentifier(ptr.Uint(team.ID), mobileconfig.FleetFileVaultPayloadIdentifier, true)

	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.True(t, teamResp.Team.Config.MDM.MacOSSettings.EnableDiskEncryption)

	// use the MDM settings endpoint with an unknown team id
	s.Do("PATCH", "/api/latest/fleet/mdm/apple/settings",
		fleet.MDMAppleSettingsPayload{TeamID: ptr.Uint(9999)}, http.StatusNotFound)
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
			mobileconfigForTestWithContent("N1", "I1", "II1", p),
		}}, http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)))
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, fmt.Sprintf("Validation Failed: unsupported PayloadType(s): %s", p))
	}

	// payloads with reserved identifiers
	for p := range mobileconfig.FleetPayloadIdentifiers() {
		res := s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{
			mobileconfigForTestWithContent("N1", "I1", p, "random"),
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

func (s *integrationMDMTestSuite) TestHostMDMProfilesStatus() {
	t := s.T()
	ctx := context.Background()

	createManualMDMEnrollWithOrbit := func(secret string) *fleet.Host {
		// orbit enrollment happens before mdm enrollment, otherwise the host would
		// always receive the "no team" profiles on mdm enrollment since it would
		// not be part of any team yet (team assignment is done when it enrolls
		// with orbit).
		mdmDevice := mdmtest.NewTestMDMClientDirect(mdmtest.EnrollInfo{
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

		err = mdmDevice.Enroll()
		require.NoError(t, err)

		return h
	}

	triggerReconcileProfiles := func() {
		ch := make(chan bool)
		s.onProfileScheduleDone = func() { close(ch) }
		_, err := s.profileSchedule.Trigger()
		require.NoError(t, err)
		<-ch
		// this will only mark them as "pending", as the response to confirm
		// profile deployment is asynchronous, so we simulate it here by
		// updating any "pending" (not NULL) profiles to "verifying"
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `UPDATE host_mdm_apple_profiles SET status = ? WHERE status = ?`, fleet.MacOSSettingsVerifying, fleet.MacOSSettingsPending)
			return err
		})
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
			{Identifier: "G1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "G2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
		},
		h2: {
			{Identifier: "G1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "G2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
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
			{Identifier: "T1.1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "T1.2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
		},
		h4: {
			{Identifier: "T1.1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "T1.2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
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
			{Identifier: "G1", OperationType: fleet.MDMAppleOperationTypeRemove, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "G2", OperationType: fleet.MDMAppleOperationTypeRemove, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "T2.1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
		},
		h2: {
			{Identifier: "G1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: "G2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
		},
	})

	// switch a team host (h3) to another team (tm2)
	s.DoJSON("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &tm2.ID, HostIDs: []uint{h3.ID}}, http.StatusOK, &moveHostResp)
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h3: {
			{Identifier: "T1.1", OperationType: fleet.MDMAppleOperationTypeRemove, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "T1.2", OperationType: fleet.MDMAppleOperationTypeRemove, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "T2.1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
		},
		h4: {
			{Identifier: "T1.1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: "T1.2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
		},
	})

	// switch a team host (h4) to no team
	s.DoJSON("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: nil, HostIDs: []uint{h4.ID}}, http.StatusOK, &moveHostResp)
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h3: {
			{Identifier: "T1.1", OperationType: fleet.MDMAppleOperationTypeRemove, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "T1.2", OperationType: fleet.MDMAppleOperationTypeRemove, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "T2.1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
		},
		h4: {
			{Identifier: "T1.1", OperationType: fleet.MDMAppleOperationTypeRemove, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "T1.2", OperationType: fleet.MDMAppleOperationTypeRemove, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "G1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "G2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
		},
	})

	// apply the pending profiles
	triggerReconcileProfiles()

	// add a profile to no team (h2 and h4 are now part of no team)
	body, headers := generateNewProfileMultipartRequest(t, nil,
		"some_name", mobileconfigForTest("G3", "G3"), s.token)
	s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusOK, headers)
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h2: {
			{Identifier: "G1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: "G2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: "G3", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
		},
		h4: {
			{Identifier: "G1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: "G2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: "G3", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
		},
	})

	// add a profile to team 2 (h1 and h3 are now part of team 2)
	body, headers = generateNewProfileMultipartRequest(t, &tm2.ID,
		"some_name", mobileconfigForTest("T2.2", "T2.2"), s.token)
	s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusOK, headers)
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: "T2.1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: "T2.2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
		},
		h3: {
			{Identifier: "T2.1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: "T2.2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
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
			{Identifier: "G1", OperationType: fleet.MDMAppleOperationTypeRemove, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "G2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: "G3", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
		},
		h4: {
			{Identifier: "G1", OperationType: fleet.MDMAppleOperationTypeRemove, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "G2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: "G3", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
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
			{Identifier: "T2.1", OperationType: fleet.MDMAppleOperationTypeRemove, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "T2.2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
		},
		h3: {
			{Identifier: "T2.1", OperationType: fleet.MDMAppleOperationTypeRemove, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "T2.2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
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
			{Identifier: "G2", OperationType: fleet.MDMAppleOperationTypeRemove, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "G2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "G3", OperationType: fleet.MDMAppleOperationTypeRemove, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "G4", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
		},
		h4: {
			{Identifier: "G2", OperationType: fleet.MDMAppleOperationTypeRemove, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "G2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "G3", OperationType: fleet.MDMAppleOperationTypeRemove, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "G4", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
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
			{Identifier: "T2.2", OperationType: fleet.MDMAppleOperationTypeRemove, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "T2.2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "T2.3", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
		},
		h3: {
			{Identifier: "T2.2", OperationType: fleet.MDMAppleOperationTypeRemove, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "T2.2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "T2.3", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
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
			{Identifier: "T2.2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: "T2.3", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
		},
		h2: {
			{Identifier: "G2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
		},
		h3: {
			{Identifier: "T2.2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: "T2.3", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
		},
		h4: {
			{Identifier: "G2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
		},
	})

	// delete team 2 (h1 and h3 are part of that team)
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d", tm2.ID), nil, http.StatusOK)
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: "T2.2b", OperationType: fleet.MDMAppleOperationTypeRemove, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "T2.3", OperationType: fleet.MDMAppleOperationTypeRemove, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "G2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "G4", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
		},
		h3: {
			{Identifier: "T2.2b", OperationType: fleet.MDMAppleOperationTypeRemove, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "T2.3", OperationType: fleet.MDMAppleOperationTypeRemove, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "G2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: "G4", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
		},
	})

	// apply the pending profiles
	triggerReconcileProfiles()

	// all profiles now verifying
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: "G2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
		},
		h2: {
			{Identifier: "G2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
		},
		h3: {
			{Identifier: "G2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
		},
		h4: {
			{Identifier: "G2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
		},
	})

	// h1 verified one of the profiles
	require.NoError(t, s.ds.UpdateVerificationHostMacOSProfiles(context.Background(), h1, []*fleet.HostMacOSProfile{
		{Identifier: "G2b", DisplayName: "G2b", InstallDate: time.Now()},
	}))
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: "G2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerified},
			{Identifier: "G4", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
		},
		h2: {
			{Identifier: "G2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
		},
		h3: {
			{Identifier: "G2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
		},
		h4: {
			{Identifier: "G2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: "G4", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryVerifying},
		},
	})
}

func (s *integrationMDMTestSuite) TestFleetdConfiguration() {
	t := s.T()
	s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetdConfigPayloadIdentifier, false)

	triggerSchedule := func() {
		ch := make(chan bool)
		s.onProfileScheduleDone = func() { close(ch) }
		_, err := s.profileSchedule.Trigger()
		require.NoError(t, err)
		<-ch
	}

	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: t.Name()}},
		},
	}, http.StatusOK, &applyResp)

	// a new fleetd configuration profile for "no team" is created
	triggerSchedule()
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
	triggerSchedule()
	p := s.assertConfigProfilesByIdentifier(&tm.ID, mobileconfig.FleetdConfigPayloadIdentifier, true)
	require.Contains(t, string(p.Mobileconfig), t.Name())

	// create an enroll secret for the team
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name:    tm.Name,
		Secrets: []fleet.EnrollSecret{{Secret: t.Name() + "team-secret"}},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	// a new fleetd configuration profile for that team is created
	triggerSchedule()
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
	mdmDevice := mdmtest.NewTestMDMClientDirect(mdmtest.EnrollInfo{
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
			Command:   base64Cmd(newRawCmd(uuid1)),
			DeviceIDs: []string{"no-such-host"},
		}, http.StatusNotFound)

	// get command results returns 404, that command does not exist
	var cmdResResp getMDMAppleCommandResultsResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/commandresults", nil, http.StatusNotFound, &cmdResResp, "command_uuid", uuid1)

	// list commands returns empty set
	var listCmdResp listMDMAppleCommandsResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/commands", nil, http.StatusOK, &listCmdResp)
	require.Empty(t, listCmdResp.Results)

	// call with unenrolled host UUID
	res := s.Do("POST", "/api/latest/fleet/mdm/apple/enqueue",
		enqueueMDMAppleCommandRequest{
			Command:   base64Cmd(newRawCmd(uuid.New().String())),
			DeviceIDs: []string{unenrolledHost.UUID},
		}, http.StatusConflict)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "at least one of the hosts is not enrolled in MDM")

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
	require.Empty(t, resp.FailedUUIDs)
	require.Equal(t, "ProfileList", resp.RequestType)

	// the command exists but no results yet
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/commandresults", nil, http.StatusOK, &cmdResResp, "command_uuid", uuid2)
	require.Len(t, cmdResResp.Results, 0)

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
	require.Equal(t, &fleet.MDMAppleCommandResult{
		DeviceID:    mdmDevice.UUID,
		CommandUUID: uuid2,
		Status:      "Acknowledged",
		RequestType: "ProfileList",
		Result:      []byte(rawCmd),
		Hostname:    "test-host",
	}, cmdResResp.Results[0])

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
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/bootstrap/0/metadata", nil, http.StatusOK, &metadataResp)
	require.Equal(t, metadataResp.MDMAppleBootstrapPackage.Name, "pkg.pkg")
	require.NotEmpty(t, metadataResp.MDMAppleBootstrapPackage.Sha256, "")
	require.NotEmpty(t, metadataResp.MDMAppleBootstrapPackage.Token)

	// download a package, wrong token
	var downloadResp downloadBootstrapPackageResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/bootstrap?token=bad", nil, http.StatusNotFound, &downloadResp)

	resp := s.DoRaw("GET", fmt.Sprintf("/api/latest/fleet/mdm/apple/bootstrap?token=%s", metadataResp.MDMAppleBootstrapPackage.Token), nil, http.StatusOK)
	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.EqualValues(t, signedPkg, respBytes)

	// missing package
	metadataResp = bootstrapPackageMetadataResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/bootstrap/1/metadata", nil, http.StatusNotFound, &metadataResp)

	// delete package
	var deleteResp deleteBootstrapPackageResponse
	s.DoJSON("DELETE", "/api/latest/fleet/mdm/apple/bootstrap/0", nil, http.StatusOK, &deleteResp)
	// check the activity log
	s.lastActivityMatches(
		fleet.ActivityTypeDeletedBootstrapPackage{}.ActivityName(),
		`{"bootstrap_package_name": "pkg.pkg", "team_id": null, "team_name": null}`,
		0,
	)

	metadataResp = bootstrapPackageMetadataResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/bootstrap/0/metadata", nil, http.StatusNotFound, &metadataResp)
	// trying to delete again is a bad request
	s.DoJSON("DELETE", "/api/latest/fleet/mdm/apple/bootstrap/0", nil, http.StatusNotFound, &deleteResp)
}

func (s *integrationMDMTestSuite) TestBootstrapPackageStatus() {
	t := s.T()
	pkg, err := os.ReadFile(filepath.Join("testdata", "bootstrap-packages", "signed.pkg"))
	require.NoError(t, err)

	// upload a bootstrap package for "no team"
	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Bytes: pkg, Name: "pkg.pkg", TeamID: 0}, http.StatusOK, "")

	// get package metadata
	var metadataResp bootstrapPackageMetadataResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/bootstrap/0/metadata", nil, http.StatusOK, &metadataResp)
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
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/mdm/apple/bootstrap/%d/metadata", team.ID), nil, http.StatusOK, &metadataResp)
	teamBootstrapPackage := metadataResp.MDMAppleBootstrapPackage

	type deviceWithResponse struct {
		bootstrapResponse string
		device            *mdmtest.TestMDMClient
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
	mdmEnrollInfo := mdmtest.EnrollInfo{
		SCEPChallenge: s.fleetCfg.MDM.AppleSCEPChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}
	noTeamDevices := []deviceWithResponse{
		{"Acknowledge", mdmtest.NewTestMDMClientDirect(mdmEnrollInfo)},
		{"Acknowledge", mdmtest.NewTestMDMClientDirect(mdmEnrollInfo)},
		{"Acknowledge", mdmtest.NewTestMDMClientDirect(mdmEnrollInfo)},
		{"Error", mdmtest.NewTestMDMClientDirect(mdmEnrollInfo)},
		{"Offline", mdmtest.NewTestMDMClientDirect(mdmEnrollInfo)},
		{"Offline", mdmtest.NewTestMDMClientDirect(mdmEnrollInfo)},
		{"Pending", mdmtest.NewTestMDMClientDirect(mdmEnrollInfo)},
		{"Pending", mdmtest.NewTestMDMClientDirect(mdmEnrollInfo)},
	}

	teamDevices := []deviceWithResponse{
		{"Acknowledge", mdmtest.NewTestMDMClientDirect(mdmEnrollInfo)},
		{"Acknowledge", mdmtest.NewTestMDMClientDirect(mdmEnrollInfo)},
		{"Error", mdmtest.NewTestMDMClientDirect(mdmEnrollInfo)},
		{"Error", mdmtest.NewTestMDMClientDirect(mdmEnrollInfo)},
		{"Error", mdmtest.NewTestMDMClientDirect(mdmEnrollInfo)},
		{"Offline", mdmtest.NewTestMDMClientDirect(mdmEnrollInfo)},
		{"Pending", mdmtest.NewTestMDMClientDirect(mdmEnrollInfo)},
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
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/bootstrap/summary", nil, http.StatusOK, &summaryResp)
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
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/mdm/apple/bootstrap/summary?team_id=%d", team.ID), nil, http.StatusOK, &summaryResp)
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
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/bootstrap/summary", nil, http.StatusOK, &summaryResp)
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
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/mdm/apple/bootstrap/summary?team_id=%d", team.ID), nil, http.StatusOK, &summaryResp)
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
	metadataResp := getMDMAppleEULAMetadataResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/setup/eula/metadata", nil, http.StatusNotFound, &metadataResp)

	// trying to upload a file that is not a PDF fails
	s.uploadEULA(&fleet.MDMAppleEULA{Bytes: []byte("should-fail"), Name: "should-fail.pdf"}, http.StatusBadRequest, "invalid file type")
	// trying to upload an empty file fails
	s.uploadEULA(&fleet.MDMAppleEULA{Bytes: []byte{}, Name: "should-fail.pdf"}, http.StatusBadRequest, "invalid file type")

	// admin is able to upload a new EULA
	s.uploadEULA(&fleet.MDMAppleEULA{Bytes: pdfBytes, Name: pdfName}, http.StatusOK, "")

	// get EULA metadata
	metadataResp = getMDMAppleEULAMetadataResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/setup/eula/metadata", nil, http.StatusOK, &metadataResp)
	require.NotEmpty(t, metadataResp.MDMAppleEULA.Token)
	require.NotEmpty(t, metadataResp.MDMAppleEULA.CreatedAt)
	require.Equal(t, pdfName, metadataResp.MDMAppleEULA.Name)
	eulaToken := metadataResp.Token

	// download EULA
	resp := s.DoRaw("GET", fmt.Sprintf("/api/latest/fleet/mdm/apple/setup/eula/%s", eulaToken), nil, http.StatusOK)
	require.EqualValues(t, len(pdfBytes), resp.ContentLength)
	require.Equal(t, "application/pdf", resp.Header.Get("content-type"))
	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.EqualValues(t, pdfBytes, respBytes)

	// try to download EULA with a bad token
	var downloadResp downloadBootstrapPackageResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/setup/eula/bad-token", nil, http.StatusNotFound, &downloadResp)

	// trying to upload any EULA without deleting the previous one first results in an error
	s.uploadEULA(&fleet.MDMAppleEULA{Bytes: pdfBytes, Name: "should-fail.pdf"}, http.StatusConflict, "")

	// delete EULA
	var deleteResp deleteMDMAppleEULAResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/mdm/apple/setup/eula/%s", eulaToken), nil, http.StatusOK, &deleteResp)
	metadataResp = getMDMAppleEULAMetadataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/mdm/apple/setup/eula/%s", eulaToken), nil, http.StatusNotFound, &metadataResp)
	// trying to delete again is a bad request
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/mdm/apple/setup/eula/%s", eulaToken), nil, http.StatusNotFound, &deleteResp)
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
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), h.ID, isServer, enrolled, mdmURL, installedFromDEP, mdmName))
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusBadRequest)
	require.False(t, webhookCalled)

	// host is not DEP so migration is not allowed
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), h.ID, !isServer, enrolled, mdmURL, !installedFromDEP, mdmName))
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusBadRequest)
	require.False(t, webhookCalled)

	// host is not enrolled to MDM so migration is not allowed
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), h.ID, !isServer, !enrolled, mdmURL, installedFromDEP, mdmName))
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusBadRequest)
	require.False(t, webhookCalled)

	// host is already enrolled to Fleet MDM so migration is not allowed
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), h.ID, !isServer, enrolled, mdmURL, installedFromDEP, fleet.WellKnownMDMFleet))
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusBadRequest)
	require.False(t, webhookCalled)

	// up to this point, the refetch critical queries timestamp has not been set
	// on the host.
	h, err := s.ds.Host(context.Background(), h.ID)
	require.NoError(t, err)
	require.Nil(t, h.RefetchCriticalQueriesUntil)

	// host is enrolled to a third-party MDM but hasn't been assigned in
	// ABM yet, so migration is not allowed
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), h.ID, !isServer, enrolled, mdmURL, installedFromDEP, mdmName))
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
	ch := make(chan bool)
	s.onDEPScheduleDone = func() { close(ch) }
	_, err = s.depSchedule.Trigger()
	require.NoError(t, err)
	<-ch

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

	// try to set the configuration_web_url key
	tmProf = `{"configuration_web_url": "https://example.com"}`
	res := s.Do("POST", "/api/latest/fleet/mdm/apple/enrollment_profile", createMDMAppleSetupAssistantRequest{
		TeamID:            &tm.ID,
		Name:              "team3",
		EnrollmentProfile: json.RawMessage(tmProf),
	}, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `The automatic enrollment profile can’t include configuration_web_url.`)
	s.lastActivityMatches(fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
		fmt.Sprintf(`{"name": "team2", "team_id": %d, "team_name": %q}`, tm.ID, tm.Name), latestChangedActID)

	// try to set the url
	tmProf = `{"url": "https://example.com"}`
	res = s.Do("POST", "/api/latest/fleet/mdm/apple/enrollment_profile", createMDMAppleSetupAssistantRequest{
		TeamID:            &tm.ID,
		Name:              "team5",
		EnrollmentProfile: json.RawMessage(tmProf),
	}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
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
		gotProfs, err := ds.GetHostMDMProfiles(ctx, h.UUID)
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

// generates the body and headers part of a multipart request ready to be
// used via s.DoRawWithHeaders to POST /api/_version_/fleet/mdm/apple/profiles.
func generateNewProfileMultipartRequest(t *testing.T, tmID *uint,
	fileName string, fileContent []byte, token string,
) (*bytes.Buffer, map[string]string) {
	var body bytes.Buffer

	writer := multipart.NewWriter(&body)
	if tmID != nil {
		err := writer.WriteField("team_id", fmt.Sprintf("%d", *tmID))
		require.NoError(t, err)
	}

	ff, err := writer.CreateFormFile("profile", fileName)
	require.NoError(t, err)
	_, err = io.Copy(ff, bytes.NewReader(fileContent))
	require.NoError(t, err)
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

	res := s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/bootstrap", b.Bytes(), expectedStatus, headers)

	if wantErr != "" {
		errMsg := extractServerErrorText(res.Body)
		assert.Contains(t, errMsg, wantErr)
	}
}

func (s *integrationMDMTestSuite) uploadEULA(
	eula *fleet.MDMAppleEULA,
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

	res := s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/setup/eula", b.Bytes(), expectedStatus, headers)

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
		"mdm": { "macos_settings": { "enable_disk_encryption": true } }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.MacOSSettings.EnableDiskEncryption)

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
			MacOSSettings: map[string]interface{}{
				"enable_disk_encryption": true,
				"custom_settings":        []interface{}{"foo", "bar"},
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
			MacOSSettings: map[string]interface{}{
				"enable_disk_encryption": true,
				"custom_settings":        []interface{}{"foo", "bar"},
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

	mdmDevice := mdmtest.NewTestMDMClientDirect(mdmtest.EnrollInfo{
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
	ch := make(chan bool)
	s.onDEPScheduleDone = func() { close(ch) }
	_, err := s.depSchedule.Trigger()
	require.NoError(t, err)
	<-ch

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
	require.Contains(t, lastSubmittedProfile.URL, acResp.ServerSettings.ServerURL+"/api/mdm/apple/enroll?token=")
	require.Equal(t, acResp.ServerSettings.ServerURL+"/mdm/sso", lastSubmittedProfile.ConfigurationWebURL)

	// patch without specifying the mdm sso settings fields and an unrelated
	// field, should not remove them
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "macos_settings": {"enable_disk_encryption": true} }
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

	// set-up valid settings
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"server_settings": {"server_url": "https://localhost:8080"},
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

	s.runWorker()
	require.Contains(t, lastSubmittedProfile.URL, acResp.ServerSettings.ServerURL+"/api/mdm/apple/enroll?token=")
	require.Equal(t, acResp.ServerSettings.ServerURL+"/mdm/sso", lastSubmittedProfile.ConfigurationWebURL)

	res := s.LoginMDMSSOUser("sso_user", "user123#")
	require.NotEmpty(t, res.Header.Get("Location"))
	require.Equal(t, http.StatusTemporaryRedirect, res.StatusCode)

	u, err := url.Parse(res.Header.Get("Location"))
	require.NoError(t, err)
	q := u.Query()
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
			q.Get("enrollment_reference"),
		),
	)

	// upload an EULA
	pdfBytes := []byte("%PDF-1.pdf-contents")
	pdfName := "eula.pdf"
	s.uploadEULA(&fleet.MDMAppleEULA{Bytes: pdfBytes, Name: pdfName}, http.StatusOK, "")

	res = s.LoginMDMSSOUser("sso_user", "user123#")
	require.NotEmpty(t, res.Header.Get("Location"))
	require.Equal(t, http.StatusTemporaryRedirect, res.StatusCode)
	u, err = url.Parse(res.Header.Get("Location"))
	require.NoError(t, err)
	q = u.Query()
	// with an EULA uploaded, all values are present
	require.True(t, q.Has("eula_token"))
	require.True(t, q.Has("profile_token"))
	require.True(t, q.Has("enrollment_reference"))
	require.False(t, q.Has("error"))
	// the url retrieves a valid profile
	prof := s.downloadAndVerifyEnrollmentProfile(
		fmt.Sprintf(
			"/api/mdm/apple/enroll?token=%s&enrollment_reference=%s",
			q.Get("profile_token"),
			q.Get("enrollment_reference"),
		),
	)
	// the url retrieves a valid EULA
	resp := s.DoRaw("GET", "/api/latest/fleet/mdm/apple/setup/eula/"+q.Get("eula_token"), nil, http.StatusOK)
	require.EqualValues(t, len(pdfBytes), resp.ContentLength)
	require.Equal(t, "application/pdf", resp.Header.Get("content-type"))
	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.EqualValues(t, pdfBytes, respBytes)

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

	// changing the server URL also updates the remote DEP profile
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
                "server_settings": {"server_url": "https://example.com"}
	}`), http.StatusOK, &acResp)

	s.runWorker()
	require.Contains(t, lastSubmittedProfile.URL, "https://example.com/api/mdm/apple/enroll?token=")
	require.Equal(t, "https://example.com/mdm/sso", lastSubmittedProfile.ConfigurationWebURL)

	// hitting the callback with an invalid session id redirects the user to the UI
	rawSSOResp := base64.StdEncoding.EncodeToString([]byte(`<samlp:Response ID="_7822b394622740aa92878ca6c7d1a28c53e80ec5ef"></samlp:Response>`))
	res = s.DoRawNoAuth("POST", "/api/v1/fleet/mdm/sso/callback?SAMLResponse="+url.QueryEscape(rawSSOResp), nil, http.StatusTemporaryRedirect)
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
	URL string
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

	var profile enrollmentProfile
	require.NoError(t, plist.Unmarshal(body, &profile))

	for _, p := range profile.PayloadContent {
		switch p.PayloadType {
		case "com.apple.security.scep":
			require.NotEmpty(t, p.PayloadContent.URL)
		case "com.apple.mdm":
			require.NotEmpty(t, p.ServerURL)
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
		ch := make(chan bool)
		s.onDEPScheduleDone = func() { close(ch) }
		_, err := s.depSchedule.Trigger()
		require.NoError(t, err)
		<-ch

		// simulate that the device is enrolled in a third-party MDM and DEP capable
		err = s.ds.SetOrUpdateMDMData(
			ctx,
			host.ID,
			false,
			true,
			"https://simplemdm.com",
			true,
			fleet.WellKnownMDMSimpleMDM,
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
	assert.True(t, acResp.MDMEnabled)
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
		err := s.ds.SetOrUpdateMDMData(ctx, h.ID, meta.isServer, meta.enrolledName != "", "https://example.com", false, meta.enrolledName)
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
	assert.True(t, acResp.MDMEnabled)
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
	err = s.ds.SetOrUpdateMDMData(ctx, noTeamHost.ID, false, true, "https://example.com", false, fleet.WellKnownMDMFleet)
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
	mdmDevice := mdmtest.NewTestMDMClientDirect(mdmtest.EnrollInfo{
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
	mdmDevice = mdmtest.NewTestMDMClientDirect(mdmtest.EnrollInfo{
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

	require.Contains(t, resp.Header["Content-Type"], microsoft_mdm.SoapContentType)

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

	require.Contains(t, resp.Header["Content-Type"], microsoft_mdm.SoapContentType)

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

	require.Contains(t, resp.Header["Content-Type"], microsoft_mdm.SoapContentType)

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
	windowsHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		ID:            1,
		OsqueryHostID: ptr.String("Desktop-ABCQWE"),
		NodeKey:       ptr.String("Desktop-ABCQWE"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.not.enrolled", s.T().Name()),
		Platform:      "windows",
	})
	require.NoError(t, err)

	// Preparing the GetPolicies Request message
	encodedBinToken, err := GetEncodedBinarySecurityToken(fleet.WindowsMDMProgrammaticEnrollmentType, windowsHost.UUID)
	require.NoError(t, err)

	requestBytes, err := s.newGetPoliciesMsg(true, encodedBinToken)
	require.NoError(t, err)

	resp := s.DoRaw("POST", microsoft_mdm.MDE2PolicyPath, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], microsoft_mdm.SoapContentType)

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

	require.Contains(t, resp.Header["Content-Type"], microsoft_mdm.SoapContentType)

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
	encodedBinToken, err := GetEncodedBinarySecurityToken(fleet.WindowsMDMProgrammaticEnrollmentType, "not_exists")
	require.NoError(t, err)

	requestBytes, err := s.newGetPoliciesMsg(true, encodedBinToken)
	require.NoError(t, err)

	resp := s.DoRaw("POST", microsoft_mdm.MDE2PolicyPath, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], microsoft_mdm.SoapContentType)

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
	linuxHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		ID:            1,
		OsqueryHostID: ptr.String("Ubuntu01"),
		NodeKey:       ptr.String("Ubuntu01"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.not.enrolled", s.T().Name()),
		Platform:      "linux",
	})
	require.NoError(t, err)

	// Preparing the GetPolicies Request message
	encodedBinToken, err := GetEncodedBinarySecurityToken(fleet.WindowsMDMProgrammaticEnrollmentType, linuxHost.UUID)
	require.NoError(t, err)

	requestBytes, err := s.newGetPoliciesMsg(true, encodedBinToken)
	require.NoError(t, err)

	resp := s.DoRaw("POST", microsoft_mdm.MDE2PolicyPath, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], microsoft_mdm.SoapContentType)

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

	// create a new Host to get the UUID on the DB
	windowsHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		ID:            1,
		OsqueryHostID: ptr.String("Desktop-ABCQWE"),
		NodeKey:       ptr.String("Desktop-ABCQWE"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.not.enrolled", s.T().Name()),
		Platform:      "windows",
	})
	require.NoError(t, err)

	// Delete the host from the list of MDM enrolled devices if present
	_ = s.ds.MDMWindowsDeleteEnrolledDevice(context.Background(), windowsHost.UUID)

	// Preparing the RequestSecurityToken Request message
	encodedBinToken, err := GetEncodedBinarySecurityToken(fleet.WindowsMDMProgrammaticEnrollmentType, windowsHost.UUID)
	require.NoError(t, err)

	requestBytes, err := s.newSecurityTokenMsg(encodedBinToken, true, false)
	require.NoError(t, err)

	resp := s.DoRaw("POST", microsoft_mdm.MDE2EnrollPath, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], microsoft_mdm.SoapContentType)

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
}

func (s *integrationMDMTestSuite) TestValidRequestSecurityTokenRequestWithAzureToken() {
	t := s.T()

	// Preparing the SecurityToken Request message with Azure JWT token
	azureADTok := "ZXlKMGVYQWlPaUpLVjFRaUxDSmhiR2NpT2lKU1V6STFOaUlzSW5nMWRDSTZJaTFMU1ROUk9XNU9VamRpVW05bWVHMWxXbTlZY1dKSVdrZGxkeUlzSW10cFpDSTZJaTFMU1ROUk9XNU9VamRpVW05bWVHMWxXbTlZY1dKSVdrZGxkeUo5LmV5SmhkV1FpT2lKb2RIUndjem92TDIxaGNtTnZjMnhoWW5NdWIzSm5MeUlzSW1semN5STZJbWgwZEhCek9pOHZjM1J6TG5kcGJtUnZkM011Ym1WMEwyWmhaVFZqTkdZekxXWXpNVGd0TkRRNE15MWlZelptTFRjMU9UVTFaalJoTUdFM01pOGlMQ0pwWVhRaU9qRTJPRGt4TnpBNE5UZ3NJbTVpWmlJNk1UWTRPVEUzTURnMU9Dd2laWGh3SWpveE5qZzVNVGMxTmpZeExDSmhZM0lpT2lJeElpd2lZV2x2SWpvaVFWUlJRWGt2T0ZSQlFVRkJOV2gwUTNFMGRERjNjbHBwUTIxQmVEQlpWaTloZGpGTVMwRkRPRXM1Vm10SGVtNUdXVGxzTUZoYWVrZHVha2N6VVRaMWVIUldNR3QxT1hCeFJXdFRZeUlzSW1GdGNpSTZXeUp3ZDJRaUxDSnljMkVpWFN3aVlYQndhV1FpT2lJeU9XUTVaV1E1T0MxaE5EWTVMVFExTXpZdFlXUmxNaTFtT1RneFltTXhaRFl3TldVaUxDSmhjSEJwWkdGamNpSTZJakFpTENKa1pYWnBZMlZwWkNJNkltRXhNMlkzWVdVd0xURXpPR0V0TkdKaU1pMDVNalF5TFRka09USXlaVGRqTkdGak15SXNJbWx3WVdSa2NpSTZJakU0Tmk0eE1pNHhPRGN1TWpZaUxDSnVZVzFsSWpvaVZHVnpkRTFoY21OdmMweGhZbk1pTENKdmFXUWlPaUpsTTJNMU5XVmtZeTFqTXpRNExUUTBNVFl0T0dZd05TMHlOVFJtWmpNd05qVmpOV1VpTENKd2QyUmZkWEpzSWpvaWFIUjBjSE02THk5d2IzSjBZV3d1YldsamNtOXpiMlowYjI1c2FXNWxMbU52YlM5RGFHRnVaMlZRWVhOemQyOXlaQzVoYzNCNElpd2ljbWdpT2lJd0xrRldTVUU0T0ZSc0xXaHFlbWN3VXpoaU0xZFdXREJ2UzJOdFZGRXpTbHB1ZUUxa1QzQTNUbVZVVm5OV2FYVkhOa0ZRYnk0aUxDSnpZM0FpT2lKdFpHMWZaR1ZzWldkaGRHbHZiaUlzSW5OMVlpSTZJa1pTUTJ4RldURk9ObXR2ZEdWblMzcFplV0pFTjJkdFdGbGxhVTVIUkZrd05FSjJOV3R6ZDJGeGJVRWlMQ0owYVdRaU9pSm1ZV1UxWXpSbU15MW1NekU0TFRRME9ETXRZbU0yWmkwM05UazFOV1kwWVRCaE56SWlMQ0oxYm1seGRXVmZibUZ0WlNJNkluUmxjM1JBYldGeVkyOXpiR0ZpY3k1dmNtY2lMQ0oxY0c0aU9pSjBaWE4wUUcxaGNtTnZjMnhoWW5NdWIzSm5JaXdpZFhScElqb2lNVGg2WkVWSU5UZFRSWFZyYWpseGJqRm9aMlJCUVNJc0luWmxjaUk2SWpFdU1DSjkuVG1FUlRsZktBdWo5bTVvQUc2UTBRblV4VEFEaTNFamtlNHZ3VXo3UTdqUUFVZVZGZzl1U0pzUXNjU2hFTXVxUmQzN1R2VlpQanljdEVoRFgwLVpQcEVVYUlSempuRVEyTWxvc21SZURYZzhrYkhNZVliWi1jb0ZucDEyQkVpQnpJWFBGZnBpaU1GRnNZZ0hSSF9tSWxwYlBlRzJuQ2p0LTZSOHgzYVA5QS1tM0J3eV91dnV0WDFNVEVZRmFsekhGa04wNWkzbjZRcjhURnlJQ1ZUYW5OanlkMjBBZFRMbHJpTVk0RVBmZzRaLThVVTctZkcteElycWVPUmVWTnYwOUFHV192MDd6UkVaNmgxVk9tNl9nelRGcElVVURuZFdabnFLTHlySDlkdkF3WnFFSG1HUmlTNElNWnRFdDJNTkVZSnhDWHhlSi1VbWZJdV9tUVhKMW9R"
	requestBytes, err := s.newSecurityTokenMsg(azureADTok, false, false)
	require.NoError(t, err)

	resp := s.DoRaw("POST", microsoft_mdm.MDE2EnrollPath, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], microsoft_mdm.SoapContentType)

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
}

func (s *integrationMDMTestSuite) TestInvalidRequestSecurityTokenRequestWithMissingAdditionalContext() {
	t := s.T()

	// create a new Host to get the UUID on the DB
	windowsHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		ID:            1,
		OsqueryHostID: ptr.String("Desktop-ABCQWE"),
		NodeKey:       ptr.String("Desktop-ABCQWE"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.not.enrolled", s.T().Name()),
		Platform:      "windows",
	})
	require.NoError(t, err)

	// Preparing the RequestSecurityToken Request message
	encodedBinToken, err := GetEncodedBinarySecurityToken(fleet.WindowsMDMProgrammaticEnrollmentType, windowsHost.UUID)
	require.NoError(t, err)

	requestBytes, err := s.newSecurityTokenMsg(encodedBinToken, true, true)
	require.NoError(t, err)

	resp := s.DoRaw("POST", microsoft_mdm.MDE2EnrollPath, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], microsoft_mdm.SoapContentType)

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

	require.Contains(t, resp.Header["Content-Type"], microsoft_mdm.WebContainerContentType)

	resTOCcontent := string(resBytes)
	require.Contains(t, resTOCcontent, "Microsoft.AAD.BrokerPlugin")
	require.Contains(t, resTOCcontent, "IsAccepted=true")
	require.Contains(t, resTOCcontent, "OpaqueBlob=")
}

func (s *integrationMDMTestSuite) TestValidSyncMLRequestNoAuth() {
	t := s.T()

	// Target Endpoint URL for the management endpoint
	targetEndpointURL := microsoft_mdm.MDE2ManagementPath

	// Preparing the SyncML request
	requestBytes, err := s.newSyncMLSessionMsg(targetEndpointURL)
	require.NoError(t, err)

	resp := s.DoRaw("POST", targetEndpointURL, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], microsoft_mdm.SyncMLContentType)

	// Checking if SyncML response can be unmarshalled to an golang type
	var xmlType interface{}
	err = xml.Unmarshal(resBytes, &xmlType)
	require.NoError(t, err)

	// Checking if SOAP response contains a valid RequestSecurityTokenResponseCollection message
	resSoapMsg := string(resBytes)
	require.True(t, s.isXMLTagPresent("SyncHdr", resSoapMsg))
	require.True(t, s.isXMLTagPresent("SyncBody", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("Exec", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("Add", resSoapMsg))
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
	tokType := microsoft_mdm.BinarySecurityAzureEnroll
	if deviceToken {
		tokType = microsoft_mdm.BinarySecurityDeviceEnroll
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
	tokType := microsoft_mdm.BinarySecurityAzureEnroll
	if deviceToken {
		tokType = microsoft_mdm.BinarySecurityDeviceEnroll
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

// TODO: Add support to add custom DeviceID when DeviceAuth is in place
func (s *integrationMDMTestSuite) newSyncMLSessionMsg(managementUrl string) ([]byte, error) {
	if len(managementUrl) == 0 {
		return nil, errors.New("managementUrl is empty")
	}

	return []byte(`
			 <SyncML xmlns="SYNCML:SYNCML1.2">
			<SyncHdr>
				<VerDTD>1.2</VerDTD>
				<VerProto>DM/1.2</VerProto>
				<SessionID>1</SessionID>
				<MsgID>1</MsgID>
				<Target>
				<LocURI>` + managementUrl + `</LocURI>
				</Target>
				<Source>
				<LocURI>DB257C3A08778F4FB61E2749066C1F27</LocURI>
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
				<Replace>
				<CmdID>4</CmdID>
				<Item>
					<Source>
					<LocURI>./DevInfo/DevId</LocURI>
					</Source>
					<Data>DB257C3A08778F4FB61E2749066C1F27</Data>
				</Item>
				<Item>
					<Source>
					<LocURI>./DevInfo/Man</LocURI>
					</Source>
					<Data>VMware, Inc.</Data>
				</Item>
				<Item>
					<Source>
					<LocURI>./DevInfo/Mod</LocURI>
					</Source>
					<Data>VMware7,1</Data>
				</Item>
				<Item>
					<Source>
					<LocURI>./DevInfo/DmV</LocURI>
					</Source>
					<Data>1.3</Data>
				</Item>
				<Item>
					<Source>
					<LocURI>./DevInfo/Lang</LocURI>
					</Source>
					<Data>en-US</Data>
				</Item>
				</Replace>
				<Final/>
			</SyncBody>
			</SyncML>`), nil
}
