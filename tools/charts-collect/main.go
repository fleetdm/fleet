// charts-collect fetches live data from a Fleet instance via API and writes
// chart data into a local database. Designed to run hourly via cron.
//
// Uptime: fetches currently online hosts, ORs into the current hour's blob.
// CVE: fetches per-host vulnerability data, upserts the current state as SCD
// rows (dataset='cve') with two statements per host: close stale open rows,
// then INSERT ... ON DUPLICATE KEY UPDATE to add/keep active rows.
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
	perPage = 500
	// scdOpenSentinel matches the value used in the chart MySQL datastore —
	// see server/chart/internal/mysql/scd.go.
	scdOpenSentinel = "9999-12-31 23:59:59"
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

	// Fetch per-host CVE lists.
	fetchStart := time.Now()
	hostCVEs := make(map[uint][]string, len(hostIDs))
	for i, hostID := range hostIDs {
		cves, err := fetchHostCVEs(api, hostID)
		if err != nil {
			log.Printf("  warning: host %d: %v", hostID, err)
			continue
		}
		hostCVEs[hostID] = cves
		if (i+1)%50 == 0 {
			log.Printf("  fetched %d/%d hosts (%.1fs)",
				i+1, len(hostIDs), time.Since(fetchStart).Seconds())
		}
	}
	log.Printf("  fetched CVEs for %d hosts in %.1fs", len(hostCVEs), time.Since(fetchStart).Seconds())

	// Record SCD state per host: close entries no longer present, upsert currently-active.
	nowStr := time.Now().UTC().Format("2006-01-02 15:04:05")
	writeStart := time.Now()
	var failed int
	for hostID, cves := range hostCVEs {
		if err := recordSCD(db, "cve", hostID, cves, nowStr); err != nil {
			log.Printf("  warning: record SCD for host %d: %v", hostID, err)
			failed++
		}
	}
	log.Printf("  recorded SCD for %d hosts (%d failed) in %.1fs",
		len(hostCVEs)-failed, failed, time.Since(writeStart).Seconds())
	return nil
}

// recordSCD applies the two-statement SCD upsert for a single host.
// Mirrors Datastore.RecordSCDData in server/chart/internal/mysql/scd.go.
func recordSCD(db *sql.DB, dataset string, hostID uint, entityIDs []string, nowStr string) error {
	// Step 1: close any open rows whose entity isn't in the current set (or all
	// open rows when the set is empty).
	if len(entityIDs) == 0 {
		_, err := db.Exec(
			`UPDATE host_scd_data SET valid_to = ?
			 WHERE dataset = ? AND host_id = ? AND valid_to = ?`,
			nowStr, dataset, hostID, scdOpenSentinel)
		return err
	}

	placeholders := make([]string, len(entityIDs))
	for i := range entityIDs {
		placeholders[i] = "?"
	}
	closeArgs := []any{nowStr, dataset, hostID, scdOpenSentinel}
	for _, e := range entityIDs {
		closeArgs = append(closeArgs, e)
	}
	// Concatenating hardcoded "?" placeholder strings, not user input.
	closeQuery := fmt.Sprintf( //nolint:gosec // G202
		`UPDATE host_scd_data SET valid_to = ?
		 WHERE dataset = ? AND host_id = ? AND valid_to = ?
		   AND entity_id NOT IN (%s)`,
		strings.Join(placeholders, ","))
	if _, err := db.Exec(closeQuery, closeArgs...); err != nil {
		return fmt.Errorf("close stale rows: %w", err)
	}

	// Step 2: upsert currently-active rows; ODKU with valid_from=valid_from is a
	// no-op that preserves the original valid_from when the row is already open.
	insertPlaceholders := make([]string, len(entityIDs))
	insertArgs := make([]any, 0, len(entityIDs)*4)
	for i, e := range entityIDs {
		insertPlaceholders[i] = "(?, ?, ?, ?)"
		insertArgs = append(insertArgs, dataset, hostID, e, nowStr)
	}
	// Concatenating hardcoded "(?, ?, ?, ?)" placeholder strings, not user input.
	insertQuery := `INSERT INTO host_scd_data (dataset, host_id, entity_id, valid_from) VALUES ` + //nolint:gosec // G202
		strings.Join(insertPlaceholders, ",") +
		` ON DUPLICATE KEY UPDATE valid_from = valid_from`
	if _, err := db.Exec(insertQuery, insertArgs...); err != nil {
		return fmt.Errorf("upsert rows: %w", err)
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
		page++
	}

	cves := make([]string, 0, len(seen))
	for cve := range seen {
		cves = append(cves, cve)
	}
	return cves, nil
}

