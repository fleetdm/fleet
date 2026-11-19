# Manage GitHub Copilot settings with Fleet

GitHub Copilot reads enterprise-managed settings from the device, and users can't override them in their own settings. GitHub lists Copilot CLI, VS Code, the GitHub Copilot app, and JetBrains IDEs as supported clients, though not every client supports every key. This guide shows how to deploy those settings with Fleet, using the Fleet UI or GitOps, to block bypass (allow-all) mode and limit the MCP servers Copilot can run. It also shows how to review where Copilot runs and which MCP servers it calls.

It covers macOS, Windows, and Linux. VS Code's own policies, like `AllowedExtensions` and `ChatAllowedMcpServers`, are in [Manage VS Code settings with Fleet](https://fleetdm.com/guides/manage-vs-code-settings-with-fleet). This guide doesn't cover server-managed settings in your enterprise's `.github-private` repository, or Copilot policies you set on GitHub.com.

## Prerequisites

- Copilot clients that support the keys you set. Check the client support table in GitHub's [enterprise-managed settings reference](https://docs.github.com/en/copilot/reference/enterprise-administrators/enterprise-managed-settings).
- fleetd 1.61.0 or later, which detects the Copilot extension that ships inside VS Code (fleetd 1.59.0 added the `ai_tools` table)
- Optional, to review Copilot CLI sessions: fleetd 1.64.0 or later, which adds the `ai_agent_session_messages` table
- macOS hosts enrolled in Fleet MDM, to deploy the profile
- Scripts enabled on your Windows and Linux hosts (see [Enable scripts](https://fleetdm.com/guides/scripts#enable-scripts))
- Optional, for GitOps: a repository created with `fleetctl new` ([install fleetctl](https://fleetdm.com/guides/fleetctl#installing-fleetctl) first)
- Optional, to fix drift automatically: Fleet Premium, because script policy automations are a Premium feature

## Choose your settings

Copilot accepts managed settings through three delivery methods. This guide uses the two that live on the device:

| OS | Delivery method | Location |
| --- | --- | --- |
| macOS | MDM (configuration profile) | `com.github.copilot` preference domain |
| Windows | MDM (registry) | String values under `HKLM\SOFTWARE\Policies\GitHubCopilot` |
| Linux | File | `/etc/github-copilot/managed-settings.json` |

GitHub doesn't support MDM delivery on Linux, so Linux uses the file. Clients check MDM settings every hour without a restart. After a file change, GitHub says to restart clients so they load it.

The examples below set three keys:

- `permissions.disableBypassPermissionsMode` set to `disable` stops users from turning on bypass mode, which lets the agent run commands, edit files, and fetch URLs without asking.
- `allowedMcpServers` allows only the MCP servers you list. Match each server with `serverUrl` (remote servers, `*` wildcards allowed) or `serverCommand` (local servers, the exact command and arguments). A `serverName` is only the label a user chose, so it isn't a security control.
- `deniedMcpServers` blocks matching servers, even ones on the allowlist. Copilot's built-in GitHub MCP server can't be blocked.

Other keys control sandboxing, plugins, the default model, and OpenTelemetry export. See GitHub's reference for the full list.

> **Warning:** Setting `allowedMcpServers` blocks every MCP server that doesn't match it, including servers your users rely on today. Review what's in use first (see [Review Copilot on your hosts](#review-copilot-on-your-hosts)) and roll out to a test fleet before your whole organization.

### How these settings combine with others

- If your enterprise also delivers server-managed settings from `.github-private`, MDM settings win over server-managed settings, which win over the file. `permissions` rules and `sandbox` combine in the most restrictive direction. GitHub documents that MCP allowlists from different sources intersect and denylists add up. To keep results predictable, set each key in one place.
- If a Copilot managed setting and a VS Code policy control the same thing, like `allowedMcpServers` and `ChatAllowedMcpServers`, the Copilot setting wins. The values aren't merged.
- While any Copilot managed setting applies, VS Code turns off its Claude and Codex agent integrations, because they don't enforce Copilot's settings. To allow them, add `vscode.allowedHarnesses` (VS Code 1.147 or later) with `claude`, `codex`, or both.

## Write the settings for each OS

The file uses nested JSON. MDM uses the same keys flattened with dots, like `permissions.disableBypassPermissionsMode`, and every MDM value is a string. Lists and objects go in the string as JSON, so `vscode.allowedHarnesses` is `["claude"]` in a string on macOS and Windows, and `{"vscode": {"allowedHarnesses": ["claude"]}}` in the file.

### macOS

Save this as `copilot-settings.mobileconfig`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>PayloadType</key>
      <string>com.github.copilot</string>
      <key>PayloadIdentifier</key>
      <string>com.github.copilot.managed</string>
      <key>PayloadUUID</key>
      <string>0D2C6A4D-2125-4D0C-9403-CDBDB7D4A6B9</string>
      <key>PayloadVersion</key>
      <integer>1</integer>
      <key>permissions.disableBypassPermissionsMode</key>
      <string>disable</string>
      <key>allowedMcpServers</key>
      <string>[{"serverUrl": "https://api.githubcopilot.com/*"}, {"serverCommand": ["npx", "@playwright/mcp@latest"]}]</string>
      <key>deniedMcpServers</key>
      <string>[{"serverUrl": "https://*.untrusted.example.com/*"}]</string>
    </dict>
  </array>
  <key>PayloadDisplayName</key>
  <string>GitHub Copilot managed settings</string>
  <key>PayloadIdentifier</key>
  <string>com.example.copilot-settings</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>EA9A4E8A-9C57-4BC6-8009-84B63FD888E2</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
</dict>
</plist>
```

Generate new UUIDs for both `PayloadUUID` values and change `com.example` to your own identifier.

> **Warning:** Use `<string>` for every value, including booleans: `<string>true</string>`, not `<true/>`. Copilot CLI skips a key whose value isn't a string.

### Windows

Copilot reads string (`REG_SZ`) values from `HKLM\SOFTWARE\Policies\GitHubCopilot`. Don't use `DWORD` values, even for booleans.

Save this as `copilot-settings.ps1`:

```powershell
$ErrorActionPreference = 'Stop'

$key = 'HKLM:\SOFTWARE\Policies\GitHubCopilot'
$settings = [ordered]@{
  'permissions.disableBypassPermissionsMode' = 'disable'
  'allowedMcpServers' = '[{"serverUrl": "https://api.githubcopilot.com/*"}, {"serverCommand": ["npx", "@playwright/mcp@latest"]}]'
  'deniedMcpServers' = '[{"serverUrl": "https://*.untrusted.example.com/*"}]'
}

# Stop here if a list is invalid JSON, so a bad policy never reaches a host.
foreach ($name in 'allowedMcpServers', 'deniedMcpServers') { $null = $settings[$name] | ConvertFrom-Json }

if (-not (Test-Path $key)) { New-Item -Path $key -Force | Out-Null }
foreach ($name in $settings.Keys) {
  New-ItemProperty -Path $key -Name $name -PropertyType String -Value $settings[$name] -Force | Out-Null
}
```

### Linux

Copilot reads `/etc/github-copilot/managed-settings.json`. Copilot CLI rejects the file unless it's a regular file owned by root that isn't group-writable or world-writable, and it rejects a symbolic link.

Save this as `copilot-settings.sh`:

```bash
#!/bin/bash
set -euo pipefail

dir="/etc/github-copilot"
mkdir -p "$dir"
tmp="$(mktemp "$dir/.managed-settings.XXXXXX")"
trap 'rm -f "$tmp"' EXIT

cat > "$tmp" <<'JSON'
{
  "permissions": {
    "disableBypassPermissionsMode": "disable"
  },
  "allowedMcpServers": [
    { "serverUrl": "https://api.githubcopilot.com/*" },
    { "serverCommand": ["npx", "@playwright/mcp@latest"] }
  ],
  "deniedMcpServers": [
    { "serverUrl": "https://*.untrusted.example.com/*" }
  ]
}
JSON

chmod 644 "$tmp"
chown root:root "$tmp"
chmod 755 "$dir"
mv "$tmp" "$dir/managed-settings.json"
```

> **Warning:** Check your JSON with `jq . managed-settings.json` before you upload the script. Copilot can't apply a file it can't parse.

> **Note:** Copilot also reads the file on macOS (`/Library/Application Support/GitHubCopilot/managed-settings.json`) and Windows (`%ProgramFiles%\GitHubCopilot\managed-settings.json`). The file has the lowest precedence and needs a client restart, so use MDM on those platforms instead of deploying both.

## Deploy with Fleet

Deploy the same files with the Fleet UI or with GitOps.

### Fleet UI

1. Go to **Controls > OS settings > Configuration profiles**, select your fleet, select **Add profile**, and upload `copilot-settings.mobileconfig`.
2. Go to **Controls > Scripts** and upload `copilot-settings.ps1` and `copilot-settings.sh`.
3. Go to **Hosts** and open a Windows or Linux test host. Select **Actions > Run script** and pick the script for that host's platform.

To run a script on many hosts at once, see [batch script execution](https://fleetdm.com/guides/scripts#batch-execute-scripts).

### GitOps

1. Save the profile in `platforms/macos/configuration-profiles/`, where the default fleet file already picks up `.mobileconfig` files.
2. Save the scripts in `lib/copilot/` and add them to your fleet file, for example `fleets/workstations.yml`:

   ```yaml
   controls:
     scripts:
       - path: ../lib/copilot/copilot-settings.ps1
       - path: ../lib/copilot/copilot-settings.sh
   ```

3. Commit your changes and open a pull request. Your CI runs a dry run. After you merge, `fleetctl gitops` uploads the profile and the scripts.

Fleet sends the macOS profile to hosts automatically. Scripts run only when you trigger them, so add the policies below to keep Windows and Linux hosts in line.

## Keep hosts in line with your settings

Each policy below passes when the host has your settings. A host that fails gets the script automatically.

1. Run `copilot-settings.sh` on a Linux test host, then run `sha256sum /etc/github-copilot/managed-settings.json` and copy the hash.
2. Add the policies to your fleet file, replacing `<sha256>` with that hash:

   ```yaml
   policies:
     - name: Copilot managed settings applied (Windows)
       platform: windows
       description: Checks that the Copilot managed settings from Git are in the registry.
       resolution: Fleet reapplies the settings automatically. To do it yourself, run copilot-settings.ps1 from Fleet.
       query: "SELECT COUNT(*) AS applied FROM registry WHERE key = 'HKEY_LOCAL_MACHINE\\SOFTWARE\\Policies\\GitHubCopilot' AND name IN ('permissions.disableBypassPermissionsMode', 'allowedMcpServers', 'deniedMcpServers') AND type = 'REG_SZ' HAVING applied = 3;"
       run_script:
         path: ../lib/copilot/copilot-settings.ps1
     - name: Copilot managed settings current (Linux)
       platform: linux
       description: Checks that the Copilot managed settings file matches the settings in Git.
       resolution: Fleet redeploys the file automatically. To do it yourself, run copilot-settings.sh from Fleet.
       query: "SELECT 1 FROM hash WHERE path = '/etc/github-copilot/managed-settings.json' AND sha256 = '<sha256>';"
       run_script:
         path: ../lib/copilot/copilot-settings.sh
   ```

To do the same in the Fleet UI, go to **Policies**, select your fleet, and select **Add policy** for each query. Then select **Manage automations > Run script**, check each policy, and pick its script.

> **Note:** The Windows policy checks that each value exists as a string, not what it holds. When you change a value in `copilot-settings.ps1`, run the script on your Windows hosts with batch execution. On Linux, update `<sha256>` at the same time as the script, and hosts pick up the new file on their own. Fleet runs the script when a policy newly fails and tries it up to three times in total. See [Automatically run scripts](https://fleetdm.com/guides/policy-automation-run-script).

## What your users see

Fleet applies the profile and runs the scripts in the background, so users see no prompt. What changes for them:

- In Copilot CLI, `--yolo`, `--allow-all`, and the other allow-all options have no effect, and `/yolo` and `/allow-all` are blocked.
- In VS Code, global auto-approve for agent tools is off, and users can't turn it back on. In the GitHub Copilot app, sessions can't use **Approve all**.
- An MCP server that isn't on your allowlist, or is on your denylist, is blocked. Users can't add it back in their own settings or `mcp.json`.
- In VS Code, the Claude and Codex agent integrations are turned off unless you allow them with `vscode.allowedHarnesses`.
- On macOS and Windows, clients pick up changes within an hour. On Linux, users need to quit and reopen Copilot.

Tell your users before you roll out a new restriction, so a blocked MCP server or a missing agent doesn't come as a surprise.

## Verify

1. On a test host, open VS Code. Open the Command Palette and run **Developer: Sync Account Policy** to load MDM settings now. On Linux, quit and reopen VS Code instead.
2. Run **Developer: Policy Diagnostics**. Under **Effective Resolution**, confirm your keys are listed and their **Winning Source** is **Native MDM** on macOS and Windows, or **File** on Linux.
3. In Copilot CLI, run `/yolo`. Confirm that Copilot CLI doesn't turn on allow-all mode.
4. In Fleet, open the host's details page and confirm the macOS profile shows as **Verified** or the Windows or Linux policy passes.

## Review Copilot on your hosts

Your allowlist only works if it includes what your users need. Three reports show where Copilot runs and which MCP servers it uses:

- `ai_tools` with `type = 'ide_plugins'` lists the Copilot extensions in VS Code and JetBrains IDEs, including the one that ships inside VS Code.
- `ai_tools` with `type = 'mcp_server'` lists the MCP servers configured in VS Code, which Copilot in VS Code uses, with their risk flags.
- `ai_agent_session_messages` lists the MCP tools Copilot CLI called, and whether each call was allowed or denied.

Add these reports to `platforms/all/reports/` (GitOps), or go to **Reports > Add report** in the Fleet UI:

```yaml
- name: Copilot IDE extensions
  description: Lists GitHub Copilot extensions in VS Code and JetBrains IDEs on each host.
  query: "SELECT username, source AS editor, identifier, version, path FROM ai_tools WHERE type = 'ide_plugins' AND identifier IN ('github.copilot', 'github.copilot-chat', 'com.github.copilot');"
  platform: darwin,windows,linux
  interval: 86400
  observer_can_run: false
  automations_enabled: false
- name: VS Code MCP servers for Copilot
  description: Lists MCP servers configured in VS Code, with their risk flags.
  query: "SELECT username, name, location, endpoint, json_extract(detail, '$.command') AS command, risk_flags, path FROM ai_tools WHERE type = 'mcp_server' AND source = 'vscode';"
  platform: darwin,windows,linux
  interval: 3600
  observer_can_run: false
  automations_enabled: false
- name: Copilot CLI MCP calls
  description: Lists MCP tools Copilot CLI called in the last day and whether each call was allowed.
  query: "SELECT username, session_id, target, decision, timestamp FROM ai_agent_session_messages WHERE agent = 'copilot_cli' AND block_type = 'tool_use' AND action = 'mcp' AND timestamp > (strftime('%s', 'now') - 86400);"
  platform: darwin,windows,linux
  interval: 86400
  observer_can_run: false
  automations_enabled: false
```

> **Note:** `ai_agent_session_messages` can return prompts and tool arguments in its `content` column. The Copilot CLI report selects only the server and tool name in `target`, so it doesn't collect what users typed.

Then review the results on a schedule:

1. Open each report and look for MCP servers you haven't decided on. For servers configured in VS Code, check `endpoint`, `command`, and `risk_flags` first.
2. Add a server you approve to `allowedMcpServers`, using its URL or its exact command and arguments.
3. Add a server you reject to `deniedMcpServers`.
4. Look for `denied` calls in the Copilot CLI report. A spike after a rollout can point to something your users need that your settings now block.
5. Update the profile, the scripts, and the Linux hash, then open a pull request.

## Troubleshoot

**A setting doesn't apply on macOS or Windows**

Check the value's type. Every MDM value must be a string: `<string>` in the profile on macOS, and `REG_SZ` in the registry on Windows, never `<true/>`, `<integer>`, or `DWORD`. Run **Developer: Sync Account Policy** in VS Code after you fix it.

**Copilot CLI ignores the Linux file**

Run `ls -l /etc/github-copilot/managed-settings.json`. The file must be owned by root, must not be group-writable or world-writable, and can't be a symbolic link. Run `copilot-settings.sh` again to fix it, then restart Copilot CLI.

**Claude or Codex disappeared from VS Code**

This is expected while any Copilot managed setting applies. To allow them, add `vscode.allowedHarnesses` to your settings. Use the lowercase IDs `claude` and `codex`. Copilot's managed settings won't apply to those agents.

**A VS Code MCP policy stopped working**

Your Copilot `allowedMcpServers` replaced VS Code's `ChatAllowedMcpServers`. The values aren't merged. Put every approved server in the Copilot allowlist, or set the allowlist in only one of them.

**The Windows policy fails on a host that ran the script**

Run the script again from the host's details page and check the output. The policy fails until all three values exist as strings under `HKLM\SOFTWARE\Policies\GitHubCopilot`.

## Further reading

- [Choosing how to deploy enterprise-managed settings to users](https://docs.github.com/en/copilot/how-tos/administer-copilot/manage-for-enterprise/use-managed-settings/deploy-managed-settings)
- [Enterprise-managed settings reference](https://docs.github.com/en/copilot/reference/enterprise-administrators/enterprise-managed-settings)
- [Manage AI settings in VS Code](https://code.visualstudio.com/docs/enterprise/manage-ai-settings)
- [Manage VS Code settings with Fleet](https://fleetdm.com/guides/manage-vs-code-settings-with-fleet)
- [Manage Claude Code and Claude Desktop settings with Fleet](https://fleetdm.com/guides/manage-claude-code-and-claude-desktop-settings)
- [Manage Cursor settings with Fleet](https://fleetdm.com/guides/manage-cursor-settings-with-fleet)
- [Automatically run scripts](https://fleetdm.com/guides/policy-automation-run-script)

<meta name="articleTitle" value="Manage GitHub Copilot settings with Fleet">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="publishedOn" value="2026-11-23">
<meta name="category" value="guides">
<meta name="description" value="Deploy GitHub Copilot managed settings with Fleet to control MCP servers and bypass mode, keep hosts in line, and review Copilot use.">
