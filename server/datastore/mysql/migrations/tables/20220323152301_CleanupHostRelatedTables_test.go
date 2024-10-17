package tables

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20220323152301(t *testing.T) {
	db := applyUpToPrev(t)

	hosts := createHostsWithSoftware(t, db)

	var count int
	err := db.Get(&count, `SELECT COUNT(*) FROM hosts`)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	err = db.Get(&count, `SELECT COUNT(*) FROM host_software`)
	require.NoError(t, err)
	require.Equal(t, 4, count)

	// delete the second host
	_, err = db.Exec(`DELETE FROM hosts WHERE id = ?`, hosts[1].ID)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	err = db.Get(&count, `SELECT COUNT(*) FROM hosts`)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	err = db.Get(&count, `SELECT COUNT(*) FROM host_software`)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	err = db.Get(&count, `SELECT COUNT(*) FROM host_software WHERE host_id = ?`, hosts[1].ID)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func createHostsWithSoftware(t *testing.T, db *sqlx.DB) []*fleet.Host {
	const insStmt = `
	INSERT INTO hosts (
		osquery_host_id,
		detail_updated_at,
		label_updated_at,
		policy_updated_at,
		node_key,
		hostname,
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
		refetch_requested
	)
	VALUES( ?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,? )
	`

	hosts := make([]*fleet.Host, 2)
	for i := range hosts {
		host := &fleet.Host{
			OsqueryHostID:   ptr.String(strconv.Itoa(i + 1)),
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         ptr.String(strconv.Itoa(i + 1)),
			UUID:            strconv.Itoa(i + 1),
			Hostname:        fmt.Sprintf("foo%d.local", i+1),
		}

		res, err := db.Exec(
			insStmt,
			host.OsqueryHostID,
			host.DetailUpdatedAt,
			host.LabelUpdatedAt,
			host.PolicyUpdatedAt,
			host.NodeKey,
			host.Hostname,
			host.UUID,
			host.Platform,
			host.OsqueryVersion,
			host.OSVersion,
			host.Uptime,
			host.Memory,
			host.TeamID,
			host.DistributedInterval,
			host.LoggerTLSPeriod,
			host.ConfigTLSRefresh,
			host.RefetchRequested,
		)
		require.NoError(t, err)
		id, _ := res.LastInsertId()
		host.ID = uint(id) //nolint:gosec // dismiss G115
		hosts[i] = host
	}

	// create software for each host
	const (
		insSw = "INSERT INTO software " +
			"(name, version, source, `release`, vendor, arch, bundle_identifier) " +
			"VALUES (?, ?, ?, ?, ?, ?, ?)"
		insHostSw = `INSERT INTO host_software (host_id, software_id) VALUES (?, ?)`
	)
	software := []*fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "bar", Version: "1.0.0", Source: "deb_packages"},
	}
	for _, sw := range software {
		res, err := db.Exec(insSw, sw.Name, sw.Version, sw.Source, sw.Release, sw.Vendor, sw.Arch, sw.BundleIdentifier)
		require.NoError(t, err)
		id, _ := res.LastInsertId()
		sw.ID = uint(id) //nolint:gosec // dismiss G115
	}

	for _, host := range hosts {
		for _, sw := range software {
			_, err := db.Exec(insHostSw, host.ID, sw.ID)
			require.NoError(t, err)
		}
	}

	return hosts
}
