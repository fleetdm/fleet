package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260217141240(t *testing.T) {
	db := applyUpToPrev(t)

	_, err := db.Exec(`
		INSERT INTO labels (name, query, platform)
		VALUES ('label 1', 'SELECT 1', 'baaaad1')
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO labels (name, query, platform)
		VALUES ('label 2', 'SELECT 1', 'baaaad2')
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO labels (name, query, platform)
		VALUES ('label 3', 'SELECT 1', '')
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO labels (name, query, platform)
		VALUES ('label 4', 'SELECT 1', 'ubuntu')
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO labels (name, query, platform)
		VALUES ('label 5', 'SELECT 1', 'windows')
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO labels (name, query, platform)
		VALUES ('label 6', 'SELECT 1', 'centos')
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO labels (name, query, platform)
		VALUES ('label 7', 'SELECT 1', 'windows')
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO labels (name, query, platform)
		VALUES ('label 8', 'SELECT 1', 'windows')
	`)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	type rowData struct {
		Name  string `db:"platform"`
		Count uint   `db:"n"`
	}
	var results []rowData
	query := "SELECT platform, COUNT(1) AS n FROM labels WHERE name LIKE 'label %' GROUP BY platform"
	err = db.Select(&results, query)
	require.NoError(t, err)

	actualResult := make(map[string]uint, len(results))
	for _, r := range results {
		actualResult[r.Name] = r.Count
	}

	expectedResult := map[string]uint{
		"ubuntu":  1,
		"centos":  1,
		"":        3,
		"windows": 3,
	}
	require.Equal(t, expectedResult, actualResult)
}
