package vulnerabilities

import (
	"context"
	"database/sql"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/vuln_centos"
	"github.com/go-kit/kit/log"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestCentOSPostProcessing(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)

	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	pkgs := make(vuln_centos.CentOSPkgSet)
	authConfigPkg := vuln_centos.CentOSPkg{
		Name:    "authconfig",
		Version: "6.2.8",
		Release: "30.el7",
		Arch:    "x86_64",
	}
	pkgs.Add(authConfigPkg, "CVE-2017-7488")
	sqlitePkg := vuln_centos.CentOSPkg{
		Name:    "sqlite",
		Version: "3.7.17",
		Release: "8.el7_7",
		Arch:    "x86_64",
	}
	pkgs.Add(sqlitePkg, "CVE-2015-3415", "CVE-2015-3416", "CVE-2015-3414")

	err = vuln_centos.GenCentOSSqlite(db, pkgs)
	require.NoError(t, err)

	vulnSoftware := []fleet.SoftwareWithCPE{
		{
			Software: fleet.Software{
				Name:    "authconfig",
				Version: "6.2.8",
				Release: "30.el7",
				Arch:    "x86_64",
				Vendor:  "CentOS",
				Vulnerabilities: fleet.VulnerabilitiesSlice{
					{
						CVE: "CVE-2017-7488",
					},
				},
			},
			CPE: 1,
		},
		{
			Software: fleet.Software{
				Name:    "sqlite",
				Version: "3.7.17",
				Release: "8.el7_7",
				Arch:    "x86_64",
				Vendor:  "CentOS",
				Vulnerabilities: fleet.VulnerabilitiesSlice{
					{
						CVE: "CVE-2015-3415",
					},
					{
						CVE: "CVE-2015-3416",
					},
					{
						CVE: "CVE-2022-9999",
					},
				},
			},
			CPE: 2,
		},
		{
			Software: fleet.Software{
				Name:    "ghostscript",
				Version: "9.25",
				Release: "5.el7",
				Arch:    "x86_64",
				Vendor:  "CentOS",
				Vulnerabilities: fleet.VulnerabilitiesSlice{
					{
						CVE: "CVE-2019-3835",
					},
				},
			},
			CPE: 3,
		},
		{
			Software: fleet.Software{
				Name:    "gnutls",
				Version: "3.3.29",
				Release: "9.el7",
				Arch:    "x86_64",
				Vendor:  "",
				Vulnerabilities: fleet.VulnerabilitiesSlice{
					{
						CVE: "CVE-8888-9999",
					},
				},
			},
			CPE: 4,
		},
	}

	ds.ListVulnerableSoftwareBySourceFunc = func(ctx context.Context, source string) ([]fleet.SoftwareWithCPE, error) {
		return vulnSoftware, nil
	}

	ds.DeleteVulnerabilitiesByCPECVEFunc = func(ctx context.Context, vulnerabilities []fleet.SoftwareVulnerability) error {
		require.Equal(t, []fleet.SoftwareVulnerability{
			{
				CPE: 1,
				CVE: "CVE-2017-7488",
			},
			{
				CPE: 2,
				CVE: "CVE-2015-3415",
			},
			{
				CPE: 2,
				CVE: "CVE-2015-3416",
			},
		}, vulnerabilities)
		return nil
	}

	err = centosPostProcessing(ctx, ds, db, log.NewNopLogger(), config.FleetConfig{})
	require.NoError(t, err)

	require.True(t, ds.ListVulnerableSoftwareBySourceFuncInvoked)
	require.True(t, ds.DeleteVulnerabilitiesByCPECVEFuncInvoked)
}

func TestCentOSPostProcessingNoPkgs(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)
	ds.ListVulnerableSoftwareBySourceFunc = func(ctx context.Context, source string) ([]fleet.SoftwareWithCPE, error) {
		t.Error("this method shouldn't be called if there are no pkgs in the CentOS table")
		return nil, nil
	}
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	err = centosPostProcessing(ctx, ds, db, log.NewNopLogger(), config.FleetConfig{})
	require.NoError(t, err)
}
