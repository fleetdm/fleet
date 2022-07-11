package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20220708095046(t *testing.T) {
	db := applyUpToPrev(t)
	_, err := db.Exec(`INSERT INTO software (id, name, version, source, bundle_identifier, vendor, arch)
	VALUES (1, 'zchunk-libs', '1.2.1', 'rpm_packages', '', 'Fedora Project','x86_64');`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO software_cve (software_id, cve, created_at, updated_at, source)
	VALUES (1, 'CVE-2019-17006', '2022-06-19 18:04:02', '2022-07-04 14:33:04', 1);`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO software_cve (software_id, cve, created_at, updated_at, source)
	VALUES (1, 'CVE-2019-17006', '2022-06-19 18:04:02', '2022-07-04 14:33:04', 1);`)
	require.NoError(t, err)

	applyNext(t, db)

	var n uint

	// Test we removed dup
	err = db.QueryRow(`SELECT COUNT(1) FROM software_cve`).Scan(&n)
	require.NoError(t, err)
	require.Equal(t, uint(1), n)

	// Test unique constraint
	_, err = db.Exec(`INSERT IGNORE INTO software_cve (software_id, cve, created_at, updated_at, source)
	VALUES (1, 'CVE-2019-17006', '2022-06-19 18:04:02', '2022-07-04 14:33:04', 1);`)
	require.NoError(t, err)

	err = db.QueryRow(`SELECT COUNT(1) FROM software_cve`).Scan(&n)
	require.NoError(t, err)
	require.Equal(t, uint(1), n)
}
