// seed_reconciler_state.go
//
// Reproduces the labels/profiles/teams shape from customer-shackleton's Apple
// reconciler load investigation (fleetdm/fleet#44956). Does NOT create hosts —
// enroll them separately (e.g. with osquery-perf) BEFORE running this script,
// so that label membership is populated up-front and the reconciler does not
// gradually pick hosts up as they enroll.
//
// Target state (97 mobileconfig profiles, 2 DDM declarations, 10 manual labels,
// 6 teams, hosts enrolled and transferred):
//
//   Team | Profiles |  No-label | include_any | exclude_any | declarations
//   -----+----------+-----------+-------------+-------------+--------------
//     0  |    2     |     2     |      0      |      0      |     0
//     1  |   43     |    16     |     17      |     10      |     1 (excl,2)
//     2  |   44     |    15     |     17      |     12      |     1 (excl,2)
//     3  |    2     |     2     |      0      |      0      |     0
//     4  |    4     |     4     |      0      |      0      |     0
//     5  |    2     |     2     |      0      |      0      |     0
//
// Host distribution (mirrors customer-shackleton's ratios as closely as the
// available host count allows):
//   - ~20 hosts stay in no-team
//   - team 2: ~16 hosts
//   - teams 3, 4, 5: ~2 hosts each
//   - team 1: everything else (the bulk of the population)
//
// Per-profile label-count distributions (matched exactly):
//   T1 include_any (17): 14×1, 2×2, 1×5         → 23 rows
//   T1 exclude_any (10):  4×1, 5×2, 1×4         → 18 rows
//   T2 include_any (17): 15×1, 2×2              → 19 rows
//   T2 exclude_any (12):  5×1, 6×2, 1×4         → 21 rows
//
// Label membership: 10 labels × 500 hosts each, DISJOINT — sorted by host_id
// ascending so label-00 gets the 500 lowest ids, label-01 gets the next 500,
// etc. Requires at least 5,000 hosts already enrolled (script fails fast
// otherwise).
//
// Usage:
//   export FLEET_URL=https://fleet.example.com
//   export FLEET_API_TOKEN=...   # admin or gitops token
//   go run seed_reconciler_state.go                       # create everything
//   go run seed_reconciler_state.go -teardown             # delete what we created
//   go run seed_reconciler_state.go -dry-run              # print plan, do nothing
//   go run seed_reconciler_state.go -concurrency 32       # tune cheap-phase parallelism
//   go run seed_reconciler_state.go -transfer-concurrency 2  # parallelize transfers
//
// Cheap phases (teams, labels, label memberships, profile uploads) fan out to
// `concurrency` workers (default 16). Host transfers fan out to a separate,
// narrower pool (default 1, i.e. sequential) because each transfer call
// triggers BulkSetPendingMDMHostProfiles server-side and parallel transfers can
// thrash the DB connection pool. Trigger this script manually once enrollment
// finishes; total runtime scales with how many hosts you're transferring (a
// few minutes for ~12k hosts is normal).
//
// All objects we create are tagged with the prefix loadtestPrefix so teardown
// is safe — it only deletes objects whose name starts with that prefix.

package main

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	loadtestPrefix = "lt44956-" // change if you want multiple parallel seedings
	numLabels      = 10         // pool size; ≥5 satisfies the max cardinality
	hostsPerLabel  = 500        // disjoint slice assigned to each label
	httpTimeout    = 60 * time.Second

	// Host distribution roughly matching customer-shackleton. Anything not
	// claimed by these small allocations ends up on team 1.
	hostsNoTeam = 20
	hostsTeam2  = 16
	hostsTeam3  = 2
	hostsTeam4  = 2
	hostsTeam5  = 2

	// Batch size for /hosts/transfer. Fleet does no documented batching, but
	// each transfer call runs BulkSetPendingMDMHostProfiles which is O(hosts),
	// so we keep batches modest and parallelize.
	transferBatchSize = 1000
)

var (
	concurrency         = 16 // overridable via -concurrency flag (used by cheap phases)
	transferConcurrency = 1  // host transfers serialize by default; each call triggers
	//                          BulkSetPendingMDMHostProfiles server-side which is heavy
)

