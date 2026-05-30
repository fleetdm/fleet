---
name: new-fma
description: Add a Fleet-maintained app (FMA) for macOS (Homebrew) and/or Windows (winget). Use when asked to "add X as a macOS/Windows FMA", "add a Fleet-maintained app", or to debug FMA validator failures. Emphasizes verifying installer metadata with real tools (msitools, plist) instead of guessing.
allowed-tools: Bash, Read, Write, Edit, Grep, Glob, WebFetch, WebSearch
model: opus
effort: high
---

You are adding a Fleet-maintained app (FMA) to this repo: $ARGUMENTS

The authoritative contributor docs are [ee/maintained-apps/README.md](../../../ee/maintained-apps/README.md). This skill captures the workflow PLUS the hard-won gotchas the README doesn't cover. Read the README too, but follow the rules here.

## Golden rule: verify, don't guess

The single biggest source of wasted cycles is trusting winget/Homebrew metadata for the fields that must match what osquery actually sees on a host. **The catalog metadata (winget `PackageName`/`Publisher`, cask names) frequently does NOT match the installed app's registry/bundle identity.** Always confirm identity fields against the real installer:

- **Windows `unique_identifier`** must equal the registry **DisplayName** (osquery `programs.name`).
- **Windows publisher** in the exists query must equal the registry **Publisher** (osquery `programs.publisher`).
- **macOS `unique_identifier`** must equal the app's **CFBundleIdentifier**.
- **Version** must reconcile with what osquery reports (`programs.version` on Windows; `bundle_short_version`/`bundle_version` on macOS).

