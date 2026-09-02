package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rename(old, newVID int64) migrationRename {
	return migrationRename{
		oldVersionID:  old,
		newVersionID:  newVID,
		migrationType: "tables",
		commitSHA:     "abcdef123456",
	}
}

// requireVersionOrder asserts that rows ordered by id have strictly
// increasing version_ids, ids are contiguous from 1, and the final state
// validator agrees.
func requireVersionOrder(t *testing.T, rows []tableRow, wantVersions []int64) {
	t.Helper()
	require.Len(t, rows, len(wantVersions))
	for i, row := range rows {
		assert.Equal(t, int64(i+1), row.ID, "ids must be compacted to 1..N")
		assert.Equal(t, wantVersions[i], row.VersionID, "version at position %d", i)
	}
	assert.Empty(t, validateFinalTableState(tableStatusName, rows))
}

// Down-move renumber where the moved rows sit mid-table and at the tail, the
// shape of the 4.90.2 -> 4.91.0 re-timestamping applied to a database that
// tracked main (rows applied between and after the moved migrations).
func TestSimulateDownMoveAtTail(t *testing.T) {
	rows := []tableRow{
		{ID: 658, VersionID: 20260723181411, IsApplied: true},
		{ID: 659, VersionID: 20260724134801, IsApplied: true},
		{ID: 660, VersionID: 20260727083533, IsApplied: true},
		{ID: 664, VersionID: 20260731100711, IsApplied: true}, // moves down, mid-table
		{ID: 676, VersionID: 20260810192005, IsApplied: true},
		{ID: 679, VersionID: 20260812170926, IsApplied: true}, // moves down, at tail
	}
	renames := []migrationRename{
		rename(20260731100711, 20260723181412),
		rename(20260812170926, 20260723181413),
	}

	simulated, _, issues := simulateTableSQL(tableStatusName, rows, renames)
	require.Empty(t, issues)
	requireVersionOrder(t, simulated, []int64{
		20260723181411,
		20260723181412,
		20260723181413,
		20260724134801,
		20260727083533,
		20260810192005,
	})
}

// A database where the version remaps were already applied without reordering
// ids, plus a duplicate row inserted by goose re-running an already-applied
// migration. The remaps must no-op, the duplicate must be removed, and the
// rebuild must still fix the ordering.
func TestSimulateHalfFixedStateWithDuplicate(t *testing.T) {
	rows := []tableRow{
		{ID: 658, VersionID: 20260723181411, IsApplied: true},
		{ID: 659, VersionID: 20260724134801, IsApplied: true},
		{ID: 660, VersionID: 20260727083533, IsApplied: true},
		{ID: 664, VersionID: 20260723181412, IsApplied: true}, // already remapped, stranded
		{ID: 676, VersionID: 20260810192005, IsApplied: true},
		{ID: 679, VersionID: 20260723181413, IsApplied: true}, // already remapped, stranded at tail
		{ID: 680, VersionID: 20260724134801, IsApplied: true}, // duplicate from a failed re-run
	}
	renames := []migrationRename{
		rename(20260731100711, 20260723181412),
		rename(20260812170926, 20260723181413),
	}

	simulated, messages, issues := simulateTableSQL(tableStatusName, rows, renames)
	require.Empty(t, issues)
	joined := strings.Join(messages, "\n")
	assert.Contains(t, joined, "20260731100711 not present")
	assert.Contains(t, joined, "duplicate version_id=20260724134801; would keep id=659, delete ids=[680]")
	requireVersionOrder(t, simulated, []int64{
		20260723181411,
		20260723181412,
		20260723181413,
		20260724134801,
		20260727083533,
		20260810192005,
	})
}

// Up-move renumber, the shape of RC-cherry-pick timestamp bumps: rows applied
// at old timestamps below migrations that already exist above them.
func TestSimulateUpMove(t *testing.T) {
	rows := []tableRow{
		{ID: 1, VersionID: 100, IsApplied: true},
		{ID: 2, VersionID: 150, IsApplied: true},
		{ID: 3, VersionID: 160, IsApplied: true},
		{ID: 4, VersionID: 200, IsApplied: true},
	}
	renames := []migrationRename{
		rename(150, 250),
		rename(160, 251),
	}

	simulated, _, issues := simulateTableSQL(tableStatusName, rows, renames)
	require.Empty(t, issues)
	requireVersionOrder(t, simulated, []int64{100, 200, 250, 251})
}