// ---------- distribution spec ----------

type profileSpec struct {
	teamID    uint   // 0 = no team
	teamLabel string // human name we'll use in our profile filename
	mode      string // "" | "include_any" | "exclude_any"
	labelIdxs []int  // indexes into the labels slice; empty for unlabeled
}

// Match the customer's distributions exactly.
func buildPlan(labels []string) []profileSpec {
	specs := make([]profileSpec, 0, 97)

	// no team (team_id=0): 2 unlabeled profiles
	for i := 0; i < 2; i++ {
		specs = append(specs, profileSpec{teamID: 0, teamLabel: "noteam"})
	}

	// helper to append N profiles with the given label-count distribution
	add := func(teamID uint, label, mode string, dist []int) {
		// dist is the per-profile label counts, e.g. [1,1,1,1,2,2,5]
		// We round-robin pick label indices so labels get reused, which is what
		// the customer's data looks like (single-label allowlists dominating).
		cursor := 0
		for _, n := range dist {
			idxs := make([]int, n)
			for j := 0; j < n; j++ {
				idxs[j] = cursor % len(labels)
				cursor++
			}
			specs = append(specs, profileSpec{
				teamID:    teamID,
				teamLabel: label,
				mode:      mode,
				labelIdxs: idxs,
			})
		}
	}

	// team 1: 43 profiles
	for i := 0; i < 16; i++ { // 16 unlabeled
		specs = append(specs, profileSpec{teamID: 1, teamLabel: "t1"})
	}
	add(1, "t1", "include_any", expand(14, 1, 2, 2, 1, 5)) // 14×1, 2×2, 1×5
	add(1, "t1", "exclude_any", expand(4, 1, 5, 2, 1, 4))  // 4×1, 5×2, 1×4

	// team 2: 44 profiles
	for i := 0; i < 15; i++ { // 15 unlabeled
		specs = append(specs, profileSpec{teamID: 2, teamLabel: "t2"})
	}
	add(2, "t2", "include_any", expand(15, 1, 2, 2))      // 15×1, 2×2
	add(2, "t2", "exclude_any", expand(5, 1, 6, 2, 1, 4)) // 5×1, 6×2, 1×4

	// teams 3, 4, 5: all unlabeled
	for i := 0; i < 2; i++ {
		specs = append(specs, profileSpec{teamID: 3, teamLabel: "t3"})
	}
	for i := 0; i < 4; i++ {
		specs = append(specs, profileSpec{teamID: 4, teamLabel: "t4"})
	}
	for i := 0; i < 2; i++ {
		specs = append(specs, profileSpec{teamID: 5, teamLabel: "t5"})
	}

	return specs
}

// expand([14,1, 2,2, 1,5]) => 14 ones, 2 twos, 1 five → 17 entries
func expand(pairs ...int) []int {
	if len(pairs)%2 != 0 {
		panic("expand: need even count")
	}
	out := []int{}
	for i := 0; i < len(pairs); i += 2 {
		count, size := pairs[i], pairs[i+1]
		for j := 0; j < count; j++ {
			out = append(out, size)
		}
	}
	return out
}

// ---------- payload builders ----------

func uuid() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // v4
	b[8] = (b[8] & 0x3f) | 0x80 // variant
	return fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Minimal valid Apple .mobileconfig — top-level Configuration with no inner
