---
name: new-fma
description: Add, fix, or debug Fleet-maintained apps (FMAs) in ee/maintained-apps — Homebrew casks for macOS, winget packages for Windows. Use whenever a task touches FMA inputs, outputs, or scripts, even if the user never says "FMA": adding an app or the other platform of an existing one, macOS/Windows parity work, a failing FMA validator or nightly "Update Fleet-maintained apps" run, an install/uninstall script (including per-user Windows apps that break under SYSTEM), the exists/patched/open queries or patch policy, freezing an app, or pruning comments in a shipped script. Centers on verifying installer identity and behavior with real tools and real osquery instead of trusting catalog metadata.
allowed-tools: Bash, Read, Write, Edit, Grep, Glob, WebFetch, WebSearch
model: opus
effort: high
---

# Fleet-maintained apps

Task: $ARGUMENTS

An FMA is an input JSON in `ee/maintained-apps/inputs/homebrew/` or `inputs/winget/` (plus optional custom scripts), run through a generator that writes `ee/maintained-apps/outputs/<slug-token>/<platform>.json`: installer URL, SHA-256, install/uninstall scripts, and three osquery queries that Fleet runs on customer hosts. A CI validator installs each changed app on a GitHub runner and looks for it with osquery. The contributor basics are in [ee/maintained-apps/README.md](../../../ee/maintained-apps/README.md). This skill is the workflow plus the gotchas the README doesn't cover; where they disagree, follow this skill.

## Where to start

