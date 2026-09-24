package tables

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20260723181412(t *testing.T) {
	db := applyUpToPrev(t)
	oldBatchSize := backfillAppleBuiltinLabelMembershipsBatchSize
	backfillAppleBuiltinLabelMembershipsBatchSize = 2
	t.Cleanup(func() { backfillAppleBuiltinLabelMembershipsBatchSize = oldBatchSize })

	_, err := db.Exec(`
		DELETE FROM label_membership;
		DELETE FROM labels WHERE name IN ('All Hosts', 'macOS', 'iOS', 'iPadOS');
		INSERT INTO labels (name, description, query, platform, label_type, label_membership_type) VALUES
			('All Hosts', '', '', '', 1, 0),
			('macOS', '', '', 'darwin', 1, 0),
			('iOS', '', '', 'ios', 1, 1),
			('iPadOS', '', '', 'ipados', 1, 1);
		INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES
			('mac', 'mac-node', 'mac', 'mac-uuid', 'darwin'),
			('iphone', 'iphone-node', 'iphone', 'iphone-uuid', 'ios'),
			('ipad', 'ipad-node', 'ipad', 'ipad-uuid', 'ipados'),
			('platformless-mdm', 'platformless-mdm-node', 'platformless-mdm', 'platformless-mdm-uuid', ''),
			('platformless-unmanaged', 'platformless-unmanaged-node', 'platformless-unmanaged', 'platformless-unmanaged-uuid', ''),
			('android', 'android-node', 'android', 'android-uuid', 'android'),
			('windows', 'windows-node', 'windows', 'windows-uuid', 'windows');
		INSERT INTO mobile_device_management_solutions (name, server_url) VALUES ('Fleet', 'https://fleet.example.com')
			ON DUPLICATE KEY UPDATE server_url = VALUES(server_url);
		INSERT INTO host_mdm (host_id, enrolled, server_url, mdm_id)
		SELECT h.id, 1, 'https://fleet.example.com', mdm.id
		FROM hosts h JOIN mobile_device_management_solutions mdm ON mdm.name = 'Fleet'
		WHERE h.hostname = 'platformless-mdm';
		INSERT INTO label_membership (host_id, label_id)
		SELECT h.id, l.id FROM hosts h JOIN labels l ON l.name = 'All Hosts' WHERE h.hostname = 'iphone';
	`)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE label_membership SET updated_at = '2020-01-02 03:04:05'`)
	require.NoError(t, err)

	applyNext(t, db)
	assertAppleBuiltinLabelMemberships(t, db)
	var existingUpdatedAt string
	require.NoError(t, db.Get(&existingUpdatedAt, `
		SELECT DATE_FORMAT(lm.updated_at, '%Y-%m-%d %H:%i:%s')
		FROM label_membership lm
		JOIN hosts h ON h.id = lm.host_id
		JOIN labels l ON l.id = lm.label_id
		WHERE h.hostname = 'iphone' AND l.name = 'All Hosts'
	`))
	require.Equal(t, "2020-01-02 03:04:05", existingUpdatedAt)

	// Running the migration again is safe and does not create duplicate rows.
	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, Up_20260723181412(tx))
	require.NoError(t, tx.Commit())
	assertAppleBuiltinLabelMemberships(t, db)
}

func assertAppleBuiltinLabelMemberships(t *testing.T, db *sqlx.DB) {
	t.Helper()
	type membership struct {
		Hostname string `db:"hostname"`
		Label    string `db:"label"`
	}
	var got []membership
	err := db.Select(&got, `
		SELECT h.hostname, l.name AS label
		FROM label_membership lm
		JOIN hosts h ON h.id = lm.host_id
		JOIN labels l ON l.id = lm.label_id
		WHERE h.hostname IN ('mac', 'iphone', 'ipad', 'platformless-mdm', 'platformless-unmanaged', 'android', 'windows')
		ORDER BY h.hostname, l.name
	`)
	require.NoError(t, err)
	require.Equal(t, []membership{
		{Hostname: "ipad", Label: "All Hosts"},
		{Hostname: "ipad", Label: "iPadOS"},
		{Hostname: "iphone", Label: "All Hosts"},
		{Hostname: "iphone", Label: "iOS"},
		{Hostname: "mac", Label: "All Hosts"},
		{Hostname: "mac", Label: "macOS"},
		{Hostname: "platformless-mdm", Label: "All Hosts"},
		{Hostname: "platformless-mdm", Label: "macOS"},
	}, got)
}
