package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/test"
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
}

// TODO: update this test with the correct config option
func TestVerifyMDMMicrosoftConfigured(t *testing.T) {
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

	err := svc.VerifyMDMMicrosoftConfigured(ctx)
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

	err = svc.VerifyMDMMicrosoftConfigured(ctx)
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

	err = svc.VerifyMDMMicrosoftConfigured(ctx)
	require.NoError(t, err)
	require.True(t, ds.AppConfigFuncInvoked)
	ds.AppConfigFuncInvoked = false
	require.False(t, authzCtx.Checked())
}
