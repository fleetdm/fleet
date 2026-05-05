package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	_ "embed"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	fleetmdm "github.com/fleetdm/fleet/v4/server/mdm"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	mdmlifecycle "github.com/fleetdm/fleet/v4/server/mdm/lifecycle"
	nanodep_client "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/tokenpki"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	nanomdm_pushsvc "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/service"
	"github.com/fleetdm/fleet/v4/server/mock"
	mdmmock "github.com/fleetdm/fleet/v4/server/mock/mdm"
	nanodep_mock "github.com/fleetdm/fleet/v4/server/mock/nanodep"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/redis_key_value"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"github.com/micromdm/nanolib/log/stdlogfmt"
	"github.com/micromdm/plist"
	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mdmtesting "github.com/fleetdm/fleet/v4/server/mdm/testing_utils"
)

type nopProfileMatcher struct{}

func (nopProfileMatcher) PreassignProfile(ctx context.Context, pld fleet.MDMApplePreassignProfilePayload) error {
	return nil
}

func (nopProfileMatcher) RetrieveProfiles(ctx context.Context, extHostID string) (fleet.MDMApplePreassignHostProfiles, error) {
	return fleet.MDMApplePreassignHostProfiles{}, nil
}

func setupAppleMDMService(t *testing.T, license *fleet.LicenseInfo) (fleet.Service, context.Context, *mock.Store, *TestServerOpts) {
	ds := new(mock.Store)
	cfg := config.TestConfig()
	testCertPEM, testKeyPEM, err := generateCertWithAPNsTopic()
	require.NoError(t, err)
	config.SetTestMDMConfig(t, &cfg, testCertPEM, testKeyPEM, "../../server/service/testdata")
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
		case strings.Contains(r.URL.Path, "/profile"):
			_, err := w.Write([]byte(`{"profile_uuid": "profile123"}`))
			require.NoError(t, err)
		}
	}))

	mdmStorage := &mdmmock.MDMAppleStore{}
	depStorage := &nanodep_mock.Storage{}
	pushFactory, _ := newMockAPNSPushProviderFactory()
	pusher := nanomdm_pushsvc.New(
		mdmStorage,
		mdmStorage,
		pushFactory,
		NewNanoMDMLogger(slog.New(slog.NewJSONHandler(os.Stdout, nil))),
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

	mdmStorage.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error, error) {
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
	ds.GetNanoMDMEnrollmentDetailsFunc = func(ctx context.Context, hostUUID string) (*fleet.NanoMDMEnrollmentDetails, error) {
		return &fleet.NanoMDMEnrollmentDetails{
			LastMDMEnrollmentTime: nil,
			LastMDMSeenTime:       nil,
			HardwareAttested:      false,
		}, nil
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
	ds.ValidateEmbeddedSecretsFunc = func(ctx context.Context, documents []string) error {
		return nil
	}
	ds.ExpandEmbeddedSecretsFunc = func(ctx context.Context, document string) (string, error) {
		return document, nil
	}
	ds.ExpandEmbeddedSecretsAndUpdatedAtFunc = func(ctx context.Context, document string) (string, *time.Time, error) {
		return document, nil, nil
	}
	apnsCert, apnsKey, err := mysql.GenerateTestCertBytes(mdmtesting.NewTestMDMAppleCertTemplate())
	require.NoError(t, err)
	crt, key, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(t, err)
	certPEM := tokenpki.PEMCertificate(crt.Raw)
	keyPEM := tokenpki.PEMRSAPrivateKey(key)
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext,
	) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetAPNSCert: {Value: apnsCert},
			fleet.MDMAssetAPNSKey:  {Value: apnsKey},
			fleet.MDMAssetCACert:   {Value: certPEM},
			fleet.MDMAssetCAKey:    {Value: keyPEM},
		}, nil
	}

	ds.GetABMTokenOrgNamesAssociatedWithTeamFunc = func(ctx context.Context, teamID *uint) ([]string, error) {
		return []string{"foobar"}, nil
	}
	ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) {
		return []*fleet.ABMToken{{ID: 1}}, nil
	}

	return svc, ctx, ds, opts
}

func TestAppleMDMAuthorization(t *testing.T) {
	svc, ctx, ds, _ := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})

	ds.GetEnrollSecretsFunc = func(ctx context.Context, teamID *uint) ([]*fleet.EnrollSecret, error) {
		return []*fleet.EnrollSecret{
			{
				Secret: "abcd",
				TeamID: nil,
			},
			{
				Secret: "efgh",
				TeamID: nil,
			},
		}, nil
	}

	ds.VerifyEnrollSecretFunc = func(ctx context.Context, enrollSecret string) (*fleet.EnrollSecret, error) {
		return &fleet.EnrollSecret{
			Secret: "abcd",
			TeamID: nil,
		}, nil
	}

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
	}

	// some eula methods read and write access for gitops users. We test them separately
	// from the other MDM methods.
	testEULAMethods := func(t *testing.T, user *fleet.User, shouldFailWithAuth bool) {
		ctx := test.UserContext(ctx, user)
		_, err := svc.MDMGetEULAMetadata(ctx)
		checkAuthErr(t, err, shouldFailWithAuth)
		err = svc.MDMCreateEULA(ctx, "eula.pdf", bytes.NewReader([]byte("%PDF-")), false)
		checkAuthErr(t, err, shouldFailWithAuth)
		err = svc.MDMDeleteEULA(ctx, "foo", false)
		checkAuthErr(t, err, shouldFailWithAuth)
	}

	// Only global admins can access the endpoints.
	testAuthdMethods(t, test.UserAdmin, false)

	// Global admin and gitops users can access the eula endpoints.
	testEULAMethods(t, test.UserAdmin, false)
	testEULAMethods(t, test.UserGitOps, false)

	// All other users should not have access to the endpoints.
	for _, user := range []*fleet.User{
		test.UserNoRoles,
		test.UserMaintainer,
		test.UserObserver,
		test.UserObserverPlus,
		test.UserTeamAdminTeam1,
	} {
		testAuthdMethods(t, user, true)
		testEULAMethods(t, user, true)
	}
	// Token authenticated endpoints can be accessed by anyone.
	ctx = test.UserContext(ctx, test.UserNoRoles)
	_, err := svc.GetMDMAppleInstallerByToken(ctx, "foo")
	require.NoError(t, err)
	_, err = svc.GetMDMAppleEnrollmentProfileByToken(ctx, "foo", "", &fleet.MDMAppleMachineInfo{})
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
	hostCtx := test.HostContext(context.Background(), &fleet.Host{})
	_, err = svc.GetDeviceMDMAppleEnrollmentProfile(hostCtx)
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

		rawB64PremiumCmd = base64.RawStdEncoding.EncodeToString(fmt.Appendf([]byte{}, `<?xml version="1.0" encoding="UTF-8"?>
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
</plist>`, "ClearPasscode"))
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

	ds.GetMDMAppleCommandResultsFunc = func(ctx context.Context, commandUUID string, hostUUID string) ([]*fleet.MDMCommandResult, error) {
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
				_, err = svc.GetMDMCommandResults(ctx, c.cmdUUID, "")
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
	svc, ctx, ds, _ := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})

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

	ds.NewMDMAppleConfigProfileFunc = func(ctx context.Context, cp fleet.MDMAppleConfigProfile, usesVars []fleet.FleetVarName) (*fleet.MDMAppleConfigProfile, error) {
		return &cp, nil
	}
	ds.ListMDMAppleConfigProfilesFunc = func(ctx context.Context, teamID *uint) ([]*fleet.MDMAppleConfigProfile, error) {
		return nil, nil
	}
	ds.GetMDMAppleProfilesSummaryFunc = func(context.Context, *uint) (*fleet.MDMProfilesSummary, error) {
		return nil, nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hids, tids []uint, puuids, uuids []string,
	) (updates fleet.MDMProfilesUpdates, err error) {
		return fleet.MDMProfilesUpdates{}, nil
	}
	ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
		return &fleet.GroupedCertificateAuthorities{}, nil
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
	mockTeamFuncWithUser := func(u *fleet.User) mock.TeamWithExtrasFunc {
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
	mockTeamLiteFunc := func(u *fleet.User) mock.TeamLiteFunc {
		return func(ctx context.Context, teamID uint) (*fleet.TeamLite, error) {
			if len(u.Teams) > 0 {
				for _, t := range u.Teams {
					if t.ID == teamID {
						return &fleet.TeamLite{ID: teamID}, nil
					}
				}
			}
			return &fleet.TeamLite{}, nil
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
		ds.TeamWithExtrasFunc = mockTeamFuncWithUser(tt.user)
		ds.TeamLiteFunc = mockTeamLiteFunc(tt.user)

		t.Run(tt.name, func(t *testing.T) {
			// test authz create new profile (no team)
			_, err := svc.NewMDMAppleConfigProfile(ctx, 0, mcBytes, nil, fleet.LabelsIncludeAll)
			checkShouldFail(err, tt.shouldFailGlobal)

			// test authz create new profile (team 1)
			_, err = svc.NewMDMAppleConfigProfile(ctx, 1, mcBytes, nil, fleet.LabelsIncludeAll)
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
	svc, ctx, ds, _ := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	identifier := "Bar.$FLEET_VAR_HOST_END_USER_EMAIL_IDP"
	mcBytes := mcBytesForTest("Foo", identifier, "UUID")

	ds.NewMDMAppleConfigProfileFunc = func(ctx context.Context, cp fleet.MDMAppleConfigProfile, usesVars []fleet.FleetVarName) (*fleet.MDMAppleConfigProfile, error) {
		require.Equal(t, "Foo", cp.Name)
		assert.Equal(t, identifier, cp.Identifier)
		require.Equal(t, mcBytes, []byte(cp.Mobileconfig))
		return &cp, nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hids, tids []uint, puuids, uuids []string,
	) (updates fleet.MDMProfilesUpdates, err error) {
		return fleet.MDMProfilesUpdates{}, nil
	}
	ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
		return &fleet.GroupedCertificateAuthorities{}, nil
	}

	cp, err := svc.NewMDMAppleConfigProfile(ctx, 0, mcBytes, nil, fleet.LabelsIncludeAll)
	require.NoError(t, err)
	require.Equal(t, "Foo", cp.Name)
	assert.Equal(t, identifier, cp.Identifier)
	require.Equal(t, mcBytes, []byte(cp.Mobileconfig))

	// Unsupported Fleet variable
	mcBytes = mcBytesForTest("Foo", identifier, "UUID${FLEET_VAR_BOZO}")
	_, err = svc.NewMDMAppleConfigProfile(ctx, 0, mcBytes, nil, fleet.LabelsIncludeAll)
	assert.ErrorContains(t, err, "Fleet variable")

	// Test profile with FLEET_SECRET in PayloadDisplayName
	mcBytes = mcBytesForTest("Profile $FLEET_SECRET_PASSWORD", "test.identifier", "UUID")
	_, err = svc.NewMDMAppleConfigProfile(ctx, 0, mcBytes, nil, fleet.LabelsIncludeAll)
	assert.ErrorContains(t, err, "PayloadDisplayName cannot contain FLEET_SECRET variables")
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

func TestBatchSetMDMAppleProfilesWithSecrets(t *testing.T) {
	svc, ctx, _, _ := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	// Test profile with FLEET_SECRET in PayloadDisplayName
	profileWithSecret := mcBytesForTest("Profile $FLEET_SECRET_PASSWORD", "test.identifier", "UUID")
	err := svc.BatchSetMDMAppleProfiles(ctx, nil, nil, [][]byte{profileWithSecret}, false, false)
	assert.ErrorContains(t, err, "PayloadDisplayName cannot contain FLEET_SECRET variables")

	// Test multiple profiles where one has a secret in PayloadDisplayName
	goodProfile := mcBytesForTest("Good Profile", "good.identifier", "UUID1")
	badProfile := mcBytesForTest("Bad $FLEET_SECRET_KEY Profile", "bad.identifier", "UUID2")
	err = svc.BatchSetMDMAppleProfiles(ctx, nil, nil, [][]byte{goodProfile, badProfile}, false, false)
	assert.ErrorContains(t, err, "PayloadDisplayName cannot contain FLEET_SECRET variables")
	assert.ErrorContains(t, err, "profiles[1]")
}

func TestNewMDMAppleDeclarationFreeLicenseTeam(t *testing.T) {
	svc, ctx, _, _ := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierFree})
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	b := declBytesForTest("D1", "d1content")

	_, err := svc.NewMDMAppleDeclaration(ctx, 1, b, nil, "name", fleet.LabelsIncludeAll)
	assert.ErrorIs(t, err, fleet.ErrMissingLicense)
}

func TestNewMDMAppleDeclaration(t *testing.T) {
	svc, ctx, ds, _ := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	// Unsupported Fleet variable
	b := declBytesForTest("D1", "d1content $FLEET_VAR_BOZO")
	_, err := svc.NewMDMAppleDeclaration(ctx, 0, b, nil, "name", fleet.LabelsIncludeAll)
	assert.ErrorContains(t, err, "Fleet variable")

	// decl type missing actual type
	b = declarationForTestWithType("D1", "com.apple.configuration")
	_, err = svc.NewMDMAppleDeclaration(ctx, 0, b, nil, "name", fleet.LabelsIncludeAll)
	assert.ErrorContains(t, err, "Only configuration declarations (com.apple.configuration.) are supported")

	ds.NewMDMAppleDeclarationFunc = func(ctx context.Context, d *fleet.MDMAppleDeclaration, usesFleetVars []fleet.FleetVarName) (*fleet.MDMAppleDeclaration, error) {
		return d, nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hids, tids []uint, puuids, uuids []string,
	) (updates fleet.MDMProfilesUpdates, err error) {
		return fleet.MDMProfilesUpdates{}, nil
	}

	// Good declaration
	b = declBytesForTest("D1", "d1content")
	d, err := svc.NewMDMAppleDeclaration(ctx, 0, b, nil, "name", fleet.LabelsIncludeAll)
	require.NoError(t, err)
	assert.NotNil(t, d)
}

func setupAppleMDMServiceWithSkipValidation(t *testing.T, license *fleet.LicenseInfo, skipValidation bool) (fleet.Service, context.Context, *mock.Store) {
	ds := new(mock.Store)
	cfg := config.TestConfig()
	cfg.MDM.AllowAllDeclarations = skipValidation
	testCertPEM, testKeyPEM, err := generateCertWithAPNsTopic()
	require.NoError(t, err)
	config.SetTestMDMConfig(t, &cfg, testCertPEM, testKeyPEM, "../../server/service/testdata")

	mdmStorage := &mdmmock.MDMAppleStore{}
	depStorage := &nanodep_mock.Storage{}
	pushFactory, _ := newMockAPNSPushProviderFactory()
	pusher := nanomdm_pushsvc.New(
		mdmStorage,
		mdmStorage,
		pushFactory,
		NewNanoMDMLogger(slog.New(slog.NewJSONHandler(os.Stdout, nil))),
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

	return svc, ctx, ds
}

func TestNewMDMAppleDeclarationSkipValidation(t *testing.T) {
	t.Run("forbidden declaration type fails validation by default", func(t *testing.T) {
		svc, ctx, ds := setupAppleMDMServiceWithSkipValidation(t, &fleet.LicenseInfo{Tier: fleet.TierPremium}, false)
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

		ds.ExpandEmbeddedSecretsAndUpdatedAtFunc = func(ctx context.Context, s string) (string, *time.Time, error) {
			return s, nil, nil
		}

		// Status subscription declarations are forbidden
		b := []byte(`{
			"Type": "com.apple.configuration.management.status-subscriptions",
			"Identifier": "test-status-sub"
		}`)
		_, err := svc.NewMDMAppleDeclaration(ctx, 0, b, nil, "test-status-sub", fleet.LabelsIncludeAll)
		require.Error(t, err)
		assert.ErrorContains(t, err, "status subscription type")
	})

	t.Run("forbidden declaration type allowed with skip validation", func(t *testing.T) {
		svc, ctx, ds := setupAppleMDMServiceWithSkipValidation(t, &fleet.LicenseInfo{Tier: fleet.TierPremium}, true)
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

		ds.ExpandEmbeddedSecretsAndUpdatedAtFunc = func(ctx context.Context, s string) (string, *time.Time, error) {
			return s, nil, nil
		}
		ds.NewMDMAppleDeclarationFunc = func(ctx context.Context, d *fleet.MDMAppleDeclaration, usesFleetVars []fleet.FleetVarName) (*fleet.MDMAppleDeclaration, error) {
			return d, nil
		}
		ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hids, tids []uint, puuids, uuids []string,
		) (updates fleet.MDMProfilesUpdates, err error) {
			return fleet.MDMProfilesUpdates{}, nil
		}

		// Status subscription declarations are forbidden but should be allowed when skip is enabled
		b := []byte(`{
			"Type": "com.apple.configuration.management.status-subscriptions",
			"Identifier": "test-status-sub"
		}`)
		d, err := svc.NewMDMAppleDeclaration(ctx, 0, b, nil, "test-status-sub", fleet.LabelsIncludeAll)
		require.NoError(t, err)
		assert.NotNil(t, d)
	})

	t.Run("invalid declaration type fails by default", func(t *testing.T) {
		svc, ctx, ds := setupAppleMDMServiceWithSkipValidation(t, &fleet.LicenseInfo{Tier: fleet.TierPremium}, false)
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

		ds.ExpandEmbeddedSecretsAndUpdatedAtFunc = func(ctx context.Context, s string) (string, *time.Time, error) {
			return s, nil, nil
		}

		// Non com.apple.configuration.* types are invalid
		b := []byte(`{
			"Type": "com.example.invalid",
			"Identifier": "test-invalid"
		}`)
		_, err := svc.NewMDMAppleDeclaration(ctx, 0, b, nil, "test-invalid", fleet.LabelsIncludeAll)
		require.Error(t, err)
		assert.ErrorContains(t, err, "Only configuration declarations")
	})

	t.Run("invalid declaration type allowed with skip validation", func(t *testing.T) {
		svc, ctx, ds := setupAppleMDMServiceWithSkipValidation(t, &fleet.LicenseInfo{Tier: fleet.TierPremium}, true)
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

		ds.ExpandEmbeddedSecretsAndUpdatedAtFunc = func(ctx context.Context, s string) (string, *time.Time, error) {
			return s, nil, nil
		}
		ds.NewMDMAppleDeclarationFunc = func(ctx context.Context, d *fleet.MDMAppleDeclaration, usesFleetVars []fleet.FleetVarName) (*fleet.MDMAppleDeclaration, error) {
			return d, nil
		}
		ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hids, tids []uint, puuids, uuids []string,
		) (updates fleet.MDMProfilesUpdates, err error) {
			return fleet.MDMProfilesUpdates{}, nil
		}

		// Invalid type should be allowed when skip is enabled
		b := []byte(`{
			"Type": "com.example.invalid",
			"Identifier": "test-invalid"
		}`)
		d, err := svc.NewMDMAppleDeclaration(ctx, 0, b, nil, "test-invalid", fleet.LabelsIncludeAll)
		require.NoError(t, err)
		assert.NotNil(t, d)
	})

	t.Run("OS update declaration blocked without custom OS updates flag", func(t *testing.T) {
		svc, ctx, ds := setupAppleMDMServiceWithSkipValidation(t, &fleet.LicenseInfo{Tier: fleet.TierPremium}, false)
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

		ds.ExpandEmbeddedSecretsAndUpdatedAtFunc = func(ctx context.Context, s string) (string, *time.Time, error) {
			return s, nil, nil
		}

		b := []byte(`{
			"Type": "com.apple.configuration.softwareupdate.enforcement.specific",
			"Identifier": "test-os-update"
		}`)
		_, err := svc.NewMDMAppleDeclaration(ctx, 0, b, nil, "test-os-update", fleet.LabelsIncludeAll)
		require.Error(t, err)
		assert.ErrorContains(t, err, "OS updates settings")
	})

	t.Run("OS update declaration allowed with skip validation even without custom OS updates flag", func(t *testing.T) {
		svc, ctx, ds := setupAppleMDMServiceWithSkipValidation(t, &fleet.LicenseInfo{Tier: fleet.TierPremium}, true)
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

		ds.ExpandEmbeddedSecretsAndUpdatedAtFunc = func(ctx context.Context, s string) (string, *time.Time, error) {
			return s, nil, nil
		}
		ds.NewMDMAppleDeclarationFunc = func(ctx context.Context, d *fleet.MDMAppleDeclaration, usesFleetVars []fleet.FleetVarName) (*fleet.MDMAppleDeclaration, error) {
			return d, nil
		}
		ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hids, tids []uint, puuids, uuids []string,
		) (updates fleet.MDMProfilesUpdates, err error) {
			return fleet.MDMProfilesUpdates{}, nil
		}

		b := []byte(`{
			"Type": "com.apple.configuration.softwareupdate.enforcement.specific",
			"Identifier": "test-os-update"
		}`)
		d, err := svc.NewMDMAppleDeclaration(ctx, 0, b, nil, "test-os-update", fleet.LabelsIncludeAll)
		require.NoError(t, err)
		assert.NotNil(t, d)
	})
}

// Fragile test: This test is fragile because of the large reliance on Datastore mocks. Consider refactoring test/logic or removing the test. It may be slowing us down more than helping us.
func TestHostDetailsMDMProfiles(t *testing.T) {
	svc, ctx, ds, _ := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})
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
	ds.ListUpcomingHostMaintenanceWindowsFunc = func(ctx context.Context, hid uint) ([]*fleet.HostMaintenanceWindow, error) {
		return nil, nil
	}
	ds.ListPoliciesForHostFunc = func(ctx context.Context, host *fleet.Host) ([]*fleet.HostPolicy, error) {
		return nil, nil
	}
	ds.GetHostMDMMacOSSetupFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDMMacOSSetup, error) {
		return nil, nil
	}
	ds.GetHostLockWipeStatusFunc = func(ctx context.Context, host *fleet.Host) (*fleet.HostLockWipeStatus, error) {
		return &fleet.HostLockWipeStatus{}, nil
	}
	ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
		return nil, nil
	}
	ds.ListHostDeviceMappingFunc = func(ctx context.Context, id uint) ([]*fleet.HostDeviceMapping, error) {
		return nil, nil
	}
	ds.ConditionalAccessBypassedAtFunc = func(ctx context.Context, hostID uint) (*time.Time, error) {
		return nil, nil
	}
	ds.GetHostIssuesLastUpdatedFunc = func(ctx context.Context, hostId uint) (time.Time, error) {
		return time.Time{}, nil
	}
	ds.UpdateHostIssuesFailingPoliciesFunc = func(ctx context.Context, hostIDs []uint) error {
		return nil
	}
	ds.UpdateHostIssuesFailingPoliciesForSingleHostFunc = func(ctx context.Context, hostID uint) error {
		return nil
	}
	ds.IsHostDiskEncryptionKeyArchivedFunc = func(ctx context.Context, hostID uint) (bool, error) {
		return false, nil
	}
	ds.ConditionalAccessBypassedAtFunc = func(ctx context.Context, hostID uint) (*time.Time, error) {
		return nil, nil
	}
	ds.GetHostRecoveryLockPasswordStatusFunc = func(ctx context.Context, hostUUID string) (*fleet.HostMDMRecoveryLockPassword, error) {
		return nil, nil
	}
	ds.GetHostManagedLocalAccountStatusFunc = func(ctx context.Context, hostUUID string) (*fleet.HostMDMManagedLocalAccount, error) {
		return nil, nil
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
						ep = append(ep, p.ToHostMDMProfile(gotHost.Platform))
					}
				}
				require.Equal(t, gotHost.MDM.Profiles, &ep)
				return
			}

			require.True(t, ds.GetHostMDMAppleProfilesFuncInvoked)
			require.NotNil(t, gotHost.MDM.Profiles)
			ep := make([]fleet.HostMDMProfile, 0, len(*gotHost.MDM.Profiles))
			for _, p := range *c.expected {
				ep = append(ep, p.ToHostMDMProfile(gotHost.Platform))
			}
			require.ElementsMatch(t, ep, *gotHost.MDM.Profiles)
		})
	}
}

func TestMDMCommandAuthz(t *testing.T) {
	svc, ctx, ds, _ := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})

	ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
		switch hostID {
		case 1:
			return &fleet.Host{UUID: "test-host-team-1", TeamID: ptr.Uint(1), Platform: "darwin"}, nil
		default:
			return &fleet.Host{UUID: "test-host-no-team", Platform: "darwin"}, nil
		}
	}

	ds.GetHostMDMCheckinInfoFunc = func(ctx context.Context, hostUUID string) (*fleet.HostMDMCheckinInfo, error) {
		return &fleet.HostMDMCheckinInfo{Platform: "darwin"}, nil
	}

	ds.MDMTurnOffFunc = func(ctx context.Context, uuid string) ([]*fleet.User, []fleet.ActivityDetails, error) {
		return nil, nil, nil
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
			err := svc.UnenrollMDM(ctx, 42) // global host
			if !tt.shouldFailGlobal {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
			}

			mdmEnabled.Store(true)
			err = svc.UnenrollMDM(ctx, 1) // host belongs to team 1
			if !tt.shouldFailTeam {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
			}
		})
	}
}

func TestMDMAuthenticateManualEnrollment(t *testing.T) {
	ds := new(mock.Store)
	mdmLifecycle := mdmlifecycle.New(ds, slog.New(slog.DiscardHandler), func(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error { return nil })
	svc := MDMAppleCheckinAndCommandService{
		ds:            ds,
		mdmLifecycle:  mdmLifecycle,
		keyValueStore: redis_key_value.New(redistest.NopRedis()),
		logger:        slog.New(slog.DiscardHandler),
	}
	ctx := context.Background()
	uuid, serial, model := "ABC-DEF-GHI", "XYZABC", "MacBookPro 16,1"

	ds.MDMAppleUpsertHostFunc = func(ctx context.Context, mdmHost *fleet.Host, fromPersonalEnrollment bool) error {
		require.Equal(t, uuid, mdmHost.UUID)
		require.Equal(t, serial, mdmHost.HardwareSerial)
		require.Equal(t, model, mdmHost.HardwareModel)
		require.False(t, fromPersonalEnrollment)
		return nil
	}

	ds.GetHostMDMCheckinInfoFunc = func(ct context.Context, hostUUID string) (*fleet.HostMDMCheckinInfo, error) {
		require.Equal(t, uuid, hostUUID)
		return &fleet.HostMDMCheckinInfo{
			HardwareSerial:   serial,
			DisplayName:      fmt.Sprintf("%s (%s)", model, serial),
			InstalledFromDEP: false,
		}, nil
	}

	ds.MDMResetEnrollmentFunc = func(ctx context.Context, hostUUID string, scepRenewalInProgress bool) error {
		require.Equal(t, uuid, hostUUID)
		return nil
	}

	err := svc.Authenticate(
		&mdm.Request{Context: ctx, EnrollID: &mdm.EnrollID{ID: uuid}},
		&mdm.Authenticate{
			Enrollment: mdm.Enrollment{
				UDID: uuid,
			},
			SerialNumber: serial,
			Model:        model,
		},
	)
	require.NoError(t, err)
	require.True(t, ds.MDMAppleUpsertHostFuncInvoked)
	require.True(t, ds.GetHostMDMCheckinInfoFuncInvoked)
	require.True(t, ds.MDMResetEnrollmentFuncInvoked)
}

func TestMDMAuthenticateADE(t *testing.T) {
	ds := new(mock.Store)
	mdmLifecycle := mdmlifecycle.New(ds, slog.New(slog.DiscardHandler), func(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error { return nil })
	svc := MDMAppleCheckinAndCommandService{
		ds:            ds,
		mdmLifecycle:  mdmLifecycle,
		keyValueStore: redis_key_value.New(redistest.NopRedis()),
		logger:        slog.New(slog.DiscardHandler),
	}
	ctx := context.Background()
	uuid, serial, model := "ABC-DEF-GHI", "XYZABC", "MacBookPro 16,1"

	ds.MDMAppleUpsertHostFunc = func(ctx context.Context, mdmHost *fleet.Host, fromPersonalEnrollment bool) error {
		require.Equal(t, uuid, mdmHost.UUID)
		require.Equal(t, serial, mdmHost.HardwareSerial)
		require.Equal(t, model, mdmHost.HardwareModel)
		require.False(t, fromPersonalEnrollment)
		return nil
	}

	ds.GetHostMDMCheckinInfoFunc = func(ct context.Context, hostUUID string) (*fleet.HostMDMCheckinInfo, error) {
		require.Equal(t, uuid, hostUUID)
		return &fleet.HostMDMCheckinInfo{
			HardwareSerial:     serial,
			DisplayName:        fmt.Sprintf("%s (%s)", model, serial),
			DEPAssignedToFleet: true,
		}, nil
	}

	ds.MDMResetEnrollmentFunc = func(ctx context.Context, hostUUID string, scepRenewalInProgress bool) error {
		require.Equal(t, uuid, hostUUID)
		return nil
	}

	err := svc.Authenticate(
		&mdm.Request{Context: ctx, EnrollID: &mdm.EnrollID{ID: uuid}},
		&mdm.Authenticate{
			Enrollment: mdm.Enrollment{
				UDID: uuid,
			},
			SerialNumber: serial,
			Model:        model,
		},
	)
	require.NoError(t, err)
	require.True(t, ds.MDMAppleUpsertHostFuncInvoked)
	require.True(t, ds.GetHostMDMCheckinInfoFuncInvoked)
	require.True(t, ds.MDMResetEnrollmentFuncInvoked)
}

func TestMDMAuthenticateSCEPRenewal(t *testing.T) {
	ds := new(mock.Store)
	var newActivityInvoked bool
	mdmLifecycle := mdmlifecycle.New(ds, slog.New(slog.DiscardHandler), func(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error {
		newActivityInvoked = true
		return nil
	})
	svc := MDMAppleCheckinAndCommandService{
		ds:           ds,
		mdmLifecycle: mdmLifecycle,
		logger:       slog.New(slog.DiscardHandler),
	}
	ctx := context.Background()
	uuid, serial, model := "ABC-DEF-GHI", "XYZABC", "MacBookPro 16,1"

	ds.GetHostMDMCheckinInfoFunc = func(ct context.Context, hostUUID string) (*fleet.HostMDMCheckinInfo, error) {
		require.Equal(t, uuid, hostUUID)
		return &fleet.HostMDMCheckinInfo{
			HardwareSerial:        serial,
			DisplayName:           fmt.Sprintf("%s (%s)", model, serial),
			SCEPRenewalInProgress: true,
		}, nil
	}

	ds.MDMResetEnrollmentFunc = func(ctx context.Context, hostUUID string, scepRenewalInProgress bool) error {
		require.Equal(t, uuid, hostUUID)
		require.True(t, scepRenewalInProgress)
		return nil
	}
	ds.MDMAppleUpsertHostFunc = func(ctx context.Context, mdmHost *fleet.Host, fromPersonalEnrollment bool) error {
		require.Equal(t, uuid, mdmHost.UUID)
		require.Equal(t, serial, mdmHost.HardwareSerial)
		require.Equal(t, model, mdmHost.HardwareModel)
		require.False(t, fromPersonalEnrollment)
		return nil
	}

	err := svc.Authenticate(
		&mdm.Request{Context: ctx, EnrollID: &mdm.EnrollID{ID: uuid}},
		&mdm.Authenticate{
			Enrollment: mdm.Enrollment{
				UDID: uuid,
			},
			SerialNumber: serial,
			Model:        model,
		},
	)
	require.NoError(t, err)
	require.False(t, ds.MDMAppleUpsertHostFuncInvoked)
	require.True(t, ds.GetHostMDMCheckinInfoFuncInvoked)
	require.False(t, newActivityInvoked)
	require.True(t, ds.MDMResetEnrollmentFuncInvoked)
}

func TestAppleMDMUnenrollment(t *testing.T) {
	svc, ctx, ds, _ := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{ID: 1, GlobalRole: ptr.String(fleet.RoleAdmin)}})

	hostOne := &fleet.Host{ID: 1, UUID: "test-host-no-team-2", Platform: "ios"}
	hostGlobal := &fleet.Host{ID: 42, UUID: "test-host-no-team", Platform: "darwin"}

	ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
		switch hostID {
		case hostOne.ID:
			return hostOne, nil
		case hostGlobal.ID:
			return hostGlobal, nil
		default:
			return nil, errors.New("not found")
		}
	}

	ds.GetHostMDMCheckinInfoFunc = func(ctx context.Context, hostUUID string) (*fleet.HostMDMCheckinInfo, error) {
		return &fleet.HostMDMCheckinInfo{Platform: "darwin"}, nil
	}

	ds.MDMTurnOffFunc = func(ctx context.Context, uuid string) ([]*fleet.User, []fleet.ActivityDetails, error) {
		return nil, nil, nil
	}

	ds.GetNanoMDMEnrollmentFunc = func(ctx context.Context, hostUUID string) (*fleet.NanoEnrollment, error) {
		enrollmentType := mdm.EnrollType(mdm.Device).String()
		if hostUUID == "test-host-no-team-2" {
			enrollmentType = mdm.EnrollType(mdm.UserEnrollmentDevice).String()
		}
		enroll := fleet.NanoEnrollment{
			Enabled: true,
			Type:    enrollmentType,
		}
		return &enroll, nil
	}

	t.Run("Unenrolls macos device", func(t *testing.T) {
		err := svc.UnenrollMDM(ctx, hostGlobal.ID) // global host
		require.NoError(t, err)
	})

	t.Run("Unenrolls personal ios device", func(t *testing.T) {
		err := svc.UnenrollMDM(ctx, hostOne.ID) // personal host
		require.NoError(t, err)
	})
}

