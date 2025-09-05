package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20250905190030(t *testing.T) {
	db := applyUpToPrev(t)

	centOSSwID := execNoErrLastID(t, db, "INSERT INTO software (name, version, source, `release`, arch, vendor, checksum) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"libcom_err", "1.42.9", "rpm_packages", "4.el9", "x86_64", "CentOS", "foo")

	fedoraSwID := execNoErrLastID(t, db, "INSERT INTO software (name, version, source, `release`, arch, vendor, checksum) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"acl", "2.3.1", "rpm_packages", "4.el9", "x86_64", "Fedora Project", "bar")

	rhelSwID := execNoErrLastID(t, db, "INSERT INTO software (name, version, source, `release`, arch, checksum) VALUES (?, ?, ?, ?, ?, ?)",
		"libcom_err", "1.42.9", "rpm_packages", "19.rhel2.0.1", "x86_64", "baz")

	// false positive; OVAL on CentOS
	execNoErr(t, db, `INSERT INTO software_cve (cve, source, software_id)
    	VALUES (?, ?, ?)`, "CVE-2019-5094", 2, centOSSwID)

	// false positive; OVAL on Fedora
	execNoErr(t, db, `INSERT INTO software_cve (cve, source, software_id)
    	VALUES (?, ?, ?)`, "CVE-2025-1234", 2, fedoraSwID)

	// true positive; Goval-Dictionary on CentOS
	execNoErr(t, db, `INSERT INTO software_cve (cve, source, software_id)
    	VALUES (?, ?, ?)`, "CVE-2025-1337", 6, centOSSwID)

	// true positive; Goval-Dictionary on Fedora
	execNoErr(t, db, `INSERT INTO software_cve (cve, source, software_id)
    	VALUES (?, ?, ?)`, "CVE-2025-4242", 6, fedoraSwID)

	// true positive; OVAL on RHEL
	execNoErr(t, db, `INSERT INTO software_cve (cve, source, software_id)
    	VALUES (?, ?, ?)`, "CVE-2019-5094", 2, rhelSwID)

	// Apply current migration.
	applyNext(t, db)

	var cveID string

	err := db.Get(&cveID, `SELECT cve FROM software_cve cve JOIN software sw ON cve.software_id = sw.id WHERE sw.id = ? ORDER BY cve ASC`, centOSSwID)
	require.NoError(t, err)
	require.Equal(t, "CVE-2025-1337", cveID)

	err = db.Get(&cveID, `SELECT cve FROM software_cve cve JOIN software sw ON cve.software_id = sw.id WHERE sw.id = ? ORDER BY cve ASC`, fedoraSwID)
	require.NoError(t, err)
	require.Equal(t, "CVE-2025-4242", cveID)

	err = db.Get(&cveID, `SELECT cve FROM software_cve cve JOIN software sw ON cve.software_id = sw.id WHERE sw.id = ? ORDER BY cve ASC`, rhelSwID)
	require.NoError(t, err)
	require.Equal(t, "CVE-2019-5094", cveID)
}