| You were asked to… | Go to |
|---|---|
| Add a macOS app | [Workflow: macOS](#workflow-macos) |
| Add a Windows app | [Workflow: Windows](#workflow-windows) |
| Add the other platform of an app that already has an FMA | [Second platform](#second-platform-of-an-existing-fma) first, then the platform workflow |
| Fix a failing validator run, nightly update PR, or ingest job | [references/validator-failures.md](references/validator-failures.md), indexed by error message |
| Check or fix the exists/patched/open queries or a patch policy | [references/queries.md](references/queries.md) |
| Write or fix a Windows install/uninstall script | [references/windows-scripts.md](references/windows-scripts.md): SYSTEM context, per-user apps, PowerShell traps |
| Clean up comments in a shipped script | [references/script-comments.md](references/script-comments.md) |

Whatever the task, finish with the [pre-ship checklist](#pre-ship-checklist).

## Golden rule: verify, don't guess

The biggest source of wasted validation cycles is trusting winget/Homebrew metadata for fields that must match what osquery sees on a host. **Catalog metadata (winget `PackageName`/`Publisher`, cask names, cask `zap` paths) frequently does not match the installed app's registry or bundle identity.** Confirm these against the real installer:

- **Windows `unique_identifier`** = the registry **DisplayName** (osquery `programs.name`).
- **Windows publisher** in the exists query = the registry **Publisher** (`programs.publisher`).
- **macOS `unique_identifier`** = the app's **CFBundleIdentifier**.
- **Version** must reconcile with what osquery reports: `programs.version` on Windows; `bundle_short_version` (or `bundle_version`) on macOS.
- **Where the app installs** under SYSTEM, not what the manifest's `Scope` claims.

Real cases where the metadata lied:

| App | winget/cask says | Registry/bundle actually is |
|-----|------------------|------------------------------|
| Amazon Corretto | PackageName "Amazon Corretto 25" | DisplayName `Amazon Corretto (x64)` (no version), Publisher `Amazon` |
| Genesys Cloud | PackageName "GenesysCloud" | DisplayName `GenesysCloud` (you'd guess "Genesys Cloud") |
| P4V | PackageName "P4 Apps", locale Publisher "Perforce Software, Inc." | DisplayName `P4 Apps`, Publisher `Perforce Software` |
| GoToMeeting | MSI ProductName "GoToMeeting 10.19.19950" | DisplayName `GoToMeeting 10.19.0.19950` (bootstrapper) |
| Tableau | locale Publisher `Tableau Software, LLC` | Publisher `Salesforce, Inc` (Burn bundle) |
| darktable | InstallerType `nullsoft`, Publisher "the darktable project" | Inno Setup, Publisher `darktable team`, DisplayName `darktable 5.6.0` |
| KiCad | cask zap `org.kicad-pcb.*` | bundle id `org.kicad.kicad` |

How to read each field from a real installer, per installer type, without a Windows host: [references/identity-verification.md](references/identity-verification.md). If you can't confirm a field, say so in your report rather than shipping a guess.

## Prerequisites

```bash
brew install msitools sevenzip   # msiinfo for MSIs, 7zz for NSIS payloads and Burn cabinets
gh auth status                   # gh reads winget-pkgs manifests
```

Bundled helpers in [scripts/](scripts/): `check_queries.sh` (run a macOS output's queries through osquery), `pe_info.py` (a Windows installer's architecture and ProductVersion from a partial download), `burn_ux.py` (a Burn bundle's ARP identity from a 30 MB range request).

## How the generator behaves

```bash
go run cmd/maintained-apps/main.go --slug="<token>/<platform>" --debug
```

- It fetches the upstream cask or winget manifest and writes `outputs/<token>/<platform>.json`. Scripts are embedded under `refs`, keyed by `sha256(script)[:8]`, so editing a script means regenerating (or recomputing the ref by hand).
- It appends new apps to `outputs/apps.json` with an **empty description**: fill it in, sentence case, "`<App>` is a(n)…". It never updates `unique_identifier` on an existing entry and never removes entries.
- Regenerating can bump the app's version, because it refetches upstream. If you only meant to change a script, check the diff for an unrelated version bump; [references/validator-failures.md](references/validator-failures.md#editing-an-output-without-regenerating) has the hand-edit recipe.
- **`frozen: true` only stops the output from being rewritten.** The generator still fetches upstream for every input, so a renamed or removed cask still breaks the run. Re-running it on a frozen app silently leaves the old output, stale script refs included: if you edit a frozen app's scripts, hand-edit its output.

## Workflow: macOS

1. **Fetch the cask** (`curl -s https://formulae.brew.sh/api/cask/<token>.json`) and read `artifacts`:
   - **It needs an `app` artifact.** The macOS exists query is `SELECT 1 FROM apps WHERE bundle_identifier = '…'`, and osquery's `apps` table only inventories `.app` bundles. A pkg-only cask (JDKs, runtimes, drivers, daemons, CLI tools, prefPanes) cannot be a working FMA: its exists and patched queries would never match. Stop and report that instead of shipping it.
   - A `suite` (a folder such as `/Applications/KiCad`) or `artifact` stanza isn't parsed by the ingester, so it needs custom scripts. See [identity-verification.md](references/identity-verification.md#macos-installers).
   - Arch-split casks: the API's top-level `url` is the arm64 build. Many macOS FMAs ship it; that's expected.
2. **Download and inspect the real installer** ([identity-verification.md](references/identity-verification.md#macos-installers)):
   - `file` the download. `installer_format` must match the file you actually get (`dmg`/`pkg`/`zip`), not the cask's artifact shape: a `dmg` format on a bare pkg makes the generated `hdiutil attach` fail. A zip that contains a dmg needs a custom install script (`inputs/homebrew/scripts/pd-install.sh`).
   - Read `CFBundleIdentifier` (→ `unique_identifier`), `CFBundleShortVersionString`, and `CFBundleVersion` from the app's `Info.plist`.
   - **Compare both versions to the cask `version`.** The generated patched query compares `bundle_short_version`. If that's missing or is a marketing version (`3.8.7` for cask `3.8.7.19194`), the patch policy fails on every host forever. The validator still passes, because it accepts a match on either column. Fix it with a per-token branch in `ee/maintained-apps/ingesters/homebrew/ingester.go` ([queries.md](references/queries.md#when-the-version-column-doesnt-track-the-manifest)).
3. **Create `inputs/homebrew/<token>.json`**: `name`, `slug` (`<token>/darwin`), `unique_identifier`, `token` (the cask token, which can differ from the slug), `installer_format`, `default_categories`. Install/uninstall scripts auto-generate from the cask's artifacts and `zap`. To change them, add per-app scripts in `inputs/homebrew/scripts/` via `install_script_path`/`uninstall_script_path` rather than editing the shared template in `ingesters/homebrew/scripts.go`, which regenerates every macOS FMA. `pre_uninstall_scripts`/`post_uninstall_scripts` can't be combined with a custom uninstall script.
   - **A custom install script starts from the generated one**, with its helpers copied verbatim from `scripts.go`. Call `quit_and_track_application '<bundle id>' || exit 1` before anything moves or replaces the app, and detach any disk image the script still has mounted before exiting. The helper returns 1 when the app won't quit (for example, the user cancels a save prompt), and an app replaced while it's running loses its files when fleetd deletes `$TMPDIR`. A copied helper that confirms the quit with `pgrep -f "$bundle_id"` is stale: that never matches, because an app's command line is its executable path.
   - **Changing a helper in `scripts.go` means syncing its copies** in `inputs/homebrew/scripts/`. Grep for the old line as well as the function name: older scripts carry their own `quit_application` variants.
4. **Generate, finalize, and run the queries** through osquery: [Finalize](#finalize), then [queries.md](references/queries.md).

## Workflow: Windows

1. **Read the winget manifests** ([identity-verification.md](references/identity-verification.md#winget-manifests)). Pick **machine** scope and **x64**, or the only architecture available. ARM64 builds are separate FMAs with an `-arm64` slug suffix (README).
2. **Verify identity from the installer**: the MSI Property table, or for an exe, `AppsAndFeaturesEntries` and then the installer itself. Set `program_publisher` whenever the registry Publisher differs from the winget locale Publisher (it often does). Don't ship an exe FMA whose DisplayName you couldn't confirm.
3. **Confirm the installer framework yourself** (`strings setup.exe | grep -iE 'Inno Setup|Nullsoft|WixBundle|InstallShield'`). winget's `InstallerType` is sometimes wrong, and the wrong silent switch shows up as an 11-minute timeout, not an error.
4. **Decide scope.** Machine-wide whenever possible, and verify the all-users switch is honored. Per-user-only apps need the scheduled-task pattern in [windows-scripts.md](references/windows-scripts.md).
5. **Create `inputs/winget/<token>.json`**: `name`, `slug` (`<token>/windows`), `package_identifier`, `unique_identifier` (the verified DisplayName), `installer_arch`, `installer_type`, `installer_scope`, `default_categories`, plus `program_publisher`, `fuzzy_match_name`, or `exists_query` as needed ([Field semantics](#field-semantics)).
6. **Add custom install and uninstall scripts** for everything except machine-scope MSI, starting from the closest existing pair (table below), then read [windows-scripts.md](references/windows-scripts.md) before you trust them under SYSTEM.
7. **Generate, finalize, and verify the queries**: [Finalize](#finalize), then [queries.md](references/queries.md).

| Real framework | FMA `installer_type` | Silent install | Uninstall | Start from |
|---|---|---|---|---|
| MSI / WiX | `msi` | auto (`msiexec /i … /quiet /norestart`) | auto upgrade-code (machine scope only) | generated |
| NSIS (incl. electron-builder) | `exe` | `/S` (+ `/allusers` for machine scope) | registry UninstallString + `/S` | `beekeeper-studio_*.ps1` |
| Inno Setup | `exe` | `/VERYSILENT /SUPPRESSMSGBOXES /NORESTART` | registry UninstallString + same | `audacity_*.ps1` |
| WiX Burn bundle | `exe` | `/quiet /norestart` | the bundle's QuietUninstallString, not the inner MSI's key | `jabra-direct_uninstall.ps1` |
| InstallShield InstallScript MSI | `exe` | `/S /v/qn` | same switches | `mindmanager_install.ps1` |
| Per-user only (any) | `exe` | as the signed-in user, via scheduled task | as the owning user | `signal_*.ps1` |
| MSIX | `msix` | n/a | n/a | generated |

Script paths are under `ee/maintained-apps/inputs/winget/scripts/`. Only machine-scope MSI gets auto-generated scripts. MSI success codes are `0`, `3010` (reboot required), and `1641` (reboot initiated). Treat all three as success in any custom script that runs an installer; otherwise a reboot-pending host fails the install.

## Second platform of an existing FMA

The FMA library merges platforms by **slug token**, not by name: the list query groups on `SUBSTRING_INDEX(fma.slug, '/', 1)` and the UI on `slug.split("/")[0]`. So the new platform's `slug` must be `<existing-token>/<platform>` even when the cask token or winget ID differs. Put the real cask token in `token` (precedents: `libreoffice` → `libreoffice-still`, `ollama` → `ollama-app`, `zoom` → `zoom-for-it-admins`). Reuse the existing `name` exactly too: icons key off the lowercased `name`, so a shared name reuses the existing icon. Docker Desktop, GitHub Desktop, Tailscale, Wireshark, and Zen Browser show up as two library rows because this was missed. Don't add to that list.

## Finalize

- Fill in the `apps.json` description; check it with `python3 -m json.tool ee/maintained-apps/outputs/apps.json >/dev/null`.
- Compare the output's `sha256` with the manifest's. For Windows, `install_script_ref`/`uninstall_script_ref` should point at your current scripts.
- **Icon**: look in `frontend/pages/SoftwarePage/components/icons/index.ts` for a `SOFTWARE_NAME_TO_ICON_MAP` key equal to the lowercased `name`. If it's missing, generate one with [tools/software/icons](../../../tools/software/icons) and check that the new key is the lowercased `name`, not the slug. Insert imports and map entries in alphabetical order rather than at the end. If an icon for that name already exists, revert any regenerated `.tsx`/`.png` and reuse it.
- **Run the queries through real osquery** ([queries.md](references/queries.md)). CI never runs them.
- If you changed shared code (`cmd/maintained-apps/validate/*.go`, the ingesters, `pkg/patch_policy`), say so in the PR and run `go test ./cmd/maintained-apps/... ./ee/maintained-apps/...`, plus `GOOS=windows go build ./cmd/maintained-apps/validate/` for validator changes.

## What CI does and doesn't prove

- **The validator searches loosely.** On Windows it uses `LOWER(name) LIKE '%<name>%'`; on macOS, `bundle_identifier LIKE '%<id>%'` or a name match. It finds apps that the shipped exact `exists` query never would. Compare its `Found app: '…'` log line with the identity you're shipping.
- **It never runs the shipped exists/patched/open queries.**
- **On macOS it accepts a version match on either `CFBundleShortVersionString` or `CFBundleVersion`** (`checkVersionMatch` in `cmd/maintained-apps/validate/darwin.go`), while the patch policy compares only one of them.
- **On Windows it runs as an interactive `runneradmin`, not SYSTEM.** Per-user installers land in a normal profile and pass while broken in the field (`signal/windows` did).
- **It only validates changed outputs** (`.github/scripts/detect-new-fmas-in-pr.sh`), skips frozen apps, and runs on an ephemeral host that you can't query afterward.

A green run means the installer downloaded, installed, and something with a similar name appeared. The rest is yours to prove.

## Field semantics

| Field | Meaning |
|-------|---------|
| `name` | Catalog display name and icon key. Reuse it exactly across platforms. |
| `slug` | `<token>/<platform>`. The token groups platforms into one library row and names the output directory. |
| `token` (homebrew) | The cask token to fetch. Can differ from the slug token. |
| `unique_identifier` | Windows registry DisplayName / macOS CFBundleIdentifier. |
| `installer_format` (homebrew) | What the download actually is: `dmg`, `pkg`, or `zip`. |
| `install_script_path` / `uninstall_script_path` | Repo-relative custom script, embedded verbatim. |
| `pre_uninstall_scripts` / `post_uninstall_scripts` (homebrew) | Lines wrapped around the generated uninstall. Not allowed with `uninstall_script_path`. |
| `program_publisher` (winget) | Exists-query publisher when the registry Publisher ≠ the winget locale Publisher. |
| `fuzzy_match_name` (winget) | `true` → `name LIKE '<unique_identifier> %'`, for versioned DisplayNames. A string → `name LIKE '<string>'` verbatim (`"Mozilla Firefox % ESR %"`). Wrong in both directions: see [queries.md](references/queries.md). |
| `exists_query` (winget) | Replaces the generated exists query; the patched query is derived from it. |
| `use_display_version_for_patch` (winget) | Compare against the manifest's `DisplayVersion` instead of `PackageVersion` (python.org registers `3.14.5150.0` for `3.14.5`). |
| `installer_scope` (winget) | Must match a `Scope` in the manifest. It selects the manifest entry; it doesn't make the installer honor that scope. |
| `ignore_hash` (winget) | SHA becomes `no_check`, for rolling URLs. It doesn't fix version drift ([validator-failures.md](references/validator-failures.md#rolling-latest-urls)). |
| `frozen` | Output is never rewritten; upstream is still fetched. |
| `patch_policy_path` | Dead code. Shape `exists_query` or add an ingester branch instead. |

## Pre-ship checklist

- [ ] Identity fields come from the real installer (MSI Property table, Burn manifest, Inno/NSIS metadata, `AppsAndFeaturesEntries`, Info.plist), not from catalog names. Anything unverified is called out in the PR.
- [ ] macOS: the cask has an `app` artifact; `installer_format` matches the real download; `bundle_short_version` tracks the cask version, or a per-token override was added.
- [ ] macOS custom install script: helpers match `scripts.go`, and a failed `quit_and_track_application` stops the script before the app is replaced.
- [ ] Second platform: slug token and `name` match the existing FMA.
- [ ] Windows: silent switches match the framework you confirmed; custom uninstall uses the defensive UninstallString parser.
- [ ] **Scope proven, not assumed**: installed on a host as SYSTEM, and the payload and registration were where you expected. Nothing under `S-1-5-18`/`.DEFAULT`. If you couldn't test it, the PR says so.
- [ ] Per-user apps: install runs as the signed-in user; uninstall enumerates every `HKEY_USERS` hive, runs the uninstaller as the owning user, and has the `system32` → `SysWOW64` fallback.
- [ ] **Queries run through real osquery with the app installed**: `exists` ≥ 1 row; `patched` 1 row as generated and **0 rows with the version bumped**; `open` 0 rows while the app runs and 1 after it quits.
- [ ] `fuzzy_match_name` matches the DisplayName you observed: set for versioned names, absent for plain ones.
- [ ] Custom script comments are short and admin-facing ([script-comments.md](references/script-comments.md)).
- [ ] `apps.json` description filled; generated SHA matches; icon exists or was generated.
- [ ] Frozen app: the output was hand-edited, because the generator skipped it.
- [ ] Bootstrapper, per-user, rolling-URL, and unverified-identity risks are flagged in the PR.
- [ ] Exit codes you report were actually measured (no blank `ExitCode` from `-PassThru` + `WaitForExit($ms)`).

## Reference files

| File | Read it when |
|---|---|
| [references/identity-verification.md](references/identity-verification.md) | Reading DisplayName/Publisher/bundle id/version/switches from an installer, per framework, without a Windows host |
| [references/windows-scripts.md](references/windows-scripts.md) | Writing or debugging any Windows install/uninstall script: SYSTEM context, per-user apps, hives and SIDs, install-script patterns, PowerShell traps |
| [references/queries.md](references/queries.md) | Verifying or fixing exists/patched/open, version-column overrides, `fuzzy_match_name`, sibling products |
| [references/validator-failures.md](references/validator-failures.md) | Any CI or ingest failure: error message → cause → fix, revert-and-freeze, rolling URLs, renamed casks, hand-editing outputs |
| [references/script-comments.md](references/script-comments.md) | Writing or pruning comments in scripts that ship to customers |
