//go:build !windows

// Windows is disabled because the TPM simulator requires CGO, which causes lint failures on Windows.

package hostidentity

import (
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/service/integrationtest"
	"github.com/go-kit/kit/log"
	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

// enrollOrbitResponse is the response structure for orbit enrollment
type enrollOrbitResponse struct {
	OrbitNodeKey string `json:"orbit_node_key,omitempty"`
	Err          error  `json:"error,omitempty"`
}

// orbitConfigRequest is used for orbit config endpoint requests
type orbitConfigRequest struct {
	OrbitNodeKey string `json:"orbit_node_key"`
}

// osqueryConfigRequest is used for osquery config endpoint requests
type osqueryConfigRequest struct {
	NodeKey string `json:"node_key"`
}

type Suite struct {
	integrationtest.BaseSuite
}

func SetUpSuite(t *testing.T, uniqueTestName string, requireSignature bool) *Suite {
	return SetUpSuiteWithConfig(t, uniqueTestName, requireSignature, nil)
}

func SetUpSuiteWithConfig(t *testing.T, uniqueTestName string, requireSignature bool, configModifier func(cfg *config.FleetConfig)) *Suite {
	// Note: t.Parallel() is called when MySQL datastore options are processed
	license := &fleet.LicenseInfo{
		Tier: fleet.TierPremium,
	}
	ds, fleetCfg, fleetSvc, ctx := integrationtest.SetUpMySQLAndService(t, uniqueTestName, &service.TestServerOpts{
		License: license,
	})

	// Apply config modifications
	if configModifier != nil {
		configModifier(&fleetCfg)
	}

	logger := log.NewLogfmtLogger(os.Stdout)
	hostIdentitySCEPDepot, err := ds.NewHostIdentitySCEPDepot(kitlog.With(logger, "component", "host-id-scep-depot"), &fleetCfg)
	require.NoError(t, err)
	users, server := service.RunServerForTestsWithServiceWithDS(t, ctx, ds, fleetSvc, &service.TestServerOpts{
		License:     license,
		FleetConfig: &fleetCfg,
		Logger:      logger,
		HostIdentity: &service.HostIdentity{
			SCEPStorage:                 hostIdentitySCEPDepot,
			RequireHTTPMessageSignature: requireSignature,
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
