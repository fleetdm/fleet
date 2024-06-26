package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20240521143023(t *testing.T) {
	db := applyUpToPrev(t)
	applyNext(t, db)
	const (
		insertStmt     = `INSERT INTO mdm_config_assets (name, value, md5_checksum) VALUES (?, ?, "foo")`
		selectStmt     = `SELECT * FROM mdm_config_assets WHERE name = ? AND deleted_at IS NULL`
		softDeleteStmt = `UPDATE mdm_config_assets SET deleted_at = NOW(), deletion_uuid = UUID() WHERE name = ?`
	)

	// insert two values
	execNoErr(t, db, insertStmt, "scep_cert", "foo")
	execNoErr(t, db, insertStmt, "scep_key", "var")

	type mdmAsset struct {
		ID           uint       `db:"id"`
		Name         string     `db:"name"`
		Value        string     `db:"value"`
		CreatedAt    time.Time  `db:"created_at"`
		DeletedAt    *time.Time `db:"deleted_at"`
		DeletionUUID string     `db:"deletion_uuid"`
		MD5Checksum  string     `db:"md5_checksum"`
	}
	var asset mdmAsset
	err := db.Get(&asset, selectStmt, "scep_cert")
	require.NoError(t, err)
	require.Equal(t, "scep_cert", asset.Name)
	require.Equal(t, "foo", asset.Value)
	require.NotNil(t, asset.CreatedAt)
	require.Nil(t, asset.DeletedAt)
	require.Empty(t, asset.DeletionUUID)

	// trying to insert a value with the same name fails if the
	// current one is not deleted
	_, err = db.Exec(insertStmt, "scep_cert", "foo")
	require.ErrorContains(t, err, "Duplicate entry")

	// soft delete the entry
	_, err = db.Exec(softDeleteStmt, "scep_cert")
	require.NoError(t, err)

	// try to insert again, it succeeds
	_, err = db.Exec(insertStmt, "scep_cert", "foo")
	require.NoError(t, err)
	asset = mdmAsset{}
	err = db.Get(&asset, selectStmt, "scep_cert")
	require.NoError(t, err)
	require.Equal(t, "scep_cert", asset.Name)
	require.Equal(t, "foo", asset.Value)
	require.NotNil(t, asset.CreatedAt)
	require.Nil(t, asset.DeletedAt)
	require.Empty(t, asset.DeletionUUID)
}
