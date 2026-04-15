// Command testdata-reconcile joins the CPE candidates list with live osquery
// state from the macOS VM and writes the accuracy golden files that
// tools/nvd/accuracy/cpe-accuracy consumes.
//
// For each candidate (vendor, product) pair, the tool queries the relevant
// osquery tables on the VM and emits a test case for every matching row.
// Candidates with no matching row land in missing_products.json, which
// tools/nvd/accuracy/recipe-generator can then consume to install them.
//
// This replaces the earlier osquery-snapshot + reconcile split: the raw
// snapshot is no longer persisted because we only need rows that match a
// candidate. VM queries are fast (~15s total), so they run on every refresh.
//
// Usage:
//
//	go run ./tools/nvd/accuracy/testdata-reconcile
//	go run ./tools/nvd/accuracy/testdata-reconcile --dry-run
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	ssh_knownhosts "golang.org/x/crypto/ssh/knownhosts"
)

// -----------------------------------------------------------------------------
// Candidate input (subset — matches tools/nvd/accuracy/cpe-candidates output)
// -----------------------------------------------------------------------------

type candidatesFile struct {
	Metadata   candidatesMetadata `json:"metadata"`
	Candidates []candidate        `json:"candidates"`
}

type candidatesMetadata struct {
	GeneratedAt  string   `json:"generated_at,omitempty"`
	Year         int      `json:"year"`
	Severity     string   `json:"severity"`
	MinCVSSScore float64  `json:"min_cvss_score,omitempty"`
	SourceFiles  []string `json:"source_files,omitempty"`
	CVECount     int      `json:"cve_count,omitempty"`
}

type candidate struct {
	Vendor        string       `json:"vendor"`
	Product       string       `json:"product"`
	TargetSWHints []string     `json:"target_sw_hints"`
	RelatedCVEs   []relatedCVE `json:"related_cves"`
}

type relatedCVE struct {
	ID                  string  `json:"id"`
	CVSSScore           float64 `json:"cvss_score"`
	VersionEndExcluding string  `json:"version_end_excluding,omitempty"`
	VersionEndIncluding string  `json:"version_end_including,omitempty"`
	PublishedDate       string  `json:"published_date,omitempty"`
}

// -----------------------------------------------------------------------------
// Golden file schema (matches tools/nvd/accuracy/cpe-accuracy)
// -----------------------------------------------------------------------------

type accuracySuite struct {
	Metadata  accuracyMetadata `json:"metadata"`
	TestCases []accuracyCase   `json:"test_cases"`
}

