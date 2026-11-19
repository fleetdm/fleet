# Manage Cursor settings with Fleet

Cursor reads enterprise policies that override what users set in their own `settings.json`, and it runs system-wide hooks that users can't turn off. This guide shows how to deploy both with Fleet, using the Fleet UI or GitOps. You'll limit Cursor to your team's accounts and approved extensions, block MCP servers you haven't approved, and review what your users already run.

It covers Cursor on macOS, Windows, and Linux. It doesn't cover settings you control from Cursor's web dashboard, like Privacy Mode or model access.

## Prerequisites

- Cursor 2.0 or later on your hosts
- A Cursor Teams or Enterprise plan, for the team ID policy
- fleetd 1.59.0 or later, which adds the `ai_tools` table
- `jq` on your macOS and Linux hosts, for the MCP hook (macOS 15 and later include it at `/usr/bin/jq`)
- macOS hosts enrolled in Fleet MDM, to deploy the profile
- Scripts enabled on your hosts (see [Enable scripts](https://fleetdm.com/guides/scripts#enable-scripts))
- Optional, for GitOps: a repository created with `fleetctl new` ([install fleetctl](https://fleetdm.com/guides/fleetctl#installing-fleetctl) first)
- Optional, to fix drift automatically: Fleet Premium, because script policy automations are a Premium feature

## Choose your policies

Cursor is built on VS Code, but it reads its own policies, not VS Code's. The examples below set two:

- `AllowedExtensions` allows every extension from Anysphere (Cursor's publisher) and GitHub, plus Prettier. Cursor blocks other extensions. Keys are publishers or extension IDs, and an extension ID wins over its publisher.
- `AllowedTeamId` signs out any user who isn't on the Cursor teams you list. Replace `12345` with your team ID. To allow more than one team, separate the IDs with commas.

Other policies include `UpdateMode` (`none`, `manual`, `start`, `default`, or `silentlyApplyOnQuit`) and `WorkspaceTrustEnabled`. See [Cursor's MDM configuration reference](https://cursor.com/docs/enterprise/deployment-patterns#mdm-configuration) for the full list.

> **Warning:** Deploying `AllowedExtensions` blocks every extension that doesn't match it, including ones your users rely on today. Review installed extensions first (see [Review extensions and MCP servers](#review-extensions-and-mcp-servers)) and roll out to a test fleet before your whole organization.

## Write the policy for each OS

### macOS

Cursor reads policy from a configuration profile with the `com.todesktop.230313mzl4w4u92` payload type, which is the bundle ID of Cursor's production channel. A sample with every policy ships at `/Applications/Cursor.app/Contents/Resources/app/policies/com.todesktop.230313mzl4w4u92.mobileconfig`. Don't deploy the sample as is, because Cursor enforces the default value of every policy left in it.

Save this as `cursor-settings.mobileconfig`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>PayloadType</key>
      <string>com.todesktop.230313mzl4w4u92</string>
      <key>PayloadIdentifier</key>
      <string>com.todesktop.230313mzl4w4u92.managed</string>
      <key>PayloadUUID</key>
      <string>6C1D8E3B-2A4F-4B97-8E05-9F3A7C2D1B48</string>
      <key>PayloadVersion</key>
      <integer>1</integer>
      <key>AllowedExtensions</key>
      <string>{"anysphere": true, "github": true, "esbenp.prettier-vscode": true}</string>
      <key>AllowedTeamId</key>
      <string>12345</string>
    </dict>
  </array>
  <key>PayloadDisplayName</key>
  <string>Cursor settings</string>
  <key>PayloadIdentifier</key>
  <string>com.example.cursor-settings</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>A93F2E61-7D5C-4C08-B1A4-3E8D6F2B9C57</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
</dict>
</plist>
```

Generate new UUIDs for both `PayloadUUID` values and change `com.example` to your own identifier.

### Windows

Cursor reads policy from `HKLM\SOFTWARE\Policies\Microsoft\Cursor`. Every policy in this example is a string.

Save this as `cursor-settings.ps1`:

```powershell
$ErrorActionPreference = 'Stop'

$key = 'HKLM:\SOFTWARE\Policies\Microsoft\Cursor'
$allowedExtensions = '{"anysphere": true, "github": true, "esbenp.prettier-vscode": true}'

# Stop here if the JSON is invalid, so a bad policy never reaches a host.
$null = $allowedExtensions | ConvertFrom-Json

if (-not (Test-Path $key)) { New-Item -Path $key -Force | Out-Null }
New-ItemProperty -Path $key -Name 'AllowedExtensions' -PropertyType String -Value $allowedExtensions -Force | Out-Null
New-ItemProperty -Path $key -Name 'AllowedTeamId' -PropertyType String -Value '12345' -Force | Out-Null
```

### Linux

Cursor reads policy from `~/.cursor/policy.json` in each user's home directory. Every value is a string, number, or boolean, so `AllowedExtensions` is a JSON string with escaped quotes, not a JSON object.

> **Warning:** On Linux, the policy file lives in the user's home directory, so users can edit or delete it. Treat it as a default you reapply, not as a lock. On macOS and Windows, users can't change the policy.

Save this as `cursor-settings.sh`:

```bash
#!/bin/bash
set -euo pipefail

policy='{
  "AllowedExtensions": "{\"anysphere\": true, \"github\": true, \"esbenp.prettier-vscode\": true}",
  "AllowedTeamId": "12345"
}'

for home in /home/*; do
  [ -d "$home" ] || continue
  user="$(stat -c %U "$home")"
  install -d -o "$user" -g "$(id -gn "$user")" "$home/.cursor"
  printf '%s\n' "$policy" > "$home/.cursor/policy.json"
  chown "$user:$(id -gn "$user")" "$home/.cursor/policy.json"
  chmod 644 "$home/.cursor/policy.json"
done
```

Cursor watches the Linux policy file and applies changes without a restart. If a change doesn't take effect, restart Cursor.

## Block MCP servers with an enterprise hook

Cursor has no policy that blocks MCP servers. Instead, deploy an enterprise `beforeMCPExecution` hook. Cursor runs it before every MCP tool call, and enterprise hooks take priority over team, project, and user hooks. Enterprise hooks run from the folder that holds `hooks.json`, so the commands below use relative paths. Cursor reads `hooks.json` from these paths:

| OS | Path |
| --- | --- |
| macOS | `/Library/Application Support/Cursor/hooks.json` |
| Linux | `/etc/cursor/hooks.json` |
| Windows | `C:\ProgramData\Cursor\hooks.json` |

The hook below allows calls to MCP servers whose URL matches your list and denies everything else, including local (stdio) servers. Setting `failClosed` to `true` blocks the call if the hook crashes or times out, instead of letting it through.

### macOS and Linux

Save this as `cursor-mcp-hook.sh` and replace the patterns in `allowed_urls` with your own:

```bash
#!/bin/bash
set -euo pipefail

case "$(uname -s)" in
  Darwin) dir="/Library/Application Support/Cursor" ;;
  Linux) dir="/etc/cursor" ;;
  *) echo "Unsupported OS"; exit 1 ;;
esac

mkdir -p "$dir/hooks"

cat > "$dir/hooks/mcp-allowlist.sh" <<'HOOK'
#!/bin/bash
allowed_urls=(
  "https://api.githubcopilot.com/*"
  "https://mcp.linear.app/*"
)

url="$(jq -r '.mcp_server_url // empty')"
for pattern in "${allowed_urls[@]}"; do
  if [[ -n "$url" && "$url" == $pattern ]]; then
    echo '{"permission": "allow"}'
    exit 0
  fi
done

echo '{"permission": "deny", "user_message": "Your organization blocks this MCP server.", "agent_message": "This MCP server is blocked by organization policy."}'
HOOK

cat > "$dir/hooks.json" <<'JSON'
{
  "version": 1,
  "hooks": {
    "beforeMCPExecution": [
      { "command": "./hooks/mcp-allowlist.sh", "failClosed": true }
    ]
  }
}
JSON

chmod 755 "$dir/hooks" "$dir/hooks/mcp-allowlist.sh"
chmod 644 "$dir/hooks.json"
chown -R root "$dir/hooks" "$dir/hooks.json"
```

### Windows

On Windows, Cursor runs hook commands through PowerShell and pipes the JSON input to them. Save this as `cursor-mcp-hook.ps1`:

```powershell
$ErrorActionPreference = 'Stop'

$dir = Join-Path $env:ProgramData 'Cursor'
New-Item -ItemType Directory -Path (Join-Path $dir 'hooks') -Force | Out-Null

$hook = @'
$allowedUrls = @(
  'https://api.githubcopilot.com/*'
  'https://mcp.linear.app/*'
)

$request = $input | Out-String | ConvertFrom-Json
foreach ($pattern in $allowedUrls) {
  if ($request.mcp_server_url -and $request.mcp_server_url -like $pattern) {
    '{"permission": "allow"}'
    exit 0
  }
}

'{"permission": "deny", "user_message": "Your organization blocks this MCP server.", "agent_message": "This MCP server is blocked by organization policy."}'
'@

$config = @'
{
  "version": 1,
  "hooks": {
    "beforeMCPExecution": [
      { "command": "powershell -NoProfile -ExecutionPolicy Bypass -File .\\hooks\\mcp-allowlist.ps1", "failClosed": true }
    ]
  }
}
'@

$utf8 = New-Object System.Text.UTF8Encoding($false)
[System.IO.File]::WriteAllText((Join-Path $dir 'hooks\mcp-allowlist.ps1'), $hook, $utf8)
[System.IO.File]::WriteAllText((Join-Path $dir 'hooks.json'), $config, $utf8)
```

> **Note:** Users with admin rights can change or remove the hook. On Windows, also check that standard users can't modify the files in `C:\ProgramData\Cursor`.

## Deploy with Fleet

Deploy the same files with the Fleet UI or with GitOps.

### Fleet UI

1. Go to **Controls > OS settings > Configuration profiles**, select your fleet, select **Add profile**, and upload `cursor-settings.mobileconfig`.
2. Go to **Controls > Scripts** and upload `cursor-settings.ps1`, `cursor-settings.sh`, `cursor-mcp-hook.sh`, and `cursor-mcp-hook.ps1`.
3. Go to **Hosts** and open a test host. Select **Actions > Run script** and run the scripts for that host's platform.

To run a script on many hosts at once, see [batch script execution](https://fleetdm.com/guides/scripts#batch-execute-scripts).

### GitOps

1. Save the profile in `platforms/macos/configuration-profiles/`, where the `fleets/workstations.yml` file from `fleetctl new` already picks up `.mobileconfig` files.
2. Save the scripts in `lib/cursor/` and add them to your fleet file, for example `fleets/workstations.yml`:

   ```yaml
   controls:
     scripts:
       - path: ../lib/cursor/cursor-settings.ps1
       - path: ../lib/cursor/cursor-settings.sh
       - path: ../lib/cursor/cursor-mcp-hook.sh
       - path: ../lib/cursor/cursor-mcp-hook.ps1
   ```

3. Commit your changes and open a pull request. Your CI runs a dry run. After you merge, `fleetctl gitops` uploads the profile and the scripts.

Fleet sends the macOS profile to hosts automatically. Scripts run only when you trigger them, so add the policies below to keep hosts in line.

## Keep hosts in line with your policy

Each policy below passes when the host has your settings. A host that fails gets the script automatically.

1. Run `cursor-mcp-hook.sh` on a macOS test host and `cursor-mcp-hook.ps1` on a Windows test host.
2. Get the SHA-256 hash of the hook script each one wrote:
   - On macOS, run `shasum -a 256 "/Library/Application Support/Cursor/hooks/mcp-allowlist.sh"`.
   - On Windows, run `(Get-FileHash 'C:\ProgramData\Cursor\hooks\mcp-allowlist.ps1').Hash.ToLower()`.
   The Linux hook script is the same file as on macOS, so it has the same hash.
3. Add the policies to your fleet file, replacing `<sha256-sh>` and `<sha256-ps1>` with those hashes:

   ```yaml
   policies:
     - name: Cursor policies applied (Windows)
       platform: windows
       description: Checks that the Cursor policies from Git are in the registry.
       resolution: Fleet reapplies the policies automatically. To do it yourself, run cursor-settings.ps1 from Fleet.
       query: "SELECT COUNT(*) AS applied FROM registry WHERE key = 'HKEY_LOCAL_MACHINE\\SOFTWARE\\Policies\\Microsoft\\Cursor' AND name IN ('AllowedExtensions', 'AllowedTeamId') HAVING applied = 2;"
       run_script:
         path: ../lib/cursor/cursor-settings.ps1
     - name: Cursor MCP hook current (macOS)
       platform: darwin
       description: Checks that the Cursor MCP hook matches the version in Git.
       resolution: Fleet redeploys the hook automatically. To do it yourself, run cursor-mcp-hook.sh from Fleet.
       query: "SELECT 1 FROM hash WHERE path = '/Library/Application Support/Cursor/hooks/mcp-allowlist.sh' AND sha256 = '<sha256-sh>' AND EXISTS (SELECT 1 FROM file WHERE path = '/Library/Application Support/Cursor/hooks.json');"
       run_script:
         path: ../lib/cursor/cursor-mcp-hook.sh
     - name: Cursor MCP hook current (Linux)
       platform: linux
       description: Checks that the Cursor MCP hook matches the version in Git.
       resolution: Fleet redeploys the hook automatically. To do it yourself, run cursor-mcp-hook.sh from Fleet.
       query: "SELECT 1 FROM hash WHERE path = '/etc/cursor/hooks/mcp-allowlist.sh' AND sha256 = '<sha256-sh>' AND EXISTS (SELECT 1 FROM file WHERE path = '/etc/cursor/hooks.json');"
       run_script:
         path: ../lib/cursor/cursor-mcp-hook.sh
     - name: Cursor MCP hook current (Windows)
       platform: windows
       description: Checks that the Cursor MCP hook matches the version in Git.
       resolution: Fleet redeploys the hook automatically. To do it yourself, run cursor-mcp-hook.ps1 from Fleet.
       query: "SELECT 1 FROM hash WHERE path = 'C:\\ProgramData\\Cursor\\hooks\\mcp-allowlist.ps1' AND sha256 = '<sha256-ps1>' AND EXISTS (SELECT 1 FROM file WHERE path = 'C:\\ProgramData\\Cursor\\hooks.json');"
       run_script:
         path: ../lib/cursor/cursor-mcp-hook.ps1
   ```

To do the same in the Fleet UI, go to **Policies**, select your fleet, and select **Add policy** for each query. Then select **Manage automations > Run script**, check each policy, and pick its script.

> **Note:** Changing the allowlist in a hook script changes its hash, so update the hash in the policy at the same time. Hosts then fail the policy and pick up the new hook. The Windows registry policy checks only that each value exists, so when you change a value in `cursor-settings.ps1`, run the script on your Windows hosts with batch execution. Fleet runs the script when a policy newly fails, and makes up to three attempts if the script fails. See [Automatically run scripts](https://fleetdm.com/guides/policy-automation-run-script).

## What your users see

Fleet applies the profile and runs the scripts in the background, so users see no prompt. What changes for them:

- A managed setting shows as managed by your organization, and users can't change it. A policy wins over the same setting in a user's or workspace's `settings.json`.
- A user signed in to an account that isn't on one of your teams is signed out.
- An extension that isn't on your allowlist can't be installed.
- When the agent calls a tool on an MCP server you haven't allowed, the call is blocked and the user sees "Your organization blocks this MCP server."
- Cursor reloads `hooks.json` when it changes. Users with Cursor already open may need to restart it to pick up a new policy.

Tell your users before you roll out a new restriction, so a blocked extension or MCP server doesn't come as a surprise.

## Verify

1. On a test host, quit and reopen Cursor.
2. Open Settings and search for `extensions.allowed`. Confirm Cursor shows it as managed by your organization.
3. Try to install an extension from a publisher you didn't allow. Confirm Cursor blocks it.
4. Ask the agent to use an MCP server that isn't on your list. Confirm Cursor blocks the call with your message.
5. In Fleet, open the host's details page and confirm the macOS profile shows as **Verified** and the policies pass.

## Review extensions and MCP servers

Your allowlists only work if they include what your users need. Two tables show what's already on your hosts:

- `vscode_extensions` lists every installed extension, including Cursor's, with `name` set to the extension ID you use in `AllowedExtensions`.
- `ai_tools` lists the MCP servers configured in Cursor, with their risk flags.

Add these reports to `platforms/all/reports/` (GitOps), or go to **Reports > Add report** in the Fleet UI:

```yaml
- name: Cursor extensions
  description: Lists Cursor extensions installed on each host.
  query: "SELECT u.username, e.name, e.publisher, e.version FROM users u CROSS JOIN vscode_extensions e USING (uid) WHERE e.vscode_edition = 'cursor';"
  platform: darwin,windows,linux
  interval: 86400
  observer_can_run: false
  automations_enabled: false
- name: Cursor MCP servers
  description: Lists MCP servers configured in Cursor, with their risk flags.
  query: "SELECT username, name, location, endpoint, risk_flags, path FROM ai_tools WHERE type = 'mcp_server' AND source = 'cursor';"
  platform: darwin,windows,linux
  interval: 3600
  observer_can_run: false
  automations_enabled: false
```

Then review the results on a schedule:

1. Open each report and look for extensions and MCP servers you haven't decided on. For MCP servers, check `endpoint` and `risk_flags` first.
2. Add an extension you approve to `AllowedExtensions`, by extension ID or by publisher.
3. Add an MCP server you approve to the URL list in both hook scripts.
4. Update the profile, scripts, and hashes, then open a pull request.

## Troubleshoot

**Every MCP call is blocked, including allowed servers**

On macOS and Linux, check that `jq` is installed. Without it, the hook can't read the server URL, so it denies every call. Run the hook by hand to see its output: `echo '{"mcp_server_url": "https://mcp.linear.app/sse"}' | "/Library/Application Support/Cursor/hooks/mcp-allowlist.sh"`.

**A local (stdio) MCP server is blocked**

The example hook only allows servers by URL. To allow a local server, extend the hook to check the `command` field, which holds the server's launch command.

**Users were signed out unexpectedly**

Check the team ID in `AllowedTeamId`. A wrong ID signs out everyone. Test on one host before rolling out.

**A policy doesn't apply on Linux**

Quit and reopen Cursor, then check that `~/.cursor/policy.json` is valid JSON and that `AllowedExtensions` is a string, not an object.

## Further reading

- [Cursor deployment patterns and MDM configuration](https://cursor.com/docs/enterprise/deployment-patterns)
- [Cursor hooks](https://cursor.com/docs/hooks)
- [Manage VS Code settings with Fleet](https://fleetdm.com/guides/manage-vs-code-settings-with-fleet)
- [Manage Claude Code and Claude Desktop settings with Fleet](https://fleetdm.com/guides/manage-claude-code-and-claude-desktop-settings)
- [Automatically run scripts](https://fleetdm.com/guides/policy-automation-run-script)

<meta name="articleTitle" value="Manage Cursor settings with Fleet">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="publishedOn" value="2026-11-23">
<meta name="category" value="guides">
<meta name="description" value="Deploy Cursor policies and an MCP allowlist hook with Fleet, keep hosts in line, and review the extensions and MCP servers users run.">