func TestMDMTokenUpdate(t *testing.T) {
	ctx := license.NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierPremium})
	ds := new(mock.Store)
	mdmStorage := &mdmmock.MDMAppleStore{}
	pushFactory, _ := newMockAPNSPushProviderFactory()
	pusher := nanomdm_pushsvc.New(
		mdmStorage,
		mdmStorage,
		pushFactory,
		NewNanoMDMLogger(slog.New(slog.NewJSONHandler(os.Stdout, nil))),
	)
	cmdr := apple_mdm.NewMDMAppleCommander(mdmStorage, pusher)
	uuid, serial, model, wantTeamID := "ABC-DEF-GHI", "XYZABC", "MacBookPro 16,1", uint(12)
	var newActivityFuncInvoked bool
	mdmLifecycle := mdmlifecycle.New(ds, slog.New(slog.DiscardHandler), func(_ context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		newActivityFuncInvoked = true
		a, ok := activity.(*fleet.ActivityTypeMDMEnrolled)
		require.True(t, ok)
		require.Nil(t, user)
		require.Equal(t, "mdm_enrolled", activity.ActivityName())
		require.NotNil(t, a.HostSerial)
		require.Equal(t, serial, *a.HostSerial)
		require.Nil(t, a.EnrollmentID)
		require.Equal(t, a.HostDisplayName, model)
		require.True(t, a.InstalledFromDEP)
		require.Equal(t, fleet.MDMPlatformApple, a.MDMPlatform)
		return nil
	})
	svc := MDMAppleCheckinAndCommandService{
		ds:           ds,
		mdmLifecycle: mdmLifecycle,
		commander:    cmdr,
		logger:       slog.New(slog.DiscardHandler),
	}

	ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	ds.GetNanoMDMEnrollmentFunc = func(ctx context.Context, hostUUID string) (*fleet.NanoEnrollment, error) {
		return &fleet.NanoEnrollment{Enabled: true, Type: "Device", TokenUpdateTally: 1}, nil
	}

	ds.GetHostMDMCheckinInfoFunc = func(ct context.Context, hostUUID string) (*fleet.HostMDMCheckinInfo, error) {
		require.Equal(t, uuid, hostUUID)
		return &fleet.HostMDMCheckinInfo{
			HostID:             1337,
			HardwareSerial:     serial,
			DisplayName:        model,
			InstalledFromDEP:   true,
			TeamID:             wantTeamID,
			DEPAssignedToFleet: true,
			Platform:           "darwin",
		}, nil
	}

	ds.GetMDMIdPAccountByHostUUIDFunc = func(ctx context.Context, hostUUID string) (*fleet.MDMIdPAccount, error) {
		require.Equal(t, uuid, hostUUID)
		return &fleet.MDMIdPAccount{
			UUID:     "some-uuid",
			Username: "some-user",
			Email:    "some-user@example.com",
			Fullname: "Some User",
		}, nil
	}

	ds.NewJobFunc = func(ctx context.Context, j *fleet.Job) (*fleet.Job, error) {
		return j, nil
	}

	ds.MDMAppleResetOnReenrollmentFunc = func(ctx context.Context, hostUUID string, preserveHostActivities bool) error {
		return nil
	}

	err := svc.TokenUpdate(
		&mdm.Request{Context: ctx, EnrollID: &mdm.EnrollID{ID: uuid}},
		&mdm.TokenUpdate{
			TokenUpdateEnrollment: mdm.TokenUpdateEnrollment{
				Enrollment: mdm.Enrollment{UDID: uuid},
			},
		},
	)
	require.NoError(t, err)
	require.True(t, ds.GetHostMDMCheckinInfoFuncInvoked)
	require.True(t, ds.NewJobFuncInvoked)
	require.True(t, newActivityFuncInvoked)
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
			TokenUpdateEnrollment: mdm.TokenUpdateEnrollment{
				Enrollment: mdm.Enrollment{UDID: uuid},
			},
		},
	)
	require.NoError(t, err)
	require.True(t, ds.GetHostMDMCheckinInfoFuncInvoked)
	require.True(t, ds.NewJobFuncInvoked)
	require.True(t, newActivityFuncInvoked)

	// With AwaitingConfiguration - should check for and enqueue SetupExperience items
	ds.EnqueueSetupExperienceItemsFunc = func(ctx context.Context, hostPlatform, hostPlatformLike string, hostUUID string, teamID uint) (bool, error) {
		require.Equal(t, "darwin", hostPlatformLike)
		require.Equal(t, uuid, hostUUID)
		require.Equal(t, wantTeamID, teamID)
		return true, nil
	}

	ds.ClearHostEnrolledFromMigrationFunc = func(ctx context.Context, hostUUID string) error {
		require.Equal(t, uuid, hostUUID)
		return nil
	}

	err = svc.TokenUpdate(
		&mdm.Request{
			Context:  ctx,
			EnrollID: &mdm.EnrollID{ID: uuid},
			Params:   map[string]string{"enroll_reference": "abcd"},
		},
		&mdm.TokenUpdate{
			TokenUpdateEnrollment: mdm.TokenUpdateEnrollment{
				AwaitingConfiguration: true,
				Enrollment: mdm.Enrollment{
					UDID: uuid,
				},
			},
		},
	)
	require.NoError(t, err)
	require.True(t, ds.EnqueueSetupExperienceItemsFuncInvoked)
	require.True(t, ds.ClearHostEnrolledFromMigrationFuncInvoked)

	ds.GetHostMDMCheckinInfoFunc = func(ct context.Context, hostUUID string) (*fleet.HostMDMCheckinInfo, error) {
		require.Equal(t, uuid, hostUUID)
		return &fleet.HostMDMCheckinInfo{
			HostID:              1337,
			HardwareSerial:      serial,
			DisplayName:         model,
			InstalledFromDEP:    true,
			TeamID:              wantTeamID,
			DEPAssignedToFleet:  true,
			Platform:            "darwin",
			MigrationInProgress: true,
		}, nil
	}

	ds.SetHostMDMMigrationCompletedFunc = func(ctx context.Context, hostID uint) error {
		require.Equal(t, uint(1337), hostID)
		return nil
	}

	ds.EnqueueSetupExperienceItemsFuncInvoked = false
	ds.ClearHostEnrolledFromMigrationFuncInvoked = false
	err = svc.TokenUpdate(
		&mdm.Request{
			Context:  ctx,
			EnrollID: &mdm.EnrollID{ID: uuid},
			Params:   map[string]string{"enroll_reference": "abcd"},
		},
		&mdm.TokenUpdate{
			TokenUpdateEnrollment: mdm.TokenUpdateEnrollment{
				AwaitingConfiguration: true,
				Enrollment: mdm.Enrollment{
					UDID: uuid,
				},
			},
		},
	)
	require.NoError(t, err)
	// Should NOT call the setup experience enqueue function but it should mark the migration complete
	require.False(t, ds.EnqueueSetupExperienceItemsFuncInvoked)
	require.True(t, ds.SetHostMDMMigrationCompletedFuncInvoked)
	require.True(t, ds.ClearHostEnrolledFromMigrationFuncInvoked)
	require.True(t, newActivityFuncInvoked)

	ds.SetHostMDMMigrationCompletedFuncInvoked = false
	err = svc.TokenUpdate(
		&mdm.Request{
			Context:  ctx,
			EnrollID: &mdm.EnrollID{ID: uuid},
			Params:   map[string]string{"enroll_reference": "abcd"},
		},
		&mdm.TokenUpdate{
			TokenUpdateEnrollment: mdm.TokenUpdateEnrollment{
				Enrollment: mdm.Enrollment{UDID: uuid},
			},
		},
	)
	require.NoError(t, err)
	// Should NOT call the setup experience enqueue function but it should mark the migration complete
	require.False(t, ds.EnqueueSetupExperienceItemsFuncInvoked)
	require.True(t, ds.SetHostMDMMigrationCompletedFuncInvoked)
	require.True(t, newActivityFuncInvoked)
}

// TestMDMTokenUpdateResetOnReenrollment exercises the gate around
// svc.ds.MDMAppleResetOnReenrollment in TokenUpdate. The reset must run only
// when:
//   - the device reports AwaitingConfiguration (DEP enrollment), AND
//   - the nano enrollment's TokenUpdateTally == 1 (first token update), AND
//   - we are NOT in a darwin migration (skipped to preserve host state).
func TestMDMTokenUpdateResetOnReenrollment(t *testing.T) {
	const hostUUID = "ABC-DEF-GHI"

	// newSvc returns a service wired up with a fresh mock.Store and minimal
	// happy-path mocks for everything TokenUpdate touches besides the gate.
	// Per-case overrides go on the returned ds.
	newSvc := func(t *testing.T) (*MDMAppleCheckinAndCommandService, *mock.Store) {
		ds := new(mock.Store)
		mdmStorage := &mdmmock.MDMAppleStore{}
		pushFactory, _ := newMockAPNSPushProviderFactory()
		pusher := nanomdm_pushsvc.New(
			mdmStorage,
			mdmStorage,
			pushFactory,
			NewNanoMDMLogger(slog.New(slog.NewJSONHandler(os.Stdout, nil))),
		)
		cmdr := apple_mdm.NewMDMAppleCommander(mdmStorage, pusher)
		mdmLifecycle := mdmlifecycle.New(ds, slog.New(slog.DiscardHandler), func(context.Context, *fleet.User, fleet.ActivityDetails) error { return nil })
		svc := &MDMAppleCheckinAndCommandService{
			ds:           ds,
			mdmLifecycle: mdmLifecycle,
			commander:    cmdr,
			logger:       slog.New(slog.DiscardHandler),
		}

		// Defaults: each case overrides the fields it cares about.
		ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{}, nil
		}
		ds.GetNanoMDMEnrollmentFunc = func(context.Context, string) (*fleet.NanoEnrollment, error) {
			return &fleet.NanoEnrollment{Enabled: true, Type: "Device", TokenUpdateTally: 1}, nil
		}
		ds.GetHostMDMCheckinInfoFunc = func(context.Context, string) (*fleet.HostMDMCheckinInfo, error) {
			return &fleet.HostMDMCheckinInfo{HostID: 1, Platform: "darwin"}, nil
		}
		ds.GetMDMIdPAccountByHostUUIDFunc = func(context.Context, string) (*fleet.MDMIdPAccount, error) {
			return nil, nil
		}
		ds.MDMAppleResetOnReenrollmentFunc = func(context.Context, string, bool) error { return nil }
		ds.ClearHostEnrolledFromMigrationFunc = func(context.Context, string) error { return nil }
		ds.SetHostMDMMigrationCompletedFunc = func(context.Context, uint) error { return nil }
		ds.CleanSCEPRenewRefsFunc = func(context.Context, string) error { return nil }
		ds.MDMResetEnrollmentFunc = func(context.Context, string, bool) error { return nil }
		ds.EnqueueSetupExperienceItemsFunc = func(context.Context, string, string, string, uint) (bool, error) {
			return false, nil
		}
		ds.NewJobFunc = func(_ context.Context, j *fleet.Job) (*fleet.Job, error) { return j, nil }

		return svc, ds
	}

	type tc struct {
		name                  string
		platform              string
		awaitingConfig        bool
		tokenUpdateTally      int
		migrationInProgress   bool
		nanoEnrollNil         bool
		scepRenewalInProgress bool
		wantResetCall         bool
	}

	cases := []tc{
		{
			name:             "manual enrollment - AwaitingConfig=false even with TokenUpdateTally=1",
			platform:         "darwin",
			awaitingConfig:   false,
			tokenUpdateTally: 1,
			wantResetCall:    false,
		},
		{
			name:                "darwin migration in progress - skipped",
			platform:            "darwin",
			awaitingConfig:      true,
			tokenUpdateTally:    1,
			migrationInProgress: true,
			wantResetCall:       false,
		},
		{
			name:             "macOS DEP first token update",
			platform:         "darwin",
			awaitingConfig:   true,
			tokenUpdateTally: 1,
			wantResetCall:    true,
		},
		{
			name:             "iOS DEP first token update",
			platform:         "ios",
			awaitingConfig:   true,
			tokenUpdateTally: 1,
			wantResetCall:    true,
		},
		{
			name:             "iPadOS DEP first token update",
			platform:         "ipados",
			awaitingConfig:   true,
			tokenUpdateTally: 1,
			wantResetCall:    true,
		},
		{
			name:             "subsequent token update - tally != 1",
			platform:         "darwin",
			awaitingConfig:   true,
			tokenUpdateTally: 2,
			wantResetCall:    false,
		},
		{
			name:                "iOS migration in progress - NOT skipped (skip is darwin-only)",
			platform:            "ios",
			awaitingConfig:      true,
			tokenUpdateTally:    1,
			migrationInProgress: true,
			wantResetCall:       true,
		},
		{
			name:                "iPadOS migration in progress - NOT skipped (skip is darwin-only)",
			platform:            "ipados",
			awaitingConfig:      true,
			tokenUpdateTally:    1,
			migrationInProgress: true,
			wantResetCall:       true,
		},
		{
			name:             "nano enrollment is nil",
			platform:         "darwin",
			awaitingConfig:   true,
			tokenUpdateTally: 1, // ignored - nanoEnroll itself is nil
			nanoEnrollNil:    true,
			wantResetCall:    false,
		},
		{
			name:                  "SCEP renewal without AwaitingConfiguration - short-circuits",
			platform:              "darwin",
			awaitingConfig:        false,
			tokenUpdateTally:      1,
			scepRenewalInProgress: true,
			wantResetCall:         false,
		},
		{
			name:                  "SCEP renewal with AwaitingConfiguration - falls through and calls reset",
			platform:              "darwin",
			awaitingConfig:        true,
			tokenUpdateTally:      1,
			scepRenewalInProgress: true,
			wantResetCall:         true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc, ds := newSvc(t)

			ds.GetHostMDMCheckinInfoFunc = func(context.Context, string) (*fleet.HostMDMCheckinInfo, error) {
				return &fleet.HostMDMCheckinInfo{
					HostID:                1,
					Platform:              c.platform,
					MigrationInProgress:   c.migrationInProgress,
					SCEPRenewalInProgress: c.scepRenewalInProgress,
				}, nil
			}
			ds.GetNanoMDMEnrollmentFunc = func(context.Context, string) (*fleet.NanoEnrollment, error) {
				if c.nanoEnrollNil {
					return nil, nil
				}
				return &fleet.NanoEnrollment{Enabled: true, Type: "Device", TokenUpdateTally: c.tokenUpdateTally}, nil
			}

			err := svc.TokenUpdate(
				&mdm.Request{Context: context.Background(), EnrollID: &mdm.EnrollID{ID: hostUUID, Type: mdm.Device}},
				&mdm.TokenUpdate{
					TokenUpdateEnrollment: mdm.TokenUpdateEnrollment{
						AwaitingConfiguration: c.awaitingConfig,
						Enrollment:            mdm.Enrollment{UDID: hostUUID},
					},
				},
			)
			require.NoError(t, err)
			assert.Equal(t, c.wantResetCall, ds.MDMAppleResetOnReenrollmentFuncInvoked,
				"MDMAppleResetOnReenrollment invocation mismatch")
		})
	}

	t.Run("forwards PreserveHostActivitiesOnReenrollment from AppConfig", func(t *testing.T) {
		svc, ds := newSvc(t)
		ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) {
			cfg := &fleet.AppConfig{}
			cfg.ActivityExpirySettings.PreserveHostActivitiesOnReenrollment = true
			return cfg, nil
		}
		var gotPreserve bool
		ds.MDMAppleResetOnReenrollmentFunc = func(_ context.Context, _ string, preserve bool) error {
			gotPreserve = preserve
			return nil
		}

		err := svc.TokenUpdate(
			&mdm.Request{Context: context.Background(), EnrollID: &mdm.EnrollID{ID: hostUUID, Type: mdm.Device}},
			&mdm.TokenUpdate{
				TokenUpdateEnrollment: mdm.TokenUpdateEnrollment{
					AwaitingConfiguration: true,
					Enrollment:            mdm.Enrollment{UDID: hostUUID},
				},
			},
		)
		require.NoError(t, err)
		require.True(t, ds.MDMAppleResetOnReenrollmentFuncInvoked)
		assert.True(t, gotPreserve, "preserve flag from AppConfig should be forwarded to the datastore call")
	})
}

func TestMDMTokenUpdateIOS(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)
	mdmStorage := &mdmmock.MDMAppleStore{}
	pushFactory, _ := newMockAPNSPushProviderFactory()
	pusher := nanomdm_pushsvc.New(
		mdmStorage,
		mdmStorage,
		pushFactory,
		NewNanoMDMLogger(slog.New(slog.NewJSONHandler(os.Stdout, nil))),
	)
	cmdr := apple_mdm.NewMDMAppleCommander(mdmStorage, pusher)
	mdmLifecycle := mdmlifecycle.New(ds, slog.New(slog.DiscardHandler), func(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error { return nil })
	svc := MDMAppleCheckinAndCommandService{
		ds:           ds,
		mdmLifecycle: mdmLifecycle,
		commander:    cmdr,
		logger:       slog.New(slog.DiscardHandler),
	}
	uuid, serial, model, wantTeamID := "ABC-DEF-GHI", "XYZABC", "MacBookPro 16,1", uint(12)

	ds.GetMDMIdPAccountByHostUUIDFunc = func(ctx context.Context, hostUUID string) (*fleet.MDMIdPAccount, error) {
		require.Equal(t, uuid, hostUUID)
		return &fleet.MDMIdPAccount{
			UUID:     "some-uuid",
			Username: "some-user",
			Email:    "some-user@example.com",
			Fullname: "Some User",
		}, nil
	}

	ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	ds.NewJobFunc = func(ctx context.Context, j *fleet.Job) (*fleet.Job, error) {
		return j, nil
	}

	ds.GetHostMDMCheckinInfoFunc = func(ct context.Context, hostUUID string) (*fleet.HostMDMCheckinInfo, error) {
		require.Equal(t, uuid, hostUUID)
		return &fleet.HostMDMCheckinInfo{
			HostID:             1337,
			HardwareSerial:     serial,
			DisplayName:        model,
			InstalledFromDEP:   true,
			TeamID:             wantTeamID,
			DEPAssignedToFleet: true,
			Platform:           "ios",
		}, nil
	}

	ds.GetNanoMDMEnrollmentFunc = func(ctx context.Context, hostUUID string) (*fleet.NanoEnrollment, error) {
		return &fleet.NanoEnrollment{Enabled: true, Type: "Device", TokenUpdateTally: 1}, nil
	}

	ds.EnqueueSetupExperienceItemsFunc = func(ctx context.Context, hostPlatform, hostPlatformLike string, hostUUID string, teamID uint) (bool, error) {
		require.Equal(t, "ios", hostPlatformLike)
		require.Equal(t, uuid, hostUUID)
		require.Equal(t, wantTeamID, teamID)
		return true, nil
	}

	// DEP-installed without AwaitingConfiguration - should not enqueue SetupExperience items
	err := svc.TokenUpdate(
		&mdm.Request{
			Context:  ctx,
			EnrollID: &mdm.EnrollID{ID: uuid, Type: mdm.Device},
			Params:   map[string]string{"enroll_reference": "abcd"},
		},
		&mdm.TokenUpdate{
			TokenUpdateEnrollment: mdm.TokenUpdateEnrollment{
				Enrollment: mdm.Enrollment{UDID: uuid},
			},
		},
	)
	require.NoError(t, err)
	require.False(t, ds.EnqueueSetupExperienceItemsFuncInvoked)

	// Non-DEP-installed, non device-type enrollment should not enqueue SetupExperience items
	err = svc.TokenUpdate(
		&mdm.Request{
			Context:  ctx,
			EnrollID: &mdm.EnrollID{ID: uuid, Type: mdm.User},
			Params:   map[string]string{"enroll_reference": "abcd"},
		},
		&mdm.TokenUpdate{
			TokenUpdateEnrollment: mdm.TokenUpdateEnrollment{
				Enrollment: mdm.Enrollment{UDID: uuid},
			},
		},
	)
	require.NoError(t, err)
	require.False(t, ds.EnqueueSetupExperienceItemsFuncInvoked)

	// Non-DEP-installed without AwaitingConfiguration - should not enqueue SetupExperience items if token count is > 1
	ds.GetHostMDMCheckinInfoFunc = func(ct context.Context, hostUUID string) (*fleet.HostMDMCheckinInfo, error) {
		require.Equal(t, uuid, hostUUID)
		return &fleet.HostMDMCheckinInfo{
			HostID:             1337,
			HardwareSerial:     serial,
			DisplayName:        model,
			InstalledFromDEP:   false,
			TeamID:             wantTeamID,
			DEPAssignedToFleet: true,
			Platform:           "ios",
		}, nil
	}

	ds.GetNanoMDMEnrollmentFunc = func(ctx context.Context, hostUUID string) (*fleet.NanoEnrollment, error) {
		return &fleet.NanoEnrollment{Enabled: true, Type: "Device", TokenUpdateTally: 2}, nil
	}

	err = svc.TokenUpdate(
		&mdm.Request{
			Context:  ctx,
			EnrollID: &mdm.EnrollID{ID: uuid, Type: mdm.Device},
			Params:   map[string]string{"enroll_reference": "abcd"},
		},
		&mdm.TokenUpdate{
			TokenUpdateEnrollment: mdm.TokenUpdateEnrollment{
				Enrollment: mdm.Enrollment{UDID: uuid},
			},
		},
	)
	require.NoError(t, err)
	require.False(t, ds.EnqueueSetupExperienceItemsFuncInvoked)

	// Non-DEP-installed without AwaitingConfiguration - should enqueue SetupExperience items if token count is 1
	ds.GetHostMDMCheckinInfoFunc = func(ct context.Context, hostUUID string) (*fleet.HostMDMCheckinInfo, error) {
		require.Equal(t, uuid, hostUUID)
		return &fleet.HostMDMCheckinInfo{
			HostID:             1337,
			HardwareSerial:     serial,
			DisplayName:        model,
			InstalledFromDEP:   false,
			TeamID:             wantTeamID,
			DEPAssignedToFleet: true,
			Platform:           "ios",
		}, nil
	}

	ds.GetNanoMDMEnrollmentFunc = func(ctx context.Context, hostUUID string) (*fleet.NanoEnrollment, error) {
		return &fleet.NanoEnrollment{Enabled: true, Type: "Device", TokenUpdateTally: 1}, nil
	}

	err = svc.TokenUpdate(
		&mdm.Request{
			Context:  ctx,
			EnrollID: &mdm.EnrollID{ID: uuid, Type: mdm.Device},
			Params:   map[string]string{"enroll_reference": "abcd"},
		},
		&mdm.TokenUpdate{
			TokenUpdateEnrollment: mdm.TokenUpdateEnrollment{
				Enrollment: mdm.Enrollment{UDID: uuid},
			},
		},
	)
	require.NoError(t, err)
	require.True(t, ds.EnqueueSetupExperienceItemsFuncInvoked)
}

