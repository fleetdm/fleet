# Manage Windsurf (Devin Desktop) settings with Fleet

Windsurf is now called Devin Desktop. It reads enterprise policies that override what users set in their own `settings.json`. This guide shows how to deploy those policies with Fleet, using the Fleet UI or GitOps, to limit which extensions your users can install. It also shows how to review the extensions and MCP servers already on your hosts.

It covers Devin Desktop on macOS, Windows, and Linux. It doesn't cover team settings you manage in the Devin app or the Windsurf dashboard. That includes the MCP allowlist, which is a team setting and has no device policy.

## Prerequisites

- Devin Desktop 3.0 or later on your hosts (the first release after the rename from Windsurf)
- fleetd 1.59.0 or later, which adds the `ai_tools` table
- osquery 5.24.0 or later, so the `vscode_extensions` table finds extensions in Devin Desktop's new extensions folder
- macOS hosts enrolled in Fleet MDM, to deploy the profile
- Scripts enabled on your Windows and Linux hosts (see [Enable scripts](https://fleetdm.com/guides/scripts#enable-scripts))
- Optional, for GitOps: a repository created with `fleetctl new` ([install fleetctl](https://fleetdm.com/guides/fleetctl#installing-fleetctl) first)
- Optional, to fix drift automatically: Fleet Premium, because script policy automations are a Premium feature

## Choose your policies

Devin Desktop is built on VS Code and uses the same policy names, like `AllowedExtensions`. The examples below set two:

- `AllowedExtensions` allows every extension from GitHub and Microsoft's Python publisher, plus Prettier. Devin Desktop blocks the rest. Keys are publishers or extension IDs.
- `TelemetryLevel` set to `error` limits the telemetry Devin Desktop sends. The other values are `all`, `crash`, and `off`.

Other policies include `UpdateMode` (`none`, `manual`, `start`, or `default`) and `ExtensionGalleryServiceUrl`, which points Devin Desktop at your own extension marketplace. See [Devin's enterprise policies reference](https://docs.devin.ai/desktop/enterprise-policies) for details.

> **Note:** The **Extension policy** team setting replaces `AllowedExtensions` on a host when both are set. If your organization uses that team setting, manage the allowlist there instead.

> **Warning:** Deploying `AllowedExtensions` blocks every extension that doesn't match it, including ones your users rely on today. Review installed extensions first (see [Review extensions and MCP servers](#review-extensions-and-mcp-servers)) and roll out to a test fleet before your whole organization.

## Write the policy for each OS

### macOS

Devin Desktop reads policy from a configuration profile with the `com.exafunction.windsurf` payload type. The bundle ID didn't change with the rename. A sample with every policy ships at `/Applications/Devin.app/Contents/Resources/app/policies/com.exafunction.windsurf.mobileconfig`. Don't deploy the sample as is. Keep only the policies you want to enforce.

Save this as `windsurf-settings.mobileconfig`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>PayloadType</key>
      <string>com.exafunction.windsurf</string>
      <key>PayloadIdentifier</key>
      <string>com.exafunction.windsurf.managed</string>
      <key>PayloadUUID</key>
      <string>1E2AED2F-AF95-45A3-BF4C-E117C296ABEF</string>
      <key>PayloadVersion</key>
      <integer>1</integer>
      <key>AllowedExtensions</key>
      <string>{"github": true, "ms-python": true, "esbenp.prettier-vscode": true}</string>
      <key>TelemetryLevel</key>
      <string>error</string>
    </dict>
  </array>
  <key>PayloadDisplayName</key>
  <string>Devin Desktop settings</string>
  <key>PayloadIdentifier</key>
  <string>com.example.windsurf-settings</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>CEE9E1E8-14DD-4E57-84D4-ECE3D00D958F</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
</dict>
</plist>
```

Generate new UUIDs for both `PayloadUUID` values and change `com.example` to your own identifier.

### Windows

Devin Desktop reads policy from `HKLM\SOFTWARE\Policies\Windsurf\Devin`. This isn't the `Microsoft` key VS Code uses. Both policies in this example are strings.

Save this as `windsurf-settings.ps1`:

```powershell
$ErrorActionPreference = 'Stop'

$key = 'HKLM:\SOFTWARE\Policies\Windsurf\Devin'
$allowedExtensions = '{"github": true, "ms-python": true, "esbenp.prettier-vscode": true}'

# Stop here if the JSON is invalid, so a bad policy never reaches a host.
$null = $allowedExtensions | ConvertFrom-Json

if (-not (Test-Path $key)) { New-Item -Path $key -Force | Out-Null }
New-ItemProperty -Path $key -Name 'AllowedExtensions' -PropertyType String -Value $allowedExtensions -Force | Out-Null
New-ItemProperty -Path $key -Name 'TelemetryLevel' -PropertyType String -Value 'error' -Force | Out-Null
```

> **Note:** Windsurf releases from before the rename read `HKLM\SOFTWARE\Policies\Windsurf\Windsurf` instead. Update those hosts to Devin Desktop, or also write the values to that key.

### Linux

Devin Desktop reads policy from `/etc/vscode/policy.json`, the same file VS Code reads. `AllowedExtensions` is a JSON string with escaped quotes, as in Devin's docs.

> **Warning:** Every policy in this file applies to both VS Code and Devin Desktop. If you also [manage VS Code settings with Fleet](https://fleetdm.com/guides/manage-vs-code-settings-with-fleet), don't deploy `windsurf-settings.sh`. Add `TelemetryLevel` and any extensions Devin Desktop users need to `vscode-settings.sh` from that guide, and skip the Linux policy below. Two scripts that write the same file overwrite each other, so their hash policies never both pass.

Save this as `windsurf-settings.sh`:

```bash
#!/bin/bash
set -euo pipefail

dir="/etc/vscode"
mkdir -p "$dir"
tmp="$(mktemp "$dir/.policy.XXXXXX")"
trap 'rm -f "$tmp"' EXIT

cat > "$tmp" <<'JSON'
{
  "AllowedExtensions": "{\"github\": true, \"ms-python\": true, \"esbenp.prettier-vscode\": true}",
  "TelemetryLevel": "error"
}
JSON

chmod 644 "$tmp"
chown root:root "$tmp"
mv "$tmp" "$dir/policy.json"
```

> **Warning:** Devin Desktop skips a policy whose value has a syntax error. Check your JSON with `jq . policy.json`, and check the `AllowedExtensions` string with `jq -r .AllowedExtensions policy.json | jq .`, before you upload the script.

## Deploy with Fleet

Deploy the same files with the Fleet UI or with GitOps.

### Fleet UI

1. Go to **Controls > OS settings > Configuration profiles**, select your fleet, select **Add profile**, and upload `windsurf-settings.mobileconfig`.
2. Go to **Controls > Scripts** and upload `windsurf-settings.ps1` and `windsurf-settings.sh`.
3. Go to **Hosts** and open a Windows or Linux test host. Select **Actions > Run script** and pick the script for that host's platform.

To run a script on many hosts at once, see [batch script execution](https://fleetdm.com/guides/scripts#batch-execute-scripts).

### GitOps

1. Save the profile in `platforms/macos/configuration-profiles/`, where the default fleet file already picks up `.mobileconfig` files.
2. Save the scripts in `lib/windsurf/` and add them to your fleet file, for example `fleets/workstations.yml`:

   ```yaml
   controls:
     scripts:
       - path: ../lib/windsurf/windsurf-settings.ps1
       - path: ../lib/windsurf/windsurf-settings.sh
   ```

3. Commit your changes and open a pull request. Your CI runs a dry run. After you merge, `fleetctl gitops` uploads the profile and the scripts.

Fleet sends the macOS profile to hosts automatically. Scripts run only when you trigger them, so add the policies below to keep Windows and Linux hosts in line.

## Keep hosts in line with your policy

Each policy below passes when the host has your settings. A host that fails gets the script automatically.

1. Run `windsurf-settings.sh` on a Linux test host, then run `sha256sum /etc/vscode/policy.json` and copy the hash.
2. Add the policies to your fleet file, replacing `<sha256>` with that hash:

   ```yaml
   policies:
     - name: Devin Desktop policies applied (Windows)
       platform: windows
       description: Checks that the Devin Desktop policies from Git are in the registry.
       resolution: Fleet reapplies the policies automatically. To do it yourself, run windsurf-settings.ps1 from Fleet.
       query: "SELECT COUNT(*) AS applied FROM registry WHERE key = 'HKEY_LOCAL_MACHINE\\SOFTWARE\\Policies\\Windsurf\\Devin' AND name IN ('AllowedExtensions', 'TelemetryLevel') HAVING applied = 2;"
       run_script:
         path: ../lib/windsurf/windsurf-settings.ps1
     - name: Devin Desktop policies applied (Linux)
       platform: linux
       description: Checks that the Devin Desktop policy file matches the policy in Git.
       resolution: Fleet redeploys the file automatically. To do it yourself, run windsurf-settings.sh from Fleet.
       query: "SELECT 1 FROM hash WHERE path = '/etc/vscode/policy.json' AND sha256 = '<sha256>';"
       run_script:
         path: ../lib/windsurf/windsurf-settings.sh
   ```

To do the same in the Fleet UI, go to **Policies**, select your fleet, and select **Add policy** for each query. Then select **Manage automations > Run script**, check each policy, and pick its script.

> **Note:** The Windows policy checks that each value exists, not what it holds. When you change a value in `windsurf-settings.ps1`, run the script on your Windows hosts with batch execution. On Linux, update `<sha256>` at the same time as the script, and hosts pick up the new file on their own. Fleet runs the script when a policy newly fails and retries it up to three times. See [Automatically run scripts](https://fleetdm.com/guides/policy-automation-run-script).

## What your users see

Fleet applies the profile and runs the scripts in the background, so users see no prompt. What changes for them:

- In the Settings editor, a managed setting shows that your organization manages it, and users can't change it. A policy wins over the same setting in a user's or workspace's `settings.json`.
- The Extensions view shows that the allowlist is managed by your organization, and an extension that isn't on it can't be installed.
- Devin Desktop applies policy when it starts. Users with Devin Desktop already open may need to restart it.

Tell your users before you roll out a new restriction, so a blocked extension doesn't come as a surprise.

## Verify

1. On a test host, quit and reopen Devin Desktop.
2. Open the Command Palette and run **Developer: Policy Diagnostics**. Confirm your policies are in the **Applied Policy** table.
3. Try to install an extension from a publisher you didn't allow. Confirm Devin Desktop blocks it.
4. In Fleet, open the host's details page and confirm the macOS profile shows as **Verified** or the Windows or Linux policy passes.

## Review extensions and MCP servers

Your allowlist only works if it includes what your users need. Three tables show what's already on your hosts:

- `vscode_extensions` lists installed extensions, with `name` set to the extension ID you use in `AllowedExtensions`. Devin Desktop's extensions report `vscode_edition = 'windsurf'`, from both `~/.devin/extensions` and the legacy `~/.windsurf/extensions`.
- `ai_tools` lists the MCP servers configured for the Cascade agent in `~/.codeium/windsurf/mcp_config.json`, with their risk flags.
- `file` finds the MCP config for the Devin Local agent, which `ai_tools` doesn't read yet. It's at `~/.config/devin/mcp_config.json` on macOS and Linux, and `%APPDATA%\devin\mcp_config.json` on Windows.

Add these reports to `platforms/all/reports/` (GitOps), or go to **Reports > Add report** in the Fleet UI:

```yaml
- name: Devin Desktop extensions
  description: Lists Devin Desktop (Windsurf) extensions installed on each host.
  query: "SELECT u.username, e.name, e.publisher, e.version, e.path FROM users u CROSS JOIN vscode_extensions e USING (uid) WHERE e.vscode_edition = 'windsurf';"
  platform: darwin,windows,linux
  interval: 86400
  observer_can_run: false
  automations_enabled: false
- name: Devin Desktop Cascade MCP servers
  description: Lists MCP servers configured for Cascade, with their risk flags.
  query: "SELECT username, name, location, endpoint, risk_flags, path FROM ai_tools WHERE type = 'mcp_server' AND source = 'windsurf';"
  platform: darwin,windows,linux
  interval: 3600
  observer_can_run: false
  automations_enabled: false
- name: Devin Local MCP configs
  description: Lists Devin Local MCP config files, so you can inspect the hosts that have one.
  query: "SELECT path, size, datetime(mtime, 'unixepoch') AS modified FROM file WHERE path LIKE '/Users/%/.config/devin/mcp_config.json' OR path LIKE '/home/%/.config/devin/mcp_config.json' OR path LIKE 'C:\\Users\\%\\AppData\\Roaming\\devin\\mcp_config.json';"
  platform: darwin,windows,linux
  interval: 86400
  observer_can_run: false
  automations_enabled: false
```

Then review the results on a schedule:

1. Open the extensions report and look for extensions you haven't decided on. Add the ones you approve to `AllowedExtensions`, by extension ID or by publisher.
2. Open the MCP reports and look for servers you haven't approved. For Cascade servers, check `endpoint` and `risk_flags` first. For a Devin Local config, run a live report on that host: `SELECT line FROM file_lines WHERE path = '<path>';`.
3. Add the MCP servers you approve to the **MCP allowlist** team setting. Once it lists one server, Devin Desktop blocks every server that isn't on it.
4. Update the profile and scripts, and the Linux hash, then open a pull request.

## Troubleshoot

**A policy isn't in the Applied Policy table**

Its value probably has a JSON syntax error, so Devin Desktop skipped it. Open the Command Palette, run **Show Window Log**, and look for the parse error.

**A policy doesn't apply on Windows**

Check that the values are under `HKLM\SOFTWARE\Policies\Windsurf\Devin`, not `HKLM\SOFTWARE\Policies\Microsoft\VSCode`. On a host still running Windsurf from before the rename, the key is `HKLM\SOFTWARE\Policies\Windsurf\Windsurf`.

**The allowlist doesn't match what you deployed**

The **Extension policy** team setting is set, and it replaces `AllowedExtensions` on the host. Clear the team setting or manage the allowlist there.

**The Linux policy fails on every check, and Fleet keeps rerunning the script**

Two scripts write `/etc/vscode/policy.json`, for example this guide's and the one in the VS Code guide. Each overwrites the other, so one hash policy always fails. Keep one script with every policy for both editors, and delete the other script and its policy.

**The extensions report is missing extensions on some hosts**

Check the host's osquery version. Versions before 5.24.0 don't read `~/.devin/extensions`, so they only report extensions installed before the rename.

## Further reading

- [Devin Desktop enterprise policies](https://docs.devin.ai/desktop/enterprise-policies)
- [Devin Desktop admin controls reference](https://docs.devin.ai/desktop/controls): team settings, including the MCP allowlist, and system-level files
- [Manage VS Code settings with Fleet](https://fleetdm.com/guides/manage-vs-code-settings-with-fleet)
- [Manage Cursor settings with Fleet](https://fleetdm.com/guides/manage-cursor-settings-with-fleet)
- [Manage Claude Code and Claude Desktop settings with Fleet](https://fleetdm.com/guides/manage-claude-code-and-claude-desktop-settings)
- [Automatically run scripts](https://fleetdm.com/guides/policy-automation-run-script)

<meta name="articleTitle" value="Manage Windsurf (Devin Desktop) settings with Fleet">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="publishedOn" value="2026-11-23">
<meta name="category" value="guides">
<meta name="description" value="Deploy Windsurf (Devin Desktop) policies with Fleet to control extensions, keep hosts in line, and review the extensions and MCP servers users run.">
