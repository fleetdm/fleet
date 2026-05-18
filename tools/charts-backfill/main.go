// charts-backfill generates realistic chart data for development and testing.
// Writes rows to host_scd_data. Safe to re-run — uses ON DUPLICATE KEY UPDATE
// to merge new data.
//
// Datasets are backfilled in one of two modes based on their sample strategy:
//
//   - Accumulate (e.g. uptime): independent rows per hour, each a fresh
//     random sample. Each row's validity is bounded to its single hour.
//   - Snapshot (e.g. cve): per-entity state-segment rows. Most entities get
//     a single open row spanning the entire backfill range; a small fraction
//     "flip" state on day boundaries, producing additional closed segments.
//     The final segment per entity has valid_to = sentinel so the live
//     collector can compare against it on its next tick. This mirrors what
//     real CVE data looks like (mostly stable, occasional churn).
//
// Usage:
//
//	go run ./tools/charts-backfill --dataset uptime --days 30
//	go run ./tools/charts-backfill --dataset uptime --days 7 --host-ids 1,2,3
//	go run ./tools/charts-backfill --dataset cve --days 30 --use-tracked-cves
//	go run ./tools/charts-backfill --mysql-dsn "fleet:fleet@tcp(localhost:3306)/fleet"
package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/str"
	"github.com/fleetdm/fleet/v4/server/chart"
	"github.com/fleetdm/fleet/v4/server/chart/bootstrap"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// snapshotDatasets are generated with the state-segment model: mostly stable
// rows with occasional churn. Everything not listed here uses the accumulate
// hourly model. Must match the live collector's sample strategy for each
// dataset (see server/chart/datasets.go) so backfilled data is shaped like
// what production will eventually produce.
var snapshotDatasets = map[string]struct{}{
	"cve": {},
}

// scdOpenSentinel mirrors the constant in server/chart/internal/mysql/data.go.
// Used as valid_to to mark rows as currently open; the live collector closes
// these on the next state change or extends them by leaving them alone.
var scdOpenSentinel = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)

// snapshotFlipsPerDayPerEntity is the per-entity probability of a state
// change on any given day. ~5% means a 30-day window produces on average
// ~1.5 state changes per entity — most entities stay stable, a few have a
// handful of segments.
const snapshotFlipsPerDayPerEntity = 0.05

// snapshotChurnFraction is the fraction of an entity's current host set that
// turns over on a flip — some hosts drop out (patched), some new hosts get
// added (newly discovered as affected). Cardinality stays roughly stable.
const snapshotChurnFraction = 0.10

// snapshotCardinality picks a per-entity affected-host count from a long-tail
// distribution shaped like real-world CVE data: most CVEs touch a handful of
// hosts (specific software/version), with an occasional wide one (browser or
// kernel). A naive uniform-density model saturates at fleet size when many
// CVEs are unioned together — this distribution keeps the union meaningful
// even with hundreds of tracked entities. Return value is capped at fleetSize.
func snapshotCardinality(fleetSize int) int {
	r := rand.Float64() //nolint:gosec // dev data generator, not crypto
	var count int
	switch {
	case r < 0.70: // very narrow: specific software build
		count = 1 + rand.IntN(5) //nolint:gosec
	case r < 0.92: // narrow: software version
		count = 5 + rand.IntN(20) //nolint:gosec
	case r < 0.99: // moderate: popular software
		count = 25 + rand.IntN(100) //nolint:gosec
	default: // wide: browser/kernel-tier, up to ~10% of fleet
		wideMax := fleetSize / 10
		if wideMax < 200 {
			wideMax = 200
		}
		count = 125 + rand.IntN(wideMax) //nolint:gosec
	}
	if count > fleetSize {
		count = fleetSize
	}
	return count
}

