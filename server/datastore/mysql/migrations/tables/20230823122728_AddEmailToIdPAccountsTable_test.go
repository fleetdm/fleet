package tables

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUp_20230823122728(t *testing.T) {
	db := applyUpToPrev(t)
	insertStmt := `
          INSERT INTO  mdm_idp_accounts
            (uuid, username, fullname)
          VALUES
            (?, ?, ?)
	`
	uuidVal := uuid.New().String()
	execNoErr(t, db, insertStmt, uuidVal, "test", "Foo Bar")

	applyNext(t, db)

	// retrieve the stored value
	var mdmIdPAccount struct {
		UUID     string
		Username string
		Fullname string
		Email    string
	}
	err := db.Get(&mdmIdPAccount, "SELECT * FROM mdm_idp_accounts WHERE uuid = ?", uuidVal)
	require.NoError(t, err)
	require.Equal(t, uuidVal, mdmIdPAccount.UUID)
	require.Equal(t, "test", mdmIdPAccount.Username)
	require.Equal(t, "Foo Bar", mdmIdPAccount.Fullname)
	require.Equal(t, "", mdmIdPAccount.Email)

	insertStmt = `
          INSERT INTO  mdm_idp_accounts
            (uuid, username, fullname, email)
          VALUES
            (?, ?, ?, ?)
	`
	uuidVal = uuid.New().String()
	execNoErr(t, db, insertStmt, uuidVal, "test", "Foo Bar", "test@example.com")
	err = db.Get(&mdmIdPAccount, "SELECT * FROM mdm_idp_accounts WHERE uuid = ?", uuidVal)
	require.NoError(t, err)
	require.Equal(t, uuidVal, mdmIdPAccount.UUID)
	require.Equal(t, "test", mdmIdPAccount.Username)
	require.Equal(t, "Foo Bar", mdmIdPAccount.Fullname)
	require.Equal(t, "test@example.com", mdmIdPAccount.Email)
}
