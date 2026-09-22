package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260922170154(t *testing.T) {
	db := applyUpToPrev(t)

	const staleDetail = "Network error during SCEP enrollment: Failed to communicate with SCEP server"

	insertTemplate := func(hostUUID, status string, detail *string) {
		execNoErr(t, db, `
			INSERT INTO host_certificate_templates
				(host_uuid, certificate_template_id, status, operation_type, name, detail)
			VALUES (?, 1, ?, 'install', 'BeyondCorp', ?)`,
			hostUUID, status, detail)
	}

	stale := staleDetail
	empty := ""

	insertTemplate("delivered-with-stale-detail", "delivered", &stale)
	insertTemplate("delivered-already-empty", "delivered", &empty)
	insertTemplate("delivered-with-null-detail", "delivered", nil)
	insertTemplate("failed-with-real-detail", "failed", &stale)
	insertTemplate("pending-with-retry-detail", "pending", &stale)
	insertTemplate("delivering-with-retry-detail", "delivering", &stale)
	insertTemplate("verified-with-reported-detail", "verified", &stale)

	type templateRow struct {
		HostUUID  string  `db:"host_uuid"`
		Detail    *string `db:"detail"`
		UpdatedAt string  `db:"updated_at"`
	}
	snapshot := func() map[string]templateRow {
		var rows []templateRow
		err := db.Select(&rows, `SELECT host_uuid, detail, updated_at FROM host_certificate_templates`)
		require.NoError(t, err)
		byHost := make(map[string]templateRow, len(rows))
		for _, row := range rows {
			byHost[row.HostUUID] = row
		}
		return byHost
	}
	before := snapshot()

	// Apply current migration.
	applyNext(t, db)

	after := snapshot()

	// The stale failure message on a delivered certificate is what this migration exists to clear.
	require.NotNil(t, after["delivered-with-stale-detail"].Detail)
	require.Empty(t, *after["delivered-with-stale-detail"].Detail)
	require.NotNil(t, after["delivered-already-empty"].Detail)
	require.Empty(t, *after["delivered-already-empty"].Detail)

	// A NULL detail marks a manual resend rather than an automatic retry, so it must stay NULL.
	require.Nil(t, after["delivered-with-null-detail"].Detail)

	// On any other status the detail is either the current error message or what the host itself
	// reported, so it has to survive. Clearing these would turn a cosmetic bug into a real one.
	for _, hostUUID := range []string{
		"failed-with-real-detail",
		"pending-with-retry-detail",
		"delivering-with-retry-detail",
		"verified-with-reported-detail",
	} {
		require.NotNil(t, after[hostUUID].Detail, hostUUID)
		require.Equal(t, staleDetail, *after[hostUUID].Detail, hostUUID)
	}

	// updated_at is ON UPDATE CURRENT_TIMESTAMP and the migration assigns it to itself, so clearing
	// the detail doesn't make the certificate look freshly delivered. The retry backoff is computed
	// off updated_at, so moving it would delay a retry that is actually due.
	require.Equal(t,
		before["delivered-with-stale-detail"].UpdatedAt,
		after["delivered-with-stale-detail"].UpdatedAt,
	)
}
