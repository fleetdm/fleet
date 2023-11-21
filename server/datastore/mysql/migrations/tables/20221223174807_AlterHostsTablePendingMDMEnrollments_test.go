package tables

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/stretchr/testify/require"
)

func TestUp_20221223174807(t *testing.T) {
	db := applyUpToPrev(t)

	someString := func() string {
		s, err := server.GenerateRandomText(16)
		require.NoError(t, err)
		return s
	}

	insertStmt := `
		INSERT INTO hosts (
			osquery_host_id,
			detail_updated_at,
			label_updated_at,
			policy_updated_at,
			node_key,
			hostname,
			computer_name,
			uuid,
			platform,
			osquery_version,
			os_version,
			uptime,
			memory,
			team_id,
			distributed_interval,
			logger_tls_period,
			config_tls_refresh,
			refetch_requested,
			hardware_serial
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	newHostArgs := func() []any {
		return []any{
			someString(),
			time.Now(),
			time.Now(),
			time.Now(),
			someString(),
			someString(),
			someString(),
			someString(),
			someString(),
			someString(),
			someString(),
			1337,
			1337,
			nil,
			1337,
			1337,
			1337,
			true,
			someString(),
		}
	}

	args := newHostArgs()
	execNoErr(t, db, insertStmt, args...)

	args = newHostArgs()
	args[0] = nil // replaces string for "osquery_host_id"
	_, err := db.Exec(insertStmt, args...)
	require.ErrorContains(t, err, "Error 1048")
	require.ErrorContains(t, err, "Column 'osquery_host_id' cannot be null")

	// Apply current migration.
	applyNext(t, db)

	args = newHostArgs()
	execNoErr(t, db, insertStmt, args...)

	args = newHostArgs()
	args[0] = nil // replaces string for "osquery_host_id"
	_, err = db.Exec(insertStmt, args...)
	require.NoError(t, err)
}
