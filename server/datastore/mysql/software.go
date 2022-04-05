package mysql

import (
	"context"
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

func softwareSliceToSet(softwares []fleet.Software) map[string]struct{} {
	result := make(map[string]struct{})
	for _, s := range softwares {
		result[softwareToUniqueString(s)] = struct{}{}
	}
	return result
}

func softwareSliceToIdMap(softwareSlice []fleet.Software) map[string]uint {
	result := make(map[string]uint)
	for _, s := range softwareSlice {
		result[softwareToUniqueString(s)] = s.ID
	}
	return result
}

// UpdateHostSoftware updates the software list of a host.
// The update consists of deleting existing entries that are not in the given `software`
// slice, updating existing entries and inserting new entries.
func (ds *Datastore) UpdateHostSoftware(ctx context.Context, hostID uint, software []fleet.Software) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		return applyChangesForNewSoftwareDB(ctx, tx, hostID, software)
	})
}

func saveHostSoftwareDB(ctx context.Context, tx sqlx.ExtContext, host *fleet.Host) error {
	if err := applyChangesForNewSoftwareDB(ctx, tx, host.ID, host.Software); err != nil {
		return err
	}

	host.HostSoftware.Modified = false
	return nil
}

func nothingChanged(current []fleet.Software, incoming []fleet.Software) bool {
	if len(current) != len(incoming) {
		return false
	}

	currentBitmap := make(map[string]bool)
	for _, s := range current {
		currentBitmap[softwareToUniqueString(s)] = true
	}
	for _, s := range incoming {
		if _, ok := currentBitmap[softwareToUniqueString(s)]; !ok {
			return false
		}
	}

	return true
}

func applyChangesForNewSoftwareDB(ctx context.Context, tx sqlx.ExtContext, hostID uint, software []fleet.Software) error {
	storedCurrentSoftware, err := listSoftwareDB(ctx, tx, &hostID, fleet.SoftwareListOptions{SkipLoadingCVEs: true})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "loading current software for host")
	}

	if nothingChanged(storedCurrentSoftware, software) {
		return nil
	}

	current := softwareSliceToIdMap(storedCurrentSoftware)
	incoming := softwareSliceToSet(software)

	if err = deleteUninstalledHostSoftwareDB(ctx, tx, hostID, current, incoming); err != nil {
		return err
	}

	if err = insertNewInstalledHostSoftwareDB(ctx, tx, hostID, current, incoming); err != nil {
		return err
	}

	return nil
}

func deleteUninstalledHostSoftwareDB(
	ctx context.Context,
	tx sqlx.ExecerContext,
	hostID uint,
	currentIdmap map[string]uint,
	incomingBitmap map[string]struct{},
) error {
	var deletesHostSoftware []interface{}
	deletesHostSoftware = append(deletesHostSoftware, hostID)

	for currentKey := range currentIdmap {
		if _, ok := incomingBitmap[currentKey]; !ok {
			deletesHostSoftware = append(deletesHostSoftware, currentIdmap[currentKey])
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
	var existingId []int64
	if err := sqlx.SelectContext(ctx, tx,
		&existingId,
		"SELECT id FROM software "+
			"WHERE name = ? AND version = ? AND source = ? AND `release` = ? AND "+
			"vendor = ? AND arch = ? AND bundle_identifier = ?",
		s.Name, s.Version, s.Source, s.Release, s.Vendor, s.Arch, s.BundleIdentifier,
	); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "get software")
	}
	if len(existingId) > 0 {
		return uint(existingId[0]), nil
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
	// LastInsertId sometimes returns 0 as it's dependent on connections and how mysql is configured
	// doing the select recursively is a bit slower, but most times, we won't end up in this situation
	return getOrGenerateSoftwareIdDB(ctx, tx, s)
}

func insertNewInstalledHostSoftwareDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	hostID uint,
	currentIdmap map[string]uint,
	incomingBitmap map[string]struct{},
) error {
	var insertsHostSoftware []interface{}
	incomingOrdered := make([]string, 0, len(incomingBitmap))
	for s := range incomingBitmap {
		incomingOrdered = append(incomingOrdered, s)
	}
	sort.Strings(incomingOrdered)
	for _, s := range incomingOrdered {
		if _, ok := currentIdmap[s]; !ok {
			id, err := getOrGenerateSoftwareIdDB(ctx, tx, uniqueStringToSoftware(s))
			if err != nil {
				return err
			}
			insertsHostSoftware = append(insertsHostSoftware, hostID, id)
		}
	}
	if len(insertsHostSoftware) > 0 {
		values := strings.TrimSuffix(strings.Repeat("(?,?),", len(insertsHostSoftware)/2), ",")
		sql := fmt.Sprintf(`INSERT IGNORE INTO host_software (host_id, software_id) VALUES %s`, values)
		if _, err := tx.ExecContext(ctx, sql, insertsHostSoftware...); err != nil {
			return ctxerr.Wrap(ctx, err, "insert host software")
		}
	}

	return nil
}

