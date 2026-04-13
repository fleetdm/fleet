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
// Environment variables FLEET_URL and FLEET_TOKEN can be used instead of flags.
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart"
	_ "github.com/go-sql-driver/mysql"
)

const perPage = 500

func main() {
	fleetURL := flag.String("fleet-url", os.Getenv("FLEET_URL"), "Fleet server URL (or FLEET_URL env var)")
	fleetToken := flag.String("fleet-token", os.Getenv("FLEET_TOKEN"), "Fleet API token (or FLEET_TOKEN env var)")
	dsn := flag.String("mysql-dsn", "fleet:fleet@tcp(localhost:3306)/fleet?parseTime=true", "MySQL connection string")
	flag.Parse()

	if *fleetURL == "" || *fleetToken == "" {
		log.Fatal("--fleet-url and --fleet-token are required (or set FLEET_URL and FLEET_TOKEN)")
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
		`SELECT host_bitmap FROM host_hourly_data_blobs WHERE dataset = 'uptime' AND entity_id = 0 AND chart_date = ? AND hour = ?`,
		dateStr, hour,
	).Scan(&existing)
	if err == nil {
		newBlob = chart.BlobOR(existing, newBlob)
	}

	_, err = db.Exec(
		`INSERT INTO host_hourly_data_blobs (dataset, entity_id, chart_date, hour, host_bitmap)
		 VALUES ('uptime', 0, ?, ?, ?)
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
			log.Printf("  processed %d/%d hosts, %d unique CVEs so far", i+1, len(hostIDs), len(cveHosts))
		}
	}
	log.Printf("  %d unique CVEs found", len(cveHosts))

	if len(cveHosts) == 0 {
		return nil
	}

	// Ensure the entity mapping table exists.
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS chart_cve_entities (
			id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			cve VARCHAR(30) NOT NULL UNIQUE
		)
	`); err != nil {
		return fmt.Errorf("create chart_cve_entities: %w", err)
	}

	entityMap, err := ensureCVEEntities(db, cveHosts)
	if err != nil {
		return fmt.Errorf("map CVE entities: %w", err)
	}

	dateStr := time.Now().UTC().Format("2006-01-02")

	// Replace today's CVE blobs with the current snapshot.
	if _, err := db.Exec(
		`DELETE FROM host_hourly_data_blobs WHERE dataset = 'cve' AND chart_date = ? AND hour = ?`,
		dateStr, chart.HourWholeDay,
	); err != nil {
		return fmt.Errorf("delete stale CVE blobs: %w", err)
	}

	for cve, hosts := range cveHosts {
		entityID := entityMap[cve]
		blob := chart.HostIDsToBlob(hosts)

		if _, err := db.Exec(
			`INSERT INTO host_hourly_data_blobs (dataset, entity_id, chart_date, hour, host_bitmap)
			 VALUES ('cve', ?, ?, ?, ?)`,
			entityID, dateStr, chart.HourWholeDay, blob,
		); err != nil {
			return fmt.Errorf("write CVE blob %s: %w", cve, err)
		}
	}

	log.Printf("  wrote %d CVE blobs for %s", len(cveHosts), dateStr)
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

// --- DB helpers ---

// ensureCVEEntities inserts any new CVE strings into the mapping table and
// returns the full CVE -> entity_id map.
func ensureCVEEntities(db *sql.DB, cveHosts map[string][]uint) (map[string]uint, error) {
	for cve := range cveHosts {
		if _, err := db.Exec(`INSERT IGNORE INTO chart_cve_entities (cve) VALUES (?)`, cve); err != nil {
			return nil, fmt.Errorf("insert CVE %s: %w", cve, err)
		}
	}

	rows, err := db.Query(`SELECT id, cve FROM chart_cve_entities`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]uint)
	for rows.Next() {
		var id uint
		var cve string
		if err := rows.Scan(&id, &cve); err != nil {
			return nil, err
		}
		result[cve] = id
	}
	return result, rows.Err()
}
