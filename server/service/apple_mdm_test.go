package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	fleetmdm "github.com/fleetdm/fleet/v4/server/mdm"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/log/stdlogfmt"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	nanomdm_pushsvc "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/service"
	"github.com/fleetdm/fleet/v4/server/mock"
	nanodep_mock "github.com/fleetdm/fleet/v4/server/mock/nanodep"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	kitlog "github.com/go-kit/kit/log"
	"github.com/google/uuid"
	nanodep_client "github.com/micromdm/nanodep/client"
	"github.com/stretchr/testify/require"
)

type nopProfileMatcher struct{}

func (nopProfileMatcher) PreassignProfile(ctx context.Context, pld fleet.MDMApplePreassignProfilePayload) error {
	return nil
}

func (nopProfileMatcher) RetrieveProfiles(ctx context.Context, extHostID string) (fleet.MDMApplePreassignHostProfiles, error) {
	return fleet.MDMApplePreassignHostProfiles{}, nil
}

func setupAppleMDMService(t *testing.T, license *fleet.LicenseInfo) (fleet.Service, context.Context, *mock.Store) {
	ds := new(mock.Store)
	cfg := config.TestConfig()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/server/devices"):
			_, err := w.Write([]byte("{}"))
			require.NoError(t, err)
			return
		case strings.Contains(r.URL.Path, "/session"):
			_, err := w.Write([]byte(`{"auth_session_token": "yoo"}`))
			require.NoError(t, err)
			return
		}
	}))

	mdmStorage := &mock.MDMAppleStore{}
	depStorage := &nanodep_mock.Storage{}
	pushFactory, _ := newMockAPNSPushProviderFactory()
	pusher := nanomdm_pushsvc.New(
		mdmStorage,
		mdmStorage,
		pushFactory,
		NewNanoMDMLogger(kitlog.NewJSONLogger(os.Stdout)),
	)

	opts := &TestServerOpts{
		FleetConfig:    &cfg,
		MDMStorage:     mdmStorage,
		DEPStorage:     depStorage,
		MDMPusher:      pusher,
		License:        license,
		ProfileMatcher: nopProfileMatcher{},
	}
	svc, ctx := newTestServiceWithConfig(t, ds, cfg, nil, nil, opts)

	mdmStorage.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.Command) (map[string]error, error) {
		return nil, nil
	}
	mdmStorage.RetrievePushInfoFunc = func(ctx context.Context, tokens []string) (map[string]*mdm.Push, error) {
		res := make(map[string]*mdm.Push, len(tokens))
		for _, t := range tokens {
			res[t] = &mdm.Push{
				PushMagic: "",
				Token:     []byte(t),
				Topic:     "",
			}
		}
		return res, nil
	}
	mdmStorage.RetrievePushCertFunc = func(ctx context.Context, topic string) (*tls.Certificate, string, error) {
		cert, err := tls.LoadX509KeyPair("testdata/server.pem", "testdata/server.key")
		return &cert, "", err
	}
	mdmStorage.IsPushCertStaleFunc = func(ctx context.Context, topic string, staleToken string) (bool, error) {
		return false, nil
	}

	depStorage.RetrieveAuthTokensFunc = func(ctx context.Context, name string) (*nanodep_client.OAuth1Tokens, error) {
		return &nanodep_client.OAuth1Tokens{}, nil
	}
	depStorage.RetrieveConfigFunc = func(context.Context, string) (*nanodep_client.Config, error) {
		return &nanodep_client.Config{
			BaseURL: ts.URL,
		}, nil
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			OrgInfo: fleet.OrgInfo{
				OrgName: "Foo Inc.",
			},
			ServerSettings: fleet.ServerSettings{
				ServerURL: "https://foo.example.com",
			},
			MDM: fleet.MDM{
				EnabledAndConfigured: true,
			},
		}, nil
	}
	ds.GetMDMAppleEnrollmentProfileByTokenFunc = func(ctx context.Context, token string) (*fleet.MDMAppleEnrollmentProfile, error) {
		return nil, nil
	}
	ds.NewMDMAppleEnrollmentProfileFunc = func(ctx context.Context, enrollmentPayload fleet.MDMAppleEnrollmentProfilePayload) (*fleet.MDMAppleEnrollmentProfile, error) {
		return &fleet.MDMAppleEnrollmentProfile{
			ID:            1,
			Token:         "foo",
			Type:          fleet.MDMAppleEnrollmentTypeManual,
			EnrollmentURL: "https://foo.example.com?token=foo",
		}, nil
	}
	ds.GetMDMAppleEnrollmentProfileByTokenFunc = func(ctx context.Context, token string) (*fleet.MDMAppleEnrollmentProfile, error) {
		return nil, nil
	}
	ds.ListMDMAppleEnrollmentProfilesFunc = func(ctx context.Context) ([]*fleet.MDMAppleEnrollmentProfile, error) {
		return nil, nil
	}
	ds.NewMDMAppleInstallerFunc = func(ctx context.Context, name string, size int64, manifest string, installer []byte, urlToken string) (*fleet.MDMAppleInstaller, error) {
		return nil, nil
	}
	ds.MDMAppleInstallerFunc = func(ctx context.Context, token string) (*fleet.MDMAppleInstaller, error) {
		return nil, nil
	}
	ds.MDMAppleInstallerDetailsByIDFunc = func(ctx context.Context, id uint) (*fleet.MDMAppleInstaller, error) {
		return nil, nil
	}
	ds.DeleteMDMAppleInstallerFunc = func(ctx context.Context, id uint) error {
		return nil
	}
	ds.MDMAppleInstallerDetailsByTokenFunc = func(ctx context.Context, token string) (*fleet.MDMAppleInstaller, error) {
		return nil, nil
	}
	ds.ListMDMAppleInstallersFunc = func(ctx context.Context) ([]fleet.MDMAppleInstaller, error) {
		return nil, nil
	}
	ds.MDMAppleListDevicesFunc = func(ctx context.Context) ([]fleet.MDMAppleDevice, error) {
		return nil, nil
	}
	ds.GetNanoMDMEnrollmentFunc = func(ctx context.Context, hostUUID string) (*fleet.NanoEnrollment, error) {
		return &fleet.NanoEnrollment{Enabled: false}, nil
	}
	ds.GetMDMAppleCommandRequestTypeFunc = func(ctx context.Context, commandUUID string) (string, error) {
		return "", nil
	}
	ds.MDMGetEULAMetadataFunc = func(ctx context.Context) (*fleet.MDMEULA, error) {
		return &fleet.MDMEULA{}, nil
	}
	ds.MDMGetEULABytesFunc = func(ctx context.Context, token string) (*fleet.MDMEULA, error) {
		return &fleet.MDMEULA{}, nil
	}
	ds.MDMInsertEULAFunc = func(ctx context.Context, eula *fleet.MDMEULA) error {
		return nil
	}
	ds.MDMDeleteEULAFunc = func(ctx context.Context, token string) error {
		return nil
	}

	return svc, ctx, ds
}