var dialect = goqu.Dialect("mysql")

// listSoftwareDB returns all the software installed in the given hostID and list options.
// If hostID is nil, then the method will look into the installed software of all hosts.
func listSoftwareDB(
	ctx context.Context, q sqlx.QueryerContext, hostID *uint, opts fleet.SoftwareListOptions,
) ([]fleet.Software, error) {
	sql, args, err := selectSoftwareSQL(hostID, opts)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "sql build")
	}

	var result []fleet.Software
	if err := sqlx.SelectContext(ctx, q, &result, sql, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select host software")
	}

	if opts.SkipLoadingCVEs {
		return result, nil
	}

	cvesBySoftware, err := loadCVEsBySoftware(ctx, q, hostID, opts)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load CVEs by software")
	}
	for i := range result {
		result[i].Vulnerabilities = cvesBySoftware[result[i].ID]
	}
	return result, nil
}

func selectSoftwareSQL(hostID *uint, opts fleet.SoftwareListOptions) (string, []interface{}, error) {
	ds := dialect.From(goqu.I("software").As("s")).Select(
		"s.*",
		goqu.COALESCE(goqu.I("scp.cpe"), "").As("generated_cpe"),
	)

	if hostID != nil || opts.TeamID != nil {
		ds = ds.Join(
			goqu.I("host_software").As("hs"),
			goqu.On(
				goqu.I("hs.software_id").Eq(goqu.I("s.id")),
			),
		)
	}

	if hostID != nil {
		ds = ds.Where(goqu.I("hs.host_id").Eq(hostID))
	}

	if opts.TeamID != nil {
		ds = ds.Join(
			goqu.I("hosts").As("h"),
			goqu.On(
				goqu.I("hs.host_id").Eq(goqu.I("h.id")),
			),
		).Where(goqu.I("h.team_id").Eq(opts.TeamID))
	}

	ds = ds.GroupBy(
		goqu.I("s.id"),
		goqu.I("s.name"),
		goqu.I("s.version"),
		goqu.I("s.source"),
		goqu.I("generated_cpe"),
	)

	if opts.VulnerableOnly {
		ds = ds.Join(
			goqu.I("software_cpe").As("scp"),
			goqu.On(
				goqu.I("s.id").Eq(goqu.I("scp.software_id")),
			),
		).Join(
			goqu.I("software_cve").As("scv"),
			goqu.On(goqu.I("scp.id").Eq(goqu.I("scv.cpe_id"))),
		)
	} else {
		ds = ds.LeftJoin(
			goqu.I("software_cpe").As("scp"),
			goqu.On(
				goqu.I("s.id").Eq(goqu.I("scp.software_id")),
			),
		)
		if opts.MatchQuery != "" {
			ds = ds.LeftJoin(
				goqu.I("software_cve").As("scv"),
				goqu.On(goqu.I("scp.id").Eq(goqu.I("scv.cpe_id"))),
			)
		}
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
		ds = ds.Join(
			goqu.I("software_host_counts").As("shc"),
			goqu.On(goqu.I("s.id").Eq(goqu.I("shc.software_id"))),
		).
			Where(goqu.I("shc.hosts_count").Gt(0)).
			SelectAppend(
				goqu.I("shc.hosts_count"),
				goqu.I("shc.updated_at").As("counts_updated_at"),
			)

		if opts.TeamID != nil {
			ds = ds.Where(goqu.I("shc.team_id").Eq(opts.TeamID))
		} else {
			ds = ds.Where(goqu.I("shc.team_id").Eq(0))
		}
	}
	ds = appendListOptionsToSelect(ds, opts.ListOptions)

	return ds.ToSQL()
}

func countSoftwareDB(
	ctx context.Context, q sqlx.QueryerContext, hostID *uint, opts fleet.SoftwareListOptions,
) (int, error) {
	opts.ListOptions = fleet.ListOptions{
		MatchQuery: opts.ListOptions.MatchQuery,
	}
	sql, args, err := selectSoftwareSQL(hostID, opts)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "sql build")
	}

	var result int
	if err := sqlx.GetContext(ctx, q, &result, "select count(*) as count from ("+sql+") s", args...); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "count host software")
	}
	return result, nil
}