func TestMDMCheckout(t *testing.T) {
	ds := new(mock.Store)
	mdmLifecycle := mdmlifecycle.New(ds, slog.New(slog.DiscardHandler), func(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error { return nil })
	var newActivityFuncInvoked bool
	svc := MDMAppleCheckinAndCommandService{
		ds:           ds,
		mdmLifecycle: mdmLifecycle,
		logger:       slog.New(slog.DiscardHandler),
	}
	ctx := context.Background()
	uuid, serial, installedFromDEP, displayName, platform := "ABC-DEF-GHI", "XYZABC", true, "Test's MacBook", "darwin"

	ds.MDMTurnOffFunc = func(ctx context.Context, hostUUID string) ([]*fleet.User, []fleet.ActivityDetails, error) {
		require.Equal(t, uuid, hostUUID)
		return nil, nil, nil
	}

	ds.GetHostMDMCheckinInfoFunc = func(ct context.Context, hostUUID string) (*fleet.HostMDMCheckinInfo, error) {
		require.Equal(t, uuid, hostUUID)
		return &fleet.HostMDMCheckinInfo{
			HardwareSerial:   serial,
			DisplayName:      displayName,
			InstalledFromDEP: installedFromDEP,
			Platform:         platform,
		}, nil
	}

	ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	svc.newActivityFn = func(
		_ context.Context, user *fleet.User, activity fleet.ActivityDetails,
	) error {
		newActivityFuncInvoked = true
		a, ok := activity.(*fleet.ActivityTypeMDMUnenrolled)
		require.True(t, ok)
		require.Nil(t, user)
		require.Equal(t, "mdm_unenrolled", activity.ActivityName())
		require.Equal(t, serial, a.HostSerial)
		require.Equal(t, displayName, a.HostDisplayName)
		require.True(t, a.InstalledFromDEP)
		require.Equal(t, platform, a.Platform)
		return nil
	}

	err := svc.CheckOut(
		&mdm.Request{
			Context:  ctx,
			EnrollID: &mdm.EnrollID{ID: uuid},
		},
		&mdm.CheckOut{
			Enrollment: mdm.Enrollment{
				UDID: uuid,
			},
		},
	)
	require.NoError(t, err)
	require.True(t, ds.MDMTurnOffFuncInvoked)
	require.True(t, ds.GetHostMDMCheckinInfoFuncInvoked)
	require.True(t, newActivityFuncInvoked)
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
			prevRetries: fleetmdm.MaxAppleProfileRetries, // expect to fail
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
		{
			status:      "Error",
			requestType: "RemoveProfile",
			errors: []mdm.ErrorChain{
				{ErrorCode: 89, ErrorDomain: "MDMClientError", USEnglishDescription: "Profile with identifier 'com.example' not found."},
			},
			want: &fleet.HostMDMAppleProfile{
				Status:        &fleet.MDMDeliveryVerifying,
				Detail:        "",
				OperationType: fleet.MDMOperationTypeRemove,
			},
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("%s%s-%d", c.requestType, c.status, i), func(t *testing.T) {
			ds := new(mock.Store)
			svc := MDMAppleCheckinAndCommandService{ds: ds, logger: slog.New(slog.DiscardHandler)}
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
					ErrorChain:  c.errors,
				},
			)
			require.NoError(t, err)
			require.True(t, ds.GetMDMAppleCommandRequestTypeFuncInvoked)
			var shouldCheckCount, shouldRetry, shouldUpdateOrDelete bool
			if c.requestType == "InstallProfile" && c.status == "Error" {
				shouldCheckCount = true
			}
			if shouldCheckCount && c.prevRetries < fleetmdm.MaxAppleProfileRetries {
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
	svc, ctx, ds, _ := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})

	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		return &fleet.Team{ID: 1, Name: name}, nil
	}
	ds.TeamWithExtrasFunc = func(ctx context.Context, id uint) (*fleet.Team, error) {
		return &fleet.Team{ID: id, Name: "team"}, nil
	}
	ds.BatchSetMDMAppleProfilesFunc = func(ctx context.Context, teamID *uint, profiles []*fleet.MDMAppleConfigProfile) error {
		return nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hids, tids []uint, puuids, uuids []string,
	) (updates fleet.MDMProfilesUpdates, err error) {
		return fleet.MDMProfilesUpdates{}, nil
	}
	ds.ListMDMConfigProfilesFunc = func(ctx context.Context, tid *uint, opt fleet.ListOptions) ([]*fleet.MDMConfigProfilePayload, *fleet.PaginationMetadata, error) {
		return nil, nil, nil
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
			[][]byte{[]byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
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
						<string>%s</string>
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
			</plist>`, mobileconfig.FleetFileVaultPayloadType))},
			mobileconfig.DiskEncryptionProfileRestrictionErrMsg,
		},
		{
			"uses a Fleet Variable",
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
						<key>Username</key>
						<string>$FLEET_VAR_HOST_END_USER_IDP_USERNAME</string>
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
			`profile variables are not supported by this deprecated endpoint`,
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
	svc, ctx, ds, svcOpts := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})

	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		return &fleet.Team{ID: 1, Name: name}, nil
	}
	ds.TeamWithExtrasFunc = func(ctx context.Context, id uint) (*fleet.Team, error) {
		return &fleet.Team{ID: id, Name: "team"}, nil
	}
	ds.BatchSetMDMAppleProfilesFunc = func(ctx context.Context, teamID *uint, profiles []*fleet.MDMAppleConfigProfile) error {
		return nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hids, tids []uint, profileUUIDs, uuids []string,
	) (updates fleet.MDMProfilesUpdates, err error) {
		return fleet.MDMProfilesUpdates{}, nil
	}
	ds.ListMDMConfigProfilesFunc = func(ctx context.Context, tid *uint, opt fleet.ListOptions) ([]*fleet.MDMConfigProfilePayload, *fleet.PaginationMetadata, error) {
		return nil, nil, nil
	}

	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})
	ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierPremium})

	// dry run doesn't call methods that save stuff in the db
	err := svc.BatchSetMDMAppleProfiles(ctx, nil, nil, [][]byte{}, true, false)
	require.NoError(t, err)
	require.False(t, ds.BatchSetMDMAppleProfilesFuncInvoked)
	require.False(t, ds.BulkSetPendingMDMHostProfilesFuncInvoked)
	require.False(t, svcOpts.ActivityMock.NewActivityFuncInvoked)

	// skipping bulk set only skips that method
	err = svc.BatchSetMDMAppleProfiles(ctx, nil, nil, [][]byte{}, false, true)
	require.NoError(t, err)
	require.True(t, ds.BatchSetMDMAppleProfilesFuncInvoked)
	require.False(t, ds.BulkSetPendingMDMHostProfilesFuncInvoked)
	require.True(t, svcOpts.ActivityMock.NewActivityFuncInvoked)
}

func TestUpdateMDMAppleSettings(t *testing.T) {
	svc, ctx, ds, _ := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})

	ds.TeamWithExtrasFunc = func(ctx context.Context, id uint) (*fleet.Team, error) {
		return &fleet.Team{ID: id, Name: "team"}, nil
	}
	ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		return team, nil
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
			fleet.ErrMissingLicense.Error(),
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
			fleet.ErrMissingLicense.Error(),
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
			fleet.ErrMissingLicense.Error(),
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

			err := svc.UpdateMDMDiskEncryption(ctx, tt.teamID, nil, nil)
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
		svc, ctx, ds, _ := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: tier})
		ds.TeamWithExtrasFunc = func(ctx context.Context, id uint) (*fleet.Team, error) {
			return &fleet.Team{ID: id, Name: "team"}, nil
		}
		ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
			return team, nil
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
	mdmStorage := &mdmmock.MDMAppleStore{}
	ds := new(mock.Store)
	kv := new(mock.AdvancedKVStore)
	pushFactory, _ := newMockAPNSPushProviderFactory()
	pusher := nanomdm_pushsvc.New(
		mdmStorage,
		mdmStorage,
		pushFactory,
		NewNanoMDMLogger(slog.New(slog.DiscardHandler)),
	)
	mdmConfig := config.MDMConfig{
		AppleSCEPCert: "./testdata/server.pem",
		AppleSCEPKey:  "./testdata/server.key",
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext,
	) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		_, pemCert, pemKey, err := mdmConfig.AppleSCEP()
		require.NoError(t, err)
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetCACert: {Value: pemCert},
			fleet.MDMAssetCAKey:  {Value: pemKey},
		}, nil
	}

	cmdr := apple_mdm.NewMDMAppleCommander(mdmStorage, pusher)
	hostUUID1, hostUUID2 := "ABC-DEF", "GHI-JKL"
	hostUUID1UserEnrollment := hostUUID1 + ":user"
	contents1 := []byte("test-content-1")
	expectedContents1 := []byte("test-content-1") // used for Fleet variable substitution
	contents2 := []byte("test-content-2")
	contents4 := []byte("test-content-4")
	contents5 := []byte("test-contents-5")
	contents7 := []byte("test-contents-7")

	p1, p2, p3, p4, p5, p6, p7 := "a"+uuid.NewString(), "a"+uuid.NewString(), "a"+uuid.NewString(), "a"+uuid.NewString(), "a"+uuid.NewString(), "a"+uuid.NewString(), "a"+uuid.NewString()
	baseProfilesToInstall := []*fleet.MDMAppleProfilePayload{
		{ProfileUUID: p1, ProfileIdentifier: "com.add.profile", HostUUID: hostUUID1, Scope: fleet.PayloadScopeSystem},
		{ProfileUUID: p2, ProfileIdentifier: "com.add.profile.two", HostUUID: hostUUID1, Scope: fleet.PayloadScopeSystem},
		{ProfileUUID: p2, ProfileIdentifier: "com.add.profile.two", HostUUID: hostUUID2, Scope: fleet.PayloadScopeSystem},
		{ProfileUUID: p4, ProfileIdentifier: "com.add.profile.four", HostUUID: hostUUID2, Scope: fleet.PayloadScopeSystem},
		{ProfileUUID: p5, ProfileIdentifier: "com.add.profile.five", HostUUID: hostUUID1, Scope: fleet.PayloadScopeUser},
		{ProfileUUID: p5, ProfileIdentifier: "com.add.profile.five", HostUUID: hostUUID2, Scope: fleet.PayloadScopeUser},
		{ProfileUUID: p7, ProfileIdentifier: "com.add.profile.seven", HostUUID: hostUUID1, Scope: fleet.PayloadScopeUser},
		{ProfileUUID: p7, ProfileIdentifier: "com.add.profile.seven", HostUUID: hostUUID2, Scope: fleet.PayloadScopeUser, HostPlatform: "ios"},
	}
	baseProfilesToRemove := []*fleet.MDMAppleProfilePayload{
		{ProfileUUID: p3, ProfileIdentifier: "com.remove.profile", HostUUID: hostUUID1, Scope: fleet.PayloadScopeSystem},
		{ProfileUUID: p3, ProfileIdentifier: "com.remove.profile", HostUUID: hostUUID2, Scope: fleet.PayloadScopeSystem},
		{ProfileUUID: p6, ProfileIdentifier: "com.remove.profile.six", HostUUID: hostUUID1, Scope: fleet.PayloadScopeUser},
		{ProfileUUID: p6, ProfileIdentifier: "com.remove.profile.six", HostUUID: hostUUID2, Scope: fleet.PayloadScopeUser},
	}
	ds.ListMDMAppleProfilesToInstallAndRemoveFunc = func(ctx context.Context) ([]*fleet.MDMAppleProfilePayload, []*fleet.MDMAppleProfilePayload, error) {
		return baseProfilesToInstall, baseProfilesToRemove, nil
	}

	kv.MGetFunc = func(ctx context.Context, keys []string) (map[string]*string, error) {
		return map[string]*string{}, nil
	}

	ds.GetMDMAppleProfilesContentsFunc = func(ctx context.Context, profileUUIDs []string) (map[string]mobileconfig.Mobileconfig, error) {
		require.ElementsMatch(t, []string{p1, p2, p4, p5, p7}, profileUUIDs)
		// only those profiles that are to be installed
		return map[string]mobileconfig.Mobileconfig{
			p1: contents1,
			p2: contents2,
			p4: contents4,
			p5: contents5,
			p7: contents7,
		}, nil
	}

	ds.BulkDeleteMDMAppleHostsConfigProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAppleProfilePayload) error {
		require.ElementsMatch(t, payload, []*fleet.MDMAppleProfilePayload{{ProfileUUID: p6, ProfileIdentifier: "com.remove.profile.six", HostUUID: hostUUID2, Scope: fleet.PayloadScopeUser}})
		return nil
	}

	ds.GetNanoMDMUserEnrollmentFunc = func(ctx context.Context, hostUUID string) (*fleet.NanoEnrollment, error) {
		if hostUUID == hostUUID1 {
			return &fleet.NanoEnrollment{
				ID:               hostUUID1UserEnrollment,
				DeviceID:         hostUUID1,
				Type:             "User",
				Enabled:          true,
				TokenUpdateTally: 1,
			}, nil
		}
		// hostUUID2 has no user enrollment
		assert.Equal(t, hostUUID2, hostUUID)
		return nil, nil
	}

	mdmStorage.BulkDeleteHostUserCommandsWithoutResultsFunc = func(ctx context.Context, commandToIDs map[string][]string) error {
		require.Empty(t, commandToIDs)
		return nil
	}

	var enqueueFailForOp fleet.MDMOperationType
	var mu sync.Mutex
	mdmStorage.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error, error) {
		require.NotNil(t, cmd)
		require.NotEmpty(t, cmd.CommandUUID)

		switch cmd.Command.Command.RequestType {
		case "InstallProfile":

			var fullCmd micromdm.CommandPayload
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
			// the p7 library doesn't support concurrent calls to Parse
			mu.Lock()
			pk7, err := pkcs7.Parse(fullCmd.Command.InstallProfile.Payload)
			mu.Unlock()
			require.NoError(t, err)

			if !bytes.Equal(pk7.Content, expectedContents1) && !bytes.Equal(pk7.Content, contents2) &&
				!bytes.Equal(pk7.Content, contents4) && !bytes.Equal(pk7.Content, contents5) && !bytes.Equal(pk7.Content, contents7) {
				require.Failf(t, "profile contents don't match", "expected to contain %s, %s or %s but got %s",
					expectedContents1, contents2, contents4, pk7.Content)
			}

			// may be called for a single host or both
			if len(id) == 2 {
				if bytes.Equal(pk7.Content, contents5) || bytes.Equal(pk7.Content, contents7) {
					require.ElementsMatch(t, []string{hostUUID1UserEnrollment, hostUUID2}, id)
				} else {
					require.ElementsMatch(t, []string{hostUUID1, hostUUID2}, id)
				}
			} else {
				require.Len(t, id, 1)
			}

		case "RemoveProfile":
			if len(id) == 1 {
				require.Equal(t, hostUUID1UserEnrollment, id[0])
			} else {
				require.ElementsMatch(t, []string{hostUUID1, hostUUID2}, id)
			}
			require.Contains(t, string(cmd.Raw), "com.remove.profile")
		}
		switch {
		case enqueueFailForOp == fleet.MDMOperationTypeInstall && cmd.Command.Command.RequestType == "InstallProfile":
			return nil, errors.New("enqueue error")
		case enqueueFailForOp == fleet.MDMOperationTypeRemove && cmd.Command.Command.RequestType == "RemoveProfile":
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
	mdmStorage.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext,
	) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		certPEM, err := os.ReadFile("./testdata/server.pem")
		require.NoError(t, err)
		keyPEM, err := os.ReadFile("./testdata/server.key")
		require.NoError(t, err)
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetCACert: {Value: certPEM},
			fleet.MDMAssetCAKey:  {Value: keyPEM},
		}, nil
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
			// clear the command UUID (in a copy so that it does not affect the
			// pointed-to struct) from the payload for the subsequent checks
			copyp := *p
			copyp.CommandUUID = ""
			copies[i] = &copyp

			// Host with no user enrollment, so install fails
			if p.HostUUID == hostUUID2 && (p.ProfileUUID == p5 || p.ProfileUUID == p7) {
				continue
			}

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

		}

		require.ElementsMatch(t, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
			{
				ProfileUUID:       p1,
				ProfileIdentifier: "com.add.profile",
				HostUUID:          hostUUID1,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeSystem,
			},
			{
				ProfileUUID:       p2,
				ProfileIdentifier: "com.add.profile.two",
				HostUUID:          hostUUID1,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeSystem,
			},
			{
				ProfileUUID:       p2,
				ProfileIdentifier: "com.add.profile.two",
				HostUUID:          hostUUID2,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeSystem,
			},
			{
				ProfileUUID:       p3,
				ProfileIdentifier: "com.remove.profile",
				HostUUID:          hostUUID1,
				OperationType:     fleet.MDMOperationTypeRemove,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeSystem,
			},
			{
				ProfileUUID:       p3,
				ProfileIdentifier: "com.remove.profile",
				HostUUID:          hostUUID2,
				OperationType:     fleet.MDMOperationTypeRemove,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeSystem,
			},
			{
				ProfileUUID:       p4,
				ProfileIdentifier: "com.add.profile.four",
				HostUUID:          hostUUID2,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeSystem,
			},
			// This host has a user enrollment so the profile is sent to it
			{
				ProfileUUID:       p5,
				ProfileIdentifier: "com.add.profile.five",
				HostUUID:          hostUUID1,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeUser,
			},
			// This host has no user enrollment so the profile is errored
			{
				ProfileUUID:       p5,
				ProfileIdentifier: "com.add.profile.five",
				HostUUID:          hostUUID2,
				OperationType:     fleet.MDMOperationTypeInstall,
				Detail:            "This setting couldn't be enforced because the user channel doesn't exist for this host. Currently, Fleet creates the user channel for hosts that automatically enroll.",
				Status:            &fleet.MDMDeliveryFailed,
				Scope:             fleet.PayloadScopeUser,
			},
			// This host has a user enrollment so the profile is removed from it
			{
				ProfileUUID:       p6,
				ProfileIdentifier: "com.remove.profile.six",
				HostUUID:          hostUUID1,
				OperationType:     fleet.MDMOperationTypeRemove,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeUser,
			},
			// Note that host2 has no user enrollment so the profile is not marked for removal
			// from it
			{
				ProfileUUID:       p7,
				ProfileIdentifier: "com.add.profile.seven",
				HostUUID:          hostUUID1,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeUser,
			},
			{
				ProfileUUID:       p7,
				ProfileIdentifier: "com.add.profile.seven",
				HostUUID:          hostUUID2,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryFailed,
				Detail:            "This setting couldn't be enforced because the user channel isn't available on iOS and iPadOS hosts.",
				Scope:             fleet.PayloadScopeUser,
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
	ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
		return &fleet.GroupedCertificateAuthorities{}, nil
	}

	ds.BulkUpsertMDMAppleConfigProfilesFunc = func(ctx context.Context, p []*fleet.MDMAppleConfigProfile) error {
		return nil
	}

	ds.AggregateEnrollSecretPerTeamFunc = func(ctx context.Context) ([]*fleet.EnrollSecret, error) {
		return []*fleet.EnrollSecret{}, nil
	}

	checkAndReset := func(t *testing.T, want bool, invoked *bool) {
		if want {
			assert.True(t, *invoked)
		} else {
			assert.False(t, *invoked)
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
		err := ReconcileAppleProfiles(ctx, ds, cmdr, kv, slog.New(slog.DiscardHandler), 0)
		require.NoError(t, err)
		require.Equal(t, 1, failedCount)
		checkAndReset(t, true, &ds.ListMDMAppleProfilesToInstallAndRemoveFuncInvoked)
		checkAndReset(t, true, &ds.GetMDMAppleProfilesContentsFuncInvoked)
		checkAndReset(t, true, &ds.BulkUpsertMDMAppleHostProfilesFuncInvoked)
		checkAndReset(t, true, &ds.GetNanoMDMUserEnrollmentFuncInvoked)
	})

	t.Run("fail enqueue remove ops", func(t *testing.T) {
		var failedCount int
		failedCall = false
		failedCheck = func(payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) {
			failedCount++
			require.Len(t, payload, 3) // the 3 remove ops
			require.ElementsMatch(t, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
				{
					ProfileUUID:       p3,
					ProfileIdentifier: "com.remove.profile",
					HostUUID:          hostUUID1,
					OperationType:     fleet.MDMOperationTypeRemove,
					Status:            nil,
					CommandUUID:       "",
					Scope:             fleet.PayloadScopeSystem,
				},
				{
					ProfileUUID:       p3,
					ProfileIdentifier: "com.remove.profile",
					HostUUID:          hostUUID2,
					OperationType:     fleet.MDMOperationTypeRemove,
					Status:            nil,
					CommandUUID:       "",
					Scope:             fleet.PayloadScopeSystem,
				},
				{
					ProfileUUID:       p6,
					ProfileIdentifier: "com.remove.profile.six",
					HostUUID:          hostUUID1,
					OperationType:     fleet.MDMOperationTypeRemove,
					Status:            nil,
					CommandUUID:       "",
					Scope:             fleet.PayloadScopeUser,
				},
			}, payload)
		}

		enqueueFailForOp = fleet.MDMOperationTypeRemove
		err := ReconcileAppleProfiles(ctx, ds, cmdr, kv, slog.New(slog.DiscardHandler), 0)
		require.NoError(t, err)
		require.Equal(t, 1, failedCount)
		checkAndReset(t, true, &ds.ListMDMAppleProfilesToInstallAndRemoveFuncInvoked)
		checkAndReset(t, true, &ds.GetMDMAppleProfilesContentsFuncInvoked)
		checkAndReset(t, true, &ds.BulkUpsertMDMAppleHostProfilesFuncInvoked)
		checkAndReset(t, true, &ds.GetNanoMDMUserEnrollmentFuncInvoked)
	})

	t.Run("fail enqueue install ops", func(t *testing.T) {
		var failedCount int
		failedCall = false
		failedCheck = func(payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) {
			failedCount++

			require.Len(t, payload, 6) // the 6 install ops
			require.ElementsMatch(t, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
				{
					ProfileUUID:       p1,
					ProfileIdentifier: "com.add.profile",
					HostUUID:          hostUUID1, OperationType: fleet.MDMOperationTypeInstall,
					Status:      nil,
					CommandUUID: "",
					Scope:       fleet.PayloadScopeSystem,
				},
				{
					ProfileUUID:       p2,
					ProfileIdentifier: "com.add.profile.two",
					HostUUID:          hostUUID1, OperationType: fleet.MDMOperationTypeInstall,
					Status:      nil,
					CommandUUID: "",
					Scope:       fleet.PayloadScopeSystem,
				},
				{
					ProfileUUID:       p2,
					ProfileIdentifier: "com.add.profile.two",
					HostUUID:          hostUUID2,
					OperationType:     fleet.MDMOperationTypeInstall,
					Status:            nil,
					CommandUUID:       "",
					Scope:             fleet.PayloadScopeSystem,
				},
				{
					ProfileUUID:       p4,
					ProfileIdentifier: "com.add.profile.four",
					HostUUID:          hostUUID2,
					OperationType:     fleet.MDMOperationTypeInstall,
					Status:            nil,
					CommandUUID:       "",
					Scope:             fleet.PayloadScopeSystem,
				},
				{
					ProfileUUID:       p5,
					ProfileIdentifier: "com.add.profile.five",
					HostUUID:          hostUUID1,
					OperationType:     fleet.MDMOperationTypeInstall,
					Status:            nil,
					CommandUUID:       "",
					Scope:             fleet.PayloadScopeUser,
				},
				{
					ProfileUUID:       p7,
					ProfileIdentifier: "com.add.profile.seven",
					HostUUID:          hostUUID1,
					OperationType:     fleet.MDMOperationTypeInstall,
					Status:            nil,
					CommandUUID:       "",
					Scope:             fleet.PayloadScopeUser,
				},
			}, payload)
		}

		enqueueFailForOp = fleet.MDMOperationTypeInstall
		err := ReconcileAppleProfiles(ctx, ds, cmdr, kv, slog.New(slog.DiscardHandler), 0)
		require.NoError(t, err)
		require.Equal(t, 1, failedCount)
		checkAndReset(t, true, &ds.ListMDMAppleProfilesToInstallAndRemoveFuncInvoked)
		checkAndReset(t, true, &ds.GetMDMAppleProfilesContentsFuncInvoked)
		checkAndReset(t, true, &ds.BulkUpsertMDMAppleHostProfilesFuncInvoked)
		checkAndReset(t, true, &ds.GetNanoMDMUserEnrollmentFuncInvoked)
	})

	// Zero profiles to remove
	ds.ListMDMAppleProfilesToInstallAndRemoveFunc = func(ctx context.Context) ([]*fleet.MDMAppleProfilePayload, []*fleet.MDMAppleProfilePayload, error) {
		return baseProfilesToInstall, nil, nil
	}
	ds.BulkDeleteMDMAppleHostsConfigProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAppleProfilePayload) error {
		require.Empty(t, payload)
		return nil
	}
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
			// clear the command UUID (in a copy so that it does not affect the
			// pointed-to struct) from the payload for the subsequent checks
			copyp := *p
			copyp.CommandUUID = ""
			copies[i] = &copyp

			// Host with no user enrollment, so install fails
			if p.HostUUID == hostUUID2 && (p.ProfileUUID == p5 || p.ProfileUUID == p7) {
				continue
			}

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
		}

		require.ElementsMatch(t, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
			{
				ProfileUUID:       p1,
				ProfileIdentifier: "com.add.profile",
				HostUUID:          hostUUID1,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeSystem,
			},
			{
				ProfileUUID:       p2,
				ProfileIdentifier: "com.add.profile.two",
				HostUUID:          hostUUID1,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeSystem,
			},
			{
				ProfileUUID:       p2,
				ProfileIdentifier: "com.add.profile.two",
				HostUUID:          hostUUID2,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeSystem,
			},
			{
				ProfileUUID:       p4,
				ProfileIdentifier: "com.add.profile.four",
				HostUUID:          hostUUID2,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeSystem,
			},
			{
				ProfileUUID:       p5,
				ProfileIdentifier: "com.add.profile.five",
				HostUUID:          hostUUID1,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeUser,
			},
			// This host has no user enrollment so the profile is sent to the device enrollment
			{
				ProfileUUID:       p5,
				ProfileIdentifier: "com.add.profile.five",
				HostUUID:          hostUUID2,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryFailed,
				Detail:            "This setting couldn't be enforced because the user channel doesn't exist for this host. Currently, Fleet creates the user channel for hosts that automatically enroll.",
				Scope:             fleet.PayloadScopeUser,
			},
			{
				ProfileUUID:       p7,
				ProfileIdentifier: "com.add.profile.seven",
				HostUUID:          hostUUID1,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeUser,
			},
			{
				ProfileUUID:       p7,
				ProfileIdentifier: "com.add.profile.seven",
				HostUUID:          hostUUID2,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryFailed,
				Detail:            "This setting couldn't be enforced because the user channel isn't available on iOS and iPadOS hosts.",
				Scope:             fleet.PayloadScopeUser,
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

	// TODO(hca): Mock this to enable NDES?
	// ds.GetAllCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) ([]*fleet.CertificateAuthority, error) {
	// 	return []*fleet.CertificateAuthority{}, nil
	// }

	ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierPremium})
	ds.BulkUpsertMDMManagedCertificatesFunc = func(ctx context.Context, payload []*fleet.MDMManagedCertificate) error {
		assert.Empty(t, payload)
		return nil
	}

	// TODO(hca): ask Magnus where/how new tests cover the CA portion of this test
	t.Run("replace $FLEET_VAR_"+string(fleet.FleetVarNDESSCEPProxyURL), func(t *testing.T) {
		var upsertCount int
		failedCall = false
		failedCheck = func(payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) {
			upsertCount++
			if upsertCount == 1 {
				// We update the profile with a new command UUID
				assert.Len(t, payload, 1, "at upsertCount %d", upsertCount)
			} else {
				assert.Len(t, payload, 0, "at upsertCount %d", upsertCount)
			}
		}
		enqueueFailForOp = ""
		newContents := "$FLEET_VAR_" + fleet.FleetVarNDESSCEPProxyURL
		originalContents1 := contents1
		originalExpectedContents1 := expectedContents1
		contents1 = []byte(newContents)
		expectedContents1 = []byte("https://test.example.com" + apple_mdm.SCEPProxyPath + url.QueryEscape(fmt.Sprintf("%s,%s,NDES", hostUUID1, p1)))
		t.Cleanup(func() {
			contents1 = originalContents1
			expectedContents1 = originalExpectedContents1
		})
		err := ReconcileAppleProfiles(ctx, ds, cmdr, kv, slog.New(slog.DiscardHandler), 0)
		require.NoError(t, err)
		assert.Equal(t, 2, upsertCount)
		// checkAndReset(t, true, &ds.GetAllCertificateAuthoritiesFuncInvoked)
		checkAndReset(t, true, &ds.ListMDMAppleProfilesToInstallAndRemoveFuncInvoked)
		checkAndReset(t, true, &ds.GetMDMAppleProfilesContentsFuncInvoked)
		checkAndReset(t, true, &ds.BulkUpsertMDMAppleHostProfilesFuncInvoked)
		checkAndReset(t, true, &ds.GetNanoMDMUserEnrollmentFuncInvoked)
	})

	// TODO(hca): ask Magnus where/how new tests cover the CA portion of this test
	t.Run("preprocessor fails on $FLEET_VAR_"+string(fleet.FleetVarHostEndUserEmailIDP), func(t *testing.T) {
		var failedCount int
		failedCall = false
		failedCheck = func(payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) {
			failedCount++
			require.Len(t, payload, 8)
		}
		enqueueFailForOp = ""
		newContents := "$FLEET_VAR_" + fleet.FleetVarHostEndUserEmailIDP
		originalContents1 := contents1
		contents1 = []byte(newContents)
		t.Cleanup(func() {
			contents1 = originalContents1
		})
		ds.GetHostEmailsFunc = func(ctx context.Context, hostUUID string, source string) ([]string, error) {
			return nil, errors.New("GetHostEmailsFuncError")
		}
		err := ReconcileAppleProfiles(ctx, ds, cmdr, kv, slog.New(slog.Default().Handler()), 0)
		assert.ErrorContains(t, err, "GetHostEmailsFuncError")
		// checkAndReset(t, true, &ds.GetAllCertificateAuthoritiesFuncInvoked)
		checkAndReset(t, true, &ds.ListMDMAppleProfilesToInstallAndRemoveFuncInvoked)
		checkAndReset(t, true, &ds.GetMDMAppleProfilesContentsFuncInvoked)
		checkAndReset(t, true, &ds.BulkUpsertMDMAppleHostProfilesFuncInvoked)
		checkAndReset(t, true, &ds.GetNanoMDMUserEnrollmentFuncInvoked)
	})

	// TODO(hca): ask Magnus where/how new tests cover the CA portion of this test
	t.Run("bad $FLEET_VAR", func(t *testing.T) {
		var failedCount int
		failedCall = false
		var hostUUIDs []string
		failedCheck = func(payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) {
			if len(payload) > 0 {
				failedCount++
			}
			for _, p := range payload {
				assert.Equal(t, fleet.MDMDeliveryFailed, *p.Status)
				assert.Contains(t, p.Detail, "FLEET_VAR_BOZO")
				for i, hu := range hostUUIDs {
					if hu == p.HostUUID {
						// remove element
						hostUUIDs = append(hostUUIDs[:i], hostUUIDs[i+1:]...)
						break
					}
				}
			}
		}
		enqueueFailForOp = ""

		// All profiles will have bad contents
		badContents := "bad-content: $FLEET_VAR_BOZO"
		originalContents1 := contents1
		originalContents2 := contents2
		originalContents4 := contents4
		originalContents5 := contents5
		originalContents7 := contents7
		contents1 = []byte(badContents)
		contents2 = []byte(badContents)
		contents4 = []byte(badContents)
		contents5 = []byte(badContents)
		contents7 = []byte(badContents)
		t.Cleanup(func() {
			contents1 = originalContents1
			contents2 = originalContents2
			contents4 = originalContents4
			contents5 = originalContents5
			contents7 = originalContents7
		})

		profilesToInstall, _, _ := ds.ListMDMAppleProfilesToInstallAndRemoveFunc(ctx)
		hostUUIDs = make([]string, 0, len(profilesToInstall))
		for _, p := range profilesToInstall {
			// This host will error before this point - should not be updated by the variable failure
			if p.HostUUID == hostUUID2 && (p.ProfileUUID == p5 || p.ProfileUUID == p7) {
				continue
			}
			hostUUIDs = append(hostUUIDs, p.HostUUID)
		}

		err := ReconcileAppleProfiles(ctx, ds, cmdr, kv, slog.New(slog.DiscardHandler), 0)
		require.NoError(t, err)
		assert.Empty(t, hostUUIDs, "all host+profile combinations should be updated")
		require.Equal(t, 5, failedCount, "number of profiles with bad content")
		// checkAndReset(t, true, &ds.GetAllCertificateAuthoritiesFuncInvoked)
		checkAndReset(t, true, &ds.ListMDMAppleProfilesToInstallAndRemoveFuncInvoked)
		checkAndReset(t, true, &ds.GetMDMAppleProfilesContentsFuncInvoked)
		checkAndReset(t, true, &ds.BulkUpsertMDMAppleHostProfilesFuncInvoked)
		checkAndReset(t, true, &ds.GetNanoMDMUserEnrollmentFuncInvoked)
		// Check that individual updates were not done (bulk update should be done)
		checkAndReset(t, false, &ds.UpdateOrDeleteHostMDMAppleProfileFuncInvoked)
	})
}

func TestReconcileAppleProfilesCAThrottle(t *testing.T) {
	ctx := t.Context()
	mdmStorage := &mdmmock.MDMAppleStore{}
	ds := new(mock.Store)
	kv := new(mock.AdvancedKVStore)
	pushFactory, _ := newMockAPNSPushProviderFactory()
	pusher := nanomdm_pushsvc.New(
		mdmStorage,
		mdmStorage,
		pushFactory,
		NewNanoMDMLogger(slog.New(slog.DiscardHandler)),
	)
	mdmConfig := config.MDMConfig{
		AppleSCEPCert: "./testdata/server.pem",
		AppleSCEPKey:  "./testdata/server.key",
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext,
	) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		_, pemCert, pemKey, err := mdmConfig.AppleSCEP()
		require.NoError(t, err)
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetCACert: {Value: pemCert},
			fleet.MDMAssetCAKey:  {Value: pemKey},
		}, nil
	}

	cmdr := apple_mdm.NewMDMAppleCommander(mdmStorage, pusher)
	hostUUIDs := []string{"host-1", "host-2", "host-3", "host-4", "host-5"}

	caProfileUUID := "a" + uuid.NewString()
	nonCAProfileUUID := "a" + uuid.NewString()
	caContent := []byte("profile with $FLEET_VAR_NDES_SCEP_CHALLENGE variable")
	nonCAContent := []byte("regular profile content")

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}, nil
	}

	// Build toInstall: CA profile for 5 hosts + non-CA profile for 5 hosts
	var profilesToInstall []*fleet.MDMAppleProfilePayload
	for _, h := range hostUUIDs {
		profilesToInstall = append(profilesToInstall,
			&fleet.MDMAppleProfilePayload{ProfileUUID: caProfileUUID, ProfileIdentifier: "com.ca.profile", ProfileName: "CA Profile", HostUUID: h, Scope: fleet.PayloadScopeSystem},
			&fleet.MDMAppleProfilePayload{ProfileUUID: nonCAProfileUUID, ProfileIdentifier: "com.regular.profile", ProfileName: "Regular Profile", HostUUID: h, Scope: fleet.PayloadScopeSystem},
		)
	}

	ds.ListMDMAppleProfilesToInstallAndRemoveFunc = func(ctx context.Context) ([]*fleet.MDMAppleProfilePayload, []*fleet.MDMAppleProfilePayload, error) {
		return profilesToInstall, nil, nil
	}

	ds.GetMDMAppleProfilesContentsFunc = func(ctx context.Context, profileUUIDs []string) (map[string]mobileconfig.Mobileconfig, error) {
		return map[string]mobileconfig.Mobileconfig{
			caProfileUUID:    caContent,
			nonCAProfileUUID: nonCAContent,
		}, nil
	}

	kv.MGetFunc = func(ctx context.Context, keys []string) (map[string]*string, error) {
		return make(map[string]*string), nil
	}

	ds.BulkDeleteMDMAppleHostsConfigProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAppleProfilePayload) error {
		return nil
	}

	ds.GetNanoMDMUserEnrollmentFunc = func(ctx context.Context, hostUUID string) (*fleet.NanoEnrollment, error) {
		return nil, nil
	}

	ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, allCAs bool) (*fleet.GroupedCertificateAuthorities, error) {
		return &fleet.GroupedCertificateAuthorities{}, nil
	}

	mdmStorage.BulkDeleteHostUserCommandsWithoutResultsFunc = func(ctx context.Context, commandToIDs map[string][]string) error {
		return nil
	}

	mdmStorage.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error, error) {
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
	mdmStorage.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext,
	) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		certPEM, err := os.ReadFile("./testdata/server.pem")
		require.NoError(t, err)
		keyPEM, err := os.ReadFile("./testdata/server.key")
		require.NoError(t, err)
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetCACert: {Value: certPEM},
			fleet.MDMAssetCAKey:  {Value: keyPEM},
		}, nil
	}

	ds.AggregateEnrollSecretPerTeamFunc = func(ctx context.Context) ([]*fleet.EnrollSecret, error) {
		return []*fleet.EnrollSecret{}, nil
	}

	ds.BulkUpsertMDMAppleConfigProfilesFunc = func(ctx context.Context, p []*fleet.MDMAppleConfigProfile) error {
		return nil
	}

	// Track upserted host profiles to verify throttling.
	// The first BulkUpsert call contains the profiles that will be sent;
	// subsequent calls are for reverting failures (empty).
	var upsertedProfiles []*fleet.MDMAppleBulkUpsertHostProfilePayload
	var bulkUpsertCallCount int
	ds.BulkUpsertMDMAppleHostProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) error {
		bulkUpsertCallCount++
		if bulkUpsertCallCount == 1 {
			upsertedProfiles = payload
		}
		return nil
	}

	t.Run("limit=0 sends all profiles", func(t *testing.T) {
		upsertedProfiles = nil
		bulkUpsertCallCount = 0
		err := ReconcileAppleProfiles(ctx, ds, cmdr, kv, slog.New(slog.DiscardHandler), 0)
		require.NoError(t, err)

		// All 10 host-profile pairs should be upserted (5 CA + 5 non-CA)
		var caCount, nonCACount int
		for _, p := range upsertedProfiles {
			if p.ProfileUUID == caProfileUUID {
				caCount++
			} else if p.ProfileUUID == nonCAProfileUUID {
				nonCACount++
			}
		}
		assert.Equal(t, 5, caCount, "all CA host-profile pairs should be sent when limit=0")
		assert.Equal(t, 5, nonCACount, "all non-CA host-profile pairs should be sent")
	})

	t.Run("limit=2 throttles CA profiles only", func(t *testing.T) {
		upsertedProfiles = nil
		bulkUpsertCallCount = 0
		err := ReconcileAppleProfiles(ctx, ds, cmdr, kv, slog.New(slog.DiscardHandler), 2)
		require.NoError(t, err)

		// Should have 2 CA + 5 non-CA = 7 host-profile pairs upserted
		var caCount, nonCACount int
		for _, p := range upsertedProfiles {
			if p.ProfileUUID == caProfileUUID {
				caCount++
			} else if p.ProfileUUID == nonCAProfileUUID {
				nonCACount++
			}
		}
		assert.Equal(t, 2, caCount, "only 2 CA host-profile pairs should be sent when limit=2")
		assert.Equal(t, 5, nonCACount, "all non-CA host-profile pairs should still be sent")
	})

	t.Run("recently enrolled hosts bypass throttle", func(t *testing.T) {
		upsertedProfiles = nil
		bulkUpsertCallCount = 0

		recentEnrollTime := time.Now().Add(-30 * time.Minute)
		var recentProfilesToInstall []*fleet.MDMAppleProfilePayload
		for _, h := range hostUUIDs {
			recentProfilesToInstall = append(recentProfilesToInstall,
				&fleet.MDMAppleProfilePayload{
					ProfileUUID: caProfileUUID, ProfileIdentifier: "com.ca.profile", ProfileName: "CA Profile",
					HostUUID: h, Scope: fleet.PayloadScopeSystem, DeviceEnrolledAt: &recentEnrollTime,
				},
				&fleet.MDMAppleProfilePayload{
					ProfileUUID: nonCAProfileUUID, ProfileIdentifier: "com.regular.profile", ProfileName: "Regular Profile",
					HostUUID: h, Scope: fleet.PayloadScopeSystem, DeviceEnrolledAt: &recentEnrollTime,
				},
			)
		}
		ds.ListMDMAppleProfilesToInstallAndRemoveFunc = func(ctx context.Context) ([]*fleet.MDMAppleProfilePayload, []*fleet.MDMAppleProfilePayload, error) {
			return recentProfilesToInstall, nil, nil
		}

		err := ReconcileAppleProfiles(ctx, ds, cmdr, kv, slog.New(slog.DiscardHandler), 2)
		require.NoError(t, err)

		var caCount, nonCACount int
		for _, p := range upsertedProfiles {
			if p.ProfileUUID == caProfileUUID {
				caCount++
			} else if p.ProfileUUID == nonCAProfileUUID {
				nonCACount++
			}
		}
		assert.Equal(t, 5, caCount, "all CA host-profile pairs should be sent for recently enrolled hosts")
		assert.Equal(t, 5, nonCACount, "all non-CA host-profile pairs should be sent")

		// Restore original profilesToInstall for subsequent subtests.
		ds.ListMDMAppleProfilesToInstallAndRemoveFunc = func(ctx context.Context) ([]*fleet.MDMAppleProfilePayload, []*fleet.MDMAppleProfilePayload, error) {
			return profilesToInstall, nil, nil
		}
	})

	t.Run("removals are not throttled", func(t *testing.T) {
		upsertedProfiles = nil
		bulkUpsertCallCount = 0

		var profilesToRemove []*fleet.MDMAppleProfilePayload
		for _, h := range hostUUIDs {
			profilesToRemove = append(profilesToRemove,
				&fleet.MDMAppleProfilePayload{
					ProfileUUID: caProfileUUID, ProfileIdentifier: "com.ca.profile", ProfileName: "CA Profile",
					HostUUID: h, Scope: fleet.PayloadScopeSystem, OperationType: fleet.MDMOperationTypeInstall,
					Status: &fleet.MDMDeliveryVerifying, CommandUUID: uuid.NewString(),
				},
			)
		}
		ds.ListMDMAppleProfilesToInstallAndRemoveFunc = func(ctx context.Context) ([]*fleet.MDMAppleProfilePayload, []*fleet.MDMAppleProfilePayload, error) {
			return nil, profilesToRemove, nil
		}

		err := ReconcileAppleProfiles(ctx, ds, cmdr, kv, slog.New(slog.DiscardHandler), 2)
		require.NoError(t, err)

		var removeCount int
		for _, p := range upsertedProfiles {
			if p.ProfileUUID == caProfileUUID && p.OperationType == fleet.MDMOperationTypeRemove {
				removeCount++
			}
		}
		assert.Equal(t, 5, removeCount, "all CA profile removals should proceed regardless of throttle limit")

		// Restore original profilesToInstall for subsequent subtests.
		ds.ListMDMAppleProfilesToInstallAndRemoveFunc = func(ctx context.Context) ([]*fleet.MDMAppleProfilePayload, []*fleet.MDMAppleProfilePayload, error) {
			return profilesToInstall, nil, nil
		}
	})
}

func TestReconcileAppleProfilesSkipsHostBeingProcessed(t *testing.T) {
	ctx := t.Context()
	mdmStorage := &mdmmock.MDMAppleStore{}
	ds := new(mock.Store)
	kv := new(mock.AdvancedKVStore)
	pushFactory, _ := newMockAPNSPushProviderFactory()
	pusher := nanomdm_pushsvc.New(
		mdmStorage,
		mdmStorage,
		pushFactory,
		NewNanoMDMLogger(slog.New(slog.DiscardHandler)),
	)
	mdmConfig := config.MDMConfig{
		AppleSCEPCert: "./testdata/server.pem",
		AppleSCEPKey:  "./testdata/server.key",
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext,
	) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		_, pemCert, pemKey, err := mdmConfig.AppleSCEP()
		require.NoError(t, err)
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetCACert: {Value: pemCert},
			fleet.MDMAssetCAKey:  {Value: pemKey},
		}, nil
	}

	cmdr := apple_mdm.NewMDMAppleCommander(mdmStorage, pusher)

	profileUUID := "a" + uuid.NewString()
	profileContent := []byte("regular profile content")
	blockedHostUUID := "host-blocked"
	nonSetupHostUUID := "host-non-setup"

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}, nil
	}
	ds.ListMDMAppleProfilesToInstallAndRemoveFunc = func(ctx context.Context) ([]*fleet.MDMAppleProfilePayload, []*fleet.MDMAppleProfilePayload, error) {
		return []*fleet.MDMAppleProfilePayload{
			{ProfileUUID: profileUUID, ProfileIdentifier: "com.test.profile", ProfileName: "Test Profile", HostUUID: blockedHostUUID, Scope: fleet.PayloadScopeSystem},
			{ProfileUUID: profileUUID, ProfileIdentifier: "com.test.profile", ProfileName: "Test Profile", HostUUID: nonSetupHostUUID, Scope: fleet.PayloadScopeSystem},
		}, nil, nil
	}
	ds.GetMDMAppleProfilesContentsFunc = func(ctx context.Context, profileUUIDs []string) (map[string]mobileconfig.Mobileconfig, error) {
		return map[string]mobileconfig.Mobileconfig{profileUUID: profileContent}, nil
	}
	ds.BulkDeleteMDMAppleHostsConfigProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAppleProfilePayload) error {
		return nil
	}
	ds.GetNanoMDMUserEnrollmentFunc = func(ctx context.Context, hostUUID string) (*fleet.NanoEnrollment, error) {
		return nil, nil
	}
	ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, allCAs bool) (*fleet.GroupedCertificateAuthorities, error) {
		return &fleet.GroupedCertificateAuthorities{}, nil
	}
	ds.AggregateEnrollSecretPerTeamFunc = func(ctx context.Context) ([]*fleet.EnrollSecret, error) {
		return []*fleet.EnrollSecret{}, nil
	}
	ds.BulkUpsertMDMAppleConfigProfilesFunc = func(ctx context.Context, p []*fleet.MDMAppleConfigProfile) error {
		return nil
	}
	mdmStorage.BulkDeleteHostUserCommandsWithoutResultsFunc = func(ctx context.Context, commandToIDs map[string][]string) error {
		return nil
	}
	mdmStorage.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error, error) {
		return nil, nil
	}
	mdmStorage.RetrievePushInfoFunc = func(ctx context.Context, tokens []string) (map[string]*mdm.Push, error) {
		res := make(map[string]*mdm.Push, len(tokens))
		for _, t := range tokens {
			res[t] = &mdm.Push{PushMagic: "", Token: []byte(t), Topic: ""}
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
	mdmStorage.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext,
	) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		certPEM, err := os.ReadFile("./testdata/server.pem")
		require.NoError(t, err)
		keyPEM, err := os.ReadFile("./testdata/server.key")
		require.NoError(t, err)
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetCACert: {Value: certPEM},
			fleet.MDMAssetCAKey:  {Value: keyPEM},
		}, nil
	}

	// Track what gets upserted and which hosts get commands enqueued
	var upsertedProfiles []*fleet.MDMAppleBulkUpsertHostProfilePayload
	var bulkUpsertCallCount int
	ds.BulkUpsertMDMAppleHostProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) error {
		bulkUpsertCallCount++
		if bulkUpsertCallCount == 1 {
			upsertedProfiles = payload
		}
		return nil
	}

	// Simulate an in-memory KV store with TTL support
	kvStore := make(map[string]string)
	kv.MGetFunc = func(ctx context.Context, keys []string) (map[string]*string, error) {
		result := make(map[string]*string, len(keys))
		for _, k := range keys {
			if v, ok := kvStore[k]; ok {
				result[k] = &v
			} else {
				result[k] = nil
			}
		}
		return result, nil
	}

	// verify host marked as going through setup does not get profiles reconciled
	blockedKey := fleet.MDMProfileProcessingKeyPrefix + ":" + blockedHostUUID
	kvStore[blockedKey] = "1"

	upsertedProfiles = nil
	bulkUpsertCallCount = 0
	err := ReconcileAppleProfiles(ctx, ds, cmdr, kv, slog.New(slog.DiscardHandler), 0)
	require.NoError(t, err)

	// Only the non setup host should have profiles with a pending status and command UUID;
	// the blocked host should have its status/command cleared.
	var pendingHosts []string
	var skippedHosts []string
	for _, p := range upsertedProfiles {
		if p.Status != nil && *p.Status == fleet.MDMDeliveryPending && p.CommandUUID != "" {
			pendingHosts = append(pendingHosts, p.HostUUID)
		} else if p.Status == nil && p.CommandUUID == "" {
			skippedHosts = append(skippedHosts, p.HostUUID)
		}
	}
	assert.Contains(t, pendingHosts, nonSetupHostUUID, "non setup host should have profiles enqueued")
	assert.NotContains(t, pendingHosts, blockedHostUUID, "blocked host should NOT have profiles enqueued")
	assert.Contains(t, skippedHosts, blockedHostUUID, "blocked host should be skipped with nil status")

	// expire the key, the host that didn't get profiles before should do now
	delete(kvStore, blockedKey) // simulate TTL expiry

	upsertedProfiles = nil
	bulkUpsertCallCount = 0
	err = ReconcileAppleProfiles(ctx, ds, cmdr, kv, slog.New(slog.DiscardHandler), 0)
	require.NoError(t, err)

	pendingHosts = nil
	for _, p := range upsertedProfiles {
		if p.Status != nil && *p.Status == fleet.MDMDeliveryPending && p.CommandUUID != "" {
			pendingHosts = append(pendingHosts, p.HostUUID)
		}
	}
	assert.Contains(t, pendingHosts, nonSetupHostUUID, "non setup host should still have profiles enqueued")
	assert.Contains(t, pendingHosts, blockedHostUUID, "previously blocked host should now have profiles enqueued after key expiry")
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
	logger := slog.New(slog.DiscardHandler)
	mdmConfig := config.MDMConfig{
		AppleSCEPCert: "./testdata/server.pem",
		AppleSCEPKey:  "./testdata/server.key",
	}
	signingCert, _, _, err := mdmConfig.AppleSCEP()
	require.NoError(t, err)

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
		err := ensureFleetProfiles(ctx, ds, logger, signingCert.Certificate[0])
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
		err := ensureFleetProfiles(ctx, ds, logger, signingCert.Certificate[0])
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
			appCfg.MDM.DeprecatedAppleBMDefaultTeam = testTeamName
			return appCfg, nil
		}
		ds.AggregateEnrollSecretPerTeamFunc = func(ctx context.Context) ([]*fleet.EnrollSecret, error) {
			return secrets, nil
		}
		ds.BulkUpsertMDMAppleConfigProfilesFunc = func(ctx context.Context, ps []*fleet.MDMAppleConfigProfile) error {
			// fleetd + CA profiles
			require.Len(t, ps, len(secrets)*2)
			var fleetd, fleetCA []*fleet.MDMAppleConfigProfile
			for _, p := range ps {
				switch p.Identifier {
				case mobileconfig.FleetdConfigPayloadIdentifier:
					fleetd = append(fleetd, p)
				case mobileconfig.FleetCARootConfigPayloadIdentifier:
					fleetCA = append(fleetCA, p)
				}
			}
			require.Len(t, fleetd, 3)
			require.Len(t, fleetCA, 3)

			for i, p := range fleetd {
				require.Contains(t, string(p.Mobileconfig), testURL)
				require.Contains(t, string(p.Mobileconfig), secrets[i].Secret)
			}
			return nil
		}

		err := ensureFleetProfiles(ctx, ds, logger, signingCert.Certificate[0])
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
			appCfg.MDM.DeprecatedAppleBMDefaultTeam = testTeamName
			return appCfg, nil
		}
		ds.AggregateEnrollSecretPerTeamFunc = func(ctx context.Context) ([]*fleet.EnrollSecret, error) {
			return secrets, nil
		}
		ds.BulkUpsertMDMAppleConfigProfilesFunc = func(ctx context.Context, ps []*fleet.MDMAppleConfigProfile) error {
			// fleetd + CA profiles
			require.Len(t, ps, len(secrets)*2)
			var fleetd, fleetCA []*fleet.MDMAppleConfigProfile
			for _, p := range ps {
				switch p.Identifier {
				case mobileconfig.FleetdConfigPayloadIdentifier:
					fleetd = append(fleetd, p)
				case mobileconfig.FleetCARootConfigPayloadIdentifier:
					fleetCA = append(fleetCA, p)
				}
			}
			require.Len(t, fleetd, 2)
			require.Len(t, fleetCA, 2)

			for i, p := range fleetd {
				require.Contains(t, string(p.Mobileconfig), testURL)
				require.Contains(t, string(p.Mobileconfig), secrets[i].Secret)
			}
			return nil
		}
		err := ensureFleetProfiles(ctx, ds, logger, signingCert.Certificate[0])
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
		err := ensureFleetProfiles(ctx, ds, logger, signingCert.Certificate[0])
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
		err := ensureFleetProfiles(ctx, ds, logger, signingCert.Certificate[0])
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
		err := ensureFleetProfiles(ctx, ds, logger, signingCert.Certificate[0])
		require.ErrorIs(t, err, testError)
		require.True(t, ds.AppConfigFuncInvoked)
		require.True(t, ds.AggregateEnrollSecretPerTeamFuncInvoked)
		require.True(t, ds.BulkUpsertMDMAppleConfigProfilesFuncInvoked)
	})
}

func TestMDMAppleSetupAssistant(t *testing.T) {
	svc, ctx, ds, _ := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})

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
	ds.TeamWithExtrasFunc = func(ctx context.Context, id uint) (*fleet.Team, error) {
		return &fleet.Team{ID: id}, nil
	}
	ds.GetMDMAppleEnrollmentProfileByTypeFunc = func(ctx context.Context, typ fleet.MDMAppleEnrollmentType) (*fleet.MDMAppleEnrollmentProfile, error) {
		return &fleet.MDMAppleEnrollmentProfile{Token: "foobar"}, nil
	}
	ds.CountABMTokensWithTermsExpiredFunc = func(ctx context.Context) (int, error) {
		return 0, nil
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
	svc, ctx, _, _ := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})

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

// Helper for creating scoped mobileconfigs. scope is optional and if set to nil is not included in
// the mobileconfig so that default behavior is used. Note that because Fleet enforces that all
// profiles sharing a given identifier have the same scope, it's a good idea to use a unique
// identifier in your test or perhaps one with the scope in its name
func scopedMobileconfigForTest(name, identifier string, scope *fleet.PayloadScope, vars ...string) []byte {
	var varsStr strings.Builder
	for i, v := range vars {
		if !strings.HasPrefix(v, "FLEET_VAR_") {
			v = "FLEET_VAR_" + v
		}
		varsStr.WriteString(fmt.Sprintf("<key>Var %d</key><string>$%s</string>", i, v))
	}
	scopeEntry := ""
	if scope != nil {
		scopeEntry = fmt.Sprintf(`\n<key>PayloadScope</key>\n<string>%s</string>`, string(*scope))
	}

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
	<string>%s</string>%s
	<key>PayloadVersion</key>
	<integer>1</integer>
	%s
</dict>
</plist>
`, name, identifier, uuid.New().String(), scopeEntry, varsStr.String()))
}

