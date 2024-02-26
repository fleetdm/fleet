package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/cryptoutil/x509util"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGetMDMApple(t *testing.T) {
	ds := new(mock.Store)
	license := &fleet.LicenseInfo{Tier: fleet.TierFree}
	cfg := config.TestConfig()
	cfg.MDM.AppleAPNsCert = "testdata/server.pem"
	cfg.MDM.AppleAPNsKey = "testdata/server.key"
	cfg.MDM.AppleSCEPCert = "testdata/server.pem"
	cfg.MDM.AppleSCEPKey = "testdata/server.key"
	svc, ctx := newTestServiceWithConfig(t, ds, cfg, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})

	_, _, _, err := cfg.MDM.AppleAPNs()
	require.NoError(t, err)

	ctx = test.UserContext(ctx, test.UserAdmin)
	got, err := svc.GetAppleMDM(ctx)
	require.NoError(t, err)

	// NOTE: to inspect the test certificate, you can use:
	// openssl x509 -in ./server/service/testdata/server.pem -text -noout
	require.Equal(t, &fleet.AppleMDM{
		CommonName:   "servq.groob.io",
		SerialNumber: "1",
		Issuer:       "groob-ca",
		RenewDate:    time.Date(2017, 10, 24, 13, 11, 44, 0, time.UTC),
	}, got)
}

func TestMDMAppleAuthorization(t *testing.T) {
	ds := new(mock.Store)
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium}
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})

	// use a custom implementation of checkAuthErr as the service call will fail
	// with a not found error (given that MDM is not really configured) in case
	// of success, and the package-wide checkAuthErr requires no error.
	checkAuthErr := func(t *testing.T, shouldFail bool, err error) {
		if shouldFail {
			require.Error(t, err)
			require.Equal(t, (&authz.Forbidden{}).Error(), err.Error())
		} else if err != nil {
			require.NotEqual(t, (&authz.Forbidden{}).Error(), err.Error())
		}
	}
	testAuthdMethods := func(t *testing.T, user *fleet.User, shouldFailWithAuth bool) {
		ctx := test.UserContext(ctx, user)
		_, err := svc.GetAppleMDM(ctx)
		checkAuthErr(t, shouldFailWithAuth, err)
		_, err = svc.GetAppleBM(ctx)
		checkAuthErr(t, shouldFailWithAuth, err)

		// deliberately send invalid args so it doesn't actually generate a CSR
		_, err = svc.RequestMDMAppleCSR(ctx, "not-an-email", "")
		require.Error(t, err) // it *will* always fail, but not necessarily due to authorization
		checkAuthErr(t, shouldFailWithAuth, err)
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
}

func TestVerifyMDMAppleConfigured(t *testing.T) {
	ds := new(mock.Store)
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium}
	cfg := config.TestConfig()
	svc, baseCtx := newTestServiceWithConfig(t, ds, cfg, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})

	// mdm not configured
	authzCtx := &authz_ctx.AuthorizationContext{}
	ctx := authz_ctx.NewContext(baseCtx, authzCtx)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: false}}, nil
	}
	err := svc.VerifyMDMAppleConfigured(ctx)
	require.ErrorIs(t, err, fleet.ErrMDMNotConfigured)
	require.True(t, ds.AppConfigFuncInvoked)
	ds.AppConfigFuncInvoked = false
	require.True(t, authzCtx.Checked())

	err = svc.VerifyMDMAppleOrWindowsConfigured(ctx)
	require.ErrorIs(t, err, fleet.ErrMDMNotConfigured)
	require.True(t, ds.AppConfigFuncInvoked)
	ds.AppConfigFuncInvoked = false
	require.True(t, authzCtx.Checked())

	// error retrieving app config
	authzCtx = &authz_ctx.AuthorizationContext{}
	ctx = authz_ctx.NewContext(baseCtx, authzCtx)
	testErr := errors.New("test err")
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return nil, testErr
	}
	err = svc.VerifyMDMAppleConfigured(ctx)
	require.ErrorIs(t, err, testErr)
	require.True(t, ds.AppConfigFuncInvoked)
	ds.AppConfigFuncInvoked = false
	require.True(t, authzCtx.Checked())

	err = svc.VerifyMDMAppleOrWindowsConfigured(ctx)
	require.ErrorIs(t, err, testErr)
	require.True(t, ds.AppConfigFuncInvoked)
	ds.AppConfigFuncInvoked = false
	require.True(t, authzCtx.Checked())

	// mdm configured
	authzCtx = &authz_ctx.AuthorizationContext{}
	ctx = authz_ctx.NewContext(baseCtx, authzCtx)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}, nil
	}
	err = svc.VerifyMDMAppleConfigured(ctx)
	require.NoError(t, err)
	require.True(t, ds.AppConfigFuncInvoked)
	ds.AppConfigFuncInvoked = false
	require.False(t, authzCtx.Checked())

	err = svc.VerifyMDMAppleOrWindowsConfigured(ctx)
	require.NoError(t, err)
	require.True(t, ds.AppConfigFuncInvoked)
	ds.AppConfigFuncInvoked = false
	require.False(t, authzCtx.Checked())
}

