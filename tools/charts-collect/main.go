// charts-collect fetches live data from a Fleet instance via API and writes
// chart data into a local database. Designed to run hourly via cron.
//
// Uptime: fetches currently online hosts and OR-merges them into the
// current-hour accumulate row (dataset='uptime'). Rows are closed at hour
// boundaries; no cross-bucket collapse.
// CVE: fetches per-host vulnerability data, builds per-CVE host bitmaps, and
// reconciles them into host_scd_data (dataset='cve') as snapshot rows.
// Unchanged CVEs keep their existing open row; changed bitmaps close the prior
// row at the current hour boundary (UTC) and open a new one; same-hour
// re-runs overwrite the open row via ODKU.
//
// Usage:
//
//	go run ./tools/charts-collect --fleet-url https://dogfood.fleetdm.com --fleet-token <token>
//	go run ./tools/charts-collect --fleet-url https://dogfood.fleetdm.com --fleet-token <token> --mysql-dsn "fleet:fleet@tcp(localhost:3306)/fleet"
//
// Env vars:
//   - FLEET_URL / FLEET_TOKEN: API target (same as --fleet-url / --fleet-token).
//   - MYSQL_DSN: full DSN (same as --mysql-dsn).
//   - FLEET_MYSQL_ADDRESS, FLEET_MYSQL_DATABASE, FLEET_MYSQL_USERNAME,
//     FLEET_MYSQL_PASSWORD: used to assemble a DSN when MYSQL_DSN/--mysql-dsn
//     is not set. Matches the fleet server's env var names so the same values
//     can be reused (e.g. via Render fromService).
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/RoaringBitmap/roaring"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/chart"
	_ "github.com/go-sql-driver/mysql"
)

const (
	perPage = 500
	// scdUpsertBatch mirrors the constant in server/chart/internal/mysql/data.go —
	// the collector writes to the same table out-of-process and must keep the
	// encoding in sync.
	scdUpsertBatch = 200
)

// scdOpenSentinel is the end-of-time marker used for valid_to on open snapshot
// rows. Must match the DEFAULT in the host_scd_data table.
var scdOpenSentinel = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)

func main() {
	fleetURL := flag.String("fleet-url", os.Getenv("FLEET_URL"), "Fleet server URL (or FLEET_URL env var)")
	fleetToken := flag.String("fleet-token", os.Getenv("FLEET_TOKEN"), "Fleet API token (or FLEET_TOKEN env var)")
	dsn := flag.String("mysql-dsn", os.Getenv("MYSQL_DSN"), "MySQL connection string (or MYSQL_DSN env var; falls back to FLEET_MYSQL_* env vars)")
	flag.Parse()

	if *fleetURL == "" || *fleetToken == "" {
		log.Fatal("--fleet-url and --fleet-token are required (or set FLEET_URL and FLEET_TOKEN)")
	}

	if *dsn == "" {
		built, err := dsnFromEnv()
		if err != nil {
			log.Fatalf("build mysql dsn: %v", err)
		}
		*dsn = built
	}

	db, err := sql.Open("mysql", *dsn)
	if err != nil {
		log.Fatalf("connect to mysql: %v", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		log.Fatalf("ping mysql: %v", err)
	}
	defer db.Close()

	api := &apiClient{
		baseURL: *fleetURL,
		token:   *fleetToken,
		http:    fleethttp.NewClient(fleethttp.WithTimeout(30 * time.Second)),
	}

	if err := collectUptime(api, db); err != nil {
		log.Printf("ERROR uptime collection: %v", err)
	}
	if err := collectCVE(api, db); err != nil {
		log.Printf("ERROR cve collection: %v", err)
	}
}

// dsnFromEnv builds a MySQL DSN from the standard FLEET_MYSQL_* env vars used
// by the fleet server. Returns an error if any required piece is missing so we
// don't silently fall back to localhost defaults in a production cron.
func dsnFromEnv() (string, error) {
	addr := os.Getenv("FLEET_MYSQL_ADDRESS")
	user := os.Getenv("FLEET_MYSQL_USERNAME")
	pass := os.Getenv("FLEET_MYSQL_PASSWORD")
	db := os.Getenv("FLEET_MYSQL_DATABASE")

	var missing []string
	if addr == "" {
		missing = append(missing, "FLEET_MYSQL_ADDRESS")
	}
	if user == "" {
		missing = append(missing, "FLEET_MYSQL_USERNAME")
	}
	if db == "" {
		missing = append(missing, "FLEET_MYSQL_DATABASE")
	}
	if len(missing) > 0 {
		return "", fmt.Errorf("missing env vars: %s (or set --mysql-dsn / MYSQL_DSN)", strings.Join(missing, ", "))
	}

	// Raw password — matches how the fleet server formats its DSN
	// (server/platform/mysql/common.go). go-sql-driver does not URL-decode
	// the password field, so encoding it here would corrupt the value.
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", user, pass, addr, db), nil
}

// apiClient wraps HTTP calls to the Fleet API.
type apiClient struct {
	baseURL string
	token   string
	http    *http.Client
}

func (a *apiClient) get(path string) (*http.Response, error) {
	url := a.baseURL + path
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.token))

	resp, err := a.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", path, err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("GET %s: status %d", path, resp.StatusCode)
	}
	return resp, nil
}