func mobileconfigForTest(name, identifier string, vars ...string) []byte {
	return scopedMobileconfigForTest(name, identifier, nil, vars...)
}

func declBytesForTest(identifier string, payloadContent string) []byte {
	tmpl := `{
		"Type": "com.apple.configuration.decl%s",
		"Identifier": "com.fleet.config%s",
		"Payload": {
			"ServiceType": "com.apple.service%s"
		}
	}`

	declBytes := []byte(fmt.Sprintf(tmpl, identifier, identifier, payloadContent))
	return declBytes
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

func setupTest(t *testing.T) (context.Context, *slog.Logger, *mock.Store, *config.FleetConfig, *mdmmock.MDMAppleStore,
	*apple_mdm.MDMAppleCommander,
) {
	ctx := context.Background()
	logger := slog.New(slog.DiscardHandler)
	cfg := config.TestConfig()
	ds := new(mock.Store)
	mdmStorage := &mdmmock.MDMAppleStore{}
	pushFactory, _ := newMockAPNSPushProviderFactory()
	pusher := nanomdm_pushsvc.New(
		mdmStorage,
		mdmStorage,
		pushFactory,
		stdlogfmt.New(),
	)
	mdmConfig := config.MDMConfig{
		AppleSCEPCert: "./testdata/server.pem",
		AppleSCEPKey:  "./testdata/server.key",
	}
	apnsCert, apnsKey, err := mysql.GenerateTestCertBytes(mdmtesting.NewTestMDMAppleCertTemplate())
	require.NoError(t, err)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		appCfg := &fleet.AppConfig{}
		appCfg.MDM.EnabledAndConfigured = true
		return appCfg, nil
	}

	_, pemCert, pemKey, err := mdmConfig.AppleSCEP()
	require.NoError(t, err)
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext,
	) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetCACert:   {Value: pemCert},
			fleet.MDMAssetCAKey:    {Value: pemKey},
			fleet.MDMAssetAPNSKey:  {Value: apnsKey},
			fleet.MDMAssetAPNSCert: {Value: apnsCert},
		}, nil
	}
	mdmStorage.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext,
	) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetCACert:   {Value: pemCert},
			fleet.MDMAssetCAKey:    {Value: pemKey},
			fleet.MDMAssetAPNSKey:  {Value: apnsKey},
			fleet.MDMAssetAPNSCert: {Value: apnsCert},
		}, nil
	}

	commander := apple_mdm.NewMDMAppleCommander(mdmStorage, pusher)

	return ctx, logger, ds, &cfg, mdmStorage, commander
}

func TestRenewSCEPCertificatesMDMConfigNotSet(t *testing.T) {
	ctx, logger, ds, cfg, _, commander := setupTest(t)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		appCfg := &fleet.AppConfig{}
		appCfg.MDM.EnabledAndConfigured = false
		return appCfg, nil
	}
	err := RenewSCEPCertificates(ctx, logger, ds, cfg, commander, &mock.MockACMEService{})
	require.NoError(t, err)
}

func TestRenewSCEPCertificatesCommanderNil(t *testing.T) {
	ctx, logger, ds, cfg, _, _ := setupTest(t)
	err := RenewSCEPCertificates(ctx, logger, ds, cfg, nil, &mock.MockACMEService{})
	require.NoError(t, err)
}

