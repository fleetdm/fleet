package mysql

import (
	"context"
	"crypto/md5" //nolint:gosec // This hash is used as a DB optimization for software row lookup, not security
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type softwareIDChecksum struct {
	ID       uint   `db:"id"`
	Checksum string `db:"checksum"`
}

// Since DB may have millions of software items, we need to batch the aggregation counts to avoid long SQL query times.
// This is a variable so it can be adjusted during unit testing.
var countHostSoftwareBatchSize = uint64(100000)

// Since a host may have a lot of software items, we need to batch the inserts.
// The maximum number of software items we can insert at one time is governed by max_allowed_packet, which already be set to a high value for MDM bootstrap packages,
// and by the maximum number of placeholders in a prepared statement, which is 65,536. These are already fairly large limits.
// This is a variable, so it can be adjusted during unit testing.
var softwareInsertBatchSize = 1000

func softwareSliceToMap(softwareItems []fleet.Software) map[string]fleet.Software {
	result := make(map[string]fleet.Software, len(softwareItems))
	for _, s := range softwareItems {
		result[s.ToUniqueStr()] = s
	}
	return result
}

func (ds *Datastore) UpdateHostSoftware(ctx context.Context, hostID uint, software []fleet.Software) (*fleet.UpdateHostSoftwareDBResult, error) {
	return ds.applyChangesForNewSoftwareDB(ctx, hostID, software)
}

func (ds *Datastore) UpdateHostSoftwareInstalledPaths(
	ctx context.Context,
	hostID uint,
	reported map[string]struct{},
	mutationResults *fleet.UpdateHostSoftwareDBResult,
) error {
	currS := mutationResults.CurrInstalled()

	hsip, err := ds.getHostSoftwareInstalledPaths(ctx, hostID)
	if err != nil {
		return err
	}

	toI, toD, err := hostSoftwareInstalledPathsDelta(hostID, reported, hsip, currS)
	if err != nil {
		return err
	}

	if len(toI) == 0 && len(toD) == 0 {
		// Nothing to do ...
		return nil
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if err := deleteHostSoftwareInstalledPaths(ctx, tx, toD); err != nil {
			return err
		}

		if err := insertHostSoftwareInstalledPaths(ctx, tx, toI); err != nil {
			return err
		}

		return nil
	})
}

// getHostSoftwareInstalledPaths returns all HostSoftwareInstalledPath for the given hostID.
func (ds *Datastore) getHostSoftwareInstalledPaths(
	ctx context.Context,
	hostID uint,
) (
	[]fleet.HostSoftwareInstalledPath,
	error,
) {
	stmt := `
		SELECT t.id, t.host_id, t.software_id, t.installed_path, t.team_identifier
		FROM host_software_installed_paths t
		WHERE t.host_id = ?
	`

	var result []fleet.HostSoftwareInstalledPath
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &result, stmt, hostID); err != nil {
		return nil, err
	}

	return result, nil
}

// hostSoftwareInstalledPathsDelta returns what should be inserted and deleted to keep the
// 'host_software_installed_paths' table in-sync with the osquery reported query results.
// 'reported' is a set of 'installed_path-software.UniqueStr' strings, built from the osquery
// results.
// 'stored' contains all 'host_software_installed_paths' rows for the given host.
// 'hostSoftware' contains the current software installed on the host.
func hostSoftwareInstalledPathsDelta(
	hostID uint,
	reported map[string]struct{},
	stored []fleet.HostSoftwareInstalledPath,
	hostSoftware []fleet.Software,
) (
	toInsert []fleet.HostSoftwareInstalledPath,
	toDelete []uint,
	err error,
) {
	if len(reported) != 0 && len(hostSoftware) == 0 {
		// Error condition, something reported implies that the host has some software
		err = fmt.Errorf("software installed paths for host %d were reported but host contains no software", hostID)
		return
	}

	sIDLookup := map[uint]fleet.Software{}
	for _, s := range hostSoftware {
		sIDLookup[s.ID] = s
	}

	sUnqStrLook := map[string]fleet.Software{}
	for _, s := range hostSoftware {
		sUnqStrLook[s.ToUniqueStr()] = s
	}

	iSPathLookup := make(map[string]fleet.HostSoftwareInstalledPath)
	for _, r := range stored {
		s, ok := sIDLookup[r.SoftwareID]
		// Software currently not found on the host, should be deleted ...
		if !ok {
			toDelete = append(toDelete, r.ID)
			continue
		}

		key := fmt.Sprintf(
			"%s%s%s%s%s",
			r.InstalledPath, fleet.SoftwareFieldSeparator, r.TeamIdentifier, fleet.SoftwareFieldSeparator, s.ToUniqueStr(),
		)
		iSPathLookup[key] = r

		// Anything stored but not reported should be deleted
		if _, ok := reported[key]; !ok {
			toDelete = append(toDelete, r.ID)
		}
	}

	for key := range reported {
		parts := strings.SplitN(key, fleet.SoftwareFieldSeparator, 3)
		installedPath, teamIdentifier, unqStr := parts[0], parts[1], parts[2]

		// Shouldn't be possible ... everything 'reported' should be in the the software table
		// because this executes after 'ds.UpdateHostSoftware'
		s, ok := sUnqStrLook[unqStr]
		if !ok {
			err = fmt.Errorf("reported installed path for %s does not belong to any stored software entry", unqStr)
			return
		}

		if _, ok := iSPathLookup[key]; ok {
			// Nothing to do
			continue
		}

		toInsert = append(toInsert, fleet.HostSoftwareInstalledPath{
			HostID:         hostID,
			SoftwareID:     s.ID,
			InstalledPath:  installedPath,
			TeamIdentifier: teamIdentifier,
		})
	}

	return
}

func deleteHostSoftwareInstalledPaths(
	ctx context.Context,
	tx sqlx.ExtContext,
	toDelete []uint,
) error {
	if len(toDelete) == 0 {
		return nil
	}

	stmt := `DELETE FROM host_software_installed_paths WHERE id IN (?)`
	stmt, args, err := sqlx.In(stmt, toDelete)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building delete statement for delete host_software_installed_paths")
	}
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "executing delete statement for delete host_software_installed_paths")
	}

	return nil
}

func insertHostSoftwareInstalledPaths(
	ctx context.Context,
	tx sqlx.ExtContext,
	toInsert []fleet.HostSoftwareInstalledPath,
) error {
	if len(toInsert) == 0 {
		return nil
	}

	stmt := "INSERT INTO host_software_installed_paths (host_id, software_id, installed_path, team_identifier) VALUES %s"
	batchSize := 500

	for i := 0; i < len(toInsert); i += batchSize {
		end := i + batchSize
		if end > len(toInsert) {
			end = len(toInsert)
		}
		batch := toInsert[i:end]

		var args []interface{}
		for _, v := range batch {
			args = append(args, v.HostID, v.SoftwareID, v.InstalledPath, v.TeamIdentifier)
		}

		placeHolders := strings.TrimSuffix(strings.Repeat("(?, ?, ?, ?), ", len(batch)), ", ")
		stmt := fmt.Sprintf(stmt, placeHolders)

		_, err := tx.ExecContext(ctx, stmt, args...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "inserting rows into host_software_installed_paths")
		}
	}

	return nil
}

func nothingChanged(current, incoming []fleet.Software, minLastOpenedAtDiff time.Duration) (
	map[string]fleet.Software, map[string]fleet.Software, bool,
) {
	// Process incoming software to ensure there are no duplicates, since the same software can be installed at multiple paths.
	incomingMap := make(map[string]fleet.Software, len(current)) // setting len(current) as the length since that should be the common case
	for _, s := range incoming {
		uniqueStr := s.ToUniqueStr()
		if duplicate, ok := incomingMap[uniqueStr]; ok {
			// Check the last opened at timestamp and keep the latest.
			if s.LastOpenedAt == nil ||
				(duplicate.LastOpenedAt != nil && !s.LastOpenedAt.After(*duplicate.LastOpenedAt)) {
				continue // keep the duplicate
			}
		}
		incomingMap[uniqueStr] = s
	}
	currentMap := softwareSliceToMap(current)
	if len(currentMap) != len(incomingMap) {
		return currentMap, incomingMap, false
	}

	for _, s := range incomingMap {
		cur, ok := currentMap[s.ToUniqueStr()]
		if !ok {
			return currentMap, incomingMap, false
		}

		// if the incoming software has a last opened at timestamp and it differs
		// significantly from the current timestamp (or there is no current
		// timestamp), then consider that something changed.
		if s.LastOpenedAt != nil {
			if cur.LastOpenedAt == nil {
				return currentMap, incomingMap, false
			}

			oldLast := *cur.LastOpenedAt
			newLast := *s.LastOpenedAt
			if newLast.Sub(oldLast) >= minLastOpenedAtDiff {
				return currentMap, incomingMap, false
			}
		}
	}

	return currentMap, incomingMap, true
}

func (ds *Datastore) ListSoftwareByHostIDShort(ctx context.Context, hostID uint) ([]fleet.Software, error) {
	return listSoftwareByHostIDShort(ctx, ds.reader(ctx), hostID)
}

func listSoftwareByHostIDShort(
	ctx context.Context,
	db sqlx.QueryerContext,
	hostID uint,
) ([]fleet.Software, error) {
	q := `
SELECT
    s.id,
    s.name,
    s.version,
    s.source,
    s.browser,
    s.bundle_identifier,
    s.release,
    s.vendor,
    s.arch,
    s.extension_id,
    hs.last_opened_at
FROM
    software s
    JOIN host_software hs ON hs.software_id = s.id
WHERE
    hs.host_id = ?
`
	var softwares []fleet.Software
	err := sqlx.SelectContext(ctx, db, &softwares, q, hostID)
	if err != nil {
		return nil, err
	}

	return softwares, nil
}

