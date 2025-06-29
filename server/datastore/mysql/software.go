package mysql

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type softwareIDChecksum struct {
	ID               uint    `db:"id"`
	Checksum         string  `db:"checksum"`
	Name             string  `db:"name"`
	TitleID          *uint   `db:"title_id"`
	BundleIdentifier *string `db:"bundle_identifier"`
	Source           string  `db:"source"`
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

	toI, toD, err := hostSoftwareInstalledPathsDelta(hostID, reported, hsip, currS, ds.logger)
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
		SELECT t.id, t.host_id, t.software_id, t.installed_path, t.team_identifier, t.executable_sha256
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
	logger log.Logger,
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
		var sha256 string
		if r.ExecutableSHA256 != nil {
			sha256 = *r.ExecutableSHA256
		}
		key := fmt.Sprintf(
			"%s%s%s%s%s%s%s",
			r.InstalledPath, fleet.SoftwareFieldSeparator, r.TeamIdentifier, fleet.SoftwareFieldSeparator, sha256, fleet.SoftwareFieldSeparator, s.ToUniqueStr(),
		)
		iSPathLookup[key] = r

		// Anything stored but not reported should be deleted
		if _, ok := reported[key]; !ok {
			toDelete = append(toDelete, r.ID)
		}
	}

	for key := range reported {
		parts := strings.SplitN(key, fleet.SoftwareFieldSeparator, 4)
		installedPath, teamIdentifier, cdHash, unqStr := parts[0], parts[1], parts[2], parts[3]

		// Shouldn't be a common occurence ... everything 'reported' should be in the the software table
		// because this executes after 'ds.UpdateHostSoftware'
		s, ok := sUnqStrLook[unqStr]
		if !ok {
			level.Debug(logger).Log("msg", "skipping installed path for software not found", "host_id", hostID, "unq_str", unqStr)
			continue
		}

		if _, ok := iSPathLookup[key]; ok {
			// Nothing to do
			continue
		}

		var executableSHA256 *string
		if cdHash != "" {
			executableSHA256 = ptr.String(cdHash)
		}

		toInsert = append(toInsert, fleet.HostSoftwareInstalledPath{
			HostID:           hostID,
			SoftwareID:       s.ID,
			InstalledPath:    installedPath,
			TeamIdentifier:   teamIdentifier,
			ExecutableSHA256: executableSHA256,
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

	stmt := "INSERT INTO host_software_installed_paths (host_id, software_id, installed_path, team_identifier, executable_sha256) VALUES %s"
	batchSize := 500

	for i := 0; i < len(toInsert); i += batchSize {
		end := i + batchSize
		if end > len(toInsert) {
			end = len(toInsert)
		}
		batch := toInsert[i:end]

		var args []interface{}
		for _, v := range batch {
			args = append(args, v.HostID, v.SoftwareID, v.InstalledPath, v.TeamIdentifier, v.ExecutableSHA256)
		}

		placeHolders := strings.TrimSuffix(strings.Repeat("(?, ?, ?, ?, ?), ", len(batch)), ", ")
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

	existingSoftware, incomingByChecksum, existingTitlesForNewSoftware, existingBundleIDsToUpdate, err := ds.getExistingSoftware(ctx, current, incoming)
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

			// Copy incomingByChecksum because ds.insertNewInstalledHostSoftwareDB is modifying it and we
			// are runnning inside ds.withRetryTxx.
			incomingByChecksumCopy := make(map[string]fleet.Software, len(incomingByChecksum))
			for key, value := range incomingByChecksum {
				incomingByChecksumCopy[key] = value
			}

			inserted, err := ds.insertNewInstalledHostSoftwareDB(
				ctx, tx, hostID, existingSoftware, incomingByChecksumCopy, existingTitlesForNewSoftware, existingBundleIDsToUpdate,
			)
			if err != nil {
				return err
			}
			r.Inserted = inserted

			if err = checkForDeletedInstalledSoftware(ctx, tx, deleted, inserted, hostID); err != nil {
				return err
			}

			if err = updateModifiedHostSoftwareDB(ctx, tx, hostID, current, incoming, existingBundleIDsToUpdate, ds.minLastOpenedAtDiff, ds.logger); err != nil {
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

//nolint:unused
func updateExistingBundleIDs(ctx context.Context, tx sqlx.ExtContext, hostID uint, bundleIDsToSoftware map[string]fleet.Software) error {
	if len(bundleIDsToSoftware) == 0 {
		return nil
	}

	updateSoftwareStmt := `UPDATE software SET software.name = ?, software.name_source = 'bundle_4.67' WHERE software.bundle_identifier = ?`

	hostSoftwareStmt := `
		INSERT IGNORE INTO host_software 
			(host_id, software_id, last_opened_at)
		VALUES
			(?, (SELECT id FROM software WHERE bundle_identifier = ? AND name_source = 'bundle_4.67' ORDER BY id DESC LIMIT 1), ?)`

	for k, v := range bundleIDsToSoftware {
		if _, err := tx.ExecContext(ctx, updateSoftwareStmt, v.Name, k); err != nil {
			return ctxerr.Wrap(ctx, err, "update software names")
		}

		if _, err := tx.ExecContext(ctx, hostSoftwareStmt, hostID, v.ID, v.LastOpenedAt); err != nil {
			return ctxerr.Wrap(ctx, err, "insert host software")
		}
	}
	return nil
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
				deletedTitles[UniqueSoftwareTitleStr(BundleIdentifierOrName(d.BundleIdentifier, d.Name), d.Source)] = struct{}{}
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
			key := UniqueSoftwareTitleStr(BundleIdentifierOrName(bundleIdentifier, title.Name), title.Source)
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
	currentSoftware []softwareIDChecksum,
	incomingChecksumToSoftware map[string]fleet.Software,
	incomingChecksumToTitle map[string]fleet.SoftwareTitle,
	existingBundleIDsToUpdate map[string]fleet.Software,
	err error,
) {
	// Compute checksums for all incoming software, which we will use for faster retrieval, since checksum is a unique index
	incomingChecksumToSoftware = make(map[string]fleet.Software, len(current))
	newSoftware := make(map[string]struct{})
	bundleIDsToChecksum := make(map[string]string)
	bundleIDsToNames := make(map[string]string)
	existingBundleIDsToUpdate = make(map[string]fleet.Software)
	for uniqueName, s := range incoming {
		_, ok := current[uniqueName]
		if !ok {
			checksum, err := s.ComputeRawChecksum()
			if err != nil {
				return nil, nil, nil, nil, err
			}
			incomingChecksumToSoftware[string(checksum)] = s
			newSoftware[string(checksum)] = struct{}{}

			if s.BundleIdentifier != "" {
				bundleIDsToChecksum[s.BundleIdentifier] = string(checksum)
				bundleIDsToNames[s.BundleIdentifier] = s.Name
			}
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
			return nil, nil, nil, nil, err
		}
		for _, s := range currentSoftware {
			sw, ok := incomingChecksumToSoftware[s.Checksum]
			if !ok {
				// This should never happen. If it does, we have a bug.
				return nil, nil, nil, nil, ctxerr.New(
					ctx, fmt.Sprintf("current software: software not found for checksum %s", hex.EncodeToString([]byte(s.Checksum))),
				)
			}
			if s.BundleIdentifier != nil && s.Source == "apps" {
				if name, ok := bundleIDsToNames[*s.BundleIdentifier]; ok && name != s.Name {
					// Then this is a software whose name has changed, so we should update the name
					existingBundleIDsToUpdate[*s.BundleIdentifier] = sw
					continue
				}
			}
			delete(newSoftware, s.Checksum)
		}
	}

	if len(newSoftware) == 0 {
		return currentSoftware, incomingChecksumToSoftware, incomingChecksumToTitle, existingBundleIDsToUpdate, nil
	}

	// There's new software, so we try to get the titles already stored in `software_titles` for them.
	incomingChecksumToTitle, _, err = ds.getIncomingSoftwareChecksumsToExistingTitles(ctx, newSoftware, incomingChecksumToSoftware)
	if err != nil {
		return nil, nil, nil, nil, ctxerr.Wrap(ctx, err, "get incoming software checksums to existing titles")
	}

	for bid := range existingBundleIDsToUpdate {
		if cs, ok := bundleIDsToChecksum[bid]; ok {
			// we don't want this to be treated as a new software title, because then a new software
			// entry will be created. Instead, we want to update the existing entries with the new
			// names.
			delete(incomingChecksumToSoftware, cs)
		}
	}

	return currentSoftware, incomingChecksumToSoftware, incomingChecksumToTitle, existingBundleIDsToUpdate, nil
}

// getIncomingSoftwareChecksumsToExistingTitles loads the existing titles for the new incoming software.
// It returns a map of software checksums to existing software titles.
//
// To make best use of separate indexes, it runs two queries to get the existing titles from the DB:
//   - One query for software with bundle_identifier.
//   - One query for software without bundle_identifier.
func (ds *Datastore) getIncomingSoftwareChecksumsToExistingTitles(
	ctx context.Context,
	newSoftwareChecksums map[string]struct{},
	incomingChecksumToSoftware map[string]fleet.Software,
) (map[string]fleet.SoftwareTitle, map[string]fleet.Software, error) {
	var (
		incomingChecksumToTitle     = make(map[string]fleet.SoftwareTitle, len(newSoftwareChecksums))
		argsWithoutBundleIdentifier []any
		argsWithBundleIdentifier    []any
		uniqueTitleStrToChecksum    = make(map[string]string)
	)
	bundleIDsToIncomingNames := make(map[string]string)
	for checksum := range newSoftwareChecksums {
		sw := incomingChecksumToSoftware[checksum]
		if sw.BundleIdentifier != "" {
			bundleIDsToIncomingNames[sw.BundleIdentifier] = sw.Name
			argsWithBundleIdentifier = append(argsWithBundleIdentifier, sw.BundleIdentifier)
		} else {
			argsWithoutBundleIdentifier = append(argsWithoutBundleIdentifier, sw.Name, sw.Source, sw.Browser)
		}
		// Map software title identifier to software checksums so that we can map checksums to actual titles later.
		uniqueTitleStrToChecksum[UniqueSoftwareTitleStr(
			BundleIdentifierOrName(sw.BundleIdentifier, sw.Name), sw.Source, sw.Browser,
		)] = checksum
	}

	// Get titles for software without bundle_identifier.
	if len(argsWithoutBundleIdentifier) > 0 {
		whereClause := strings.TrimSuffix(
			strings.Repeat(`
			  (
			    (name = ? AND source = ? AND browser = ?)
			  ) OR`, len(argsWithoutBundleIdentifier)/3), " OR",
		)
		stmt := fmt.Sprintf(
			"SELECT id, name, source, browser FROM software_titles WHERE %s",
			whereClause,
		)
		var existingSoftwareTitlesForNewSoftwareWithoutBundleIdentifier []fleet.SoftwareTitle
		if err := sqlx.SelectContext(ctx,
			ds.reader(ctx),
			&existingSoftwareTitlesForNewSoftwareWithoutBundleIdentifier,
			stmt,
			argsWithoutBundleIdentifier...,
		); err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "get existing titles without bundle identifier")
		}
		for _, title := range existingSoftwareTitlesForNewSoftwareWithoutBundleIdentifier {
			checksum, ok := uniqueTitleStrToChecksum[UniqueSoftwareTitleStr(title.Name, title.Source, title.Browser)]
			if ok {
				incomingChecksumToTitle[checksum] = title
			}
		}
	}

	// Get titles for software with bundle_identifier
	existingBundleIDsToUpdate := make(map[string]fleet.Software)
	if len(argsWithBundleIdentifier) > 0 {
		// no-op code change
		incomingChecksumToTitle = make(map[string]fleet.SoftwareTitle, len(newSoftwareChecksums))
		stmtBundleIdentifier := `SELECT id, name, source, browser, bundle_identifier FROM software_titles WHERE bundle_identifier IN (?)`
		stmtBundleIdentifier, argsWithBundleIdentifier, err := sqlx.In(stmtBundleIdentifier, argsWithBundleIdentifier)
		if err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "build query to existing titles with bundle_identifier")
		}
		var existingSoftwareTitlesForNewSoftwareWithBundleIdentifier []fleet.SoftwareTitle
		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &existingSoftwareTitlesForNewSoftwareWithBundleIdentifier, stmtBundleIdentifier, argsWithBundleIdentifier...); err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "get existing titles with bundle_identifier")
		}
		// Map software titles to software checksums.
		for _, title := range existingSoftwareTitlesForNewSoftwareWithBundleIdentifier {
			uniqueStrWithoutName := UniqueSoftwareTitleStr(*title.BundleIdentifier, title.Source, title.Browser)
			withoutNameCS, withoutName := uniqueTitleStrToChecksum[uniqueStrWithoutName]

			if withoutName {
				incomingChecksumToTitle[withoutNameCS] = title
			}
		}
	}

	return incomingChecksumToTitle, existingBundleIDsToUpdate, nil
}