func TestVerifyMDMWindowsConfigured(t *testing.T) {
	ds := new(mock.Store)
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium}
	cfg := config.TestConfig()
	svc, baseCtx := newTestServiceWithConfig(t, ds, cfg, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})

	// mdm not configured
	authzCtx := &authz_ctx.AuthorizationContext{}
	ctx := authz_ctx.NewContext(baseCtx, authzCtx)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{WindowsEnabledAndConfigured: false}}, nil
	}

	err := svc.VerifyMDMWindowsConfigured(ctx)
	require.ErrorIs(t, err, fleet.ErrMDMNotConfigured)
	require.True(t, ds.AppConfigFuncInvoked)
	ds.AppConfigFuncInvoked = false
	require.True(t, authzCtx.Checked())

	err = svc.VerifyMDMAppleOrWindowsConfigured(ctx)
	require.ErrorIs(t, err, fleet.ErrMDMNotConfigured)
	require.True(t, ds.AppConfigFuncInvoked)
	ds.AppConfigFuncInvoked = false
	require.True(t, authzCtx.Checked())

	// error retrieving app config
	authzCtx = &authz_ctx.AuthorizationContext{}
	ctx = authz_ctx.NewContext(baseCtx, authzCtx)
	testErr := errors.New("test err")
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return nil, testErr
	}

	err = svc.VerifyMDMWindowsConfigured(ctx)
	require.ErrorIs(t, err, testErr)
	require.True(t, ds.AppConfigFuncInvoked)
	ds.AppConfigFuncInvoked = false
	require.True(t, authzCtx.Checked())

	err = svc.VerifyMDMAppleOrWindowsConfigured(ctx)
	require.ErrorIs(t, err, testErr)
	require.True(t, ds.AppConfigFuncInvoked)
	ds.AppConfigFuncInvoked = false
	require.True(t, authzCtx.Checked())

	// mdm configured
	authzCtx = &authz_ctx.AuthorizationContext{}
	ctx = authz_ctx.NewContext(baseCtx, authzCtx)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{WindowsEnabledAndConfigured: true}}, nil
	}

	err = svc.VerifyMDMWindowsConfigured(ctx)
	require.NoError(t, err)
	require.True(t, ds.AppConfigFuncInvoked)
	ds.AppConfigFuncInvoked = false
	require.False(t, authzCtx.Checked())

	err = svc.VerifyMDMAppleOrWindowsConfigured(ctx)
	require.NoError(t, err)
	require.True(t, ds.AppConfigFuncInvoked)
	ds.AppConfigFuncInvoked = false
	require.False(t, authzCtx.Checked())
}

func TestMicrosoftWSTEPConfig(t *testing.T) {
	ds := new(mock.Store)
	license := &fleet.LicenseInfo{Tier: fleet.TierFree}

	ds.WSTEPNewSerialFunc = func(context.Context) (*big.Int, error) {
		return big.NewInt(1337), nil
	}
	ds.WSTEPStoreCertificateFunc = func(ctx context.Context, name string, crt *x509.Certificate) error {
		require.Equal(t, "test-client", name)
		require.Equal(t, "test-client", crt.Subject.CommonName)
		require.Equal(t, "Fleet", crt.Subject.OrganizationalUnit[0])
		return nil
	}

	certPath := "testdata/server.pem"
	keyPath := "testdata/server.key"

	// sanity check that the test data is valid
	wantCertPEM, err := os.ReadFile(certPath)
	require.NoError(t, err)
	wantKeyPEM, err := os.ReadFile(keyPath)
	require.NoError(t, err)

	// specify the test data in the server config
	cfg := config.TestConfig()
	cfg.MDM.WindowsWSTEPIdentityCert = certPath
	cfg.MDM.WindowsWSTEPIdentityKey = keyPath

	// check that config.MDM.MicrosoftWSTEP() returns the expected values
	_, cfgCertPEM, cfgKeyPEM, err := cfg.MDM.MicrosoftWSTEP()
	require.NoError(t, err)
	require.NotEmpty(t, cfgCertPEM)
	require.Equal(t, wantCertPEM, cfgCertPEM)
	require.NotEmpty(t, cfgKeyPEM)
	require.Equal(t, wantKeyPEM, cfgKeyPEM)

	// start the test service
	svc, ctx := newTestServiceWithConfig(t, ds, cfg, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})
	ctx = test.UserContext(ctx, test.UserAdmin)

	// test CSR signing
	clienPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	csrTemplate := x509util.CertificateRequest{
		CertificateRequest: x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName: "test-cient",
			},
			SignatureAlgorithm: x509.SHA256WithRSA,
		},
	}
	csrDerBytes, err := x509util.CreateCertificateRequest(rand.Reader, &csrTemplate, clienPrivateKey)
	require.NoError(t, err)
	csr, err := x509.ParseCertificateRequest(csrDerBytes)
	require.NoError(t, err)

	// test the service method
	rawDER, _, err := svc.SignMDMMicrosoftClientCSR(ctx, "test-client", csr)
	require.NoError(t, err)
	require.True(t, ds.WSTEPNewSerialFuncInvoked)
	require.True(t, ds.WSTEPStoreCertificateFuncInvoked)

	// TODO: additional assertions on the signed certificate
	parsedCert, err := x509.ParseCertificate(rawDER)
	require.NoError(t, err)
	require.Equal(t, "test-client", parsedCert.Subject.CommonName)
	require.Equal(t, "Fleet", parsedCert.Subject.OrganizationalUnit[0])
}

