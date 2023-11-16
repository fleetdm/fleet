package tables

import (
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20230405232025(t *testing.T) {
	db := applyUpToPrev(t)
	applyNext(t, db)

	var (
		teamID uint
		name   string
		sha    []byte
		bytes  []byte
		token  string
	)

	insertStmt := `
          INSERT INTO mdm_apple_bootstrap_packages (team_id, name, sha256, bytes, token)
	  VALUES (?, ?, ?, ?, ?)
	`

	selectStmt := `
	  SELECT team_id, name, sha256, bytes, token
	  FROM mdm_apple_bootstrap_packages
	  WHERE team_id = ?
	`

	hash := sha256.New()
	hash.Write([]byte("test"))
	sum := hash.Sum(nil)
	_, err := db.Exec(insertStmt, 0, "b1_t0.pkg", sum, []byte("all teams"), "tok_0")
	require.NoError(t, err)
	_, err = db.Exec(insertStmt, 1, "b1_t1.pkg", sum, []byte("team_1"), "tok_1")
	require.NoError(t, err)

	// team_id is the primary key
	_, err = db.Exec(insertStmt, 1, "b1_t2.pkg", sum, []byte("team_1_pkg_2"), "tok_2")
	require.ErrorContains(t, err, "Error 1062")

	// uniqueness constraint on token
	_, err = db.Exec(insertStmt, 2, "b1_t2.pkg", sum, []byte("team_1_pkg_2"), "tok_1")
	require.ErrorContains(t, err, "Error 1062")

	err = db.QueryRow(selectStmt, 1).Scan(&teamID, &name, &sha, &bytes, &token)
	require.NoError(t, err)
	require.EqualValues(t, 1, teamID)
	require.Equal(t, "b1_t1.pkg", name)
	require.Equal(t, sum, sha)
	require.Equal(t, []byte("team_1"), bytes)
	require.Equal(t, "tok_1", token)
}
