# Manage Gemini CLI settings with Fleet

Gemini CLI reads a system settings file and a folder of admin policies that only an admin can write. This guide shows how to deploy both with Fleet, using the Fleet UI or GitOps. It also shows how to use the `ai_tools`, `parse_json`, and `ai_agent_sessions` tables to review what your users run.

It covers Gemini CLI on macOS, Windows, and Linux. It doesn't cover [Enterprise Admin Controls](https://geminicli.com/docs/admin/enterprise-controls/), which Google applies from its own management console.

## Prerequisites

- Gemini CLI on your hosts. This guide is based on Gemini CLI 0.70.0, and settings change between releases, so test on the version your users run.
- fleetd 1.59.0 or later, which adds the `ai_tools` table, and fleetd 1.64.0 or later for the `ai_agent_sessions` table
- Scripts enabled on your hosts (on by default for macOS hosts enrolled in Fleet MDM, and [set up separately](https://fleetdm.com/guides/scripts#enable-scripts) for other platforms)
- Optional, for GitOps: a repository created with `fleetctl new` ([install fleetctl](https://fleetdm.com/guides/fleetctl#installing-fleetctl) first)
- Optional, to fix drift automatically: Fleet Premium, because script policy automations are a Premium feature

## Write the Gemini CLI policy

Gemini CLI reads two kinds of files from a system directory:

- `settings.json` holds system overrides. Gemini CLI merges it last, so it wins over the same key in a user's `~/.gemini/settings.json` or a project's `.gemini/settings.json`.
- `policies/*.toml` holds admin policy rules. They outrank every rule a user writes in `~/.gemini/policies`.

| OS | Directory |
| --- | --- |
| macOS | `/Library/Application Support/GeminiCli` |
| Linux | `/etc/gemini-cli` |
| Windows | `C:\ProgramData\gemini-cli` |

A `system-defaults.json` file in the same directory sets defaults that users can override. This guide doesn't use it.

The example `settings.json` in the scripts below does four things:

- `mcpServers` defines an approved MCP server, and `mcp.allowed` blocks every server that isn't on the list. Gemini CLI also runs tools from servers on the list without asking for confirmation, so limit what each server exposes with `includeTools` in its definition.
- `security.disableYoloMode` stops Gemini CLI from starting in YOLO mode, which approves every tool call.
- `security.auth.enforcedType` requires one sign-in method. Values include `oauth-personal` (sign in with Google), `gemini-api-key`, and `vertex-ai`.
- `privacy.usageStatisticsEnabled` turns off usage statistics.

`mcp.allowed` and `mcp.excluded` match servers by name. A name is only the label a user chose, so define every approved server in your `settings.json`. Your definition then replaces any user server with the same name. To block specific servers instead of allowing a list, use `mcp.excluded`. Gemini CLI combines every exclude list, so users can't remove yours.

The example admin policy blocks shell commands that start with `curl` or `wget` and asks before `git push`. See [Policy engine](https://geminicli.com/docs/reference/policy-engine/) for the full rule syntax.

> **Warning:** Google says these files aren't a foolproof security boundary. A user can set `GEMINI_CLI_SYSTEM_SETTINGS_PATH` to skip your `settings.json`, or start Gemini CLI with `--allowed-mcp-server-names` to replace your MCP allowlist for that session. That flag also stops Gemini CLI from using your `mcp.excluded` list when it loads servers, though the policy rules built from that list still block those servers' tools. Telemetry settings also give way to command-line flags and environment variables. The admin policy directory has no override like this, so put rules that must hold in `policies/`.

> **Warning:** Gemini CLI won't start if `settings.json` isn't valid JSON. Check it with `jq . settings.json` before you upload a script.

### macOS and Linux

Starting with Gemini CLI 0.60.0, it skips system files unless root owns them and their folders, and neither group nor others can write to them. The script sets that up. Save this as `gemini-cli-settings.sh` and replace the JSON and TOML with your policy:

```bash
#!/bin/bash
set -euo pipefail

case "$(uname -s)" in
  Darwin) dir="/Library/Application Support/GeminiCli" ;;
  Linux) dir="/etc/gemini-cli" ;;
  *) echo "Unsupported OS"; exit 1 ;;
esac

mkdir -p "$dir/policies"
chown root "$dir" "$dir/policies"
chmod 755 "$dir" "$dir/policies"

settings_tmp="$(mktemp "$dir/.settings.XXXXXX")"
policy_tmp="$(mktemp "$dir/policies/.fleet.XXXXXX")"
trap 'rm -f "$settings_tmp" "$policy_tmp"' EXIT

cat > "$settings_tmp" <<'JSON'
{
  "mcpServers": {
    "corp-tools": {
      "httpUrl": "https://mcp.example.com/mcp"
    }
  },
  "mcp": {
    "allowed": ["corp-tools"]
  },
  "security": {
    "disableYoloMode": true,
    "auth": {
      "enforcedType": "oauth-personal"
    }
  },
  "privacy": {
    "usageStatisticsEnabled": false
  }
}
JSON

cat > "$policy_tmp" <<'TOML'
[[rule]]
toolName = "run_shell_command"
commandPrefix = ["curl", "wget"]
decision = "deny"
priority = 100
denyMessage = "Your organization blocks downloads from the shell."

[[rule]]
toolName = "run_shell_command"
commandPrefix = "git push"
decision = "ask_user"
priority = 100
TOML

chown root "$settings_tmp" "$policy_tmp"
chmod 644 "$settings_tmp" "$policy_tmp"
mv "$settings_tmp" "$dir/settings.json"
mv "$policy_tmp" "$dir/policies/fleet.toml"
```

### Windows

On Windows, Gemini CLI skips system files that standard users can write to. Folders under `C:\ProgramData` let standard users create files by default, so the script replaces the folder's permissions with full control for SYSTEM and Administrators and read access for Users. Save this as `gemini-cli-settings.ps1`:

```powershell
$ErrorActionPreference = 'Stop'

$dir = Join-Path $env:ProgramData 'gemini-cli'
$settings = @'
{
  "mcpServers": {
    "corp-tools": {
      "httpUrl": "https://mcp.example.com/mcp"
    }
  },
  "mcp": {
    "allowed": ["corp-tools"]
  },
  "security": {
    "disableYoloMode": true,
    "auth": {
      "enforcedType": "oauth-personal"
    }
  },
  "privacy": {
    "usageStatisticsEnabled": false
  }
}
'@
$policy = @'
[[rule]]
toolName = "run_shell_command"
commandPrefix = ["curl", "wget"]
decision = "deny"
priority = 100
denyMessage = "Your organization blocks downloads from the shell."

[[rule]]
toolName = "run_shell_command"
commandPrefix = "git push"
decision = "ask_user"
priority = 100
'@

# Stop here if the JSON is invalid, so a bad policy never reaches a host.
$null = $settings | ConvertFrom-Json

New-Item -ItemType Directory -Path (Join-Path $dir 'policies') -Force | Out-Null
# SIDs: SYSTEM, Administrators, Users.
icacls.exe $dir /inheritance:r /grant:r '*S-1-5-18:(OI)(CI)F' '*S-1-5-32-544:(OI)(CI)F' '*S-1-5-32-545:(OI)(CI)RX' | Out-Null
if ($LASTEXITCODE -ne 0) { throw "icacls failed with exit code $LASTEXITCODE" }

# Write UTF-8 without a BOM. Windows PowerShell 5.1's Set-Content adds one.
$utf8 = New-Object System.Text.UTF8Encoding($false)
[System.IO.File]::WriteAllText((Join-Path $dir 'settings.json'), $settings, $utf8)
[System.IO.File]::WriteAllText((Join-Path $dir 'policies\fleet.toml'), $policy, $utf8)
```

## Deploy with Fleet

Deploy the same scripts with the Fleet UI or with GitOps.

### Fleet UI

1. Go to **Controls > Scripts**, select your fleet, and upload `gemini-cli-settings.sh` and `gemini-cli-settings.ps1`.
2. Go to **Hosts** and open a test host. Select **Actions > Run script** and pick the script for that host's platform.

To run a script on many hosts at once, see [batch script execution](https://fleetdm.com/guides/scripts#batch-execute-scripts).

### GitOps

1. Save the scripts in `lib/gemini/`.
2. Add them to the existing `controls.scripts` list in your fleet file, for example `fleets/workstations.yml`:

   ```yaml
   controls:
     scripts:
       - path: ../lib/gemini/gemini-cli-settings.sh
       - path: ../lib/gemini/gemini-cli-settings.ps1
   ```

3. Commit your changes and open a pull request. Your CI runs a dry run. After you merge, `fleetctl gitops` uploads the scripts.

Scripts run only when you trigger them, so add the policies below to keep hosts in line with your files.

## Keep hosts in line with your policy

Each policy below passes when both files match what you intend. A host that fails gets the script automatically.

1. Run each script on a test host.
2. Get the SHA-256 hash of both files it wrote:
   - On macOS, run `shasum -a 256 "/Library/Application Support/GeminiCli/settings.json" "/Library/Application Support/GeminiCli/policies/fleet.toml"`.
   - On Linux, run `sha256sum /etc/gemini-cli/settings.json /etc/gemini-cli/policies/fleet.toml`. The script writes the same files as on macOS, so the hashes match.
   - On Windows, run `(Get-FileHash 'C:\ProgramData\gemini-cli\settings.json').Hash.ToLower()` and `(Get-FileHash 'C:\ProgramData\gemini-cli\policies\fleet.toml').Hash.ToLower()`.
3. Add the policies to the existing `policies` list in your fleet file, replacing the placeholders with the hashes from the previous step:

   ```yaml
   policies:
     - name: Gemini CLI settings current (macOS)
       platform: darwin
       description: Checks that the Gemini CLI system settings and admin policy match the files in Git.
       resolution: Fleet redeploys the files automatically. To do it yourself, run gemini-cli-settings.sh from Fleet.
       query: "SELECT 1 FROM hash WHERE path = '/Library/Application Support/GeminiCli/settings.json' AND sha256 = '<sha256-json>' AND EXISTS (SELECT 1 FROM hash WHERE path = '/Library/Application Support/GeminiCli/policies/fleet.toml' AND sha256 = '<sha256-toml>');"
       run_script:
         path: ../lib/gemini/gemini-cli-settings.sh
     - name: Gemini CLI settings current (Linux)
       platform: linux
       description: Checks that the Gemini CLI system settings and admin policy match the files in Git.
       resolution: Fleet redeploys the files automatically. To do it yourself, run gemini-cli-settings.sh from Fleet.
       query: "SELECT 1 FROM hash WHERE path = '/etc/gemini-cli/settings.json' AND sha256 = '<sha256-json>' AND EXISTS (SELECT 1 FROM hash WHERE path = '/etc/gemini-cli/policies/fleet.toml' AND sha256 = '<sha256-toml>');"
       run_script:
         path: ../lib/gemini/gemini-cli-settings.sh
     - name: Gemini CLI settings current (Windows)
       platform: windows
       description: Checks that the Gemini CLI system settings and admin policy match the files in Git.
       resolution: Fleet redeploys the files automatically. To do it yourself, run gemini-cli-settings.ps1 from Fleet.
       query: "SELECT 1 FROM hash WHERE path = 'C:\\ProgramData\\gemini-cli\\settings.json' AND sha256 = '<sha256-json-windows>' AND EXISTS (SELECT 1 FROM hash WHERE path = 'C:\\ProgramData\\gemini-cli\\policies\\fleet.toml' AND sha256 = '<sha256-toml-windows>');"
       run_script:
         path: ../lib/gemini/gemini-cli-settings.ps1
   ```

To do the same in the Fleet UI, go to **Policies**, select your fleet, and select **Add policy** for each query. Then select **Manage automations > Run script**, check each policy, and pick its script.

> **Note:** Changing the JSON or TOML in a script changes the files' hashes, so update the placeholders in the policy at the same time. Hosts then fail the policy and pick up the new files. Fleet runs the script when a policy newly fails, and if the script fails, tries it up to three times in total. See [Automatically run scripts](https://fleetdm.com/guides/policy-automation-run-script).

## What your users see

Fleet runs the scripts in the background, so users see no prompt. Ask users to restart Gemini CLI after a change. What changes for them:

- `gemini --yolo` exits with an error that starts with `YOLO mode is disabled by your administrator`. The message links to Google's management console, which doesn't control the files you deploy with Fleet, so tell users to contact you instead.
- An MCP server that isn't on your allowlist shows as `Blocked` in `/mcp`, and Gemini CLI doesn't load its tools. Users can still add servers to their own settings, but they stay blocked.
- Tools from servers on your allowlist run without a confirmation prompt, except in plan mode.
- A user signed in with a different method sees `Authentication is enforced to be oauth-personal` in Gemini CLI and has to sign in again. In non-interactive mode, Gemini CLI exits with an error instead.
- A shell command your admin policy denies is blocked, and the user and model see your `denyMessage`.

Tell your users before you roll out a new restriction, so a blocked server or command doesn't come as a surprise.

## Verify

1. On a test host, start Gemini CLI and confirm it shows no warning that starts with `Security Warning: Skipping`.
2. Run `/mcp`. Confirm your approved server is listed and any other server shows as `Blocked`.
3. Run `/policies list`. Confirm your admin rules are listed.
4. Quit, then run `gemini --yolo`. Confirm it exits with `YOLO mode is disabled by your administrator`.
5. In Fleet, open the host's details page and confirm the policies pass.

## Review Gemini CLI usage on your hosts

Your allowlist only works if it includes what your users need. Three tables show what's already on your hosts:

- `ai_tools` lists Gemini CLI installs. `risk_flags` includes `skip_permissions_runtime` when Gemini CLI is running with `--yolo`. Your settings block that, so the flag points to a host where they aren't applied.
- `parse_json` reads the MCP servers users define in `~/.gemini/settings.json`. `ai_tools` doesn't read this file, so use `parse_json` for Gemini CLI's MCP servers.
- `ai_agent_sessions` shows the account or sign-in method behind each Gemini CLI session, so you can check `enforcedType` before you turn it on.

Add these reports to `platforms/all/reports/` (GitOps), or go to **Reports > Add report** in the Fleet UI:

```yaml
- name: Gemini CLI installs
  description: Lists Gemini CLI installs, with their risk flags.
  query: "SELECT username, version, source AS install_method, running, risk_flags, path FROM ai_tools WHERE type = 'agents' AND name = 'gemini-cli';"
  platform: darwin,windows,linux
  interval: 3600
  observer_can_run: false
  automations_enabled: false
- name: Gemini CLI MCP servers
  description: Lists MCP servers defined in each user's Gemini CLI settings.
  query: "SELECT path, substr(parent, 12) AS server, key, value FROM parse_json WHERE path IN (SELECT directory || '/.gemini/settings.json' FROM users) AND parent LIKE 'mcpServers/%' AND parent NOT LIKE 'mcpServers/%/%' AND key IN ('command', 'url', 'httpUrl');"
  platform: darwin,windows,linux
  interval: 86400
  observer_can_run: false
  automations_enabled: false
- name: Gemini CLI sign-ins
  description: Lists the account or sign-in method each user's Gemini CLI sessions use.
  query: "SELECT username, account_identifier, MAX(ended_at) AS last_active FROM ai_agent_sessions WHERE agent = 'gemini_cli' GROUP BY username, account_identifier;"
  platform: darwin,windows,linux
  interval: 86400
  observer_can_run: false
  automations_enabled: false
```

`account_identifier` shows the email address for a Google sign-in, `API KEY` for an API key, and `vertex` for Vertex AI. Users can edit the files it comes from, so treat it as a hint, not an authenticated identity.

To see which MCP tools Gemini CLI called in the last day, run this as a live query. The `ai_agent_session_messages` table can return prompts and command output, so don't schedule it:

```sql
SELECT username, session_id, target, decision FROM ai_agent_session_messages WHERE agent = 'gemini_cli' AND block_type = 'tool_use' AND action = 'mcp' AND timestamp > (strftime('%s', 'now') - 86400);
```

Then review the results on a schedule:

1. Open each report and look for MCP servers you haven't decided on. Check the `command`, `url`, or `httpUrl` value first.
2. Add a server you approve to `mcpServers` and `mcp.allowed` in both scripts.
3. Add a server you reject to `mcp.excluded`, or leave it off your allowlist.
4. Update the hashes in the policies and open a pull request. After you merge, hosts fail the policy and pick up the new files.

## Troubleshoot

**Gemini CLI shows `Security Warning: Skipping system settings file`**

Gemini CLI checks the file and every parent folder. On macOS and Linux, the warning names the path that isn't owned by root or is writable by group or others, and how to fix it. On Windows, it names the groups that can write to the path. If it names `C:\ProgramData` itself, test on one host before you change those permissions, because other apps use that folder. Gemini CLI checks only the `policies` folder itself, not its parents, so your admin policy still applies.

**Gemini CLI won't start after a deployment**

`settings.json` isn't valid JSON, and Gemini CLI exits with `Error in` followed by the file's path. Fix the JSON in your script, update the hash in the policy, and redeploy.

**`/policies list` doesn't show your rules**

Check the TOML syntax. Gemini CLI shows policy file errors when it starts. Every rule needs `toolName`, `decision`, and a `priority` from 0 to 999. If Gemini CLI shows `Security Warning: Skipping system policies`, fix the `policies` folder's owner or permissions as the warning describes.

**The policy never passes on Windows**

Check that the hashes in the policy are lowercase. osquery reports `sha256` in lowercase, and `Get-FileHash` returns uppercase unless you call `.ToLower()`.

## Further reading

- [Gemini CLI for the enterprise](https://geminicli.com/docs/cli/enterprise/): Google's guide to system settings, MCP allowlists, and authentication.
- [Policy engine](https://geminicli.com/docs/reference/policy-engine/): rule syntax, tiers, and admin policy locations.
- [Gemini CLI configuration](https://geminicli.com/docs/reference/configuration/): every setting in `settings.json`.
- [Manage Claude Code and Claude Desktop settings with Fleet](https://fleetdm.com/guides/manage-claude-code-and-claude-desktop-settings)
- [Manage Cursor settings with Fleet](https://fleetdm.com/guides/manage-cursor-settings-with-fleet)
- [Manage VS Code settings with Fleet](https://fleetdm.com/guides/manage-vs-code-settings-with-fleet)
- [Automatically run scripts](https://fleetdm.com/guides/policy-automation-run-script)

<meta name="articleTitle" value="Manage Gemini CLI settings with Fleet">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="publishedOn" value="2026-11-23">
<meta name="category" value="guides">
<meta name="description" value="Deploy Gemini CLI system settings and admin policies with Fleet, keep them in line, and review MCP servers and sign-ins on your hosts.">
