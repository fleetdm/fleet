// Repair for software rows whose software_titles link is broken: preInsertSoftwareInventory
// writes a NULL title_id as a last resort, and CleanupSoftwareTitles nulls links to titles it
// deleted out from under a concurrent ingestion. Such a row is invisible to every query that
// joins software to software_titles, and ingestion never revisits it because its checksum
// already exists, so CleanupSoftwareTitles re-links it here at the start of each hourly run.

package mysql

import (
	"context"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
)

// softwareRepairCandidate is a software row whose software_titles link is broken, with
// the columns needed to match or (re)create its title.
type softwareRepairCandidate struct {
	ID               uint    `db:"id"`
	Name             string  `db:"name"`
	Source           string  `db:"source"`
	ExtensionFor     string  `db:"extension_for"`
	BundleIdentifier *string `db:"bundle_identifier"`
	ApplicationID    *string `db:"application_id"`
	UpgradeCode      *string `db:"upgrade_code"`
}

// linuxPackageSources are the sources whose software names can denote a Linux kernel
// package. Ingestion gates the same check on the host's platform, which a software row
// does not carry.
var linuxPackageSources = map[string]struct{}{
	"deb_packages":    {},
	"rpm_packages":    {},
	"pacman_packages": {},
}

// softwareTitleLinkStmts re-link a batch of broken software rows, mirroring ingestion's
// match semantics (see preInsertSoftwareInventory) and precedence: bundle identifier,
// upgrade code, name, then Android application ID — last so batches without Android rows
// can skip it, and with it a full scan of software_titles, which has no index on
// application_id. Each only touches rows that are still unlinked, so a row keeps the first
// match it gets. Name matching is left to MySQL's utf8mb4_unicode_ci collation, which is
// what enforces title uniqueness in the first place.
var softwareTitleLinkStmts = []string{
	`
	UPDATE software s
	JOIN software_titles st
		ON st.bundle_identifier = s.bundle_identifier
		AND st.source = s.source
		AND st.extension_for = s.extension_for
	SET s.title_id = st.id
	WHERE s.id IN (?) AND s.title_id IS NULL AND s.bundle_identifier IS NOT NULL AND s.bundle_identifier != ''`,
	`
	UPDATE software s
	JOIN software_titles st
		ON st.source = 'programs'
		AND st.unique_identifier = s.upgrade_code
	SET s.title_id = st.id
	WHERE s.id IN (?) AND s.title_id IS NULL AND s.source = 'programs'
		AND s.upgrade_code IS NOT NULL AND s.upgrade_code != ''`,
	`
	UPDATE software s
	JOIN software_titles st
		ON st.name = s.name
		AND st.source = s.source
		AND st.extension_for = s.extension_for
		AND st.bundle_identifier IS NULL
	SET s.title_id = st.id
	WHERE s.id IN (?) AND s.title_id IS NULL AND (s.bundle_identifier IS NULL OR s.bundle_identifier = '')`,
	`
	UPDATE software s
	JOIN software_titles st
		ON st.application_id = s.application_id
		AND st.source = s.source
	SET s.title_id = st.id
	WHERE s.id IN (?) AND s.title_id IS NULL AND s.source = 'android_apps'
		AND s.application_id IS NOT NULL AND s.application_id != ''`,
}

// repairUnlinkedSoftware re-links software rows with a NULL title_id, creating the titles
// that no longer exist. It heals them whether or not any host reports the software again,
// and returns the number of rows re-linked.
func (ds *Datastore) repairUnlinkedSoftware(ctx context.Context) (int, error) {
	// FORCE INDEX because the optimizer otherwise picks an index merge with the primary
	// key and sorts the result; the title_id index alone answers both the IS NULL filter
	// and the ordering (id is its implicit suffix), so this reads only the broken rows
	// instead of scanning the (large) software table.
	const findNullTitleSoftwareStmt = `
		SELECT id, name, source, extension_for, bundle_identifier, application_id, upgrade_code
		FROM software FORCE INDEX (title_id)
		WHERE title_id IS NULL AND id > ?
		ORDER BY id
		LIMIT ?`

	var repaired int

	// Read from the replica, as the other cleanups do: every repair statement re-checks on
	// the writer that the row is still unlinked, so a stale read is at worst wasted work.
	var lastID uint
	for range cleanupMaxIterations {
		var candidates []softwareRepairCandidate
		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &candidates, findNullTitleSoftwareStmt, lastID, cleanupBatchSize); err != nil {
			return repaired, ctxerr.Wrap(ctx, err, "find software with no title for repair")
		}
		if len(candidates) == 0 {
			break
		}
		lastID = candidates[len(candidates)-1].ID

		n, err := ds.repairSoftwareTitleLinks(ctx, candidates)
		if err != nil {
			return repaired, err
		}
		repaired += n
	}

	return repaired, nil
}