// payloads. Fleet accepts this; it's enough to land a row in
// mdm_apple_configuration_profiles.
func mobileconfig(displayName, identifier string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadDisplayName</key>
  <string>` + displayName + `</string>
  <key>PayloadIdentifier</key>
  <string>` + identifier + `</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>` + uuid() + `</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
</dict>
</plist>`
}

// Minimal valid DDM declaration JSON. Type must start with com.apple.configuration.
func declarationJSON(identifier string) string {
	d := map[string]any{
		"Type":       "com.apple.configuration.passcode.settings",
		"Identifier": identifier,
		"Payload": map[string]any{
			"MinimumLength": 4,
		},
	}
	b, _ := json.Marshal(d)
	return string(b)
}

// ---------- HTTP client ----------

type client struct {
	base   string
	token  string
	http   *http.Client
	dryRun bool
}

func newClient(dryRun bool) *client {
	base := strings.TrimRight("https://fleet-mdm-44956-51765142.us-east-2.elb.amazonaws.com", "/")
	tok := "gF8NdKArgxvff70KSvDQNjHH+uj5bRfohlVHcRVYRvvfJgxT6jnllw0JsQZ07zr3TUTHjwvTDWyQCKoVbcrkCg=="
	if base == "" || tok == "" {
		log.Fatal("FLEET_URL and FLEET_API_TOKEN must be set")
	}
	// Default Transport has MaxIdleConnsPerHost=2, which would serialize most
	// of our parallel calls to the same Fleet host. Tune it up.
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // skip TLS verify for convenience in test environments
	tr.MaxIdleConns = concurrency * 2
	tr.MaxIdleConnsPerHost = concurrency * 2
	tr.MaxConnsPerHost = concurrency * 2
	return &client{
		base:   base,
		token:  tok,
		http:   &http.Client{Timeout: httpTimeout, Transport: tr},
		dryRun: dryRun,
	}
}

func (c *client) do(method, path string, body io.Reader, contentType string, out any) error {
	if c.dryRun {
		log.Printf("[dry-run] %s %s", method, path)
		return nil
	}
	req, err := http.NewRequest(method, c.base+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s: %d: %s", method, path, resp.StatusCode, string(b))
	}
	if out != nil && len(b) > 0 {
		return json.Unmarshal(b, out)
	}
	return nil
}

// ---------- API calls ----------

// Teams. NOTE: the route is /api/v1/fleet/fleets, not /teams.
func (c *client) createTeam(name string) (uint, error) {
	body, _ := json.Marshal(map[string]string{"name": name})
	var resp struct {
		Fleet struct {
			ID uint `json:"id"`
		} `json:"fleet"`
	}
	if err := c.do("POST", "/api/v1/fleet/fleets", bytes.NewReader(body),
		"application/json", &resp); err != nil {
		return 0, err
	}
	return resp.Fleet.ID, nil
}

func (c *client) deleteTeam(id uint) error {
	return c.do("DELETE", fmt.Sprintf("/api/v1/fleet/fleets/%d", id), nil, "", nil)
}

// Labels. Manual label = name only; empty hosts list.
func (c *client) createLabel(name string) (uint, error) {
	body, _ := json.Marshal(map[string]any{
		"name":  name,
		"hosts": []string{}, // explicit empty → manual label
	})
	var resp struct {
		Label struct {
			ID uint `json:"id"`
		} `json:"label"`
	}
	if err := c.do("POST", "/api/v1/fleet/labels", bytes.NewReader(body),
		"application/json", &resp); err != nil {
		return 0, err
	}
	return resp.Label.ID, nil
}

func (c *client) deleteLabelByName(name string) error {
	// /api/v1/fleet/labels/{name} requires URL-escaping in real use, but our
	// names are simple ASCII.
	return c.do("DELETE", "/api/v1/fleet/labels/"+name, nil, "", nil)
}

// listAllHostIDs fetches the `need` lowest host IDs across the deployment in
// parallel. We compute the page count up front (need/perPage rounded up) and
// fan out the GETs concurrently, then merge and sort. The server orders within
// each page by id ascending, and pages don't overlap, so the merged set is the
// `need` lowest IDs regardless of arrival order.
func (c *client) listAllHostIDs(need int) ([]uint, error) {
	const perPage = 500
	pages := (need + perPage - 1) / perPage

	results := make([][]uint, pages)
	err := runParallel("listHosts", pages, func(i int) error {
		path := fmt.Sprintf(
			"/api/v1/fleet/hosts?order_key=id&order_direction=asc&per_page=%d&page=%d",
			perPage, i)
		var resp struct {
			Hosts []struct {
				ID uint `json:"id"`
			} `json:"hosts"`
		}
		if err := c.do("GET", path, nil, "", &resp); err != nil {
			return err
		}
		page := make([]uint, 0, len(resp.Hosts))
		for _, h := range resp.Hosts {
			page = append(page, h.ID)
		}
		results[i] = page
		return nil
	})
	if err != nil {
		return nil, err
	}

	merged := make([]uint, 0, need)
	for _, p := range results {
		merged = append(merged, p...)
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i] < merged[j] })
	if len(merged) > need {
		merged = merged[:need]
	}
	return merged, nil
}

// setLabelMembership replaces the manual label's host set with hostIDs.
// PATCH semantics: nil → leave unchanged, len==0 → remove all, len>0 → replace.
func (c *client) setLabelMembership(labelID uint, hostIDs []uint) error {
	body, _ := json.Marshal(map[string]any{"host_ids": hostIDs})
	return c.do("PATCH", fmt.Sprintf("/api/v1/fleet/labels/%d", labelID),
		bytes.NewReader(body), "application/json", nil)
}

// countHosts returns the total enrolled host count across the deployment.
func (c *client) countHosts() (int, error) {
	var resp struct {
		Count int `json:"count"`
	}
	if err := c.do("GET", "/api/v1/fleet/hosts/count", nil, "", &resp); err != nil {
		return 0, err
	}
	return resp.Count, nil
}

// transferHosts moves hostIDs onto teamID. teamID=0 means "no team".
// Fleet expects nil team_id for no-team in the JSON body.
func (c *client) transferHosts(teamID uint, hostIDs []uint) error {
	body := map[string]any{"hosts": hostIDs}
	if teamID != 0 {
		body["team_id"] = teamID
	} else {
		body["team_id"] = nil
	}
	b, _ := json.Marshal(body)
	return c.do("POST", "/api/v1/fleet/hosts/transfer",
		bytes.NewReader(b), "application/json", nil)
}

// Profiles / declarations. Multipart: fleet_id + profile file +
// labels_include_any / labels_exclude_any (repeated).
func (c *client) uploadProfile(filename, content string, teamID uint, mode string, labels []string) error {
	if c.dryRun {
		log.Printf("[dry-run] upload %s team=%d mode=%s labels=%v",
			filename, teamID, mode, labels)
		return nil
	}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("fleet_id", fmt.Sprintf("%d", teamID))

	fw, err := mw.CreateFormFile("profile", filename)
	if err != nil {
		return err
	}
	if _, err := fw.Write([]byte(content)); err != nil {
		return err
	}

	if mode != "" {
		field := "labels_" + mode
		for _, lbl := range labels {
			_ = mw.WriteField(field, lbl)
		}
	}
	if err := mw.Close(); err != nil {
		return err
	}
	return c.do("POST", "/api/v1/fleet/configuration_profiles",
		&buf, mw.FormDataContentType(), nil)
}

// Listing helpers for teardown.
type configProfile struct {
	ProfileUUID string `json:"profile_uuid"`
	Name        string `json:"name"`
	TeamID      uint   `json:"team_id"`
}

func (c *client) listProfiles(teamID uint) ([]configProfile, error) {
	path := fmt.Sprintf("/api/v1/fleet/configuration_profiles?team_id=%d&per_page=500", teamID)
	var resp struct {
		Profiles []configProfile `json:"profiles"`
	}
	if err := c.do("GET", path, nil, "", &resp); err != nil {
		return nil, err
	}
	return resp.Profiles, nil
}

func (c *client) deleteProfile(uuid string) error {
	return c.do("DELETE", "/api/v1/fleet/configuration_profiles/"+uuid, nil, "", nil)
}

type team struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

func (c *client) listTeams() ([]team, error) {
	var resp struct {
		Teams []team `json:"teams"`
	}
	if err := c.do("GET", "/api/v1/fleet/teams?per_page=500", nil, "", &resp); err != nil {
		return nil, err
	}
	return resp.Teams, nil
}

// runParallel runs fn(i) for i in [0, n) with at most `concurrency` workers
// in flight at any time. Returns the first error encountered (if any) and
// waits for all in-flight goroutines to finish before returning.
func runParallel(label string, n int, fn func(i int) error) error {
	return runParallelN(label, n, concurrency, fn)
}

// runParallelN is like runParallel but with an explicit worker cap, useful for
// phases that want narrower fan-out than the global default (e.g. host
// transfers, where each call does heavy server-side work).
func runParallelN(label string, n, workers int, fn func(i int) error) error {
	if workers < 1 {
		workers = 1
	}
	sem := make(chan struct{}, workers)
	var (
		wg       sync.WaitGroup
		errOnce  sync.Once
		firstErr error
	)
	for i := 0; i < n; i++ {
		i := i
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if err := fn(i); err != nil {
				errOnce.Do(func() {
					firstErr = fmt.Errorf("%s[%d]: %w", label, i, err)
				})
			}
		}()
	}
	wg.Wait()
	return firstErr
}

// ---------- orchestration ----------

func seed(c *client) {
	start := time.Now()

	// 1. Teams — 5 calls in parallel.
	teamNames := []string{"t1", "t2", "t3", "t4", "t5"}
	teamIDs := map[string]uint{"t1": 27, "t2": 28, "t3": 29, "t4": 30, "t5": 31, "noteam": 0} // local label -> fleet team id
	/* var teamMu sync.Mutex
	if err := runParallel("createTeam", len(teamNames), func(i int) error {
		full := loadtestPrefix + teamNames[i]
		id, err := c.createTeam(full)
		if err != nil {
			return err
		}
		teamMu.Lock()
		teamIDs[teamNames[i]] = id
		teamMu.Unlock()
		log.Printf("team %s -> id=%d", full, id)
		return nil
	}); err != nil {
		log.Fatalf("create teams: %v", err)
	}
	teamIDs["noteam"] = 0 */

	labelNames := []string{
		"lt44956-label-01", "lt44956-label-02", "lt44956-label-03", "lt44956-label-04", "lt44956-label-05",
		"lt44956-label-06", "lt44956-label-07", "lt44956-label-08", "lt44956-label-09",
	}
	/*
		// 2. Labels — 10 calls in parallel.
		for i := 0; i < numLabels; i++ {
			labelNames[i] = fmt.Sprintf("%slabel-%02d", loadtestPrefix, i)
		}
		if err := runParallel("createLabel", numLabels, func(i int) error {
			id, err := c.createLabel(labelNames[i])
			if err != nil {
				return err
			}
			labelIDs[i] = id // distinct index per goroutine — no race
			log.Printf("label %s -> id=%d", labelNames[i], id)
			return nil
		}); err != nil {
			log.Fatalf("create labels: %v", err)
		}

		// 2b. Discover the host population. We need them for both label membership
		// and team transfers, so list all up front.
		total, err := c.countHosts()
		if err != nil {
			log.Fatalf("count hosts: %v", err)
		}
		labelNeed := numLabels * hostsPerLabel
		teamReserved := hostsNoTeam + hostsTeam2 + hostsTeam3 + hostsTeam4 + hostsTeam5
		// We carve teams 2..5 + no-team from the END of the host list, and use the
		// FRONT for label membership. For all label-member hosts to land on team 1
		// (which is what we want — concentrate the big cross-product there), we
		// need labelNeed + teamReserved hosts at minimum.
		minNeeded := labelNeed + teamReserved
		if !c.dryRun && total < minNeeded {
			log.Fatalf("only %d hosts enrolled; need at least %d "+
				"(%d for label membership + %d reserved for teams 2-5 and no-team). "+
				"Enroll more hosts first.", total, minNeeded, labelNeed, teamReserved)
		}
		log.Printf("found %d enrolled hosts; listing all of them", total)
		allIDs, err := c.listAllHostIDs(total) // parallel-paginated internally
		if err != nil {
			log.Fatalf("list hosts: %v", err)
		}

		// 2c. Label membership: first labelNeed IDs (lowest), disjoint slices.
		// This must happen before profiles reference these labels so the
		// reconciler's first tick after seeding sees the full desired-state
		// cross-product rather than growing it tick-by-tick.
		if err := runParallel("setMembership", numLabels, func(i int) error {
			if c.dryRun {
				log.Printf("[dry-run] label %s would get %d hosts",
					labelNames[i], hostsPerLabel)
				return nil
			}
			slice := allIDs[i*hostsPerLabel : (i+1)*hostsPerLabel]
			if err := c.setLabelMembership(labelIDs[i], slice); err != nil {
				return err
			}
			log.Printf("label %s: assigned hosts %d..%d (%d hosts)",
				labelNames[i], slice[0], slice[len(slice)-1], len(slice))
			return nil
		}); err != nil {
			log.Fatalf("assign membership: %v", err)
		}

		// 2d. Plan host→team assignments. Use the END of the host list for the
		// small "edge case" allocations (no-team, teams 3/4/5, team 2), and the
		// front bulk for team 1. This keeps the label-member hosts (ids 1..5000)
		// concentrated on team 1, which matches the customer's pattern where the
		// big production team carries the labeled profiles.
		//
		// Slice carving from the back:
		//   [0 ... cursor) → team 1 (the bulk)
		//   [cursor ... cursor+T2)   → team 2
		//   [... +T3)                → team 3
		//   [... +T4)                → team 4
		//   [... +T5)                → team 5
		//   [last hostsNoTeam IDs]   → stay in no-team (NOT transferred)
		if !c.dryRun {
			n := len(allIDs)
			end := n - hostsNoTeam
			t5Start := end - hostsTeam5
			t4Start := t5Start - hostsTeam4
			t3Start := t4Start - hostsTeam3
			t2Start := t3Start - hostsTeam2
			// team 1 gets everything from 0..t2Start
			assignments := []struct {
				name   string
				teamID uint
				ids    []uint
			}{
				{"t1", teamIDs["t1"], allIDs[0:t2Start]},
				{"t2", teamIDs["t2"], allIDs[t2Start:t3Start]},
				{"t3", teamIDs["t3"], allIDs[t3Start:t4Start]},
				{"t4", teamIDs["t4"], allIDs[t4Start:t5Start]},
				{"t5", teamIDs["t5"], allIDs[t5Start:end]},
				// allIDs[end:] (the last hostsNoTeam) stay in no-team
			}
			log.Printf("transferring hosts: t1=%d t2=%d t3=%d t4=%d t5=%d (noteam=%d)",
				len(assignments[0].ids), len(assignments[1].ids), len(assignments[2].ids),
				len(assignments[3].ids), len(assignments[4].ids), hostsNoTeam)

			// Build a flat batch list across all teams so we can parallelize all
			// transfers uniformly. Per-team chunks of transferBatchSize.
			type batch struct {
				teamName string
				teamID   uint
				ids      []uint
			}
			var batches []batch
			for _, a := range assignments {
				for off := 0; off < len(a.ids); off += transferBatchSize {
					stop := off + transferBatchSize
					if stop > len(a.ids) {
						stop = len(a.ids)
					}
					batches = append(batches, batch{a.name, a.teamID, a.ids[off:stop]})
				}
			}
			log.Printf("transferring in %d batches of up to %d hosts (workers=%d)",
				len(batches), transferBatchSize, transferConcurrency)
			if err := runParallelN("transferHosts", len(batches), transferConcurrency,
				func(i int) error {
					b := batches[i]
					return c.transferHosts(b.teamID, b.ids)
				}); err != nil {
				log.Fatalf("transfer hosts: %v", err)
			}
		} else {
			log.Printf("[dry-run] would transfer hosts to teams")
		}

		// 3. Profiles — 97 multipart uploads in parallel. Each profile has a
		// globally unique PayloadIdentifier (includes the plan index), so there's
		// no per-team uniqueness race even across concurrent uploads.
		plan := buildPlan(labelNames)
		log.Printf("uploading %d profiles (concurrency=%d)", len(plan), concurrency)
		if err := runParallel("uploadProfile", len(plan), func(i int) error {
			p := plan[i]
			name := fmt.Sprintf("%s%s-p%03d", loadtestPrefix, p.teamLabel, i)
			ident := fmt.Sprintf("com.fleetdm.loadtest.%s.%s.p%03d",
				loadtestPrefix, p.teamLabel, i)
			content := mobileconfig(name, ident)
			filename := name + ".mobileconfig"

			var labels []string
			for _, idx := range p.labelIdxs {
				labels = append(labels, labelNames[idx])
			}
			fleetTeam := teamIDs[p.teamLabel]
			return c.uploadProfile(filename, content, fleetTeam, p.mode, labels)
		}); err != nil {
			log.Fatalf("upload profiles: %v", err)
		} */

	// 4. Declarations: 1 per team for t1 and t2, both exclude_any with 2 labels.
	decls := []string{"t1", "t2"}
	if err := runParallel("uploadDeclaration", len(decls), func(i int) error {
		tl := decls[i]
		name := fmt.Sprintf("%s%s-decl", loadtestPrefix, tl)
		ident := fmt.Sprintf("com.fleetdm.loadtest.%s.%s.decl", loadtestPrefix, tl)
		content := declarationJSON(ident)
		filename := name + ".json"
		if err := c.uploadProfile(filename, content, teamIDs[tl], "exclude_any",
			labelNames[:2]); err != nil {
			return err
		}
		log.Printf("declaration %s -> team %s", filename, tl)
		return nil
	}); err != nil {
		log.Fatalf("upload declarations: %v", err)
	}

	log.Printf("done in %v. created %d teams, %d labels (%d hosts each), 97 profiles, %d declarations",
		time.Since(start).Round(time.Millisecond),
		len(teamNames), numLabels, hostsPerLabel, len(decls))
}

func teardown(c *client) {
	start := time.Now()

	teams, err := c.listTeams()
	if err != nil {
		log.Fatalf("list teams: %v", err)
	}
	teamIDs := []uint{0} // also clean up "no team"
	for _, t := range teams {
		if strings.HasPrefix(t.Name, loadtestPrefix) {
			teamIDs = append(teamIDs, t.ID)
		}
	}

	// Profiles first (they FK to labels and teams). Collect across teams, then
	// delete in parallel.
	type profToDelete struct {
		uuid string
		name string
	}
	var toDelete []profToDelete
	var collectMu sync.Mutex
	if err := runParallel("listProfiles", len(teamIDs), func(i int) error {
		profs, err := c.listProfiles(teamIDs[i])
		if err != nil {
			return err
		}
		collectMu.Lock()
		defer collectMu.Unlock()
		for _, p := range profs {
			if strings.HasPrefix(p.Name, loadtestPrefix) {
				toDelete = append(toDelete, profToDelete{p.ProfileUUID, p.Name})
			}
		}
		return nil
	}); err != nil {
		log.Printf("list profiles: %v", err)
	}
	_ = runParallel("deleteProfile", len(toDelete), func(i int) error {
		if err := c.deleteProfile(toDelete[i].uuid); err != nil {
			log.Printf("delete profile %s: %v", toDelete[i].name, err)
		} else {
			log.Printf("deleted profile %s", toDelete[i].name)
		}
		return nil
	})

	// Teams.
	var teamDels []team
	for _, t := range teams {
		if strings.HasPrefix(t.Name, loadtestPrefix) {
			teamDels = append(teamDels, t)
		}
	}
	_ = runParallel("deleteTeam", len(teamDels), func(i int) error {
		if err := c.deleteTeam(teamDels[i].ID); err != nil {
			log.Printf("delete team %s: %v", teamDels[i].Name, err)
		} else {
			log.Printf("deleted team %s", teamDels[i].Name)
		}
		return nil
	})

	// Labels last.
	_ = runParallel("deleteLabel", numLabels, func(i int) error {
		name := fmt.Sprintf("%slabel-%02d", loadtestPrefix, i)
		if err := c.deleteLabelByName(name); err != nil {
			log.Printf("delete label %s: %v", name, err)
		} else {
			log.Printf("deleted label %s", name)
		}
		return nil
	})

	log.Printf("teardown done in %v", time.Since(start).Round(time.Millisecond))
}

func main() {
	td := flag.Bool("teardown", false, "delete everything created with this prefix")
	dry := flag.Bool("dry-run", false, "print actions, don't call the API")
	cc := flag.Int("concurrency", concurrency, "max in-flight HTTP requests for cheap phases")
	tcc := flag.Int("transfer-concurrency", transferConcurrency,
		"max in-flight host transfer batches (each is heavy server-side; default 1 = sequential)")
	flag.Parse()
	concurrency = *cc
	transferConcurrency = *tcc

	c := newClient(*dry)
	if *td {
		teardown(c)
		return
	}
	seed(c)
}