func TestRunMDMCommandAuthz(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	singleUnenrolledHost := []*fleet.Host{{ID: 1, TeamID: ptr.Uint(1), UUID: "a"}}
	team1And2UnenrolledHosts := []*fleet.Host{{ID: 1, TeamID: ptr.Uint(1), UUID: "a"}, {ID: 2, TeamID: ptr.Uint(2), UUID: "b"}}
	team2And3UnenrolledHosts := []*fleet.Host{{ID: 2, TeamID: ptr.Uint(2), UUID: "b"}, {ID: 3, TeamID: ptr.Uint(3), UUID: "c"}}

	userTeamMaintainerTeam1And2 := &fleet.User{
		ID: 100,
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: 1},
				Role: fleet.RoleMaintainer,
			},
			{
				Team: fleet.Team{ID: 2},
				Role: fleet.RoleMaintainer,
			},
		},
	}
	userTeamAdminTeam1And2 := &fleet.User{
		ID: 101,
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: 1},
				Role: fleet.RoleAdmin,
			},
			{
				Team: fleet.Team{ID: 2},
				Role: fleet.RoleAdmin,
			},
		},
	}
	userTeamAdminTeam1ObserverTeam2 := &fleet.User{
		ID: 102,
		Teams: []fleet.UserTeam{
			{
				Team: fleet.Team{ID: 1},
				Role: fleet.RoleAdmin,
			},
			{
				Team: fleet.Team{ID: 2},
				Role: fleet.RoleObserver,
			},
		},
	}

	checkAuthErr := func(t *testing.T, shouldFailWithAuth bool, err error) {
		t.Helper()

		if shouldFailWithAuth {
			require.Error(t, err)
			require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
		} else {
			// call always fails, but due to the host not being enrolled in MDM
			require.Error(t, err)
			require.NotContains(t, err.Error(), authz.ForbiddenErrorMessage)
		}
	}

	enqueueCmdCases := []struct {
		desc               string
		user               *fleet.User
		hosts              []*fleet.Host
		shouldFailWithAuth bool
	}{
		{"no role", test.UserNoRoles, singleUnenrolledHost, true},
		{"maintainer", test.UserMaintainer, singleUnenrolledHost, false},
		{"admin", test.UserAdmin, singleUnenrolledHost, false},
		{"observer", test.UserObserver, singleUnenrolledHost, true},
		{"observer+", test.UserObserverPlus, singleUnenrolledHost, true},
		{"gitops", test.UserGitOps, singleUnenrolledHost, true},
		{"team 1 admin", test.UserTeamAdminTeam1, singleUnenrolledHost, false},
		{"team 2 admin", test.UserTeamAdminTeam2, singleUnenrolledHost, true},
		{"team 1 maintainer", test.UserTeamMaintainerTeam1, singleUnenrolledHost, false},
		{"team 2 maintainer", test.UserTeamMaintainerTeam2, singleUnenrolledHost, true},
		{"team 1 observer", test.UserTeamObserverTeam1, singleUnenrolledHost, true},
		{"team 2 observer", test.UserTeamObserverTeam2, singleUnenrolledHost, true},
		{"team 1 observer+", test.UserTeamObserverPlusTeam1, singleUnenrolledHost, true},
		{"team 2 observer+", test.UserTeamObserverPlusTeam2, singleUnenrolledHost, true},
		{"team 1 gitops", test.UserTeamGitOpsTeam1, singleUnenrolledHost, true},
		{"team 2 gitops", test.UserTeamGitOpsTeam2, singleUnenrolledHost, true},
		{"team 1 admin mix of teams", test.UserTeamAdminTeam1, team1And2UnenrolledHosts, true},
		{"team 1 maintainer mix of teams", test.UserTeamMaintainerTeam1, team1And2UnenrolledHosts, true},
		{"admin mix of teams", test.UserAdmin, team1And2UnenrolledHosts, false},
		{"team 1 admin 2 other teams", test.UserTeamAdminTeam1, team2And3UnenrolledHosts, true},
		{"team 1 maintainer 2 other teams", test.UserTeamMaintainerTeam1, team2And3UnenrolledHosts, true},
		{"admin mix of teams", test.UserAdmin, team1And2UnenrolledHosts, false},
		{"admin mix of 2 other teams", test.UserAdmin, team2And3UnenrolledHosts, false},
		{"team 1 and 2 admin on allowed teams", userTeamAdminTeam1And2, team1And2UnenrolledHosts, false},
		{"team 1 and 2 maintainer on allowed teams", userTeamMaintainerTeam1And2, team1And2UnenrolledHosts, false},
		{"team 1 and 2 admin on other teams", userTeamAdminTeam1And2, team2And3UnenrolledHosts, true},
		{"team 1 and 2 maintainer on other teams", userTeamMaintainerTeam1And2, team2And3UnenrolledHosts, true},
		{"team 1 admin and 2 observer on team 1", userTeamAdminTeam1ObserverTeam2, singleUnenrolledHost, false},
		{"team 1 admin and 2 observer on team 2 and 3", userTeamAdminTeam1ObserverTeam2, team2And3UnenrolledHosts, true},
		{"team 1 admin and 2 observer on team 1 and 2", userTeamAdminTeam1ObserverTeam2, team1And2UnenrolledHosts, true},
	}
	for _, c := range enqueueCmdCases {
		t.Run(c.desc, func(t *testing.T) {
			ds.ListHostsLiteByUUIDsFunc = func(ctx context.Context, filter fleet.TeamFilter, uuids []string) ([]*fleet.Host, error) {
				return c.hosts, nil
			}

			ctx = test.UserContext(ctx, c.user)
			_, err := svc.RunMDMCommand(ctx, "base64command", []string{"uuid"})
			checkAuthErr(t, c.shouldFailWithAuth, err)
		})
	}
}

