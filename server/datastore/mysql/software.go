package mysql

import (
	"context"
	"fmt"
	"sort"
	"strings"

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
)

func truncateString(str string, length int) string {
	if len(str) > length {
		return str[:length]
	}
	return str
}

func softwareToUniqueString(s fleet.Software) string {
	return strings.Join([]string{s.Name, s.Version, s.Source, s.BundleIdentifier}, "\u0000")
}

func uniqueStringToSoftware(s string) fleet.Software {
	parts := strings.Split(s, "\u0000")
	return fleet.Software{
		Name:             truncateString(parts[0], maxSoftwareNameLen),
		Version:          truncateString(parts[1], maxSoftwareVersionLen),
		Source:           truncateString(parts[2], maxSoftwareSourceLen),
		BundleIdentifier: truncateString(parts[3], maxSoftwareBundleIdentifierLen),
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

func (d *Datastore) SaveHostSoftware(ctx context.Context, host *fleet.Host) error {
	if !host.HostSoftware.Modified {
		return nil
	}

	return d.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		return saveHostSoftwareDB(ctx, tx, host)
	})
}

func saveHostSoftwareDB(ctx context.Context, tx sqlx.ExtContext, host *fleet.Host) error {
	if err := applyChangesForNewSoftwareDB(ctx, tx, host); err != nil {
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

func applyChangesForNewSoftwareDB(ctx context.Context, tx sqlx.ExtContext, host *fleet.Host) error {
	storedCurrentSoftware, err := listSoftwareDB(ctx, tx, &host.ID, fleet.SoftwareListOptions{SkipLoadingCVEs: true})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "loading current software for host")
	}

	if nothingChanged(storedCurrentSoftware, host.Software) {
		return nil
	}

	current := softwareSliceToIdMap(storedCurrentSoftware)
	incoming := softwareSliceToSet(host.Software)

	if err = deleteUninstalledHostSoftwareDB(ctx, tx, host.ID, current, incoming); err != nil {
		return err
	}

	if err = insertNewInstalledHostSoftwareDB(ctx, tx, host.ID, current, incoming); err != nil {
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
			// TODO: delete from software if no host has it
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
		`SELECT id FROM software WHERE name = ? AND version = ? AND source = ? AND bundle_identifier = ?`,
		s.Name, s.Version, s.Source, s.BundleIdentifier,
	); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "get software")
	}
	if len(existingId) > 0 {
		return uint(existingId[0]), nil
	}

	result, err := tx.ExecContext(ctx,
		`INSERT INTO software (name, version, source, bundle_identifier) VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE bundle_identifier=VALUES(bundle_identifier)`,
		s.Name, s.Version, s.Source, s.BundleIdentifier,
	)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "insert software")
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "last id from software")
	}
	return uint(id), nil
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
		return nil, ctxerr.Wrap(ctx, err, "load host software")
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

	if match := opts.MatchQuery; match != "" {
		match = likePattern(match)
		ds = ds.Where(
			goqu.Or(
				goqu.I("s.name").ILike(match),
				goqu.I("s.version").ILike(match),
			),
		)
	}

	ds = ds.GroupBy(
		goqu.I("s.id"),
		goqu.I("s.name"),
		goqu.I("s.version"),
		goqu.I("s.source"),
		goqu.I("generated_cpe"),
	)

	ds = appendListOptionsToSelect(ds, opts.ListOptions)

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
	}

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

func (d *Datastore) LoadHostSoftware(ctx context.Context, host *fleet.Host) error {
	host.HostSoftware = fleet.HostSoftware{Modified: false}
	software, err := listSoftwareDB(ctx, d.reader, &host.ID, fleet.SoftwareListOptions{})
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

func (d *Datastore) AllSoftwareWithoutCPEIterator(ctx context.Context) (fleet.SoftwareIterator, error) {
	sql := `SELECT s.* FROM software s LEFT JOIN software_cpe sc on (s.id=sc.software_id) WHERE sc.id is null`
	// The rows.Close call is done by the caller once iteration using the
	// returned fleet.SoftwareIterator is done.
	rows, err := d.reader.QueryxContext(ctx, sql) //nolint:sqlclosecheck
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load host software")
	}
	return &softwareIterator{rows: rows}, nil
}

func (d *Datastore) AddCPEForSoftware(ctx context.Context, software fleet.Software, cpe string) error {
	_, err := addCPEForSoftwareDB(ctx, d.writer, software, cpe)
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

func (d *Datastore) AllCPEs(ctx context.Context) ([]string, error) {
	sql := `SELECT cpe FROM software_cpe`
	var cpes []string
	err := sqlx.SelectContext(ctx, d.reader, &cpes, sql)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "loads cpes")
	}
	return cpes, nil
}

func (d *Datastore) InsertCVEForCPE(ctx context.Context, cve string, cpes []string) error {
	values := strings.TrimSuffix(strings.Repeat("((SELECT id FROM software_cpe WHERE cpe=?),?),", len(cpes)), ",")
	sql := fmt.Sprintf(`INSERT IGNORE INTO software_cve (cpe_id, cve) VALUES %s`, values)
	var args []interface{}
	for _, cpe := range cpes {
		args = append(args, cpe, cve)
	}
	_, err := d.writer.ExecContext(ctx, sql, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "insert software cve")
	}
	return nil
}

func (d *Datastore) ListSoftware(ctx context.Context, opt fleet.SoftwareListOptions) ([]fleet.Software, error) {
	return listSoftwareDB(ctx, d.reader, nil, opt)
}

func (d *Datastore) CountSoftware(ctx context.Context, opt fleet.SoftwareListOptions) (int, error) {
	return countSoftwareDB(ctx, d.reader, nil, opt)
}

func (d *Datastore) SoftwareByID(ctx context.Context, id uint) (*fleet.Software, error) {
	software := fleet.Software{}
	err := sqlx.GetContext(ctx, d.reader, &software, `SELECT * FROM software WHERE id=?`, id)
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

	rows, err := d.reader.QueryxContext(ctx, query, id)
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
