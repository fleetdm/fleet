package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260507160833(t *testing.T) {
	db := applyUpToPrev(t)

	// Pre-existing row using a current enum value: confirms the migration
	// preserves rows authored under the old NOT NULL DEFAULT 'ndes' shape.
	execNoErr(t, db, `
		INSERT INTO host_mdm_managed_certificates
			(host_uuid, profile_uuid, ca_name, type)
		VALUES (?, ?, ?, ?)`,
		"host-1", "profile-1", "ca-existing", "ndes")

	applyNext(t, db)

	// Existing row still readable with the same value.
	var existingType string
	require.NoError(t, db.Get(&existingType, `
		SELECT type FROM host_mdm_managed_certificates
		WHERE host_uuid = 'host-1' AND profile_uuid = 'profile-1' AND ca_name = 'ca-existing'`))
	require.Equal(t, "ndes", existingType)

	// NULL accepted for ingestion-created rows where Fleet doesn't know the CA type.
	execNoErr(t, db, `
		INSERT INTO host_mdm_managed_certificates
			(host_uuid, profile_uuid, ca_name)
		VALUES (?, ?, ?)`,
		"host-1", "profile-3", "non-proxied-ca")
	var nullType *string
	require.NoError(t, db.Get(&nullType, `
		SELECT type FROM host_mdm_managed_certificates
		WHERE host_uuid = 'host-1' AND profile_uuid = 'profile-3' AND ca_name = 'non-proxied-ca'`))
	require.Nil(t, nullType, "type should be NULL when not specified after the migration")
}