func main() {
	dataset := flag.String("dataset", "uptime", "dataset name (e.g. uptime, policy, cve)")
	days := flag.Int("days", 30, "number of days to backfill")
	startDate := flag.String("start-date", "", "start date (YYYY-MM-DD), defaults to now - days")
	entityIDsStr := flag.String("entity-ids", "", "comma-separated entity IDs (default: '' for non-entity datasets)")
	useTrackedCVEs := flag.Bool("use-tracked-cves", false, "for --dataset cve, auto-discover entity IDs from the production tracked-CVE query (overrides --entity-ids)")
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

	rawDB, err := sql.Open("mysql", *dsn)
	if err != nil {
		log.Fatalf("failed to connect to mysql: %v", err)
	}
	if err := rawDB.Ping(); err != nil {
		rawDB.Close()
		log.Fatalf("failed to ping mysql: %v", err)
	}
	defer rawDB.Close()

	// sqlx wraps the raw connection so we can hand it to the chart bootstrap
	// helpers (TrackedCriticalCVEs) without opening a second pool.
	db := sqlx.NewDb(rawDB, "mysql")

	hostIDs := str.ParseUintList(*hostIDsStr)
	if len(hostIDs) == 0 {
		hostIDs, err = queryHostIDs(rawDB)
		if err != nil {
			log.Fatalf("failed to query host IDs: %v", err) //nolint:gocritic // dev tool, OS reclaims db handle on exit
		}
		if len(hostIDs) == 0 {
			log.Fatal("no hosts found in database")
		}
	}

	var entityIDs []string
	switch {
	case *useTrackedCVEs:
		if *dataset != "cve" {
			log.Fatalf("--use-tracked-cves only applies to --dataset cve (got %q)", *dataset)
		}
		ctx := context.Background()
		cves, err := bootstrap.TrackedCriticalCVEs(ctx, db, slog.New(slog.DiscardHandler))
		if err != nil {
			log.Fatalf("failed to query tracked CVEs: %v", err)
		}
		if len(cves) == 0 {
			log.Fatal("tracked-CVE query returned no CVEs (vulnerability data may not be populated yet)")
		}
		entityIDs = cves
		log.Printf("discovered %d tracked CVEs from the live database", len(entityIDs))
	case *entityIDsStr != "":
		entityIDs = str.ParseStringList(*entityIDsStr)
	default:
		entityIDs = []string{""}
	}

	log.Printf("backfilling dataset=%q, days=%d, start=%s, hosts=%d, entities=%d",
		*dataset, *days, start.Format("2006-01-02"), len(hostIDs), len(entityIDs))

	startTime := time.Now()
	totalRows := backfill(rawDB, *dataset, *days, start, hostIDs, entityIDs)
	log.Printf("done: %d SCD rows inserted/updated in %.1fs", totalRows, time.Since(startTime).Seconds())
}