// applyChangesForNewSoftwareDB returns the current host software and the applied mutations: what
// was inserted and what was deleted
func (ds *Datastore) applyChangesForNewSoftwareDB(
	ctx context.Context,
	hostID uint,
	software []fleet.Software,
) (*fleet.UpdateHostSoftwareDBResult, error) {
	r := &fleet.UpdateHostSoftwareDBResult{}

	// This code executes once an hour for each host, so we should optimize for MySQL master (writer) DB performance.
	// We use a slave (reader) DB to avoid accessing the master. If nothing has changed, we avoid all access to the master.
	// It is possible that the software list is out of sync between the slave and the master. This is unlikely because
	// it is updated once an hour under normal circumstances. If this does occur, the software list will be updated
	// once again in an hour.
	currentSoftware, err := listSoftwareByHostIDShort(ctx, ds.reader(ctx), hostID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "loading current software for host")
	}
	r.WasCurrInstalled = currentSoftware

	current, incoming, notChanged := nothingChanged(currentSoftware, software, ds.minLastOpenedAtDiff)
	if notChanged {
		return r, nil
	}

	existingSoftware, incomingByChecksum, existingTitlesForNewSoftware, err := ds.getExistingSoftware(ctx, current, incoming)
	if err != nil {
		return r, err
	}

	err = ds.withRetryTxx(
		ctx, func(tx sqlx.ExtContext) error {
			deleted, err := deleteUninstalledHostSoftwareDB(ctx, tx, hostID, current, incoming)
			if err != nil {
				return err
			}
			r.Deleted = deleted

			inserted, err := ds.insertNewInstalledHostSoftwareDB(
				ctx, tx, hostID, existingSoftware, incomingByChecksum, existingTitlesForNewSoftware,
			)
			if err != nil {
				return err
			}
			r.Inserted = inserted

			if err = checkForDeletedInstalledSoftware(ctx, tx, deleted, inserted, hostID); err != nil {
				return err
			}

			if err = updateModifiedHostSoftwareDB(ctx, tx, hostID, current, incoming, ds.minLastOpenedAtDiff); err != nil {
				return err
			}

			if err = updateSoftwareUpdatedAt(ctx, tx, hostID); err != nil {
				return err
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	return r, err
}

func checkForDeletedInstalledSoftware(ctx context.Context, tx sqlx.ExtContext, deleted []fleet.Software, inserted []fleet.Software,
	hostID uint,
) error {
	// Between deleted and inserted software, check which software titles were deleted.
	// If software titles were deleted, get the software titles of the installed software.
	// See if deleted titles match installed software titles.
	// If so, mark the installed software as removed.
	var deletedTitles map[string]struct{}
	if len(deleted) > 0 {
		deletedTitles = make(map[string]struct{}, len(deleted))
		for _, d := range deleted {
			// We don't support installing browser plugins as of 2024/08/22
			if d.Browser == "" {
				deletedTitles[UniqueSoftwareTitleStr(d.Name, d.Source, d.BundleIdentifier)] = struct{}{}
			}
		}
		for _, i := range inserted {
			// We don't support installing browser plugins as of 2024/08/22
			if i.Browser == "" {
				key := UniqueSoftwareTitleStr(i.Name, i.Source, i.BundleIdentifier)
				delete(deletedTitles, key)
			}
		}
	}
	if len(deletedTitles) > 0 {
		installedTitles, err := getInstalledByFleetSoftwareTitles(ctx, tx, hostID)
		if err != nil {
			return err
		}
		type deletedValue struct {
			vpp bool
		}
		deletedTitleIDs := make(map[uint]deletedValue, 0)
		for _, title := range installedTitles {
			bundleIdentifier := ""
			if title.BundleIdentifier != nil {
				bundleIdentifier = *title.BundleIdentifier
			}
			key := UniqueSoftwareTitleStr(title.Name, title.Source, bundleIdentifier)
			if _, ok := deletedTitles[key]; ok {
				deletedTitleIDs[title.ID] = deletedValue{vpp: title.VPPAppsCount > 0}
			}
		}
		if len(deletedTitleIDs) > 0 {
			IDs := make([]uint, 0, len(deletedTitleIDs))
			vppIDs := make([]uint, 0, len(deletedTitleIDs))
			for id, value := range deletedTitleIDs {
				if value.vpp {
					vppIDs = append(vppIDs, id)
				} else {
					IDs = append(IDs, id)
				}
			}
			if len(IDs) > 0 {
				if err = markHostSoftwareInstallsRemoved(ctx, tx, hostID, IDs); err != nil {
					return err
				}
			}
			if len(vppIDs) > 0 {
				if err = markHostVPPSoftwareInstallsRemoved(ctx, tx, hostID, vppIDs); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (ds *Datastore) getExistingSoftware(
	ctx context.Context, current map[string]fleet.Software, incoming map[string]fleet.Software,
) (
	currentSoftware []softwareIDChecksum, incomingChecksumToSoftware map[string]fleet.Software,
	incomingChecksumToTitle map[string]fleet.SoftwareTitle, err error,
) {
	// Compute checksums for all incoming software, which we will use for faster retrieval, since checksum is a unique index
	incomingChecksumToSoftware = make(map[string]fleet.Software, len(current))
	newSoftware := make(map[string]struct{})
	for uniqueName, s := range incoming {
		if _, ok := current[uniqueName]; !ok {
			checksum, err := computeRawChecksum(s)
			if err != nil {
				return nil, nil, nil, err
			}
			incomingChecksumToSoftware[string(checksum)] = s
			newSoftware[string(checksum)] = struct{}{}
		}
	}

	if len(incomingChecksumToSoftware) > 0 {
		keys := make([]string, 0, len(incomingChecksumToSoftware))
		for checksum := range incomingChecksumToSoftware {
			keys = append(keys, checksum)
		}
		// We use the replica DB for retrieval to minimize the traffic to the master DB.
		// It is OK if the software is not found in the replica DB, because we will then attempt to insert it in the master DB.
		currentSoftware, err = getSoftwareIDsByChecksums(ctx, ds.reader(ctx), keys)
		if err != nil {
			return nil, nil, nil, err
		}
		for _, s := range currentSoftware {
			_, ok := incomingChecksumToSoftware[s.Checksum]
			if !ok {
				// This should never happen. If it does, we have a bug.
				return nil, nil, nil, ctxerr.New(
					ctx, fmt.Sprintf("software not found for checksum %s", hex.EncodeToString([]byte(s.Checksum))),
				)
			}
			delete(newSoftware, s.Checksum)
		}
	}

	// Get software titles for new software, if any
	incomingChecksumToTitle = make(map[string]fleet.SoftwareTitle, len(newSoftware))
	if len(newSoftware) > 0 {
		totalToProcess := len(newSoftware)
		const numberOfArgsPerSoftwareTitle = 4 // number of ? in each WHERE clause
		whereClause := strings.TrimSuffix(
			strings.Repeat(`
			  (
			    (bundle_identifier = ?) OR
			    (name = ? AND source = ? AND browser = ? AND bundle_identifier IS NULL)
			  ) OR`, totalToProcess), " OR",
		)
		stmt := fmt.Sprintf(
			"SELECT id, name, source, browser, COALESCE(bundle_identifier, '') as bundle_identifier FROM software_titles WHERE %s",
			whereClause,
		)
		args := make([]interface{}, 0, totalToProcess*numberOfArgsPerSoftwareTitle)
		uniqueTitleStrToChecksum := make(map[string]string, totalToProcess)
		for checksum := range newSoftware {
			sw := incomingChecksumToSoftware[checksum]
			args = append(args, sw.BundleIdentifier, sw.Name, sw.Source, sw.Browser)
			// Map software title identifier to software checksums so that we can map checksums to actual titles later.
			uniqueTitleStrToChecksum[UniqueSoftwareTitleStr(sw.Name, sw.Source, sw.Browser)] = checksum
		}
		var existingSoftwareTitlesForNewSoftware []fleet.SoftwareTitle
		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &existingSoftwareTitlesForNewSoftware, stmt, args...); err != nil {
			return nil, nil, nil, ctxerr.Wrap(ctx, err, "get existing titles")
		}

		// Map software titles to software checksums.
		for _, title := range existingSoftwareTitlesForNewSoftware {
			checksum, ok := uniqueTitleStrToChecksum[UniqueSoftwareTitleStr(title.Name, title.Source, title.Browser)]
			if ok {
				incomingChecksumToTitle[checksum] = title
			}
		}
	}

	return currentSoftware, incomingChecksumToSoftware, incomingChecksumToTitle, nil
}

// UniqueSoftwareTitleStr creates a unique string representation of the software title
func UniqueSoftwareTitleStr(values ...string) string {
	return strings.Join(values, fleet.SoftwareFieldSeparator)
}

// delete host_software that is in current map, but not in incoming map.
// returns the deleted software on the host
func deleteUninstalledHostSoftwareDB(
	ctx context.Context,
	tx sqlx.ExecerContext,
	hostID uint,
	currentMap map[string]fleet.Software,
	incomingMap map[string]fleet.Software,
) ([]fleet.Software, error) {
	var deletesHostSoftwareIDs []uint
	var deletedSoftware []fleet.Software

	for currentKey, curSw := range currentMap {
		if _, ok := incomingMap[currentKey]; !ok {
			deletedSoftware = append(deletedSoftware, curSw)
			deletesHostSoftwareIDs = append(deletesHostSoftwareIDs, curSw.ID)
		}
	}
	if len(deletesHostSoftwareIDs) == 0 {
		return nil, nil
	}

	stmt := `DELETE FROM host_software WHERE host_id = ? AND software_id IN (?);`
	stmt, args, err := sqlx.In(stmt, hostID, deletesHostSoftwareIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build delete host software query")
	}
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "delete host software")
	}

	return deletedSoftware, nil
}

// computeRawChecksum computes the checksum for a software entry
// The calculation must match the one in softwareChecksumComputedColumn
func computeRawChecksum(sw fleet.Software) ([]byte, error) {
	h := md5.New() //nolint:gosec // This hash is used as a DB optimization for software row lookup, not security
	cols := []string{sw.Name, sw.Version, sw.Source, sw.BundleIdentifier, sw.Release, sw.Arch, sw.Vendor, sw.Browser, sw.ExtensionID}
	_, err := fmt.Fprint(h, strings.Join(cols, "\x00"))
	if err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

// Insert host_software that is in softwareChecksums map, but not in existingSoftware.
// Also insert any new software titles that are needed.
// returns the inserted software on the host
func (ds *Datastore) insertNewInstalledHostSoftwareDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	hostID uint,
	existingSoftware []softwareIDChecksum,
	softwareChecksums map[string]fleet.Software,
	existingTitlesForNewSoftware map[string]fleet.SoftwareTitle,
) ([]fleet.Software, error) {
	var insertsHostSoftware []interface{}
	var insertedSoftware []fleet.Software

	// First, we remove incoming software that already exists in the software table.
	if len(softwareChecksums) > 0 {
		for _, s := range existingSoftware {
			software, ok := softwareChecksums[s.Checksum]
			if !ok {
				return nil, ctxerr.New(ctx, fmt.Sprintf("software not found for checksum %s", hex.EncodeToString([]byte(s.Checksum))))
			}
			software.ID = s.ID
			insertsHostSoftware = append(insertsHostSoftware, hostID, software.ID, software.LastOpenedAt)
			insertedSoftware = append(insertedSoftware, software)
			delete(softwareChecksums, s.Checksum)
		}
	}

	// For software items that don't already exist in the software table, we insert them.
	if len(softwareChecksums) > 0 {
		keys := make([]string, 0, len(softwareChecksums))
		for checksum := range softwareChecksums {
			keys = append(keys, checksum)
		}
		for i := 0; i < len(keys); i += softwareInsertBatchSize {
			start := i
			end := i + softwareInsertBatchSize
			if end > len(keys) {
				end = len(keys)
			}
			totalToProcess := end - start

			// Insert into software
			const numberOfArgsPerSoftware = 11 // number of ? in each VALUES clause
			values := strings.TrimSuffix(
				strings.Repeat("(?,?,?,?,?,?,?,?,?,?,?),", totalToProcess), ",",
			)
			// INSERT IGNORE is used to avoid duplicate key errors, which may occur since our previous read came from the replica.
			stmt := fmt.Sprintf(
				`INSERT IGNORE INTO software (
					name,
					version,
					source,
					`+"`release`"+`,
					vendor,
					arch,
					bundle_identifier,
					extension_id,
					browser,
					title_id,
					checksum
				) VALUES %s`,
				values,
			)
			args := make([]interface{}, 0, totalToProcess*numberOfArgsPerSoftware)
			newTitlesNeeded := make(map[string]fleet.SoftwareTitle)
			for j := start; j < end; j++ {
				checksum := keys[j]
				sw := softwareChecksums[checksum]
				var titleID *uint
				title, ok := existingTitlesForNewSoftware[checksum]
				if ok {
					titleID = &title.ID
				} else if _, ok := newTitlesNeeded[checksum]; !ok {
					st := fleet.SoftwareTitle{
						Name:    sw.Name,
						Source:  sw.Source,
						Browser: sw.Browser,
					}

					if sw.BundleIdentifier != "" {
						st.BundleIdentifier = ptr.String(sw.BundleIdentifier)
					}

					newTitlesNeeded[checksum] = st
				}
				args = append(
					args, sw.Name, sw.Version, sw.Source, sw.Release, sw.Vendor, sw.Arch, sw.BundleIdentifier, sw.ExtensionID, sw.Browser,
					titleID, checksum,
				)
			}
			if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "insert software")
			}

			// Insert into software_titles
			totalTitlesToProcess := len(newTitlesNeeded)
			if totalTitlesToProcess > 0 {
				const numberOfArgsPerSoftwareTitles = 4 // number of ? in each VALUES clause
				titlesValues := strings.TrimSuffix(strings.Repeat("(?,?,?,?),", totalTitlesToProcess), ",")
				// INSERT IGNORE is used to avoid duplicate key errors, which may occur since our previous read came from the replica.
				titlesStmt := fmt.Sprintf("INSERT IGNORE INTO software_titles (name, source, browser, bundle_identifier) VALUES %s", titlesValues)
				titlesArgs := make([]interface{}, 0, totalTitlesToProcess*numberOfArgsPerSoftwareTitles)
				titleChecksums := make([]string, 0, totalTitlesToProcess)
				for checksum, title := range newTitlesNeeded {
					titlesArgs = append(titlesArgs, title.Name, title.Source, title.Browser, title.BundleIdentifier)
					titleChecksums = append(titleChecksums, checksum)
				}
				if _, err := tx.ExecContext(ctx, titlesStmt, titlesArgs...); err != nil {
					return nil, ctxerr.Wrap(ctx, err, "insert software_titles")
				}

				updateSoftwareWithoutIdentifierStmt := `
				    UPDATE software s
				    JOIN software_titles st
				    ON COALESCE(s.bundle_identifier, '') = '' AND s.name = st.name AND s.source = st.source AND s.browser = st.browser
				    SET s.title_id = st.id
				    WHERE (s.title_id IS NULL OR s.title_id != st.id)
				    AND COALESCE(s.bundle_identifier, '') = ''
				    AND s.checksum IN (?)
				    `
				updateSoftwareWithoutIdentifierStmt, updateArgs, err := sqlx.In(updateSoftwareWithoutIdentifierStmt, titleChecksums)
				if err != nil {
					return nil, ctxerr.Wrap(ctx, err, "build update software title_id without identifier")
				}
				if _, err = tx.ExecContext(ctx, updateSoftwareWithoutIdentifierStmt, updateArgs...); err != nil {
					return nil, ctxerr.Wrap(ctx, err, "update software title_id without identifier")
				}

				// update new title ids for new software table entries
				updateSoftwareStmt := `
				      UPDATE software s
				      JOIN software_titles st
				      ON s.bundle_identifier = st.bundle_identifier AND
				          IF(s.source IN ('apps', 'ios_apps', 'ipados_apps'), s.source = st.source, 1)
				      SET s.title_id = st.id
				      WHERE s.title_id IS NULL
				      OR s.title_id != st.id
				      AND s.checksum IN (?)`
				updateSoftwareStmt, updateArgs, err = sqlx.In(updateSoftwareStmt, titleChecksums)
				if err != nil {
					return nil, ctxerr.Wrap(ctx, err, "build update software title_id with identifier")
				}
				if _, err = tx.ExecContext(ctx, updateSoftwareStmt, updateArgs...); err != nil {
					return nil, ctxerr.Wrap(ctx, err, "update software title_id with identifier")
				}
			}
		}

		// Here, we use the transaction (tx) for retrieval because we must retrieve the software IDs that we just inserted.
		updatedExistingSoftware, err := getSoftwareIDsByChecksums(ctx, tx, keys)
		if err != nil {
			return nil, err
		}
		for _, s := range updatedExistingSoftware {
			software, ok := softwareChecksums[s.Checksum]
			if !ok {
				return nil, ctxerr.New(ctx, fmt.Sprintf("software not found for checksum %s", hex.EncodeToString([]byte(s.Checksum))))
			}
			software.ID = s.ID
			insertsHostSoftware = append(insertsHostSoftware, hostID, software.ID, software.LastOpenedAt)
			insertedSoftware = append(insertedSoftware, software)
			delete(softwareChecksums, s.Checksum)
		}
	}

	if len(softwareChecksums) > 0 {
		// We log and continue. We should almost never see this error. If we see it regularly, we need to investigate.
		level.Error(ds.logger).Log(
			"msg", "could not find or create software items. This error may be caused by master and replica DBs out of sync.", "host_id",
			hostID, "number", len(softwareChecksums),
		)
		for checksum, software := range softwareChecksums {
			uuidString := ""
			checksumAsUUID, err := uuid.FromBytes([]byte(checksum))
			if err == nil {
				// We ignore error
				uuidString = checksumAsUUID.String()
			}
			level.Debug(ds.logger).Log(
				"msg", "software item not found or created", "name", software.Name, "version", software.Version, "source", software.Source,
				"bundle_identifier", software.BundleIdentifier, "checksum", uuidString,
			)
		}
	}

	if len(insertsHostSoftware) > 0 {
		values := strings.TrimSuffix(strings.Repeat("(?,?,?),", len(insertsHostSoftware)/3), ",")
		sql := fmt.Sprintf(`INSERT IGNORE INTO host_software (host_id, software_id, last_opened_at) VALUES %s`, values)
		if _, err := tx.ExecContext(ctx, sql, insertsHostSoftware...); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "insert host software")
		}
	}

	return insertedSoftware, nil
}