func TestRenewSCEPCertificatesBranches(t *testing.T) {
	// NOTE: These tests assume appConfig.MDM.AppleRequireHardwareAttestation is false.
	// They do not cover ACME renewal logic.
	tests := []struct {
		name               string
		customExpectations func(*testing.T, *mock.Store, *config.FleetConfig, *mdmmock.MDMAppleStore, *apple_mdm.MDMAppleCommander)
		expectedError      bool
	}{
		{
			name: "No Certs to Renew",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mdmmock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				ds.GetHostCertAssociationsToExpireFunc = func(ctx context.Context, expiryDays int, limit int) ([]fleet.SCEPIdentityAssociation, error) {
					return nil, nil
				}
			},
			expectedError: false,
		},
		{
			name: "GetHostCertAssociationsToExpire Errors",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mdmmock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				ds.GetHostCertAssociationsToExpireFunc = func(ctx context.Context, expiryDays int, limit int) ([]fleet.SCEPIdentityAssociation, error) {
					return nil, errors.New("database error")
				}
			},
			expectedError: true,
		},
		{
			name: "AppConfig Errors",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mdmmock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
					return nil, errors.New("app config error")
				}
			},
			expectedError: true,
		},
		{
			name: "InstallProfile for hostsWithoutRefs",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mdmmock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				var wantCommandUUID string
				ds.GetHostCertAssociationsToExpireFunc = func(ctx context.Context, expiryDays int, limit int) ([]fleet.SCEPIdentityAssociation, error) {
					return []fleet.SCEPIdentityAssociation{{HostUUID: "hostUUID1", EnrollReference: ""}}, nil
				}

				appleStore.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error,
					error,
				) {
					require.Equal(t, "InstallProfile", cmd.Command.Command.RequestType)
					require.Equal(t, "fl33t enrollment", cmd.Name)
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
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mdmmock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				ds.GetHostCertAssociationsToExpireFunc = func(ctx context.Context, expiryDays int, limit int) ([]fleet.SCEPIdentityAssociation, error) {
					return []fleet.SCEPIdentityAssociation{{HostUUID: "hostUUID1", EnrollReference: ""}}, nil
				}

				appleStore.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error,
					error,
				) {
					return map[string]error{}, errors.New("foo")
				}
			},
			expectedError: true,
		},
		{
			name: "InstallProfile for hostsWithRefs",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mdmmock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				var wantCommandUUID string
				ds.GetHostCertAssociationsToExpireFunc = func(ctx context.Context, expiryDays int, limit int) ([]fleet.SCEPIdentityAssociation, error) {
					return []fleet.SCEPIdentityAssociation{{HostUUID: "hostUUID2", EnrollReference: "ref1"}}, nil
				}
				appleStore.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error,
					error,
				) {
					require.Equal(t, "InstallProfile", cmd.Command.Command.RequestType)
					require.Equal(t, "fl33t enrollment", cmd.Name)
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
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mdmmock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				ds.GetHostCertAssociationsToExpireFunc = func(ctx context.Context, expiryDays int, limit int) ([]fleet.SCEPIdentityAssociation, error) {
					return []fleet.SCEPIdentityAssociation{{HostUUID: "hostUUID1", EnrollReference: "ref1"}}, nil
				}

				appleStore.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error,
					error,
				) {
					return map[string]error{}, errors.New("foo")
				}
			},
			expectedError: true,
		},
		{
			name: "InstallProfile for userDeviceAssocs",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mdmmock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				wantCommandUUIDs := make(map[string]string)
				ds.GetHostCertAssociationsToExpireFunc = func(ctx context.Context, expiryDays int, limit int) ([]fleet.SCEPIdentityAssociation, error) {
					return []fleet.SCEPIdentityAssociation{{HostUUID: "hostUUID1", EnrollmentType: "User Enrollment (Device)"}, {HostUUID: "hostUUID2", EnrollmentType: "User Enrollment (Device)"}}, nil
				}
				user1Email := "user1@example.com"
				user2Email := "user2@example.com"
				ds.GetMDMIdPAccountsByHostUUIDsFunc = func(ctx context.Context, hostUUIDs []string) (map[string]*fleet.MDMIdPAccount, error) {
					require.Len(t, hostUUIDs, 2)
					return map[string]*fleet.MDMIdPAccount{
						"hostUUID2": {
							UUID:     "userUUID2",
							Username: "user2",
							Email:    user2Email,
						},
						"hostUUID1": {
							UUID:     "userUUID1",
							Username: "user1",
							Email:    user1Email,
						},
					}, nil
				}
				appleStore.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error,
					error,
				) {
					require.Equal(t, "InstallProfile", cmd.Command.Command.RequestType)
					require.Equal(t, "fl33t account driven enrollment", cmd.Name)
					require.Equal(t, 1, len(id))
					_, idAlreadyExists := wantCommandUUIDs[id[0]]
					// Should only get one for each host
					require.False(t, idAlreadyExists, "Command UUID for host %s already exists: %s", id[0], wantCommandUUIDs[id[0]])
					wantCommandUUIDs[id[0]] = cmd.CommandUUID

					// Make sure the user's email made it into the profile
					var fullCmd micromdm.CommandPayload
					require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
					switch id[0] {
					case "hostUUID1":
						require.True(t, bytes.Contains(fullCmd.Command.InstallProfile.Payload, []byte(user1Email)), "The profile for hostUUID 1 should contain the associated user email")
					case "hostUUID2":
						require.True(t, bytes.Contains(fullCmd.Command.InstallProfile.Payload, []byte(user2Email)), "The profile for hostUUID 2 should contain the associated user email")
					default:
						require.Fail(t, "Unexpected host ID for command: %s", id[0])
					}
					return map[string]error{}, nil
				}
				ds.SetCommandForPendingSCEPRenewalFunc = func(ctx context.Context, assocs []fleet.SCEPIdentityAssociation, cmdUUID string) error {
					require.Len(t, assocs, 1)
					require.Contains(t, []string{"hostUUID1", "hostUUID2"}, assocs[0].HostUUID)
					require.Equal(t, cmdUUID, wantCommandUUIDs[assocs[0].HostUUID])
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
			name: "InstallProfile for userDeviceAssocs does not return email for one device",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mdmmock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				wantCommandUUIDs := make(map[string]string)
				ds.GetHostCertAssociationsToExpireFunc = func(ctx context.Context, expiryDays int, limit int) ([]fleet.SCEPIdentityAssociation, error) {
					return []fleet.SCEPIdentityAssociation{{HostUUID: "hostUUID1", EnrollmentType: "User Enrollment (Device)"}, {HostUUID: "hostUUID2", EnrollmentType: "User Enrollment (Device)"}}, nil
				}
				user1Email := "user1@example.com"
				ds.GetMDMIdPAccountsByHostUUIDsFunc = func(ctx context.Context, hostUUIDs []string) (map[string]*fleet.MDMIdPAccount, error) {
					require.Len(t, hostUUIDs, 2)
					return map[string]*fleet.MDMIdPAccount{
						"hostUUID1": {
							UUID:     "userUUID1",
							Username: "user1",
							Email:    user1Email,
						},
					}, nil
				}
				appleStore.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error,
					error,
				) {
					require.Equal(t, "InstallProfile", cmd.Command.Command.RequestType)
					require.Equal(t, "fl33t account driven enrollment", cmd.Name)
					require.Equal(t, 1, len(id))
					_, idAlreadyExists := wantCommandUUIDs[id[0]]
					// Should only get one for each host
					require.False(t, idAlreadyExists, "Command UUID for host %s already exists: %s", id[0], wantCommandUUIDs[id[0]])
					wantCommandUUIDs[id[0]] = cmd.CommandUUID

					// Make sure the user's email made it into the profile if it was returned
					var fullCmd micromdm.CommandPayload
					require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
					switch id[0] {
					// Only hostUUID1 has an email associated with it
					// so we expect it to be present in the profile
					case "hostUUID1":
						require.True(t, bytes.Contains(fullCmd.Command.InstallProfile.Payload, []byte(user1Email)), "The profile for hostUUID 1 should contain the associated user email")
					case "hostUUID2":
						require.False(t, bytes.Contains(fullCmd.Command.InstallProfile.Payload, []byte("@example.com")), "The profile for hostUUID 2 should not contain any user email")
					default:
						require.Fail(t, "Unexpected host ID for command: %s", id[0])
					}
					return map[string]error{}, nil
				}
				ds.SetCommandForPendingSCEPRenewalFunc = func(ctx context.Context, assocs []fleet.SCEPIdentityAssociation, cmdUUID string) error {
					require.Len(t, assocs, 1)
					require.Contains(t, []string{"hostUUID1", "hostUUID2"}, assocs[0].HostUUID)
					require.Equal(t, cmdUUID, wantCommandUUIDs[assocs[0].HostUUID])
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
			name: "InstallProfile for userDeviceAssocs fails",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mdmmock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				ds.GetHostCertAssociationsToExpireFunc = func(ctx context.Context, expiryDays int, limit int) ([]fleet.SCEPIdentityAssociation, error) {
					return []fleet.SCEPIdentityAssociation{{HostUUID: "hostUUID1", EnrollmentType: "User Enrollment (Device)"}, {HostUUID: "hostUUID2", EnrollmentType: "User Enrollment (Device)"}}, nil
				}
				user1Email := "user1@example.com"
				user2Email := "user2@example.com"
				ds.GetMDMIdPAccountsByHostUUIDsFunc = func(ctx context.Context, hostUUIDs []string) (map[string]*fleet.MDMIdPAccount, error) {
					require.Len(t, hostUUIDs, 2)
					return map[string]*fleet.MDMIdPAccount{
						"hostUUID2": {
							UUID:     "userUUID2",
							Username: "user2",
							Email:    user2Email,
						},
						"hostUUID1": {
							UUID:     "userUUID1",
							Username: "user1",
							Email:    user1Email,
						},
					}, nil
				}
				appleStore.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error,
					error,
				) {
					return map[string]error{}, errors.New("foo")
				}
			},
			expectedError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, logger, ds, cfg, appleStorage, commander := setupTest(t)

			acmeSvc := &mock.MockACMEService{}
			acmeSvc.NewACMEEnrollmentFunc = func(ctx context.Context, hostIdentifier string) (string, error) {
				return uuid.NewString(), nil
			}

			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				appCfg := &fleet.AppConfig{}
				appCfg.OrgInfo.OrgName = "fl33t"
				appCfg.ServerSettings.ServerURL = "https://foo.example.com"
				appCfg.MDM.EnabledAndConfigured = true
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
				apnsCert, apnsKey, err := mysql.GenerateTestCertBytes(mdmtesting.NewTestMDMAppleCertTemplate())
				require.NoError(t, err)
				cert, err := tls.X509KeyPair(apnsCert, apnsKey)
				return &cert, "", err
			}

			tc.customExpectations(t, ds, cfg, appleStorage, commander)

			err := RenewSCEPCertificates(ctx, logger, ds, cfg, commander, acmeSvc)
			if tc.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.False(t, acmeSvc.NewACMEEnrollmentFuncInvoked, "NewACMEEnrollment should not be called during SCEP cert renewal")
		})
	}
}

func TestRenewACMECertificatesBranches(t *testing.T) {
	// These tests cover the ACME renewal branch in RenewSCEPCertificates, specifically when
	// AppConfig.MDM.AppleRequireHardwareAttestation is true and eligible devices require ACME.
	const (
		appleSiliconModel = "Mac13,1"   // Apple Silicon (Mac Studio); satisfies IsMacAppleSilicon
		intelMacModel     = "MacPro7,1" // Intel Mac Pro; IsMacAppleSilicon returns false
		acmeMacOSVersion  = "14.0"
	)

	tests := []struct {
		name               string
		customExpectations func(*testing.T, *mock.Store, *config.FleetConfig, *mdmmock.MDMAppleStore, *apple_mdm.MDMAppleCommander, *mock.MockACMEService)
		expectedError      bool
	}{
		{
			name: "AppleRequireHardwareAttestation false skips ACME check",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mdmmock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander, acmeSvc *mock.MockACMEService) {
				ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
					appCfg := &fleet.AppConfig{}
					appCfg.OrgInfo.OrgName = "fl33t"
					appCfg.ServerSettings.ServerURL = "https://foo.example.com"
					appCfg.MDM.EnabledAndConfigured = true
					appCfg.MDM.AppleRequireHardwareAttestation = false
					return appCfg, nil
				}
				ds.GetHostCertAssociationsToExpireFunc = func(ctx context.Context, expiryDays int, limit int) ([]fleet.SCEPIdentityAssociation, error) {
					return []fleet.SCEPIdentityAssociation{{HostUUID: "hostUUID1", EnrollReference: ""}}, nil
				}
				appleStore.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error, error) {
					return map[string]error{}, nil
				}
				t.Cleanup(func() {
					require.False(t, ds.GetDeviceInfoForACMERenewalFuncInvoked, "GetDeviceInfoForACMERenewal should not be called when AppleRequireHardwareAttestation is false")
					require.False(t, acmeSvc.NewACMEEnrollmentFuncInvoked, "NewACMEEnrollment should not be called when AppleRequireHardwareAttestation is false")
				})
			},
			expectedError: false,
		},
		{
			name: "GetDeviceInfoForACMERenewal errors",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mdmmock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander, acmeSvc *mock.MockACMEService) {
				ds.GetHostCertAssociationsToExpireFunc = func(ctx context.Context, expiryDays int, limit int) ([]fleet.SCEPIdentityAssociation, error) {
					return []fleet.SCEPIdentityAssociation{{HostUUID: "hostUUID1", EnrollReference: ""}}, nil
				}
				ds.GetDeviceInfoForACMERenewalFunc = func(ctx context.Context, hostUUIDs []string) ([]fleet.DeviceInfoForACMERenewal, error) {
					return nil, errors.New("database error")
				}
			},
			expectedError: true,
		},
		{
			name: "Non-Apple-Silicon device gets regular SCEP renewal, no ACME",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mdmmock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander, acmeSvc *mock.MockACMEService) {
				ds.GetHostCertAssociationsToExpireFunc = func(ctx context.Context, expiryDays int, limit int) ([]fleet.SCEPIdentityAssociation, error) {
					return []fleet.SCEPIdentityAssociation{{HostUUID: "hostUUID1", EnrollReference: ""}}, nil
				}
				ds.GetDeviceInfoForACMERenewalFunc = func(ctx context.Context, hostUUIDs []string) ([]fleet.DeviceInfoForACMERenewal, error) {
					require.Equal(t, []string{"hostUUID1"}, hostUUIDs)
					return []fleet.DeviceInfoForACMERenewal{{
						HostUUID:       "hostUUID1",
						HardwareSerial: "INTEL001",
						HardwareModel:  intelMacModel,
						OSVersion:      acmeMacOSVersion,
					}}, nil
				}
				appleStore.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error, error) {
					require.Equal(t, "fl33t enrollment", cmd.Name)
					return map[string]error{}, nil
				}
				t.Cleanup(func() {
					require.True(t, ds.GetDeviceInfoForACMERenewalFuncInvoked, "GetDeviceInfoForACMERenewal should be called when AppleRequireHardwareAttestation is true")
					require.False(t, acmeSvc.NewACMEEnrollmentFuncInvoked, "NewACMEEnrollment should not be called for non-Apple-Silicon device")
					require.True(t, appleStore.EnqueueCommandFuncInvoked, "EnqueueCommand should be called for regular SCEP renewal")
				})
			},
			expectedError: false,
		},
		{
			name: "Apple Silicon macOS 14+ host without enrollment reference gets ACME profile",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mdmmock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander, acmeSvc *mock.MockACMEService) {
				const serial = "APPLESILICON001"
				var wantCommandUUID string
				acmeSvc.NewACMEEnrollmentFunc = func(ctx context.Context, hostIdentifier string) (string, error) {
					require.Equal(t, serial, hostIdentifier)
					return "acme-ident-001", nil
				}
				ds.GetHostCertAssociationsToExpireFunc = func(ctx context.Context, expiryDays int, limit int) ([]fleet.SCEPIdentityAssociation, error) {
					return []fleet.SCEPIdentityAssociation{{HostUUID: "hostUUID1", EnrollReference: ""}}, nil
				}
				ds.GetDeviceInfoForACMERenewalFunc = func(ctx context.Context, hostUUIDs []string) ([]fleet.DeviceInfoForACMERenewal, error) {
					require.Equal(t, []string{"hostUUID1"}, hostUUIDs)
					return []fleet.DeviceInfoForACMERenewal{{
						HostUUID:       "hostUUID1",
						HardwareSerial: serial,
						HardwareModel:  appleSiliconModel,
						OSVersion:      acmeMacOSVersion,
					}}, nil
				}
				appleStore.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error, error) {
					require.Equal(t, "InstallProfile", cmd.Command.Command.RequestType)
					require.Equal(t, "fl33t ACME enrollment", cmd.Name)
					wantCommandUUID = cmd.CommandUUID
					// Verify the profile is an ACME profile by checking it contains the device serial
					var fullCmd micromdm.CommandPayload
					require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
					require.True(t, bytes.Contains(fullCmd.Command.InstallProfile.Payload, []byte(serial)), "ACME profile should contain the device serial number as ClientIdentifier")
					require.True(t, bytes.Contains(fullCmd.Command.InstallProfile.Payload, []byte("com.apple.security.acme")), "profile should be of ACME payload type")
					return map[string]error{}, nil
				}
				ds.SetCommandForPendingSCEPRenewalFunc = func(ctx context.Context, assocs []fleet.SCEPIdentityAssociation, cmdUUID string) error {
					require.Len(t, assocs, 1)
					require.Equal(t, "hostUUID1", assocs[0].HostUUID)
					require.Equal(t, wantCommandUUID, cmdUUID)
					return nil
				}
				t.Cleanup(func() {
					require.True(t, acmeSvc.NewACMEEnrollmentFuncInvoked, "NewACMEEnrollment should be called for Apple Silicon macOS 14+ device")
					require.True(t, appleStore.EnqueueCommandFuncInvoked, "EnqueueCommand should be called to send the ACME profile")
					require.True(t, ds.SetCommandForPendingSCEPRenewalFuncInvoked, "SetCommandForPendingSCEPRenewal should be called")
				})
			},
			expectedError: false,
		},
		{
			name: "Apple Silicon macOS 14+ host with enrollment reference gets ACME profile",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mdmmock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander, acmeSvc *mock.MockACMEService) {
				const serial = "APPLESILICON002"
				const enrollRef = "ref123"
				var wantCommandUUID string
				acmeSvc.NewACMEEnrollmentFunc = func(ctx context.Context, hostIdentifier string) (string, error) {
					require.Equal(t, serial, hostIdentifier)
					return "acme-ident-002", nil
				}
				ds.GetHostCertAssociationsToExpireFunc = func(ctx context.Context, expiryDays int, limit int) ([]fleet.SCEPIdentityAssociation, error) {
					return []fleet.SCEPIdentityAssociation{{HostUUID: "hostUUID2", EnrollReference: enrollRef}}, nil
				}
				ds.GetDeviceInfoForACMERenewalFunc = func(ctx context.Context, hostUUIDs []string) ([]fleet.DeviceInfoForACMERenewal, error) {
					require.Equal(t, []string{"hostUUID2"}, hostUUIDs)
					return []fleet.DeviceInfoForACMERenewal{{
						HostUUID:       "hostUUID2",
						HardwareSerial: serial,
						HardwareModel:  appleSiliconModel,
						OSVersion:      acmeMacOSVersion,
					}}, nil
				}
				appleStore.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error, error) {
					require.Equal(t, "InstallProfile", cmd.Command.Command.RequestType)
					require.Equal(t, "fl33t ACME enrollment", cmd.Name)
					wantCommandUUID = cmd.CommandUUID
					var fullCmd micromdm.CommandPayload
					require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
					require.True(t, bytes.Contains(fullCmd.Command.InstallProfile.Payload, []byte(serial)), "ACME profile should contain the device serial number as ClientIdentifier")
					require.True(t, bytes.Contains(fullCmd.Command.InstallProfile.Payload, []byte("com.apple.security.acme")), "profile should be of ACME payload type")
					return map[string]error{}, nil
				}
				ds.SetCommandForPendingSCEPRenewalFunc = func(ctx context.Context, assocs []fleet.SCEPIdentityAssociation, cmdUUID string) error {
					require.Len(t, assocs, 1)
					require.Equal(t, "hostUUID2", assocs[0].HostUUID)
					require.Equal(t, wantCommandUUID, cmdUUID)
					return nil
				}
				t.Cleanup(func() {
					require.True(t, acmeSvc.NewACMEEnrollmentFuncInvoked, "NewACMEEnrollment should be called for Apple Silicon macOS 14+ device with enrollment reference")
					require.True(t, appleStore.EnqueueCommandFuncInvoked, "EnqueueCommand should be called to send the ACME profile")
					require.True(t, ds.SetCommandForPendingSCEPRenewalFuncInvoked, "SetCommandForPendingSCEPRenewal should be called")
				})
			},
			expectedError: false,
		},
		{
			name: "NewACMEEnrollment errors returns error",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mdmmock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander, acmeSvc *mock.MockACMEService) {
				acmeSvc.NewACMEEnrollmentFunc = func(ctx context.Context, hostIdentifier string) (string, error) {
					return "", errors.New("ACME service unavailable")
				}
				ds.GetHostCertAssociationsToExpireFunc = func(ctx context.Context, expiryDays int, limit int) ([]fleet.SCEPIdentityAssociation, error) {
					return []fleet.SCEPIdentityAssociation{{HostUUID: "hostUUID1", EnrollReference: ""}}, nil
				}
				ds.GetDeviceInfoForACMERenewalFunc = func(ctx context.Context, hostUUIDs []string) ([]fleet.DeviceInfoForACMERenewal, error) {
					return []fleet.DeviceInfoForACMERenewal{{
						HostUUID:       "hostUUID1",
						HardwareSerial: "SERIAL001",
						HardwareModel:  appleSiliconModel,
						OSVersion:      acmeMacOSVersion,
					}}, nil
				}
			},
			expectedError: true,
		},
		{
			name: "Mixed hosts: Apple Silicon gets ACME profile, Intel Mac gets regular SCEP profile",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mdmmock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander, acmeSvc *mock.MockACMEService) {
				const acmeSerial = "APPLESILICON003"
				acmeEnrollmentCallCount := 0
				acmeSvc.NewACMEEnrollmentFunc = func(ctx context.Context, hostIdentifier string) (string, error) {
					require.Equal(t, acmeSerial, hostIdentifier)
					acmeEnrollmentCallCount++
					return "acme-ident-003", nil
				}
				ds.GetHostCertAssociationsToExpireFunc = func(ctx context.Context, expiryDays int, limit int) ([]fleet.SCEPIdentityAssociation, error) {
					return []fleet.SCEPIdentityAssociation{
						{HostUUID: "hostUUID-acme", EnrollReference: ""},
						{HostUUID: "hostUUID-scep", EnrollReference: ""},
					}, nil
				}
				ds.GetDeviceInfoForACMERenewalFunc = func(ctx context.Context, hostUUIDs []string) ([]fleet.DeviceInfoForACMERenewal, error) {
					require.Len(t, hostUUIDs, 2)
					return []fleet.DeviceInfoForACMERenewal{
						{
							HostUUID:       "hostUUID-acme",
							HardwareSerial: acmeSerial,
							HardwareModel:  appleSiliconModel,
							OSVersion:      acmeMacOSVersion,
						},
						{
							HostUUID:       "hostUUID-scep",
							HardwareSerial: "INTELMAC003",
							HardwareModel:  intelMacModel,
							OSVersion:      acmeMacOSVersion,
						},
					}, nil
				}
				enqueuedHostUUIDs := make(map[string]bool)
				appleStore.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error, error) {
					require.Equal(t, "InstallProfile", cmd.Command.Command.RequestType)
					for _, hostUUID := range id {
						switch hostUUID {
						case "hostUUID-acme":
							require.Equal(t, "fl33t ACME enrollment", cmd.Name)
						case "hostUUID-scep":
							require.Equal(t, "fl33t enrollment", cmd.Name)
						default:
							require.Failf(t, "Unexpected host UUID", "Unexpected host UUID: %s", hostUUID)
						}
						enqueuedHostUUIDs[hostUUID] = true
					}
					return map[string]error{}, nil
				}
				t.Cleanup(func() {
					require.Equal(t, 1, acmeEnrollmentCallCount, "NewACMEEnrollment should be called exactly once for the Apple Silicon host")
					require.True(t, enqueuedHostUUIDs["hostUUID-acme"], "ACME host should have an InstallProfile command enqueued")
					require.True(t, enqueuedHostUUIDs["hostUUID-scep"], "SCEP host should have an InstallProfile command enqueued")
				})
			},
			expectedError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, logger, ds, cfg, appleStorage, commander := setupTest(t)

			acmeSvc := &mock.MockACMEService{}
			acmeSvc.NewACMEEnrollmentFunc = func(ctx context.Context, hostIdentifier string) (string, error) {
				return uuid.NewString(), nil
			}

			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				appCfg := &fleet.AppConfig{}
				appCfg.OrgInfo.OrgName = "fl33t"
				appCfg.ServerSettings.ServerURL = "https://foo.example.com"
				appCfg.MDM.EnabledAndConfigured = true
				appCfg.MDM.AppleRequireHardwareAttestation = true
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
				for _, id := range targets {
					pushes[id] = &mdm.Push{
						PushMagic: "magic" + id,
						Token:     []byte("token" + id),
						Topic:     "topic" + id,
					}
				}
				return pushes, nil
			}

			appleStorage.RetrievePushCertFunc = func(ctx context.Context, topic string) (*tls.Certificate, string, error) {
				apnsCert, apnsKey, err := mysql.GenerateTestCertBytes(mdmtesting.NewTestMDMAppleCertTemplate())
				require.NoError(t, err)
				cert, err := tls.X509KeyPair(apnsCert, apnsKey)
				return &cert, "", err
			}

			tc.customExpectations(t, ds, cfg, appleStorage, commander, acmeSvc)

			err := RenewSCEPCertificates(ctx, logger, ds, cfg, commander, acmeSvc)
			if tc.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMDMCommandAndReportResultsIOSIPadOSRefetch(t *testing.T) {
	ctx := context.Background()
	hostID := uint(42)
	hostUUID := "ABC-DEF-GHI"
	commandUUID := fleet.RefetchDeviceCommandUUIDPrefix + "UUID"
	lostModeCommandUUID := uuid.NewString()

	ds := new(mock.Store)
	svc := MDMAppleCheckinAndCommandService{ds: ds, logger: slog.New(slog.DiscardHandler)}

	ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.Host, error) {
		return &fleet.Host{
			ID:   hostID,
			UUID: hostUUID,
			MDM: fleet.MDMHostData{
				EnrollmentStatus: ptr.String("Pending"), // We check it in as a new device, to trigger lost mode flow
			},
		}, nil
	}
	ds.UpdateHostFunc = func(ctx context.Context, host *fleet.Host) error {
		require.Equal(t, "Work iPad", host.ComputerName)
		require.Equal(t, "Work iPad", host.Hostname)
		require.Equal(t, "iPadOS 17.5.1", host.OSVersion)
		require.Equal(t, "ff:ff:ff:ff:ff:ff", host.PrimaryMac)
		require.Equal(t, "iPad13,18", host.HardwareModel)
		require.WithinDuration(t, time.Now(), host.DetailUpdatedAt, 1*time.Minute)
		require.WithinDuration(t, time.Now(), host.LabelUpdatedAt, 1*time.Minute)
		return nil
	}
	ds.SetOrUpdateHostDisksSpaceFunc = func(ctx context.Context, incomingHostID uint, gigsAvailable, percentAvailable, gigsTotal float64, gigsAll *float64) error {
		require.Equal(t, hostID, incomingHostID)
		require.NotZero(t, 51, int64(gigsAvailable))
		require.NotZero(t, 79, int64(percentAvailable))
		require.NotZero(t, 64, int64(gigsTotal))
		return nil
	}
	ds.UpdateHostOperatingSystemFunc = func(ctx context.Context, incomingHostID uint, hostOS fleet.OperatingSystem) error {
		require.Equal(t, hostID, incomingHostID)
		require.Equal(t, "iPadOS", hostOS.Name)
		require.Equal(t, "17.5.1", hostOS.Version)
		require.Equal(t, "ipados", hostOS.Platform)
		return nil
	}
	ds.RemoveHostMDMCommandFunc = func(ctx context.Context, command fleet.HostMDMCommand) error {
		assert.Equal(t, hostID, command.HostID)
		assert.Equal(t, fleet.RefetchDeviceCommandUUIDPrefix, command.CommandType)
		return nil
	}
	ds.UpdateMDMDataFunc = func(ctx context.Context, incomingHostID uint, enrolled bool) error {
		require.Equal(t, hostID, incomingHostID)
		return nil
	}
	ds.GetLatestAppleMDMCommandOfTypeFunc = func(ctx context.Context, incomingHostUUID, commandType string) (*fleet.MDMCommand, error) {
		require.Equal(t, hostUUID, incomingHostUUID)
		require.Equal(t, "EnableLostMode", commandType)
		return &fleet.MDMCommand{
			CommandUUID: lostModeCommandUUID,
		}, nil
	}
	ds.SetLockCommandForLostModeCheckinFunc = func(ctx context.Context, incomingHostUUID uint, commandUUID string) error {
		require.Equal(t, hostID, incomingHostUUID)
		require.Equal(t, lostModeCommandUUID, commandUUID)
		return nil
	}
	ds.CleanupStaleNanoRefetchCommandsFunc = func(ctx context.Context, enrollmentID string, commandUUIDPrefix string, currentCommandUUID string) error {
		require.Equal(t, hostUUID, enrollmentID)
		require.Equal(t, fleet.RefetchDeviceCommandUUIDPrefix, commandUUIDPrefix)
		require.Equal(t, commandUUID, currentCommandUUID)
		return nil
	}

	_, err := svc.CommandAndReportResults(
		&mdm.Request{Context: ctx},
		&mdm.CommandResults{
			Enrollment:  mdm.Enrollment{UDID: hostUUID},
			CommandUUID: commandUUID,
			Raw: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>CommandUUID</key>
        <string>REFETCH-fd23f8ac-1c50-41c7-a5bb-f13633c9ea97</string>
        <key>QueryResponses</key>
        <dict>
                <key>AvailableDeviceCapacity</key>
                <real>51.260395520000003</real>
                <key>DeviceCapacity</key>
                <real>64</real>
                <key>DeviceName</key>
                <string>Work iPad</string>
                <key>OSVersion</key>
                <string>17.5.1</string>
                <key>ProductName</key>
                <string>iPad13,18</string>
                <key>WiFiMAC</key>
                <string>ff:ff:ff:ff:ff:ff</string>
				<key>IsMDMLostModeEnabled</key>
				<true />
        </dict>
        <key>Status</key>
        <string>Acknowledged</string>
        <key>UDID</key>
        <string>FFFFFFFF-FFFFFFFFFFFFFFFF</string>
</dict>
</plist>`),
		},
	)
	require.NoError(t, err)

	require.True(t, ds.UpdateHostFuncInvoked)
	require.True(t, ds.HostByIdentifierFuncInvoked)
	require.True(t, ds.SetOrUpdateHostDisksSpaceFuncInvoked)
	require.True(t, ds.UpdateHostOperatingSystemFuncInvoked)
	assert.True(t, ds.RemoveHostMDMCommandFuncInvoked)
	require.True(t, ds.UpdateMDMDataFuncInvoked)
	require.True(t, ds.GetLatestAppleMDMCommandOfTypeFuncInvoked)
	require.True(t, ds.SetLockCommandForLostModeCheckinFuncInvoked)

	_, err = svc.CommandAndReportResults(
		&mdm.Request{Context: ctx},
		&mdm.CommandResults{
			Enrollment:  mdm.Enrollment{UDID: hostUUID},
			CommandUUID: commandUUID,
			Raw: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>CommandUUID</key>
        <string>REFETCH-fd23f8ac-1c50-41c7-a5bb-f13633c9ea97</string>
        <key>QueryResponses</key>
        <dict>
                <key>AvailableDeviceCapacity</key>
                <real>51.260395520000003</real>
                <key>DeviceCapacity</key>
                <real>64</real>
                <key>DeviceName</key>
                <string>Work iPad</string>
                <key>OSVersion</key>
                <string>17.5.1</string>
                <key>ProductName</key>
                <string>iPad13,18</string>
                <key>WiFiMAC</key>
                <string>ff:ff:ff:ff:ff:ff</string>
        </dict>
        <key>Status</key>
        <string>Acknowledged</string>
        <key>UDID</key>
        <string>FFFFFFFF-FFFFFFFFFFFFFFFF</string>
</dict>
</plist>`),
		},
	)
	require.NoError(t, err)
}

