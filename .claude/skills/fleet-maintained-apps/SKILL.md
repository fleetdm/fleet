---
name: fleet-maintained-apps
description: Add, fix, or debug Fleet-maintained apps (FMAs) in ee/maintained-apps — Homebrew casks for macOS, winget packages for Windows. Use whenever a task touches FMA inputs, outputs, or install/uninstall scripts, even if the user never says "FMA" — adding an app or its other platform, a failing FMA validator or nightly "Update Fleet-maintained apps" PR, an ingest panic, an `~fma-soak` issue or customer report (install hangs, "app not detected", patch policy always failing or never failing, update loop), a per-user Windows app that breaks under SYSTEM, the exists/patched/open queries or patch policy, freezing or unfreezing an app, or pruning comments in a shipped script. Centers on verifying installer identity and behavior with real tools and real osquery instead of trusting catalog metadata.
allowed-tools: Bash, Read, Write, Edit, Grep, Glob, WebFetch, WebSearch
model: opus
effort: high
---

# Fleet-maintained apps

Task: $ARGUMENTS

An FMA is an input JSON in `ee/maintained-apps/inputs/homebrew/` or `inputs/winget/` (plus optional custom scripts), run through a generator that writes `ee/maintained-apps/outputs/<slug-token>/<platform>.json`: installer URL, SHA-256, install/uninstall scripts, and three osquery queries. Fleet servers fetch these manifests at runtime from `maintained-apps.fleetdm.com`, so a fix merged to main reaches every server without a release. The scripts run as root (macOS) or SYSTEM (Windows) on customer hosts, and the queries decide whether Fleet installs, reports, and patches the app. The contributor basics are in [ee/maintained-apps/README.md](../../../ee/maintained-apps/README.md); this skill is the workflow plus what the README doesn't cover. Where they disagree, follow this skill.

## Where to start

