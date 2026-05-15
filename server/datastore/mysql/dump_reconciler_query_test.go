package mysql

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// TestDumpReconcilerQuery prints the full SQL of the combined Apple profile
// reconciler query with `?` placeholders replaced by literals, so it can be
// pasted into a MySQL shell for EXPLAIN ANALYZE. Run with:
//
//	go test -v -count 1 -run TestDumpReconcilerQuery ./server/datastore/mysql/
//
// This test never accesses MySQL; it only does string formatting, so it
// runs without MYSQL_TEST set.
func TestDumpReconcilerQuery(t *testing.T) {
	desiredState := fmt.Sprintf(generateDesiredStateQuery("profile"), "TRUE", "TRUE", "TRUE", "TRUE")

	query := fmt.Sprintf(`
	WITH ds AS (
		%s
	)
	SELECT /*+ NO_MERGE(ds) */
		'install' AS op,
		ds.profile_uuid AS profile_uuid,
		ds.host_uuid AS host_uuid,
		ds.host_platform AS host_platform,
		ds.profile_identifier AS profile_identifier,
		ds.profile_name AS profile_name,
		ds.checksum AS checksum,
		ds.secrets_updated_at AS secrets_updated_at,
		ds.scope AS scope,
		ds.device_enrolled_at AS device_enrolled_at,
		'' AS operation_type,
		'' AS detail,
		NULL AS status,
		'' AS command_uuid
	FROM ds
		LEFT JOIN host_mdm_apple_profiles hmae
			ON hmae.profile_uuid = ds.profile_uuid AND hmae.host_uuid = ds.host_uuid
	WHERE
		( hmae.checksum != ds.checksum ) OR IFNULL(hmae.secrets_updated_at < ds.secrets_updated_at, FALSE) OR
		( hmae.profile_uuid IS NULL AND hmae.host_uuid IS NULL ) OR
		( hmae.host_uuid IS NOT NULL AND ( hmae.operation_type = ? OR hmae.operation_type IS NULL ) ) OR
		( hmae.host_uuid IS NOT NULL AND hmae.operation_type = ? AND hmae.status IS NULL )

	UNION ALL

	SELECT /*+ NO_MERGE(ds) */
		'remove' AS op,
		hmae.profile_uuid AS profile_uuid,
		hmae.host_uuid AS host_uuid,
		'' AS host_platform,
		hmae.profile_identifier AS profile_identifier,
		hmae.profile_name AS profile_name,
		hmae.checksum AS checksum,
		hmae.secrets_updated_at AS secrets_updated_at,
		hmae.scope AS scope,
		NULL AS device_enrolled_at,
		hmae.operation_type AS operation_type,
		COALESCE(hmae.detail, '') AS detail,
		hmae.status AS status,
		hmae.command_uuid AS command_uuid
	FROM ds
		RIGHT JOIN host_mdm_apple_profiles hmae
			ON hmae.profile_uuid = ds.profile_uuid AND hmae.host_uuid = ds.host_uuid
	WHERE
		ds.profile_uuid IS NULL AND ds.host_uuid IS NULL AND
		( hmae.operation_type IS NULL OR hmae.operation_type != ? OR hmae.status IS NULL ) AND
		NOT EXISTS (
			SELECT 1
			FROM mdm_configuration_profile_labels mcpl
			WHERE
				mcpl.apple_profile_uuid = hmae.profile_uuid AND
				mcpl.label_id IS NULL
		)
`, desiredState)

	// Substitute the three placeholders in order: install half's
	// operation_type = remove, operation_type = install, then remove
	// half's operation_type != remove.
	args := []string{
		fmt.Sprintf("'%s'", fleet.MDMOperationTypeRemove),
		fmt.Sprintf("'%s'", fleet.MDMOperationTypeInstall),
		fmt.Sprintf("'%s'", fleet.MDMOperationTypeRemove),
	}
	for _, a := range args {
		query = strings.Replace(query, "?", a, 1)
	}

	out := "/tmp/reconciler_query.sql"
	if env := os.Getenv("DUMP_RECONCILER_QUERY_PATH"); env != "" {
		out = env
	}
	if err := os.WriteFile(out, []byte(query+"\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", out, err)
	}
	t.Logf("wrote reconciler query to %s (%d bytes)", out, len(query))
}
