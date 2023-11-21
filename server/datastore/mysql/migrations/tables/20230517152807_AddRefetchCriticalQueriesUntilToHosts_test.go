package tables

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/stretchr/testify/require"
)

func TestUp_20230517152807(t *testing.T) {
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

	// Apply current migration.
	applyNext(t, db)

	// existing host has a null refetch_critical_queries_until
	var until *time.Time
	err := db.Get(&until, "SELECT refetch_critical_queries_until FROM hosts WHERE osquery_host_id = ?", args[0])
	require.NoError(t, err)
	require.Nil(t, until)
}
