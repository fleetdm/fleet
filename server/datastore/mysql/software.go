package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log/level"
	"github.com/jmoiron/sqlx"
)

func softwareSliceToMap(softwares []fleet.Software) map[string]fleet.Software {
	result := make(map[string]fleet.Software)
	for _, s := range softwares {
		result[s.ToUniqueStr()] = s
	}
	return result
}

func (ds *Datastore) UpdateHostSoftware(ctx context.Context, hostID uint, software []fleet.Software) (*fleet.UpdateHostSoftwareDBResult, error) {
	var result *fleet.UpdateHostSoftwareDBResult
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		r, err := applyChangesForNewSoftwareDB(ctx, tx, hostID, software, ds.minLastOpenedAtDiff)
		result = r
		return err
	})
	if err != nil {
		return result, err
	}

	// We perform the following cleanup on a separate transaction to avoid deadlocks.
	//
	// Cleanup the software table when no more hosts have the deleted host_software
	// table entries. Otherwise the software will be listed by ds.ListSoftware but
	// ds.SoftwareByID, ds.CountHosts and ds.ListHosts will return a *notFoundError
	// error for such software.
	if len(result.Deleted) > 0 {
		deletesHostSoftwareIDs := make([]uint, 0, len(result.Deleted))
		for _, software := range result.Deleted {
			deletesHostSoftwareIDs = append(deletesHostSoftwareIDs, software.ID)
		}
		slices.Sort(deletesHostSoftwareIDs)
		if err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
			stmt := `DELETE FROM software WHERE id IN (?) AND NOT EXISTS (
				SELECT 1 FROM host_software hsw WHERE hsw.software_id = software.id
			)`
			stmt, args, err := sqlx.In(stmt, deletesHostSoftwareIDs)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "build delete software query")
			}
			if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
				return ctxerr.Wrap(ctx, err, "delete software")
			}
			return nil
		}); err != nil {
			return result, err
		}
	}

	return result, err
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
		SELECT t.id, t.host_id, t.software_id, t.installed_path
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

		key := fmt.Sprintf("%s%s%s", r.InstalledPath, fleet.SoftwareFieldSeparator, s.ToUniqueStr())
		iSPathLookup[key] = r

		// Anything stored but not reported should be deleted
		if _, ok := reported[key]; !ok {
			toDelete = append(toDelete, r.ID)
		}
	}

	for key := range reported {
		parts := strings.SplitN(key, fleet.SoftwareFieldSeparator, 2)
		iSPath, unqStr := parts[0], parts[1]

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
			HostID:        hostID,
			SoftwareID:    s.ID,
			InstalledPath: iSPath,
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

	stmt := "INSERT INTO host_software_installed_paths (host_id, software_id, installed_path) VALUES %s"
	batchSize := 500

	for i := 0; i < len(toInsert); i += batchSize {
		end := i + batchSize
		if end > len(toInsert) {
			end = len(toInsert)
		}
		batch := toInsert[i:end]

		var args []interface{}
		for _, v := range batch {
			args = append(args, v.HostID, v.SoftwareID, v.InstalledPath)
		}

		placeHolders := strings.TrimSuffix(strings.Repeat("(?, ?, ?), ", len(batch)), ", ")
		stmt := fmt.Sprintf(stmt, placeHolders)

		_, err := tx.ExecContext(ctx, stmt, args...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "inserting rows into host_software_installed_paths")
		}
	}

	return nil
}

