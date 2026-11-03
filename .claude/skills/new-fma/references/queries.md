# Verifying and fixing the queries

Every output carries three queries, and **CI runs none of them.** The validator installs the app and then looks for it with its *own* loose search, so a manifest whose shipped queries never match still validates green. These queries are what Fleet runs on customer hosts, so run them yourself.

| Query | Fleet uses it for | Must return a row when | Must return no rows when |
|-------|-------------------|------------------------|--------------------------|
| `exists` | automatic-install policy; matching the app to inventory | the app is installed | it isn't |
| `patched` | the patch policy | the host is up to date (or the app is absent) | an older build is installed |
| `open` | the **Patch when closed** pre-install check | the app is closed (or absent) | the app is running |

How to read them:
- **`exists`** on macOS is `SELECT 1 FROM apps WHERE bundle_identifier = '<id>'`. On Windows it's `programs` matched on `name` (exact, or `LIKE` via `fuzzy_match_name`) and `publisher`, unless `exists_query` replaces it.
- **`patched`** is `SELECT 1 WHERE NOT EXISTS (<exists body> AND version_compare(<column>, '<version>') < 0)`. It only inspects rows the exists body matches. So an identity bug that blanks `exists` makes `patched` pass forever (Fleet thinks nothing is outdated), and a version column that doesn't track the manifest version makes it fail forever or pass forever.
- **`open`** on macOS joins `apps` to `processes` on the bundle's own executable path and needs no per-app data. On Windows it's `LOWER(name) = '<lowercased catalog name>.exe'`, unless `windowsOpenQueryOverrides` in [pkg/patch_policy/patch_policy.go](../../../../pkg/patch_policy/patch_policy.go) has an entry keyed by the catalog `name`. `NOT EXISTS` over a process name that never matches is always true: the app reads as permanently closed, the gate waves the install through over a running app, and nothing logs an error.

## Run them through real osquery

Any fleetd-enrolled Mac already has osquery, and it answers ad hoc queries without root. `scripts/check_queries.sh` runs a macOS output's three queries, plus `patched` with the version bumped past anything real:

```bash
.claude/skills/new-fma/scripts/check_queries.sh ee/maintained-apps/outputs/<app>/darwin.json
# No fleetd on this Mac? Extract osqueryd from the cask without installing anything:
#   brew fetch --cask osquery && pkgutil --expand-full "$(brew --cache --cask osquery)" osq
#   OSQ=osq/Payload/opt/osquery/lib/osquery.app/Contents/MacOS/osqueryd .claude/skills/new-fma/scripts/check_queries.sh …
```

