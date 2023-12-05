package tables

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUp_20230518114155(t *testing.T) {
	db := applyUpToPrev(t)

	insertStmt := `
          INSERT INTO  mdm_idp_accounts
            (uuid, username, salt, entropy, iterations)
          VALUES
            (?, ?, ?, ?, ?)
	`
	uuidVal := uuid.New().String()
	execNoErr(t, db, insertStmt, uuidVal, "test@example.com", "salt", "entropy", 10000)

	applyNext(t, db)

	// retrieve the stored value
	var mdmIdPAccount struct {
		UUID     string
		Username string
		Fullname string
	}
	err := db.Get(&mdmIdPAccount, "SELECT * FROM mdm_idp_accounts WHERE uuid = ?", uuidVal)
	require.NoError(t, err)
	require.Equal(t, uuidVal, mdmIdPAccount.UUID)
	require.Equal(t, "test@example.com", mdmIdPAccount.Username)
	require.Equal(t, "", mdmIdPAccount.Fullname)

	insertStmt = `
          INSERT INTO  mdm_idp_accounts
            (uuid, username, fullname)
          VALUES
            (?, ?, ?)
	`
	uuidVal = uuid.New().String()
	execNoErr(t, db, insertStmt, uuidVal, "test+1@example.com", "Foo Bar")
	err = db.Get(&mdmIdPAccount, "SELECT * FROM mdm_idp_accounts WHERE uuid = ?", uuidVal)
	require.NoError(t, err)
	require.Equal(t, uuidVal, mdmIdPAccount.UUID)
	require.Equal(t, "test+1@example.com", mdmIdPAccount.Username)
	require.Equal(t, "Foo Bar", mdmIdPAccount.Fullname)
}
