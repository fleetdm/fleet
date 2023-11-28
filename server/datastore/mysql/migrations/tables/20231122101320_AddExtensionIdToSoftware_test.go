package tables

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUp_20231122101320(t *testing.T) {
	db := applyUpToPrev(t)

	softwareNames := []string{"1Password", "AdBlocker"}

	setupStmts := fmt.Sprintf(`
		INSERT INTO software (id, name, version, source) VALUES
			(1,'%s','version','source');
	`, softwareNames[0],
	)

	_, err := db.Exec(setupStmts)
	require.NoError(t, err)
	// Apply current migration.
	applyNext(t, db)

	stmt := `
		SELECT name, extension_id FROM software WHERE id = 1;
	`
	rows, err := db.Query(stmt)
	require.NoError(t, rows.Err())
	require.NoError(t, err)
	defer rows.Close()

	count := 0
	for rows.Next() {
		count += 1
		var name, extensionId string
		err := rows.Scan(&name, &extensionId)
		require.NoError(t, err)
		require.Equal(t, softwareNames[0], name)
		require.Equal(t, "", extensionId)
	}
	require.Equal(t, 1, count)

	extensions := []string{"abc", "def"}
	stmt = fmt.Sprintf(`
		INSERT INTO software (id, name, version, source, extension_id) VALUES
			(2,'%s','version','source', '%s');
	`, softwareNames[1], extensions[0],
	)
	_, err = db.Exec(stmt)
	require.NoError(t, err)

	stmt = fmt.Sprintf(
		`
		INSERT INTO software (id, name, version, source, extension_id) VALUES
			(3,'%s','version','source', '%s');
	`, softwareNames[1], extensions[1],
	)
	_, err = db.Exec(stmt)
	require.NoError(t, err)

	stmt = `
		SELECT name, extension_id FROM software WHERE id = 2;
	`
	rows, err = db.Query(stmt)
	require.NoError(t, rows.Err())
	require.NoError(t, err)
	defer rows.Close()

	count = 0
	for rows.Next() {
		count += 1
		var name, extensionId string
		err := rows.Scan(&name, &extensionId)
		require.NoError(t, err)
		require.Equal(t, softwareNames[1], name)
		require.Equal(t, extensions[0], extensionId)
	}
	require.Equal(t, 1, count)

}
