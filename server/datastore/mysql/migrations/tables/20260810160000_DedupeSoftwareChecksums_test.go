package tables

import (
	"database/sql"
	"fmt"
	"maps"

	"crypto/md5" //nolint:gosec // matches the software checksum hash, not used for security
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260810160000(t *testing.T) {
	db := applyUpToPrev(t)

	// Shrink the batch size so this fixture exercises the batching loop (multiple
	// dedupe batches) rather than a single pass.
	oldDedupe := dedupeSoftwareBatchSize
	dedupeSoftwareBatchSize = 2
	t.Cleanup(func() { dedupeSoftwareBatchSize = oldDedupe })

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

	// 1. In-scope duplicate group: canonical giflib row (host 1) + legacy row (host 2, and
	//    host 1 -> a collision), plus an install path on the legacy row.
	giflibCanon := insertSW("giflib", "5.2.2", canonical("giflib", "5.2.2"))
	giflibStale := insertSW("giflib", "5.2.2", nameFirst("giflib", "5.2.2"))
	link(1, giflibCanon)
	link(2, giflibStale)
	link(1, giflibStale) // collision: host 1 is on both rows
	execNoErr(t, db,
		`INSERT INTO host_software_installed_paths (host_id, software_id, installed_path) VALUES (2, ?, '/opt/homebrew/Cellar/giflib')`,
		giflibStale)
	// Host 2's link carries a last_opened_at that must survive the repoint, and both rows
	// carry the derived/vuln rows the merge has to clean up on the duplicate only.
	execNoErr(t, db, `UPDATE host_software SET last_opened_at = '2026-01-02 03:04:05' WHERE host_id = 2 AND software_id = ?`, giflibStale)
	for _, id := range []int64{giflibCanon, giflibStale} {
		execNoErr(t, db, `INSERT INTO software_cve (software_id, cve) VALUES (?, 'CVE-2024-45993')`, id)
		execNoErr(t, db, `INSERT INTO software_cpe (software_id, cpe) VALUES (?, ?)`, id, fmt.Sprintf("cpe-%d", id))
		execNoErr(t, db, `INSERT INTO software_host_counts (software_id, hosts_count, team_id, global_stats) VALUES (?, 1, 0, 1)`, id)
		execNoErr(t, db, `INSERT INTO kernel_host_counts (software_id, os_version_id, team_id, hosts_count) VALUES (?, ?, 0, 1)`, id, id)
	}

	// 2. Second in-scope version with a three-member group (canonical + two
	//    differently-stale rows). Together with 5.2.2 this makes three duplicate rows, so
	//    the merge runs over several batches.
	giflib521Canon := insertSW("giflib", "5.2.1", canonical("giflib", "5.2.1"))
	giflib521StaleA := insertSW("giflib", "5.2.1", nameFirst("giflib", "5.2.1"))
	giflib521StaleB := insertSW("giflib", "5.2.1", ck("giflib-other-legacy-formula"))
	link(10, giflib521Canon)
	link(11, giflib521StaleA)
	link(12, giflib521StaleB)

	// 3. In-scope lone legacy row (no twin yet): must be recomputed to canonical so it
	//    can't re-duplicate when its host reports again.
	giflib520Stale := insertSW("giflib", "5.2.0", nameFirst("giflib", "5.2.0"))
	link(13, giflib520Stale)

	// 4. Out of scope: a lone legacy row of another homebrew package. Same staleness as
	//    giflib 5.2.0, but not the reported software, so it must be left untouched.
	zlibStale := insertSW("zlib", "1.0", nameFirst("zlib", "1.0"))
	link(3, zlibStale)

	// 4b. Out of scope on the source axis: a stale giflib row from another source.
	giflibDebCk := ck("giflib", "5.2.2", "deb_packages", "", "", "", "", "", "")
	giflibDeb := execNoErrLastID(t, db,
		`INSERT INTO software (name, version, source, checksum) VALUES ('giflib', '5.2.2', 'deb_packages', ?)`,
		giflibDebCk)
	link(14, giflibDeb)

	// 5. Out of scope: an already-canonical row.
	openssl := insertSW("openssl", "3.0", canonical("openssl", "3.0"))
	link(4, openssl)

	// 6. Out of scope: macOS apps rows, canonical and stale (pre-v4.76.0 apps formula,
	//    which left the name out entirely). Both must be left untouched.
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

	// 7. Out of scope: a duplicate group of another homebrew package. Same shape as the
	//    giflib groups, but it must NOT be merged.
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
	getChecksum := func(id int64) []byte {
		var cksum []byte
		require.NoError(t, db.QueryRow(`SELECT checksum FROM software WHERE id=?`, id).Scan(&cksum))
		return cksum
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

	// The duplicate's derived rows are gone and the survivor's are intact, and the
	// repointed link kept its last_opened_at.
	countFor := func(table string, id int64) int {
		var n int
		require.NoError(t, db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE software_id = ?`, table), id).Scan(&n))
		return n
	}
	for _, table := range []string{"software_cve", "software_cpe", "software_host_counts", "kernel_host_counts"} {
		require.Zerof(t, countFor(table, giflibStale), "%s rows for the duplicate should be deleted", table)
		require.Equalf(t, 1, countFor(table, giflibCanon), "%s rows for the survivor should be kept", table)
	}
	var lastOpened sql.NullString
	require.NoError(t, db.QueryRow(`SELECT last_opened_at FROM host_software WHERE host_id = 2 AND software_id = ?`, giflibID).Scan(&lastOpened))
	require.True(t, lastOpened.Valid, "repointed link should keep last_opened_at")

	// 2. Three-member group collapsed onto the lowest id, with all three hosts kept.
	require.Equal(t, 1, countSW("giflib", "5.2.1"))
	giflib521ID, giflib521Ck := getOne("giflib", "5.2.1")
	require.Equal(t, giflib521Canon, giflib521ID)
	require.Equal(t, canonical("giflib", "5.2.1"), giflib521Ck)
	require.Equal(t, []int{10, 11, 12}, hostsOn(giflib521ID))
	for _, staleID := range []int64{giflib521StaleA, giflib521StaleB} {
		var n int
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM software WHERE id=?`, staleID).Scan(&n))
		require.Zerof(t, n, "stale row %d should be deleted", staleID)
	}

	// 3. In-scope lone legacy row: same row kept, checksum recomputed to canonical, host
	//    retained.
	require.Equal(t, 1, countSW("giflib", "5.2.0"))
	giflib520ID, giflib520Ck := getOne("giflib", "5.2.0")
	require.Equal(t, giflib520Stale, giflib520ID)
	require.Equal(t, canonical("giflib", "5.2.0"), giflib520Ck)
	require.Equal(t, []int{13}, hostsOn(giflib520ID))

	// 4. Out-of-scope lone legacy row: untouched, stale checksum and all.
	require.Equal(t, 1, countSW("zlib", "1.0"))
	zlibID, zlibCk := getOne("zlib", "1.0")
	require.Equal(t, zlibStale, zlibID)
	require.Equal(t, nameFirst("zlib", "1.0"), zlibCk, "out-of-scope stale row must keep its legacy checksum")
	require.Equal(t, []int{3}, hostsOn(zlibID))

	// 4b. Same name, different source: untouched, stale checksum and all.
	require.Equal(t, giflibDebCk, getChecksum(giflibDeb), "giflib from another source must keep its legacy checksum")
	require.Equal(t, []int{14}, hostsOn(giflibDeb))

	// 5. Out-of-scope canonical row: untouched.
	require.Equal(t, 1, countSW("openssl", "3.0"))
	opensslID, opensslCk := getOne("openssl", "3.0")
	require.Equal(t, openssl, opensslID)
	require.Equal(t, canonical("openssl", "3.0"), opensslCk)
	require.Equal(t, []int{4}, hostsOn(opensslID))

	// 6. Apps rows: both untouched, including the stale one.
	require.Equal(t, appsCanonical("Safari", "17.0", "com.apple.Safari"), getChecksum(safariID))
	require.Equal(t, appsNameExcluded, getChecksum(noBundleID), "out-of-scope stale apps row must keep its legacy checksum")
	require.Equal(t, []int{9}, hostsOn(noBundleID))

	// 7. Out-of-scope duplicate group: not merged — all three rows, checksums, and host
	//    links remain exactly as seeded.
	require.Equal(t, 3, countSW("curl", "8.1"))
	require.Equal(t, canonical("curl", "8.1"), getChecksum(curlCanon))
	require.Equal(t, nameFirst("curl", "8.1"), getChecksum(curlStaleA))
	require.Equal(t, ck("curl-other-legacy-formula"), getChecksum(curlStaleB))
	require.Equal(t, []int{5}, hostsOn(curlCanon))
	require.Equal(t, []int{6}, hostsOn(curlStaleA))
	require.Equal(t, []int{7}, hostsOn(curlStaleB))
}

// The migration merges rows that share a checksum identity, so the dangerous direction is
// over-merging: if a column that feeds the checksum were dropped from the partition key,
// genuinely different software would be collapsed into one row and host data lost. Assert
// that in-scope (giflib) rows differing in exactly one identity column are always left
// alone, and that the NULL/” handling for the optional columns matches ComputeRawChecksum.
func TestUp_20260810160000_DoesNotMergeDistinctSoftware(t *testing.T) {
	db := applyUpToPrev(t)

	base := map[string]any{
		"name": "giflib", "version": "1.0", "source": "homebrew_packages",
		"bundle_identifier": "com.example", "release": "1", "arch": "amd64",
		"vendor": "vendorA", "extension_for": "chrome", "extension_id": "extA",
		"application_id": "app-a", "upgrade_code": "upgrade-a",
	}
	cols := []string{
		"name", "version", "source", "bundle_identifier", "release", "arch",
		"vendor", "extension_for", "extension_id", "application_id", "upgrade_code",
	}
	insert := func(row map[string]any, checksum string) int64 {
		args := make([]any, 0, len(cols)+1)
		for _, c := range cols {
			args = append(args, row[c])
		}
		sum := md5.Sum([]byte(checksum)) //nolint:gosec // arbitrary distinct checksum
		args = append(args, sum[:])
		return execNoErrLastID(t, db,
			"INSERT INTO software (name, version, source, bundle_identifier, `release`, arch, vendor, "+
				"extension_for, extension_id, application_id, upgrade_code, checksum) "+
				"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", args...)
	}
	exists := func(id int64) bool {
		var n int
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM software WHERE id = ?`, id).Scan(&n))
		return n == 1
	}

	// One pair per identity column, each differing only in that column. Every pair stays
	// in scope (name/source from base) except the name and source variants themselves,
	// which leave it — either way the rows are distinct software and must both survive.
	// Each pair gets its own version so the pairs are in separate identity groups.
	pairs := make(map[string][2]int64, len(cols))
	for _, col := range cols {
		a := maps.Clone(base)
		a["version"] = "v-" + col
		b := maps.Clone(a)
		b[col] = a[col].(string) + "-variant"
		pairs[col] = [2]int64{insert(a, "a-"+col), insert(b, "b-"+col)}
	}

	// NULL and '' are equivalent for the optional columns, so these two ARE duplicates.
	nullish := maps.Clone(base)
	nullish["version"] = "v-nullish"
	nullish["application_id"], nullish["upgrade_code"] = nil, nil
	emptyish := maps.Clone(nullish)
	emptyish["application_id"], emptyish["upgrade_code"] = "", ""
	nullID, emptyID := insert(nullish, "nullish"), insert(emptyish, "emptyish")

	applyNext(t, db)

	for _, col := range cols {
		require.Truef(t, exists(pairs[col][0]), "rows differing only in %q must not be merged", col)
		require.Truef(t, exists(pairs[col][1]), "rows differing only in %q must not be merged", col)
	}

	var nullishRows int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM software WHERE version = 'v-nullish'`).Scan(&nullishRows))
	require.Equal(t, 1, nullishRows, "NULL and '' optional columns are the same identity and must merge")
	require.NotEqual(t, exists(nullID), exists(emptyID), "exactly one of the pair survives")
}

// A fresh install has no software at all. Both the merge and the recompute must cope with
// an empty (or giflib-free) software table rather than erroring.
func TestUp_20260810160000_EmptySoftwareTable(t *testing.T) {
	db := applyUpToPrev(t)

	execNoErr(t, db, `DELETE FROM software`)
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM software`).Scan(&count))
	require.Zero(t, count, "fixture should start with an empty software table")

	applyNext(t, db)

	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM software`).Scan(&count))
	require.Zero(t, count)
}