func TestAppleMDMAuthorization(t *testing.T) {
	svc, ctx, ds := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})

	checkAuthErr := func(t *testing.T, err error, shouldFailWithAuth bool) {
		t.Helper()

		if shouldFailWithAuth {
			require.Error(t, err)
			require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
		} else {
			require.NoError(t, err)
		}
	}

	testAuthdMethods := func(t *testing.T, user *fleet.User, shouldFailWithAuth bool) {
		ctx := test.UserContext(ctx, user)
		_, err := svc.UploadMDMAppleInstaller(ctx, "foo", 3, bytes.NewReader([]byte("foo")))
		checkAuthErr(t, err, shouldFailWithAuth)
		_, err = svc.GetMDMAppleInstallerByID(ctx, 42)
		checkAuthErr(t, err, shouldFailWithAuth)
		err = svc.DeleteMDMAppleInstaller(ctx, 42)
		checkAuthErr(t, err, shouldFailWithAuth)
		_, err = svc.ListMDMAppleInstallers(ctx)
		checkAuthErr(t, err, shouldFailWithAuth)
		_, err = svc.ListMDMAppleDevices(ctx)
		checkAuthErr(t, err, shouldFailWithAuth)
		_, err = svc.ListMDMAppleDEPDevices(ctx)
		checkAuthErr(t, err, shouldFailWithAuth)

		// check EULA routes
		_, err = svc.MDMGetEULAMetadata(ctx)
		checkAuthErr(t, err, shouldFailWithAuth)
		err = svc.MDMCreateEULA(ctx, "eula.pdf", bytes.NewReader([]byte("%PDF-")))
		checkAuthErr(t, err, shouldFailWithAuth)
		err = svc.MDMDeleteEULA(ctx, "foo")
		checkAuthErr(t, err, shouldFailWithAuth)
	}

	// Only global admins can access the endpoints.
	testAuthdMethods(t, test.UserAdmin, false)

	// All other users should not have access to the endpoints.
	for _, user := range []*fleet.User{
		test.UserNoRoles,
		test.UserMaintainer,
		test.UserObserver,
		test.UserObserverPlus,
		test.UserTeamAdminTeam1,
	} {
		testAuthdMethods(t, user, true)
	}
	// Token authenticated endpoints can be accessed by anyone.
	ctx = test.UserContext(ctx, test.UserNoRoles)
	_, err := svc.GetMDMAppleInstallerByToken(ctx, "foo")
	require.NoError(t, err)
	_, err = svc.GetMDMAppleEnrollmentProfileByToken(ctx, "foo", "")
	require.NoError(t, err)
	_, err = svc.GetMDMAppleInstallerDetailsByToken(ctx, "foo")
	require.NoError(t, err)
	_, err = svc.MDMGetEULABytes(ctx, "foo")
	require.NoError(t, err)
	// Generating a new key pair does not actually make any changes to fleet, or expose any
	// information. The user must configure fleet with the new key pair and restart the server.
	_, err = svc.NewMDMAppleDEPKeyPair(ctx)
	require.NoError(t, err)

	// Should work for all user types
	for _, user := range []*fleet.User{
		test.UserAdmin,
		test.UserMaintainer,
		test.UserObserver,
		test.UserObserverPlus,
		test.UserTeamAdminTeam1,
		test.UserTeamGitOpsTeam1,
		test.UserGitOps,
		test.UserTeamMaintainerTeam1,
		test.UserTeamObserverTeam1,
		test.UserTeamObserverPlusTeam1,
	} {
		usrctx := test.UserContext(ctx, user)
		_, err = svc.GetMDMManualEnrollmentProfile(usrctx)
		require.NoError(t, err)
	}

	// Must be device-authenticated, should fail
	_, err = svc.GetDeviceMDMAppleEnrollmentProfile(ctx)
	checkAuthErr(t, err, true)
	// works with device-authenticated context
	ctx = test.HostContext(context.Background(), &fleet.Host{})
	_, err = svc.GetDeviceMDMAppleEnrollmentProfile(ctx)
	require.NoError(t, err)

	hostUUIDsToTeamID := map[string]uint{
		"host1": 1,
		"host2": 1,
		"host3": 2,
		"host4": 0,
	}
	ds.ListHostsLiteByUUIDsFunc = func(ctx context.Context, filter fleet.TeamFilter, uuids []string) ([]*fleet.Host, error) {
		hosts := make([]*fleet.Host, 0, len(uuids))
		for _, uuid := range uuids {
			tmID := hostUUIDsToTeamID[uuid]
			if tmID == 0 {
				hosts = append(hosts, &fleet.Host{UUID: uuid, TeamID: nil})
			} else {
				hosts = append(hosts, &fleet.Host{UUID: uuid, TeamID: &tmID})
			}
		}
		return hosts, nil
	}

	rawB64FreeCmd := base64.RawStdEncoding.EncodeToString([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Command</key>
    <dict>
        <key>RequestType</key>
        <string>FooBar</string>
    </dict>
    <key>CommandUUID</key>
    <string>uuid</string>
</dict>
</plist>`))

	t.Run("EnqueueMDMAppleCommand", func(t *testing.T) {
		enqueueCmdCases := []struct {
			desc              string
			user              *fleet.User
			uuids             []string
			shoudFailWithAuth bool
		}{
			{"no role", test.UserNoRoles, []string{"host1", "host2", "host3", "host4"}, true},
			{"maintainer can run", test.UserMaintainer, []string{"host1", "host2", "host3", "host4"}, false},
			{"admin can run", test.UserAdmin, []string{"host1", "host2", "host3", "host4"}, false},
			{"observer cannot run", test.UserObserver, []string{"host1", "host2", "host3", "host4"}, true},
			{"team 1 admin can run team 1", test.UserTeamAdminTeam1, []string{"host1", "host2"}, false},
			{"team 2 admin can run team 2", test.UserTeamAdminTeam2, []string{"host3"}, false},
			{"team 1 maintainer can run team 1", test.UserTeamMaintainerTeam1, []string{"host1", "host2"}, false},
			{"team 1 observer cannot run team 1", test.UserTeamObserverTeam1, []string{"host1", "host2"}, true},
			{"team 1 admin cannot run team 2", test.UserTeamAdminTeam1, []string{"host3"}, true},
			{"team 1 admin cannot run no team", test.UserTeamAdminTeam1, []string{"host4"}, true},
			{"team 1 admin cannot run mix of team 1 and 2", test.UserTeamAdminTeam1, []string{"host1", "host3"}, true},
		}
		for _, c := range enqueueCmdCases {
			t.Run(c.desc, func(t *testing.T) {
				ctx = test.UserContext(ctx, c.user)
				_, err = svc.EnqueueMDMAppleCommand(ctx, rawB64FreeCmd, c.uuids)
				checkAuthErr(t, err, c.shoudFailWithAuth)
			})
		}

		// test with a command that requires a premium license
		ctx = test.UserContext(ctx, test.UserAdmin)
		ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierFree})
		rawB64PremiumCmd := base64.RawStdEncoding.EncodeToString([]byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Command</key>
    <dict>
        <key>RequestType</key>
        <string>%s</string>
    </dict>
    <key>CommandUUID</key>
    <string>uuid</string>
</dict>
</plist>`, "DeviceLock")))
		_, err = svc.EnqueueMDMAppleCommand(ctx, rawB64PremiumCmd, []string{"host1"})
		require.Error(t, err)
		require.ErrorContains(t, err, fleet.ErrMissingLicense.Error())
	})

	cmdUUIDToHostUUIDs := map[string][]string{
		"uuidTm1":       {"host1", "host2"},
		"uuidTm2":       {"host3"},
		"uuidNoTm":      {"host4"},
		"uuidMixTm1Tm2": {"host1", "host3"},
	}
	getResults := func(commandUUID string) ([]*fleet.MDMCommandResult, error) {
		hosts := cmdUUIDToHostUUIDs[commandUUID]
		res := make([]*fleet.MDMCommandResult, 0, len(hosts))
		for _, h := range hosts {
			res = append(res, &fleet.MDMCommandResult{
				HostUUID: h,
			})
		}
		return res, nil
	}

	ds.GetMDMAppleCommandResultsFunc = func(ctx context.Context, commandUUID string) ([]*fleet.MDMCommandResult, error) {
		return getResults(commandUUID)
	}

	ds.GetMDMCommandPlatformFunc = func(ctx context.Context, commandUUID string) (string, error) {
		return "darwin", nil
	}

	t.Run("GetMDMAppleCommandResults", func(t *testing.T) {
		cmdResultsCases := []struct {
			desc              string
			user              *fleet.User
			cmdUUID           string
			shoudFailWithAuth bool
		}{
			{"no role", test.UserNoRoles, "uuidTm1", true},
			{"maintainer can view", test.UserMaintainer, "uuidTm1", false},
			{"maintainer can view", test.UserMaintainer, "uuidTm2", false},
			{"maintainer can view", test.UserMaintainer, "uuidNoTm", false},
			{"maintainer can view", test.UserMaintainer, "uuidMixTm1Tm2", false},
			{"observer can view", test.UserObserver, "uuidTm1", false},
			{"observer can view", test.UserObserver, "uuidTm2", false},
			{"observer can view", test.UserObserver, "uuidNoTm", false},
			{"observer can view", test.UserObserver, "uuidMixTm1Tm2", false},
			{"observer+ can view", test.UserObserverPlus, "uuidTm1", false},
			{"observer+ can view", test.UserObserverPlus, "uuidTm2", false},
			{"observer+ can view", test.UserObserverPlus, "uuidNoTm", false},
			{"observer+ can view", test.UserObserverPlus, "uuidMixTm1Tm2", false},
			{"admin can view", test.UserAdmin, "uuidTm1", false},
			{"admin can view", test.UserAdmin, "uuidTm2", false},
			{"admin can view", test.UserAdmin, "uuidNoTm", false},
			{"admin can view", test.UserAdmin, "uuidMixTm1Tm2", false},
			{"tm1 maintainer can view tm1", test.UserTeamMaintainerTeam1, "uuidTm1", false},
			{"tm1 maintainer cannot view tm2", test.UserTeamMaintainerTeam1, "uuidTm2", true},
			{"tm1 maintainer cannot view no team", test.UserTeamMaintainerTeam1, "uuidNoTm", true},
			{"tm1 maintainer cannot view mix", test.UserTeamMaintainerTeam1, "uuidMixTm1Tm2", true},
			{"tm1 observer can view tm1", test.UserTeamObserverTeam1, "uuidTm1", false},
			{"tm1 observer cannot view tm2", test.UserTeamObserverTeam1, "uuidTm2", true},
			{"tm1 observer cannot view no team", test.UserTeamObserverTeam1, "uuidNoTm", true},
			{"tm1 observer cannot view mix", test.UserTeamObserverTeam1, "uuidMixTm1Tm2", true},
			{"tm1 observer+ can view tm1", test.UserTeamObserverPlusTeam1, "uuidTm1", false},
			{"tm1 observer+ cannot view tm2", test.UserTeamObserverPlusTeam1, "uuidTm2", true},
			{"tm1 observer+ cannot view no team", test.UserTeamObserverPlusTeam1, "uuidNoTm", true},
			{"tm1 observer+ cannot view mix", test.UserTeamObserverPlusTeam1, "uuidMixTm1Tm2", true},
			{"tm1 admin can view tm1", test.UserTeamAdminTeam1, "uuidTm1", false},
			{"tm1 admin cannot view tm2", test.UserTeamAdminTeam1, "uuidTm2", true},
			{"tm1 admin cannot view no team", test.UserTeamAdminTeam1, "uuidNoTm", true},
			{"tm1 admin cannot view mix", test.UserTeamAdminTeam1, "uuidMixTm1Tm2", true},
		}
		for _, c := range cmdResultsCases {
			t.Run(c.desc, func(t *testing.T) {
				ctx = test.UserContext(ctx, c.user)
				_, err = svc.GetMDMAppleCommandResults(ctx, c.cmdUUID)
				checkAuthErr(t, err, c.shoudFailWithAuth)

				// TODO(sarah): move test to shared file
				_, err = svc.GetMDMCommandResults(ctx, c.cmdUUID)
				checkAuthErr(t, err, c.shoudFailWithAuth)
			})
		}
	})

	t.Run("ListMDMAppleCommands", func(t *testing.T) {
		ds.ListMDMAppleCommandsFunc = func(ctx context.Context, tmFilter fleet.TeamFilter, opt *fleet.MDMCommandListOptions) ([]*fleet.MDMAppleCommand, error) {
			return []*fleet.MDMAppleCommand{
				{DeviceID: "no team", TeamID: nil},
				{DeviceID: "tm1", TeamID: ptr.Uint(1)},
				{DeviceID: "tm2", TeamID: ptr.Uint(2)},
			}, nil
		}

		listCmdsCases := []struct {
			desc       string
			user       *fleet.User
			want       []string // the expected device ids in the results
			shouldFail bool     // with forbidden error
		}{
			{"no role", test.UserNoRoles, []string{}, true},
			{"maintainer can view", test.UserMaintainer, []string{"no team", "tm1", "tm2"}, false},
			{"observer can view", test.UserObserver, []string{"no team", "tm1", "tm2"}, false},
			{"observer+ can view", test.UserObserverPlus, []string{"no team", "tm1", "tm2"}, false},
			{"admin can view", test.UserAdmin, []string{"no team", "tm1", "tm2"}, false},
			{"tm1 maintainer can view tm1", test.UserTeamMaintainerTeam1, []string{"tm1"}, false},
			{"tm1 observer can view tm1", test.UserTeamObserverTeam1, []string{"tm1"}, false},
			{"tm1 observer+ can view tm1", test.UserTeamObserverPlusTeam1, []string{"tm1"}, false},
			{"tm1 admin can view tm1", test.UserTeamAdminTeam1, []string{"tm1"}, false},
		}
		for _, c := range listCmdsCases {
			t.Run(c.desc, func(t *testing.T) {
				ctx = test.UserContext(ctx, c.user)
				res, err := svc.ListMDMAppleCommands(ctx, &fleet.MDMCommandListOptions{})
				checkAuthErr(t, err, c.shouldFail)
				if c.shouldFail {
					return
				}

				got := make([]string, len(res))
				for i, r := range res {
					got[i] = r.DeviceID
				}
				require.Equal(t, c.want, got)
			})
		}
	})
}

func TestMDMAppleConfigProfileAuthz(t *testing.T) {
	svc, ctx, ds := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})

	profUUID := "a" + uuid.NewString()
	testCases := []struct {
		name             string
		user             *fleet.User
		shouldFailGlobal bool
		shouldFailTeam   bool
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			false,
			false,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
			true,
		},
		{
			"team admin, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			true,
			false,
		},
		{
			"team admin, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleAdmin}}},
			true,
			true,
		},
		{
			"team maintainer, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			true,
			false,
		},
		{
			"team maintainer, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleMaintainer}}},
			true,
			true,
		},
		{
			"team observer, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
			true,
		},
		{
			"team observer, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserver}}},
			true,
			true,
		},
		{
			"user no roles",
			&fleet.User{ID: 1337},
			true,
			true,
		},
	}

	ds.NewMDMAppleConfigProfileFunc = func(ctx context.Context, cp fleet.MDMAppleConfigProfile) (*fleet.MDMAppleConfigProfile, error) {
		return &cp, nil
	}
	ds.ListMDMAppleConfigProfilesFunc = func(ctx context.Context, teamID *uint) ([]*fleet.MDMAppleConfigProfile, error) {
		return nil, nil
	}
	ds.NewActivityFunc = func(context.Context, *fleet.User, fleet.ActivityDetails) error {
		return nil
	}
	ds.GetMDMAppleProfilesSummaryFunc = func(context.Context, *uint) (*fleet.MDMProfilesSummary, error) {
		return nil, nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hids, tids []uint, puuids, uuids []string) error {
		return nil
	}
	mockGetFuncWithTeamID := func(teamID uint) mock.GetMDMAppleConfigProfileFunc {
		return func(ctx context.Context, puid string) (*fleet.MDMAppleConfigProfile, error) {
			require.Equal(t, profUUID, puid)
			return &fleet.MDMAppleConfigProfile{TeamID: &teamID}, nil
		}
	}
	mockDeleteFuncWithTeamID := func(teamID uint) mock.DeleteMDMAppleConfigProfileFunc {
		return func(ctx context.Context, puid string) error {
			require.Equal(t, profUUID, puid)
			return nil
		}
	}
	mockTeamFuncWithUser := func(u *fleet.User) mock.TeamFunc {
		return func(ctx context.Context, teamID uint) (*fleet.Team, error) {
			if len(u.Teams) > 0 {
				for _, t := range u.Teams {
					if t.ID == teamID {
						return &fleet.Team{ID: teamID, Users: []fleet.TeamUser{{User: *u, Role: t.Role}}}, nil
					}
				}
			}
			return &fleet.Team{}, nil
		}
	}

	checkShouldFail := func(err error, shouldFail bool) {
		if !shouldFail {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
			require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
		}
	}

	mcBytes := mcBytesForTest("Foo", "Bar", "UUID")

	for _, tt := range testCases {
		ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})
		ds.TeamFunc = mockTeamFuncWithUser(tt.user)

		t.Run(tt.name, func(t *testing.T) {
			// test authz create new profile (no team)
			_, err := svc.NewMDMAppleConfigProfile(ctx, 0, bytes.NewReader(mcBytes), nil)
			checkShouldFail(err, tt.shouldFailGlobal)

			// test authz create new profile (team 1)
			_, err = svc.NewMDMAppleConfigProfile(ctx, 1, bytes.NewReader(mcBytes), nil)
			checkShouldFail(err, tt.shouldFailTeam)

			// test authz list profiles (no team)
			_, err = svc.ListMDMAppleConfigProfiles(ctx, 0)
			checkShouldFail(err, tt.shouldFailGlobal)

			// test authz list profiles (team 1)
			_, err = svc.ListMDMAppleConfigProfiles(ctx, 1)
			checkShouldFail(err, tt.shouldFailTeam)

			// test authz get config profile (no team)
			ds.GetMDMAppleConfigProfileFunc = mockGetFuncWithTeamID(0)
			_, err = svc.GetMDMAppleConfigProfile(ctx, profUUID)
			checkShouldFail(err, tt.shouldFailGlobal)

			// test authz delete config profile (no team)
			ds.DeleteMDMAppleConfigProfileFunc = mockDeleteFuncWithTeamID(0)
			err = svc.DeleteMDMAppleConfigProfile(ctx, profUUID)
			checkShouldFail(err, tt.shouldFailGlobal)

			// test authz get config profile (team 1)
			ds.GetMDMAppleConfigProfileFunc = mockGetFuncWithTeamID(1)
			_, err = svc.GetMDMAppleConfigProfile(ctx, profUUID)
			checkShouldFail(err, tt.shouldFailTeam)

			// test authz delete config profile (team 1)
			ds.DeleteMDMAppleConfigProfileFunc = mockDeleteFuncWithTeamID(1)
			err = svc.DeleteMDMAppleConfigProfile(ctx, profUUID)
			checkShouldFail(err, tt.shouldFailTeam)

			// test authz get profiles summary (no team)
			_, err = svc.GetMDMAppleProfilesSummary(ctx, nil)
			checkShouldFail(err, tt.shouldFailGlobal)

			// test authz get profiles summary (no team)
			_, err = svc.GetMDMAppleProfilesSummary(ctx, ptr.Uint(1))
			checkShouldFail(err, tt.shouldFailTeam)
		})
	}
}