func nothingChanged(current, incoming []fleet.Software, minLastOpenedAtDiff time.Duration) bool {
	if len(current) != len(incoming) {
		return false
	}

	currentMap := make(map[string]fleet.Software)
	for _, s := range current {
		currentMap[s.ToUniqueStr()] = s
	}
	for _, s := range incoming {
		cur, ok := currentMap[s.ToUniqueStr()]
		if !ok {
			return false
		}

		// if the incoming software has a last opened at timestamp and it differs
		// significantly from the current timestamp (or there is no current
		// timestamp), then consider that something changed.
		if s.LastOpenedAt != nil {
			if cur.LastOpenedAt == nil {
				return false
			}

			oldLast := *cur.LastOpenedAt
			newLast := *s.LastOpenedAt
			if newLast.Sub(oldLast) >= minLastOpenedAtDiff {
				return false
			}
		}
	}

	return true
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
func applyChangesForNewSoftwareDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	hostID uint,
	software []fleet.Software,
	minLastOpenedAtDiff time.Duration,
) (*fleet.UpdateHostSoftwareDBResult, error) {
	r := &fleet.UpdateHostSoftwareDBResult{}

	currentSoftware, err := listSoftwareByHostIDShort(ctx, tx, hostID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "loading current software for host")
	}
	r.WasCurrInstalled = currentSoftware

	if nothingChanged(currentSoftware, software, minLastOpenedAtDiff) {
		return r, nil
	}

	current := softwareSliceToMap(currentSoftware)
	incoming := softwareSliceToMap(software)

	deleted, err := deleteUninstalledHostSoftwareDB(ctx, tx, hostID, current, incoming)
	if err != nil {
		return nil, err
	}
	r.Deleted = deleted

	inserted, err := insertNewInstalledHostSoftwareDB(ctx, tx, hostID, current, incoming)
	if err != nil {
		return nil, err
	}
	r.Inserted = inserted

	if err = updateModifiedHostSoftwareDB(ctx, tx, hostID, current, incoming, minLastOpenedAtDiff); err != nil {
		return nil, err
	}

	if err = updateSoftwareUpdatedAt(ctx, tx, hostID); err != nil {
		return nil, err
	}

	return r, nil
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

func getOrGenerateSoftwareIdDB(ctx context.Context, tx sqlx.ExtContext, s fleet.Software) (uint, error) {
	getExistingID := func() (int64, error) {
		var existingID int64
		if err := sqlx.GetContext(ctx, tx, &existingID,
			"SELECT id FROM software "+
				"WHERE name = ? AND version = ? AND source = ? AND `release` = ? AND "+
				"vendor = ? AND arch = ? AND bundle_identifier = ? AND extension_id = ? AND browser = ? LIMIT 1",
			s.Name, s.Version, s.Source, s.Release, s.Vendor, s.Arch, s.BundleIdentifier, s.ExtensionID, s.Browser,
		); err != nil {
			return 0, err
		}
		return existingID, nil
	}

	switch id, err := getExistingID(); {
	case err == nil:
		return uint(id), nil
	case errors.Is(err, sql.ErrNoRows):
		// OK
	default:
		return 0, ctxerr.Wrap(ctx, err, "get software")
	}

	_, err := tx.ExecContext(ctx,
		fmt.Sprintf("INSERT INTO software "+
			"(name, version, source, `release`, vendor, arch, bundle_identifier, extension_id, browser, checksum) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, %s)", softwareChecksumComputedColumn("")),
		s.Name, s.Version, s.Source, s.Release, s.Vendor, s.Arch, s.BundleIdentifier, s.ExtensionID, s.Browser,
	)
	if err != nil {
		if !isDuplicate(err) {
			return 0, ctxerr.Wrap(ctx, err, "insert software")
		}
		// if the error is a duplicate software entry, there was a race and another
		// process inserted that software, so continue and try to get its id as it
		// now exists.
	}

	// LastInsertId sometimes returns 0 as it's dependent on connections and how mysql is
	// configured.
	switch id, err := getExistingID(); {
	case err == nil:
		return uint(id), nil
	case errors.Is(err, sql.ErrNoRows):
		return 0, doRetryErr
	default:
		return 0, ctxerr.Wrap(ctx, err, "get software")
	}
}

func softwareChecksumComputedColumn(tableAlias string) string {
	if tableAlias != "" && !strings.HasSuffix(tableAlias, ".") {
		tableAlias += "."
	}

	// concatenate with separator \x00
	return fmt.Sprintf(` UNHEX(
		MD5(
			CONCAT_WS(CHAR(0),
				%sname,
				%[1]sversion,
				%[1]ssource,
				COALESCE(%[1]sbundle_identifier, ''),
				`+"%[1]s`release`"+`,
				%[1]sarch,
				%[1]svendor,
				%[1]sbrowser,
				%[1]sextension_id
			)
		)
	) `, tableAlias)
}

