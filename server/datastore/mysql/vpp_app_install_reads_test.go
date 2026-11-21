package mysql

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/migrations/tables"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestVPPAppInstallHotAppReads(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer TruncateTables(t, ds)
	ctx := t.Context()
	test.CreateInsertGlobalVPPToken(t, ds)
	app, err := ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		AdamID: "hot-app", Platform: fleet.IOSPlatform, Name: "Hot app", BundleIdentifier: "com.example.hotapp",
	}, nil, nil)
	require.NoError(t, err)
	other, err := ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		AdamID: "other-app", Platform: fleet.IOSPlatform, Name: "Other app", BundleIdentifier: "com.example.otherapp",
	}, nil, nil)
	require.NoError(t, err)
	conn, err := ds.writer(ctx).Connx(ctx)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()
	exec := func(query string, args ...any) {
		_, err := conn.ExecContext(ctx, query, args...)
		require.NoError(t, err)
	}
	// One request per host exposes plans that scan unrelated hosts instead of the selected app.
	const hotHosts = 5000
	exec(`SET SESSION cte_max_recursion_depth = 20000`)
	exec(`INSERT INTO hosts (id, osquery_host_id, uuid, hostname, platform,
		detail_updated_at, label_updated_at, policy_updated_at, last_enrolled_at)
		WITH RECURSIVE seq(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM seq WHERE i < ?)
		SELECT i, CONCAT('hot-', i), CONCAT('hot-', i), CONCAT('hot-', i), 'ios', NOW(), NOW(), NOW(), NOW() FROM seq`, hotHosts*3)
	exec(`INSERT INTO host_vpp_software_installs
		(host_id, adam_id, platform, command_uuid, vpp_app_team_id, created_at, verification_at, verification_failed_at)
		SELECT id, IF(id <= ?, ?, ?), 'ios', CONCAT('hot-command-', id),
			IF(MOD(id, 10) = 0, NULL, IF(id <= ?, ?, ?)),
			TIMESTAMP('2026-02-01') + INTERVAL id SECOND,
			IF(MOD(id, 10) < 7, NOW(), NULL), IF(MOD(id, 10) IN (7, 8), NOW(), NULL)
		FROM hosts ORDER BY id`, hotHosts, app.AdamID, other.AdamID, hotHosts, app.AppTeamID, other.AppTeamID)
	exec(`INSERT INTO host_vpp_app_installs
		(host_id, adam_id, platform, version_bucket, vpp_app_team_id, installed_install_id, failed_install_id,
		unfinished_install_id, visible_install_id, latest_install_id, unfailed_install_id,
		visible_created_at, visible_command_uuid, visible_verification_at, visible_verification_failed_at,
		latest_created_at, latest_command_uuid, latest_verification_at, latest_verification_failed_at) ` + tables.VPPAppInstallStateQuery)
	exec(`INSERT INTO nano_devices (id, serial_number, authenticate)
		SELECT uuid, uuid, '<plist/>' FROM hosts`)
	exec(`INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, enabled)
		SELECT uuid, uuid, 'Device', 'com.example.hotapp', 'hot', '00aa', 1 FROM hosts`)
	exec(`INSERT INTO nano_commands (command_uuid, request_type, command)
		SELECT command_uuid, 'InstallApplication', '<plist/>' FROM host_vpp_software_installs`)
	exec(`INSERT INTO nano_command_results (id, command_uuid, status, result)
		SELECT CONCAT('hot-', host_id), command_uuid, IF(verification_at IS NULL, 'Error', 'Acknowledged'), '<?xml version="1.0"?><plist/>'
		FROM host_vpp_software_installs WHERE verification_at IS NOT NULL OR verification_failed_at IS NOT NULL`)
	_, err = ds.UpdateHostSoftware(ctx, 1, []fleet.Software{{Name: app.Name, Version: "1", Source: "ios_apps", BundleIdentifier: app.BundleIdentifier}})
	require.NoError(t, err)
	exec(`INSERT IGNORE INTO host_software (host_id, software_id)
		SELECT h.id, hs.software_id FROM hosts h JOIN host_software hs ON hs.host_id = 1 WHERE h.id <= ? AND MOD(h.id, 10) != 0`, hotHosts)

	reads := func() int64 {
		rows, err := conn.QueryxContext(ctx, `SHOW SESSION STATUS LIKE 'Handler_read%'`)
		require.NoError(t, err)
		defer rows.Close()
		var total int64
		for rows.Next() {
			var name string
			var n int64
			require.NoError(t, rows.Scan(&name, &n))
			total += n
		}
		require.NoError(t, rows.Err())
		return total
	}
	measure := func(query string, args []any) ([]string, int64) {
		before := reads()
		rows, err := conn.QueryxContext(ctx, query, args...)
		require.NoError(t, err)
		var result []string
		for rows.Next() {
			values, err := rows.SliceScan()
			require.NoError(t, err)
			for i, value := range values {
				if b, ok := value.([]byte); ok {
					values[i] = string(b)
				}
			}
			value, err := json.Marshal(values)
			require.NoError(t, err)
			result = append(result, string(value))
		}
		require.NoError(t, rows.Err())
		// Closed before reading the counters so the delta covers the whole result.
		require.NoError(t, rows.Close()) //nolint:sqlclosecheck
		delta := reads() - before
		slices.Sort(result)
		return result, delta
	}
	named := func(query string, softwareStatus fleet.SoftwareInstallerStatus) (string, []any) {
		query, args, err := sqlx.Named(query, map[string]any{
			"version_ids": []uint{app.AppTeamID}, "adam_id": app.AdamID, "platform": app.Platform,
			"software_status_installed": fleet.SoftwareInstalled, "software_status_failed": fleet.SoftwareInstallFailed,
			"software_status_pending": fleet.SoftwareInstallPending, "status": softwareStatus, "vpp_app_team_id": app.AppTeamID,
			"mdm_status_error": fleet.MDMAppleStatusError, "mdm_status_format_error": fleet.MDMAppleStatusCommandFormatError,
		})
		require.NoError(t, err)
		query, args, err = sqlx.In(query, args...)
		require.NoError(t, err)
		return query, args
	}
	status := vppAppHostStatusNamedQuery("ranked", "ncr", "")
	wrapSummary := func(query string) string {
		return fmt.Sprintf(`SELECT vpp_app_team_id,
			COALESCE(SUM(IF(status = :software_status_pending, 1, 0)), 0) AS pending,
			COALESCE(SUM(IF(status = :software_status_failed, 1, 0)), 0) AS failed,
			COALESCE(SUM(IF(status = :software_status_installed, 1, 0)), 0) AS installed
			FROM (%s) summary GROUP BY vpp_app_team_id`, query)
	}
	hostIDs := make([]uint, 1000)
	for i := range hostIDs {
		hostIDs[i] = uint(i + 1) //nolint:gosec // fixture indices are positive and bounded
	}
	versionFilter := func(query string) string {
		return fmt.Sprintf(`SELECT h.id FROM hosts h JOIN (%s) hss ON hss.host_id = h.id
			AND hss.vpp_app_team_id = :vpp_app_team_id AND hss.status = :status`, query)
	}
	for sample := range 3 {
		exec(`ANALYZE TABLE hosts, host_vpp_software_installs, host_vpp_app_installs, vpp_apps_teams, nano_command_results, host_software`)
		for _, tc := range []struct {
			name, legacy, current string
			status                fleet.SoftwareInstallerStatus
		}{
			{"summary", wrapSummary(legacyVPPAppVersionHostStatusesQuery), wrapSummary(vppAppVersionHostStatusesQuery), ""},
			{"failed", `SELECT h.id FROM hosts h ` + fmt.Sprintf(legacyVPPAppTerminalJoinQuery, status), `SELECT h.id FROM hosts h ` + fmt.Sprintf(vppAppTerminalJoinQuery, status), fleet.SoftwareInstallFailed},
			{"installed", `SELECT h.id FROM hosts h ` + fmt.Sprintf(legacyVPPAppTerminalJoinQuery, status), `SELECT h.id FROM hosts h ` + fmt.Sprintf(vppAppTerminalJoinQuery, status), fleet.SoftwareInstalled},
			{"failed count", `SELECT COUNT(*) FROM hosts h ` + fmt.Sprintf(legacyVPPAppTerminalJoinQuery, status), `SELECT COUNT(*) FROM hosts h ` + fmt.Sprintf(vppAppTerminalJoinQuery, status), fleet.SoftwareInstallFailed},
			{"version failed", versionFilter(legacyVPPAppVersionHostStatusesQuery), versionFilter(vppAppVersionHostStatusesQuery), fleet.SoftwareInstallFailed},
			{"reconcile", legacyHostAppStoreAppInstallVersionsQuery, hostAppStoreAppInstallVersionsQuery, ""},
		} {
			t.Run(fmt.Sprintf("%s/%d", tc.name, sample), func(t *testing.T) {
				var legacy, current string
				var legacyArgs, currentArgs []any
				if tc.name == "reconcile" {
					legacy, legacyArgs, err = sqlx.In(tc.legacy, 0, []string{"ios"}, hostIDs, nil)
					require.NoError(t, err)
					current, currentArgs, err = sqlx.In(tc.current, 0, []string{"ios"}, hostIDs, nil)
					require.NoError(t, err)
				} else {
					legacy, legacyArgs = named(tc.legacy, tc.status)
					current, currentArgs = named(tc.current, tc.status)
				}
				want, legacyReads := measure(legacy, legacyArgs)
				require.Len(t, want, map[string]int{
					"summary": 1, "failed": 1000, "installed": 3500, "failed count": 1, "version failed": 1000, "reconcile": 700,
				}[tc.name])
				got, currentReads := measure(current, currentArgs)
				require.Equal(t, want, got, tc.name)
				t.Logf("sample %d %s: legacy=%d current=%d", sample, tc.name, legacyReads, currentReads)
				switch tc.name {
				case "reconcile":
					// Reconcile still reads needs_resend from requests because SCIM updates it without refreshing state.
					require.LessOrEqual(t, currentReads*100, legacyReads*115, tc.name)
				case "summary":
					// Optimizer sampling can vary summary reads slightly even with the same plan.
					require.LessOrEqual(t, currentReads, legacyReads*101/100, tc.name)
				default:
					require.LessOrEqual(t, currentReads, legacyReads, tc.name)
				}
				if sample == 0 {
					var plan []string
					require.NoError(t, sqlx.SelectContext(ctx, conn, &plan, `EXPLAIN FORMAT=TREE `+current, currentArgs...))
					t.Logf("EXPLAIN %s\n%s", tc.name, strings.Join(plan, "\n"))
				}
			})
		}
	}
}
