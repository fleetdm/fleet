# ServiceNow's AI Passport defines what an agent may do, and leaves open which agents are installed on your devices

*A signed credential can limit an agent that knows to ask for one. It can't tell you about the agents running on laptops that nobody registered.*

## Key takeaways

- **Authorization is moving from policy documents to runtime checks.** ServiceNow's AI Passport, announced October 15, 2026, has agents and MCP servers check a signed credential for their scope while they run, with a kill switch as the last resort. Watching behavior continuously beats writing a policy once.
- **A passport covers agents that carry one.** It's issued to agents ServiceNow's platform knows about, including third-party agents. An agent a developer installed on a laptop last week isn't in that picture until someone finds it.
- **Discovery is the step before authorization.** You can't scope what you haven't listed. The first question is which AI tools exist on which devices.
- **Fleet answers that for the device.** Fleet's agent reports MCP servers, agent CLIs, desktop apps, IDE plugins, and live AI connections on macOS, Windows, and Linux, with risk flags for what each was granted.
- **Compare what's declared with what's running.** A passport states intent. A device report shows what's installed and connected right now, and the gap between the two is the finding.
- **Plan for Q4, not for the announcement.** AI Passport isn't generally available yet, so the work worth doing now is the inventory it will depend on.

<a purpose="cta-button" href="https://fleetdm.com/guides/get-current-telemetry-from-your-devices-with-live-queries">See live reports in Fleet</a>

On October 15, 2026, ServiceNow announced AI Passport, a cryptographically signed credential that sets what AI agents and MCP servers are permitted to do. Nvidia is the launch partner, and ServiceNow expects the feature in the fourth quarter of 2026. [Techzine](https://www.techzine.eu/news/applications/145011/servicenow-provides-ai-agents-with-a-passport-for-runtime-control/) reports that the Super Intelligence Control Tower issues passports based on the CMDB, GRC policies, and workflow approvals.

The idea is sound: treat an agent's permissions as something to enforce while it runs, not something to define once and trust. It also raises a plain question for the people who manage devices. Before an agent can carry a passport, someone has to know it exists.

## What the passport does

According to the announcement coverage, an agent consults its passport at runtime to confirm its authorized scope. Inside those limits it works freely. If it drifts, the platform can redirect it, restrict it, or give it the right context again, and a kill switch is the last resort. The design is open, so other governance tools and identity providers can issue and verify passports too.

ServiceNow describes this as prevention rather than an after-the-fact audit trail. Coverage notes that the announcement doesn't publish low-level protocol details yet, and the kill switch and the MCP integration aren't described in technical depth, so treat the specifics as unproven until the documentation ships.

## The agents a passport never sees

A passport helps most when the agent is registered with the platform that issues it. Plenty of agent software doesn't arrive that way. An engineer installs a coding assistant, adds a few MCP servers, and points it at a repository and a credentials file. No ticket is filed, and no control tower is told.

That isn't a flaw in the passport. It's a scope. The control tower sees what is connected to it, and your devices hold what isn't.

## Ask the device first

Fleet's agent includes an `ai_tools` table that returns one row per AI tool on a host: desktop apps, IDE plugins, agent CLIs, MCP servers, live AI and MCP sockets, agent instruction files, and browser extensions. Start with the shape of it:

```sql
SELECT type, count(*) FROM ai_tools GROUP BY type;
```

Then look at what each tool was allowed to do:

```sql
SELECT type, name, risk_flags, path FROM ai_tools WHERE risk_flags != '';
```

Flags such as `mcp_shell_exec` and `mcp_fs_write` mean an MCP server was granted shell execution or filesystem writes, and `bypass_permissions` means someone turned a guardrail off. Two limits apply. Process and socket data is a snapshot from the moment the report runs, so a tool that starts and exits between runs leaves no row. And the table reads every home directory only when it runs as root or SYSTEM, so confirm that before you read an empty result as an absence.

## Compare declared scope with running reality

Once AI Passport ships, the useful exercise is a comparison. One list is what your governance platform believes each agent may do. The other is what the devices show: which MCP servers are running, what they're connected to, and which have shell or write access. An agent on the second list but not the first has no passport to check. An agent on both with broader access on the device than on paper is the one to investigate.

[Reports](https://fleetdm.com/guides/get-current-telemetry-from-your-devices-with-live-queries) give you the device side now, and [policies](https://fleetdm.com/guides/how-to-use-policies-for-patch-management-in-fleet) let you flag hosts that match a pattern you don't want, such as an MCP server with shell execution enabled.

## What to do before Q4

Build the inventory first. List the AI tools on every managed device, decide which risk flags you won't accept, and name an owner for anything flagged. When the passport arrives, you'll have a device-level baseline to check it against instead of a control plane's account of itself.

A kill switch is a good last resort. It works best when you already know what it would be switching off.

## See it live

- [**Get a demo**](https://fleetdm.com/contact) to see MCP servers and AI tools reported live from real devices.
- [**Join a GitOps training session**](https://fleetdm.com/gitops-workshop) to learn how to keep reports and policies like these reviewed in pull requests.

<meta name="articleTitle" value="ServiceNow's AI Passport defines what an agent may do, and leaves open which agents are installed on your devices">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-10-16">
<meta name="description" value="ServiceNow's AI Passport scopes agents at runtime. Inventory the AI tools and MCP servers on your devices first, and compare them with what's declared.">