// --- API response types (minimal) ---

type hostsResponse struct {
	Hosts []struct {
		ID uint `json:"id"`
	} `json:"hosts"`
}

type hostSoftwareResponse struct {
	Software []struct {
		InstalledVersions []struct {
			Vulnerabilities []string `json:"vulnerabilities"`
		} `json:"installed_versions"`
	} `json:"software"`
	Meta *struct {
		HasNextResults bool `json:"has_next_results"`
	} `json:"meta"`
}

// --- Uptime collection ---

func collectUptime(api *apiClient, db *sql.DB) error {
	log.Println("collecting uptime data...")

	hostIDs, err := fetchHostIDs(api, "status=online")
	if err != nil {
		return fmt.Errorf("fetch online hosts: %w", err)
	}
	log.Printf("  %d online hosts", len(hostIDs))

	if len(hostIDs) == 0 {
		return nil
	}

	now := time.Now().UTC()
	bucketStart := now.Truncate(time.Hour)
	validTo := bucketStart.Add(time.Hour)
	merged := chart.NewBitmap(hostIDs)

	// OR with existing in-bucket bitmap (accumulate semantic).
	var existingBytes []byte
	var existingEncoding uint8
	err = db.QueryRow(
		`SELECT host_bitmap, encoding_type FROM host_scd_data
		 WHERE dataset = 'uptime' AND entity_id = '' AND valid_from = ?`,
		bucketStart,
	).Scan(&existingBytes, &existingEncoding)
	if err == nil {
		existing, decErr := chart.DecodeBitmap(chart.Blob{Bytes: existingBytes, Encoding: existingEncoding})
		if decErr != nil {
			return fmt.Errorf("decode existing uptime bitmap: %w", decErr)
		}
		merged = chart.BlobOR(merged, existing)
	}

	blob := chart.BitmapToBlob(merged)
	_, err = db.Exec(
		`INSERT INTO host_scd_data (dataset, entity_id, host_bitmap, encoding_type, valid_from, valid_to)
		 VALUES ('uptime', '', ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE host_bitmap = VALUES(host_bitmap), encoding_type = VALUES(encoding_type)`,
		blob.Bytes, blob.Encoding, bucketStart, validTo,
	)
	if err != nil {
		return fmt.Errorf("write uptime SCD row: %w", err)
	}

	log.Printf("  wrote uptime row: %d hosts, valid_from %s", chart.BlobPopcount(merged), bucketStart)
	return nil
}

// --- CVE collection ---

