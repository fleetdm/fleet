package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260908184939(t *testing.T) {
	db := applyUpToPrev(t)

	execNoErr(t, db, `
		INSERT INTO nano_commands (command_uuid, request_type, command, name, created_at) VALUES
			('old-cmd', 'DeviceInformation', '<?xml version="1.0"?><plist/>', '', NOW(6) - INTERVAL 2 DAY),
			('new-cmd', 'DeviceInformation', '<?xml version="1.0"?><plist/>', '', NOW(6))`)

	applyNext(t, db)

	require.Equal(t, []string{"created_at"}, indexColumns(t, db, "nano_commands", "idx_nano_commands_created_at"))

	// Seeded rows survive the ALTER and are readable through the orphan mop's walk.
	var uuids []string
	require.NoError(t, db.Select(&uuids, `
		SELECT command_uuid FROM nano_commands
		WHERE created_at < NOW(6) - INTERVAL 1 DAY
		ORDER BY created_at`))
	require.Equal(t, []string{"old-cmd"}, uuids)
}

func TestUp_20260908184939_AlreadyApplied(t *testing.T) {
	db := applyUpToPrev(t)
	execNoErr(t, db, `ALTER TABLE nano_commands ADD INDEX idx_nano_commands_created_at (created_at)`)

	applyNext(t, db)

	require.Equal(t, []string{"created_at"}, indexColumns(t, db, "nano_commands", "idx_nano_commands_created_at"))
}
