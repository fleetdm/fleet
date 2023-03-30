package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	mathrand "math/rand"
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

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/push"
	nanomdm_pushsvc "github.com/micromdm/nanomdm/push/service"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/externalsvc"
	"github.com/fleetdm/fleet/v4/server/service/mock"
	"github.com/fleetdm/fleet/v4/server/service/schedule"
	"github.com/fleetdm/fleet/v4/server/test"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/uuid"
	"github.com/groob/plist"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	nanodep_client "github.com/micromdm/nanodep/client"
	"github.com/micromdm/nanodep/godep"
	nanodep_storage "github.com/micromdm/nanodep/storage"
	"github.com/micromdm/nanodep/tokenpki"
	scepclient "github.com/micromdm/scep/v2/client"
	"github.com/micromdm/scep/v2/cryptoutil/x509util"
	"github.com/micromdm/scep/v2/scep"
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
	onScheduleDone       func() // function called when profileSchedule.Trigger() job completed
	oktaMock             *externalsvc.MockOktaServer
}

func (s *integrationMDMTestSuite) SetupSuite() {
	s.withDS.SetupSuite("integrationMDMTestSuite")

	appConf, err := s.ds.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.MDM.EnabledAndConfigured = true
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)

	testCert, testKey, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(s.T(), err)
	testCertPEM := tokenpki.PEMCertificate(testCert.Raw)
	testKeyPEM := tokenpki.PEMRSAPrivateKey(testKey)

	oktaMock := externalsvc.RunMockOktaServer(s.T())
	fleetCfg := config.TestConfig()
	config.SetTestMDMConfig(s.T(), &fleetCfg, testCertPEM, testKeyPEM, testBMToken)
	fleetCfg.Osquery.EnrollCooldown = 0
	fleetCfg.MDM.OktaServerURL = oktaMock.Srv.URL
	fleetCfg.MDM.OktaClientID = oktaMock.ClientID
	fleetCfg.MDM.OktaClientSecret = oktaMock.ClientSecret()

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
		StartCronSchedules: []TestNewScheduleFunc{
			func(ctx context.Context, ds fleet.Datastore) fleet.NewCronScheduleFunc {
				return func() (fleet.CronSchedule, error) {
					const name = string(fleet.CronAppleMDMDEPProfileAssigner)
					logger := kitlog.NewJSONLogger(os.Stdout)
					fleetSyncer := apple_mdm.NewDEPService(ds, depStorage, logger, true)
					depSchedule = schedule.New(
						ctx, name, s.T().Name(), 1*time.Hour, ds, ds,
						schedule.WithLogger(logger),
						schedule.WithJob("dep_syncer", func(ctx context.Context) error {
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
							if s.onScheduleDone != nil {
								defer s.onScheduleDone()
							}
							return ReconcileProfiles(ctx, ds, NewMDMAppleCommander(mdmStorage, mdmPushService), logger)
						}),
					)
					return profileSchedule, nil
				}
			},
		},
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
	s.oktaMock = oktaMock

	fleetdmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := s.fleetDMNextCSRStatus.Swap(http.StatusOK)
		w.WriteHeader(status.(int))
		_, _ = w.Write([]byte(fmt.Sprintf("status: %d", status)))
	}))
	s.T().Setenv("TEST_FLEETDM_API_URL", fleetdmSrv.URL)

	s.T().Cleanup(fleetdmSrv.Close)
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

	appCfg := s.getConfig()
	if appCfg.MDM.MacOSSettings.EnableDiskEncryption {
		s.token = s.getTestAdminToken()
		// ensure global disk encryption is disabled on exit
		s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "macos_settings": { "enable_disk_encryption": false } }
  }`), http.StatusOK)
	}

	s.withServer.commonTearDownTest(t)

	// use a sql statement to delete all profiles, since the datastore prevents
	// deleting the fleet-specific ones.
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM mdm_apple_configuration_profiles")
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
	require.Equal(t, "https://example.org/mdm/apple/mdm", getAppleBMResp.MDMServerURL)
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
	require.Equal(t, "https://example.org/mdm/apple/mdm", getAppleBMResp.MDMServerURL)
	require.Equal(t, tm.Name, getAppleBMResp.DefaultTeam)
}

func (s *integrationMDMTestSuite) TestABMExpiredToken() {
	t := s.T()
	s.mockDEPResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"code": "T_C_NOT_SIGNED"}`))
	}))

	config := s.getConfig()
	require.False(t, config.MDM.AppleBMTermsExpired)

	var getAppleBMResp getAppleBMResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple_bm", nil, http.StatusInternalServerError, &getAppleBMResp)

	config = s.getConfig()
	require.True(t, config.MDM.AppleBMTermsExpired)
}