func collectCVE(api *apiClient, db *sql.DB) error {
	log.Println("collecting CVE data...")

	hostIDs, err := fetchHostIDs(api, "")
	if err != nil {
		return fmt.Errorf("fetch all hosts: %w", err)
	}
	log.Printf("  %d total hosts", len(hostIDs))

	// Invert per-host fetches into per-CVE host sets.
	fetchStart := time.Now()
	cveHosts := make(map[string][]uint)
	for i, hostID := range hostIDs {
		cves, err := fetchHostCVEs(api, hostID)
		if err != nil {
			log.Printf("  warning: host %d: %v", hostID, err)
			continue
		}
		for _, cve := range cves {
			cveHosts[cve] = append(cveHosts[cve], hostID)
		}
		if (i+1)%50 == 0 {
			log.Printf("  fetched %d/%d hosts, %d unique CVEs so far (%.1fs)",
				i+1, len(hostIDs), len(cveHosts), time.Since(fetchStart).Seconds())
		}
	}
	log.Printf("  %d unique CVEs found in %.1fs", len(cveHosts), time.Since(fetchStart).Seconds())

	// Build the desired entity->bitmap map for the current hourly bucket.
	entityBitmaps := make(map[string]*roaring.Bitmap, len(cveHosts))
	for cve, hosts := range cveHosts {
		entityBitmaps[cve] = chart.NewBitmap(hosts)
	}

	// Snapshot rows are keyed to 1h boundaries (not 24h) so that row transitions
	// fall on hour marks. This lets tz-offset users' local-day queries resolve
	// "state at end of my day" to a row boundary observed at or before that
	// moment, rather than being pulled forward by the artificial UTC-midnight
	// transition that 24h keying would impose.
	writeStart := time.Now()
	bucketStart := time.Now().UTC().Truncate(time.Hour)
	if err := reconcileSnapshot(db, "cve", entityBitmaps, bucketStart); err != nil {
		return fmt.Errorf("reconcile SCD: %w", err)
	}
	log.Printf("  reconciled %d entities in %.1fs", len(entityBitmaps), time.Since(writeStart).Seconds())
	return nil
}

