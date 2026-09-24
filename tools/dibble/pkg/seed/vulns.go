package seed

import (
	"context"
	"crypto/md5" //nolint:gosec // matches fleet.Software.ComputeRawChecksum
	"database/sql"
	"embed"
	"encoding/csv"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
)

//go:embed data/software-macos.csv data/software-ubuntu.csv data/software-win.csv
var vulnCSVs embed.FS

// VulnsOptions configures the vuln seeder. Counts are per-platform; pass 0
// to skip a platform. DSN is a MySQL connection string.
type VulnsOptions struct {
	DSN      string
	MacOS    int
	Ubuntu   int
	Windows  int
	BatchSiz int
}

// Vulns writes plausible-looking software rows directly to MySQL so the
// background vulnerability scanner has inventory to chew on. Each row gets
// a fleet-compatible checksum and is inserted with INSERT IGNORE, so
// re-runs are idempotent against the unique software-checksum index.
//
// Scope (intentional): this seeder only writes the `software` table. It
// does NOT create hosts, `host_software` rows, or `software_cpe`
// associations — the legacy tools/software/vulnerabilities/seed_data tool
// did all of those, but dibble keeps the surface minimal and leaves
// host/CPE wiring to the real ingest path (or a future, opt-in flag).
// Vulnerabilities themselves are derived by Fleet's vuln scanner after
// CPE matching, so an empty `software_cpe` means no CVEs will surface
// from these rows on their own.
func Vulns(ctx context.Context, log Logger, opt VulnsOptions) Result {
	res := Result{Entity: "vulns"}
	if opt.BatchSiz <= 0 {
		opt.BatchSiz = 500
	}

	dsn, err := mysqlDSN(opt.DSN, true)
	if err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("parse DSN: %w", err))
		return res
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("open mysql: %w", err))
		return res
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("mysql ping: %w", err))
		return res
	}

	plans := []struct {
		platform string
		file     string
		count    int
	}{
		{"darwin", "data/software-macos.csv", opt.MacOS},
		{"ubuntu", "data/software-ubuntu.csv", opt.Ubuntu},
		{"windows", "data/software-win.csv", opt.Windows},
	}

	for _, p := range plans {
		if p.count <= 0 {
			continue
		}
		rows, err := readCSV(p.file)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Errorf("read %s: %w", p.file, err))
			continue
		}
		if err := insertSoftware(ctx, db, p.platform, rows, p.count, opt.BatchSiz); err != nil {
			res.Errors = append(res.Errors, fmt.Errorf("insert %s: %w", p.platform, err))
			continue
		}
		log.Printf("vulns: %d %s rows inserted from %s", p.count, p.platform, p.file)
		res.Created += p.count
	}
	return res
}

// mysqlDSN parses dsn with the MySQL driver, enables ParseTime and
// (optionally) MultiStatements, and returns the re-formatted DSN. Building
// the DSN this way instead of `dsn + "?parseTime=true"` preserves any
// query params the caller already set (e.g. tls=true, charset=utf8mb4).
func mysqlDSN(dsn string, multiStatements bool) (string, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return "", err
	}
	cfg.ParseTime = true
	if multiStatements {
		cfg.MultiStatements = true
	}
	return cfg.FormatDSN(), nil
}

// readCSV returns the data rows of an embedded CSV, with the header row
// stripped. Returns an error if the file has no data rows.
func readCSV(name string) ([][]string, error) {
	f, err := vulnCSVs.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	all, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(all) <= 1 {
		return nil, errors.New("csv has no data rows after header")
	}
	return all[1:], nil
}

// softwareChecksum mirrors fleet.Software.ComputeRawChecksum so that rows
// inserted here satisfy the unique idx_software_checksum index. The order
// of fields here MUST match server/fleet/software.go.
func softwareChecksum(name, version, source, bundleID, release, arch, vendor, extensionFor, extensionID string) []byte {
	h := md5.New() //nolint:gosec // DB lookup optimization, not security
	cols := []string{version, source, bundleID, release, arch, vendor, extensionFor, extensionID, name}
	_, _ = fmt.Fprint(h, strings.Join(cols, "\x00"))
	return h.Sum(nil)
}

// insertSoftware appends `count` rows to the software table. Each row gets
// a checksum that matches fleet.Software.ComputeRawChecksum so the unique
// idx_software_checksum index is satisfied. Idempotency here is
// intentionally weak — INSERT IGNORE will drop duplicate-checksum rows on
// re-runs, which matches the legacy tool's expectations.
//
// The platform argument is unused by the INSERT itself — source values in
// the CSVs (e.g. "apps", "deb_packages", "programs") already encode the
// platform. It's kept on the signature so the caller can log it.
func insertSoftware(ctx context.Context, db *sql.DB, _platform string, rows [][]string, count, batch int) error {
	if len(rows) == 0 {
		return errors.New("empty csv")
	}
	// SET FOREIGN_KEY_CHECKS=0 is a session variable. Pin everything below
	// to a single connection so the disable, the inserts, and the restore
	// all hit the same MySQL session — otherwise the pool can hand the
	// FK-disabled connection to an unrelated caller.
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire dedicated conn: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS=0"); err != nil {
		return err
	}
	defer func() {
		// Use a fresh context so the restore still runs even if ctx was
		// cancelled mid-insert.
		_, _ = conn.ExecContext(context.Background(), "SET FOREIGN_KEY_CHECKS=1")
	}()

	for i := 0; i < count; i += batch {
		end := i + batch
		if end > count {
			end = count
		}
		size := end - i
		placeholders := make([]string, 0, size)
		args := make([]any, 0, size*6)
		for k := 0; k < size; k++ {
			row := rows[(i+k)%len(rows)]
			if len(row) < 3 {
				continue
			}
			// CSV columns: name, version, source, bundle_identifier,
			// release, vendor_old, arch, vendor. Older rows may be
			// short; fall back to empty strings for missing fields.
			name, version, source := row[0], row[1], row[2]
			bundleID := csvField(row, 3)
			release := csvField(row, 4)
			arch := csvField(row, 6)
			vendor := csvField(row, 7)
			sum := softwareChecksum(name, version, source, bundleID, release, arch, vendor, "", "")
			placeholders = append(placeholders, "(?,?,?,?,?,?,?,?,?)")
			args = append(args,
				name, version, source, bundleID, release, arch, vendor, "", sum,
			)
		}
		if len(placeholders) == 0 {
			continue
		}
		stmt := "INSERT IGNORE INTO software " +
			"(name, version, source, bundle_identifier, `release`, arch, vendor, extension_for, checksum) " +
			"VALUES " + strings.Join(placeholders, ",")
		if _, err := conn.ExecContext(ctx, stmt, args...); err != nil {
			return err
		}
	}
	return nil
}

func csvField(row []string, i int) string {
	if i < len(row) {
		return row[i]
	}
	return ""
}
