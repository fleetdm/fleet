# Enforce OS updates on macOS with Nudge

[Nudge](https://github.com/macadmins/nudge) is an open-source app from the Mac admins community that reminds end users to update macOS. This guide shows how to deploy Nudge with Fleet: the app, a LaunchAgent that opens it on a schedule, and a configuration profile with your update requirements. It's the same setup Fleet uses on its own computers.

## Nudge or Fleet's built-in OS updates

Fleet's [built-in OS updates](https://fleetdm.com/guides/enforce-os-updates) use Apple's declarative device management (DDM). You set a minimum version and a deadline. macOS notifies the end user, and when the deadline passes, macOS installs the update and restarts the Mac. It needs no extra software, and the update is guaranteed to happen.

Nudge never installs an update or restarts the Mac itself. Instead, it shows a window that asks the end user to update and opens **Software Update** for them. Use Nudge when you want more control over that experience:

- Customize the message, logo, and links, like a link to your internal update policy.
- Let end users defer, and control how often Nudge reappears as the deadline gets closer.
- Show end users which actively exploited vulnerabilities the update fixes.
- Avoid interrupting end users while their camera is on or they're sharing their screen.
- Give newly enrolled Macs a grace period before the deadline applies.

> **Note:** You can use both. If you also turn on Fleet's built-in OS updates, set the same minimum version and deadline in both places so end users don't see conflicting dates.

## Prerequisites

- Fleet Premium. Installing software is a Premium feature.
- Apple MDM turned on, with your Macs enrolled.
- The Nudge LaunchAgent package (`Nudge_LaunchAgent-<version>.pkg`), downloaded from the [Nudge releases page](https://github.com/macadmins/nudge/releases). The Nudge Fleet-maintained app installs only the app, so the LaunchAgent comes in a separate package.

## Step 1: Add the Nudge app

Add Nudge as a [Fleet-maintained app](https://fleetdm.com/guides/fleet-maintained-apps).

1. Go to **Software**, select the fleet, and select **Add software > Fleet-maintained**.
2. Find **Nudge** and select it.
3. Select **Add software**.

In GitOps, add this to the fleet's YAML file:

```yaml
software:
  fleet_maintained_apps:
    - slug: nudge/darwin
```

## Step 2: Add the Nudge LaunchAgent

The LaunchAgent opens Nudge on a schedule, so end users see the reminder even if they never open Nudge themselves.

1. Download the [post-install script](https://github.com/fleetdm/fleet/blob/main/docs/solutions/macos/scripts/nudge-postinstall.sh) (`nudge-postinstall.sh`). It loads the LaunchAgent right after install, without waiting for the next login.

2. Go to **Software > Add software > Custom package** and upload the Nudge LaunchAgent package.
3. Under **Advanced options**, paste the script into **Post-install script**.
4. Select **Add software**.

In GitOps, add a package YAML file next to the downloaded package and reference it in the fleet's YAML file:

```yaml
# lib/macos/software/nudge-launchagent.yml
path: ../packages/Nudge_LaunchAgent.pkg
post_install_script:
  path: ../scripts/nudge-postinstall.sh
```

```yaml
software:
  packages:
    - path: ../lib/macos/software/nudge-launchagent.yml
```

## Step 3: Configure Nudge with a profile

Nudge reads its settings from a configuration profile. The example profile requires Macs to install the latest minor update for their major version. Nudge gets the latest versions from the [SOFA feed](https://sofa.macadmins.io/) and sets the deadline based on each update's release date, so you don't have to update the profile for every macOS release. The profile also lets Nudge run as a background task without end users turning it off.

1. Download the [Nudge profile](https://github.com/fleetdm/fleet/blob/main/docs/solutions/macos/configuration-profiles/nudge.mobileconfig) (`nudge.mobileconfig`). Replace each `REPLACE-WITH-UUID` with a unique value from `uuidgen`.

2. Go to **Controls > OS settings > Configuration profiles** and select **Add profile**.
3. Upload `nudge.mobileconfig`.

In GitOps, add the profile to the fleet's YAML file:

```yaml
controls:
  apple_settings:
    configuration_profiles:
      - path: ../lib/macos/configuration-profiles/nudge.mobileconfig
```

To set a fixed deadline, require a major upgrade, or customize the reminder text, deferrals, and how often Nudge appears, see the [Nudge wiki](https://github.com/macadmins/nudge/wiki). Fleet's own [Nudge profile](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/configuration-profiles/nudge-configuration.mobileconfig) is a fuller example.

## Step 4: Optional: Reinstall Nudge if it's removed

Add policies that install Nudge and the LaunchAgent automatically on hosts that are missing them:

```yaml
- name: Nudge installed
  query: SELECT 1 FROM apps WHERE bundle_identifier = 'com.github.macadmins.Nudge';
  platform: darwin
  install_software:
    fleet_maintained_app_slug: nudge/darwin
```

Add a second policy for the LaunchAgent with `install_software.package_path` pointing to `nudge-launchagent.yml`.

## Verify

1. In Fleet, open a Mac's **Host details** page.
2. On the **Software** tab, confirm Nudge and the LaunchAgent package show as **Installed**.
3. On the **OS settings** list, confirm the Nudge profile shows as **Verified**.
4. On the host, run this in Terminal as the logged-in user to confirm the LaunchAgent is loaded:

```bash
launchctl list | grep com.github.macadmins.Nudge
```

## Troubleshoot

**Nudge never appears**

Nudge stays hidden on hosts that already meet `requiredMinimumOSVersion`. Check that the host is below the required version and that the profile is **Verified**. Nudge writes its logs to the unified log. To stream them, run:

```bash
log stream --predicate 'subsystem == "com.github.macadmins.Nudge"' --info --style json --debug
```

**LaunchAgent package install fails**

Check the post-install script output on the host's **Activity** tab. If the plist isn't found at `/Library/LaunchAgents/com.github.macadmins.Nudge.plist`, confirm you uploaded the Nudge LaunchAgent package, not the Nudge app package.

## Further reading

- [Enforce OS updates](https://fleetdm.com/guides/enforce-os-updates)
- [Nudge wiki](https://github.com/macadmins/nudge/wiki)
- [Fleet's own Nudge setup](https://github.com/fleetdm/fleet/tree/main/it-and-security)

<meta name="articleTitle" value="Enforce OS updates on macOS with Nudge">
<meta name="authorFullName" value="Lucas Manuel Rodriguez">
<meta name="authorGitHubUsername" value="lucasmrod">
<meta name="publishedOn" value="2026-10-08">
<meta name="category" value="guides">
<meta name="description" value="Deploy Nudge with Fleet to remind end users to update macOS, with custom messaging, deferrals, and deadlines.">
