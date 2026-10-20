# Validator, nightly PR, and ingest failures

Where failures come from:
- **Ingest** (`.github/workflows/ingest-maintained-apps.yml`, every 4 hours and on pushes that touch `ee/maintained-apps/`) runs the generator over every input and opens or updates the nightly "Update Fleet-maintained apps" PR. It fails by **panicking**, and the run aborts at the first broken app.
- **Validation** runs on PRs. `.github/scripts/detect-new-fmas-in-pr.sh` picks apps whose `outputs/*/<platform>.json` changed (plus new `apps.json` slugs) and shards them across macOS and Windows runners. Each app is installed, looked up with osquery, uninstalled, and checked again. Frozen apps are skipped ("App is frozen, skipping validation").

Diagnose before you change anything. Most of these failures look alike in the summary and differ only in the log.

## Reading the log

```bash
gh run view <run-id> --repo fleetdm/fleet --log-failed | grep -v harden-runner | grep -E 'level=ERROR|Found app:|app=' | head -50
```

- `Found app: '<name>' Version: <v>` lines are what osquery actually reported. No `Found app:` line at all means osquery returned zero rows.
- For an ingest panic, the last `ingesting … app` line before the panic names the failing app: that line is logged *before* ingestion.
- The runner is ephemeral. You can't query it afterward, so reproduce locally (a Mac for darwin, a Windows VM as SYSTEM for Windows) when the log isn't enough.

## Error index

