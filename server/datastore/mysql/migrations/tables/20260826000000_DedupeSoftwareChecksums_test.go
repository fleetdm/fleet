package tables

import (
	"crypto/md5" //nolint:gosec // matches the software checksum hash, not used for security
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260826000000(t *testing.T) {
	db := applyUpToPrev(t)

	// Shrink the batch sizes so this fixture exercises the batching loops (multiple
	// dedupe batches and multiple recompute id ranges) rather than a single pass.
	oldDedupe, oldRecompute := dedupeSoftwareBatchSize, recomputeSoftwareBatchSize
	dedupeSoftwareBatchSize, recomputeSoftwareBatchSize = 2, 1
	t.Cleanup(func() { dedupeSoftwareBatchSize, recomputeSoftwareBatchSize = oldDedupe, oldRecompute })

	ck := func(cols ...string) []byte {
		sum := md5.Sum([]byte(strings.Join(cols, "\x00"))) //nolint:gosec // matches the software checksum hash
		return sum[:]
	}
	// canonical (current name-last ordering) and legacy (pre-v4.76.0 name-first ordering)
	// checksums for a homebrew row where only name/version/source are populated (the other
	// identity columns default to '').
	canonical := func(name, version string) []byte {
		return ck(version, "homebrew_packages", "", "", "", "", "", "", name)
	}
	nameFirst := func(name, version string) []byte {
		return ck(name, version, "homebrew_packages", "", "", "", "", "", "")
	}
	insertSW := func(name, version string, checksum []byte) int64 {
		return execNoErrLastID(t, db,
			`INSERT INTO software (name, version, source, checksum) VALUES (?, ?, 'homebrew_packages', ?)`,
			name, version, checksum)
	}
	link := func(hostID int, swID int64) {
		execNoErr(t, db, `INSERT INTO host_software (host_id, software_id) VALUES (?, ?)`, hostID, swID)
	}

	// 1. Duplicate group: canonical row (host 1) + legacy row (host 2, and host 1 -> a
	//    collision), plus an install path on the legacy row.
	giflibCanon := insertSW("giflib", "5.2.2", canonical("giflib", "5.2.2"))
	giflibStale := insertSW("giflib", "5.2.2", nameFirst("giflib", "5.2.2"))
	link(1, giflibCanon)
	link(2, giflibStale)
	link(1, giflibStale) // collision: host 1 is on both rows
	execNoErr(t, db,
		`INSERT INTO host_software_installed_paths (host_id, software_id, installed_path) VALUES (2, ?, '/opt/homebrew/Cellar/giflib')`,
		giflibStale)

	// 2. Lone legacy row (no twin yet): must be recomputed to canonical so it can't
	//    re-duplicate when its host reports again.
	zlibStale := insertSW("zlib", "1.0", nameFirst("zlib", "1.0"))
	link(3, zlibStale)

	// 3. Control: an already-canonical row that must be left untouched.
	openssl := insertSW("openssl", "3.0", canonical("openssl", "3.0"))
	link(4, openssl)

	// 4. macOS apps. Migration 20251015103505 only recomputed apps rows that have a
	//    bundle_identifier, so those are already canonical and must not change. Apps
	//    without one still carry the original apps formula, which left the name out
	//    entirely; they are stale and must be repaired here or they duplicate as soon as
	//    their host reports again.
	appsCanonical := func(name, version, bundleID string) []byte {
		return ck(version, "apps", bundleID, "", "", "", "", "", name)
	}
	appsNameExcluded := ck("1.0", "apps", "", "", "", "", "", "") // pre-v4.76.0 apps formula
	safariID := execNoErrLastID(t, db,
		`INSERT INTO software (name, version, source, bundle_identifier, checksum) VALUES ('Safari', '17.0', 'apps', 'com.apple.Safari', ?)`,
		appsCanonical("Safari", "17.0", "com.apple.Safari"))
	noBundleID := execNoErrLastID(t, db,
		`INSERT INTO software (name, version, source, checksum) VALUES ('LegacyApp', '1.0', 'apps', ?)`,
		appsNameExcluded)
	link(8, safariID)
	link(9, noBundleID)

	// 5. A three-member group (canonical + two differently-stale rows). Together with
	//    giflib this makes three duplicate rows, so the merge runs over several batches.
	curlCanon := insertSW("curl", "8.1", canonical("curl", "8.1"))
	curlStaleA := insertSW("curl", "8.1", nameFirst("curl", "8.1"))
	curlStaleB := insertSW("curl", "8.1", ck("curl-other-legacy-formula"))
	link(5, curlCanon)
	link(6, curlStaleA)
	link(7, curlStaleB)

	applyNext(t, db)

	countSW := func(name, version string) int {
		var n int
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM software WHERE name=? AND version=? AND source='homebrew_packages'`, name, version).Scan(&n))
		return n
	}
	getOne := func(name, version string) (int64, []byte) {
		var id int64
		var cksum []byte
		require.NoError(t, db.QueryRow(`SELECT id, checksum FROM software WHERE name=? AND version=? AND source='homebrew_packages'`, name, version).Scan(&id, &cksum))
		return id, cksum
	}
	hostsOn := func(swID int64) []int {
		rows, err := db.Query(`SELECT host_id FROM host_software WHERE software_id=? ORDER BY host_id`, swID)
		require.NoError(t, err)
		defer rows.Close()
		var hs []int
		for rows.Next() {
			var h int
			require.NoError(t, rows.Scan(&h))
			hs = append(hs, h)
		}
		require.NoError(t, rows.Err())
		return hs
	}

	// 1. Merged to a single canonical row; hosts 1 and 2 consolidated (host 1's collision
	//    resolved to one link); the stale row and its links are gone; the install path is
	//    repointed onto the survivor.
	require.Equal(t, 1, countSW("giflib", "5.2.2"))
	giflibID, giflibCk := getOne("giflib", "5.2.2")
	require.Equal(t, canonical("giflib", "5.2.2"), giflibCk)
	require.Equal(t, []int{1, 2}, hostsOn(giflibID))
	var staleRows int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM software WHERE id=?`, giflibStale).Scan(&staleRows))
	require.Zero(t, staleRows)
	var pathSWID int64
	require.NoError(t, db.QueryRow(`SELECT software_id FROM host_software_installed_paths WHERE host_id=2 AND installed_path='/opt/homebrew/Cellar/giflib'`).Scan(&pathSWID))
	require.Equal(t, giflibID, pathSWID)

	// 2. Lone legacy row: same row kept, checksum recomputed to canonical, host retained.
	require.Equal(t, 1, countSW("zlib", "1.0"))
	zlibID, zlibCk := getOne("zlib", "1.0")
	require.Equal(t, zlibStale, zlibID)
	require.Equal(t, canonical("zlib", "1.0"), zlibCk)
	require.Equal(t, []int{3}, hostsOn(zlibID))

	// 3. Control: untouched.
	require.Equal(t, 1, countSW("openssl", "3.0"))
	opensslID, opensslCk := getOne("openssl", "3.0")
	require.Equal(t, openssl, opensslID)
	require.Equal(t, canonical("openssl", "3.0"), opensslCk)
	require.Equal(t, []int{4}, hostsOn(opensslID))

	// 4. Apps: the bundle-identifier row was already canonical and is untouched; the one
	//    without a bundle identifier is repaired to the current formula (so it can no
	//    longer duplicate), keeping its row and host.
	var safariCk []byte
	require.NoError(t, db.QueryRow(`SELECT checksum FROM software WHERE id=?`, safariID).Scan(&safariCk))
	require.Equal(t, appsCanonical("Safari", "17.0", "com.apple.Safari"), safariCk)

	var legacyCk []byte
	require.NoError(t, db.QueryRow(`SELECT checksum FROM software WHERE id=?`, noBundleID).Scan(&legacyCk))
	require.NotEqual(t, appsNameExcluded, legacyCk, "pre-v4.76.0 apps row should be repaired")
	require.Equal(t, appsCanonical("LegacyApp", "1.0", ""), legacyCk)
	require.Equal(t, []int{9}, hostsOn(noBundleID))

	// 5. Three-member group collapsed onto the lowest id, with all three hosts kept.
	require.Equal(t, 1, countSW("curl", "8.1"))
	curlID, curlCk := getOne("curl", "8.1")
	require.Equal(t, curlCanon, curlID)
	require.Equal(t, canonical("curl", "8.1"), curlCk)
	require.Equal(t, []int{5, 6, 7}, hostsOn(curlID))
	for _, staleID := range []int64{curlStaleA, curlStaleB} {
		var n int
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM software WHERE id=?`, staleID).Scan(&n))
		require.Zerof(t, n, "stale row %d should be deleted", staleID)
	}
}