// TestMDMCommandAndReportResultsIOSIPadOSRefetchDefensive covers handling of
// DeviceInformation responses where one or more of the QueryResponses fields are
// missing or have an unexpected type. The handler must not panic, must preserve
// previously-known host values where the new value is unavailable, and must skip
// dependent datastore updates whose required inputs are missing.
func TestMDMCommandAndReportResultsIOSIPadOSRefetchDefensive(t *testing.T) {
	const (
		hostID            = uint(42)
		hostUUID          = "ABC-DEF-GHI"
		existingHostname  = "Existing iPad"
		existingModel     = "iPad12,2"       // differs from incoming ProductName so we can detect preservation
		existingOSVersion = "iPadOS 16.7.10" // differs from incoming OSVersion so we can detect preservation
	)
	commandUUID := fleet.RefetchDeviceCommandUUIDPrefix + "UUID"

	rawWithFields := func(fields map[string]string) []byte {
		var inner strings.Builder
		for _, v := range fields {
			inner.WriteString(v) // v is a complete <key>...</key><type>...</type> pair
		}
		return []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>CommandUUID</key>
        <string>REFETCH-fd23f8ac-1c50-41c7-a5bb-f13633c9ea97</string>
        <key>QueryResponses</key>
        <dict>` + inner.String() + `</dict>
        <key>Status</key>
        <string>Acknowledged</string>
        <key>UDID</key>
        <string>FFFFFFFF-FFFFFFFFFFFFFFFF</string>
</dict>
</plist>`)
	}

	allFields := func() map[string]string {
		return map[string]string{
			"DeviceName":              `<key>DeviceName</key><string>Work iPad</string>`,
			"DeviceCapacity":          `<key>DeviceCapacity</key><real>64</real>`,
			"AvailableDeviceCapacity": `<key>AvailableDeviceCapacity</key><real>51.26</real>`,
			"OSVersion":               `<key>OSVersion</key><string>17.5.1</string>`,
			"ProductName":             `<key>ProductName</key><string>iPad13,18</string>`,
			"WiFiMAC":                 `<key>WiFiMAC</key><string>ff:ff:ff:ff:ff:ff</string>`,
		}
	}

	type expectations struct {
		expectComputerName  string
		expectHostname      string
		expectOSVersion     string
		expectHardwareModel string
		expectGigsTotal     float64
		expectGigsAvailable float64
		expectDiskUpdate    bool
		expectOSUpdate      bool
		expectOSName        string
		expectOSVersionRow  string
		expectOSPlatform    string
	}

	cases := []struct {
		name   string
		mutate func(map[string]string)
		expect expectations
	}{
		{
			name:   "missing DeviceName preserves existing hostname",
			mutate: func(f map[string]string) { delete(f, "DeviceName") },
			expect: expectations{
				expectComputerName:  existingHostname,
				expectHostname:      existingHostname,
				expectOSVersion:     "iPadOS 17.5.1",
				expectHardwareModel: "iPad13,18",
				expectGigsTotal:     64,
				expectGigsAvailable: 51.26,
				expectDiskUpdate:    true,
				expectOSUpdate:      true,
				expectOSName:        "iPadOS",
				expectOSVersionRow:  "17.5.1",
				expectOSPlatform:    "ipados",
			},
		},
		{
			name:   "missing ProductName preserves hardware model but still writes OS using preserved platform",
			mutate: func(f map[string]string) { delete(f, "ProductName") },
			expect: expectations{
				expectComputerName:  "Work iPad",
				expectHostname:      "Work iPad",
				expectOSVersion:     "iPadOS 17.5.1", // prefix from preserved host.Platform, version from new payload
				expectHardwareModel: existingModel,   // preserved
				expectGigsTotal:     64,
				expectGigsAvailable: 51.26,
				expectDiskUpdate:    true,
				expectOSUpdate:      true,
				expectOSName:        "iPadOS",
				expectOSVersionRow:  "17.5.1",
				expectOSPlatform:    "ipados",
			},
		},
		{
			name:   "missing DeviceCapacity skips disk update",
			mutate: func(f map[string]string) { delete(f, "DeviceCapacity") },
			expect: expectations{
				expectComputerName:  "Work iPad",
				expectHostname:      "Work iPad",
				expectOSVersion:     "iPadOS 17.5.1",
				expectHardwareModel: "iPad13,18",
				expectGigsTotal:     0, // preserved/zero — no incoming value
				expectGigsAvailable: 51.26,
				expectDiskUpdate:    false,
				expectOSUpdate:      true,
				expectOSName:        "iPadOS",
				expectOSVersionRow:  "17.5.1",
				expectOSPlatform:    "ipados",
			},
		},
		{
			name:   "missing AvailableDeviceCapacity skips disk update",
			mutate: func(f map[string]string) { delete(f, "AvailableDeviceCapacity") },
			expect: expectations{
				expectComputerName:  "Work iPad",
				expectHostname:      "Work iPad",
				expectOSVersion:     "iPadOS 17.5.1",
				expectHardwareModel: "iPad13,18",
				expectGigsTotal:     64,
				expectGigsAvailable: 0,
				expectDiskUpdate:    false,
				expectOSUpdate:      true,
				expectOSName:        "iPadOS",
				expectOSVersionRow:  "17.5.1",
				expectOSPlatform:    "ipados",
			},
		},
		{
			name: "wrong-type DeviceCapacity (string) skips disk update without panicking",
			mutate: func(f map[string]string) {
				f["DeviceCapacity"] = `<key>DeviceCapacity</key><string>not-a-number</string>`
			},
			expect: expectations{
				expectComputerName:  "Work iPad",
				expectHostname:      "Work iPad",
				expectOSVersion:     "iPadOS 17.5.1",
				expectHardwareModel: "iPad13,18",
				expectGigsTotal:     0,
				expectGigsAvailable: 51.26,
				expectDiskUpdate:    false,
				expectOSUpdate:      true,
				expectOSName:        "iPadOS",
				expectOSVersionRow:  "17.5.1",
				expectOSPlatform:    "ipados",
			},
		},
		{
			name:   "missing OSVersion skips OS update",
			mutate: func(f map[string]string) { delete(f, "OSVersion") },
			expect: expectations{
				expectComputerName:  "Work iPad",
				expectHostname:      "Work iPad",
				expectOSVersion:     existingOSVersion, // preserved
				expectHardwareModel: "iPad13,18",
				expectGigsTotal:     64,
				expectGigsAvailable: 51.26,
				expectDiskUpdate:    true,
				expectOSUpdate:      false,
			},
		},
		{
			name: "all DeviceInformation fields missing yields no panic and no dependent updates",
			mutate: func(f map[string]string) {
				delete(f, "DeviceName")
				delete(f, "DeviceCapacity")
				delete(f, "AvailableDeviceCapacity")
				delete(f, "OSVersion")
				delete(f, "ProductName")
				delete(f, "WiFiMAC")
			},
			expect: expectations{
				expectComputerName:  existingHostname,
				expectHostname:      existingHostname,
				expectOSVersion:     existingOSVersion,
				expectHardwareModel: existingModel,
				expectGigsTotal:     0,
				expectGigsAvailable: 0,
				expectDiskUpdate:    false,
				expectOSUpdate:      false,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			ds := new(mock.Store)
			svc := MDMAppleCheckinAndCommandService{ds: ds, logger: slog.New(slog.DiscardHandler)}

			ds.HostByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.Host, error) {
				return &fleet.Host{
					ID:            hostID,
					UUID:          hostUUID,
					Hostname:      existingHostname,
					ComputerName:  existingHostname,
					HardwareModel: existingModel,
					OSVersion:     existingOSVersion,
					Platform:      "ipados",
					MDM: fleet.MDMHostData{
						EnrollmentStatus: ptr.String("On (automatic)"), // not Pending; skip lost-mode flow
					},
				}, nil
			}

			ds.RemoveHostMDMCommandFunc = func(ctx context.Context, command fleet.HostMDMCommand) error {
				return nil
			}
			ds.CleanupStaleNanoRefetchCommandsFunc = func(ctx context.Context, enrollmentID string, commandUUIDPrefix string, currentCommandUUID string) error {
				return nil
			}

			ds.UpdateHostFunc = func(ctx context.Context, host *fleet.Host) error {
				assert.Equal(t, tc.expect.expectComputerName, host.ComputerName, "ComputerName")
				assert.Equal(t, tc.expect.expectHostname, host.Hostname, "Hostname")
				assert.Equal(t, tc.expect.expectOSVersion, host.OSVersion, "OSVersion")
				assert.Equal(t, tc.expect.expectHardwareModel, host.HardwareModel, "HardwareModel")
				assert.InDelta(t, tc.expect.expectGigsTotal, host.GigsTotalDiskSpace, 0.001, "GigsTotalDiskSpace")
				assert.InDelta(t, tc.expect.expectGigsAvailable, host.GigsDiskSpaceAvailable, 0.001, "GigsDiskSpaceAvailable")
				return nil
			}

			ds.SetOrUpdateHostDisksSpaceFunc = func(ctx context.Context, incomingHostID uint, gigsAvailable, percentAvailable, gigsTotal float64, gigsAll *float64) error {
				assert.Equal(t, hostID, incomingHostID)
				assert.InDelta(t, tc.expect.expectGigsAvailable, gigsAvailable, 0.001)
				assert.InDelta(t, tc.expect.expectGigsTotal, gigsTotal, 0.001)
				assert.False(t, math.IsNaN(percentAvailable), "percentAvailable should not be NaN")
				assert.False(t, math.IsInf(percentAvailable, 0), "percentAvailable should not be Inf")
				return nil
			}

			ds.UpdateHostOperatingSystemFunc = func(ctx context.Context, incomingHostID uint, hostOS fleet.OperatingSystem) error {
				assert.Equal(t, hostID, incomingHostID)
				assert.Equal(t, tc.expect.expectOSName, hostOS.Name)
				assert.Equal(t, tc.expect.expectOSVersionRow, hostOS.Version)
				assert.Equal(t, tc.expect.expectOSPlatform, hostOS.Platform)
				return nil
			}

			fields := allFields()
			tc.mutate(fields)

			require.NotPanics(t, func() {
				_, err := svc.CommandAndReportResults(
					&mdm.Request{Context: ctx},
					&mdm.CommandResults{
						Enrollment:  mdm.Enrollment{UDID: hostUUID},
						CommandUUID: commandUUID,
						Raw:         rawWithFields(fields),
					},
				)
				require.NoError(t, err)
			})

			require.True(t, ds.UpdateHostFuncInvoked, "UpdateHost should always be called")
			require.True(t, ds.RemoveHostMDMCommandFuncInvoked, "RemoveHostMDMCommand should always be called")
			assert.Equal(t, tc.expect.expectDiskUpdate, ds.SetOrUpdateHostDisksSpaceFuncInvoked,
				"SetOrUpdateHostDisksSpace invocation")
			assert.Equal(t, tc.expect.expectOSUpdate, ds.UpdateHostOperatingSystemFuncInvoked,
				"UpdateHostOperatingSystem invocation")
		})
	}
}

func TestUnmarshalAppList(t *testing.T) {
	ctx := context.Background()
	noApps := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CommandUUID</key>
	<string>c05c1a68-4127-4fde-b0da-965cbd63f88f</string>
	<key>InstalledApplicationList</key>
	<array/>
	<key>Status</key>
	<string>Acknowledged</string>
	<key>UDID</key>
	<string>00008030-000E6D623CD2202E</string>
</dict>
</plist>`)
	software, err := unmarshalAppList(ctx, noApps, "ipados_apps")
	require.NoError(t, err)
	assert.Empty(t, software)

	apps := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CommandUUID</key>
	<string>21ed54fc-0e6d-4fe3-8c4f-feca0c548ce1</string>
	<key>InstalledApplicationList</key>
	<array>
		<dict>
			<key>Identifier</key>
			<string>com.google.ios.youtube</string>
			<key>Name</key>
			<string>YouTube</string>
			<key>ShortVersion</key>
			<string>19.29.1</string>
		</dict>
		<dict>
			<key>Identifier</key>
			<string>com.evernote.iPhone.Evernote</string>
			<key>Name</key>
			<string>Evernote</string>
			<key>Installing</key>
			<false/>
			<key>ShortVersion</key>
			<string>10.98.0</string>
		</dict>
		<dict>
			<key>Identifier</key>
			<string>com.netflix.Netflix</string>
			<key>Name</key>
			<string>Netflix</string>
			<key>ShortVersion</key>
			<string>16.41.0</string>
		</dict>
	</array>
	<key>Status</key>
	<string>Acknowledged</string>
	<key>UDID</key>
	<string>00008101-001514810EA3A01E</string>
</dict>
</plist>`)
	expectedSoftware := []fleet.Software{
		{
			Name:             "YouTube",
			Version:          "19.29.1",
			Source:           "ios_apps",
			BundleIdentifier: "com.google.ios.youtube",
		},
		{
			Name:             "Evernote",
			Version:          "10.98.0",
			Source:           "ios_apps",
			BundleIdentifier: "com.evernote.iPhone.Evernote",
			Installed:        true,
		},
		{
			Name:             "Netflix",
			Version:          "16.41.0",
			Source:           "ios_apps",
			BundleIdentifier: "com.netflix.Netflix",
		},
	}
	software, err = unmarshalAppList(ctx, apps, "ios_apps")
	require.NoError(t, err)
	assert.ElementsMatch(t, expectedSoftware, software)
}

func TestShouldOSUpdateForDEPEnrollment(t *testing.T) {
	testCases := []struct {
		name                  string
		platform              string
		appleMachineInfo      fleet.MDMAppleMachineInfo
		appleOSUpdateSettings fleet.AppleOSUpdateSettings
		returnedErr           error

		expectedResult bool
		expectedErr    error
	}{
		{
			name:           "when settings not found",
			returnedErr:    newNotFoundError(),
			expectedResult: false,
		},
		{
			name:        "error getting settings",
			returnedErr: errors.New("Whoops"),
			expectedErr: errors.New("Whoops"),
		},
		{
			name:     "if platform is macOS and update_new_hosts not set",
			platform: string(fleet.MacOSPlatform),
			appleMachineInfo: fleet.MDMAppleMachineInfo{
				OSVersion: "16.0.1",
			},
			appleOSUpdateSettings: fleet.AppleOSUpdateSettings{
				UpdateNewHosts: optjson.SetBool(false),
				MinimumVersion: optjson.SetString("16.0.2"),
			},
			expectedResult: false,
		},
		{
			name:     "if platform is macOS and both update_new_hosts and minimum_version are set and host is below the minimum version",
			platform: string(fleet.MacOSPlatform),
			appleMachineInfo: fleet.MDMAppleMachineInfo{
				OSVersion: "16.0.1",
			},
			appleOSUpdateSettings: fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString("16.0.2"),
				UpdateNewHosts: optjson.SetBool(true),
			},
			expectedResult: true,
		},
		{
			name:     "if platform is macOS and both update_new_hosts and minimum_version are set and host is at the minimum version",
			platform: string(fleet.MacOSPlatform),
			appleMachineInfo: fleet.MDMAppleMachineInfo{
				OSVersion: "16.0.2",
			},
			appleOSUpdateSettings: fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString("16.0.2"),
				UpdateNewHosts: optjson.SetBool(true),
			},
			expectedResult: false,
		},
		{
			name:     "if platform is macOS and update_new_hosts is set but minimum_version is not set",
			platform: string(fleet.MacOSPlatform),
			appleMachineInfo: fleet.MDMAppleMachineInfo{
				OSVersion: "16.0.1",
			},
			appleOSUpdateSettings: fleet.AppleOSUpdateSettings{
				UpdateNewHosts: optjson.SetBool(true),
			},
			expectedResult: true,
		},
		{
			name:     "if platform is not macOS and min_version is not set",
			platform: string(fleet.IPadOSPlatform),
			appleMachineInfo: fleet.MDMAppleMachineInfo{
				OSVersion: "16.0.1",
			},
			expectedResult: false,
		},
		{
			name:     "if platform is not macOS and min_version is set and host's version is greater than min required",
			platform: string(fleet.IPadOSPlatform),
			appleMachineInfo: fleet.MDMAppleMachineInfo{
				OSVersion: "16.0.3",
			},
			appleOSUpdateSettings: fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString("16.0.2"),
				UpdateNewHosts: optjson.SetBool(false),
			},
			expectedResult: false,
		},
		{
			name:     "if platform is not macOS and min_version is set and host's version is less than min required",
			platform: string(fleet.IPadOSPlatform),
			appleMachineInfo: fleet.MDMAppleMachineInfo{
				OSVersion: "16.0.1",
			},
			appleOSUpdateSettings: fleet.AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString("16.0.2"),
				UpdateNewHosts: optjson.SetBool(false),
			},
			expectedResult: true,
		},
	}

	ctx := context.Background()
	ds := new(mock.Store)
	for _, tt := range testCases {
		ds.GetMDMAppleOSUpdatesSettingsByHostSerialFunc = func(ctx context.Context, hostSerial string) (string, *fleet.AppleOSUpdateSettings, error) {
			return tt.platform, &tt.appleOSUpdateSettings, tt.returnedErr
		}

		svc := &Service{ds: ds, logger: slog.New(slog.DiscardHandler)}

		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.shouldOSUpdateForDEPEnrollment(ctx, tt.appleMachineInfo)
			require.Equal(t, tt.expectedResult, result)
			require.Equal(t, tt.expectedErr, err)
		})
	}
}

func TestCheckMDMAppleEnrollmentWithMinimumOSVersion(t *testing.T) {
	svc, ctx, ds, _ := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})

	gdmf := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// load the test data from the file
		b, err := os.ReadFile("../mdm/apple/gdmf/testdata/gdmf.json")
		require.NoError(t, err)
		_, err = w.Write(b)
		require.NoError(t, err)
	}))
	defer gdmf.Close()
	dev_mode.SetOverride("FLEET_DEV_GDMF_URL", gdmf.URL, t)

	latestMacOSVersion := "14.6.1"
	latestMacOSBuild := "23G93"

	latestIOSVersion := "17.6.1"
	latestIOSBuild := "21G93"

	testCases := []struct {
		name           string
		machineInfo    *fleet.MDMAppleMachineInfo
		updateRequired *fleet.MDMAppleSoftwareUpdateRequiredDetails
		err            string
	}{
		{
			name: "OS version is greater than latest",
			machineInfo: &fleet.MDMAppleMachineInfo{
				MDMCanRequestSoftwareUpdate: true,
				Product:                     "Mac15,7",
				OSVersion:                   "14.6.2",
				SupplementalBuildVersion:    "IRRELEVANT",
				SoftwareUpdateDeviceID:      "J516sAP",
			},
			updateRequired: nil,
		},
		{
			name: "OS version is equal to latest",
			machineInfo: &fleet.MDMAppleMachineInfo{
				MDMCanRequestSoftwareUpdate: true,
				Product:                     "Mac15,7",
				OSVersion:                   latestMacOSVersion,
				SupplementalBuildVersion:    "IRRELEVANT",
				SoftwareUpdateDeviceID:      "J516sAP",
			},
			updateRequired: nil,
		},
		{
			name: "OS version is less than latest",
			machineInfo: &fleet.MDMAppleMachineInfo{
				MDMCanRequestSoftwareUpdate: true,
				Product:                     "Mac15,7",
				OSVersion:                   "14.4",
				SupplementalBuildVersion:    "IRRELEVANT",
				SoftwareUpdateDeviceID:      "J516sAP",
			},
			updateRequired: &fleet.MDMAppleSoftwareUpdateRequiredDetails{
				OSVersion:    latestMacOSVersion,
				BuildVersion: latestMacOSBuild,
			},
		},
		{
			name: "OS version is less than latest but MDM cannot request software update",
			machineInfo: &fleet.MDMAppleMachineInfo{
				MDMCanRequestSoftwareUpdate: false,
				Product:                     "Mac15,7",
				OSVersion:                   "14.4",
				SupplementalBuildVersion:    "IRRELEVANT",
				SoftwareUpdateDeviceID:      "J516sAP",
			},
			updateRequired: nil,
		},
		{
			name: "no match for software update device ID",
			machineInfo: &fleet.MDMAppleMachineInfo{
				MDMCanRequestSoftwareUpdate: true,
				Product:                     "Mac15,7",
				OSVersion:                   "14.4",
				SupplementalBuildVersion:    "IRRELEVANT",
				SoftwareUpdateDeviceID:      "INVALID",
			},
			updateRequired: nil,
			err:            "", // no error, allow enrollment to proceed without software update
		},
		{
			name:           "no machine info",
			machineInfo:    nil,
			updateRequired: nil,
			err:            "", // no error, allow enrollment to proceed without software update
		},
		{
			name: "cannot parse OS version",
			machineInfo: &fleet.MDMAppleMachineInfo{
				MDMCanRequestSoftwareUpdate: true,
				Product:                     "Mac15,7",
				OSVersion:                   "INVALID",
				SupplementalBuildVersion:    "IRRELEVANT",
				SoftwareUpdateDeviceID:      "J516sAP",
			},
			updateRequired: nil,
			err:            "", // no error, allow enrollment to proceed without software update
		},
	}

	// FIXME: When we have more time, this whole test is overdue for a refactor because a bunch of jank
	// came with the update new hosts settings for macOS that made the test cases more dependent on
	// subtle differences in the machine info for macOS vs non-macOS platforms and made the setup
	// more complex and harder to reason about. For now, we can get away with some nested subtests
	// to reuse the test cases for both macOS and non-macOS platforms, but ideally we would refactor
	// the function under test to separate out the platform-specific logic so that we can have
	// clearer and more focused tests for each platform without needing to have a bunch of
	// conditional logic in the test itself.
	for _, tt := range testCases {
		// Non-macOS platforms
		for _, platform := range []string{"ios", "ipados"} {

			if tt.name == "no match for software update device ID" {
				// skip this test case for non-macOS platforms since SUDeviceID is really only relevant for macOS updates
				continue
			}

			t.Run(fmt.Sprintf("%s: %s", platform, tt.name), func(t *testing.T) {
				// switch up the machine info to match the platform because test cases were
				// originally written with macOS in mind
				var product, osVersion, suDeviceID string
				var mi *fleet.MDMAppleMachineInfo
				if tt.machineInfo != nil {
					osVersion = strings.Replace(tt.machineInfo.OSVersion, "14", "17", 1)
					if platform == "ios" {
						product = "iPhone16,2"
						suDeviceID = strings.Replace(tt.machineInfo.SoftwareUpdateDeviceID, "J516sAP", "iPhone", 1)
					} else {
						product = "iPad14,11"
						suDeviceID = strings.Replace(tt.machineInfo.SoftwareUpdateDeviceID, "J516sAP", "iPad", 1)
					}

					mi = &fleet.MDMAppleMachineInfo{
						MDMCanRequestSoftwareUpdate: tt.machineInfo.MDMCanRequestSoftwareUpdate,
						Product:                     product,
						OSVersion:                   osVersion,
						SupplementalBuildVersion:    tt.machineInfo.SupplementalBuildVersion,
						SoftwareUpdateDeviceID:      suDeviceID,
					}
				}
				// same for update required details
				var details *fleet.MDMAppleSoftwareUpdateRequiredDetails
				if tt.updateRequired != nil {
					details = &fleet.MDMAppleSoftwareUpdateRequiredDetails{
						OSVersion:    latestIOSVersion,
						BuildVersion: latestIOSBuild,
					}
				}

				t.Run("settings minimum equal to latest", func(t *testing.T) {
					ds.GetMDMAppleOSUpdatesSettingsByHostSerialFunc = func(ctx context.Context, serial string) (string, *fleet.AppleOSUpdateSettings, error) {
						return platform, &fleet.AppleOSUpdateSettings{
							MinimumVersion: optjson.SetString(latestIOSVersion),
						}, nil
					}
					sur, err := svc.CheckMDMAppleEnrollmentWithMinimumOSVersion(ctx, mi)
					if tt.err != "" {
						require.Error(t, err)
						require.Contains(t, err.Error(), tt.err)
					} else {
						require.NoError(t, err)
					}
					if tt.updateRequired != nil {
						require.Equal(t, &fleet.MDMAppleSoftwareUpdateRequired{
							Code:    fleet.MDMAppleSoftwareUpdateRequiredCode,
							Details: *details,
						}, sur)
					} else {
						require.Nil(t, sur)
					}
				})

				t.Run("settings minimum below latest", func(t *testing.T) {
					ds.GetMDMAppleOSUpdatesSettingsByHostSerialFunc = func(ctx context.Context, serial string) (string, *fleet.AppleOSUpdateSettings, error) {
						return platform, &fleet.AppleOSUpdateSettings{
							MinimumVersion: optjson.SetString("17.5"),
						}, nil
					}
					sur, err := svc.CheckMDMAppleEnrollmentWithMinimumOSVersion(ctx, mi)
					if tt.err != "" {
						require.Error(t, err)
						require.Contains(t, err.Error(), tt.err)
					} else {
						require.NoError(t, err)
					}
					if tt.updateRequired != nil {
						require.Equal(t, &fleet.MDMAppleSoftwareUpdateRequired{
							Code:    fleet.MDMAppleSoftwareUpdateRequiredCode,
							Details: *details,
						}, sur)
					} else {
						require.Nil(t, sur)
					}
				})

				t.Run("settings minimum above latest", func(t *testing.T) {
					// edge case, but in practice it would get treated as if minimum was equal to latest
					ds.GetMDMAppleOSUpdatesSettingsByHostSerialFunc = func(ctx context.Context, serial string) (string, *fleet.AppleOSUpdateSettings, error) {
						return platform, &fleet.AppleOSUpdateSettings{
							MinimumVersion: optjson.SetString("17.7"),
						}, nil
					}
					sur, err := svc.CheckMDMAppleEnrollmentWithMinimumOSVersion(ctx, mi)
					if tt.err != "" {
						require.Error(t, err)
						require.Contains(t, err.Error(), tt.err)
					} else {
						require.NoError(t, err)
					}
					if tt.updateRequired != nil {
						require.Equal(t, &fleet.MDMAppleSoftwareUpdateRequired{
							Code:    fleet.MDMAppleSoftwareUpdateRequiredCode,
							Details: *details,
						}, sur)
					} else {
						require.Nil(t, sur)
					}
				})

				t.Run("device above settings minimum", func(t *testing.T) {
					ds.GetMDMAppleOSUpdatesSettingsByHostSerialFunc = func(ctx context.Context, serial string) (string, *fleet.AppleOSUpdateSettings, error) {
						return platform, &fleet.AppleOSUpdateSettings{
							MinimumVersion: optjson.SetString("17.1"),
						}, nil
					}
					sur, err := svc.CheckMDMAppleEnrollmentWithMinimumOSVersion(ctx, mi)
					if tt.err != "" {
						require.Error(t, err)
						require.Contains(t, err.Error(), tt.err)
					} else {
						require.NoError(t, err)
					}

					require.Nil(t, sur)
				})

				t.Run("minimum not set", func(t *testing.T) {
					ds.GetMDMAppleOSUpdatesSettingsByHostSerialFunc = func(ctx context.Context, serial string) (string, *fleet.AppleOSUpdateSettings, error) {
						return platform, &fleet.AppleOSUpdateSettings{}, nil
					}
					sur, err := svc.CheckMDMAppleEnrollmentWithMinimumOSVersion(ctx, mi)
					require.NoError(t, err)
					require.Nil(t, sur)

					ds.GetMDMAppleOSUpdatesSettingsByHostSerialFunc = func(ctx context.Context, serial string) (string, *fleet.AppleOSUpdateSettings, error) {
						return platform, &fleet.AppleOSUpdateSettings{
							MinimumVersion: optjson.SetString(""),
						}, nil
					}
					sur, err = svc.CheckMDMAppleEnrollmentWithMinimumOSVersion(ctx, mi)
					require.NoError(t, err)
					require.Nil(t, sur)
				})

				t.Run("minimum not found", func(t *testing.T) {
					ds.GetMDMAppleOSUpdatesSettingsByHostSerialFunc = func(ctx context.Context, serial string) (string, *fleet.AppleOSUpdateSettings, error) {
						return platform, nil, &notFoundError{}
					}
					sur, err := svc.CheckMDMAppleEnrollmentWithMinimumOSVersion(ctx, mi)
					require.NoError(t, err)
					require.Nil(t, sur)
				})
			})
		}
	}

	for _, tt := range testCases {
		t.Run(fmt.Sprintf("%s for macOS", tt.name), func(t *testing.T) {
			t.Run("when UpdateNewHosts is not set should never update", func(t *testing.T) {
				ds.GetMDMAppleOSUpdatesSettingsByHostSerialFunc = func(ctx context.Context, serial string) (string, *fleet.AppleOSUpdateSettings, error) {
					return "darwin", &fleet.AppleOSUpdateSettings{
						UpdateNewHosts: optjson.SetBool(false),
						MinimumVersion: optjson.SetString(latestMacOSVersion),
					}, nil
				}
				sur, err := svc.CheckMDMAppleEnrollmentWithMinimumOSVersion(ctx, tt.machineInfo)
				require.NoError(t, err)
				require.Nil(t, sur)
			})

			t.Run("when UpdateNewHosts is set and minimum is not set", func(t *testing.T) {
				// test with min version not present
				ds.GetMDMAppleOSUpdatesSettingsByHostSerialFunc = func(ctx context.Context, serial string) (string, *fleet.AppleOSUpdateSettings, error) {
					return "darwin", &fleet.AppleOSUpdateSettings{
						UpdateNewHosts: optjson.SetBool(true),
					}, nil
				}
				sur, err := svc.CheckMDMAppleEnrollmentWithMinimumOSVersion(ctx, tt.machineInfo)
				require.NoError(t, err)

				// min version is not important for determining whether an update is required so the logic is based on
				// the installed version only
				if tt.updateRequired != nil {
					require.Equal(t, &fleet.MDMAppleSoftwareUpdateRequired{
						Code:    fleet.MDMAppleSoftwareUpdateRequiredCode,
						Details: *tt.updateRequired,
					}, sur)
				} else {
					require.Nil(t, sur)
				}

				// test again with min version explicitly set to empty string
				ds.GetMDMAppleOSUpdatesSettingsByHostSerialFunc = func(ctx context.Context, serial string) (string, *fleet.AppleOSUpdateSettings, error) {
					return "darwin", &fleet.AppleOSUpdateSettings{
						UpdateNewHosts: optjson.SetBool(true),
						MinimumVersion: optjson.SetString(""),
					}, nil
				}
				sur, err = svc.CheckMDMAppleEnrollmentWithMinimumOSVersion(ctx, tt.machineInfo)
				require.NoError(t, err)

				// Ditto previous comment
				if tt.updateRequired != nil {
					require.Equal(t, &fleet.MDMAppleSoftwareUpdateRequired{
						Code:    fleet.MDMAppleSoftwareUpdateRequiredCode,
						Details: *tt.updateRequired,
					}, sur)
				} else {
					require.Nil(t, sur)
				}
			})

			t.Run("when apple OS settings not found", func(t *testing.T) {
				ds.GetMDMAppleOSUpdatesSettingsByHostSerialFunc = func(ctx context.Context, serial string) (string, *fleet.AppleOSUpdateSettings, error) {
					return "darwin", nil, &notFoundError{}
				}
				// never block enrollment when settings are not found
				sur, err := svc.CheckMDMAppleEnrollmentWithMinimumOSVersion(ctx, tt.machineInfo)
				require.NoError(t, err)
				require.Nil(t, sur)
			})

			t.Run("when UpdateNewHosts is set and required minimum is equal to latest", func(t *testing.T) {
				ds.GetMDMAppleOSUpdatesSettingsByHostSerialFunc = func(ctx context.Context, serial string) (string, *fleet.AppleOSUpdateSettings, error) {
					return "darwin", &fleet.AppleOSUpdateSettings{
						UpdateNewHosts: optjson.SetBool(true),
						MinimumVersion: optjson.SetString(latestMacOSVersion),
					}, nil
				}
				sur, err := svc.CheckMDMAppleEnrollmentWithMinimumOSVersion(ctx, tt.machineInfo)
				if tt.err != "" {
					require.Error(t, err)
					require.Contains(t, err.Error(), tt.err)
				} else {
					require.NoError(t, err)
				}
				if tt.updateRequired != nil {
					require.Equal(t, &fleet.MDMAppleSoftwareUpdateRequired{
						Code:    fleet.MDMAppleSoftwareUpdateRequiredCode,
						Details: *tt.updateRequired,
					}, sur)
				} else {
					require.Nil(t, sur)
				}
			})

			t.Run("when UpdateNewHosts is set and required minimum is less than latest", func(t *testing.T) {
				ds.GetMDMAppleOSUpdatesSettingsByHostSerialFunc = func(ctx context.Context, serial string) (string, *fleet.AppleOSUpdateSettings, error) {
					return "darwin", &fleet.AppleOSUpdateSettings{
						UpdateNewHosts: optjson.SetBool(true),
						MinimumVersion: optjson.SetString("14.5"),
					}, nil
				}
				sur, err := svc.CheckMDMAppleEnrollmentWithMinimumOSVersion(ctx, tt.machineInfo)
				if tt.err != "" {
					require.Error(t, err)
					require.Contains(t, err.Error(), tt.err)
				} else {
					require.NoError(t, err)
				}
				if tt.updateRequired != nil {
					require.Equal(t, &fleet.MDMAppleSoftwareUpdateRequired{
						Code:    fleet.MDMAppleSoftwareUpdateRequiredCode,
						Details: *tt.updateRequired,
					}, sur)
				} else {
					require.Nil(t, sur)
				}
			})
		})
	}

	t.Run("gdmf server is down", func(t *testing.T) {
		gdmf.Close()

		for _, tt := range testCases {
			t.Run(tt.name, func(t *testing.T) {
				ds.GetMDMAppleOSUpdatesSettingsByHostSerialFunc = func(ctx context.Context, serial string) (string, *fleet.AppleOSUpdateSettings, error) {
					return "macos", &fleet.AppleOSUpdateSettings{MinimumVersion: optjson.SetString(latestMacOSVersion), UpdateNewHosts: optjson.SetBool(true)}, nil
				}

				sur, err := svc.CheckMDMAppleEnrollmentWithMinimumOSVersion(ctx, tt.machineInfo)
				if tt.err != "" {
					require.Error(t, err)
					require.Contains(t, err.Error(), tt.err) // still can get errors parsing the versions from the device info header or config settings
				} else {
					require.NoError(t, err)
				}

				require.Nil(t, sur) // if gdmf server is down, we don't enforce os updates for DEP
			})
		}
	})
}

func TestValidateConfigProfileFleetVariablesLicense(t *testing.T) {
	t.Parallel()
	profileWithVars := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadDescription</key>
	<string>Test profile with Fleet variable</string>
	<key>PayloadDisplayName</key>
	<string>Test Profile</string>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>ComputerName</key>
			<string>$FLEET_VAR_HOST_END_USER_EMAIL_IDP</string>
		</dict>
	</array>
</dict>
</plist>`

	// Test with free license
	freeLic := &fleet.LicenseInfo{Tier: fleet.TierFree}
	_, err := validateConfigProfileFleetVariables(profileWithVars, freeLic, nil)
	require.ErrorIs(t, err, fleet.ErrMissingLicense)

	// Test with premium license
	premiumLic := &fleet.LicenseInfo{Tier: fleet.TierPremium}
	vars, err := validateConfigProfileFleetVariables(profileWithVars, premiumLic, &fleet.GroupedCertificateAuthorities{})
	require.NoError(t, err)
	require.Contains(t, vars, "HOST_END_USER_EMAIL_IDP")

	// Test profile without variables (should work with free license)
	profileNoVars := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadDescription</key>
	<string>Test profile without Fleet variables</string>
	<key>PayloadDisplayName</key>
	<string>Test Profile</string>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>ComputerName</key>
			<string>StaticValue</string>
		</dict>
	</array>
</dict>
</plist>`
	vars, err = validateConfigProfileFleetVariables(profileNoVars, freeLic, &fleet.GroupedCertificateAuthorities{})
	require.NoError(t, err)
	require.Empty(t, vars)
}

func TestValidateConfigProfileFleetVariables(t *testing.T) {
	t.Parallel()
	groupedCAs := &fleet.GroupedCertificateAuthorities{
		DigiCert: []fleet.DigiCertCA{
			newMockDigicertCA("https://example.com", "caName"),
			newMockDigicertCA("https://example.com", "caName2"),
		},
		CustomScepProxy: []fleet.CustomSCEPProxyCA{
			newMockCustomSCEPProxyCA("https://example.com", "scepName"),
			newMockCustomSCEPProxyCA("https://example.com", "scepName2"),
		},
		Smallstep: []fleet.SmallstepSCEPProxyCA{
			newMockSmallstepSCEPProxyCA("https://example.com", "https://example.com/challenge", "smallstepName"),
			newMockSmallstepSCEPProxyCA("https://example.com", "https://example.com/challenge", "smallstepName2"),
		},
	}

	cases := []struct {
		name    string
		profile string
		errMsg  string
		vars    []string
	}{
		{
			name: "DigiCert profile is not pkcs12",
			profile: digiCertForValidation("$FLEET_VAR_DIGICERT_PASSWORD_caName", "$FLEET_VAR_DIGICERT_DATA_caName", "Name",
				"com.apple.security.pkcs13"),
			errMsg: "Variables $FLEET_VAR_DIGICERT_PASSWORD_caName and $FLEET_VAR_DIGICERT_DATA_caName can only be included in the 'com.apple.security.pkcs12' payload",
		},
		{
			name: "DigiCert password is not a fleet variable",
			profile: digiCertForValidation("x$FLEET_VAR_DIGICERT_PASSWORD_caName", "${FLEET_VAR_DIGICERT_DATA_caName}", "Name",
				"com.apple.security.pkcs12"),
			errMsg: "included in the 'com.apple.security.pkcs12' payload under Password and PayloadContent, respectively",
		},
		{
			name: "DigiCert data is not a fleet variable",
			profile: digiCertForValidation("${FLEET_VAR_DIGICERT_PASSWORD_caName}", "x${FLEET_VAR_DIGICERT_DATA_caName}", "Name",
				"com.apple.security.pkcs12"),
			errMsg: "Failed to parse PKCS12 payload with Fleet variables",
		},
		{
			name: "DigiCert happy path",
			profile: digiCertForValidation("${FLEET_VAR_DIGICERT_PASSWORD_caName}", "${FLEET_VAR_DIGICERT_DATA_caName}", "Name",
				"com.apple.security.pkcs12"),
			errMsg: "",
			vars:   []string{"DIGICERT_PASSWORD_caName", "DIGICERT_DATA_caName"},
		},
		{
			name: "DigiCert 2 profiles with swapped variables",
			profile: digiCertForValidation2("${FLEET_VAR_DIGICERT_PASSWORD_caName}", "${FLEET_VAR_DIGICERT_DATA_caName2}",
				"$FLEET_VAR_DIGICERT_PASSWORD_caName2", "$FLEET_VAR_DIGICERT_DATA_caName"),
			errMsg: "CA name mismatch between $FLEET_VAR_DIGICERT_PASSWORD_caName",
		},
		{
			name: "DigiCert 2 profiles happy path",
			profile: digiCertForValidation2("${FLEET_VAR_DIGICERT_PASSWORD_caName}", "${FLEET_VAR_DIGICERT_DATA_caName}",
				"$FLEET_VAR_DIGICERT_PASSWORD_caName2", "$FLEET_VAR_DIGICERT_DATA_caName2"),
			errMsg: "",
			vars:   []string{"DIGICERT_PASSWORD_caName", "DIGICERT_DATA_caName", "DIGICERT_PASSWORD_caName2", "DIGICERT_DATA_caName2"},
		},
		{
			name: "Custom SCEP renewal ID shows up in the wrong place",
			profile: customSCEPForValidationWithoutRenewalID("$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_scepName", "$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_scepName",
				"$FLEET_VAR_SCEP_RENEWAL_ID",
				"com.apple.security.scep"),
			errMsg: "Variable $FLEET_VAR_SCEP_RENEWAL_ID must be in the SCEP certificate's organizational unit (OU).",
		},
		{
			name: "Custom SCEP profile is not scep",
			profile: customSCEPForValidation("$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_scepName", "$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_scepName",
				"Name", "com.apple.security.SCEP"),
			errMsg: fleet.SCEPVariablesNotInSCEPPayloadErrMsg,
		},
		{
			name: "Custom SCEP challenge is not a fleet variable",
			profile: customSCEPForValidation("x$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_scepName", "${FLEET_VAR_CUSTOM_SCEP_PROXY_URL_scepName}",
				"Name", "com.apple.security.scep"),
			errMsg: "must be in the SCEP certificate's \"Challenge\" field",
		},
		{
			name: "Custom SCEP url is not a fleet variable",
			profile: customSCEPForValidation("${FLEET_VAR_CUSTOM_SCEP_CHALLENGE_scepName}", "x${FLEET_VAR_CUSTOM_SCEP_PROXY_URL_scepName}",
				"Name", "com.apple.security.scep"),
			errMsg: "must be in the SCEP certificate's \"URL\" field",
		},
		{
			name: "Custom SCEP happy path",
			profile: customSCEPForValidation("${FLEET_VAR_CUSTOM_SCEP_CHALLENGE_scepName}", "${FLEET_VAR_CUSTOM_SCEP_PROXY_URL_scepName}",
				"Name", "com.apple.security.scep"),
			errMsg: "",
			vars:   []string{"CUSTOM_SCEP_CHALLENGE_scepName", "CUSTOM_SCEP_PROXY_URL_scepName", "SCEP_RENEWAL_ID"},
		},
		{
			name: "Custom SCEP happy path with OU renewal ID",
			profile: customSCEPWithOURenewalIDForValidation("${FLEET_VAR_CUSTOM_SCEP_CHALLENGE_scepName}", "${FLEET_VAR_CUSTOM_SCEP_PROXY_URL_scepName}",
				"Name", "com.apple.security.scep"),
			errMsg: "",
			vars:   []string{"CUSTOM_SCEP_CHALLENGE_scepName", "CUSTOM_SCEP_PROXY_URL_scepName", "SCEP_RENEWAL_ID"},
		},
		{
			name: "Custom SCEP 2 profiles with swapped variables",
			profile: customSCEPForValidation2("${FLEET_VAR_CUSTOM_SCEP_CHALLENGE_scepName2}", "${FLEET_VAR_CUSTOM_SCEP_PROXY_URL_scepName}",
				"$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_scepName", "$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_scepName2"),
			errMsg: fleet.MultipleSCEPPayloadsErrMsg,
		},
		{
			name: "Custom SCEP 2 valid profiles should error",
			profile: customSCEPForValidation2("${FLEET_VAR_CUSTOM_SCEP_CHALLENGE_scepName}", "${FLEET_VAR_CUSTOM_SCEP_PROXY_URL_scepName}",
				"challenge", "http://example2.com"),
			errMsg: fleet.MultipleSCEPPayloadsErrMsg,
		},
		{
			name:    "Custom SCEP and DigiCert profiles happy path",
			profile: customSCEPDigiCertForValidation("${FLEET_VAR_CUSTOM_SCEP_CHALLENGE_scepName}", "${FLEET_VAR_CUSTOM_SCEP_PROXY_URL_scepName}"),
			errMsg:  "",
			vars:    []string{"DIGICERT_PASSWORD_caName", "DIGICERT_DATA_caName", "CUSTOM_SCEP_CHALLENGE_scepName", "CUSTOM_SCEP_PROXY_URL_scepName", "SCEP_RENEWAL_ID"},
		},
		{
			name:    "Custom profile with IdP variables and unknown variable",
			profile: customProfileForValidation("$FLEET_VAR_HOST_END_USER_IDP_NO_SUCH_VAR"),
			errMsg:  "Fleet variable $FLEET_VAR_HOST_END_USER_IDP_NO_SUCH_VAR is not supported in configuration profiles.",
		},
		{
			name:    "Custom profile with IdP variables happy path",
			profile: customProfileForValidation("value"),
			errMsg:  "",
			vars: []string{
				"HOST_END_USER_IDP_USERNAME",
				"HOST_END_USER_IDP_USERNAME_LOCAL_PART",
				"HOST_END_USER_IDP_GROUPS",
				"HOST_END_USER_IDP_DEPARTMENT",
			},
		},
		{
			name: "Custom SCEP and NDES 2 valid profiles should error",
			profile: customSCEPForValidation2("${FLEET_VAR_CUSTOM_SCEP_CHALLENGE_scepName}", "${FLEET_VAR_CUSTOM_SCEP_PROXY_URL_scepName}",
				"$FLEET_VAR_NDES_SCEP_CHALLENGE", "$FLEET_VAR_NDES_SCEP_PROXY_URL"),
			errMsg: fleet.MultipleSCEPPayloadsErrMsg,
		},
		{
			name: "NDES renewal ID shows up in the wrong place",
			profile: customSCEPForValidationWithoutRenewalID("$FLEET_VAR_NDES_SCEP_CHALLENGE", "$FLEET_VAR_NDES_SCEP_PROXY_URL",
				"$FLEET_VAR_SCEP_RENEWAL_ID",
				"com.apple.security.scep"),
			errMsg: "Variable $FLEET_VAR_SCEP_RENEWAL_ID must be in the SCEP certificate's organizational unit (OU).",
		},
		{
			name: "NDES profile is not scep",
			profile: customSCEPForValidation("$FLEET_VAR_NDES_SCEP_CHALLENGE", "$FLEET_VAR_NDES_SCEP_PROXY_URL",
				"Name", "com.apple.security.SCEP"),
			errMsg: fleet.SCEPVariablesNotInSCEPPayloadErrMsg,
		},
		{
			name: "NDES challenge is not a fleet variable",
			profile: customSCEPForValidation("x$FLEET_VAR_NDES_SCEP_CHALLENGE", "${FLEET_VAR_NDES_SCEP_PROXY_URL}",
				"Name", "com.apple.security.scep"),
			errMsg: "Variable \"$FLEET_VAR_NDES_SCEP_CHALLENGE\" must be in the SCEP certificate's \"Challenge\" field.",
		},
		{
			name: "NDES url is not a fleet variable",
			profile: customSCEPForValidation("${FLEET_VAR_NDES_SCEP_CHALLENGE}", "x${FLEET_VAR_NDES_SCEP_PROXY_URL}",
				"Name", "com.apple.security.scep"),
			errMsg: "Variable \"$FLEET_VAR_NDES_SCEP_PROXY_URL\" must be in the SCEP certificate's \"URL\" field.",
		},
		{
			name: "SCEP renewal ID without other variables",
			profile: customSCEPForValidation("challenge", "url",
				"Name", "com.apple.security.scep"),
			errMsg: fleet.SCEPRenewalIDWithoutURLChallengeErrMsg,
		},
		{
			name: "NDES happy path",
			profile: customSCEPForValidation("${FLEET_VAR_NDES_SCEP_CHALLENGE}", "${FLEET_VAR_NDES_SCEP_PROXY_URL}",
				"Name", "com.apple.security.scep"),
			errMsg: "",
			vars:   []string{"NDES_SCEP_CHALLENGE", "NDES_SCEP_PROXY_URL", "SCEP_RENEWAL_ID"},
		},
		{
			name: "NDES happy path with OU renewal ID",
			profile: customSCEPWithOURenewalIDForValidation("${FLEET_VAR_NDES_SCEP_CHALLENGE}", "${FLEET_VAR_NDES_SCEP_PROXY_URL}",
				"Name", "com.apple.security.scep"),
			errMsg: "",
			vars:   []string{"NDES_SCEP_CHALLENGE", "NDES_SCEP_PROXY_URL", "SCEP_RENEWAL_ID"},
		},
		{
			name: "NDES 2 valid profiles should error",
			profile: customSCEPForValidation2("${FLEET_VAR_NDES_SCEP_CHALLENGE}", "${FLEET_VAR_NDES_SCEP_PROXY_URL}",
				"challenge", "http://example2.com"),
			errMsg: fleet.MultipleSCEPPayloadsErrMsg,
		},
		{
			name:    "NDES and DigiCert profiles happy path",
			profile: customSCEPDigiCertForValidation("${FLEET_VAR_NDES_SCEP_CHALLENGE}", "${FLEET_VAR_NDES_SCEP_PROXY_URL}"),
			errMsg:  "",
			vars: []string{
				"DIGICERT_PASSWORD_caName", "DIGICERT_DATA_caName", "NDES_SCEP_CHALLENGE", "NDES_SCEP_PROXY_URL",
				"SCEP_RENEWAL_ID",
			},
		},
		{
			name: "Smallstep profile is not scep",
			profile: customSCEPForValidation("$FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_smallstepName", "$FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_smallstepName",
				"Name", "com.apple.security.SCEP"),
			errMsg: fleet.SCEPVariablesNotInSCEPPayloadErrMsg,
		},
		{
			name: "Smallstep happy path",
			profile: customSCEPForValidation("${FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_smallstepName}", "${FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_smallstepName}",
				"Name", "com.apple.security.scep"),
			errMsg: "",
			vars:   []string{"SMALLSTEP_SCEP_CHALLENGE_smallstepName", "SMALLSTEP_SCEP_PROXY_URL_smallstepName", "SCEP_RENEWAL_ID"},
		},
		{
			name: "Smallstep happy path with OU renewal ID",
			profile: customSCEPWithOURenewalIDForValidation("${FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_smallstepName}", "${FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_smallstepName}",
				"Name", "com.apple.security.scep"),
			errMsg: "",
			vars:   []string{"SMALLSTEP_SCEP_CHALLENGE_smallstepName", "SMALLSTEP_SCEP_PROXY_URL_smallstepName", "SCEP_RENEWAL_ID"},
		},
		{
			name: "Smallstep 2 profiles with swapped variables",
			profile: customSCEPForValidation2("${FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_smallstepName2}", "${FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_smallstepName}",
				"$FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_smallstepName", "$FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_smallstepName2"),
			errMsg: fleet.MultipleSCEPPayloadsErrMsg,
		},
		{
			name: "Smallstep 2 valid profiles should error",
			profile: customSCEPForValidation2("${FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_smallstepName}", "${FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_smallstepName}",
				"challenge", "http://example2.com"),
			errMsg: fleet.MultipleSCEPPayloadsErrMsg,
		},
		{
			name: "Smallstep renewal ID shows up in the wrong place",
			profile: customSCEPForValidationWithoutRenewalID("$FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_smallstepName", "$FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_smallstepName",
				"$FLEET_VAR_SCEP_RENEWAL_ID",
				"com.apple.security.scep"),
			errMsg: "Variable $FLEET_VAR_SCEP_RENEWAL_ID must be in the SCEP certificate's organizational unit (OU).",
		},
		{
			name: "Smallstep renewal ID in both CN and OU",
			profile: customSCEPWithOURenewalIDForValidation("${FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_smallstepName}", "${FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_smallstepName}",
				"Name $FLEET_VAR_SCEP_RENEWAL_ID", "com.apple.security.scep"),
			errMsg: "Variable $FLEET_VAR_SCEP_RENEWAL_ID must be in the SCEP certificate's organizational unit (OU).",
		},
		{
			name: "Smallstep challenge is not a fleet variable",
			profile: customSCEPForValidation("x$FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_smallstepName", "${FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_smallstepName}",
				"Name", "com.apple.security.scep"),
			errMsg: "Variable \"$FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_smallstepName\" must be in the SCEP certificate's \"Challenge\" field.",
		},
		{
			name: "Smallstep url is not a fleet variable",
			profile: customSCEPForValidation("${FLEET_VAR_SMALLSTEP_SCEP_CHALLENGE_smallstepName}", "x${FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_smallstepName}",
				"Name", "com.apple.security.scep"),
			errMsg: "Variable \"$FLEET_VAR_SMALLSTEP_SCEP_PROXY_URL_smallstepName\" must be in the SCEP certificate's \"URL\" field.",
		},
		{
			name: "Custom profile with IdP full name var",
			profile: string(scopedMobileconfigForTest(
				"FullName Var",
				"com.example.fullname",
				nil,
				"HOST_END_USER_IDP_FULL_NAME", // will be prefixed to $FLEET_VAR_ by helper
			)),
			errMsg: "",
			vars:   []string{"HOST_END_USER_IDP_FULL_NAME"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Pass a premium license for testing (we're not testing license validation here)
			premiumLic := &fleet.LicenseInfo{Tier: fleet.TierPremium}
			vars, err := validateConfigProfileFleetVariables(tc.profile, premiumLic, groupedCAs)
			if tc.errMsg != "" {
				assert.ErrorContains(t, err, tc.errMsg)
				assert.Empty(t, vars)
			} else {
				assert.NoError(t, err)
				assert.ElementsMatch(t, tc.vars, vars)
			}
		})
	}
}

func TestValidateDeclarationFleetVariables(t *testing.T) {
	t.Parallel()

	premiumLic := &fleet.LicenseInfo{Tier: fleet.TierPremium}
	freeLic := &fleet.LicenseInfo{Tier: fleet.TierFree}

	// helper to create a simple DDM declaration JSON with a value field
	makeDecl := func(value string) string {
		return fmt.Sprintf(`{"Type": "com.apple.configuration.management.test", "Identifier": "com.example.test", "Payload": {"Value": %q}}`, value)
	}

	t.Run("no variables, free license", func(t *testing.T) {
		vars, err := validateDeclarationFleetVariables(makeDecl("static-value"), freeLic)
		require.NoError(t, err)
		require.Nil(t, vars)
	})

	t.Run("supported variable with premium license", func(t *testing.T) {
		vars, err := validateDeclarationFleetVariables(makeDecl("$FLEET_VAR_HOST_HARDWARE_SERIAL"), premiumLic)
		require.NoError(t, err)
		require.Equal(t, []string{"HOST_HARDWARE_SERIAL"}, vars)
	})

	t.Run("supported variable with braces", func(t *testing.T) {
		vars, err := validateDeclarationFleetVariables(makeDecl("${FLEET_VAR_HOST_UUID}"), premiumLic)
		require.NoError(t, err)
		require.Equal(t, []string{"HOST_UUID"}, vars)
	})

	t.Run("multiple supported variables", func(t *testing.T) {
		vars, err := validateDeclarationFleetVariables(
			makeDecl(`["$FLEET_VAR_HOST_HARDWARE_SERIAL", "$FLEET_VAR_HOST_END_USER_IDP_USERNAME"]`), premiumLic)
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"HOST_HARDWARE_SERIAL", "HOST_END_USER_IDP_USERNAME"}, vars)
	})

	t.Run("all supported variables", func(t *testing.T) {
		// Build the declaration content and expected results from the allowed list
		var jsonVars, expectedVars []string
		for _, v := range fleetVarsSupportedInDDMDeclarations {
			jsonVars = append(jsonVars, fmt.Sprintf(`"$FLEET_VAR_%s"`, v))
			expectedVars = append(expectedVars, string(v))
		}
		vars, err := validateDeclarationFleetVariables(
			makeDecl("["+strings.Join(jsonVars, ", ")+"]"), premiumLic)
		require.NoError(t, err)
		require.ElementsMatch(t, expectedVars, vars)
	})

	t.Run("supported variable without premium license", func(t *testing.T) {
		_, err := validateDeclarationFleetVariables(makeDecl("$FLEET_VAR_HOST_HARDWARE_SERIAL"), freeLic)
		require.ErrorIs(t, err, fleet.ErrMissingLicense)
	})

	t.Run("supported variable with nil license", func(t *testing.T) {
		_, err := validateDeclarationFleetVariables(makeDecl("$FLEET_VAR_HOST_UUID"), nil)
		require.ErrorIs(t, err, fleet.ErrMissingLicense)
	})

	t.Run("unsupported variable", func(t *testing.T) {
		_, err := validateDeclarationFleetVariables(makeDecl("$FLEET_VAR_NDES_SCEP_CHALLENGE"), premiumLic)
		require.Error(t, err)
		require.ErrorContains(t, err, "Fleet variable $FLEET_VAR_NDES_SCEP_CHALLENGE is not supported in DDM profiles")
	})

	t.Run("supported and unsupported variables", func(t *testing.T) {
		_, err := validateDeclarationFleetVariables(
			makeDecl(`["$FLEET_VAR_HOST_UUID", "$FLEET_VAR_DIGICERT_DATA_myCA"]`), premiumLic)
		require.Error(t, err)
		require.ErrorContains(t, err, "Fleet variable $FLEET_VAR_DIGICERT_DATA_myCA is not supported in DDM profiles")
	})
}

func TestJSONEscapeString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain string", "hello", "hello"},
		{"with double quotes", `say "hi"`, `say \"hi\"`},
		{"with backslash", `path\to\file`, `path\\to\\file`},
		{"with newline", "line1\nline2", `line1\nline2`},
		{"with tab", "col1\tcol2", `col1\tcol2`},
		{"empty string", "", ""},
		{"unicode", "café ☕", "café ☕"},
		{"control chars", "a\x00b", `a\u0000b`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, jsonEscapeString(tc.input))
		})
	}
}

