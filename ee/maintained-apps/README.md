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

### macOS input file schema

| Field                    | Required? | Description                                                                                                                                                                                                                                   |
|--------------------------|-----------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `name`                   | Yes       | User-facing name of the application.                                                                                                                                                                                                          |
| `unique_identifier`      | Yes       | Platform-specific unique identifier (e.g., bundle identifier on macOS).                                                                                                                                                                       |
| `token`                  | Yes       | Homebrew's unique identifier. It's the `token` field of the Homebrew API response.                                                                                                                                                                   |
| `installer_format`       | Yes       | File format of the installer (`zip`, `dmg`, `pkg`). Determine via the file extension in the Homebrew API `url` field or by downloading the installer if the extension isn’t present.                                                         |
| `slug`                   | Yes       | Identifies the app/platform combination (e.g., `box-drive/darwin`). Used to name manifest files and reference the app in [Fleet's best practice GitOps](https://fleetdm.com/docs/configuration/yaml-files#fleet-maintained-apps). Format: `<app-name>/<platform>`, where app name is filesystem-friendly and platform is `darwin`.             |
| `pre_uninstall_scripts`  | No        | Command lines run **before** the generated uninstall script (e.g., for [Box](inputs/homebrew/box-drive.json)).                                                                                                                                                                  |
| `post_uninstall_scripts` | No        | Command lines run **after** the generated uninstall script (e.g., for [Box](inputs/homebrew/box-drive.json)).                                                                                                                                                                   |
| `default_categories`     | Yes       | Default categories for self-service if none are specified. Valid values: `Browsers`, `Communication`, `Developer Tools`, `Productivity`.                                                                                                      |
| `install_script_path`    | No        | Filepath to a custom install script (`.sh`). Overrides the generated install script. Script must be placed in `inputs/homebrew/scripts/`.                                                                                                      |
| `uninstall_script_path`  | No        | Filepath to a custom uninstall script (`.sh`). Overrides the generated uninstall script. Cannot be used together with `pre_uninstall_scripts` or `post_uninstall_scripts`. Script must be placed in `inputs/homebrew/scripts/`.                 |

## Adding a new app (Windows)

1. Find the Winget PackageIdentifier in the [winget-pkgs repo](https://github.com/microsoft/winget-pkgs).

2. Get the unique identifier that Fleet will use for matching the software with software inventory:
  - On a test Windows host, install the app manually, then run the following PowerShell script: `Get-ItemProperty 'HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall\*' | Where-Object {$_.DisplayName -like '*<App Name>*'} | Select-Object DisplayName, DisplayVersion, Publisher`
  - Use the exact value from `DisplayName` as the `unique_identifier`.

If the `unique_identifier` doesn't match the `DisplayName`, then Fleet will incorrectly create two software titles when the Fleet-maintained app is added and later installed. One title for the Fleet-maintained app and a separate title for the inventoried software.

3. Fill out the file according to the [input schema](#windows-input-file-schema). For example, Box Drive looks like this:

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

4. Run the following command from the root of the Fleet repo to generate the app's output data:

```bash
go run cmd/maintained-apps/main.go --slug="box-drive/windows" --debug
```

5. Icon? TODO

6. Add a description for the app in outputs/apps.json file. You can use descriptions from the wingest manifest.

7. Open a PR to the fleet repository with the above changes. The [#g-software Engineering Manager (EM)](https://fleetdm.com/handbook/company/product-groups#software-group) is automatically added reviewer. Also, @ mention the #g-software Product Designer (PD) in a comment that points them to the new icon. This way, the icon change gets a second pair of eyes.

8. If the app passes automated tests, it is approved and merged. The EM reviews the PR within 1 business day. The app should appear shortly in the Fleet-maintained apps section when adding new software to Fleet. The app icon will not appear in Fleet until the following release.

### Windows input file schema

| Field                    | Required? | Description                                                                                                                                                                                                                                   |
|--------------------------|-----------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `name`                   | Yes       | User-facing name of the application.                                                                                                                                                                                                          |
| `unique_identifier`      | Yes       | Platform-specific unique identifier. For Windows, this is the `DisplayName`.                                                                                                                                                                       |
| `package_identifier`                  | Yes       | TODO                                                                                                                                                             |
| `slug`                   | Yes       | Identifies the app/platform combination (e.g., `box-drive/windows`). Used to name manifest files and reference the app in [Fleet's best practice GitOps](https://fleetdm.com/docs/configuration/yaml-files#fleet-maintained-apps). Format: `<app-name>/<platform>`, where app name is filesystem-friendly and platform is `darwin`.             |
| `installer_arch`       | Yes       | `x64` or `x86` (most apps use `x64`).                                                        |
| `installer_type`       | Yes       | `exe`, `msi`, or `msix` (file type, not vendor tech like "wix")                                                        |
| `installer_scope`       | Yes       | `machine` or `user` (prefer `machine` for managed installs)                                                        |
| `default_categories`     | Yes       | Default categories for self-service if none are specified. Valid values: `Browsers`, `Communication`, `Developer Tools`, `Productivity`.                                                                                                      |
| `install_script_path`    | No        | Filepath to a custom install script (`.ps1`). Overrides the generated install script. Script must be placed in `inputs/winget/scripts/`. For `.msi` apps, the ingestor automatically generates install scripts. Do not add scripts unless you need to override the generated behavior. For `.exe` apps, you must provide PowerShell scripts that run the installer file directly. Fleet stores the installer and sends it to the host at install time; your script must execute it using the `INSTALLER_PATH` environment variable.                                                                                                |
| `uninstall_script_path`  | No        | Filepath to a custom uninstall script (`.ps1`). Overrides the generated uninstall script. Script must be placed in `inputs/winget/scripts/`. For `.msi` apps, the ingestor automatically generates uninstall scripts. Do not add scripts unless you need to override the generated behavior. For `.exe` apps, you must provide a script to uninstall the app. Scripts for `.exe` apps are vendor-specific. Use the vendor’s documented silent uninstall switch or the registered UninstallString (if available), ensuring the script runs silently and returns the installer’s exit code.                  |
| `installer_locale`       | No       | TODO                                                        |
| `program_publisher`       | No       | TODO                                                        |
| `uninstall_type`       | No       | TODO                                                        |
| `fuzzy_match_name`       | No       | TODO                                                        |

#### Windows troubleshooting

- App not found in Fleet UI: ensure `apps.json` was updated by the generator and your override URL is correct
- Install fails silently: confirm your `installer_type`, `installer_arch`, and `installer_scope` match the selected winget installer; run your PowerShell script manually on a test host
- Uninstall doesn’t remove the app: prefer explicit uninstall scripts; otherwise, ensure the winget manifest exposes `ProductCode` or `UpgradeCode`
- Hash mismatch errors: if the upstream manifest is in flux, you can set `ignore_hash: true` in the input JSON (use sparingly)

#### Can I do this on macOS?

The instructions below are meant to be run on a Windows host. But, you can run most of this on a macOS host, as well:
- You can author Windows inputs and run the generator on macOS. The ingester is Go code that fetches data from winget/GitHub and works cross‑platform.
- To find the PackageName and Publisher, you can look in the locale and installer yaml files in the winget-pkgs repo.
- Validation and testing still require a Windows host (to verify programs.name and to run install/uninstall).

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