// BundleIdentifierOrName returns the bundle identifier if it is not empty, otherwise name
func BundleIdentifierOrName(bundleIdentifier, name string) string {
	if bundleIdentifier != "" {
		return bundleIdentifier
	}
	return name
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

// insertNewInstalledHostSoftwareDB inserts host_software that is in softwareChecksums map,
// but not in existingSoftware. It also inserts any new software titles that are needed.
//
// It returns the inserted software on the host.
func (ds *Datastore) insertNewInstalledHostSoftwareDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	hostID uint,
	existingSoftware []softwareIDChecksum,
	softwareChecksums map[string]fleet.Software,
	existingTitlesForNewSoftware map[string]fleet.SoftwareTitle,
	existingBundleIDsToUpdate map[string]fleet.Software,
) ([]fleet.Software, error) {
	var insertsHostSoftware []interface{}
	var insertedSoftware []fleet.Software
	existingTitleNames := make(map[uint]string)
	for _, s := range existingSoftware {
		if s.TitleID != nil {
			existingTitleNames[*s.TitleID] = s.Name
		}
	}

	// First, we remove incoming software that already exists in the software table.
	if len(softwareChecksums) > 0 {
		for _, s := range existingSoftware {
			software, ok := softwareChecksums[s.Checksum]
			if !ok {
				if s.BundleIdentifier != nil {
					// If this is a softwarea we know we have to update (rename), then it's expected
					// that we wouldn't find it in softwareChecksums (we deleted it from that map
					// earlier in ds.getExistingSoftware.
					if _, ok := existingBundleIDsToUpdate[*s.BundleIdentifier]; ok {
						continue
					}
				}
				return nil, ctxerr.New(ctx, fmt.Sprintf("existing software: software not found for checksum %q", hex.EncodeToString([]byte(s.Checksum))))
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
				return nil, ctxerr.New(ctx, fmt.Sprintf("updated existing software: software not found for checksum %s", hex.EncodeToString([]byte(s.Checksum))))
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
	stmt, args, err := sqlx.In("SELECT name, id, checksum, title_id, bundle_identifier, source FROM software WHERE checksum IN (?)", checksums)
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
	existingBundleIDsToUpdate map[string]fleet.Software,
	minLastOpenedAtDiff time.Duration,
	logger log.Logger,
) error {
	var keysToUpdate []string
	for key, newSw := range incomingMap {
		curSw, ok := currentMap[key]
		// software must exist in current map for us to update it.
		if !ok {
			continue
		}
		// if the new software has no last opened timestamp, we only
		// update if the current software has no last opened timestamp
		// and is marked as having a name change.
		if newSw.LastOpenedAt == nil {
			if _, ok := existingBundleIDsToUpdate[newSw.BundleIdentifier]; ok && curSw.LastOpenedAt == nil {
				keysToUpdate = append(keysToUpdate, key)
			}
			// Log cases where the new software has no last opened timestamp, the current software does,
			// and the software is marked as having a name change.
			if ok && curSw.LastOpenedAt != nil {
				level.Warn(logger).Log(
					"msg", "updateModifiedHostSoftwareDB: last opened at is nil for new software, but not for current software",
					"new_software", newSw.Name, "current_software", curSw.Name,
					"bundle_identifier", newSw.BundleIdentifier,
				)
			}
			continue
		}
		// update if the new software has been opened more recently.
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
// Used on software/versions not software/titles
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
			if opts.IncludeCVEScores && !opts.WithoutVulnerabilityDetails {
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

		updateNamesStmt := `
		UPDATE software_titles st
		JOIN software s on st.id = s.title_id
		SET st.name = (
			SELECT
				software.name
			FROM
				software
			WHERE
				software.bundle_identifier = st.bundle_identifier
			ORDER BY
				id DESC
			LIMIT 1
		)
		WHERE 
			st.bundle_identifier IS NOT NULL AND 
			st.bundle_identifier != '' AND
			s.name_source = 'bundle_4.67'
		`
		res, err = tx.ExecContext(ctx, updateNamesStmt)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "update software title names")
		}
		n, _ = res.RowsAffected()
		level.Debug(ds.logger).Log("msg", "update software title names", "rows_affected", n)

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

// Deprecated: ** DEPRECATED **
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

type hostSoftware struct {
	fleet.HostSoftwareWithInstaller

	LastInstallInstalledAt         *time.Time `db:"last_install_installed_at"`
	LastInstallInstallUUID         *string    `db:"last_install_install_uuid"`
	LastUninstallUninstalledAt     *time.Time `db:"last_uninstall_uninstalled_at"`
	LastUninstallScriptExecutionID *string    `db:"last_uninstall_script_execution_id"`

	ExitCode           *int       `db:"exit_code"`
	LastOpenedAt       *time.Time `db:"last_opened_at"`
	BundleIdentifier   *string    `db:"bundle_identifier"`
	Version            *string    `db:"version"`
	SoftwareID         *uint      `db:"software_id"`
	SoftwareSource     *string    `db:"software_source"`
	InstallerID        *uint      `db:"installer_id"`
	PackageSelfService *bool      `db:"package_self_service"`
	PackageName        *string    `db:"package_name"`
	PackagePlatform    *string    `db:"package_platform"`
	PackageVersion     *string    `db:"package_version"`
	VPPAppSelfService  *bool      `db:"vpp_app_self_service"`
	VPPAppAdamID       *string    `db:"vpp_app_adam_id"`
	VPPAppVersion      *string    `db:"vpp_app_version"`
	VPPAppPlatform     *string    `db:"vpp_app_platform"`
	VPPAppIconURL      *string    `db:"vpp_app_icon_url"`

	VulnerabilitiesList   *string `db:"vulnerabilities_list"`
	SoftwareIDList        *string `db:"software_id_list"`
	SoftwareSourceList    *string `db:"software_source_list"`
	VersionList           *string `db:"version_list"`
	BundleIdentifierList  *string `db:"bundle_identifier_list"`
	VPPAppSelfServiceList *string `db:"vpp_app_self_service_list"`
	VPPAppAdamIDList      *string `db:"vpp_app_adam_id_list"`
	VPPAppVersionList     *string `db:"vpp_app_version_list"`
	VPPAppPlatformList    *string `db:"vpp_app_platform_list"`
	VPPAppIconUrlList     *string `db:"vpp_app_icon_url_list"`
}

func hostInstalledSoftware(ds *Datastore, ctx context.Context, hostID uint) ([]*hostSoftware, error) {
	installedSoftwareStmt := `
		SELECT
			software_titles.id AS id,
			host_software.software_id AS software_id,
			host_software.last_opened_at,
			software.source AS software_source,
			software.version AS version,
			software.bundle_identifier AS bundle_identifier
		FROM 
			host_software
		INNER JOIN
			software ON host_software.software_id = software.id
		INNER JOIN
			software_titles ON software.title_id = software_titles.id
		WHERE
			host_software.host_id = ?
	`

	var hostInstalledSoftware []*hostSoftware
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &hostInstalledSoftware, installedSoftwareStmt, hostID)
	if err != nil {
		return nil, err
	}

	return hostInstalledSoftware, nil
}

func hostSoftwareInstalls(ds *Datastore, ctx context.Context, hostID uint) ([]*hostSoftware, error) {
	softwareInstallsStmt := `
        WITH upcoming_software_install AS (
            SELECT
                ua.execution_id AS last_install_install_uuid,
                ua.created_at AS last_install_installed_at,
                siua.software_installer_id AS installer_id,
                'pending_install' AS status
            FROM
                upcoming_activities ua
            INNER JOIN
                software_install_upcoming_activities siua ON ua.id = siua.upcoming_activity_id
            LEFT JOIN (
                upcoming_activities ua2
                INNER JOIN software_install_upcoming_activities siua2 ON ua2.id = siua2.upcoming_activity_id
            ) ON ua.host_id = ua2.host_id AND
                siua.software_installer_id = siua2.software_installer_id AND
                ua.activity_type = ua2.activity_type AND
                (ua2.priority < ua.priority OR ua2.created_at > ua.created_at)
            WHERE
                ua.host_id = ? AND
                ua.activity_type = 'software_install' AND
                ua2.id IS NULL
        ),
        last_software_install AS (
            SELECT
                hsi.execution_id AS last_install_install_uuid,
                hsi.updated_at AS last_install_installed_at,
                hsi.software_installer_id AS installer_id,
                hsi.status AS status
            FROM
                host_software_installs hsi
            LEFT JOIN
                host_software_installs hsi2 ON hsi.host_id = hsi2.host_id AND
                    hsi.software_installer_id = hsi2.software_installer_id AND
                    hsi.uninstall = hsi2.uninstall AND
                    hsi2.removed = 0 AND
					hsi2.canceled = 0 AND
                    hsi2.host_deleted_at IS NULL AND
                    (hsi.created_at < hsi2.created_at OR (hsi.created_at = hsi2.created_at AND hsi.id < hsi2.id))
            WHERE
                hsi.host_id = ? AND
                hsi.removed = 0 AND
				hsi.canceled = 0 AND
                hsi.uninstall = 0 AND
                hsi.host_deleted_at IS NULL AND
                hsi2.id IS NULL AND
                NOT EXISTS (
                    SELECT 1
                    FROM
                        upcoming_activities ua
                    INNER JOIN
                        software_install_upcoming_activities siua ON ua.id = siua.upcoming_activity_id
                    WHERE
                        ua.host_id = hsi.host_id AND
                        siua.software_installer_id = hsi.software_installer_id AND
                        ua.activity_type = 'software_install'
                )
        )
        SELECT
			software_installers.id AS installer_id,
			software_installers.self_service AS package_self_service,
			software_titles.id AS id,
			lsia.*
		FROM
			(SELECT * FROM upcoming_software_install UNION SELECT * FROM last_software_install) AS lsia
		INNER JOIN
			software_installers ON lsia.installer_id = software_installers.id
		INNER JOIN
			software_titles ON software_installers.title_id = software_titles.id
    `
	var softwareInstalls []*hostSoftware
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &softwareInstalls, softwareInstallsStmt, hostID, hostID)
	if err != nil {
		return nil, err
	}

	return softwareInstalls, nil
}

func hostSoftwareUninstalls(ds *Datastore, ctx context.Context, hostID uint) ([]*hostSoftware, error) {
	softwareUninstallsStmt := `
        WITH upcoming_software_uninstall AS (
            SELECT
                ua.execution_id AS last_uninstall_script_execution_id,
                ua.created_at AS last_uninstall_uninstalled_at,
                siua.software_installer_id AS installer_id,
                'pending_uninstall' AS status
            FROM
                upcoming_activities ua
            INNER JOIN
                software_install_upcoming_activities siua ON ua.id = siua.upcoming_activity_id
            LEFT JOIN (
                upcoming_activities ua2
                INNER JOIN software_install_upcoming_activities siua2 ON ua2.id = siua2.upcoming_activity_id
            ) ON ua.host_id = ua2.host_id AND
                siua.software_installer_id = siua2.software_installer_id AND
                ua.activity_type = ua2.activity_type AND
                (ua2.priority < ua.priority OR ua2.created_at > ua.created_at)
            WHERE
                ua.host_id = ? AND
                ua.activity_type = 'software_uninstall' AND
                ua2.id IS NULL
        ),
        last_software_uninstall AS (
            SELECT
                hsi.execution_id AS last_uninstall_script_execution_id,
                hsi.updated_at AS last_uninstall_uninstalled_at,
                hsi.software_installer_id AS installer_id,
                hsi.status AS status
            FROM
                host_software_installs hsi
            LEFT JOIN
                host_software_installs hsi2 ON hsi.host_id = hsi2.host_id AND
                    hsi.software_installer_id = hsi2.software_installer_id AND
                    hsi.uninstall = hsi2.uninstall AND
                    hsi2.removed = 0 AND
					hsi2.canceled = 0 AND
                    hsi2.host_deleted_at IS NULL AND
                    (hsi.created_at < hsi2.created_at OR (hsi.created_at = hsi2.created_at AND hsi.id < hsi2.id))
            WHERE
                hsi.host_id = ? AND
                hsi.removed = 0 AND
                hsi.uninstall = 1 AND
				hsi.canceled = 0 AND
                hsi.host_deleted_at IS NULL AND
                hsi2.id IS NULL AND
                NOT EXISTS (
                    SELECT 1
                    FROM
                        upcoming_activities ua
                    INNER JOIN
                        software_install_upcoming_activities siua ON ua.id = siua.upcoming_activity_id
                    WHERE
                        ua.host_id = hsi.host_id AND
                        siua.software_installer_id = hsi.software_installer_id AND
                        ua.activity_type = 'software_uninstall'
                )
        )
        SELECT
			software_installers.id AS installer_id,
			software_titles.id AS id,
			host_script_results.exit_code AS exit_code,
			lsua.*
		FROM
            (SELECT * FROM upcoming_software_uninstall UNION SELECT * FROM last_software_uninstall) AS lsua
		INNER JOIN
			software_installers ON lsua.installer_id = software_installers.id
		INNER JOIN
			software_titles ON software_installers.title_id = software_titles.id
		LEFT OUTER JOIN
			host_script_results ON host_script_results.host_id = ? AND host_script_results.execution_id = lsua.last_uninstall_script_execution_id
    `
	var softwareUninstalls []*hostSoftware
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &softwareUninstalls, softwareUninstallsStmt, hostID, hostID, hostID)
	if err != nil {
		return nil, err
	}

	return softwareUninstalls, nil
}

func filterSoftwareInstallersByLabel(
	ds *Datastore,
	ctx context.Context,
	host *fleet.Host,
	bySoftwareTitleID map[uint]*hostSoftware,
) (map[uint]*hostSoftware, error) {
	if len(bySoftwareTitleID) == 0 {
		return bySoftwareTitleID, nil
	}

	filteredbySoftwareTitleID := make(map[uint]*hostSoftware, len(bySoftwareTitleID))
	softwareInstallersIDsToCheck := make([]uint, 0, len(bySoftwareTitleID))

	for _, st := range bySoftwareTitleID {
		if st.InstallerID != nil {
			softwareInstallersIDsToCheck = append(softwareInstallersIDsToCheck, *st.InstallerID)
		}
	}

	if len(softwareInstallersIDsToCheck) > 0 {
		labelSqlFilter := `
			WITH no_labels AS (
				SELECT
					software_installers.id AS installer_id,
					0 AS count_installer_labels,
					0 AS count_host_labels,
					0 AS count_host_updated_after_labels
				FROM
					software_installers
				WHERE NOT EXISTS (
					SELECT 1
					FROM software_installer_labels
					WHERE software_installer_labels.software_installer_id = software_installers.id
				)
			),
			include_any AS (
				SELECT
					software_installers.id AS installer_id,
					COUNT(*) AS count_installer_labels,
					COUNT(label_membership.label_id) AS count_host_labels,
					0 AS count_host_updated_after_labels
				FROM
					software_installers
				INNER JOIN software_installer_labels
					ON software_installer_labels.software_installer_id = software_installers.id AND software_installer_labels.exclude = 0
				LEFT JOIN label_membership
					ON label_membership.label_id = software_installer_labels.label_id
					AND label_membership.host_id = :host_id
				GROUP BY
					software_installers.id
				HAVING
					COUNT(*) > 0 AND COUNT(label_membership.label_id) > 0
			),
			exclude_any AS (
				SELECT
					software_installers.id AS installer_id,
					COUNT(software_installer_labels.label_id) AS count_installer_labels,
					COUNT(label_membership.label_id) AS count_host_labels,
					SUM(
						CASE
							WHEN labels.created_at IS NOT NULL AND :host_label_updated_at >= labels.created_at THEN 1
							ELSE 0
						END
					) AS count_host_updated_after_labels
				FROM
					software_installers
				INNER JOIN software_installer_labels
					ON software_installer_labels.software_installer_id = software_installers.id AND software_installer_labels.exclude = 1
				INNER JOIN labels
					ON labels.id = software_installer_labels.label_id
				LEFT JOIN label_membership
					ON label_membership.label_id = software_installer_labels.label_id
					AND label_membership.host_id = :host_id
				GROUP BY
					software_installers.id
				HAVING
					COUNT(*) > 0
					AND COUNT(*) = SUM(
						CASE
							WHEN labels.created_at IS NOT NULL AND :host_label_updated_at >= labels.created_at THEN 1
							ELSE 0
						END
					)
					AND COUNT(label_membership.label_id) = 0
			)
			SELECT
				software_installers.id AS id,
				software_installers.title_id AS title_id
			FROM
				software_installers
			LEFT JOIN no_labels
				ON no_labels.installer_id = software_installers.id
			LEFT JOIN include_any
				ON include_any.installer_id = software_installers.id
			LEFT JOIN exclude_any
				ON exclude_any.installer_id = software_installers.id
			WHERE
				software_installers.id IN (:software_installer_ids)
				AND (
					no_labels.installer_id IS NOT NULL
					OR include_any.installer_id IS NOT NULL
					OR exclude_any.installer_id IS NOT NULL
				)
		`
		labelSqlFilter, args, err := sqlx.Named(labelSqlFilter, map[string]any{
			"host_id":                host.ID,
			"host_label_updated_at":  host.LabelUpdatedAt,
			"software_installer_ids": softwareInstallersIDsToCheck,
		})
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "filterSoftwareInstallersByLabel building named query args")
		}

		labelSqlFilter, args, err = sqlx.In(labelSqlFilter, args...)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "filterSoftwareInstallersByLabel building in query args")
		}

		labelSqlFilter = ds.reader(ctx).Rebind(labelSqlFilter)

		var validSoftwareInstallers []struct {
			Id      uint `db:"id"`
			TitleId uint `db:"title_id"`
		}
		err = sqlx.SelectContext(ctx, ds.reader(ctx), &validSoftwareInstallers, labelSqlFilter, args...)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "filterSoftwareInstallersByLabel executing query")
		}

		// go through the returned list of validSoftwareInstaller and add all the titles that meet the label criteria to be returned
		for _, validSoftwareInstaller := range validSoftwareInstallers {
			filteredbySoftwareTitleID[validSoftwareInstaller.TitleId] = bySoftwareTitleID[validSoftwareInstaller.TitleId]
		}
	}

	return filteredbySoftwareTitleID, nil
}

