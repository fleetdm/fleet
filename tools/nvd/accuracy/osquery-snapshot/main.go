// Command osquery-snapshot SSHes to a macOS VM, runs a set of osquery
// SELECTs against the bundled Orbit osqueryd, and merges the results into
// osquery_snapshot_macos.json — the master inventory consumed by
// tools/nvd/accuracy/testdata-reconcile.
//
// The snapshot is append-only: re-running the tool only updates version
// fields and adds new rows. Rows for products the VM no longer has are
// preserved, because the point is to accumulate a complete test corpus, not
// to mirror current VM state.
//
// Prerequisites:
//   - macOS VM reachable via SSH (default alias `fleet-testdata-vm`).
//   - Fleet's Orbit agent installed, so
//     /opt/orbit/bin/osqueryd/macos-app/stable/osquery.app/Contents/MacOS/osqueryd
//     is available.
//
// Usage:
//
//	go run ./tools/nvd/accuracy/osquery-snapshot
//	go run ./tools/nvd/accuracy/osquery-snapshot --ssh-host admin@10.211.55.3 --dry-run
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
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	ssh_knownhosts "golang.org/x/crypto/ssh/knownhosts"
)

// -----------------------------------------------------------------------------
// Snapshot schema
// -----------------------------------------------------------------------------

type snapshotFile struct {
	Metadata snapshotMetadata `json:"metadata"`
	Software []softwareRow    `json:"software"`
}

type snapshotMetadata struct {
	LastUpdated    string `json:"last_updated"`
	Platform       string `json:"platform"`
	MacOSVersion   string `json:"macos_version,omitempty"`
	OSQueryVersion string `json:"osquery_version,omitempty"`
	Notes          string `json:"notes,omitempty"`
}

type softwareRow struct {
	Name             string `json:"name"`
	Version          string `json:"version"`
	Source           string `json:"source"`
	BundleIdentifier string `json:"bundle_identifier,omitempty"`
	ExtensionID      string `json:"extension_id,omitempty"`
}

// -----------------------------------------------------------------------------
// Query definitions
// -----------------------------------------------------------------------------

// sourceQuery binds an osquery SELECT to the `fleet.Software.Source` value
// we want on every resulting row. The query must return columns named
// `name`, `version`, and optionally `bundle_identifier` / `extension_id`.
type sourceQuery struct {
	Source string
	SQL    string
}

var sourceQueries = []sourceQuery{
	{
		Source: "apps",
		// bundle_short_version aligns with Fleet's ingestion (server/service/osquery_utils/queries.go).
		SQL: `SELECT name, bundle_short_version AS version, bundle_identifier FROM apps`,
	},
	{
		Source: "homebrew_packages",
		SQL:    `SELECT name, version FROM homebrew_packages`,
	},
	{
		Source: "chrome_extensions",
		SQL:    `SELECT name, version, identifier AS extension_id FROM chrome_extensions`,
	},
	{
		Source: "firefox_addons",
		SQL:    `SELECT name, version, identifier AS extension_id FROM firefox_addons`,
	},
	{
		Source: "safari_extensions",
		SQL:    `SELECT name, version, identifier AS extension_id FROM safari_extensions`,
	},
	{
		Source: "python_packages",
		// Deduplicated via DISTINCT because pip packages can be installed per-user.
		SQL: `SELECT DISTINCT name, version FROM python_packages`,
	},
	{
		Source: "npm_packages",
		SQL:    `SELECT DISTINCT name, version FROM npm_packages`,
	},
	{
		Source: "vscode_extensions",
		SQL:    `SELECT DISTINCT name, version, uuid AS extension_id FROM vscode_extensions`,
	},
}

// metaQuery returns the macOS and osquery versions.
const metaQuery = `SELECT
    (SELECT version FROM os_version) AS macos_version,
    (SELECT version FROM osquery_info) AS osquery_version`

// -----------------------------------------------------------------------------
// Entry point
// -----------------------------------------------------------------------------