func TestRunMDMCommandValidations(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	enrolledMDMInfo := &fleet.HostMDM{Enrolled: true, InstalledFromDep: false, Name: fleet.WellKnownMDMFleet, IsServer: false}
	singleUnenrolledHost := []*fleet.Host{{ID: 1, TeamID: ptr.Uint(1), UUID: "a"}}
	differentPlatformsHosts := []*fleet.Host{
		{ID: 1, UUID: "a", MDMInfo: enrolledMDMInfo, Platform: "darwin"},
		{ID: 2, UUID: "b", MDMInfo: enrolledMDMInfo, Platform: "windows"},
	}
	linuxSingleHost := []*fleet.Host{{ID: 1, TeamID: ptr.Uint(1), UUID: "a", MDMInfo: enrolledMDMInfo, Platform: "linux"}}
	windowsSingleHost := []*fleet.Host{{ID: 1, TeamID: ptr.Uint(1), UUID: "a", MDMInfo: enrolledMDMInfo, Platform: "windows"}}
	macosSingleHost := []*fleet.Host{{ID: 1, TeamID: ptr.Uint(1), UUID: "a", MDMInfo: enrolledMDMInfo, Platform: "darwin"}}

	cases := []struct {
		desc          string
		hosts         []*fleet.Host
		mdmConfigured bool
		wantErr       string
	}{
		{"no hosts", []*fleet.Host{}, false, "No hosts targeted."},
		{"unenrolled host", singleUnenrolledHost, false, "Can't run the MDM command because one or more hosts have MDM turned off."},
		{"different platforms", differentPlatformsHosts, false, "All hosts must be on the same platform."},
		{"invalid platform", linuxSingleHost, false, "Invalid platform."},
		{"mdm not configured (windows)", windowsSingleHost, false, "Windows MDM isn't turned on."},
		{"mdm not configured (macos)", macosSingleHost, false, "macOS MDM isn't turned on."},
		{"invalid base64 encoding", macosSingleHost, true, "unable to decode base64 command"},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			ds.ListHostsLiteByUUIDsFunc = func(ctx context.Context, filter fleet.TeamFilter, uuids []string) ([]*fleet.Host, error) {
				return c.hosts, nil
			}
			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{
					MDM: fleet.MDM{
						EnabledAndConfigured:        c.mdmConfigured,
						WindowsEnabledAndConfigured: c.mdmConfigured,
					},
				}, nil
			}
			ctx = test.UserContext(ctx, test.UserAdmin)
			_, err := svc.RunMDMCommand(ctx, "!@#", []string{"unused for this test"})
			require.Error(t, err)
			require.ErrorContains(t, err, c.wantErr)
		})
	}
}

func TestMDMCommonAuthorization(t *testing.T) {
	ds := new(mock.Store)
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium}
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true, WindowsEnabledAndConfigured: true}}, nil
	}

	ds.GetMDMAppleFileVaultSummaryFunc = func(ctx context.Context, teamID *uint) (*fleet.MDMAppleFileVaultSummary, error) {
		return &fleet.MDMAppleFileVaultSummary{}, nil
	}
	ds.GetMDMWindowsBitLockerSummaryFunc = func(ctx context.Context, teamID *uint) (*fleet.MDMWindowsBitLockerSummary, error) {
		return &fleet.MDMWindowsBitLockerSummary{}, nil
	}
	ds.GetMDMWindowsProfilesSummaryFunc = func(ctx context.Context, teamID *uint) (*fleet.MDMProfilesSummary, error) {
		return &fleet.MDMProfilesSummary{}, nil
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

	checkShouldFail := func(err error, shouldFail bool) {
		if !shouldFail {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
			require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
		}
	}

	for _, tt := range testCases {
		ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})
		ds.TeamFunc = mockTeamFuncWithUser(tt.user)

		t.Run(tt.name, func(t *testing.T) {
			// test authz for MDM summary endpoints (no team)
			_, err := svc.GetMDMDiskEncryptionSummary(ctx, nil)
			checkShouldFail(err, tt.shouldFailGlobal)
			_, err = svc.GetMDMWindowsProfilesSummary(ctx, nil)
			checkShouldFail(err, tt.shouldFailGlobal)

			// test authz for MDM summary endpoints (team 1)
			_, err = svc.GetMDMDiskEncryptionSummary(ctx, ptr.Uint(1))
			checkShouldFail(err, tt.shouldFailTeam)
			_, err = svc.GetMDMWindowsProfilesSummary(ctx, ptr.Uint(1))
			checkShouldFail(err, tt.shouldFailTeam)
		})
	}
}

