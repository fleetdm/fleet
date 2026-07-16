---
name: vuln-triage
description: Triage Fleet vulnerability false positives and false negatives across NVD, OSV, OVAL, MSRC, and Office data sources. Use when asked to "triage vuln", "investigate CVE", "fix false positive", "fix false negative", "vulnerability bug", invoked as `/vuln-triage`, or working on a vulnerability detection issue.
allowed-tools: Bash(go run -tags fts5 ./tools/nvd/nvdvuln*), Bash(ls /tmp/vulndbs*), Bash(ls -la /tmp/vulndbs*), Bash(file /tmp/vulndbs/*), Bash(zcat /tmp/vulndbs/*), Bash(sqlite3 /tmp/vulndbs/*), Bash(curl -s https://api.osv.dev/v1/vulns/*), Bash(git log*), Bash(git show*), Bash(gh issue list*), Bash(gh pr list*), Read, Grep, Glob, Edit, WebFetch
model: opus
effort: high
---

Triage a Fleet vulnerability detection bug (false positive or false negative): $ARGUMENTS

Two non-negotiable guardrails for this skill:
1. **Cross-source verification.** Always reconcile the CVE against multiple upstream sources before assuming a Fleet bug.
2. **Systemic-vs-one-off.** Always consider whether the bug points at a category problem (wrong data source for a whole ecosystem) before adding a one-off override.

If the systemic answer changes which scanner handles a `software.source`, propose an edit to the routing table in this skill alongside the code change.

## Step 1: Gather inputs

First, parse `$ARGUMENTS` for any of the fields below — invocations like `/vuln-triage Firefox 119 macOS apps CVE-2024-X false positive` already carry most of what's needed. Only ask the user for fields still missing after the parse. Echo back what you extracted so the user can correct misparses.

Required fields:
- Software `name`, `version`, and `software.source` (e.g. `apps`, `programs`, `deb_packages`, `rpm_packages`, `python_packages`, `npm_packages`, `homebrew_packages`, `chocolatey_packages`, `vscode_extensions`, `chrome_extensions`, `ios_apps`).
- Host platform (macOS, Windows, Ubuntu/Debian, RHEL/Fedora/CentOS, Amazon Linux, Arch, iOS).
- For macOS apps: `bundle_identifier`.
- The CVE ID in question.
- Direction: **false positive** (incorrectly flagged) or **false negative** (missed).

If direction is **false positive**, ask one disambiguating question before continuing — which of these does the engineer believe is happening?
- (a) Wrong CPE generated for the software (vendor/product mismatch).
- (b) Right CPE, but NVD's affected list is wrong for this CVE.
- (c) Right CPE and CVE, but the version comparison flagged a version that shouldn't match.

The answer steers steps 5–6: (a) → CPE generation / translations; (b) → upstream-data reconciliation / feed override; (c) → version comparator. A best-guess answer is fine — it's a hypothesis, not a contract.

Optional: a live Fleet URL + API token to drive nvdvuln Mode 2.

## Step 2: Identify the data source

Use the routing table below. If the table looks stale (line numbers shifted, a constant renamed), trust the linked code and update the table in step 9 alongside the fix.

### software.source × platform → scanner

| software.source | Platform | Primary scanner | Notes |
|---|---|---|---|
| `apps` | macOS | NVD; **macoffice** for Microsoft Office | macoffice match by bundle ID prefix `com.microsoft.{word,excel,powerpoint,outlook,onenote}` — see `server/vulnerabilities/macoffice/release_note.go` |
| `programs` | Windows | NVD; **winoffice** for Microsoft Office; CustomCVE rules with `SourceMatch="programs"` | winoffice regex `Microsoft (365\|Office)` excluding "Companion" — see `server/vulnerabilities/winoffice/analyzer.go` |
| `deb_packages` | Ubuntu | OSV when enabled, else OVAL; NVD fallback for `linux-image-*` outside known variants | OVAL routing: `server/vulnerabilities/oval/oval_platform.go`. `linux-image-*` NVD fallback: `BuildLinuxExclusionRegex` in `server/vulnerabilities/nvd/cpe.go` |
| `deb_packages` | Debian | OVAL | Same source list |
| `rpm_packages` | RHEL/CentOS/Fedora | OSV when enabled, else OVAL; **goval-dictionary** kernel-only on RHEL | `server/vulnerabilities/goval_dictionary/analyzer.go` |
| `rpm_packages` | Amazon Linux | goval-dictionary | amzn_01/02/2022/2023 |
| `pacman_packages` | Arch | NVD | Not covered by OVAL |
| `homebrew_packages` | macOS | NVD; CustomCVE rules (e.g. `git-gui`) | `server/vulnerabilities/customcve/matching_rules.go` |
| `npm_packages` | any | NVD (target SW = node.js) | `server/vulnerabilities/nvd/cpe.go` |
| `python_packages` | any | NVD (target SW = python) | Same |
| `chocolatey_packages` | Windows | NVD | |
| `vscode_extensions`, `jetbrains_plugins`, `chrome_extensions`, `firefox_addons`, `safari_extensions`, `ie_extensions` | any | NVD | |
| `go_binaries`, `portage_packages` | any | NVD | |
| `ios_apps`, `ipados_apps` | iOS/iPadOS | None — excluded from NVD | `server/vulnerabilities/nvd/cpe.go` `AllSoftwareIterator` |
| (n/a — `os_versions` table) | Windows | **MSRC** | Operates on `os_versions`, not `software` — `server/vulnerabilities/msrc/analyzer.go` |

<!-- Routing table last verified at commit f92e1e2c34. If you re-verify against current code, bump this. -->

### Detail-query → source mapping

`software_macos` → `apps`. `software_windows` → `programs`. `software_linux` → `deb_packages`/`rpm_packages`/`pacman_packages`. `software_python_packages_with_users_dir` → `python_packages`. `software_npm_packages` → `npm_packages`. `software_homebrew_packages` → `homebrew_packages`. `software_chocolatey_packages` → `chocolatey_packages`. `software_vscode_extensions`, `software_jetbrains_plugins`, `software_chrome_extensions`, `software_firefox_addons`, `software_safari_extensions`, `software_ie_extensions`, `software_ios_apps`, `software_ipados_apps`.

### Dispatch entry points (read on each invocation)

`cmd/fleet/cron.go` — `checkNVDVulnerabilities`, `checkOvalVulnerabilities`, `checkOSVVulnerabilities`, `checkRHELOSVVulnerabilities`, `checkMacOfficeVulnerabilities`, `checkWinOfficeVulnerabilities`, `checkCustomVulnerabilities`. vulncheck supplements NVD: `server/vulnerabilities/nvd/sync/cve_syncer.go`.

Pick the scanner from the table. If the bug is on `apps`/`programs` and the software is Microsoft Office, the scanner is macoffice/winoffice — not NVD. Confirm before continuing.

## Step 3: Verify /tmp/vulndbs is set up

```sh
ls /tmp/vulndbs
```

Expected files:
- `cpe.sqlite`, `cpe_translations.json`
- `nvdcve-*.json.gz` (multiple year ranges)
- `epss_scores-current.csv`, `known_exploited_vulnerabilities.json`
- `osv-ubuntu-*.json.gz`, `osv-rhel-*.json.gz` (if doing an OSV bug)
- `fleet_oval_*.json` (OVAL bugs)
- `fleet_goval_dictionary_*.sqlite3` (Amazon Linux / RHEL kernel)
- `macoffice/`, `winoffice/` directories

If files are missing, advise the engineer to run `fleetctl vulnerability-data-stream --dir /tmp/vulndbs`. It syncs everything listed above — NVD, OVAL, MSRC, OSV (via `osv.RefreshAll` at `cmd/fleetctl/fleetctl/vulnerability_data_stream.go:106`), macoffice, and goval-dictionary. `nvdvuln -sync` uses the same `osv.RefreshAll`, so either entry point lands the same files.

Do not run sync automatically — the download is large. Tell the engineer and wait.

## Step 4: NVD path — run nvdvuln

Only for NVD-handled software per the table. For OSV/OVAL/MSRC/Office, skip to step 7.

Mode 1 (single software):

```sh
go run -tags fts5 ./tools/nvd/nvdvuln \
    -software_name "<name>" \
    -software_version "<version>" \
    -software_source "<source>" \
    -software_bundle_identifier "<bundle id, if applicable>" \
    -db_dir /tmp/vulndbs
```

Add `-sync` only if step 3 found data missing — and only with user approval.

Mode 2 (compare against live Fleet):

```sh
go run -tags fts5 ./tools/nvd/nvdvuln \
    -software_from_url "<https://fleet.example.com>" \
    -software_from_api_token "<token>" \
    -db_dir /tmp/vulndbs
```

Read the `Matched CPE:` lines (CPE generation result) and the `CVEs found for ...` line (matching result). Compare to expected.

See `tools/nvd/nvdvuln/README.md`.

## Step 5: Cross-source verification (mandatory before any fix)

Always run the baseline. Add the conditional fetches that apply to this routing:

**Baseline (always):**
- NVD: WebFetch `https://nvd.nist.gov/vuln/detail/<CVE-ID>` — what CPEs does NVD list?
- MITRE: WebFetch `https://www.cve.org/CVERecord?id=<CVE-ID>` — canonical description.
- OSV.dev: `curl -s "https://api.osv.dev/v1/vulns/<CVE-ID>" > /tmp/<CVE-ID>.osv.json` then `Read` it. The endpoint returns raw JSON; WebFetch's HTML→markdown pass mangles it.

**Conditional (skip if not applicable):**
- GHSA — only when `software.source` is a language ecosystem (`npm_packages`, `python_packages`, `gem_packages`, `maven_packages`, etc.). WebFetch `https://github.com/advisories?query=<CVE-ID>`.
- MSRC vendor page — only when the scanner from step 2 is MSRC, or when vendor matches Microsoft. WebFetch `https://msrc.microsoft.com/update-guide/vulnerability/<CVE-ID>`.
- First-party vendor PSIRT — only when vendor matches a first-party publisher (Apple, Mozilla, Adobe, Google for Chrome, etc.). WebFetch the relevant security advisory page.

Reconcile:
- **NVD agrees with other sources** → if Fleet still misdetects, the bug is in Fleet logic (steps 6–8).
- **NVD disagrees with the others** (wrong vendor, missing product, version range too broad) → upstream NVD data is wrong. Fix layer is `cpe_matching_rules.go` or the dictionary `Override()` pattern, **not** Fleet logic.
- **CVE not in NVD but present in GHSA/OSV** → NVD coverage gap for this ecosystem; this is a systemic concern (step 9), not a CustomCVE candidate by default.

Capture the disagreement in the final report.

## Step 6: FP triage (NVD-handled software)

If nvdvuln reproduces the bad CVE, narrow the cause:

- **Wrong CPE generated** (e.g. `vendor=apple, product=icloud` matched against a Windows host) — open `server/vulnerabilities/nvd/cpe_matching_rules.go` and propose a rule in `GetKnownNVDBugRules`.
- **Wrong vendor/product mapping** — edit [`server/vulnerabilities/nvd/cpe_translations.json`](server/vulnerabilities/nvd/cpe_translations.json) in this repo. The matching logic lives in `cpe_translations.go`. Note: running Fleet servers pull this file from `github.com/fleetdm/nvd` releases, which is republished from the in-repo source daily — the edit lands here, not in `fleetdm/nvd`.
- **sanitize stripping the wrong substring** — open `server/vulnerabilities/nvd/sanitize.go`, look at `sanitizeSoftwareName` and `productVariations`. Reason about why the variation set landed on the wrong product.
- **Detail query producing the wrong row** — open `server/service/osquery_utils/queries.go` and check `SoftwareOverrideMatch` for the platform. macOS Firefox is the canonical example.
- **NVD upstream data is wrong** (confirmed in step 5) — feed override at `server/vulnerabilities/nvd/tools/cvefeed/dictionary.go` (`Override()`) and `OverrideVuln` in `vuln.go`.

## Step 7: FP triage (non-NVD sources)

No nvdvuln. Read the analyzer and the local feed for the relevant source:

- **OSV** — `server/vulnerabilities/osv/analyzer.go`. Local feed: `/tmp/vulndbs/osv-{ubuntu,rhel}-*.json.gz` (`zcat` to inspect). Cross-check against `https://api.osv.dev/v1/vulns/<CVE-ID>`.
- **OVAL** — `server/vulnerabilities/oval/`. Local feed: `/tmp/vulndbs/fleet_oval_*.json`. Cross-check the upstream OVAL definition for the distro version.
- **goval-dictionary** — `server/vulnerabilities/goval_dictionary/analyzer.go`. Local DB: `/tmp/vulndbs/fleet_goval_dictionary_*.sqlite3` (`sqlite3` CLI).
- **MSRC** — `server/vulnerabilities/msrc/analyzer.go`. Local artifacts: `/tmp/vulndbs/fleet_msrc_*.json`. Cross-check `https://msrc.microsoft.com/update-guide/vulnerability/<CVE-ID>`.
- **macoffice / winoffice** — release-notes JSON in `/tmp/vulndbs/macoffice/` or `winoffice/`. Cross-check Microsoft's Office release notes pages.

## Step 8: FN triage

"Fleet missed CVE-X on software Y." Walk the funnel in order:

1. **Is the software being collected?** Inspect the relevant detail query in `server/service/osquery_utils/queries.go`. Confirm rows reach `host_software_installed_paths`.
2. **Is a CPE generated?** Run nvdvuln Mode 1 — does it report `Matched CPE`? If not, the gap is upstream of CVE matching.
3. **Is the CPE in the NVD data?** `sqlite3 /tmp/vulndbs/cpe.sqlite` and query for the vendor/product. If absent, cross-check OSV.dev / GHSA. If they list it but NVD doesn't → this is a **systemic gap** (step 9), not a CustomCVE candidate.
4. **For OSV/OVAL FN** — verify the package source/name in the local feed file and against upstream OSV before assuming a code bug.
5. Only after step 9 rules out a systemic fix: propose adding a rule in `server/vulnerabilities/customcve/matching_rules.go` (one-off override).

## Step 9: Systemic-vs-one-off check (mandatory before any override)

Before suggesting a CPE matching rule, CustomCVE entry, sanitize regex, or feed override, ask:

- Does this bug class affect a whole ecosystem, not one CVE? (e.g., dozens of npm packages misdetected → "use OSV/GHSA for npm" beats 30 CPE rules.)
- Is there a better-suited data source? GHSA for npm/pip/maven/rubygems; OSV for distro and OSS ecosystems; vendor PSIRT for first-party software.
- Is the data missing in the source we use but present in another? (See step 5 reconciliation.)
- Has this kind of bug been reported before for similar software? Run:

```sh
git log --oneline -20 --since=2.years --grep="<vendor or product>" -- server/vulnerabilities/
gh issue list --limit 20 --state all --search "<software> vulnerability"
```

If a systemic fix exists, **surface it first** in the report. Only fall back to an override when the user explicitly opts out, and frame the override as a stopgap.

**Routing-table self-maintenance.** If the proposed systemic fix changes which scanner handles a `software.source` (example: route `npm_packages` through OSV instead of NVD), the proposal **must** include a corresponding edit to the routing table in step 2 of this SKILL.md. Present both diffs (code + skill) together. Do not let the routing reference drift.

## Step 10: Propose the edit

This skill is **diagnose-then-apply-on-approval**: print the exact file + line + diff intended in chat, wait for the user's explicit "go" (or a revised diff), then apply. `Edit` is in `allowed-tools` so the apply step doesn't re-prompt — the gate is the explicit approval in this step, not the tool-permission prompt.

- Propose first, apply on approval. Never skip the propose step.
- For `cpe_translations.json`: the file is in this repo at `server/vulnerabilities/nvd/cpe_translations.json` (republished daily into `fleetdm/nvd` releases for running servers to pull). Propose the diff against the in-repo file.
- For systemic fixes: produce a written recommendation rather than a diff. Identify the routing table edit needed in this SKILL.md.
- For one-off overrides (CPE matching rule, CustomCVE rule, sanitize regex, feed override): propose a precise diff to the in-repo file.

## Step 11: Report

Print this summary at the end of the run:

- **Software & CVE** — `<name> <version>` on `<platform>` / `<software.source>` vs `<CVE-ID>`.
- **Direction** — false positive | false negative.
- **Scanner involved** — NVD | OSV | OVAL | goval-dictionary | MSRC | macoffice | winoffice | CustomCVE.
- **Cross-source agreement** — one line per source: NVD says X, GHSA says Y, OSV says Z, vendor says W. Highlight any disagreement.
- **Override layer touched (or systemic recommendation)** — name the file/function or describe the recommended data-source change.
- **Root cause** — one sentence.
- **Proposed fix** — file + diff (or written recommendation for systemic). Note whether it is a stopgap or systemic.
- **Routing-table change required?** — yes/no and the diff if yes.
