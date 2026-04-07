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

		for _, entityID := range entityIDs {
			// Pre-compute bitmaps for all hosts for this day+entity.
			// The density determines what fraction of hosts are active,
			// not what fraction of hours each host has set.
			bitmaps := generateDayBitmaps(*dataset, hostIDs)

			for i, hostID := range hostIDs {
				if bitmaps[i] == 0 {
					continue // sparse storage: skip all-zero rows
				}

				batch = append(batch, "(?, ?, ?, ?, ?)")
				args = append(args, hostID, *dataset, entityID, dateStr, bitmaps[i])

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

// generateDayBitmaps generates bitmaps for all hosts for a single day. The density
// controls what fraction of hosts are active for each hour, so e.g. 80% uptime means
// ~80% of hosts have the bit set for any given hour.
func generateDayBitmaps(dataset string, hostIDs []uint) []uint32 {
	bitmaps := make([]uint32, len(hostIDs))

	// Per-hour density: what fraction of hosts should have bit set this hour.
	var minDensity, maxDensity float64
	switch dataset {
	case "uptime":
		minDensity, maxDensity = 0.0, 1.0
	case "policy":
		minDensity, maxDensity = 0.05, 0.20
	case "cve":
		minDensity, maxDensity = 0.10, 0.30
	default:
		minDensity, maxDensity = 0.40, 0.80
	}

	// For each hour, pick a density in the range and select exactly that
	// many hosts to have the bit set.
	n := len(hostIDs)
	for hour := range 24 {
		density := minDensity + rand.Float64()*(maxDensity-minDensity)
		fmt.Printf("  hour %02d: density=%.2f%%\n", hour, density*100)
		count := int(float64(n) * density)
		for _, idx := range rand.Perm(n)[:count] {
			bitmaps[idx] |= 1 << hour
		}
	}

	return bitmaps
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
