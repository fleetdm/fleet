# Shadow AI is already on your fleet. Here's how to see it.

*AI tooling lands on endpoints faster than teams can see it, let alone govern it. Here's how to find shadow AI across macOS, Windows, and Linux, and act on what you find.*

## Key takeaways

- **Shadow AI is an endpoint problem your current tools miss.** AI tooling leaves its footprint on disk and in process lists, but an identity provider or SaaS catalog never sees it because it was never a sanctioned app, and EDR waves it through because a signed assistant wiring up an MCP server isn't an attack. You find it by inventorying what's on the device, not by waiting for a malicious-behavior alert.

- **Standardizing on one AI vendor doesn't contain the risk.** The fastest-moving agentic work happens outside that boundary, in native desktop apps, IDE forks, and the command line, so the boundary you drew doesn't cover what you're actually worried about.

- **Fleet sees it across every OS, in real time.** Fleet's agent turns macOS, Windows, and Linux devices into a live database you can query for an instant, fleet-wide picture of what AI tooling is running, not a once-a-day snapshot.

- **The agent is open source and transparent.** You can read exactly what Fleet collects and what it doesn't — which matters most when the machines you're auditing belong to the engineers most likely to scrutinize your tooling.

- **Governance is code, not console clicks.** Reports and policies live in Git as YAML, get reviewed in a pull request, and deploy through CI, so your AI governance posture is auditable and reversible instead of a click someone made six months ago.

- **One platform takes you from "see it" to "govern it."** The same tool that finds shadow AI rolls it into software inventory, matches it against CVE data, and lets you patch, enforce, and remediate across the fleet.

<a purpose="cta-button" href="/reports">Explore the reports library</a>

I've spent the last few months talking with IT and security teams about AI tooling on their endpoints, and the pattern is consistent: adoption is running well ahead of anyone's ability to see it, let alone govern it. It feels like the early days of SaaS and shadow IT, only faster and with far fewer guardrails. A developer can install an AI coding assistant, wire it up to a handful of MCP servers, and hand an agent real access to code, credentials, and internal systems — all before lunch, no help ticket required.

Standardizing inside a single vendor's AI ecosystem feels safe, but most of the genuinely transformative work — agentic development, autonomous coding — is happening outside that boundary, in native desktop apps, in IDE forks like Cursor, and on the command line. So teams are stuck on two questions: what's already running in our environment, and how do we adopt agentic development without taking on uncontrolled risk? You can't answer either from a dashboard that only knows about sanctioned apps. You answer it from the endpoint, where the tools actually live.

## Why this is an endpoint problem

AI tooling leaves a very specific footprint on a device: an installed app, a CLI binary, an IDE extension, a browser extension, a config file pointing at one or more MCP servers, sometimes a local server listening on a port. None of that shows up reliably in an identity provider or a SaaS catalog, because it was never a sanctioned app to begin with. And EDR tends to let it pass, because a signed AI assistant wiring up an MCP server isn't an attack — it's software doing exactly what it was installed to do. It shows up on disk and in process lists, which is why finding it takes an inventory of what's there, not an alert on what's malicious.

That's the layer Fleet operates at. Fleet's agent turns every macOS, Windows, and Linux device into a live database you can ask questions of and run reports against, in real time. AI governance shouldn't stop at your Macs, and with Fleet it doesn't.

Three things matter when the machines you're inventorying belong to developers. First, the agent is open source, so anyone can read what it collects and what it doesn't — "trust us, it's a black box" is not an answer for the engineers most likely to scrutinize your tooling. Second, live queries return results from every host right now instead of on a daily collection cycle, which is the whole game when a new extension CVE drops on a Friday afternoon. And because Fleet is API-first and GitOps-native, every policy and report can live in a Git repo as YAML, get reviewed in a pull request, and deploy through CI — governance you can audit and roll back, not an undocumented console edit from six months ago.

## A starter pack: reports to find AI tooling on your fleet