func main() {
	sshHost := flag.String("ssh-host", "fleet-testdata-vm",
		"SSH host (alias from ~/.ssh/config or user@host).")
	sshUser := flag.String("ssh-user", "admin", "SSH user (overrides ~/.ssh/config).")
	sshKey := flag.String("ssh-key", defaultSSHKey(),
		"Private key file for SSH auth (default: ~/.ssh/fleet_testdata_vm_ed25519).")
	osqueryd := flag.String("osqueryd",
		"/opt/orbit/bin/osqueryd/macos-app/stable/osquery.app/Contents/MacOS/osqueryd",
		"Path to the osqueryd binary on the VM.")
	snapshotPath := flag.String("snapshot",
		"server/vulnerabilities/nvd/testdata/accuracy/osquery_snapshot_macos.json",
		"Path to the master snapshot JSON.")
	dryRun := flag.Bool("dry-run", false,
		"Print what would be captured without writing the snapshot file.")
	flag.Parse()

	host, userOverride := parseSSHHost(*sshHost)
	if userOverride != "" {
		sshUser = &userOverride
	}

	client, err := newSSHClient(host, *sshUser, *sshKey)
	if err != nil {
		fatalf("ssh dial: %v", err)
	}
	defer client.Close()

	meta, err := fetchMetadata(client, *osqueryd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not fetch metadata: %v\n", err)
	}

	// Run each source's query; collect rows.
	rowsPerSource := make(map[string][]softwareRow, len(sourceQueries))
	for _, q := range sourceQueries {
		rows, err := runQuery(client, *osqueryd, q.SQL, q.Source)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %-20s ERROR: %v\n", q.Source, err)
			continue
		}
		rowsPerSource[q.Source] = rows
		fmt.Fprintf(os.Stderr, "  %-20s %d row(s)\n", q.Source, len(rows))
	}

	// Merge with existing snapshot.
	existing, err := loadSnapshot(*snapshotPath)
	if err != nil {
		fatalf("load existing snapshot: %v", err)
	}
	merged := mergeSnapshot(existing, rowsPerSource)
	merged.Metadata.LastUpdated = time.Now().UTC().Format(time.RFC3339)
	merged.Metadata.Platform = "macos"
	if meta.MacOSVersion != "" {
		merged.Metadata.MacOSVersion = meta.MacOSVersion
	}
	if meta.OSQueryVersion != "" {
		merged.Metadata.OSQueryVersion = meta.OSQueryVersion
	}
	if merged.Metadata.Notes == "" {
		merged.Metadata.Notes = "Captured via bundled Orbit osqueryd on the Fleet testdata macOS VM"
	}

	added, updated := diffSnapshot(existing, merged)
	fmt.Fprintf(os.Stderr, "\nmerged: %d total rows (%d added, %d version-updated)\n",
		len(merged.Software), added, updated)

	if *dryRun {
		fmt.Fprintln(os.Stderr, "--dry-run: not writing snapshot")
		return
	}

	if err := writeSnapshot(*snapshotPath, merged); err != nil {
		fatalf("write snapshot: %v", err)
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", *snapshotPath)
}

// -----------------------------------------------------------------------------
// Metadata capture
// -----------------------------------------------------------------------------

type hostMeta struct {
	MacOSVersion   string
	OSQueryVersion string
}

func fetchMetadata(client *ssh.Client, osqueryd string) (hostMeta, error) {
	var out hostMeta
	raw, err := runRaw(client, osqueryd, metaQuery)
	if err != nil {
		return out, err
	}
	var parsed []map[string]string
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return out, fmt.Errorf("decode metadata: %w", err)
	}
	if len(parsed) == 0 {
		return out, errors.New("empty metadata result")
	}
	out.MacOSVersion = parsed[0]["macos_version"]
	out.OSQueryVersion = parsed[0]["osquery_version"]
	return out, nil
}

// -----------------------------------------------------------------------------
// osquery over SSH
// -----------------------------------------------------------------------------