func getSoftwareIDsByChecksums(ctx context.Context, tx sqlx.QueryerContext, checksums []string) ([]softwareIDChecksum, error) {
	// get existing software ids for checksums
	stmt, args, err := sqlx.In("SELECT id, checksum FROM software WHERE checksum IN (?)", checksums)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build select software query")
	}
	var existingSoftware []softwareIDChecksum
	if err = sqlx.SelectContext(ctx, tx, &existingSoftware, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get existing software")
	}
	return existingSoftware, nil
}

// update host_software when incoming software has a significantly more recent
// last opened timestamp (or didn't have on in currentMap). Note that it only
// processes software that is in both current and incoming maps, as the case
// where it is only in incoming is already handled by
// insertNewInstalledHostSoftwareDB.
func updateModifiedHostSoftwareDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	hostID uint,
	currentMap map[string]fleet.Software,
	incomingMap map[string]fleet.Software,
	minLastOpenedAtDiff time.Duration,
) error {
	var keysToUpdate []string
	for key, newSw := range incomingMap {
		curSw, ok := currentMap[key]
		if !ok || newSw.LastOpenedAt == nil {
			// software must also exist in current map, and new software must have a
			// last opened at timestamp (otherwise we don't overwrite the old one)
			continue
		}

		if curSw.LastOpenedAt == nil || newSw.LastOpenedAt.Sub(*curSw.LastOpenedAt) >= minLastOpenedAtDiff {
			keysToUpdate = append(keysToUpdate, key)
		}
	}
	sort.Strings(keysToUpdate)

	for i := 0; i < len(keysToUpdate); i += softwareInsertBatchSize {
		start := i
		end := i + softwareInsertBatchSize
		if end > len(keysToUpdate) {
			end = len(keysToUpdate)
		}
		totalToProcess := end - start

		const numberOfArgsPerSoftware = 3 // number of ? in each UPDATE
		// Using UNION ALL (instead of UNION) because it is faster since it does not check for duplicates.
		values := strings.TrimSuffix(
			strings.Repeat(" SELECT ? as host_id, ? as software_id, ? as last_opened_at UNION ALL", totalToProcess), "UNION ALL",
		)
		stmt := fmt.Sprintf(
			`UPDATE host_software hs JOIN (%s) a ON hs.host_id = a.host_id AND hs.software_id = a.software_id SET hs.last_opened_at = a.last_opened_at`,
			values,
		)

		args := make([]interface{}, 0, totalToProcess*numberOfArgsPerSoftware)
		for j := start; j < end; j++ {
			key := keysToUpdate[j]
			curSw, newSw := currentMap[key], incomingMap[key]
			args = append(args, hostID, curSw.ID, newSw.LastOpenedAt)
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "update host software")
		}
	}

	return nil
}

func updateSoftwareUpdatedAt(
	ctx context.Context,
	tx sqlx.ExtContext,
	hostID uint,
) error {
	const stmt = `INSERT INTO host_updates(host_id, software_updated_at) VALUES (?, CURRENT_TIMESTAMP) ON DUPLICATE KEY UPDATE software_updated_at=VALUES(software_updated_at)`

	if _, err := tx.ExecContext(ctx, stmt, hostID); err != nil {
		return ctxerr.Wrap(ctx, err, "update host updates")
	}

	return nil
}

var dialect = goqu.Dialect("mysql")

// listSoftwareDB returns software installed on hosts. Use opts for pagination, filtering, and controlling
// fields populated in the returned software.
func listSoftwareDB(
	ctx context.Context,
	q sqlx.QueryerContext,
	opts fleet.SoftwareListOptions,
) ([]fleet.Software, error) {
	sql, args, err := selectSoftwareSQL(opts)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "sql build")
	}

	var results []softwareCVE
	if err := sqlx.SelectContext(ctx, q, &results, sql, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select host software")
	}

	var softwares []fleet.Software
	ids := make(map[uint]int) // map of ids to index into softwares
	for _, result := range results {
		result := result // create a copy because we need to take the address to fields below

		idx, ok := ids[result.ID]
		if !ok {
			idx = len(softwares)
			softwares = append(softwares, result.Software)
			ids[result.ID] = idx
		}

		// handle null cve from left join
		if result.CVE != nil {
			cveID := *result.CVE
			cve := fleet.CVE{
				CVE:         cveID,
				DetailsLink: fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", cveID),
				CreatedAt:   *result.CreatedAt,
			}
			if opts.IncludeCVEScores {
				cve.CVSSScore = &result.CVSSScore
				cve.EPSSProbability = &result.EPSSProbability
				cve.CISAKnownExploit = &result.CISAKnownExploit
				cve.CVEPublished = &result.CVEPublished
				cve.Description = &result.Description
				cve.ResolvedInVersion = &result.ResolvedInVersion
			}
			softwares[idx].Vulnerabilities = append(softwares[idx].Vulnerabilities, cve)
		}
	}

	return softwares, nil
}

// softwareCVE is used for left joins with cve
//
//

type softwareCVE struct {
	fleet.Software

	// CVE is the CVE identifier pulled from the NVD json (e.g. CVE-2019-1234)
	CVE *string `db:"cve"`

	// CVSSScore is the CVSS score pulled from the NVD json (premium only)
	CVSSScore *float64 `db:"cvss_score"`

	// EPSSProbability is the EPSS probability pulled from FIRST (premium only)
	EPSSProbability *float64 `db:"epss_probability"`

	// CISAKnownExploit is the CISAKnownExploit pulled from CISA (premium only)
	CISAKnownExploit *bool `db:"cisa_known_exploit"`

	// CVEPublished is the CVE published date pulled from the NVD json (premium only)
	CVEPublished *time.Time `db:"cve_published"`

	// Description is the CVE description field pulled from the NVD json
	Description *string `db:"description"`

	// ResolvedInVersion is the version of software where the CVE is no longer applicable.
	// This is pulled from the versionEndExcluding field in the NVD json
	ResolvedInVersion *string `db:"resolved_in_version"`

	// CreatedAt is the time the software vulnerability was created
	CreatedAt *time.Time `db:"created_at"`
}