func TestEnqueueWindowsMDMCommand(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	ds.MDMWindowsInsertCommandForHostsFunc = func(ctx context.Context, deviceIDs []string, cmd *fleet.MDMWindowsCommand) error {
		return nil
	}

	cases := []struct {
		desc        string
		premium     bool
		xmlCmd      string
		wantErr     string
		wantReqType string
	}{
		{"invalid xml", false, `!!$$`, "The payload isn't valid XML", ""},
		{"empty xml", false, ``, "The payload isn't valid XML", ""},
		{"unrelated xml", false, `<Unrelated></Unrelated>`, "You can run only <Exec> command type", ""},
		{"no command Exec", false, `<Exec></Exec>`, "You can run only a single <Exec> command", ""},
		{"non-exec command", false, `
			<Get>
				<CmdID>1</CmdID>
				<Item>
					<Target>
						<LocURI>./DevDetail/SwV</LocURI>
					</Target>
				</Item>
			</Get>`, "You can run only <Exec> command type", ""},
		{"multi-exec command", false, `
			<Exec>
				<CmdID>1</CmdID>
				<Item>
					<Target>
						<LocURI>./DevDetail/SwV</LocURI>
					</Target>
				</Item>
				<Item>
					<Target>
						<LocURI>./DevDetail/SwV2</LocURI>
					</Target>
				</Item>
			</Exec>`, "You can run only a single <Exec> command", ""},
		{"premium command, non premium license", false, `
			<Exec>
				<CmdID>1</CmdID>
				<Item>
					<Target>
						<LocURI>./Device/Vendor/MSFT/RemoteWipe/doWipe</LocURI>
					</Target>
				</Item>
			</Exec>`, "Requires Fleet Premium license", ""},
		{"premium command, premium license", true, `
			<Exec>
				<CmdID>1</CmdID>
				<Item>
					<Target>
						<LocURI>./Device/Vendor/MSFT/RemoteWipe/doWipe</LocURI>
					</Target>
				</Item>
			</Exec>`, "", "./Device/Vendor/MSFT/RemoteWipe/doWipe"},
		{"non-premium command", false, `
			<Exec>
				<CmdID>1</CmdID>
				<Item>
					<Target>
						<LocURI>./FooBar</LocURI>
					</Target>
				</Item>
			</Exec>`, "", "./FooBar"},
		{"multi top-level Execs", false, `
			<Exec>
				<CmdID>1</CmdID>
				<Item>
					<Target>
						<LocURI>./FooBar</LocURI>
					</Target>
				</Item>
			</Exec>
			<Exec>
				<CmdID>2</CmdID>
				<Item>
					<Target>
						<LocURI>./FooBar2</LocURI>
					</Target>
				</Item>
			</Exec>`, "You can run only a single <Exec> command", ""},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			ctx = test.UserContext(ctx, test.UserAdmin)
			if c.premium {
				ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierPremium})
			}

			var svcImpl *Service
			switch v := svc.(type) {
			case validationMiddleware:
				svcImpl = v.Service.(*Service)
			case *Service:
				svcImpl = v
			}
			res, err := svcImpl.enqueueMicrosoftMDMCommand(ctx, []byte(c.xmlCmd), []string{"uuid"})

			if c.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, c.wantErr)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, res.CommandUUID)
				require.Equal(t, "windows", res.Platform)
				require.Equal(t, c.wantReqType, res.RequestType)
			}
		})
	}
}

func TestGetMDMDiskEncryptionSummary(t *testing.T) {
	ds := new(mock.Store)
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium}
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license})

	ctx = test.UserContext(ctx, test.UserAdmin)

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}, nil
	}
	ds.GetMDMAppleFileVaultSummaryFunc = func(ctx context.Context, teamID *uint) (*fleet.MDMAppleFileVaultSummary, error) {
		require.Nil(t, teamID)
		return &fleet.MDMAppleFileVaultSummary{Verified: 1, Verifying: 2, ActionRequired: 3, Failed: 4, Enforcing: 5, RemovingEnforcement: 6}, nil
	}
	ds.GetMDMWindowsBitLockerSummaryFunc = func(ctx context.Context, teamID *uint) (*fleet.MDMWindowsBitLockerSummary, error) {
		require.Nil(t, teamID)
		// Use default zeros verifying, action_required, or removing_enforcement
		return &fleet.MDMWindowsBitLockerSummary{Verified: 7, Failed: 8, Enforcing: 9}, nil
	}

	// Test that the summary properly combines the results of the two methods
	des, err := svc.GetMDMDiskEncryptionSummary(ctx, nil)
	require.NoError(t, err)
	require.NotNil(t, des)
	require.Equal(t, *des, fleet.MDMDiskEncryptionSummary{
		Verified: fleet.MDMPlatformsCounts{
			MacOS:   1,
			Windows: 7,
		},
		Verifying: fleet.MDMPlatformsCounts{
			MacOS:   2,
			Windows: 0,
		},
		ActionRequired: fleet.MDMPlatformsCounts{
			MacOS:   3,
			Windows: 0,
		},
		Failed: fleet.MDMPlatformsCounts{
			MacOS:   4,
			Windows: 8,
		},
		Enforcing: fleet.MDMPlatformsCounts{
			MacOS:   5,
			Windows: 9,
		},
		RemovingEnforcement: fleet.MDMPlatformsCounts{
			MacOS:   6,
			Windows: 0,
		},
	})
}

