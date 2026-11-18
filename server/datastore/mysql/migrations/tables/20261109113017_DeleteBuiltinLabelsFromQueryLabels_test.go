package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261109113017(t *testing.T) {
	db := applyUpToPrev(t)

	var macOSID, allHostsID uint
	require.NoError(t, db.Get(&macOSID, `SELECT id FROM labels WHERE name = 'macOS' AND label_type = 1`))
	require.NoError(t, db.Get(&allHostsID, `SELECT id FROM labels WHERE name = 'All Hosts' AND label_type = 1`))
	customID := execNoErrLastID(t, db, `INSERT INTO labels (name, query, label_type) VALUES ('custom', 'SELECT 1', 0)`)

	newQuery := func(name string) int64 {
		return execNoErrLastID(t, db, `INSERT INTO queries (name, description, query) VALUES (?, '', 'SELECT 1')`, name)
	}
	builtinOnly := newQuery("builtin-only")
	mixed := newQuery("mixed")
	customOnly := newQuery("custom-only")

	execNoErr(t, db, `INSERT INTO query_labels (query_id, label_id, require_all) VALUES
		(?, ?, 0),
		(?, ?, 1), (?, ?, 1),
		(?, ?, 0)`,
		builtinOnly, macOSID,
		mixed, allHostsID, mixed, customID,
		customOnly, customID,
	)

	applyNext(t, db)

	type queryLabel struct {
		QueryID    int64 `db:"query_id"`
		LabelID    int64 `db:"label_id"`
		RequireAll bool  `db:"require_all"`
	}
	var got []queryLabel
	require.NoError(t, db.Select(&got, `SELECT query_id, label_id, require_all FROM query_labels ORDER BY query_id, label_id`))
	require.Equal(t, []queryLabel{
		{QueryID: mixed, LabelID: customID, RequireAll: true},
		{QueryID: customOnly, LabelID: customID, RequireAll: false},
	}, got)

	// Built-in labels themselves are untouched.
	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM labels WHERE id IN (?, ?)`, macOSID, allHostsID))
	require.Equal(t, 2, count)
}
