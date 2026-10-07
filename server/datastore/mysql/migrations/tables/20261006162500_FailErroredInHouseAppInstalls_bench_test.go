package tables

import (
	"database/sql"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const benchFailErroredHosts = 50000

// Run with FLEET_BENCH_FAIL_ERRORED_IN_HOUSE=1, seeds 50k hosts, 200k in-house installs and 1M nano command results
func TestUp_20261006162500_BenchJoinOrder(t *testing.T) {
	if os.Getenv("FLEET_BENCH_FAIL_ERRORED_IN_HOUSE") == "" {
		t.Skip("set FLEET_BENCH_FAIL_ERRORED_IN_HOUSE=1 to run")
	}
	db := applyUpToPrev(t)
	const batchSize = 1000

	seedStart := time.Now()
	execNoErr(t, db, `SET SESSION cte_max_recursion_depth = 1000000`)
	execNoErr(t, db, `
INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform)
WITH RECURSIVE seq (n) AS (SELECT 1 UNION ALL SELECT n + 1 FROM seq WHERE n < ?)
SELECT CONCAT('oh-', n), CONCAT('nk-', n), CONCAT('h', n, '.local'), CONCAT('UUID-', n), 'ios' FROM seq`, benchFailErroredHosts)
	execNoErr(t, db, `INSERT INTO nano_devices (id, authenticate) SELECT uuid, 'auth' FROM hosts`)
	execNoErr(t, db, `
INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, enabled)
SELECT uuid, uuid, 'Device', 'topic', 'magic', 'abcdef', 1 FROM hosts`)
	appID := execNoErrLastID(t, db, `INSERT INTO in_house_apps (global_or_team_id, storage_id, platform, filename) VALUES (0, 'storage-1', 'ios', 'app.ipa')`)

	// 4 installs per host, every 20th install errored and pending, every 20th + 1 acknowledged and pending, the rest verified
	execNoErr(t, db, `
INSERT INTO host_in_house_software_installs (host_id, in_house_app_id, platform, command_uuid)
SELECT h.id, ?, 'ios', CONCAT('install-', h.id, '-', k.k)
FROM hosts h CROSS JOIN (SELECT 1 AS k UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4) k`, appID)
	execNoErr(t, db, `UPDATE host_in_house_software_installs SET verification_at = NOW(6) WHERE id % 20 NOT IN (0, 1)`)
	execNoErr(t, db, `INSERT INTO nano_commands (command_uuid, request_type, command, name) SELECT command_uuid, 'InstallApplication', 'x', '' FROM host_in_house_software_installs`)
	execNoErr(t, db, `
INSERT INTO nano_command_results (id, command_uuid, status, result)
SELECT h.uuid, hihsi.command_uuid, IF(hihsi.id % 20 = 0, 'Error', 'Acknowledged'), '<?xml'
FROM host_in_house_software_installs hihsi JOIN hosts h ON h.id = hihsi.host_id`)

	// 16 other commands per host so nano_command_results is large, every 5th one errored
	execNoErr(t, db, `
INSERT INTO nano_commands (command_uuid, request_type, command, name)
WITH RECURSIVE seq (k) AS (SELECT 1 UNION ALL SELECT k + 1 FROM seq WHERE k < 16)
SELECT CONCAT('other-', h.id, '-', seq.k), 'DeviceInformation', 'x', '' FROM hosts h CROSS JOIN seq`)
	execNoErr(t, db, `
INSERT INTO nano_command_results (id, command_uuid, status, result)
WITH RECURSIVE seq (k) AS (SELECT 1 UNION ALL SELECT k + 1 FROM seq WHERE k < 16)
SELECT h.uuid, CONCAT('other-', h.id, '-', seq.k), IF(seq.k % 5 = 0, 'Error', 'Acknowledged'), '<?xml' FROM hosts h CROSS JOIN seq`)
	execNoErr(t, db, `ANALYZE TABLE hosts, host_in_house_software_installs, nano_command_results, nano_commands`)

	var installCount, resultCount, expected int
	require.NoError(t, db.Get(&installCount, `SELECT COUNT(*) FROM host_in_house_software_installs`))
	require.NoError(t, db.Get(&resultCount, `SELECT COUNT(*) FROM nano_command_results`))
	require.NoError(t, db.Get(&expected, `SELECT COUNT(*) FROM host_in_house_software_installs WHERE id % 20 = 0`))
	t.Logf("seeded %d installs, %d command results, %d errored pending installs in %s", installCount, resultCount, expected, time.Since(seedStart))

	const pending = `hihsi.verification_at IS NULL AND hihsi.verification_failed_at IS NULL AND hihsi.canceled = 0 AND ncr.status IN ('Error', 'CommandFormatError')`

	variants := []struct {
		name    string
		batched bool
		stmt    string
	}{
		{
			name:    "batched hihsi -> hosts -> ncr (current)",
			batched: true,
			stmt: `UPDATE host_in_house_software_installs hihsi
JOIN hosts h ON h.id = hihsi.host_id
JOIN nano_command_results ncr ON ncr.id = h.uuid AND ncr.command_uuid = hihsi.command_uuid
SET hihsi.verification_failed_at = ncr.updated_at
WHERE ` + pending + ` AND hihsi.id > ? AND hihsi.id <= ?`,
		},
		{
			name:    "batched hihsi -> ncr on command_uuid, no hosts join",
			batched: true,
			stmt: `UPDATE host_in_house_software_installs hihsi
JOIN nano_command_results ncr ON ncr.command_uuid = hihsi.command_uuid
SET hihsi.verification_failed_at = ncr.updated_at
WHERE ` + pending + ` AND hihsi.id > ? AND hihsi.id <= ?`,
		},
		{
			name: "unbatched hihsi -> hosts -> ncr",
			stmt: `UPDATE host_in_house_software_installs hihsi
JOIN hosts h ON h.id = hihsi.host_id
JOIN nano_command_results ncr ON ncr.id = h.uuid AND ncr.command_uuid = hihsi.command_uuid
SET hihsi.verification_failed_at = ncr.updated_at
WHERE ` + pending,
		},
		{
			name: "unbatched ncr by status -> hihsi",
			stmt: `UPDATE nano_command_results ncr
STRAIGHT_JOIN host_in_house_software_installs hihsi ON hihsi.command_uuid = ncr.command_uuid
SET hihsi.verification_failed_at = ncr.updated_at
WHERE ` + pending,
		},
	}

	var maxInstallID int64
	require.NoError(t, db.Get(&maxInstallID, `SELECT MAX(id) FROM host_in_house_software_installs`))

	for _, variant := range variants {
		explainArgs := []any{}
		if variant.batched {
			explainArgs = []any{0, batchSize}
		}
		explainRows, err := db.Queryx(`EXPLAIN `+variant.stmt, explainArgs...)
		require.NoError(t, err)
		for explainRows.Next() {
			row := map[string]any{}
			require.NoError(t, explainRows.MapScan(row))
			t.Logf("explain %s: table=%s type=%s key=%s rows=%v", variant.name, row["table"], row["type"], row["key"], row["rows"])
		}
		require.NoError(t, explainRows.Close())
	}

	// run every variant once per round, starting each round at a different variant so none always runs first
	const rounds = 5
	durations := make([][]time.Duration, len(variants))
	for round := range rounds {
		for offset := range variants {
			variantIndex := (round + offset) % len(variants)
			variant := variants[variantIndex]

			tx, err := db.Begin()
			require.NoError(t, err)
			start := time.Now()
			var affected int64
			if variant.batched {
				for startID := int64(0); startID < maxInstallID; startID += batchSize {
					var res sql.Result
					res, err = tx.Exec(variant.stmt, startID, startID+batchSize)
					require.NoError(t, err)
					rowsAffected, _ := res.RowsAffected()
					affected += rowsAffected
				}
			} else {
				var res sql.Result
				res, err = tx.Exec(variant.stmt)
				require.NoError(t, err)
				affected, _ = res.RowsAffected()
			}
			durations[variantIndex] = append(durations[variantIndex], time.Since(start))
			require.NoError(t, tx.Rollback())
			require.EqualValues(t, expected, affected)
		}
	}

	for variantIndex, variant := range variants {
		sorted := slices.Clone(durations[variantIndex])
		slices.Sort(sorted)
		t.Logf("%s: median %s, min %s, max %s, runs %v", variant.name, sorted[len(sorted)/2], sorted[0], sorted[len(sorted)-1], durations[variantIndex])
	}
}