func (s *integrationMDMTestSuite) TestProfileManagement() {
	t := s.T()
	ctx := context.Background()

	globalProfiles := [][]byte{
		mobileconfigForTest("N1", "I1"),
		mobileconfigForTest("N2", "I2"),
	}

	// add global profiles
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: globalProfiles}, http.StatusNoContent)

	// create a new team
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "batch_set_mdm_profiles"})
	require.NoError(t, err)

	teamProfiles := [][]byte{
		mobileconfigForTest("N3", "I3"),
	}
	// add profiles to the team
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: teamProfiles}, http.StatusNoContent, "team_id", strconv.Itoa(int(tm.ID)))

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

	// create and enroll a host in MDM
	d := newDevice(s)
	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name()),
		NodeKey:         ptr.String(t.Name()),
		UUID:            d.uuid,
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
		HardwareSerial:  d.serial,
	})
	require.NoError(t, err)
	d.mdmEnroll(s)

	var wg sync.WaitGroup
	wg.Add(2)
	s.pushProvider.PushFunc = func(pushes []*mdm.Push) (map[string]*push.Response, error) {
		wg.Done()
		require.Len(t, pushes, 1)
		require.Equal(t, pushes[0].PushMagic, "pushmagic"+d.serial)
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
				cmd = d.idle()
			} else {
				cmd = d.acknowledge(cmd.CommandUUID)
			}

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
	_, err = s.profileSchedule.Trigger()
	require.NoError(t, err)
	wg.Wait()

	installs, removes := checkNextPayloads()
	// verify that we received both profiles
	require.ElementsMatch(t, installs, globalProfiles)
	require.Empty(t, removes)

	// add the host to a team
	err = s.ds.AddHostsToTeam(ctx, &tm.ID, []uint{host.ID})
	require.NoError(t, err)

	// trigger a profile sync
	wg.Add(3)
	_, err = s.profileSchedule.Trigger()
	require.NoError(t, err)
	wg.Wait()

	installs, removes = checkNextPayloads()
	// verify that we should install the team profile
	require.ElementsMatch(t, installs, teamProfiles)
	// verify that we should delete both profiles
	require.ElementsMatch(t, removes, []string{"I1", "I2"})

	// set new team profiles (delete + addition)
	teamProfiles = [][]byte{
		mobileconfigForTest("N4", "I4"),
		mobileconfigForTest("N5", "I5"),
	}
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: teamProfiles}, http.StatusNoContent, "team_id", strconv.Itoa(int(tm.ID)))

	// trigger a profile sync
	wg.Add(3)
	_, err = s.profileSchedule.Trigger()
	require.NoError(t, err)
	wg.Wait()
	installs, removes = checkNextPayloads()
	// verify that we should install the team profiles
	require.ElementsMatch(t, installs, teamProfiles)
	// verify that we should delete the old team profiles
	require.ElementsMatch(t, removes, []string{"I3"})

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
	require.Len(t, resProfiles, len(teamProfiles))

	var teamSummaryResp getMDMAppleProfilesSummaryResponse
	s.DoJSON("GET", "/api/v1/fleet/mdm/apple/profiles/summary", getMDMAppleProfilesSummaryRequest{TeamID: &tm.ID}, http.StatusOK, &teamSummaryResp)
	require.Equal(t, uint(0), teamSummaryResp.Pending)
	require.Equal(t, uint(0), teamSummaryResp.Failed)
	require.Equal(t, uint(1), teamSummaryResp.Latest)

	var noTeamSummaryResp getMDMAppleProfilesSummaryResponse
	s.DoJSON("GET", "/api/v1/fleet/mdm/apple/profiles/summary", getMDMAppleProfilesSummaryRequest{}, http.StatusOK, &noTeamSummaryResp)
	require.Equal(t, uint(0), noTeamSummaryResp.Pending)
	require.Equal(t, uint(0), noTeamSummaryResp.Failed)
	require.Equal(t, uint(0), noTeamSummaryResp.Latest)
}