func selectSoftwareSQL(opts fleet.SoftwareListOptions) (string, []interface{}, error) {
	ds := dialect.
		From(goqu.I("software").As("s")).
		Select(
			"s.id",
			"s.name",
			"s.version",
			"s.source",
			"s.bundle_identifier",
			"s.extension_id",
			"s.browser",
			"s.release",
			"s.vendor",
			"s.arch",
			goqu.I("scp.cpe").As("generated_cpe"),
		).
		// Include this in the sub-query in case we want to sort by 'generated_cpe'
		LeftJoin(
			goqu.I("software_cpe").As("scp"),
			goqu.On(
				goqu.I("s.id").Eq(goqu.I("scp.software_id")),
			),
		)

	if opts.HostID != nil {
		ds = ds.
			Join(
				goqu.I("host_software").As("hs"),
				goqu.On(
					goqu.I("hs.software_id").Eq(goqu.I("s.id")),
					goqu.I("hs.host_id").Eq(opts.HostID),
				),
			).
			SelectAppend("hs.last_opened_at")
		if opts.TeamID != nil {
			ds = ds.
				Join(
					goqu.I("hosts").As("h"),
					goqu.On(
						goqu.I("hs.host_id").Eq(goqu.I("h.id")),
						goqu.I("h.team_id").Eq(opts.TeamID),
					),
				)
		}

	} else {
		// When loading software from all hosts, filter out software that is not associated with any
		// hosts.
		ds = ds.
			Join(
				goqu.I("software_host_counts").As("shc"),
				goqu.On(
					goqu.I("s.id").Eq(goqu.I("shc.software_id")),
					goqu.I("shc.hosts_count").Gt(0),
				),
			).
			GroupByAppend(
				"shc.hosts_count",
				"shc.updated_at",
				"shc.global_stats",
				"shc.team_id",
			)

		if opts.TeamID == nil { //nolint:gocritic // ignore ifElseChain
			ds = ds.Where(
				goqu.And(
					goqu.I("shc.team_id").Eq(0),
					goqu.I("shc.global_stats").Eq(1),
				),
			)
		} else if *opts.TeamID == 0 {
			ds = ds.Where(
				goqu.And(
					goqu.I("shc.team_id").Eq(0),
					goqu.I("shc.global_stats").Eq(0),
				),
			)
		} else {
			ds = ds.Where(
				goqu.And(
					goqu.I("shc.team_id").Eq(*opts.TeamID),
					goqu.I("shc.global_stats").Eq(0),
				),
			)
		}
	}

	if opts.VulnerableOnly {
		ds = ds.
			Join(
				goqu.I("software_cve").As("scv"),
				goqu.On(goqu.I("s.id").Eq(goqu.I("scv.software_id"))),
			)
	} else {
		ds = ds.
			LeftJoin(
				goqu.I("software_cve").As("scv"),
				goqu.On(goqu.I("s.id").Eq(goqu.I("scv.software_id"))),
			)
	}

	if opts.IncludeCVEScores {

		baseJoinConditions := goqu.Ex{
			"c.cve": goqu.I("scv.cve"),
		}

		if opts.KnownExploit || opts.MinimumCVSS > 0 || opts.MaximumCVSS > 0 {

			if opts.KnownExploit {
				baseJoinConditions["c.cisa_known_exploit"] = true
			}

			if opts.MinimumCVSS > 0 {
				baseJoinConditions["c.cvss_score"] = goqu.Op{"gte": opts.MinimumCVSS}
			}

			if opts.MaximumCVSS > 0 {
				baseJoinConditions["c.cvss_score"] = goqu.Op{"lte": opts.MaximumCVSS}
			}

			ds = ds.InnerJoin(
				goqu.I("cve_meta").As("c"),
				goqu.On(baseJoinConditions),
			)

		} else {
			ds = ds.
				LeftJoin(
					goqu.I("cve_meta").As("c"),
					goqu.On(baseJoinConditions),
				)
		}

		ds = ds.SelectAppend(
			goqu.MAX("c.cvss_score").As("cvss_score"),                     // for ordering
			goqu.MAX("c.epss_probability").As("epss_probability"),         // for ordering
			goqu.MAX("c.cisa_known_exploit").As("cisa_known_exploit"),     // for ordering
			goqu.MAX("c.published").As("cve_published"),                   // for ordering
			goqu.MAX("c.description").As("description"),                   // for ordering
			goqu.MAX("scv.resolved_in_version").As("resolved_in_version"), // for ordering
		)
	}

	if match := opts.ListOptions.MatchQuery; match != "" {
		match = likePattern(match)
		ds = ds.Where(
			goqu.Or(
				goqu.I("s.name").ILike(match),
				goqu.I("s.version").ILike(match),
				goqu.I("scv.cve").ILike(match),
			),
		)
	}

	if opts.WithHostCounts {
		ds = ds.
			SelectAppend(
				goqu.I("shc.hosts_count"),
				goqu.I("shc.updated_at").As("counts_updated_at"),
			)
	}

	ds = ds.GroupBy(
		"s.id",
		"s.name",
		"s.version",
		"s.source",
		"s.bundle_identifier",
		"s.extension_id",
		"s.browser",
		"s.release",
		"s.vendor",
		"s.arch",
		"generated_cpe",
	)

	// Pagination is a bit more complex here due to the join with software_cve table and aggregated columns from cve_meta table.
	// Apply order by again after joining on sub query
	ds = appendListOptionsToSelect(ds, opts.ListOptions)

	// join on software_cve and cve_meta after apply pagination using the sub-query above
	ds = dialect.From(ds.As("s")).
		Select(
			"s.id",
			"s.name",
			"s.version",
			"s.source",
			"s.bundle_identifier",
			"s.extension_id",
			"s.browser",
			"s.release",
			"s.vendor",
			"s.arch",
			goqu.COALESCE(goqu.I("s.generated_cpe"), "").As("generated_cpe"),
			"scv.cve",
			"scv.created_at",
		).
		LeftJoin(
			goqu.I("software_cve").As("scv"),
			goqu.On(goqu.I("scv.software_id").Eq(goqu.I("s.id"))),
		).
		LeftJoin(
			goqu.I("cve_meta").As("c"),
			goqu.On(goqu.I("c.cve").Eq(goqu.I("scv.cve"))),
		)

	// select optional columns
	if opts.IncludeCVEScores {
		ds = ds.SelectAppend(
			"c.cvss_score",
			"c.epss_probability",
			"c.cisa_known_exploit",
			"c.description",
			goqu.I("c.published").As("cve_published"),
			"scv.resolved_in_version",
		)
	}

	if opts.HostID != nil {
		ds = ds.SelectAppend(
			goqu.I("s.last_opened_at"),
		)
	}

	if opts.WithHostCounts {
		ds = ds.SelectAppend(
			goqu.I("s.hosts_count"),
			goqu.I("s.counts_updated_at"),
		)
	}

	ds = appendOrderByToSelect(ds, opts.ListOptions)

	return ds.ToSQL()
}

func countSoftwareDB(
	ctx context.Context,
	q sqlx.QueryerContext,
	opts fleet.SoftwareListOptions,
) (int, error) {
	opts.ListOptions = fleet.ListOptions{
		MatchQuery: opts.ListOptions.MatchQuery,
	}

	sql, args, err := selectSoftwareSQL(opts)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "sql build")
	}

	sql = `SELECT COUNT(DISTINCT s.id) FROM (` + sql + `) AS s`

	var count int
	if err := sqlx.GetContext(ctx, q, &count, sql, args...); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "count host software")
	}

	return count, nil
}

func (ds *Datastore) LoadHostSoftware(ctx context.Context, host *fleet.Host, includeCVEScores bool) error {
	opts := fleet.SoftwareListOptions{
		HostID:           &host.ID,
		IncludeCVEScores: includeCVEScores,
	}
	software, err := listSoftwareDB(ctx, ds.reader(ctx), opts)
	if err != nil {
		return err
	}

	installedPaths, err := ds.getHostSoftwareInstalledPaths(
		ctx,
		host.ID,
	)
	if err != nil {
		return err
	}

	installedPathsList := make(map[uint][]string)
	pathSignatureInformation := make(map[uint][]fleet.PathSignatureInformation)
	for _, ip := range installedPaths {
		installedPathsList[ip.SoftwareID] = append(installedPathsList[ip.SoftwareID], ip.InstalledPath)
		pathSignatureInformation[ip.SoftwareID] = append(pathSignatureInformation[ip.SoftwareID], fleet.PathSignatureInformation{
			InstalledPath:  ip.InstalledPath,
			TeamIdentifier: ip.TeamIdentifier,
		})
	}

	host.Software = make([]fleet.HostSoftwareEntry, 0, len(software))
	for _, s := range software {
		host.Software = append(host.Software, fleet.HostSoftwareEntry{
			Software:                 s,
			InstalledPaths:           installedPathsList[s.ID],
			PathSignatureInformation: pathSignatureInformation[s.ID],
		})
	}
	return nil
}

type softwareIterator struct {
	rows *sqlx.Rows
}

func (si *softwareIterator) Value() (*fleet.Software, error) {
	dest := fleet.Software{}
	err := si.rows.StructScan(&dest)
	if err != nil {
		return nil, err
	}
	return &dest, nil
}

func (si *softwareIterator) Err() error {
	return si.rows.Err()
}

func (si *softwareIterator) Close() error {
	return si.rows.Close()
}

func (si *softwareIterator) Next() bool {
	return si.rows.Next()
}

// AllSoftwareIterator Returns an iterator for the 'software' table, filtering out
// software entries based on the 'query' param. The rows.Close call is done by the caller once
// iteration using the returned fleet.SoftwareIterator is done.
func (ds *Datastore) AllSoftwareIterator(
	ctx context.Context,
	query fleet.SoftwareIterQueryOptions,
) (fleet.SoftwareIterator, error) {
	if !query.IsValid() {
		return nil, fmt.Errorf("invalid query params %+v", query)
	}

	var err error
	var args []interface{}

	stmt := `SELECT
		s.id, s.name, s.version, s.source, s.bundle_identifier, s.release, s.arch, s.vendor, s.browser, s.extension_id, s.title_id,
		COALESCE(sc.cpe, '') AS generated_cpe
	FROM software s
	LEFT JOIN software_cpe sc ON (s.id=sc.software_id)`

	var conditionals []string

	if len(query.ExcludedSources) != 0 {
		conditionals = append(conditionals, "s.source NOT IN (?)")
		args = append(args, query.ExcludedSources)
	}

	if len(query.IncludedSources) != 0 {
		conditionals = append(conditionals, "s.source IN (?)")
		args = append(args, query.IncludedSources)
	}

	if query.NameMatch != "" {
		conditionals = append(conditionals, "s.name REGEXP ?")
		args = append(args, query.NameMatch)
	}

	if query.NameExclude != "" {
		conditionals = append(conditionals, "s.name NOT REGEXP ?")
		args = append(args, query.NameExclude)
	}

	if len(conditionals) != 0 {
		stmt += " WHERE " + strings.Join(conditionals, " AND ")
	}

	stmt, args, err = sqlx.In(stmt, args...)
	if err != nil {
		return nil, fmt.Errorf("error building 'In' query part on software iterator: %w", err)
	}

	rows, err := ds.reader(ctx).QueryxContext(ctx, stmt, args...) //nolint:sqlclosecheck
	if err != nil {
		return nil, fmt.Errorf("executing all software iterator %w", err)
	}
	return &softwareIterator{rows: rows}, nil
}

func (ds *Datastore) UpsertSoftwareCPEs(ctx context.Context, cpes []fleet.SoftwareCPE) (int64, error) {
	var args []interface{}

	if len(cpes) == 0 {
		return 0, nil
	}

	values := strings.TrimSuffix(strings.Repeat("(?,?),", len(cpes)), ",")
	sql := fmt.Sprintf(
		`INSERT INTO software_cpe (software_id, cpe) VALUES %s ON DUPLICATE KEY UPDATE cpe = VALUES(cpe)`,
		values,
	)

	for _, cpe := range cpes {
		args = append(args, cpe.SoftwareID, cpe.CPE)
	}
	res, err := ds.writer(ctx).ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "insert software cpes")
	}
	count, _ := res.RowsAffected()

	return count, nil
}

func (ds *Datastore) DeleteSoftwareCPEs(ctx context.Context, cpes []fleet.SoftwareCPE) (int64, error) {
	if len(cpes) == 0 {
		return 0, nil
	}

	stmt := `DELETE FROM software_cpe WHERE (software_id) IN (?)`

	softwareIDs := make([]uint, 0, len(cpes))
	for _, cpe := range cpes {
		softwareIDs = append(softwareIDs, cpe.SoftwareID)
	}

	query, args, err := sqlx.In(stmt, softwareIDs)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "error building 'In' query part when deleting software CPEs")
	}

	res, err := ds.writer(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		return 0, ctxerr.Wrapf(ctx, err, "deleting cpes software")
	}

	count, _ := res.RowsAffected()

	return count, nil
}

func (ds *Datastore) ListSoftwareCPEs(ctx context.Context) ([]fleet.SoftwareCPE, error) {
	var result []fleet.SoftwareCPE

	var err error
	var args []interface{}

	stmt := `SELECT id, software_id, cpe FROM software_cpe`
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &result, stmt, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "loads cpes")
	}
	return result, nil
}

func (ds *Datastore) ListSoftware(ctx context.Context, opt fleet.SoftwareListOptions) ([]fleet.Software, *fleet.PaginationMetadata, error) {
	if !opt.VulnerableOnly && (opt.MinimumCVSS > 0 || opt.MaximumCVSS > 0 || opt.KnownExploit) {
		return nil, nil, fleet.NewInvalidArgumentError(
			"query", "min_cvss_score, max_cvss_score, and exploit can only be provided with vulnerable=true",
		)
	}

	software, err := listSoftwareDB(ctx, ds.reader(ctx), opt)
	if err != nil {
		return nil, nil, err
	}

	perPage := opt.ListOptions.PerPage
	var metaData *fleet.PaginationMetadata
	if opt.ListOptions.IncludeMetadata {
		if perPage <= 0 {
			perPage = defaultSelectLimit
		}
		metaData = &fleet.PaginationMetadata{HasPreviousResults: opt.ListOptions.Page > 0}
		if len(software) > int(perPage) { //nolint:gosec // dismiss G115
			metaData.HasNextResults = true
			software = software[:len(software)-1]
		}
	}

	return software, metaData, nil
}

func (ds *Datastore) CountSoftware(ctx context.Context, opt fleet.SoftwareListOptions) (int, error) {
	return countSoftwareDB(ctx, ds.reader(ctx), opt)
}

// DeleteSoftwareVulnerabilities deletes the given list of software vulnerabilities
func (ds *Datastore) DeleteSoftwareVulnerabilities(ctx context.Context, vulnerabilities []fleet.SoftwareVulnerability) error {
	if len(vulnerabilities) == 0 {
		return nil
	}

	sql := fmt.Sprintf(
		`DELETE FROM software_cve WHERE (software_id, cve) IN (%s)`,
		strings.TrimSuffix(strings.Repeat("(?,?),", len(vulnerabilities)), ","),
	)
	var args []interface{}
	for _, vulnerability := range vulnerabilities {
		args = append(args, vulnerability.SoftwareID, vulnerability.CVE)
	}
	if _, err := ds.writer(ctx).ExecContext(ctx, sql, args...); err != nil {
		return ctxerr.Wrapf(ctx, err, "deleting vulnerable software")
	}
	return nil
}

