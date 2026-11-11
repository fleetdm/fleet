# Verifying installer identity and behavior

Read this before you write any identity field (`unique_identifier`, `program_publisher`, `fuzzy_match_name`, `exists_query`), a silent switch, or `installer_format`. Every recipe here runs on a macOS dev box with no Windows host.

- [Winget manifests](#winget-manifests)
- [MSI](#msi)
- [EXE installers](#exe-installers): NSIS/electron-builder, Inno Setup, WiX Burn, InstallShield
- [macOS installers](#macos-installers)
- [Silent switches](#silent-switches)
- [Partial-download probes](#partial-download-probes)

Download each installer into its own new, empty directory, check its size first (`curl -sIL <url> | grep -i content-length`), and delete it when you're done. Don't download a multi-GB installer just to read a name; the range-request recipes below are enough.

## Winget manifests

```bash
# Versions sort alphabetically ("21.0.11" before "21.0.9"), so sort -V to find the latest.
gh api 'repos/microsoft/winget-pkgs/contents/manifests/<x>/<Pub>/<Pkg>' --jq '.[].name' | sort -V | tail
# Installer manifest: InstallerType, Scope, arch, URL, SHA, ProductCode, AppsAndFeaturesEntries, InstallerSwitches
gh api 'repos/microsoft/winget-pkgs/contents/manifests/<x>/<Pub>/<Pkg>/<ver>/<Pub>.<Pkg>.installer.yaml' --jq .content | base64 -d
# Locale manifest: Publisher, PackageName, ShortDescription
gh api 'repos/microsoft/winget-pkgs/contents/manifests/<x>/<Pub>/<Pkg>/<ver>/<Pub>.<Pkg>.locale.en-US.yaml' --jq .content | base64 -d
```

- GitHub code search returns nothing for `microsoft/winget-pkgs`, even for packages that exist. Browse with the contents API as above, and check your lookup against a package you know exists before you conclude something is missing.
- The **locale** `Publisher` is what the generator puts in the exists query unless you set `program_publisher`. It's often a person's or a parent company's name, or an old company name (Clockify: `COING Inc.` in the locale, `CAKE.com Inc.` in the registry).
- `AppsAndFeaturesEntries` (`DisplayName`, `Publisher`, `DisplayVersion`) in the installer manifest is the best offline source for an exe's registry identity. It's optional, so it's often absent.
- Fields can change between versions, or in place without a version bump (`Scope`, architectures, `InstallerType`). When the ingest job panics with `failed to find installer`, diff the latest and previous installer manifests ([validator-failures.md](validator-failures.md#winget-manifest-changed-underneath-the-input)).

## MSI

The Property table is authoritative for MSI identity.

```bash
msiinfo export app.msi Property | grep -iE "ProductName|ARPDISPLAY|Manufacturer|ProductVersion|UpgradeCode|ProductCode|ALLUSERS|ARPSYSTEMCOMPONENT"
msiinfo export app.msi Registry   # custom ARP writes, if any
msiinfo export app.msi Shortcut   # column 5 names the GUI exe, e.g. [#_7zFM.exe] (for the open query)
```

- `ProductName` → registry DisplayName → `unique_identifier`, unless `ARPDISPLAYNAME` overrides it.
- `Manufacturer` → registry Publisher → `program_publisher` if it differs from the winget locale Publisher.
- `ProductVersion` → the expected `programs.version`, except for a bootstrapper (below).
- `UpgradeCode` → used by the generated upgrade-code uninstall.
- `ALLUSERS=1` → installs per-machine regardless of switches.

**`ARPSYSTEMCOMPONENT=1` usually means a bootstrapper.** Other red flags are a "Setup"-style filename and properties such as `G2MACTION` or `…CLIENT=Setup`. These MSIs install a separate app that registers its own ARP entry, with a different version and uninstaller. The MSI's `ProductVersion`, `ProductCode`, and `UpgradeCode` then don't match the registry, the upgrade-code uninstall fails, and the real app is often per-user. GoToMeeting registers `10.19.0.19950` from an MSI that says `10.19.19950`. Treat bootstrappers as poor FMA candidates. If you must ship one, use a registry-lookup uninstall and flag it as unverifiable.

(Inside a Burn bundle, `ARPSYSTEMCOMPONENT=1` on the inner MSI is normal and not a bootstrapper; see [WiX Burn](#wix-burn-bundle).)

## EXE installers

For an exe, the registry DisplayName feeds both the exists query and the uninstall script's registry lookup, so a wrong guess fails silently in two places. Use these sources, best first:

1. The installer manifest's `AppsAndFeaturesEntries`.
2. The installer itself, read per framework, below. **Confirm the framework first**, because winget's `InstallerType` can be wrong (darktable was declared `nullsoft` and is Inno):
   ```bash
   strings -n 8 setup.exe | grep -iE 'Inno Setup|Nullsoft|NSIS|WixBundle|\.wixburn|InstallShield' | sort -u | head
   ```
3. A real install on a Windows host, or the validator log's `Found app: '<DisplayName>' Version: …` line from a CI run.

If none of these confirms the DisplayName, don't derive it from `PackageName`. Ship nothing, or ship with the gap called out.

### NSIS and electron-builder

- electron-builder registers DisplayName as `<productName> <version>` (`Signal 8.24.1`, `Bdash 1.35.1`), which needs `fuzzy_match_name: true`. Hand-written NSIS scripts vary.
- Publisher is usually the code-signing certificate's CN: `strings -n 8 setup.exe | grep -i '<vendor>'`. Podman Desktop signs as `Red Hat, Inc.` while winget says `RedHat`.
- `7zz l setup.exe` lists the payload, which gives you the GUI exe name for the `open` query. electron-builder nests the app in `$PLUGINSDIR/app-64.7z`; extract and list that.
- Machine scope is `/S /allusers` when the build allows per-machine installs. Verify it on a SYSTEM host: Signal accepts `/allusers` and ignores it.
- Most electron-builder stubs are 32-bit. That matters for per-user installs under SYSTEM ([windows-scripts.md](windows-scripts.md#the-wow64-trap)); `scripts/pe_info.py` tells you which you have.

### Inno Setup

- DisplayName is `UninstallDisplayName` if set, else `AppVerName` (versioned, so it needs `fuzzy_match_name`), else `AppName`. Publisher is `AppPublisher`.
- For open-source apps the `.iss` in the repo at the release tag is authoritative: `gh api 'repos/<org>/<repo>/contents/<path>.iss?ref=<tag>' --jq .content | base64 -d`.
- `innoextract --info setup.exe` prints the app name for older Inno versions. innoextract 1.9 fails on setup data 6.3 and later ("Unexpected setup data version"), and `7zz` can't open Inno at all. Don't burn time hand-decompressing a 6.3+ header.
- Check the `.iss` `[Run]` section for `postinstall` entries without `skipifsilent`. They run even under `/VERYSILENT`, and the install then never exits ([windows-scripts.md](windows-scripts.md#install-scripts)).
- Omit `/ALLUSERS` if the `.iss` has `PrivilegesRequiredOverridesAllowed=dialog` rather than `commandline`. The default `PrivilegesRequired=admin` already installs for all users.

### WiX Burn bundle

The ARP identity is the `<Arp DisplayName=… Publisher=…>` element in the bundle's `BurnManifest.xml`. It sits in the UX cabinet near the start of the file, so a 30 MB range request is enough even for a 1 GB installer:

```bash
curl -sL -r 0-31457279 -o head.bin "<InstallerUrl>"
python3 .claude/skills/new-fma/scripts/burn_ux.py head.bin ux.cab
7zz x -y -oux ux.cab >/dev/null && grep -o '<Arp [^>]*>' ux/0
```

- The bundle Publisher and the inner MSI's Manufacturer can differ: Monotype's bundle Publisher is a copyright line. osquery's `programs` table returns **both** the bundle row and the hidden inner-MSI row (`SystemComponent=1` isn't filtered), so if the publishers differ, use `exists_query` with `publisher LIKE '%<Vendor>%'` instead of `program_publisher`.
- The uninstall script must pick the bundle's key, not the MSI's ([windows-scripts.md](windows-scripts.md#uninstall-scripts)).
- Zip-wrapped bundles (winget `InstallerType: zip`, `NestedInstallerType: burn`) work with `installer_type: "zip"` and an `Expand-Archive` install script (`agent-ransack_install.ps1`).

### InstallShield

An InstallShield InstallScript MSI needs `/S /v/qn`. `/S` silences only the wrapper, and `/v/qn` passes `/qn` to the inner msiexec. winget's declared `/Silent` exits `0x80042000`. To confirm the project type, `7zz x` the setup and read the engine string table. Some vendor CDNs return 403 without a browser User-Agent.

## macOS installers

```bash
curl -sL -o app.dmg "<cask url>" && file app.dmg     # "Apple Disk Image" / "xar archive" (pkg) / "Zip archive"
MP=$(mktemp -d); hdiutil attach -nobrowse -readonly -mountpoint "$MP" app.dmg >/dev/null
APP=$(find "$MP" -maxdepth 1 -name "*.app" | head -1)
for k in CFBundleIdentifier CFBundleShortVersionString CFBundleVersion CFBundleExecutable; do
  printf '%s=' "$k"; /usr/libexec/PlistBuddy -c "Print :$k" "$APP/Contents/Info.plist"
done
hdiutil detach "$MP" >/dev/null
```

- **pkg**: `pkgutil --expand-full app.pkg out` needs no sudo and installs nothing. Read `out/*.pkg/Payload/Applications/*.app/Contents/Info.plist`, or the `<bundle path="./Applications/…app"` lines in each component's `PackageInfo`. Read the payload's version, not just the cask's: vendors ship builds that differ from the cask in both directions (Acrobat lags, ocenaudio leads).
- **zip**: `unzip -l` first. A zip that contains a dmg needs a custom install script that unzips, mounts, and copies (`inputs/homebrew/scripts/pd-install.sh`). Discover the dmg and app names at run time so the script survives version bumps.
- **DMGs with a license agreement** need `yes | hdiutil attach …`; a plain attach prints the EULA and cancels (`cycling74-max-install.sh`).
- **`suite` casks** put a folder in `/Applications`. osquery's `apps` table doesn't recurse into it, so a nested `.app` only shows up after `lsregister -f <app>` (`openvpn-connect-install.sh` and `box-tools-install.sh` do this; KiCad, in fleetdm/fleet#54613, is the full suite example).
- The cask's `uninstall quit`, `pkgutil`, `launchctl`, and `zap` entries (`<bundleid>.savedState`) are hints for the bundle id, not proof. They can be stale.
- **Compare `CFBundleShortVersionString` and `CFBundleVersion` to the cask `version`** and note which one tracks it: that decides whether the patched query needs a per-token override ([queries.md](queries.md#when-the-version-column-doesnt-track-the-manifest)).

## Silent switches

Sources, in order: the winget manifest's `InstallerSwitches` (`Silent`, `SilentWithProgress`, `Custom`); then the framework's standard switches (table in [SKILL.md](../SKILL.md#workflow-windows)); then the vendor's docs, or silentinstallhq.com (`https://silentinstallhq.com/<app>-silent-install-how-to-guide/`). Never invent one.

When an install hangs or fails, get the installer's own diagnostics before you guess another switch. Inno takes `/LOG=<file>` and states its abort reason; a Burn bundle takes `/log <file>` and reports blockers such as `Variable: RebootPending = 1`; msiexec takes `/lv <file>`. Four guessed switches taught nothing about `jetbrains-toolbox`, while one log line explained `antigravity-ide` and `devtoys`.

## Partial-download probes

`scripts/pe_info.py` prints a Windows installer's CPU architecture (from the PE header) and its VERSIONINFO `ProductVersion`, `FileVersion`, `ProductName`, and `CompanyName`. The first few MB of the file are usually enough:

```bash
curl -sL -r 0-4194303 -o head.bin "<InstallerUrl>"
python3 .claude/skills/new-fma/scripts/pe_info.py head.bin
```

Use it to check whether a version-less "latest" URL serves the build the manifest claims ([validator-failures.md](validator-failures.md#rolling-latest-urls)), and whether a stub is 32-bit.