func (s *integrationMDMTestSuite) TestDEPProfileAssignment() {
	t := s.T()
	devices := []godep.Device{
		{SerialNumber: uuid.New().String(), Model: "MacBook Pro", OS: "osx", OpType: "added"},
		{SerialNumber: uuid.New().String(), Model: "MacBook Mini", OS: "osx", OpType: "added"},
	}

	var wg sync.WaitGroup
	wg.Add(2)
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
			err := encoder.Encode(godep.DeviceResponse{Devices: devices})
			require.NoError(t, err)
		case "/profile/devices":
			wg.Done()
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
	_, err := s.depSchedule.Trigger()
	require.NoError(t, err)
	wg.Wait()

	// both hosts should be returned from the hosts endpoint
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 2)
	require.Equal(t, listHostsRes.Hosts[0].HardwareSerial, devices[0].SerialNumber)
	require.Equal(t, listHostsRes.Hosts[1].HardwareSerial, devices[1].SerialNumber)
	require.EqualValues(
		t,
		[]string{devices[0].SerialNumber, devices[1].SerialNumber},
		[]string{listHostsRes.Hosts[0].HardwareSerial, listHostsRes.Hosts[1].HardwareSerial},
	)

	// create a new host
	createHostAndDeviceToken(t, s.ds, "not-dep")
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 3)

	// filtering by MDM status works
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts?mdm_enrollment_status=pending", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 2)

	// enroll one of the hosts
	d := newDevice(s)
	d.serial = devices[0].SerialNumber
	d.mdmEnroll(s)

	// only one shows up as pending
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts?mdm_enrollment_status=pending", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 1)

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
					`{"host_serial": "%s", "host_display_name": "%s (%s)", "installed_from_dep": true}`,
					devices[0].SerialNumber, devices[0].Model, devices[0].SerialNumber,
				),
				string(*activity.Details),
			)
		}
	}
	require.True(t, found)
}

func (s *integrationMDMTestSuite) TestDeviceMDMManualEnroll() {
	t := s.T()

	token := "token_test_manual_enroll"
	createHostAndDeviceToken(t, s.ds, token)

	// invalid token fails
	s.DoRaw("GET", "/api/latest/fleet/device/invalid_token/mdm/apple/manual_enrollment_profile", nil, http.StatusUnauthorized)

	// valid token downloads the profile
	resp := s.DoRaw("GET", "/api/latest/fleet/device/"+token+"/mdm/apple/manual_enrollment_profile", nil, http.StatusOK)
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

	var profile struct {
		PayloadIdentifier string `plist:"PayloadIdentifier"`
	}
	require.NoError(t, plist.Unmarshal(body, &profile))
	require.Equal(t, apple_mdm.FleetPayloadIdentifier, profile.PayloadIdentifier)
}

func (s *integrationMDMTestSuite) TestAppleMDMDeviceEnrollment() {
	t := s.T()

	// Enroll two devices into MDM
	deviceA := newMDMEnrolledDevice(s)
	deviceB := newMDMEnrolledDevice(s)

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
		[]string{deviceA.uuid, deviceB.uuid},
		[]string{listHostsRes.Hosts[0].UUID, listHostsRes.Hosts[1].UUID},
	)

	var targetHostID uint
	var lastEnroll time.Time
	for _, host := range listHostsRes.Hosts {
		if host.UUID == deviceA.uuid {
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
	require.JSONEq(t, fmt.Sprintf(`{"host_serial": "%s", "host_display_name": "%s (%s)", "installed_from_dep": false}`, deviceA.serial, deviceA.model, deviceA.serial), string(*details[len(details)-2]))
	require.JSONEq(t, fmt.Sprintf(`{"host_serial": "%s", "host_display_name": "%s (%s)", "installed_from_dep": false}`, deviceB.serial, deviceB.model, deviceB.serial), string(*details[len(details)-1]))

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
		HostIdentifier: deviceA.uuid,
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
	deviceA.checkout()

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
			require.JSONEq(t, fmt.Sprintf(`{"host_serial": "%s", "host_display_name": "%s (%s)", "installed_from_dep": false}`, deviceA.serial, deviceA.model, deviceA.serial), string(*activity.Details))
		}
	}
	require.True(t, found)
}

