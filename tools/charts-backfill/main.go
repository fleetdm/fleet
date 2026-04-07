// charts-backfill generates realistic bitmap data in the host_hourly_data table
// for development and testing. Safe to re-run — uses ON DUPLICATE KEY UPDATE to
// OR new bits with existing data.
//
// Usage:
//
//	go run ./tools/charts-backfill --dataset uptime --days 30
//	go run ./tools/charts-backfill --dataset uptime --days 7 --host-ids 1,2,3
//	go run ./tools/charts-backfill --mysql-dsn "fleet:fleet@tcp(localhost:3306)/fleet"
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const batchSize = 500

func main() {
	dataset := flag.String("dataset", "uptime", "dataset name (e.g. uptime, policy, cve)")
	days := flag.Int("days", 30, "number of days to backfill")
	startDate := flag.String("start-date", "", "start date (YYYY-MM-DD), defaults to now - days")
	entityIDsStr := flag.String("entity-ids", "", "comma-separated entity IDs (default: 0 for non-entity datasets)")
	hostIDsStr := flag.String("host-ids", "", "comma-separated host IDs (default: all from hosts table)")
	dsn := flag.String("mysql-dsn", "fleet:fleet@tcp(localhost:3306)/fleet?parseTime=true", "MySQL connection string")
	flag.Parse()

	db, err := sql.Open("mysql", *dsn)
	if err != nil {
		log.Fatalf("failed to connect to mysql: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping mysql: %v", err)
	}

	// Determine start date.
	var start time.Time
	if *startDate != "" {
		start, err = time.Parse("2006-01-02", *startDate)
		if err != nil {
			log.Fatalf("invalid start-date %q: %v", *startDate, err)
		}
	} else {
		start = time.Now().AddDate(0, 0, -(*days - 1))
	}

	// Determine host IDs.
	hostIDs := parseUintList(*hostIDsStr)
	if len(hostIDs) == 0 {
		hostIDs, err = queryHostIDs(db)
		if err != nil {
			log.Fatalf("failed to query host IDs: %v", err)
		}
		if len(hostIDs) == 0 {
			log.Fatal("no hosts found in database")
		}
	}

	// Determine entity IDs.
	entityIDs := parseUintList(*entityIDsStr)
	if len(entityIDs) == 0 {
		entityIDs = []uint{0}
	}

	log.Printf("backfilling dataset=%q, days=%d, start=%s, hosts=%d, entities=%d",
		*dataset, *days, start.Format("2006-01-02"), len(hostIDs), len(entityIDs))

	totalRows := 0
	startTime := time.Now()

	for day := range *days {
		date := start.AddDate(0, 0, day)
		dateStr := date.Format("2006-01-02")

		var batch []string
		var args []any

		for _, hostID := range hostIDs {
			for _, entityID := range entityIDs {
				bitmap := generateBitmap(*dataset)
				if bitmap == 0 {
					continue // sparse storage: skip all-zero rows
				}

				batch = append(batch, "(?, ?, ?, ?, ?)")
				args = append(args, hostID, *dataset, entityID, dateStr, bitmap)

				if len(batch) >= batchSize {
					if err := insertBatch(db, batch, args); err != nil {
						log.Fatalf("insert failed on day %s: %v", dateStr, err)
					}
					totalRows += len(batch)
					batch = batch[:0]
					args = args[:0]
				}
			}
		}

		// Flush remaining.
		if len(batch) > 0 {
			if err := insertBatch(db, batch, args); err != nil {
				log.Fatalf("insert failed on day %s: %v", dateStr, err)
			}
			totalRows += len(batch)
		}

		if (day+1)%5 == 0 || day == *days-1 {
			log.Printf("  day %d/%d (%s) — %d rows so far (%.1fs)",
				day+1, *days, dateStr, totalRows, time.Since(startTime).Seconds())
		}
	}

	log.Printf("done: %d rows inserted/updated in %.1fs", totalRows, time.Since(startTime).Seconds())
}

// generateBitmap returns a random 24-bit bitmap with a realistic distribution
// based on the dataset type.
func generateBitmap(dataset string) uint32 {
	switch dataset {
	case "uptime":
		// Uptime: 80-95% of hours have the bit set (host is online most of the time).
		return randomBitmapWithDensity(0.75, 0.95)
	case "policy":
		// Policy failure: only 5-20% of hosts fail, and failures are sparse across hours.
		// First decide if this host fails at all.
		if rand.Float64() > 0.15 {
			return 0 // host is compliant — no row needed
		}
		return randomBitmapWithDensity(0.3, 0.7)
	case "cve":
		// CVE vulnerability: ~10-30% of hosts are vulnerable, persistent across hours.
		if rand.Float64() > 0.25 {
			return 0
		}
		return randomBitmapWithDensity(0.5, 0.9)
	default:
		// Generic: moderate density.
		return randomBitmapWithDensity(0.4, 0.8)
	}
}

// randomBitmapWithDensity returns a 24-bit bitmap where each bit is set with a probability
// uniformly sampled between minDensity and maxDensity.
func randomBitmapWithDensity(minDensity, maxDensity float64) uint32 {
	density := minDensity + rand.Float64()*(maxDensity-minDensity)
	var bitmap uint32
	for hour := range 24 {
		if rand.Float64() < density {
			bitmap |= 1 << hour
		}
	}
	return bitmap
}

func insertBatch(db *sql.DB, valueClauses []string, args []any) error {
	query := fmt.Sprintf(
		`INSERT INTO host_hourly_data (host_id, dataset, entity_id, chart_date, hours_bitmap)
		 VALUES %s
		 ON DUPLICATE KEY UPDATE hours_bitmap = hours_bitmap | VALUES(hours_bitmap)`,
		strings.Join(valueClauses, ","),
	)
	_, err := db.Exec(query, args...)
	return err
}

func queryHostIDs(db *sql.DB) ([]uint, error) {
	rows, err := db.Query("SELECT id FROM hosts ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uint
	for rows.Next() {
		var id uint
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func parseUintList(s string) []uint {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []uint
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var v uint
		if _, err := fmt.Sscanf(p, "%d", &v); err == nil {
			result = append(result, v)
		}
	}
	return result
}
