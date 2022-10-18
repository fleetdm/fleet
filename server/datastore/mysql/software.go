package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

const (
	maxSoftwareNameLen             = 255
	maxSoftwareVersionLen          = 255
	maxSoftwareSourceLen           = 64
	maxSoftwareBundleIdentifierLen = 255

	maxSoftwareReleaseLen = 64
	maxSoftwareVendorLen  = 32
	maxSoftwareArchLen    = 16
)

func truncateString(str string, length int) string {
	if len(str) > length {
		return str[:length]
	}
	return str
}

func softwareToUniqueString(s fleet.Software) string {
	ss := []string{s.Name, s.Version, s.Source, s.BundleIdentifier}
	// Release, Vendor and Arch fields were added on a migration,
	// thus we only include them in the string if at least one of them is defined.
	if s.Release != "" || s.Vendor != "" || s.Arch != "" {
		ss = append(ss, s.Release, s.Vendor, s.Arch)
	}
	return strings.Join(ss, "\u0000")
}

func uniqueStringToSoftware(s string) fleet.Software {
	parts := strings.Split(s, "\u0000")

	// Release, Vendor and Arch fields were added on a migration,
	// If one of them is defined, then they are included in the string.
	var release, vendor, arch string
	if len(parts) > 4 {
		release = truncateString(parts[4], maxSoftwareReleaseLen)
		vendor = truncateString(parts[5], maxSoftwareVendorLen)
		arch = truncateString(parts[6], maxSoftwareArchLen)
	}

	return fleet.Software{
		Name:             truncateString(parts[0], maxSoftwareNameLen),
		Version:          truncateString(parts[1], maxSoftwareVersionLen),
		Source:           truncateString(parts[2], maxSoftwareSourceLen),
		BundleIdentifier: truncateString(parts[3], maxSoftwareBundleIdentifierLen),

		Release: release,
		Vendor:  vendor,
		Arch:    arch,
	}
}

func softwareSliceToMap(softwares []fleet.Software) map[string]fleet.Software {
	result := make(map[string]fleet.Software)
	for _, s := range softwares {
		result[softwareToUniqueString(s)] = s
	}
	return result
}

// UpdateHostSoftware updates the software list of a host.
// The update consists of deleting existing entries that are not in the given `software`
// slice, updating existing entries and inserting new entries.
func (ds *Datastore) UpdateHostSoftware(ctx context.Context, hostID uint, software []fleet.Software) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		return applyChangesForNewSoftwareDB(ctx, tx, hostID, software, ds.minLastOpenedAtDiff)
	})
}

func nothingChanged(current, incoming []fleet.Software, minLastOpenedAtDiff time.Duration) bool {
	if len(current) != len(incoming) {
		return false
	}

	currentMap := make(map[string]fleet.Software)
	for _, s := range current {
		currentMap[softwareToUniqueString(s)] = s
	}
	for _, s := range incoming {
		cur, ok := currentMap[softwareToUniqueString(s)]
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
	return listSoftwareByHostIDShort(ctx, ds.reader, hostID)
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
    s.bundle_identifier,
    s.release,
    s.vendor,
    s.arch,
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

func applyChangesForNewSoftwareDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	hostID uint,
	software []fleet.Software,
	minLastOpenedAtDiff time.Duration,
) error {
	currentSoftware, err := listSoftwareByHostIDShort(ctx, tx, hostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "loading current software for host")
	}

	if nothingChanged(currentSoftware, software, minLastOpenedAtDiff) {
		return nil
	}

	current := softwareSliceToMap(currentSoftware)
	incoming := softwareSliceToMap(software)

	if err = deleteUninstalledHostSoftwareDB(ctx, tx, hostID, current, incoming); err != nil {
		return err
	}

	if err = insertNewInstalledHostSoftwareDB(ctx, tx, hostID, current, incoming); err != nil {
		return err
	}

	if err = updateModifiedHostSoftwareDB(ctx, tx, hostID, current, incoming, minLastOpenedAtDiff); err != nil {
		return err
	}

	return nil
}

// delete host_software that is in current map, but not in incoming map.
func deleteUninstalledHostSoftwareDB(
	ctx context.Context,
	tx sqlx.ExecerContext,
	hostID uint,
	currentMap map[string]fleet.Software,
	incomingMap map[string]fleet.Software,
) error {
	var deletesHostSoftware []interface{}
	deletesHostSoftware = append(deletesHostSoftware, hostID)

	for currentKey, curSw := range currentMap {
		if _, ok := incomingMap[currentKey]; !ok {
			deletesHostSoftware = append(deletesHostSoftware, curSw.ID)
		}
	}
	if len(deletesHostSoftware) <= 1 {
		return nil
	}
	sql := fmt.Sprintf(
		`DELETE FROM host_software WHERE host_id = ? AND software_id IN (%s)`,
		strings.TrimSuffix(strings.Repeat("?,", len(deletesHostSoftware)-1), ","),
	)
	if _, err := tx.ExecContext(ctx, sql, deletesHostSoftware...); err != nil {
		return ctxerr.Wrap(ctx, err, "delete host software")
	}

	return nil
}