func filterVppAppsByLabel(
	ds *Datastore,
	ctx context.Context,
	host *fleet.Host,
	byVppAppID map[string]*hostSoftware,
	hostVPPInstalledTitles map[uint]*hostSoftware,
) (map[string]*hostSoftware, map[string]*hostSoftware, error) {
	filteredbyVppAppID := make(map[string]*hostSoftware, len(byVppAppID))
	otherVppAppsInInventory := make(map[string]*hostSoftware, len(hostVPPInstalledTitles))
	// This is the list of VPP apps that are installed on the host by fleet or the user
	// that we want to check are in scope or not
	vppAppIDsToCheck := make([]string, 0, len(byVppAppID))

	for _, st := range byVppAppID {
		vppAppIDsToCheck = append(vppAppIDsToCheck, *st.VPPAppAdamID)
	}
	for _, st := range hostVPPInstalledTitles {
		if st.VPPAppAdamID != nil {
			vppAppIDsToCheck = append(vppAppIDsToCheck, *st.VPPAppAdamID)
		}
	}

	if len(vppAppIDsToCheck) > 0 {
		var globalOrTeamID uint
		if host.TeamID != nil {
			globalOrTeamID = *host.TeamID
		}

		labelSqlFilter := `
			WITH no_labels AS (
				SELECT
					vpp_apps_teams.id AS team_id,
					0 AS count_installer_labels,
					0 AS count_host_labels,
					0 as count_host_updated_after_labels
				FROM
					vpp_apps_teams
				WHERE NOT EXISTS (
					SELECT 1
					FROM vpp_app_team_labels
					WHERE vpp_app_team_labels.vpp_app_team_id = vpp_apps_teams.id
				)
			),
			include_any AS (
				SELECT
					vpp_apps_teams.id AS team_id,
					COUNT(vpp_app_team_labels.label_id) AS count_installer_labels,
					COUNT(label_membership.label_id) AS count_host_labels,
					0 as count_host_updated_after_labels
				FROM
					vpp_apps_teams
				INNER JOIN vpp_app_team_labels
					ON vpp_app_team_labels.vpp_app_team_id = vpp_apps_teams.id AND vpp_app_team_labels.exclude = 0
				LEFT JOIN label_membership
					ON label_membership.label_id = vpp_app_team_labels.label_id
					AND label_membership.host_id = :host_id
				GROUP BY
					vpp_apps_teams.id
				HAVING
					count_installer_labels > 0 AND count_host_labels > 0
			),
			exclude_any AS (
				SELECT
					vpp_apps_teams.id AS team_id,
					COUNT(vpp_app_team_labels.label_id) AS count_installer_labels,
					COUNT(label_membership.label_id) AS count_host_labels,
					SUM(
						CASE
							WHEN labels.created_at IS NOT NULL AND labels.label_membership_type = 0 AND :host_label_updated_at >= labels.created_at THEN 1
							WHEN labels.created_at IS NOT NULL AND labels.label_membership_type = 1 THEN 1
							ELSE 0
						END
					) AS count_host_updated_after_labels
				FROM
					vpp_apps_teams
				INNER JOIN vpp_app_team_labels
					ON vpp_app_team_labels.vpp_app_team_id = vpp_apps_teams.id AND vpp_app_team_labels.exclude = 1
				INNER JOIN labels
					ON labels.id = vpp_app_team_labels.label_id
				LEFT OUTER JOIN label_membership
					ON label_membership.label_id = vpp_app_team_labels.label_id AND label_membership.host_id = :host_id
				GROUP BY
					vpp_apps_teams.id
				HAVING
					count_installer_labels > 0
					AND count_installer_labels = count_host_updated_after_labels
					AND count_host_labels = 0
			)
			SELECT
				vpp_apps.adam_id AS adam_id,
				vpp_apps.title_id AS title_id
			FROM
				vpp_apps
			INNER JOIN
				vpp_apps_teams ON vpp_apps.adam_id = vpp_apps_teams.adam_id AND vpp_apps.platform = vpp_apps_teams.platform AND vpp_apps_teams.global_or_team_id = :global_or_team_id
			LEFT JOIN no_labels
				ON no_labels.team_id = vpp_apps_teams.id
			LEFT JOIN include_any
				ON include_any.team_id = vpp_apps_teams.id
			LEFT JOIN exclude_any
				ON exclude_any.team_id = vpp_apps_teams.id
			WHERE
				vpp_apps.adam_id IN (:vpp_app_adam_ids)
				AND (
					no_labels.team_id IS NOT NULL
					OR include_any.team_id IS NOT NULL
					OR exclude_any.team_id IS NOT NULL
				)
		`

		labelSqlFilter, args, err := sqlx.Named(labelSqlFilter, map[string]any{
			"host_id":               host.ID,
			"host_label_updated_at": host.LabelUpdatedAt,
			"vpp_app_adam_ids":      vppAppIDsToCheck,
			"global_or_team_id":     globalOrTeamID,
		})
		if err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "filterVppAppsByLabel building named query args")
		}

		labelSqlFilter, args, err = sqlx.In(labelSqlFilter, args...)
		if err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "filterVppAppsByLabel building in query args")
		}

		var validVppApps []struct {
			AdamId  string `db:"adam_id"`
			TitleId uint   `db:"title_id"`
		}
		err = sqlx.SelectContext(ctx, ds.reader(ctx), &validVppApps, labelSqlFilter, args...)
		if err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "filterVppAppsByLabel executing query")
		}

		// differentiate between VPP apps that were installed by Fleet (show install details +
		// ability to reinstall in self-service) vs. VPP apps that Fleet knows about but either
		// weren't installed by Fleet or were installed by Fleet but are no longer in scope
		// (treat as in inventory and not re-installable in self-service)
		for _, validAppApp := range validVppApps {
			if _, ok := byVppAppID[validAppApp.AdamId]; ok {
				filteredbyVppAppID[validAppApp.AdamId] = byVppAppID[validAppApp.AdamId]
			} else if svpp, ok := hostVPPInstalledTitles[validAppApp.TitleId]; ok {
				otherVppAppsInInventory[validAppApp.AdamId] = svpp
			}
		}
	}

	return filteredbyVppAppID, otherVppAppsInInventory, nil
}

