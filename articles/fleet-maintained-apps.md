# Fleet-maintained apps

_Available in Fleet Premium_

In Fleet, you can install Fleet-maintained apps on macOS and Windows hosts without the need for manual uploads or extra configuration. This simplifies the process and adds another source of applications for your fleet.

Fleet maintains installation metadata for [a number of apps](https://github.com/fleetdm/fleet/blob/main/ee/maintained-apps/outputs/apps.json), letting you add them to your own Fleet instance and install them on your hosts without any additional configuration.

## Important notes on CPU architecture

### macOS

Currently, the macOS versions of these apps are Apple Silicon-only rather than universal:

* 1Password
* Brave
* Docker Desktop
* Figma
* Microsoft Visual Studio (VS) Code,
* Notion
* Postman
* Slack
* Zoom

### Windows

Fleet prefers 64-bit x86 versions of applications when available. Installing on Arm hosts (e.g. in a VM on an Apple Silicon machine) may not work or have other unintended consequences.

## Add a Fleet-maintained app

1. Head to the **Software** page for a team, then click **Add software**. You'll land on the Fleet-maintained apps list.
2. Click the **Add** button for the app and platform you wish to add.

> You'll see a ✅ icon instead of an **Add** button if the application has already been added to your team as a custom package or VPP app, or if you've already added the Fleet-maintained app.

3. Click **Add software** to download the installer package from the app's publisher into Fleet and make it available for install for your selected team.

Fleet verifies install and uninstall scripts for each maintained app, and keeps the scripts up to date as an app's vendor releases new versions. You can override Fleet's scripts, or add pre-install queries or post-install scripts, either when adding the app (by clicking **Advanced options**) or later on (by editing the package).

## Install the app

You can install a Fleet-maintained app three ways:

1. Manually in the **Host Details** page under the **Software** tab. Select the app you just added and choose **Install** from the **Actions** dropdown.
2. Manually from the **Self-service** tab on the **My Device** page from an end user's machine, if you've [enabled Self-service](https://fleetdm.com/guides/software-self-service) for the app.
3. Automatically on hosts via [policy automations](https://fleetdm.com/guides/automatic-software-install-in-fleet).

You can track the installation process in the **Activities** section on the **Details** tab of this **Host Details** page.

## Uninstall the app

To remove the app, navigate to the **Host Details** page for the appropriate host, then to the **Software** tab. Find the app, then click on the **Actions** drop-down, then **Uninstall**.

Fleet will run the uninstall script configured for the software title. For macOS, Fleet generates default scripts based on the Homebrew recipe (see `zap` in recipe). For Windows, Fleet leverages MSI or .exe data to generate default scripts.

The uninstallation process is also visible in the  **Activities** section on the **Details** tab of this **Host Details** page.

## Update the app

To get the latest version of a Fleet-maintained app,

1. Remove the app from the team.
2. Re-add it from the Fleet-maintained list on the **Software** page.
3. Install the new version of the app via one of the three methods above.

A streamlined flow for pulling the latest version of a Fleet-maintained app is [coming soon](https://github.com/fleetdm/fleet/issues/25636).

## How does Fleet maintain these apps?

Fleet:

- verifies, installs, uninstalls & tests all Fleet-maintained apps alongside the install and uninstall scripts we generate
- transforms data from multiple sources, including [Homebrew Casks](https://github.com/Homebrew/homebrew-cask) and [WinGet manifests](https://github.com/microsoft/winget-pkgs/tree/master/manifests), into [standardized manifests](https://github.com/fleetdm/fleet/blob/main/ee/maintained-apps/outputs/), checking data sources [multiple times per day](https://github.com/fleetdm/fleet/blob/main/.github/workflows/ingest-maintained-apps.yml)
- fetches the [full maintained apps list](https://github.com/fleetdm/fleet/blob/main/ee/maintained-apps/outputs/apps.json) from GitHub daily (or when you run `fleetctl trigger --name=maintained_apps`)
- fetches an individual app's manifest when the **Add** button is pressed from the maintained apps list in the UI, and when an individual app is [retrieved](https://fleetdm.com/docs/rest-api/rest-api#get-fleet-maintained-app) or [added](https://fleetdm.com/docs/rest-api/rest-api#add-fleet-maintained-app) via the REST API
- DOES NOT directly pull data from WinGet or Homebrew to end-user devices

<meta name="category" value="guides">
<meta name="authorFullName" value="Gabriel Hernandez">
<meta name="authorGitHubUsername" value="ghernandez345">
<meta name="publishedOn" value="2025-04-03">
<meta name="articleTitle" value="Fleet-maintained apps">
