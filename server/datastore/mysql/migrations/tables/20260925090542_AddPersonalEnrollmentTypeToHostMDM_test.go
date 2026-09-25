package tables

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20260925090542(t *testing.T) {
	db := applyUpToPrev(t)

	insertHost := func(uuid, platform string) uint {
		return uint(execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, ?)`, //nolint:gosec // dismiss G115
			"oh-"+uuid, "nk-"+uuid, uuid+".local", uuid, platform))
	}
	insertHostMDM := func(hostID uint, enrolled, fromDEP, personal bool) {
		execNoErr(t, db, `INSERT INTO host_mdm (host_id, enrolled, server_url, installed_from_dep, is_personal_enrollment) VALUES (?, ?, 'https://fleet.local', ?, ?)`,
			hostID, enrolled, fromDEP, personal)
	}
	insertNanoEnrollment := func(id, enrollType string, enabled bool) {
		execNoErr(t, db, `INSERT INTO nano_devices (id, authenticate) VALUES (?, 'auth')`, id)
		execNoErr(t, db, `INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, enabled) VALUES (?, ?, ?, 'topic', 'magic', 'token', ?)`,
			id, id, enrollType, enabled)
	}

	androidPersonal := insertHost("android-personal", "android")
	insertHostMDM(androidPersonal, true, false, true)

	androidCOBO := insertHost("android-cobo", "android")
	insertHostMDM(androidCOBO, true, true, false)

	adue := insertHost("adue", "ios")
	insertHostMDM(adue, true, false, true)
	insertNanoEnrollment("adue", "User Enrollment (Device)", true)

	adueUnenrolled := insertHost("adue-unenrolled", "ios")
	insertHostMDM(adueUnenrolled, false, false, true)
	insertNanoEnrollment("adue-unenrolled", "User Enrollment (Device)", false)

	manualBYOD := insertHost("manual-byod", "ios")
	insertHostMDM(manualBYOD, true, false, true)
	insertNanoEnrollment("manual-byod", "Device", true)

	personalNoNano := insertHost("personal-no-nano", "darwin")
	insertHostMDM(personalNoNano, true, false, true)

	companyManual := insertHost("company-manual", "darwin")
	insertHostMDM(companyManual, true, false, false)
	insertNanoEnrollment("company-manual", "Device", true)

	ade := insertHost("ade", "darwin")
	insertHostMDM(ade, true, true, false)

	pending := insertHost("pending", "darwin")
	insertHostMDM(pending, false, true, false)

	applyNext(t, db)

	cases := []struct {
		name     string
		hostID   uint
		wantType sql.NullString
		wantStat sql.NullString
	}{
		{"android work profile", androidPersonal, validString("work_profile"), validString("On (personal)")},
		{"android company-owned", androidCOBO, sql.NullString{}, validString("On (automatic)")},
		{"account-driven user enrollment", adue, validString("account_driven"), validString("On (personal)")},
		{"unenrolled account-driven", adueUnenrolled, validString("account_driven"), validString("Off")},
		{"manual BYOD", manualBYOD, validString("manual_profile"), validString("On (manual - personal)")},
		{"personal without nano enrollment", personalNoNano, validString("manual_profile"), validString("On (manual - personal)")},
		{"company-owned manual", companyManual, sql.NullString{}, validString("On (manual)")},
		{"ADE", ade, sql.NullString{}, validString("On (automatic)")},
		{"pending", pending, sql.NullString{}, validString("Pending")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var row struct {
				Type   sql.NullString `db:"personal_enrollment_type"`
				Status sql.NullString `db:"enrollment_status"`
			}
			require.NoError(t, db.Get(&row, `SELECT personal_enrollment_type, enrollment_status FROM host_mdm WHERE host_id = ?`, c.hostID))
			require.Equal(t, c.wantType, row.Type)
			require.Equal(t, c.wantStat, row.Status)
		})
	}

	t.Run("backfill re-run is a no-op", func(t *testing.T) {
		execNoErr(t, db, `UPDATE host_mdm SET personal_enrollment_type = 'work_profile' WHERE host_id = ?`, manualBYOD)
		updated := runPersonalEnrollmentTypeBackfill(t, db)
		require.Zero(t, updated)

		var typ string
		require.NoError(t, db.Get(&typ, `SELECT personal_enrollment_type FROM host_mdm WHERE host_id = ?`, manualBYOD))
		require.Equal(t, "work_profile", typ)
	})

	t.Run("server host has no status", func(t *testing.T) {
		execNoErr(t, db, `UPDATE host_mdm SET is_server = 1 WHERE host_id = ?`, adue)
		var status sql.NullString
		require.NoError(t, db.Get(&status, `SELECT enrollment_status FROM host_mdm WHERE host_id = ?`, adue))
		require.False(t, status.Valid)
	})

	t.Run("backfill runs in batches", func(t *testing.T) {
		for i := range 12 {
			hostID := insertHost(fmt.Sprintf("android-batch-%d", i), "android")
			insertHostMDM(hostID, true, false, true)
		}

		defer func(prev int) { personalEnrollmentTypeBackfillBatchSize = prev }(personalEnrollmentTypeBackfillBatchSize)
		personalEnrollmentTypeBackfillBatchSize = 5

		require.Equal(t, 12, runPersonalEnrollmentTypeBackfill(t, db))

		var remaining int
		require.NoError(t, db.Get(&remaining, `SELECT COUNT(*) FROM host_mdm WHERE is_personal_enrollment = 1 AND personal_enrollment_type IS NULL`))
		require.Zero(t, remaining)
	})

	require.Equal(t, []string{"enrolled", "installed_from_dep", "is_personal_enrollment", "personal_enrollment_type"},
		indexColumns(t, db, "host_mdm", "host_mdm_enrolled_dep_personal_type_idx"))
	require.Empty(t, indexColumns(t, db, "host_mdm", "host_mdm_enrolled_installed_from_dep_is_personal_enrollment_idx"))
}

func runPersonalEnrollmentTypeBackfill(t *testing.T, db *sqlx.DB) int {
	tx, err := db.Begin()
	require.NoError(t, err)
	var updated int
	require.NoError(t, backfillPersonalEnrollmentType(tx, func() { updated++ }))
	require.NoError(t, tx.Commit())
	return updated
}

func validString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: true}
}
