package tables

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUp_20230303135738(t *testing.T) {
	db := applyUpToPrev(t)
	applyNext(t, db)

	insertStmt := `
    INSERT INTO  mdm_idp_accounts
      (uuid, username, salt, entropy, iterations)
		VALUES
      (?, ?, ?, ?, ?)
	`

	// insert a value
	uuidVal := uuid.New().String()
	execNoErr(t, db, insertStmt, uuidVal, "test@example.com", "salt", "entropy", 10000)

	// retrieve the stored value
	var mdmIdPAccount struct {
		UUID       string
		Username   string
		Salt       string
		Entropy    string
		Iterations int
	}
	err := db.Get(&mdmIdPAccount, "SELECT * FROM mdm_idp_accounts WHERE uuid = ?", uuidVal)
	require.NoError(t, err)
	require.Equal(t, uuidVal, mdmIdPAccount.UUID)
	require.Equal(t, "test@example.com", mdmIdPAccount.Username)
	require.Equal(t, "salt", mdmIdPAccount.Salt)
	require.Equal(t, "entropy", mdmIdPAccount.Entropy)
	require.Equal(t, 10000, mdmIdPAccount.Iterations)

	// uuid is the primary key, can't insert duplicates
	_, err = db.Exec(insertStmt, uuidVal, "another@example.com", "salt", "entropy", 50000)
	require.Error(t, err)

}
