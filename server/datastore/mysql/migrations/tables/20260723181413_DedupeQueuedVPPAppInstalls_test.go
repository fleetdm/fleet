package tables

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260723181413(t *testing.T) {
	db := applyUpToPrev(t)

	// vpp_app_upcoming_activities has an FK on (adam_id, platform).
	execNoErr(t, db, `
		INSERT INTO vpp_apps (adam_id, platform, name, latest_version)
		VALUES ('app_x', 'ios', 'App X', '1.0.0'), ('app_y', 'ios', 'App Y', '1.0.0')`)

	policyID := execNoErrLastID(t, db, `
		INSERT INTO policies (name, query, description, checksum)
		VALUES ('Install App X', 'SELECT 1', '', UNHEX(MD5('policy-app-x')))`)

	var execCount int
	// queueInstallWithPayload stores the payload verbatim, so a test can hold a real JSON boolean
	// rather than the JSON number a Go bool produces.
	queueInstallWithPayload := func(hostID uint, adamID string, activated bool, payload string) (int64, string) {
		execCount++
		execID := fmt.Sprintf("exec-%d", execCount)
		activatedAt := "NULL"
		if activated {
			activatedAt = "NOW(6)"
		}
		id := execNoErrLastID(t, db, fmt.Sprintf(`
			INSERT INTO upcoming_activities (host_id, activity_type, execution_id, payload, activated_at)
			VALUES (?, 'vpp_app_install', ?, ?, %s)`, activatedAt),
			hostID, execID, payload)
		execNoErr(t, db, `
			INSERT INTO vpp_app_upcoming_activities (upcoming_activity_id, adam_id, platform)
			VALUES (?, ?, 'ios')`, id, adamID)
		return id, execID
	}
	// queueInstall writes the payload the way the datastore does, via JSON_OBJECT with a Go bool,
	// which lands as the JSON number 1 or 0 rather than a JSON boolean.
	queueInstall := func(hostID uint, adamID string, activated, fromAutoUpdate bool) (int64, string) {
		execCount++
		execID := fmt.Sprintf("exec-%d", execCount)
		activatedAt := "NULL"
		if activated {
			activatedAt = "NOW(6)"
		}
		id := execNoErrLastID(t, db, fmt.Sprintf(`
			INSERT INTO upcoming_activities (host_id, activity_type, execution_id, payload, activated_at)
			VALUES (?, 'vpp_app_install', ?, JSON_OBJECT('from_auto_update', ?), %s)`, activatedAt),
			hostID, execID, fromAutoUpdate)
		execNoErr(t, db, `
			INSERT INTO vpp_app_upcoming_activities (upcoming_activity_id, adam_id, platform)
			VALUES (?, ?, 'ios')`, id, adamID)
		return id, execID
	}
	// queueInstallForPolicy is what a policy automation writes: from_auto_update false, policy_id set.
	queueInstallForPolicy := func(hostID uint, adamID string) int64 {
		execCount++
		id := execNoErrLastID(t, db, `
			INSERT INTO upcoming_activities (host_id, activity_type, execution_id, payload)
			VALUES (?, 'vpp_app_install', ?, JSON_OBJECT('from_auto_update', false))`,
			hostID, fmt.Sprintf("exec-%d", execCount))
		execNoErr(t, db, `
			INSERT INTO vpp_app_upcoming_activities (upcoming_activity_id, adam_id, platform, policy_id)
			VALUES (?, ?, 'ios', ?)`, id, adamID, policyID)
		return id
	}
	queuedIDs := func(hostID uint, adamID string) []int64 {
		var ids []int64
		require.NoError(t, db.Select(&ids, `
			SELECT ua.id
			FROM upcoming_activities ua
			JOIN vpp_app_upcoming_activities vaua ON vaua.upcoming_activity_id = ua.id
			WHERE ua.host_id = ? AND vaua.adam_id = ?
			ORDER BY ua.id`, hostID, adamID))
		return ids
	}
	allActivityIDs := func() []int64 {
		var ids []int64
		require.NoError(t, db.Select(&ids, `SELECT id FROM upcoming_activities ORDER BY id`))
		return ids
	}

	// Host 1: the reported state. One install sent to the device, five more queued behind it. The
	// sent one occupies the slot, so every queued duplicate goes.
	hostOneActivated, _ := queueInstall(1, "app_x", true, true)
	for range 5 {
		queueInstall(1, "app_x", false, true)
	}
	// A single queued install for a different app on the same host is not a duplicate.
	hostOneAppY, _ := queueInstall(1, "app_y", false, true)

	// Host 2: duplicates with nothing activated. Grouping is per host, so this host keeps its own.
	hostTwoQueued := make([]int64, 0, 3)
	for range 3 {
		id, _ := queueInstall(2, "app_x", false, true)
		hostTwoQueued = append(hostTwoQueued, id)
	}

	// Host 3: two activated installs, which batch activation makes possible. Neither may be deleted,
	// because both have already been sent to the device.
	hostThreeActivatedA, _ := queueInstall(3, "app_x", true, true)
	hostThreeActivatedB, _ := queueInstall(3, "app_x", true, true)
	for range 2 {
		queueInstall(3, "app_x", false, true)
	}

	// Host 4: duplicates that did not come from an automatic update. A user or a policy asked for
	// each of these, so they are left alone.
	hostFourQueued := make([]int64, 0, 5)
	for range 5 {
		id, _ := queueInstall(4, "app_x", false, false)
		hostFourQueued = append(hostFourQueued, id)
	}

	// Host 5: a duplicate referenced by a setup experience step survives even though it is neither
	// the newest nor the earliest. Deleting it would leave that step running forever. Production
	// cannot reach this state, since a setup experience install never sets from_auto_update, but the
	// guard has to hold if that filter ever widens.
	queueInstall(5, "app_x", false, true)
	hostFiveSetupExp, setupExpExecID := queueInstall(5, "app_x", false, true)
	hostFiveNewest, _ := queueInstall(5, "app_x", false, true)
	execNoErr(t, db, `
		INSERT INTO setup_experience_status_results (host_uuid, name, status, nano_command_uuid)
		VALUES ('host-5-uuid', 'App X', 'running', ?)`, setupExpExecID)

	// Host 6: enough duplicates to run the batched delete more than once.
	const volumeCount = 2500
	hostSixIDs := make([]int64, 0, volumeCount)
	for range volumeCount {
		id, _ := queueInstall(6, "app_x", false, true)
		hostSixIDs = append(hostSixIDs, id)
	}

	// Host 7: an automatic update queued alongside an install someone asked for. The manual one does
	// not claim the slot, so the host keeps one of each.
	hostSevenManual, _ := queueInstall(7, "app_x", false, false)
	queueInstall(7, "app_x", false, true)
	hostSevenNewestAuto, _ := queueInstall(7, "app_x", false, true)

	// Host 8: payloads holding a real JSON boolean false. `IS TRUE` matches those, so it would have
	// deleted one of these; `= 1` does not match them at all.
	hostEightFalse := make([]int64, 0, 2)
	for range 2 {
		id, _ := queueInstallWithPayload(8, "app_x", false, `{"from_auto_update": false}`)
		hostEightFalse = append(hostEightFalse, id)
	}

	// Host 9: payloads holding a real JSON boolean true. `= 1` does not match a JSON boolean either,
	// so these are skipped rather than deleted, which is the safe direction for an irreversible delete.
	hostNineTrue := make([]int64, 0, 2)
	for range 2 {
		id, _ := queueInstallWithPayload(9, "app_x", false, `{"from_auto_update": true}`)
		hostNineTrue = append(hostNineTrue, id)
	}

	// Host 10: an install a policy asked for, already sent to the device, with automatic updates
	// queued behind it. The sent one occupies the slot whoever asked for it, so every queued
	// duplicate goes rather than one surviving to install the app a second time.
	hostTenPolicyActivated, _ := queueInstall(10, "app_x", true, false)
	for range 2 {
		queueInstall(10, "app_x", false, true)
	}

	// Host 11: a policy automation's backlog. Its filters were blind to a delivered-but-unverified
	// install too, so it appends one row per interval exactly as automatic updates do, and the same
	// rule applies. The install button and self-service are what stay untouched.
	queueInstallForPolicy(11, "app_x")
	queueInstallForPolicy(11, "app_x")
	hostElevenNewest := queueInstallForPolicy(11, "app_x")

	// Other activity types on an affected host are untouched.
	scriptID := execNoErrLastID(t, db, `
		INSERT INTO upcoming_activities (host_id, activity_type, execution_id, payload)
		VALUES (1, 'script', 'script-exec-1', '{}')`)
	softwareInstallID := execNoErrLastID(t, db, `
		INSERT INTO upcoming_activities (host_id, activity_type, execution_id, payload)
		VALUES (1, 'software_install', 'si-exec-1', '{}')`)

	applyNext(t, db)

	// Host 1: only the install already sent survives. Anything left queued behind it would install the
	// same app a second time.
	require.Equal(t, []int64{hostOneActivated}, queuedIDs(1, "app_x"))
	require.Equal(t, []int64{hostOneAppY}, queuedIDs(1, "app_y"))

	// The join table rows went with their parents through the FK's ON DELETE CASCADE.
	var orphanedChildren int
	require.NoError(t, db.Get(&orphanedChildren, `
		SELECT COUNT(*)
		FROM vpp_app_upcoming_activities vaua
		LEFT JOIN upcoming_activities ua ON ua.id = vaua.upcoming_activity_id
		WHERE ua.id IS NULL`))
	require.Zero(t, orphanedChildren)

	// Host 2 has nothing sent, so the newest queued install survives. It is kept rather than the
	// earliest because the nano queue copies its created_at and the APNs retry cron ignores commands
	// over a week old.
	require.Equal(t, []int64{hostTwoQueued[2]}, queuedIDs(2, "app_x"))
	require.Equal(t, []int64{hostThreeActivatedA, hostThreeActivatedB}, queuedIDs(3, "app_x"))
	require.Equal(t, hostFourQueued, queuedIDs(4, "app_x"))
	require.Equal(t, []int64{hostFiveSetupExp, hostFiveNewest}, queuedIDs(5, "app_x"))
	require.Equal(t, []int64{hostSixIDs[volumeCount-1]}, queuedIDs(6, "app_x"))
	require.Equal(t, []int64{hostSevenManual, hostSevenNewestAuto}, queuedIDs(7, "app_x"))
	require.Equal(t, hostEightFalse, queuedIDs(8, "app_x"),
		"a payload holding JSON false must not be read as an automatic update")
	require.Equal(t, hostNineTrue, queuedIDs(9, "app_x"),
		"a payload holding JSON true is skipped rather than deleted")
	require.Equal(t, []int64{hostTenPolicyActivated}, queuedIDs(10, "app_x"),
		"an install already sent occupies the slot whatever its origin")
	require.Equal(t, []int64{hostElevenNewest}, queuedIDs(11, "app_x"),
		"a policy automation's backlog is deduplicated like an automatic update's")

	var setupExpStatus string
	require.NoError(t, db.Get(&setupExpStatus,
		`SELECT status FROM setup_experience_status_results WHERE nano_command_uuid = ?`, setupExpExecID))
	require.Equal(t, "running", setupExpStatus)

	var otherActivityIDs []int64
	require.NoError(t, db.Select(&otherActivityIDs,
		`SELECT id FROM upcoming_activities WHERE activity_type IN ('script', 'software_install') ORDER BY id`))
	require.Equal(t, []int64{scriptID, softwareInstallID}, otherActivityIDs)

	// Running it again deletes nothing anywhere, including the rows kept by an exception.
	before := allActivityIDs()
	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, Up_20260723181413(tx))
	require.NoError(t, tx.Commit())
	require.Equal(t, before, allActivityIDs())
}
