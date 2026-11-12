# Verifying installer identity and behavior

Read this before you write any identity field (`unique_identifier`, `program_publisher`, `fuzzy_match_name`, `exists_query`, `open_process_names`), a silent switch, or `installer_format`. Every recipe here runs on a Mac with no Windows host. Follow [Handling installers safely](../SKILL.md#handling-installers-safely): a fresh directory per download, size check first, hash check, nothing executed.

- [What has to match](#what-has-to-match)
- [Winget manifests](#winget-manifests)
- [MSI](#msi)
- [EXE installers](#exe-installers): NSIS/electron-builder, Inno Setup, WiX Burn, InstallShield, Squirrel, MSIX, web bootstrappers
- [macOS installers](#macos-installers)
- [Silent switches](#silent-switches)
- [Partial-download probes](#partial-download-probes)

## What has to match

| Field | Must equal | Read it from |
|---|---|---|
| Windows `unique_identifier` | the registry DisplayName (osquery `programs.name`) | MSI `ProductName`/`ARPDISPLAYNAME`; Burn `<Arp DisplayName>`; NSIS/Inno header; a real install |
| Windows publisher | the registry Publisher (`programs.publisher`) | MSI `Manufacturer`; Burn `<Arp Publisher>`; NSIS `WriteRegStr … Publisher`; Inno `AppPublisher` |
| Windows version | `programs.version` (DisplayVersion) | MSI `ProductVersion`; Inno `AppVersion`; NSIS header; `AppsAndFeaturesEntries` |
| macOS `unique_identifier` | the top-level app's `CFBundleIdentifier` | `Info.plist` in the dmg/zip/pkg payload |
| macOS version | `bundle_short_version` (or `bundle_version` with an override) | `CFBundleShortVersionString`, `CFBundleVersion` |
| Scope | where the app really lands under SYSTEM/root | a real install, the MSI's `ALLUSERS` and custom actions, the NSIS/Inno install-mode logic |

Real cases where the catalog lied:

| App | winget/cask says | Installed identity is |
|-----|------------------|------------------------|
| Amazon Corretto | PackageName "Amazon Corretto 25" | DisplayName `Amazon Corretto (x64)` (no version), Publisher `Amazon` |
| ActivePresenter | locale Publisher `Atomi Systems` | Publisher `Atomi Systems, Inc.` |
| Chatbox | locale Publisher `Benn Huang` | DisplayName `Chatbox <ver>`, Publisher `Mediocre Company` (`Benn Huang` is the old Community Edition) |
| darktable | InstallerType `nullsoft`, Publisher "the darktable project" | Inno Setup, Publisher `darktable team`, DisplayName `darktable 5.6.0` |
| GoToMeeting | MSI ProductName `GoToMeeting 10.19.19950` | DisplayName `GoToMeeting 10.19.0.19950` (bootstrapper) |
| KNIME | (fuzzy name assumed) | exactly `KNIME Analytics Platform` in every release |
| Ollama | PackageName `Ollama` | `Ollama version 0.40.13` (Inno default) |
| PostgreSQL | PackageName `PostgreSQL 18` | `PostgreSQL 18 ` (trailing space) |
| REAPER | `AppsAndFeaturesEntries` `REAPER` | the x64 build registers `REAPER (x64)` |
| Tableau | locale Publisher `Tableau Software, LLC` | Publisher `Salesforce, Inc` (Burn bundle) |
| ForkLift, GIMP | cask `quit`/`zap` ids `…ForkLift-4`, `org.gimp.gimp-3.0` | `com.binarynights.ForkLift`, `org.gimp.gimp` |
| Tableau (macOS) | pkg receipt `com.tableausoftware.Desktop.app` | `com.tableausoftware.tableaudesktop` |
| KiCad | cask zap `org.kicad-pcb.*` | `org.kicad.kicad` |
| GitHub Desktop | winget machine-scope MSI | only installs a "Deployment Tool"; the real app installs per user at next sign-in |

If you can't confirm a field from one of the sources below, don't derive it from `PackageName`. Ship nothing, or ship with the gap called out in the PR.

## Winget manifests

```bash
.claude/skills/fleet-maintained-apps/scripts/winget_manifest.sh <PackageIdentifier> [version] [--diff]
```

- GitHub code search returns nothing for `microsoft/winget-pkgs`, even for packages that exist, so a zero result proves nothing. The script reads the contents API. To find a publisher folder, list `manifests/<letter>` and grep; its casing isn't the brand's (`manifests/n/Nvidia`).
- The **locale** `Publisher` is what the generator puts in the exists query unless you set `program_publisher`, and it's often a person's name, a parent company, an old name, or truncated.
- `AppsAndFeaturesEntries` (`DisplayName`, `Publisher`, `DisplayVersion`, `UpgradeCode`) is the best offline source for an exe's registry identity, but most manifests don't have it. For a Burn bundle it may describe the inner MSI rather than the bundle (Okta Verify).
- Some packages have no `en-US` locale (Araxis Merge: `en-GB`). `winget_manifest.sh` falls back to the version manifest's `DefaultLocale`, but the ingester reads only `<id>.locale.en-US.yaml` and fails ingest without it, so such a package can't ship without an ingester change.
- `PackageFamilyName` for MSIX can sit at the top level, where the ingester doesn't read it; then `$PACKAGE_ID` stays literal and the uninstall script has to hardcode the PFN.
- Manifests change in place and get deleted; see [validator-failures.md](validator-failures.md#winget-manifest-changed-underneath-the-input).

## MSI

The Property table is authoritative for MSI identity.

```bash
msiinfo export app.msi Property | grep -iE "ProductName|ARPDISPLAY|Manufacturer|ProductVersion|UpgradeCode|ProductCode|ALLUSERS|MSIINSTALLPERUSER|ARPSYSTEMCOMPONENT"
msiinfo export app.msi Shortcut          # column 5 names the GUI exe, e.g. [#_7zFM.exe] (open query)
msiinfo export app.msi File              # every exe the MSI installs
msiinfo export app.msi InstallExecuteSequence; msiinfo export app.msi CustomAction   # scope and launch behavior
msiinfo export app.msi Registry          # custom ARP writes, if any
```

- `ProductName` → registry DisplayName → `unique_identifier`, unless `ARPDISPLAYNAME` overrides it.
- `Manufacturer` → registry Publisher → `program_publisher` if it differs from the winget locale Publisher.
- `ProductVersion` → `programs.version`, except for a bootstrapper. A WiX 4-part version against a 3-part winget version with the same prefix (`4.0.0.262451524` vs `4.0.0`) is fine.
- `UpgradeCode` → the generated upgrade-code uninstall, and the key Fleet uses to link a Windows FMA to its software title. Two products that share one (Acrobat Reader and Pro) land on one title.
- `ALLUSERS=1` installs per-machine regardless of switches. **No `ALLUSERS`, or `ALLUSERS=2` with `MSIINSTALLPERUSER=1`, installs per-user under SYSTEM** (into `systemprofile`, registered under `S-1-5-18`). Pass `ALLUSERS=1 MSIINSTALLPERUSER=""` in a custom install script (Audacity 4, Tower).
- **Custom actions can override the switches.** A WixUI_Advanced MSI sets `ALLUSERS=1` only when `UILevel` isn't 2, so `/quiet` goes per-user (Wrike). Velopack MSIs pick a per-user folder in quiet mode (Tower). A vendor action can read HKCU and clear `ALLUSERS` again when it finds a previous per-user install. Read `InstallExecuteSequence` and `CustomAction` for conditions on `UILevel` and for actions that read the registry. To read a DTF (.NET) custom action: `msiinfo extract app.msi Binary.<Name> > ca.dll`, `7zz x ca.dll`, then `monodis` the managed DLL.
- **A custom action can launch the app** (WireGuard's `LaunchApplicationAndAbort` makes a same-version reinstall exit 1602 unless `DO_NOT_LAUNCH=1`), install a per-user MSIX that fails as SYSTEM (eM Client 11's `InstallMsixFromAppInstaller`), install prerequisites only in the UI sequence (Tower's .NET runtime), or create shortcuts with a VBScript that resolves to `C:\Users\Default` under SYSTEM (Clockify). winget's `InstallerSwitches.Custom` (`DO_NOT_LAUNCH=1`) isn't carried into the generated script.

**Bootstrappers and deployment tools.** `ARPSYSTEMCOMPONENT=1` on a standalone MSI, a "Setup"-style filename, or properties such as `G2MACTION` or `…CLIENT=Setup` mean the MSI installs something else that registers its own ARP entry, with a different version and uninstaller (GoToMeeting). Squirrel apps' "machine-wide" MSIs are often deployment tools that install the real app per user at next sign-in (GitHub Desktop); check what lands on a host before trusting one (Crestron AirMedia's really does install machine-wide). Treat these as poor FMA candidates; if you must ship one, use a registry-lookup uninstall and flag it as unverifiable. (Inside a Burn bundle, `ARPSYSTEMCOMPONENT=1` on the inner MSI is normal; see [WiX Burn](#wix-burn-bundle).)

## EXE installers

For an exe, the registry DisplayName feeds both the exists query and the uninstall script's registry lookup, so a wrong guess fails silently in two places. **Confirm the framework first**, because winget's `InstallerType` can be wrong:

```bash
strings -n 8 setup.exe | grep -iE 'Inno Setup|Nullsoft|NSIS|WixBundle|\.wixburn|InstallShield|Squirrel|velopack' | sort -u | head
7zz l setup.exe | head -30     # Type = Nsis / 7z SFX / PE; NSIS lists the payload
```

Sources, best first: the installer's own script (public `.iss`/`.nsi`/installer project at the release tag); the decoded installer header (below); `AppsAndFeaturesEntries`; a real install on a Windows host; the validator log's `Found app:` line.

### NSIS and electron-builder

- electron-builder registers `<productName> <version>` by default (`Signal 8.24.1`), which needs `fuzzy_match_name: true`. Hand-written NSIS varies (`Name "<product> <version>"` makes `$(^Name)` versioned too: Geany).
- Publisher is usually the code-signing certificate's CN (`strings -n 8 setup.exe | grep -i '<vendor>'`): Podman Desktop signs as `Red Hat, Inc.`, winget says `RedHat`.
- `7zz l setup.exe` lists the payload, which gives the GUI exe for the `open` query. electron-builder nests the app in `$PLUGINSDIR/app-64.7z`; extract and list that.
- **Machine scope** is `/S /allusers` only when the build has a per-machine mode. electron-builder with `nsis.oneClick: true` and no `perMachine` ignores `/allusers` silently (Signal, Requestly, Notesnook); those need the per-user pattern. Check the header: if `$installMode` is only ever assigned `CurrentUser`, there's no machine mode. Some vendors use their own switch (MEGAsync: `/MULTIUSER`).
- Most electron-builder stubs are 32-bit, which matters under SYSTEM ([windows-scripts.md](windows-scripts.md#the-wow64-trap)); `scripts/pe_info.py` tells you.

**Decoding the NSIS header offline** (when `7zz l` shows `Type = Nsis` but no `[NSIS].nsi`). The header holds every `WriteRegStr` (so DisplayName, Publisher, DisplayVersion), every MessageBox and its `/SD` default, and `.onInit`:
1. Find `\xef\xbe\xad\xdeNullsoftInst`; the 28-byte firstheader starts 4 bytes earlier (`flags, sig, magic[12], header_len, total_len`).
2. electron-builder headers are non-solid raw deflate: a 4-byte length (high bit = compressed), then `zlib.decompressobj(-15)`. Others are solid LZMA: 5 props bytes, then a raw LZMA1 stream (`lzma.LZMADecompressor(FORMAT_RAW, filters=[{"id": lzma.FILTER_LZMA1, …}])`); the first 4 decompressed bytes are the header length. The header sits in the first few hundred KB, so `curl -r 0-4194303` is enough even for a 400 MB installer.
3. After `flags` come 8 blocks (pages, sections, entries, strings, langtables, …) as `{offset, num}`: 32-bit pairs, or `{int64, int32}` for an x64 exehead. Entries are 7 int32s (`opcode, p0..p5`); strings are UTF-16LE, offsets in WCHARs; a negative string offset is a langstring (`-(idx+1)`, idx 2 = `^Name`).
4. Don't trust opcode numbers across builds; find entries by their string params instead: `WriteRegStr`'s p2 is the value name (`DisplayName`), p3 the value. A MessageBox's `p0 >> 21` is its `/SD` default (0 = none: it shows under `/S` and hangs as SYSTEM). Page pre-callbacks don't run under `/S`.
5. The uninstaller's header follows the `WriteUninstaller` entry's icon item; decode it the same way.

Grep the decoded strings for `nsProcess`/`_FindProcess`/"is currently running" (running-app aborts, exit 2), `IsProcessorFeaturePresent`/`IsWow64Process2` (ARM64/AVX refusals), and `isDirEmpty` (JetBrains' non-empty-directory box).

### Inno Setup

- DisplayName is `UninstallDisplayName` if set, else `AppVerName` (versioned), else `"<AppName> version <AppVersion>"` when `AppVersion` is set (Ollama), else `AppName`. Publisher is `AppPublisher`; DisplayVersion is `AppVersion` (Raspberry Pi Imager's is `v2.0.8`). The `AppId` key is `{…}_is1`.
- For open-source apps the `.iss` at the release tag is authoritative: `gh api 'repos/<org>/<repo>/contents/<path>.iss?ref=<tag>' --jq .content | base64 -d`.
- `innoextract --info setup.exe` reads Inno up to 6.2. It fails on 6.3+ ("Unexpected setup data version", "Stream error"), and `7zz` can't open Inno at all.
- **Decoding the setup header** (any 6.x): find the **last** `Inno Setup Setup Data` match (the first is the embedded uninstaller's), skip the 64-byte ID, then a CRC and a block header (`StoredSize` u32 + `Compressed` u8; 6.7 widens `StoredSize` to u64; 6.5+ inserts a 53-byte encryption header first; find the LZMA props byte `0x5d` to locate the stream). Strip the 4-byte CRC that precedes every 4096-byte chunk, then `lzma.LZMADecompressor(FORMAT_ALONE)` on `props(5) + b'\xff'*8 + rest`. Strings are UTF-16LE with 4-byte length prefixes; the setup header's list starts `AppName, AppVerName, AppId, Copyright, Publisher, PublisherURL, SupportPhone, SupportURL, UpdatesURL, AppVersion`. In 6.x the header often sits near the end of the exe, so the last 6 MB (`curl -r -6291456`, or an explicit range for GitHub release assets) is enough.
- Check `[Run]` for `postinstall` entries (a `Description` means postinstall) without `skipifsilent`: they run even under `/VERYSILENT`, and the install never exits. Vendor-specific switches show up as bare constants near `/silent` (iMazing's `/nolaunch`; pass it lowercase).
- Check `[Code]` for plain `MsgBox` calls (not `SuppressibleMsgBox`): `/SUPPRESSMSGBOXES` doesn't suppress them (KNIME's "remove current installation?", Rtools' "target directory already exists").
- `ArchitecturesAllowed=x64` (`x64os` in 6.3+) refuses ARM64 hosts. Omit `/ALLUSERS` if the `.iss` has `PrivilegesRequiredOverridesAllowed=dialog` rather than `commandline`; the default `PrivilegesRequired=admin` already installs for all users.

### WiX Burn bundle

The ARP identity is the `<Arp DisplayName=… DisplayVersion=… Publisher=…>` element in the bundle's `BurnManifest.xml`. It's in the UX cabinet near the start of the file, so a 30 MB range request is enough even for a 1 GB installer:

```bash
curl -sL -r 0-31457279 -o head.bin "<InstallerUrl>"
python3 -I .claude/skills/fleet-maintained-apps/scripts/burn_ux.py head.bin ux.cab
7zz x -y -oux ux.cab >/dev/null && grep -o '<Arp [^>]*>' ux/0 && grep -o '<Registration [^>]*>' ux/0
```

- osquery's `programs` returns **both** the bundle row and the hidden inner-MSI row (`SystemComponent=1` isn't filtered). If the bundle Publisher differs from the MSI Manufacturer (Monotype's bundle Publisher is a copyright line), use `exists_query` with `publisher LIKE '%<Vendor>%'`.
- The uninstall script must pick the bundle's key, not the MSI's ([windows-scripts.md](windows-scripts.md#uninstall-scripts)).
- A bundle with a custom bootstrapper application can reject switches standard Burn accepts (Okta Verify 7's WiX 6 BA fails on a bare `/norestart`). To read a .NET BA's logic, `monodis` it with its dependency DLLs from the UX cab on `MONO_PATH`.
- The GUI exe name (for `open`) needs the inner MSI: carve the attached cabinet at the `MSCF` magic after the UX container, `7zz x` it, then `msiinfo export <msi> Shortcut`.
- Zip-wrapped bundles (winget `InstallerType: zip`, `NestedInstallerType: burn`) work with `installer_type: "zip"` and an `Expand-Archive` install script (`agent-ransack_install.ps1`).

### InstallShield

An InstallShield InstallScript MSI needs `/S /v/qn`: `/S` silences only the wrapper, and `/v/qn` passes `/qn` to the inner msiexec. winget's declared `/Silent` exits `0x80042000`. A 32-bit setup also needs `/b"<dir under [IO.Path]::GetTempPath()>"` before `/v`, or it caches the MSI in WOW64-redirected `systemprofile` and fails with 1619 (Sonos). Confirm the project type by `7zz x`-ing the setup and reading the engine string table. Some vendor CDNs return 403 without a browser User-Agent.

### Squirrel, Velopack, MSIX, and web bootstrappers

- **Squirrel.Windows** `Setup.exe` (`--silent`) installs per user into `%LOCALAPPDATA%\<App>` with an HKU uninstall key; it has no machine mode. Its error dialog shows even with `--silent`.
- **Velopack** MSIs default to per-user in quiet mode; see [MSI](#msi).
- **MSIX** is a zip: `unzip -p app.msix AppxManifest.xml` gives `Identity Name`, `PublisherDisplayName`, `DisplayName`, and `Executable`. osquery's `programs` lists MSIX rows from loaded user hives and the all-user store, with `install_location` under `WindowsApps\<PackageFullName>`. Installers are often 100 MB+.
- **Web bootstrappers** (JetBrains `.web.exe`, Adobe stubs) download the product at install time, so the pinned SHA covers only the stub and the identity often isn't in it. Look for an offline installer from the vendor's product API, or take the identity from a real install or the validator log.

## macOS installers

```bash
.claude/skills/fleet-maintained-apps/scripts/inspect_mac_installer.sh "<cask url>" --sha256 <cask sha256> [--keep]
```

It prints the real format, every top-level app's `CFBundleIdentifier`, `CFBundleShortVersionString`, `CFBundleVersion`, `CFBundleExecutable`, Team ID, and Gatekeeper verdict, folders containing apps (suites), nested apps, and the lines in pkg scripts that kill, quit, relaunch, or remove the app. What to do with what it finds:

- **The id is the top-level app's.** The cask's `uninstall quit`, `pkgutil`, `launchctl`, and `zap` entries (`<bundleid>.savedState`, prefs-plist names) are hints, not proof, and they're often stale or wrong. A pkg receipt id isn't a bundle id. A nested XPC service or helper isn't the app.
- **Versions**: note which of `CFBundleShortVersionString` and `CFBundleVersion` tracks the cask `version` ([queries.md](queries.md#when-the-version-column-doesnt-track-the-manifest)). Read the payload's version, not the cask's: vendors ship builds that differ in both directions (Acrobat lags, ocenaudio leads, MeetingBar's v5.0.0 tag builds 5.0.1).
- **pkg**: `pkgutil --expand-full app.pkg out` needs no sudo and runs nothing. The `<bundle path="./Applications/…app" CFBundleShortVersionString=… id=…>` lines in `PackageInfo` give identity without reading the payload. Read `preinstall`/`postinstall`: they decide whether your script must quit the app, whether the vendor relaunches it, and whether the postinstall modifies the signed bundle (which Gatekeeper then rejects: Topaz, Omnissa, WiFiman).
- **zip**: AppleDouble (`._*`, `__MACOSX/`) entries mean only `ditto -x -k` keeps the signature intact. A zip containing a dmg or pkg needs a custom install script.
- **dmg**: a license agreement needs `yes | hdiutil attach …`; a plain attach prints the EULA and cancels.
- **A top-level "app" that is an installer** (RoboForm 10's `RoboForm.app` is `RoboFormInstaller` carrying the old id; the real app is a zip inside it) isn't installable by the template.
- **Gatekeeper must accept the app.** "Unnotarized" on a brand-new release can be a ticket that isn't published yet; "Apple Development" or "Software Signing" without notarization won't pass.
- **`suite` casks** put a folder in `/Applications` that osquery's `apps` table doesn't recurse into ([macos-scripts.md](macos-scripts.md#suite-casks-and-nested-apps)).

## Silent switches

Sources, in order: the winget manifest's `InstallerSwitches` (`Silent`, `SilentWithProgress`, `Custom`); then the framework's standard switches (table in [SKILL.md](../SKILL.md#workflow-windows)); then the vendor's docs, the installer's own decoded header or string table, or silentinstallhq.com / Chocolatey package scripts. Never invent one.

When an install hangs or fails, get the installer's own diagnostics before you guess another switch. Inno takes `/LOG=<file>` and states its abort reason; a Burn bundle takes `/log <file>` and reports blockers such as `Variable: RebootPending = 1`; msiexec takes `/lv <file>`; InstallShield takes `/debuglog`. Four guessed switches taught nothing about one hang, while one log line explained the next two.

## Partial-download probes

Most identity lives in a small part of the file, so read that part instead of downloading the whole installer. Resolve redirects first (`curl -sIL -o /dev/null -w '%{url_effective}' <url>`) and check `accept-ranges: bytes`. GitHub release assets reject suffix ranges (`-r -N`); request `0-0` to learn the size, then an explicit range.

| Need | Range |
|---|---|
| A Windows exe's architecture and VERSIONINFO | first 4 MB → `scripts/pe_info.py head.bin` |
| A Burn bundle's ARP identity | first 30 MB → `scripts/burn_ux.py` |
| An NSIS header | first 4 MB |
| An Inno 6.x header | last 6 MB |
| A pkg's scripts and `PackageInfo` | the 28-byte xar header (`>4sHHQQI`), then the zlib TOC, then each file at `header_size + toc_len + offset`. `Scripts` is a gzip'd cpio (`tar -xzf`); `PackageInfo`/`Distribution` are zlib |
| One member of a zip | the last 64 KB (end of central directory), the central directory, then the member's local header and data (`zlib.decompress(d, -15)`) |

Use `pe_info.py` to check whether a version-less "latest" URL serves the build the manifest claims ([validator-failures.md](validator-failures.md#rolling-latest-urls)), and whether a stub is 32-bit. A UDIF DMG can't be read this way; download it (it's usually small) or skip it.
