# Fleet-maintained apps (FMA)

> **Using Claude Code?** The `new-fma` skill ([.claude/skills/new-fma/SKILL.md](../../.claude/skills/new-fma/SKILL.md)) automates this workflow and bakes in the gotchas this README doesn't cover — verifying the installed app's identity with `msiinfo`/`PlistBuddy` instead of trusting winget/cask metadata, handling bootstrapper installers, parsing unquoted `UninstallString`s, version-matching quirks, and more. Just ask it to "add X as a macOS/Windows FMA."

## Adding a new app (macOS)

1. Find the app's metadata in its [Homebrew formulae](https://formulae.brew.sh/)
2. Create a new manifest file called `$YOUR_APP_NAME.json` in the `inputs/homebrew/` directory. For
   example, if you wanted to add Box Drive, create the file `inputs/homebrew/box-drive.json`. 
3. Fill out the file according to the [input schema below](#macos-input-file-schema). For our example Box Drive app, it would look like this:

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

5. The contributor is responsible for adding the icon to Fleet (e.g. the TypeScript and website PNG components of [#29175](https://github.com/fleetdm/fleet/pull/29175/files)). These are generated using the [generate-icons](https://github.com/fleetdm/fleet/tree/main/tools/software/icons) script. **The script automatically adds the import statement and map entry to `frontend/pages/SoftwarePage/components/icons/index.ts`**, so you don't need to manually update the index file.

6. Add a description for the app in `outputs/apps.json` file. You can use descriptions from [Homebrew formulae](https://formulae.brew.sh/). For consistency and presentation on the website, the description should follow sentence casing and the following format: `<App Name>` is a(n) (copy description from Homebrew)., making sure to end with a `.`.

7. Open a PR to the `fleet` repository with the above changes. The [#g-software Engineering Manager (EM)](https://fleetdm.com/handbook/company/product-groups#software-group) is automatically added reviewer. Also, @ mention the [Fleet-maintained apps DRI](https://fleetdm.com/handbook/company/communications#:~:text=Fleet%2Dmaintained%20apps).

8. If the app passes automated tests, it is approved and merged. The EM reviews the PR within 3 business days. The app should appear shortly in the Fleet-maintained apps section when adding new software to Fleet. The app icon will not appear in Fleet until the following release.

### macOS input file schema

| Name                    | Type | Description                                                                                                                                                                                                                                   |
|--------------------------|-----------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `name`                   | string       | **Required.** User-facing name of the application.                                                                                                                                                                                                          |
| `unique_identifier`      | string       | **Required.** Platform-specific unique identifier (e.g., bundle identifier on macOS).                                                                                                                                                                       |
| `token`                  | string       | **Required.** Homebrew's unique identifier. It's the `token` field of the Homebrew API response.                                                                                                                                                                   |
| `installer_format`       | string       | **Required.** File format of the installer (`zip`, `dmg`, `pkg`). Determine via the file extension in the Homebrew API `url` field or by downloading the installer if the extension isn’t present.                                                         |
| `slug`                   | string       | **Required.** Identifies the app/platform combination (e.g., `box-drive/darwin`). Used to name manifest files and reference the app in [Fleet's best practice GitOps](https://fleetdm.com/docs/configuration/yaml-files#fleet-maintained-apps). Format: `<app-name>/<platform>`, where app name is filesystem-friendly and platform is `darwin`.             |
| `default_categories`     | string       | **Required.** Default categories for self-service if none are specified. Valid values: `Browsers`, `Communication`, `Developer Tools`, `Productivity`.                                                                                                      |
| `pre_uninstall_scripts`  | string        | Command lines run **before** the generated uninstall script (e.g., for [Box](inputs/homebrew/box-drive.json)).                                                                                                                                                                  |
| `post_uninstall_scripts` | string        | Command lines run **after** the generated uninstall script (e.g., for [Box](inputs/homebrew/box-drive.json)).                                                                                                                                                                   |
| `install_script_path`    | string        | Filepath to a custom install script (`.sh`). Overrides the generated install script. Script must be placed in `inputs/homebrew/scripts/`.                                                                                                      |
| `uninstall_script_path`  | string        | Filepath to a custom uninstall script (`.sh`). Overrides the generated uninstall script. Cannot be used together with `pre_uninstall_scripts` or `post_uninstall_scripts`. Script must be placed in `inputs/homebrew/scripts/`.                 |
| `cask_path`              | string        | Path (relative to the repo root) to a local file containing the cask JSON in the same schema as `https://formulae.brew.sh/api/cask/<token>.json`. Used to commit cask metadata for third-party taps directly into this repo under [`inputs/homebrew/custom-tap/`](inputs/homebrew/custom-tap/). See [Ingesting apps from a custom tap](#ingesting-apps-from-a-custom-tap) below. |

### Ingesting apps from a custom tap

Apps that live in a third-party Homebrew tap (not `Homebrew/homebrew-cask`) are not proxied by `https://formulae.brew.sh/api/`. To ingest them, commit both the `.rb` source and the generated `.json` into [`inputs/homebrew/custom-tap/`](inputs/homebrew/custom-tap/), laid out like a Homebrew tap:

```
custom-tap/
├── Casks/<token>.rb      # Cask DSL source
├── api/<token>.json      # Generated with regenerate.sh
└── regenerate.sh         # Rebuild api/*.json from Casks/*.rb
```

1. Write the cask DSL in `inputs/homebrew/custom-tap/Casks/<token>.rb`.
2. Run `./regenerate.sh` from inside `custom-tap/` to produce `api/<token>.json`. Requires macOS with Homebrew and `jq`.
3. In the app's input manifest (`inputs/homebrew/<token>.json`), set `cask_path` to `ee/maintained-apps/inputs/homebrew/custom-tap/api/<token>.json`. See `inputs/homebrew/fleet-desktop.json` for an example.

See [`inputs/homebrew/custom-tap/README.md`](inputs/homebrew/custom-tap/README.md) for the full contributor flow. Apps without `cask_path` continue to be fetched from `formulae.brew.sh`.

## Adding a new app (Windows)

1. Find the Winget `PackageIdentifier` in the relevant [winget-pkgs repo manifest](https://github.com/microsoft/winget-pkgs/tree/master/manifests).

2. Get the unique identifier that Fleet will use for matching the software with software inventory:
  - On a test Windows host, install the app manually, then run the following PowerShell script that correlates to the defined `installer_scope`:
    - Machine scope: `Get-ItemProperty 'HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall\*' -ErrorAction SilentlyContinue | Where-Object {$_.DisplayName -like '*<App Name>*'} | Select-Object DisplayName, DisplayVersion, Publisher`
    - User scope: `Get-ItemProperty 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\*' -ErrorAction SilentlyContinue | Where-Object {$_.DisplayName -like '*<App Name>*'} | Select-Object DisplayName, DisplayVersion, Publisher`

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


4. Run `go run cmd/maintained-apps/main.go --slug="<app-name>/windows" --debug` from the root of the
   Fleet repo to generate the app's output data, replacing `<app-name>` with your app's name, for example:

```bash
go run cmd/maintained-apps/main.go --slug="box-drive/windows" --debug
```

5. The contributor is responsible for adding the icon to Fleet (e.g. the TypeScript and website PNG components of [#29175](https://github.com/fleetdm/fleet/pull/29175/files)). These are generated using the [generate-icons](https://github.com/fleetdm/fleet/tree/main/tools/software/icons) script. **The script automatically adds the import statement and map entry to `frontend/pages/SoftwarePage/components/icons/index.ts`**, so you don't need to manually update the index file.

6. Add a description for the app in outputs/apps.json file. You can use descriptions from the wingest manifest.

7. Open a PR to the fleet repository with the above changes. The [#g-software Engineering Manager (EM)](https://fleetdm.com/handbook/company/product-groups#software-group) is automatically added reviewer. Also, @ mention the #g-software Product Designer (PD) in a comment that points them to the new icon. This way, the icon change gets a second pair of eyes.

8. If the app passes automated tests, it is approved and merged. The EM reviews the PR within 3 business days. The app should appear shortly in the Fleet-maintained apps section when adding new software to Fleet. The app icon will not appear in Fleet until the following release.

### Windows input file schema

| Name                    | Type | Description                                                                                                                                                                                                                                   |
|--------------------------|-----------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `name`                   | string       | **Required.** User-facing name of the application.                                                                                                                                                                                                          |
| `unique_identifier`      | string       | **Required.** Platform-specific unique identifier. For Windows, this is the `DisplayName`.                                                                                                                                                                       |
| `package_identifier`                  | string       | **Required.** The `PackageIdentifier` from winget. Fleet uses this to pull the correct metadata for the app.                                                                                                                                                             |
| `slug`                   | string       | **Required.** Identifies the app/platform combination (e.g., `box-drive/windows`). Used to name manifest files and reference the app in [Fleet's best practice GitOps](https://fleetdm.com/docs/configuration/yaml-files#fleet-maintained-apps). Format: `<app-name>/<platform>`, where app name is filesystem-friendly and platform is `darwin`.             |
| `installer_arch`       | string       | **Required.** `x64` or `x86` (most apps use `x64`).                                                        |
| `installer_type`       | string       | **Required.** `exe`, `msi`, or `msix` (file type, not vendor tech like "wix")                                                        |
| `installer_scope`       | string       | **Required.** `machine` or `user`. Prefer `machine` for managed installs, but it must match the scope the installer actually uses (see [Install scope vs. detection](#install-scope-vs-detection-per-user-vs-per-machine)). Verify with the installer's requested execution level (`requireAdministrator`/`highestAvailable` ⇒ machine; `asInvoker` ⇒ user) or the winget manifest `Scope`; don't guess.                                                        |
| `default_categories`     | string       | **Required.** Default categories for self-service if none are specified. Valid values: `Browsers`, `Communication`, `Developer Tools`, `Productivity`.                                                                                                      |
| `install_script_path`    | string        | Filepath to a custom install script (`.ps1`). Overrides the generated install script. Script must be placed in `inputs/winget/scripts/`. For `.msi` apps, the ingestor automatically generates install scripts. Do not add scripts unless you need to override the generated behavior. For `.exe` apps, you must provide PowerShell scripts that run the installer file directly. Fleet stores the installer and sends it to the host at install time; your script must execute it using the `INSTALLER_PATH` environment variable.                                                                                                |
| `uninstall_script_path`  | string        | Filepath to a custom uninstall script (`.ps1`). Overrides the generated uninstall script. Script must be placed in `inputs/winget/scripts/`. For `.msi` apps, the ingestor automatically generates uninstall scripts. Do not add scripts unless you need to override the generated behavior. For `.exe` apps, you must provide a script to uninstall the app. Scripts for `.exe` apps are vendor-specific. Use the vendor’s documented silent uninstall switch or the registered UninstallString (if available), ensuring the script runs silently and returns the installer’s exit code.                  |
| `fuzzy_match_name`       | boolean       | If the `unique_identifier` doesn't match the `DisplayName`, use `fuzzy_match_name` to specify that Fleet uses "fuzzy matching" to match the Fleet-maintained app and the inventoried software. For example, for Pritunl, the `unique_identifier` is "Pritunl" and the inventories software's `DisplayName` is "Pritunl Client". With `fuzzy_match_name` set to true, Pritunl app will be matched to the inventories software.  |

#### Install scope vs. detection (per-user vs. per-machine)

Windows apps can be installed **per-user** (`HKCU`, `%LOCALAPPDATA%`) or **per-machine** (`HKLM`, `Program Files`), and some ship both. A Fleet-maintained app's patch policy is its `exists` detection query with a version comparison appended, so **detection scope and "is it patched?" scope are the same predicate.** The `programs` osquery table reads both the machine and per-user registry hives, so the default `exists` query matches an app at *either* scope.

This creates a trap when a host has the app at one scope and the FMA installs at the other: a second copy is installed, the stale copy is left behind unmanaged, and because the scope-blind query still sees the stale copy, the policy never reports patched and installs may keep retrying. Reported for Slack, PowerToys, and GIMP (see [#48248](https://github.com/fleetdm/fleet/issues/48248)).

Follow these guardrails:

- **Never scope-narrow the detection query.** Do not "fix" duplicate copies by restricting the `exists` query to one hive (e.g. system-only). That makes the policy go **false-green**: it stops seeing the stale copy and reports patched while an unmanaged, unpatched copy remains on the device. The hard requirement is that the policy must **never** report "patched" while a stale copy remains at *any* scope. Keep the `exists`/`patched` query scope-blind.
- **Fix scope in remediation, not the query.** Set `installer_scope` to the scope the installer actually uses (verify it — see the schema note), and make the install/uninstall scripts converge the device on a single canonical copy:
  - **Pattern A — remove-and-replace (default):** the install script first removes any *other-scope* copy, then installs one copy at the target scope. The scope-blind policy then truthfully goes green. (PowerToys and GIMP install scripts enumerate both `HKLM` and `HKCU` to do this.)
  - **Pattern B — update-in-place per scope:** detect which scope the existing copy lives in and upgrade *that* one. More data-safe; per-user installs from the SYSTEM context are the hard case (run in the logged-on user's session via a scheduled task — see the Slack MSIX script).
- **MSIX-managed apps: every Win32 copy is legacy.** For apps Fleet manages as MSIX (e.g. Slack, Microsoft Teams), there is no "same-scope" Win32 copy to leave for the installer — a leftover exe/MSI copy at *either* scope keeps the scope-blind policy red while the MSIX is current. Their install scripts sweep **both** Win32 uninstall hives before provisioning, with guards: never touch PackageFullName-style keys or entries under `\WindowsApps\` (so a re-run can't remove the MSIX itself), skip entries with no quiet uninstall path (a raw `UninstallString` run as SYSTEM can hang on UI), and delete a per-user `HKEY_USERS` uninstall key only after verifying the uninstaller removed itself from disk (per-user uninstallers run as SYSTEM can't clean their own key — but deleting the key while files remain would be false-green).
- **Keep installer ↔ detection ↔ uninstaller consistent.** `installer_scope`, the install/uninstall scripts, and the detection query must agree about the app. Avoid over-narrow `name = '…'` matches that could miss a scope variant; use `fuzzy_match_name` or a custom `exists_query` where a package's `DisplayName` differs by scope.
- **Data preservation is a separate guarantee from "no false-green."** A same-scope upgrade preserves data normally. Cross-scope removal (Pattern A) can lose data: config in `%APPDATA%`/`HKCU` for the same packaging is usually **preserved**, Squirrel/Electron local session/cache may be **partially** lost, and cross-packaging to MSIX generally **resets** data. Call this out in the PR when it applies.

`installer_scope` is enforced to be `machine` or `user` for every winget input by `TestInputInstallerScopeIsSet`.

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