func (ds *Datastore) DeleteOutOfDateVulnerabilities(ctx context.Context, source fleet.VulnerabilitySource, duration time.Duration) error {
	sql := `DELETE FROM software_cve WHERE source = ? AND updated_at < ?`

	var args []interface{}
	cutPoint := time.Now().UTC().Add(-1 * duration)
	args = append(args, source, cutPoint)

	if _, err := ds.writer(ctx).ExecContext(ctx, sql, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting out of date vulnerabilities")
	}
	return nil
}

func (ds *Datastore) SoftwareByID(ctx context.Context, id uint, teamID *uint, includeCVEScores bool, tmFilter *fleet.TeamFilter) (*fleet.Software, error) {
	q := dialect.From(goqu.I("software").As("s")).
		Select(
			"s.id",
			"s.name",
			"s.version",
			"s.source",
			"s.browser",
			"s.bundle_identifier",
			"s.release",
			"s.vendor",
			"s.arch",
			"s.extension_id",
			"scv.cve",
			"scv.created_at",
			goqu.COALESCE(goqu.I("scp.cpe"), "").As("generated_cpe"),
		).
		LeftJoin(
			goqu.I("software_cpe").As("scp"),
			goqu.On(
				goqu.I("s.id").Eq(goqu.I("scp.software_id")),
			),
		).
		LeftJoin(
			goqu.I("software_cve").As("scv"),
			goqu.On(goqu.I("s.id").Eq(goqu.I("scv.software_id"))),
		)

	// join only on software_id as we'll need counts for all teams
	// to filter down to the team's the user has access to
	if tmFilter != nil {
		q = q.LeftJoin(
			goqu.I("software_host_counts").As("shc"),
			goqu.On(goqu.I("s.id").Eq(goqu.I("shc.software_id"))),
		)
	}

	if includeCVEScores {
		q = q.
			LeftJoin(
				goqu.I("cve_meta").As("c"),
				goqu.On(goqu.I("c.cve").Eq(goqu.I("scv.cve"))),
			).
			SelectAppend(
				"c.cvss_score",
				"c.epss_probability",
				"c.cisa_known_exploit",
				"c.description",
				goqu.I("c.published").As("cve_published"),
				"scv.resolved_in_version",
			)
	}

	q = q.Where(goqu.I("s.id").Eq(id))
	// If teamID is not specified, we still return the software even if it is not associated with any hosts.
	// Software is cleaned up by a cron job, so it is possible to have software in software_hosts_counts that has been deleted from a host.
	if teamID != nil {
		// If teamID filter is used, host counts need to be up-to-date.
		// This should generally be the case, since unused software is cleared when host counts are updated.
		// However, it is possible that the software was deleted from all hosts after the last host count update.
		q = q.Where(
			goqu.L(
				"EXISTS (SELECT 1 FROM software_host_counts WHERE software_id = ? AND team_id = ? AND hosts_count > 0 AND global_stats = 0)", id, *teamID,
			),
		)
	}

	// filter by teams
	if tmFilter != nil {
		q = q.Where(goqu.L(ds.whereFilterGlobalOrTeamIDByTeams(*tmFilter, "shc")))
	}

	sql, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	var results []softwareCVE
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &results, sql, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get software")
	}

	if len(results) == 0 {
		return nil, ctxerr.Wrap(ctx, notFound("Software").WithID(id))
	}

	var software fleet.Software
	for i, result := range results {
		result := result // create a copy because we need to take the address to fields below

		if i == 0 {
			software = result.Software
		}

		if result.CVE != nil {
			cveID := *result.CVE
			cve := fleet.CVE{
				CVE:         cveID,
				DetailsLink: fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", cveID),
				CreatedAt:   *result.CreatedAt,
			}
			if includeCVEScores {
				cve.CVSSScore = &result.CVSSScore
				cve.EPSSProbability = &result.EPSSProbability
				cve.CISAKnownExploit = &result.CISAKnownExploit
				cve.CVEPublished = &result.CVEPublished
				cve.ResolvedInVersion = &result.ResolvedInVersion
			}
			software.Vulnerabilities = append(software.Vulnerabilities, cve)
		}
	}

	return &software, nil
}

// SyncHostsSoftware calculates the number of hosts having each
// software installed and stores that information in the software_host_counts
// table.
//
// After aggregation, it cleans up unused software (e.g. software installed
// on removed hosts, software uninstalled on hosts, etc.)
func (ds *Datastore) SyncHostsSoftware(ctx context.Context, updatedAt time.Time) error {
	const (
		resetStmt = `
      UPDATE software_host_counts
      SET hosts_count = 0, updated_at = ?`

		// team_id is added to the select list to have the same structure as
		// the teamCountsStmt, making it easier to use a common implementation
		globalCountsStmt = `
      SELECT count(*), 0 as team_id, software_id, 1 as global_stats
      FROM host_software
      WHERE software_id > ? AND software_id <= ?
      GROUP BY software_id`

		teamCountsStmt = `
      SELECT count(*), h.team_id, hs.software_id, 0 as global_stats
      FROM host_software hs
      INNER JOIN hosts h
      ON hs.host_id = h.id
      WHERE h.team_id IS NOT NULL AND hs.software_id > ? AND hs.software_id <= ?
      GROUP BY hs.software_id, h.team_id`

		noTeamCountsStmt = `
      SELECT count(*), 0 as team_id, software_id, 0 as global_stats
      FROM host_software hs
      INNER JOIN hosts h
      ON hs.host_id = h.id
      WHERE h.team_id IS NULL AND hs.software_id > ? AND hs.software_id <= ?
      GROUP BY hs.software_id`

		insertStmt = `
      INSERT INTO software_host_counts
        (software_id, hosts_count, team_id, global_stats, updated_at)
      VALUES
        %s
      ON DUPLICATE KEY UPDATE
        hosts_count = VALUES(hosts_count),
        updated_at = VALUES(updated_at)`

		valuesPart = `(?, ?, ?, ?, ?),`

		// We must ensure that software is not in host_software table before deleting it.
		// This prevents a race condition where a host just added the software, but it is not part of software_host_counts yet.
		// When a host adds software, software table and host_software table are updated in the same transaction.
		cleanupSoftwareStmt = `
      DELETE s
      FROM software s
      LEFT JOIN software_host_counts shc
      ON s.id = shc.software_id
      WHERE
        (shc.software_id IS NULL OR
        (shc.team_id = 0 AND shc.hosts_count = 0)) AND
		NOT EXISTS (SELECT 1 FROM host_software hsw WHERE hsw.software_id = s.id)
	  `

		cleanupOrphanedStmt = `
		  DELETE shc
		  FROM
		    software_host_counts shc
		    LEFT JOIN software s ON s.id = shc.software_id
		  WHERE
		    s.id IS NULL
		`

		cleanupTeamStmt = `
      DELETE shc
      FROM software_host_counts shc
      LEFT JOIN teams t
      ON t.id = shc.team_id
      WHERE
        shc.team_id > 0 AND
        t.id IS NULL`
	)

	// first, reset all counts to 0
	if _, err := ds.writer(ctx).ExecContext(ctx, resetStmt, updatedAt); err != nil {
		return ctxerr.Wrap(ctx, err, "reset all software_host_counts to 0")
	}

	db := ds.reader(ctx)

	// Figure out how many software items we need to count.
	type minMaxIDs struct {
		Min uint64 `db:"min"`
		Max uint64 `db:"max"`
	}
	minMax := minMaxIDs{}
	err := sqlx.GetContext(
		ctx, db, &minMax, "SELECT COALESCE(MIN(software_id),1) as min, COALESCE(MAX(software_id),0) as max FROM host_software",
	)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get min/max software_id")
	}

	for minSoftwareID, maxSoftwareID := minMax.Min-1, minMax.Min-1+countHostSoftwareBatchSize; minSoftwareID < minMax.Max; minSoftwareID, maxSoftwareID = maxSoftwareID, maxSoftwareID+countHostSoftwareBatchSize {

		// next get a cursor for the global and team counts for each software
		stmtLabel := []string{"global", "team", "noteam"}
		for i, countStmt := range []string{globalCountsStmt, teamCountsStmt, noTeamCountsStmt} {
			rows, err := db.QueryContext(ctx, countStmt, minSoftwareID, maxSoftwareID)
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "read %s counts from host_software", stmtLabel[i])
			}
			defer rows.Close()

			// use a loop to iterate to prevent loading all in one go in memory, as it
			// could get pretty big at >100K hosts with 1000+ software each. Use a write
			// batch to prevent making too many single-row inserts.
			const batchSize = 100
			var batchCount int
			args := make([]interface{}, 0, batchSize*4)
			for rows.Next() {
				var (
					count        int
					teamID       uint
					sid          uint
					global_stats bool
				)

				if err := rows.Scan(&count, &teamID, &sid, &global_stats); err != nil {
					return ctxerr.Wrapf(ctx, err, "scan %s row into variables", stmtLabel[i])
				}

				args = append(args, sid, count, teamID, global_stats, updatedAt)
				batchCount++

				if batchCount == batchSize {
					values := strings.TrimSuffix(strings.Repeat(valuesPart, batchCount), ",")
					if _, err := ds.writer(ctx).ExecContext(ctx, fmt.Sprintf(insertStmt, values), args...); err != nil {
						return ctxerr.Wrapf(ctx, err, "insert %s batch into software_host_counts", stmtLabel[i])
					}

					args = args[:0]
					batchCount = 0
				}
			}
			if batchCount > 0 {
				values := strings.TrimSuffix(strings.Repeat(valuesPart, batchCount), ",")
				if _, err := ds.writer(ctx).ExecContext(ctx, fmt.Sprintf(insertStmt, values), args...); err != nil {
					return ctxerr.Wrapf(ctx, err, "insert last %s batch into software_host_counts", stmtLabel[i])
				}
			}
			if err := rows.Err(); err != nil {
				return ctxerr.Wrapf(ctx, err, "iterate over %s host_software counts", stmtLabel[i])
			}
			rows.Close()
		}

	}

	// remove any unused software (global counts = 0)
	if _, err := ds.writer(ctx).ExecContext(ctx, cleanupSoftwareStmt); err != nil {
		return ctxerr.Wrap(ctx, err, "delete unused software")
	}

	// remove any software count row for software that don't exist anymore
	if _, err := ds.writer(ctx).ExecContext(ctx, cleanupOrphanedStmt); err != nil {
		return ctxerr.Wrap(ctx, err, "delete software_host_counts for non-existing software")
	}

	// remove any software count row for teams that don't exist anymore
	if _, err := ds.writer(ctx).ExecContext(ctx, cleanupTeamStmt); err != nil {
		return ctxerr.Wrap(ctx, err, "delete software_host_counts for non-existing teams")
	}
	return nil
}

