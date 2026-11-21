# Limit AI agents to compliant devices with conditional access

AI coding agents like Claude Code, Codex, GitHub Copilot CLI, and Cursor sign in as the user. When that sign-in goes through your identity provider (IdP), Fleet's [conditional access](https://fleetdm.com/guides/conditional-access) blocks it on hosts that aren't managed by Fleet or are failing your policies. This guide puts each agent behind your IdP and adds policies that catch the ways agents skip it, like plaintext API keys and MCP tokens. It also gives you scripts to stop agents on a host and restore them later.

How it works:

1. Each AI tool requires single sign-on (SSO) through your IdP.
2. Your IdP only allows sign-in to those tools from hosts that Fleet reports as compliant.
3. Fleet policies fail when a host stores credentials that let an agent skip the IdP.
4. When a host fails, the user can't sign in to the agent until they fix the issue.

> **Note:** Conditional access is checked at sign-in. An agent that signed in before its host started failing keeps working until its session expires. To cut off access right away, [stop agents on the host](#stop-and-restore-agents) and revoke the user's sessions in the AI tool's admin console.

## Prerequisites

- Fleet Premium.
- Conditional access set up with your IdP:
  - [Okta](https://fleetdm.com/guides/okta-conditional-access-integration) (macOS)
  - [Entra](https://fleetdm.com/guides/entra-conditional-access-integration) (macOS and Windows)
  - [PingFederate](https://fleetdm.com/guides/pingfederate-conditional-access-integration) (macOS, Windows, and Linux)
  - [Duo](https://fleetdm.com/guides/duo-conditional-access-integration) (macOS, Windows, and Linux)
- Admin access to each AI tool your organization uses, on a plan that supports SSO.
- Scripts turned on in fleetd, to stop and restore agents.

> **Note:** Duo blocks hosts that aren't managed by Fleet. Blocking hosts that fail policies needs extra setup, covered in the [Duo guide](https://fleetdm.com/guides/duo-conditional-access-integration). [Google](https://fleetdm.com/guides/google-conditional-access-integration) conditional access covers iOS, iPadOS, and Android, so it doesn't apply to agents on computers.

## Step 1: Find agents on your hosts

Start with a report of which agents are on which hosts, so you know which AI tools to put behind your IdP.

1. In Fleet, head to **Reports** and select **Add report**.
2. Use this query, target macOS, Windows, and Linux, and select **Save**:

```sql
WITH agents(agent, suffix) AS (
  VALUES
    ('Claude Code', '/.claude'),
    ('Codex', '/.codex'),
    ('Copilot CLI', '/.copilot'),
    ('Cursor', '/.cursor'),
    ('Gemini CLI', '/.gemini'),
    ('Windsurf', '/.codeium/windsurf')
),
paths AS (
  SELECT u.username, a.agent,
    CASE WHEN instr(u.directory, '\') > 0
      THEN replace(u.directory || a.suffix, '/', '\')
      ELSE u.directory || a.suffix END AS path
  FROM users u CROSS JOIN agents a
)
SELECT p.username, p.agent, f.path, datetime(f.mtime, 'unixepoch') AS last_modified
FROM paths p JOIN file f ON f.path = p.path;
```

Each agent keeps its settings in a folder in the user's home folder, so the report finds agents whether they were installed with npm, Homebrew, or a native installer.

> **Note:** An agent's folder stays after the agent is uninstalled. Use `last_modified` to spot agents that haven't run in a while.

To block agents you haven't approved, add a policy that fails when one is found. Use the same `agents` and `paths` lists, list your approved agents in `NOT IN`, and replace the last two lines with:

```sql
SELECT 1 WHERE NOT EXISTS (
  SELECT 1 FROM paths p JOIN file f ON f.path = p.path
  WHERE p.agent NOT IN ('Claude Code', 'Cursor')
);
```

## Step 2: Require SSO for each AI tool

Conditional access only sees sign-ins that go through your IdP. If users can sign in to an AI tool with a password or a personal account, the agent skips your IdP.

For each AI tool from Step 1:

1. In the tool's admin console, connect SSO to your IdP. Each vendor's docs list the plans that include SSO.
2. Require SSO for your company's email domains, so users can't sign in with a password.
3. In your IdP, assign the tool's app to the users who need it.

## Step 3: Add AI tools to your conditional access policy

Apply your IdP's conditional access policy to each AI tool's app:

- Okta: go to **Applications**, select the app, and on the **Sign On** tab, set **Authentication policy** to the policy that requires Fleet. See [Step 6 of the Okta guide](https://fleetdm.com/guides/okta-conditional-access-integration#step-6-add-fleet-to-an-authentication-policy).
- Entra: add each app to the target resources of the Conditional Access policy that blocks hosts Fleet reports as non-compliant. See [Step 6 of the Entra guide](https://fleetdm.com/guides/entra-conditional-access-integration#step-6-add-entra-policies).
- PingFederate: use the authentication policy from the [PingFederate guide](https://fleetdm.com/guides/pingfederate-conditional-access-integration) for each app's connection.
- Duo: protect each app with a Duo policy that requires trusted endpoints. See the [Duo guide](https://fleetdm.com/guides/duo-conditional-access-integration).

## Step 4: Add policies that catch IdP bypasses

An agent that uses an API key never signs in through your IdP, so conditional access can't block it. The same goes for tokens in an MCP server's config. These policies fail when a host stores keys in plaintext.

For each policy, head to **Policies**, select the fleet, select **Add policy**, paste the query, and select **Save**.

### No plaintext AI or MCP credentials

This policy works on macOS, Windows, and Linux. It checks shell profiles, agent settings (Claude Code, Codex, Copilot CLI, and Gemini CLI), and MCP configs (Claude Desktop, Cursor, and VS Code) for Anthropic, OpenAI, GitHub, GitLab, and Slack keys:

```sql
WITH files(suffix) AS (
  VALUES
    ('/.zshrc'), ('/.zprofile'), ('/.zshenv'), ('/.bashrc'), ('/.bash_profile'), ('/.profile'),
    ('/.config/fish/config.fish'),
    ('/Documents/PowerShell/Microsoft.PowerShell_profile.ps1'),
    ('/Documents/WindowsPowerShell/Microsoft.PowerShell_profile.ps1'),
    ('/.claude.json'), ('/.claude/settings.json'),
    ('/.codex/auth.json'), ('/.codex/config.toml'),
    ('/.copilot/mcp-config.json'),
    ('/.cursor/mcp.json'),
    ('/.gemini/settings.json'),
    ('/Library/Application Support/Claude/claude_desktop_config.json'),
    ('/AppData/Roaming/Claude/claude_desktop_config.json'),
    ('/Library/Application Support/Code/User/mcp.json'),
    ('/.config/Code/User/mcp.json'),
    ('/AppData/Roaming/Code/User/mcp.json')
),
paths AS (
  SELECT CASE WHEN instr(u.directory, '\') > 0
    THEN replace(u.directory || f.suffix, '/', '\')
    ELSE u.directory || f.suffix END AS path
  FROM users u CROSS JOIN files f
),
existing AS (
  SELECT f.path FROM paths p JOIN file f ON f.path = p.path
)
SELECT 1 WHERE NOT EXISTS (
  SELECT 1 FROM existing e JOIN file_lines l ON l.path = e.path
  WHERE l.line GLOB '*sk-ant-*'
     OR l.line GLOB '*sk-proj-*'
     OR l.line GLOB '*sk-svcacct-*'
     OR l.line GLOB '*ghp_*'
     OR l.line GLOB '*github_pat_*'
     OR l.line GLOB '*glpat-*'
     OR l.line GLOB '*xox[bp]-*'
);
```

The `existing` step matters: `file_lines` fails the whole query if any path doesn't exist, so the policy checks each path with the `file` table first.

For the resolution, tell users to remove the key from the file and sign in to the agent with their company account. Codex saves an API key in `~/.codex/auth.json` when a user signs in with one, so signing in again with ChatGPT fixes it.

> **Note:** The policy matches key formats, not variable names. A key loaded at runtime, like `export OPENAI_API_KEY=$(op read ...)`, passes. That key still skips your IdP, so also limit who can create API keys in each AI tool's admin console. Add a `GLOB` line for any other service your MCP servers use.

### No AI keys in Windows environment variables

On Windows, users often save keys with `setx`, which writes them to the registry instead of a file. This policy fails when a user's environment variables hold an Anthropic, OpenAI, or GitHub key:

```sql
SELECT 1 WHERE NOT EXISTS (
  SELECT 1 FROM registry
  WHERE path LIKE 'HKEY_USERS\%\Environment\%'
    AND (data GLOB 'sk-ant-*'
      OR data GLOB 'sk-proj-*'
      OR data GLOB 'sk-svcacct-*'
      OR data GLOB 'ghp_*'
      OR data GLOB 'github_pat_*')
);
```

## Step 5: Block sign-in when policies fail

> **Warning:** When a host fails a conditional access policy, the user loses access to every app behind conditional access, not just AI tools. Leave these policies running for a week before you turn on conditional access, so you can see how many hosts fail and give users time to fix them.

- Okta and Entra: head to **Policies**, select the fleet, and go to **Manage automations** > **Conditional access**. Select the policies from Step 4 and save.
- PingFederate and Duo: open each policy from Step 4, select **Critical**, and select **Save**.

## Stop and restore agents

Use these scripts to stop agents on a host, for example during an incident, and restore them afterward:

- [`stop-ai-agents.sh`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/all/scripts/stop-ai-agents.sh) and [`restore-ai-agents.sh`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/all/scripts/restore-ai-agents.sh) for macOS and Linux
- [`stop-ai-agents.ps1`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/all/scripts/stop-ai-agents.ps1) and [`restore-ai-agents.ps1`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/all/scripts/restore-ai-agents.ps1) for Windows

The stop script renames the `claude`, `codex`, `copilot`, `cursor-agent`, and `gemini` commands and the Cursor app with a `.fleet-disabled` suffix, then stops any that are running. The restore script removes the suffix. On Linux, the stop script quits Cursor but doesn't rename it.

1. Head to **Controls** > **Scripts**, select the fleet, and upload all four scripts.
2. To stop agents on a host, head to **Hosts**, select the host, and select **Actions** > **Run script**. Run `stop-ai-agents.sh` or `stop-ai-agents.ps1`.
3. When the host is safe again, run `restore-ai-agents.sh` or `restore-ai-agents.ps1` the same way.

To stop agents automatically when a host fails a policy, head to **Policies**, select **Manage automations** > **Run script**, and pick the stop script for that policy.

> **Note:** A user who owns an install, like `~/.local/bin/claude`, can rename it back or reinstall the agent. The scripts stop agents right away. Conditional access keeps them from signing in again.

## Verify

1. On a host that's failing a policy you selected in Step 5, sign out of an agent and sign in again. Your IdP blocks the sign-in.
2. Fix the issue, then select **Refetch** on the host's **My device** page.
3. Sign in again. The sign-in works.

## Further reading

- [Conditional access](https://fleetdm.com/guides/conditional-access)
- [Shadow AI is already on your fleet](https://fleetdm.com/articles/shadow-ai-is-already-on-your-fleet)
- [Get MCP client configurations report](https://fleetdm.com/reports/get-mcp-client-configurations)

<meta name="articleTitle" value="Limit AI agents to compliant devices with conditional access">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-11-19">
<meta name="description" value="Block AI agents like Claude Code, Codex, and Cursor from reaching company data unless the host is managed by Fleet and passing policies.">
