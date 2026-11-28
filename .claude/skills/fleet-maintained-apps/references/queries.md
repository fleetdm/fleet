# Verifying and fixing the queries

The step-by-step checks are in [SKILL.md](../SKILL.md#validating-the-queries). This file explains how each query is built, how to run them in harder cases, and how to fix each kind of failure.

- [How the queries are built](#how-the-queries-are-built)
- [Running them](#running-them)
- [`patched`: prove it can fail](#patched-prove-it-can-fail)
- [When the version column doesn't track the manifest](#when-the-version-column-doesnt-track-the-manifest)
- [`exists`: name and publisher matching](#exists-name-and-publisher-matching)
- [Rebrands, renamed ids, and side-by-side installs](#rebrands-renamed-ids-and-side-by-side-installs)
- [`open`: run it with the app running](#open-run-it-with-the-app-running)
- [How fixes reach existing deployments](#how-fixes-reach-existing-deployments)

## How the queries are built

| Query | Fleet uses it for | Must return a row when | Must return no rows when |
|-------|-------------------|------------------------|--------------------------|
| `exists` | the automatic-install policy (Fleet installs where it fails); linking the app to inventory | the app is installed | it isn't |
| `patched` | the patch policy | the host is up to date, or the app is absent | an older build is installed |
| `open` | the **Patch when closed** pre-install gate | the app is closed, or absent | the app is running |

- **`exists`** on macOS is `SELECT 1 FROM apps WHERE bundle_identifier = '<id>'`, with a few per-token variants in `ingesters/homebrew/ingester.go` (a path filter for swiftDialog and QGIS, nested-bundle exclusion for Syncovery, a two-id match for ChatGPT). On Windows it's `programs` matched on `name` (exact, or `LIKE` via `fuzzy_match_name`) and `publisher`, unless `exists_query` replaces it.
- **`patched`** is `SELECT 1 WHERE NOT EXISTS (<exists body> AND version_compare(<column>, '<version>') < 0)`. It only looks at rows the exists body matches, so an identity bug that blanks `exists` makes `patched` pass forever, and a column that doesn't track the manifest version makes it fail forever or pass forever.
- **`open`** on macOS joins `apps` to `processes` on exactly `<app path>/Contents/MacOS/<CFBundleExecutable>` and needs no per-app data. On Windows it's `LOWER(name) = '<lowercased catalog name>.exe'`, unless the input sets `open_process_names` or `open_process_name_prefix`. A multi-word catalog name guesses wrong by construction (`'colour contrast analyser.exe'`).
- An arm64 Windows FMA's policies are wrapped so they pass on hosts that aren't ARM64 (`ScopeToWindowsARM`), so test them on an ARM64 host.
- `version_compare` has no notion of "unknown" or of zero-padding: `version_compare('4.9', '4.9.0') = -1`, `''` sorts below everything, and `version_compare(NULL, x)` is an error ("Must provide two version strings"), not false.

## Running them

`scripts/check_queries.py` covers the normal cases ([SKILL.md](../SKILL.md#validating-the-queries)). It runs osqueryd with `-S --disable_extensions`: without that flag, a fleetd-enrolled Mac autoloads Fleet's extensions into every ad hoc run and spams registry conflicts.

**macOS without installing.** osquery's `apps` table reads any bundle you name in a `path` constraint, which is what `--app-path` injects (and it turns the open query's `JOIN` into a `CROSS JOIN`, or SQLite reruns the `apps` generator once per process row). Point it at the bundle inside a mounted DMG, an extracted zip, or an expanded pkg payload. A query that pins `path` to its install location (QGIS, XQuartz, WezTerm) can't match a bundle anywhere else, so `check_queries.py` skips it and says so: test that one with the app installed at that path. Without a path constraint, `apps` also lists bundles LaunchServices knows about anywhere (`~/Downloads`, caches, build directories), not just `/Applications`: keep that in mind before writing an "old id anywhere" clause.

**The open query's running state on macOS.** LaunchServices won't resolve `open -b <id>` or `application id "<id>"` for a bundle under `/private/tmp` or `$TMPDIR` (where DMGs and scratch copies live), even after `lsregister -f`. Copy the app to a path under your home directory (`~/Library/Caches/<probe>/`), `lsregister -f` it, launch it, run the checks, then quit it, `lsregister -u` it, and delete the copy along with whatever the app wrote to `~/Library`. Confirm the process is still alive when you run the "running" check: some apps exit on their own a few seconds after launch, and closing a first-run window can quit the app (Rectangle's "Authorize" window), which invalidates it. Launch with `open -g`, run the check, and quit within seconds. A registered copy anywhere is listed by `apps`, so fleetd's inventory can pick it up while it exists.

**Windows.** Install with the FMA's own script as SYSTEM, then run the `.ps1` that `check_queries.py --ps1` writes, as SYSTEM, with `-File`. Nesting the SQL's single quotes through `-Command`, `-EncodedCommand`, or a bash heredoc breaks every time. The generated script handles the parsing traps you'd otherwise hit in your own: `osqueryi --json` prints an empty result as `[`, blank lines, `]` (not `[]`), and `ConvertFrom-Json` turns that into one `$null`, so a naive count reads 1; Windows PowerShell 5.1's `ConvertFrom-Json` emits a parsed array as one object, so counting straight off the pipeline reads 1 for any row count; and a warning on stderr merged in by `2>&1` makes `ConvertFrom-Json` fail, which reads as 0 rows. It's written with a BOM, because Windows PowerShell 5.1 reads a BOM-less script as ANSI and mangles non-ASCII literals (`Cédric Trévisan`, `ETH Zürich`). For the running state, launch the app in the signed-in user's session, not as SYSTEM.

If you can't run a query at all, say so in the PR. If the CI log is your only Windows evidence, its `Found app: '<DisplayName>' … Version: …` lines are post-`MutateSoftwareOnIngestion`, after inventory's sanitizers rewrote the name or version.

## `patched`: prove it can fail

1. `exists` → at least one row.
2. `patched` as generated, on an up-to-date install → one row. Zero rows on a host that really is behind is the policy working; upgrade it first.
3. `patched@bumped` (`check_queries.py` raises every `version_compare` literal by 1000 in its first number) → **zero rows**. A row here means the compared column isn't something the manifest version can order against, or the identity clause matches nothing.
4. The previous build. `git show origin/main:ee/maintained-apps/outputs/<token>/<platform>.json | jq -r '.versions[0].installer_url'` gives its URL. Install it and expect zero rows, then install the new build on top, with no uninstall in between (that's what Fleet's remediation does), and expect one row. This is the check that catches side-by-side installs and renamed bundle ids.

Then look at the column itself:

```sql
-- macOS
SELECT bundle_short_version, bundle_version, path FROM apps WHERE bundle_identifier = '<id>';
-- Windows
SELECT name, publisher, version, install_location FROM programs WHERE name LIKE '%<name>%';
```

## When the version column doesn't track the manifest

The shapes that break `version_compare`:
- `''` (no `CFBundleShortVersionString`): fails forever (Steam, Axure RP).
- A shorter version: `26.3` for cask `26.3.5`, `4.6` for `4.6.0`, `6.9` for winget `6.9.0.0`. Fails on every current host, including x.y.0 releases, because there's no zero-padding.
- A marketing version below a build-suffixed manifest version (`3.8.7` vs `3.8.7.19194`: i1Profiler, Sonos): fails forever.
- A longer registry version (`3.14.5150.0` vs `3.14.5`: python.org): sorts *above*, so outdated hosts within the same minor version pass.
- Descriptive text (R's `R R 4.6.1 GUI 1.83 …`) or a placeholder (`1`, WezTerm): meaningless.
- A 4-part registry version against a 3-part manifest version with the same prefix (`28.1.7.13231` vs `28.1.7`) is not `< 0`: no fix needed.

On macOS, CI now catches the shorter, missing, and marketing cases: when the installed `bundle_short_version` isn't the manifest version, the validator runs the patched query and fails with an error that names the fix. On Windows it still accepts a shorter registry version, so check it yourself. Fix each case at the app, never in the shared generator:

- **macOS, the short version drops a build or a trailing `.0`** (`9.5.0` for `9.5.0.4590`, `4.6` for `4.6.0`): shorten the manifest version with a transformer in `ingesters/homebrew/external_refs` (`makeVersionShortener(n)`, `makeTrailingZeroTrimmer(n)`), so inventory and the library show the same version. Punctuation differences (`0.56-3` vs `0.56.3`) get a transformer too (Pd, Sublime Text).
- **macOS, `CFBundleVersion` carries the full cask version, or the short version is missing or a marketing version**: add the token to the `bundle_version` case in `ingesters/homebrew/ingester.go` (4K apps, JASP, Malwarebytes, Disk Drill, Steam, i1Profiler). That case compares against the cask version, not a shortened one, so builds within one short version still fail correctly. For descriptive strings, use `version_compare(COALESCE(regex_match(<col>, '<pattern>', 1), <col>), '<ver>')`: `regex_match` returns NULL on no match, so the `COALESCE` is load-bearing (R). Add a matching `basicAppSanitizers` entry in `server/service/osquery_utils/queries.go` so inventory and the validator show the clean version.
- **Windows**: `use_display_version_for_patch` in the input when `AppsAndFeaturesEntries` has the registry `DisplayVersion` (it hard-errors when it doesn't). Otherwise add a per-slug enricher in `ingesters/winget/external_refs` that rewrites the version (AnyBurn registers `6.9` for `6.9.0.0`; Geany `2.1` for `2.1.0`; 1Password drops the fourth part), or shape `exists_query`.
- **JetBrains on Windows**: the registry version is a build number (`261.24374.185`), but `MutateSoftwareOnIngestion` rewrites inventory to the marketing version in the name (`PhpStorm 2026.1.2` → `2026.1.2`) when the publisher contains "jetbrains". The Windows validator sets `Software.Vendor` so this fires in CI too.

Two cases that look broken and aren't:
- **Build-only bumps on comma-versioned casks** (`1.7,9010` → `1.7,9059`): the ingester versions by the part before the comma, so Fleet shows one version for both builds and the policy passes on the old build. That's by design (about 170 macOS FMAs); don't add `bundle_version` overrides for it. Servers refresh a same-version rebuild's installer in place on a hash change.
- **Not every version mismatch is wrong**: an override is only right when the policy can't pass on an up-to-date host, or every release shares one short version (Firefox Nightly).

After any override, rerun the four checks against it. `patch_policy_path` in the input structs is dead code: the patched query can only change through `exists_query` or an ingester branch.

## `exists`: name and publisher matching

**Versioned vs. plain DisplayNames.** Many installers register `"<Name> <version>"`: electron-builder by default (`Signal 8.24.1`, `Twine 2.12.1`, `draw.io 32.6.10`), Inno's `AppVerName`, NSIS `Name "<product> <version>"` (Geany). An Inno `.iss` with `AppName` + `AppVersion` and neither `UninstallDisplayName` nor `AppVerName` registers `"<AppName> version <AppVersion>"` (Ollama: needs `fuzzy_match_name: "Ollama version %"`). All of these need `fuzzy_match_name`. But plenty register a plain name (`Asana`, `Discord`, `KNIME Analytics Platform`, `NordVPN`) and must keep an exact match: `name LIKE 'Asana %'` never matches `Asana`. Observe the DisplayName, then decide. Inventory showing the app under the right title proves nothing when the app has `fuzzy_match_name` or `exists_query`: ingestion then matches titles by prefix, the exists query doesn't. Without either, the ingester marks the app `exact_name_match` in `apps.json`, and Fleet files only programs named exactly `unique_identifier` under its title.

**Publishers.** The generator uses the winget locale `Publisher` unless `program_publisher` is set, and the locale value is often a person, a parent company, an old name, or truncated (`Atomi Systems` vs the registry's `Atomi Systems, Inc.`; `Benn Huang` vs Chatbox's `Mediocre Company`; `Tableau Software, LLC` vs `Salesforce, Inc`). When a vendor renames its publisher between versions (Autopsy `The Sleuth Kit` → `Sleuth Kit Labs`; Evernote `Evernote Corporation` → `Bending Spoons US Inc.`), match both names in `exists_query` (`publisher IN (…)`), so hosts on the old version still fail the patch policy. When a Burn bundle's Publisher differs from its inner MSI's Manufacturer, use `exists_query` with `publisher LIKE '%<Vendor>%'`: osquery returns both rows.

**Odd DisplayNames.**
- Trailing spaces: EDB's PostgreSQL registers `PostgreSQL 18 ` → `exists_query` with `TRIM(name) = 'PostgreSQL 18'`.
- NUL-padded `REG_SZ` values (Fork: `"Fork\0\0\0…"`): give the exists query a second look on a real host.
- Arch suffixes: REAPER registers `REAPER` (x86), `REAPER (x64)`, and `REAPER (arm64)` side by side. Match the exact name your installer writes. A fuzzy prefix would also match another arch's copy, which fails the policy forever while the FMA installs alongside it.
- Version and arch in the name (`Beats filebeat 8.x (x86_64)`, `Pale Moon <ver> (x64 en-US)`): a string `fuzzy_match_name` pattern.

**Sibling products that share a name.**
- Corretto 21 and 25 both register as `Amazon Corretto (x64)`: pin each with `exists_query … AND version LIKE '<major>.%'`.
- IntelliJ Ultimate's `IntelliJ IDEA <ver>` also matches `IntelliJ IDEA Community Edition <ver>`; Chatbox's `Chatbox <ver>` also matches the old `Chatbox CE <ver>`. Exclude the sibling (`AND name NOT LIKE 'Chatbox CE %'`).
- Vendors rename ARP entries between versions (VC++ `… 2015-2022 …` → `… v14 …`). Match both names.

**Changing `unique_identifier` on Windows** changes the software title key: GitOps deletes and re-adds the installer for existing fleets. When customers depend on the app, keep `unique_identifier` and put the corrected match in `exists_query`.

**Scope.** Every Windows exists query is scope-blind (all hives) on purpose. Never "fix" a per-user vs. per-machine problem by narrowing the query to one scope: the policy goes green while a stale copy in the other scope stays unmanaged. Fix remediation in the install script instead (remove the other-scope copy, then install).

## Rebrands, renamed ids, and side-by-side installs

**A bundle id that changes at the same path** (RoboForm 10: `com.SiberSystems.RoboForm` → `com.sibersystems.RoboFormMac`): per-token branches in `ingesters/homebrew/ingester.go` make `patched` flag the old id (or the new id below the version) and `open` match both ids. The install script quits both ids (`roboform-install.sh`), and the uninstall gets `pre_uninstall_scripts: ["quit_application '<old>' || exit 1"]`. After the swap, `open -b <new id>` can fail because the user's LaunchServices database has never seen it: register the new bundle as the console user before relaunching ([macos-scripts.md](macos-scripts.md#quit-and-relaunch)). Before adding legacy-id handling, confirm an FMA-shipped build actually carried the old id: GIMP, ForkLift, and Mellel were wrong or changed before any FMA host had them, and needed only the id fix.

**A rebrand that installs a new app next to the old one** (RealVNC Viewer → Connect Viewer):
- `exists` matches the **new id only**. It's the automatic-install policy, and Fleet installs where it fails: matching both ids would make legacy hosts look installed and skip them.
- `patched` flags the old id **or** an outdated new id: `NOT EXISTS (… WHERE bundle_identifier = '<old>' OR (bundle_identifier = '<new>' AND version_compare(…) < 0))`.
- The install script must remove the legacy app after a successful install, or the old id keeps the policy red forever. Check the vendor's pre/postinstall first; it may already do it.

**Side-by-side versioned bundles** (`Tableau Desktop 2026.3.app` next to `2026.1.app`, kept on purpose to match a server version): the standard patched query stays red after every upgrade. Tableau's branch uses "newest copy counts": `NOT EXISTS (id) OR EXISTS (id AND version_compare(…) >= 0)`. Raise this convergence question before changing ids.

**Windows publisher or name renames** are the same idea: match both in `exists_query`, and remove the legacy product after the new one installs (Audacity's Inno → MSI switch installed side by side under a new UpgradeCode).

## `open`: run it with the app running

Check three states: app closed → one row; launch it → **zero rows**; quit it → one row again. A query that stays at zero after quitting matches a helper, updater, tray process, or service that outlives the window. One that stays at one while the app is up matches nothing, and Patch when closed silently patches over the running app. No `open` query at all means the toggle does nothing.

**macOS** needs no per-app data, but still check it, because the join needs the live process path to be exactly the main executable. Known exceptions with per-token overrides in the homebrew ingester: Chromium browsers relaunch from a `.code_sign_clone` after self-updating; KiCad's editors run from nested bundles under `Contents/Applications/`; MindManager's bundle id carries the major version (`GLOB`); ChatGPT matches either id at one path.

**Windows**: get the real executable name, not the catalog name.
- Live: `Get-Process | Where-Object Path -like 'C:\Program Files\<Vendor>*' | Select-Object Name, Path` while the app is open, in the user's session.
- MSI: `msiinfo export app.msi Shortcut` (column 5, `[#_7zFM.exe]`) or `msiinfo export app.msi File`.
- MSIX: `unzip -p app.msix AppxManifest.xml | grep Executable=`.
- NSIS: `7zz l app.exe` (electron-builder nests the app in `$PLUGINSDIR/app-64.7z`).
- Burn: carve the attached cabinet and read the inner MSI's Shortcut table ([identity-verification.md](identity-verification.md#wix-burn-bundle)).

Pick the GUI executable(s) the user has open, including separately named editions (`Code - Insiders.exe`) and renamed majors (`audacity.exe`, `Audacity4.exe`). Leave out updaters (`GUP.exe`) and short-lived CLIs (`7z.exe`). NetBeans-platform apps run the JVM inside `<app>64.exe`. An app whose own service runs the same executable permanently (WireGuard's `wireguard.exe /managerservice`) can't be separated by name; note in the PR that Patch when closed rarely passes for it. Set `open_process_names` (or `open_process_name_prefix` for a family of names), regenerate, and confirm the output's `open` changed. The README's "Editing an app's pre-install query" section covers replacing the whole query per slug.

## How fixes reach existing deployments

Servers pull manifests from `maintained-apps.fleetdm.com` (fallback: raw main), so a merged fix is live without a release. What happens next on a server:
- The hourly auto-update refreshes `software_installers` scripts and queries for unpinned and caret-pinned fleets, including a same-version manifest change (keeping admin-edited scripts). `SyncPatchPolicyQueries` then regenerates each patch policy's query from its active installer. Literal version pins keep their old metadata by design.
- The automatic-install policy (`exists`) is never regenerated after it's created.
- A software title stays keyed the way it was created. If adding the FMA created a title on a wrong bundle id, the installer stays on that title after the fix, while inventory lands on the right one; only removing and re-adding the FMA moves it. That's server-side and out of scope for an FMA PR. A wrong macOS bundle id also makes GitOps report the FMA as deleted software on every run.
- FMAs that share a bundle id or a Windows upgrade code (GovCloud variants, Firefox and ESR, Acrobat Reader and Pro) share one software title, and can't both be added to one fleet.

So for an identity fix, add an "Existing installs" note to the PR that says which of these apply, rather than changing code to chase them. Reviewers (including CodeRabbit) often flag "no migration for stranded installs" or "exact pins won't update"; answer with this scope.