func (ds *Datastore) ReconcileSoftwareTitles(ctx context.Context) error {
	// TODO: consider if we should batch writes to software or software_titles table

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// ensure all software titles are in the software_titles table
		upsertTitlesStmt := `
INSERT INTO software_titles (name, source, browser, bundle_identifier)
SELECT
    name,
    source,
    browser,
    bundle_identifier
FROM (
    SELECT DISTINCT
        name,
        source,
        browser,
        bundle_identifier
    FROM
        software s
    WHERE
        NOT EXISTS (
            SELECT 1 FROM software_titles st
            WHERE s.bundle_identifier = st.bundle_identifier AND
				IF(s.source IN ('apps', 'ios_apps', 'ipados_apps'), s.source = st.source, 1)
        )
        AND COALESCE(bundle_identifier, '') != ''

    UNION ALL

    SELECT DISTINCT
        name,
        source,
        browser,
        NULL as bundle_identifier
    FROM
        software s
    WHERE
        NOT EXISTS (
            SELECT 1 FROM software_titles st
            WHERE (s.name, s.source, s.browser) = (st.name, st.source, st.browser)
        )
        AND COALESCE(s.bundle_identifier, '') = ''
) as combined_results
ON DUPLICATE KEY UPDATE
    software_titles.name = software_titles.name,
    software_titles.source = software_titles.source,
    software_titles.browser = software_titles.browser,
    software_titles.bundle_identifier = software_titles.bundle_identifier
`
		res, err := tx.ExecContext(ctx, upsertTitlesStmt)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "upsert software titles")
		}
		n, _ := res.RowsAffected()
		level.Debug(ds.logger).Log("msg", "upsert software titles", "rows_affected", n)

		// update title ids for software table entries
		updateSoftwareWithoutIdentifierStmt := `
UPDATE software s
JOIN software_titles st
ON COALESCE(s.bundle_identifier, '') = '' AND s.name = st.name AND s.source = st.source AND s.browser = st.browser
SET s.title_id = st.id
WHERE (s.title_id IS NULL OR s.title_id != st.id)
AND COALESCE(s.bundle_identifier, '') = '';
`

		res, err = tx.ExecContext(ctx, updateSoftwareWithoutIdentifierStmt)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "update software title_id without bundle identifier")
		}
		n, _ = res.RowsAffected()
		level.Debug(ds.logger).Log("msg", "update software title_id without bundle identifier", "rows_affected", n)

		updateSoftwareWithIdentifierStmt := `
UPDATE software s
JOIN software_titles st
ON s.bundle_identifier = st.bundle_identifier AND
    IF(s.source IN ('apps', 'ios_apps', 'ipados_apps'), s.source = st.source, 1)
SET s.title_id = st.id
WHERE s.title_id IS NULL
OR s.title_id != st.id;
`

		res, err = tx.ExecContext(ctx, updateSoftwareWithIdentifierStmt)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "update software title_id with bundle identifier")
		}
		n, _ = res.RowsAffected()
		level.Debug(ds.logger).Log("msg", "update software title_id with bundle identifier", "rows_affected", n)

		// clean up orphaned software titles
		cleanupStmt := `
DELETE st FROM software_titles st
	LEFT JOIN software s ON s.title_id = st.id
	WHERE s.title_id IS NULL AND
		NOT EXISTS (SELECT 1 FROM software_installers si WHERE si.title_id = st.id) AND
		NOT EXISTS (SELECT 1 FROM vpp_apps vap WHERE vap.title_id = st.id)`

		res, err = tx.ExecContext(ctx, cleanupStmt)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "cleanup orphaned software titles")
		}
		n, _ = res.RowsAffected()
		level.Debug(ds.logger).Log("msg", "cleanup orphaned software titles", "rows_affected", n)

		return nil
	})
}

func (ds *Datastore) HostVulnSummariesBySoftwareIDs(ctx context.Context, softwareIDs []uint) ([]fleet.HostVulnerabilitySummary, error) {
	stmt := `
		SELECT DISTINCT
			h.id,
			h.hostname,
			if(h.computer_name = '', h.hostname, h.computer_name) display_name,
			COALESCE(hsip.installed_path, '') AS software_installed_path
		FROM hosts h
				INNER JOIN host_software hs ON h.id = hs.host_id AND hs.software_id IN (?)
				LEFT JOIN host_software_installed_paths hsip ON hs.host_id = hsip.host_id AND hs.software_id = hsip.software_id
		ORDER BY h.id`

	stmt, args, err := sqlx.In(stmt, softwareIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building query args")
	}

	var qR []struct {
		HostID      uint   `db:"id"`
		HostName    string `db:"hostname"`
		DisplayName string `db:"display_name"`
		SPath       string `db:"software_installed_path"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &qR, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting hosts by softwareIDs")
	}

	var result []fleet.HostVulnerabilitySummary
	lookup := make(map[uint]int)

	for _, r := range qR {
		i, ok := lookup[r.HostID]

		if ok {
			result[i].AddSoftwareInstalledPath(r.SPath)
			continue
		}

		mapped := fleet.HostVulnerabilitySummary{
			ID:          r.HostID,
			Hostname:    r.HostName,
			DisplayName: r.DisplayName,
		}
		mapped.AddSoftwareInstalledPath(r.SPath)
		result = append(result, mapped)

		lookup[r.HostID] = len(result) - 1
	}

	return result, nil
}

// ** DEPRECATED **
func (ds *Datastore) HostsByCVE(ctx context.Context, cve string) ([]fleet.HostVulnerabilitySummary, error) {
	stmt := `
		SELECT DISTINCT
				(h.id),
				h.hostname,
				if(h.computer_name = '', h.hostname, h.computer_name) display_name,
				COALESCE(hsip.installed_path, '') AS software_installed_path
		FROM hosts h
			INNER JOIN host_software hs ON h.id = hs.host_id
			INNER JOIN software_cve scv ON scv.software_id = hs.software_id
			LEFT JOIN host_software_installed_paths hsip ON hs.host_id = hsip.host_id AND hs.software_id = hsip.software_id
		WHERE scv.cve = ?
		ORDER BY h.id`

	var qR []struct {
		HostID      uint   `db:"id"`
		HostName    string `db:"hostname"`
		DisplayName string `db:"display_name"`
		SPath       string `db:"software_installed_path"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &qR, stmt, cve); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting hosts by softwareIDs")
	}

	var result []fleet.HostVulnerabilitySummary
	lookup := make(map[uint]int)

	for _, r := range qR {
		i, ok := lookup[r.HostID]

		if ok {
			result[i].AddSoftwareInstalledPath(r.SPath)
			continue
		}

		mapped := fleet.HostVulnerabilitySummary{
			ID:          r.HostID,
			Hostname:    r.HostName,
			DisplayName: r.DisplayName,
		}
		mapped.AddSoftwareInstalledPath(r.SPath)
		result = append(result, mapped)

		lookup[r.HostID] = len(result) - 1
	}

	return result, nil
}

func (ds *Datastore) InsertCVEMeta(ctx context.Context, cveMeta []fleet.CVEMeta) error {
	query := `
INSERT INTO cve_meta (cve, cvss_score, epss_probability, cisa_known_exploit, published, description)
VALUES %s
ON DUPLICATE KEY UPDATE
    cvss_score = VALUES(cvss_score),
    epss_probability = VALUES(epss_probability),
    cisa_known_exploit = VALUES(cisa_known_exploit),
    published = VALUES(published),
    description = VALUES(description)
`

	batchSize := 500
	for i := 0; i < len(cveMeta); i += batchSize {
		end := i + batchSize
		if end > len(cveMeta) {
			end = len(cveMeta)
		}

		batch := cveMeta[i:end]

		valuesFrag := strings.TrimSuffix(strings.Repeat("(?, ?, ?, ?, ?, ?), ", len(batch)), ", ")
		var args []interface{}
		for _, meta := range batch {
			args = append(args, meta.CVE, meta.CVSSScore, meta.EPSSProbability, meta.CISAKnownExploit, meta.Published, meta.Description)
		}

		query := fmt.Sprintf(query, valuesFrag)

		_, err := ds.writer(ctx).ExecContext(ctx, query, args...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert cve scores")
		}
	}

	return nil
}

func (ds *Datastore) InsertSoftwareVulnerability(
	ctx context.Context,
	vuln fleet.SoftwareVulnerability,
	source fleet.VulnerabilitySource,
) (bool, error) {
	if vuln.CVE == "" {
		return false, nil
	}

	var args []interface{}

	stmt := `
		INSERT INTO software_cve (cve, source, software_id, resolved_in_version)
		VALUES (?,?,?,?)
		ON DUPLICATE KEY UPDATE
			source = VALUES(source),
			resolved_in_version = VALUES(resolved_in_version),
			updated_at=?
	`
	args = append(args, vuln.CVE, source, vuln.SoftwareID, vuln.ResolvedInVersion, time.Now().UTC())

	res, err := ds.writer(ctx).ExecContext(ctx, stmt, args...)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "insert software vulnerability")
	}

	return insertOnDuplicateDidInsertOrUpdate(res), nil
}

func (ds *Datastore) ListSoftwareVulnerabilitiesByHostIDsSource(
	ctx context.Context,
	hostIDs []uint,
	source fleet.VulnerabilitySource,
) (map[uint][]fleet.SoftwareVulnerability, error) {
	result := make(map[uint][]fleet.SoftwareVulnerability)

	type softwareVulnerabilityWithHostId struct {
		fleet.SoftwareVulnerability
		HostID uint `db:"host_id"`
	}
	var queryR []softwareVulnerabilityWithHostId

	stmt := dialect.
		From(goqu.T("software_cve").As("sc")).
		Join(
			goqu.T("host_software").As("hs"),
			goqu.On(goqu.Ex{
				"sc.software_id": goqu.I("hs.software_id"),
			}),
		).
		Select(
			goqu.I("hs.host_id"),
			goqu.I("sc.software_id"),
			goqu.I("sc.cve"),
			goqu.I("sc.resolved_in_version"),
		).
		Where(
			goqu.I("hs.host_id").In(hostIDs),
			goqu.I("sc.source").Eq(source),
		)

	sql, args, err := stmt.ToSQL()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "error generating SQL statement")
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &queryR, sql, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "error executing SQL statement")
	}

	for _, r := range queryR {
		result[r.HostID] = append(result[r.HostID], r.SoftwareVulnerability)
	}

	return result, nil
}

func (ds *Datastore) ListSoftwareForVulnDetection(
	ctx context.Context,
	filters fleet.VulnSoftwareFilter,
) ([]fleet.Software, error) {
	var result []fleet.Software
	var sqlstmt string
	var args []interface{}

	baseSQL := `
		SELECT
			s.id,
			s.name,
			s.version,
			s.release,
			s.arch,
			COALESCE(cpe.cpe, '') AS generated_cpe
		FROM
			software s
		LEFT JOIN
			software_cpe cpe ON s.id = cpe.software_id
	`

	if filters.HostID != nil {
		baseSQL += "JOIN host_software hs ON s.id = hs.software_id "
	}

	conditions := []string{}

	if filters.HostID != nil {
		conditions = append(conditions, "hs.host_id = ?")
		args = append(args, *filters.HostID)
	}

	if filters.Name != "" {
		conditions = append(conditions, "s.name LIKE ?")
		args = append(args, "%"+filters.Name+"%")
	}

	if filters.Source != "" {
		conditions = append(conditions, "s.source = ?")
		args = append(args, filters.Source)
	}

	if len(conditions) > 0 {
		sqlstmt = baseSQL + "WHERE " + strings.Join(conditions, " AND ")
	} else {
		sqlstmt = baseSQL
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &result, sqlstmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "error executing SQL statement")
	}

	return result, nil
}

// ListCVEs returns all cve_meta rows published after 'maxAge'
func (ds *Datastore) ListCVEs(ctx context.Context, maxAge time.Duration) ([]fleet.CVEMeta, error) {
	var result []fleet.CVEMeta

	maxAgeDate := time.Now().Add(-1 * maxAge)
	stmt := dialect.From(goqu.T("cve_meta")).
		Select(
			goqu.C("cve"),
			goqu.C("cvss_score"),
			goqu.C("epss_probability"),
			goqu.C("cisa_known_exploit"),
			goqu.C("published"),
			goqu.C("description"),
		).
		Where(goqu.C("published").Gte(maxAgeDate))

	sql, args, err := stmt.ToSQL()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "error generating SQL statement")
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &result, sql, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "error executing SQL statement")
	}

	return result, nil
}