Here's a set of reports you can run today to get a picture of what's actually out there. Each one is a standard Fleet query that you can run live for an instant snapshot, or save it as a scheduled report or policy to keep watch over time.

### 1. Find MCP servers running on a host

MCP (Model Context Protocol) is how agents get hands. It's the connective tissue between an AI client and the tools, data, and systems it can act on. A running MCP server is one of the clearest signals that agentic tooling is live on a machine. Fleet's `mcp_listening_servers` table probes local listening ports and reports the ones responding as MCP servers over HTTP.

```sql
SELECT * FROM mcp_listening_servers;
```

This catches HTTP-transport servers. Many MCP servers run over stdio and won't bind a port, which is exactly why the next one matters.

### 2. Read MCP client configurations (Claude Desktop, Claude Code, Cursor, and more)

Instead of looking for running processes, this reads the config files where AI clients declare which MCP servers they're wired up to across Claude Desktop, Claude Code, Cursor, VS Code (including RooCode and Augment), Windsurf, Gemini CLI, and LM Studio. It tells you not just that a tool is installed, but what you've effectively granted an agent access to.

```sql
WITH path_suffixes(path) AS (
  VALUES
    ('/.cursor/mcp.json'),                                                   -- Cursor
    ('/Library/Application Support/Claude/claude_desktop_config.json'),      -- Claude Desktop (macOS)
    ('/.claude.json'),                                                       -- Claude Code
    ('/Library/Application Support/Code/User/mcp.json'),                     -- VS Code (macOS)
    ('/.config/Code/User/mcp.json'),                                         -- VS Code (Linux)
    ('/.gemini/settings.json'),                                              -- Gemini CLI
    ('/.lmstudio/mcp.json')                                                  -- LM Studio
),
full_paths AS (
  SELECT u.directory || p.path AS full_path, p.path AS suffix
  FROM users u
  JOIN path_suffixes p ON 1=1
),
config_files AS (
  SELECT f.path, group_concat(f.line, '') AS contents
  FROM file_lines f
  JOIN full_paths fp ON f.path = fp.full_path
  GROUP BY f.path
)
SELECT cf.path,
       je.key   AS name,
       je.value AS mcp_config
FROM config_files cf
JOIN json_each(
  COALESCE(json_extract(cf.contents, '$.mcpServers'),
           json_extract(cf.contents, '$.servers'))
) AS je;
```

