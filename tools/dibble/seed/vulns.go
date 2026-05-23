package seed

import (
	"context"
	"database/sql"
	"embed"
	"encoding/csv"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
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
	Kernels  int
	BatchSiz int
}

// Vulns writes vulnerable software rows directly to MySQL. The Fleet API
// doesn't expose a "create vulnerability" endpoint — vulnerabilities are
// derived from the software inventory by background scanners. For testing
// purposes we shortcut that pipeline by injecting host + software_cpe rows.
//
// This is a direct port of tools/software/vulnerabilities/seed_data/seed_vuln_data.go
// adapted to read CSVs from the embedded filesystem.
func Vulns(ctx context.Context, log Logger, opt VulnsOptions) Result {
	res := Result{Entity: "vulns"}
	if opt.BatchSiz <= 0 {
		opt.BatchSiz = 500
	}

	db, err := sql.Open("mysql", opt.DSN+"?multiStatements=true&parseTime=true")
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

func readCSV(name string) ([][]string, error) {
	f, err := vulnCSVs.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return csv.NewReader(f).ReadAll()
}

// insertSoftware appends `count` rows to the software table. Idempotency
// here is intentionally weak — the legacy tool was the same way, on the
// assumption that this is only run against a wiped or scratch DB.
func insertSoftware(ctx context.Context, db *sql.DB, platform string, rows [][]string, count, batch int) error {
	if len(rows) == 0 {
		return fmt.Errorf("empty csv")
	}
	if _, err := db.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS=0"); err != nil {
		return err
	}
	defer func() {
		_, _ = db.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS=1")
	}()

	for i := 0; i < count; i += batch {
		end := i + batch
		if end > count {
			end = count
		}
		size := end - i
		placeholders := make([]string, 0, size)
		args := make([]any, 0, size*5)
		for k := 0; k < size; k++ {
			row := rows[(i+k)%len(rows)]
			if len(row) < 3 {
				continue
			}
			name, version, source := row[0], row[1], row[2]
			placeholders = append(placeholders, "(?,?,?,?,?)")
			args = append(args, name, version, source, platform, "")
		}
		if len(placeholders) == 0 {
			continue
		}
		stmt := "INSERT IGNORE INTO software (name, version, source, browser, bundle_identifier) VALUES " + strings.Join(placeholders, ",")
		if _, err := db.ExecContext(ctx, stmt, args...); err != nil {
			return err
		}
	}
	return nil
}