func (ds *Datastore) ListHostSoftware(ctx context.Context, host *fleet.Host, opts fleet.HostSoftwareTitleListOptions) ([]*fleet.HostSoftwareWithInstaller, *fleet.PaginationMetadata, error) {
	var onlySelfServiceClause string
	if opts.SelfServiceOnly {
		onlySelfServiceClause = ` AND ( si.self_service = 1 OR ( vat.self_service = 1 AND :is_mdm_enrolled ) ) `
	}

	var excludeVPPAppsClause string
	if !opts.IsMDMEnrolled {
		excludeVPPAppsClause = ` AND vat.id IS NULL `
	}

	var onlyVulnerableJoin string
	if opts.VulnerableOnly {
		onlyVulnerableJoin = `
INNER JOIN software_cve scve ON scve.software_id = s.id
		`
	}

	softwareIsInstalledOnHostClause := fmt.Sprintf(`
			EXISTS (
				SELECT 1
				FROM
					host_software hs
				INNER JOIN
					software s ON hs.software_id = s.id
					%s
				WHERE
					hs.host_id = :host_id AND
					s.title_id = st.id
			) OR `, onlyVulnerableJoin)
	status := fmt.Sprintf(`COALESCE(%s, %s)`, "hsi.last_status", vppAppHostStatusNamedQuery("hvsi", "ncr", ""))
	if opts.OnlyAvailableForInstall {
		// Get software that has a package/VPP installer but was not installed with Fleet
		softwareIsInstalledOnHostClause = fmt.Sprintf(` %s IS NULL AND (si.id IS NOT NULL OR vat.adam_id IS NOT NULL) AND %s`, status,
			softwareIsInstalledOnHostClause)
	}

	// this statement lists only the software that is reported as installed on
	// the host or has been attempted to be installed on the host.
	stmtInstalled := fmt.Sprintf(`
		SELECT
			st.id,
			st.name,
			st.source,
			si.self_service as package_self_service,
			si.filename as package_name,
			si.version as package_version,
			vat.self_service as vpp_app_self_service,
			vat.adam_id as vpp_app_adam_id,
			vap.latest_version as vpp_app_version,
			NULLIF(vap.icon_url, '') as vpp_app_icon_url,
			COALESCE(hsi.last_installed_at, hvsi.created_at) as last_install_installed_at,
			COALESCE(hsi.last_install_execution_id, hvsi.command_uuid) as last_install_install_uuid,
			hsi.last_uninstalled_at as last_uninstall_uninstalled_at,
			hsi.last_uninstall_execution_id as last_uninstall_script_execution_id,
			-- get either the software installer status or the vpp app status
			%s as status
		FROM
			software_titles st
		LEFT OUTER JOIN
			software_installers si ON st.id = si.title_id AND si.global_or_team_id = :global_or_team_id
		LEFT OUTER JOIN -- get the latest status and install/uninstall attempts (merge 3 host_software_installs rows into 1)
			(
				SELECT
					hsi_group.host_id,
					hsi_group.software_installer_id,
					TIMESTAMP(GROUP_CONCAT(hsi_installed_at)) as last_installed_at,
					GROUP_CONCAT(hsi_install_execution_id) as last_install_execution_id,
					TIMESTAMP(GROUP_CONCAT(hsi_uninstalled_at)) as last_uninstalled_at,
					GROUP_CONCAT(hsi_uninstall_execution_id) as last_uninstall_execution_id,
					IF(GROUP_CONCAT(hsi_status) = '', NULL, GROUP_CONCAT(hsi_status)) as last_status
				FROM (
					-- get latest install/uninstall status
					SELECT
						host_id, software_installer_id,
						NULL as hsi_installed_at, NULL as hsi_install_execution_id,
						NULL as hsi_uninstalled_at, NULL as hsi_uninstall_execution_id,
						-- get the status of the latest attempt; 27-1 is the length of the timestamp
					    SUBSTRING(MAX(CONCAT(created_at, COALESCE(status, ''))), 27) AS hsi_status
					FROM host_software_installs
					WHERE host_id = :host_id AND removed = 0
					GROUP BY host_id, software_installer_id
					UNION
					-- get latest install attempt
					SELECT
						host_id, software_installer_id,
						MAX(created_at) as hsi_installed_at,
						-- get the execution_id of the latest attempt; 27-1 is the length of the timestamp
					    SUBSTRING(MAX(CONCAT(created_at, execution_id)), 27) AS hsi_install_execution_id,
						NULL as hsi_uninstalled_at, NULL as hsi_uninstall_execution_id,
						NULL as hsi_status
					FROM host_software_installs
					WHERE host_id = :host_id AND removed = 0 AND uninstall = 0
					GROUP BY host_id, software_installer_id
					UNION
					-- get latest uninstall attempt
					SELECT
						host_id, software_installer_id,
						NULL as hsi_installed_at, NULL as hsi_install_execution_id,
						MAX(created_at) as hsi_uninstalled_at,
						-- get the execution_id of the latest attempt; 27-1 is the length of the timestamp
					    SUBSTRING(MAX(CONCAT(created_at, execution_id)), 27) AS hsi_uninstall_execution_id,
						NULL as hsi_status
						FROM host_software_installs
					WHERE host_id = :host_id AND removed = 0 AND uninstall = 1
					GROUP BY host_id, software_installer_id
				) as hsi_group
				GROUP BY hsi_group.host_id, hsi_group.software_installer_id
			) as hsi ON si.id = hsi.software_installer_id
		LEFT OUTER JOIN
			vpp_apps vap ON st.id = vap.title_id AND vap.platform = :host_platform
		LEFT OUTER JOIN
			vpp_apps_teams vat ON vap.adam_id = vat.adam_id AND vap.platform = vat.platform AND vat.global_or_team_id = :global_or_team_id
		LEFT OUTER JOIN
			host_vpp_software_installs hvsi ON vat.adam_id = hvsi.adam_id AND hvsi.host_id = :host_id AND hvsi.removed = 0
		LEFT OUTER JOIN
			nano_command_results ncr ON ncr.command_uuid = hvsi.command_uuid
		WHERE
			-- use the latest VPP install attempt only
			( hvsi.id IS NULL OR hvsi.id = (
				SELECT hvsi2.id
				FROM host_vpp_software_installs hvsi2
				WHERE hvsi2.host_id = hvsi.host_id AND hvsi2.adam_id = hvsi.adam_id AND hvsi2.platform = hvsi.platform AND hvsi2.removed = 0
				ORDER BY hvsi2.created_at DESC
				LIMIT 1 ) ) AND

			-- software is installed on host or software install has been attempted
			-- on host (via installer or VPP app). If only available for install is
			-- requested, then the software installed on host clause is empty.
			( %s hsi.host_id IS NOT NULL OR hvsi.host_id IS NOT NULL )
			%s
`, status, softwareIsInstalledOnHostClause, onlySelfServiceClause)

	// this statement lists only the software that has never been installed nor
	// attempted to be installed on the host, but that is available to be
	// installed on the host's platform.

	stmtAvailable := fmt.Sprintf(`
		SELECT
			st.id,
			st.name,
			st.source,
			si.self_service as package_self_service,
			si.filename as package_name,
			si.version as package_version,
			vat.self_service as vpp_app_self_service,
			vat.adam_id as vpp_app_adam_id,
			vap.latest_version as vpp_app_version,
			NULLIF(vap.icon_url, '') as vpp_app_icon_url,
			NULL as last_install_installed_at,
			NULL as last_install_install_uuid,
			NULL as last_uninstall_uninstalled_at,
			NULL as last_uninstall_script_execution_id,
			NULL as status
		FROM
			software_titles st
		LEFT OUTER JOIN
			-- filter out software that is not available for install on the host's platform
			software_installers si ON st.id = si.title_id AND si.platform IN (:host_compatible_platforms) AND si.global_or_team_id = :global_or_team_id
		LEFT OUTER JOIN
			-- include VPP apps only if the host is on a supported platform
			vpp_apps vap ON st.id = vap.title_id AND :host_platform IN (:vpp_apps_platforms)
		LEFT OUTER JOIN
			vpp_apps_teams vat ON vap.adam_id = vat.adam_id AND vap.platform = vat.platform AND vat.global_or_team_id = :global_or_team_id
		WHERE
			-- software is not installed on host (but is available in host's team)
			NOT EXISTS (
				SELECT 1
				FROM
					host_software hs
				INNER JOIN
					software s ON hs.software_id = s.id
				WHERE
					hs.host_id = :host_id AND
					s.title_id = st.id
			) AND
			-- sofware install has not been attempted on host
			NOT EXISTS (
				SELECT 1
				FROM
					host_software_installs hsi
				WHERE
					hsi.host_id = :host_id AND
					hsi.software_installer_id = si.id AND
					hsi.removed = 0
			) AND
			NOT EXISTS (
				SELECT 1
				FROM
					host_vpp_software_installs hvsi
				WHERE
					hvsi.host_id = :host_id AND
					hvsi.adam_id = vat.adam_id AND
					hvsi.removed = 0
			) AND
			-- either the software installer or the vpp app exists for the host's team
			( si.id IS NOT NULL OR vat.platform = :host_platform ) AND
			-- label membership check
			(
			 	-- do the label membership check only for software installers
				CASE WHEN si.ID IS NOT NULL THEN
				(
					EXISTS (

					SELECT 1 FROM (

						-- no labels
						SELECT 0 AS count_installer_labels, 0 AS count_host_labels
						WHERE NOT EXISTS (
							SELECT 1 FROM software_installer_labels sil WHERE sil.software_installer_id = si.id
						)

						UNION

						-- include any
						SELECT
							COUNT(*) AS count_installer_labels,
							COUNT(lm.label_id) AS count_host_labels
						FROM
							software_installer_labels sil
							LEFT OUTER JOIN label_membership lm ON lm.label_id = sil.label_id
							AND lm.host_id = :host_id
						WHERE
							sil.software_installer_id = si.id
							AND sil.exclude = 0
						HAVING
							count_installer_labels > 0 AND count_host_labels > 0 
							
						UNION

						-- exclude any
						SELECT
							COUNT(*) AS count_installer_labels,
							COUNT(lm.label_id) AS count_host_labels
						FROM
							software_installer_labels sil
							LEFT OUTER JOIN label_membership lm ON lm.label_id = sil.label_id
							AND lm.host_id = :host_id
						WHERE
							sil.software_installer_id = si.id
							AND sil.exclude = 1
						HAVING
							count_installer_labels > 0 AND count_host_labels = 0
						) t
					)
				)
				-- it's some other type of software that has been checked above		
				ELSE true END
			)
			%s %s
`, onlySelfServiceClause, excludeVPPAppsClause)

	// this is the top-level SELECT of fields from the UNION of the sub-selects
	// (stmtAvailable and stmtInstalled).
	const selectColNames = `
	SELECT
		id,
		name,
		source,
		package_self_service,
		package_name,
		package_version,
		vpp_app_self_service,
		vpp_app_adam_id,
		vpp_app_version,
		vpp_app_icon_url,
		last_install_installed_at,
		last_install_install_uuid,
		last_uninstall_uninstalled_at,
		last_uninstall_script_execution_id,
		status
`

	var globalOrTeamID uint
	if host.TeamID != nil {
		globalOrTeamID = *host.TeamID
	}
	namedArgs := map[string]any{
		"host_id":                   host.ID,
		"host_platform":             host.FleetPlatform(),
		"software_status_failed":    fleet.SoftwareInstallFailed,
		"software_status_pending":   fleet.SoftwareInstallPending,
		"software_status_installed": fleet.SoftwareInstalled,
		"mdm_status_acknowledged":   fleet.MDMAppleStatusAcknowledged,
		"mdm_status_error":          fleet.MDMAppleStatusError,
		"mdm_status_format_error":   fleet.MDMAppleStatusCommandFormatError,
		"global_or_team_id":         globalOrTeamID,
		"is_mdm_enrolled":           opts.IsMDMEnrolled,
	}

	stmt := stmtInstalled
	if opts.OnlyAvailableForInstall || (opts.IncludeAvailableForInstall && !opts.VulnerableOnly) {
		namedArgs["vpp_apps_platforms"] = fleet.VPPAppsPlatforms
		if fleet.IsLinux(host.Platform) {
			namedArgs["host_compatible_platforms"] = fleet.HostLinuxOSs
		} else {
			namedArgs["host_compatible_platforms"] = []string{host.FleetPlatform()}
		}
		stmt += ` UNION ` + stmtAvailable
	}

	// must resolve the named bindings here, before adding the searchLike which
	// uses standard placeholders.
	stmt, args, err := sqlx.Named(stmt, namedArgs)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "build named query for list host software")
	}
	stmt, args, err = sqlx.In(stmt, args...)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "expand IN query for list host software")
	}

	stmt = selectColNames + ` FROM ( ` + stmt + ` ) AS tbl `

	if opts.ListOptions.MatchQuery != "" {
		stmt += " WHERE TRUE " // searchLike adds a "AND <condition>"
		stmt, args = searchLike(stmt, args, opts.ListOptions.MatchQuery, "name")
	}

	// build the count statement before adding pagination constraints
	countStmt := fmt.Sprintf(`SELECT COUNT(DISTINCT s.id) FROM (%s) AS s`, stmt)
	stmt, _ = appendListOptionsToSQL(stmt, &opts.ListOptions)

	// perform a second query to grab the titleCount
	var titleCount uint
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &titleCount, countStmt, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "get host software count")
	}

	type hostSoftware struct {
		fleet.HostSoftwareWithInstaller
		LastInstallInstalledAt         *time.Time `db:"last_install_installed_at"`
		LastInstallInstallUUID         *string    `db:"last_install_install_uuid"`
		LastUninstallUninstalledAt     *time.Time `db:"last_uninstall_uninstalled_at"`
		LastUninstallScriptExecutionID *string    `db:"last_uninstall_script_execution_id"`
		PackageSelfService             *bool      `db:"package_self_service"`
		PackageName                    *string    `db:"package_name"`
		PackageVersion                 *string    `db:"package_version"`
		VPPAppSelfService              *bool      `db:"vpp_app_self_service"`
		VPPAppAdamID                   *string    `db:"vpp_app_adam_id"`
		VPPAppVersion                  *string    `db:"vpp_app_version"`
		VPPAppIconURL                  *string    `db:"vpp_app_icon_url"`
	}
	var hostSoftwareList []*hostSoftware
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hostSoftwareList, stmt, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "list host software")
	}

	// collect the title ids to get the versions, vulnerabilities and installed
	// paths for each software in the list.
	titleIDs := make([]uint, 0, len(hostSoftwareList))
	byTitleID := make(map[uint]*hostSoftware, len(hostSoftwareList))
	for _, hs := range hostSoftwareList {

		// promote the package name and version to the proper destination fields
		if hs.PackageName != nil {
			var version string
			if hs.PackageVersion != nil {
				version = *hs.PackageVersion
			}
			hs.SoftwarePackage = &fleet.SoftwarePackageOrApp{
				Name:        *hs.PackageName,
				Version:     version,
				SelfService: hs.PackageSelfService,
			}

			// promote the last install info to the proper destination fields
			if hs.LastInstallInstallUUID != nil && *hs.LastInstallInstallUUID != "" {
				hs.SoftwarePackage.LastInstall = &fleet.HostSoftwareInstall{
					InstallUUID: *hs.LastInstallInstallUUID,
				}
				if hs.LastInstallInstalledAt != nil {
					hs.SoftwarePackage.LastInstall.InstalledAt = *hs.LastInstallInstalledAt
				}
			}

			// promote the last uninstall info to the proper destination fields
			if hs.LastUninstallScriptExecutionID != nil && *hs.LastUninstallScriptExecutionID != "" {
				hs.SoftwarePackage.LastUninstall = &fleet.HostSoftwareUninstall{
					ExecutionID: *hs.LastUninstallScriptExecutionID,
				}
				if hs.LastUninstallUninstalledAt != nil {
					hs.SoftwarePackage.LastUninstall.UninstalledAt = *hs.LastUninstallUninstalledAt
				}
			}
		}

		// promote the VPP app id and version to the proper destination fields
		if hs.VPPAppAdamID != nil {
			var version string
			if hs.VPPAppVersion != nil {
				version = *hs.VPPAppVersion
			}
			hs.AppStoreApp = &fleet.SoftwarePackageOrApp{
				AppStoreID:  *hs.VPPAppAdamID,
				Version:     version,
				SelfService: hs.VPPAppSelfService,
				IconURL:     hs.VPPAppIconURL,
			}

			// promote the last install info to the proper destination fields
			if hs.LastInstallInstallUUID != nil && *hs.LastInstallInstallUUID != "" {
				hs.AppStoreApp.LastInstall = &fleet.HostSoftwareInstall{
					CommandUUID: *hs.LastInstallInstallUUID,
				}
				if hs.LastInstallInstalledAt != nil {
					hs.AppStoreApp.LastInstall.InstalledAt = *hs.LastInstallInstalledAt
				}
			}
		}

		titleIDs = append(titleIDs, hs.ID)
		byTitleID[hs.ID] = hs
	}

	if len(titleIDs) > 0 {
		// get the software versions installed on that host
		const versionStmt = `
		SELECT
			st.id as software_title_id,
			s.id as software_id,
			s.version,
			s.bundle_identifier,
			s.source,
			hs.last_opened_at
		FROM
			software s
		INNER JOIN
			software_titles st ON s.title_id = st.id
		INNER JOIN
			host_software hs ON s.id = hs.software_id AND hs.host_id = ?
		WHERE
			st.id IN (?)
`
		var installedVersions []*fleet.HostSoftwareInstalledVersion
		stmt, args, err := sqlx.In(versionStmt, host.ID, titleIDs)
		if err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "building query args to list versions")
		}
		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &installedVersions, stmt, args...); err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "list software versions")
		}

		// store the installed versions with the proper software entry and collect
		// the software ids.
		softwareIDs := make([]uint, 0, len(installedVersions))
		bySoftwareID := make(map[uint]*fleet.HostSoftwareInstalledVersion, len(hostSoftwareList))
		for _, ver := range installedVersions {
			hs := byTitleID[ver.SoftwareTitleID]
			hs.InstalledVersions = append(hs.InstalledVersions, ver)
			softwareIDs = append(softwareIDs, ver.SoftwareID)
			bySoftwareID[ver.SoftwareID] = ver
		}

		if len(softwareIDs) > 0 {
			const cveStmt = `
			SELECT
				sc.software_id,
				sc.cve
			FROM
				software_cve sc
			WHERE
				sc.software_id IN (?)
			ORDER BY
				software_id, cve
	`
			type softwareCVE struct {
				SoftwareID uint   `db:"software_id"`
				CVE        string `db:"cve"`
			}
			var softwareCVEs []softwareCVE
			stmt, args, err = sqlx.In(cveStmt, softwareIDs)
			if err != nil {
				return nil, nil, ctxerr.Wrap(ctx, err, "building query args to list cves")
			}
			if err := sqlx.SelectContext(ctx, ds.reader(ctx), &softwareCVEs, stmt, args...); err != nil {
				return nil, nil, ctxerr.Wrap(ctx, err, "list software cves")
			}

			// store the CVEs with the proper software entry
			for _, cve := range softwareCVEs {
				ver := bySoftwareID[cve.SoftwareID]
				ver.Vulnerabilities = append(ver.Vulnerabilities, cve.CVE)
			}

			const pathsStmt = `
			SELECT
				hsip.software_id,
				hsip.installed_path,
				hsip.team_identifier
			FROM
				host_software_installed_paths hsip
			WHERE
				hsip.host_id = ? AND
				hsip.software_id IN (?)
			ORDER BY
				software_id, installed_path
	`
			type installedPath struct {
				SoftwareID     uint   `db:"software_id"`
				InstalledPath  string `db:"installed_path"`
				TeamIdentifier string `db:"team_identifier"`
			}
			var installedPaths []installedPath
			stmt, args, err = sqlx.In(pathsStmt, host.ID, softwareIDs)
			if err != nil {
				return nil, nil, ctxerr.Wrap(ctx, err, "building query args to list installed paths")
			}
			if err := sqlx.SelectContext(ctx, ds.reader(ctx), &installedPaths, stmt, args...); err != nil {
				return nil, nil, ctxerr.Wrap(ctx, err, "list software installed paths")
			}

			// store the installed paths with the proper software entry
			for _, path := range installedPaths {
				ver := bySoftwareID[path.SoftwareID]
				ver.InstalledPaths = append(ver.InstalledPaths, path.InstalledPath)
				if ver.Source == "apps" {
					ver.SignatureInformation = append(ver.SignatureInformation, fleet.PathSignatureInformation{
						InstalledPath:  path.InstalledPath,
						TeamIdentifier: path.TeamIdentifier,
					})
				}
			}
		}
	}

	perPage := opts.ListOptions.PerPage
	var metaData *fleet.PaginationMetadata
	if opts.ListOptions.IncludeMetadata {
		if perPage <= 0 {
			perPage = defaultSelectLimit
		}
		metaData = &fleet.PaginationMetadata{
			HasPreviousResults: opts.ListOptions.Page > 0,
			TotalResults:       titleCount,
		}
		if len(hostSoftwareList) > int(perPage) { //nolint:gosec // dismiss G115
			metaData.HasNextResults = true
			hostSoftwareList = hostSoftwareList[:len(hostSoftwareList)-1]
		}
	}

	software := make([]*fleet.HostSoftwareWithInstaller, 0, len(hostSoftwareList))
	for _, hs := range hostSoftwareList {
		hs := hs
		software = append(software, &hs.HostSoftwareWithInstaller)
	}
	return software, metaData, nil
}

