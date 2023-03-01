package service

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
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

	cases := []struct {
		name       string
		user       *fleet.User
		shouldFail bool
	}{
		{"TestGlobalAdmin", test.UserAdmin, false},
		{"TestGlobalMaintainer", test.UserMaintainer, false},
		{"TestTeamAdmin", test.UserTeamAdminTeam1, false},
		{"TestTeamMaintainer", test.UserTeamMaintainerTeam1, false},
		{"TestGlobalObserver", test.UserObserver, true},
		{"TestTeamObserver", test.UserTeamObserverTeam1, true},
		{"TestNoRoles", test.UserNoRoles, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := test.UserContext(ctx, c.user)
			_, err := svc.GetAppleMDM(ctx)
			checkAuthErr(t, c.shouldFail, err)
			_, err = svc.GetAppleBM(ctx)
			checkAuthErr(t, c.shouldFail, err)

			// deliberately send invalid args so it doesn't actually generate a CSR
			_, err = svc.RequestMDMAppleCSR(ctx, "not-an-email", "")
			require.Error(t, err) // it *will* always fail, but not necessarily due to authorization
			checkAuthErr(t, c.shouldFail, err)
		})
	}
}
