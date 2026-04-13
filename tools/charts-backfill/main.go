// charts-backfill generates realistic chart data for development and testing.
// For blob-storage datasets (uptime), it writes to host_hourly_data_blobs.
// For bitmap-storage datasets, it writes to host_daily_data_bitmaps.
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
	"fmt"
	"log"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart"
	_ "github.com/go-sql-driver/mysql"
)

const batchSize = 500

// blobDatasets lists datasets that use blob storage.
var blobDatasets = map[string]struct{}{
	"uptime": {},
	"cve":    {},
}

// dailyBlobDatasets use blob storage with hour=-1 (whole-day granularity).
var dailyBlobDatasets = map[string]struct{}{
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
	entityIDs := parseStringList(*entityIDsStr)
	if len(entityIDs) == 0 {
		entityIDs = []string{""}
	}

	_, isBlob := blobDatasets[*dataset]

	log.Printf("backfilling dataset=%q (blob=%v), days=%d, start=%s, hosts=%d, entities=%d",
		*dataset, isBlob, *days, start.Format("2006-01-02"), len(hostIDs), len(entityIDs))

	startTime := time.Now()

	if isBlob {
		totalRows := backfillBlob(db, *dataset, *days, start, hostIDs, entityIDs)
		log.Printf("done: %d blob rows inserted/updated in %.1fs", totalRows, time.Since(startTime).Seconds())
	} else {
		totalRows := backfillBitmap(db, *dataset, *days, start, hostIDs, entityIDs)
		log.Printf("done: %d bitmap rows inserted/updated in %.1fs", totalRows, time.Since(startTime).Seconds())
	}
}

func backfillBlob(db *sql.DB, dataset string, days int, start time.Time, hostIDs []uint, entityIDs []string) int {
	_, isDaily := dailyBlobDatasets[dataset]
	if isDaily {
		return backfillDailyBlob(db, dataset, days, start, hostIDs, entityIDs)
	}
	return backfillHourlyBlob(db, dataset, days, start, hostIDs, entityIDs)
}

func backfillHourlyBlob(db *sql.DB, dataset string, days int, start time.Time, hostIDs []uint, entityIDs []string) int {
	totalRows := 0

	for day := range days {
		date := start.AddDate(0, 0, day)
		dateStr := date.Format("2006-01-02")

		for _, entityID := range entityIDs {
			// Generate per-hour host activity.
			hourlyHosts := generateHourlyHosts(dataset, hostIDs)

			for hour, activeHosts := range hourlyHosts {
				if len(activeHosts) == 0 {
					continue
				}
				blob := chart.HostIDsToBlob(activeHosts)

				_, err := db.Exec(
					`INSERT INTO host_hourly_data_blobs (dataset, entity_id, chart_date, hour, host_bitmap)
					 VALUES (?, ?, ?, ?, ?)
					 ON DUPLICATE KEY UPDATE host_bitmap = VALUES(host_bitmap)`,
					dataset, entityID, dateStr, hour, blob)
				if err != nil {
					log.Fatalf("insert blob failed on day %s hour %d: %v", dateStr, hour, err)
				}
				totalRows++
			}
		}

		if (day+1)%5 == 0 || day == days-1 {
			log.Printf("  day %d/%d (%s) — %d rows so far",
				day+1, days, dateStr, totalRows)
		}
	}

	return totalRows
}

func backfillDailyBlob(db *sql.DB, dataset string, days int, start time.Time, hostIDs []uint, entityIDs []string) int {
	totalRows := 0
	minDensity, maxDensity := densityRange(dataset)
	n := len(hostIDs)

	for day := range days {
		date := start.AddDate(0, 0, day)
		dateStr := date.Format("2006-01-02")

		for _, entityID := range entityIDs {
			density := minDensity + rand.Float64()*(maxDensity-minDensity)
			count := int(float64(n) * density)
			if count == 0 {
				continue
			}
			active := make([]uint, count)
			for i, idx := range rand.Perm(n)[:count] {
				active[i] = hostIDs[idx]
			}
			blob := chart.HostIDsToBlob(active)

			_, err := db.Exec(
				`INSERT INTO host_hourly_data_blobs (dataset, entity_id, chart_date, hour, host_bitmap)
				 VALUES (?, ?, ?, ?, ?)
				 ON DUPLICATE KEY UPDATE host_bitmap = VALUES(host_bitmap)`,
				dataset, entityID, dateStr, chart.HourWholeDay, blob)
			if err != nil {
				log.Fatalf("insert daily blob failed on day %s entity %q: %v", dateStr, entityID, err)
			}
			totalRows++
		}

		if (day+1)%5 == 0 || day == days-1 {
			log.Printf("  day %d/%d (%s) — %d rows so far",
				day+1, days, dateStr, totalRows)
		}
	}

	return totalRows
}

func backfillBitmap(db *sql.DB, dataset string, days int, start time.Time, hostIDs []uint, entityIDs []string) int {
	totalRows := 0

	for day := range days {
		date := start.AddDate(0, 0, day)
		dateStr := date.Format("2006-01-02")

		var batch []string
		var args []any

		for _, entityID := range entityIDs {
			bitmaps := generateDayBitmaps(dataset, hostIDs)

			for i, hostID := range hostIDs {
				if bitmaps[i] == 0 {
					continue
				}

				batch = append(batch, "(?, ?, ?, ?, ?)")
				args = append(args, hostID, dataset, entityID, dateStr, bitmaps[i])

				if len(batch) >= batchSize {
					if err := insertBitmapBatch(db, batch, args); err != nil {
						log.Fatalf("insert failed on day %s: %v", dateStr, err)
					}
					totalRows += len(batch)
					batch = batch[:0]
					args = args[:0]
				}
			}
		}

		if len(batch) > 0 {
			if err := insertBitmapBatch(db, batch, args); err != nil {
				log.Fatalf("insert failed on day %s: %v", dateStr, err)
			}
			totalRows += len(batch)
		}

		if (day+1)%5 == 0 || day == days-1 {
			log.Printf("  day %d/%d (%s) — %d rows so far",
				day+1, days, dateStr, totalRows)
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
		density := minDensity + rand.Float64()*(maxDensity-minDensity)
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

// generateDayBitmaps generates per-host bitmaps for a single day.
func generateDayBitmaps(dataset string, hostIDs []uint) []uint32 {
	bitmaps := make([]uint32, len(hostIDs))
	minDensity, maxDensity := densityRange(dataset)
	n := len(hostIDs)

	for hour := range 24 {
		density := minDensity + rand.Float64()*(maxDensity-minDensity)
		count := int(float64(n) * density)
		for _, idx := range rand.Perm(n)[:count] {
			bitmaps[idx] |= 1 << hour
		}
	}

	return bitmaps
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

func insertBitmapBatch(db *sql.DB, valueClauses []string, args []any) error {
	query := fmt.Sprintf(
		`INSERT INTO host_daily_data_bitmaps (host_id, dataset, entity_id, chart_date, hours_bitmap)
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

func parseStringList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		result = append(result, p)
	}
	return result
}
