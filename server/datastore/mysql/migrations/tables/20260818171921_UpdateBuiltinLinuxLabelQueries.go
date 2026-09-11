package tables

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func init() {
	MigrationClient.AddMigration(Up_20260818171921, Down_20260818171921)
}

// osquery reports a distribution's own ID in os_version.platform and lists only
// its ancestors in platform_like. The Red Hat family is the exception: Fedora,
// RHEL, Rocky Linux, AlmaLinux and CentOS Stream all report "rhel" for both, so
// labels singling out one of them have to match on os_version.name.
func Up_20260818171921(tx *sql.Tx) error {
	updatedAt := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)

	updates := []struct {
		name  string
		query string
	}{
		// Pop!_OS, Linux Mint, Zorin OS and KDE neon carry "ubuntu" in
		// platform_like only.
		{
			name:  fleet.BuiltinLabelNameUbuntuLinux,
			query: "select 1 from os_version where platform = 'ubuntu' or platform_like like '%ubuntu%';",
		},
		// Fedora 33 and earlier report name "Fedora", not "Fedora Linux", as does
		// Fedora Asahi Remix with a suffix.
		{
			name:  fleet.BuiltinLabelFedoraLinux,
			query: "select 1 from os_version where name like '%fedora%';",
		},
	}

	const stmt = "UPDATE labels SET query = ?, updated_at = ? WHERE name = ? AND label_type = ?"
	for _, u := range updates {
		if _, err := tx.Exec(stmt, u.query, updatedAt, u.name, fleet.LabelTypeBuiltIn); err != nil {
			return fmt.Errorf("update %s builtin label query: %w", u.name, err)
		}
	}
	return nil
}

func Down_20260818171921(tx *sql.Tx) error {
	return nil
}