func getOrGenerateSoftwareIdDB(ctx context.Context, tx sqlx.ExtContext, s fleet.Software) (uint, error) {
	getExistingID := func() (int64, error) {
		var existingID int64
		if err := sqlx.GetContext(ctx, tx, &existingID,
			"SELECT id FROM software "+
				"WHERE name = ? AND version = ? AND source = ? AND `release` = ? AND "+
				"vendor = ? AND arch = ? AND bundle_identifier = ? LIMIT 1",
			s.Name, s.Version, s.Source, s.Release, s.Vendor, s.Arch, s.BundleIdentifier,
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
		"INSERT INTO software "+
			"(name, version, source, `release`, vendor, arch, bundle_identifier) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?) "+
			"ON DUPLICATE KEY UPDATE bundle_identifier=VALUES(bundle_identifier)",
		s.Name, s.Version, s.Source, s.Release, s.Vendor, s.Arch, s.BundleIdentifier,
	)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "insert software")
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

// insert host_software that is in incoming map, but not in current map.
func insertNewInstalledHostSoftwareDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	hostID uint,
	currentMap map[string]fleet.Software,
	incomingMap map[string]fleet.Software,
) error {
	var insertsHostSoftware []interface{}

	incomingOrdered := make([]string, 0, len(incomingMap))
	for s := range incomingMap {
		incomingOrdered = append(incomingOrdered, s)
	}
	sort.Strings(incomingOrdered)

	for _, s := range incomingOrdered {
		if _, ok := currentMap[s]; !ok {
			id, err := getOrGenerateSoftwareIdDB(ctx, tx, uniqueStringToSoftware(s))
			if err != nil {
				return err
			}
			sw := incomingMap[s]
			insertsHostSoftware = append(insertsHostSoftware, hostID, id, sw.LastOpenedAt)
		}
	}

	if len(insertsHostSoftware) > 0 {
		values := strings.TrimSuffix(strings.Repeat("(?,?,?),", len(insertsHostSoftware)/3), ",")
		sql := fmt.Sprintf(`INSERT IGNORE INTO host_software (host_id, software_id, last_opened_at) VALUES %s`, values)
		if _, err := tx.ExecContext(ctx, sql, insertsHostSoftware...); err != nil {
			return ctxerr.Wrap(ctx, err, "insert host software")
		}
	}

	return nil
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
			}
			softwares[idx].Vulnerabilities = append(softwares[idx].Vulnerabilities, cve)
		}
	}

	return softwares, nil
}

// softwareCVE is used for left joins with cve
type softwareCVE struct {
	fleet.Software
	CVE              *string  `db:"cve"`
	CVSSScore        *float64 `db:"cvss_score"`
	EPSSProbability  *float64 `db:"epss_probability"`
	CISAKnownExploit *bool    `db:"cisa_known_exploit"`
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
				goqu.MAX("c.cvss_score").As("cvss_score"),                 // for ordering
				goqu.MAX("c.epss_probability").As("epss_probability"),     // for ordering
				goqu.MAX("c.cisa_known_exploit").As("cisa_known_exploit"), // for ordering
			)
	}

	if match := opts.MatchQuery; match != "" {
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
		MatchQuery: opts.MatchQuery,
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
	software, err := listSoftwareDB(ctx, ds.reader, opts)
	if err != nil {
		return err
	}
	host.Software = software
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

// AllSoftwareWithoutCPEIterator Returns an iterator for the 'software' table, filtering out
// software entries with CPEs and from the sources included in the 'excludedSources' param.
func (ds *Datastore) AllSoftwareWithoutCPEIterator(ctx context.Context, excludedSources []string) (fleet.SoftwareIterator, error) {
	var err error
	var args []interface{}

	stmt := `SELECT s.* FROM software s LEFT JOIN software_cpe sc ON (s.id=sc.software_id) WHERE sc.id IS NULL`
	// The rows.Close call is done by the caller once iteration using the
	// returned fleet.SoftwareIterator is done.
	if excludedSources != nil {
		stmt += ` AND s.source NOT IN (?)`
		stmt, args, err = sqlx.In(stmt, excludedSources)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "loads cpes")
		}
	}

	rows, err := ds.reader.QueryxContext(ctx, stmt, args...) //nolint:sqlclosecheck
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load host software")
	}
	return &softwareIterator{rows: rows}, nil
}

func (ds *Datastore) AddCPEForSoftware(ctx context.Context, software fleet.Software, cpe string) error {
	_, err := addCPEForSoftwareDB(ctx, ds.writer, software, cpe)
	return err
}

