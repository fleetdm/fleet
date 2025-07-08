package hostidentity

import (
	"os"
	"testing"

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
	// Note: t.Parallel() is called when MySQL datastore options are processed
	license := &fleet.LicenseInfo{
		Tier: fleet.TierPremium,
	}
	ds, fleetCfg, fleetSvc, ctx := integrationtest.SetUpMySQLAndService(t, uniqueTestName, &service.TestServerOpts{
		License: license,
	})
	logger := log.NewLogfmtLogger(os.Stdout)
	hostIdentitySCEPDepot, err := ds.NewHostIdentitySCEPDepot(kitlog.With(logger, "component", "host-id-scep-depot"))
	require.NoError(t, err)
	users, server := service.RunServerForTestsWithServiceWithDS(t, ctx, ds, fleetSvc, &service.TestServerOpts{
		License:                 license,
		FleetConfig:             &fleetCfg,
		Logger:                  logger,
		HostIdentitySCEPStorage: hostIdentitySCEPDepot,
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