func TestMDMWindowsConfigProfileAuthz(t *testing.T) {
	ds := new(mock.Store)
	// while the config profiles are not premium-only, teams are and we want to test with teams.
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium}
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})

	testCases := []struct {
		name                  string
		user                  *fleet.User
		shouldFailGlobalRead  bool
		shouldFailTeamRead    bool
		shouldFailGlobalWrite bool
		shouldFailTeamWrite   bool
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			false,
			false,
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			false,
			false,
			false,
			false,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
			true,
			true,
			true,
		},
		{
			"global observer+",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserverPlus)},
			true,
			true,
			true,
			true,
		},
		{
			// this is authorized because any logged-in user can read teams (the
			// first authorization check) and then gitops have write-access the the
			// profiles.
			"global gitops",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)},
			true,
			true,
			false,
			false,
		},
		{
			"team admin, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			true,
			false,
			true,
			false,
		},
		{
			"team admin, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleAdmin}}},
			true,
			true,
			true,
			true,
		},
		{
			"team maintainer, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			true,
			false,
			true,
			false,
		},
		{
			"team maintainer, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleMaintainer}}},
			true,
			true,
			true,
			true,
		},
		{
			"team observer, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
			true,
			true,
			true,
		},
		{
			"team observer, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserver}}},
			true,
			true,
			true,
			true,
		},
		{
			"team observer+, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserverPlus}}},
			true,
			true,
			true,
			true,
		},
		{
			"team observer+, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserverPlus}}},
			true,
			true,
			true,
			true,
		},
		{
			// this is authorized because any logged-in user can read teams (the
			// first authorization check) and then gitops have write-access the the
			// profiles.
			"team gitops, belongs to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleGitOps}}},
			true,
			true,
			true,
			false,
		},
		{
			"team gitops, DOES NOT belong to team",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleGitOps}}},
			true,
			true,
			true,
			true,
		},
		{
			"user no roles",
			&fleet.User{ID: 1337},
			true,
			true,
			true,
			true,
		},
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			MDM: fleet.MDM{
				EnabledAndConfigured:        true,
				WindowsEnabledAndConfigured: true,
			},
		}, nil
	}
	ds.NewActivityFunc = func(context.Context, *fleet.User, fleet.ActivityDetails) error {
		return nil
	}
	ds.GetMDMWindowsConfigProfileFunc = func(ctx context.Context, pid string) (*fleet.MDMWindowsConfigProfile, error) {
		var tid uint
		if pid == "team-1" {
			tid = 1
		}
		return &fleet.MDMWindowsConfigProfile{
			ProfileUUID: pid,
			TeamID:      &tid,
		}, nil
	}
	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		return &fleet.Team{ID: tid, Name: "team1"}, nil
	}
	ds.DeleteMDMWindowsConfigProfileFunc = func(ctx context.Context, profileUUID string) error {
		return nil
	}
	ds.NewMDMWindowsConfigProfileFunc = func(ctx context.Context, cp fleet.MDMWindowsConfigProfile) (*fleet.MDMWindowsConfigProfile, error) {
		return &cp, nil
	}
	ds.ListMDMConfigProfilesFunc = func(ctx context.Context, teamID *uint, opt fleet.ListOptions) ([]*fleet.MDMConfigProfilePayload, *fleet.PaginationMetadata, error) {
		return nil, nil, nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hostIDs []uint, teamIDs []uint, profileUUIDs []string, hostUUIDs []string) error {
		return nil
	}

	checkShouldFail := func(t *testing.T, err error, shouldFail bool) {
		if !shouldFail {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
			require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
		}
	}

	const winProfContent = `<Replace></Replace>`
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			// test authz get config profile (no team)
			_, err := svc.GetMDMWindowsConfigProfile(ctx, "global")
			checkShouldFail(t, err, tt.shouldFailGlobalRead)

			// test authz get config profile (team 1)
			_, err = svc.GetMDMWindowsConfigProfile(ctx, "team-1")
			checkShouldFail(t, err, tt.shouldFailTeamRead)

			// test authz list config profiles (no team)
			_, _, err = svc.ListMDMConfigProfiles(ctx, nil, fleet.ListOptions{})
			checkShouldFail(t, err, tt.shouldFailGlobalRead)

			// test authz list config profiles (team 1)
			_, _, err = svc.ListMDMConfigProfiles(ctx, ptr.Uint(1), fleet.ListOptions{})
			checkShouldFail(t, err, tt.shouldFailTeamRead)

			// test authz create new profile (no team)
			_, err = svc.NewMDMWindowsConfigProfile(ctx, 0, "prof", strings.NewReader(winProfContent), nil)
			checkShouldFail(t, err, tt.shouldFailGlobalWrite)

			// test authz create new profile (team 1)
			_, err = svc.NewMDMWindowsConfigProfile(ctx, 1, "prof", strings.NewReader(winProfContent), nil)
			checkShouldFail(t, err, tt.shouldFailTeamWrite)

			// test authz delete config profile (no team)
			err = svc.DeleteMDMWindowsConfigProfile(ctx, "global")
			checkShouldFail(t, err, tt.shouldFailGlobalWrite)

			// test authz delete config profile (team 1)
			err = svc.DeleteMDMWindowsConfigProfile(ctx, "team-1")
			checkShouldFail(t, err, tt.shouldFailTeamWrite)
		})
	}
}

