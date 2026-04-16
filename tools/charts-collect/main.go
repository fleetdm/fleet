// charts-collect fetches live data from a Fleet instance via API and writes
// chart data into a local database. Designed to run hourly via cron.
//
// Uptime: fetches currently online hosts, ORs into the current hour's blob.
// CVE: fetches per-host vulnerability data, builds per-CVE host bitmaps, and
// reconciles them into host_scd_data (dataset='cve'). Unchanged CVEs keep their
// existing open row; changed bitmaps close the prior-day row and open a new one
// for today; intra-day changes overwrite today's row via ODKU.
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
	"bytes"
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
	// scdOpenSentinel / scdDateFormat / scdUpsertBatch mirror constants in
	// server/chart/internal/mysql/scd.go — the collector writes to the same table
	// out-of-process and must keep the encoding in sync.
	scdOpenSentinel = "9999-12-31"
	scdDateFormat   = "2006-01-02"
	scdUpsertBatch  = 200
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
	if err := collectPolicyFailing(api, db); err != nil {
		log.Printf("ERROR policy_failing collection: %v", err)
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

	// Build the desired entity->bitmap map for today.
	entityBitmaps := make(map[string][]byte, len(cveHosts))
	for cve, hosts := range cveHosts {
		entityBitmaps[cve] = chart.HostIDsToBlob(hosts)
	}

	// Reconcile against the SCD table.
	writeStart := time.Now()
	today := time.Now().UTC().Format(scdDateFormat)
	if err := reconcileSCD(db, "cve", entityBitmaps, today); err != nil {
		return fmt.Errorf("reconcile SCD: %w", err)
	}
	log.Printf("  reconciled %d entities in %.1fs", len(entityBitmaps), time.Since(writeStart).Seconds())
	return nil
}

