# Enforce OS updates on macOS 13 hosts with Nudge

Fleet enforces macOS updates with Apple's declarative device management (DDM), which requires macOS 14 or later. An upcoming Fleet release removes fleetd's built-in [Nudge](https://github.com/macadmins/nudge) for macOS 13 and older hosts ([#55136](https://github.com/fleetdm/fleet/issues/55136)). If you still have hosts on macOS 13, this guide shows how to deploy Nudge yourself with Fleet so those end users keep getting reminded to update. It's the same setup Fleet uses on its own computers.

## Prerequisites

- Fleet Premium. Label-scoped software and profiles are Premium features.
- Apple MDM turned on, with your macOS 13 hosts enrolled.
- The Nudge LaunchAgent package (`Nudge_LaunchAgent-<version>.pkg`), downloaded from the [Nudge releases page](https://github.com/macadmins/nudge/releases). The Nudge Fleet-maintained app installs only the app, so the LaunchAgent comes in a separate package.

> **Note:** Nudge only shows its window on hosts below the required version. You can deploy it to every Mac, but scoping it to macOS 13 hosts keeps it out of the way of DDM on macOS 14 and later.

## Step 1: Create a label for macOS 13 hosts

1. In Fleet, open the account menu in the top-right corner, select **Labels**, and then select **Add label**.
2. Select **Dynamic**.
3. Name the label `macOS 13 and older`, set the platform to macOS, and use this query:

```sql
SELECT 1 FROM os_version WHERE major <= 13;
```

If you use GitOps, add the label to your `labels` file instead:

```yaml
- name: macOS 13 and older
  description: macOS hosts that need Nudge for OS update reminders
  query: SELECT 1 FROM os_version WHERE major <= 13;
  label_membership_type: dynamic
  platform: darwin
```

## Step 2: Add the Nudge app

Add Nudge as a [Fleet-maintained app](https://fleetdm.com/guides/fleet-maintained-apps) and scope it to the label.

1. Go to **Software**, select the fleet, and select **Add software > Fleet-maintained**.
2. Find **Nudge** and select it.
3. Under **Target**, select **Custom** and choose the `macOS 13 and older` label.
4. Select **Add software**.

In GitOps, add this to the fleet's YAML file:

```yaml
software:
  fleet_maintained_apps:
    - slug: nudge/darwin
      labels_include_any:
        - macOS 13 and older
```

## Step 3: Add the Nudge LaunchAgent

The LaunchAgent opens Nudge on a schedule, so end users see the reminder without fleetd involved.

1. Save this post-install script as `nudge-postinstall.sh`. It loads the LaunchAgent right after install, without waiting for the next login.

```bash
#!/bin/bash

PLIST_PATH="/Library/LaunchAgents/com.github.macadmins.Nudge.plist"
LABEL="com.github.macadmins.Nudge"

if [[ ! -f "$PLIST_PATH" ]]; then
    echo "Error: LaunchAgent plist not found at $PLIST_PATH"
    exit 1
fi

/usr/sbin/chown root:wheel "$PLIST_PATH"
/bin/chmod 644 "$PLIST_PATH"

if /bin/launchctl list | /usr/bin/grep -q "$LABEL"; then
    /bin/launchctl unload "$PLIST_PATH" 2>/dev/null
fi

/bin/launchctl load "$PLIST_PATH" || exit 1
```

2. Go to **Software > Add software > Custom package** and upload the Nudge LaunchAgent package.
3. Under **Advanced options**, paste the script into **Post-install script**.
4. Under **Target**, select **Custom** and choose the `macOS 13 and older` label.
5. Select **Add software**.

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
      labels_include_any:
        - macOS 13 and older
```

## Step 4: Configure Nudge with a profile

Nudge reads its settings from a configuration profile. The example below requires macOS 13 hosts to upgrade to the latest macOS version their hardware supports by the deadline. It also lets Nudge run as a background task without end users turning it off.

1. Save the profile as `nudge-macos-13.mobileconfig`. Replace each `REPLACE-WITH-UUID` with a unique value from `uuidgen`, and set `requiredInstallationDate` to your deadline.

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>PayloadDisplayName</key>
			<string>Nudge Preferences</string>
			<key>PayloadIdentifier</key>
			<string>com.example.nudge.preferences</string>
			<key>PayloadType</key>
			<string>com.github.macadmins.Nudge</string>
			<key>PayloadUUID</key>
			<string>REPLACE-WITH-UUID</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
			<key>optionalFeatures</key>
			<dict>
				<key>attemptToFetchMajorUpgrade</key>
				<true/>
				<key>utilizeSOFAFeed</key>
				<true/>
			</dict>
			<key>osVersionRequirements</key>
			<array>
				<dict>
					<key>requiredInstallationDate</key>
					<string>2026-12-31T12:00:00</string>
					<key>requiredMinimumOSVersion</key>
					<string>latest-supported</string>
					<key>targetedOSVersionsRule</key>
					<string>13</string>
				</dict>
			</array>
			<key>userInterface</key>
			<dict>
				<key>simpleMode</key>
				<true/>
			</dict>
		</dict>
		<dict>
			<key>PayloadDisplayName</key>
			<string>Allow Nudge background tasks</string>
			<key>PayloadIdentifier</key>
			<string>com.example.nudge.servicemanagement</string>
			<key>PayloadType</key>
			<string>com.apple.servicemanagement</string>
			<key>PayloadUUID</key>
			<string>REPLACE-WITH-UUID</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
			<key>Rules</key>
			<array>
				<dict>
					<key>RuleType</key>
					<string>BundleIdentifier</string>
					<key>RuleValue</key>
					<string>com.github.macadmins.Nudge</string>
				</dict>
			</array>
		</dict>
	</array>
	<key>PayloadDisplayName</key>
	<string>Nudge settings</string>
	<key>PayloadIdentifier</key>
	<string>com.example.nudge</string>
	<key>PayloadScope</key>
	<string>System</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>REPLACE-WITH-UUID</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>
```

2. Go to **Controls > OS settings > Configuration profiles** and select **Add profile**.
3. Upload `nudge-macos-13.mobileconfig`, select **Custom** under **Target**, and choose the `macOS 13 and older` label.

In GitOps, add the profile to the fleet's YAML file:

```yaml
controls:
  apple_settings:
    configuration_profiles:
      - path: ../lib/macos/configuration-profiles/nudge-macos-13.mobileconfig
        labels_include_any:
          - macOS 13 and older
```

To customize the reminder text, deferrals, and how often Nudge appears, see the [Nudge wiki](https://github.com/macadmins/nudge/wiki). Fleet's own [Nudge profile](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/macos/configuration-profiles/nudge-configuration.mobileconfig) is a fuller example.

## Step 5: Optional: Reinstall Nudge if it's removed

Add policies that install Nudge and the LaunchAgent automatically on hosts that are missing them:

```yaml
- name: Nudge installed
  query: SELECT 1 FROM apps WHERE bundle_identifier = 'com.github.macadmins.Nudge';
  platform: darwin
  labels_include_any:
    - macOS 13 and older
  install_software:
    fleet_maintained_app_slug: nudge/darwin
```

Add a second policy for the LaunchAgent with `install_software.package_path` pointing to `nudge-launchagent.yml`.

## Verify

1. In Fleet, open a macOS 13 host's **Host details** page.
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

<meta name="articleTitle" value="Enforce OS updates on macOS 13 hosts with Nudge">
<meta name="authorFullName" value="Lucas Manuel Rodriguez">
<meta name="authorGitHubUsername" value="lucasmrod">
<meta name="publishedOn" value="2026-10-08">
<meta name="category" value="guides">
<meta name="description" value="Deploy Nudge with Fleet to keep reminding macOS 13 users to update after fleetd's built-in Nudge is removed.">