type accuracyMetadata struct {
	GeneratedAt string `json:"generated_at"`
	Platform    string `json:"platform"`
	Year        int    `json:"year"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

type accuracyCase struct {
	ID          string        `json:"id"`
	Software    softwareInput `json:"software"`
	ExpectedCPE string        `json:"expected_cpe"`
	SourceCVE   string        `json:"source_cve,omitempty"`
	CVSSScore   float64       `json:"cvss_score,omitempty"`
	Tags        []string      `json:"tags,omitempty"`
	Notes       string        `json:"notes,omitempty"`
}

type softwareInput struct {
	Name             string `json:"name"`
	Version          string `json:"version"`
	Source           string `json:"source"`
	Vendor           string `json:"vendor,omitempty"`
	BundleIdentifier string `json:"bundle_identifier,omitempty"`
	ExtensionID      string `json:"extension_id,omitempty"`
}

// -----------------------------------------------------------------------------
// missing_products.json (input to recipe-generator)
// -----------------------------------------------------------------------------

type missingProductsFile struct {
	Metadata   candidatesMetadata `json:"metadata"`
	Candidates []candidate        `json:"candidates"`
}

// -----------------------------------------------------------------------------
// Entry point
// -----------------------------------------------------------------------------

const goldenDir = "server/vulnerabilities/nvd/testdata/accuracy"

func main() {
	candidatesPath := flag.String("candidates",
		filepath.Join(goldenDir, "cpe_candidates_2026_critical.json"),
		"Path to the cpe-candidates output file.")
	outputDir := flag.String("output-dir", goldenDir,
		"Directory to write golden files and missing_products.json into.")
	sshHost := flag.String("ssh-host", "fleet-testdata-vm",
		"SSH host alias (from ~/.ssh/config) or user@host.")
	sshUser := flag.String("ssh-user", "admin", "SSH user.")
	sshKey := flag.String("ssh-key", defaultSSHKey(),
		"Private key file (default ~/.ssh/fleet_testdata_vm_ed25519).")
	osqueryd := flag.String("osqueryd",
		"/opt/orbit/bin/osqueryd/macos-app/stable/osquery.app/Contents/MacOS/osqueryd",
		"Path to osqueryd on the VM.")
	dryRun := flag.Bool("dry-run", false,
		"Print what would be written without touching files.")
	flag.Parse()

	cands, err := loadCandidates(*candidatesPath)
	if err != nil {
		fatalf("load candidates: %v", err)
	}
	fmt.Fprintf(os.Stderr, "loaded %d candidates from %s\n", len(cands.Candidates), *candidatesPath)

	host, userOverride := parseSSHHost(*sshHost)
	if userOverride != "" {
		sshUser = &userOverride
	}
	client, err := newSSHClient(host, *sshUser, *sshKey)
	if err != nil {
		fatalf("ssh dial: %v", err)
	}
	defer client.Close()

	inventory, err := captureInventory(client, *osqueryd)
	if err != nil {
		fatalf("capture inventory: %v", err)
	}
	for _, src := range sourceOrder {
		fmt.Fprintf(os.Stderr, "  %-20s %d row(s)\n", src, len(inventory[src]))
	}

	m := newMatcher(inventory)
	cases, missing := matchAll(cands, m)
	byCategory := groupByCategory(cases)

	fmt.Fprintf(os.Stderr, "\nresolved %d test case(s); %d candidate(s) missing from VM\n",
		len(cases), len(missing))
	for _, cat := range categoryOrder {
		fmt.Fprintf(os.Stderr, "  cves_%d_%s_%s.json — %d case(s)\n",
			cands.Metadata.Year, strings.ToLower(cands.Metadata.Severity), cat, len(byCategory[cat]))
	}

	if *dryRun {
		fmt.Fprintln(os.Stderr, "--dry-run: not writing files")
		return
	}

	// Write one golden file per category (including empty ones, so downstream
	// tooling can rely on their presence).
	for _, cat := range categoryOrder {
		path := filepath.Join(*outputDir,
			fmt.Sprintf("cves_%d_%s_%s.json",
				cands.Metadata.Year, strings.ToLower(cands.Metadata.Severity), cat))
		suite := accuracySuite{
			Metadata: accuracyMetadata{
				GeneratedAt: time.Now().UTC().Format(time.RFC3339),
				Platform:    "macos",
				Year:        cands.Metadata.Year,
				Severity:    cands.Metadata.Severity,
				Description: fmt.Sprintf("%s test cases generated by testdata-reconcile from %d-%s candidates matched against live VM osquery state",
					cat, cands.Metadata.Year, strings.ToLower(cands.Metadata.Severity)),
			},
			TestCases: byCategory[cat],
		}
		if err := writeJSON(path, suite); err != nil {
			fatalf("write %s: %v", path, err)
		}
	}

	// missing_products.json has the same shape as cpe-candidates' output so
	// recipe-generator can consume it via --input without modification.
	missingPath := filepath.Join(*outputDir, "missing_products.json")
	mp := missingProductsFile{
		Metadata:   cands.Metadata,
		Candidates: missing,
	}
	if err := writeJSON(missingPath, mp); err != nil {
		fatalf("write %s: %v", missingPath, err)
	}
	fmt.Fprintf(os.Stderr, "wrote %s (%d entries)\n", missingPath, len(missing))
}

// -----------------------------------------------------------------------------
// Candidate loading
// -----------------------------------------------------------------------------

func loadCandidates(path string) (candidatesFile, error) {
	var c candidatesFile
	f, err := os.Open(path)
	if err != nil {
		return c, err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&c); err != nil {
		return c, fmt.Errorf("decode: %w", err)
	}
	if c.Metadata.Year == 0 || c.Metadata.Severity == "" {
		return c, errors.New("candidates file missing year/severity metadata")
	}
	return c, nil
}

// -----------------------------------------------------------------------------
// osquery inventory (one-shot capture)
// -----------------------------------------------------------------------------

// sourceOrder drives deterministic reporting.
var sourceOrder = []string{
	"apps",
	"homebrew_packages",
	"chrome_extensions",
	"firefox_addons",
	"safari_extensions",
	"python_packages",
	"npm_packages",
	"vscode_extensions",
}

var osquerySQLBySource = map[string]string{
	"apps":              `SELECT name, bundle_short_version AS version, bundle_identifier FROM apps`,
	"homebrew_packages": `SELECT name, version FROM homebrew_packages`,
	"chrome_extensions": `SELECT name, version, identifier AS extension_id FROM chrome_extensions`,
	"firefox_addons":    `SELECT name, version, identifier AS extension_id FROM firefox_addons`,
	"safari_extensions": `SELECT name, version, identifier AS extension_id FROM safari_extensions`,
	"python_packages":   `SELECT DISTINCT name, version FROM python_packages`,
	"npm_packages":      `SELECT DISTINCT name, version FROM npm_packages`,
	"vscode_extensions": `SELECT DISTINCT name, version, uuid AS extension_id FROM vscode_extensions`,
}

// captureInventory runs every source query on the VM and returns the rows
// keyed by source name. Rows are stamped with `Source` before being returned.
func captureInventory(client *ssh.Client, osqueryd string) (map[string][]softwareInput, error) {
	out := make(map[string][]softwareInput, len(sourceOrder))
	for _, src := range sourceOrder {
		raw, err := runRaw(client, osqueryd, osquerySQLBySource[src])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", src, err)
		}
		var rows []softwareInput
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &rows); err != nil {
				return nil, fmt.Errorf("%s: decode: %w", src, err)
			}
		}
		for i := range rows {
			rows[i].Source = src
		}
		out[src] = rows
	}
	return out, nil
}

// -----------------------------------------------------------------------------
// Candidate → inventory matching
// -----------------------------------------------------------------------------

type matcher struct {
	inventory map[string][]softwareInput
}

func newMatcher(inv map[string][]softwareInput) *matcher {
	return &matcher{inventory: inv}
}

// findMatches returns every inventory row that could plausibly represent the
// given candidate. Rules:
//
//  1. homebrew_packages.name matches `^<product>(@\d+)?$` (case-insensitive).
//     Catches `openssl`, `openssl@3`, `python@3.12`, etc.
//  2. apps.bundle_identifier ends in `.<product>` OR contains `<vendor>.<product>`
//     OR apps.name (minus `.app`) equals `<product>` (case-insensitive).
//  3. python_packages.name equals `<product>` (case-insensitive).
//  4. npm_packages.name equals `<product>` (case-insensitive).
//  5. Extension sources: name contains `<product>` OR extension_id contains
//     `<vendor>.<product>` OR extension_id contains `<product>`.
//
// The goal is a forgiving first pass. Inventory rows that snuck past (false
// positives) can be pruned manually in the generated golden files.
func (m *matcher) findMatches(c candidate) []softwareInput {
	product := strings.ToLower(c.Product)
	vendor := strings.ToLower(c.Vendor)
	var hits []softwareInput

	brewPattern := regexp.MustCompile(`^` + regexp.QuoteMeta(product) + `(@[0-9.]+)?$`)
	for _, r := range m.inventory["homebrew_packages"] {
		if brewPattern.MatchString(strings.ToLower(r.Name)) {
			hits = append(hits, r)
		}
	}

	vendorDotProduct := vendor + "." + product
	for _, r := range m.inventory["apps"] {
		bid := strings.ToLower(r.BundleIdentifier)
		nameNoExt := strings.ToLower(strings.TrimSuffix(r.Name, ".app"))
		switch {
		case bid != "" && strings.HasSuffix(bid, "."+product):
			hits = append(hits, r)
		case bid != "" && strings.Contains(bid, vendorDotProduct):
			hits = append(hits, r)
		case nameNoExt == product:
			hits = append(hits, r)
		}
	}

	for _, r := range m.inventory["python_packages"] {
		if strings.EqualFold(r.Name, c.Product) {
			hits = append(hits, r)
		}
	}

	for _, r := range m.inventory["npm_packages"] {
		if strings.EqualFold(r.Name, c.Product) {
			hits = append(hits, r)
		}
	}

	for _, src := range []string{"chrome_extensions", "firefox_addons", "safari_extensions", "vscode_extensions"} {
		for _, r := range m.inventory[src] {
			lowerName := strings.ToLower(r.Name)
			lowerID := strings.ToLower(r.ExtensionID)
			switch {
			case strings.Contains(lowerName, product):
				hits = append(hits, r)
			case lowerID != "" && strings.Contains(lowerID, vendorDotProduct):
				hits = append(hits, r)
			case lowerID != "" && strings.Contains(lowerID, product):
				hits = append(hits, r)
			}
		}
	}

	return hits
}

// matchAll applies the matcher to every candidate and returns (cases, missing).
// "missing" preserves the candidate struct verbatim so missing_products.json
// is directly consumable by recipe-generator --input.
func matchAll(cands candidatesFile, m *matcher) ([]accuracyCase, []candidate) {
	var cases []accuracyCase
	var missing []candidate
	seenID := make(map[string]int)

	for _, c := range cands.Candidates {
		hits := m.findMatches(c)
		if len(hits) == 0 {
			missing = append(missing, c)
			continue
		}
		topCVE, topScore := pickTopCVE(c.RelatedCVEs)
		for _, sw := range hits {
			baseID := fmt.Sprintf("%s-%s-%s", c.Vendor, c.Product, sw.Source)
			id := baseID
			seenID[baseID]++
			if n := seenID[baseID]; n > 1 {
				id = fmt.Sprintf("%s-%d", baseID, n)
			}
			cases = append(cases, accuracyCase{
				ID:          id,
				Software:    sw,
				ExpectedCPE: buildExpectedCPE(c.Vendor, c.Product, sw.Version),
				SourceCVE:   topCVE,
				CVSSScore:   topScore,
				Tags:        []string{"auto-reconciled"},
			})
		}
	}

	sort.Slice(cases, func(i, j int) bool { return cases[i].ID < cases[j].ID })
	sort.Slice(missing, func(i, j int) bool {
		return missing[i].Vendor+"/"+missing[i].Product < missing[j].Vendor+"/"+missing[j].Product
	})
	return cases, missing
}

func pickTopCVE(cves []relatedCVE) (string, float64) {
	var top relatedCVE
	for _, c := range cves {
		if c.CVSSScore > top.CVSSScore {
			top = c
		}
	}
	return top.ID, top.CVSSScore
}

// buildExpectedCPE constructs the CPE we expect Fleet's CPE generator to
// produce for this (vendor, product, version) on macOS.
func buildExpectedCPE(vendor, product, version string) string {
	esc := func(s string) string {
		s = strings.ReplaceAll(s, ":", `\:`)
		s = strings.ReplaceAll(s, " ", "_")
		return strings.ToLower(s)
	}
	v := esc(version)
	if v == "" {
		v = "*"
	}
	return fmt.Sprintf("cpe:2.3:a:%s:%s:%s:*:*:*:*:macos:*:*", esc(vendor), esc(product), v)
}

// -----------------------------------------------------------------------------
// Category routing
// -----------------------------------------------------------------------------

var categoryOrder = []string{"apps", "homebrew", "extensions", "language_pkgs"}

var sourceToCategory = map[string]string{
	"apps":              "apps",
	"homebrew_packages": "homebrew",
	"chrome_extensions": "extensions",
	"firefox_addons":    "extensions",
	"safari_extensions": "extensions",
	"vscode_extensions": "extensions",
	"python_packages":   "language_pkgs",
	"npm_packages":      "language_pkgs",
}

func groupByCategory(cases []accuracyCase) map[string][]accuracyCase {
	out := make(map[string][]accuracyCase, len(categoryOrder))
	for _, cat := range categoryOrder {
		out[cat] = nil
	}
	for _, c := range cases {
		cat := sourceToCategory[c.Software.Source]
		if cat == "" {
			cat = "apps"
		}
		out[cat] = append(out[cat], c)
	}
	return out
}

// -----------------------------------------------------------------------------
// SSH + osquery (copied from the retired osquery-snapshot)
// -----------------------------------------------------------------------------

func runRaw(client *ssh.Client, osqueryd, sql string) ([]byte, error) {
	sess, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("new session: %w", err)
	}
	defer sess.Close()

	var outBuf, errBuf bytes.Buffer
	sess.Stdout = &outBuf
	sess.Stderr = &errBuf

	cmd := fmt.Sprintf("%s -S --json %s", shellQuote(osqueryd), shellQuote(sql))
	if err := sess.Run(cmd); err != nil {
		stderr := strings.TrimSpace(errBuf.String())
		if stderr != "" {
			return nil, fmt.Errorf("osqueryd: %w: %s", err, stderr)
		}
		return nil, fmt.Errorf("osqueryd: %w", err)
	}
	return bytes.TrimSpace(outBuf.Bytes()), nil
}

func defaultSSHKey() string {
	u, err := user.Current()
	if err != nil {
		return ""
	}
	return filepath.Join(u.HomeDir, ".ssh", "fleet_testdata_vm_ed25519")
}

func parseSSHHost(spec string) (host, user string) {
	if u, h, ok := strings.Cut(spec, "@"); ok {
		return h, u
	}
	return spec, ""
}

func newSSHClient(host, sshUser, keyPath string) (*ssh.Client, error) {
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read ssh key %s: %w", keyPath, err)
	}
	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("parse ssh key: %w", err)
	}

	hostKeyCallback, err := loadKnownHostsCallback()
	if err != nil {
		hostKeyCallback = ssh.InsecureIgnoreHostKey() //nolint:gosec // test infra
	}

	cfg := &ssh.ClientConfig{
		User:            sshUser,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: hostKeyCallback,
		Timeout:         10 * time.Second,
	}
	resolvedHost := resolveSSHHost(host)
	addr := net.JoinHostPort(resolvedHost, "22")
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	return client, nil
}

func resolveSSHHost(alias string) string {
	u, err := user.Current()
	if err != nil {
		return alias
	}
	data, err := os.ReadFile(filepath.Join(u.HomeDir, ".ssh", "config"))
	if err != nil {
		return alias
	}
	lines := strings.Split(string(data), "\n")
	inBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		low := strings.ToLower(trimmed)
		if strings.HasPrefix(low, "host ") {
			inBlock = strings.TrimSpace(trimmed[len("Host "):]) == alias
			continue
		}
		if inBlock && strings.HasPrefix(low, "hostname ") {
			return strings.TrimSpace(trimmed[len("HostName "):])
		}
	}
	return alias
}

func loadKnownHostsCallback() (ssh.HostKeyCallback, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	return ssh_knownhosts.New(filepath.Join(u.HomeDir, ".ssh", "known_hosts"))
}

// -----------------------------------------------------------------------------
// JSON helpers
// -----------------------------------------------------------------------------

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".reconcile-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func fatalf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", a...)
	os.Exit(1)
}
