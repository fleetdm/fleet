package tables

import (
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestUp_20260723181402_Performance inserts a realistic number of hosts and
// certificates, then runs the migration and reports how long it takes. This
// validates the fix for #50228 where the original migration took ~14 minutes
// at 100K hosts due to repeated JOIN on the hosts table per batch.
//
// The test uses 1,000 Windows hosts x 20 certs each = 20,000 cert rows.
// This is enough to exercise the batching logic (hostBatchSize=500) while
// staying fast enough for CI (~seconds, not minutes).
func TestUp_20260723181402_Performance(t *testing.T) {
	db := applyUpToPrev(t)

	const (
		numWindowsHosts = 1000
		numMacHosts     = 200
		certsPerHost    = 20
	)

	// Insert Windows hosts.
	for i := 0; i < numWindowsHosts; i++ {
		uid := fmt.Sprintf("win-%d", i)
		execNoErr(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, 'windows')`,
			uid, uid, uid, uid)
	}

	// Insert macOS hosts (should be untouched by migration).
	for i := 0; i < numMacHosts; i++ {
		uid := fmt.Sprintf("mac-%d", i)
		execNoErr(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, 'darwin')`,
			uid, uid, uid, uid)
	}

	// Collect host IDs.
	var windowsIDs []uint
	require.NoError(t, db.Select(&windowsIDs, `SELECT id FROM hosts WHERE platform = 'windows'`))
	var macIDs []uint
	require.NoError(t, db.Select(&macIDs, `SELECT id FROM hosts WHERE platform = 'darwin'`))

	// Insert certs for all hosts.
	insertCerts := func(hostIDs []uint, prefix string) {
		for _, hid := range hostIDs {
			for j := 0; j < certsPerHost; j++ {
				fullHash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d-%d", prefix, hid, j)))
				sha1Bytes := fullHash[:20] // sha1_sum column is 20 bytes
				serial := fmt.Sprintf("%s-%d-%d", prefix, hid, j)
				execNoErr(t, db, `
					INSERT INTO host_certificates (
						host_id, not_valid_after, not_valid_before, certificate_authority,
						common_name, key_algorithm, key_strength, key_usage,
						serial, signing_algorithm,
						subject_country, subject_org, subject_org_unit, subject_common_name,
						issuer_country, issuer_org, issuer_org_unit, issuer_common_name,
						sha1_sum, origin, deleted_at
					) VALUES (?, '2027-01-01', '2026-01-01', 0, 'cn', 'rsa', 2048, 'digitalSignature',
						?, 'sha256WithRSAEncryption', '', '', '', '', '', '', '', '',
						?, 'osquery', NULL)`,
					hid, serial, sha1Bytes)
			}
		}
	}
	insertCerts(windowsIDs, "win")
	insertCerts(macIDs, "mac")

	// Verify row counts before migration.
	var totalCerts int
	require.NoError(t, db.Get(&totalCerts, `SELECT COUNT(*) FROM host_certificates WHERE deleted_at IS NULL`))
	t.Logf("Total live certs before migration: %d", totalCerts)
	require.Equal(t, (numWindowsHosts+numMacHosts)*certsPerHost, totalCerts)

	// Run the migration and time it.
	start := time.Now()
	applyNext(t, db)
	elapsed := time.Since(start)
	t.Logf("Migration completed in %s for %d Windows hosts x %d certs = %d cert rows",
		elapsed, numWindowsHosts, certsPerHost, numWindowsHosts*certsPerHost)

	// Verify: all Windows osquery certs are soft-deleted.
	var windowsLive int
	require.NoError(t, db.Get(&windowsLive, `
		SELECT COUNT(*) FROM host_certificates hc
		JOIN hosts h ON h.id = hc.host_id
		WHERE h.platform = 'windows' AND hc.origin = 'osquery' AND hc.deleted_at IS NULL`))
	require.Equal(t, 0, windowsLive, "all Windows osquery certs should be soft-deleted")

	// Verify: all macOS certs are untouched.
	var macLive int
	require.NoError(t, db.Get(&macLive, `
		SELECT COUNT(*) FROM host_certificates hc
		JOIN hosts h ON h.id = hc.host_id
		WHERE h.platform = 'darwin' AND hc.origin = 'osquery' AND hc.deleted_at IS NULL`))
	require.Equal(t, numMacHosts*certsPerHost, macLive, "macOS certs should be untouched")

	// At 1,000 hosts x 20 certs, the migration should complete in under 30 seconds.
	// The original implementation took proportionally ~14 minutes at 100K hosts.
	require.Less(t, elapsed, 30*time.Second, "migration took too long, possible performance regression")
}