// insert host_software that is in incoming map, but not in current map.
// returns the inserted software on the host
func insertNewInstalledHostSoftwareDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	hostID uint,
	currentMap map[string]fleet.Software,
	incomingMap map[string]fleet.Software,
) ([]fleet.Software, error) {
	var insertsHostSoftware []interface{}
	var insertedSoftware []fleet.Software

	type softwareWithUniqueName struct {
		uniqueName string
		software   fleet.Software
	}
	incomingOrdered := make([]softwareWithUniqueName, 0, len(incomingMap))
	for uniqueName, software := range incomingMap {
		incomingOrdered = append(incomingOrdered, softwareWithUniqueName{
			uniqueName: uniqueName,
			software:   software,
		})
	}
	sort.Slice(incomingOrdered, func(i, j int) bool {
		return incomingOrdered[i].uniqueName < incomingOrdered[j].uniqueName
	})

	for _, s := range incomingOrdered {
		if _, ok := currentMap[s.uniqueName]; !ok {
			id, err := getOrGenerateSoftwareIdDB(ctx, tx, s.software)
			if err != nil {
				return nil, err
			}
			insertsHostSoftware = append(insertsHostSoftware, hostID, id, s.software.LastOpenedAt)

			s.software.ID = id
			insertedSoftware = append(insertedSoftware, s.software)
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
	const stmt = `UPDATE host_software SET last_opened_at = ? WHERE host_id = ? AND software_id = ?`

	var keysToUpdate []string
	for key, newSw := range incomingMap {
		curSw, ok := currentMap[key]
		if !ok || newSw.LastOpenedAt == nil {
			// software must also exist in current map, and new software must have a
			// last opened at timestamp (otherwise we don't overwrite the old one)
			continue
		}

		if curSw.LastOpenedAt == nil || (*newSw.LastOpenedAt).Sub(*curSw.LastOpenedAt) >= minLastOpenedAtDiff {
			keysToUpdate = append(keysToUpdate, key)
		}
	}
	sort.Strings(keysToUpdate)

	for _, key := range keysToUpdate {
		curSw, newSw := currentMap[key], incomingMap[key]
		if _, err := tx.ExecContext(ctx, stmt, newSw.LastOpenedAt, hostID, curSw.ID); err != nil {
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
			)

		if opts.TeamID != nil {
			ds = ds.Where(goqu.I("shc.team_id").Eq(opts.TeamID))
		} else {
			ds = ds.Where(goqu.I("shc.team_id").Eq(0))
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
		ds = ds.
			LeftJoin(
				goqu.I("cve_meta").As("c"),
				goqu.On(goqu.I("c.cve").Eq(goqu.I("scv.cve"))),
			).
			SelectAppend(
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

	lookup := make(map[uint][]string)
	for _, ip := range installedPaths {
		lookup[ip.SoftwareID] = append(lookup[ip.SoftwareID], ip.InstalledPath)
	}

	host.Software = make([]fleet.HostSoftwareEntry, 0, len(software))
	for _, s := range software {
		host.Software = append(host.Software, fleet.HostSoftwareEntry{
			Software:       s,
			InstalledPaths: lookup[s.ID],
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
		s.id, s.name, s.version, s.source, s.bundle_identifier, s.release, s.arch, s.vendor, s.browser, s.extension_id, s.title_id ,
		COALESCE(sc.cpe, '') AS generated_cpe
	FROM software s
	LEFT JOIN software_cpe sc ON (s.id=sc.software_id)`

	var conditionals []string
	arg := map[string]interface{}{}

	if len(query.ExcludedSources) != 0 {
		conditionals = append(conditionals, "s.source NOT IN (:excluded_sources)")
		arg["excluded_sources"] = query.ExcludedSources
	}

	if len(query.IncludedSources) != 0 {
		conditionals = append(conditionals, "s.source IN (:included_sources)")
		arg["included_sources"] = query.IncludedSources
	}

	if len(conditionals) != 0 {
		cond := strings.Join(conditionals, " AND ")
		stmt, args, err = sqlx.Named(stmt+" WHERE "+cond, arg)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "error binding named arguments on software iterator")
		}
		stmt, args, err = sqlx.In(stmt, args...)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "error building 'In' query part on software iterator")
		}
	}

	rows, err := ds.reader(ctx).QueryxContext(ctx, stmt, args...) //nolint:sqlclosecheck
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load host software")
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
		if len(software) > int(perPage) {
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
	// filter software that is not associated with any hosts
	if teamID == nil {
		q = q.Where(goqu.L("EXISTS (SELECT 1 FROM host_software WHERE software_id = ? LIMIT 1)", id))
	} else {
		// if teamID filter is used, host counts need to be up-to-date
		q = q.Where(
			goqu.L(
				"EXISTS (SELECT 1 FROM software_host_counts WHERE software_id = ? AND team_id = ? AND hosts_count > 0)", id, *teamID,
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
      SELECT count(*), 0 as team_id, software_id
      FROM host_software
      WHERE software_id > 0
      GROUP BY software_id`

		teamCountsStmt = `
      SELECT count(*), h.team_id, hs.software_id
      FROM host_software hs
      INNER JOIN hosts h
      ON hs.host_id = h.id
      WHERE h.team_id IS NOT NULL AND hs.software_id > 0
      GROUP BY hs.software_id, h.team_id`

		insertStmt = `
      INSERT INTO software_host_counts
        (software_id, hosts_count, team_id, updated_at)
      VALUES
        %s
      ON DUPLICATE KEY UPDATE
        hosts_count = VALUES(hosts_count),
        updated_at = VALUES(updated_at)`

		valuesPart = `(?, ?, ?, ?),`

		cleanupSoftwareStmt = `
      DELETE s
      FROM software s
      LEFT JOIN software_host_counts shc
      ON s.id = shc.software_id
      WHERE
        shc.software_id IS NULL OR
        (shc.team_id = 0 AND shc.hosts_count = 0)`

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

	// next get a cursor for the global and team counts for each software
	stmtLabel := []string{"global", "team"}
	for i, countStmt := range []string{globalCountsStmt, teamCountsStmt} {
		rows, err := ds.reader(ctx).QueryContext(ctx, countStmt)
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
				count  int
				teamID uint
				sid    uint
			)

			if err := rows.Scan(&count, &teamID, &sid); err != nil {
				return ctxerr.Wrapf(ctx, err, "scan %s row into variables", stmtLabel[i])
			}

			args = append(args, sid, count, teamID, updatedAt)
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

	// ensure all software titles are in the software_titles table
	upsertTitlesStmt := `
INSERT INTO software_titles (name, source, browser)
SELECT DISTINCT
	name,
	source,
	browser
FROM
	software s
WHERE
	NOT EXISTS (SELECT 1 FROM software_titles st WHERE (s.name, s.source, s.browser) = (st.name, st.source, st.browser))
ON DUPLICATE KEY UPDATE software_titles.id = software_titles.id`
	// TODO: consider the impact of on duplicate key update vs. risk of insert ignore
	// or performing a select first to see if the title exists and only inserting
	// new titles

	res, err := ds.writer(ctx).ExecContext(ctx, upsertTitlesStmt)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "upsert software titles")
	}
	n, _ := res.RowsAffected()
	level.Debug(ds.logger).Log("msg", "upsert software titles", "rows_affected", n)

	// update title ids for software table entries
	updateSoftwareStmt := `
UPDATE
	software s,
	software_titles st
SET
	s.title_id = st.id
WHERE
	(s.name, s.source, s.browser) = (st.name, st.source, st.browser)
	AND (s.title_id IS NULL OR s.title_id != st.id)`

	res, err = ds.writer(ctx).ExecContext(ctx, updateSoftwareStmt)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "update software title_id")
	}
	n, _ = res.RowsAffected()
	level.Debug(ds.logger).Log("msg", "update software title_id", "rows_affected", n)

	// clean up orphaned software titles
	cleanupStmt := `
DELETE st FROM software_titles st
	LEFT JOIN software s ON s.title_id = st.id
	WHERE s.title_id IS NULL`

	res, err = ds.writer(ctx).ExecContext(ctx, cleanupStmt)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "cleanup orphaned software titles")
	}
	n, _ = res.RowsAffected()
	level.Debug(ds.logger).Log("msg", "cleanup orphaned software titles", "rows_affected", n)

	return nil
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

	return insertOnDuplicateDidInsert(res), nil
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
	hostID uint,
) ([]fleet.Software, error) {
	var result []fleet.Software

	stmt := dialect.
		From(goqu.T("software").As("s")).
		LeftJoin(
			goqu.T("software_cpe").As("cpe"),
			goqu.On(goqu.Ex{
				"s.id": goqu.I("cpe.software_id"),
			}),
		).
		Join(
			goqu.T("host_software").As("hs"),
			goqu.On(goqu.Ex{
				"s.id": goqu.I("hs.software_id"),
			}),
		).
		Select(
			goqu.I("s.id"),
			goqu.I("s.name"),
			goqu.I("s.version"),
			goqu.I("s.release"),
			goqu.I("s.arch"),
			goqu.COALESCE(goqu.I("cpe.cpe"), "").As("generated_cpe"),
		).
		Where(goqu.C("host_id").Eq(hostID))

	sql, args, err := stmt.ToSQL()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "error generating SQL statement")
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &result, sql, args...); err != nil {
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
