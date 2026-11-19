# Manage Codex settings with Fleet

OpenAI's Codex reads a `requirements.toml` file that only an admin can write. Requirements limit what users can choose, like the approval policy, the sandbox mode, and which MCP servers can run, and nothing in a user's `~/.codex/config.toml` can override them. This guide shows how to deploy requirements with Fleet, using the Fleet UI or GitOps. It also shows how to use the `ai_tools` and `file_lines` tables to review what your users have configured.

It covers Codex CLI and the Codex IDE extension, which share the same configuration, on macOS, Windows, and Linux. OpenAI's docs say the desktop app applies requirements too, so test it on your hosts before you rely on it. This guide doesn't cover Agent security policies you set in the ChatGPT admin console.

## Prerequisites

- fleetd 1.62.0 or later on your hosts, so the `ai_tools` table reports Codex MCP servers (fleetd 1.59.0 added the table, and 1.62.0 added Codex's `config.toml` as a source)
- Scripts enabled on your hosts (on by default for macOS hosts enrolled in Fleet MDM, and [set up separately](https://fleetdm.com/guides/scripts#enable-scripts) for other platforms)
- macOS hosts enrolled in Fleet MDM, to deploy the configuration profile
- Optional, for GitOps: a repository created with `fleetctl new` ([install fleetctl](https://fleetdm.com/guides/fleetctl#installing-fleetctl) first)
- Optional, to fix drift automatically: Fleet Premium, because script policy automations are a Premium feature

> **Warning:** Codex combines requirements from several sources. From lowest to highest precedence, the main ones are the system `requirements.toml` file, Agent security policies from the ChatGPT admin console, and macOS managed preferences. A higher source replaces a lower source's value for the same setting, while tables like `mcp_servers` merge by key. If your organization uses Agent security, its requirements win over the file you deploy on Windows and Linux, but not over the macOS profile.

## Write the requirements

The example below does four things:

- `allowed_approval_policies` blocks `--ask-for-approval never`, so Codex can't run commands without asking.
- `allowed_sandbox_modes` blocks `danger-full-access`, including `--yolo`. The list must include `read-only`.
- `allowed_web_search_modes` allows cached web search and blocks live search. Codex always allows `disabled`.
- `mcp_servers` is an allowlist. Codex enables an MCP server only when its name and its identity both match an entry. Match a remote server with `url` and a local (stdio) server with `command`. An empty `mcp_servers` table turns off every MCP server.

Save this as `codex-requirements.toml`:

```toml
allowed_approval_policies = ["untrusted", "on-request"]
allowed_sandbox_modes = ["read-only", "workspace-write"]
allowed_web_search_modes = ["cached"]

[mcp_servers.github]
identity = { url = "https://api.githubcopilot.com/mcp/" }

[mcp_servers.linear]
identity = { url = { match = "prefix", value = "https://mcp.linear.app/" } }
```

Other requirements include `allowed_login_methods` and `allowed_chatgpt_workspaces`, which limit how users sign in, a `[rules]` table that makes Codex prompt for or forbid matching commands, and `allow_managed_hooks_only`. See the [`requirements.toml` reference](https://developers.openai.com/codex/config-reference) for every key, and check that the Codex versions on your hosts support each key you use.

> **Warning:** Codex won't load its configuration if `requirements.toml` isn't valid TOML. Check your file with `python3 -c 'import tomllib; tomllib.load(open("codex-requirements.toml", "rb"))'` (Python 3.11 or later) before you deploy it.

## Deliver the requirements on each OS

| OS | Where Codex reads requirements |
| --- | --- |
| macOS | The `requirements_toml_base64` key in the `com.openai.codex` preference domain, from a configuration profile |
| Linux | `/etc/codex/requirements.toml` |
| Windows | `C:\ProgramData\OpenAI\Codex\requirements.toml` |

Codex also reads `/etc/codex/requirements.toml` on macOS, but the profile ranks higher and users can't remove it.

### macOS

Encode the TOML file as base64 on one line:

```bash
base64 < codex-requirements.toml | tr -d '\n'
```

Save this as `codex-requirements.mobileconfig` and replace the `requirements_toml_base64` string with your output. The `PayloadType` must be `com.openai.codex`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>PayloadType</key>
      <string>com.openai.codex</string>
      <key>PayloadIdentifier</key>
      <string>com.openai.codex.managed</string>
      <key>PayloadUUID</key>
      <string>9446336B-510A-4CAD-AC52-25BD908803E6</string>
      <key>PayloadVersion</key>
      <integer>1</integer>
      <key>requirements_toml_base64</key>
      <string>YWxsb3dlZF9hcHByb3ZhbF9wb2xpY2llcyA9IFsidW50cnVzdGVkIiwgIm9uLXJlcXVlc3QiXQphbGxvd2VkX3NhbmRib3hfbW9kZXMgPSBbInJlYWQtb25seSIsICJ3b3Jrc3BhY2Utd3JpdGUiXQphbGxvd2VkX3dlYl9zZWFyY2hfbW9kZXMgPSBbImNhY2hlZCJdCgpbbWNwX3NlcnZlcnMuZ2l0aHViXQppZGVudGl0eSA9IHsgdXJsID0gImh0dHBzOi8vYXBpLmdpdGh1YmNvcGlsb3QuY29tL21jcC8iIH0KClttY3Bfc2VydmVycy5saW5lYXJdCmlkZW50aXR5ID0geyB1cmwgPSB7IG1hdGNoID0gInByZWZpeCIsIHZhbHVlID0gImh0dHBzOi8vbWNwLmxpbmVhci5hcHAvIiB9IH0K</string>
    </dict>
  </array>
  <key>PayloadDisplayName</key>
  <string>Codex requirements</string>
  <key>PayloadIdentifier</key>
  <string>com.example.codex-requirements</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>CA91B6B6-BF6E-4452-B021-A83F2E7EA8D8</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
</dict>
</plist>
```

Generate new UUIDs for both `PayloadUUID` values and change `com.example` to your own identifier.

### Linux

Save this as `codex-requirements.sh` and replace the TOML with your requirements:

```bash
#!/bin/bash
set -euo pipefail

dir="/etc/codex"
mkdir -p "$dir"
tmp="$(mktemp "$dir/.requirements.XXXXXX")"
trap 'rm -f "$tmp"' EXIT

cat > "$tmp" <<'TOML'
allowed_approval_policies = ["untrusted", "on-request"]
allowed_sandbox_modes = ["read-only", "workspace-write"]
allowed_web_search_modes = ["cached"]

[mcp_servers.github]
identity = { url = "https://api.githubcopilot.com/mcp/" }

[mcp_servers.linear]
identity = { url = { match = "prefix", value = "https://mcp.linear.app/" } }
TOML

chmod 644 "$tmp"
chown root:root "$tmp"
chmod 755 "$dir"
mv "$tmp" "$dir/requirements.toml"
```

### Windows

Save this as `codex-requirements.ps1` and replace the TOML with your requirements:

```powershell
$ErrorActionPreference = 'Stop'

$dir = Join-Path $env:ProgramData 'OpenAI\Codex'
$toml = @'
allowed_approval_policies = ["untrusted", "on-request"]
allowed_sandbox_modes = ["read-only", "workspace-write"]
allowed_web_search_modes = ["cached"]

[mcp_servers.github]
identity = { url = "https://api.githubcopilot.com/mcp/" }

[mcp_servers.linear]
identity = { url = { match = "prefix", value = "https://mcp.linear.app/" } }
'@

New-Item -ItemType Directory -Path $dir -Force | Out-Null
# Write UTF-8 without a BOM. Windows PowerShell 5.1's Set-Content adds one.
[System.IO.File]::WriteAllText((Join-Path $dir 'requirements.toml'), $toml, (New-Object System.Text.UTF8Encoding($false)))
```

> **Note:** By default, standard users can create folders in `C:\ProgramData`. Check that standard users can't modify `C:\ProgramData\OpenAI\Codex` or the file in it, especially if the folder existed before you deployed the script.

### Optional: set defaults users can change

Requirements limit choices but don't pick a starting value. To set defaults, like `approval_policy = "on-request"`, deploy a system `config.toml` the same way: `/etc/codex/config.toml` on macOS and Linux, or `C:\ProgramData\OpenAI\Codex\config.toml` on Windows. A user's own `~/.codex/config.toml` overrides these defaults, but only with values your requirements allow. On macOS, the `config_toml_base64` key in the same profile sets defaults that Codex reapplies each time it starts, over the user's own settings. This guide doesn't edit users' `~/.codex/config.toml` files, because requirements already win over them.

## Deploy with Fleet

Deploy the same files with the Fleet UI or with GitOps.

### Fleet UI

1. Go to **Controls > OS settings > Configuration profiles**, select your fleet, select **Add profile**, and upload `codex-requirements.mobileconfig`.
2. Go to **Controls > Scripts** and upload `codex-requirements.sh` and `codex-requirements.ps1`.
3. Go to **Hosts** and open a Linux or Windows test host. Select **Actions > Run script** and pick the script for that host's platform.

To run a script on many hosts at once, see [batch script execution](https://fleetdm.com/guides/scripts#batch-execute-scripts).

### GitOps

1. Save the profile in `platforms/macos/configuration-profiles/`. The `fleets/workstations.yml` file that `fleetctl new` creates already picks up `.mobileconfig` files there.
2. Save the scripts in `lib/codex/` and add them to your fleet file, for example `fleets/workstations.yml`:

   ```yaml
   controls:
     scripts:
       - path: ../lib/codex/codex-requirements.sh
       - path: ../lib/codex/codex-requirements.ps1
   ```

3. Commit your changes and open a pull request. Your CI runs a dry run. After you merge, `fleetctl gitops` uploads the profile and the scripts.

Fleet sends the macOS profile to hosts automatically. Scripts run only when you trigger them, so add the policies below to keep Linux and Windows hosts in line.

## Keep hosts in line with your requirements

Each policy below passes when the requirements file matches what you intend. A host that fails gets the script automatically.

1. Run each script on a test host.
2. Get the SHA-256 hash of the file it wrote:
   - On Linux, run `sha256sum /etc/codex/requirements.toml`.
   - On Windows, run `(Get-FileHash 'C:\ProgramData\OpenAI\Codex\requirements.toml').Hash.ToLower()`.
3. Add the policies to your fleet file, replacing `<sha256-linux>` and `<sha256-windows>` with those hashes:

   ```yaml
   policies:
     - name: Codex requirements current (Linux)
       platform: linux
       description: Checks that the Codex requirements file matches the version in Git.
       resolution: Fleet redeploys the file automatically. To do it yourself, run codex-requirements.sh from Fleet.
       query: "SELECT 1 FROM hash WHERE path = '/etc/codex/requirements.toml' AND sha256 = '<sha256-linux>';"
       run_script:
         path: ../lib/codex/codex-requirements.sh
     - name: Codex requirements current (Windows)
       platform: windows
       description: Checks that the Codex requirements file matches the version in Git.
       resolution: Fleet redeploys the file automatically. To do it yourself, run codex-requirements.ps1 from Fleet.
       query: "SELECT 1 FROM hash WHERE path = 'C:\\ProgramData\\OpenAI\\Codex\\requirements.toml' AND sha256 = '<sha256-windows>';"
       run_script:
         path: ../lib/codex/codex-requirements.ps1
   ```

To do the same in the Fleet UI, go to **Policies**, select your fleet, and select **Add policy** for each query. Then select **Manage automations > Run script**, check each policy, and pick its script.

> **Note:** The same requirements live in three files: the profile, the shell script, and the PowerShell script. When you change them, update all three, re-encode the base64 string, and update both hashes. Hosts then fail the policy and pick up the new file. Fleet runs the script when a policy newly fails and makes up to three attempts in total. See [Automatically run scripts](https://fleetdm.com/guides/policy-automation-run-script).

## What your users see

Fleet applies the profile and runs the scripts in the background, so users see no prompt. What changes for them:

- Codex reads requirements when it starts. Have users quit and restart Codex after a change.
- If a user's `config.toml`, a profile, or a command-line flag asks for a value your requirements don't allow, Codex falls back to an allowed value and tells the user.
- An MCP server that isn't on your allowlist doesn't load. `/mcp` lists it as `(disabled)`, with `requirements` and the requirements source as the reason.
- A server name must match your allowlist exactly. If a user named a server `gh` and your allowlist says `github`, Codex disables it even when the URL matches. Share the names you use before you roll out an allowlist.
- `/debug-config` shows each requirement and the source it came from.

## Verify

1. On a test host, quit and restart Codex CLI. Run `/debug-config` and check the `Requirements` section. On Linux, `allowed_sandbox_modes` shows `(source: /etc/codex/requirements.toml)`. On macOS, it shows `MDM com.openai.codex:requirements_toml_base64`.
2. Quit, then run `codex --sandbox danger-full-access`. Confirm that Codex doesn't start a session with full access.
3. Add an MCP server that isn't on your allowlist to `~/.codex/config.toml` and restart Codex. Run `/mcp` and confirm the server shows as `(disabled)`.
4. In Fleet, open the host's details page and confirm the macOS profile shows as **Verified** and the policies pass.

## Review Codex settings on your hosts

Your allowlist only works if it includes what your users need. Three tables show what's already on your hosts:

- `ai_tools` lists the MCP servers in each user's `~/.codex/config.toml` (`source` is `codex`) and each Codex CLI install (`type` is `agents` and `name` is `codex`).
- `file_lines` reads users' `~/.codex/config.toml` files line by line. The report below keeps only the approval, sandbox, and web search lines, so it doesn't collect tokens or headers from the rest of the file.
- `ai_agent_sessions` (fleetd 1.64.0 and later) lists Codex CLI sessions with the account each one used.

Add these reports to `platforms/all/reports/` (GitOps), or go to **Reports > Add report** in the Fleet UI:

```yaml
- name: Codex MCP servers
  description: Lists MCP servers configured in Codex, with their risk flags.
  query: "SELECT username, name, location, endpoint, json_extract(detail, '$.command') AS command, risk_flags, path FROM ai_tools WHERE type = 'mcp_server' AND source = 'codex';"
  platform: darwin,windows,linux
  interval: 3600
  observer_can_run: false
  automations_enabled: false
- name: Codex CLI installs
  description: Lists Codex CLI installs, and flags ones running with approvals and the sandbox turned off.
  query: "SELECT username, version, running, risk_flags, path FROM ai_tools WHERE type = 'agents' AND name = 'codex';"
  platform: darwin,windows,linux
  interval: 3600
  observer_can_run: false
  automations_enabled: false
- name: Codex approval and sandbox settings
  description: Lists the approval, sandbox, and web search lines in users' Codex config files.
  query: "SELECT f.path, TRIM(l.line) AS setting FROM file f CROSS JOIN file_lines l WHERE (f.path LIKE '/Users/%/.codex/config.toml' OR f.path LIKE '/home/%/.codex/config.toml' OR f.path LIKE 'C:\\Users\\%\\.codex\\config.toml') AND l.path = f.path AND (TRIM(l.line) LIKE 'approval_policy%' OR TRIM(l.line) LIKE 'sandbox_mode%' OR TRIM(l.line) LIKE 'web_search%');"
  platform: darwin,windows,linux
  interval: 86400
  observer_can_run: false
  automations_enabled: false
- name: Codex CLI accounts
  description: Lists the accounts or sign-in methods Codex CLI sessions used in the last 30 days.
  query: "SELECT DISTINCT username, account_identifier, agent_version FROM ai_agent_sessions WHERE agent = 'codex_cli';"
  platform: darwin,windows,linux
  interval: 86400
  observer_can_run: false
  automations_enabled: false
```

The `file_lines` report matches lines anywhere in the file, including under a `[profiles.<name>]` table. Codex also reads `config.toml` from `CODEX_HOME` when a user sets that variable, which fleetd can't see.

Then review the results on a schedule:

1. Open the MCP servers report and look for servers you haven't decided on. Check `endpoint`, `command`, and `risk_flags` first.
2. Add a server you approve to `mcp_servers`, using its `name` as the key and its `endpoint` as `url`, or its `command` as `command`.
3. Look for `danger-full-access` or `never` in the settings report and `skip_permissions_runtime` in the installs report. These show who tried to run Codex without guardrails, and your requirements now block them.
4. If you see an `API KEY` account and you only want ChatGPT sign-in, add `allowed_login_methods = ["chatgpt"]` to your requirements.
5. Update the profile, scripts, and hashes, then open a pull request.

## Troubleshoot

**`/debug-config` doesn't list your requirements**

On Linux and Windows, check that the file exists at the path for the host's OS. On macOS, check that the profile shows as **Verified** on the host's details page. If your organization uses Agent security, a requirement set there replaces the same key from the system file. See the warning in Prerequisites.

**Codex reports an error at startup after a deployment**

The requirements aren't valid TOML, or a key has a value Codex doesn't accept. Check the file, then redeploy. On macOS, decode the profile's string with `echo '<string>' | base64 -d` to check what you deployed.

**An allowed MCP server is disabled**

Check that the name in the user's `config.toml` matches the key in `mcp_servers` exactly, and that the URL or command matches the identity. A `url` string matches only that exact URL. Use `match = "prefix"` to allow every path under a host.

**The policy never passes on Windows**

Check that the hash in the policy is lowercase. osquery reports `sha256` in lowercase, and `Get-FileHash` returns uppercase unless you call `.ToLower()`.

## Further reading

- [Codex managed configuration](https://developers.openai.com/codex/enterprise/managed-configuration): OpenAI's guide to requirements, precedence, and macOS managed preferences.
- [Codex configuration reference](https://developers.openai.com/codex/config-reference): every key in `config.toml` and `requirements.toml`.
- [Manage Claude Code and Claude Desktop settings with Fleet](https://fleetdm.com/guides/manage-claude-code-and-claude-desktop-settings)
- [Manage Cursor settings with Fleet](https://fleetdm.com/guides/manage-cursor-settings-with-fleet)
- [Manage VS Code settings with Fleet](https://fleetdm.com/guides/manage-vs-code-settings-with-fleet)
- [Automatically run scripts](https://fleetdm.com/guides/policy-automation-run-script)

<meta name="articleTitle" value="Manage Codex settings with Fleet">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="publishedOn" value="2026-11-23">
<meta name="category" value="guides">
<meta name="description" value="Enforce Codex requirements with Fleet using a macOS profile and scripts, keep hosts in line, and review MCP servers and settings users configure.">