func hostVPPInstalls(ds *Datastore, ctx context.Context, hostID uint, globalOrTeamID uint, selfServiceOnly bool, isMDMEnrolled bool) ([]*hostSoftware, error) {
	var selfServiceFilter string
	if selfServiceOnly {
		if isMDMEnrolled {
			selfServiceFilter = "(vat.self_service = 1) AND "
		} else {
			selfServiceFilter = "FALSE AND "
		}
	}
	vppInstallsStmt := fmt.Sprintf(`
        (   -- upcoming_vpp_install
            SELECT
				vpp_apps.title_id AS id,
                ua.execution_id AS last_install_install_uuid,
                ua.created_at AS last_install_installed_at,
                vaua.adam_id AS vpp_app_adam_id,
				vat.self_service AS vpp_app_self_service,
                'pending_install' AS status
            FROM
                upcoming_activities ua
            INNER JOIN
                vpp_app_upcoming_activities vaua ON ua.id = vaua.upcoming_activity_id
            LEFT JOIN (
                upcoming_activities ua2
                INNER JOIN vpp_app_upcoming_activities vaua2 ON ua2.id = vaua2.upcoming_activity_id
            ) ON ua.host_id = ua2.host_id AND
                vaua.adam_id = vaua2.adam_id AND
                vaua.platform = vaua2.platform AND
                ua.activity_type = ua2.activity_type AND
                (ua2.priority < ua.priority OR ua2.created_at > ua.created_at)
			LEFT JOIN
				vpp_apps_teams vat ON vaua.adam_id = vat.adam_id AND vaua.platform = vat.platform AND vat.global_or_team_id = :global_or_team_id
			INNER JOIN
				vpp_apps ON vaua.adam_id = vpp_apps.adam_id AND vaua.platform = vpp_apps.platform
			WHERE
				-- selfServiceFilter
				%s
                ua.host_id = :host_id AND
                ua.activity_type = 'vpp_app_install' AND
                ua2.id IS NULL
        ) UNION (
		 	-- last_vpp_install
            SELECT
				vpp_apps.title_id AS id,
                hvsi.command_uuid AS last_install_install_uuid,
                hvsi.created_at AS last_install_installed_at,
                hvsi.adam_id AS vpp_app_adam_id,
				vat.self_service AS vpp_app_self_service,
				-- vppAppHostStatusNamedQuery(hvsi, ncr, status)
                %s
            FROM
                host_vpp_software_installs hvsi
            LEFT JOIN
                nano_command_results ncr ON ncr.command_uuid = hvsi.command_uuid
            LEFT JOIN
                host_vpp_software_installs hvsi2 ON hvsi.host_id = hvsi2.host_id AND
                    hvsi.adam_id = hvsi2.adam_id AND
                    hvsi.platform = hvsi2.platform AND
                    hvsi2.removed = 0 AND
					hvsi2.canceled = 0 AND
                    (hvsi.created_at < hvsi2.created_at OR (hvsi.created_at = hvsi2.created_at AND hvsi.id < hvsi2.id))
			INNER JOIN
				vpp_apps_teams vat ON hvsi.adam_id = vat.adam_id AND hvsi.platform = vat.platform AND vat.global_or_team_id = :global_or_team_id
            INNER JOIN
				vpp_apps ON hvsi.adam_id = vpp_apps.adam_id AND hvsi.platform = vpp_apps.platform
			WHERE
				-- selfServiceFilter
				%s
                hvsi.host_id = :host_id AND
                hvsi.removed = 0 AND
				hvsi.canceled = 0 AND
                hvsi2.id IS NULL AND
                NOT EXISTS (
                    SELECT 1
                    FROM
                        upcoming_activities ua
                    INNER JOIN
                        vpp_app_upcoming_activities vaua ON ua.id = vaua.upcoming_activity_id
                    WHERE
                        ua.host_id = hvsi.host_id AND
                        vaua.adam_id = hvsi.adam_id AND
                        vaua.platform = hvsi.platform AND
                        ua.activity_type = 'vpp_app_install'
                )
        )
    `, selfServiceFilter, vppAppHostStatusNamedQuery("hvsi", "ncr", "status"), selfServiceFilter)
	vppInstallsStmt, args, err := sqlx.Named(vppInstallsStmt, map[string]any{
		"host_id":                   hostID,
		"global_or_team_id":         globalOrTeamID,
		"software_status_installed": fleet.SoftwareInstalled,
		"mdm_status_acknowledged":   fleet.MDMAppleStatusAcknowledged,
		"mdm_status_error":          fleet.MDMAppleStatusError,
		"mdm_status_format_error":   fleet.MDMAppleStatusCommandFormatError,
		"software_status_failed":    fleet.SoftwareInstallFailed,
		"software_status_pending":   fleet.SoftwareInstallPending,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build named query for host vpp installs")
	}
	var vppInstalls []*hostSoftware
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &vppInstalls, vppInstallsStmt, args...)
	if err != nil {
		return nil, err
	}

	for _, hs := range vppInstalls {
		fmt.Printf("hs: %+v\n", hs)
	}

	return vppInstalls, nil
}

func pushVersion(softwareIDStr string, softwareTitleRecord *hostSoftware, hostInstalledSoftware hostSoftware) {
	seperator := ","
	if softwareTitleRecord.SoftwareIDList == nil {
		softwareTitleRecord.SoftwareIDList = ptr.String("")
		softwareTitleRecord.SoftwareSourceList = ptr.String("")
		softwareTitleRecord.VersionList = ptr.String("")
		softwareTitleRecord.BundleIdentifierList = ptr.String("")
		seperator = ""
	}
	softwareIDList := strings.Split(*softwareTitleRecord.SoftwareIDList, ",")
	found := false
	for _, id := range softwareIDList {
		if id == softwareIDStr {
			found = true
			break
		}
	}
	if !found {
		*softwareTitleRecord.SoftwareIDList += seperator + softwareIDStr
		if hostInstalledSoftware.SoftwareSource != nil {
			*softwareTitleRecord.SoftwareSourceList += seperator + *hostInstalledSoftware.SoftwareSource
		}
		*softwareTitleRecord.VersionList += seperator + *hostInstalledSoftware.Version
		*softwareTitleRecord.BundleIdentifierList += seperator + *hostInstalledSoftware.BundleIdentifier
	}
}

func hostInstalledVpps(ds *Datastore, ctx context.Context, hostID uint) ([]*hostSoftware, error) {
	vppInstalledStmt := `
		SELECT
			vpp_apps.title_id AS id,
			hvsi.command_uuid AS last_install_install_uuid,
			hvsi.created_at AS last_install_installed_at,
			vpp_apps.adam_id AS vpp_app_adam_id,
			vpp_apps.latest_version AS vpp_app_version,
			vpp_apps.platform as vpp_app_platform,
			NULLIF(vpp_apps.icon_url, '') as vpp_app_icon_url,
			vpp_apps_teams.self_service AS vpp_app_self_service,
			'installed' AS status
		FROM
			host_vpp_software_installs hvsi
		INNER JOIN
			vpp_apps ON hvsi.adam_id = vpp_apps.adam_id AND hvsi.platform = vpp_apps.platform
		INNER JOIN
			vpp_apps_teams ON vpp_apps.adam_id = vpp_apps_teams.adam_id AND vpp_apps.platform = vpp_apps_teams.platform
		WHERE
			hvsi.host_id = ?
	`
	var vppInstalled []*hostSoftware
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &vppInstalled, vppInstalledStmt, hostID)
	if err != nil {
		return nil, err
	}

	return vppInstalled, nil
}

// hydrated is the base record from the db
// it contains most of the information we need to return back, however,
// we need to copy over the install/uninstall data from the softwareTitle we fetched
// from hostSoftwareInstalls and hostSoftwareUninstalls
func hydrateHostSoftwareRecordFromDb(hydrated *hostSoftware, softwareTitle *hostSoftware) {
	var version,
		platform string
	if hydrated.PackageVersion != nil {
		version = *hydrated.PackageVersion
	}
	if hydrated.PackagePlatform != nil {
		platform = *hydrated.PackagePlatform
	}
	hydrated.SoftwarePackage = &fleet.SoftwarePackageOrApp{
		Name:        *hydrated.PackageName,
		Version:     version,
		Platform:    platform,
		SelfService: hydrated.PackageSelfService,
	}

	// promote the last install info to the proper destination fields
	if softwareTitle.LastInstallInstallUUID != nil && *softwareTitle.LastInstallInstallUUID != "" {
		hydrated.SoftwarePackage.LastInstall = &fleet.HostSoftwareInstall{
			InstallUUID: *softwareTitle.LastInstallInstallUUID,
		}
		if softwareTitle.LastInstallInstalledAt != nil {
			hydrated.SoftwarePackage.LastInstall.InstalledAt = *softwareTitle.LastInstallInstalledAt
		}
	}

	// promote the last uninstall info to the proper destination fields
	if softwareTitle.LastUninstallScriptExecutionID != nil && *softwareTitle.LastUninstallScriptExecutionID != "" {
		hydrated.SoftwarePackage.LastUninstall = &fleet.HostSoftwareUninstall{
			ExecutionID: *softwareTitle.LastUninstallScriptExecutionID,
		}
		if softwareTitle.LastUninstallUninstalledAt != nil {
			hydrated.SoftwarePackage.LastUninstall.UninstalledAt = *softwareTitle.LastUninstallUninstalledAt
		}
	}
}

// softwareTitleRecord is the base record, we will be modifying it
func promoteSoftwareTitleVPPApp(softwareTitleRecord *hostSoftware) {
	var version,
		platform string
	if softwareTitleRecord.VPPAppVersion != nil {
		version = *softwareTitleRecord.VPPAppVersion
	}
	if softwareTitleRecord.VPPAppPlatform != nil {
		platform = *softwareTitleRecord.VPPAppPlatform
	}
	softwareTitleRecord.AppStoreApp = &fleet.SoftwarePackageOrApp{
		AppStoreID:  *softwareTitleRecord.VPPAppAdamID,
		Version:     version,
		Platform:    platform,
		SelfService: softwareTitleRecord.VPPAppSelfService,
		IconURL:     softwareTitleRecord.VPPAppIconURL,
	}
	if softwareTitleRecord.VPPAppPlatform != nil {
		softwareTitleRecord.AppStoreApp.Platform = *softwareTitleRecord.VPPAppPlatform
	}

	// promote the last install info to the proper destination fields
	if softwareTitleRecord.LastInstallInstallUUID != nil && *softwareTitleRecord.LastInstallInstallUUID != "" {
		softwareTitleRecord.AppStoreApp.LastInstall = &fleet.HostSoftwareInstall{
			CommandUUID: *softwareTitleRecord.LastInstallInstallUUID,
		}
		if softwareTitleRecord.LastInstallInstalledAt != nil {
			softwareTitleRecord.AppStoreApp.LastInstall.InstalledAt = *softwareTitleRecord.LastInstallInstalledAt
		}
	}
}

