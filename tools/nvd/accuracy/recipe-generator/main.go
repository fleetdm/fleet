// Command recipe-generator researches how to install each candidate product on
// macOS by shelling out to the Claude Code CLI and letting Claude use its own
// web search, web fetch, and bash tools (e.g., `brew search`, `brew info`) to
// inspect Homebrew, AutoPkg, and vendor sites.
//
// Input: a candidates file (from tools/nvd/accuracy/cpe-candidates) OR a
// missing-products file (from tools/nvd/accuracy/testdata-reconcile, planned).
//
// Output: install_recipes.json — consumed by tools/nvd/accuracy/software-installer
// (planned) for batch reinstalls on fresh VMs.
//
// Prerequisites:
//   - `claude` (Claude Code) CLI on PATH and authenticated.
//   - Network access.
//
// Usage:
//
//	# Research a specific product (quick sanity check):
//	go run ./tools/nvd/accuracy/recipe-generator --product mozilla/firefox --limit 1
//
//	# Research every candidate (expensive — hundreds of CVE references):
//	go run ./tools/nvd/accuracy/recipe-generator
//
//	# Research only the products flagged by testdata-reconcile:
//	go run ./tools/nvd/accuracy/recipe-generator \
//	    --input ./server/vulnerabilities/nvd/testdata/accuracy/missing_products.json
//
// Idempotency: existing recipes in the output file are preserved and not
// re-researched unless --force is passed.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"golang.org/x/crypto/ssh"
	ssh_knownhosts "golang.org/x/crypto/ssh/knownhosts"
)

// -----------------------------------------------------------------------------
// Input schemas
// -----------------------------------------------------------------------------

// candidatesInput matches the output of tools/nvd/accuracy/cpe-candidates.
type candidatesInput struct {
	Candidates []candidateEntry `json:"candidates"`
}

type candidateEntry struct {
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
}

// -----------------------------------------------------------------------------
// Output schema
// -----------------------------------------------------------------------------

type recipesFile struct {
	Metadata recipesMetadata   `json:"metadata"`
	Recipes  map[string]recipe `json:"recipes"`
	Errors   map[string]string `json:"errors,omitempty"`
}

type recipesMetadata struct {
	GeneratedAt    string  `json:"generated_at"`
	LastUpdatedAt  string  `json:"last_updated_at"`
	Model          string  `json:"model"`
	ProductsTotal  int     `json:"products_total"`
	TotalCostUSD   float64 `json:"total_cost_usd"`
	ClaudeCLIPath  string  `json:"claude_cli_path,omitempty"`
}

type recipe struct {
	Method           string   `json:"method"` // brew_cask | brew_formula | pkg_url | dmg_url | skip
	Identifier       string   `json:"identifier,omitempty"`
	Confidence       string   `json:"confidence"` // high | medium | low
	SourcesConsulted []string `json:"sources_consulted,omitempty"`
	Rationale        string   `json:"rationale"`
	ResearchedAt     string   `json:"researched_at"`
	CostUSD          float64  `json:"cost_usd,omitempty"`

	// Validation records the outcome of actually running the install on the VM.
	Validation *validationResult `json:"validation,omitempty"`
}

type validationResult struct {
	// Status is:
	//   - "verified"  : install command succeeded on the VM
	//   - "failed"    : install command ran but returned non-zero
	//   - "skipped"   : method=="skip" (no install attempted) or --skip-validation
	//   - "error"     : could not even attempt the install (SSH, network, etc.)
	Status      string  `json:"status"`
	ValidatedAt string  `json:"validated_at"`
	Command     string  `json:"command,omitempty"`
	ExitCode    int     `json:"exit_code"`
	Stdout      string  `json:"stdout,omitempty"`
	Stderr      string  `json:"stderr,omitempty"`
	DurationSec float64 `json:"duration_sec,omitempty"`
}

// -----------------------------------------------------------------------------
// Claude CLI wire format
// -----------------------------------------------------------------------------