The full version of this report, including Windows paths and the newer VS Code extension config locations, lives at [fleetdm.com/reports/get-mcp-client-configurations](https://fleetdm.com/reports/get-mcp-client-configurations).

### 3. Inventory IDE extensions

IDE extensions are a fast-growing attack surface and a primary vector for AI tooling. The `vscode_extensions` table enumerates installed extensions, and its `vscode_edition` column distinguishes stock VS Code from forks like Cursor, so you can see AI-first editors specifically.

```sql
SELECT u.username,
       e.name,
       e.publisher,
       e.version,
       e.vscode_edition
FROM users u
CROSS JOIN vscode_extensions e USING (uid);
```

Want only the AI-editor forks? Add `WHERE e.vscode_edition IN ('cursor')`.

### 4. Inventory browser extensions

The browser is the other place AI tooling quietly accumulates things like assistants, autofill agents, and "summarize this page" extensions that request broad permissions.

**Chromium-family browsers** (Chrome, Edge, Brave, Opera, Yandex):

```sql
SELECT u.username,
       e.name,
       e.identifier,
       e.version,
       e.from_webstore,
       e.permissions
FROM users u
CROSS JOIN chrome_extensions e USING (uid);
```

**Firefox:**

```sql
SELECT u.username,
       f.name,
       f.identifier,
       f.creator,
       f.version,
       f.autoupdate
FROM users u
CROSS JOIN firefox_addons f USING (uid)
WHERE f.active = '1';
```

**Safari** (macOS):

```sql
SELECT u.username,
       s.name,
       s.identifier,
       s.version,
       s.path
FROM users u
CROSS JOIN safari_extensions s USING (uid);
```

Across all three, the `permissions` and `from_webstore` fields (and the equivalents) are where to focus: a sideloaded extension with broad host permissions is worth a closer look.

### 5. Catch the apps and install folders themselves

Some tools won't expose an MCP config or a listening port so you might just want to know they're installed. On macOS, the `apps` table covers native applications and the `file` table lets you pick up CLI tools and config directories that don't register as apps, like Claude Code's `~/.claude` directory.

```sql
-- Native AI apps on macOS
SELECT name, bundle_identifier, bundle_short_version, path
FROM apps
WHERE name LIKE '%Claude%'
   OR name LIKE '%Cursor%'
   OR name LIKE '%Windsurf%'
   OR name LIKE '%Ollama%'
   OR name LIKE '%LM Studio%';

-- Claude Code's install/config footprint, per user
SELECT path FROM file
WHERE path LIKE '/Users/%/.claude/%'
   OR path LIKE '/Users/%/.claude.json';
```

Swap in the names and paths that matter to your environment.

The point is that once a tool is on disk, Fleet can find it.

## From "see it" to "govern it"

Visibility is step one. The reason Fleet is useful here is that the same platform takes you the rest of the way.

**Software detection.** Everything those reports surface — apps, packages, browser plugins, and IDE extensions — rolls up into Fleet's software inventory automatically. You get one searchable, cross-platform view of what's installed everywhere, with no separate collection tool to deploy and maintain.

**Vulnerability management.** Fleet matches your installed software against published CVE data and surfaces which hosts are exposed to which vulnerabilities. And when a brand-new CVE is announced, you don't wait! You run a live query and get an answer across the fleet in seconds.

**Patching and enforcement.** Detection without remediation is just a nicer-looking spreadsheet. Fleet lets you turn findings into action: deploy and update software through software installers and Fleet-maintained apps, enforce minimum OS versions with a grace period before enforcement kicks in, and run scripts on macOS, Windows, and Linux to remediate at scale. Pair a report that detects a disallowed AI tool with a policy that flags or remediates it, and you've closed the loop.

**Policies as code.** Turn any of the reports above into a policy, store it in Git, and review changes in a pull request. Onboarding a new sanctioned AI tool, or retiring a risky one, becomes a reviewable, reversible change instead of an undocumented console edit.

## Enable it, don't just block it

Agentic development is genuinely worth adopting. The teams getting real leverage from it aren't the ones who blocked everything; they're the ones who got visibility first, set sane guardrails, and then said yes deliberately. You can't make that bet responsibly if you can't answer "what's already running, and what systems is it capable of accessing?"

That's the role we think endpoint management should play in AI governance: give you a true, current, cross-platform picture of AI tooling on every device, and the controls to act on it, without a black box and without standing up yet another point solution.

## See it live

The fastest way to see what this looks like in your environment is to run the reports above against your own devices. If you'd like a hand getting there, two good next steps:

- [**Get a demo**](https://fleetdm.com/contact)**.** We'll walk through seeing, controlling, and governing AI tooling at scale across your fleet and answer the "what's actually running in *our* environment?" question against real machines.
- [**Join a GitOps training session**](https://fleetdm.com/gitops-workshop)**.** If you want to manage AI governance as code — reports and policies in Git, reviewed in pull requests, deployed through CI — our hands-on workshop is the place to start.

If shadow AI is on your mind, and it should be, either one is a solid first move.

*Fleet is the open-source endpoint management platform for macOS, Windows, Linux, and more. Want to try these reports on your own fleet?* [*Get a demo*](https://fleetdm.com/contact) *or explore the* [*reports library*](https://fleetdm.com/reports)*.*

<meta name="articleTitle" value="Shadow AI is already on your fleet. Here's how to see it.">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-06-10">
<meta name="description" value="AI tooling lands on endpoints faster than teams can see it. Here are Fleet reports to find shadow AI across macOS, Windows, and Linux.">