| Symptom | Likely causes | Go to |
|---|---|---|
| darwin: `App version 'X' was not found by osquery`, with `Found app:` showing another version | The version-less installer drifted from the cask (Acrobat lags it, ocenaudio leads it) | [Revert and freeze](#revert-and-freeze) |
| darwin: same error, **no** `Found app:` line | Bundle id wrong; app installed outside `/Applications`/not registered (suite folder); app replaced mid-check by its own updater (OneDrive's StandaloneUpdater when the cask lags Microsoft's channel) | [identity-verification.md](identity-verification.md#macos-installers); reproduce locally |
| windows: `App version 'X' was not found by osquery` | Install script exited before the ARP entry was written (registry race); DisplayName or version shape mismatch; rolling URL served a newer build | [windows-scripts.md](windows-scripts.md#install-scripts), [Rolling URLs](#rolling-latest-urls) |
| `SHA256 hash in manifest does not match installer file hash` | Rolling "latest" URL; vendor re-uploaded the same version; CDN serving the runner an HTML decoy | [Rolling URLs](#rolling-latest-urls), [Same-version re-upload](#same-version-re-upload), [Download blocked](#download-blocked) |
| `response status code 403` / `465` / other 4xx on download | WAF blocking Go's default User-Agent | [Download blocked](#download-blocked) |
| Ingest panic: `failed to find installer for app` | The winget manifest no longer matches the input's arch/scope/type/locale | [Winget manifest changed](#winget-manifest-changed-underneath-the-input) |
| Ingest panic on a cask 404 | Cask renamed or removed upstream (`frozen` doesn't help) | [Cask renamed or removed](#cask-renamed-or-removed) |
| `exit status 3010` or `1641` | Reboot required/initiated: success | `$expectedExitCodes = @(0, 1641, 3010)` in the input script, regenerate ([windows-scripts.md](windows-scripts.md#install-scripts)) |
| Timeout after ~11 min, empty script output, installer still locked | Silent switch ignored (wrong framework: confirm with `strings`); Inno `[Run]` postinstall without `skipifsilent`; installer launches the app; MSI maintenance-form `/X /I{GUID}` | [identity-verification.md](identity-verification.md#exe-installers), [windows-scripts.md](windows-scripts.md#install-scripts) |
| Exit `-1073741819` (`0xC0000005`), "no changes detected" | Transient NSIS self-extractor crash, or a per-user installer run as SYSTEM (Notion) | [windows-scripts.md](windows-scripts.md#install-scripts) |
| Exit `1603` right after `Action start: LaunchConditions` | MSI refuses `REBOOT=ReallySuppress` | [windows-scripts.md](windows-scripts.md#install-scripts) |
| Exit `1618` | Another msiexec is still running, usually an earlier app's hung uninstall in the same shard | Fix the app that hung, not this one |
| `0x800702E4` (`ERROR_ELEVATION_REQUIRED`) | Per-user installer that also needs admin | `-RunLevel Highest` ([windows-scripts.md](windows-scripts.md#when-the-app-has-no-machine-wide-mode)) |
| Exit `0x80042000` | InstallShield wrapper silenced, inner MSI not | `/S /v/qn` ([identity-verification.md](identity-verification.md#installshield)) |
| Inno `/LOG`: "can only be installed on … x64" | x64-only installer on an ARM64 host | Not a defect ([windows-scripts.md](windows-scripts.md#testing-as-system)) |

## Revert and freeze

The standard fix for a nightly update PR when one app's new version fails validation and the cause is upstream (the installer and the cask/manifest disagree), not something you can fix in the input:

1. Read the failing app's `Found app:` lines for the version osquery actually saw.
2. Revert the output to main: `git checkout origin/main -- ee/maintained-apps/outputs/<app>/<platform>.json`.
3. Add `"frozen": true` as the last key of the input.

That clears CI twice over: the output now matches main, so the detector doesn't select the app, and frozen apps are skipped anyway. `isFrozen()` reads `inputs/<source>/<slug-token>.json`, so the input filename must match the output directory. The generator then stops overwriting the pinned output.

Before trusting the revert, check that the pinned version is right. If the installed version is *higher* than the pin, the vendor's version-less installer has moved past main. That's still safe (`patched` uses `< 0`, so a newer installed build reads as compliant), but the version Fleet displays will lag. If a payload version disagrees with the cask, read it from the installer itself ([identity-verification.md](identity-verification.md#macos-installers)) and pin that. Leave a note in the PR so someone unfreezes it once upstream agrees again.

## Rolling "latest" URLs

Some manifests point at a "latest" redirect (`link.gotomeeting.com/latest-msi`, `download.scdn.co/SpotifyFullSetupX64.exe`). The pinned SHA drifts as soon as the vendor ships, and validation fails on the hash. Spotify served the pinned build one day and a newer one the next, so chasing the hash isn't viable.

`ignore_hash: true` (winget) sets the output SHA to `no_check`, and the validator skips the hash comparison (the route already taken for `google-chrome`, `teamviewer`, and about 15 others). **It doesn't fix the version assertion.** The validator then compares the manifest version with what osquery reports, and it has no general drift tolerance, only hardcoded per-app exemptions in `appExists` (Chrome for auto-update, Office for Click-to-Run). So a rolling URL whose build runs ahead of winget still fails, one step later. The options are a matching exemption, which weakens that app to an existence-only version check (shared tooling, so get maintainer agreement), or not shipping the app until the vendor offers a versioned URL. Find out which case you're in by reading the served installer's real version first: `scripts/pe_info.py` on its first few MB ([identity-verification.md](identity-verification.md#partial-download-probes)).

Don't add existence-only version skips to the validator lightly. They make the patch policy always report "patched", so it never flags an outdated install. Only add one when the version genuinely can't be reconciled; if the `Found app:` version matches the FMA version, no skip is needed.

## Same-version re-upload

If the cask or manifest still lists the old hash for the same version, the vendor silently re-uploaded a new build at the same URL, and no upstream fix is coming. Download it, `shasum -a 256` it, check the payload's real version, write the hash into the output, and freeze the input so the generator stops restoring the stale hash (i1Profiler, LibreOffice).

## Cask renamed or removed

`frozen` doesn't skip the upstream fetch, so a cask 404 panics ingest even for a frozen app. Look the dead token up in the `old_tokens` arrays of `https://formulae.brew.sh/api/cask.json` (check every input token at once, because ingest aborts at the first 404 and one fix can just move the panic to the next app):

- **Renamed, same app**: change `token` only. Keep `slug`, the input filename, `name`, and `frozen` (`worksheet-crafter` → `worksheetcrafter`).
- **Renamed and rebranded with a new bundle id**: a token swap would ship a dead exists query. Migrate the FMA instead: a new slug if no customers depend on the old one (Windsurf → Devin, Swifty → Rowel), or a migration-aware query and script if they do.
- **Removed**: delete the input (the orphaned output and `apps.json` entry persist, because nothing prunes them), or point `cask_path` at a committed cask JSON (`inputs/homebrew/custom-tap/`).

## Winget manifest changed underneath the input

`failed to find installer for app` means the newest manifest has no installer matching the input's `installer_arch`, `installer_scope`, `installer_type`, or locale. Diff the latest `<Pkg>.installer.yaml` against the previous version's. Contributors flip `Scope` between versions (sometimes as a correction), vendors drop architectures (Genesys Cloud dropped x86), schema bumps add a top-level `Scope` for a whole publisher (`Microsoft.DotNet.*`), and manifests are sometimes edited in place without a version bump. The ingester normalizes `wix` to `msi`. Check which value is actually true for the installer, align the input, and regenerate. For exe apps with custom scripts, `installer_scope` only selects the manifest entry, and `programs` reads per-user (HKU) keys too, so the queries keep working across a machine→user flip.

## Download blocked

The installer downloader sends Go's default `Go-http-client` User-Agent, and some WAFs reject it, for real Fleet users as well as in CI. Before declaring a URL dead:

```bash
curl -sSIL -A "Go-http-client/2.0" "<url>"   # reproduces the failure
curl -sSIL "<url>"                           # 200 => blocked by UA, not a bad URL
```

A variant blocks by client IP and returns **200 with an HTML page**, which surfaces as a SHA mismatch. If a local download's hash equals the manifest's, the manifest never drifted; the runner is being served a decoy. Fixing the downloader is a server change outside FMA scope, so the usual call is to drop the app, or revert its output to main and freeze it.

## Editing an output without regenerating

Use this to change a frozen app, or to swap a script without pulling an upstream version bump into the PR:

- A ref is `sha256(script text)` as hex, first 8 characters. Remove the old ref from `refs`, add the new one, and update `install_script_ref`/`uninstall_script_ref`.
- Custom-script outputs embed `inputs/*/scripts/<file>` verbatim. Check first that the output's current ref text matches the old input script.
- Re-encode the way the generator does (Go `json.Encoder` with `SetEscapeHTML(false)`, two-space indent, trailing newline). Python's `json` escapes non-ASCII characters (`—` becomes `\u2014`) and churns the file unless you pass `ensure_ascii=False`, so prove that an untouched round trip is byte-identical before writing anything.
- To check a hand edit, run the generator on one non-frozen app with the same change and compare. Then `git checkout` every file it touched, because it can also rewrite `apps.json`.