// repairSoftwareTitleLinks links a batch of broken software rows to their titles,
// creating the titles that do not exist yet. It returns the number of rows linked.
func (ds *Datastore) repairSoftwareTitleLinks(ctx context.Context, candidates []softwareRepairCandidate) (int, error) {
	if len(candidates) == 0 {
		return 0, nil
	}

	if err := ds.insertMissingSoftwareTitles(ctx, candidates); err != nil {
		return 0, err
	}

	ids := make([]uint, 0, len(candidates))
	hasAndroid := false
	for _, c := range candidates {
		ids = append(ids, c.ID)
		hasAndroid = hasAndroid || c.Source == "android_apps"
	}

	linkStmts := softwareTitleLinkStmts
	if !hasAndroid {
		linkStmts = linkStmts[:len(linkStmts)-1]
	}
	for _, linkStmt := range linkStmts {
		stmt, args, err := sqlx.In(linkStmt, ids)
		if err != nil {
			return 0, ctxerr.Wrap(ctx, err, "build link software to title query")
		}
		if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
			return 0, ctxerr.Wrap(ctx, err, "link software to title")
		}
	}

	stmt, args, err := sqlx.In(`SELECT COUNT(*) FROM software WHERE id IN (?) AND title_id IS NULL`, ids)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "build count unlinked software query")
	}
	var remaining int
	if err := sqlx.GetContext(ctx, ds.writer(ctx), &remaining, stmt, args...); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "count unlinked software")
	}
	if remaining > 0 && ds.logger != nil {
		ds.logger.WarnContext(ctx, "could not link software to a software title",
			"count", remaining,
		)
	}
	return len(ids) - remaining, nil
}

// insertMissingSoftwareTitles creates the titles the given software rows need, built from
// the rows themselves. Creating unconditionally is safe: when a matching title already
// exists, the INSERT IGNORE collides with it on software_titles' unique indexes — whose
// unique_identifier coalesces the same identities the link statements match on — so linking
// afterwards cannot pick up a duplicate.
func (ds *Datastore) insertMissingSoftwareTitles(ctx context.Context, candidates []softwareRepairCandidate) error {
	var (
		values []string
		args   []any
	)
	seen := make(map[string]struct{}, len(candidates))
	for _, c := range candidates {
		var bundleID *string
		if ptr.ValOrZero(c.BundleIdentifier) != "" {
			bundleID = c.BundleIdentifier
		}
		// An empty application_id has to become NULL: unique_identifier coalesces to it
		// ahead of the name, so empty strings would all collide. An empty upgrade_code is
		// passed through, as ingestion does, because unique_identifier NULLIFs it.
		var applicationID *string
		if ptr.ValOrZero(c.ApplicationID) != "" {
			applicationID = c.ApplicationID
		}
		var isKernel bool
		if _, ok := linuxPackageSources[c.Source]; ok {
			isKernel = fleet.IsKernelSoftwareName(c.Name)
		}

		key := UniqueSoftwareTitleStr(c.Name, c.Source, c.ExtensionFor, ptr.ValOrZero(bundleID),
			strconv.FormatBool(isKernel))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		values = append(values, "(?,?,?,?,?,?,?)")
		args = append(args, c.Name, c.Source, c.ExtensionFor, bundleID, isKernel, applicationID, c.UpgradeCode)
	}
	if len(values) == 0 {
		return nil
	}

	stmt := `INSERT IGNORE INTO software_titles (name, source, extension_for, bundle_identifier, is_kernel, application_id, upgrade_code) VALUES ` +
		strings.Join(values, ",")
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "insert missing software titles")
	}
	return nil
}

// nullDanglingTitleLinks clears software.title_id for rows pointing at any of the given
// titles that no longer exist. The join guards titles that survived the orphan re-check.
func (ds *Datastore) nullDanglingTitleLinks(ctx context.Context, titleIDs []uint) error {
	const nullDanglingStmt = `
		UPDATE software s
		LEFT JOIN software_titles st ON st.id = s.title_id
		SET s.title_id = NULL
		WHERE s.title_id IN (?) AND st.id IS NULL`
	stmt, args, err := sqlx.In(nullDanglingStmt, titleIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build null dangling software title links query")
	}
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "null dangling software title links")
	}
	return nil
}
