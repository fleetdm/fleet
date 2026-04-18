# CPE Accuracy Test Suite

A pipeline of four Go CLI tools that measures how accurately Fleet's fuzzy CPE
matcher generates CPE strings for real macOS software. The tools compose via
JSON files on disk and SSH to a macOS VM running Fleet's Orbit agent.

## Why this exists

Fleet's vulnerability scanning maps software inventory → CPE strings → NVD CVEs.
The CPE generation step (`CPEFromSoftware()`) uses fuzzy matching against a
SQLite CPE dictionary. This is inherently best-effort — sanitization heuristics,
vendor/product variation generation, and full-text search can guess wrong.

This toolset provides a repeatable way to:

1. Identify which real-world products (via NVD CRITICAL/HIGH CVEs) Fleet should
   be generating CPEs for.
2. Install those products on a macOS VM and capture how osquery reports them.
3. Check whether Fleet's CPE matcher produces the correct CPE for each product.
4. Surface vendor-disambiguation bugs, missing CPE dictionary entries, and
   version-normalization issues.

## Tools

| Tool | Purpose | Touches VM? | Cost |
|------|---------|-------------|------|
| `cpe-candidates` | Extract macOS-relevant vendor/product pairs from NVD CVE feeds | No | Free |
| `recipe-generator` | Determine install method for each product (Homebrew probe + Claude fallback), install on VM, validate | Yes (install) | Free for Homebrew probe; ~$0.08/product for Claude fallback |
| `testdata-reconcile` | Query VM's osquery tables, match against candidates, write golden test files | Yes (read-only) | Free |
| `cpe-accuracy` | Run Fleet's CPE matcher against the golden files and report pass/fail | No | Free |

## Sequence diagram

