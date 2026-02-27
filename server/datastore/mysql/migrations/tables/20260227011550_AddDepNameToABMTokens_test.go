package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20260227011550(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert an existing ABM token with an organization_name (simulating pre-migration state).
	_, err := db.Exec(`
		INSERT INTO abm_tokens (organization_name, apple_id, terms_expired, renew_at, token)
		VALUES ('Test Org', 'admin@testorg.com', 0, ?, 'encryptedtoken')
	`, time.Now().Add(365*24*time.Hour))
	require.NoError(t, err)

	// Verify the unique constraint on organization_name exists before migration.
	_, err = db.Exec(`
		INSERT INTO abm_tokens (organization_name, apple_id, terms_expired, renew_at, token)
		VALUES ('Test Org', 'admin2@testorg.com', 0, ?, 'encryptedtoken2')
	`, time.Now().Add(365*24*time.Hour))
	require.Error(t, err, "should fail due to unique constraint on organization_name")

	// Apply current migration.
	applyNext(t, db)

	// After migration, dep_name should be backfilled to organization_name.
	var depName string
	err = db.QueryRow(`SELECT dep_name FROM abm_tokens WHERE organization_name = 'Test Org'`).Scan(&depName)
	require.NoError(t, err)
	require.Equal(t, "Test Org", depName)

	// After migration, multiple tokens from the same org should be allowed.
	_, err = db.Exec(`
		INSERT INTO abm_tokens (organization_name, dep_name, apple_id, terms_expired, renew_at, token)
		VALUES ('Test Org', 'test_consumer_key_2', 'admin2@testorg.com', 0, ?, 'encryptedtoken2')
	`, time.Now().Add(365*24*time.Hour))
	require.NoError(t, err, "should succeed: multiple tokens from same org now allowed")

	// Uploading the same token (same dep_name) twice should still fail.
	_, err = db.Exec(`
		INSERT INTO abm_tokens (organization_name, dep_name, apple_id, terms_expired, renew_at, token)
		VALUES ('Test Org', 'test_consumer_key_2', 'admin3@testorg.com', 0, ?, 'encryptedtoken3')
	`, time.Now().Add(365*24*time.Hour))
	require.Error(t, err, "should fail: duplicate dep_name not allowed")
}
