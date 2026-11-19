# Manage Claude Code and Claude Desktop settings with Fleet

Claude Code and Claude Desktop read policy from files, registry keys, and configuration profiles that only an admin can write. This guide shows how to deploy that policy with Fleet, using the Fleet UI or GitOps. It also shows how to use the `ai_tools` table to review what your users run.

It covers Claude Code on macOS, Windows, and Linux, and Claude Desktop on macOS and Windows. It doesn't cover server-managed settings from the claude.ai admin console.

## Prerequisites

- fleetd 1.59.0 or later on your hosts, which adds the `ai_tools` table
- Scripts enabled on your hosts (on by default for macOS hosts enrolled in Fleet MDM, and [set up separately](https://fleetdm.com/guides/scripts#enable-scripts) for other platforms)
- macOS hosts enrolled in Fleet MDM, to deploy the Claude Desktop profile
- Optional, for GitOps: a repository created with `fleetctl new` ([install fleetctl](https://fleetdm.com/guides/fleetctl#installing-fleetctl) first)
- Optional, to fix drift automatically: Fleet Premium, because script policy automations are a Premium feature

> **Warning:** By default, Claude Code uses only the highest-ranked managed source that sets at least one policy key and ignores the rest. The order is server-managed settings from claude.ai, then MDM or registry policy (the `com.anthropic.claudecode` domain on macOS, or `HKLM\SOFTWARE\Policies\ClaudeCode` on Windows), then files. If your organization delivers server-managed settings, or you already deploy a Claude Code profile or registry policy, Claude Code ignores the file you deploy here. To combine sources, set `managedSourcesBehavior` to `merge` in the highest-ranked source (Claude Code v2.1.242 or later).

## Write the Claude Code policy

Claude Code reads `managed-settings.json` from a system directory. No setting in a user's `~/.claude/settings.json` or in a project can override it.

| OS | Path |
| --- | --- |
| macOS | `/Library/Application Support/ClaudeCode/managed-settings.json` |
| Linux | `/etc/claude-code/managed-settings.json` |
| Windows | `C:\Program Files\ClaudeCode\managed-settings.json` |

The example policy in the scripts below does four things:

- `permissions.deny` blocks Claude Code from reading `.env` files and anything under `secrets/`.
- `permissions.disableBypassPermissionsMode` stops anyone from turning on `bypassPermissions` mode.
- `allowedMcpServers` with `allowManagedMcpServersOnly` limits MCP servers to the ones you list. Claude Code ignores users' own allowlists.
- `deniedMcpServers` blocks matching servers. Denylists from every scope still apply.

Match MCP servers with `serverUrl` or `serverCommand`. A `serverName` is only the label a user chose, so it isn't a security control.

> **Warning:** Claude Code refuses to start if the managed settings file isn't valid JSON. Check your JSON with `jq . managed-settings.json` before you upload a script.

### macOS and Linux

Save this as `claude-code-managed-settings.sh` and replace the JSON with your policy:

```bash
#!/bin/bash
set -euo pipefail

case "$(uname -s)" in
  Darwin) dir="/Library/Application Support/ClaudeCode" ;;
  Linux) dir="/etc/claude-code" ;;
  *) echo "Unsupported OS"; exit 1 ;;
esac

mkdir -p "$dir"
tmp="$(mktemp "$dir/.managed-settings.XXXXXX")"
trap 'rm -f "$tmp"' EXIT

cat > "$tmp" <<'JSON'
{
  "permissions": {
    "deny": ["Read(./.env)", "Read(./secrets/**)"],
    "disableBypassPermissionsMode": "disable"
  },
  "allowManagedMcpServersOnly": true,
  "allowedMcpServers": [
    { "serverUrl": "https://api.githubcopilot.com/*" }
  ],
  "deniedMcpServers": [
    { "serverUrl": "https://*.untrusted.example.com/*" }
  ]
}
JSON

chmod 644 "$tmp"
chown root "$tmp"
chmod 755 "$dir"
mv "$tmp" "$dir/managed-settings.json"
```

### Windows

Save this as `claude-code-managed-settings.ps1` and replace the JSON with your policy:

```powershell
$ErrorActionPreference = 'Stop'

$dir = Join-Path $env:ProgramFiles 'ClaudeCode'
$json = @'
{
  "permissions": {
    "deny": ["Read(./.env)", "Read(./secrets/**)"],
    "disableBypassPermissionsMode": "disable"
  },
  "allowManagedMcpServersOnly": true,
  "allowedMcpServers": [
    { "serverUrl": "https://api.githubcopilot.com/*" }
  ],
  "deniedMcpServers": [
    { "serverUrl": "https://*.untrusted.example.com/*" }
  ]
}
'@

# Stop here if the JSON is invalid, so a bad policy never reaches a host.
$null = $json | ConvertFrom-Json

New-Item -ItemType Directory -Path $dir -Force | Out-Null
# Write UTF-8 without a BOM. Windows PowerShell 5.1's Set-Content adds one.
[System.IO.File]::WriteAllText((Join-Path $dir 'managed-settings.json'), $json, (New-Object System.Text.UTF8Encoding($false)))
```

## Write the Claude Desktop policy

Claude Desktop reads policy from the `com.anthropic.claudefordesktop` preference domain on macOS and from `HKLM\SOFTWARE\Policies\Claude` on Windows. Anthropic documents only macOS and Windows. The example turns off local MCP servers with `isLocalDevMcpEnabled`.

Other keys include `isDesktopExtensionEnabled`, `isClaudeCodeForDesktopEnabled`, and `forceLoginOrgUUID`. See [Enterprise configuration for Claude Desktop](https://support.claude.com/en/articles/12622667-enterprise-configuration-for-claude-desktop) for the full list.

> **Warning:** `forceLoginOrgUUID` requires sign-ins to belong to the organization you name. Test it on one host first, because a wrong UUID stops people from signing in.

### macOS

Save this as `claude-desktop-settings.mobileconfig`. The `PayloadType` must be `com.anthropic.claudefordesktop`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>PayloadType</key>
      <string>com.anthropic.claudefordesktop</string>
      <key>PayloadIdentifier</key>
      <string>com.anthropic.claudefordesktop.managed</string>
      <key>PayloadUUID</key>
      <string>5B0E6B54-7C4D-4F1E-9F0B-2D8A3C6E1A41</string>
      <key>PayloadVersion</key>
      <integer>1</integer>
      <key>isLocalDevMcpEnabled</key>
      <false/>
    </dict>
  </array>
  <key>PayloadDisplayName</key>
  <string>Claude Desktop settings</string>
  <key>PayloadIdentifier</key>
  <string>com.example.claude-desktop-settings</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>0C7D1E29-3B8A-4E6F-A4D2-91F5B7C03E68</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
</dict>
</plist>
```

Generate new UUIDs for both `PayloadUUID` values and change `com.example` to your own identifier.

### Windows

Save this as `claude-desktop-settings.ps1`:

```powershell
$ErrorActionPreference = 'Stop'

$key = 'HKLM:\SOFTWARE\Policies\Claude'
New-Item -Path $key -Force | Out-Null
Set-ItemProperty -Path $key -Name 'isLocalDevMcpEnabled' -Value 0 -Type DWord
```

## Deploy with Fleet

Deploy the same files with the Fleet UI or with GitOps.

### Fleet UI

1. Go to **Controls > Scripts**, select your fleet, and upload `claude-code-managed-settings.sh`, `claude-code-managed-settings.ps1`, and `claude-desktop-settings.ps1`.
2. Go to **Controls > OS settings > Configuration profiles**, select **Add profile**, and upload `claude-desktop-settings.mobileconfig`.
3. Go to **Hosts** and open a test host. Select **Actions > Run script** and pick the script for that host's platform.

To run a script on many hosts at once, see [batch script execution](https://fleetdm.com/guides/scripts#batch-execute-scripts).

### GitOps

1. Save the scripts in `lib/claude/` and the profile in `platforms/macos/configuration-profiles/`, where the default fleet file already picks up `.mobileconfig` files.
2. Add the scripts to your fleet file, for example `fleets/workstations.yml`:

   ```yaml
   controls:
     scripts:
       - path: ../lib/claude/claude-code-managed-settings.sh
       - path: ../lib/claude/claude-code-managed-settings.ps1
       - path: ../lib/claude/claude-desktop-settings.ps1
   ```

3. Commit your changes and open a pull request. Your CI runs a dry run. After you merge, `fleetctl gitops` uploads the scripts and the profile.

Fleet sends the macOS profile to hosts automatically. Scripts run only when you trigger them, so add the policies below to keep hosts in line with your files.

## Keep hosts in line with your policy

Each policy below passes when the deployed settings match what you intend. A host that fails gets the script automatically.

1. Run each script on a test host.
2. Get the SHA-256 hash of the file it wrote:
   - On macOS, run `shasum -a 256 "<path to managed-settings.json>"`.
   - On Linux, run `sha256sum <path to managed-settings.json>`.
   - On Windows, run `(Get-FileHash 'C:\Program Files\ClaudeCode\managed-settings.json').Hash.ToLower()`.
3. Add the policies to your fleet file, replacing `<sha256>` with the hash from the previous step:

   ```yaml
   policies:
     - name: Claude Code managed settings current (macOS)
       platform: darwin
       description: Checks that the Claude Code managed settings file matches the policy in Git.
       resolution: Fleet redeploys the file automatically. To do it yourself, run claude-code-managed-settings.sh from Fleet.
       query: "SELECT 1 FROM hash WHERE path = '/Library/Application Support/ClaudeCode/managed-settings.json' AND sha256 = '<sha256>';"
       run_script:
         path: ../lib/claude/claude-code-managed-settings.sh
     - name: Claude Code managed settings current (Linux)
       platform: linux
       description: Checks that the Claude Code managed settings file matches the policy in Git.
       resolution: Fleet redeploys the file automatically. To do it yourself, run claude-code-managed-settings.sh from Fleet.
       query: "SELECT 1 FROM hash WHERE path = '/etc/claude-code/managed-settings.json' AND sha256 = '<sha256>';"
       run_script:
         path: ../lib/claude/claude-code-managed-settings.sh
     - name: Claude Code managed settings current (Windows)
       platform: windows
       description: Checks that the Claude Code managed settings file matches the policy in Git.
       resolution: Fleet redeploys the file automatically. To do it yourself, run claude-code-managed-settings.ps1 from Fleet.
       query: "SELECT 1 FROM hash WHERE path = 'C:\\Program Files\\ClaudeCode\\managed-settings.json' AND sha256 = '<sha256>';"
       run_script:
         path: ../lib/claude/claude-code-managed-settings.ps1
     - name: Claude Desktop settings applied (Windows)
       platform: windows
       description: Checks that Claude Desktop's local MCP servers are turned off.
       resolution: Fleet reapplies the setting automatically. To do it yourself, run claude-desktop-settings.ps1 from Fleet.
       query: "SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\\SOFTWARE\\Policies\\Claude\\isLocalDevMcpEnabled' AND data = '0';"
       run_script:
         path: ../lib/claude/claude-desktop-settings.ps1
   ```

To do the same in the Fleet UI, go to **Policies**, select your fleet, and select **Add policy** for each query. Then select **Manage automations > Run script**, check each policy, and pick its script.

> **Note:** Changing the JSON in a script changes the file's hash, so update `<sha256>` in the policy at the same time. Hosts then fail the policy and pick up the new file. Fleet runs the script when a policy newly fails and retries it up to three times. See [Automatically run scripts](https://fleetdm.com/guides/policy-automation-run-script).

## What your users see

Fleet runs the scripts in the background, so users see no prompt. What changes for them:

- Claude Code reads the same managed settings file in the terminal, in the IDE extensions, and in the **Code** tab in Claude Desktop. A managed key wins over the same key in a user's own settings. `/status` shows `Enterprise managed settings (file)` on the `Setting sources` line.
- `claude mcp add` fails for a server that isn't on your allowlist, with `Cannot add MCP server "<name>": not allowed by enterprise policy`. A server a user configured earlier disappears from `/mcp` and `claude mcp list` once your policy blocks it, with no message. Tell your users before you roll out a new restriction.
- Have users fully quit and reopen Claude Desktop after a change. Fleet doesn't restart it for them.
- Once any Claude Desktop policy key is set, the app hides its release channel option and stays on the Stable channel.

## Verify

1. On a test host, open Claude Code and run `/status`. Confirm the `Setting sources` line includes `Enterprise managed settings (file)`.
2. Run `claude mcp add --transport http test https://example.com/mcp`. Confirm it fails with `not allowed by enterprise policy`.
3. On a Claude Desktop host, quit and reopen the app. Confirm that local MCP servers no longer load.
4. In Fleet, open the host's details page and confirm the macOS profile shows as **Verified** and the policies pass.

## Review AI tools on your hosts

The policy only works as well as your allowlist. The `ai_tools` table shows which MCP servers and AI agents your users run, so you can decide what to add to `allowedMcpServers` or `deniedMcpServers`.

Add these reports to `platforms/all/reports/` (GitOps), or go to **Reports > Add report** in the Fleet UI:

```yaml
- name: Claude MCP servers
  description: Lists MCP servers configured in Claude Code and Claude Desktop, with their risk flags.
  query: "SELECT username, source AS client, name, location, endpoint, risk_flags, path FROM ai_tools WHERE type = 'mcp_server' AND source IN ('claude-code', 'claude-desktop');"
  platform: darwin,windows,linux
  interval: 3600
  observer_can_run: false
  automations_enabled: false
- name: Claude Code with risky permissions
  description: Lists Claude Code installs that run with permission checks off or edits auto-accepted.
  query: "SELECT username, version, json_extract(detail, '$.permission_mode') AS permission_mode, risk_flags, path FROM ai_tools WHERE type = 'agents' AND name = 'claude-code' AND risk_flags != '';"
  platform: darwin,windows,linux
  interval: 3600
  observer_can_run: false
  automations_enabled: false
```

Then review the results on a schedule:

1. Open each report and look for MCP servers you haven't approved. Check `endpoint` and `risk_flags` first.
2. Add a server you approve to `allowedMcpServers` in your script, using its `serverUrl` or `serverCommand`.
3. Add a server you reject to `deniedMcpServers`.
4. Update `<sha256>` in the policies and open a pull request. After you merge, hosts fail the policy and pick up the new file.

## Troubleshoot

**`/status` doesn't show `Enterprise managed settings (file)`**

Check that the file exists at the path for the host's OS and that its parent directories aren't writable by users. If your claude.ai organization uses server-managed settings, or a Claude Code profile or registry policy is on the host, Claude Code uses that instead. See the warning in Prerequisites.

**Claude Code won't start after a deployment**

The managed settings file isn't valid JSON. Fix the JSON in your script, update the hash in the policy, and redeploy.

**A Claude Desktop setting doesn't apply**

Quit and reopen the app. Claude Desktop ignores a misspelled key without an error, so compare your key names against Anthropic's list.

**The policy never passes on Windows**

Check that the hash in the policy is lowercase. osquery reports `sha256` in lowercase, and `Get-FileHash` returns uppercase unless you call `.ToLower()`.

## Further reading

- [Deploy managed settings](https://code.claude.com/docs/en/managed-settings): Anthropic's reference for Claude Code delivery options, including configuration profiles and registry policy.
- [Control MCP server access](https://code.claude.com/docs/en/managed-mcp): `managed-mcp.json`, allowlists, and denylists.
- [Enterprise configuration for Claude Desktop](https://support.claude.com/en/articles/12622667-enterprise-configuration-for-claude-desktop)
- [Use Claude Code with Fleet](https://fleetdm.com/guides/use-claude-code-with-fleet)
- [Automatically run scripts](https://fleetdm.com/guides/policy-automation-run-script)

<meta name="articleTitle" value="Manage Claude Code and Claude Desktop settings with Fleet">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="publishedOn" value="2026-10-13">
<meta name="category" value="guides">
<meta name="description" value="Deploy managed settings for Claude Code and Claude Desktop with Fleet, keep them in line, and review MCP servers with the ai_tools table.">