// reconcileSnapshot mirrors Datastore.recordSnapshot in
// server/chart/internal/mysql/data.go.
func reconcileSnapshot(db *sql.DB, dataset string, entityBitmaps map[string]*roaring.Bitmap, bucketStart time.Time) error {
	rows, err := db.Query(
		`SELECT entity_id, host_bitmap, encoding_type, valid_from
		 FROM host_scd_data
		 WHERE dataset = ? AND valid_to = ?`,
		dataset, scdOpenSentinel)
	if err != nil {
		return fmt.Errorf("fetch open SCD rows: %w", err)
	}
	defer rows.Close()
	type openEntry struct {
		bitmap    *roaring.Bitmap
		validFrom time.Time
	}
	openByEntity := make(map[string]openEntry)
	for rows.Next() {
		var entityID string
		var bitmapBytes []byte
		var encoding uint8
		var validFrom time.Time
		if err := rows.Scan(&entityID, &bitmapBytes, &encoding, &validFrom); err != nil {
			return fmt.Errorf("scan open SCD row: %w", err)
		}
		rb, err := chart.DecodeBitmap(chart.Blob{Bytes: bitmapBytes, Encoding: encoding})
		if err != nil {
			return fmt.Errorf("decode open bitmap for %q: %w", entityID, err)
		}
		openByEntity[entityID] = openEntry{bitmap: rb, validFrom: validFrom}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate open SCD rows: %w", err)
	}

	var toClose []string
	type upsertRow struct {
		entityID string
		blob     chart.Blob
	}
	var toUpsert []upsertRow

	for entityID, bitmap := range entityBitmaps {
		existing, hasOpen := openByEntity[entityID]
		if hasOpen && existing.bitmap.Equals(bitmap) {
			continue
		}
		if hasOpen && existing.validFrom.Before(bucketStart) {
			toClose = append(toClose, entityID)
		}
		toUpsert = append(toUpsert, upsertRow{entityID: entityID, blob: chart.BitmapToBlob(bitmap)})
	}

	for entityID := range openByEntity {
		if _, ok := entityBitmaps[entityID]; !ok {
			toClose = append(toClose, entityID)
		}
	}

	if len(toClose) > 0 {
		placeholders := make([]string, len(toClose))
		args := []any{bucketStart, dataset, scdOpenSentinel}
		for i, e := range toClose {
			placeholders[i] = "?"
			args = append(args, e)
		}
		// Concatenating hardcoded "?" placeholder strings, not user input.
		stmt := fmt.Sprintf( //nolint:gosec // G202
			`UPDATE host_scd_data SET valid_to = ?
			 WHERE dataset = ? AND valid_to = ? AND entity_id IN (%s)`,
			strings.Join(placeholders, ","))
		if _, err := db.Exec(stmt, args...); err != nil {
			return fmt.Errorf("close stale rows: %w", err)
		}
	}

	for i := 0; i < len(toUpsert); i += scdUpsertBatch {
		end := min(i+scdUpsertBatch, len(toUpsert))
		batch := toUpsert[i:end]

		placeholders := make([]string, len(batch))
		args := make([]any, 0, len(batch)*5)
		for j, r := range batch {
			placeholders[j] = "(?, ?, ?, ?, ?)"
			args = append(args, dataset, r.entityID, r.blob.Bytes, r.blob.Encoding, bucketStart)
		}
		// Concatenating hardcoded "(?,?,?,?,?)" placeholder strings, not user input.
		stmt := `INSERT INTO host_scd_data (dataset, entity_id, host_bitmap, encoding_type, valid_from) VALUES ` + //nolint:gosec // G202
			strings.Join(placeholders, ", ") +
			` ON DUPLICATE KEY UPDATE host_bitmap = VALUES(host_bitmap), encoding_type = VALUES(encoding_type)`
		if _, err := db.Exec(stmt, args...); err != nil {
			return fmt.Errorf("upsert rows: %w", err)
		}
	}
	return nil
}

// --- API helpers ---

// fetchHostIDs pages through the hosts list endpoint. Pass extra query params
// (e.g. "status=online") or "" for all hosts.
func fetchHostIDs(api *apiClient, extraParams string) ([]uint, error) {
	var all []uint
	for page := 0; ; page++ {
		path := fmt.Sprintf("/api/v1/fleet/hosts?per_page=%d&page=%d", perPage, page)
		if extraParams != "" {
			path += "&" + extraParams
		}

		resp, err := api.get(path)
		if err != nil {
			return nil, err
		}

		var result hostsResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("decode hosts page %d: %w", page, err)
		}

		for _, h := range result.Hosts {
			all = append(all, h.ID)
		}

		if len(result.Hosts) < perPage {
			break
		}
	}
	return all, nil
}

// fetchHostCVEs returns deduplicated CVE IDs for a single host.
func fetchHostCVEs(api *apiClient, hostID uint) ([]string, error) {
	seen := make(map[string]struct{})
	for page := 0; ; page++ {
		path := fmt.Sprintf("/api/v1/fleet/hosts/%d/software?vulnerable=true&per_page=%d&page=%d", hostID, perPage, page)

		resp, err := api.get(path)
		if err != nil {
			return nil, err
		}

		var result hostSoftwareResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("decode software for host %d page %d: %w", hostID, page, err)
		}

		for _, sw := range result.Software {
			for _, iv := range sw.InstalledVersions {
				for _, cve := range iv.Vulnerabilities {
					seen[cve] = struct{}{}
				}
			}
		}

		if result.Meta == nil || !result.Meta.HasNextResults {
			break
		}
	}

	cves := make([]string, 0, len(seen))
	for cve := range seen {
		cves = append(cves, cve)
	}
	return cves, nil
}