func (ds *Datastore) ListHostSoftware(ctx context.Context, host *fleet.Host, opts fleet.HostSoftwareTitleListOptions) ([]*fleet.HostSoftwareWithInstaller, *fleet.PaginationMetadata, error) {
	if !opts.VulnerableOnly && (opts.MinimumCVSS > 0 || opts.MaximumCVSS > 0 || opts.KnownExploit) {
		return nil, nil, fleet.NewInvalidArgumentError(
			"query", "min_cvss_score, max_cvss_score, and exploit can only be provided with vulnerable=true",
		)
	}

	var globalOrTeamID uint
	if host.TeamID != nil {
		globalOrTeamID = *host.TeamID
	}
	namedArgs := map[string]any{
		"host_id":               host.ID,
		"host_platform":         host.FleetPlatform(),
		"global_or_team_id":     globalOrTeamID,
		"is_mdm_enrolled":       opts.IsMDMEnrolled,
		"host_label_updated_at": host.LabelUpdatedAt,
		"avail":                 opts.OnlyAvailableForInstall,
		"self_service":          opts.SelfServiceOnly,
		"min_cvss":              opts.MinimumCVSS,
		"max_cvss":              opts.MaximumCVSS,
		"vpp_apps_platforms":    fleet.VPPAppsPlatforms,
		"known_exploit":         1,
	}
	var hasCVEMetaFilters bool
	if opts.KnownExploit || opts.MinimumCVSS > 0 || opts.MaximumCVSS > 0 {
		hasCVEMetaFilters = true
	}

	bySoftwareTitleID := make(map[uint]*hostSoftware)
	bySoftwareID := make(map[uint]*hostSoftware)

	hostSoftwareInstalls, err := hostSoftwareInstalls(ds, ctx, host.ID)
	if err != nil {
		return nil, nil, err
	}
	for _, s := range hostSoftwareInstalls {
		if _, ok := bySoftwareTitleID[s.ID]; !ok {
			bySoftwareTitleID[s.ID] = s
		} else {
			bySoftwareTitleID[s.ID].LastInstallInstalledAt = s.LastInstallInstalledAt
			bySoftwareTitleID[s.ID].LastInstallInstallUUID = s.LastInstallInstallUUID
		}
	}

	hostSoftwareUninstalls, err := hostSoftwareUninstalls(ds, ctx, host.ID)
	if err != nil {
		return nil, nil, err
	}
	for _, s := range hostSoftwareUninstalls {
		if _, ok := bySoftwareTitleID[s.ID]; !ok {
			bySoftwareTitleID[s.ID] = s
		} else if bySoftwareTitleID[s.ID].LastInstallInstalledAt == nil ||
			(s.LastUninstallUninstalledAt != nil && s.LastUninstallUninstalledAt.After(*bySoftwareTitleID[s.ID].LastInstallInstalledAt)) {
			// if the uninstall is more recent than the install, we should update the status
			bySoftwareTitleID[s.ID].Status = s.Status
			bySoftwareTitleID[s.ID].LastUninstallUninstalledAt = s.LastUninstallUninstalledAt
			bySoftwareTitleID[s.ID].LastUninstallScriptExecutionID = s.LastUninstallScriptExecutionID
			bySoftwareTitleID[s.ID].ExitCode = s.ExitCode
		}
	}

	hostInstalledSoftware, err := hostInstalledSoftware(ds, ctx, host.ID)
	hostInstalledSoftwareTitleSet := make(map[uint]struct{})
	hostInstalledSoftwareSet := make(map[uint]*hostSoftware)
	if err != nil {
		return nil, nil, err
	}
	for _, s := range hostInstalledSoftware {
		if _, ok := bySoftwareTitleID[s.ID]; !ok {
			bySoftwareTitleID[s.ID] = s
		} else {
			bySoftwareTitleID[s.ID].LastOpenedAt = s.LastOpenedAt
		}

		hostInstalledSoftwareTitleSet[s.ID] = struct{}{}
		if s.SoftwareID != nil {
			bySoftwareID[*s.SoftwareID] = s
			hostInstalledSoftwareSet[*s.SoftwareID] = s
		}
	}

	hostVPPInstalls, err := hostVPPInstalls(ds, ctx, host.ID, globalOrTeamID, opts.SelfServiceOnly, opts.IsMDMEnrolled)
	if err != nil {
		return nil, nil, err
	}
	byVPPAdamID := make(map[string]*hostSoftware)
	for _, s := range hostVPPInstalls {
		if s.VPPAppAdamID != nil {
			// If a VPP app is already installed on the host, we don't need to double count it
			// until we merge the two fetch queries later on in this method.
			// Until then if the host_software record is not a software installer, we delete it and keep the vpp app
			if _, exists := hostInstalledSoftwareTitleSet[s.ID]; exists {
				installedTitle := bySoftwareTitleID[s.ID]
				if installedTitle.InstallerID == nil {
					// not a software installer, so copy over
					// the installed title information
					s.LastOpenedAt = installedTitle.LastOpenedAt
					s.SoftwareID = installedTitle.SoftwareID
					s.SoftwareSource = installedTitle.SoftwareSource
					s.Version = installedTitle.Version
					s.BundleIdentifier = installedTitle.BundleIdentifier
					if !opts.VulnerableOnly && !hasCVEMetaFilters {
						// When we are filtering by vulnerable only
						// we want to treat the installed vpp app as a regular software title
						delete(bySoftwareTitleID, s.ID)
					}
				} else {
					continue
				}
			}
			byVPPAdamID[*s.VPPAppAdamID] = s
		}
	}

	hostInstalledVppsApps, err := hostInstalledVpps(ds, ctx, host.ID)
	if err != nil {
		return nil, nil, err
	}
	installedVppsByAdamID := make(map[string]*hostSoftware)
	for _, s := range hostInstalledVppsApps {
		if s.VPPAppAdamID != nil {
			installedVppsByAdamID[*s.VPPAppAdamID] = s
		}
	}

	hostVPPInstalledTitles := make(map[uint]*hostSoftware)
	for _, s := range installedVppsByAdamID {
		if _, ok := hostInstalledSoftwareTitleSet[s.ID]; ok {
			// we copied over all the installed title information
			// from bySoftwareTitleID, but deleted the record from the map
			// when going through hostVPPInstalls. Copy over the
			// data from the byVPPAdamID to hostVPPInstalledTitles
			// so we can later push to InstalledVersions
			installedTitle := byVPPAdamID[*s.VPPAppAdamID]
			if installedTitle == nil {
				// This can happen when mdm_enrolled is false
				// because in hostVPPInstalls we filter those out
				installedTitle = bySoftwareTitleID[s.ID]
			}
			if installedTitle == nil {
				// We somehow have a vpp app in host_vpp_software_installs,
				// however osquery didn't pick it up in inventory
				continue
			}
			s.SoftwareID = installedTitle.SoftwareID
			s.SoftwareSource = installedTitle.SoftwareSource
			s.Version = installedTitle.Version
			s.BundleIdentifier = installedTitle.BundleIdentifier
		}
		if s.VPPAppAdamID != nil {
			// Override the status; if there's a pending re-install, we should show that status.
			if hs, ok := byVPPAdamID[*s.VPPAppAdamID]; ok {
				s.Status = hs.Status
			}
		}
		hostVPPInstalledTitles[s.ID] = s
	}

	var stmtAvailable string

	if opts.OnlyAvailableForInstall || opts.IncludeAvailableForInstall {
		namedArgs["vpp_apps_platforms"] = fleet.VPPAppsPlatforms
		namedArgs["host_compatible_platforms"] = host.FleetPlatform()

		var availableSoftwareTitles []*hostSoftware

		if !opts.VulnerableOnly {
			stmtAvailable = `
				SELECT
				st.id,
				st.name,
				st.source,
				si.id as installer_id,
				si.self_service as package_self_service,
				si.filename as package_name,
				si.version as package_version,
				si.platform as package_platform,
				vat.self_service as vpp_app_self_service,
				vat.adam_id as vpp_app_adam_id,
				vap.latest_version as vpp_app_version,
				vap.platform as vpp_app_platform,
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
				software_installers si ON st.id = si.title_id AND si.platform = :host_compatible_platforms AND si.global_or_team_id = :global_or_team_id
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
						hsi.removed = 0 AND
						hsi.canceled = 0
				) AND
				-- sofware install/uninstall is not upcoming on host
				NOT EXISTS (
					SELECT 1
					FROM
						upcoming_activities ua
						INNER JOIN
							software_install_upcoming_activities siua ON siua.upcoming_activity_id = ua.id
					WHERE
						ua.host_id = :host_id AND
						ua.activity_type IN ('software_install', 'software_uninstall') AND
						siua.software_installer_id = si.id
				) AND
				-- VPP install has not been attempted on host
				NOT EXISTS (
					SELECT 1
					FROM
						host_vpp_software_installs hvsi
					WHERE
						hvsi.host_id = :host_id AND
						hvsi.adam_id = vat.adam_id AND
						hvsi.removed = 0 AND
						hvsi.canceled = 0
				) AND
				-- VPP install is not upcoming on host
				NOT EXISTS (
					SELECT 1
					FROM
						upcoming_activities ua
						INNER JOIN
							vpp_app_upcoming_activities vaua ON vaua.upcoming_activity_id = ua.id
					WHERE
						ua.host_id = :host_id AND
						ua.activity_type = 'vpp_app_install' AND
						vaua.adam_id = vat.adam_id
				) AND
				-- either the software installer or the vpp app exists for the host's team
				( si.id IS NOT NULL OR vat.platform = :host_platform ) AND
				-- label membership check
				(
					-- do the label membership check for software installers and VPP apps
						EXISTS (

						SELECT 1 FROM (

							-- no labels
							SELECT 0 AS count_installer_labels, 0 AS count_host_labels, 0 as count_host_updated_after_labels
							WHERE NOT EXISTS (
								SELECT 1 FROM software_installer_labels sil WHERE sil.software_installer_id = si.id
							) AND NOT EXISTS (SELECT 1 FROM vpp_app_team_labels vatl WHERE vatl.vpp_app_team_id = vat.id)

							UNION

							-- include any
							SELECT
								COUNT(*) AS count_installer_labels,
								COUNT(lm.label_id) AS count_host_labels,
								0 as count_host_updated_after_labels
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

							-- exclude any, ignore software that depends on labels created
							-- _after_ the label_updated_at timestamp of the host (because
							-- we don't have results for that label yet, the host may or may
							-- not be a member).
							SELECT
								COUNT(*) AS count_installer_labels,
								COUNT(lm.label_id) AS count_host_labels,
								SUM(CASE WHEN lbl.created_at IS NOT NULL AND :host_label_updated_at >= lbl.created_at THEN 1 ELSE 0 END) as count_host_updated_after_labels
							FROM
								software_installer_labels sil
								LEFT OUTER JOIN labels lbl
									ON lbl.id = sil.label_id
								LEFT OUTER JOIN label_membership lm
									ON lm.label_id = sil.label_id AND lm.host_id = :host_id
							WHERE
								sil.software_installer_id = si.id
								AND sil.exclude = 1
							HAVING
								count_installer_labels > 0 AND count_installer_labels = count_host_updated_after_labels AND count_host_labels = 0

							UNION

							-- vpp include any
							SELECT
								COUNT(*) AS count_installer_labels,
								COUNT(lm.label_id) AS count_host_labels,
								0 as count_host_updated_after_labels
							FROM
								vpp_app_team_labels vatl
								LEFT OUTER JOIN label_membership lm ON lm.label_id = vatl.label_id
								AND lm.host_id = :host_id
							WHERE
								vatl.vpp_app_team_id = vat.id
								AND vatl.exclude = 0
							HAVING
								count_installer_labels > 0 AND count_host_labels > 0

							UNION

							-- vpp exclude any
							SELECT
								COUNT(*) AS count_installer_labels,
								COUNT(lm.label_id) AS count_host_labels,
								SUM(CASE
								WHEN lbl.created_at IS NOT NULL AND lbl.label_membership_type = 0 AND :host_label_updated_at >= lbl.created_at THEN 1
								WHEN lbl.created_at IS NOT NULL AND lbl.label_membership_type = 1 THEN 1
								ELSE 0 END) as count_host_updated_after_labels
							FROM
								vpp_app_team_labels vatl
								LEFT OUTER JOIN labels lbl
									ON lbl.id = vatl.label_id
								LEFT OUTER JOIN label_membership lm
									ON lm.label_id = vatl.label_id AND lm.host_id = :host_id
							WHERE
								vatl.vpp_app_team_id = vat.id
								AND vatl.exclude = 1
							HAVING
								count_installer_labels > 0 AND count_installer_labels = count_host_updated_after_labels AND count_host_labels = 0
							) t
						)
				)
			`
			if opts.SelfServiceOnly {
				stmtAvailable += "\nAND ( si.self_service = 1 OR ( vat.self_service = 1 AND :is_mdm_enrolled ) )"
			}

			if !opts.IsMDMEnrolled {
				stmtAvailable += "\nAND vat.id IS NULL"
			}

			stmtAvailable, args, err := sqlx.Named(stmtAvailable, namedArgs)
			if err != nil {
				return nil, nil, err
			}
			stmtAvailable, args, err = sqlx.In(stmtAvailable, args...)
			if err != nil {
				return nil, nil, err
			}

			err = sqlx.SelectContext(ctx, ds.reader(ctx), &availableSoftwareTitles, stmtAvailable, args...)
			if err != nil {
				return nil, nil, err
			}
		}

		// These slices are meant to keep track of software that is available for install.
		// When we are filtering by `OnlyAvailableForInstall`, we will replace the existing
		// software title records held in bySoftwareTitleID and byVPPAdamID.
		// If we are just using the `IncludeAvailableForInstall` options, we will simply
		// add these addtional software titles to bySoftwareTitleID and byVPPAdamID.
		tempBySoftwareTitleID := make(map[uint]*hostSoftware, len(availableSoftwareTitles))
		tmpByVPPAdamID := make(map[string]*hostSoftware, len(byVPPAdamID))
		if opts.OnlyAvailableForInstall {
			// drop in anything that has been installed or uninstalled as it can be installed again regardless of status
			for _, s := range hostSoftwareUninstalls {
				tempBySoftwareTitleID[s.ID] = s
			}
			if !opts.VulnerableOnly {
				for _, s := range hostSoftwareInstalls {
					tempBySoftwareTitleID[s.ID] = s
				}
				for _, s := range hostVPPInstalls {
					tmpByVPPAdamID[*s.VPPAppAdamID] = s
				}
			}
		}
		// software installed on the host not by fleet and there exists a software installer that matches this software
		// so that makes it available for install
		installedInstallersSql := `
			SELECT
				software.title_id,
				software_installers.id AS installer_id,
				software_installers.self_service AS package_self_service
			FROM
				host_software
			INNER JOIN
				software ON host_software.software_id = software.id
			INNER JOIN
				software_installers ON software.title_id = software_installers.title_id
				  AND software_installers.platform = ?
				  AND software_installers.global_or_team_id = ?
			WHERE host_software.host_id = ?
			`
		type InstalledSoftwareTitle struct {
			TitleID     uint `db:"title_id"`
			InstallerID uint `db:"installer_id"`
			SelfService bool `db:"package_self_service"`
		}
		var installedSoftwareTitleIDs []InstalledSoftwareTitle
		err = sqlx.SelectContext(ctx, ds.reader(ctx), &installedSoftwareTitleIDs, installedInstallersSql, namedArgs["host_compatible_platforms"], globalOrTeamID, host.ID)
		if err != nil {
			return nil, nil, err
		}
		for _, s := range installedSoftwareTitleIDs {
			if software := bySoftwareTitleID[s.TitleID]; software != nil {
				software.InstallerID = &s.InstallerID
				software.PackageSelfService = &s.SelfService
				tempBySoftwareTitleID[s.TitleID] = software
			}
		}
		if !opts.SelfServiceOnly || (opts.SelfServiceOnly && opts.IsMDMEnrolled) {
			// software installed on the host not by fleet and there exists a vpp app that matches this software
			// so that makes it available for install
			installedVPPAppsSql := `
				SELECT
					vpp_apps.title_id AS id,
					vpp_apps.adam_id AS vpp_app_adam_id,
					vpp_apps.latest_version AS vpp_app_version,
					vpp_apps.platform as vpp_app_platform,
					NULLIF(vpp_apps.icon_url, '') as vpp_app_icon_url,
					vpp_apps_teams.self_service AS vpp_app_self_service
				FROM
					host_software
				INNER JOIN
					software ON host_software.software_id = software.id
				INNER JOIN
					vpp_apps ON software.title_id = vpp_apps.title_id AND :host_platform IN (:vpp_apps_platforms)
				INNER JOIN
					vpp_apps_teams ON vpp_apps.adam_id = vpp_apps_teams.adam_id AND vpp_apps.platform = vpp_apps_teams.platform AND vpp_apps_teams.global_or_team_id = :global_or_team_id
				WHERE
					host_software.host_id = :host_id
				`
			installedVPPAppsSql, args, err := sqlx.Named(installedVPPAppsSql, namedArgs)
			if err != nil {
				return nil, nil, err
			}
			installedVPPAppsSql, args, err = sqlx.In(installedVPPAppsSql, args...)
			if err != nil {
				return nil, nil, err
			}
			var installedVPPAppIDs []*hostSoftware
			err = sqlx.SelectContext(ctx, ds.reader(ctx), &installedVPPAppIDs, installedVPPAppsSql, args...)
			if err != nil {
				return nil, nil, err
			}
			for _, s := range installedVPPAppIDs {
				if s.VPPAppAdamID != nil {
					tmpByVPPAdamID[*s.VPPAppAdamID] = s
				}
				if VPPAppByFleet, ok := hostVPPInstalledTitles[s.ID]; ok {
					// Vpp app installed by fleet, so we need to copy over the status,
					// because all fleet installed apps show an installed status if available
					tmpByVPPAdamID[*s.VPPAppAdamID].Status = VPPAppByFleet.Status
				}
				// If a VPP app is installed on the host, but not by fleet
				// it will be present in bySoftwareTitleID, because osquery returned it as inventory.
				// We need to remove it from bySoftwareTitleID and add it to byVPPAdamID
				if invetoriedSoftware, ok := bySoftwareTitleID[s.ID]; ok {
					invetoriedSoftware.VPPAppAdamID = s.VPPAppAdamID
					invetoriedSoftware.VPPAppVersion = s.VPPAppVersion
					invetoriedSoftware.VPPAppPlatform = s.VPPAppPlatform
					invetoriedSoftware.VPPAppIconURL = s.VPPAppIconURL
					invetoriedSoftware.VPPAppSelfService = s.VPPAppSelfService
					if !opts.VulnerableOnly && !hasCVEMetaFilters {
						// When we are filtering by vulnerable only
						// we want to treat the installed vpp app as a regular software title
						delete(bySoftwareTitleID, s.ID)
						byVPPAdamID[*s.VPPAppAdamID] = invetoriedSoftware
					}
					hostVPPInstalledTitles[s.ID] = invetoriedSoftware
				}
			}
		}

		for _, s := range availableSoftwareTitles {
			// If it's a VPP app
			if s.VPPAppAdamID != nil {
				existingVPP, found := byVPPAdamID[*s.VPPAppAdamID]

				if opts.OnlyAvailableForInstall {
					if !found {
						tmpByVPPAdamID[*s.VPPAppAdamID] = s
					} else {
						tmpByVPPAdamID[*s.VPPAppAdamID] = existingVPP
					}
				} else {
					// We have an existing vpp record in an installed or pending state, do not overwrite with the
					// one that's available for install. We would lose specifics about the installed version
					if !found {
						byVPPAdamID[*s.VPPAppAdamID] = s
					}
				}

			} else {
				existingSoftware, found := bySoftwareTitleID[s.ID]

				if opts.OnlyAvailableForInstall {
					if !found {
						tempBySoftwareTitleID[s.ID] = s
					} else {
						tempBySoftwareTitleID[s.ID] = existingSoftware
					}
				} else {
					// We have an existing software record in an installed or pending state, do not overwrite with the
					// one that's available for install. We would lose specifics about the previous record
					if !found {
						bySoftwareTitleID[s.ID] = s
					}
				}
			}
		}
		// Clear out all the previous software titles as we are only filtering for available software
		if opts.OnlyAvailableForInstall {
			bySoftwareTitleID = tempBySoftwareTitleID
			byVPPAdamID = tmpByVPPAdamID
		}
	}

	// filter out software installers due to label scoping
	filteredBySoftwareTitleID, err := filterSoftwareInstallersByLabel(
		ds,
		ctx,
		host,
		bySoftwareTitleID,
	)
	if err != nil {
		return nil, nil, err
	}

	filteredByVPPAdamID, otherVppAppsInInventory, err := filterVppAppsByLabel(
		ds,
		ctx,
		host,
		byVPPAdamID,
		hostVPPInstalledTitles,
	)
	if err != nil {
		return nil, nil, err
	}

	// We ignored the VPP apps that were installed on the host while filtering in filterSoftwareInstallersByLabel
	// so we need to add them back in if they are allowed by filterVppAppsByLabel
	for _, value := range otherVppAppsInInventory {
		if st, ok := bySoftwareTitleID[value.ID]; ok {
			filteredBySoftwareTitleID[value.ID] = st
		}
	}

	if opts.OnlyAvailableForInstall {
		bySoftwareTitleID = filteredBySoftwareTitleID
		byVPPAdamID = filteredByVPPAdamID
	}
	// self service impacts inventory, when a software title is excluded because of a filter,
	// it should be excluded from the inventory as well, because we cannot "reinstall" it on the self service page
	if opts.SelfServiceOnly {
		for _, software := range bySoftwareTitleID {
			if software.PackageSelfService != nil && *software.PackageSelfService {
				if filteredBySoftwareTitleID[software.ID] == nil {
					// remove the software title from bySoftwareTitleID
					delete(bySoftwareTitleID, software.ID)
				}
			}
		}
		for vppAppAdamID, software := range byVPPAdamID {
			if software.VPPAppSelfService != nil && *software.VPPAppSelfService {
				if filteredByVPPAdamID[vppAppAdamID] == nil {
					// remove the software title from byVPPAdamID
					delete(byVPPAdamID, vppAppAdamID)
				}
			}
		}
	}

	// since these host installed vpp apps are already added in bySoftwareTitleID,
	// we need to avoid adding them to byVPPAdamID
	// but we need to store them in filteredByVPPAdamID so they are able to be
	// promoted when returning the software title
	for key, value := range otherVppAppsInInventory {
		if _, ok := filteredByVPPAdamID[key]; !ok {
			filteredByVPPAdamID[key] = value
		}
	}

	var softwareTitleIds []uint
	for softwareTitleID := range bySoftwareTitleID {
		softwareTitleIds = append(softwareTitleIds, softwareTitleID)
	}

	var softwareIDs []uint
	for softwareID := range bySoftwareID {
		softwareIDs = append(softwareIDs, softwareID)
	}

	var vppAdamIDs []string
	for key := range byVPPAdamID {
		vppAdamIDs = append(vppAdamIDs, key)
	}

	var titleCount uint
	var hostSoftwareList []*hostSoftware
	if len(softwareTitleIds) > 0 || len(vppAdamIDs) > 0 {
		var args []interface{}
		var stmt string
		var softwareTitleStatement string
		var vppAdamStatment string

		matchClause := ""
		matchArgs := []interface{}{}
		if opts.ListOptions.MatchQuery != "" {
			matchClause, matchArgs = searchLike(matchClause, matchArgs, opts.ListOptions.MatchQuery, "software_titles.name")
		}

		var softwareOnlySelfServiceClause string
		var vppOnlySelfServiceClause string
		if opts.SelfServiceOnly {
			softwareOnlySelfServiceClause = ` AND software_installers.self_service = 1 `
			if opts.IsMDMEnrolled {
				vppOnlySelfServiceClause = ` AND vpp_apps_teams.self_service = 1 `
			}
		}

		var cveMetaFilter string
		var cveMatchClause string
		var cveNamedArgs []interface{}
		var cveMatchArgs []interface{}
		if opts.KnownExploit {
			cveMetaFilter += "\nAND cve_meta.cisa_known_exploit = :known_exploit"
		}
		if opts.MinimumCVSS > 0 {
			cveMetaFilter += "\nAND cve_meta.cvss_score >= :min_cvss"
		}
		if opts.MaximumCVSS > 0 {
			cveMetaFilter += "\nAND cve_meta.cvss_score <= :max_cvss"
		}
		if hasCVEMetaFilters {
			cveMetaFilter, cveNamedArgs, err = sqlx.Named(cveMetaFilter, namedArgs)
			if err != nil {
				return nil, nil, ctxerr.Wrap(ctx, err, "build named query for cve meta filters")
			}
		}
		if opts.ListOptions.MatchQuery != "" {
			cveMatchClause, cveMatchArgs = searchLike(cveMatchClause, cveMatchArgs, opts.ListOptions.MatchQuery, "software_cve.cve")
		}

		var softwareVulnerableJoin string
		if len(softwareTitleIds) > 0 {
			if opts.VulnerableOnly || opts.ListOptions.MatchQuery != "" {
				softwareVulnerableJoin += " AND ( "
				if !opts.VulnerableOnly && opts.ListOptions.MatchQuery != "" {
					softwareVulnerableJoin += `
					    -- Software without vulnerabilities
						(
							NOT EXISTS (
								SELECT 1
								FROM
									software_cve
								WHERE
									software_cve.software_id = software.id
							) ` + matchClause + `
					    ) OR
					`
				}

				softwareVulnerableJoin += `
				-- Software with vulnerabilities
				EXISTS (
					SELECT 1
					FROM
						software_cve
				`
				cveMetaJoin := "\n INNER JOIN cve_meta ON software_cve.cve = cve_meta.cve"

				// Only join CVE table if there are filters
				if hasCVEMetaFilters {
					softwareVulnerableJoin += cveMetaJoin
				}
				softwareVulnerableJoin += `
					WHERE
						software_cve.software_id = software.id
				`
				softwareVulnerableJoin += cveMetaFilter
				softwareVulnerableJoin += "\n" + strings.ReplaceAll(cveMatchClause, "AND", "AND (")
				softwareVulnerableJoin += strings.ReplaceAll(matchClause, "AND", "OR") + ")"
				softwareVulnerableJoin += "\n)"
				if !opts.VulnerableOnly || opts.ListOptions.MatchQuery != "" {
					softwareVulnerableJoin += ")"
				}
			}

			installedSoftwareJoinsCondition := ""
			if len(softwareIDs) > 0 {
				installedSoftwareJoinsCondition = `AND software.id IN (?)`
			}

			softwareTitleStatement = `
			-- SELECT for software
			%s
			FROM
				software_titles
			LEFT JOIN
				software_installers ON software_titles.id = software_installers.title_id 
				AND software_installers.global_or_team_id = :global_or_team_id
			LEFT JOIN
				software ON software_titles.id = software.title_id ` + installedSoftwareJoinsCondition + `
			WHERE
				software_titles.id IN (?)
			%s
			` + softwareOnlySelfServiceClause + `
			-- GROUP by for software
			%s
			`

			var softwareTitleArgs []interface{}
			if len(softwareIDs) > 0 {
				softwareTitleStatement, softwareTitleArgs, err = sqlx.In(softwareTitleStatement, softwareIDs, softwareTitleIds)
			} else {
				softwareTitleStatement, softwareTitleArgs, err = sqlx.In(softwareTitleStatement, softwareTitleIds)
			}
			if err != nil {
				return nil, nil, ctxerr.Wrap(ctx, err, "expand IN query for software titles")
			}
			softwareTitleStatement, softwareTitleArgsNamedArgs, err := sqlx.Named(softwareTitleStatement, namedArgs)
			if err != nil {
				return nil, nil, ctxerr.Wrap(ctx, err, "build named query for software titles")
			}
			args = append(args, softwareTitleArgsNamedArgs...)
			args = append(args, softwareTitleArgs...)
			if len(cveNamedArgs) > 0 {
				args = append(args, cveNamedArgs...)
			}
			if len(cveMatchArgs) > 0 {
				args = append(args, cveMatchArgs...)
			}
			if len(matchArgs) > 0 {
				args = append(args, matchArgs...)
				// Have to conditionally add the additional match for software without vulnerabilities
				if !opts.VulnerableOnly && opts.ListOptions.MatchQuery != "" {
					args = append(args, matchArgs...)
				}
			}
			stmt += softwareTitleStatement
		}

		if !opts.VulnerableOnly && len(vppAdamIDs) > 0 {
			if len(softwareTitleIds) > 0 {
				vppAdamStatment = ` UNION `
			}

			vppAdamStatment += `
			-- SELECT for vpp apps
			%s
			FROM
				software_titles
			INNER JOIN
				vpp_apps ON software_titles.id = vpp_apps.title_id AND vpp_apps.platform = :host_platform
			INNER JOIN
				vpp_apps_teams ON vpp_apps.adam_id = vpp_apps_teams.adam_id AND vpp_apps.platform = vpp_apps_teams.platform AND vpp_apps_teams.global_or_team_id = :global_or_team_id
			WHERE
				vpp_apps.adam_id IN (?)
				AND true
			` + vppOnlySelfServiceClause + `
			-- GROUP BY for vpp apps
			%s
			`

			vppAdamStatement, vppAdamArgs, err := sqlx.In(vppAdamStatment, vppAdamIDs)
			if err != nil {
				return nil, nil, ctxerr.Wrap(ctx, err, "expand IN query for vpp titles")
			}
			vppAdamStatement, vppAdamArgsNamedArgs, err := sqlx.Named(vppAdamStatement, namedArgs)
			if err != nil {
				return nil, nil, ctxerr.Wrap(ctx, err, "build named query for vpp titles")
			}
			vppAdamStatement = strings.ReplaceAll(vppAdamStatement, "AND true", matchClause)
			args = append(args, vppAdamArgsNamedArgs...)
			args = append(args, vppAdamArgs...)
			if len(matchArgs) > 0 {
				args = append(args, matchArgs...)
			}
			stmt += vppAdamStatement
		}

		var countStmt string
		// we do not scan vulnerabilities on vpp software available for install
		includeVPP := !opts.VulnerableOnly && len(vppAdamIDs) > 0
		switch {
		case len(softwareTitleIds) > 0 && includeVPP:
			countStmt = fmt.Sprintf(stmt, `SELECT software_titles.id`, softwareVulnerableJoin, `GROUP BY software_titles.id`, `SELECT software_titles.id`, `GROUP BY software_titles.id`)
		case len(softwareTitleIds) > 0:
			countStmt = fmt.Sprintf(stmt, `SELECT software_titles.id`, softwareVulnerableJoin, `GROUP BY software_titles.id`)
		case includeVPP:
			countStmt = fmt.Sprintf(stmt, `SELECT software_titles.id`, `GROUP BY software_titles.id`)
		default:
			return []*fleet.HostSoftwareWithInstaller{}, &fleet.PaginationMetadata{}, nil
		}

		if err := sqlx.GetContext(
			ctx,
			ds.reader(ctx),
			&titleCount,
			fmt.Sprintf("SELECT COUNT(id) FROM (%s) AS combined_results", countStmt),
			args...,
		); err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "get host software count")
		}

		var replacements []any
		if len(softwareTitleIds) > 0 {
			replacements = append(replacements,
				// For software installers
				`
				SELECT
					software_titles.id,
					software_titles.name,
					software_titles.source AS source,
					software_installers.id AS installer_id,
					software_installers.self_service AS package_self_service,
					software_installers.filename AS package_name,
					software_installers.version AS package_version,
					software_installers.platform as package_platform,
					GROUP_CONCAT(software.id) AS software_id_list,
					GROUP_CONCAT(software.source) AS software_source_list,
					GROUP_CONCAT(software.version) AS version_list,
					GROUP_CONCAT(software.bundle_identifier) AS bundle_identifier_list,
					NULL AS vpp_app_adam_id_list,
					NULL AS vpp_app_version_list,
					NULL AS vpp_app_platform_list,
					NULL AS vpp_app_icon_url_list,
					NULL AS vpp_app_self_service_list
			`, softwareVulnerableJoin, `
				GROUP BY
					software_titles.id,
					software_titles.name,
					software_titles.source,
					software_installers.id,
					software_installers.self_service,
					software_installers.filename,
					software_installers.version,
					software_installers.platform
			`)
		}
		if includeVPP {
			replacements = append(replacements,
				// For vpp apps
				`
				SELECT
					software_titles.id,
					software_titles.name,
					software_titles.source AS source,
					NULL AS installer_id,
					NULL AS package_self_service,
					NULL AS package_name,
					NULL AS package_version,
					NULL as package_platform,
					NULL AS software_id_list,
					NULL AS software_source_list,
					NULL AS version_list,
					NULL AS bundle_identifier_list,
					GROUP_CONCAT(vpp_apps.adam_id) AS vpp_app_adam_id_list,
					GROUP_CONCAT(vpp_apps.latest_version) AS vpp_app_version_list,
					GROUP_CONCAT(vpp_apps.platform) as vpp_app_platform_list,
					GROUP_CONCAT(vpp_apps.icon_url) AS vpp_app_icon_url_list,
					GROUP_CONCAT(vpp_apps_teams.self_service) AS vpp_app_self_service_list
			`, `
				GROUP BY
					software_titles.id,
					software_titles.name,
					software_titles.source
			`)
		}
		stmt = fmt.Sprintf(stmt, replacements...)
		stmt = fmt.Sprintf("SELECT * FROM (%s) AS combined_results", stmt)
		stmt, _ = appendListOptionsToSQL(stmt, &opts.ListOptions)

		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hostSoftwareList, stmt, args...); err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "list host software")
		}

		// collect install paths by software.id
		installedPaths, err := ds.getHostSoftwareInstalledPaths(ctx, host.ID)
		if err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "Could not get software installed paths")
		}
		installedPathBySoftwareId := make(map[uint][]string)
		pathSignatureInformation := make(map[uint][]fleet.PathSignatureInformation)
		for _, ip := range installedPaths {
			installedPathBySoftwareId[ip.SoftwareID] = append(installedPathBySoftwareId[ip.SoftwareID], ip.InstalledPath)
			pathSignatureInformation[ip.SoftwareID] = append(pathSignatureInformation[ip.SoftwareID], fleet.PathSignatureInformation{
				InstalledPath:  ip.InstalledPath,
				TeamIdentifier: ip.TeamIdentifier,
				HashSha256:     ip.ExecutableSHA256,
			})
		}

		// extract into vulnerabilitiesBySoftwareID
		type softwareCVE struct {
			SoftwareID uint   `db:"software_id"`
			CVE        string `db:"cve"`
		}
		var softwareCVEs []softwareCVE

		if len(softwareIDs) > 0 {
			cveStmt := `
				SELECT
					software_id,
					cve
				FROM
					software_cve
				WHERE
					software_id IN (?)
				ORDER BY
					software_id, cve
			`
			cveStmt, args, err = sqlx.In(cveStmt, softwareIDs)
			if err != nil {
				return nil, nil, ctxerr.Wrap(ctx, err, "building query args to list cves")
			}
			if err := sqlx.SelectContext(ctx, ds.reader(ctx), &softwareCVEs, cveStmt, args...); err != nil {
				return nil, nil, ctxerr.Wrap(ctx, err, "list software cves")
			}
		}

		// group by softwareID
		vulnerabilitiesBySoftwareID := make(map[uint][]string)
		for _, cve := range softwareCVEs {
			vulnerabilitiesBySoftwareID[cve.SoftwareID] = append(vulnerabilitiesBySoftwareID[cve.SoftwareID], cve.CVE)
		}

		indexOfSoftwareTitle := make(map[uint]uint)
		deduplicatedList := make([]*hostSoftware, 0, len(hostSoftwareList))
		for _, softwareTitleRecord := range hostSoftwareList {
			softwareTitle := bySoftwareTitleID[softwareTitleRecord.ID]
			inventoriedVPPApp := hostVPPInstalledTitles[softwareTitleRecord.ID]

			if softwareTitle != nil && softwareTitle.SoftwareID != nil {
				// if we have a software id, that means that this record has been installed on the host,
				// we should double check the hostInstalledSoftwareSet,
				// but we want to make sure that software id is present on the InstalledVersions list to be processed
				if s, ok := hostInstalledSoftwareSet[*softwareTitle.SoftwareID]; ok {
					softwareIDStr := strconv.FormatUint(uint64(*softwareTitle.SoftwareID), 10)
					pushVersion(softwareIDStr, softwareTitleRecord, *s)
				}
			}
			if inventoriedVPPApp != nil && inventoriedVPPApp.SoftwareID != nil {
				// Vpp app installed on the host, we need to push this into the installed versions list as well
				if s, ok := hostInstalledSoftwareSet[*inventoriedVPPApp.SoftwareID]; ok {
					softwareIDStr := strconv.FormatUint(uint64(*inventoriedVPPApp.SoftwareID), 10)
					pushVersion(softwareIDStr, softwareTitleRecord, *s)
				}
			}

			if softwareTitleRecord.SoftwareIDList != nil {
				softwareIDList := strings.Split(*softwareTitleRecord.SoftwareIDList, ",")
				softwareSourceList := strings.Split(*softwareTitleRecord.SoftwareSourceList, ",")
				softwareVersionList := strings.Split(*softwareTitleRecord.VersionList, ",")
				softwareBundleIdentifierList := strings.Split(*softwareTitleRecord.BundleIdentifierList, ",")

				for index, softwareIdStr := range softwareIDList {
					version := &fleet.HostSoftwareInstalledVersion{}

					if softwareId, err := strconv.ParseUint(softwareIdStr, 10, 32); err == nil {

						softwareId := uint(softwareId)
						if software, ok := bySoftwareID[softwareId]; ok {
							version.Version = softwareVersionList[index]
							version.BundleIdentifier = softwareBundleIdentifierList[index]
							version.Source = softwareSourceList[index]
							version.LastOpenedAt = software.LastOpenedAt
							version.SoftwareID = softwareId
							version.SoftwareTitleID = softwareTitleRecord.ID

							version.InstalledPaths = installedPathBySoftwareId[softwareId]
							version.Vulnerabilities = vulnerabilitiesBySoftwareID[softwareId]

							if version.Source == "apps" {
								version.SignatureInformation = pathSignatureInformation[softwareId]
							}

							if storedIndex, ok := indexOfSoftwareTitle[softwareTitleRecord.ID]; ok {
								deduplicatedList[storedIndex].InstalledVersions = append(deduplicatedList[storedIndex].InstalledVersions, version)
							} else {
								softwareTitleRecord.InstalledVersions = append(softwareTitleRecord.InstalledVersions, version)
							}
						}
					}
				}
			}

			if softwareTitleRecord.VPPAppAdamIDList != nil {
				vppAppAdamIDList := strings.Split(*softwareTitleRecord.VPPAppAdamIDList, ",")
				vppAppSelfServiceList := strings.Split(*softwareTitleRecord.VPPAppSelfServiceList, ",")
				vppAppVersionList := strings.Split(*softwareTitleRecord.VPPAppVersionList, ",")
				vppAppPlatformList := strings.Split(*softwareTitleRecord.VPPAppPlatformList, ",")
				vppAppIconURLList := strings.Split(*softwareTitleRecord.VPPAppIconUrlList, ",")

				if storedIndex, ok := indexOfSoftwareTitle[softwareTitleRecord.ID]; ok {
					softwareTitleRecord = deduplicatedList[storedIndex]
				}

				for index, vppAppAdamIdStr := range vppAppAdamIDList {
					if vppAppAdamIdStr != "" {
						softwareTitle = byVPPAdamID[vppAppAdamIdStr]
						softwareTitleRecord.VPPAppAdamID = &vppAppAdamIdStr
					}

					vppAppSelfService := vppAppSelfServiceList[index]
					if vppAppSelfService != "" {
						if vppAppSelfService == "1" {
							softwareTitleRecord.VPPAppSelfService = ptr.Bool(true)
						} else {
							softwareTitleRecord.VPPAppSelfService = ptr.Bool(false)
						}
					}

					vppAppVersion := vppAppVersionList[index]
					if vppAppVersion != "" {
						softwareTitleRecord.VPPAppVersion = &vppAppVersion
					}

					vppAppPlatform := vppAppPlatformList[index]
					if vppAppPlatform != "" {
						softwareTitleRecord.VPPAppPlatform = &vppAppPlatform
					}
					VPPAppIconURL := vppAppIconURLList[index]
					if VPPAppIconURL != "" {
						softwareTitleRecord.VPPAppIconURL = &VPPAppIconURL
					}
				}
			}

			if storedIndex, ok := indexOfSoftwareTitle[softwareTitleRecord.ID]; ok {
				softwareTitleRecord = deduplicatedList[storedIndex]
			}

			// Merge the data of `software title` into `softwareTitleRecord`
			// We should try to move as much of these attributes into the `stmt` query
			if softwareTitle != nil {
				softwareTitleRecord.Status = softwareTitle.Status
				softwareTitleRecord.LastInstallInstallUUID = softwareTitle.LastInstallInstallUUID
				softwareTitleRecord.LastInstallInstalledAt = softwareTitle.LastInstallInstalledAt
				softwareTitleRecord.LastUninstallScriptExecutionID = softwareTitle.LastUninstallScriptExecutionID
				softwareTitleRecord.LastUninstallUninstalledAt = softwareTitle.LastUninstallUninstalledAt
				if softwareTitle.PackageSelfService != nil {
					softwareTitleRecord.PackageSelfService = softwareTitle.PackageSelfService
				}
			}

			// promote the package name and version to the proper destination fields
			if softwareTitleRecord.PackageName != nil {
				if _, ok := filteredBySoftwareTitleID[softwareTitleRecord.ID]; ok {
					hydrateHostSoftwareRecordFromDb(softwareTitleRecord, softwareTitle)
				}
			}

			// This happens when there is a software installed on the host but it is also a vpp record, so we want
			// to grab the vpp data from the installed vpp record and merge it onto the software record
			if installedVppRecord, ok := hostVPPInstalledTitles[softwareTitleRecord.ID]; ok {
				softwareTitleRecord.VPPAppAdamID = installedVppRecord.VPPAppAdamID
				softwareTitleRecord.VPPAppVersion = installedVppRecord.VPPAppVersion
				softwareTitleRecord.VPPAppPlatform = installedVppRecord.VPPAppPlatform
				softwareTitleRecord.VPPAppIconURL = installedVppRecord.VPPAppIconURL
				softwareTitleRecord.VPPAppSelfService = installedVppRecord.VPPAppSelfService
			}
			// promote the VPP app id and version to the proper destination fields
			if softwareTitleRecord.VPPAppAdamID != nil {
				if _, ok := filteredByVPPAdamID[*softwareTitleRecord.VPPAppAdamID]; ok {
					promoteSoftwareTitleVPPApp(softwareTitleRecord)
				}
			}

			if _, ok := indexOfSoftwareTitle[softwareTitleRecord.ID]; !ok {
				indexOfSoftwareTitle[softwareTitleRecord.ID] = uint(len(deduplicatedList))
				deduplicatedList = append(deduplicatedList, softwareTitleRecord)
			}
		}

		hostSoftwareList = deduplicatedList
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