func backfill(db *sql.DB, dataset string, days int, start time.Time, hostIDs []uint, entityIDs []string) int {
	if _, ok := snapshotDatasets[dataset]; ok {
		return backfillSnapshot(db, dataset, days, start, hostIDs, entityIDs)
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
					`INSERT INTO host_scd_data (dataset, entity_id, host_bitmap, encoding_type, valid_from, valid_to)
					 VALUES (?, ?, ?, ?, ?, ?)
					 ON DUPLICATE KEY UPDATE host_bitmap = VALUES(host_bitmap), encoding_type = VALUES(encoding_type), valid_to = VALUES(valid_to)`,
					dataset, entityID, blob.Bytes, blob.Encoding, validFrom, validTo)
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

// backfillSnapshot models per-entity state-segment data shaped like what the
// production snapshot collector produces over time. For each entity:
//   - Pick an initial host set with density in the dataset's typical range.
//   - Walk day by day; on each day, with probability snapshotFlipsPerDayPerEntity,
//     churn the set (drop ~churn% / add ~churn% of new hosts).
//   - Each contiguous run of unchanged days is written as a single row.
//   - The final segment per entity leaves valid_to at the sentinel (open), so
//     the live collector compares against it on its next tick rather than
//     opening a fresh row over the top.
func backfillSnapshot(db *sql.DB, dataset string, days int, start time.Time, hostIDs []uint, entityIDs []string) int {
	totalRows := 0

	type segment struct {
		validFrom time.Time
		active    []uint
	}

	for entityIdx, entityID := range entityIDs {
		active := randomSubset(hostIDs, snapshotCardinality(len(hostIDs)))

		segments := []segment{{validFrom: start, active: active}}
		for day := 1; day < days; day++ {
			if rand.Float64() >= snapshotFlipsPerDayPerEntity { //nolint:gosec // dev data generator, not crypto
				continue
			}
			active = churn(active, hostIDs, snapshotChurnFraction)
			segments = append(segments, segment{
				validFrom: start.AddDate(0, 0, day),
				active:    active,
			})
		}

		for i, seg := range segments {
			validTo := scdOpenSentinel
			if i+1 < len(segments) {
				validTo = segments[i+1].validFrom
			}
			blob := chart.HostIDsToBlob(seg.active)

			_, err := db.Exec(
				`INSERT INTO host_scd_data (dataset, entity_id, host_bitmap, encoding_type, valid_from, valid_to)
				 VALUES (?, ?, ?, ?, ?, ?)
				 ON DUPLICATE KEY UPDATE host_bitmap = VALUES(host_bitmap), encoding_type = VALUES(encoding_type), valid_to = VALUES(valid_to)`,
				dataset, entityID, blob.Bytes, blob.Encoding, seg.validFrom, validTo)
			if err != nil {
				log.Fatalf("insert snapshot SCD row failed for entity %q at %s: %v", entityID, seg.validFrom, err)
			}
			totalRows++
		}

		if (entityIdx+1)%500 == 0 || entityIdx == len(entityIDs)-1 {
			log.Printf("  entity %d/%d — %d rows so far",
				entityIdx+1, len(entityIDs), totalRows)
		}
	}

	return totalRows
}

// randomSubset returns a uniformly random `count`-sized subset of pool. If
// count >= len(pool), returns a shuffled clone of the entire pool. The result
// is a fresh slice that the caller can mutate.
func randomSubset(pool []uint, count int) []uint {
	if count <= 0 {
		return nil
	}
	if count >= len(pool) {
		out := make([]uint, len(pool))
		copy(out, pool)
		return out
	}
	out := make([]uint, count)
	for i, idx := range rand.Perm(len(pool))[:count] { //nolint:gosec // dev data generator, not crypto
		out[i] = pool[idx]
	}
	return out
}

// churn produces a new host set from `prev` by dropping a `fraction` of its
// members and adding a `fraction` of currently-unaffected hosts from the pool.
// Cardinality stays roughly stable; identity shifts. Models the realistic CVE
// state-change pattern (some hosts patched, some new hosts discovered as
// affected) without inventing wholly new bitmaps.
func churn(prev, pool []uint, fraction float64) []uint {
	prevSet := make(map[uint]struct{}, len(prev))
	for _, id := range prev {
		prevSet[id] = struct{}{}
	}

	dropCount := int(float64(len(prev)) * fraction)
	if dropCount < 1 && len(prev) > 0 {
		dropCount = 1
	}
	addCount := dropCount

	// Drop: keep prev members not in the random dropout sample. If dropCount
	// >= len(prev) we drop everything, leaving kept empty for the add step
	// below to fill.
	kept := make([]uint, 0, len(prev))
	if dropCount < len(prev) {
		dropIdx := make(map[int]struct{}, dropCount)
		for _, idx := range rand.Perm(len(prev))[:dropCount] { //nolint:gosec // dev data generator, not crypto
			dropIdx[idx] = struct{}{}
		}
		for i, id := range prev {
			if _, drop := dropIdx[i]; drop {
				continue
			}
			kept = append(kept, id)
		}
	}

	// Add: walk a shuffled pool, picking hosts that aren't already in prev.
	added := 0
	for _, idx := range rand.Perm(len(pool)) { //nolint:gosec // dev data generator, not crypto
		if added >= addCount {
			break
		}
		candidate := pool[idx]
		if _, exists := prevSet[candidate]; exists {
			continue
		}
		kept = append(kept, candidate)
		added++
	}
	return kept
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