// loadCVEsbySoftware loads all the CVEs on software installed on the given hostID and list options.
// If hostID is nil, then the method will look into the installed software of all hosts.
func loadCVEsBySoftware(
	ctx context.Context, q sqlx.QueryerContext, hostID *uint, opt fleet.SoftwareListOptions,
) (map[uint]fleet.VulnerabilitiesSlice, error) {
	ds := dialect.From(goqu.I("host_software").As("hs")).SelectDistinct(
		goqu.I("hs.software_id"),
		goqu.I("scv.cve"),
	).Join(
		goqu.I("hosts").As("h"),
		goqu.On(
			goqu.I("hs.host_id").Eq(goqu.I("h.id")),
		),
	).Join(
		goqu.I("software_cpe").As("scp"),
		goqu.On(
			goqu.I("hs.software_id").Eq(goqu.I("scp.software_id")),
		),
	).Join(
		goqu.I("software_cve").As("scv"),
		goqu.On(
			goqu.I("scp.id").Eq(goqu.I("scv.cpe_id")),
		),
	)

	if hostID != nil {
		ds = ds.Where(goqu.I("hs.host_id").Eq(hostID))
	}
	if opt.TeamID != nil {
		ds = ds.Where(goqu.I("h.team_id").Eq(opt.TeamID))
	}

	sql, args, err := ds.ToSQL()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "sql2 build")
	}

	rows, err := q.QueryxContext(ctx, sql, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load host software")
	}
	defer rows.Close()

	cvesBySoftware := make(map[uint]fleet.VulnerabilitiesSlice)
	for rows.Next() {
		var id uint
		var cve string
		if err := rows.Scan(&id, &cve); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "scanning cve")
		}
		cvesBySoftware[id] = append(cvesBySoftware[id], fleet.SoftwareCVE{
			CVE:         cve,
			DetailsLink: fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", cve),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "error iterating through cve rows")
	}
	return cvesBySoftware, nil
}

func (ds *Datastore) LoadHostSoftware(ctx context.Context, host *fleet.Host) error {
	host.HostSoftware = fleet.HostSoftware{Modified: false}
	software, err := listSoftwareDB(ctx, ds.reader, &host.ID, fleet.SoftwareListOptions{})
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

func (ds *Datastore) AllSoftwareWithoutCPEIterator(ctx context.Context) (fleet.SoftwareIterator, error) {
	sql := `SELECT s.* FROM software s LEFT JOIN software_cpe sc on (s.id=sc.software_id) WHERE sc.id is null`
	// The rows.Close call is done by the caller once iteration using the
	// returned fleet.SoftwareIterator is done.
	rows, err := ds.reader.QueryxContext(ctx, sql) //nolint:sqlclosecheck
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

func (ds *Datastore) AllCPEs(ctx context.Context) ([]string, error) {
	sql := `SELECT cpe FROM software_cpe`
	var cpes []string
	err := sqlx.SelectContext(ctx, ds.reader, &cpes, sql)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "loads cpes")
	}
	return cpes, nil
}

// InsertCVEForCPE inserts the cve into software_cve, linking it to all the
// provided cpes. It returns the number of new rows inserted or an error. If
// the CVE already existed for all CPEs, it would return 0, nil.
func (ds *Datastore) InsertCVEForCPE(ctx context.Context, cve string, cpes []string) (int64, error) { // TODO: this should return a bool (inserted) so that caller can determine if it's actually inserted
	var totalCount int64
	for _, cpe := range cpes {
		var ids []uint
		err := sqlx.Select(ds.writer, &ids, `SELECT id FROM software_cpe WHERE cpe=?`, cpe)
		if err != nil {
			return 0, err
		}

		values := strings.TrimSuffix(strings.Repeat("(?,?),", len(ids)), ",")
		sql := fmt.Sprintf(`INSERT IGNORE INTO software_cve (cpe_id, cve) VALUES %s`, values)

		var args []interface{}
		for _, id := range ids {
			args = append(args, id, cve)
		}
		res, err := ds.writer.ExecContext(ctx, sql, args...)
		if err != nil {
			return 0, ctxerr.Wrap(ctx, err, "insert software cve")
		}
		count, _ := res.RowsAffected()
		totalCount += count
	}

	return totalCount, nil
}

func (ds *Datastore) ListSoftware(ctx context.Context, opt fleet.SoftwareListOptions) ([]fleet.Software, error) {
	return listSoftwareDB(ctx, ds.reader, nil, opt)
}

func (ds *Datastore) CountSoftware(ctx context.Context, opt fleet.SoftwareListOptions) (int, error) {
	return countSoftwareDB(ctx, ds.reader, nil, opt)
}