func (ds *Datastore) SetHostSoftwareInstallResult(ctx context.Context, result *fleet.HostSoftwareInstallResultPayload) error {
	const stmt = `
		UPDATE
			host_software_installs
		SET
			pre_install_query_output = ?,
			install_script_exit_code = ?,
			install_script_output = ?,
			post_install_script_exit_code = ?,
			post_install_script_output = ?
		WHERE
			execution_id = ? AND
			host_id = ?
`

	truncateOutput := func(output *string) *string {
		if output != nil {
			output = ptr.String(truncateScriptResult(*output))
		}
		return output
	}

	res, err := ds.writer(ctx).ExecContext(ctx, stmt,
		truncateOutput(result.PreInstallConditionOutput),
		result.InstallScriptExitCode,
		truncateOutput(result.InstallScriptOutput),
		result.PostInstallScriptExitCode,
		truncateOutput(result.PostInstallScriptOutput),
		result.InstallUUID,
		result.HostID,
	)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "update host software installation result")
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ctxerr.Wrap(ctx, notFound("HostSoftwareInstall").WithName(result.InstallUUID), "host software installation not found")
	}
	return nil
}

func getInstalledByFleetSoftwareTitles(ctx context.Context, qc sqlx.QueryerContext, hostID uint) ([]fleet.SoftwareTitle, error) {
	// We are overloading vpp_apps_count to indicate whether installed title is a VPP app or not.
	const stmt = `
SELECT
	st.id,
	st.name,
	st.source,
	st.browser,
	st.bundle_identifier,
	0 as vpp_apps_count
FROM software_titles st
INNER JOIN software_installers si ON si.title_id = st.id
INNER JOIN host_software_installs hsi ON hsi.host_id = :host_id AND hsi.software_installer_id = si.id
WHERE hsi.removed = 0 AND hsi.status = :software_status_installed

UNION

SELECT
	st.id,
	st.name,
	st.source,
	st.browser,
	st.bundle_identifier,
	1 as vpp_apps_count
FROM software_titles st
INNER JOIN vpp_apps vap ON vap.title_id = st.id
INNER JOIN host_vpp_software_installs hvsi ON hvsi.host_id = :host_id AND hvsi.adam_id = vap.adam_id AND hvsi.platform = vap.platform
INNER JOIN nano_command_results ncr ON ncr.command_uuid = hvsi.command_uuid
WHERE hvsi.removed = 0 AND ncr.status = :mdm_status_acknowledged
`
	selectStmt, args, err := sqlx.Named(stmt, map[string]interface{}{
		"host_id":                   hostID,
		"software_status_installed": fleet.SoftwareInstalled,
		"mdm_status_acknowledged":   fleet.MDMAppleStatusAcknowledged,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build query to get installed software titles")
	}

	var titles []fleet.SoftwareTitle
	if err := sqlx.SelectContext(ctx, qc, &titles, selectStmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get installed software titles")
	}
	return titles, nil
}

func markHostSoftwareInstallsRemoved(ctx context.Context, ex sqlx.ExtContext, hostID uint, titleIDs []uint) error {
	const stmt = `
UPDATE host_software_installs hsi
INNER JOIN software_installers si ON hsi.software_installer_id = si.id
INNER JOIN software_titles st ON si.title_id = st.id
SET hsi.removed = 1
WHERE hsi.host_id = ? AND st.id IN (?)
`
	stmtExpanded, args, err := sqlx.In(stmt, hostID, titleIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build query args to mark host software install removed")
	}
	if _, err := ex.ExecContext(ctx, stmtExpanded, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "mark host software install removed")
	}
	return nil
}

func markHostVPPSoftwareInstallsRemoved(ctx context.Context, ex sqlx.ExtContext, hostID uint, titleIDs []uint) error {
	const stmt = `
UPDATE host_vpp_software_installs hvsi
INNER JOIN vpp_apps vap ON hvsi.adam_id = vap.adam_id AND hvsi.platform = vap.platform
INNER JOIN software_titles st ON vap.title_id = st.id
SET hvsi.removed = 1
WHERE hvsi.host_id = ? AND st.id IN (?)
`
	stmtExpanded, args, err := sqlx.In(stmt, hostID, titleIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build query args to mark host vpp software install removed")
	}
	if _, err := ex.ExecContext(ctx, stmtExpanded, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "mark host vpp software install removed")
	}
	return nil
}