```
                          Developer workstation                          macOS VM
                    ┌──────────────────────────────┐            ┌──────────────────┐
                    │                              │            │  Orbit + osqueryd │
                    │                              │            │  Homebrew         │
                    │                              │            │  SSH (key auth)   │
                    └──────────────────────────────┘            └──────────────────┘

    ┌─────────────────────────────────────────────────────────────────────────────────┐
    │ STEP 1: Refresh NVD feeds (one-time)                                            │
    │                                                                                 │
    │   $ fleetctl vulnerability-data-stream --dir /tmp/vulnsdb                       │
    │                                                                                 │
    │   Downloads cpe.sqlite, cpe_translations.json, nvdcve-1.1-*.json.gz, etc.      │
    └─────────────────────────────────────────────────────────────────────────────────┘
                    │
                    ▼
    ┌─────────────────────────────────────────────────────────────────────────────────┐
    │ STEP 2: Extract candidates (offline)                                            │
    │                                                                                 │
    │   $ go run ./tools/nvd/accuracy/cpe-candidates                                  │
    │                                                                                 │
    │   /tmp/vulnsdb/nvdcve-1.1-2026.json.gz                                         │
    │       │                                                                         │
    │       ▼  Filter: year=2026, CVSS≥7.0, part=application, macOS-relevant          │
    │       │  Deduplicate by (vendor, product)                                       │
    │       ▼                                                                         │
    │   testdata/accuracy/cpe_candidates_2026_high.json                               │
    │       { candidates: [{vendor, product, target_sw_hints, related_cves}] }        │
    └─────────────────────────────────────────────────────────────────────────────────┘
                    │
                    ▼
    ┌─────────────────────────────────────────────────────────────────────────────────┐
    │ STEP 3: Research + install (Homebrew pre-filter + optional Claude)               │
    │                                                                                 │
    │   $ go run ./tools/nvd/accuracy/recipe-generator                                │
    │                                                                                 │
    │   For each candidate:                                                           │
    │                                                                                 │
    │       ┌─ Probe formulae.brew.sh/api/formula/<product>.json ─── 200? ──► brew    │
    │       │                                                        formula  │
    │       ├─ Probe formulae.brew.sh/api/cask/<product>.json ────── 200? ──► brew    │
    │       │                                                        cask     │
    │       ├─ Both 404? ─────────────────────────────────────────── auto-skip │
    │       │                                                        (98%+    │
    │       │                                                        accurate)│
    │       └─ --no-shortcut? ── Claude Code CLI ── WebSearch/Bash ─► recipe  │
    │                                                                         │
    │       If method ≠ skip:                                                 │
    │           SSH ──────────────────────────────────────────────►  VM       │
    │           brew install [--cask] <name>                        installs  │
    │           ◄──────────────── stdout/stderr/exit code ─────────  ✓ or ✗  │
    │                                                                         │
    │   testdata/accuracy/install_recipes.json                                │
    │       { recipes: { "vendor/product": {method, identifier, validation} } │
    │                                                                         │
    │   Idempotent: verified recipes skipped on re-run.                       │
    └─────────────────────────────────────────────────────────────────────────────────┘
                    │
                    ▼
    ┌─────────────────────────────────────────────────────────────────────────────────┐
    │ STEP 4: Reconcile candidates with VM state → golden files                       │
    │                                                                                 │
    │   $ go run ./tools/nvd/accuracy/testdata-reconcile                              │
    │                                                                                 │
    │       SSH ──────────────────────────────────────────────────►  VM               │
    │       osqueryd -S --json "SELECT ... FROM apps"               (read-only)       │
    │       osqueryd -S --json "SELECT ... FROM homebrew_packages"                    │
    │       osqueryd -S --json "SELECT ... FROM python_packages"                      │
    │       ... (8 source tables total, ~15s)                                         │
    │       ◄──────────────── JSON rows ──────────────────────────                    │
    │                                                                                 │
    │       For each candidate:                                                       │
    │           Match (vendor, product) against inventory rows                        │
    │               apps:              bundle_identifier / name heuristic             │
    │               homebrew_packages: name == product (+ @N variants)                │
    │               python/npm:        name == product                                │
    │               extensions:        name/id contains product                       │
    │                                                                                 │
    │           Hit  → emit accuracyCase (software + expected CPE)                    │
    │           Miss → add to missing_products.json                                   │
    │                                                                                 │
    │   testdata/accuracy/cves_2026_high_apps.json                                    │
    │   testdata/accuracy/cves_2026_high_homebrew.json                                │
    │   testdata/accuracy/cves_2026_high_extensions.json                              │
    │   testdata/accuracy/cves_2026_high_language_pkgs.json                            │
    │   testdata/accuracy/missing_products.json                                       │
    │                                                                                 │
    │   If missing_products.json is non-empty:                                        │
    │       Loop back to STEP 3 with --input missing_products.json                    │
    └─────────────────────────────────────────────────────────────────────────────────┘
                    │
                    ▼
    ┌─────────────────────────────────────────────────────────────────────────────────┐
    │ STEP 5: Run accuracy check (offline)                                            │
    │                                                                                 │
    │   $ go run ./tools/nvd/accuracy/cpe-accuracy --vuln-path /tmp/vulnsdb           │
    │                                                                                 │
    │       Load cves_*.json golden files                                             │
    │       Load cpe.sqlite + cpe_translations.json                                   │
    │                                                                                 │
    │       For each test case:                                                       │
    │           Fleet's CPEFromSoftware(software) → actual CPE                        │
    │           Compare actual vs expected                                            │
    │                                                                                 │
    │       ┌──────────────────────────────────────────────────┐                      │
    │       │  CPE Accuracy Report                             │                      │
    │       │  Total: 91 | PASS: 72 (79%) | FAIL: 19 (21%)    │                      │
    │       │                                                  │                      │
    │       │  FAIL (mismatch):  druid → alibaba not apache    │                      │
    │       │  FAIL (missing):   Cursor.app → no CPE at all    │                      │
    │       └──────────────────────────────────────────────────┘                      │
    │                                                                                 │
    │   Exit code: 0 if all pass, 1 if any fail                                      │
    └─────────────────────────────────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────────────────────────────┐
    │ FEEDBACK LOOP                                                                   │
    │                                                                                 │
    │   Failures from STEP 5 inform fixes in Fleet's CPE matcher:                     │
    │     • Vendor mismatch → add cpe_translations.json rules                         │
    │     • Missing CPE     → add cpe_translations.json for new products              │
    │     • Version issues  → fix sanitizeVersion() or test expectations              │
    │                                                                                 │
    │   Re-run STEP 5 after fixes to confirm regressions are resolved.                │
    └─────────────────────────────────────────────────────────────────────────────────┘
```

## Quick start

