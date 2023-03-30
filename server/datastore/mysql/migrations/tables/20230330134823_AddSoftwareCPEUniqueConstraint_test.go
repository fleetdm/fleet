package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20230330134823(t *testing.T) {
	db := applyUpToPrev(t)

	_, err := db.Exec(`INSERT INTO software (id, name, version, source, bundle_identifier, vendor, arch)
	VALUES (1, 'zchunk-libs', '1.2.1', 'rpm_packages', '', 'Fedora Project','x86_64');`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO software_cpe (software_id, cpe, created_at, updated_at)
	VALUES (1, 'some_cpe', '2022-06-19 18:04:02', '2022-07-04 14:33:04');`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO software_cpe (software_id, cpe, created_at, updated_at)
	VALUES (1, 'some_cpe', '2022-06-19 18:04:02', '2022-07-04 14:33:04');`)
	require.NoError(t, err)

	applyNext(t, db)

	var n uint

	// Test we removed dup
	err = db.QueryRow(`SELECT COUNT(1) FROM software_cpe`).Scan(&n)
	require.NoError(t, err)
	require.Equal(t, uint(1), n)

	// Test unique constraint
	_, err = db.Exec(`INSERT IGNORE INTO software_cpe (software_id, cpe, created_at, updated_at)
	VALUES (1, 'some_cpe', '2022-06-19 18:04:02', '2022-07-04 14:33:04');`)
	require.NoError(t, err)

	err = db.QueryRow(`SELECT COUNT(1) FROM software_cpe`).Scan(&n)
	require.NoError(t, err)
	require.Equal(t, uint(1), n)
}