func TestUploadWindowsMDMConfigProfileValidations(t *testing.T) {
	ds := new(mock.Store)
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium}
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})

	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		if tid != 1 {
			return nil, &notFoundError{}
		}
		return &fleet.Team{ID: tid, Name: "team1"}, nil
	}
	ds.NewActivityFunc = func(context.Context, *fleet.User, fleet.ActivityDetails) error {
		return nil
	}
	ds.NewMDMWindowsConfigProfileFunc = func(ctx context.Context, cp fleet.MDMWindowsConfigProfile) (*fleet.MDMWindowsConfigProfile, error) {
		if bytes.Contains(cp.SyncML, []byte("duplicate")) {
			return nil, &alreadyExistsError{}
		}
		cp.ProfileUUID = uuid.New().String()
		return &cp, nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hostIDs []uint, teamIDs []uint, profileUUIDs []string, hostUUIDs []string) error {
		return nil
	}

	cases := []struct {
		desc          string
		tmID          uint
		profile       string
		mdmConfigured bool
		wantErr       string
	}{
		{"empty profile", 0, "", true, "The file should include valid XML."},
		{"plist data", 0, string(mcBytesForTest("Foo", "Bar", "UUID")), true, "The file should include valid XML: processing instructions are not allowed."},
		{"random non-xml data", 0, "\x00\x01\x02", true, "The file should include valid XML:"},
		{"valid windows profile", 0, `<Replace></Replace>`, true, ""},
		{"mdm not enabled", 0, `<Replace></Replace>`, false, "Windows MDM isn't turned on."},
		{"duplicate profile name", 0, `<Replace>duplicate</Replace>`, true, "configuration profile with this name already exists."},
		{"multiple Replace", 0, `<Replace>a</Replace><Replace>b</Replace>`, true, ""},
		{"Replace and non-Replace", 0, `<Replace>a</Replace><Get>b</Get>`, true, "Only <Replace> supported as a top level element."},
		{"BitLocker profile", 0, `<Replace><Item><Target><LocURI>./Device/Vendor/MSFT/BitLocker/AllowStandardUserEncryption</LocURI></Target></Item></Replace>`, true, "Custom configuration profiles can't include BitLocker settings."},
		{"Windows updates profile", 0, `<Replace><Item><Target><LocURI> ./Device/Vendor/MSFT/Policy/Config/Update/ConfigureDeadlineNoAutoRebootForFeatureUpdates </LocURI></Target></Item></Replace>`, true, "Custom configuration profiles can't include Windows updates settings."},

		{"team empty profile", 1, "", true, "The file should include valid XML."},
		{"team plist data", 1, string(mcBytesForTest("Foo", "Bar", "UUID")), true, "The file should include valid XML: processing instructions are not allowed."},
		{"team random non-xml data", 1, "\x00\x01\x02", true, "The file should include valid XML:"},
		{"team valid windows profile", 1, `<Replace></Replace>`, true, ""},
		{"team mdm not enabled", 1, `<Replace></Replace>`, false, "Windows MDM isn't turned on."},
		{"team duplicate profile name", 1, `<Replace>duplicate</Replace>`, true, "configuration profile with this name already exists."},
		{"team multiple Replace", 1, `<Replace>a</Replace><Replace>b</Replace>`, true, ""},
		{"team Replace and non-Replace", 1, `<Replace>a</Replace><Get>b</Get>`, true, "Only <Replace> supported as a top level element."},
		{"team BitLocker profile", 1, `<Replace><Item><Target><LocURI>./Device/Vendor/MSFT/BitLocker/AllowStandardUserEncryption</LocURI></Target></Item></Replace>`, true, "Custom configuration profiles can't include BitLocker settings."},
		{"team Windows updates profile", 1, `<Replace><Item><Target><LocURI> ./Device/Vendor/MSFT/Policy/Config/Update/ConfigureDeadlineNoAutoRebootForFeatureUpdates </LocURI></Target></Item></Replace>`, true, "Custom configuration profiles can't include Windows updates settings."},

		{"invalid team", 2, `<Replace></Replace>`, true, "not found"},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{
					MDM: fleet.MDM{
						EnabledAndConfigured:        true,
						WindowsEnabledAndConfigured: c.mdmConfigured,
					},
				}, nil
			}
			ctx = test.UserContext(ctx, test.UserAdmin)
			_, err := svc.NewMDMWindowsConfigProfile(ctx, c.tmID, "foo", strings.NewReader(c.profile), nil)
			if c.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, c.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMDMBatchSetProfiles(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: fleet.TierPremium}, SkipCreateTestUsers: true})

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			OrgInfo: fleet.OrgInfo{
				OrgName: "Foo Inc.",
			},
			ServerSettings: fleet.ServerSettings{
				ServerURL: "https://foo.example.com",
			},
			MDM: fleet.MDM{
				EnabledAndConfigured:        true,
				WindowsEnabledAndConfigured: true,
			},
		}, nil
	}

	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		return &fleet.Team{ID: 1, Name: name}, nil
	}
	ds.TeamFunc = func(ctx context.Context, id uint) (*fleet.Team, error) {
		return &fleet.Team{ID: id, Name: "team"}, nil
	}
	ds.BatchSetMDMProfilesFunc = func(ctx context.Context, tmID *uint, macProfiles []*fleet.MDMAppleConfigProfile, winProfiles []*fleet.MDMWindowsConfigProfile) error {
		return nil
	}
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hostIDs []uint, teamIDs []uint, profileUUIDs []string, hostUUIDs []string) error {
		return nil
	}

	testCases := []struct {
		name     string
		user     *fleet.User
		premium  bool
		teamID   *uint
		teamName *string
		profiles []fleet.MDMProfileBatchPayload
		wantErr  string
	}{
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
			"duplicate macOS profile name",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			true,
			ptr.Uint(1),
			nil,
			[]fleet.MDMProfileBatchPayload{
				{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
				{Name: "N2", Contents: mobileconfigForTest("N1", "I2")},
			},
			`The name provided for the profile must match the profile PayloadDisplayName: "N1"`,
		},
		{
			"duplicate macOS profile identifier",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			true,
			ptr.Uint(1),
			nil,
			[]fleet.MDMProfileBatchPayload{
				{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
				{Name: "N2", Contents: mobileconfigForTest("N2", "I2")},
				{Name: "N3", Contents: mobileconfigForTest("N3", "I1")},
			},
			`More than one configuration profile have the same identifier (PayloadIdentifier): "I1"`,
		},
		{
			"only macOS",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			nil,
			nil,
			[]fleet.MDMProfileBatchPayload{
				{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
				{Name: "N2", Contents: mobileconfigForTest("N2", "I2")},
				{Name: "N3", Contents: mobileconfigForTest("N3", "I3")},
			},
			``,
		},
		{
			"mixed profiles",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			nil,
			nil,
			[]fleet.MDMProfileBatchPayload{
				{Name: "N1", Contents: syncMLForTest("./foo/bar")},
				{Name: "N2", Contents: syncMLForTest("./baz")},
				{Name: "N3", Contents: syncMLForTest("./zab")},
				{Name: "N4", Contents: mobileconfigForTest("N4", "I1")},
				{Name: "N5", Contents: mobileconfigForTest("N5", "I2")},
				{Name: "N6", Contents: mobileconfigForTest("N6", "I3")},
			},
			``,
		},
		{
			"only windows",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			nil,
			nil,
			[]fleet.MDMProfileBatchPayload{
				{Name: "N1", Contents: syncMLForTest("./foo/bar")},
				{Name: "N2", Contents: syncMLForTest("./baz")},
				{Name: "N3", Contents: syncMLForTest("./zab")},
			},
			``,
		},
		{
			"unsupported payload type",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			nil,
			nil,
			[]fleet.MDMProfileBatchPayload{
				{
					Name: "foo", Contents: []byte(`<?xml version="1.0" encoding="UTF-8"?>
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
			</plist>`),
				},
			},
			"unsupported PayloadType(s)",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			defer func() { ds.BatchSetMDMProfilesFuncInvoked = false }()

			// prepare the context with the user and license
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})
			tier := fleet.TierFree
			if tt.premium {
				tier = fleet.TierPremium
			}
			ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: tier})

			err := svc.BatchSetMDMProfiles(ctx, tt.teamID, tt.teamName, tt.profiles, false, false, false)
			if tt.wantErr == "" {
				require.NoError(t, err)
				require.True(t, ds.BatchSetMDMProfilesFuncInvoked)
				return
			}
			require.Error(t, err)
			require.ErrorContains(t, err, tt.wantErr)
			require.False(t, ds.BatchSetMDMProfilesFuncInvoked)
		})
	}
}

