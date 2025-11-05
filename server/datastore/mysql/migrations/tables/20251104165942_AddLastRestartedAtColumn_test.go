package tables

import "testing"

func TestUp_20251104165942(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert test hosts
	host1ID := execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, uuid, platform, uptime, detail_updated_at) VALUES (?, ?, ?, ?, ?, ?)`, "host1", "key1", "uuid1", "darwin", 2388335000000000, "2025-11-04 23:07:56")
	host2ID := execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, uuid, platform, uptime, detail_updated_at) VALUES (?, ?, ?, ?, ?, ?)`, "host2", "key2", "uuid2", "darwin", 0, "2025-11-04 23:07:56")
	host3ID := execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, uuid, platform, uptime, detail_updated_at) VALUES (?, ?, ?, ?, ?, ?)`, "host3", "key3", "uuid3", "darwin", 2388335000000000, 0)

	// Apply current migration.
	applyNext(t, db)

	//
	// Check data, insert new entries, e.g. to verify migration is safe.
	//
	// ...
}