func TestNewMDMAppleConfigProfile(t *testing.T) {
	svc, ctx, ds := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	mcBytes := mcBytesForTest("Foo", "Bar", "UUID")
	r := bytes.NewReader(mcBytes)

	ds.NewMDMAppleConfigProfileFunc = func(ctx context.Context, cp fleet.MDMAppleConfigProfile) (*fleet.MDMAppleConfigProfile, error) {
		require.Equal(t, "Foo", cp.Name)
		require.Equal(t, "Bar", cp.Identifier)
		require.Equal(t, mcBytes, []byte(cp.Mobileconfig))
		return &cp, nil
	}
	ds.NewActivityFunc = func(context.Context, *fleet.User, fleet.ActivityDetails) error {
		return nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hids, tids []uint, puuids, uuids []string) error {
		return nil
	}

	cp, err := svc.NewMDMAppleConfigProfile(ctx, 0, r, nil)
	require.NoError(t, err)
	require.Equal(t, "Foo", cp.Name)
	require.Equal(t, "Bar", cp.Identifier)
	require.Equal(t, mcBytes, []byte(cp.Mobileconfig))
}

func mcBytesForTest(name, identifier, uuid string) []byte {
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array/>
	<key>PayloadDisplayName</key>
	<string>%s</string>
	<key>PayloadIdentifier</key>
	<string>%s</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>%s</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>
`, name, identifier, uuid))
}

func TestHostDetailsMDMProfiles(t *testing.T) {
	svc, ctx, ds := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	expected := []fleet.HostMDMAppleProfile{
		{HostUUID: "H057-UU1D-1337", Name: "NAME-5", ProfileUUID: "a" + uuid.NewString(), CommandUUID: "CMD-UU1D-5", Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, Detail: ""},
		{HostUUID: "H057-UU1D-1337", Name: "NAME-9", ProfileUUID: "a" + uuid.NewString(), CommandUUID: "CMD-UU1D-8", Status: &fleet.MDMDeliveryVerifying, OperationType: fleet.MDMOperationTypeInstall, Detail: ""},
		{HostUUID: "H057-UU1D-1337", Name: "NAME-13", ProfileUUID: "a" + uuid.NewString(), CommandUUID: "CMD-UU1D-13", Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeRemove, Detail: "Error removing profile"},
	}

	ds.GetHostMDMAppleProfilesFunc = func(ctx context.Context, hostUUID string) ([]fleet.HostMDMAppleProfile, error) {
		if hostUUID == "H057-UU1D-1337" {
			return expected, nil
		}
		return []fleet.HostMDMAppleProfile{}, nil
	}
	ds.HostFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
		if hostID == uint(42) {
			return &fleet.Host{ID: uint(42), UUID: "H057-UU1D-1337", Platform: "darwin"}, nil
		}
		return &fleet.Host{ID: hostID, UUID: "WR0N6-UU1D", Platform: "darwin"}, nil
	}
	ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.Host, error) {
		if identifier == "h0571d3n71f13r" {
			return &fleet.Host{ID: uint(42), UUID: "H057-UU1D-1337", Platform: "darwin"}, nil
		}
		return &fleet.Host{ID: uint(21), UUID: "WR0N6-UU1D", Platform: "darwin"}, nil
	}
	ds.LoadHostSoftwareFunc = func(ctx context.Context, host *fleet.Host, includeCVEScores bool) error {
		return nil
	}
	ds.ListLabelsForHostFunc = func(ctx context.Context, hid uint) ([]*fleet.Label, error) {
		return nil, nil
	}
	ds.ListPacksForHostFunc = func(ctx context.Context, hid uint) (packs []*fleet.Pack, err error) {
		return nil, nil
	}
	ds.ListHostBatteriesFunc = func(ctx context.Context, id uint) ([]*fleet.HostBattery, error) {
		return nil, nil
	}
	ds.ListPoliciesForHostFunc = func(ctx context.Context, host *fleet.Host) ([]*fleet.HostPolicy, error) {
		return nil, nil
	}
	ds.GetHostMDMMacOSSetupFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDMMacOSSetup, error) {
		return nil, nil
	}
	ds.GetHostLockWipeStatusFunc = func(ctx context.Context, hostID uint, fleetPlatform string) (*fleet.HostLockWipeStatus, error) {
		return &fleet.HostLockWipeStatus{}, nil
	}

	expectedNilSlice := []fleet.HostMDMAppleProfile(nil)
	expectedEmptySlice := []fleet.HostMDMAppleProfile{}

	cases := []struct {
		name           string
		mdmEnabled     bool
		hostID         *uint
		hostIdentifier *string
		expected       *[]fleet.HostMDMAppleProfile
	}{
		{
			name:           "TestGetHostMDMProfilesOK",
			mdmEnabled:     true,
			hostID:         ptr.Uint(42),
			hostIdentifier: nil,
			expected:       &expected,
		},
		{
			name:           "TestGetHostMDMProfilesEmpty",
			mdmEnabled:     true,
			hostID:         ptr.Uint(21),
			hostIdentifier: nil,
			expected:       &expectedEmptySlice,
		},
		{
			name:           "TestGetHostMDMProfilesNil",
			mdmEnabled:     false,
			hostID:         ptr.Uint(42),
			hostIdentifier: nil,
			expected:       &expectedNilSlice,
		},
		{
			name:           "TestHostByIdentifierMDMProfilesOK",
			mdmEnabled:     true,
			hostID:         nil,
			hostIdentifier: ptr.String("h0571d3n71f13r"),
			expected:       &expected,
		},
		{
			name:           "TestHostByIdentifierMDMProfilesNil",
			mdmEnabled:     false,
			hostID:         nil,
			hostIdentifier: ptr.String("h0571d3n71f13r"),
			expected:       &expectedNilSlice,
		},
		{
			name:           "TestHostByIdentifierMDMProfilesEmpty",
			mdmEnabled:     true,
			hostID:         nil,
			hostIdentifier: ptr.String("4n07h3r1d3n71f13r"),
			expected:       &expectedEmptySlice,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: c.mdmEnabled}}, nil
			}
			ds.AppConfigFuncInvoked = false
			ds.HostFuncInvoked = false
			ds.HostByIdentifierFuncInvoked = false
			ds.GetHostMDMAppleProfilesFuncInvoked = false

			var gotHost *fleet.HostDetail
			if c.hostID != nil {
				h, err := svc.GetHost(ctx, *c.hostID, fleet.HostDetailOptions{})
				require.NoError(t, err)
				require.True(t, ds.HostFuncInvoked)
				gotHost = h
			}
			if c.hostIdentifier != nil {
				h, err := svc.HostByIdentifier(ctx, *c.hostIdentifier, fleet.HostDetailOptions{})
				require.NoError(t, err)
				require.True(t, ds.HostByIdentifierFuncInvoked)
				gotHost = h
			}
			require.NotNil(t, gotHost)
			require.True(t, ds.AppConfigFuncInvoked)

			if !c.mdmEnabled {
				var ep []fleet.HostMDMProfile
				switch c.expected {
				case &expectedNilSlice:
					ns := []fleet.HostMDMProfile(nil)
					ep = ns
				case &expectedEmptySlice:
					ep = []fleet.HostMDMProfile{}
				default:
					for _, p := range *c.expected {
						ep = append(ep, p.ToHostMDMProfile())
					}
				}
				require.Equal(t, gotHost.MDM.Profiles, &ep)
				return
			}

			require.True(t, ds.GetHostMDMAppleProfilesFuncInvoked)
			require.NotNil(t, gotHost.MDM.Profiles)
			ep := make([]fleet.HostMDMProfile, 0, len(*gotHost.MDM.Profiles))
			for _, p := range *c.expected {
				ep = append(ep, p.ToHostMDMProfile())
			}
			require.ElementsMatch(t, ep, *gotHost.MDM.Profiles)
		})
	}
}

func TestMDMCommandAuthz(t *testing.T) {
	svc, ctx, ds := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})

	ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
		switch hostID {
		case 1:
			return &fleet.Host{UUID: "test-host-team-1", TeamID: ptr.Uint(1)}, nil
		default:
			return &fleet.Host{UUID: "test-host-no-team"}, nil
		}
	}

	ds.GetHostMDMCheckinInfoFunc = func(ctx context.Context, hostUUID string) (*fleet.HostMDMCheckinInfo, error) {
		return &fleet.HostMDMCheckinInfo{}, nil
	}

	ds.NewActivityFunc = func(context.Context, *fleet.User, fleet.ActivityDetails) error {
		return nil
	}

	var mdmEnabled atomic.Bool
	ds.GetNanoMDMEnrollmentFunc = func(ctx context.Context, hostUUID string) (*fleet.NanoEnrollment, error) {
		// This function is called twice during EnqueueMDMAppleCommandRemoveEnrollmentProfile.
		// It first is called to check that the device is enrolled as a pre-condition to enqueueing the
		// command. It is called second time after the command has been enqueued to check whether
		// the device was successfully unenrolled.
		//
		// For each test run, the bool should be initialized to true to simulate an existing device
		// that is initially enrolled to Fleet's MDM.
		enroll := fleet.NanoEnrollment{
			Enabled: mdmEnabled.Swap(!mdmEnabled.Load()),
		}
		return &enroll, nil
	}

	testCases := []struct {
		name             string
		user             *fleet.User
		shouldFailGlobal bool
		shouldFailTeam   bool
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			false,
			false,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
			true,
		},
		{
			"team admin, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			true,
			false,
		},
		{
			"team admin, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleAdmin}}},
			true,
			true,
		},
		{
			"team maintainer, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			true,
			false,
		},
		{
			"team maintainer, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleMaintainer}}},
			true,
			true,
		},
		{
			"team observer, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
			true,
		},
		{
			"team observer, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserver}}},
			true,
			true,
		},
		{
			"user no roles",
			&fleet.User{ID: 1337},
			true,
			true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			mdmEnabled.Store(true)
			err := svc.EnqueueMDMAppleCommandRemoveEnrollmentProfile(ctx, 42) // global host
			if !tt.shouldFailGlobal {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
			}

			mdmEnabled.Store(true)
			err = svc.EnqueueMDMAppleCommandRemoveEnrollmentProfile(ctx, 1) // host belongs to team 1
			if !tt.shouldFailTeam {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
			}
		})
	}
}

func TestMDMAuthenticate(t *testing.T) {
	ds := new(mock.Store)
	svc := MDMAppleCheckinAndCommandService{ds: ds}
	ctx := context.Background()
	uuid, serial, model := "ABC-DEF-GHI", "XYZABC", "MacBookPro 16,1"

	ds.IngestMDMAppleDeviceFromCheckinFunc = func(ctx context.Context, mdmHost fleet.MDMAppleHostDetails) error {
		require.Equal(t, uuid, mdmHost.UDID)
		require.Equal(t, serial, mdmHost.SerialNumber)
		require.Equal(t, model, mdmHost.Model)
		return nil
	}

	ds.GetHostMDMCheckinInfoFunc = func(ct context.Context, hostUUID string) (*fleet.HostMDMCheckinInfo, error) {
		require.Equal(t, uuid, hostUUID)
		return &fleet.HostMDMCheckinInfo{HardwareSerial: serial, DisplayName: fmt.Sprintf("%s (%s)", model, serial), InstalledFromDEP: false}, nil
	}

	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		a, ok := activity.(*fleet.ActivityTypeMDMEnrolled)
		require.True(t, ok)
		require.Nil(t, user)
		require.Equal(t, "mdm_enrolled", activity.ActivityName())
		require.Equal(t, serial, a.HostSerial)
		require.Equal(t, a.HostDisplayName, fmt.Sprintf("%s (%s)", model, serial))
		require.False(t, a.InstalledFromDEP)
		require.Equal(t, fleet.MDMPlatformApple, a.MDMPlatform)
		return nil
	}

	ds.ResetMDMAppleEnrollmentFunc = func(ctx context.Context, hostUUID string) error {
		require.Equal(t, uuid, hostUUID)
		return nil
	}

	err := svc.Authenticate(
		&mdm.Request{Context: ctx},
		&mdm.Authenticate{
			Enrollment: mdm.Enrollment{
				UDID: uuid,
			},
			SerialNumber: serial,
			Model:        model,
		},
	)
	require.NoError(t, err)
	require.True(t, ds.IngestMDMAppleDeviceFromCheckinFuncInvoked)
	require.True(t, ds.GetHostMDMCheckinInfoFuncInvoked)
	require.True(t, ds.NewActivityFuncInvoked)
	require.True(t, ds.ResetMDMAppleEnrollmentFuncInvoked)
}

func TestMDMTokenUpdate(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)
	mdmStorage := &mock.MDMAppleStore{}
	pushFactory, _ := newMockAPNSPushProviderFactory()
	pusher := nanomdm_pushsvc.New(
		mdmStorage,
		mdmStorage,
		pushFactory,
		NewNanoMDMLogger(kitlog.NewJSONLogger(os.Stdout)),
	)
	cmdr := apple_mdm.NewMDMAppleCommander(mdmStorage, pusher)
	svc := MDMAppleCheckinAndCommandService{ds: ds, commander: cmdr, logger: kitlog.NewNopLogger()}
	uuid, serial, model, wantTeamID := "ABC-DEF-GHI", "XYZABC", "MacBookPro 16,1", uint(12)

	ds.GetNanoMDMEnrollmentFunc = func(ctx context.Context, hostUUID string) (*fleet.NanoEnrollment, error) {
		return &fleet.NanoEnrollment{Enabled: true, Type: "Device", TokenUpdateTally: 1}, nil
	}

	ds.GetHostMDMCheckinInfoFunc = func(ct context.Context, hostUUID string) (*fleet.HostMDMCheckinInfo, error) {
		require.Equal(t, uuid, hostUUID)
		return &fleet.HostMDMCheckinInfo{
			HardwareSerial:     serial,
			DisplayName:        model,
			InstalledFromDEP:   true,
			TeamID:             wantTeamID,
			DEPAssignedToFleet: true,
		}, nil
	}

	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hids, tids []uint, puuids, uuids []string) error {
		return nil
	}

	ds.NewJobFunc = func(ctx context.Context, j *fleet.Job) (*fleet.Job, error) {
		return j, nil
	}

	err := svc.TokenUpdate(
		&mdm.Request{Context: ctx, EnrollID: &mdm.EnrollID{ID: uuid}},
		&mdm.TokenUpdate{
			Enrollment: mdm.Enrollment{
				UDID: uuid,
			},
		},
	)
	require.NoError(t, err)
	require.True(t, ds.BulkSetPendingMDMHostProfilesFuncInvoked)
	require.True(t, ds.GetHostMDMCheckinInfoFuncInvoked)
	require.True(t, ds.NewJobFuncInvoked)
	ds.BulkSetPendingMDMHostProfilesFuncInvoked = false
	ds.GetHostMDMCheckinInfoFuncInvoked = false
	ds.NewJobFuncInvoked = false

	// with enrollment reference
	err = svc.TokenUpdate(
		&mdm.Request{
			Context:  ctx,
			EnrollID: &mdm.EnrollID{ID: uuid},
			Params:   map[string]string{"enroll_reference": "abcd"},
		},
		&mdm.TokenUpdate{
			Enrollment: mdm.Enrollment{
				UDID: uuid,
			},
		},
	)
	require.NoError(t, err)
	require.True(t, ds.BulkSetPendingMDMHostProfilesFuncInvoked)
	require.True(t, ds.GetHostMDMCheckinInfoFuncInvoked)
	require.True(t, ds.NewJobFuncInvoked)
}

func TestMDMCheckout(t *testing.T) {
	ds := new(mock.Store)
	svc := MDMAppleCheckinAndCommandService{ds: ds}
	ctx := context.Background()
	uuid, serial, installedFromDEP, displayName := "ABC-DEF-GHI", "XYZABC", true, "Test's MacBook"

	ds.UpdateHostTablesOnMDMUnenrollFunc = func(ctx context.Context, hostUUID string) error {
		require.Equal(t, uuid, hostUUID)
		return nil
	}

	ds.GetHostMDMCheckinInfoFunc = func(ct context.Context, hostUUID string) (*fleet.HostMDMCheckinInfo, error) {
		require.Equal(t, uuid, hostUUID)
		return &fleet.HostMDMCheckinInfo{
			HardwareSerial:   serial,
			DisplayName:      displayName,
			InstalledFromDEP: installedFromDEP,
		}, nil
	}

	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		a, ok := activity.(*fleet.ActivityTypeMDMUnenrolled)
		require.True(t, ok)
		require.Nil(t, user)
		require.Equal(t, "mdm_unenrolled", activity.ActivityName())
		require.Equal(t, serial, a.HostSerial)
		require.Equal(t, displayName, a.HostDisplayName)
		require.True(t, a.InstalledFromDEP)
		return nil
	}

	err := svc.CheckOut(
		&mdm.Request{Context: ctx},
		&mdm.CheckOut{
			Enrollment: mdm.Enrollment{
				UDID: uuid,
			},
		},
	)
	require.NoError(t, err)
	require.True(t, ds.UpdateHostTablesOnMDMUnenrollFuncInvoked)
	require.True(t, ds.GetHostMDMCheckinInfoFuncInvoked)
	require.True(t, ds.NewActivityFuncInvoked)
}

func TestMDMCommandAndReportResultsProfileHandling(t *testing.T) {
	ctx := context.Background()
	hostUUID := "ABC-DEF-GHI"
	commandUUID := "COMMAND-UUID"
	profileIdentifier := "PROFILE-IDENTIFIER"

	cases := []struct {
		status      string
		requestType string
		errors      []mdm.ErrorChain
		want        *fleet.HostMDMAppleProfile
		prevRetries uint
	}{
		{
			status:      "Acknowledged",
			requestType: "InstallProfile",
			errors:      nil,
			want: &fleet.HostMDMAppleProfile{
				Status:        &fleet.MDMDeliveryVerifying,
				Detail:        "",
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		{
			status:      "Acknowledged",
			requestType: "RemoveProfile",
			errors:      nil,
			want: &fleet.HostMDMAppleProfile{
				Status:        &fleet.MDMDeliveryVerifying,
				Detail:        "",
				OperationType: fleet.MDMOperationTypeRemove,
			},
		},
		{
			status:      "Error",
			requestType: "InstallProfile",
			errors: []mdm.ErrorChain{
				{ErrorCode: 123, ErrorDomain: "testDomain", USEnglishDescription: "testMessage"},
			},
			prevRetries: 0, // expect to retry
			want: &fleet.HostMDMAppleProfile{
				Status:        &fleet.MDMDeliveryPending,
				Detail:        "",
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		{
			status:      "Error",
			requestType: "InstallProfile",
			errors: []mdm.ErrorChain{
				{ErrorCode: 123, ErrorDomain: "testDomain", USEnglishDescription: "testMessage"},
			},
			prevRetries: 1, // expect to fail
			want: &fleet.HostMDMAppleProfile{
				Status:        &fleet.MDMDeliveryFailed,
				Detail:        "testDomain (123): testMessage\n",
				OperationType: fleet.MDMOperationTypeInstall,
			},
		},
		{
			status:      "Error",
			requestType: "RemoveProfile",
			errors: []mdm.ErrorChain{
				{ErrorCode: 123, ErrorDomain: "testDomain", USEnglishDescription: "testMessage"},
				{ErrorCode: 321, ErrorDomain: "domainTest", USEnglishDescription: "messageTest"},
			},
			want: &fleet.HostMDMAppleProfile{
				Status:        &fleet.MDMDeliveryFailed,
				Detail:        "testDomain (123): testMessage\ndomainTest (321): messageTest\n",
				OperationType: fleet.MDMOperationTypeRemove,
			},
		},
		{
			status:      "Error",
			requestType: "RemoveProfile",
			errors:      nil,
			want: &fleet.HostMDMAppleProfile{
				Status:        &fleet.MDMDeliveryFailed,
				Detail:        "",
				OperationType: fleet.MDMOperationTypeRemove,
			},
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("%s%s-%d", c.requestType, c.status, i), func(t *testing.T) {
			ds := new(mock.Store)
			svc := MDMAppleCheckinAndCommandService{ds: ds}
			ds.GetMDMAppleCommandRequestTypeFunc = func(ctx context.Context, targetCmd string) (string, error) {
				require.Equal(t, commandUUID, targetCmd)
				return c.requestType, nil
			}
			ds.UpdateOrDeleteHostMDMAppleProfileFunc = func(ctx context.Context, profile *fleet.HostMDMAppleProfile) error {
				c.want.CommandUUID = commandUUID
				c.want.HostUUID = hostUUID
				require.Equal(t, c.want, profile)
				return nil
			}
			ds.GetHostMDMProfileRetryCountByCommandUUIDFunc = func(ctx context.Context, host *fleet.Host, cmdUUID string) (fleet.HostMDMProfileRetryCount, error) {
				require.Equal(t, hostUUID, host.UUID)
				require.Equal(t, commandUUID, cmdUUID)
				return fleet.HostMDMProfileRetryCount{ProfileIdentifier: profileIdentifier, Retries: c.prevRetries}, nil
			}
			ds.UpdateHostMDMProfilesVerificationFunc = func(ctx context.Context, host *fleet.Host, toVerify, toFail, toRetry []string) error {
				require.Equal(t, hostUUID, host.UUID)
				require.Nil(t, toVerify)
				require.Nil(t, toFail)
				require.ElementsMatch(t, toRetry, []string{profileIdentifier})
				return nil
			}

			_, err := svc.CommandAndReportResults(
				&mdm.Request{Context: ctx},
				&mdm.CommandResults{
					Enrollment:  mdm.Enrollment{UDID: hostUUID},
					CommandUUID: commandUUID,
					Status:      c.status,
					RequestType: c.requestType,
					ErrorChain:  c.errors,
				},
			)
			require.NoError(t, err)
			require.True(t, ds.GetMDMAppleCommandRequestTypeFuncInvoked)
			var shouldCheckCount, shouldRetry, shouldUpdateOrDelete bool
			if c.requestType == "InstallProfile" && c.status == "Error" {
				shouldCheckCount = true
			}
			if shouldCheckCount && c.prevRetries == uint(0) {
				shouldRetry = true
			}
			if c.requestType == "RemoveProfile" || (c.requestType == "InstallProfile" && !shouldRetry) {
				shouldUpdateOrDelete = true
			}
			require.Equal(t, shouldCheckCount, ds.GetHostMDMProfileRetryCountByCommandUUIDFuncInvoked)
			require.Equal(t, shouldRetry, ds.UpdateHostMDMProfilesVerificationFuncInvoked)
			require.Equal(t, shouldUpdateOrDelete, ds.UpdateOrDeleteHostMDMAppleProfileFuncInvoked)
		})
	}
}

func TestMDMBatchSetAppleProfiles(t *testing.T) {
	svc, ctx, ds := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})

	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		return &fleet.Team{ID: 1, Name: name}, nil
	}
	ds.TeamFunc = func(ctx context.Context, id uint) (*fleet.Team, error) {
		return &fleet.Team{ID: id, Name: "team"}, nil
	}
	ds.BatchSetMDMAppleProfilesFunc = func(ctx context.Context, teamID *uint, profiles []*fleet.MDMAppleConfigProfile) error {
		return nil
	}
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hids, tids []uint, puuids, uuids []string) error {
		return nil
	}

	type testCase struct {
		name     string
		user     *fleet.User
		premium  bool
		teamID   *uint
		teamName *string
		profiles [][]byte
		wantErr  string
	}

	testCases := []testCase{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			nil,
			nil,
			nil,
			"",
		},
		{
			"global admin, team",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			true,
			ptr.Uint(1),
			nil,
			nil,
			"",
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			false,
			nil,
			nil,
			nil,
			"",
		},
		{
			"global maintainer, team",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			true,
			ptr.Uint(1),
			nil,
			nil,
			"",
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			false,
			nil,
			nil,
			nil,
			authz.ForbiddenErrorMessage,
		},
		{
			"team admin, DOES belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			true,
			ptr.Uint(1),
			nil,
			nil,
			"",
		},
		{
			"team admin, DOES belong to team by name",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			true,
			nil,
			ptr.String("team"),
			nil,
			"",
		},
		{
			"team admin, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleAdmin}}},
			true,
			ptr.Uint(1),
			nil,
			nil,
			authz.ForbiddenErrorMessage,
		},
		{
			"team admin, DOES NOT belong to team by name",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleAdmin}}},
			true,
			nil,
			ptr.String("team"),
			nil,
			authz.ForbiddenErrorMessage,
		},
		{
			"team maintainer, DOES belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			true,
			ptr.Uint(1),
			nil,
			nil,
			"",
		},
		{
			"team maintainer, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleMaintainer}}},
			true,
			ptr.Uint(1),
			nil,
			nil,
			authz.ForbiddenErrorMessage,
		},
		{
			"team observer, DOES belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
			ptr.Uint(1),
			nil,
			nil,
			authz.ForbiddenErrorMessage,
		},
		{
			"team observer, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserver}}},
			true,
			ptr.Uint(1),
			nil,
			nil,
			authz.ForbiddenErrorMessage,
		},
		{
			"user no roles",
			&fleet.User{ID: 1337},
			false,
			nil,
			nil,
			nil,
			authz.ForbiddenErrorMessage,
		},
		{
			"team id with free license",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			ptr.Uint(1),
			nil,
			nil,
			ErrMissingLicense.Error(),
		},
		{
			"team name with free license",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			nil,
			ptr.String("team"),
			nil,
			ErrMissingLicense.Error(),
		},
		{
			"team id and name specified",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			true,
			ptr.Uint(1),
			ptr.String("team"),
			nil,
			"cannot specify both team_id and team_name",
		},
		{
			"duplicate profile name",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			true,
			ptr.Uint(1),
			nil,
			[][]byte{
				mobileconfigForTest("N1", "I1"),
				mobileconfigForTest("N1", "I2"),
			},
			`More than one configuration profile have the same name (PayloadDisplayName): "N1"`,
		},
		{
			"duplicate profile identifier",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			true,
			ptr.Uint(1),
			nil,
			[][]byte{
				mobileconfigForTest("N1", "I1"),
				mobileconfigForTest("N2", "I2"),
				mobileconfigForTest("N3", "I1"),
			},
			`More than one configuration profile have the same identifier (PayloadIdentifier): "I1"`,
		},
		{
			"no duplicates",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			nil,
			nil,
			[][]byte{
				mobileconfigForTest("N1", "I1"),
				mobileconfigForTest("N2", "I2"),
				mobileconfigForTest("N3", "I3"),
			},
			``,
		},
		{
			"unsupported payload type",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			nil,
			nil,
			[][]byte{[]byte(`<?xml version="1.0" encoding="UTF-8"?>
			<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
			<plist version="1.0">
			<dict>
				<key>PayloadContent</key>
				<array>
					<dict>
						<key>Enable</key>
						<string>On</string>
						<key>PayloadDisplayName</key>
						<string>FileVault 2</string>
						<key>PayloadIdentifier</key>
						<string>com.apple.MCX.FileVault2.A5874654-D6BA-4649-84B5-43847953B369</string>
						<key>PayloadType</key>
						<string>com.apple.MCX.FileVault2</string>
						<key>PayloadUUID</key>
						<string>A5874654-D6BA-4649-84B5-43847953B369</string>
						<key>PayloadVersion</key>
						<integer>1</integer>
					</dict>
				</array>
				<key>PayloadDisplayName</key>
				<string>Config Profile Name</string>
				<key>PayloadIdentifier</key>
				<string>com.example.config.FE42D0A2-DBA9-4B72-BC67-9288665B8D59</string>
				<key>PayloadType</key>
				<string>Configuration</string>
				<key>PayloadUUID</key>
				<string>FE42D0A2-DBA9-4B72-BC67-9288665B8D59</string>
				<key>PayloadVersion</key>
				<integer>1</integer>
			</dict>
			</plist>`)},
			"unsupported PayloadType(s)",
		},
	}
	for name := range fleetmdm.FleetReservedProfileNames() {
		testCases = append(testCases,
			testCase{
				"reserved payload outer name " + name,
				&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
				true,
				nil,
				nil,
				[][]byte{mobileconfigForTest(name, "I1")},
				name,
			},
			testCase{
				"reserved payload inner name " + name,
				&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
				true,
				nil,
				nil,
				[][]byte{mobileconfigForTestWithContent("N1", "I1", "I1", "PayloadType", name)},
				name,
			},
		)
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			defer func() { ds.BatchSetMDMAppleProfilesFuncInvoked = false }()

			// prepare the context with the user and license
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})
			tier := fleet.TierFree
			if tt.premium {
				tier = fleet.TierPremium
			}
			ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: tier})

			err := svc.BatchSetMDMAppleProfiles(ctx, tt.teamID, tt.teamName, tt.profiles, false, false)
			if tt.wantErr == "" {
				require.NoError(t, err)
				require.True(t, ds.BatchSetMDMAppleProfilesFuncInvoked)
				return
			}
			require.Error(t, err)
			require.ErrorContains(t, err, tt.wantErr)
			require.False(t, ds.BatchSetMDMAppleProfilesFuncInvoked)
		})
	}
}

func TestMDMBatchSetAppleProfilesBoolArgs(t *testing.T) {
	svc, ctx, ds := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})

	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		return &fleet.Team{ID: 1, Name: name}, nil
	}
	ds.TeamFunc = func(ctx context.Context, id uint) (*fleet.Team, error) {
		return &fleet.Team{ID: id, Name: "team"}, nil
	}
	ds.BatchSetMDMAppleProfilesFunc = func(ctx context.Context, teamID *uint, profiles []*fleet.MDMAppleConfigProfile) error {
		return nil
	}
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hids, tids []uint, profileUUIDs, uuids []string) error {
		return nil
	}

	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})
	ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierPremium})

	// dry run doesn't call methods that save stuff in the db
	err := svc.BatchSetMDMAppleProfiles(ctx, nil, nil, [][]byte{}, true, false)
	require.NoError(t, err)
	require.False(t, ds.BatchSetMDMAppleProfilesFuncInvoked)
	require.False(t, ds.BulkSetPendingMDMHostProfilesFuncInvoked)
	require.False(t, ds.NewActivityFuncInvoked)

	// skipping bulk set only skips that method
	err = svc.BatchSetMDMAppleProfiles(ctx, nil, nil, [][]byte{}, false, true)
	require.NoError(t, err)
	require.True(t, ds.BatchSetMDMAppleProfilesFuncInvoked)
	require.False(t, ds.BulkSetPendingMDMHostProfilesFuncInvoked)
	require.True(t, ds.NewActivityFuncInvoked)
}

func TestUpdateMDMAppleSettings(t *testing.T) {
	svc, ctx, ds := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})

	ds.TeamFunc = func(ctx context.Context, id uint) (*fleet.Team, error) {
		return &fleet.Team{ID: id, Name: "team"}, nil
	}
	ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		return team, nil
	}
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, appConfig *fleet.AppConfig) error {
		return nil
	}

	testCases := []struct {
		name    string
		user    *fleet.User
		premium bool
		teamID  *uint
		wantErr string
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			nil,
			ErrMissingLicense.Error(),
		},
		{
			"global admin premium",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			true,
			nil,
			"",
		},
		{
			"global admin, team",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			true,
			ptr.Uint(1),
			"",
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			false,
			nil,
			ErrMissingLicense.Error(),
		},
		{
			"global maintainer premium",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			true,
			nil,
			"",
		},
		{
			"global maintainer, team",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			true,
			ptr.Uint(1),
			"",
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
			nil,
			authz.ForbiddenErrorMessage,
		},
		{
			"team admin, DOES belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			true,
			ptr.Uint(1),
			"",
		},
		{
			"team admin, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleAdmin}}},
			true,
			ptr.Uint(1),
			authz.ForbiddenErrorMessage,
		},
		{
			"team maintainer, DOES belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			true,
			ptr.Uint(1),
			"",
		},
		{
			"team maintainer, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleMaintainer}}},
			true,
			ptr.Uint(1),
			authz.ForbiddenErrorMessage,
		},
		{
			"team observer, DOES belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
			ptr.Uint(1),
			authz.ForbiddenErrorMessage,
		},
		{
			"team observer, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserver}}},
			true,
			ptr.Uint(1),
			authz.ForbiddenErrorMessage,
		},
		{
			"user no roles",
			&fleet.User{ID: 1337},
			true,
			nil,
			authz.ForbiddenErrorMessage,
		},
		{
			"team id with free license",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			ptr.Uint(1),
			ErrMissingLicense.Error(),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// prepare the context with the user and license
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})
			tier := fleet.TierFree
			if tt.premium {
				tier = fleet.TierPremium
			}
			ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: tier})

			err := svc.UpdateMDMAppleSettings(ctx, fleet.MDMAppleSettingsPayload{TeamID: tt.teamID})
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestUpdateMDMAppleSetup(t *testing.T) {
	setupTest := func(tier string) (fleet.Service, context.Context, *mock.Store) {
		svc, ctx, ds := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: tier})
		ds.TeamFunc = func(ctx context.Context, id uint) (*fleet.Team, error) {
			return &fleet.Team{ID: id, Name: "team"}, nil
		}
		ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
			return team, nil
		}
		ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			return nil
		}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}, nil
		}
		ds.SaveAppConfigFunc = func(ctx context.Context, appConfig *fleet.AppConfig) error {
			return nil
		}
		return svc, ctx, ds
	}

	type testCase struct {
		name    string
		user    *fleet.User
		teamID  *uint
		wantErr string
	}
	// TODO: Add tests for gitops and observer plus roles? (Settings endpoint test above may also need to be updated)

	t.Run("FreeTier", func(t *testing.T) {
		freeTestCases := []testCase{
			{
				"global admin",
				&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
				nil,
				"Requires Fleet Premium license",
			},
			{
				"global maintainer",
				&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
				nil,
				"Requires Fleet Premium license",
			},
			{
				"team id with free license",
				&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
				ptr.Uint(1),
				"Requires Fleet Premium license",
			},
		}
		svc, ctx, _ := setupTest(fleet.TierFree)
		for _, tt := range freeTestCases {
			t.Run(tt.name, func(t *testing.T) {
				// prepare the context with the user and license
				ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})
				err := svc.UpdateMDMAppleSetup(ctx, fleet.MDMAppleSetupPayload{TeamID: tt.teamID})
				if tt.wantErr == "" {
					require.NoError(t, err)
					return
				}
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			})
		}
	})
	t.Run("PremiumTier", func(t *testing.T) {
		premiumTestCases := []testCase{
			{
				"global admin premium",
				&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
				nil,
				"",
			},
			{
				"global admin, team",
				&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
				ptr.Uint(1),
				"",
			},
			{
				"global maintainer premium",
				&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
				nil,
				"",
			},
			{
				"global maintainer, team",
				&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
				ptr.Uint(1),
				"",
			},
			{
				"global observer",
				&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
				nil,
				authz.ForbiddenErrorMessage,
			},
			{
				"team admin, DOES belong to team",
				&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
				ptr.Uint(1),
				"",
			},
			{
				"team admin, DOES NOT belong to team",
				&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleAdmin}}},
				ptr.Uint(1),
				authz.ForbiddenErrorMessage,
			},
			{
				"team maintainer, DOES belong to team",
				&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
				ptr.Uint(1),
				"",
			},
			{
				"team maintainer, DOES NOT belong to team",
				&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleMaintainer}}},
				ptr.Uint(1),
				authz.ForbiddenErrorMessage,
			},
			{
				"team observer, DOES belong to team",
				&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
				ptr.Uint(1),
				authz.ForbiddenErrorMessage,
			},
			{
				"team observer, DOES NOT belong to team",
				&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserver}}},
				ptr.Uint(1),
				authz.ForbiddenErrorMessage,
			},
			{
				"user no roles",
				&fleet.User{ID: 1337},
				nil,
				authz.ForbiddenErrorMessage,
			},
		}
		svc, ctx, _ := setupTest(fleet.TierPremium)
		for _, tt := range premiumTestCases {
			t.Run(tt.name, func(t *testing.T) {
				// prepare the context with the user and license
				ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})
				err := svc.UpdateMDMAppleSetup(ctx, fleet.MDMAppleSetupPayload{TeamID: tt.teamID})
				if tt.wantErr == "" {
					require.NoError(t, err)
					return
				}
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			})
		}
	})
}

func TestMDMAppleReconcileAppleProfiles(t *testing.T) {
	ctx := context.Background()
	mdmStorage := &mock.MDMAppleStore{}
	ds := new(mock.Store)
	pushFactory, _ := newMockAPNSPushProviderFactory()
	pusher := nanomdm_pushsvc.New(
		mdmStorage,
		mdmStorage,
		pushFactory,
		NewNanoMDMLogger(kitlog.NewNopLogger()),
	)
	cmdr := apple_mdm.NewMDMAppleCommander(mdmStorage, pusher)
	hostUUID, hostUUID2 := "ABC-DEF", "GHI-JKL"
	contents1 := []byte("test-content-1")
	contents1Base64 := base64.StdEncoding.EncodeToString(contents1)
	contents2 := []byte("test-content-2")
	contents2Base64 := base64.StdEncoding.EncodeToString(contents2)
	contents4 := []byte("test-content-4")
	contents4Base64 := base64.StdEncoding.EncodeToString(contents4)

	p1, p2, p3, p4 := "a"+uuid.NewString(), "a"+uuid.NewString(), "a"+uuid.NewString(), "a"+uuid.NewString()
	ds.ListMDMAppleProfilesToInstallFunc = func(ctx context.Context) ([]*fleet.MDMAppleProfilePayload, error) {
		return []*fleet.MDMAppleProfilePayload{
			{ProfileUUID: p1, ProfileIdentifier: "com.add.profile", HostUUID: hostUUID},
			{ProfileUUID: p2, ProfileIdentifier: "com.add.profile.two", HostUUID: hostUUID},
			{ProfileUUID: p2, ProfileIdentifier: "com.add.profile.two", HostUUID: hostUUID2},
			{ProfileUUID: p4, ProfileIdentifier: "com.add.profile.four", HostUUID: hostUUID2},
		}, nil
	}

	ds.ListMDMAppleProfilesToRemoveFunc = func(ctx context.Context) ([]*fleet.MDMAppleProfilePayload, error) {
		return []*fleet.MDMAppleProfilePayload{
			{ProfileUUID: p3, ProfileIdentifier: "com.remove.profile", HostUUID: hostUUID},
			{ProfileUUID: p3, ProfileIdentifier: "com.remove.profile", HostUUID: hostUUID2},
		}, nil
	}

	ds.GetMDMAppleProfilesContentsFunc = func(ctx context.Context, profileUUIDs []string) (map[string]mobileconfig.Mobileconfig, error) {
		require.ElementsMatch(t, []string{p1, p2, p4}, profileUUIDs)
		// only those profiles that are to be installed
		return map[string]mobileconfig.Mobileconfig{
			p1: contents1,
			p2: contents2,
			p4: contents4,
		}, nil
	}

	ds.BulkDeleteMDMAppleHostsConfigProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAppleProfilePayload) error {
		require.Empty(t, payload)
		return nil
	}

	var enqueueFailForOp fleet.MDMOperationType
	mdmStorage.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.Command) (map[string]error, error) {
		require.NotNil(t, cmd)
		require.NotEmpty(t, cmd.CommandUUID)

		switch cmd.Command.RequestType {
		case "InstallProfile":
			// may be called for a single host or both
			if len(id) == 2 {
				require.ElementsMatch(t, []string{hostUUID, hostUUID2}, id)
			} else {
				require.Len(t, id, 1)
			}

			if !strings.Contains(string(cmd.Raw), contents1Base64) && !strings.Contains(string(cmd.Raw), contents2Base64) &&
				!strings.Contains(string(cmd.Raw), contents4Base64) {
				require.Failf(t, "profile contents don't match", "expected to contain %s, %s or %s but got %s",
					contents1Base64, contents2Base64, contents4Base64, string(cmd.Raw))
			}
		case "RemoveProfile":
			require.ElementsMatch(t, []string{hostUUID, hostUUID2}, id)
			require.Contains(t, string(cmd.Raw), "com.remove.profile")
		}
		switch {
		case enqueueFailForOp == fleet.MDMOperationTypeInstall && cmd.Command.RequestType == "InstallProfile":
			return nil, errors.New("enqueue error")
		case enqueueFailForOp == fleet.MDMOperationTypeRemove && cmd.Command.RequestType == "RemoveProfile":
			return nil, errors.New("enqueue error")
		}
		return nil, nil
	}

	mdmStorage.RetrievePushInfoFunc = func(ctx context.Context, tokens []string) (map[string]*mdm.Push, error) {
		res := make(map[string]*mdm.Push, len(tokens))
		for _, t := range tokens {
			res[t] = &mdm.Push{
				PushMagic: "",
				Token:     []byte(t),
				Topic:     "",
			}
		}
		return res, nil
	}
	mdmStorage.RetrievePushCertFunc = func(ctx context.Context, topic string) (*tls.Certificate, string, error) {
		cert, err := tls.LoadX509KeyPair("testdata/server.pem", "testdata/server.key")
		return &cert, "", err
	}
	mdmStorage.IsPushCertStaleFunc = func(ctx context.Context, topic string, staleToken string) (bool, error) {
		return false, nil
	}

	var failedCall bool
	var failedCheck func([]*fleet.MDMAppleBulkUpsertHostProfilePayload)
	ds.BulkUpsertMDMAppleHostProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) error {
		if failedCall {
			failedCheck(payload)
			return nil
		}

		// next call will be failed call, until reset
		failedCall = true

		// first time it is called, it is to set the status to pending and all
		// host profiles have a command uuid
		cmdUUIDByProfileUUIDInstall := make(map[string]string)
		cmdUUIDByProfileUUIDRemove := make(map[string]string)
		copies := make([]*fleet.MDMAppleBulkUpsertHostProfilePayload, len(payload))
		for i, p := range payload {
			if p.OperationType == fleet.MDMOperationTypeInstall {
				existing, ok := cmdUUIDByProfileUUIDInstall[p.ProfileUUID]
				if ok {
					require.Equal(t, existing, p.CommandUUID)
				} else {
					cmdUUIDByProfileUUIDInstall[p.ProfileUUID] = p.CommandUUID
				}
			} else {
				require.Equal(t, fleet.MDMOperationTypeRemove, p.OperationType)
				existing, ok := cmdUUIDByProfileUUIDRemove[p.ProfileUUID]
				if ok {
					require.Equal(t, existing, p.CommandUUID)
				} else {
					cmdUUIDByProfileUUIDRemove[p.ProfileUUID] = p.CommandUUID
				}
			}

			// clear the command UUID (in a copy so that it does not affect the
			// pointed-to struct) from the payload for the subsequent checks
			copyp := *p
			copyp.CommandUUID = ""
			copies[i] = &copyp
		}

		require.ElementsMatch(t, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
			{
				ProfileUUID:       p1,
				ProfileIdentifier: "com.add.profile",
				HostUUID:          hostUUID,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
			},
			{
				ProfileUUID:       p2,
				ProfileIdentifier: "com.add.profile.two",
				HostUUID:          hostUUID,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
			},
			{
				ProfileUUID:       p2,
				ProfileIdentifier: "com.add.profile.two",
				HostUUID:          hostUUID2,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
			},
			{
				ProfileUUID:       p3,
				ProfileIdentifier: "com.remove.profile",
				HostUUID:          hostUUID,
				OperationType:     fleet.MDMOperationTypeRemove,
				Status:            &fleet.MDMDeliveryPending,
			},
			{
				ProfileUUID:       p3,
				ProfileIdentifier: "com.remove.profile",
				HostUUID:          hostUUID2,
				OperationType:     fleet.MDMOperationTypeRemove,
				Status:            &fleet.MDMDeliveryPending,
			},
			{
				ProfileUUID:       p4,
				ProfileIdentifier: "com.add.profile.four",
				HostUUID:          hostUUID2,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
			},
		}, copies)
		return nil
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		appCfg := &fleet.AppConfig{}
		appCfg.ServerSettings.ServerURL = "https://test.example.com"
		appCfg.MDM.EnabledAndConfigured = true
		return appCfg, nil
	}

	ds.BulkUpsertMDMAppleConfigProfilesFunc = func(ctx context.Context, p []*fleet.MDMAppleConfigProfile) error {
		return nil
	}

	ds.AggregateEnrollSecretPerTeamFunc = func(ctx context.Context) ([]*fleet.EnrollSecret, error) {
		return []*fleet.EnrollSecret{}, nil
	}

	checkAndReset := func(t *testing.T, want bool, invoked *bool) {
		if want {
			require.True(t, *invoked)
		} else {
			require.False(t, *invoked)
		}
		*invoked = false
	}

	t.Run("success", func(t *testing.T) {
		var failedCount int
		failedCall = false
		failedCheck = func(payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) {
			failedCount++
			require.Len(t, payload, 0)
		}
		err := ReconcileAppleProfiles(ctx, ds, cmdr, kitlog.NewNopLogger())
		require.NoError(t, err)
		require.Equal(t, 1, failedCount)
		checkAndReset(t, true, &ds.ListMDMAppleProfilesToInstallFuncInvoked)
		checkAndReset(t, true, &ds.ListMDMAppleProfilesToRemoveFuncInvoked)
		checkAndReset(t, true, &ds.GetMDMAppleProfilesContentsFuncInvoked)
		checkAndReset(t, true, &ds.BulkUpsertMDMAppleHostProfilesFuncInvoked)
	})

	t.Run("fail enqueue remove ops", func(t *testing.T) {
		var failedCount int
		failedCall = false
		failedCheck = func(payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) {
			failedCount++
			require.Len(t, payload, 2) // the 2 remove ops
			require.ElementsMatch(t, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
				{
					ProfileUUID:       p3,
					ProfileIdentifier: "com.remove.profile",
					HostUUID:          hostUUID,
					OperationType:     fleet.MDMOperationTypeRemove,
					Status:            nil,
					CommandUUID:       "",
				},
				{
					ProfileUUID:       p3,
					ProfileIdentifier: "com.remove.profile",
					HostUUID:          hostUUID2,
					OperationType:     fleet.MDMOperationTypeRemove,
					Status:            nil,
					CommandUUID:       "",
				},
			}, payload)
		}

		enqueueFailForOp = fleet.MDMOperationTypeRemove
		err := ReconcileAppleProfiles(ctx, ds, cmdr, kitlog.NewNopLogger())
		require.NoError(t, err)
		require.Equal(t, 1, failedCount)
		checkAndReset(t, true, &ds.ListMDMAppleProfilesToInstallFuncInvoked)
		checkAndReset(t, true, &ds.ListMDMAppleProfilesToRemoveFuncInvoked)
		checkAndReset(t, true, &ds.GetMDMAppleProfilesContentsFuncInvoked)
		checkAndReset(t, true, &ds.BulkUpsertMDMAppleHostProfilesFuncInvoked)
	})

	t.Run("fail enqueue install ops", func(t *testing.T) {
		var failedCount int
		failedCall = false
		failedCheck = func(payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) {
			failedCount++

			require.Len(t, payload, 4) // the 4 install ops
			require.ElementsMatch(t, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
				{
					ProfileUUID:       p1,
					ProfileIdentifier: "com.add.profile",
					HostUUID:          hostUUID,
					OperationType:     fleet.MDMOperationTypeInstall,
					Status:            nil,
					CommandUUID:       "",
				},
				{
					ProfileUUID:       p2,
					ProfileIdentifier: "com.add.profile.two",
					HostUUID:          hostUUID,
					OperationType:     fleet.MDMOperationTypeInstall,
					Status:            nil,
					CommandUUID:       "",
				},
				{
					ProfileUUID:       p2,
					ProfileIdentifier: "com.add.profile.two",
					HostUUID:          hostUUID2,
					OperationType:     fleet.MDMOperationTypeInstall,
					Status:            nil,
					CommandUUID:       "",
				},
				{
					ProfileUUID:       p4,
					ProfileIdentifier: "com.add.profile.four",
					HostUUID:          hostUUID2,
					OperationType:     fleet.MDMOperationTypeInstall,
					Status:            nil,
					CommandUUID:       "",
				},
			}, payload)
		}

		enqueueFailForOp = fleet.MDMOperationTypeInstall
		err := ReconcileAppleProfiles(ctx, ds, cmdr, kitlog.NewNopLogger())
		require.NoError(t, err)
		require.Equal(t, 1, failedCount)
		checkAndReset(t, true, &ds.ListMDMAppleProfilesToInstallFuncInvoked)
		checkAndReset(t, true, &ds.ListMDMAppleProfilesToRemoveFuncInvoked)
		checkAndReset(t, true, &ds.GetMDMAppleProfilesContentsFuncInvoked)
		checkAndReset(t, true, &ds.BulkUpsertMDMAppleHostProfilesFuncInvoked)
	})
}

func TestAppleMDMFileVaultEscrowFunctions(t *testing.T) {
	svc := Service{}

	err := svc.MDMAppleEnableFileVaultAndEscrow(context.Background(), ptr.Uint(1))
	require.ErrorIs(t, fleet.ErrMissingLicense, err)

	err = svc.MDMAppleDisableFileVaultAndEscrow(context.Background(), ptr.Uint(1))
	require.ErrorIs(t, fleet.ErrMissingLicense, err)
}

func TestGenerateEnrollmentProfileMobileConfig(t *testing.T) {
	// SCEP challenge should be escaped for XML
	b, err := apple_mdm.GenerateEnrollmentProfileMobileconfig("foo", "https://example.com", "foo&bar", "topic")
	require.NoError(t, err)
	require.Contains(t, string(b), "foo&amp;bar")
}

func TestEnsureFleetdConfig(t *testing.T) {
	testError := errors.New("test error")
	testURL := "https://example.com"
	testTeamName := "test-team"
	logger := kitlog.NewNopLogger()

	t.Run("no enroll secret found", func(t *testing.T) {
		ctx := context.Background()
		ds := new(mock.Store)

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{}, nil
		}
		ds.AggregateEnrollSecretPerTeamFunc = func(ctx context.Context) ([]*fleet.EnrollSecret, error) {
			return []*fleet.EnrollSecret{}, nil
		}
		ds.BulkUpsertMDMAppleConfigProfilesFunc = func(ctx context.Context, ps []*fleet.MDMAppleConfigProfile) error {
			require.Empty(t, ps)
			return nil
		}
		err := ensureFleetdConfig(ctx, ds, logger)
		require.NoError(t, err)
		require.True(t, ds.BulkUpsertMDMAppleConfigProfilesFuncInvoked)
		require.True(t, ds.AggregateEnrollSecretPerTeamFuncInvoked)
		require.True(t, ds.AppConfigFuncInvoked)
	})

	t.Run("all enroll secrets empty", func(t *testing.T) {
		ctx := context.Background()
		ds := new(mock.Store)
		secrets := []*fleet.EnrollSecret{
			{Secret: "", TeamID: nil},
			{Secret: "", TeamID: ptr.Uint(1)},
			{Secret: "", TeamID: ptr.Uint(2)},
		}
		ds.AggregateEnrollSecretPerTeamFunc = func(ctx context.Context) ([]*fleet.EnrollSecret, error) {
			return secrets, nil
		}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{}, nil
		}
		ds.BulkUpsertMDMAppleConfigProfilesFunc = func(ctx context.Context, ps []*fleet.MDMAppleConfigProfile) error {
			require.Empty(t, ps)
			return nil
		}
		err := ensureFleetdConfig(ctx, ds, logger)
		require.NoError(t, err)
		require.True(t, ds.BulkUpsertMDMAppleConfigProfilesFuncInvoked)
		require.True(t, ds.AggregateEnrollSecretPerTeamFuncInvoked)
		require.True(t, ds.AppConfigFuncInvoked)
	})

	t.Run("uses the enroll secret of each team if available", func(t *testing.T) {
		ctx := context.Background()
		ds := new(mock.Store)
		secrets := []*fleet.EnrollSecret{
			{Secret: "global", TeamID: nil},
			{Secret: "team-1", TeamID: ptr.Uint(1)},
			{Secret: "team-2", TeamID: ptr.Uint(2)},
		}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			appCfg := &fleet.AppConfig{}
			appCfg.ServerSettings.ServerURL = testURL
			appCfg.MDM.AppleBMDefaultTeam = testTeamName
			return appCfg, nil
		}
		ds.AggregateEnrollSecretPerTeamFunc = func(ctx context.Context) ([]*fleet.EnrollSecret, error) {
			return secrets, nil
		}
		ds.BulkUpsertMDMAppleConfigProfilesFunc = func(ctx context.Context, ps []*fleet.MDMAppleConfigProfile) error {
			require.Len(t, ps, len(secrets))
			for i, p := range ps {
				require.Contains(t, string(p.Mobileconfig), testURL)
				require.Contains(t, string(p.Mobileconfig), secrets[i].Secret)
				require.Equal(t, mobileconfig.FleetdConfigPayloadIdentifier, p.Identifier)
			}
			return nil
		}

		err := ensureFleetdConfig(ctx, ds, logger)
		require.NoError(t, err)
		require.True(t, ds.AggregateEnrollSecretPerTeamFuncInvoked)
		require.True(t, ds.BulkUpsertMDMAppleConfigProfilesFuncInvoked)
	})

	t.Run("if the team doesn't have an enroll secret, fallback to no team", func(t *testing.T) {
		ctx := context.Background()
		ds := new(mock.Store)
		secrets := []*fleet.EnrollSecret{
			{Secret: "global", TeamID: nil},
			{Secret: "", TeamID: ptr.Uint(1)},
		}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			appCfg := &fleet.AppConfig{}
			appCfg.ServerSettings.ServerURL = testURL
			appCfg.MDM.AppleBMDefaultTeam = testTeamName
			return appCfg, nil
		}
		ds.AggregateEnrollSecretPerTeamFunc = func(ctx context.Context) ([]*fleet.EnrollSecret, error) {
			return secrets, nil
		}
		ds.BulkUpsertMDMAppleConfigProfilesFunc = func(ctx context.Context, ps []*fleet.MDMAppleConfigProfile) error {
			require.Len(t, ps, len(secrets))
			for i, p := range ps {
				require.Contains(t, string(p.Mobileconfig), testURL)
				require.Contains(t, string(p.Mobileconfig), secrets[i].Secret)
				require.Equal(t, mobileconfig.FleetdConfigPayloadIdentifier, p.Identifier)
			}
			return nil
		}
		err := ensureFleetdConfig(ctx, ds, logger)
		require.NoError(t, err)
		require.True(t, ds.AppConfigFuncInvoked)
		require.True(t, ds.AggregateEnrollSecretPerTeamFuncInvoked)
		require.True(t, ds.BulkUpsertMDMAppleConfigProfilesFuncInvoked)
	})

	t.Run("returns an error if there's a problem retrieving AppConfig", func(t *testing.T) {
		ctx := context.Background()
		ds := new(mock.Store)
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return nil, testError
		}
		err := ensureFleetdConfig(ctx, ds, logger)
		require.ErrorIs(t, err, testError)
	})

	t.Run("returns an error if there's a problem retrieving secrets", func(t *testing.T) {
		ctx := context.Background()
		ds := new(mock.Store)
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{}, nil
		}
		ds.AggregateEnrollSecretPerTeamFunc = func(ctx context.Context) ([]*fleet.EnrollSecret, error) {
			return nil, testError
		}
		err := ensureFleetdConfig(ctx, ds, logger)
		require.ErrorIs(t, err, testError)
	})

	t.Run("returns an error if there's a problem upserting profiles", func(t *testing.T) {
		ctx := context.Background()
		ds := new(mock.Store)
		secrets := []*fleet.EnrollSecret{
			{Secret: "global", TeamID: nil},
			{Secret: "team-1", TeamID: ptr.Uint(1)},
		}
		ds.AggregateEnrollSecretPerTeamFunc = func(ctx context.Context) ([]*fleet.EnrollSecret, error) {
			return secrets, nil
		}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{}, nil
		}
		ds.BulkUpsertMDMAppleConfigProfilesFunc = func(ctx context.Context, p []*fleet.MDMAppleConfigProfile) error {
			return testError
		}
		err := ensureFleetdConfig(ctx, ds, logger)
		require.ErrorIs(t, err, testError)
		require.True(t, ds.AppConfigFuncInvoked)
		require.True(t, ds.AggregateEnrollSecretPerTeamFuncInvoked)
		require.True(t, ds.BulkUpsertMDMAppleConfigProfilesFuncInvoked)
	})
}

func TestMDMAppleSetupAssistant(t *testing.T) {
	svc, ctx, ds := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})

	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}
	ds.NewJobFunc = func(ctx context.Context, j *fleet.Job) (*fleet.Job, error) {
		return j, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.GetMDMAppleSetupAssistantFunc = func(ctx context.Context, teamID *uint) (*fleet.MDMAppleSetupAssistant, error) {
		return &fleet.MDMAppleSetupAssistant{}, nil
	}
	ds.SetOrUpdateMDMAppleSetupAssistantFunc = func(ctx context.Context, asst *fleet.MDMAppleSetupAssistant) (*fleet.MDMAppleSetupAssistant, error) {
		return asst, nil
	}
	ds.DeleteMDMAppleSetupAssistantFunc = func(ctx context.Context, teamID *uint) error {
		return nil
	}
	ds.TeamFunc = func(ctx context.Context, id uint) (*fleet.Team, error) {
		return &fleet.Team{ID: id}, nil
	}

	testCases := []struct {
		name            string
		user            *fleet.User
		teamID          *uint
		shouldFailRead  bool
		shouldFailWrite bool
	}{
		{"no role no team", test.UserNoRoles, nil, true, true},
		{"no role team", test.UserNoRoles, ptr.Uint(1), true, true},
		{"global admin no team", test.UserAdmin, nil, false, false},
		{"global admin team", test.UserAdmin, ptr.Uint(1), false, false},
		{"global maintainer no team", test.UserMaintainer, nil, false, false},
		{"global maintainer team", test.UserMaintainer, ptr.Uint(1), false, false},
		{"global observer no team", test.UserObserver, nil, true, true},
		{"global observer team", test.UserObserver, ptr.Uint(1), true, true},
		{"global observer+ no team", test.UserObserverPlus, nil, true, true},
		{"global observer+ team", test.UserObserverPlus, ptr.Uint(1), true, true},
		{"global gitops no team", test.UserGitOps, nil, true, false},
		{"global gitops team", test.UserGitOps, ptr.Uint(1), true, false},
		{"team admin no team", test.UserTeamAdminTeam1, nil, true, true},
		{"team admin team", test.UserTeamAdminTeam1, ptr.Uint(1), false, false},
		{"team admin other team", test.UserTeamAdminTeam2, ptr.Uint(1), true, true},
		{"team maintainer no team", test.UserTeamMaintainerTeam1, nil, true, true},
		{"team maintainer team", test.UserTeamMaintainerTeam1, ptr.Uint(1), false, false},
		{"team maintainer other team", test.UserTeamMaintainerTeam2, ptr.Uint(1), true, true},
		{"team observer no team", test.UserTeamObserverTeam1, nil, true, true},
		{"team observer team", test.UserTeamObserverTeam1, ptr.Uint(1), true, true},
		{"team observer other team", test.UserTeamObserverTeam2, ptr.Uint(1), true, true},
		{"team observer+ no team", test.UserTeamObserverPlusTeam1, nil, true, true},
		{"team observer+ team", test.UserTeamObserverPlusTeam1, ptr.Uint(1), true, true},
		{"team observer+ other team", test.UserTeamObserverPlusTeam2, ptr.Uint(1), true, true},
		{"team gitops no team", test.UserTeamGitOpsTeam1, nil, true, true},
		{"team gitops team", test.UserTeamGitOpsTeam1, ptr.Uint(1), true, false},
		{"team gitops other team", test.UserTeamGitOpsTeam2, ptr.Uint(1), true, true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// prepare the context with the user and license
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			_, err := svc.GetMDMAppleSetupAssistant(ctx, tt.teamID)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{
				Name:    "test",
				Profile: json.RawMessage("{}"),
				TeamID:  tt.teamID,
			})
			checkAuthErr(t, tt.shouldFailWrite, err)

			err = svc.DeleteMDMAppleSetupAssistant(ctx, tt.teamID)
			checkAuthErr(t, tt.shouldFailWrite, err)
		})
	}
}

func TestMDMApplePreassignEndpoints(t *testing.T) {
	svc, ctx, _ := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})

	checkAuthErr := func(t *testing.T, err error, shouldFailWithAuth bool) {
		t.Helper()

		if shouldFailWithAuth {
			require.Error(t, err)
			require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
		} else {
			require.NoError(t, err)
		}
	}

	testCases := []struct {
		name       string
		user       *fleet.User
		shouldFail bool
	}{
		{"no role", test.UserNoRoles, true},
		{"global admin", test.UserAdmin, false},
		{"global maintainer", test.UserMaintainer, true},
		{"global observer", test.UserObserver, true},
		{"global observer+", test.UserObserverPlus, true},
		{"global gitops", test.UserGitOps, false},
		{"team admin", test.UserTeamAdminTeam1, true},
		{"team maintainer", test.UserTeamMaintainerTeam1, true},
		{"team observer", test.UserTeamObserverTeam1, true},
		{"team observer+", test.UserTeamObserverPlusTeam1, true},
		{"team gitops", test.UserTeamGitOpsTeam1, true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// prepare the context with the user
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			err := svc.MDMApplePreassignProfile(ctx, fleet.MDMApplePreassignProfilePayload{
				ExternalHostIdentifier: "test",
				HostUUID:               "test",
				Profile:                mobileconfigForTest("N1", "I1"),
			})
			checkAuthErr(t, err, tt.shouldFail)

			err = svc.MDMAppleMatchPreassignment(ctx, "test")
			checkAuthErr(t, err, tt.shouldFail)
		})
	}
}

func mobileconfigForTest(name, identifier string) []byte {
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array/>
	<key>PayloadDisplayName</key>
	<string>%s</string>
	<key>PayloadIdentifier</key>
	<string>%s</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>%s</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>
`, name, identifier, uuid.New().String()))
}