| You were asked to… | Go to |
|---|---|
| Add a macOS app | [Workflow: macOS](#workflow-macos) |
| Add a Windows app | [Workflow: Windows](#workflow-windows) |
| Add the other platform of an existing FMA | [Second platform](#second-platform-of-an-existing-fma), then the platform workflow |
| Fix a validator failure, nightly PR, or ingest panic | [references/validator-failures.md](references/validator-failures.md), indexed by error message |
| Work an `~fma-soak` issue or customer report | [validator-failures.md#soak-issues-and-customer-reports](references/validator-failures.md#soak-issues-and-customer-reports) first: it may already be fixed |
| Check or fix exists/patched/open, or a patch policy | [Validating the queries](#validating-the-queries), then [references/queries.md](references/queries.md) |
| Write or fix a macOS install/uninstall script | [references/macos-scripts.md](references/macos-scripts.md) |
| Write or fix a Windows install/uninstall script | [references/windows-scripts.md](references/windows-scripts.md) |
| Clean up comments in a shipped script | [references/script-comments.md](references/script-comments.md) |

Whatever the task, finish with the [pre-ship checklist](#pre-ship-checklist).

## Ground rules

1. **Verify identity from the real installer, never from catalog metadata.** winget `PackageName`/`Publisher`/`InstallerType` and cask `zap`/`quit` entries are wrong often enough that guessing is the single biggest source of broken FMAs. Read the field from the installer itself ([identity-verification.md](references/identity-verification.md)), and if you can't, say so in the PR instead of shipping a guess.
2. **Prove the queries with real osquery.** CI now runs the exists query and, on macOS, the patched query in some cases, but never the open query, never a patched query that should fail, and never as SYSTEM. See [Validating the queries](#validating-the-queries).
3. **Fix the one app, not shared code.** The homebrew template (`ingesters/homebrew/scripts.go`), the generators, and `pkg/patch_policy` produce every FMA's scripts and queries, so a change there regenerates hundreds of outputs. Reach for, in order: an input field, a per-app custom script, a per-token/per-slug ingester branch, a per-app version transformer in `external_refs`. Change the shared template only when it is the root cause for many apps (the zip `ditto` fix), and then sync every copied helper in `inputs/*/scripts/`.
4. **`inputs/homebrew/custom-tap/` is for Fleet's internal use.** It holds casks Fleet maintains for its own purposes (Fleet Desktop and a few others) and is updated by an internal process. Don't add a cask there to get an app into the catalog, to replace a cask Homebrew removed or renamed, or to work around a stale or binary-only cask. An app with no usable Homebrew cask or winget manifest isn't an FMA candidate: report that and stop.
5. **Installers are untrusted input; shipped scripts are root/SYSTEM code.** Follow [Handling installers safely](#handling-installers-safely), and never let a value from a cask, a registry key in a user's hive, or a file on disk become code in a script.

## Handling installers safely

- Download each installer into a new, empty temp directory, check its size first (`curl -sIL <url>`), and delete it when you're done. For large installers, use the range-request recipes in [identity-verification.md](references/identity-verification.md#partial-download-probes); don't pull gigabytes to read a name.
- Compare the download's SHA-256 with the manifest's before trusting anything you read from it.
- Never run a downloaded installer or uninstaller on your Mac. Inspect it (`hdiutil attach -readonly -nobrowse`, `pkgutil --expand-full`, `ditto -x -k`, `7zz l`, `msiinfo`), and install only on a disposable VM or test host. The one thing you launch locally is the app itself for the open-query check, and only once its hash matches the manifest and Gatekeeper accepts it.
- Keep your own helper scripts outside the download directory and run Python with `-I`, so nothing in the payload can shadow a module.
- Don't fabricate or install an `.app` in `/Applications` on an enrolled Mac to test a query: it shows up in real inventory. Point the queries at the mounted or extracted bundle instead (`check_queries.py --app-path`).
- In shipped scripts: quote every expansion; pass bundle ids and paths to `osascript` as data (`on run argv` … `--`), never by string interpolation; validate ids read from disk before use; on Windows, run an `UninstallString` from a user's hive only as that user ([windows-scripts.md](references/windows-scripts.md#uninstall-must-run-in-the-hive-that-owns-the-registration)); stage files a user-context task must read in a directory whose ACL you set, not a world-writable one.
- Don't weaken integrity checks to get green: `ignore_hash`/`no_check` and frozen pins on rolling URLs have real costs ([validator-failures.md](references/validator-failures.md#rolling-latest-urls)).

## Prerequisites and helpers

```bash
brew install msitools sevenzip innoextract   # msiinfo; 7zz for NSIS/Burn/MSIX payloads; innoextract for Inno <= 6.2
gh auth status                               # winget manifests come from the GitHub contents API
```

Bundled helpers in [scripts/](scripts/) (run Python with `python3 -I`):

| Script | Does |
|---|---|
| `inspect_mac_installer.sh <url\|file> [--sha256 X] [--keep]` | Real installer format, each app's bundle id, versions, executable, signer, Gatekeeper verdict, nested apps, and pkg scripts that kill or relaunch the app. `--keep` leaves the bundle mounted for `check_queries.py --app-path`. |
| `check_queries.py <output.json> [--app-path P] [--ps1 F]` | Runs exists, patched, open, and patched with its version literals bumped, through real osquery (darwin), or writes a `.ps1` that does the same as SYSTEM on Windows. |
| `winget_manifest.sh <PackageIdentifier> [version] [--diff]` | Newest versions, recent commits (in-place edits, auto-deletions, moves), installer and locale manifests (with the `DefaultLocale` fallback), and a diff against the previous version. |
| `pe_info.py <file>` | A Windows installer's architecture and VERSIONINFO from the first few MB. |
| `burn_ux.py <head.bin> <out.cab>` | A WiX Burn bundle's UX cabinet (ARP identity) from a 30 MB range request. |

On macOS, osquery comes from fleetd (`/opt/orbit/bin/osqueryd/macos-app/stable/osquery.app/Contents/MacOS/osqueryd`). Without fleetd, extract it from the cask without installing anything: `brew fetch --cask osquery && pkgutil --expand-full "$(brew --cache --cask osquery)" <dir>`, then pass `--osqueryd <dir>/Payload/opt/osquery/lib/osquery.app/Contents/MacOS/osqueryd`.

## How the generator behaves

```bash
go run cmd/maintained-apps/main.go --slug="<token>/<platform>" --debug
```

- It fetches the cask or winget manifest and writes `outputs/<token>/<platform>.json`. Scripts are embedded under `refs`, keyed by `sha256(script)[:8]`, so a script change means regenerating (or recomputing the ref by hand).
- `--slug` is a **substring** match: `r/darwin` also regenerates `codexbar`, `keycastr`, and `rustrover`. `git checkout` collateral outputs before committing.
- New apps are appended to `outputs/apps.json` with an empty description. It never updates `name` or `unique_identifier` on an existing entry and never removes entries: change those by hand.
- Regenerating refetches upstream and can bump the version. If you only meant to change a script, check the diff, or hand-edit ([validator-failures.md](references/validator-failures.md#editing-an-output-without-regenerating)).
- **`frozen: true` only stops the output from being rewritten.** Upstream is still fetched for every input, so a removed cask or a winget manifest with no matching installer still breaks the run. Regenerating a frozen app silently leaves the old output, stale script refs included: edit a frozen output by hand.
- `default_categories` must be one of `Browsers`, `Communication`, `Developer tools`, `Productivity`, `Security`, `Support`, `Utilities` (the README's list is outdated).
- A full run aborts at the first failing app. To check many inputs, build once and loop per slug ([validator-failures.md](references/validator-failures.md#winget-manifest-changed-underneath-the-input)).

## Workflow: macOS

1. **Read the cask** (`curl -s https://formulae.brew.sh/api/cask/<token>.json`), especially `artifacts`, `url`, `version`, and `sha256`.
   - **It needs an `app` artifact.** The exists query reads osquery's `apps` table, which only lists `.app` bundles. A pkg-only or binary-only cask (JDKs, runtimes, drivers, daemons, CLI tools, prefPanes) can't be a working FMA. Stop and report it.
   - A `suite` (a folder such as `/Applications/KiCad`) or `artifact` stanza isn't parsed by the ingester, so it needs custom scripts ([macos-scripts.md](references/macos-scripts.md#suite-casks-and-nested-apps)).
   - No cask at all, or only a formula: not an FMA candidate (ground rule 4).
   - Arch-split casks: the API's top-level `url` is the arm64 build. That's expected.
2. **Inspect the real installer**: `scripts/inspect_mac_installer.sh "<url>" --sha256 <cask sha>`.
   - `installer_format` is what the file really is (`dmg`/`zip`/`pkg`), not what the URL or artifact implies. A zip or pkg inside a dmg, or a dmg inside a zip, needs a custom install script.
   - `unique_identifier` is the top-level app's `CFBundleIdentifier`. Never take it from the cask's `zap`, `quit`, `pkgutil`, or `launchctl` entries, a pkg receipt id, or a prefs-plist name: each of those has shipped a wrong id (ForkLift `-4`, GIMP `-3.0`, Tableau's receipt id, 4K Video Downloader's prefs name, ShareFile's XPC service).
   - Compare `CFBundleShortVersionString` and `CFBundleVersion` with the cask `version`. The default patched query compares `bundle_short_version`, so if that's missing, shorter, or a marketing version, the patch policy fails on current hosts ([queries.md](references/queries.md#when-the-version-column-doesnt-track-the-manifest)).
   - Gatekeeper must say `accepted`. A rejected or unnotarized app fails CI and won't launch for users.
   - Note nested apps (the open query only sees the main executable), pkg scripts that kill or relaunch the app, and a payload whose version differs from the cask's.
3. **Create `inputs/homebrew/<token>.json`**: `name`, `slug` (`<token>/darwin`), `unique_identifier`, `token` (the cask token, which can differ from the slug), `installer_format`, `default_categories`. Scripts generate from the cask's artifacts and `zap`. Add custom scripts only when the generated ones can't work, and read [macos-scripts.md](references/macos-scripts.md) first: custom scripts must copy the template's helpers verbatim, and guard tests in `ingesters/homebrew/scripts_test.go` fail on stale copies.
4. **Generate, finalize, and [validate the queries](#validating-the-queries).**

## Workflow: Windows

1. **Read the winget manifests**: `scripts/winget_manifest.sh <PackageIdentifier>`. Pick **machine** scope and **x64**, or the only architecture available. ARM64 builds are separate FMAs with an `-arm64` slug suffix. Before adding, check that no input already uses this `package_identifier`; names alone give false negatives.
2. **Confirm the installer framework yourself** (`strings -n 8 setup.exe | grep -iE 'Inno Setup|Nullsoft|WixBundle|\.wixburn|InstallShield|Squirrel|velopack'`). winget's `InstallerType` is sometimes wrong, and the wrong silent switch shows up as a timeout, not an error.
3. **Verify identity from the installer**: the MSI Property table, the Burn manifest, the NSIS or Inno header, or `AppsAndFeaturesEntries` (often absent) ([identity-verification.md](references/identity-verification.md)). Set `program_publisher` whenever the registry Publisher differs from the winget locale Publisher (it often does). Don't ship an exe FMA whose DisplayName you couldn't confirm.
4. **Decide scope.** Machine-wide whenever the installer supports it, and prove the switch is honored. Per-user-only apps need the scheduled-task pattern ([windows-scripts.md](references/windows-scripts.md#when-the-app-has-no-machine-wide-mode)).
5. **Create `inputs/winget/<token>.json`**: `name`, `slug` (`<token>/windows`), `package_identifier`, `unique_identifier` (the verified DisplayName), `installer_arch`, `installer_type`, `installer_scope`, `default_categories`, plus `program_publisher`, `fuzzy_match_name`, `exists_query`, or `open_process_names` as needed ([Field semantics](#field-semantics)).
6. **Add custom install and uninstall scripts** for everything except a plain machine-scope MSI, starting from the closest existing pair in the table below. Read [windows-scripts.md](references/windows-scripts.md) before you trust them under SYSTEM.
7. **Generate, finalize, and [validate the queries](#validating-the-queries)**, on a Windows host as SYSTEM.

| Real framework | `installer_type` | Silent install | Uninstall | Start from |
|---|---|---|---|---|
| MSI / WiX | `msi` | auto (`msiexec /i … /quiet /norestart`) | auto upgrade-code (machine scope) | generated; custom if it needs `ALLUSERS=1` or vendor properties |
| NSIS / electron-builder | `exe` | `/S` (+ `/allusers` only if the build has a per-machine mode) | registry `UninstallString` + `/S` | `beekeeper-studio_*.ps1` |
| Inno Setup | `exe` | `/VERYSILENT /SUPPRESSMSGBOXES /NORESTART` | `unins000.exe` + same switches | `audacity_*.ps1` |
| WiX Burn bundle | `exe` | `/quiet /norestart` (some custom BAs reject `/norestart`) | the bundle's `QuietUninstallString`, not the inner MSI's key | `jabra-direct_uninstall.ps1` |
| InstallShield InstallScript MSI | `exe` | `/S /v/qn` (+ `/b"<temp dir>"` for 32-bit setups) | same switches | `mindmanager_install.ps1` |
| Squirrel / per-user only | `exe` | as the signed-in user, via a scheduled task | as the owning user | `signal_*.ps1` |
| MSIX | `msix` | provision + register per user | remove provisioned + registered | `slack_*.ps1` ([windows-scripts.md](references/windows-scripts.md#msix)) |
| Zip wrapping an installer | `zip` (+ `nested_installer_type`) | extract, then the inner framework's switches | inner framework | `agent-ransack_install.ps1` |

Script paths are under `ee/maintained-apps/inputs/winget/scripts/`. Any custom script that runs an installer treats `0`, `3010`, and `1641` as success, and must never drop `/norestart`-style switches: Fleet installs must not reboot the host.

## Second platform of an existing FMA

The library merges platforms by **slug token**, not by name (`SUBSTRING_INDEX(fma.slug, '/', 1)` in the list query, `slug.split("/")[0]` in the UI). So the new platform's `slug` is `<existing-token>/<platform>` even when the cask token or winget ID differs; put the real cask token in `token` (precedents: `libreoffice` → `libreoffice-still`, `ollama` → `ollama-app`). Reuse the existing `name` exactly: icons key off it. Docker Desktop, GitHub Desktop, Tailscale, Tableau, Wireshark, and Zen Browser show as two library rows because this was missed; don't add to that list.

## Validating the queries

Every output ships three queries. Fleet runs `exists` as the automatic-install policy and to link the app to inventory, `patched` as the patch policy, and `open` as the **Patch when closed** gate. A wrong query fails silently: a dead exists query never detects the app and makes the patch policy pass forever, a patched query that can't fail never flags an outdated host, and an open query that never matches lets Fleet patch over a running app. Run all of them yourself; reading SQL isn't verifying it.

**macOS**, on your Mac, without installing:

```bash
scripts/inspect_mac_installer.sh "<installer_url>" --keep           # prints the mounted app path
python3 -I scripts/check_queries.py ee/maintained-apps/outputs/<token>/darwin.json --app-path "<that path>"
```

Then the open query's running state: copy the app to a non-temp path under your home directory, launch it with `open -g`, confirm its process is still alive, rerun with `--app-path` pointing there (expect `open 0`), quit it (expect `open 1`), and delete the copy and its `~/Library` data. Keep that window to seconds and tell the user first: the app's windows (first-run or permission prompts) appear on their screen, and while the copy is registered, fleetd's inventory can report it. If the app is already installed on your Mac, drop `--app-path`.

**Windows**, on a Windows host or VM, as SYSTEM:

```bash
python3 -I scripts/check_queries.py ee/maintained-apps/outputs/<token>/windows.json --ps1 check.ps1
# On the host, after installing with the FMA's own install script as SYSTEM:
#   powershell -NoProfile -ExecutionPolicy Bypass -File check.ps1
```

Run it after install, then with the app open in the user's session, then after the uninstall script.

| Check | Expect | A wrong result means |
|---|---|---|
| `exists` after install | ≥ 1 row | wrong `unique_identifier`, publisher, version-embedded name, or scope ([queries.md](references/queries.md#exists-name-and-publisher-matching)) |
| `patched` on the new version | 1 row | the compared column doesn't track the manifest version ([queries.md](references/queries.md#when-the-version-column-doesnt-track-the-manifest)) |
| `patched@bumped` (literal raised by 1000) | 0 rows | the patch policy can never flag an outdated host |
| `patched` with the previous version installed | 0 rows; then 1 after installing the new one on top | upgrades don't converge (side-by-side installs, renamed ids) |
| `open` with the app closed / running / quit | 1 / 0 / 1 | wrong process name, a helper or service that outlives the window, nested apps ([queries.md](references/queries.md#open-run-it-with-the-app-running)) |
| `exists` after uninstall | 0 rows | the uninstall missed a copy or a registration |

The previous build's URL is in `git show origin/main:ee/maintained-apps/outputs/<token>/<platform>.json`. If you couldn't run a check, say which one in the PR.

## Finalize and open the PR

- Fill in the `apps.json` description: sentence case, "`<App>` is a(n) …", ending with a period. Check it parses: `python3 -m json.tool ee/maintained-apps/outputs/apps.json >/dev/null`.
- Compare the output's `sha256` with the manifest's, and check that `install_script_ref`/`uninstall_script_ref` point at your current scripts.
- **Icon**: look in `frontend/pages/SoftwarePage/components/icons/index.ts` for a `SOFTWARE_NAME_TO_ICON_MAP` key matching the lowercased `name`. The map serves two names, matched as a whole-word, case-insensitive prefix: the catalog `name` (Add software page) and the name osquery reports (inventory: the `.app` name, or the Windows DisplayName). Add a key for each when they differ, and keep old keys when an app is renamed. If the icon is missing, generate it with [tools/software/icons](../../../tools/software/icons), preferably from the real `.app` (`-a`); an icon from a third-party catalog needs a human look. Then check `git diff` of `index.ts`: the key must be the lowercased `name`, not the slug; the import must sit in the png import block; a token starting with a digit or containing `@` makes a bad identifier or path; and an existing component with the same name may have been overwritten (restore it). Keep imports and map entries in alphabetical order, and run the repo's eslint and prettier (2.2.1) on the icons directory. If an icon for that name already exists, revert any regenerated `.tsx`/`.png` and reuse it.
- **Upstream fixes**: when the real bug is in a cask (`zap`, `quit`) or a winget manifest, fix it there too. Homebrew allows one open AI-assisted PR per non-maintainer, wants the tool and model disclosed in the PR body (no AI trailers in commits), and its attestation checkbox is the human's to tick.
- Run `go test ./ee/maintained-apps/... ./cmd/maintained-apps/...` (guard tests catch stale helper copies, missing scope, and extraction or quit handling that doesn't stop), plus `GOOS=windows go build ./cmd/maintained-apps/validate/` for validator changes.
- Open the PR from `.github/pull_request_template.md`. Put what you verified and how, everything you couldn't verify, and catalog archaeology (what winget or Homebrew claimed vs. reality) in the PR body, not in script comments. For an identity fix, add an "Existing installs" note ([queries.md](references/queries.md#how-fixes-reach-existing-deployments)).
- Keep a human-authored PR under 100 changed files; split larger ones into PRs with disjoint file sets ([validator-failures.md](references/validator-failures.md#large-prs)). Nightly PRs opened by `fleet-release` aren't split: push fixes to the bot branch.

## What CI does and doesn't prove

PR validation (`test-fma-*-pr-only.yml`) installs each **changed** output on a GitHub runner, then:

- Finds the app with a loose search (macOS: `bundle_identifier LIKE '%<id>%'` or a name match; Windows: `name LIKE` on the catalog name and `unique_identifier`, or the exists clause) and checks its version loosely. Its `Found app:` log lines show what osquery really saw.
- On macOS, when the found `bundle_short_version` isn't exactly the manifest version, runs the **patched** query and fails if it returns no rows.
- On macOS, runs Gatekeeper (`spctl --assess`) on every `apps` row matching the exists clause, and fails on a rejection or on no rows.
- Runs the shipped **exists** query and fails with `exists query returned no rows with the app installed`. Two Windows exceptions only warn (`Exists query returned no rows: app was …`) and still pass: an app found only as a provisioned AppX package, and Codex CLI found only as a portable binary.
- Runs the uninstall script and checks the app is gone.

It doesn't prove:

- That the **open** query matches anything, or that **patched** can ever fail (no bumped-version or previous-version check).
- That the Windows patched query passes: the Windows validator still accepts a registry version that's a shorter prefix (`6.9` for `6.9.0.0`), which fails the policy.
- **SYSTEM context.** Windows runs as an interactive `runneradmin` with UAC off, on Windows Server (x64) or `windows-11-arm` (arm64 apps and `requires_client_os`). Per-user installers, session-0 dialogs, WOW64 profile redirection, and UAC prompts in scheduled tasks all pass in CI and break in the field.
- Upgrades over an existing install, same-version reinstalls, or an app that's running.
- Anything about frozen apps, or apps whose output didn't change: they aren't validated.

A green run means the installer downloaded, installed, registered under the exists query, and uninstalled on a clean runner. An app listed in `Some apps were validated with warnings` may not have matched its exists query at all; read its warnings. The rest is yours to prove.

## Field semantics

| Field | Meaning |
|-------|---------|
| `name` | Catalog display name and icon key. Reuse it exactly across platforms. |
| `slug` | `<token>/<platform>`. The token groups platforms into one library row and names the output directory and input file. |
| `token` (homebrew) | The cask token to fetch. Can differ from the slug token. |
| `unique_identifier` | Windows registry DisplayName / macOS CFBundleIdentifier. Changing it on a Windows app changes its software title key: GitOps deletes and re-adds the installer for existing fleets. |
| `installer_format` (homebrew) | What the download actually is: `dmg`, `pkg`, or `zip`. |
| `install_script_path` / `uninstall_script_path` | Repo-relative custom script, embedded verbatim. Custom winget uninstall scripts get `$PACKAGE_ID` substituted, nothing else. |
| `pre_uninstall_scripts` / `post_uninstall_scripts` (homebrew) | Lines wrapped around the generated uninstall. Not allowed with `uninstall_script_path`. |
| `cask_path` (homebrew) | Internal: points at `custom-tap/`. Not a way to add FMAs (ground rule 4). |
| `program_publisher` (winget) | Exists-query publisher when the registry Publisher ≠ the winget locale Publisher. |
| `fuzzy_match_name` (winget) | `true` → `name LIKE '<unique_identifier> %'`, for versioned DisplayNames. A string → `name LIKE '<string>'` verbatim (`"Mozilla Firefox % ESR %"`). Wrong in both directions: see [queries.md](references/queries.md#exists-name-and-publisher-matching). |
| `exists_query` (winget) | Replaces the generated exists query; the patched query is derived from it (a WHERE body with ` OR ` is parenthesized for you). |
| `open_process_names` / `open_process_name_prefix` (winget) | The processes the open query matches when the app's exe isn't `<name>.exe`. Set at most one. |
| `use_display_version_for_patch` (winget) | Compare against `AppsAndFeaturesEntries` `DisplayVersion` instead of `PackageVersion`. Hard-errors when the manifest has no DisplayVersion. |
| `installer_scope` (winget) | Must match a `Scope` in the manifest (an installer with no scope accepts the input's). It selects the manifest entry; it doesn't make the installer honor that scope. Always set it. |
| `installer_locale` (winget) | Only match installers with this `InstallerLocale` (unset matches any, so a manifest whose only installer is `ko-KR` gets picked silently). A root-level locale isn't read. |
| `nested_installer_type` (winget) | Pick among zip installers that differ only by what's inside (Paint.NET: `wix`). |
| `uninstall_type` (winget) | `msi`, with machine scope, gives a non-MSI `installer_type` the generated MSI uninstall (by upgrade code when the manifest has one). |
| `ignore_upgrade_code` (winget) | For vendors that mint a new UpgradeCode every build (QGIS). The title then links by `unique_identifier`, which must be distinct. |
| `ignore_hash` (winget) | SHA becomes `no_check`, for rolling URLs. It doesn't fix version drift ([validator-failures.md](references/validator-failures.md#rolling-latest-urls)). |
| `requires_client_os` (winget) | The installer refuses Windows Server; CI validates on `windows-11-arm` instead. |
| `frozen` | Output is never rewritten and CI skips the app; upstream is still fetched. |
| `patch_policy_path` | Dead code. Shape `exists_query` or add an ingester branch instead. |

## Pre-ship checklist

- [ ] Identity fields come from the real installer (Info.plist, MSI Property table, Burn manifest, NSIS/Inno header, `AppsAndFeaturesEntries`), not catalog names. Anything unverified is called out in the PR.
- [ ] macOS: the cask has an `app` artifact; `installer_format` matches the real download; Gatekeeper accepts the app; the bundle id is the top-level app's.
- [ ] Custom macOS scripts: helpers match `scripts.go`; a failed quit stops the script before the app is replaced; any mounted image is detached on every path.
- [ ] Second platform: slug token and `name` match the existing FMA.
- [ ] Windows: silent switches match the framework you confirmed; the uninstall uses the defensive `UninstallString` parser and picks the right key.
- [ ] **Scope proven, not assumed**: installed as SYSTEM on a real host, payload and registration where you expected, nothing under `S-1-5-18`/`.DEFAULT`. If you couldn't test it, the PR says so.
- [ ] Per-user apps: install runs as the signed-in user; uninstall finds every `HKEY_USERS` hive, runs the uninstaller as the owning user, and has the `system32` → `SysWOW64` fallback.
- [ ] **Queries run through real osquery**: exists ≥ 1, patched 1 and 0 when bumped, open 1/0/1 across closed/running/quit, exists 0 after uninstall.
- [ ] `fuzzy_match_name` matches the DisplayName you observed: set for versioned names, absent for plain ones.
- [ ] Custom script comments are short and admin-facing ([script-comments.md](references/script-comments.md)).
- [ ] `apps.json` description filled; `name`/`unique_identifier` there match the input; generated SHA matches; icon exists or was generated; `go test ./ee/maintained-apps/...` passes.
- [ ] Frozen app: the output was hand-edited, and the PR says what must happen before it can be unfrozen.
- [ ] Bootstrapper, per-user, rolling-URL, side-by-side-install, and unverified-identity risks are flagged in the PR.
- [ ] Every exit code you report was measured, not inferred.

## Reference files

| File | Read it when |
|---|---|
| [references/identity-verification.md](references/identity-verification.md) | Reading DisplayName/Publisher/bundle id/version/switches/process names from an installer, per framework, on a Mac |
| [references/queries.md](references/queries.md) | Fixing exists/patched/open, version-column overrides, `fuzzy_match_name`, sibling products, rebrands, and how fixes reach existing deployments |
| [references/macos-scripts.md](references/macos-scripts.md) | Writing or debugging a macOS install/uninstall script: quit and relaunch, DMG/zip/pkg handling, ownership, suites, rebrands |
| [references/windows-scripts.md](references/windows-scripts.md) | Writing or debugging a Windows script: SYSTEM context, per-user apps, hives and SIDs, MSIX, hangs, ARM64, testing in a VM, PowerShell traps |
| [references/validator-failures.md](references/validator-failures.md) | Any CI, ingest, or soak failure: error message → cause → fix, revert-and-freeze, rolling URLs, renamed casks, nightly PRs, hand-editing outputs |
| [references/script-comments.md](references/script-comments.md) | Writing or pruning comments in scripts that ship to customers |
