package vuln_ubuntu

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/go-kit/log"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestParseUbuntuRepository(t *testing.T) {
	if os.Getenv("NETWORK_TEST") == "" {
		t.Skip("set environment variable NETWORK_TEST=1 to run")
	}

	// Parse a subset of the Ubuntu repository
	pkgs, err := ParseUbuntuRepository(WithRoot("/ubuntu/pool/main/a/aspell/"))
	require.NoError(t, err)

	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() {
		db.Close()
	})

	err = GenUbuntuSqlite(db, pkgs)
	require.NoError(t, err)

	fixedCVEsByPackage, err := LoadUbuntuFixedCVEs(context.Background(), db, log.NewNopLogger())
	require.NoError(t, err)

	// aspell_0.60.8-1ubuntu0.1 should have fixed CVE-2019-25051
	// https://ubuntu.pkgs.org/20.04/ubuntu-updates-main-amd64/aspell_0.60.8-1ubuntu0.1_amd64.deb.html
	pkg := Package{
		Name:    "aspell",
		Version: "0.60.8-1ubuntu0.1",
	}
	require.Contains(t, fixedCVEsByPackage[pkg], "CVE-2019-25051")
}