func mobileconfigForTestWithContent(outerName, outerIdentifier, innerIdentifier, innerType, innerName string) []byte {
	if innerName == "" {
		innerName = outerName + ".inner"
	}

	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
          <dict>
            <key>PayloadDisplayName</key>
            <string>%s</string>
            <key>PayloadIdentifier</key>
            <string>%s</string>
            <key>PayloadType</key>
            <string>%s</string>
            <key>PayloadUUID</key>
            <string>3548D750-6357-4910-8DEA-D80ADCE2C787</string>
            <key>PayloadVersion</key>
            <integer>1</integer>
            <key>ShowRecoveryKey</key>
            <false/>
          </dict>
	</array>
	<key>PayloadDisplayName</key>
	<string>%s</string>
	<key>PayloadIdentifier</key>
	<string>%s</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>%s</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>
`, innerName, innerIdentifier, innerType, outerName, outerIdentifier, uuid.New().String()))
}

func generateCertWithAPNsTopic() ([]byte, []byte, error) {
	// generate a new private key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	// set up the OID for UID
	oidUID := asn1.ObjectIdentifier{0, 9, 2342, 19200300, 100, 1, 1}

	// set up a certificate template with the required UID in the Subject
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			ExtraNames: []pkix.AttributeTypeAndValue{
				{
					Type:  oidUID,
					Value: "com.apple.mgmt.Example",
				},
			},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// create a self-signed certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, err
	}

	// encode to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return certPEM, keyPEM, nil
}

func setupTest(t *testing.T) (context.Context, kitlog.Logger, *mock.Store, *config.FleetConfig, *mock.MDMAppleStore, *apple_mdm.MDMAppleCommander) {
	ctx := context.Background()
	logger := kitlog.NewNopLogger()
	cfg := config.TestConfig()
	testCertPEM, testKeyPEM, err := generateCertWithAPNsTopic()
	require.NoError(t, err)
	config.SetTestMDMConfig(t, &cfg, testCertPEM, testKeyPEM, testBMToken, "../../server/service/testdata")
	ds := new(mock.Store)
	mdmStorage := &mock.MDMAppleStore{}
	pushFactory, _ := newMockAPNSPushProviderFactory()
	pusher := nanomdm_pushsvc.New(
		mdmStorage,
		mdmStorage,
		pushFactory,
		stdlogfmt.New(),
	)
	commander := apple_mdm.NewMDMAppleCommander(mdmStorage, pusher)

	return ctx, logger, ds, &cfg, mdmStorage, commander
}

func TestRenewSCEPCertificatesMDMConfigNotSet(t *testing.T) {
	ctx, logger, ds, cfg, _, commander := setupTest(t)
	cfg.MDM = config.MDMConfig{} // ensure MDM is not fully configured
	err := RenewSCEPCertificates(ctx, logger, ds, cfg, commander)
	require.NoError(t, err)
}

func TestRenewSCEPCertificatesCommanderNil(t *testing.T) {
	ctx, logger, ds, cfg, _, _ := setupTest(t)
	err := RenewSCEPCertificates(ctx, logger, ds, cfg, nil)
	require.NoError(t, err)
}

func TestRenewSCEPCertificatesBranches(t *testing.T) {
	tests := []struct {
		name               string
		customExpectations func(*testing.T, *mock.Store, *config.FleetConfig, *mock.MDMAppleStore, *apple_mdm.MDMAppleCommander)
		expectedError      bool
	}{
		{
			name: "No Certs to Renew",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				ds.GetHostCertAssociationsToExpireFunc = func(ctx context.Context, expiryDays int, limit int) ([]fleet.SCEPIdentityAssociation, error) {
					return nil, nil
				}
			},
			expectedError: false,
		},
		{
			name: "GetHostCertAssociationsToExpire Errors",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				ds.GetHostCertAssociationsToExpireFunc = func(ctx context.Context, expiryDays int, limit int) ([]fleet.SCEPIdentityAssociation, error) {
					return nil, errors.New("database error")
				}
			},
			expectedError: true,
		},
		{
			name: "AppConfig Errors",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
					return nil, errors.New("app config error")
				}
			},
			expectedError: true,
		},
		{
			name: "InstallProfile for hostsWithoutRefs",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				var wantCommandUUID string
				ds.GetHostCertAssociationsToExpireFunc = func(ctx context.Context, expiryDays int, limit int) ([]fleet.SCEPIdentityAssociation, error) {
					return []fleet.SCEPIdentityAssociation{{HostUUID: "hostUUID1", EnrollReference: ""}}, nil
				}

				appleStore.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.Command) (map[string]error, error) {
					require.Equal(t, "InstallProfile", cmd.Command.RequestType)
					wantCommandUUID = cmd.CommandUUID
					return map[string]error{}, nil
				}
				ds.SetCommandForPendingSCEPRenewalFunc = func(ctx context.Context, assocs []fleet.SCEPIdentityAssociation, cmdUUID string) error {
					require.Len(t, assocs, 1)
					require.Equal(t, "hostUUID1", assocs[0].HostUUID)
					require.Equal(t, cmdUUID, wantCommandUUID)
					return nil
				}

				t.Cleanup(func() {
					require.True(t, appleStore.EnqueueCommandFuncInvoked)
					require.True(t, ds.SetCommandForPendingSCEPRenewalFuncInvoked)
				})
			},
			expectedError: false,
		},
		{
			name: "InstallProfile for hostsWithoutRefs fails",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				ds.GetHostCertAssociationsToExpireFunc = func(ctx context.Context, expiryDays int, limit int) ([]fleet.SCEPIdentityAssociation, error) {
					return []fleet.SCEPIdentityAssociation{{HostUUID: "hostUUID1", EnrollReference: ""}}, nil
				}

				appleStore.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.Command) (map[string]error, error) {
					return map[string]error{}, errors.New("foo")
				}
			},
			expectedError: true,
		},
		{
			name: "InstallProfile for hostsWithRefs",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				var wantCommandUUID string
				ds.GetHostCertAssociationsToExpireFunc = func(ctx context.Context, expiryDays int, limit int) ([]fleet.SCEPIdentityAssociation, error) {
					return []fleet.SCEPIdentityAssociation{{HostUUID: "hostUUID2", EnrollReference: "ref1"}}, nil
				}
				appleStore.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.Command) (map[string]error, error) {
					require.Equal(t, "InstallProfile", cmd.Command.RequestType)
					wantCommandUUID = cmd.CommandUUID
					return map[string]error{}, nil
				}
				ds.SetCommandForPendingSCEPRenewalFunc = func(ctx context.Context, assocs []fleet.SCEPIdentityAssociation, cmdUUID string) error {
					require.Len(t, assocs, 1)
					require.Equal(t, "hostUUID2", assocs[0].HostUUID)
					require.Equal(t, cmdUUID, wantCommandUUID)
					return nil
				}
				t.Cleanup(func() {
					require.True(t, appleStore.EnqueueCommandFuncInvoked)
					require.True(t, ds.SetCommandForPendingSCEPRenewalFuncInvoked)
				})
			},
			expectedError: false,
		},
		{
			name: "InstallProfile for hostsWithRefs fails",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				ds.GetHostCertAssociationsToExpireFunc = func(ctx context.Context, expiryDays int, limit int) ([]fleet.SCEPIdentityAssociation, error) {
					return []fleet.SCEPIdentityAssociation{{HostUUID: "hostUUID1", EnrollReference: "ref1"}}, nil
				}

				appleStore.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.Command) (map[string]error, error) {
					return map[string]error{}, errors.New("foo")
				}
			},
			expectedError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, logger, ds, cfg, appleStorage, commander := setupTest(t)

			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				appCfg := &fleet.AppConfig{}
				appCfg.OrgInfo.OrgName = "fl33t"
				appCfg.ServerSettings.ServerURL = "https://foo.example.com"
				return appCfg, nil
			}

			ds.GetHostCertAssociationsToExpireFunc = func(ctx context.Context, expiryDays int, limit int) ([]fleet.SCEPIdentityAssociation, error) {
				return []fleet.SCEPIdentityAssociation{}, nil
			}

			ds.SetCommandForPendingSCEPRenewalFunc = func(ctx context.Context, assocs []fleet.SCEPIdentityAssociation, cmdUUID string) error {
				return nil
			}

			appleStorage.RetrievePushInfoFunc = func(ctx context.Context, targets []string) (map[string]*mdm.Push, error) {
				pushes := make(map[string]*mdm.Push, len(targets))
				for _, uuid := range targets {
					pushes[uuid] = &mdm.Push{
						PushMagic: "magic" + uuid,
						Token:     []byte("token" + uuid),
						Topic:     "topic" + uuid,
					}
				}

				return pushes, nil
			}

			appleStorage.RetrievePushCertFunc = func(ctx context.Context, topic string) (*tls.Certificate, string, error) {
				cert, err := tls.LoadX509KeyPair("./testdata/server.pem", "./testdata/server.key")
				return &cert, "", err
			}

			tc.customExpectations(t, ds, cfg, appleStorage, commander)

			err := RenewSCEPCertificates(ctx, logger, ds, cfg, commander)
			if tc.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
