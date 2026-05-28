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
- **Icon**: check `frontend/pages/SoftwarePage/components/icons/index.ts` for a key matching the lowercased catalog `name`. If missing, generate via [tools/software/icons](../../../tools/software/icons) before merge. Icons key off the lowercased `name`, so platforms sharing a `name` share an icon.
- The validator is a Windows/macOS host (often **ephemeral** — you can't query it after the run). To cross-compile the Windows validator after editing it: `GOOS=windows go build ./cmd/maintained-apps/validate/`.

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

**5. Scope / SYSTEM context.** Fleet runs installs as SYSTEM (elevated). A per-user installer lands in the SYSTEM profile (useless). Force machine-wide with the installer's all-users switch (`ALLUSERS=1`/`2`, `/ALLUSERS`, `G2MINSTALLFORALLUSERS=1`). A per-user uninstaller likewise can't be reached from a SYSTEM-context script (its ARP entry is in the logged-in user's HKCU).

**6. Multi-version / sibling products sharing a DisplayName.**
- Corretto 21 and 25 both register as `Amazon Corretto (x64)` — pin each with `exists_query ... AND version LIKE '<major>.%'`.
- IntelliJ Ultimate's DisplayName `IntelliJ IDEA <ver>` also matches Community's `IntelliJ IDEA Community Edition <ver>` — exclude siblings in `exists_query` (`AND name NOT LIKE 'IntelliJ IDEA Community%'`) or use a custom `fuzzy_match_name` pattern.

**7. Non-pinned installer URLs.** Some manifests point at a "latest" redirect (e.g. `link.gotomeeting.com/latest-msi`). The pinned SHA drifts when the vendor ships a new build, breaking Fleet installs until the FMA auto-update bumps it. Note this in the PR.

## Pre-ship checklist
- [ ] Identity fields verified against the real installer (MSI Property table / Info.plist), not guessed.
- [ ] `unique_identifier` = registry DisplayName / bundle id; `program_publisher` set if needed.
- [ ] Silent install/uninstall flags from winget `InstallerSwitches` or silentinstallhq, not invented.
- [ ] Custom uninstall (non-MSI-machine) uses the defensive UninstallString parser.
- [ ] Version reconciles with osquery (or a documented validator exception applies — not a blanket skip).
- [ ] Generated SHA matches the manifest; exists/patched queries reviewed; `apps.json` valid + description filled.
- [ ] Icon exists or is generated.
- [ ] Bootstrapper / per-user / latest-URL risks flagged in the PR if present.
- [ ] If you changed shared code (`cmd/maintained-apps/validate/*.go`, ingesters), call it out in the PR and run `GOOS=windows go build ./cmd/maintained-apps/validate/` + `go test ./cmd/maintained-apps/...`.
