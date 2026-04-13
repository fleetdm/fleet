// charts-collect fetches live data from a Fleet instance via API and writes
// chart blob data into a local database. Designed to run hourly via cron.
//
// Uptime: fetches currently online hosts, ORs into the current hour's blob.
// CVE: fetches per-host vulnerability data, writes a daily snapshot (hour=-1)
// for each CVE.
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

	"github.com/fleetdm/fleet/v4/server/chart"
	_ "github.com/go-sql-driver/mysql"
)

const (
	perPage        = 500
	cveInsertBatch = 200
)

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
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("ping mysql: %v", err)
	}

	api := &apiClient{
		baseURL: *fleetURL,
		token:   *fleetToken,
		http:    &http.Client{Timeout: 30 * time.Second}, //nolint:vet
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
	hour := now.Hour()
	dateStr := now.Format("2006-01-02")

	newBlob := chart.HostIDsToBlob(hostIDs)

	// OR with existing blob for this hour.
	var existing []byte
	err = db.QueryRow(
		`SELECT host_bitmap FROM host_hourly_data_blobs WHERE dataset = 'uptime' AND entity_id = '' AND chart_date = ? AND hour = ?`,
		dateStr, hour,
	).Scan(&existing)
	if err == nil {
		newBlob = chart.BlobOR(existing, newBlob)
	}

	_, err = db.Exec(
		`INSERT INTO host_hourly_data_blobs (dataset, entity_id, chart_date, hour, host_bitmap)
		 VALUES ('uptime', '', ?, ?, ?)
		 ON DUPLICATE KEY UPDATE host_bitmap = VALUES(host_bitmap)`,
		dateStr, hour, newBlob,
	)
	if err != nil {
		return fmt.Errorf("write uptime blob: %w", err)
	}

	log.Printf("  wrote uptime blob: %d hosts, %s hour %d", chart.BlobPopcount(newBlob), dateStr, hour)
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

	// Build CVE -> host IDs mapping.
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

	if len(cveHosts) == 0 {
		return nil
	}

	dateStr := time.Now().UTC().Format("2006-01-02")

	// Replace today's CVE blobs with the current snapshot.
	log.Printf("  deleting stale CVE blobs for %s...", dateStr)
	delStart := time.Now()
	if _, err := db.Exec(
		`DELETE FROM host_hourly_data_blobs WHERE dataset = 'cve' AND chart_date = ? AND hour = ?`,
		dateStr, chart.HourWholeDay,
	); err != nil {
		return fmt.Errorf("delete stale CVE blobs: %w", err)
	}
	log.Printf("  delete took %.1fs", time.Since(delStart).Seconds())

	// Batch inserts: ~200 per statement is a big win over individual round trips.
	log.Printf("  inserting %d CVE blobs in batches of %d...", len(cveHosts), cveInsertBatch)
	insertStart := time.Now()

	var (
		placeholders []string
		args         []any
		inserted     int
	)
	flush := func() error {
		if len(placeholders) == 0 {
			return nil
		}
		// Concatenating hardcoded "(...)" placeholder strings, not user input.
		stmt := `INSERT INTO host_hourly_data_blobs (dataset, entity_id, chart_date, hour, host_bitmap) VALUES ` + //nolint:gosec // G202
			strings.Join(placeholders, ",")
		if _, err := db.Exec(stmt, args...); err != nil {
			return fmt.Errorf("batch insert: %w", err)
		}
		inserted += len(placeholders)
		placeholders = placeholders[:0]
		args = args[:0]
		return nil
	}

	for cve, hosts := range cveHosts {
		blob := chart.HostIDsToBlob(hosts)
		placeholders = append(placeholders, "('cve', ?, ?, ?, ?)")
		args = append(args, cve, dateStr, chart.HourWholeDay, blob)

		if len(placeholders) >= cveInsertBatch {
			if err := flush(); err != nil {
				return err
			}
			log.Printf("  inserted %d/%d CVE blobs (%.1fs)", inserted, len(cveHosts), time.Since(insertStart).Seconds())
		}
	}
	if err := flush(); err != nil {
		return err
	}

	log.Printf("  wrote %d CVE blobs for %s in %.1fs", inserted, dateStr, time.Since(insertStart).Seconds())
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
		page++
	}

	cves := make([]string, 0, len(seen))
	for cve := range seen {
		cves = append(cves, cve)
	}
	return cves, nil
}

