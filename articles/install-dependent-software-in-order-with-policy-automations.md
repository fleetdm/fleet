# Install dependent software in order with policy automations

Some apps need another piece of software, like a runtime or framework, installed first. Fleet doesn't support install ordering or dependencies between policy-triggered installs. Each policy's `install_software` automation targets one software item, and policies run independently on each host's check-in cadence, so there's no guarantee which install runs first. This guide covers two ways to install the prerequisite before the app that depends on it on macOS, Windows, and Linux hosts: gate the app's policy on the prerequisite, or install both from one custom install script.

## Prerequisites

- Fleet Premium. Policy automations that install software are a Premium feature.
- A GitOps repository for your Fleet instance. The examples use GitOps YAML, but you can set up the same policies and install scripts in the Fleet UI.
- Both the prerequisite and the dependent app added to the same fleet (or "Unassigned"). Policy automations can only install software defined in the same fleet as the policy.

> **Note:** Listing two software items under one `install_software` key isn't valid. Each policy installs exactly one software item.

## Choose an approach

| | Two gated policies | Custom install script |
|---|---|---|
| **How it works** | The app's policy only fails after the prerequisite is installed | One script installs the prerequisite, then the app |
| **Prerequisite tracked in Fleet** | Yes, it's its own software item with its own install status | No, Fleet only sees the app's install |
| **Timing** | The app installs one or more policy cycles after the prerequisite | Both install in the same run |
| **Maintenance** | Two policies, no custom scripts | You maintain the full install script, including the app's install command |
| **Supported software** | Fleet-maintained apps, custom packages, and macOS App Store apps | Fleet-maintained apps and custom packages only |

Use two gated policies when the prerequisite is something you want Fleet to install and report on, or when the dependent app is an App Store app. Use a custom install script when the prerequisite only exists to support one app and you need both installed in a single run.

## Option 1: Gate the app's policy on the prerequisite

Create two policies:

1. The prerequisite's policy fails when the prerequisite is missing, which triggers its install.
2. The app's policy passes while the prerequisite is missing. It only fails when the prerequisite is present and the app is missing, which triggers the app's install.

This works with Fleet's default automation behavior. Fleet triggers an install when a host's policy result changes from passing to failing. When the prerequisite lands, the app's policy flips from passing to failing on the next policy run, and Fleet installs the app. If the prerequisite install fails, the app's policy keeps passing, so Fleet never tries to install the app without its prerequisite.