func (s *integrationMDMTestSuite) TestDeviceMultipleAuthMessages() {
	d := newMDMEnrolledDevice(s)

	listHostsRes := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(s.T(), listHostsRes.Hosts, 1)

	// send the auth message again, we still have only one host
	d.authenticate()
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
	// enroll into mdm
	d := newMDMEnrolledDevice(s)

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
		HostIdentifier: d.uuid,
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
	require.Len(t, *hostResp.Host.MDM.Profiles, 3)

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
	s.pushProvider.PushFunc = func(pushes []*mdm.Push) (map[string]*push.Response, error) {
		res, err := mockSuccessfulPush(pushes)
		d.checkout()
		return res, err
	}
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/mdm/hosts/%d/unenroll", h.ID), nil, http.StatusOK)

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
			Status:            &fleet.MDMAppleDeliveryApplied,
		},
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		err := s.ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
			HostUUID:      host.UUID,
			CommandUUID:   hostCmdUUID,
			ProfileID:     fileVaultProf.ProfileID,
			Status:        &fleet.MDMAppleDeliveryApplied,
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

	// we're about to mess up with the token, make sure to set it to the
	// default value when the test ends
	currToken := s.token
	t.Cleanup(func() { s.token = currToken })

	// admins are able to see the host encryption key
	s.token = s.getTestAdminToken()
	checkDecryptableKey(s.users["admin1@example.com"])

	// get that host - it has an encryption key that is decryptable, so it
	// should report "applied" disk encryption.
	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
	require.Equal(t, fleet.DiskEncryptionApplied, *getHostResp.Host.MDM.MacOSSettings.DiskEncryption)
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
	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          4827,
		Name:        "team1_" + t.Name(),
		Description: "desc team1_" + t.Name(),
	})
	require.NoError(t, err)
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
	require.Equal(t, uint(0), fvsResp.Applied)
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
	generateAggregateValue(hosts[0:2], fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryApplied, true)
	fvsResp = getMDMAppleFileVauleSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/filevault/summary", nil, http.StatusOK, &fvsResp)
	require.Equal(t, uint(2), fvsResp.Applied)
	require.Equal(t, uint(0), fvsResp.ActionRequired)
	require.Equal(t, uint(0), fvsResp.Enforcing)
	require.Equal(t, uint(0), fvsResp.Failed)
	require.Equal(t, uint(0), fvsResp.RemovingEnforcement)

	// hosts 3,4 have disk encryption "action required" status
	generateAggregateValue(hosts[2:4], fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryApplied, false)
	fvsResp = getMDMAppleFileVauleSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/filevault/summary", nil, http.StatusOK, &fvsResp)
	require.Equal(t, uint(2), fvsResp.Applied)
	require.Equal(t, uint(2), fvsResp.ActionRequired)
	require.Equal(t, uint(0), fvsResp.Enforcing)
	require.Equal(t, uint(0), fvsResp.Failed)
	require.Equal(t, uint(0), fvsResp.RemovingEnforcement)

	// hosts 5,6 have disk encryption "enforcing" status

	// host profiles status are `pending`
	generateAggregateValue(hosts[4:6], fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryPending, true)
	fvsResp = getMDMAppleFileVauleSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/filevault/summary", nil, http.StatusOK, &fvsResp)
	require.Equal(t, uint(2), fvsResp.Applied)
	require.Equal(t, uint(2), fvsResp.ActionRequired)
	require.Equal(t, uint(2), fvsResp.Enforcing)
	require.Equal(t, uint(0), fvsResp.Failed)
	require.Equal(t, uint(0), fvsResp.RemovingEnforcement)

	// host profiles status dont exist
	generateAggregateValue(hosts[4:6], fleet.MDMAppleOperationTypeInstall, nil, true)
	fvsResp = getMDMAppleFileVauleSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/filevault/summary", nil, http.StatusOK, &fvsResp)
	require.Equal(t, uint(2), fvsResp.Applied)
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
	require.Equal(t, uint(2), fvsResp.Applied)
	require.Equal(t, uint(2), fvsResp.ActionRequired)
	require.Equal(t, uint(2), fvsResp.Enforcing)
	require.Equal(t, uint(0), fvsResp.Failed)
	require.Equal(t, uint(0), fvsResp.RemovingEnforcement)

	// hosts 7,8 have disk encryption "failed" status
	generateAggregateValue(hosts[6:8], fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryFailed, true)
	fvsResp = getMDMAppleFileVauleSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/filevault/summary", nil, http.StatusOK, &fvsResp)
	require.Equal(t, uint(2), fvsResp.Applied)
	require.Equal(t, uint(2), fvsResp.ActionRequired)
	require.Equal(t, uint(2), fvsResp.Enforcing)
	require.Equal(t, uint(2), fvsResp.Failed)
	require.Equal(t, uint(0), fvsResp.RemovingEnforcement)

	// hosts 9,10 have disk encryption "removing enforcement" status
	generateAggregateValue(hosts[8:10], fleet.MDMAppleOperationTypeRemove, &fleet.MDMAppleDeliveryPending, true)
	fvsResp = getMDMAppleFileVauleSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/filevault/summary", nil, http.StatusOK, &fvsResp)
	require.Equal(t, uint(2), fvsResp.Applied)
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
	generateAggregateValue(hosts[0:2], fleet.MDMAppleOperationTypeInstall, &fleet.MDMAppleDeliveryApplied, true)
	fvsResp = getMDMAppleFileVauleSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/filevault/summary", nil, http.StatusOK, &fvsResp, "team_id", strconv.Itoa(int(tm.ID)))
	require.Equal(t, uint(2), fvsResp.Applied)
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