Real examples from this codebase where the metadata lied:
| App | winget/cask says | Registry/bundle actually is |
|-----|------------------|------------------------------|
| Amazon Corretto | PackageName "Amazon Corretto 25" | DisplayName `Amazon Corretto (x64)` (no version), Publisher `Amazon` |
| Genesys Cloud | PackageName "GenesysCloud" | DisplayName `GenesysCloud` (you'd guess "Genesys Cloud") |
| P4V | PackageName "P4 Apps", locale Publisher "Perforce Software, Inc." | DisplayName `P4 Apps`, Publisher `Perforce Software` |
| GoToMeeting | MSI ProductName "GoToMeeting 10.19.19950" | registry DisplayName `GoToMeeting 10.19.0.19950` (bootstrapper!) |

## Prerequisites (one-time)

```bash
brew install msitools          # provides msiinfo for MSI inspection (macOS dev box)
gh auth status                 # gh CLI for reading winget-pkgs manifests
```

## Verification toolkit

### 1. Read the winget manifest (Windows)
```bash
# List packages under a publisher, then versions (NOTE: dirs sort alphabetically,
# so "21.0.11" sorts before "21.0.9" — use sort -V to find the true latest)
gh api 'repos/microsoft/winget-pkgs/contents/manifests/<x>/<Publisher>' --jq '.[].name'
gh api 'repos/microsoft/winget-pkgs/contents/manifests/<x>/<Pub>/<Pkg>' --jq '.[].name' | sort -V | tail
# Installer manifest: InstallerType, Scope, arch, URL, SHA, ProductCode, UpgradeCode, InstallerSwitches
gh api 'repos/microsoft/winget-pkgs/contents/manifests/<x>/<Pub>/<Pkg>/<ver>/<Pkg>.installer.yaml' --jq '.content' | base64 -d
# Locale manifest: Publisher, PackageName, ShortDescription
gh api 'repos/.../<Pkg>.locale.en-US.yaml' --jq '.content' | base64 -d | grep -E "Publisher:|PackageName:|ShortDescription:"
```

### 2. Inspect the MSI (Windows) — the authoritative source for identity
```bash
curl -sIL "<InstallerUrl>" | grep -i content-length    # check size first
cd /tmp && curl -sL -o app.msi "<InstallerUrl>"
msiinfo export /tmp/app.msi Property | grep -iE "ProductName|ARPDISPLAY|Manufacturer|ProductVersion|UpgradeCode|ProductCode|ALLUSERS|ARPSYSTEMCOMPONENT"
msiinfo export /tmp/app.msi Registry   # custom ARP writes, if any
rm -f /tmp/app.msi
```
Map MSI properties → FMA fields:
- `ProductName` → registry DisplayName → `unique_identifier` (unless `ARPDISPLAYNAME` overrides it)
- `Manufacturer` → registry Publisher → `program_publisher` (if it differs from the winget locale Publisher)
- `ProductVersion` → expected `programs.version` (but see bootstrapper caveat below)
- `UpgradeCode` → for upgrade-code uninstall scripts
- `ALLUSERS=1` → installs per-machine regardless of switches
- **`ARPSYSTEMCOMPONENT=1` → STOP: this is a bootstrapper (see Pitfall 2)**

### 3. Inspect the macOS app bundle (DMG)
```bash
cd /tmp && curl -sL -o app.dmg "<cask url>"
MP=$(mktemp -d); hdiutil attach -nobrowse -readonly -mountpoint "$MP" app.dmg >/dev/null
APP=$(find "$MP" -maxdepth 1 -name "*.app" | head -1)
/usr/libexec/PlistBuddy -c "Print :CFBundleIdentifier" "$APP/Contents/Info.plist"        # → unique_identifier
/usr/libexec/PlistBuddy -c "Print :CFBundleShortVersionString" "$APP/Contents/Info.plist"
/usr/libexec/PlistBuddy -c "Print :CFBundleVersion" "$APP/Contents/Info.plist"
hdiutil detach "$MP" >/dev/null; rm -f app.dmg
```
(For pkg-format casks, the bundle id is harder to read offline — the cask `zap`/`uninstall` `pkgutil`/`launchctl`/`savedState` paths are strong hints, e.g. `<bundleid>.savedState`.)

### 4. Silent install/uninstall flags — use documented sources, never guess
- The winget installer manifest's `InstallerSwitches` (`Silent`, `Custom`) is the first source.
- **silentinstallhq.com** has per-app guides with the exact switches (e.g. GoToMeeting uses `/silent`, not `/S`). Use `WebFetch` on `https://silentinstallhq.com/<app>-silent-install-how-to-guide/`.
- Cross-check the vendor's own docs.

## Workflow

### macOS (Homebrew cask)
1. Find the cask: `curl -s https://formulae.brew.sh/api/cask/<token>.json`
2. Inspect the DMG/pkg for the real `CFBundleIdentifier` (toolkit #3).
3. Create `ee/maintained-apps/inputs/homebrew/<token>.json` — minimal: `name`, `slug` (`<app>/darwin`), `unique_identifier` (bundle id), `token`, `installer_format` (`dmg`/`pkg`/`zip`), `default_categories`. Install/uninstall scripts auto-generate from the cask (artifacts + zap).
4. Generate, add description, check icon (below).

### Windows (winget)
1. Read the winget manifests (toolkit #1). Pick **machine** scope, **x64** (or the only arch available — some apps are x86-only).
2. **Inspect the MSI** (toolkit #2) to confirm DisplayName, Publisher, version, codes, and to detect bootstrappers.
3. Create `ee/maintained-apps/inputs/winget/<slug-name>.json`:
   - `name` (catalog display, can be friendly), `slug` (`<app>/windows`), `package_identifier`, `unique_identifier` (= verified DisplayName), `installer_arch`, `installer_type`, `installer_scope`, `default_categories`.
   - `program_publisher` if registry Publisher ≠ winget locale Publisher.
   - `fuzzy_match_name` / `exists_query` as needed (below).
   - `install_script_path` / `uninstall_script_path` for any non-MSI-machine installer.
4. Generate, add description, check icon.

### Installer type mapping (winget `InstallerType` → FMA `installer_type` + silent flags)
| winget type | FMA type | install silent | uninstall |
|-------------|----------|----------------|-----------|
| `msi`, `wix` | `msi` | auto (`msiexec /i /quiet /norestart`) | auto upgrade-code (machine scope only) |
| `nullsoft` (NSIS) | `exe` | `/S` | registry UninstallString + `/S` |
| `inno` (Inno Setup) | `exe` | `/VERYSILENT /SUPPRESSMSGBOXES /NORESTART` | registry UninstallString + same |
| `burn` (WiX bundle) | `exe` | `/quiet /norestart` | bundle UninstallString `/uninstall /quiet /norestart` |
| `msix` | `msix` | n/a | n/a |

The ingester only auto-generates scripts for **machine-scope MSI**. Everything else needs custom `install_script_path` + `uninstall_script_path`. MSI success codes to treat as success: `0`, `3010` (reboot required), `1641` (reboot initiated).

### Generate, validate, finalize
```bash
go run cmd/maintained-apps/main.go --slug="<app>/<platform>" --debug
```
- Output lands in `ee/maintained-apps/outputs/<slug>.json`; an entry is appended to `outputs/apps.json` with an **empty description** — fill it in (sentence case, "`<App>` is a(n)..."). The generator does NOT update `unique_identifier` on an existing apps.json entry — edit it manually if you change it.
- Verify the generated SHA matches the manifest, and the exists/patched queries look right: `grep -E 'exists|patched|sha256' outputs/<slug>.json`.
- `python3 -m json.tool ee/maintained-apps/outputs/apps.json >/dev/null` to confirm valid JSON.
- **Icon**: check `frontend/pages/SoftwarePage/components/icons/index.ts` for a key matching the lowercased catalog `name`. If missing, generate via [tools/software/icons](../../../tools/software/icons) before merge. Icons key off the lowercased `name`, so platforms sharing a `name` share an icon. **Tip:** when the vendor doesn't publish a clean app icon, MSIs almost always embed one — extract with `msiinfo extract <app>.msi Icon.<name>.ico > app.ico` (find the icon stream name with `msiinfo streams <app>.msi` or `msiinfo export <app>.msi Icon`). The embedded icon is typically multi-resolution and high quality (Azul Zulu JDK precedent).
- The validator is a Windows/macOS host (often **ephemeral** — you can't query it after the run). To cross-compile the Windows validator after editing it: `GOOS=windows go build ./cmd/maintained-apps/validate/`.

> **🚨 CRITICAL — re-run the generator any time you edit an install/uninstall script.**
> The output JSON at `ee/maintained-apps/outputs/<slug>/windows.json` (and `darwin.json`) embeds the **full script body** as a string under `refs`, not a file path. The validator (and Fleet in production) execute the script content from the output JSON, not the file at `inputs/winget/scripts/`. Editing the input `.ps1` without re-running `go run cmd/maintained-apps/main.go --slug="<app>/windows"` ships stale scripts to CI — the validator will look exactly like it ran your old code (this has burned multiple validation cycles; the windsurf and VirtualBox `QuietUninstallString` fixes both shipped stale because the source was edited without regenerating).
>
> Sanity check before pushing fixes: `grep -F 'your-new-snippet' ee/maintained-apps/outputs/<slug>/windows.json` should match. If it doesn't, you forgot to regenerate.

## Field semantics

| Field | Meaning |
|-------|---------|
| `name` | Catalog display name. Can be friendly; share across platforms to group in the FMA library. |
| `unique_identifier` | Value that matches inventory: Windows registry DisplayName, macOS CFBundleIdentifier. |
| `program_publisher` (winget) | Overrides the exists-query publisher when registry Publisher ≠ winget locale Publisher. |
| `fuzzy_match_name` (winget) | `true` → `name LIKE '<unique_identifier> %'`. A string → `name LIKE '<that string>'` verbatim (e.g. `"Mozilla Firefox % ESR %"`, `"IntelliJ IDEA 20%"`). |
| `exists_query` (winget) | Replaces the generated exists query verbatim. The patched query is DERIVED from it (appends `AND version_compare(...) < 0`). |
| `installer_scope` | Must match the winget manifest's Scope — you can't pick machine if only user exists. |

`patch_policy_path` exists in the input struct but is **dead code** (unused since the patched query became auto-generated). Don't use it; there is no patched-query override other than shaping `exists_query` or a hard-coded per-app branch in the ingester (Docker Desktop precedent).

## Pitfalls (each one cost a validation cycle in practice)

**1. Identity mismatch.** Covered above — always verify DisplayName/Publisher/bundle-id from the real installer. A wrong `unique_identifier` or publisher makes the exists query silently never match (Fleet thinks the app is never installed; the validator may still pass because it searches loosely). When the catalog `name` differs from the DisplayName (e.g. name "Genesys Cloud", DisplayName "GenesysCloud"), the Windows validator finds it via the `unique_identifier` search clause — so set `unique_identifier` correctly even if `name` stays friendly.

**2. Bootstrapper installers.** Red flags in the MSI Property table: `ARPSYSTEMCOMPONENT=1`, a "Setup"-style filename, or properties like `G2MACTION`/`...CLIENT=Setup`. These MSIs install a *separate* app that self-registers its own ARP entry with a *different* version and uninstaller — so the MSI's `ProductVersion`/`ProductCode`/`UpgradeCode` do NOT match the registry, and upgrade-code uninstall fails. They often install per-user (invisible to a SYSTEM-context uninstall). Treat as poor FMA candidates; if you must ship, use a registry-lookup uninstall and flag it as unverifiable.

**Confirming a bootstrapper via `msiinfo`:** export the `CustomAction` table — a bootstrapper's Install action will exec an embedded EXE (often via Binary table reference) with a long argument string referencing other MSI properties like `[ACCEPTGDPR]`, `[SILENT]`, `[INSTALLDIR]`. Zoom Rooms is the canonical example: `--targetdir=[ProgramFiles64Folder]ZoomRooms\Installer --accept_gdpr=[ACCEPTGDPR] --silent=[SILENT] ...`. When you see this shape, the MSI's documented silent flags must include the property assignments (`ACCEPTGDPR=true SILENT=true ...`) or the inner EXE hangs on a UI prompt and the validator kills it at the 5-minute install timeout. The registry will then carry **two ARP entries** — one for the bootstrapper MSI ("Zoom Rooms Installer"), one for the actual app ("Zoom Rooms") — and the exists query must explicitly exclude the bootstrapper (e.g. `name LIKE 'Zoom Rooms%' AND name NOT LIKE '%Installer%'`).

**Read the winget `Silent:` line as a string, not as a flag list.** A few wrapper-style bootstrappers (Splashtop Business, others) document `Silent:` as a literal *command* that winget runs internally — e.g. `Silent: msiexec /norestart /i setup.msi CA_UPGRADE=1 /qn`. That is NOT the flag set the bootstrapper EXE accepts; it's what winget invokes after the bootstrapper extracts `setup.msi`. Confirm the actual EXE flags from the vendor or silentinstallhq, or pivot to a sibling product's known-good wrapper pattern (Splashtop Business uses the same `prevercheck /s /i <PROPERTIES>` form as Splashtop Streamer, with vendor-specific MSI properties slotted into `/i`). Don't blindly pass the contents of `Silent:` to the bootstrapper EXE.

**Oracle-style wrappers use `--silent --ignore-reboot --msiparams "<KEY=val ...>"`.** VirtualBox is the canonical example: a custom Oracle bootstrapper that wraps a WiX MSI. The bootstrapper accepts `--msiparams "REBOOT=ReallySuppress VBOX_INSTALLDESKTOPSHORTCUT=0 VBOX_INSTALLQUICKLAUNCHSHORTCUT=0 VBOX_START=0"` to push MSI properties through. After install returns, **poll for `msiexec`/`drvinst`/`DIFxApp*` processes to settle** (60–120s ceiling) before exiting — otherwise the temp-installer cleanup hits "Access is denied" and the validator reports a noisy warning even though the install succeeded. Same poll pattern works for any wrapper that spawns driver-install helpers.

**3. Unquoted UninstallString with spaces.** Registry uninstall strings come in three shapes; parse defensively (this broke every JetBrains app — `C:\Program Files\JetBrains\PhpStorm 2026.1.2\bin\Uninstall.exe` is unquoted WITH spaces):
```powershell
if ($u -match '^\s*"([^"]+)"\s*(.*)$') {            # quoted
} elseif ($u -match '(?i)^\s*(.+?\.exe)\s*(.*)$') { # unquoted, may contain spaces — capture through .exe
} elseif ($u -match '^\s*(\S+)\s*(.*)$') {          # bare token (e.g. MsiExec.exe /X{GUID})
}
```

**4. Version mismatches.**
- **JetBrains (Windows):** registry version is a build number (`261.24374.185`), but Fleet's `MutateSoftwareOnIngestion` rewrites it to the marketing version parsed from the NAME ("PhpStorm 2026.1.2" → "2026.1.2"). This requires `Vendor` (publisher) to contain "jetbrains". The validator must select `publisher` and set `Software.Vendor` for this to fire (already wired in `windows.go`).
- **macOS:** the validator's `checkVersionMatch` compares the cask version against BOTH `CFBundleShortVersionString` AND `CFBundleVersion` — so a cask version that equals `CFBundleVersion` passes even if `CFBundleShortVersionString` differs.
- **Don't add existence-only version skips to the validator lightly.** They make the patch policy always report "patched" (never flags outdated installs). Only when the version genuinely can't be reconciled. If osquery's version actually matches the FMA version (verify in the validator log: `Found app: '...' Version: X`), no skip is needed.

**5. Scope / SYSTEM context.** Fleet runs installs as SYSTEM (elevated). A per-user installer lands in the SYSTEM profile (useless). Force machine-wide with the installer's all-users switch (`ALLUSERS=1`/`2`, `/ALLUSERS`, `G2MINSTALLFORALLUSERS=1`). A per-user uninstaller likewise can't be reached from a SYSTEM-context script (its ARP entry is in the logged-in user's HKCU).

**6. Multi-version / sibling products sharing a DisplayName.**
- Corretto 21 and 25 both register as `Amazon Corretto (x64)` — pin each with `exists_query ... AND version LIKE '<major>.%'`.
- IntelliJ Ultimate's DisplayName `IntelliJ IDEA <ver>` also matches Community's `IntelliJ IDEA Community Edition <ver>` — exclude siblings in `exists_query` (`AND name NOT LIKE 'IntelliJ IDEA Community%'`) or use a custom `fuzzy_match_name` pattern.

**7. Non-pinned installer URLs.** Some manifests point at a "latest" redirect (e.g. `link.gotomeeting.com/latest-msi`). The pinned SHA drifts when the vendor ships a new build, breaking Fleet installs until the FMA auto-update bumps it. Note this in the PR.

**8. NSIS in-place uninstall (`_?=`) path with spaces.** NSIS silent uninstall (`/S`) normally copies the uninstaller to temp and detaches, so the script can't wait on it. Pass `_?=<InstallLocation>` to force in-place execution so `Start-Process -Wait` gets a real exit code. The value MUST be quoted — `_?="$installDir"` — because install dirs routinely contain spaces (`C:\Program Files\Android\Android Studio`); an unquoted path is truncated and the uninstall silently fails (Android Studio precedent). In a double-quoted PowerShell string, escape the inner quotes with backticks: `("$existingArgs _?=`"$installDir`"").Trim()`. This is separate from pitfall #3, which only quotes the *exe path*, not the `_?=` argument.

**`_?=` is NOT universally supported by NSIS uninstallers.** Confirmed cases where passing `_?=` causes the uninstaller to return exit code 2 ("user cancelled") and refuse to run: DBeaver (CE / EE / Lite / Ultimate), electron-builder-based NSIS (Notion Calendar). Only add `_?=` when:
- silentinstallhq's documented uninstall line includes it, OR
- you've actually observed the early-return symptom (sub-1-second uninstall + app still found afterward) for THIS specific installer.

If silentinstallhq documents a plain `Uninstall.exe /S` (or `/allusers /S`) form without `_?=`, use that and trust the validator timing — DBeaver Community, for example, has validated successfully across many runs without `_?=`. Don't preemptively add it.

**9. NSIS DisplayName with embedded version — fuzzy match required.** Many NSIS installers (DBeaver, Obsidian, Audacity, draw.io, etc.) write `DisplayName = "<App> <Version>"` to the registry. An exact-match `unique_identifier` ("DBeaver") + default `fuzzy_match_name: false` produces `WHERE name = 'DBeaver'` which never matches "DBeaver 26.0.5". Default to `fuzzy_match_name: true` for NSIS installers unless you have specifically verified the DisplayName excludes the version. If you guess wrong, uninstall scripts that match `DisplayName -eq 'X'` will also silently fail to find anything — uninstall script's match clause must match the same shape as the exists query.

**10. Per-vendor version-string quirks (registry vs catalog).** osquery's `programs.version` comes directly from the registry's `DisplayVersion`. A few vendors mangle it:
- **AnyDesk** writes `ad 9.7.4` (literal "ad " prefix on the version).
- **Surfshark** writes `6.9.0999` (drops the dot before the last segment — `6.9.0.999` in winget).
- **Lens** ships catalog versions like `2026.5.250609-latest` (suffix that won't `version_compare` cleanly).

The validator's `normalizeVersion` only pads/trims dot-segments — it doesn't handle prefixes or missing punctuation. These apps need either a per-app sanitizer in `server/service/osquery_utils/queries.go` (like the JetBrains build-number normalizer) or an existence-only validator skip. Both have downsides — don't ship these until you've decided which.

**Non-standard installer exit codes.** Beyond MSI's 3010 (reboot required) and 1641 (reboot initiated), some EXE installers use their own codes for benign outcomes:
- **Wacom Tablet Driver** returns exit code `2` to signal "reboot required" (driver is fully installed at that point, visible in Add/Remove Programs). Treat `2` as success for Wacom.
- Other InstallShield-derived drivers occasionally do the same. When you see exit-2 plus the app actually present in `programs`, check the vendor's deployment notes before treating it as a failure.

**11. Empty `DisplayVersion` in registry.** Some installers write `DisplayName` but leave `DisplayVersion` empty (osquery returns `Version: ""`). Confirmed cases: Sublime Text, Sublime Merge (Inno Setup variants that disable DisplayVersion), DeepL. Validator log signature: `Found app: 'X' at ..., Version: ` followed by `App version 'Y' was not found by osquery`. The only fix is an existence-only check in the validator (see the Sublime Text branch in `cmd/maintained-apps/validate/windows.go`); patch policy will then never flag outdated installs, so make sure that's acceptable for the app.

**12. Portable / per-user installers that don't register in `programs`.** Some "installers" just extract a self-contained directory and don't write to HKLM\Uninstall or HKCU\Uninstall (Tor Browser is the canonical example). Validator log signature: install script returns 0 but `App version 'X' was not found by osquery` with no `Found app:` line. The validator's `programs` query returns nothing; even the AppX provisioned-package fallback won't catch it. These need a Codex-CLI-style file-based detection path in the validator — not a one-line input fix. Treat as poor FMA candidates without that scaffolding.

**13. NSIS `/AllUsers` is case-sensitive on some installers.** Most NSIS apps accept `/S /allusers` (any case), but Evernote 10.x+ documents `/AllUsers /S` (capital A, capital U, order matters). Passing `/S /allusers` caused an access violation (`0xc0000005`) in CI. Follow silentinstallhq's exact casing/order when documented, especially for installers that distinguish multi-user behavior via custom switches rather than the NSIS plugin.

**14. WiX Burn bundle / Oracle wrapper uninstall via registry is NOT silent by default.** For WiX Burn bundles (ExpressVPN) and Oracle-style EXE wrappers around MSIs (VirtualBox), the registry `UninstallString` is the wrapper exe. Invoking it without explicit silent flags shows a confirmation UI; in CI the install hangs (ExpressVPN) or the uninstall returns success but the app stays installed (VirtualBox). The fix is to:
- For burn bundles: append `/uninstall /quiet /norestart` to the UninstallString and treat 3010 as success.
- For Oracle's exe wrapper: parse the MSI ProductCode from `UninstallString` (or fall back to the registry key's `PSChildName` when it's a GUID), then run `MsiExec.exe /X{GUID} /qn /norestart`. After it returns, poll for residual `msiexec`/`drvinst` processes (see pitfall #2 Oracle addendum).
- For both: handle the case where the entry exposes only `QuietUninstallString` (not `UninstallString`). The guard should be `if (-not $selected -or (-not $selected.UninstallString -and -not $selected.QuietUninstallString))`, and the command picker should `if ($selected.QuietUninstallString) { ... } else { ... }` (windsurf, VirtualBox precedent).

**15. NSIS uninstaller self-fork — `Start-Process -Wait` returns too early.** Standard NSIS uninstallers copy themselves to `%TEMP%\Au_.exe` and exec that copy, then the original `Uninstall *.exe` exits immediately. `Start-Process -Wait` waits only for the *parent*, so your uninstall script returns ~1 second after launch while the real work is still in progress — and the validator's post-uninstall check then finds the app still installed (Evernote 11.x hit this). Two mitigations, use both:

1. Pass `_?=<install_dir>` (an NSIS-documented option) to disable the self-copy — see pitfall #8 for the quoting rules and #8 follow-up for installers where `_?=` is rejected.
2. After `Start-Process -Wait` returns, poll for any leftover NSIS helper processes (`Au_*.exe`, `Un_*.exe`) and any process whose `.Path` is under the app's install dir, with a 5-minute deadline. Burn a few seconds polling rather than risk an early false-pass.

Same caveat applies to install scripts for NSIS installers that also use the multi-user plugin — most just block correctly, but if you see a `no changes detected in C:\Program Files` warning followed by the app appearing in `programs` anyway, the install probably also forked and you got lucky.

**16. Validator download timeout (~2 minutes) — large installers fail before they run.** The validator downloads the installer over HTTP before executing the script and has a hard download deadline (observed ~2 minutes). Installers above roughly 250–300MB on slow CDNs hit `context deadline exceeded` and never start. Confirmed cases: LibreOffice (370MB MSI from documentfoundation.org), Tableau Desktop (617MB EXE from downloads.tableau.com). Workarounds at the FMA level: none — this is a validator-infrastructure constraint. Drop the app for now and note it in the PR; revisit when the validator gains streaming/longer timeout support.

**17. Validator install timeout (~5 minutes) — bootstrappers that hang on prompts.** Separate from the 2-min download cap, the validator gives install scripts ~5 minutes total before killing them with exit 1. Confirmed-failed apps: Citrix Workspace, Splashtop Business, Splashtop Streamer (re-run case), Zoom Rooms (before silent properties fix). Log signature: install starts, exactly 5:00 later the validator logs `Error executing install script: exit status 1` with no script output, often followed by `New application detected at: <path>` because the filesystem check sees a partial install. Root cause is almost always an interactive prompt the silent flag didn't suppress — confirm against silentinstallhq and the MSI's `CustomAction` table (pitfall #2 addendum) before ever assuming the installer is "just slow."

**18. Per-user installers and Squirrel exit codes.** Several Squirrel/Electron installers (Mattermost, Proton Mail, Grammarly) document `/S` but in CI return non-zero exit codes with no stdout. The cause is usually one of:
- The installer downloads more bits after the bootstrap exit (return value tracks the bootstrap, not the full install).
- The installer needs an interactive user session that doesn't exist in SYSTEM context.
- The installer writes to AppData under the SYSTEM profile and fails because that path doesn't make sense.

Workarounds when you must ship per-user Squirrel installers: prefer apps that publish a separate MSI/WiX option in the winget manifest (Bruno, Snagit, **Mattermost** all do this — set `installer_type: msi` and the generator picks the WiX/MSI variant over the nullsoft user-scope one). Otherwise, accept that these may need the Slack/Claude-style scheduled-task pattern.

**19. Multi-variant winget manifests — prefer machine-scope MSI.** Many winget manifests list multiple installer variants (different InstallerTypes / Scopes / Architectures). The ingester picks based on `installer_type` + `installer_arch` + `installer_scope` in your input JSON; if you don't pin it explicitly, you may get the first listed variant which is often a per-user Squirrel/NSIS that's tricky in SYSTEM context. Always check `gh api .../manifest/installer.yaml` for ALL variants before choosing, and prefer machine-scope MSI/WiX when offered. Bruno, Snagit, Mattermost, Splashtop products all expose both forms — the MSI path is dramatically more reliable.

## Reading validator output

The validator emits a known set of log lines you can pattern-match for triage:

| Log line | Diagnosis |
|---|---|
| `Error executing install script: exit status N` | Install script returned non-zero. Look up the code: 1223 = `ERROR_CANCELLED` (silent flag wrong / UAC denied); 0xc0000005 = access violation (bad arguments crashed installer); 1603 = MSI generic failure; 3010 = success but reboot required (treat as success in scripts); 2 sometimes = reboot required for InstallShield (pitfall #10 addendum). |
| 5:00 between install start and `exit status 1` | Hit the validator's ~5-minute install timeout (pitfall #17). Almost always an interactive prompt the silent flag didn't suppress. |
| `context deadline exceeded` during Downloading | Hit the validator's ~2-minute download cap (pitfall #16). Installer too large or CDN too slow. No script-level fix. |
| `no changes detected in C:\Program Files directory after running application script` | Either a per-user install (writes to %LOCALAPPDATA%, expected for user-scope) or the install silently failed. Cross-reference with the next `Found app:` line. |
| `Found app: 'X' at <path>, Version: <ver>` | Registry/osquery actually returned a match. `unique_identifier` and exists query are working at the broad search level — but compare the returned DisplayName/Version with what you set in the input. |
| `Found app: 'X' at , Version: ` (empty path AND empty version) | App found but registry skipped DisplayVersion. See pitfall #11. |
| `App version 'X' was not found by osquery` after a `Found app:` line | Either version mismatch (compare exact strings — prefix-padding rules in `normalizeVersion` only handle dot-segments) or the app wasn't in `programs` at all (no `Found app:` printed). |
| `App version 'X' was found after uninstall` | Uninstall script's pattern didn't match the registry entry, OR the uninstall command ran but didn't actually uninstall (most often: registry UninstallString invoked without the right silent flags). |
| `failed to remove ... Access is denied` on temp dir cleanup | Installer process or its child (drvinst, DIFxApp, msiexec) still holding files. Add a process-settling poll after `Start-Process -Wait` (see pitfall #2 Oracle addendum). |
| 20+ minutes between two app start/end timestamps | Install hung. Almost always an interactive prompt the silent flag didn't suppress. |

A green "passed with warnings" outcome can hide real problems — read the warning list. A `Found app: 'Tower Deployment Tool' at , Version: 12.1.557.0` line passing for the **Tower** Git client looked fine in summary, but the broad `LOWER LIKE '%tower%'` query accidentally matched an unrelated SaaSGroup product. Tighten the exists query whenever the catalog name is generic (Tower, Lens, Compass, Bridge).

## Pre-ship checklist
- [ ] Identity fields verified against the real installer (MSI Property table / Info.plist), not guessed.
- [ ] `unique_identifier` = registry DisplayName / bundle id; `program_publisher` set if needed.
- [ ] Multi-variant winget manifest? Picked machine-scope MSI/WiX over per-user NSIS/Squirrel where available (pitfall #19).
- [ ] Silent install/uninstall flags from winget `InstallerSwitches` or silentinstallhq, not invented. For bootstrappers, MSI property assignments (`ACCEPTGDPR=true`, `--msiparams "..."`) are passed through (pitfall #2 addendum).
- [ ] Custom uninstall (non-MSI-machine) uses the defensive UninstallString parser AND accepts `QuietUninstallString` when `UninstallString` is missing (pitfall #14).
- [ ] NSIS in-place uninstall passes `_?="$installDir"` (quoted) so `/S` waits and survives spaces in the path — unless the specific installer rejects `_?=` (DBeaver, electron-builder; see pitfall #8 follow-up).
- [ ] For Oracle/driver-installing wrappers, the install script polls `msiexec`/`drvinst`/`DIFxApp*` to settle so temp cleanup doesn't fail with "Access is denied" (pitfall #2 Oracle addendum).
- [ ] Version reconciles with osquery (or a documented validator exception applies — not a blanket skip).
- [ ] Generated SHA matches the manifest; exists/patched queries reviewed; `apps.json` valid + description filled.
- [ ] Bootstrapper sibling collision? `exists_query` explicitly excludes the bootstrapper's own ARP entry (e.g. `AND name NOT LIKE '%Installer%'`) (pitfall #2 addendum, Zoom Rooms).
- [ ] **Re-ran the generator after any script edit** (`go run cmd/maintained-apps/main.go --slug=<app>/windows`) — the embedded script body in `outputs/<slug>/windows.json` matches the on-disk `inputs/winget/scripts/*.ps1` (CRITICAL workflow note).
- [ ] Installer size under ~250–300MB OR you accept the validator-download-cap failure (pitfall #16).
- [ ] Icon exists or is generated. (MSI installers? `msiinfo extract <app>.msi Icon.<name>.ico > app.ico` for a high-fidelity source.)
- [ ] Bootstrapper / per-user / latest-URL risks flagged in the PR if present.
- [ ] If you changed shared code (`cmd/maintained-apps/validate/*.go`, ingesters), call it out in the PR and run `GOOS=windows go build ./cmd/maintained-apps/validate/` + `go test ./cmd/maintained-apps/...`.