func addCPEForSoftwareDB(ctx context.Context, exec sqlx.ExecerContext, software fleet.Software, cpe string) (uint, error) {
	sql := `INSERT INTO software_cpe (software_id, cpe) VALUES (?, ?)`
	res, err := exec.ExecContext(ctx, sql, software.ID, cpe)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "insert software cpe")
	}
	id, _ := res.LastInsertId() // cannot fail with the mysql driver
	return uint(id), nil
}

func (ds *Datastore) ListSoftwareCPEs(ctx context.Context) ([]fleet.SoftwareCPE, error) {
	var result []fleet.SoftwareCPE

	var err error
	var args []interface{}

	stmt := `SELECT id, software_id, cpe FROM software_cpe`
	err = sqlx.SelectContext(ctx, ds.reader, &result, stmt, args...)

	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "loads cpes")
	}
	return result, nil
}

func (ds *Datastore) ListSoftware(ctx context.Context, opt fleet.SoftwareListOptions) ([]fleet.Software, error) {
	return listSoftwareDB(ctx, ds.reader, opt)
}

func (ds *Datastore) CountSoftware(ctx context.Context, opt fleet.SoftwareListOptions) (int, error) {
	return countSoftwareDB(ctx, ds.reader, opt)
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
	if _, err := ds.writer.ExecContext(ctx, sql, args...); err != nil {
		return ctxerr.Wrapf(ctx, err, "deleting vulnerable software")
	}
	return nil
}

func (ds *Datastore) SoftwareByID(ctx context.Context, id uint, includeCVEScores bool) (*fleet.Software, error) {
	q := dialect.From(goqu.I("software").As("s")).
		Select(
			"s.id",
			"s.name",
			"s.version",
			"s.source",
			"s.bundle_identifier",
			"s.release",
			"s.vendor",
			"s.arch",
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
			)
	}

	q = q.Where(goqu.I("s.id").Eq(id))
	// filter software that is not associated with any hosts
	q = q.Where(goqu.L("EXISTS (SELECT 1 FROM host_software WHERE software_id = ? LIMIT 1)", id))

	sql, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	var results []softwareCVE
	err = sqlx.SelectContext(ctx, ds.reader, &results, sql, args...)
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
	if _, err := ds.writer.ExecContext(ctx, resetStmt, updatedAt); err != nil {
		return ctxerr.Wrap(ctx, err, "reset all software_host_counts to 0")
	}

	// next get a cursor for the global and team counts for each software
	stmtLabel := []string{"global", "team"}
	for i, countStmt := range []string{globalCountsStmt, teamCountsStmt} {
		rows, err := ds.reader.QueryContext(ctx, countStmt)
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
				if _, err := ds.writer.ExecContext(ctx, fmt.Sprintf(insertStmt, values), args...); err != nil {
					return ctxerr.Wrapf(ctx, err, "insert %s batch into software_host_counts", stmtLabel[i])
				}

				args = args[:0]
				batchCount = 0
			}
		}
		if batchCount > 0 {
			values := strings.TrimSuffix(strings.Repeat(valuesPart, batchCount), ",")
			if _, err := ds.writer.ExecContext(ctx, fmt.Sprintf(insertStmt, values), args...); err != nil {
				return ctxerr.Wrapf(ctx, err, "insert last %s batch into software_host_counts", stmtLabel[i])
			}
		}
		if err := rows.Err(); err != nil {
			return ctxerr.Wrapf(ctx, err, "iterate over %s host_software counts", stmtLabel[i])
		}
		rows.Close()
	}

	// remove any unused software (global counts = 0)
	if _, err := ds.writer.ExecContext(ctx, cleanupSoftwareStmt); err != nil {
		return ctxerr.Wrap(ctx, err, "delete unused software")
	}

	// remove any software count row for software that don't exist anymore
	if _, err := ds.writer.ExecContext(ctx, cleanupOrphanedStmt); err != nil {
		return ctxerr.Wrap(ctx, err, "delete software_host_counts for non-existing teams")
	}

	// remove any software count row for teams that don't exist anymore
	if _, err := ds.writer.ExecContext(ctx, cleanupTeamStmt); err != nil {
		return ctxerr.Wrap(ctx, err, "delete software_host_counts for non-existing teams")
	}
	return nil
}

