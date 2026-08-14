package tables

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func init() {
	MigrationClient.AddMigration(Up_20260814130009, Down_20260814130009)
}

// Values below were measured with osquery 5.18.1 against each distribution's
// official container image, except Pop!_OS, measured on a VM, and Zorin OS
// (desktop-only, no image), measured by mounting its /etc/os-release. Grouped by
// what each group implies for the queries, as "platform / platform_like":
//
//	Ubuntu and Debian themselves:
//	  Ubuntu 22.04         ubuntu / debian
//	  Debian 12            debian / (empty)   <- no platform_like at all, so a
//	                                             label covering Debian has to
//	                                             match on platform
//	Debian-family derivatives, which name only their ancestors:
//	  Pop!_OS 24           pop / ubuntu debian
//	  Linux Mint 21        linuxmint / ubuntu  <- "ubuntu" but not "debian", so
//	  Zorin OS 17          zorin / ubuntu         Debian-based has to match
//	  KDE neon             neon / ubuntu debian   platform_like "%ubuntu%" too
//	  Kali rolling         kali / debian
//	The Red Hat family all collapses onto rhel / rhel, so labels that single out
//	one of its members must match on os_version.name (shown here):
//	  Fedora 40            "Fedora Linux"
//	  Fedora 33            "Fedora"           <- pre-34 name, so the Fedora
//	                                             label needs LIKE, not =
//	  RHEL 9               "Red Hat Enterprise Linux"
//	  Rocky Linux 9        "Rocky Linux"
//	  AlmaLinux 9          "AlmaLinux"
//	  CentOS Stream 9      "CentOS Stream"
//	RPM-based distributions outside that collapse:
//	  Amazon Linux 2023    amzn / fedora
//	  SLES 15.6            sles / suse
//	  openSUSE Leap 15.5   opensuse-leap / suse opensuse
//	  openSUSE Tumbleweed  opensuse-tumbleweed / opensuse suse
//	Negative control, matching neither Debian-based nor RPM-based:
//	  Arch Linux           arch / (empty)
func Up_20260814130009(tx *sql.Tx) error {
	updatedAt := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)

	updates := []struct {
		name  string
		query string
	}{
		// Ubuntu-derived distributions (Pop!_OS, Linux Mint, Zorin) report their
		// own value in platform and list "ubuntu" in platform_like, so an exact
		// platform match misses them.
		{
			name:  fleet.BuiltinLabelNameUbuntuLinux,
			query: "select 1 from os_version where platform = 'ubuntu' or platform_like like '%ubuntu%';",
		},
		// Fedora releases before 34 report name "Fedora" rather than "Fedora
		// Linux", and Fedora Asahi Remix reports "Fedora Linux Asahi Remix", so an
		// exact name match misses both. Matching on name (rather than platform or
		// platform_like) is what keeps the rest of the Red Hat family out: RHEL,
		// Rocky Linux and CentOS Stream all report platform and platform_like
		// "rhel" just like Fedora does, but none of their names contain "fedora".
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

func Down_20260814130009(tx *sql.Tx) error {
	return nil
}
