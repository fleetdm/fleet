# Fleet-maintained apps (FMA)

## Adding a new app (macOS)

1. Find the app's metadata in its [Homebrew formulae](https://formulae.brew.sh/)
2. Create a new mainfiest file called `$YOUR_APP_NAME.json` in the `inputs/homebrew/` directory. For
   example, if you wanted to add Box Drive, create the file `inputs/homebrew/box-drive.json`. 
3. Fill out the file according to the [input schema below](#input-file-schema). For our example Box Drive app, it would look like this:

   ```json
   {
      "name": "Box Drive",
      "slug": "box-drive/darwin",
      "unique_identifier": "com.box.desktop",
      "token": "box-drive",
      "installer_format": "pkg",
      "default_categories": ["Productivity"]
   }
   ```

4. Run the following command from the root of the Fleet repo to generate the app's output data:

    ```bash
   go run cmd/maintained-apps/main.go --slug="<slug-name>" --debug
   ```

5. The contributor is responsible for adding the icon to Fleet (e.g. the TypeScript and website PNG components of [#29175](https://github.com/fleetdm/fleet/pull/29175/files)). These can be generated using the [generate-icons](https://github.com/fleetdm/fleet/tree/main/tools/software/icons) script.

6. Add a description for the app in `outputs/apps.json` file. You can use descriptions from [Homebrew formulae](https://formulae.brew.sh/).

7. Open a PR to the `fleet` repository with the above changes. The [#g-software Engineering Manager (EM)](https://fleetdm.com/handbook/company/product-groups#software-group) is automatically added reviewer. Also, @ mention the #g-software Product Designer (PD) in a comment that points them to the new icon. This way, the icon change gets a second pair of eyes.

8. If the app passes automated tests, it is approved and merged. The EM reviews the PR within 1 business day. The app should appear shortly in the Fleet-maintained apps section when adding new software to Fleet. The app icon will not appear in Fleet until the following release.

### Input file schema

#### `name` (required)

This is the user-facing name of the application.

#### `unique_identifier` (required)

This is the platform-specific unique identifier for the app. On macOS, this is the app's bundle identifier.

#### `token` (required)

This is the identifier used by homebrew for the app; it is the `token` field on the homebrew API response.

#### `installer_format` (required)

This is the file format for the app's installer. Currently supported values are:

- `zip`
- `dmg`
- `pkg`

To find the app's installer format, you can look at the `url` field on the homebrew API response. The installer's extension should be at the end of this URL.

Sometimes the file type is not included in the installer's URL. In this case, you can download the installer and use the extension of the downloaded file.

#### `slug` (required)

The `slug` identifies a specific app and platform combination. It is used to name the manifest files that contain the metadata that Fleet needs to add, install, and uninstall this app.  This is what is used when referring to the app in GitOps.

The slug is composed of a filesystem-friendly version of the app name, and an operating system platform identifier, separated by a `/`.

For the app name part, use `-` to separate words if necessary, for example `adobe-acrobat-reader`.

Use `darwin` as the platform part.  For example, use a `slug` of `box-drive/darwin` for Box Drive on macOS.

#### `pre_uninstall_scripts` (optional)

These are command lines that will be run _before_ the generated uninstall script is executed, e.g. for [Box](inputs/homebrew/box-drive.json).

#### `post_uninstall_scripts` (optional)

These are command lines that will be run _after_ the generated uninstall script is executed, e.g. for [Box](inputs/homebrew/box-drive.json).

#### `default_categories` (required)

These are the default categories assigned to the installer in [self-service](https://fleetdm.com/guides/software-self-service) if no categories are specified when it is added to a team's library.  Categories must be one or more of these values:

- `Browsers`
- `Communication`
- `Developer Tools`
- `Productivity`

#### `install_script_path` (optional)

This is a filepath to an install script. If provided, this script will be used instead of the generated install script. Only shell scripts (`.sh`) are supported.

The script should be added to the `inputs/homebrew/scripts` directory.

#### `uninstall_script_path` (optional)

This is a filepath to an uninstall script. If provided, this script will be used instead of the generated uninstall script. Only shell scripts (`.sh`) are supported.

`uninstall_script_path` can't be used together with `pre_uninstall_scripts` and `post_uninstall_scripts`.

The script should be added to the `inputs/homebrew/scripts` directory.

## Adding a new app (Windows)

Use the Winget ingester. You will author:
- An input JSON in `ee/maintained-apps/inputs/winget/`
- Optional PowerShell scripts in `ee/maintained-apps/inputs/winget/scripts/`

### Can I do this on macOS?

The instructions below are meant to be run on a Windows host. But, you can run most of this on a macOS host, as well:
- You can author Windows inputs and run the generator on macOS. The ingester is Go code that fetches data from winget/GitHub and works cross‑platform.
- To find a Winget PackageIdentifier without a Windows host, browse the winget-pkgs repo: https://github.com/microsoft/winget-pkgs (search for your app’s manifests).
- To find the PackageName and Publisher, you can look in the locale and installer yaml files in the winget-pkgs repo.
- Validation and testing still require a Windows host (to verify programs.name and to run install/uninstall).

### Step 1: Find the Winget PackageIdentifier
- On a Windows host, run: `winget search <app name>`
- Note the `PackageIdentifier`. For example, Box Drive is typically `Box.Box`.

### Step 2: Find the unique identifier used by osquery
The Windows ingester expects `unique_identifier` to match the value in `programs.name` on the host after install (this is what Fleet uses to confirm the app exists).
- On a test Windows host, install the app manually, then check either:
  - Fleet live query: `SELECT name, version, publisher FROM programs WHERE name LIKE '%<App Name>%';`
  - PowerShell: `Get-ItemProperty 'HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall\*' | Where-Object {$_.DisplayName -like '*<App Name>*'} | Select-Object DisplayName, DisplayVersion, Publisher`
- Use the exact `DisplayName`/`programs.name` string as `unique_identifier`.

### Step 3: Choose installer metadata
If the winget manifest supports multiple installers, these fields select the right one:
- `installer_arch`: `x64` or `x86` (most apps use `x64`)
- `installer_type`: `exe`, `msi`, or `msix` (file type, not vendor tech like "wix")
- `installer_scope`: `machine` or `user` (prefer `machine` for managed installs)
- Optional: `installer_locale` if a specific locale is required
- Optional: `program_publisher`, `uninstall_type`, `fuzzy_match_name` (rare)
- `default_categories`: one or more of: `Browsers`, `Communication`, `Developer Tools`, `Productivity`

Tip: Setting these accurately avoids ambiguity when multiple installers exist.

### Step 4: Provide install/uninstall scripts

How scripts work:
- MSI installers: The ingester automatically generates install and uninstall scripts. Do not add scripts unless you need to override the generated behavior.
- EXE installers: You must provide PowerShell scripts that run the installer file directly. Fleet stores the installer and sends it to the host at install time; your script must execute it using the INSTALLER_PATH environment variable.

Place scripts in `ee/maintained-apps/inputs/winget/scripts/`.

Example install script `ee/maintained-apps/inputs/winget/scripts/box_drive_install.ps1`:
```powershell
# Install system-wide, silent
# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {
  # Silent arguments vary by installer (e.g., /S, /silent, /VERYSILENT)
  $processOptions = @{
    FilePath     = "$exeFilePath"
    ArgumentList = "/VERYSILENT /NORESTART"
    PassThru     = $true
    Wait         = $true
  }

  $process = Start-Process @processOptions
  $exitCode = $process.ExitCode

  Write-Host "Install exit code: $exitCode"
  Exit $exitCode
}
catch {
  Write-Host "Error: $_"
  Exit 1
}
```

Uninstall scripts for EXE installers are vendor-specific. Use the vendor’s documented silent uninstall switch or the registered UninstallString (if available), ensuring the script runs silently and returns the installer’s exit code.

For MSI installers, you can omit scripts; for EXE installers, scripts are required.

### Step 5: Create the Winget input JSON
Create `ee/maintained-apps/inputs/winget/box-drive.json`:
```json
{
  "name": "Box Drive",
  "slug": "box-drive/windows",
  "package_identifier": "Box.Box",
  "unique_identifier": "Box",
  "installer_arch": "x64",
  "installer_type": "msi",
  "installer_scope": "machine",
  "default_categories": ["Productivity"]
}
```
Notes:
- `slug` uses `<app-name>/windows` (lowercase, dash-separated name)
- `unique_identifier` must match `programs.name` exactly

### Step 6: Generate outputs
From the repo root:
```bash
go run cmd/maintained-apps/main.go --slug="box-drive/windows" --debug
```
This updates/creates:
- `ee/maintained-apps/outputs/apps.json` (catalog entry)
- `ee/maintained-apps/outputs/box-drive/windows.json` (manifest + script refs)

### Step 7: Add description
Edit `ee/maintained-apps/outputs/apps.json` to add a human-friendly `description` for your app’s entry. For Windows entries, use the vendor description (from the winget manifest or vendor site).

### Step 8: Test in a Fleet instance
- Set an override to point your Fleet instance at your branch’s catalog:
```bash
export FLEET_DEV_MAINTAINED_APPS_BASE_URL="https://raw.githubusercontent.com/<repository-name>/fleet/refs/heads/<PR-branch-name>/ee/maintained-apps/outputs"
```
- Trigger a refresh:
```bash
fleetctl trigger --name maintained_apps
```
- Add the app to a team, deploy to a Windows host, and verify:
  - Install completes
  - App launches
  - App uninstalls cleanly

### Step 9: Open the PR
- Include “Fixes #<issue-number>” and any validation notes/screenshots
- The software group will review, validate, and merge when ready

### Troubleshooting (Windows)
- App not found in Fleet UI: ensure `apps.json` was updated by the generator and your override URL is correct
- Install fails silently: confirm your `installer_type`, `installer_arch`, and `installer_scope` match the selected winget installer; run your PowerShell script manually on a test host
- Uninstall doesn’t remove the app: prefer explicit uninstall scripts; otherwise, ensure the winget manifest exposes `ProductCode` or `UpgradeCode`
- Hash mismatch errors: if the upstream manifest is in flux, you can set `ignore_hash: true` in the input JSON (use sparingly)

## Updating existing Fleet-maintained apps

Fleet-maintained apps need to be updated as frequently as possible while maintaining reliability.  This is currently a balancing act as both scenarios below result in customer workflow blocking bugs:

- App vendor updates to installers can break install/uninstall scripts
- App vendors will deprecate download links for older installers

A Github action periodically creates a PR that updates one or more apps in the catalog by:

- Bumping versions
- Regenerating install/uninstall scripts

Each app updated in the PR must be validated independently.  Only merge the PR if all apps changed meet the following criteria:

- [X] App can be downloaded using manifest URL
- [X] App installs successfully on host using manifest install script
- [X] App exists on host
- [X] App uninstalls successfully on host using manifest uninstall script

If an app does not pass test criteria:

- [Freeze the app](#freezing-an-existing-fleet-maintained-app)
- File a bug for tracking

## Freezing an existing Fleet-maintained app

If any app fails validation:

1. Do not merge the PR as-is.
2. Add `"frozen": true"` to the failing app's input file (e.g.,`inputs/homebrew/<app>.json`).
3. Revert its corresponding output manifest file (e.g., `outputs/<slug>.json`) to the version in the `main` branch:

   ```bash
   git checkout origin/main -- ee/maintained-apps/outputs/<slug>.json
   ```

4. Validate changes in the frozen input file by running the following.  This should output no errors and generate no changes.

   ```bash
   go run cmd/maintained-apps/main.go --slug="<slug>" --debug
   ```

5. Commit both the input change and the output file revert to the same PR.
