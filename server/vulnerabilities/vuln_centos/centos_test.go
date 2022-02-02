package vuln_centos

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/go-kit/kit/log"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestLoadCentOSFixedCVEsMissingTable(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	pkgSet, err := LoadCentOSFixedCVEs(context.Background(), db, log.NewNopLogger())
	require.NoError(t, err)
	require.Nil(t, pkgSet)
}

func TestCentOSPkgSetAdd(t *testing.T) {
	pkgSet := make(CentOSPkgSet)
	authConfig := CentOSPkg{
		Name:    "authconfig",
		Version: "6.2.8",
		Release: "30.el7",
		Arch:    "x86_64",
	}
	cve1 := "CVE-2017-7488"
	pkgSet.Add(authConfig, cve1)

	cve2 := "CVE-2017-7489"
	pkgSet.Add(authConfig, cve2)

	curl := CentOSPkg{
		Name:    "curl",
		Version: "4.2",
		Release: "30.el7",
		Arch:    "x86_64",
	}
	cve3 := "CVE-2017-7490"
	pkgSet.Add(curl, cve1)
	pkgSet.Add(curl, cve3)

	require.Len(t, pkgSet, 2)

	require.Len(t, pkgSet[authConfig], 2)
	require.Contains(t, pkgSet[authConfig], cve1)
	require.Contains(t, pkgSet[authConfig], cve2)
	require.NotContains(t, pkgSet[authConfig], cve3)

	require.Len(t, pkgSet[curl], 2)
	require.Contains(t, pkgSet[curl], cve1)
	require.NotContains(t, pkgSet[curl], cve2)
	require.Contains(t, pkgSet[curl], cve3)
}

func TestParseCentOSRepository(t *testing.T) {
	if os.Getenv("NETWORK_TEST") == "" {
		t.Skip("set environment variable NETWORK_TEST=1 to run")
	}

	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	// Parse a subset of the CentOS repository.
	err = ParseCentOSRepository(db, WithRoot("/centos/7/os/x86_64/repodata/"))
	require.NoError(t, err)

	pkgSet, err := LoadCentOSFixedCVEs(context.Background(), db, log.NewNopLogger())
	require.NoError(t, err)

	// Shouldn't get _lower_ than what was reported during the development of this test (2221),
	// as these are already published releases.
	require.GreaterOrEqual(t, len(pkgSet), 2221)
	for pkg, cveSet := range pkgSet {
		require.NotEmpty(t, pkg.Name)
		require.NotEmpty(t, pkg.Version)
		require.NotEmpty(t, pkg.Release)
		require.NotEmpty(t, pkg.Arch)
		require.NotEmpty(t, cveSet)
	}

	// Check a known vulnerability fixed on a CentOS release.
	authConfig := CentOSPkg{
		Name:    "authconfig",
		Version: "6.2.8",
		Release: "30.el7",
		Arch:    "x86_64",
	}
	cve := "CVE-2017-7488"
	require.Contains(t, pkgSet, authConfig)
	require.Len(t, pkgSet[authConfig], 1)
	require.Contains(t, pkgSet[authConfig], cve)
}