// ListVulnerableSoftwareBySource lists all the vulnerable software that matches the given source.
func (ds *Datastore) ListVulnerableSoftwareBySource(ctx context.Context, source string) ([]fleet.SoftwareWithCPE, error) {
	var softwareCVEs []struct {
		fleet.Software
		CPE  uint   `db:"cpe_id"`
		CVEs string `db:"cves"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader, &softwareCVEs, `
		SELECT s.*, scv.cpe_id, GROUP_CONCAT(scv.cve SEPARATOR ',') as cves
		FROM software s
		JOIN software_cpe scp ON (s.id=scp.software_id)
		JOIN software_cve scv ON (scp.id=scv.cpe_id)
		WHERE s.source = ?
		GROUP BY scv.cpe_id
	`, source); err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "listing vulnerable software by source")
	}
	software := make([]fleet.SoftwareWithCPE, 0, len(softwareCVEs))
	for _, sc := range softwareCVEs {
		for _, cve := range strings.Split(sc.CVEs, ",") {
			sc.Software.Vulnerabilities = append(sc.Software.Vulnerabilities, fleet.SoftwareCVE{
				CVE:         cve,
				DetailsLink: fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", cve),
			})
		}
		software = append(software, fleet.SoftwareWithCPE{
			Software: sc.Software,
			CPEID:    sc.CPE,
		})
	}
	return software, nil
}

// DeleteVulnerabilitiesByCPECVE deletes the given list of vulnerabilities identified by CPE+CVE.
func (ds *Datastore) DeleteVulnerabilitiesByCPECVE(ctx context.Context, vulnerabilities []fleet.SoftwareVulnerability) error {
	if len(vulnerabilities) == 0 {
		return nil
	}

	sql := fmt.Sprintf(
		`DELETE FROM software_cve WHERE (cpe_id, cve) IN (%s)`,
		strings.TrimSuffix(strings.Repeat("(?,?),", len(vulnerabilities)), ","),
	)
	var args []interface{}
	for _, vulnerability := range vulnerabilities {
		args = append(args, vulnerability.CPEID, vulnerability.CVE)
	}
	if _, err := ds.writer.ExecContext(ctx, sql, args...); err != nil {
		return ctxerr.Wrapf(ctx, err, "deleting vulnerable software")
	}
	return nil
}

func (ds *Datastore) SoftwareByID(ctx context.Context, id uint) (*fleet.Software, error) {
	software := fleet.Software{}
	err := sqlx.GetContext(ctx, ds.reader, &software, `SELECT * FROM software WHERE id=?`, id)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "software by id")
	}

	query := `
		SELECT DISTINCT scv.cve
		FROM software s
		JOIN software_cpe scp ON (s.id=scp.software_id)
		JOIN software_cve scv ON (scp.id=scv.cpe_id)
		WHERE s.id=?
	`

	rows, err := ds.reader.QueryxContext(ctx, query, id)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load software cves")
	}
	defer rows.Close()

	for rows.Next() {
		var cve string
		if err := rows.Scan(&cve); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "scanning cve")
		}

		software.Vulnerabilities = append(software.Vulnerabilities, fleet.SoftwareCVE{
			CVE:         cve,
			DetailsLink: fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", cve),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "error iterating through cve rows")
	}

	return &software, nil
}

// CalculateHostsPerSoftware calculates the number of hosts having each
// software installed and stores that information in the software_host_counts
// table.
//
// After aggregation, it cleans up unused software (e.g. software installed
// on removed hosts, software uninstalled on hosts, etc.)
func (ds *Datastore) CalculateHostsPerSoftware(ctx context.Context, updatedAt time.Time) error {
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

	// remove any software count row for teams that don't exist anymore
	if _, err := ds.writer.ExecContext(ctx, cleanupTeamStmt); err != nil {
		return ctxerr.Wrap(ctx, err, "delete software_host_counts for non-existing teams")
	}
	return nil
}

// HostsByCPEs returns a list of all hosts that have the software corresponding
// to at least one of the CPEs installed. It returns a minimal represention of
// matching hosts.
func (ds *Datastore) HostsByCPEs(ctx context.Context, cpes []string) ([]*fleet.CPEHost, error) {
	queryStmt := `
    SELECT
      h.id,
      h.hostname
    FROM
      hosts h
    INNER JOIN
      host_software hs
    ON
      h.id = hs.host_id
    INNER JOIN
      software_cpe scp
    ON
      hs.software_id = scp.software_id
    WHERE
      scp.cpe IN (?)
    ORDER BY
      h.id`

	stmt, args, err := sqlx.In(queryStmt, cpes)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building query args")
	}
	var hosts []*fleet.CPEHost
	if err := sqlx.SelectContext(ctx, ds.reader, &hosts, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select hosts by cpes")
	}
	return hosts, nil
}
