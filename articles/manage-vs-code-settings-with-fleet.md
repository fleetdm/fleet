# Manage VS Code settings with Fleet

VS Code reads enterprise policies that override what users set in their own `settings.json`. This guide shows how to deploy those policies with Fleet, using the Fleet UI or GitOps, to control which extensions and MCP servers your users can run. It also shows how to review the extensions and MCP servers already on your hosts, so you know what to allow.

It covers VS Code on macOS, Windows, and Linux. It doesn't cover forks like Cursor or Windsurf, or GitHub Copilot's enterprise-managed settings.

## Prerequisites

- VS Code 1.132 or later on your hosts, for the MCP server policies in this guide
- fleetd 1.59.0 or later, which adds the `ai_tools` table
- macOS hosts enrolled in Fleet MDM, to deploy the profile
- Scripts enabled on your Windows and Linux hosts (see [Enable scripts](https://fleetdm.com/guides/scripts#enable-scripts))
- Optional, for GitOps: a repository created with `fleetctl new` ([install fleetctl](https://fleetdm.com/guides/fleetctl#installing-fleetctl) first)
- Optional, to fix drift automatically: Fleet Premium, because script policy automations are a Premium feature

## Choose your policies

Each VS Code policy has a name, like `AllowedExtensions`, that maps to a setting, like `extensions.allowed`. Use the policy name on every OS. See [VS Code's policy reference](https://code.visualstudio.com/docs/enterprise/policies) for the full list.

The examples below set four policies:

- `AllowedExtensions` allows every extension from Microsoft and GitHub, plus Prettier. VS Code blocks everything else and disables blocked extensions that are already installed.
- `ChatAllowedMcpServers` allows only the MCP servers you list. Match each server with `serverUrl` or `serverCommand`. A `serverName` is only the label a user chose, so it isn't a security control.
- `ChatAllowManagedMcpServersOnly` makes VS Code ignore users' own MCP allowlists.
- `ChatToolsAutoApprove` set to `false` stops users from turning on global auto-approve for agent tools.

Other common policies include `UpdateMode` (`default`, `start`, `manual`, or `none`) and `TelemetryLevel` (`all`, `error`, `crash`, or `off`).

> **Warning:** Deploying `AllowedExtensions` blocks every extension that doesn't match it, including ones your users rely on today. Review installed extensions first (see [Review extensions and MCP servers](#review-extensions-and-mcp-servers)) and roll out to a test fleet before your whole organization.

## Write the policy for each OS

### macOS

VS Code reads policy from a configuration profile with the `com.microsoft.VSCode` payload type. A sample with every policy ships in `/Applications/Visual Studio Code.app/Contents/Resources/app/policies/`. Don't deploy the sample as is. Keep only the policies you want to enforce.

Save this as `vscode-settings.mobileconfig`. Policies that take an object or a list, like `AllowedExtensions`, go in a `<string>` as JSON:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>PayloadType</key>
      <string>com.microsoft.VSCode</string>
      <key>PayloadIdentifier</key>
      <string>com.microsoft.VSCode.managed</string>
      <key>PayloadUUID</key>
      <string>3F2A9C1E-6B7D-4E85-9A0C-5D1B8E4F7A26</string>
      <key>PayloadVersion</key>
      <integer>1</integer>
      <key>AllowedExtensions</key>
      <string>{"microsoft": true, "github": true, "esbenp.prettier-vscode": true}</string>
      <key>ChatAllowedMcpServers</key>
      <string>[{"serverUrl": "https://api.githubcopilot.com/*"}]</string>
      <key>ChatAllowManagedMcpServersOnly</key>
      <true/>
      <key>ChatToolsAutoApprove</key>
      <false/>
    </dict>
  </array>
  <key>PayloadDisplayName</key>
  <string>VS Code settings</string>
  <key>PayloadIdentifier</key>
  <string>com.example.vscode-settings</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>8D4E7B2A-1C9F-4A63-B5E0-27F6C3D9A814</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
</dict>
</plist>
```

Generate new UUIDs for both `PayloadUUID` values and change `com.example` to your own identifier.

### Windows

VS Code reads policy from `HKLM\SOFTWARE\Policies\Microsoft\VSCode`. On/off policies are `DWORD` values. Policies that take an object or a list are `REG_MULTI_SZ` values holding JSON, which is what VS Code's Group Policy templates write.

Save this as `vscode-settings.ps1`:

```powershell
$ErrorActionPreference = 'Stop'

$key = 'HKLM:\SOFTWARE\Policies\Microsoft\VSCode'
$allowedExtensions = '{"microsoft": true, "github": true, "esbenp.prettier-vscode": true}'
$allowedMcpServers = '[{"serverUrl": "https://api.githubcopilot.com/*"}]'

# Stop here if the JSON is invalid, so a bad policy never reaches a host.
foreach ($json in $allowedExtensions, $allowedMcpServers) { $null = $json | ConvertFrom-Json }

if (-not (Test-Path $key)) { New-Item -Path $key -Force | Out-Null }
New-ItemProperty -Path $key -Name 'AllowedExtensions' -PropertyType MultiString -Value @($allowedExtensions) -Force | Out-Null
New-ItemProperty -Path $key -Name 'ChatAllowedMcpServers' -PropertyType MultiString -Value @($allowedMcpServers) -Force | Out-Null
New-ItemProperty -Path $key -Name 'ChatAllowManagedMcpServersOnly' -PropertyType DWord -Value 1 -Force | Out-Null
New-ItemProperty -Path $key -Name 'ChatToolsAutoApprove' -PropertyType DWord -Value 0 -Force | Out-Null
```

### Linux

VS Code reads policy from `/etc/vscode/policy.json`. The keys are policy names, not setting IDs. Policies that take an object or a list go in as JSON strings with escaped quotes, the same as on macOS and Windows. VS Code also accepts plain JSON objects here, but strings keep the file usable by Devin Desktop, which reads it too.

> **Note:** Devin Desktop (formerly Windsurf) reads this same file, so every policy in it applies to both editors. If you manage both, keep all their Linux policies in this one script. See [Manage Windsurf (Devin Desktop) settings with Fleet](https://fleetdm.com/guides/manage-windsurf-settings-with-fleet).

Save this as `vscode-settings.sh`:

```bash
#!/bin/bash
set -euo pipefail

dir="/etc/vscode"
mkdir -p "$dir"
tmp="$(mktemp "$dir/.policy.XXXXXX")"
trap 'rm -f "$tmp"' EXIT

cat > "$tmp" <<'JSON'
{
  "AllowedExtensions": "{\"microsoft\": true, \"github\": true, \"esbenp.prettier-vscode\": true}",
  "ChatAllowedMcpServers": "[{\"serverUrl\": \"https://api.githubcopilot.com/*\"}]",
  "ChatAllowManagedMcpServersOnly": true,
  "ChatToolsAutoApprove": false
}
JSON

chmod 644 "$tmp"
chown root:root "$tmp"
mv "$tmp" "$dir/policy.json"
```

> **Warning:** VS Code skips a policy whose value has a syntax error. Check your JSON with `jq . policy.json`, and check each string value with `jq -r .AllowedExtensions policy.json | jq .`, before you upload the script.

## Deploy with Fleet

Deploy the same files with the Fleet UI or with GitOps.

### Fleet UI

1. Go to **Controls > OS settings > Configuration profiles**, select your fleet, select **Add profile**, and upload `vscode-settings.mobileconfig`.
2. Go to **Controls > Scripts** and upload `vscode-settings.ps1` and `vscode-settings.sh`.
3. Go to **Hosts** and open a Windows or Linux test host. Select **Actions > Run script** and pick the script for that host's platform.

To run a script on many hosts at once, see [batch script execution](https://fleetdm.com/guides/scripts#batch-execute-scripts).

### GitOps

1. Save the profile in `platforms/macos/configuration-profiles/`, where `fleets/workstations.yml` from `fleetctl new` already picks up `.mobileconfig` files.
2. Save the scripts in `lib/vscode/` and add them to the existing `controls.scripts` list in your fleet file, for example `fleets/workstations.yml`:

   ```yaml
   controls:
     scripts:
       - path: ../lib/vscode/vscode-settings.ps1
       - path: ../lib/vscode/vscode-settings.sh
   ```

3. Commit your changes and open a pull request. Your CI runs a dry run. After you merge, `fleetctl gitops` uploads the profile and the scripts.

Fleet sends the macOS profile to hosts automatically. Scripts run only when you trigger them, so add the policies below to keep Windows and Linux hosts in line.

## Keep hosts in line with your policy

Each policy below passes when the host has your settings. A host that fails gets the script automatically.

1. Run `vscode-settings.sh` on a Linux test host, then run `sha256sum /etc/vscode/policy.json` and copy the hash.
2. Add the policies to the existing `policies` list in your fleet file, replacing `<sha256>` with that hash:

   ```yaml
   policies:
     - name: VS Code policies applied (Windows)
       platform: windows
       description: Checks that the VS Code policies from Git are in the registry.
       resolution: Fleet reapplies the policies automatically. To do it yourself, run vscode-settings.ps1 from Fleet.
       query: "SELECT COUNT(*) AS applied FROM registry WHERE key = 'HKEY_LOCAL_MACHINE\\SOFTWARE\\Policies\\Microsoft\\VSCode' AND name IN ('AllowedExtensions', 'ChatAllowedMcpServers', 'ChatAllowManagedMcpServersOnly', 'ChatToolsAutoApprove') HAVING applied = 4;"
       run_script:
         path: ../lib/vscode/vscode-settings.ps1
     - name: VS Code policies applied (Linux)
       platform: linux
       description: Checks that the VS Code policy file matches the policy in Git.
       resolution: Fleet redeploys the file automatically. To do it yourself, run vscode-settings.sh from Fleet.
       query: "SELECT 1 FROM hash WHERE path = '/etc/vscode/policy.json' AND sha256 = '<sha256>';"
       run_script:
         path: ../lib/vscode/vscode-settings.sh
   ```

To do the same in the Fleet UI, go to **Policies**, select your fleet, and select **Add policy** for each query. Then select **Manage automations > Run script**, check each policy, and pick its script.

> **Note:** The Windows policy checks that each value exists, not what it holds. When you change a value in `vscode-settings.ps1`, run the script on your Windows hosts with batch execution. On Linux, update `<sha256>` at the same time as the script, and hosts pick up the new file on their own. Fleet runs the script when a policy newly fails, and makes up to three attempts in total if the script fails. See [Automatically run scripts](https://fleetdm.com/guides/policy-automation-run-script).

## What your users see

Fleet applies the profile and runs the scripts in the background, so users see no prompt. What changes for them:

- In the Settings editor, a managed setting shows that your organization manages it, and users can't change it. A managed policy wins over the same setting in a user's or workspace's `settings.json`.
- An extension that isn't on your allowlist can't be installed, and one that's already installed is disabled.
- An MCP server that isn't on your allowlist can't be installed or run.
- VS Code applies policy to new VS Code instances right away. Users with VS Code already open may need to restart it.

Tell your users before you roll out a new restriction, so a disabled extension doesn't come as a surprise.

## Verify

1. On a test host, quit and reopen VS Code.
2. Open the Command Palette and run **Developer: Policy Diagnostics**. Confirm your policies are in the **Applied Policy** table.
3. Try to install an extension from a publisher you didn't allow. Confirm VS Code blocks it.
4. In Fleet, open the host's details page and confirm the macOS profile shows as **Verified** or the Windows or Linux policy passes.

## Review extensions and MCP servers

Your allowlists only work if they include what your users need. Two tables show what's already on your hosts:

- `vscode_extensions` lists every installed extension, with `name` set to the extension ID you use in `AllowedExtensions`.
- `ai_tools` lists the MCP servers configured in VS Code, with their risk flags.

Add these reports to `platforms/all/reports/` (GitOps), or go to **Reports > Add report** in the Fleet UI:

```yaml
- name: VS Code extensions
  description: Lists VS Code extensions installed on each host.
  query: "SELECT u.username, e.name, e.publisher, e.version FROM users u CROSS JOIN vscode_extensions e USING (uid) WHERE e.vscode_edition = 'vscode';"
  platform: darwin,windows,linux
  interval: 86400
  observer_can_run: false
  automations_enabled: false
- name: VS Code MCP servers
  description: Lists MCP servers configured in VS Code, with their risk flags.
  query: "SELECT username, name, location, endpoint, risk_flags, path FROM ai_tools WHERE type = 'mcp_server' AND source = 'vscode';"
  platform: darwin,windows,linux
  interval: 3600
  observer_can_run: false
  automations_enabled: false
```

Then review the results on a schedule:

1. Open each report and look for extensions and MCP servers you haven't decided on. For MCP servers, check `endpoint` and `risk_flags` first.
2. Add an extension you approve to `AllowedExtensions`, by extension ID or by publisher.
3. Add an MCP server you approve to `ChatAllowedMcpServers`, using its `serverUrl` or `serverCommand`. To block a server outright, add it to `ChatDeniedMcpServers`, which wins over the allowlist.
4. Update the profile and scripts, and the Linux hash, then open a pull request.

## Troubleshoot

**A policy is in the Non-applied Policy table**

The policy name is misspelled, or `policy.json` uses a setting ID like `extensions.allowed` instead of the policy name `AllowedExtensions`. Fix the name and redeploy.

**A policy doesn't apply and isn't listed as applied**

Its value probably has a JSON syntax error, so VS Code skipped it. Run **Developer: Show Logs** and pick **Window** to see the parse error.

**A user's extension stopped working after rollout**

It isn't on your `AllowedExtensions` list. Check the VS Code extensions report, add the extension ID or publisher, and redeploy.

**The Linux policy fails on every check, and Fleet keeps rerunning the script**

Two scripts write `/etc/vscode/policy.json`, for example this guide's and the one in the Windsurf (Devin Desktop) guide. Each overwrites the other, so one hash policy always fails. Keep one script with every policy for both editors, and delete the other script and its policy.

**The Windows policy fails on a host that ran the script**

Run the script again from the host's details page and check the output. The policy fails until all four values exist under `HKLM\SOFTWARE\Policies\Microsoft\VSCode`.

## Further reading

- [VS Code policy reference](https://code.visualstudio.com/docs/enterprise/policies)
- [Manage extensions in enterprise environments](https://code.visualstudio.com/docs/enterprise/extensions)
- [Manage AI settings in VS Code](https://code.visualstudio.com/docs/enterprise/manage-ai-settings)
- [Manage Claude Code and Claude Desktop settings with Fleet](https://fleetdm.com/guides/manage-claude-code-and-claude-desktop-settings)
- [Automatically run scripts](https://fleetdm.com/guides/policy-automation-run-script)

<meta name="articleTitle" value="Manage VS Code settings with Fleet">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="publishedOn" value="2026-11-23">
<meta name="category" value="guides">
<meta name="description" value="Deploy VS Code policies with Fleet to control extensions and MCP servers, keep hosts in line, and review what users run.">
