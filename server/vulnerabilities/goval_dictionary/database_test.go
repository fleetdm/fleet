package goval_dictionary

import (
	"database/sql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/oval"
	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDatabase(t *testing.T) {
	// build minimal slice of goval-dictionary sqlite schema
	sqlite, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	dbSetupQueries := []string{
		// create schema
		"CREATE TABLE packages (name TEXT NOT NULL, arch TEXT NOT NULL, version TEXT NOT NULL, definition_id INTEGER NOT NULL)",
		"CREATE TABLE definitions (id INTEGER NOT NULL PRIMARY KEY)",
		"CREATE TABLE advisories (id INTEGER NOT NULL PRIMARY KEY, definition_id INTEGER NOT NULL)",
		"CREATE TABLE cves (cve_id TEXT NOT NULL, advisory_id INTEGER NOT NULL)",
		// insert records
		`INSERT INTO packages (name, arch, version, definition_id) VALUES
		  ('expat', 'aarch64', '0:2.1.0-15.amzn2.0.3', 1), ('krb5-server', 'aarch64', '0:1.15.1-55.amzn2.2.8', 2)`,
		"INSERT INTO definitions (id) VALUES (1), (2)",
		"INSERT INTO advisories (id, definition_id) VALUES (1, 1), (2, 2)",
		`INSERT INTO cves (cve_id, advisory_id) VALUES
		   ('CVE-2022-23990', 1), ('CVE-2022-25313', 1), ('CVE-2024-37370', 2), ('CVE-2024-37371', 2)`,
	}
	for _, query := range dbSetupQueries {
		if _, err := sqlite.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	db := NewDB(sqlite, oval.NewPlatform("amzn", "Amazon Linux 2.0.0"))
	logger := kitlog.NewNopLogger()

	t.Run("Non-matching architecture", func(t *testing.T) {
		require.Len(t, db.Eval([]fleet.Software{{Name: "expat", Version: "2.1.0", Release: "", Arch: "x86_64"}}, logger), 0)
	})

	t.Run("Non-matching package name", func(t *testing.T) {
		require.Len(t, db.Eval([]fleet.Software{{Name: "expath", Version: "2.1.0", Release: "", Arch: "aarch64"}}, logger), 0)
	})

	t.Run("Fixed version", func(t *testing.T) {
		require.Len(t, db.Eval([]fleet.Software{{Name: "expath", Version: "2.1.0", Release: "15.amzn2.0.3", Arch: "aarch64"}}, logger), 0)
	})

	t.Run("Newer than fixed version", func(t *testing.T) {
		require.Len(t, db.Eval([]fleet.Software{{Name: "expath", Version: "2.1.0", Release: "15.amzn2.0.5", Arch: "aarch64"}}, logger), 0)
	})

	t.Run("Older than fixed version", func(t *testing.T) {
		vulns := db.Eval([]fleet.Software{{Name: "expat", Version: "2.1.0", Release: "", Arch: "aarch64", ID: 123}}, logger)
		require.Len(t, vulns, 2)
		require.Equal(t, "2.1.0-15.amzn2.0.3", *vulns[0].ResolvedInVersion)
		require.Equal(t, "2.1.0-15.amzn2.0.3", *vulns[1].ResolvedInVersion)
		require.Equal(t, "CVE-2022-23990", vulns[0].CVE)
		require.Equal(t, "CVE-2022-25313", vulns[1].CVE)
		require.Equal(t, uint(123), vulns[0].SoftwareID)
		require.Equal(t, uint(123), vulns[1].SoftwareID)
	})

	t.Run("Multiple packages, fixed version", func(t *testing.T) {
		require.Len(t, db.Eval([]fleet.Software{
			{Name: "expat", Version: "2.1.0", Release: "15.amzn2.1.0", Arch: "aarch64"},
			{Name: "krb5-server", Version: "1.15.1", Release: "55.amzn2.2.8", Arch: "aarch64"},
		}, logger), 0)
	})

	t.Run("Multiple packages, multiple vulnerabilities", func(t *testing.T) {
		vulns := db.Eval([]fleet.Software{
			{Name: "expat", Version: "2.1.0", Release: "15.amzn2.0.2", Arch: "aarch64", ID: 234},
			{Name: "krb5-server", Version: "1.15.1", Release: "55.amzn2.2.7", Arch: "aarch64", ID: 235},
		}, logger)
		require.Len(t, vulns, 4)

		require.Equal(t, "2.1.0-15.amzn2.0.3", *vulns[0].ResolvedInVersion)
		require.Equal(t, "2.1.0-15.amzn2.0.3", *vulns[1].ResolvedInVersion)
		require.Equal(t, "1.15.1-55.amzn2.2.8", *vulns[2].ResolvedInVersion)
		require.Equal(t, "CVE-2022-23990", vulns[0].CVE)
		require.Equal(t, "CVE-2022-25313", vulns[1].CVE)
		require.Equal(t, "CVE-2024-37370", vulns[2].CVE)
		require.Equal(t, "CVE-2024-37371", vulns[3].CVE)
		require.Equal(t, uint(234), vulns[0].SoftwareID)
		require.Equal(t, uint(234), vulns[1].SoftwareID)
		require.Equal(t, uint(235), vulns[2].SoftwareID)
		require.Equal(t, uint(235), vulns[3].SoftwareID)
	})
}