//go:embed testdata/profiles/digicert-validation.mobileconfig
var digiCertValidationMobileconfig string

func digiCertForValidation(password, data, name, payloadType string) string {
	return fmt.Sprintf(digiCertValidationMobileconfig, password, data, name, payloadType)
}

//go:embed testdata/profiles/digicert-validation2.mobileconfig
var digiCertValidation2Mobileconfig string

func digiCertForValidation2(password1, data1, password2, data2 string) string {
	return fmt.Sprintf(digiCertValidation2Mobileconfig, password1, data1, password2, data2)
}

//go:embed testdata/profiles/custom-scep-validation.mobileconfig
var customSCEPValidationMobileconfig string

func customSCEPForValidation(challenge, url, name, payloadType string) string {
	return fmt.Sprintf(customSCEPValidationMobileconfig, challenge, url, name, payloadType)
}

func customSCEPForValidationWithoutRenewalID(challenge, url, name, payloadType string) string {
	configProfile := strings.ReplaceAll(customSCEPValidationMobileconfig, "$FLEET_VAR_SCEP_RENEWAL_ID", "")
	return fmt.Sprintf(configProfile, challenge, url, name, payloadType)
}

//go:embed testdata/profiles/custom-scep-validation2.mobileconfig
var customSCEPValidation2Mobileconfig string

func customSCEPForValidation2(challenge1, url1, challenge2, url2 string) string {
	return fmt.Sprintf(customSCEPValidation2Mobileconfig, challenge1, url1, challenge2, url2)
}

