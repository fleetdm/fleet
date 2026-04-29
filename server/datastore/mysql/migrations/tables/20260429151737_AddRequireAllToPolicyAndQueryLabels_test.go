package tables

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20260429151737(t *testing.T) {
	db := applyUpToPrev(t)

	// Seed both tables with pre-migration rows so we can verify the migration
	// preserves their semantics under the new schema.
	policyIncludeAnyID := execNoErrLastID(t, db,
		`INSERT INTO policies (name, description, query, checksum) VALUES ('p_include_any', '', 'SELECT 1', 'cIA')`)
	policyExcludeAnyID := execNoErrLastID(t, db,
		`INSERT INTO policies (name, description, query, checksum) VALUES ('p_exclude_any', '', 'SELECT 1', 'cEA')`)
	policyLabelIncludeAnyRowID := execNoErrLastID(t, db,
		`INSERT INTO policy_labels (policy_id, label_id, exclude) VALUES (?, 1, 0)`, policyIncludeAnyID)
	policyLabelExcludeAnyRowID := execNoErrLastID(t, db,
		`INSERT INTO policy_labels (policy_id, label_id, exclude) VALUES (?, 2, 1)`, policyExcludeAnyID)

	queryIncludeAnyID := execNoErrLastID(t, db,
		`INSERT INTO queries (name, description, query) VALUES ('q_include_any', '', 'SELECT 1')`)
	queryLabelIncludeAnyRowID := execNoErrLastID(t, db,
		`INSERT INTO query_labels (query_id, label_id) VALUES (?, 1)`, queryIncludeAnyID)

	applyNext(t, db)

	// policy_labels: existing rows keep their exclude bit and default require_all=0.
	var policyRows []struct {
		ID         int64 `db:"id"`
		Exclude    bool  `db:"exclude"`
		RequireAll bool  `db:"require_all"`
	}
	err := sqlx.SelectContext(context.Background(), db, &policyRows,
		`SELECT id, exclude, require_all FROM policy_labels`)
	require.NoError(t, err)
	require.Len(t, policyRows, 2)
	for _, r := range policyRows {
		switch r.ID {
		case policyLabelIncludeAnyRowID:
			require.False(t, r.Exclude)
			require.False(t, r.RequireAll)
		case policyLabelExcludeAnyRowID:
			require.True(t, r.Exclude)
			require.False(t, r.RequireAll)
		default:
			t.Fatalf("unexpected policy_labels row id %d", r.ID)
		}
	}

	// New include_all row can be inserted into policy_labels.
	policyIncludeAllID := execNoErrLastID(t, db,
		`INSERT INTO policies (name, description, query, checksum) VALUES ('p_include_all', '', 'SELECT 1', 'cIL')`)
	policyIncludeAllRowID := execNoErrLastID(t, db,
		`INSERT INTO policy_labels (policy_id, label_id, exclude, require_all) VALUES (?, 3, 0, 1)`, policyIncludeAllID)
	var pl struct {
		Exclude    bool `db:"exclude"`
		RequireAll bool `db:"require_all"`
	}
	require.NoError(t, sqlx.GetContext(context.Background(), db, &pl,
		`SELECT exclude, require_all FROM policy_labels WHERE id = ?`, policyIncludeAllRowID))
	require.False(t, pl.Exclude)
	require.True(t, pl.RequireAll)

	// query_labels: existing row defaults to require_all=0 (i.e. include_any).
	var ql struct {
		RequireAll bool `db:"require_all"`
	}
	require.NoError(t, sqlx.GetContext(context.Background(), db, &ql,
		`SELECT require_all FROM query_labels WHERE id = ?`, queryLabelIncludeAnyRowID))
	require.False(t, ql.RequireAll, "pre-existing query_labels row should default to require_all=false")

	// New include_all row can be inserted into query_labels.
	queryIncludeAllID := execNoErrLastID(t, db,
		`INSERT INTO queries (name, description, query) VALUES ('q_include_all', '', 'SELECT 1')`)
	queryIncludeAllRowID := execNoErrLastID(t, db,
		`INSERT INTO query_labels (query_id, label_id, require_all) VALUES (?, 2, 1)`, queryIncludeAllID)
	require.NoError(t, sqlx.GetContext(context.Background(), db, &ql,
		`SELECT require_all FROM query_labels WHERE id = ?`, queryIncludeAllRowID))
	require.True(t, ql.RequireAll)
}