// HostsBySoftwareIDs returns a list of all hosts that have at least one of the specified Software
// installed. It returns a minimal represention of matching hosts.
func (ds *Datastore) HostsBySoftwareIDs(ctx context.Context, softwareIDs []uint) ([]*fleet.HostShort, error) {
	queryStmt := `
    SELECT 
      h.id,
      h.hostname,
      if(h.computer_name = '', h.hostname, h.computer_name) display_name
    FROM
      hosts h
    INNER JOIN
      host_software hs
    ON
      h.id = hs.host_id
    WHERE
      hs.software_id IN (?)
	GROUP BY h.id, h.hostname
    ORDER BY
      h.id`

	stmt, args, err := sqlx.In(queryStmt, softwareIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building query args")
	}
	var hosts []*fleet.HostShort
	if err := sqlx.SelectContext(ctx, ds.reader, &hosts, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select hosts by cpes")
	}
	return hosts, nil
}

func (ds *Datastore) HostsByCVE(ctx context.Context, cve string) ([]*fleet.HostShort, error) {
	query := `
SELECT DISTINCT
    	(h.id),
    	h.hostname,
    	if(h.computer_name = '', h.hostname, h.computer_name) display_name
FROM
    hosts h
    INNER JOIN host_software hs ON h.id = hs.host_id
    INNER JOIN software_cve scv ON scv.software_id = hs.software_id
WHERE
    scv.cve = ?
ORDER BY
    h.id
`

	var hosts []*fleet.HostShort
	if err := sqlx.SelectContext(ctx, ds.reader, &hosts, query, cve); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select hosts by cves")
	}
	return hosts, nil
}

func (ds *Datastore) InsertCVEMeta(ctx context.Context, cveMeta []fleet.CVEMeta) error {
	query := `
INSERT INTO cve_meta (cve, cvss_score, epss_probability, cisa_known_exploit, published)
VALUES %s
ON DUPLICATE KEY UPDATE
    cvss_score = VALUES(cvss_score),
    epss_probability = VALUES(epss_probability),
    cisa_known_exploit = VALUES(cisa_known_exploit),
    published = VALUES(published)
`

	batchSize := 500
	for i := 0; i < len(cveMeta); i += batchSize {
		end := i + batchSize
		if end > len(cveMeta) {
			end = len(cveMeta)
		}

		batch := cveMeta[i:end]

		valuesFrag := strings.TrimSuffix(strings.Repeat("(?, ?, ?, ?, ?), ", len(batch)), ", ")
		var args []interface{}
		for _, meta := range batch {
			args = append(args, meta.CVE, meta.CVSSScore, meta.EPSSProbability, meta.CISAKnownExploit, meta.Published)
		}

		query := fmt.Sprintf(query, valuesFrag)

		_, err := ds.writer.ExecContext(ctx, query, args...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert cve scores")
		}
	}

	return nil
}

func (ds *Datastore) InsertVulnerabilities(
	ctx context.Context,
	vulns []fleet.SoftwareVulnerability,
	source fleet.VulnerabilitySource,
) (int64, error) {
	var args []interface{}

	if len(vulns) == 0 {
		return 0, nil
	}

	values := strings.TrimSuffix(strings.Repeat("(?,?,?),", len(vulns)), ",")
	sql := fmt.Sprintf(`INSERT IGNORE INTO software_cve (cve, source, software_id) VALUES %s`, values)

	for _, v := range vulns {
		args = append(args, v.CVE, source, v.SoftwareID)
	}
	res, err := ds.writer.ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "insert software vulnerabilities")
	}
	count, _ := res.RowsAffected()

	return count, nil
}

func (ds *Datastore) ListSoftwareVulnerabilities(
	ctx context.Context,
	hostIDs []uint,
) (map[uint][]fleet.SoftwareVulnerability, error) {
	result := make(map[uint][]fleet.SoftwareVulnerability)

	type softwareVulnerabilityWithHostId struct {
		fleet.SoftwareVulnerability
		HostId uint `db:"host_id"`
	}
	var queryR []softwareVulnerabilityWithHostId

	stmt := dialect.
		From(goqu.T("software_cve").As("cve")).
		Join(
			goqu.T("host_software").As("hs"),
			goqu.On(goqu.Ex{
				"cve.software_id": goqu.I("hs.software_id"),
			}),
		).
		Select(
			goqu.I("hs.host_id").As("host_id"),
			goqu.I("cve.software_id"),
			goqu.I("cve"),
		).
		Where(goqu.C("host_id").In(hostIDs))

	sql, args, err := stmt.ToSQL()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "error generating SQL statement")
	}

	if err := sqlx.SelectContext(ctx, ds.reader, &queryR, sql, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "error executing SQL statement")
	}

	for _, r := range queryR {
		result[r.HostId] = append(result[r.HostId], r.SoftwareVulnerability)
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

	if err := sqlx.SelectContext(ctx, ds.reader, &result, sql, args...); err != nil {
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
		).
		Where(goqu.C("published").Gte(maxAgeDate))

	sql, args, err := stmt.ToSQL()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "error generating SQL statement")
	}

	if err := sqlx.SelectContext(ctx, ds.reader, &result, sql, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "error executing SQL statement")
	}

	return result, nil
}