//go:embed testdata/profiles/custom-scep-validation-ourenewal.mobileconfig
var customSCEPValidationWithOURenewalIDMobileconfig string

func customSCEPWithOURenewalIDForValidation(challenge, url, name, payloadType string) string {
	return fmt.Sprintf(customSCEPValidationWithOURenewalIDMobileconfig, challenge, url, name, payloadType)
}

//go:embed testdata/profiles/custom-scep-digicert-validation.mobileconfig
var customSCEPDigiCertValidationMobileconfig string

func customSCEPDigiCertForValidation(challenge, url string) string {
	return fmt.Sprintf(customSCEPDigiCertValidationMobileconfig, challenge, url)
}

//go:embed testdata/profiles/custom-profile-validation.mobileconfig
var customProfileValidationMobileconfig string

func customProfileForValidation(value string) string {
	return fmt.Sprintf(customProfileValidationMobileconfig, value)
}

func TestParseHHMM(t *testing.T) {
	tests := []struct {
		input    string
		wantHour int
		wantMin  int
		wantErr  bool
	}{
		{"00:00", 0, 0, false},
		{"09:30", 9, 30, false},
		{"23:59", 23, 59, false},
		{"12:05", 12, 5, false},
		{"invalid", 0, 0, true},
		{"12", 0, 0, true},
		{"12:60", 0, 0, true},
		{"24:00", 0, 0, true},
		{"-01:00", 0, 0, true},
		{"12:xx", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			h, m, err := parseHHMM(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseHHMM(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err == nil && (h != tt.wantHour || m != tt.wantMin) {
				t.Fatalf("parseHHMM(%q) = %d:%d, want %02d:%02d", tt.input, h, m, tt.wantHour, tt.wantMin)
			}
		})
	}
}

func TestGetCurrentLocalTimeInHostTimeZone(t *testing.T) {
	// Save original and restore after test
	originalMock := nowFunc
	defer func() { nowFunc = originalMock }()

	// Fix "now" to a known UTC time: 2026-01-02 14:30:00 UTC
	fixedNow := time.Date(2026, 1, 2, 14, 30, 0, 0, time.UTC)
	nowFunc = func() time.Time {
		return fixedNow
	}

	tests := []struct {
		name     string
		timezone string
		wantHour int
		wantMin  int
		wantErr  bool
	}{
		{"UTC", "UTC", 14, 30, false},
		{"New York", "America/New_York", 9, 30, false},       // EST = UTC-5
		{"Los Angeles", "America/Los_Angeles", 6, 30, false}, // PST = UTC-8
		{"London", "Europe/London", 14, 30, false},           // GMT in winter
		{"Tokyo", "Asia/Tokyo", 23, 30, false},               // JST = UTC+9
		{"Invalid TZ", "Invalid/Timezone", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getCurrentLocalTimeInHostTimeZone(context.Background(), tt.timezone)
			if (err != nil) != tt.wantErr {
				t.Fatalf("getCurrentLocalTimeInHostTimeZone(%q) error = %v, wantErr %v", tt.timezone, err, tt.wantErr)
			}
			if err == nil {
				if got.Hour() != tt.wantHour || got.Minute() != tt.wantMin {
					t.Fatalf("getCurrentLocalTimeInHostTimeZone(%q) = %02d:%02d, want %02d:%02d",
						tt.timezone, got.Hour(), got.Minute(), tt.wantHour, tt.wantMin)
				}
			}
		})
	}
}

func TestIsTimezoneInWindow(t *testing.T) {
	// Save original and restore after test
	originalMock := nowFunc
	defer func() { nowFunc = originalMock }()

	// Helper to set a fixed UTC time, which will be converted to local time
	setMockUTCNow := func(year, month, day, hour, min_ int) {
		nowFunc = func() time.Time {
			return time.Date(year, time.Month(month), day, hour, min_, 0, 0, time.UTC)
		}
	}

	tests := []struct {
		name         string
		mockUTCHour  int
		mockUTCMin   int
		timezone     string
		start        string
		end          string
		wantInWindow bool
		wantErr      bool
	}{
		// Normal window (no midnight cross)
		{name: "normal inside", mockUTCHour: 14, mockUTCMin: 0, timezone: "America/New_York", start: "08:00", end: "17:00", wantInWindow: true, wantErr: false},  // 09:00 EST
		{name: "normal before", mockUTCHour: 12, mockUTCMin: 0, timezone: "America/New_York", start: "09:00", end: "17:00", wantInWindow: false, wantErr: false}, // 07:00 EST
		{name: "normal at start", mockUTCHour: 14, mockUTCMin: 0, timezone: "America/New_York", start: "09:00", end: "17:00", wantInWindow: true, wantErr: false},
		{name: "normal at end", mockUTCHour: 22, mockUTCMin: 0, timezone: "America/New_York", start: "09:00", end: "17:00", wantInWindow: true, wantErr: false}, // 17:00 EST

		// Midnight-crossing window
		{name: "too early", mockUTCHour: 4, mockUTCMin: 0, timezone: "America/Los_Angeles", start: "22:00", end: "06:00", wantInWindow: false, wantErr: false},              // 20:00 PST
		{name: "crossing late night", mockUTCHour: 7, mockUTCMin: 0, timezone: "America/Los_Angeles", start: "22:00", end: "06:00", wantInWindow: true, wantErr: false},     // 23:00 PST
		{name: "crossing early morning", mockUTCHour: 12, mockUTCMin: 0, timezone: "America/Los_Angeles", start: "22:00", end: "06:00", wantInWindow: true, wantErr: false}, // 04:00 PST
		{name: "crossing outside", mockUTCHour: 16, mockUTCMin: 0, timezone: "America/Los_Angeles", start: "22:00", end: "06:00", wantInWindow: false, wantErr: false},      // 08:00 PST

		// Edge cases
		{name: "full day", mockUTCHour: 12, mockUTCMin: 0, timezone: "UTC", start: "00:00", end: "23:59", wantInWindow: true, wantErr: false},
		{name: "single minute", mockUTCHour: 10, mockUTCMin: 5, timezone: "UTC", start: "10:05", end: "10:05", wantInWindow: true, wantErr: false},
		{name: "invalid start", mockUTCHour: 12, mockUTCMin: 0, timezone: "UTC", start: "25:00", end: "17:00", wantInWindow: false, wantErr: true},
		{name: "invalid timezone", mockUTCHour: 12, mockUTCMin: 0, timezone: "Invalid/TZ", start: "09:00", end: "17:00", wantInWindow: false, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setMockUTCNow(2026, 1, 2, tt.mockUTCHour, tt.mockUTCMin)

			got, err := isTimezoneInWindow(context.Background(), tt.timezone, tt.start, tt.end)
			if (err != nil) != tt.wantErr {
				t.Fatalf("isTimezoneInWindow(%q, %s-%s) error = %v, wantErr %v", tt.timezone, tt.start, tt.end, err, tt.wantErr)
			}
			if got != tt.wantInWindow {
				t.Fatalf("isTimezoneInWindow(%q, %s-%s) = %v, want %v", tt.timezone, tt.start, tt.end, got, tt.wantInWindow)
			}
		})
	}
}

func TestToValidSemVer(t *testing.T) {
	testVersions := []struct {
		rawVersion                           string
		expectedVersion                      string
		versionToSemverVersionExpectedToFail bool
	}{
		{
			"25.48.0",
			"25.48.0",
			false,
		},
		{
			" 353.0 ", // Meta Horizon like version.
			"353.0",
			false,
		},
		{
			"18.14.0",
			"18.14.0",
			false,
		},
		{
			"412.0.0",
			"412.0.0",
			false,
		},
		{
			"00.001010.01",
			"0.1010.1",
			false,
		},
		{
			"6.0.251229",
			"6.0.251229",
			false,
		},
		{
			"4.2602.11600",
			"4.2602.11600",
			false,
		},
		{
			"144.0.7559.53", // Google Chrome like version.
			"144.0.7559-53",
			false,
		},
		{
			"144.0.7559.03", // Google Chrome like version, leading zeros.
			"144.0.7559-3",
			false,
		},
		{
			"4.9999999999999999999999999.11600", // Not a valid semantic version, so we leave unchanged.
			"4.9999999999999999999999999.11600",
			true,
		},
		{
			"04.0000099999999999999999999999990.011600", // Not a valid semantic version, but we clean it anyway.
			"4.99999999999999999999999990.11600",
			true,
		},
		{
			"21.02.3", // YouTube like version.
			"21.2.3",
			false,
		},
		{
			"21", // Just major version.
			"21",
			false,
		},
		{
			"v2.3.4", // Remove leading v.
			"2.3.4",
			false,
		},
		{
			"02.03.04-01",
			"2.3.4-1",
			false,
		},
	}
	for _, tc := range testVersions {
		cleanedVersion := toValidSemVer(tc.rawVersion)
		require.Equal(t, tc.expectedVersion, cleanedVersion)
		_, err := fleet.VersionToSemverVersion(cleanedVersion)
		if !tc.versionToSemverVersionExpectedToFail {
			require.NoError(t, err, tc.rawVersion)
		} else {
			require.Error(t, err, tc.rawVersion)
		}
	}
}

func TestMDMTokenUpdateSCEPRenewal(t *testing.T) {
	ctx := license.NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierPremium})
	ds := new(mock.Store)
	mdmStorage := &mdmmock.MDMAppleStore{}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	pushFactory, _ := newMockAPNSPushProviderFactory()
	pusher := nanomdm_pushsvc.New(
		mdmStorage,
		mdmStorage,
		pushFactory,
		NewNanoMDMLogger(logger),
	)
	cmdr := apple_mdm.NewMDMAppleCommander(mdmStorage, pusher)
	uuid, serial, model, wantTeamID := "ABC-DEF-GHI", "XYZABC", "MacBookPro 16,1", uint(12)

	t.Run("awaiting configuration continues enrollment", func(t *testing.T) {
		// When a host re-enrolls via DEP (AwaitingConfiguration=true) while a
		// SCEP renewal is pending, the handler should clear SCEP refs and
		// continue with the normal enrollment flow (not short-circuit).

		var newActivityFuncInvoked bool
		mdmLifecycle := mdmlifecycle.New(ds, logger, func(_ context.Context, _ *fleet.User, activity fleet.ActivityDetails) error {
			newActivityFuncInvoked = true
			_, ok := activity.(*fleet.ActivityTypeMDMEnrolled)
			require.True(t, ok)
			return nil
		})
		svc := MDMAppleCheckinAndCommandService{
			ds:           ds,
			mdmLifecycle: mdmLifecycle,
			commander:    cmdr,
			logger:       logger,
		}
		scepRenewalInProgress := true
		ds.GetHostMDMCheckinInfoFunc = func(ct context.Context, hostUUID string) (*fleet.HostMDMCheckinInfo, error) {
			return &fleet.HostMDMCheckinInfo{
				HostID:                1337,
				HardwareSerial:        serial,
				DisplayName:           model,
				InstalledFromDEP:      true,
				TeamID:                wantTeamID,
				DEPAssignedToFleet:    true,
				SCEPRenewalInProgress: scepRenewalInProgress,
				Platform:              "darwin",
			}, nil
		}
		ds.CleanSCEPRenewRefsFunc = func(ctx context.Context, hostUUID string) error {
			require.Equal(t, uuid, hostUUID)
			scepRenewalInProgress = false
			return nil
		}
		ds.EnqueueSetupExperienceItemsFunc = func(ctx context.Context, hostPlatform, hostPlatformLike string, hostUUID string, teamID uint) (bool, error) {
			require.Equal(t, "darwin", hostPlatformLike)
			require.Equal(t, uuid, hostUUID)
			require.Equal(t, wantTeamID, teamID)
			return true, nil
		}
		ds.GetNanoMDMEnrollmentFunc = func(ctx context.Context, hostUUID string) (*fleet.NanoEnrollment, error) {
			return &fleet.NanoEnrollment{Enabled: true, Type: "Device", TokenUpdateTally: 1}, nil
		}
		ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{}, nil
		}
		ds.GetMDMIdPAccountByHostUUIDFunc = func(ctx context.Context, hostUUID string) (*fleet.MDMIdPAccount, error) {
			return nil, nil
		}
		ds.NewJobFunc = func(ctx context.Context, j *fleet.Job) (*fleet.Job, error) {
			return j, nil
		}
		ds.MDMResetEnrollmentFunc = func(ctx context.Context, hostUUID string, scepRenewalInProgress bool) error {
			return nil
		}
		ds.ClearHostEnrolledFromMigrationFunc = func(ctx context.Context, hostUUID string) error {
			require.Equal(t, uuid, hostUUID)
			return nil
		}
		ds.MDMAppleResetOnReenrollmentFunc = func(ctx context.Context, hostUUID string, preserveHostActivities bool) error {
			return nil
		}

		err := svc.TokenUpdate(
			&mdm.Request{Context: ctx, EnrollID: &mdm.EnrollID{ID: uuid}},
			&mdm.TokenUpdate{
				TokenUpdateEnrollment: mdm.TokenUpdateEnrollment{
					AwaitingConfiguration: true,
					Enrollment: mdm.Enrollment{
						UDID: uuid,
					},
				},
			},
		)
		require.NoError(t, err)
		require.True(t, ds.CleanSCEPRenewRefsFuncInvoked)
		require.True(t, ds.EnqueueSetupExperienceItemsFuncInvoked)
		require.True(t, ds.NewJobFuncInvoked)
		require.True(t, newActivityFuncInvoked)
		require.True(t, ds.MDMResetEnrollmentFuncInvoked)
		require.True(t, ds.ClearHostEnrolledFromMigrationFuncInvoked)
	})

	t.Run("not awaiting configuration short-circuits", func(t *testing.T) {
		// When a SCEP renewal is in progress but the host is NOT awaiting
		// configuration (normal renewal), the handler should clean SCEP refs
		// and return early without enqueueing setup experience or lifecycle.
		var newActivityFuncInvoked bool
		mdmLifecycle := mdmlifecycle.New(ds, logger, func(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error {
			newActivityFuncInvoked = true
			return nil
		})
		svc := MDMAppleCheckinAndCommandService{
			ds:           ds,
			mdmLifecycle: mdmLifecycle,
			commander:    cmdr,
			logger:       logger,
		}

		ds.GetHostMDMCheckinInfoFunc = func(ct context.Context, hostUUID string) (*fleet.HostMDMCheckinInfo, error) {
			return &fleet.HostMDMCheckinInfo{
				HostID:                1337,
				HardwareSerial:        serial,
				DisplayName:           model,
				InstalledFromDEP:      true,
				TeamID:                wantTeamID,
				DEPAssignedToFleet:    true,
				SCEPRenewalInProgress: true,
				Platform:              "darwin",
			}, nil
		}
		ds.CleanSCEPRenewRefsFunc = func(ctx context.Context, hostUUID string) error {
			require.Equal(t, uuid, hostUUID)
			return nil
		}
		ds.CleanSCEPRenewRefsFuncInvoked = false
		ds.EnqueueSetupExperienceItemsFuncInvoked = false
		ds.NewJobFuncInvoked = false

		err := svc.TokenUpdate(
			&mdm.Request{Context: ctx, EnrollID: &mdm.EnrollID{ID: uuid}},
			&mdm.TokenUpdate{
				TokenUpdateEnrollment: mdm.TokenUpdateEnrollment{
					Enrollment: mdm.Enrollment{UDID: uuid},
				},
			},
		)
		require.NoError(t, err)
		require.True(t, ds.CleanSCEPRenewRefsFuncInvoked)
		require.False(t, ds.EnqueueSetupExperienceItemsFuncInvoked)
		require.False(t, ds.NewJobFuncInvoked)
		require.False(t, newActivityFuncInvoked)
	})
}

// decodeSignedEnrollmentProfile parses a PKCS7-signed enrollment profile and
// returns the inner raw mobileconfig bytes for content assertions.
func decodeSignedEnrollmentProfile(t *testing.T, signed []byte) []byte {
	t.Helper()
	p7, err := pkcs7.Parse(signed)
	require.NoError(t, err, "parsing PKCS7 signed profile")
	require.NoError(t, p7.Verify(), "verifying PKCS7 signature")
	require.NotEmpty(t, p7.Content, "PKCS7 content should not be empty")
	return p7.Content
}

// assertSCEPProfile checks that the decoded mobileconfig XML contains a SCEP
// payload (com.apple.security.scep) and does not contain an ACME payload.
func assertSCEPProfile(t *testing.T, content []byte) {
	t.Helper()
	require.Contains(t, string(content), "com.apple.security.scep", "expected SCEP payload type in profile")
	require.NotContains(t, string(content), "com.apple.security.acme", "SCEP profile must not contain an ACME payload")
}

// assertACMEProfile checks that the decoded mobileconfig XML contains an ACME
// payload (com.apple.security.acme) and does not contain a SCEP payload.
func assertACMEProfile(t *testing.T, content []byte, deviceSerial string) {
	t.Helper()
	require.Contains(t, string(content), "com.apple.security.acme", "expected ACME payload type in profile")
	require.NotContains(t, string(content), "com.apple.security.scep", "ACME profile must not contain a SCEP payload")
	require.Contains(t, string(content), deviceSerial, "ACME profile should embed the device serial as ClientIdentifier")
}

func TestGetMDMAppleEnrollmentProfileByToken(t *testing.T) {
	svc, ctx, ds, _ := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})

	// Extend the existing asset mock to also include the SCEP challenge needed
	// by generateMDMAppleSCEPEnrollProfile.
	apnsCert, apnsKey, err := mysql.GenerateTestCertBytes(mdmtesting.NewTestMDMAppleCertTemplate())
	require.NoError(t, err)
	crt, key, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(t, err)
	certPEM := tokenpki.PEMCertificate(crt.Raw)
	keyPEM := tokenpki.PEMRSAPrivateKey(key)
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetAPNSCert:      {Value: apnsCert},
			fleet.MDMAssetAPNSKey:       {Value: apnsKey},
			fleet.MDMAssetCACert:        {Value: certPEM},
			fleet.MDMAssetCAKey:         {Value: keyPEM},
			fleet.MDMAssetSCEPChallenge: {Value: []byte("test-scep-challenge")},
		}, nil
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			OrgInfo:        fleet.OrgInfo{OrgName: "Foo Inc."},
			ServerSettings: fleet.ServerSettings{ServerURL: "https://foo.example.com"},
			MDM:            fleet.MDM{EnabledAndConfigured: true, AppleRequireHardwareAttestation: true},
		}, nil
	}

	foundProfileFunc := func(ctx context.Context, token string) (*fleet.MDMAppleEnrollmentProfile, error) {
		require.Equal(t, "valid-token", token)
		return &fleet.MDMAppleEnrollmentProfile{
			ID:    1,
			Token: "valid-token",
			// Type:  fleet.MDMAppleEnrollmentTypeManual,
			// other fields are not relevant for this test
		}, nil
	}
	ds.GetMDMAppleEnrollmentProfileByTokenFunc = foundProfileFunc

	t.Run("happy path", func(t *testing.T) {
		ds.GetMDMAppleEnrollmentProfileByTokenFunc = foundProfileFunc

		testDevices := []struct {
			name       string
			mi         fleet.MDMAppleMachineInfo
			expectACME bool // if true, the profile should contain an ACME payload when hardware attestation is required; if false, it should contain a SCEP payload
		}{
			{
				name: "Apple Silicon Mac", mi: fleet.MDMAppleMachineInfo{
					Product:   "MacBookPro18,3", // major 18 >= threshold of 17 → Apple Silicon
					Serial:    "MACSILSERIAL",
					UDID:      "mac-sil-udid",
					OSVersion: "15.0", // macOS 15 >= 14 → eligible for ACME
				},
				expectACME: true,
			},
			{
				// macOS < 14 disqualifies ACME even for Apple Silicon with DEP enrollment
				name: "Apple Silicon Mac macOS 13", mi: fleet.MDMAppleMachineInfo{
					Product:   "MacBookPro18,3", // Apple Silicon
					Serial:    "MACSILSERIAL13",
					UDID:      "mac-sil-13-udid",
					OSVersion: "13.6.0", // macOS 13 < 14 → SCEP regardless of DEP assignment
				},
				expectACME: false,
			},
			{
				// missing serial (account-driven user enrollment) disqualifies ACME regardless of device or DEP assignment
				name: "Apple Silicon Mac without serial", mi: fleet.MDMAppleMachineInfo{
					Product:   "MacBookPro18,3", // Apple Silicon
					Serial:    "",               // no serial → SCEP without error
					UDID:      "mac-sil-noserial-udid",
					OSVersion: "15.0",
				},
				expectACME: false,
			},
			{
				name: "Intel Mac",
				mi: fleet.MDMAppleMachineInfo{
					Product: "MacBookPro16,1", // major 16 < threshold of 17 → Intel
					Serial:  "INTELSERIAL",
					UDID:    "intel-udid",
				},
				expectACME: false,
			},
			{
				name: "iPhone",
				mi: fleet.MDMAppleMachineInfo{
					Product: "iPhone14,3",
					Serial:  "IPHONESERIAL",
					UDID:    "iphone-udid",
				},
				expectACME: false,
			},
		}

		t.Run("hardware attestation enabled", func(t *testing.T) {
			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{
					OrgInfo:        fleet.OrgInfo{OrgName: "Foo Inc."},
					ServerSettings: fleet.ServerSettings{ServerURL: "https://foo.example.com"},
					MDM:            fleet.MDM{EnabledAndConfigured: true, AppleRequireHardwareAttestation: true},
				}, nil
			}
			for _, device := range testDevices {
				t.Run(device.name, func(t *testing.T) {
					t.Run(fmt.Sprintf("not DEP assigned requires ACME %t", device.expectACME), func(t *testing.T) {
						ds.GetHostDEPAssignmentsBySerialFunc = func(ctx context.Context, serial string) ([]*fleet.HostDEPAssignment, error) {
							return []*fleet.HostDEPAssignment{}, nil
						}
						profile, err := svc.GetMDMAppleEnrollmentProfileByToken(ctx, "valid-token", "", &device.mi)
						require.NoError(t, err)
						// always expect SCEP if not DEP assigned, even for Apple Silicon, since we currently limit ACME to DEP enrollment
						assertSCEPProfile(t, decodeSignedEnrollmentProfile(t, profile))
					})
					t.Run(fmt.Sprintf("DEP assigned requires ACME %t", device.expectACME), func(t *testing.T) {
						ds.GetHostDEPAssignmentsBySerialFunc = func(ctx context.Context, serial string) ([]*fleet.HostDEPAssignment, error) {
							return []*fleet.HostDEPAssignment{{HostID: 1}}, nil
						}
						profile, err := svc.GetMDMAppleEnrollmentProfileByToken(ctx, "valid-token", "", &device.mi)
						require.NoError(t, err)
						if device.expectACME {
							assertACMEProfile(t, decodeSignedEnrollmentProfile(t, profile), device.mi.Serial)
						} else {
							assertSCEPProfile(t, decodeSignedEnrollmentProfile(t, profile))
						}
					})
				})
			}
		})

		t.Run("hardware attestation disabled", func(t *testing.T) {
			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{
					OrgInfo:        fleet.OrgInfo{OrgName: "Foo Inc."},
					ServerSettings: fleet.ServerSettings{ServerURL: "https://foo.example.com"},
					MDM:            fleet.MDM{EnabledAndConfigured: true},
				}, nil
			}
			for _, device := range testDevices {
				t.Run(device.name, func(t *testing.T) {
					profile, err := svc.GetMDMAppleEnrollmentProfileByToken(ctx, "valid-token", "", &device.mi)
					require.NoError(t, err)
					// always expect SCEP if hardware attestation is disabled; DEP assignment and Apple Silicon do not matter in this case
					assertSCEPProfile(t, decodeSignedEnrollmentProfile(t, profile))
				})
			}
		})
	})

	t.Run("miscellaneous error handling", func(t *testing.T) {
		// For these tests we can just use a single device type since we're not asserting on the
		// profile content, just that errors are handled and returned properly.
		machineInfo := fleet.MDMAppleMachineInfo{
			Product:   "MacBookPro18,3",
			Serial:    "MACSILSERIAL",
			UDID:      "mac-sil-udid",
			OSVersion: "15.0", // macOS 15 >= 14 so that the OSVersion check in isMDMAppleACMERequired passes through to the DEP check
		}

		t.Run("AppConfig error returns error", func(t *testing.T) {
			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return nil, errors.New("some unexpected error")
			}
			_, err := svc.GetMDMAppleEnrollmentProfileByToken(ctx, "valid-token", "", &machineInfo)
			require.Error(t, err)
		})

		// now test the various error cases for both hardware attestation enabled and disabled, since that changes the logic flow and we want to ensure all cases are covered
		for _, hardwareAttestationRequired := range []bool{true, false} {
			t.Run(fmt.Sprintf("hardware attestation required %t", hardwareAttestationRequired), func(t *testing.T) {
				ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
					return &fleet.AppConfig{
						OrgInfo:        fleet.OrgInfo{OrgName: "Foo Inc."},
						ServerSettings: fleet.ServerSettings{ServerURL: "https://foo.example.com"},
						MDM:            fleet.MDM{EnabledAndConfigured: true, AppleRequireHardwareAttestation: hardwareAttestationRequired},
					}, nil
				}
				t.Run("nil machineInfo returns error", func(t *testing.T) {
					_, err := svc.GetMDMAppleEnrollmentProfileByToken(ctx, "some-token", "", nil)
					require.Error(t, err)
					require.Contains(t, err.Error(), "missing machine info")
				})
				t.Run("token not found returns auth failed error", func(t *testing.T) {
					ds.GetMDMAppleEnrollmentProfileByTokenFunc = func(ctx context.Context, token string) (*fleet.MDMAppleEnrollmentProfile, error) {
						return nil, newNotFoundError()
					}
					_, err := svc.GetMDMAppleEnrollmentProfileByToken(ctx, "unknown-token", "", &machineInfo)
					require.Error(t, err)
					var authErr *fleet.AuthFailedError
					require.ErrorAs(t, err, &authErr)
					// restore the happy path for subsequent tests
					ds.GetMDMAppleEnrollmentProfileByTokenFunc = foundProfileFunc
				})
				t.Run("datastore error returns wrapped error", func(t *testing.T) {
					ds.GetMDMAppleEnrollmentProfileByTokenFunc = func(ctx context.Context, token string) (*fleet.MDMAppleEnrollmentProfile, error) {
						return nil, errors.New("some unexpected error")
					}
					_, err := svc.GetMDMAppleEnrollmentProfileByToken(ctx, "valid-token", "", &machineInfo)
					require.Error(t, err)
					require.Contains(t, err.Error(), "get enrollment profile")
					// restore the happy path for subsequent tests
					ds.GetMDMAppleEnrollmentProfileByTokenFunc = foundProfileFunc
				})

				t.Run("error fetching DEP assignments returns error", func(t *testing.T) {
					ds.GetHostDEPAssignmentsBySerialFunc = func(ctx context.Context, serial string) ([]*fleet.HostDEPAssignment, error) {
						return nil, errors.New("some unexpected error")
					}
					ds.GetHostDEPAssignmentsBySerialFuncInvoked = false // reset invocation flag

					_, err := svc.GetMDMAppleEnrollmentProfileByToken(ctx, "valid-token", "", &machineInfo)
					if hardwareAttestationRequired {
						// when hardware attestation is required, DEP assignment is checked before deciding on ACME vs SCEP, so an error here should be returned
						require.True(t, ds.GetHostDEPAssignmentsBySerialFuncInvoked)
						require.Error(t, err)
						require.Contains(t, err.Error(), "checking DEP assignment")
					} else {
						// when hardware attestation is not required, DEP assignment is not relevant since we always return SCEP, so an error here should not be returned
						require.False(t, ds.GetHostDEPAssignmentsBySerialFuncInvoked)
						require.NoError(t, err)
					}
				})

				t.Run("invalid OS version for Apple Silicon Mac returns error", func(t *testing.T) {
					// restore foundProfileFunc in case a previous sub-test left a different func
					ds.GetMDMAppleEnrollmentProfileByTokenFunc = foundProfileFunc
					ds.GetHostDEPAssignmentsBySerialFuncInvoked = false // reset invocation flag
					invalidOSMI := fleet.MDMAppleMachineInfo{
						Product:   "MacBookPro18,3", // Apple Silicon — reaches the OSVersion check
						Serial:    "MACSILSERIAL",
						UDID:      "mac-sil-udid",
						OSVersion: "not-a-valid-version",
					}
					_, err := svc.GetMDMAppleEnrollmentProfileByToken(ctx, "valid-token", "", &invalidOSMI)
					if hardwareAttestationRequired {
						// isMDMAppleACMERequired is called and fails at the OSVersion check — DEP check is never reached
						require.False(t, ds.GetHostDEPAssignmentsBySerialFuncInvoked)
						require.Error(t, err)
						require.Contains(t, err.Error(), "checking if device is less than macOS 14")
					} else {
						// isMDMAppleACMERequired is never called, so OSVersion is irrelevant — SCEP is always returned
						require.False(t, ds.GetHostDEPAssignmentsBySerialFuncInvoked)
						require.NoError(t, err)
					}
				})
			})
		}
	})
}

func TestGetDefaultMDMAppleSetupAssistantProfileFreeLicense(t *testing.T) {
	svc, ctx, _, _ := setupAppleMDMService(t, &fleet.LicenseInfo{Tier: fleet.TierFree})
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	_, _, err := svc.GetDefaultMDMAppleSetupAssistantProfile(ctx)
	assert.ErrorIs(t, err, fleet.ErrMissingLicense)
}
