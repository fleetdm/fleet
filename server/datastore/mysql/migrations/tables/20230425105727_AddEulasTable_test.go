package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20230425105727(t *testing.T) {
	db := applyUpToPrev(t)
	applyNext(t, db)

	insertStmt := `
          INSERT INTO eulas (id, token, name, bytes)
	  VALUES (?, ?, ?, ?)
	`

	selectStmt := `
	  SELECT id, name, bytes, token
	  FROM eulas
	  WHERE token = ?
	`

	_, err := db.Exec(insertStmt, 1, "ABC-DEF", "eula.pdf", []byte("eula"))
	require.NoError(t, err)

	_, err = db.Exec(insertStmt, 1, "ABC-DEF", "eula_2.pdf", []byte("eula_2"))
	require.ErrorContains(t, err, "Error 1062")

	_, err = db.Exec(insertStmt, 2, "ABC-DEF", "eula_2.pdf", []byte("eula_2"))
	require.NoError(t, err)

	var (
		token string
		name  string
		bytes []byte
		id    uint
	)

	err = db.QueryRow(selectStmt, "ABC-DEF").Scan(&id, &name, &bytes, &token)
	require.NoError(t, err)
	require.Equal(t, "ABC-DEF", token)
	require.Equal(t, "eula.pdf", name)
	require.Equal(t, []byte("eula"), bytes)
}