```sh
# Prerequisites:
#   - macOS VM with SSH key auth (alias "fleet-testdata-vm" in ~/.ssh/config)
#   - Orbit + osqueryd installed on the VM
#   - Homebrew installed on the VM
#   - fleetctl built

# 1. Download NVD feeds
NETWORK_TEST_GITHUB_TOKEN=$(gh auth token) fleetctl vulnerability-data-stream --dir /tmp/vulnsdb

# 2. Extract candidates
go run ./tools/nvd/accuracy/cpe-candidates --severity HIGH

# 3. Research + install (Homebrew pre-filter handles ~95%, free)
go run ./tools/nvd/accuracy/recipe-generator \
    --input server/vulnerabilities/nvd/testdata/accuracy/cpe_candidates_2026_high.json

# 4. Generate golden files from live VM state
go run ./tools/nvd/accuracy/testdata-reconcile

# 5. Check accuracy
go run ./tools/nvd/accuracy/cpe-accuracy --vuln-path /tmp/vulnsdb
```

## File layout

```
tools/nvd/accuracy/
├── README.md                       ← this file
├── cpe-accuracy/
│   ├── main.go                     (603 lines)
│   └── main_test.go                (126 lines)
├── cpe-candidates/
│   ├── main.go                     (371 lines)
│   └── main_test.go                (88 lines)
├── recipe-generator/
│   └── main.go                     (897 lines)
└── testdata-reconcile/
    └── main.go                     (606 lines)

server/vulnerabilities/nvd/testdata/accuracy/
├── README.md                       Golden file format docs
├── RESULTS-2026-04-16.md           Baseline accuracy report
├── cpe_candidates_2026_critical.json
├── cpe_candidates_2026_high.json
├── cves_2026_critical_apps.json
├── cves_2026_critical_homebrew.json
├── cves_2026_critical_extensions.json
├── cves_2026_critical_language_pkgs.json
├── cves_example.json               Hand-authored smoke test
├── install_recipes.json            Per-product install method + validation
└── missing_products.json           Products not yet on the VM
```

## Design principles

**Idempotent everywhere.** Every tool can be re-run safely. `recipe-generator`
skips verified installs. `testdata-reconcile` overwrites golden files from live
VM state. `cpe-accuracy` is a pure function of golden files + CPE DB.

**Homebrew pre-filter eliminates ~90% of Claude calls.** Before the CRITICAL run,
every candidate went to Claude (~$48 for 591 products). After analyzing the
results, we found that products not on Homebrew are skipped 98%+ of the time.
Now the tool probes `formulae.brew.sh` first (free, ~100ms) and only falls back
to Claude for ambiguous cases via `--no-shortcut`.

**VM is append-only.** Software installed on the VM stays there across runs.
The reconcile step reads whatever's on the VM each time — no persisted snapshot
file. This means re-running reconcile after NVD feeds update is free (no
reinstalls), and only genuinely new products trigger the install path.

**Golden files are the single source of truth.** No intermediate snapshot files.
Each `cves_*.json` entry contains the full osquery row (name, version, source,
bundle_identifier) plus the expected CPE and provenance CVE. Git history of these
files is the audit trail.

## Key results (baseline 2026-04-16)

From the initial CRITICAL-severity run:

| Metric | Value |
|--------|-------|
| Products researched | 591 |
| Verified installs | 83 |
| Skipped (not macOS desktop) | 502 |
| Golden test cases | 88 |
| **CPE accuracy** | **79.1% (72/91 pass)** |
| Vendor mismatch failures | 11 |
| Missing CPE failures | 8 |

See `RESULTS-2026-04-16.md` for the full breakdown and actionable follow-ups.

## VM requirements

A single long-lived macOS VM:

- **SSH key auth** — `~/.ssh/fleet_testdata_vm_ed25519`, config alias
  `fleet-testdata-vm` → `admin@<ip>`.
- **Homebrew** — at `/opt/homebrew/bin/brew` (Apple Silicon).
- **Fleet Orbit** — provides the bundled osqueryd at
  `/opt/orbit/bin/osqueryd/macos-app/stable/osquery.app/Contents/MacOS/osqueryd`.
  Used for `osqueryd -S --json '<query>'` (no root required).
- **No Fleet server enrollment required** — osqueryd runs standalone in shell mode.