// reconcileSCD applies the close-then-upsert flow for a dataset against the
// entity->bitmap map for today. Mirrors Datastore.RecordSCDData in
// server/chart/internal/mysql/scd.go.
func reconcileSCD(db *sql.DB, dataset string, entityBitmaps map[string][]byte, today string) error {
	// Fetch current open rows.
	rows, err := db.Query(
		`SELECT entity_id, host_bitmap, DATE_FORMAT(valid_from, '%Y-%m-%d')
		 FROM host_scd_data
		 WHERE dataset = ? AND valid_to = ?`,
		dataset, scdOpenSentinel)
	if err != nil {
		return fmt.Errorf("fetch open SCD rows: %w", err)
	}
	type openRow struct {
		bitmap    []byte
		validFrom string
	}
	openByEntity := make(map[string]openRow)
	for rows.Next() {
		var entityID, validFrom string
		var bitmap []byte
		if err := rows.Scan(&entityID, &bitmap, &validFrom); err != nil {
			rows.Close()
			return fmt.Errorf("scan open SCD row: %w", err)
		}
		openByEntity[entityID] = openRow{bitmap: bitmap, validFrom: validFrom}
	}
	rows.Close()

	// Partition: unchanged rows are skipped; changed rows get an upsert and (if
	// their open row is from a previous day) a close.
	var toClose []string
	var toUpsert []struct {
		entityID string
		bitmap   []byte
	}

	for entityID, bitmap := range entityBitmaps {
		existing, hasOpen := openByEntity[entityID]
		if hasOpen && bytes.Equal(existing.bitmap, bitmap) {
			continue
		}
		if hasOpen && existing.validFrom < today {
			toClose = append(toClose, entityID)
		}
		toUpsert = append(toUpsert, struct {
			entityID string
			bitmap   []byte
		}{entityID, bitmap})
	}

	// Entities no longer present: close their open rows.
	for entityID := range openByEntity {
		if _, ok := entityBitmaps[entityID]; !ok {
			toClose = append(toClose, entityID)
		}
	}

	if len(toClose) > 0 {
		placeholders := make([]string, len(toClose))
		args := []any{today, dataset, scdOpenSentinel}
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
		args := make([]any, 0, len(batch)*4)
		for j, r := range batch {
			placeholders[j] = "(?, ?, ?, ?)"
			args = append(args, dataset, r.entityID, r.bitmap, today)
		}
		// Concatenating hardcoded "(?,?,?,?)" placeholder strings, not user input.
		stmt := `INSERT INTO host_scd_data (dataset, entity_id, host_bitmap, valid_from) VALUES ` + //nolint:gosec // G202
			strings.Join(placeholders, ", ") +
			` ON DUPLICATE KEY UPDATE host_bitmap = VALUES(host_bitmap)`
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

// policyResponse is the minimal subset of fleet.Policy fields we need.
type policyResponse struct {
	ID               uint  `json:"id"`
	FailingHostCount uint  `json:"failing_host_count"`
	TeamID           *uint `json:"team_id"`
}

// listPoliciesResponse matches both /policies and /fleets/{id}/policies.
type listPoliciesResponse struct {
	Policies []policyResponse `json:"policies"`
}

// listTeamsResponse matches /fleets.
type listTeamsResponse struct {
	Teams []struct {
		ID uint `json:"id"`
	} `json:"teams"`
}

func collectPolicyFailing(api *apiClient, db *sql.DB) error {
	log.Println("collecting policy_failing data...")

	policies, err := fetchAllPolicies(api)
	if err != nil {
		return fmt.Errorf("fetch policies: %w", err)
	}
	log.Printf("  %d policies across all scopes", len(policies))

	entityBitmaps := make(map[string][]byte, len(policies))
	totalFailing := 0
	for _, p := range policies {
		key := fmt.Sprintf("%d", p.ID)
		if p.FailingHostCount == 0 {
			entityBitmaps[key] = []byte{}
			continue
		}
		hostIDs, err := fetchFailingHostIDsForPolicy(api, p.ID, p.TeamID)
		if err != nil {
			log.Printf("  warning: policy %d: %v", p.ID, err)
			entityBitmaps[key] = []byte{}
			continue
		}
		blob := chart.HostIDsToBlob(hostIDs)
		if blob == nil {
			blob = []byte{}
		}
		entityBitmaps[key] = blob
		totalFailing += len(hostIDs)
	}
	log.Printf("  %d total (policy, failing-host) pairs", totalFailing)

	writeStart := time.Now()
	today := time.Now().UTC().Format(scdDateFormat)
	if err := reconcileSCD(db, "policy_failing", entityBitmaps, today); err != nil {
		return fmt.Errorf("reconcile SCD: %w", err)
	}
	log.Printf("  reconciled %d entities in %.1fs", len(entityBitmaps), time.Since(writeStart).Seconds())
	return nil
}

// fetchAllPolicies returns every policy across global and team scopes.
func fetchAllPolicies(api *apiClient) ([]policyResponse, error) {
	var all []policyResponse

	globals, err := fetchPoliciesFromPath(api, "/api/v1/fleet/global/policies")
	if err != nil {
		return nil, fmt.Errorf("global: %w", err)
	}
	all = append(all, globals...)

	// Team policies — iterate teams.
	teamIDs, err := fetchTeamIDs(api)
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}
	for _, tid := range teamIDs {
		teamPath := fmt.Sprintf("/api/v1/fleet/fleets/%d/policies", tid)
		teamPolicies, err := fetchPoliciesFromPath(api, teamPath)
		if err != nil {
			log.Printf("  warning: team %d policies: %v", tid, err)
			continue
		}
		all = append(all, teamPolicies...)
	}
	return all, nil
}

// fetchPoliciesFromPath pages through a policy list endpoint.
func fetchPoliciesFromPath(api *apiClient, basePath string) ([]policyResponse, error) {
	var all []policyResponse
	sep := "?"
	if strings.Contains(basePath, "?") {
		sep = "&"
	}
	for page := 0; ; page++ {
		path := fmt.Sprintf("%s%sper_page=%d&page=%d", basePath, sep, perPage, page)
		resp, err := api.get(path)
		if err != nil {
			return nil, err
		}
		var result listPoliciesResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("decode %s page %d: %w", basePath, page, err)
		}
		all = append(all, result.Policies...)
		if len(result.Policies) < perPage {
			break
		}
	}
	return all, nil
}

// fetchTeamIDs returns all team IDs in the Fleet instance.
func fetchTeamIDs(api *apiClient) ([]uint, error) {
	var ids []uint
	for page := 0; ; page++ {
		path := fmt.Sprintf("/api/v1/fleet/fleets?per_page=%d&page=%d", perPage, page)
		resp, err := api.get(path)
		if err != nil {
			return nil, err
		}
		var result listTeamsResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("decode teams page %d: %w", page, err)
		}
		for _, t := range result.Teams {
			ids = append(ids, t.ID)
		}
		if len(result.Teams) < perPage {
			break
		}
	}
	return ids, nil
}

// fetchFailingHostIDsForPolicy returns the IDs of hosts currently failing a policy.
// Team-scoped policies require a team_id query param; global policies don't.
func fetchFailingHostIDsForPolicy(api *apiClient, policyID uint, teamID *uint) ([]uint, error) {
	var hostIDs []uint
	for page := 0; ; page++ {
		path := fmt.Sprintf(
			"/api/v1/fleet/hosts?per_page=%d&page=%d&policy_id=%d&policy_response=failing",
			perPage, page, policyID)
		if teamID != nil {
			path += fmt.Sprintf("&team_id=%d", *teamID)
		}
		resp, err := api.get(path)
		if err != nil {
			return nil, err
		}
		var result hostsResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("decode failing hosts page %d: %w", page, err)
		}
		for _, h := range result.Hosts {
			hostIDs = append(hostIDs, h.ID)
		}
		if len(result.Hosts) < perPage {
			break
		}
	}
	return hostIDs, nil
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
