package mysql

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/stretchr/testify/require"
)

func TestStoreCommandReportNotForEnrollment(t *testing.T) {
	testDSN := os.Getenv("NANOMDM_MYSQL_STORAGE_TEST_DSN")
	if testDSN == "" {
		t.Skip("NANOMDM_MYSQL_STORAGE_TEST_DSN not set")
	}

	s, err := New(WithDSN(testDSN))
	require.NoError(t, err)

	ctx := t.Context()
	queuedID := "queue-test-queued"
	otherID := "queue-test-other"
	cmdUUID := "queue-test-command"

	for _, id := range []string{queuedID, otherID} {
		_, err := s.db.ExecContext(
			ctx,
			`INSERT INTO nano_devices (id, authenticate, authenticate_at) VALUES (?, '<?xml>', CURRENT_TIMESTAMP)`,
			id,
		)
		require.NoError(t, err)
		_, err = s.db.ExecContext(
			ctx,
			`INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex) VALUES (?, ?, 'Device', 'topic', 'magic', 'token')`,
			id, id,
		)
		require.NoError(t, err)
	}
	// t.Context is already canceled when Cleanup runs, so cleanup uses a background context.
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = s.db.ExecContext(bg, `DELETE FROM nano_commands WHERE command_uuid = ?`, cmdUUID)
		_, _ = s.db.ExecContext(bg, `DELETE FROM nano_devices WHERE id IN (?, ?)`, queuedID, otherID)
	})

	cmd := &mdm.CommandWithSubtype{
		Command: mdm.Command{
			CommandUUID: cmdUUID,
			Command:     struct{ RequestType string }{RequestType: "InstallProfile"},
			Raw:         []byte("<?xml>"),
		},
		Subtype: mdm.CommandSubtypeNone,
	}
	_, err = s.EnqueueCommand(ctx, []string{queuedID}, cmd)
	require.NoError(t, err)

	countResults := func(t *testing.T, id, uuid string) int {
		t.Helper()
		var count int
		err := s.db.QueryRowContext(
			ctx,
			`SELECT COUNT(*) FROM nano_command_results WHERE id = ? AND command_uuid = ?`,
			id, uuid,
		).Scan(&count)
		require.NoError(t, err)
		return count
	}

	t.Run("command matches, enrollment does not", func(t *testing.T) {
		r := &mdm.Request{EnrollID: &mdm.EnrollID{Type: mdm.Device, ID: otherID}, Context: ctx}
		result := &mdm.CommandResults{CommandUUID: cmdUUID, Status: "Acknowledged", Raw: []byte("<?xml>")}
		require.ErrorIs(t, s.StoreCommandReport(r, result), sql.ErrNoRows)
		require.Zero(t, countResults(t, otherID, cmdUUID))
	})

	t.Run("no command matches", func(t *testing.T) {
		missingUUID := "queue-test-missing"
		r := &mdm.Request{EnrollID: &mdm.EnrollID{Type: mdm.Device, ID: otherID}, Context: ctx}
		result := &mdm.CommandResults{CommandUUID: missingUUID, Status: "Acknowledged", Raw: []byte("<?xml>")}
		require.ErrorIs(t, s.StoreCommandReport(r, result), sql.ErrNoRows)
		require.Zero(t, countResults(t, otherID, missingUUID))
	})

	t.Run("inactive queue entry is not reported", func(t *testing.T) {
		_, err := s.db.ExecContext(
			ctx,
			`UPDATE nano_enrollment_queue SET active = 0 WHERE id = ? AND command_uuid = ?`,
			queuedID, cmdUUID,
		)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = s.db.ExecContext(
				context.Background(),
				`UPDATE nano_enrollment_queue SET active = 1 WHERE id = ? AND command_uuid = ?`,
				queuedID, cmdUUID,
			)
		})

		r := &mdm.Request{EnrollID: &mdm.EnrollID{Type: mdm.Device, ID: queuedID}, Context: ctx}
		result := &mdm.CommandResults{CommandUUID: cmdUUID, Status: "Acknowledged", Raw: []byte("<?xml>")}
		require.ErrorIs(t, s.StoreCommandReport(r, result), sql.ErrNoRows)
		require.Zero(t, countResults(t, queuedID, cmdUUID))
	})

	t.Run("identical re-report is accepted", func(t *testing.T) {
		r := &mdm.Request{EnrollID: &mdm.EnrollID{Type: mdm.Device, ID: queuedID}, Context: ctx}
		result := &mdm.CommandResults{CommandUUID: cmdUUID, Status: "Acknowledged", Raw: []byte("<?xml>")}
		require.NoError(t, s.StoreCommandReport(r, result))
		require.NoError(t, s.StoreCommandReport(r, result))
		require.Equal(t, 1, countResults(t, queuedID, cmdUUID))
	})
}
