package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20220704101843(t *testing.T) {
	db := applyUpToPrev(t)

	_, err := db.Exec(`INSERT INTO software (id, name, version, source, bundle_identifier, vendor, arch)
	VALUES (1, 'zchunk-libs', '1.2.1', 'rpm_packages', '', 'Fedora Project','x86_64');`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO software_cpe (id, software_id, created_at, updated_at, cpe)
	VALUES (2, 1, '2022-06-19 18:01:14', '2022-06-19 18:01:14', 'none:1704');`)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO software_cve (id, cpe_id, cve, created_at, updated_at, source)
	VALUES (3, 2, 'CVE-2019-17006', '2022-06-19 18:04:02', '2022-07-04 14:33:04', 1);`)
	require.NoError(t, err)

	applyNext(t, db)

	var softwareId uint
	err = db.QueryRow(`SELECT software_id FROM software_cve WHERE id = 3`).Scan(&softwareId)
	require.NoError(t, err)
	require.Equal(t, uint(1), softwareId)
}
