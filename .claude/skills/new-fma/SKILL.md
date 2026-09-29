---
name: new-fma
description: Add a Fleet-maintained app (FMA) for macOS (Homebrew) and/or Windows (winget), or write/clean up an FMA's custom install or uninstall script. Use when asked to "add X as a macOS/Windows FMA", "add a Fleet-maintained app", to debug FMA validator failures, or to review comments in an FMA script. Emphasizes verifying installer metadata with real tools (msitools, plist) instead of guessing, proving where an installer actually lands when run as SYSTEM, and keeping shipped script comments admin-facing.
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

The same rule applies to **where the app installs**: verify it on a host running as SYSTEM rather than believing the manifest's `Scope`. See [Per-user installers and the SYSTEM context](#per-user-installers-and-the-system-context).

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

### Custom script comments: these ship to customers

FMA install/uninstall scripts are not internal code. They're returned verbatim by `GET /fleet/software/fleet_maintained_apps/:id` and by the software title endpoint, and rendered in the "Install script" / "Uninstall script" editors of the Edit software modal ([AdvancedOptionsFields.tsx](../../../frontend/pages/SoftwarePage/components/forms/AdvancedOptionsFields/AdvancedOptionsFields.tsx)), where an admin reads them and can edit them. Every comment you leave is product copy — treat it like the app description, not like a commit message.

Budget: the Fleet template header (`# Learn more about .exe install scripts:` + URL) if the script started from a template, then **at most ~4 lines** of app-specific comment. Of the 580 scripts in `inputs/*/scripts/`, only 51 open with a longer block than that — a big header is the exception you have to justify, not the norm.

**Keep** a comment only if an admin who edits this script would break something without it, or would be surprised at install time:
- Host-visible side effects: the app is force-quit, users are logged out, a reboot happens, existing config is preserved or deleted.
- Scope and destructiveness decisions — e.g. [box-tools-uninstall.sh](../../../ee/maintained-apps/inputs/homebrew/scripts/box-tools-uninstall.sh): removal sweeps every local user's home, and only the Box Edit subdirectory goes because the parent is shared with Box Drive.
- Constraints that must survive an edit: the required switch and why the obvious one is wrong (`/VERYSILENT` — this is Inno Setup, `/S` opens the GUI), removal ordering, "must run as the logged-in user."
- Exit-code meanings (`1605` = not installed, `3010` = reboot required).

**Cut** — this belongs in the PR description, not the shipped script:
- Fleet's own tooling: "the validator's 10-minute timeout", "hangs in CI", "the ingester", "osquery's programs table". A customer has no validator.
- Catalog archaeology: what winget/Homebrew metadata claimed vs. reality, `silentinstallhq.com` links, PR/issue numbers.
- Debugging narrative: what you tried first and why it failed ("a plain `Start-Process -Wait` would block until killed").
- First person ("we", "our", "ourselves") — describe what the script does, in present tense and sentence case.
- Restating the next line (`# Prints the exit code` above a `Write-Host`).

If the fact matters at run time rather than at edit time, `Write-Host`/`echo` it instead of commenting it — script output lands in the host's software install details, which is where an admin debugging a failure actually looks.

Before/after — [darktable_install.ps1](../../../ee/maintained-apps/inputs/winget/scripts/darktable_install.ps1)'s 20-line header carries three admin-relevant facts and 16 lines of internal history:
```powershell
# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# darktable uses an Inno Setup installer: it needs /VERYSILENT (the NSIS /S
# switch winget's metadata implies does nothing) and installs machine-wide when
# elevated. Its installer stays running after a silent install, so this script
# waits for darktable to register in Programs and Features, then stops it.
```
Dropped: that winget mislabeled the installer type, that `PrivilegesRequiredOverridesAllowed=dialog` rules out `/ALLUSERS`, that the lingering process holds the installer file lock, what a plain `-Wait` did. All of it goes in the PR body, where reviewers need it and customers don't see it.

Body comments follow the same rule — keep the one above a non-obvious registry match or a load-bearing helper, drop the rest. **When you touch an existing script for any reason, prune its comments in the same edit.**

