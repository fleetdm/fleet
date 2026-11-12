# Validator, nightly PR, ingest, and soak failures

- [Where failures come from](#where-failures-come-from)
- [Reading the log](#reading-the-log)
- [Error index](#error-index)
- [Soak issues and customer reports](#soak-issues-and-customer-reports)
- [Revert and freeze](#revert-and-freeze)
- [Pin to what the installer really installs](#pin-to-what-the-installer-really-installs)
- [Rolling "latest" URLs](#rolling-latest-urls)
- [Same-version re-upload](#same-version-re-upload)
- [Cask renamed or removed](#cask-renamed-or-removed)
- [Winget manifest changed underneath the input](#winget-manifest-changed-underneath-the-input)
- [Download blocked](#download-blocked)
- [Transient failures and reruns](#transient-failures-and-reruns)
- [Nightly PRs](#nightly-prs)
- [Large PRs](#large-prs)
- [Editing an output without regenerating](#editing-an-output-without-regenerating)

## Where failures come from

- **Ingest** (`.github/workflows/ingest-maintained-apps.yml`, every 4 hours and on pushes that touch `ee/maintained-apps/`) runs the generator over every input and opens or updates the nightly "Update Fleet-maintained apps" PR. It fails by **panicking**, and the run aborts at the first broken app, so one bad input blocks every other app's update (macOS included).
- **Validation** runs on PRs. `.github/scripts/detect-new-fmas-in-pr.sh` picks apps whose `outputs/*/<platform>.json` changed (plus new `apps.json` slugs) and shards them across runners. Each app is downloaded, hash-checked, installed, found with osquery, checked with its exists query (and on macOS, Gatekeeper and sometimes its patched query), uninstalled, and checked again. Frozen apps are skipped ("App is frozen, skipping validation").
- **Soak issues** (`~fma-soak`) come from installing the catalog on real hosts against a release candidate. They report what the RC fetched on the day of the run.

Diagnose before you change anything. Most failures look alike in the summary and differ only in the log.

## Reading the log

```bash
gh run view <run-id> --repo fleetdm/fleet --log-failed | grep -v harden-runner | grep -E 'level=ERROR|Found app:|Looking for app|Gatekeeper|app=' | head -60
```

- `Found app: '<name>' at <path>, Version: <v>` lines are what osquery reported (after `MutateSoftwareOnIngestion` rewrote the name or version). No `Found app:` line means the loose search returned nothing.
- The validator stops at the first accepting row, which can be a helper or an uninstaller with another bundle id. Map rows to paths with the `Gatekeeper assessment: <path>` lines before trusting one.
- For an ingest panic, the error names the slug; in older logs, the last `ingesting … app` line before the panic is the failing app.
- Script output is logged only at debug level, unless the script fails. To see inside a script that passes but misbehaves, push a temporary instrumented script that prints what you need, then revert it.
- Per-job logs (`gh api repos/fleetdm/fleet/actions/jobs/<id>/logs`) outlive per-run log zips, which return 410 after about 90 days.
- The runner is ephemeral. Reproduce locally when the log isn't enough: your Mac for darwin (without installing, see [queries.md](queries.md#running-them)), a Windows VM as SYSTEM for Windows ([windows-scripts.md](windows-scripts.md#testing-as-system)).

## Error index

| Symptom | Likely causes | Go to |
|---|---|---|
| darwin: `App version 'X' was not found by osquery`, `Found app:` shows another version | The installer doesn't install what the cask says: a version-less URL drifted (Acrobat lags; ocenaudio, Keeper, MEGAsync, Mobirise lead), a mislabeled release (MeetingBar tag v5.0.0 builds 5.0.1), a bogus cask version (DockFix), a pkg whose plist wasn't bumped (BrickLink), a stub installer (RoboForm 10) | [Pin to what the installer really installs](#pin-to-what-the-installer-really-installs) |
| darwin: same error, **no** `Found app:` line | Wrong bundle id; the app isn't registered (suite folder, nested app); the app's own updater replaced it mid-check (OneDrive's StandaloneUpdater when the cask lags Microsoft's channel) | [identity-verification.md](identity-verification.md#macos-installers), [macos-scripts.md](macos-scripts.md#suite-casks-and-nested-apps) |
| darwin: `osquery reports bundle_short_version 'X', a shorter form of 'Y'`, or `the patched query returns no rows with version 'Y' installed` | The compared column doesn't track the cask version, so the patch policy would fail on current hosts | A transformer or a `bundle_version` branch ([queries.md](queries.md#when-the-version-column-doesnt-track-the-manifest)) |
| `exists query returned no rows with the app installed` | The shipped exists query never matched: wrong `unique_identifier`, publisher, versioned DisplayName, or exact vs. fuzzy match. Often a latent bug that surfaces on the app's next bump (Chatbox, GIMP, KNIME) | [queries.md](queries.md#exists-name-and-publisher-matching) |
| `Gatekeeper rejected: … source=Unnotarized Developer ID` | Usually Apple hasn't published the new build's notarization ticket yet | Check locally (`spctl -a -t execute -vv` on the mounted app) and rerun; [Transient failures](#transient-failures-and-reruns) |
| `Gatekeeper rejected: … sealed resource is missing or invalid` / `no usable signature` | The bundle changed after signing: a custom script used `unzip` instead of `ditto -x -k`, or the vendor's postinstall edits its own bundle (Topaz, Omnissa Horizon, WiFiman); or the app is signed with a development certificate | Fix the extraction ([macos-scripts.md](macos-scripts.md#unpacking-dmg-zip-and-pkg)); otherwise freeze or remove the app |
| `Gatekeeper assessment found no app` | The exists clause matches no `apps` row: wrong bundle id | [identity-verification.md](identity-verification.md#macos-installers) |
| `hdiutil: attach failed - image not recognized` | `installer_format` is `dmg` but the download isn't one (the cask moved to a zip, a pkg, or a bare binary: Kiro CLI) | Fix `installer_format`; a binary-only cask can't be an FMA: revert and freeze, or remove |
| windows: `App version 'X' was not found by osquery` | The script exited before the ARP entry was written (registry race); DisplayName or version shape mismatch; a rolling URL served a newer build; a per-user install crashed but the script exited 0 (exit code above `Int32.MaxValue`) | [windows-scripts.md](windows-scripts.md#install-scripts), [Rolling URLs](#rolling-latest-urls) |
| `App still present after uninstall` | The uninstall picked the inner MSI's key instead of the Burn bundle's; an NSIS uninstaller relaunched itself and the script didn't wait; a Burn bundle refused because a restart is pending (exit 3010 with its key still present); Squirrel leftovers | [windows-scripts.md](windows-scripts.md#uninstall-scripts) |
| `SHA256 hash in manifest does not match installer file hash` | Rolling "latest" URL; the vendor re-uploaded the same version; the CDN served the runner an HTML decoy | [Rolling URLs](#rolling-latest-urls), [Same-version re-upload](#same-version-re-upload), [Download blocked](#download-blocked) |
| `response status code 403` / `465` / other 4xx on download | A WAF blocks Go's default User-Agent, or the runner's IP range | [Download blocked](#download-blocked) |
| Ingest panic: `failed to find installer for app` | The newest winget manifest has no installer matching the input's arch, scope, type, or locale | [Winget manifest changed](#winget-manifest-changed-underneath-the-input) |
| Ingest panic: `no version manifest directories found under package path` | winget automation deleted every version (hash mismatch on a rolling URL), or the package moved to a new identifier | [Winget manifest changed](#winget-manifest-changed-underneath-the-input) |
| Ingest panic on a cask 404 | Cask renamed or removed upstream (`frozen` doesn't help) | [Cask renamed or removed](#cask-renamed-or-removed) |
| Ingest: `no install script found for app` | An `msix` input without custom scripts, or an input whose type changed | [windows-scripts.md](windows-scripts.md#msix) |
| Ingest: `no DisplayVersion found in winget manifest` | `use_display_version_for_patch` on a manifest without `AppsAndFeaturesEntries` | Drop it; use an enricher ([queries.md](queries.md#when-the-version-column-doesnt-track-the-manifest)) |
| Ingest: `invalid category` | `default_categories` outside the allowed list | [SKILL.md](../SKILL.md#how-the-generator-behaves) |
| `exit status 3010` or `1641` | Reboot required/initiated: success, unless a Burn bundle's key is still there afterward | `$expectedExitCodes = @(0, 1641, 3010)` ([windows-scripts.md](windows-scripts.md#install-scripts)) |
| Timeout after ~11 min, empty output, installer still locked | Silent switch ignored (wrong framework); Inno `[Run]` postinstall without `skipifsilent`; the installer launches the app; MSI maintenance form `/X /I{GUID}`; a hidden MessageBox with no `/SD` default (see next row) | [windows-scripts.md](windows-scripts.md#installers-that-never-exit) |
| Install never finishes on a real host (soak), passes in CI | A dialog under `/S` or `/VERYSILENT` in session 0: JetBrains same-version reinstall ("installation dir is not empty"), KNIME upgrade, Rtools existing folder, Android Studio running, x64 installer or AVX check on ARM64 (Pale Moon), Squirrel's error dialog | [windows-scripts.md](windows-scripts.md#installers-that-never-exit) |
| Exit `-1073741819` / `3221225477` (`0xC0000005`), "no changes detected" | Transient NSIS self-extractor crash, or a per-user installer run as SYSTEM (Notion) | [Transient failures](#transient-failures-and-reruns), [windows-scripts.md](windows-scripts.md#when-the-app-has-no-machine-wide-mode) |
| NSIS exit `2` | A running-app check in `.onInit` aborted (RStudio), or `InstallLib` hit a WOW64 path as SYSTEM (MEGAsync) | [windows-scripts.md](windows-scripts.md#installers-that-never-exit) |
| Inno exit `5`, `1`, or `8` with the app open | Restart Manager can't close a session-1 app from SYSTEM | Stop the app first ([windows-scripts.md](windows-scripts.md#install-scripts)) |
| Burn exit `1` in seconds; log says `Index was outside the bounds` | A custom bootstrapper application rejects `/norestart` (Okta Verify 7) | [windows-scripts.md](windows-scripts.md#install-scripts) |
| Exit `1603` right after `Action start: LaunchConditions` | The MSI refuses `REBOOT=ReallySuppress`; or an x64 MSI refuses ARM64 ("You must use the native version": WireGuard) | [windows-scripts.md](windows-scripts.md#install-scripts) |
| Exit `1602` on a same-version reinstall | A custom action launched the app and aborted (WireGuard: `DO_NOT_LAUNCH=1`) | [windows-scripts.md](windows-scripts.md#install-scripts) |
| Exit `1618` | Another msiexec is still running: an earlier app's hung uninstall in the same shard, or another session on a shared VM | Fix the app that hung, not this one |
| Exit `1619` | A 32-bit InstallShield setup cached its MSI under WOW64-redirected `systemprofile` | `/b"<temp dir>"` ([windows-scripts.md](windows-scripts.md#the-wow64-trap)) |
| Exit `1610` (Google Drive) | A restart is pending after the last uninstall | Report "restart required"; Fleet can't reboot |
| `0x800702E4` (`ERROR_ELEVATION_REQUIRED`) | Per-user installer that also needs admin | `-RunLevel Highest` ([windows-scripts.md](windows-scripts.md#when-the-app-has-no-machine-wide-mode)) |
| Exit `0x80042000` | InstallShield wrapper silenced, inner MSI not | `/S /v/qn` ([identity-verification.md](identity-verification.md#installshield)) |
| Inno `/LOG`: "can only be installed on … x64" or "does not support the version of Windows" | x64-only installer on an ARM64 host | Not a defect; see [windows-scripts.md](windows-scripts.md#testing-as-system) |

## Soak issues and customer reports

Before you change anything:
1. **Check whether it's already fixed.** Run `git log --since=<issue date> -- ee/maintained-apps/inputs/<source>/<token>.json ee/maintained-apps/outputs/<token>/` and fetch the live manifest (`curl -s https://maintained-apps.fleetdm.com/manifests/<token>/<platform>.json | jq .versions[0].queries`). A fix on main is already live for every server version. If it's fixed, report the fixing PR with evidence and offer to close the issue; don't open a duplicate PR.
2. **"Stuck on an old version"**: check the input for `frozen` first.
3. **"Installed but never detected" or "patch policy always passes"**: check whether the slug's `unique_identifier` changed on main recently. The RC may still carry the old id, or its server may lack the patch-policy sync ([queries.md](queries.md#how-fixes-reach-existing-deployments)).
4. **"Install never finishes"** on Windows: look for a hidden dialog first ([windows-scripts.md](windows-scripts.md#installers-that-never-exit)), and check whether the host is ARM64. Don't reach for a longer timeout or a validator cap until you've ruled one out.
5. Read the issue's claims as hypotheses. "Fails when the app is open" turned out to be "every same-version reinstall" (JetBrains); "publisher mismatch" turned out to be a versioned DisplayName (Vernier).

Evidence without a VM: since the exists check landed, a nightly PR's validate log that contains `All checks passed for app: <Name>`, from a run whose `headSha` includes the fix, shows the shipped exists query matched on a real install, unless that app also logged `Exists query returned no rows` (a warning for provisioned-only AppX packages and Codex CLI, not a failure).

## Revert and freeze

The standard fix for a nightly update PR when one app's new version fails and the cause is upstream (the installer and the cask or manifest disagree), not something you can fix in the input:

1. Read the failing app's `Found app:` lines for the version osquery actually saw.
2. Revert the output to main: `git checkout origin/main -- ee/maintained-apps/outputs/<token>/<platform>.json`.
3. Add `"frozen": true` as the last key of the input.

That clears CI twice over: the output now matches main, so the detector doesn't select it, and frozen apps are skipped anyway. `isFrozen()` reads `inputs/<source>/<slug-token>.json`, so the input filename must match the output directory. The generator then stops overwriting the pinned output.

Before trusting the revert, check that the pinned version is what the installer installs. If the installed version is higher than the pin, the version-less installer has moved past main: still safe for the patch policy (a newer build reads as compliant), but Fleet's displayed version lags. Prefer pinning to the real version ([below](#pin-to-what-the-installer-really-installs)).

**Freezes get forgotten**, and nightly skips are silent. Windows Firefox sat frozen at 154.0 for weeks after the real fix landed, until a customer reported it. So:
- Say in the PR exactly what has to happen before the app can be unfrozen (cask reaches X, winget PR Y merges, vendor ships a versioned URL).
- Before unfreezing, compare the live installer's real version and hash with the cask or manifest. Unfreezing a version-less app is a race; expect to refreeze.
- A frozen output with a real SHA on a URL the vendor overwrites breaks for customers the moment the file changes: `AddFleetMaintainedApp` rejects an installer whose hash doesn't match.
- A frozen app is never validated, so a freeze can hide an identity bug nobody has seen: frozen Kiro CLI ships `dev.kiro.cli`, but its app is `com.amazon.codewhisperer`. Whenever you touch a frozen app, inspect the pinned installer's identity too (`inspect_mac_installer.sh <installer_url> --sha256 <sha>`).
- To audit frozen apps cheaply, `curl -sIL` each pinned `installer_url` and compare `Last-Modified` with the freeze date, and check the cask's or manifest's commit history for whether upstream caught up. The freeze PR's commit messages usually say why it was frozen.
- To get one CI run on a frozen app (for example, an x64-only installer you can't run on an ARM VM), push a temporary commit that drops `frozen`, read the result, and revert that commit by itself.
- Unfreezing an output that was frozen long ago regenerates more than the version (new open query, new template lines). Review the whole diff.

## Pin to what the installer really installs

When the download installs a different version than the cask or manifest claims:
1. Read the real version from the installer (`inspect_mac_installer.sh`, `pe_info.py`, `msiinfo`), not from release notes: release notes and livecheck pages run ahead of the downloadable build (Evernote, Keeper, Acrobat). Pinning ahead of the installer makes the patch policy fail on freshly installed hosts.
2. Set both the output's `version` and the patched query's `version_compare` literal to that version. Keep the URL and SHA.
3. Freeze the input. For a version-less URL, every future bump is now manual.

If the installer is a stub, a launcher, or a different product than the cask describes (RoboForm 10's DMG app is an installer carrying the old bundle id; the real app is inside it), it needs a custom install script and usually a migration ([queries.md](queries.md#rebrands-renamed-ids-and-side-by-side-installs)), or it isn't shippable.

## Rolling "latest" URLs

Some manifests point at a "latest" redirect (`link.gotomeeting.com/latest-msi`, `download.scdn.co/SpotifyFullSetupX64.exe`). The pinned SHA drifts as soon as the vendor ships, and validation fails on the hash. winget automation also deletes such manifests when the hash stops matching ([below](#winget-manifest-changed-underneath-the-input)).

`ignore_hash: true` (winget) sets the output SHA to `no_check`, and the validator skips the hash comparison (`google-chrome`, `teamviewer`, and about 15 others). **It doesn't fix the version check**: the validator then compares the manifest version with what osquery reports, with only hardcoded per-app exemptions (Chrome and OneDrive auto-update, Office Click-to-Run, a few macOS apps). A rolling URL whose build runs ahead still fails, one step later. Find out which case you're in by reading the served installer's real version (`scripts/pe_info.py` on its first few MB). The options are pinning and freezing, an exemption (which weakens that app to an existence check and makes its patch policy meaningless; get maintainer agreement), or not shipping the app until the vendor offers a versioned URL. `no_check` also means Fleet can't detect a tampered download, so prefer a versioned URL whenever one exists.

## Same-version re-upload

If the cask or manifest still lists the old hash for the same version, the vendor silently re-uploaded a new build at the same URL, and no upstream fix is coming. Download it, `shasum -a 256` it, check the payload's real version, write the hash into the output, and freeze the input so the generator stops restoring the stale hash (i1Profiler, LibreOffice). Note it in the PR: it's a hash you chose to trust, not one upstream published.

## Cask renamed or removed

`frozen` doesn't skip the upstream fetch, so a cask 404 panics ingest even for a frozen app. Look the dead token up in the `old_tokens` arrays of `https://formulae.brew.sh/api/cask.json`, and check every input token at once, because ingest aborts at the first 404 and one fix can just move the panic to the next app:

- **Renamed, same app**: change `token` only. Keep `slug`, the input filename, `name`, and `frozen` (`worksheet-crafter` → `worksheetcrafter`).
- **Renamed and rebranded with a new bundle id**: a token swap would ship a dead exists query. If no customers depend on the old slug, create a new FMA under the new name and remove the old one (Windsurf → Devin Desktop, Swifty → Rowel: `git mv` the input, output, and website icon; drop the old `apps.json` entry). If customers do, keep the slug and migrate the queries and scripts ([queries.md](queries.md#rebrands-renamed-ids-and-side-by-side-installs)). Removing a slug from `apps.json` makes servers clear that FMA.
- **Removed** (the API 404s): remove the FMA. Freezing doesn't help, since ingest still fetches the cask.
- **Went binary-only or stale** (the cask still resolves): remove the FMA, or, if a stopgap is needed while that's decided, revert and freeze.
- In either case, don't recreate the cask in `inputs/homebrew/custom-tap/`: that directory is for Fleet's internal use, not a fallback for casks Homebrew dropped.

**Removing an FMA** touches: the input, the output (and its directory if no platform is left), the `apps.json` entry (nothing prunes it), custom scripts only that app uses, and, if no other platform of the app remains, `website/assets/images/app-icon-<token>-60x60@2x.png` (the catalog's icon, by slug). Keep the frontend icon in `icons/index.ts`: it matches software by name across all inventory, so hosts that still have the app keep showing it. Leave `tools/software/fma-script-hashes/` alone. In the PR body, say what customers will see: servers drop the slug on their next catalog sync (`ClearRemovedFleetMaintainedApps`); installers fleets already added stay, as custom packages; and GitOps files that still reference the slug fail until it's removed from them.

## Winget manifest changed underneath the input

`failed to find installer for app` means the newest manifest has no installer matching the input's `installer_arch`, `installer_scope`, `installer_type`, or `installer_locale`. Run `scripts/winget_manifest.sh <PackageIdentifier> --diff` to diff it against the previous version and see recent commits. What changes:
- A contributor flips `Scope` between versions, sometimes as a correction (Portfolio Performance's installer always was per-user).
- A schema bump adds a top-level `Scope` for a whole publisher (`Microsoft.DotNet.*`), which masks per-installer scopes.
- A manifest is edited in place without a version bump (GeoGebra split one MSI entry into machine and user entries). Check the commits on the version you already ingested, not just for a newer version.
- A vendor drops an architecture (Genesys Cloud dropped x86; Canva dropped x86) or changes installer technology (Audacity Inno → MSI; RealVNC zip → bare MSI; eM Client and PowerShell 7.7 MSIX-only). A technology swap changes identity, publisher, `ALLUSERS`, exe names, and side-by-side behavior at once: re-verify all of them.
- A manifest mislabels the type (Logi Options+ `portable` with an old binary; VS Code 1.142.0 pointing at 1.141.2's installer). Hash the live file and read its version before trusting a bump.
- The ingester normalizes `wix` to `msi`, accepts a missing scope as the input's, ignores a root-level `InstallerLocale`, and only treats folders that start with a digit and contain a dot as versions.

Align the input with what's true for the installer and regenerate. For exe apps with custom scripts, `installer_scope` only selects the manifest entry, and `programs` reads per-user keys too, so the queries keep working across a machine→user flip. When upstream drops your installer type entirely and there's no matching entry to move to, a temporary input-only fix is to freeze the app and point `installer_type`/`installer_scope` at an entry the new manifest does have; the frozen output is never rewritten. Revert those fields before unfreezing, or the next regeneration ships the wrong installer with your scripts.

`no version manifest directories found under package path` means every version folder is gone. `winget_manifest.sh` shows why: "Automatic deletion of …" (winget removed manifests whose rolling URL stopped matching its hash; a replacement "New version" PR may be pending upstream) or "Moved: <NewId> …" (repoint `package_identifier`). This panics the whole run, so act quickly: repoint, or if the package is gone for good, remove the FMA.

**Fix the whole family, not just the app in the log**: the run stops at the first failure, so siblings later in filename order look healthy (all three `.NET` inputs broke at once). **Verify by sweeping every input**, not by rerunning the full ingest, which dies on the first transient error:

```bash
go build -o /tmp/ma ./cmd/maintained-apps
for f in ee/maintained-apps/inputs/winget/*.json; do
  slug=$(jq -r .slug "$f")
  NETWORK_TEST_GITHUB_TOKEN=$(gh auth token) /tmp/ma -slug "$slug" >/dev/null 2>&1 || echo "FAIL $slug"
done
git checkout -- ee/maintained-apps/outputs   # the sweep rewrites outputs; keep only what you meant to change
```

Only rate-limit and 5xx GitHub errors are retried; a dropped connection is fatal to the whole run.

## Download blocked

Fleet's downloader sends Go's default `Go-http-client` User-Agent, and some WAFs reject it, for real Fleet users as well as in CI. Before declaring a URL dead:

```bash
curl -sSIL -A "Go-http-client/2.0" "<url>"   # reproduces the failure
curl -sSIL "<url>"                           # 200 => blocked by UA, not a bad URL
```

Variants: a block by client IP that returns **200 with an HTML page**, which surfaces as a SHA mismatch (if your local download's hash equals the manifest's, the runner is being served a decoy); a 403 only from GitHub's runner IPs (Twobird); a CDN that requires a `Referer` (jamovi) or a vendor User-Agent (FileZilla). Changing the downloader is a server change outside FMA scope, so the usual call is to drop the app, or revert its output to main and freeze it, and say in the PR that customers are affected too.

## Transient failures and reruns

Rerun only after you've shown the failure is transient: `gh run rerun <run-id> --failed`.
- **NSIS `0xC0000005` before extracting anything**: find a run where the same installer passed (`git log -S '<version>' -- ee/maintained-apps/outputs/<token>/windows.json`, then that run's log). For a brand-new version, compare `7zz l` of the old and new installers: an identical NSIS stub, header, and plugin set means nothing about the installer machinery changed. A deterministic failure never passes; don't paper over it with retries.
- **Gatekeeper "Unnotarized Developer ID"** on a fresh release: notarization tickets can lag the download by minutes. Check `spctl -a -t execute -vv` on the mounted app locally; if it now says "Notarized Developer ID" with the same Team ID, rerun. If it stays unnotarized, revert and freeze or remove.
- **OneDrive-style updater swaps** on macOS: the app's own updater replaces the bundle right after install. The Gatekeeper check retries this; the version check has per-app exemptions.
- `1618` collateral: rerun after the app that hung is fixed.

## Nightly PRs

- Push CI fixes (bundle ids, pins, freezes, removals) to the bot's branch (`fma-<timestamp>`), not a new PR. Fetch it first: others push to it too.
- Nightly PRs opened by `fleet-release` don't need splitting at any size. "Check Script Diff" posts the whole diff as one comment and fails above about 65K characters; it isn't required.
- When a shared-template change lands, the next nightly regenerates every non-frozen generated output, so expect hundreds of files and a full pass of the exists and Gatekeeper checks. Latent bugs in apps nobody touched surface there; fix or freeze them on the bot branch, and decide removals with a maintainer.
- When a nightly regenerates outputs that an open PR also changes, merge main into that PR (don't force-push): take main's output, reapply your script change, recompute the ref, and keep `refs` sorted.

## Large PRs

Human-authored PRs over 100 changed files can't be auto-approved. Split them into PRs with **disjoint file sets** (the only guarantee against conflicts between the parts):
- Keep each input with its output, so `frozen: true` lands with the output the validator would otherwise check. `apps.json` and the icons `index.ts` travel with whichever part adds or removes apps.
- Size parts by the script-diff comment, not file count, if you want "Check Script Diff" to pass.
- Merge main into the source branch first (check it out, then `git merge origin/main`), so a split doesn't revert something main changed after the nightly ran. `git merge-tree --write-tree` only previews conflicts; it doesn't update the branch.
- Tests that read every output (`TestOutputInstallScriptsStopWhenExtractionFails`) fail on whichever part carries the generator and test until the other parts are on main. Put generator, test, custom scripts, and version bumps in the last part, mark it "merge last", and prove the order by testing that part alone (fails) and all parts together (passes).
- The repository rejects a push of many refs at once (GH013); push one branch per `git push`.

## Editing an output without regenerating

Use this to change a frozen app, or to swap a script without pulling an upstream version bump into the PR:

- A ref is `sha256(script text)` as hex, first 8 characters. Remove the old ref from `refs`, add the new one, and update `install_script_ref`/`uninstall_script_ref`.
- Custom-script outputs embed `inputs/*/scripts/<file>` verbatim. Check first that the output's current ref text matches the old input script.
- Re-encode the way the generator does: Go `json.Encoder` with `SetEscapeHTML(false)`, two-space indent, trailing newline, field order as in `FMAManifestFile`. Python's `json` escapes `<`, `>`, `&`, and non-ASCII (`—` becomes `—`) and churns the file unless you pass `ensure_ascii=False` and handle the rest, so prove that an untouched round trip is byte-identical before writing anything. A throwaway `go run` helper in the worktree that imports `maintained_apps.FMAManifestFile` and `GetScriptRef` is the safest route; delete it before committing.
- For a patched-query or version pin, edit the `version` field and the `version_compare` literal together.
- To check a hand edit, run the generator on one non-frozen app with the same change and compare. Then restore each file it touched that you didn't mean to change, by path (`git checkout -- <file>`), because it can also rewrite `apps.json` and other slugs (`--slug` is a substring match); undo the throwaway input change the same way. Never reset a whole directory (`ee/maintained-apps`, `inputs/`, `outputs/`): that also reverts your uncommitted input JSON, custom scripts, and hand-edited outputs.
