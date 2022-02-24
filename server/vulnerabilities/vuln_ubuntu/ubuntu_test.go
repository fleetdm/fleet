package vuln_ubuntu

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestParseUbuntuRepository(t *testing.T) {
	if os.Getenv("NETWORK_TEST") == "" {
		t.Skip("set environment variable NETWORK_TEST=1 to run")
	}

	// Parse a subset of the CentOS repository.
	pkgs, err := ParseUbuntuRepository(WithRoot("/centos/7/os/x86_64/repodata/"))
	require.NoError(t, err)

	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() {
		db.Close()
	})

	err = GenCentOSSqlite(db, pkgs)
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