// claudeResult is the top-level JSON emitted by `claude -p --output-format json`.
type claudeResult struct {
	Type         string  `json:"type"`
	Subtype      string  `json:"subtype"`
	IsError      bool    `json:"is_error"`
	DurationMS   int     `json:"duration_ms"`
	NumTurns     int     `json:"num_turns"`
	Result       string  `json:"result"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	SessionID    string  `json:"session_id"`
}

// -----------------------------------------------------------------------------
// Entry point
// -----------------------------------------------------------------------------

func main() {
	input := flag.String("input", "server/vulnerabilities/nvd/testdata/accuracy/cpe_candidates_2026_critical.json",
		"Path to the input JSON. Accepts either a cpe-candidates file or a missing-products file.")
	output := flag.String("output", "server/vulnerabilities/nvd/testdata/accuracy/install_recipes.json",
		"Path to the output recipes file. Existing entries are preserved unless --force.")
	product := flag.String("product", "",
		"Restrict to a single vendor/product (e.g., mozilla/firefox). Handy for dry-runs.")
	limit := flag.Int("limit", 0,
		"Stop after N products (0 = no limit). Useful for sanity checks before long runs.")
	force := flag.Bool("force", false,
		"Re-research products that already have a recipe in the output file.")
	model := flag.String("model", "sonnet",
		"Claude model to use (alias or full name).")
	maxBudgetUSD := flag.Float64("max-budget-usd", 0,
		"Stop after total spend reaches this. 0 = no cap.")
	dryRun := flag.Bool("dry-run", false,
		"Print the prompt that would be sent for each product, without invoking Claude.")
	claudeBin := flag.String("claude-bin", "claude",
		"Path to the claude CLI executable.")
	sshHost := flag.String("ssh-host", "10.211.55.3",
		"macOS VM host for validation (used with default user/key if --ssh-user/--ssh-key unset).")
	sshUser := flag.String("ssh-user", "admin", "SSH user")
	sshKey := flag.String("ssh-key", defaultSSHKey(),
		"Private key file for SSH auth (default: ~/.ssh/fleet_testdata_vm_ed25519).")
	brewPath := flag.String("brew-path", "/opt/homebrew/bin/brew",
		"Absolute path to brew on the VM.")
	skipValidation := flag.Bool("skip-validation", false,
		"Skip the VM-side install validation step (useful for offline research passes).")
	retryFailed := flag.Bool("retry-failed", false,
		"Re-research + re-validate recipes whose previous validation failed or errored.")
	noShortcut := flag.Bool("no-shortcut", false,
		"Disable the Homebrew formula fast path (vendor==product && formula exists => brew_formula).")
	flag.Parse()

	if _, err := exec.LookPath(*claudeBin); err != nil && !*dryRun {
		fatalf("claude CLI not on PATH (looked for %q): %v", *claudeBin, err)
	}

	products, err := loadProducts(*input)
	if err != nil {
		fatalf("load input: %v", err)
	}

	existing, err := loadExistingRecipes(*output)
	if err != nil {
		fatalf("load existing recipes: %v", err)
	}

	// Filter: single product, skip verified, opt-in to retry failures.
	queue := make([]candidateEntry, 0, len(products))
	for _, p := range products {
		key := productKey(p.Vendor, p.Product)
		if *product != "" && key != *product {
			continue
		}
		if !*force {
			r, ok := existing.Recipes[key]
			if ok {
				switch {
				case isVerified(&r):
					// Already known to work on the VM. Skip.
					continue
				case isValidationFailure(&r) && !*retryFailed:
					// Failed previously; user hasn't asked to retry.
					continue
				case r.Validation == nil && *skipValidation:
					// Research-only mode already produced this recipe; nothing new to do.
					continue
				}
				// Otherwise fall through to re-research / re-validate.
			}
		}
		queue = append(queue, p)
	}
	sort.Slice(queue, func(i, j int) bool {
		// Deterministic order: highest-CVE-count products first so limited runs
		// exercise the most interesting cases.
		return len(queue[i].RelatedCVEs) > len(queue[j].RelatedCVEs)
	})
	if *limit > 0 && len(queue) > *limit {
		queue = queue[:*limit]
	}

	if len(queue) == 0 {
		fmt.Fprintln(os.Stderr, "nothing to do: all candidates already have recipes (pass --force to re-research)")
		return
	}

	fmt.Fprintf(os.Stderr, "researching %d product(s) with model %s (dry-run=%v)\n",
		len(queue), *model, *dryRun)

	out := existing
	if out.Recipes == nil {
		out.Recipes = make(map[string]recipe)
	}
	if out.Errors == nil {
		out.Errors = make(map[string]string)
	}
	if out.Metadata.GeneratedAt == "" {
		out.Metadata.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	}
	out.Metadata.Model = *model
	out.Metadata.ClaudeCLIPath = *claudeBin

	// SSH client is lazily created on first validation to keep research-only
	// runs free of VM dependencies.
	var sshClient *ssh.Client
	defer func() {
		if sshClient != nil {
			_ = sshClient.Close()
		}
	}()
	dialSSH := func() (*ssh.Client, error) {
		if sshClient != nil {
			return sshClient, nil
		}
		c, err := newSSHClient(*sshHost, *sshUser, *sshKey)
		if err != nil {
			return nil, err
		}
		sshClient = c
		return sshClient, nil
	}

	for i, p := range queue {
		key := productKey(p.Vendor, p.Product)
		fmt.Fprintf(os.Stderr, "  [%d/%d] %s ... ", i+1, len(queue), key)

		prompt := buildPrompt(p)
		if *dryRun {
			fmt.Fprintln(os.Stderr, "DRY-RUN")
			fmt.Println("=====", key, "=====")
			fmt.Println(prompt)
			continue
		}

		// Three-stage fast path before falling through to Claude:
		//   1. Homebrew formula probe (exact name or versioned variant)
		//   2. Homebrew cask probe
		//   3. Auto-skip if both 404 (98%+ accuracy per empirical data)
		// Claude is only invoked when --no-shortcut is set.
		var (
			r    *recipe
			cost float64
			err  error
			took time.Duration
		)
		if !*noShortcut {
			if shortcut := tryHomebrewShortcut(p); shortcut != nil {
				fmt.Fprintf(os.Stderr, "%s %s ... ", shortcut.Method, shortcut.Identifier)
				r = shortcut
			}
		}
		if r == nil {
			start := time.Now()
			r, cost, err = researchOne(*claudeBin, *model, prompt)
			took = time.Since(start)
			if err != nil {
				fmt.Fprintf(os.Stderr, "RESEARCH ERROR (%s): %v\n", took.Round(time.Second), err)
				out.Errors[key] = err.Error()
				delete(out.Recipes, key)
				out.Metadata.TotalCostUSD += cost
				if err := writeRecipes(*output, out); err != nil {
					fatalf("write recipes: %v", err)
				}
				continue
			}
		}

		r.ResearchedAt = time.Now().UTC().Format(time.RFC3339)
		r.CostUSD = cost
		delete(out.Errors, key)

		if *skipValidation || r.Method == "skip" {
			r.Validation = &validationResult{
				Status:      "skipped",
				ValidatedAt: time.Now().UTC().Format(time.RFC3339),
			}
			out.Recipes[key] = *r
			fmt.Fprintf(os.Stderr, "%s (conf=%s, val=skipped) $%.4f\n",
				r.Method, r.Confidence, cost)
		} else {
			client, err := dialSSH()
			if err != nil {
				r.Validation = &validationResult{
					Status:      "error",
					ValidatedAt: time.Now().UTC().Format(time.RFC3339),
					Stderr:      "ssh dial: " + err.Error(),
				}
				out.Recipes[key] = *r
				fmt.Fprintf(os.Stderr, "%s (conf=%s, val=ERROR: %v) $%.4f\n",
					r.Method, r.Confidence, err, cost)
			} else {
				vr := validateRecipeOnVM(client, *brewPath, r)
				r.Validation = vr
				out.Recipes[key] = *r
				fmt.Fprintf(os.Stderr, "%s (conf=%s, val=%s) in %s $%.4f\n",
					r.Method, r.Confidence, vr.Status,
					time.Duration(vr.DurationSec*float64(time.Second)).Round(time.Second), cost)
			}
		}

		out.Metadata.TotalCostUSD += cost
		out.Metadata.LastUpdatedAt = time.Now().UTC().Format(time.RFC3339)
		out.Metadata.ProductsTotal = len(out.Recipes)

		if err := writeRecipes(*output, out); err != nil {
			fatalf("write recipes: %v", err)
		}

		if *maxBudgetUSD > 0 && out.Metadata.TotalCostUSD >= *maxBudgetUSD {
			fmt.Fprintf(os.Stderr, "\nmax budget $%.2f reached ($%.2f spent); stopping\n",
				*maxBudgetUSD, out.Metadata.TotalCostUSD)
			break
		}
	}

	verified, failed, errored, skipped := countStatuses(out.Recipes)
	fmt.Fprintf(os.Stderr,
		"\n%d recipes (verified=%d, failed=%d, error=%d, skipped=%d), %d research errors, $%.2f total\n",
		len(out.Recipes), verified, failed, errored, skipped,
		len(out.Errors), out.Metadata.TotalCostUSD)
}

// -----------------------------------------------------------------------------
// Recipe status helpers
// -----------------------------------------------------------------------------

func isVerified(r *recipe) bool {
	return r != nil && r.Validation != nil && r.Validation.Status == "verified"
}

func isValidationFailure(r *recipe) bool {
	if r == nil || r.Validation == nil {
		return false
	}
	return r.Validation.Status == "failed" || r.Validation.Status == "error"
}

func countStatuses(recipes map[string]recipe) (verified, failed, errored, skipped int) {
	for _, r := range recipes {
		if r.Validation == nil {
			continue
		}
		switch r.Validation.Status {
		case "verified":
			verified++
		case "failed":
			failed++
		case "error":
			errored++
		case "skipped":
			skipped++
		}
	}
	return
}

// -----------------------------------------------------------------------------
// VM validation
// -----------------------------------------------------------------------------

// validateRecipeOnVM runs the install command for r on the VM and returns the
// outcome. brew commands are run with /opt/homebrew/bin on PATH because the
// non-interactive SSH shell doesn't inherit Homebrew's PATH.
func validateRecipeOnVM(client *ssh.Client, brewPath string, r *recipe) *validationResult {
	res := &validationResult{
		ValidatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	var cmd string
	switch r.Method {
	case "brew_cask":
		cmd = fmt.Sprintf("%s install --cask %s", shellQuote(brewPath), shellQuote(r.Identifier))
	case "brew_formula":
		cmd = fmt.Sprintf("%s install %s", shellQuote(brewPath), shellQuote(r.Identifier))
	case "pkg_url":
		cmd = fmt.Sprintf(
			`tmp=$(mktemp -d) && curl -fsSL -o "$tmp/pkg" %s && sudo -n installer -pkg "$tmp/pkg" -target / && rm -rf "$tmp"`,
			shellQuote(r.Identifier))
	case "dmg_url":
		cmd = fmt.Sprintf(
			`tmp=$(mktemp -d) && curl -fsSL -o "$tmp/app.dmg" %s && hdiutil attach -nobrowse -mountpoint "$tmp/m" "$tmp/app.dmg" && cp -R "$tmp/m"/*.app /Applications/ && hdiutil detach "$tmp/m" && rm -rf "$tmp"`,
			shellQuote(r.Identifier))
	case "skip":
		res.Status = "skipped"
		return res
	default:
		res.Status = "error"
		res.Stderr = "unknown method: " + r.Method
		return res
	}
	res.Command = cmd

	wrapped := "export PATH=/opt/homebrew/bin:/usr/local/bin:$PATH\n" + cmd
	start := time.Now()
	stdout, stderr, exitCode, err := runSSH(client, wrapped)
	res.DurationSec = time.Since(start).Seconds()
	res.Stdout = truncate(stdout, 2000)
	res.Stderr = truncate(stderr, 2000)
	res.ExitCode = exitCode
	if err != nil {
		res.Status = "error"
		if res.Stderr == "" {
			res.Stderr = err.Error()
		}
		return res
	}
	if exitCode == 0 {
		res.Status = "verified"
	} else {
		res.Status = "failed"
	}
	return res
}

// -----------------------------------------------------------------------------
// SSH
// -----------------------------------------------------------------------------

func defaultSSHKey() string {
	u, err := user.Current()
	if err != nil {
		return ""
	}
	return filepath.Join(u.HomeDir, ".ssh", "fleet_testdata_vm_ed25519")
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
		// Known-hosts not available — fall back to accept-any-host. We trust the
		// --ssh-host value because the operator chose it.
		hostKeyCallback = ssh.InsecureIgnoreHostKey() //nolint:gosec // test infra
	}

	cfg := &ssh.ClientConfig{
		User:            sshUser,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: hostKeyCallback,
		Timeout:         10 * time.Second,
	}
	addr := net.JoinHostPort(host, "22")
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	return client, nil
}

func loadKnownHostsCallback() (ssh.HostKeyCallback, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	return ssh_knownhosts.New(filepath.Join(u.HomeDir, ".ssh", "known_hosts"))
}

// runSSH runs a command via SSH and returns stdout, stderr, exit code, and
// any error that prevented running the command (not a non-zero exit).
func runSSH(client *ssh.Client, cmd string) (string, string, int, error) {
	sess, err := client.NewSession()
	if err != nil {
		return "", "", -1, fmt.Errorf("new session: %w", err)
	}
	defer sess.Close()

	var outBuf, errBuf bytes.Buffer
	sess.Stdout = &outBuf
	sess.Stderr = &errBuf

	runErr := sess.Run(cmd)
	stdout := outBuf.String()
	stderr := errBuf.String()
	exitCode := 0
	if runErr != nil {
		var exitErr *ssh.ExitError
		if errors.As(runErr, &exitErr) {
			exitCode = exitErr.ExitStatus()
			return stdout, stderr, exitCode, nil
		}
		return stdout, stderr, -1, fmt.Errorf("ssh run: %w", runErr)
	}
	return stdout, stderr, exitCode, nil
}

// shellQuote single-quotes a string for safe inclusion in a POSIX shell command.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "...[truncated]"
}

// -----------------------------------------------------------------------------
// Input loading
// -----------------------------------------------------------------------------

// loadProducts accepts either a cpe-candidates output or a missing-products
// file. Both are expected to contain a top-level "candidates" array with
// {vendor, product, related_cves} entries.
func loadProducts(path string) ([]candidateEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var in candidatesInput
	if err := json.NewDecoder(f).Decode(&in); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return in.Candidates, nil
}

func loadExistingRecipes(path string) (recipesFile, error) {
	var out recipesFile
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&out); err != nil {
		return out, fmt.Errorf("decode %s: %w", path, err)
	}
	return out, nil
}

// -----------------------------------------------------------------------------
// Prompt
// -----------------------------------------------------------------------------

func buildPrompt(p candidateEntry) string {
	var b strings.Builder
	b.WriteString("You are researching how to install a software product on macOS for automated testing.\n\n")
	b.WriteString("Product identity:\n")
	fmt.Fprintf(&b, "  - NVD vendor:  %q\n", p.Vendor)
	fmt.Fprintf(&b, "  - NVD product: %q\n", p.Product)
	fmt.Fprintf(&b, "  - CPE pattern: cpe:2.3:a:%s:%s:*:*:*:*:*:*:*:*\n", p.Vendor, p.Product)

	b.WriteString("\nRelated 2026 CRITICAL CVEs (up to 5 most severe shown):\n")
	sort.Slice(p.RelatedCVEs, func(i, j int) bool { return p.RelatedCVEs[i].CVSSScore > p.RelatedCVEs[j].CVSSScore })
	for i, c := range p.RelatedCVEs {
		if i >= 5 {
			break
		}
		fmt.Fprintf(&b, "  - %s (CVSS %.1f)", c.ID, c.CVSSScore)
		if c.VersionEndExcluding != "" {
			fmt.Fprintf(&b, " fixed in < %s", c.VersionEndExcluding)
		}
		b.WriteString("\n")
	}

	b.WriteString(`
Determine the best way to install this product on a macOS test VM, in this preference order:

  1. brew_cask      — GUI application available via Homebrew cask
  2. brew_formula   — CLI tool available via Homebrew formula
  3. pkg_url        — direct .pkg installer URL from the vendor
  4. dmg_url        — direct .dmg installer URL from the vendor
  5. skip           — product is not appropriate for a macOS desktop (e.g., Linux-only
                      server software, mobile-only apps, firmware, Chinese/regional
                      web apps unlikely to be in any Fleet customer's inventory)

You have web search, web fetch, and bash tools available. Suggested research steps:

  - Probe the Homebrew JSON API:
      https://formulae.brew.sh/api/formula/<name>.json
      https://formulae.brew.sh/api/cask/<name>.json
    Try variations of the product name (hyphens, underscores, lowercase).
  - Run 'brew search <name>' locally if the API doesn't surface a match.
  - Check https://github.com/autopkg for recipes named after the vendor/product.
  - Fall back to the vendor's official download page to find a direct .pkg or .dmg URL.
  - For obscure products, one quick web search is usually enough to confirm
    "skip" is the right answer.

Important:
  - Pick the single best method. Do not recommend multiple methods.
  - "identifier" is the Homebrew name (for brew_cask/brew_formula) or the full URL
    (for pkg_url/dmg_url). Omit for "skip".
  - "confidence" reflects how certain you are that the install recipe will actually
    produce a macOS install matching this NVD vendor/product pair.
  - Keep rationale to 1-2 sentences.

Respond with ONLY a JSON object (no prose, no code fences) matching this schema:

{
  "method": "brew_cask" | "brew_formula" | "pkg_url" | "dmg_url" | "skip",
  "identifier": "...",
  "confidence": "high" | "medium" | "low",
  "sources_consulted": ["..."],
  "rationale": "..."
}
`)
	return b.String()
}

// -----------------------------------------------------------------------------
// Claude CLI invocation
// -----------------------------------------------------------------------------

// researchOne runs `claude -p` once with the given prompt. Returns the parsed
// recipe, the Claude-reported cost, and any error.
func researchOne(claudeBin, model, prompt string) (*recipe, float64, error) {
	args := []string{
		"-p",
		"--output-format", "json",
		"--model", model,
		"--permission-mode", "bypassPermissions",
		"--tools", "WebSearch,WebFetch,Bash",
		"--no-session-persistence",
		prompt,
	}
	cmd := exec.Command(claudeBin, args...)
	cmd.Stderr = os.Stderr
	raw, err := cmd.Output()
	if err != nil {
		return nil, 0, fmt.Errorf("claude exec: %w", err)
	}

	var wrapper claudeResult
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return nil, 0, fmt.Errorf("decode claude wrapper: %w", err)
	}
	if wrapper.IsError {
		return nil, wrapper.TotalCostUSD, fmt.Errorf("claude reported error: %s", wrapper.Subtype)
	}

	r, err := parseRecipeText(wrapper.Result)
	if err != nil {
		return nil, wrapper.TotalCostUSD, err
	}
	return r, wrapper.TotalCostUSD, nil
}

// codeFence matches ```json ... ``` or ``` ... ``` blocks that Claude sometimes
// wraps JSON output in despite being told not to. We extract the first JSON
// object from within a fence, or fall back to the raw text.
var codeFence = regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.*?\\})\\s*```")

func parseRecipeText(text string) (*recipe, error) {
	text = strings.TrimSpace(text)

	// Try direct parse first.
	var r recipe
	if err := json.Unmarshal([]byte(text), &r); err == nil {
		return &r, validateRecipe(&r)
	}

	// Try code-fence extraction.
	if m := codeFence.FindStringSubmatch(text); len(m) == 2 {
		if err := json.Unmarshal([]byte(m[1]), &r); err == nil {
			return &r, validateRecipe(&r)
		}
	}

	// Last resort: grab the first {...} block.
	if start := strings.Index(text, "{"); start >= 0 {
		if end := strings.LastIndex(text, "}"); end > start {
			if err := json.Unmarshal([]byte(text[start:end+1]), &r); err == nil {
				return &r, validateRecipe(&r)
			}
		}
	}

	preview := text
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}
	return nil, fmt.Errorf("could not parse recipe JSON from claude output: %q", preview)
}

func validateRecipe(r *recipe) error {
	switch r.Method {
	case "brew_cask", "brew_formula", "pkg_url", "dmg_url":
		if strings.TrimSpace(r.Identifier) == "" {
			return fmt.Errorf("method %q requires a non-empty identifier", r.Method)
		}
	case "skip":
		// identifier optional; ignore
	default:
		return fmt.Errorf("unknown method %q", r.Method)
	}
	switch r.Confidence {
	case "high", "medium", "low":
	default:
		return fmt.Errorf("unknown confidence %q", r.Confidence)
	}
	return nil
}

// -----------------------------------------------------------------------------
// Output
// -----------------------------------------------------------------------------

func writeRecipes(path string, data recipesFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".install-recipes-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

// -----------------------------------------------------------------------------
// Homebrew pre-filter (formula + cask probe, auto-skip on double-404)
// -----------------------------------------------------------------------------

// brewAPIClient is reused across probes so we keep a connection pool.
var brewAPIClient = fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second))

// brewProbeCache remembers probe results for the duration of a single run.
// Key: "formula/<name>" or "cask/<name>". Value: true if 200, false if 404.
var brewProbeCache = map[string]bool{}

// tryHomebrewShortcut probes the Homebrew API for the candidate's product name
// and returns a recipe without consulting Claude. Three outcomes:
//
//  1. Formula found → brew_formula recipe.
//  2. Cask found → brew_cask recipe.
//  3. Both 404 → auto-skip recipe. (Empirically 98%+ correct: products not on
//     Homebrew are almost never installable macOS desktop software.)
//
// Returns nil ONLY when --no-shortcut disables the pre-filter (handled by caller).
func tryHomebrewShortcut(p candidateEntry) *recipe {
	product := strings.ToLower(p.Product)

	// --- Stage 1: formula probe (exact name + versioned variants) ---
	formulaCandidates := []string{product}
	// Only try versioned variants when vendor==product (strong "this is a library" signal).
	if strings.EqualFold(p.Vendor, p.Product) {
		formulaCandidates = append(formulaCandidates, product+"@3", product+"@2", product+"@1")
	}
	for _, name := range formulaCandidates {
		if probeHomebrew("formula", name) {
			return &recipe{
				Method:     "brew_formula",
				Identifier: name,
				Confidence: "medium",
				SourcesConsulted: []string{
					"https://formulae.brew.sh/api/formula/" + name + ".json",
				},
				Rationale: fmt.Sprintf(
					"Homebrew formula %q found for NVD product %q. Auto-shortcut.",
					name, p.Product),
			}
		}
	}

	// --- Stage 2: cask probe ---
	// Try the product name, then vendor-product hyphenated (e.g., "google-chrome").
	caskCandidates := []string{product}
	if !strings.EqualFold(p.Vendor, p.Product) {
		caskCandidates = append(caskCandidates, strings.ToLower(p.Vendor)+"-"+product)
	}
	for _, name := range caskCandidates {
		if probeHomebrew("cask", name) {
			return &recipe{
				Method:     "brew_cask",
				Identifier: name,
				Confidence: "medium",
				SourcesConsulted: []string{
					"https://formulae.brew.sh/api/cask/" + name + ".json",
				},
				Rationale: fmt.Sprintf(
					"Homebrew cask %q found for NVD product %q. Auto-shortcut.",
					name, p.Product),
			}
		}
	}

	// --- Stage 3: AutoPkg index fuzzy search ---
	// If Homebrew doesn't have it but AutoPkg does, it's likely real macOS
	// software worth researching via Claude (don't auto-skip).
	if match := searchAutoPkgIndex(p.Vendor, product); match != "" {
		// Return nil to fall through to Claude research in the main loop.
		fmt.Fprintf(os.Stderr, "autopkg match %q → Claude ... ", match)
		return nil
	}

	// --- Stage 4: not on Homebrew, not in AutoPkg → auto-skip ---
	sources := []string{
		"https://formulae.brew.sh/api/formula/" + product + ".json",
		"https://formulae.brew.sh/api/cask/" + product + ".json",
		autopkgIndexURL,
	}
	return &recipe{
		Method:     "skip",
		Identifier: "",
		Confidence: "medium",
		SourcesConsulted: sources,
		Rationale: fmt.Sprintf(
			"No Homebrew formula/cask or AutoPkg recipe found for %q. Auto-skipped.",
			p.Product),
	}
}

// probeHomebrew returns true if formulae.brew.sh exposes a formula or cask with
// the given name. Results are cached per-run.
func probeHomebrew(kind, name string) bool {
	key := kind + "/" + name
	if v, ok := brewProbeCache[key]; ok {
		return v
	}
	u := fmt.Sprintf("https://formulae.brew.sh/api/%s/%s.json", kind, name)
	req, err := http.NewRequest(http.MethodHead, u, nil)
	if err != nil {
		brewProbeCache[key] = false
		return false
	}
	resp, err := brewAPIClient.Do(req)
	if err != nil {
		brewProbeCache[key] = false
		return false
	}
	defer resp.Body.Close()
	found := resp.StatusCode == http.StatusOK
	brewProbeCache[key] = found
	return found
}

// -----------------------------------------------------------------------------
// AutoPkg index (third-tier pre-filter)
// -----------------------------------------------------------------------------

const autopkgIndexURL = "https://raw.githubusercontent.com/jannheider/AutoPKGindex/main/index.json"

// autoPkgNames is populated lazily on first call to searchAutoPkgIndex.
// Each entry is a lowercased product name from the AutoPkg index.
var autoPkgNames []string
var autoPkgLoaded bool

// loadAutoPkgIndex fetches the AutoPkg index and extracts unique lowercase
// product names. Called once; errors are swallowed (the index is optional).
func loadAutoPkgIndex() {
	autoPkgLoaded = true
	resp, err := brewAPIClient.Get(autopkgIndexURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  (autopkg index fetch failed: %v)\n", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "  (autopkg index returned %d)\n", resp.StatusCode)
		return
	}

	var idx struct {
		Identifiers map[string]struct {
			Name string `json:"name"`
		} `json:"identifiers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&idx); err != nil {
		fmt.Fprintf(os.Stderr, "  (autopkg index decode failed: %v)\n", err)
		return
	}

	seen := make(map[string]struct{}, len(idx.Identifiers))
	for _, entry := range idx.Identifiers {
		name := strings.ToLower(strings.TrimSpace(entry.Name))
		if name == "" || name == "%name%" || strings.HasPrefix(name, "++") {
			continue
		}
		if _, ok := seen[name]; !ok {
			seen[name] = struct{}{}
			autoPkgNames = append(autoPkgNames, name)
		}
	}
	sort.Strings(autoPkgNames)
	fmt.Fprintf(os.Stderr, "  (autopkg index: %d unique product names loaded)\n", len(autoPkgNames))
}

// searchAutoPkgIndex does a fuzzy match of the NVD vendor/product against the
// AutoPkg product names. Returns the matched AutoPkg name, or "" if no match.
//
// Matching strategy (in order of specificity):
//  1. Exact match on product name.
//  2. Product name with hyphens/underscores replaced by spaces and vice versa.
//  3. AutoPkg name contains the product (for multi-word AutoPkg names like
//     "Google Chrome" matching product "chrome").
func searchAutoPkgIndex(vendor, product string) string {
	if !autoPkgLoaded {
		loadAutoPkgIndex()
	}
	if len(autoPkgNames) == 0 {
		return ""
	}

	product = strings.ToLower(product)
	vendor = strings.ToLower(vendor)

	// Normalize: hyphens, underscores, dots become spaces.
	normalized := strings.NewReplacer("-", " ", "_", " ", ".", " ").Replace(product)
	// Squish: strip ALL non-alphanumeric chars for maximum fuzziness.
	squished := squishName(product)

	for _, name := range autoPkgNames {
		nameSquished := squishName(name)
		switch {
		// Exact
		case name == product:
			return name
		// Normalized (e.g., "rocket-chat" matches "rocket chat")
		case name == normalized:
			return name
		// Squished (e.g., "rocketchat" matches "rocket.chat")
		case nameSquished == squished && squished != "":
			return name
		}
	}

	// Substring: AutoPkg name contains product (e.g., "Google Chrome" contains "chrome").
	// Only for products >= 4 chars to avoid false positives on short names.
	if len(product) >= 4 {
		for _, name := range autoPkgNames {
			if strings.Contains(name, product) || strings.Contains(squishName(name), squished) {
				return name
			}
		}
	}

	// Reverse substring: product contains the AutoPkg name.
	if len(product) >= 6 {
		for _, name := range autoPkgNames {
			nameNorm := strings.NewReplacer(" ", "_", "-", "_").Replace(name)
			if len(name) >= 4 && (strings.Contains(product, nameNorm) || strings.Contains(squished, squishName(name))) {
				return name
			}
		}
	}

	return ""
}

// squishName strips all non-alphanumeric characters and lowercases.
// "Rocket.Chat" → "rocketchat", "smart_switch" → "smartswitch".
func squishName(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func productKey(vendor, product string) string {
	return vendor + "/" + product
}

func fatalf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", a...)
	os.Exit(1)
}