func (s *integrationMDMTestSuite) TestDEPOktaLogin() {
	t := s.T()

	params := func(u, p string) []byte {
		j, err := json.Marshal(&mdmAppleDEPLoginRequest{
			Username: u,
			Password: p,
		})
		require.NoError(t, err)
		return j
	}

	// invalid user/pass combination returns an unauthorized
	// error
	s.DoRawNoAuth(
		"POST",
		"/api/latest/fleet/mdm/apple/dep_login",
		params(s.oktaMock.Username, "bad"),
		http.StatusUnauthorized,
	)

	// invalid clientID/clientSecret returns a server error
	oldSecret := s.oktaMock.ClientSecret()
	s.oktaMock.SetClientSecret("new-secret")
	s.DoRawNoAuth(
		"POST",
		"/api/latest/fleet/mdm/apple/dep_login",
		params(s.oktaMock.Username, s.oktaMock.UserPassword),
		http.StatusInternalServerError,
	)
	s.oktaMock.SetClientSecret(oldSecret)

	// valid user/pass authenticates the user and creates a new
	// entry in `mdm_idp_accounts`
	s.DoRawNoAuth(
		"POST",
		"/api/latest/fleet/mdm/apple/dep_login",
		params(s.oktaMock.Username, s.oktaMock.UserPassword),
		http.StatusOK,
	)
	var uuid string
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &uuid,
			`SELECT uuid FROM mdm_idp_accounts WHERE username = ?`, s.oktaMock.Username)
	})
	require.NotEmpty(t, uuid)
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
		d := newDevice(s)

		// enroll the device with orbit
		var resp EnrollOrbitResponse
		s.DoJSON("POST", "/api/fleet/orbit/enroll", EnrollOrbitRequest{
			EnrollSecret:   secret,
			HardwareUUID:   d.uuid, // will not match any existing host
			HardwareSerial: d.serial,
		}, http.StatusOK, &resp)
		require.NotEmpty(t, resp.OrbitNodeKey)
		orbitNodeKey := resp.OrbitNodeKey
		h, err := s.ds.LoadHostByOrbitNodeKey(ctx, orbitNodeKey)
		require.NoError(t, err)
		h.OrbitNodeKey = &orbitNodeKey

		d.mdmEnroll(s)

		return h
	}

	triggerReconcileProfiles := func() {
		ch := make(chan bool)
		s.onScheduleDone = func() { close(ch) }
		_, err := s.profileSchedule.Trigger()
		require.NoError(t, err)
		<-ch
		// this will only mark them as "pending", as the response to confirm
		// profile deployment is asynchronous, so we simulate it here by
		// updating any "pending" (not NULL) profiles to "applied"
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `UPDATE host_mdm_apple_profiles SET status = 'applied' WHERE status = 'pending'`)
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
			{Identifier: "G1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
			{Identifier: "G2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
		},
		h2: {
			{Identifier: "G1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
			{Identifier: "G2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
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
			{Identifier: "T1.1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
			{Identifier: "T1.2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
		},
		h4: {
			{Identifier: "T1.1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
			{Identifier: "T1.2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
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
			{Identifier: "G1", OperationType: fleet.MDMAppleOperationTypeRemove, Status: nil},
			{Identifier: "G2", OperationType: fleet.MDMAppleOperationTypeRemove, Status: nil},
			{Identifier: "T2.1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
		},
		h2: {
			{Identifier: "G1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
			{Identifier: "G2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
		},
	})

	// switch a team host (h3) to another team (tm2)
	s.DoJSON("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &tm2.ID, HostIDs: []uint{h3.ID}}, http.StatusOK, &moveHostResp)
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h3: {
			{Identifier: "T1.1", OperationType: fleet.MDMAppleOperationTypeRemove, Status: nil},
			{Identifier: "T1.2", OperationType: fleet.MDMAppleOperationTypeRemove, Status: nil},
			{Identifier: "T2.1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
		},
		h4: {
			{Identifier: "T1.1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
			{Identifier: "T1.2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
		},
	})

	// switch a team host (h4) to no team
	s.DoJSON("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: nil, HostIDs: []uint{h4.ID}}, http.StatusOK, &moveHostResp)
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h3: {
			{Identifier: "T1.1", OperationType: fleet.MDMAppleOperationTypeRemove, Status: nil},
			{Identifier: "T1.2", OperationType: fleet.MDMAppleOperationTypeRemove, Status: nil},
			{Identifier: "T2.1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
		},
		h4: {
			{Identifier: "T1.1", OperationType: fleet.MDMAppleOperationTypeRemove, Status: nil},
			{Identifier: "T1.2", OperationType: fleet.MDMAppleOperationTypeRemove, Status: nil},
			{Identifier: "G1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
			{Identifier: "G2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
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
			{Identifier: "G1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
			{Identifier: "G2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
			{Identifier: "G3", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
		},
		h4: {
			{Identifier: "G1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
			{Identifier: "G2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
			{Identifier: "G3", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
		},
	})

	// add a profile to team 2 (h1 and h3 are now part of team 2)
	body, headers = generateNewProfileMultipartRequest(t, &tm2.ID,
		"some_name", mobileconfigForTest("T2.2", "T2.2"), s.token)
	s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", body.Bytes(), http.StatusOK, headers)
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: "T2.1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
			{Identifier: "T2.2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
		},
		h3: {
			{Identifier: "T2.1", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
			{Identifier: "T2.2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
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
			{Identifier: "G1", OperationType: fleet.MDMAppleOperationTypeRemove, Status: nil},
			{Identifier: "G2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
			{Identifier: "G3", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
		},
		h4: {
			{Identifier: "G1", OperationType: fleet.MDMAppleOperationTypeRemove, Status: nil},
			{Identifier: "G2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
			{Identifier: "G3", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
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
			{Identifier: "T2.1", OperationType: fleet.MDMAppleOperationTypeRemove, Status: nil},
			{Identifier: "T2.2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
		},
		h3: {
			{Identifier: "T2.1", OperationType: fleet.MDMAppleOperationTypeRemove, Status: nil},
			{Identifier: "T2.2", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
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
			{Identifier: "G2", OperationType: fleet.MDMAppleOperationTypeRemove, Status: nil},
			{Identifier: "G2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
			{Identifier: "G3", OperationType: fleet.MDMAppleOperationTypeRemove, Status: nil},
			{Identifier: "G4", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
		},
		h4: {
			{Identifier: "G2", OperationType: fleet.MDMAppleOperationTypeRemove, Status: nil},
			{Identifier: "G2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
			{Identifier: "G3", OperationType: fleet.MDMAppleOperationTypeRemove, Status: nil},
			{Identifier: "G4", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
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
			{Identifier: "T2.2", OperationType: fleet.MDMAppleOperationTypeRemove, Status: nil},
			{Identifier: "T2.2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
			{Identifier: "T2.3", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
		},
		h3: {
			{Identifier: "T2.2", OperationType: fleet.MDMAppleOperationTypeRemove, Status: nil},
			{Identifier: "T2.2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
			{Identifier: "T2.3", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
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
			{Identifier: "T2.2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
			{Identifier: "T2.3", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
		},
		h2: {
			{Identifier: "G2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
			{Identifier: "G4", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
		},
		h3: {
			{Identifier: "T2.2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
			{Identifier: "T2.3", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
		},
		h4: {
			{Identifier: "G2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
			{Identifier: "G4", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
		},
	})

	// delete team 2 (h1 and h3 are part of that team)
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d", tm2.ID), nil, http.StatusOK)
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: "T2.2b", OperationType: fleet.MDMAppleOperationTypeRemove, Status: nil},
			{Identifier: "T2.3", OperationType: fleet.MDMAppleOperationTypeRemove, Status: nil},
			{Identifier: "G2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
			{Identifier: "G4", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
		},
		h3: {
			{Identifier: "T2.2b", OperationType: fleet.MDMAppleOperationTypeRemove, Status: nil},
			{Identifier: "T2.3", OperationType: fleet.MDMAppleOperationTypeRemove, Status: nil},
			{Identifier: "G2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
			{Identifier: "G4", OperationType: fleet.MDMAppleOperationTypeInstall, Status: nil},
		},
	})

	// apply the pending profiles
	triggerReconcileProfiles()

	// final state
	s.assertHostConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		h1: {
			{Identifier: "G2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
			{Identifier: "G4", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
		},
		h2: {
			{Identifier: "G2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
			{Identifier: "G4", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
		},
		h3: {
			{Identifier: "G2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
			{Identifier: "G4", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
		},
		h4: {
			{Identifier: "G2b", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
			{Identifier: "G4", OperationType: fleet.MDMAppleOperationTypeInstall, Status: &fleet.MDMAppleDeliveryApplied},
		},
	})
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
	fileName string, fileContent []byte, token string) (*bytes.Buffer, map[string]string) {
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

type device struct {
	uuid   string
	serial string
	model  string

	s        *integrationMDMTestSuite
	scepCert *x509.Certificate
	scepKey  *rsa.PrivateKey
}

func newDevice(s *integrationMDMTestSuite) *device {
	return &device{
		uuid:   strings.ToUpper(uuid.New().String()),
		serial: randSerial(),
		model:  "MacBookPro16,1",
		s:      s,
	}
}

func newMDMEnrolledDevice(s *integrationMDMTestSuite) *device {
	d := newDevice(s)
	d.mdmEnroll(s)
	return d
}

func (d *device) mdmEnroll(s *integrationMDMTestSuite) {
	d.scepEnroll()
	d.authenticate()
	d.tokenUpdate()
}

func (d *device) authenticate() {
	payload := map[string]any{
		"MessageType":  "Authenticate",
		"UDID":         d.uuid,
		"Model":        d.model,
		"DeviceName":   "testdevice" + d.serial,
		"Topic":        "com.apple.mgmt.External." + d.uuid,
		"EnrollmentID": "testenrollmentid-" + d.uuid,
		"SerialNumber": d.serial,
	}
	d.request("application/x-apple-aspen-mdm-checkin", payload)
}

func (d *device) tokenUpdate() {
	payload := map[string]any{
		"MessageType":  "TokenUpdate",
		"UDID":         d.uuid,
		"Topic":        "com.apple.mgmt.External." + d.uuid,
		"EnrollmentID": "testenrollmentid-" + d.uuid,
		"NotOnConsole": "false",
		"PushMagic":    "pushmagic" + d.serial,
		"Token":        []byte("token" + d.serial),
	}
	d.request("application/x-apple-aspen-mdm-checkin", payload)
}

func (d *device) checkout() {
	payload := map[string]any{
		"MessageType":  "CheckOut",
		"Topic":        "com.apple.mgmt.External." + d.uuid,
		"UDID":         d.uuid,
		"EnrollmentID": "testenrollmentid-" + d.uuid,
	}
	d.request("application/x-apple-aspen-mdm-checkin", payload)
}

// Devices send an Idle status to signal the server that they're ready to
// receive commands.
// The server can signal back with either a command to run
// or an empty response body to end the communication.
func (d *device) idle() *micromdm.CommandPayload {
	payload := map[string]any{
		"Status":       "Idle",
		"Topic":        "com.apple.mgmt.External." + d.uuid,
		"UDID":         d.uuid,
		"EnrollmentID": "testenrollmentid-" + d.uuid,
	}
	return d.sendAndDecodeCommandResponse(payload)
}

func (d *device) acknowledge(cmdUUID string) *micromdm.CommandPayload {
	payload := map[string]any{
		"Status":       "Acknowledged",
		"Topic":        "com.apple.mgmt.External." + d.uuid,
		"UDID":         d.uuid,
		"EnrollmentID": "testenrollmentid-" + d.uuid,
		"CommandUUID":  cmdUUID,
	}
	return d.sendAndDecodeCommandResponse(payload)
}

func (d *device) sendAndDecodeCommandResponse(payload map[string]any) *micromdm.CommandPayload {
	res := d.request("", payload)
	if res.ContentLength == 0 {
		return nil
	}
	raw, err := io.ReadAll(res.Body)
	require.NoError(d.s.T(), err)
	cmd, err := mdm.DecodeCommand(raw)
	require.NoError(d.s.T(), err)

	var p micromdm.CommandPayload
	err = plist.Unmarshal(cmd.Raw, &p)
	require.NoError(d.s.T(), err)
	return &p
}

func (d *device) request(reqType string, payload map[string]any) *http.Response {
	body, err := plist.Marshal(payload)
	require.NoError(d.s.T(), err)

	signedData, err := pkcs7.NewSignedData(body)
	require.NoError(d.s.T(), err)
	err = signedData.AddSigner(d.scepCert, d.scepKey, pkcs7.SignerInfoConfig{})
	require.NoError(d.s.T(), err)
	sig, err := signedData.Finish()
	require.NoError(d.s.T(), err)

	return d.s.DoRawWithHeaders(
		"POST",
		"/mdm/apple/mdm",
		body,
		200,
		map[string]string{
			"Content-Type":  reqType,
			"Mdm-Signature": base64.StdEncoding.EncodeToString(sig),
		},
	)
}

func (d *device) scepEnroll() {
	t := d.s.T()
	ctx := context.Background()
	logger := kitlog.NewJSONLogger(os.Stdout)
	logger = level.NewFilter(logger, level.AllowDebug())
	client, err := scepclient.New(d.s.server.URL+apple_mdm.SCEPPath, logger)
	require.NoError(t, err)

	resp, _, err := client.GetCACert(ctx, "")
	require.NoError(t, err)

	certs, err := x509.ParseCertificates(resp)
	require.NoError(t, err)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	csrTemplate := x509util.CertificateRequest{
		CertificateRequest: x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName: "fleet-test",
			},
			SignatureAlgorithm: x509.SHA256WithRSA,
		},
		ChallengePassword: d.s.fleetCfg.MDM.AppleSCEPChallenge,
	}
	csrDerBytes, err := x509util.CreateCertificateRequest(rand.Reader, &csrTemplate, key)
	require.NoError(t, err)
	csr, err := x509.ParseCertificateRequest(csrDerBytes)
	require.NoError(t, err)

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour)

	certTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "SCEP SIGNER",
			Organization: csr.Subject.Organization,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDerBytes, err := x509.CreateCertificate(rand.Reader, &certTemplate, &certTemplate, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(certDerBytes)
	require.NoError(t, err)

	tmpl := &scep.PKIMessage{
		MessageType: scep.PKCSReq,
		Recipients:  certs,
		SignerKey:   key,
		SignerCert:  cert,
		CSRReqMessage: &scep.CSRReqMessage{
			ChallengePassword: d.s.fleetCfg.MDM.AppleSCEPChallenge,
		},
	}

	msg, err := scep.NewCSRRequest(csr, tmpl, scep.WithLogger(logger))
	require.NoError(t, err)

	respBytes, err := client.PKIOperation(ctx, msg.Raw)
	require.NoError(t, err)

	respMsg, err := scep.ParsePKIMessage(respBytes, scep.WithLogger(logger), scep.WithCACerts(msg.Recipients))
	require.NoError(t, err)
	require.Equal(t, scep.SUCCESS, respMsg.PKIStatus)

	err = respMsg.DecryptPKIEnvelope(cert, key)
	require.NoError(t, err)

	d.scepCert = respMsg.CertRepMessage.Certificate
	d.scepKey = key
}

// numbers plus capital letters without I, L, O for readability
const serialLetters = "0123456789ABCDEFGHJKMNPQRSTUVWXYZ"

func randSerial() string {
	b := make([]byte, 12)
	for i := range b {
		//nolint:gosec // not used for crypto, only to generate random serial for testing
		b[i] = serialLetters[mathrand.Intn(len(serialLetters))]
	}
	return string(b)
}

var testBMToken = &nanodep_client.OAuth1Tokens{
	ConsumerKey:       "test_consumer",
	ConsumerSecret:    "test_secret",
	AccessToken:       "test_access_token",
	AccessSecret:      "test_access_secret",
	AccessTokenExpiry: time.Date(2999, 1, 1, 0, 0, 0, 0, time.UTC),
}
