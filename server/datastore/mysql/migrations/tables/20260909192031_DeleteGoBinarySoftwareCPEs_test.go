package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20260909192031(t *testing.T) {
	db := applyUpToPrev(t)

	insertSoftware := func(name, source string) int64 {
		return execNoErrLastID(t, db, `
			INSERT INTO software (name, version, source, checksum)
			VALUES (?, '1.0.0', ?, UNHEX(MD5(CONCAT(?, ?))))`,
			name, source, name, source)
	}
	insertCPE := func(softwareID int64, cpe string) {
		execNoErr(t, db, `INSERT INTO software_cpe (software_id, cpe) VALUES (?, ?)`, softwareID, cpe)
	}
	insertCVE := func(softwareID int64, cve string, source fleet.VulnerabilitySource) {
		execNoErr(t, db, `INSERT INTO software_cve (software_id, cve, source) VALUES (?, ?, ?)`,
			softwareID, cve, source)
	}
	count := func(query string, softwareID int64) int {
		var n int
		require.NoError(t, db.Get(&n, query, softwareID))
		return n
	}
	cpeCount := func(softwareID int64) int {
		return count(`SELECT COUNT(*) FROM software_cpe WHERE software_id = ?`, softwareID)
	}
	cveCount := func(softwareID int64) int {
		return count(`SELECT COUNT(*) FROM software_cve WHERE software_id = ?`, softwareID)
	}
	softwareCount := func(softwareID int64) int {
		return count(`SELECT COUNT(*) FROM software WHERE id = ?`, softwareID)
	}

	// A Go binary whose name collided with an unrelated product, which is how it got a CPE
	// and that product's CVEs.
	air := insertSoftware("air", "go_binaries")
	insertCPE(air, "cpe:2.3:a:adobe:air:1.0.0:*:*:*:*:*:*:*")
	insertCVE(air, "CVE-2011-0611", fleet.NVDSource)
	insertCVE(air, "CVE-2011-2110", fleet.NVDSource)

	// A Go binary whose name collided with a distro package, so it picked up an OSV row. The
	// OSV pass diffs per software ID and no longer lists this source, so nothing would ever
	// revisit this row — it has to be cleared here.
	etcd := insertSoftware("etcd", "go_binaries")
	insertCVE(etcd, "CVE-2020-15106", fleet.UbuntuOSVSource)

	// A Go binary that never matched anything: the migration must not touch the software row.
	gopls := insertSoftware("gopls", "go_binaries")

	// Other sources are still scanned through CPEs and must come out untouched. Both reuse an
	// identifier a Go binary also carried, so a delete keyed on the CVE or the CPE instead of
	// the software source would take them with it.
	acrobat := insertSoftware("Adobe Acrobat", "apps")
	insertCPE(acrobat, "cpe:2.3:a:adobe:air:1.0.0:*:*:*:*:*:*:*")
	insertCVE(acrobat, "CVE-2011-0611", fleet.NVDSource)

	openssl := insertSoftware("openssl", "deb_packages")
	insertCVE(openssl, "CVE-2020-15106", fleet.UbuntuOSVSource)

	applyNext(t, db)

	// Every CPE and vulnerability on a Go binary is gone, whichever source wrote it.
	require.Zero(t, cpeCount(air))
	require.Zero(t, cveCount(air))
	require.Zero(t, cveCount(etcd))

	// The software rows survive; only what was derived from them is deleted.
	require.Equal(t, 1, softwareCount(air))
	require.Equal(t, 1, softwareCount(etcd))
	require.Equal(t, 1, softwareCount(gopls))

	// Software from every other source keeps its CPE and its vulnerabilities.
	require.Equal(t, 1, cpeCount(acrobat))
	require.Equal(t, 1, cveCount(acrobat))
	require.Equal(t, 1, cveCount(openssl))
}
