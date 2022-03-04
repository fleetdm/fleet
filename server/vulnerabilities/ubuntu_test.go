package vulnerabilities

import (
	"context"
	"database/sql"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/vuln_ubuntu"
	"github.com/go-kit/kit/log"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestUbuntuPostProcessing(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)

	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	// see https://ubuntu.com/security/notices/USN-4038-1 for details related to the following cves.
	fixedCVEsByPackage := vuln_ubuntu.FixedCVEsByPackage{
		{
			Name:    "bzip2",
			Version: "1.0.6-8.1ubuntu0.2",
		}: {
			"CVE-2016-3189":  {},
			"CVE-2016-12900": {},
		},
	}

	err = vuln_ubuntu.GenUbuntuSqlite(db, fixedCVEsByPackage)
	require.NoError(t, err)

	vulnSoftware := []fleet.SoftwareWithCPE{
		{
			Software: fleet.Software{
				Name:    "bzip2",
				Version: "1.0.6-8.1ubuntu0.2",
				Release: "8.1ubuntu0.2",
				Arch:    "amd64",
				Vendor:  "Ubuntu",
				Vulnerabilities: fleet.VulnerabilitiesSlice{
					{
						CVE: "CVE-2016-3189",
					},
					{
						CVE: "CVE-2016-12900",
					},
				},
			},
			CPEID: 1,
		},
	}
	ds.ListVulnerableSoftwareBySourceFunc = func(ctx context.Context, source string) ([]fleet.SoftwareWithCPE, error) {
		return vulnSoftware, nil
	}

	ds.DeleteVulnerabilitiesByCPECVEFunc = func(ctx context.Context, vulnerabilities []fleet.SoftwareVulnerability) error {
		require.Equal(t, []fleet.SoftwareVulnerability{
			{
				CPEID: 1,
				CVE:   "CVE-2016-3189",
			},
			{
				CPEID: 1,
				CVE:   "CVE-2016-12900",
			},
		}, vulnerabilities)
		return nil
	}

	err = ubuntuPostProcessing(ctx, ds, db, log.NewNopLogger(), config.FleetConfig{})
	require.NoError(t, err)

	require.True(t, ds.ListVulnerableSoftwareBySourceFuncInvoked)
	require.True(t, ds.DeleteVulnerabilitiesByCPECVEFuncInvoked)
}