### Generate, validate, finalize
```bash
go run cmd/maintained-apps/main.go --slug="<app>/<platform>" --debug
```
- Output lands in `ee/maintained-apps/outputs/<slug>.json`; an entry is appended to `outputs/apps.json` with an **empty description** — fill it in (sentence case, "`<App>` is a(n)..."). The generator does NOT update `unique_identifier` on an existing apps.json entry — edit it manually if you change it.
- Verify the generated SHA matches the manifest, and the exists/patched queries look right: `grep -E 'exists|patched|sha256' outputs/<slug>.json`.
- `python3 -m json.tool ee/maintained-apps/outputs/apps.json >/dev/null` to confirm valid JSON.
- **Icon**: check `frontend/pages/SoftwarePage/components/icons/index.ts` for a key matching the lowercased catalog `name`. If missing, generate via [tools/software/icons](../../../tools/software/icons) before merge. Icons key off the lowercased `name`, so platforms sharing a `name` share an icon. After generating, confirm the new `SOFTWARE_NAME_TO_ICON_MAP` key really is the lowercased `name` — when `name` and slug differ it is easy to end up keyed off the slug, and the lookup then misses. If an icon component for that name already exists, revert any regenerated `.tsx`/`.png` and reuse it.
- The validator is a Windows/macOS host (often **ephemeral** — you can't query it after the run). To cross-compile the Windows validator after editing it: `GOOS=windows go build ./cmd/maintained-apps/validate/`.

## Per-user installers and the SYSTEM context

Fleet runs install and uninstall scripts as **SYSTEM**. Most Windows FMA breakage traces back to this, and it does not reproduce in CI (see [Validating for real](#validating-for-real)), so it ships silently.

**Always prefer machine-wide.** Try the installer's all-users switch first (`ALLUSERS=1`/`2`, `/ALLUSERS`, `G2MINSTALLFORALLUSERS=1`). Machine-wide installs land in `Program Files` with an HKLM registration and everything downstream just works.

**Verify the switch was honoured — do not trust that it was.** Signal accepts `/S /allusers` and silently ignores it, still installing per-user. Install it on a real host as SYSTEM and look at where the payload and the registration actually went. Equally, do not trust the input's `installer_scope`: it comes from the winget manifest and is sometimes a default rather than a fact. `bluej`, `julia-app` and `readest` are all declared `installer_scope: user` yet install machine-wide to `Program Files` under HKLM, because their custom scripts already handle SYSTEM deliberately (`bluej` passes `ALLUSERS=2`). Scope alone is not evidence of a bug.

### When the app has no machine-wide mode

Electron/NSIS/Squirrel apps and some Inno apps only ever install into the running user's profile. Run as SYSTEM they land in `C:\Windows\system32\config\systemprofile\AppData\Local\...`, where **no signed-in user can launch them** — the install "succeeds" and is useless. Symptoms vary and none of them says "wrong scope": Notion's installer crashes outright (`0xC0000005`, installing nothing), `amazon-chime` hangs until the timeout, `granola` returns 0 and installs into SYSTEM's profile.

The fix is to hand the installer to the signed-in user via a scheduled task. `figma`, `slack`, `brave`, `arc`, `postman`, `notion` and others follow this shape:

```powershell
$owner = Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' -ErrorAction SilentlyContinue |
    Invoke-CimMethod -MethodName GetOwner -ErrorAction SilentlyContinue |
    Where-Object { $_.User } | Select-Object -First 1
if (-not $owner) { Throw "<App> installs per user and no user is signed in to this host. Sign in and try again." }
$userAccount = "$($owner.Domain)\$($owner.User)"
# Fleet's installer directory is not readable by that user - stage a copy under $env:PUBLIC.
```

Two variations worth knowing:
- **The installer also demands elevation.** `portfolioperformance` fails as a plain user with `ERROR_ELEVATION_REQUIRED` (`0x800702E4`) *and* strands itself when run as SYSTEM. Add `-RunLevel Highest` to `New-ScheduledTaskPrincipal`; it only works if the signed-in user is an admin.
- **`/currentuser` can be counterproductive.** `granola` shipped `/S /currentuser`; plain `/S` as the signed-in user is what lands it correctly.

### The WOW64 trap (why these installs are also unremovable)

Most Electron NSIS stubs are **32-bit** (PE machine `0x014C`). Run as SYSTEM, the WOW64 file system redirector rewrites their `%LOCALAPPDATA%` writes into `C:\Windows\SysWOW64\config\systemprofile\...`, but the `UninstallString` they record still names the unredirected `C:\Windows\system32\...` path. Uninstall scripts run in 64-bit PowerShell, where no redirection applies, so that path does not resolve:

```
Error running uninstaller: This command cannot be run due to the error: The system cannot find the file specified.
```

Any uninstall script for a per-user app needs this fallback so hosts already carrying a stranded install can be cleaned up:

```powershell
if (-not (Test-Path -LiteralPath $exePath)) {
    $redirected = $exePath -replace '(?i)\\system32\\', '\SysWOW64\'
    if ($redirected -ne $exePath -and (Test-Path -LiteralPath $redirected)) { $exePath = $redirected }
}
```

Check the stub's architecture before assuming: read the PE machine type at `e_lfanew + 4`. An HTTP range request for the first 8 KB is enough, so you never download the installer to find out.

### Uninstall must run in the hive that owns the registration

A per-user app's ARP entry lives in the installing user's hive, so a SYSTEM-context script must enumerate every hive, not `HKCU` (which *is* SYSTEM's hive when running as SYSTEM):

```powershell
foreach ($hive in (Get-ChildItem 'Registry::HKEY_USERS' -ErrorAction SilentlyContinue)) {
    if ($hive.Name -match '_Classes$') { continue }
    $roots.Add("Registry::$($hive.Name)\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall")
    $roots.Add("Registry::$($hive.Name)\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall")
}
```

Finding the entry is not enough. **These uninstallers read the directory to remove out of the hive of whoever runs them**, so run as SYSTEM against a real user's install they exit **0 and delete nothing** — Signal left 473 MB behind while reporting success. Run the uninstaller as the user who owns the entry (derive the SID from the key path with `'HKEY_USERS\\(S-1-5-21-[\d-]+)\\'`, translate it to an account, and launch via a scheduled task). Entries genuinely under `S-1-5-18`/`.DEFAULT` — the stranded legacy installs — are the one case where running directly as SYSTEM is correct.

Two details that avoid per-app data:
- **Stop processes by install directory, not by name.** Most apps are not running when you look, so a process-name list is usually empty and useless; matching on the directory also leaves another user's copy of the same app alone.
- **Sweep shortcuts by resolving each `.lnk` target** against the removed directory, rather than by filename — that catches the vendor subfolders installers create under `Start Menu\Programs`.

## Validating for real

**CI cannot catch SYSTEM-context bugs.** The Windows FMA validator's steps run as an interactive **`runneradmin`**, not SYSTEM. Per-user installers therefore land in an ordinary user profile, every path resolves, and the app passes — `signal/windows` passed its shard while broken in the field. If your change concerns scope, profile location or uninstall path resolution, a green CI run proves nothing.

**A passing validator does not prove the shipped queries work.** `appExists` searches with its own fuzzy `LOWER(name) LIKE '%<name>%'`, so it finds an app whose real DisplayName the shipped exact `exists` query would never match. Read the validator log line — `Found app: 'Signal 8.18.0'` — and compare that string against the query you are shipping.

**`fuzzy_match_name` must be verified in both directions.** Inno/NSIS/Electron installers often register `"<Name> <version>"` (`Signal 8.24.1`, `Bdash 1.35.1`, `Notion Calendar 1.133.0`) — those need `fuzzy_match_name`. But plenty register a plain name (`Asana`, `Discord`, `Canva`, `Kiro (User)`) and must keep an exact match; setting `fuzzy_match_name` on those breaks them just as badly, because `name LIKE 'Asana %'` never matches `Asana`. Observe the DisplayName, then decide.

**Test on a host that is actually SYSTEM.** A local Windows VM works: `prlctl exec "<vm>" cmd /c "..."` (Parallels) runs as `NT AUTHORITY\SYSTEM`, which is exactly orbit's context. Assert three things after install — the registration is under a real user SID (`S-1-5-21-…`), there is **nothing** under `S-1-5-18`/`.DEFAULT`, and the payload is in that user's profile — then assert the uninstall leaves no registration, directory or shortcut.

**Mind the test host's architecture.** An ARM64 VM (Parallels on Apple silicon) can only produce false *failures*, never false passes: the per-user/SYSTEM-profile behaviour and the `System32`→`SysWOW64` redirection are OS-level and identical on x64, but x64-only installers with a native-architecture `LaunchCondition` refuse to run at all. Inno says so plainly in `/LOG` — *"This program can only be installed on versions of Windows designed for the following processor architectures: x64"*. Do not read that as an app defect, and do not "fix" it; note that the app needs an x64 host. `kiro` and `antigravity-ide` are the same Inno 6.4.0.1 with identical switches, and only `kiro` runs on ARM64.

**Get the installer's own diagnostics before guessing switches.** Inno takes `/LOG=<file>` and states its abort reason; a WiX Burn bundle takes `/log <file>` and reports blockers such as `Variable: RebootPending = 1` (which makes the inner MSI return 1603 and can survive a reboot on a dirty host). Four guessed switches taught nothing about `jetbrains-toolbox`; one log line explained `antigravity-ide` and `devtoys` completely.

## PowerShell traps in FMA scripts

Each of these silently produced a wrong answer in practice, not an error.

- **`Start-Process -PassThru` + `WaitForExit($ms)` leaves `ExitCode` empty**, even once `HasExited` is `$true` and even after a parameterless `WaitForExit()`. Use `-Wait -PassThru` when you can. Treat any blank exit code as "not measured" — this one falsely reported an entire 30-app validation run as failing.
- **`'\b/S\b'` never matches a switch preceded by a space.** Neither the space nor the `/` is a word character, so there is no boundary between them, and a `-notmatch` guard appends a duplicate switch forever. Anchor on whitespace: `'(?i)(^|\s)/S($|\s)'`.
- **A single result is a scalar, so `.Count` is `$null`.** Wrap in `@(...)` before counting or comparing, or a one-match check silently reads as zero.
- **`New-ScheduledTaskAction -Argument ""` is rejected outright.** Omit the parameter when there are no arguments.
- **Do not wait for a scheduled task to be observed `Running`** — a fast task enters and leaves that state between polls. Poll until `LastTaskResult -ne 267009` (`SCHED_S_TASK_RUNNING`) instead.
- **A nested `"` inside a `$(...)` subexpression terminates the surrounding string.** Build the value into a variable first; a parse error means the script never ran at all, which is easy to mistake for a silent failure. Parse-check before trusting a run: `[System.Management.Automation.Language.Parser]::ParseFile($p,[ref]$null,[ref]$errors)`.
- **NSIS uninstallers relaunch themselves from `%TEMP%`** and the process you waited on exits while removal is still in flight. Poll until *either* the install directory or the registration disappears, then force-remove any remainder — waiting only on the directory burns the full timeout for an app that clears its registration first (Notion: 120s vs 6s).
- **Carry over every element of a multi-value `ArgumentList`.** The original `ArgumentList = "/silent", "/skip-app-launch"` is an array; taking only the first element dropped `/skip-app-launch` and would have let Spotify launch itself after installing.

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

**5. Scope / SYSTEM context.** See [Per-user installers and the SYSTEM context](#per-user-installers-and-the-system-context) — this is the single most common way a Windows FMA ships broken, and it has its own section.

**6. Multi-version / sibling products sharing a DisplayName.**
- Corretto 21 and 25 both register as `Amazon Corretto (x64)` — pin each with `exists_query ... AND version LIKE '<major>.%'`.
- IntelliJ Ultimate's DisplayName `IntelliJ IDEA <ver>` also matches Community's `IntelliJ IDEA Community Edition <ver>` — exclude siblings in `exists_query` (`AND name NOT LIKE 'IntelliJ IDEA Community%'`) or use a custom `fuzzy_match_name` pattern.

**7. Non-pinned installer URLs.** Some manifests point at a "latest" redirect (e.g. `link.gotomeeting.com/latest-msi`, `download.scdn.co/SpotifyFullSetupX64.exe`). The pinned SHA drifts as soon as the vendor ships, and validation fails with `SHA256 hash in manifest does not match installer file hash`. Spotify served the pinned build one day and a newer one the next, so chasing the hash is not viable.

`ignore_hash: true` sets the output SHA to `no_check` and the validator skips the hash comparison (the route already taken for `google-chrome`, `teamviewer` and ~15 others). **It does not fix the version assertion**: the validator then compares the manifest version against what osquery reports and has no general drift tolerance — only hardcoded per-app exemptions in `appExists` (Chrome for auto-update, Office for Click-to-Run). So a rolling URL whose build runs ahead of winget still fails, one step later. The options are a matching exemption (weakens that app to an existence-only version check — get maintainer agreement, it is shared tooling) or not shipping the app until the vendor offers a versioned URL. Diagnose which case you are in by reading the served installer's real version before touching anything:

```bash
# PE ProductVersion of the binary the URL actually serves right now
python3 - <<'EOF'
import re; d=open('installer.exe','rb').read()
k='ProductVersion'.encode('utf-16-le'); m=re.search(re.escape(k), d)
t=d[m.end():m.end()+120]; i=0
while t[i:i+2]==b'\x00\x00': i+=2
print(t[i:].split(b'\x00\x00')[0].decode('utf-16-le','ignore'))
EOF
```

**8. `frozen: true` outputs are never regenerated.** `cmd/maintained-apps/main.go` only writes the output when `!app.Frozen || !outFileExists`, so re-running the generator on a frozen app silently leaves the old file — including stale `install_script_ref`/`uninstall_script_ref`. If you edit a frozen app's scripts you must hand-edit `outputs/<slug>/<platform>.json`: replace the `refs` map contents and set each `*_script_ref` to the new `sha256[:8]` of the script text. Check for `"frozen": true` in the input before assuming a regeneration took.

**9. NUL-padded registry values.** Some installers write `REG_SZ` values padded with NULs — Fork records `DisplayName`, `Publisher` *and* `DisplayVersion` as `"Fork\0\0\0…"`. A pattern built from the observed string then contains the padding (`^Fork               $`) and never matches, and an exact publisher comparison fails too. Strip NULs on both sides when matching in a script:
```powershell
$name = ($key.DisplayName -replace "`0", "").Trim()
```
It also means the shipped `exists` query for such an app deserves a second look.

**10. MSI "maintenance form" UninstallString.** Some MSIs record `MsiExec.exe /I{ProductCode}` (the *maintenance* form) rather than `/X`. A generic helper that prepends `/X` produces `MsiExec.exe /X /I{GUID}`, which hangs for ~11 minutes and fails (Foxit). Resolve the ProductCode — the ARP key name if it is a GUID, else a `\{[0-9A-Fa-f-]+\}` match on the string — and run a clean `msiexec /x {GUID} /qn /norestart`. Never reuse the `/I` from the registry.

## Pre-ship checklist
- [ ] Identity fields verified against the real installer (MSI Property table / Info.plist), not guessed.
- [ ] `unique_identifier` = registry DisplayName / bundle id; `program_publisher` set if needed.
- [ ] Silent install/uninstall flags from winget `InstallerSwitches` or silentinstallhq, not invented.
- [ ] Custom uninstall (non-MSI-machine) uses the defensive UninstallString parser.
- [ ] Custom script comments are admin-facing and short (~4 lines past the template header): no validator/CI/ingester references, no catalog archaeology, no debugging narrative, no first person. Internal rationale moved to the PR body.
- [ ] Version reconciles with osquery (or a documented validator exception applies — not a blanket skip).
- [ ] Generated SHA matches the manifest; exists/patched queries reviewed; `apps.json` valid + description filled.
- [ ] Icon exists or is generated.
- [ ] Bootstrapper / per-user / latest-URL risks flagged in the PR if present.
- [ ] **Scope proven, not assumed**: installed on a host as SYSTEM and confirmed where the payload and registration actually landed. Nothing under `S-1-5-18`/`.DEFAULT`.
- [ ] **Per-user apps**: install runs as the signed-in user; uninstall enumerates every `HKEY_USERS` hive, runs the uninstaller as the owning user, and has the `system32` → `SysWOW64` fallback.
- [ ] `fuzzy_match_name` matches the DisplayName you actually observed — set for versioned names, absent for plain ones.
- [ ] If the app is `frozen`, the output was hand-edited (refs + `sha256[:8]`) because the generator skipped it.
- [ ] Rolling "latest" URL: `ignore_hash` added if needed, and the version-assertion consequence understood and stated in the PR.
- [ ] Exit codes in any validation you report were actually measured (no blank `ExitCode` from `-PassThru` + `WaitForExit($ms)`).
- [ ] If you changed shared code (`cmd/maintained-apps/validate/*.go`, ingesters), call it out in the PR and run `GOOS=windows go build ./cmd/maintained-apps/validate/` + `go test ./cmd/maintained-apps/...`. Adding a per-app version-check exemption weakens that app to existence-only — get maintainer agreement first.
