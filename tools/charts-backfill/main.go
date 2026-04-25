// charts-backfill generates realistic chart data for development and testing.
// Writes rows to host_scd_data in closed form (explicit valid_to); the live
// collector can then extend from these rows via its normal write path.
// Safe to re-run — uses ON DUPLICATE KEY UPDATE to merge new data.
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
	"log"
	"math/rand/v2"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/str"
	"github.com/fleetdm/fleet/v4/server/chart"
	_ "github.com/go-sql-driver/mysql"
)

// dailyDatasets bucket at 24h granularity; all others are hourly.
var dailyDatasets = map[string]struct{}{
	"cve": {},
}

func main() {
	dataset := flag.String("dataset", "uptime", "dataset name (e.g. uptime, policy, cve)")
	days := flag.Int("days", 30, "number of days to backfill")
	startDate := flag.String("start-date", "", "start date (YYYY-MM-DD), defaults to now - days")
	entityIDsStr := flag.String("entity-ids", "", "comma-separated entity IDs (default: '' for non-entity datasets)")
	hostIDsStr := flag.String("host-ids", "", "comma-separated host IDs (default: all from hosts table)")
	dsn := flag.String("mysql-dsn", "fleet:fleet@tcp(localhost:3306)/fleet?parseTime=true", "MySQL connection string")
	flag.Parse()

	var start time.Time
	if *startDate != "" {
		s, err := time.Parse("2006-01-02", *startDate)
		if err != nil {
			log.Fatalf("invalid start-date %q: %v", *startDate, err)
		}
		start = s
	} else {
		start = time.Now().UTC().AddDate(0, 0, -(*days - 1))
	}
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)

	db, err := sql.Open("mysql", *dsn)
	if err != nil {
		log.Fatalf("failed to connect to mysql: %v", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		log.Fatalf("failed to ping mysql: %v", err)
	}
	defer db.Close()

	hostIDs := str.ParseUintList(*hostIDsStr)
	if len(hostIDs) == 0 {
		hostIDs, err = queryHostIDs(db)
		if err != nil {
			log.Fatalf("failed to query host IDs: %v", err) //nolint:gocritic // dev tool, OS reclaims db handle on exit
		}
		if len(hostIDs) == 0 {
			log.Fatal("no hosts found in database")
		}
	}

	entityIDs := str.ParseStringList(*entityIDsStr)
	if len(entityIDs) == 0 {
		entityIDs = []string{""}
	}

	log.Printf("backfilling dataset=%q, days=%d, start=%s, hosts=%d, entities=%d",
		*dataset, *days, start.Format("2006-01-02"), len(hostIDs), len(entityIDs))

	startTime := time.Now()
	totalRows := backfill(db, *dataset, *days, start, hostIDs, entityIDs)
	log.Printf("done: %d SCD rows inserted/updated in %.1fs", totalRows, time.Since(startTime).Seconds())
}

func backfill(db *sql.DB, dataset string, days int, start time.Time, hostIDs []uint, entityIDs []string) int {
	if _, ok := dailyDatasets[dataset]; ok {
		return backfillDaily(db, dataset, days, start, hostIDs, entityIDs)
	}
	return backfillHourly(db, dataset, days, start, hostIDs, entityIDs)
}

func backfillHourly(db *sql.DB, dataset string, days int, start time.Time, hostIDs []uint, entityIDs []string) int {
	totalRows := 0
	for day := range days {
		date := start.AddDate(0, 0, day)

		for _, entityID := range entityIDs {
			hourlyHosts := generateHourlyHosts(dataset, hostIDs)

			for hour, activeHosts := range hourlyHosts {
				if len(activeHosts) == 0 {
					continue
				}
				validFrom := date.Add(time.Duration(hour) * time.Hour)
				validTo := validFrom.Add(time.Hour)
				blob := chart.HostIDsToBlob(activeHosts)

				_, err := db.Exec(
					`INSERT INTO host_scd_data (dataset, entity_id, host_bitmap, valid_from, valid_to)
					 VALUES (?, ?, ?, ?, ?)
					 ON DUPLICATE KEY UPDATE host_bitmap = VALUES(host_bitmap), valid_to = VALUES(valid_to)`,
					dataset, entityID, blob, validFrom, validTo)
				if err != nil {
					log.Fatalf("insert hourly SCD row failed on %s hour %d: %v", validFrom, hour, err)
				}
				totalRows++
			}
		}

		if (day+1)%5 == 0 || day == days-1 {
			log.Printf("  day %d/%d (%s) — %d rows so far",
				day+1, days, date.Format("2006-01-02"), totalRows)
		}
	}
	return totalRows
}

func backfillDaily(db *sql.DB, dataset string, days int, start time.Time, hostIDs []uint, entityIDs []string) int {
	totalRows := 0
	minDensity, maxDensity := densityRange(dataset)
	n := len(hostIDs)

	for day := range days {
		date := start.AddDate(0, 0, day)

		for _, entityID := range entityIDs {
			density := minDensity + rand.Float64()*(maxDensity-minDensity) //nolint:gosec // dev data generator, not crypto
			count := int(float64(n) * density)
			if count == 0 {
				continue
			}
			active := make([]uint, count)
			for i, idx := range rand.Perm(n)[:count] {
				active[i] = hostIDs[idx]
			}
			blob := chart.HostIDsToBlob(active)
			validFrom := date
			validTo := date.AddDate(0, 0, 1)

			_, err := db.Exec(
				`INSERT INTO host_scd_data (dataset, entity_id, host_bitmap, valid_from, valid_to)
				 VALUES (?, ?, ?, ?, ?)
				 ON DUPLICATE KEY UPDATE host_bitmap = VALUES(host_bitmap), valid_to = VALUES(valid_to)`,
				dataset, entityID, blob, validFrom, validTo)
			if err != nil {
				log.Fatalf("insert daily SCD row failed on %s entity %q: %v", date, entityID, err)
			}
			totalRows++
		}

		if (day+1)%5 == 0 || day == days-1 {
			log.Printf("  day %d/%d (%s) — %d rows so far",
				day+1, days, date.Format("2006-01-02"), totalRows)
		}
	}

	return totalRows
}

// generateHourlyHosts returns a map of hour -> active host IDs for a single day.
func generateHourlyHosts(dataset string, hostIDs []uint) map[int][]uint {
	minDensity, maxDensity := densityRange(dataset)
	n := len(hostIDs)
	result := make(map[int][]uint, 24)

	for hour := range 24 {
		density := minDensity + rand.Float64()*(maxDensity-minDensity) //nolint:gosec // dev data generator, not crypto
		count := int(float64(n) * density)
		if count == 0 {
			continue
		}
		active := make([]uint, count)
		for i, idx := range rand.Perm(n)[:count] {
			active[i] = hostIDs[idx]
		}
		result[hour] = active
	}

	return result
}

func densityRange(dataset string) (float64, float64) {
	switch dataset {
	case "uptime":
		return 0.0, 1.0
	case "policy":
		return 0.05, 0.20
	case "cve":
		return 0.10, 0.30
	default:
		return 0.40, 0.80
	}
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
