package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/micromdm/scep/v2/cryptoutil/x509util"
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
func TestVerifyMDMWindowsConfigured(t *testing.T) {
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

	err := svc.VerifyMDMWindowsConfigured(ctx)
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
		require.Equal(t, "FleetDM", crt.Subject.OrganizationalUnit[0])
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
	require.Equal(t, "FleetDM", parsedCert.Subject.OrganizationalUnit[0])
}

// TODO: Add auth tests for Windows-specific endpoints the only require windows configured and for common MDM endpoints that require
// either mac or windows configured (e.g., GetMDMDiskEncryptionSummary)

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
