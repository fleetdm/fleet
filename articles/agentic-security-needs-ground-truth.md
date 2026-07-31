# Agentic security is only as good as the device data underneath it

*AI agents are moving into security operations, and they will act on whatever picture of your devices you hand them. Here's what that picture has to get right.*

## Key takeaways

- **Agentic security tooling is shipping now, not next year.** Microsoft put a security-specific model and a multi-agent defense platform into public preview in early August 2026. For most IT and security teams, the open question is no longer whether agents show up in the workflow; it's what those agents are allowed to believe about your devices.

- **An agent inherits every gap in its input.** Give a remediation agent a day-old inventory, and it will act with confidence on a machine that changed this morning. Bad data plus autonomous action lands you somewhere worse than a slow human working from good data.

- **Ground truth has to be current, cross-platform, and inspectable.** Fleet queries macOS, Windows, and Linux devices on demand, so an agent reads what is true now rather than what was true at last night's collection run. fleetd is open source, so you can also read exactly what it collects, which matters more once an AI system is consuming that data and proposing action on it.

- **You can inventory the agents already running on your devices.** fleetd's `ai_tools` table returns one row per AI tool across every OS, covering MCP servers, agent CLIs, desktop apps, IDE plugins, sockets, and instruction files, with per-row risk flags describing what each one has been permitted to do.

- **Keep the write path narrow.** The useful pattern so far is read-only by default, with a human approving anything that changes state on a device. Validation before execution beats a well-worded prompt.

- **Governance-as-code gives agent-driven work an audit trail.** When reports and policies live in Git as YAML and change through pull requests, an agent's proposal arrives as a reviewable diff rather than an unlogged console edit that nobody can reconstruct later.

<a purpose="cta-button" href="/visibility-and-reporting">See what Fleet can tell you</a>

On July 27, Microsoft launched its first cybersecurity-specific model alongside an agentic defense platform, with public preview opening August 3. Whatever you make of the launch, the direction is clear enough: agents that triage, prioritize, and remediate are arriving in security operations, and they are arriving fast.

That shifts where the hard problem sits. The interesting question is not which model reasons best about a threat. It's what the model is reading when it decides. An agent's conclusion is a function of its input, and for anything touching devices, that input is your inventory.

## What just shipped

Microsoft released MAI-Cyber-1-Flash, a model built for security work, and Project Perception, a platform that splits work across red team agents that model adversary movement, blue team agents that identify and prioritize active threats, and green team agents that carry out remediation steps. Project Perception enters public preview on August 3.

Microsoft reports that the model, paired with GPT-5.4 inside its MDASH vulnerability management harness, scores 96% on the CyberGym benchmark at half the cost of its previous configuration, and says the model draws on more than 100 trillion daily security signals across identity, device, cloud, and network telemetry. Those are vendor figures on a vendor benchmark, so weigh them accordingly, but the architecture is the part worth paying attention to. Agents are being handed remediation, not just analysis.

Microsoft is not alone here, and this is not a knock on the approach. Compressing hours of specialist triage into minutes is a real gain. It also removes most of the slack that used to absorb bad input.

## An agent inherits your blind spots

A human analyst who reads a stale inventory usually notices. They recognize a decommissioned hostname, pause because the OS version looks wrong for that team, or open a terminal and check. That instinct is doing quiet, load-bearing work.

An agent working from the same stale record lacks such a reflex. It reads the row, believes it, and acts. If the record indicates a laptop is running a patched build and it rolled back last night, the agent closes the finding. If the record predates a contractor installing a local inference server, the agent never sees it. The failure is not that the model reasoned poorly. It reasoned correctly about a world that no longer exists.

This is why "we already have an inventory" is not the same as being ready. Most inventories were built for reporting, where a daily snapshot is fine, and a slightly stale row costs nothing. Feeding an autonomous remediation loop is a different job with a different tolerance for age.

## What ground truth requires

Four properties matter once an AI system is reading your device data and acting on it.

### It has to be current

Device state changes constantly. Software gets installed, configuration drifts, a machine comes back from two weeks in a drawer. Fleet turns each enrolled device into a live database you can query on demand, so the answer reflects the device as it is now, not as it was at the last collection cycle. When an agent asks whether a fix landed, it should get today's answer.

### It has to cover every platform

Agentic workflows fall apart at the boundary of what you can see. If your visibility stops at macOS, every conclusion an agent draws about "the fleet" carries a silent asterisk. Fleet queries macOS, Windows, and Linux in the same way, which keeps the picture whole rather than stitched together from tools that disagree.

### It has to be inspectable

An AI system reading everything on every device is exactly the wrong place for "trust us." fleetd is open source, and the queries Fleet runs are visible as SQL. You can read what gets collected, show it to the engineers whose laptops you are inventorying, and answer questions about it without appealing to a vendor's word. The same transparency lets you review what an agent asked for, helping you catch a query that quietly overreaches.

### It has to answer questions nobody wrote a rule for

The hardest work happens before the catalog catches up. A proof of concept lands with no CVE assigned, and your scanner returns nothing, because there is nothing to match against yet. What you need, then, is not a better feed; it's the ability to describe the artifacts and go look: kernel versions, loaded modules, config files, listening ports. Fleet's live queries answer that shape of question in minutes across every host. A catalog lookup tells you which CVEs apply. An artifact query tells you what the machines look like right now, which is the one threat hunting depends on.