> **Note:** The app installs on a later policy run, not in the same run as the prerequisite. Fleet updates policy results once per [policy update interval](https://fleetdm.com/docs/configuration/fleet-server-configuration#osquery-policy-update-interval) (1 hour by default), so expect the app to install up to one interval after the prerequisite.

### Write the policy queries

Replace the placeholders with the names your hosts report. Find them on the host's **Software** tab or in the [software inventory](https://fleetdm.com/guides/software-inventory-reference).

**macOS** uses the `apps` table:

```sql
-- Prerequisite: passes when the prerequisite is installed
SELECT 1 FROM apps WHERE bundle_identifier = '<PREREQ_BUNDLE_ID>';

-- App: passes when the prerequisite is missing or the app is installed
SELECT 1 WHERE NOT EXISTS (
  SELECT 1 FROM apps WHERE bundle_identifier = '<PREREQ_BUNDLE_ID>'
) OR EXISTS (
  SELECT 1 FROM apps WHERE bundle_identifier = '<APP_BUNDLE_ID>'
);
```

If the prerequisite is a package without an app bundle (for example, a command-line runtime), check the `package_receipts` table instead: `SELECT 1 FROM package_receipts WHERE package_id = '<PREREQ_PACKAGE_ID>'`.

**Windows** uses the `programs` table:

```sql
-- Prerequisite
SELECT 1 FROM programs WHERE name LIKE '<PREREQ_NAME>%';

-- App
SELECT 1 WHERE NOT EXISTS (
  SELECT 1 FROM programs WHERE name LIKE '<PREREQ_NAME>%'
) OR EXISTS (
  SELECT 1 FROM programs WHERE name LIKE '<APP_NAME>%'
);
```

**Linux** uses the `deb_packages` table on Debian-based hosts and `rpm_packages` on RPM-based hosts. On RPM-based hosts, swap the table name:

```sql
-- Prerequisite
SELECT 1 FROM deb_packages WHERE name = '<PREREQ_PACKAGE_NAME>';

-- App
SELECT 1 WHERE NOT EXISTS (
  SELECT 1 FROM deb_packages WHERE name = '<PREREQ_PACKAGE_NAME>'
) OR EXISTS (
  SELECT 1 FROM deb_packages WHERE name = '<APP_PACKAGE_NAME>'
);
```

> **Note:** On Linux, a `.deb` or `.rpm` that declares its dependencies may not need either approach. Fleet's default install scripts use `apt-get` and `dnf`, which install declared dependencies from the host's configured repositories. Use these approaches when the prerequisite isn't declared or isn't in those repositories.

### Add the policies in GitOps

In `fleets/fleet-name.yml`, add both software items and both policies. This example installs a custom package prerequisite and a Fleet-maintained app on Windows:

```yaml
software:
  packages:
    - path: ../lib/windows/software/prereq.package.yml
  fleet_maintained_apps:
    - slug: <app-slug>/windows

policies:
  - name: Windows - <Prerequisite> installed
    platform: windows
    description: Checks that <Prerequisite> is installed.
    resolution: Fleet installs <Prerequisite> automatically.
    query: "SELECT 1 FROM programs WHERE name LIKE '<PREREQ_NAME>%';"
    install_software:
      package_path: ../lib/windows/software/prereq.package.yml
  - name: Windows - <App> installed
    platform: windows
    description: Checks that <App> is installed. Passes until <Prerequisite> is installed.
    resolution: Fleet installs <App> automatically after <Prerequisite> is installed.
    query: >-
      SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name LIKE '<PREREQ_NAME>%')
      OR EXISTS (SELECT 1 FROM programs WHERE name LIKE '<APP_NAME>%');
    install_software:
      fleet_maintained_app_slug: <app-slug>/windows
```

Use the same shape on macOS and Linux with the matching `platform` (`darwin` or `linux`) and query. To install an App Store app on macOS, use `install_software.app_store_id` instead. See the [policies reference](https://fleetdm.com/docs/configuration/yaml-files#policies) for every `install_software` option.

> **Warning:** Don't set `continuous_automations_enabled: true` on the app's policy unless you need it. With it on, Fleet retries the install on every failing result, which can loop if the app installs but the policy query doesn't match what the host reports.

## Option 2: Install the prerequisite from a custom install script

Set `install_script.path` on the dependent app to a script that installs the prerequisite first, then the app. One policy triggers the whole install.

Before you write the script:

- Custom install scripts are supported for Fleet-maintained apps (macOS and Windows) and custom packages (`.pkg`, `.msi`, `.exe`, `.deb`, `.rpm`, and `.tar.gz`). They aren't supported for App Store apps or script-only packages (`.sh`, `.py`, and `.ps1`).
- The script replaces Fleet's install script for the app. It must include the app's complete install command, not only the prerequisite step. Start from the app's default script: Fleet's [default scripts](https://github.com/fleetdm/fleet/tree/main/pkg/file/scripts) for custom packages, or the app's [metadata on GitHub](https://github.com/fleetdm/fleet/tree/main/ee/maintained-apps/outputs) for Fleet-maintained apps.
- Fleet only sends the app's installer to the host, at `$INSTALLER_PATH`. The script has to download the prerequisite itself, so the host needs network access to the prerequisite's URL.

> **Warning:** The prerequisite isn't a Fleet-tracked install. Fleet doesn't report its install status, and it can't be uninstalled or patched from Fleet as part of the app. You maintain the script when either installer changes.

### Write the install script

Each script skips the prerequisite if it's already installed and exits with an error if the prerequisite install fails, so Fleet reports the app install as failed.

**macOS** (`lib/macos/scripts/app-install.sh`):

```sh
#!/bin/sh
set -e

PREREQ_URL="https://example.com/prereq.pkg"
PREREQ_PKG_ID="<PREREQ_PACKAGE_ID>"

if ! pkgutil --pkg-info "$PREREQ_PKG_ID" >/dev/null 2>&1; then
  TMP_DIR="$(mktemp -d)"
  curl -fsSL "$PREREQ_URL" -o "$TMP_DIR/prereq.pkg"
  installer -pkg "$TMP_DIR/prereq.pkg" -target /
  rm -rf "$TMP_DIR"
fi

# Replace with the app's default install script
installer -pkg "$INSTALLER_PATH" -target /
```

**Windows** (`lib/windows/scripts/app-install.ps1`):

```powershell
$prereqUrl = "https://example.com/prereq.msi"
$prereqName = "<PREREQ_NAME>*"
$successCodes = @(0, 3010, 1641)

$uninstallKeys = @(
  "HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall\*",
  "HKLM:\Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*"
)
$prereqInstalled = Get-ItemProperty $uninstallKeys -ErrorAction SilentlyContinue |
  Where-Object { $_.DisplayName -like $prereqName }

try {
  if (-not $prereqInstalled) {
    $prereqPath = Join-Path $env:TEMP "prereq.msi"
    Invoke-WebRequest -Uri $prereqUrl -OutFile $prereqPath -UseBasicParsing
    $prereq = Start-Process msiexec.exe -ArgumentList "/i `"$prereqPath`" /quiet /norestart" -Wait -PassThru
    Remove-Item $prereqPath -Force
    if ($successCodes -notcontains $prereq.ExitCode) { Exit $prereq.ExitCode }
  }

  # Replace with the app's default install script
  $app = Start-Process msiexec.exe -ArgumentList "/i `"$env:INSTALLER_PATH`" /quiet /norestart" -Wait -PassThru
  if ($successCodes -contains $app.ExitCode) { Exit 0 }
  Exit $app.ExitCode
} catch {
  Write-Host "Error: $_"
  Exit 1
}
```

**Linux** (`lib/linux/scripts/app-install.sh`), for a Debian-based host where the prerequisite is in the host's package repositories:

```sh
#!/bin/sh
set -e

apt-get update
apt-get install --assume-yes <PREREQ_PACKAGE_NAME>

# Replace with the app's default install script
apt-get install --assume-yes -f "$INSTALLER_PATH"
```

On RPM-based hosts, use `dnf install --assumeyes` for both lines and drop `apt-get update`.

### Add the script and policy in GitOps

For a Fleet-maintained app, set `install_script.path` on the app in `fleets/fleet-name.yml`, then point a policy at it:

```yaml
software:
  fleet_maintained_apps:
    - slug: <app-slug>/windows
      install_script:
        path: ../lib/windows/scripts/app-install.ps1

policies:
  - name: Windows - <App> installed
    platform: windows
    description: Checks that <App> is installed.
    resolution: Fleet installs <Prerequisite> and <App> automatically.
    query: "SELECT 1 FROM programs WHERE name LIKE '<APP_NAME>%';"
    install_software:
      fleet_maintained_app_slug: <app-slug>/windows
```

For a custom package, set `install_script.path` in the package YAML file instead:

```yaml
# lib/linux/software/app.package.yml
- url: https://example.com/app.deb
  install_script:
    path: ../scripts/app-install.sh
```

Then reference the package from a policy with `install_software.package_path`. See the [software reference](https://fleetdm.com/docs/configuration/yaml-files#software) for every install script option.

## Verify

1. In Fleet, go to **Hosts** and select a host in scope.
2. Select the **Policies** tab and confirm the policies show as passing after the next policy run.
3. Select the **Software** tab and confirm both the prerequisite and the app appear in inventory.
4. With Option 1, select the **Details** tab and, in the **Activity** card, confirm the prerequisite install ran before the app install.

## Troubleshoot

**The app never installs (Option 1)**

Confirm the prerequisite actually installed and appears in the host's software inventory. The app's policy only fails once the prerequisite's name in the query matches what the host reports. Run the prerequisite part of the app's query as a live report against the host to check.

**The app's policy fails on hosts without the prerequisite (Option 1)**

The `NOT EXISTS` part of the query isn't matching. Check that the prerequisite query in the app's policy is identical to the prerequisite's own policy query.

**The install fails with a download error (Option 2)**

The host can't reach the prerequisite's URL. Confirm the URL is reachable from the host's network without authentication, or host the prerequisite somewhere the host can reach.

**The app installed but the prerequisite didn't (Option 2)**

The script's "already installed" check returned a false positive. Run the check on its own on the host (`pkgutil --pkg-info` on macOS, or the registry lookup on Windows) and tighten the name or package ID.

## Further reading

- [Automatically install software](https://fleetdm.com/guides/automatic-software-install-in-fleet)
- [Deploy software](https://fleetdm.com/guides/deploy-software-packages)
- [GitOps policies reference](https://fleetdm.com/docs/configuration/yaml-files#policies)

<meta name="articleTitle" value="Install dependent software in order with policy automations">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-10-20">
<meta name="description" value="Install a prerequisite before the app that depends on it on macOS, Windows, and Linux using gated policies or a custom install script.">
