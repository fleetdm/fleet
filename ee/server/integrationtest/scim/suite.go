package scim

import (
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/service/integrationtest"
	"github.com/go-kit/log"
)

type Suite struct {
	integrationtest.BaseSuite
}

func SetUpSuite(t *testing.T, uniqueTestName string) *Suite {
	// Note: t.Parallel() is called when MySQL datastore options are processed
	license := &fleet.LicenseInfo{
		Tier: fleet.TierPremium,
	}
	ds, fleetCfg, fleetSvc, ctx := integrationtest.SetUpMySQLAndService(t, uniqueTestName, &service.TestServerOpts{
		License: license,
	})
	logger := log.NewLogfmtLogger(os.Stdout)
	users, server := service.RunServerForTestsWithServiceWithDS(t, ctx, ds, fleetSvc, &service.TestServerOpts{
		License:     license,
		FleetConfig: &fleetCfg,
		Logger:      logger,
		EnableSCIM:  true,
	})

	s := &Suite{
		BaseSuite: integrationtest.BaseSuite{
			Logger:   logger,
			DS:       ds,
			FleetCfg: fleetCfg,
			Users:    users,
			Server:   server,
		},
	}

	integrationtest.SetUpServerURL(t, ds, server)

	s.Token = s.GetTestAdminToken(t)
	return s
}