On Windows, run `osqueryi.exe --json "<sql>"` on the host where you installed the app as SYSTEM, from a `.ps1` run with `-File` ([windows-scripts.md](windows-scripts.md#testing-as-system)).

Two parsing traps: an empty result renders as `[`, a blank line, `]` rather than `[]`, so count rows (`jq length`; in PowerShell, strip whitespace before comparing and filter nulls out of `ConvertFrom-Json`). And `version_compare(NULL, ...)` is an error, not false, so a `regex_match` override needs `COALESCE` around it.

On a dev Mac without the app, don't fabricate an `.app` in `/Applications` to test against: an enrolled Mac reports it to real inventory. Find an installed app with the same column shape instead (`SELECT bundle_identifier, bundle_short_version, bundle_version FROM apps WHERE bundle_short_version = '';`) and run the query shape against its bundle id. If you can't run a query at all, say so in your report; reading a query is not verifying it.

## `patched`: prove it can fail

Check four cases on a host with the app installed, not just the happy path:
1. `exists` → at least one row.
2. `patched` as generated, on an up-to-date install → one row. Zero rows on a host that really is behind is the policy working; upgrade it, or treat it as case 4.
3. `patched` with the version bumped past anything real (first segment +1000, e.g. `'1004.52.171'`) on the same host → **zero rows**. This is the case that catches false greens: a row here means the compared column isn't something the manifest version can order against.
4. If you can, the previous build. `git show origin/main:ee/maintained-apps/outputs/<app>/<platform>.json | jq -r '.versions[0].installer_url'` gives its URL. Install it and expect zero rows, then install the new build on top (no uninstall in between, which is what Fleet's remediation does) and expect one row.

Then look at the column itself:
```sql
-- macOS
SELECT bundle_short_version, bundle_version, path FROM apps WHERE bundle_identifier = '<id>';
-- Windows
SELECT name, publisher, version FROM programs WHERE name LIKE '%<name>%';
```

## When the version column doesn't track the manifest

`version_compare` has no notion of "unknown":
- `''` sorts below everything, so a missing `CFBundleShortVersionString` fails the policy forever (Steam).
- A marketing version sorts below the cask's build-suffixed version (`3.8.7` vs `3.8.7.19194`: Sonos, i1Profiler), with the same result.
- A longer registry version sorts *above* (`3.14.5150.0` vs `3.14.5`: python.org), which passes outdated hosts within the same minor version.
- A 4-part registry version against a 3-part manifest version with the same prefix (`28.1.7.13231` vs `28.1.7`) is not `< 0`, so it needs no fix.

The validator doesn't catch any of this: on macOS it accepts a match on either `CFBundleShortVersionString` or `CFBundleVersion`. Fix each case at the app, never in the shared generator:
- **macOS**: add a per-token branch in `ee/maintained-apps/ingesters/homebrew/ingester.go` that sets `out.Queries.Patched` to compare `bundle_version` (Sonos, i1Profiler, Steam), or `COALESCE(regex_match(...))` for descriptive version strings (R.app). Version-string mismatches in punctuation (`0.56-3` vs `0.56.3`) are fixed with a version transformer in `ingesters/homebrew/external_refs` instead (Pd, Sublime Text).
- **Windows**: `use_display_version_for_patch` in the input, or shape `exists_query`.
- JetBrains on Windows: the registry version is a build number (`261.24374.185`), but `MutateSoftwareOnIngestion` rewrites inventory to the marketing version parsed from the name (`PhpStorm 2026.1.2` → `2026.1.2`), which requires the publisher to contain "jetbrains". The Windows validator selects `publisher` and sets `Software.Vendor` so this fires in CI too (`cmd/maintained-apps/validate/windows.go`).

Then rerun the four cases against the override. `patch_policy_path` in the input structs is dead code: the patched query can only be changed by shaping `exists_query` or by an ingester branch (Docker Desktop precedent).

## `exists`: name matching in both directions

Inno, NSIS, and electron-builder installers often register `"<Name> <version>"` (`Signal 8.24.1`, `Bdash 1.35.1`, `Notion Calendar 1.133.0`); those need `fuzzy_match_name: true`. But plenty register a plain name (`Asana`, `Discord`, `Canva`, `Kiro (User)`) and must keep an exact match. Setting `fuzzy_match_name` on those breaks them just as badly, because `name LIKE 'Asana %'` never matches `Asana`. Observe the DisplayName, then decide.

**Sibling products that share a name:**
- Corretto 21 and 25 both register as `Amazon Corretto (x64)`. Pin each with `exists_query … AND version LIKE '<major>.%'`.
- IntelliJ Ultimate's `IntelliJ IDEA <ver>` also matches Community's `IntelliJ IDEA Community Edition <ver>`. Exclude the sibling in `exists_query` (`AND name NOT LIKE 'IntelliJ IDEA Community%'`) or use a custom `fuzzy_match_name` pattern.
- Vendors rename ARP entries between versions (Microsoft's VC++ redistributable went from `… 2015-2022 …` to `… v14 …`). If you see a rename, match both names.

## `open`: run it with the app running

Check three states, in order: app closed → one row; launch it → **zero rows**; quit it → one row again. A query that stays at zero after quitting is matching a helper, updater, or trial-nag process that outlives the window. One that stays at one while the app is up matches nothing.

- **macOS** needs no per-app data, but still gets the three-state run: the join requires the live process path to be exactly `<app path>/Contents/MacOS/<CFBundleExecutable>`. Chromium browsers relaunch from a `.code_sign_clone` after updating themselves and would read as closed, which is why they carry a per-token override in the homebrew ingester. Apps whose windows run from nested bundles (KiCad's editors under `Contents/Applications/`) need one too.
- **Windows**: get the real executable name, not the catalog name. Live: `Get-Process | Where-Object Path -like 'C:\Program Files\<Vendor>*' | Select-Object Name, Path` while the app is open. Offline: `msiinfo export app.msi Shortcut` (column 5, `[#_7zFM.exe]`), `unzip -p app.msix AppxManifest.xml | grep Executable=`, or `7zz l app.exe` for NSIS. Pick the GUI executable(s) the user has open, separately named editions included (`Code - Insiders.exe`), and leave out updaters (`GUP.exe`) and short-lived CLIs (`7z.exe`). A multi-word catalog name guesses wrong by construction (`'amazon chime.exe'`), as does any app whose exe isn't its name (`Android Studio` → `studio64.exe`). Add the override to `windowsOpenQueryOverrides`, regenerate, and confirm that the output's `open` changed. The README's "Editing an app's pre-install query" section covers replacing the whole query per slug in the winget ingester.