func (ds *Datastore) SetHostSoftwareInstallResult(ctx context.Context, result *fleet.HostSoftwareInstallResultPayload) (wasCanceled bool, err error) {
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

	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx, stmt,
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

		if result.Status() != fleet.SoftwareInstallPending {
			if _, err := ds.activateNextUpcomingActivity(ctx, tx, result.HostID, result.InstallUUID); err != nil {
				return ctxerr.Wrap(ctx, err, "activate next activity")
			}
		}

		// load whether or not the result was for a canceled activity
		err = sqlx.GetContext(ctx, tx, &wasCanceled, `SELECT canceled FROM host_software_installs WHERE execution_id = ?`, result.InstallUUID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		return nil
	})
	return wasCanceled, err
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
WHERE hsi.removed = 0 AND hsi.canceled = 0 AND hsi.status = :software_status_installed

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
WHERE hvsi.removed = 0 AND hvsi.canceled = 0 AND ncr.status = :mdm_status_acknowledged
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

func (ds *Datastore) NewSoftwareCategory(ctx context.Context, name string) (*fleet.SoftwareCategory, error) {
	stmt := `INSERT INTO software_categories (name) VALUES (?)`
	res, err := ds.writer(ctx).ExecContext(ctx, stmt, name)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "new software category")
	}

	r, _ := res.LastInsertId()
	id := uint(r) //nolint:gosec // dismiss G115
	return &fleet.SoftwareCategory{Name: name, ID: id}, nil
}

