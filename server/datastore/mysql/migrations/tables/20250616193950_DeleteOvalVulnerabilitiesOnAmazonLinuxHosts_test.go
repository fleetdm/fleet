package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20250616193950(t *testing.T) {
	db := applyUpToPrev(t)

	amznSwID := execNoErrLastID(t, db, "INSERT INTO software (name, version, source, `release`, arch, vendor, checksum) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"libcom_err", "1.42.9", "rpm_packages", "19.amzn2.0.1", "x86_64", "Amazon Linux", "foo")

	rhelSwID := execNoErrLastID(t, db, "INSERT INTO software (name, version, source, `release`, arch, checksum) VALUES (?, ?, ?, ?, ?, ?)",
		"libcom_err", "1.42.9", "rpm_packages", "19.rhel2.0.1", "x86_64", "bar")

	// false positive; OVAL on Amazon Linux
	execNoErr(t, db, `INSERT INTO software_cve (cve, source, software_id)
    	VALUES (?, ?, ?)`, "CVE-2019-5094", 2, amznSwID)

	// true positive; Goval-Dictionary on Amazon Linux
	execNoErr(t, db, `INSERT INTO software_cve (cve, source, software_id)
    	VALUES (?, ?, ?)`, "CVE-2025-1337", 6, amznSwID)

	// true positive; OVAL on RHEL
	execNoErr(t, db, `INSERT INTO software_cve (cve, source, software_id)
    	VALUES (?, ?, ?)`, "CVE-2019-5094", 2, rhelSwID)

	// Apply current migration.
	applyNext(t, db)

	var cveID string

	err := db.Get(&cveID, `SELECT cve FROM software_cve cve JOIN software sw ON cve.software_id = sw.id WHERE sw.id = ? ORDER BY cve ASC`, amznSwID)
	require.NoError(t, err)
	require.Equal(t, "CVE-2025-1337", cveID)

	err = db.Get(&cveID, `SELECT cve FROM software_cve cve JOIN software sw ON cve.software_id = sw.id WHERE sw.id = ? ORDER BY cve ASC`, rhelSwID)
	require.NoError(t, err)
	require.Equal(t, "CVE-2019-5094", cveID)
}
