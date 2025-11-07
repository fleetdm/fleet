package condaccess

import (
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/service/integrationtest"
	"github.com/go-kit/kit/log"
	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

type Suite struct {
	integrationtest.BaseSuite
}

func SetUpSuite(t *testing.T, uniqueTestName string) *Suite {
	return SetUpSuiteWithConfig(t, uniqueTestName, nil)
}

func SetUpSuiteWithConfig(t *testing.T, uniqueTestName string, configModifier func(cfg *config.FleetConfig)) *Suite {
	// Note: t.Parallel() is called when MySQL datastore options are processed
	license := &fleet.LicenseInfo{
		Tier: fleet.TierPremium,
	}
	ds, fleetCfg, fleetSvc, ctx := integrationtest.SetUpMySQLAndService(t, uniqueTestName, &service.TestServerOpts{
		License: license,
		Pool:    redistest.SetupRedis(t, t.Name(), false, false, false),
	})

	// Apply config modifications
	if configModifier != nil {
		configModifier(&fleetCfg)
	}

	logger := log.NewLogfmtLogger(os.Stdout)
	condAccessSCEPDepot, err := ds.NewConditionalAccessSCEPDepot(kitlog.With(logger, "component", "conditional-access-scep-depot"), &fleetCfg)
	require.NoError(t, err)

	users, server := service.RunServerForTestsWithServiceWithDS(t, ctx, ds, fleetSvc, &service.TestServerOpts{
		License:     license,
		FleetConfig: &fleetCfg,
		Logger:      logger,
		ConditionalAccess: &service.ConditionalAccess{
			SCEPStorage: condAccessSCEPDepot,
		},
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

	s.BaseSuite.Token = s.BaseSuite.GetTestAdminToken(t)
	return s
}
