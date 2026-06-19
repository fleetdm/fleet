# Fleet 4.87.0 | 800+ new apps, custom OS updates, non-admin local accounts, and more...

<div purpose="embedded-content">
   <iframe src="TODO" title="0" allowfullscreen></iframe>
</div>

Fleet 4.87.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.87.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- [800+ new Fleet-maintained apps](#800--new-fleet-maintained-apps)
- [Custom OS update profiles](#custom-os-update-profiles)
- [Configuration profiles: Include + exclude](#configuration-profiles-include--exclude)
- [macOS local account: non-admin (standard) or skip](#macos-local-account-non-admin-standard-or-skip)
- [Self-service software categories](#self-service-software-categories)
- [Android commands: Lock, wipe, & clear passcode](#android-commands-lock-wipe--clear-passcode)
- [Policy automation continuous retry](#policy-automation-continuous-retry)
- [Command palette](#command-palette)

### 800+ new Fleet-maintained apps

_Available in Fleet Premium_

Fleet 4.87 adds 800+ new Fleet-maintained apps which brings [the catalog](https://fleetdm.com/software-catalog) to over 1,250 apps. IT admins can add any of these under **Software > Add software > Fleet-maintained** and deploy with a single click.

Windows gets its biggest catalog expansion yet. Highlights include:

- **Microsoft Office**, **PowerShell**, **PowerToys**, **Power BI**, **Power Automate**, and **SQL Server Management Studio** for Windows-centric environments
- **Git**, **Node.js**, **Python 3.13 and 3.14**, and **PostgreSQL 15–18** for development teams
- **Windsurf** and **Kiro** for developers using AI-powered coding IDEs
- **Dell Command Update** and **Lenovo Dock Manager** for hardware fleet management
- **Nessus Agent** for vulnerability scanning and **Bitwarden** for password management

New macOS apps include **Kiro**, **Codex**, and **OpenCode** for AI-assisted development, plus hundreds more tools across productivity, design, security, and media.

### Custom OS update profiles

_Available in Fleet Premium_

Fleet now supports deploying custom [Declarative Device Management (DDM) Software Update enforcement](https://github.com/apple/device-management/blob/release/declarative/declarations/configurations/softwareupdate.enforcement.specific.yaml) declarations on macOS, iOS, and iPadOS, as well as custom Windows profiles using the [Windows Update CSPs](https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-csp-update). This gives IT admins full control over OS update enforcement, including the exact enforcement deadline time.

Fleet enforces mutual exclusion with its built-in OS update controls: configuring both returns a clear error, so nothing conflicts silently.

GitHub issue: [#38802](https://github.com/fleetdm/fleet/issues/38802)

### Configuration profiles: Include + exclude

_Available in Fleet Premium_

Configuration profiles now support combining the **Include any** label targeting, a host receives a profile if it matches any label in the include list, with the new **Exclude any** option. This way, IT admins can define broad inclusions and exclude specific hosts without writing complex label queries.

For example: deliver a Wi-Fi profile to all macOS devices (`include_any: macOS`) while excluding hosts tagged "Guest" or "Loaner." Both options work across all platforms: macOS, iOS, iPadOS, Windows, and Android.

GitHub issue: [#32073](https://github.com/fleetdm/fleet/issues/32073)

### macOS local account: non-admin (standard) or skip

_Available in Fleet Premium_

Building on the [local admin account](https://fleetdm.com/releases/fleet-4-85-0#create-a-local-admin-account-during-macos-setup) introduced in 4.85 and [password rotation](https://fleetdm.com/releases/fleet-4-86-0#rotate-local-admin-password) added in 4.86, Fleet now lets IT admins control the end-user account type during macOS Setup Assistant. On the **Controls > Setup experience > Users** page, choose **Standard** to create a non-admin end-user account, or **Skip** to skip end-user account creation entirely. This is useful when the hidden admin is the only local account the device needs. Selecting **Standard** or **Skip** automatically requires the hidden local admin to be created.

GitHub issue: [#41781](https://github.com/fleetdm/fleet/issues/41781)

### Self-service software categories

_Available in Fleet Premium_

IT admins can now create custom software categories to bucket applications by team, role, or project (e.g., "Product development") so end users can get fully set up for their projects. End users see an **Install all in category** button that installs all apps in a category, in alphanumeric order, with a single click.

GitHub issue: [#39018](https://github.com/fleetdm/fleet/issues/39018)

### Android commands: Lock, wipe, & clear passcode

_Available in Fleet Premium_

Fleet can now send lock, wipe, and clear passcode commands to Android hosts directly from the **Host details** page. For company-owned (fully managed) devices, all three commands are available. For personally-owned (BYOD) Android hosts, lock and clear passcode are available and scoped to the work profile. Each action is logged in Fleet's [audit logs](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/reference/audit-logs.md). The [`fleetctl` CLI tool](https://fleetdm.com/guides/fleetctl) also supports these via `fleetctl mdm lock`, `fleetctl mdm wipe`, and `fleetctl mdm clear-passcode` commands.

GitHub issue: [#41683](https://github.com/fleetdm/fleet/issues/41683)

### Policy automation continuous retry

_Available in Fleet Premium_

A new **Run automation on every failure** option lets IT admins trigger software installation or script-run automations every time a host fails a policy check, not just the first time. If a host falls back out of compliance after an initial remediation or the initial remediation fails, Fleet automatically runs the fix again without manual intervention. 

GitHub issue: [#42651](https://github.com/fleetdm/fleet/issues/42651)

### Command palette

Fleet now includes a command palette. Press ⌘+K (or Ctrl+K on Windows and Linux) from anywhere in the app to instantly navigate to any page, trigger any action, or jump to any setting. The palette respects your role by showing or hiding items based on your permissions. Fleet Premium users with multiple fleets can jump directly to the fleet switcher with ⌘+Shift+F (Ctrl+Shift+F on Windows and Linux). Sub-pages let you search hosts, software titles, reports, and policies by name without leaving the keyboard.

GitHub issue: [#43757](https://github.com/fleetdm/fleet/issues/43757)

## Changes

- Added 236 new Fleet-maintained apps for Windows, including Microsoft Office, PowerShell, PowerToys, Power BI, Power Automate, SQL Server Management Studio, Microsoft .NET Runtime 8 and 10, Git, Node.js, Python 3.13 and 3.14, PostgreSQL 15–18, Windsurf, Kiro, Dell Command Update, Lenovo Dock Manager, Nessus Agent, Bitwarden, Canva, Miro, Snagit, Tableau Desktop, VirtualBox, TortoiseGit, GitHub Desktop, and more.
- Added 727 new Fleet-maintained apps for macOS, including Kiro, Codex, OpenCode, Claude DevTools, Granola, Logitune, and hundreds more tools across development, security, productivity, and design.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.87.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-06-17">
<meta name="articleTitle" value="Fleet 4.87.0 | 800+ new apps, custom OS updates, Android commands, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.87.0-1600x900@2x.png">
