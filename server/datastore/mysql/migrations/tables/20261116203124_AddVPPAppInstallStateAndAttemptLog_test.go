package tables

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261116203124(t *testing.T) {
	db := applyUpToPrev(t)
	// Pin the session so both migration runs use MySQL 8.0's default strict mode.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	var originalSQLMode string
	require.NoError(t, db.Get(&originalSQLMode, `SELECT @@SESSION.sql_mode`))
	execNoErr(t, db, `SET SESSION sql_mode = 'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION'`)
	t.Cleanup(func() { execNoErr(t, db, `SET SESSION sql_mode = ?`, originalSQLMode) })

	insert := func(stmt string, args ...any) uint {
		return uint(execNoErrLastID(t, db, stmt, args...)) //nolint:gosec // dismiss G115
	}
	insert(`INSERT INTO vpp_apps (adam_id, platform) VALUES ('app-a', 'ios'), ('app-b', 'android')`)
	version := func(adamID, platform, name string) uint {
		return insert(`INSERT INTO vpp_apps_teams (adam_id, platform, global_or_team_id, name) VALUES (?, ?, 0, ?)`, adamID, platform, name)
	}
	vA1, vA2, vA3 := version("app-a", "ios", "a1"), version("app-a", "ios", "a2"), version("app-a", "ios", "a3")
	vB1 := version("app-b", "android", "b1")

	const h1, h2, h3, h4, h5 = uint(1), uint(2), uint(3), uint(4), uint(5)
	const verified, failed, canceled = `verification_at = NOW(6)`, `verification_failed_at = NOW(6)`, `canceled = 1`
	var n int
	request := func(hostID uint, adamID, platform string, versionID *uint, createdAt *string, assignments string) uint {
		n++
		id := insert(`INSERT INTO host_vpp_software_installs (host_id, adam_id, platform, command_uuid, vpp_app_team_id, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`, hostID, adamID, platform, fmt.Sprint("cmd-", n), versionID, createdAt)
		if assignments != "" {
			execNoErr(t, db, `UPDATE host_vpp_software_installs SET `+assignments+` WHERE id = ?`, id)
		}
		return id
	}
	at := func(hour string) *string { return new("2026-01-01 " + hour + ":00:00") }

	// vA2 rows get lower ids but later created_at than vA1 rows.
	b1 := request(h1, "app-a", "ios", &vA2, at("13"), failed)
	b2 := request(h1, "app-a", "ios", &vA2, at("12"), "")
	a1 := request(h1, "app-a", "ios", &vA1, at("09"), failed)
	a2 := request(h1, "app-a", "ios", &vA1, at("10"), verified) // verified after a failed attempt
	a3 := request(h1, "app-a", "ios", &vA1, at("08"), "")
	request(h1, "app-a", "ios", &vA1, at("11"), canceled)
	a5 := request(h1, "app-a", "ios", &vA1, at("12"), `removed = 1, `+verified)
	c1 := request(h1, "app-a", "ios", nil, nil, "")
	c2 := request(h1, "app-a", "ios", &vA3, at("06"), verified)
	execNoErr(t, db, `DELETE FROM vpp_apps_teams WHERE id = ?`, vA3) // deleted version nulls c2's vpp_app_team_id
	p1 := request(h2, "app-a", "ios", &vA1, at("10"), failed)        // preflight failure
	request(h2, "app-a", "ios", &vA2, at("09"), canceled)
	e1 := request(h2, "app-b", "android", &vB1, at("10"), failed)
	e2 := request(h2, "app-b", "android", &vB1, at("11"), "")

	r1 := request(h3, "app-a", "ios", &vA1, at("09"), `removed = 1, `+verified)
	m0 := request(h4, "app-a", "ios", &vA1, new("2026-03-01 00:00:10"), failed)
	m1 := request(h4, "app-a", "ios", &vA1, new("2026-04-01 00:00:00"), verified)
	g1 := request(h5, "app-a", "ios", &vA1, at("09"), verified)
	request(h5, "app-a", "ios", &vA1, at("10"), canceled)

	applyNext(t, db)
	// Retry after the first backfill committed, including replacement of stale pointers.
	execNoErr(t, db, `UPDATE host_vpp_app_installs SET installed_install_id = NULL, failed_install_id = NULL,
  unfinished_install_id = NULL, visible_install_id = NULL, latest_install_id = NULL, unfailed_install_id = NULL,
  vpp_app_team_id = NULL,
  visible_created_at = NULL, visible_command_uuid = NULL,
  visible_verification_at = NULL, visible_verification_failed_at = NULL,
  latest_created_at = NULL, latest_command_uuid = NULL,
  latest_verification_at = NULL, latest_verification_failed_at = NULL`)
	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, Up_20261116203124(tx))
	require.NoError(t, tx.Commit())

	type state struct {
		HostID     uint   `db:"host_id"`
		AdamID     string `db:"adam_id"`
		Platform   string `db:"platform"`
		Bucket     uint   `db:"version_bucket"`
		VersionID  *uint  `db:"vpp_app_team_id"`
		Installed  *uint  `db:"installed_install_id"`
		Failed     *uint  `db:"failed_install_id"`
		Unfinished *uint  `db:"unfinished_install_id"`
		Visible    *uint  `db:"visible_install_id"`
		Latest     *uint  `db:"latest_install_id"`
		Unfailed   *uint  `db:"unfailed_install_id"`
	}
	var got []state
	require.NoError(t, db.Select(&got, `
SELECT host_id, adam_id, platform, version_bucket, vpp_app_team_id,
	installed_install_id, failed_install_id, unfinished_install_id,
	visible_install_id, latest_install_id, unfailed_install_id
FROM host_vpp_app_installs ORDER BY host_id, adam_id, platform, version_bucket`))

	// c1's NULL created_at sorts last in a DESC order, so c2 wins the ordered pointers of bucket 0.
	require.Equal(t, []state{
		{h1, "app-a", "ios", 0, nil, &c2, nil, &c1, &c2, &c2, &c2},
		{h1, "app-a", "ios", vA1, &vA1, &a2, &a1, &a3, &a2, &a5, &a5},
		{h1, "app-a", "ios", vA2, &vA2, nil, &b1, &b2, &b1, &b1, &b2},
		{h2, "app-a", "ios", vA1, &vA1, nil, &p1, nil, &p1, &p1, nil},
		{h2, "app-a", "ios", vA2, &vA2, nil, nil, nil, nil, nil, nil},
		{h2, "app-b", "android", vB1, &vB1, nil, &e1, &e2, &e2, &e2, &e2},
		{h3, "app-a", "ios", vA1, &vA1, nil, nil, nil, nil, &r1, &r1},
		{h4, "app-a", "ios", vA1, &vA1, &m1, &m0, nil, &m1, &m1, &m1},
		{h5, "app-a", "ios", vA1, &vA1, &g1, nil, nil, &g1, &g1, &g1},
	}, got)

	var nullSnapshots int
	require.NoError(t, db.Get(&nullSnapshots, `SELECT COUNT(*) FROM host_vpp_app_installs
WHERE ((host_id = ? AND version_bucket = ?) OR host_id = ?)
  AND visible_install_id IS NULL AND visible_created_at IS NULL AND visible_command_uuid IS NULL
  AND visible_verification_at IS NULL AND visible_verification_failed_at IS NULL`, h2, vA2, h3))
	require.Equal(t, 2, nullSnapshots)
	require.NoError(t, db.Get(&nullSnapshots, `SELECT COUNT(*) FROM host_vpp_app_installs
WHERE host_id = ? AND version_bucket = ? AND latest_install_id IS NULL
  AND latest_created_at IS NULL AND latest_command_uuid IS NULL
  AND latest_verification_at IS NULL AND latest_verification_failed_at IS NULL`, h2, vA2))
	require.Equal(t, 1, nullSnapshots)

	var snapshotMismatches int
	require.NoError(t, db.Get(&snapshotMismatches, `
SELECT COUNT(*)
FROM host_vpp_app_installs state
LEFT JOIN host_vpp_software_installs visible ON visible.id = state.visible_install_id
LEFT JOIN host_vpp_software_installs latest ON latest.id = state.latest_install_id
WHERE NOT (
	state.visible_created_at <=> visible.created_at AND
	state.visible_command_uuid <=> visible.command_uuid AND
	state.visible_verification_at <=> visible.verification_at AND
	state.visible_verification_failed_at <=> visible.verification_failed_at AND
	state.latest_created_at <=> latest.created_at AND
	state.latest_command_uuid <=> latest.command_uuid AND
	state.latest_verification_at <=> latest.verification_at AND
	state.latest_verification_failed_at <=> latest.verification_failed_at
)`))
	require.Zero(t, snapshotMismatches)

	var logRows int
	require.NoError(t, db.Get(&logRows, `SELECT COUNT(*) FROM log_vpp_app_install_attempts`))
	require.Zero(t, logRows)
}