// runQuery executes an osquery SELECT on the VM via `osqueryd -S --json`
// and decodes the result into softwareRow values stamped with the given
// source. Columns not declared in softwareRow are silently dropped.
func runQuery(client *ssh.Client, osqueryd, sql, source string) ([]softwareRow, error) {
	raw, err := runRaw(client, osqueryd, sql)
	if err != nil {
		return nil, err
	}
	var rows []softwareRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("decode rows: %w", err)
	}
	// Stamp the source on every row and drop entries missing required fields.
	out := rows[:0]
	for _, r := range rows {
		r.Source = source
		if strings.TrimSpace(r.Name) == "" {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

// runRaw executes a single osqueryd -S --json query via SSH and returns the
// raw stdout bytes.
func runRaw(client *ssh.Client, osqueryd, sql string) ([]byte, error) {
	sess, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("new session: %w", err)
	}
	defer sess.Close()

	var outBuf, errBuf bytes.Buffer
	sess.Stdout = &outBuf
	sess.Stderr = &errBuf

	// Single-quote the query for the remote shell; use shellQuote to escape
	// any embedded single quotes (our queries don't have them, but be safe).
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

// -----------------------------------------------------------------------------
// Snapshot IO
// -----------------------------------------------------------------------------

func loadSnapshot(path string) (snapshotFile, error) {
	var s snapshotFile
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&s); err != nil {
		return s, fmt.Errorf("decode: %w", err)
	}
	return s, nil
}

func writeSnapshot(path string, s snapshotFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".osquery-snapshot-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

// -----------------------------------------------------------------------------
// Merge logic
// -----------------------------------------------------------------------------

// mergeSnapshot joins the existing snapshot's rows with the freshly captured
// rows. New rows are added; rows with the same (source, name, bundle_identifier,
// extension_id) key have their Version updated. Old rows not seen in this
// capture are preserved (append-only semantics).
func mergeSnapshot(existing snapshotFile, captured map[string][]softwareRow) snapshotFile {
	byKey := make(map[string]softwareRow, len(existing.Software))
	for _, r := range existing.Software {
		byKey[rowKey(r)] = r
	}
	for _, rows := range captured {
		for _, r := range rows {
			byKey[rowKey(r)] = r
		}
	}

	merged := existing
	merged.Software = make([]softwareRow, 0, len(byKey))
	for _, r := range byKey {
		merged.Software = append(merged.Software, r)
	}
	sort.Slice(merged.Software, func(i, j int) bool {
		a, b := merged.Software[i], merged.Software[j]
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.BundleIdentifier < b.BundleIdentifier
	})
	return merged
}

func diffSnapshot(before, after snapshotFile) (added, updated int) {
	beforeByKey := make(map[string]softwareRow, len(before.Software))
	for _, r := range before.Software {
		beforeByKey[rowKey(r)] = r
	}
	for _, r := range after.Software {
		prev, ok := beforeByKey[rowKey(r)]
		if !ok {
			added++
			continue
		}
		if prev.Version != r.Version {
			updated++
		}
	}
	return added, updated
}

func rowKey(r softwareRow) string {
	return r.Source + "\x00" + r.Name + "\x00" + r.BundleIdentifier + "\x00" + r.ExtensionID
}

// -----------------------------------------------------------------------------
// SSH setup (mirrors tools/nvd/accuracy/recipe-generator)
// -----------------------------------------------------------------------------

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
	// Resolve SSH config alias if present.
	resolvedHost := resolveSSHHost(host)
	addr := net.JoinHostPort(resolvedHost, "22")
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	return client, nil
}

// resolveSSHHost does a minimal ~/.ssh/config HostName lookup for the given
// alias. Unknown aliases are returned unchanged (the caller gets an ordinary
// DNS resolution failure if the name isn't resolvable).
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
			// Only support exact-match single-alias blocks for our needs.
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
// Helpers
// -----------------------------------------------------------------------------

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func fatalf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", a...)
	os.Exit(1)
}