// Up-moves and down-moves in the same scope, which the previous
// single-direction shift logic could not represent.
func TestSimulateMixedDirections(t *testing.T) {
	rows := []tableRow{
		{ID: 1, VersionID: 100, IsApplied: true},
		{ID: 2, VersionID: 130, IsApplied: true},
		{ID: 3, VersionID: 200, IsApplied: true},
		{ID: 4, VersionID: 300, IsApplied: true},
	}
	renames := []migrationRename{
		rename(130, 250), // up
		rename(300, 150), // down
	}

	simulated, _, issues := simulateTableSQL(tableStatusName, rows, renames)
	require.Empty(t, issues)
	requireVersionOrder(t, simulated, []int64{100, 150, 200, 250})
}

// Chained renames across two commits (A -> B, then B -> C) must land rows on
// the terminal version regardless of whether the database applied A or B.
// Renames arrive oldest-commit-first from findRenameCommits.
func TestSimulateChainedRenames(t *testing.T) {
	renames := []migrationRename{
		rename(111, 222),
		rename(222, 333),
	}

	for name, startVID := range map[string]int64{"applied at A": 111, "applied at B": 222} {
		t.Run(name, func(t *testing.T) {
			rows := []tableRow{
				{ID: 1, VersionID: 50, IsApplied: true},
				{ID: 2, VersionID: startVID, IsApplied: true},
			}
			simulated, _, issues := simulateTableSQL(tableStatusName, rows, renames)
			require.Empty(t, issues)
			requireVersionOrder(t, simulated, []int64{50, 333})
		})
	}
}

func TestCountOrderingViolations(t *testing.T) {
	assert.Equal(t, 0, countOrderingViolations([]tableRow{
		{ID: 1, VersionID: 10, IsApplied: true},
		{ID: 2, VersionID: 20, IsApplied: true},
	}))
	assert.Equal(t, 2, countOrderingViolations([]tableRow{
		{ID: 1, VersionID: 10, IsApplied: true},
		{ID: 2, VersionID: 30, IsApplied: true},
		{ID: 3, VersionID: 20, IsApplied: true}, // out of order
		{ID: 4, VersionID: 40, IsApplied: true},
		{ID: 5, VersionID: 15, IsApplied: true}, // out of order
	}))
}

func TestBuildSQLStatements(t *testing.T) {
	renames := []migrationRename{
		rename(20260731100711, 20260723181412),
		rename(20260812170926, 20260723181413),
	}
	lines := buildSQL(tableStatusName, renames)

	// remaps first, in commit order
	require.GreaterOrEqual(t, len(lines), 2)
	assert.Equal(t, "UPDATE `migration_status_tables` SET version_id = 20260723181412 WHERE version_id = 20260731100711;", lines[0])
	assert.Equal(t, "UPDATE `migration_status_tables` SET version_id = 20260723181413 WHERE version_id = 20260812170926;", lines[1])

	joined := strings.Join(lines, "\n")
	// dedup block
	assert.Contains(t, joined, "_fix_dups_migration_status_tables")
	// two-pass rebuild: rebase above MAX(id), then compact to 1..N
	assert.Contains(t, joined, "SELECT MAX(id) INTO @rebase_migration_status_tables")
	assert.Contains(t, joined, "ROW_NUMBER() OVER (ORDER BY version_id ASC, id ASC)")
	assert.Contains(t, joined, "SET t.id = @rebase_migration_status_tables + o.rn;")
	assert.Contains(t, joined, "ON t.id = @rebase_migration_status_tables + o.rn SET t.id = o.rn;")
	// the fragile minimal-shift machinery must be gone
	assert.NotContains(t, joined, "COALESCE")
	assert.NotContains(t, joined, "increment_by")
}

func TestBuildSQLNoRenames(t *testing.T) {
	assert.Empty(t, buildSQL(tableStatusName, nil))
}

// End-to-end dry-run verification over the real-world incident shape must
// come back clean.
func TestVerifyDryRunIncidentShape(t *testing.T) {
	tableRows := []tableRow{
		{ID: 658, VersionID: 20260723181411, IsApplied: true},
		{ID: 659, VersionID: 20260724134801, IsApplied: true},
		{ID: 660, VersionID: 20260727083533, IsApplied: true},
		{ID: 664, VersionID: 20260723181412, IsApplied: true},
		{ID: 676, VersionID: 20260810192005, IsApplied: true},
		{ID: 679, VersionID: 20260723181413, IsApplied: true},
		{ID: 680, VersionID: 20260724134801, IsApplied: true},
	}
	renames := []migrationRename{
		rename(20260731100711, 20260723181412),
		rename(20260812170926, 20260723181413),
	}

	clean, messages := verifyDryRun(renames, tableRows, nil)
	assert.True(t, clean, "messages:\n%s", strings.Join(messages, "\n"))
}