## Start with the agents you already have

Before you evaluate anyone's agent platform, it's worth knowing what agentic tooling is already running on your devices and what permissions it has been granted. That's harder to answer than it sounds, because this software arrives without a purchase order. An engineer installs an assistant, wires up a few MCP servers, and grants an agent access to code and credentials, and no help ticket is ever filed.

fleetd's `ai_tools` table answers it in one place. It returns one row per AI tool across macOS, Windows, and Linux, covering desktop apps, IDE plugins, agent CLIs, MCP servers, live AI and MCP sockets, agent instruction files, and browser extensions, with a `type` column to tell them apart. Start with the shape of what's out there:

```sql
SELECT type, count(*) FROM ai_tools GROUP BY type;
```

Then the column that carries the most weight for this argument, `risk_flags`:

```sql
SELECT type, name, risk_flags, path
FROM ai_tools
WHERE risk_flags != '';
```

Those flags describe what an agent on that device can reach and how carefully it was set up. `mcp_shell_exec` and `mcp_fs_write` mean that an MCP server was granted shell execution or filesystem writes. `bypass_permissions` and `skip_permissions_runtime` mean someone turned the guardrails off. `plaintext_secret` and `world_readable_config` mean credentials are sitting somewhere they shouldn't be. `injection_markers` and `hidden_unicode` point at instruction files worth reading closely.

That's the ground truth question pointed back at itself. You are deciding whether to let a vendor's agents act in your environment, and this tells you which agents are already acting in it, with what reach, and on whose authority.

Two more are worth running. Live MCP servers and how they're connected:

```sql
SELECT name, source AS client, location, running, pid
FROM ai_tools
WHERE type = 'mcp_server' AND running = 1;
```

And where data is leaving:

```sql
SELECT name, endpoint
FROM ai_tools
WHERE type = 'sockets' AND location = 'remote';
```

Two limits are worth knowing before you read an empty result as an absence. The extension enumerates every home directory on the host rather than only the daemon accounts', so running as root is what gives you full visibility across users. And because it runs as root or SYSTEM, it reads only regular files and does not follow symlinks, so a config or binary path managed by a dotfile tool is deliberately left unresolved.

This piece is about the data underneath agentic security, so it stops at inventory. For the fuller tour of finding AI tooling across a fleet, including the browser and IDE extension angles and what to do once you find something, that's [its own article](https://fleetdm.com/articles/shadow-ai-is-already-on-your-fleet).

## Keep the write path narrow

Reading device data is low risk. Changing device state is not, and the gap between those two is where agentic security either earns trust or loses it.

The pattern that has held up in practice is to expose a narrow, typed set of capabilities rather than a general-purpose shell. One worked example is [fleet-mcp](https://github.com/karmine05/fleet-mcp), an open-source MCP server built by Dhruv Majumdar, Fleet's VP of Security Solutions, which puts Fleet's API behind natural language for MCP-compatible clients. It's a community project rather than a supported Fleet product, but its design choices are the instructive part:

- Live queries run through a prepare step that validates the target set and fetches the schema before any SQL executes, so a hallucinated table name fails cheaply instead of firing at ten thousand hosts.
- Queries run read-only. There is no tool to run arbitrary shell commands, nor to delete a host.
- Anything that changes state stays a proposal for a human to approve.
- Every answer ships with the SQL that produced it, because an answer you cannot review is an answer you cannot act on.

None of that comes from prompt wording. It comes from what the tool surface does and does not expose. If you wire an agent into a workflow that genuinely needs to run scripts, keep the approval gate. The discipline belongs in the boundary, not the instructions.

## Give agent-driven change an audit trail

Six months from now, someone will ask why a policy changed. "An agent suggested it, and someone clicked yes" is not an answer you want to give a regulator or a colleague.

Because Fleet is API-first and GitOps-native, reports and policies can live in a Git repository as YAML, be reviewed in a pull request, and be deployed via CI. An agent proposing a new detection or a tightened policy produces a diff with an author, a reviewer, a timestamp, and a revert path. That turns agent-assisted operations into something you can audit and roll back, which is a clearer and more durable answer than any prompt log.

## Get the substrate right first

Agentic security is worth adopting, and the teams that get real use from it will not be the ones that wait. They will be the ones whose device data was already up to date, cross-platform, and honest about what it does not know, so the agents had something solid to stand on.

That's the role we think device management plays here. Not another layer of AI on top of the stack, but a live, inspectable, queryable picture of every device, plus controls narrow enough that handing an agent partial access is a decision you can defend.

## See it live

- [**Get a demo**](https://fleetdm.com/contact)**.** We'll run live queries against real machines and show you what the picture looks like before you point anything automated at it.
- [**Join a GitOps training session**](https://fleetdm.com/gitops-workshop)**.** If you want policies and reports reviewed in pull requests before agents start proposing changes to them, this is where to start.

*Fleet is the open-source device management platform for macOS, Windows, Linux, and more. Want to see what your devices would tell an agent?* [*Get a demo*](https://fleetdm.com/contact) *or explore the* [*reports library*](https://fleetdm.com/reports)*.*

<meta name="articleTitle" value="Agentic security is only as good as the device data underneath it">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-07-30">
<meta name="description" value="AI agents are moving into security operations. Here's what your device data has to get right before you let one act on it.">