func TestValidateProfiles(t *testing.T) {
	tests := []struct {
		name     string
		profiles []fleet.MDMProfileBatchPayload
		wantErr  bool
	}{
		{
			name: "Valid Darwin Profile",
			profiles: []fleet.MDMProfileBatchPayload{
				{Name: "darwinProfile", Contents: []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>")},
			},
			wantErr: false,
		},
		{
			name: "Valid Windows Profile",
			profiles: []fleet.MDMProfileBatchPayload{
				{Name: "windowsProfile", Contents: []byte("<replace><Target><LocURI>Custom/URI</LocURI></Target></replace>")},
			},
			wantErr: false,
		},
		{
			name: "Invalid Profile",
			profiles: []fleet.MDMProfileBatchPayload{
				{Name: "invalidProfile", Contents: []byte("invalid data")},
			},
			wantErr: true,
		},
		{
			name: "Mixed Valid and Invalid Profiles",
			profiles: []fleet.MDMProfileBatchPayload{
				{Name: "validProfile", Contents: []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>")},
				{Name: "invalidProfile", Contents: []byte("invalid data")},
			},
			wantErr: true,
		},
		{
			name: "Empty Profile",
			profiles: []fleet.MDMProfileBatchPayload{
				{Name: "emptyProfile", Contents: []byte("")},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProfiles(tt.profiles)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBackwardsCompatProfilesParamUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expect      backwardsCompatProfilesParam
		expectError bool
	}{
		{
			name:        "empty input",
			input:       []byte(""),
			expect:      nil,
			expectError: false,
		},
		{
			name:  "new format",
			input: []byte(`[{"name": "profile1", "contents": "Zm9vCg=="}, {"name": "profile2", "contents": "YmFyCg=="}]`),
			expect: backwardsCompatProfilesParam{
				{Name: "profile1", Contents: []byte("foo\n")},
				{Name: "profile2", Contents: []byte("bar\n")},
			},
			expectError: false,
		},
		{
			name:  "new format with labels",
			input: []byte(`[{"name": "profile1", "contents": "Zm9vCg==", "labels": ["foo", "bar"]}, {"name": "profile2", "contents": "YmFyCg=="}]`),
			expect: backwardsCompatProfilesParam{
				{Name: "profile1", Contents: []byte("foo\n"), Labels: []string{"foo", "bar"}},
				{Name: "profile2", Contents: []byte("bar\n")},
			},
			expectError: false,
		},
		{
			name:  "old format",
			input: []byte(`{"profile1": "Zm9vCg==", "profile2": "YmFyCg=="}`),
			expect: backwardsCompatProfilesParam{
				{Name: "profile1", Contents: []byte("foo\n")},
				{Name: "profile2", Contents: []byte("bar\n")},
			},
			expectError: false,
		},
		{
			name:        "invalid json",
			input:       []byte(`{invalid json}`),
			expect:      nil,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var bcp backwardsCompatProfilesParam
			err := bcp.UnmarshalJSON(tc.input)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.ElementsMatch(t, tc.expect, bcp)
			}
		})
	}
}