func (ds *Datastore) GetSoftwareCategoryIDs(ctx context.Context, names []string) ([]uint, error) {
	if len(names) == 0 {
		return []uint{}, nil
	}

	stmt := `SELECT id FROM software_categories WHERE name IN (?)`
	stmt, args, err := sqlx.In(stmt, names)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "sqlx.In for get software category ids")
	}

	var ids []uint
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &ids, stmt, args...); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, err, "get software category ids")
		}
	}

	return ids, nil
}

func (ds *Datastore) GetCategoriesForSoftwareTitles(ctx context.Context, softwareTitleIDs []uint, teamID *uint) (map[uint][]string, error) {
	if len(softwareTitleIDs) == 0 {
		return map[uint][]string{}, nil
	}

	stmt := `
SELECT
	st.id AS title_id,
	sc.name AS software_category_name
FROM
	software_installers si
	JOIN software_titles st ON st.id = si.title_id
	JOIN software_installer_software_categories sisc ON sisc.software_installer_id = si.id
	JOIN software_categories sc ON sc.id = sisc.software_category_id
WHERE
	st.id IN (?) AND si.global_or_team_id = ?

UNION

SELECT
	st.id AS title_id,
	sc.name AS software_category_name
FROM
	vpp_apps va
	JOIN vpp_apps_teams vat ON va.adam_id = vat.adam_id AND va.platform = vat.platform
	JOIN software_titles st ON st.id = va.title_id
	JOIN vpp_app_team_software_categories vatsc ON vatsc.vpp_app_team_id = vat.id
	JOIN software_categories sc ON vatsc.software_category_id = sc.id
WHERE
	st.id IN (?) AND vat.global_or_team_id = ?;
`

	var tmID uint
	if teamID != nil {
		tmID = *teamID
	}

	stmt, args, err := sqlx.In(stmt, softwareTitleIDs, tmID, softwareTitleIDs, tmID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "sqlx.In for get categories for software installers")
	}
	var categories []struct {
		TitleID      uint   `db:"title_id"`
		CategoryName string `db:"software_category_name"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &categories, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get categories for software installers")
	}

	ret := make(map[uint][]string, len(categories))
	for _, c := range categories {
		ret[c.TitleID] = append(ret[c.TitleID], c.CategoryName)
	}

	return ret, nil
}
