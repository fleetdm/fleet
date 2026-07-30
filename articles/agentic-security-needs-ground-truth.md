# Agentic security is only as good as the device data underneath it

*AI agents are moving into security operations, and they will act on whatever picture of your devices you hand them. Here's what that picture has to get right.*

## Key takeaways

- **Agentic security tooling is shipping now, not next year.** Microsoft put a security-specific model and a multi-agent defense platform into public preview in early August 2026. For most IT and security teams the open question is no longer whether agents show up in the workflow, it's what those agents are allowed to believe about your devices.

- **An agent inherits every gap in its input.** Give a remediation agent a day-old inventory and it will act with confidence on a machine that changed this morning. Bad data plus autonomous action lands you somewhere worse than a slow human working from good data.

- **Ground truth has to be current, not collected overnight.** Fleet queries macOS, Windows, and Linux devices on demand, so an agent can ask what is true right now instead of reading what was true at last night's collection run.

- **Nothing that reads every device should be a black box.** fleetd is open source, so you can read exactly what it collects and what it leaves alone. That matters more, not less, once an AI system is consuming that data and proposing action on it.

- **Keep the write path narrow.** The useful pattern so far is read-only by default, with a human approving anything that changes state on a device. Validation before execution beats a well-worded prompt.

- **Governance as code gives agent-driven work an audit trail.** When reports and policies live in Git as YAML and change through pull requests, an agent's proposal arrives as a reviewable diff instead of an unlogged console edit nobody can reconstruct later.

<a purpose="cta-button" href="/visibility-and-reporting">See what Fleet can tell you</a>

On July 27, Microsoft launched its first cybersecurity-specific model alongside an agentic defense platform, with public preview opening August 3. Whatever you make of the launch, the direction is clear enough: agents that triage, prioritize, and remediate are arriving in security operations, and they are arriving fast.

That shifts where the hard problem sits. The interesting question is not which model reasons best about a threat. It's what the model is reading when it decides. An agent's conclusion is a function of its input, and for anything touching devices, that input is your inventory.

## What just shipped

Microsoft released MAI-Cyber-1-Flash, a model built for security work, and Project Perception, a platform that splits work across red team agents that model adversary movement, blue team agents that identify and prioritize active threats, and green team agents that carry out remediation steps. Project Perception enters public preview on August 3.

Microsoft reports that the model, paired with GPT-5.4 inside its MDASH vulnerability management harness, scores 96% on the CyberGym benchmark at half the cost of its previous configuration, and says the model draws on more than 100 trillion daily security signals across identity, device, cloud, and network telemetry. Those are vendor figures on a vendor benchmark, so weigh them accordingly, but the architecture is the part worth paying attention to. Agents are being handed remediation, not just analysis.

Microsoft is not alone here, and this is not a knock on the approach. Compressing hours of specialist triage into minutes is a real gain. It also removes most of the slack that used to absorb bad input.

## An agent inherits your blind spots

A human analyst who reads a stale inventory usually notices. They recognize a hostname that was decommissioned, or pause because the OS version looks wrong for that team, or open a terminal and check. That instinct is doing quiet, load-bearing work.

An agent working from the same stale record has no such reflex. It reads the row, believes it, and acts. If the record says a laptop is running a patched build and the laptop rolled back last night, the agent closes the finding. If the record predates a contractor installing a local inference server, the agent never sees it. The failure is not that the model reasoned poorly. It reasoned correctly about a world that no longer exists.

This is why "we already have an inventory" is not the same as being ready. Most inventories were built for reporting, where a daily snapshot is fine and a slightly stale row costs nothing. Feeding an autonomous remediation loop is a different job with a different tolerance for age.

## What ground truth requires

Four properties matter once an AI system is reading your device data and acting on it.

### It has to be current

Device state changes constantly. Software gets installed, configuration drifts, a machine comes back from two weeks in a drawer. Fleet turns each enrolled device into a live database you can query on demand, so the answer reflects the device as it is now rather than as it was at the last collection cycle. When an agent asks whether a fix landed, it should get today's answer.

### It has to cover every platform

Agentic workflows fall apart at the boundary of what you can see. If your visibility stops at macOS, every conclusion an agent draws about "the fleet" carries a silent asterisk. Fleet queries macOS, Windows, and Linux the same way, which keeps the picture whole instead of stitched together from tools that disagree.

### It has to be inspectable

An AI system reading everything on every device is exactly the wrong place for "trust us." fleetd is open source, and the queries Fleet runs are visible as SQL. You can read what gets collected, show it to the engineers whose laptops you are inventorying, and answer questions about it without appealing to a vendor's word. The same transparency lets you review what an agent asked for, which is how you catch a query that quietly overreaches.

### It has to answer questions nobody wrote a rule for

The hardest work happens before the catalog catches up. A proof of concept lands with no CVE assigned and your scanner returns nothing, because there is nothing to match against yet. What you need then is not a better feed, it's the ability to describe the artifacts and go look: kernel versions, loaded modules, config files, listening ports. Fleet's live queries answer that shape of question in minutes across every host. A catalog lookup tells you which CVEs apply. An artifact query tells you what the machines look like right now, which is the one threat hunting depends on.

## Keep the write path narrow

Reading device data is low risk. Changing device state is not, and the gap between those two is where agentic security either earns trust or loses it.

The pattern that has held up in practice is to expose a narrow, typed set of capabilities rather than a general-purpose shell. One worked example is [fleet-mcp](https://github.com/karmine05/fleet-mcp), an open-source MCP server built by Dhruv Majumdar, Fleet's VP of Security Solutions, which puts Fleet's API behind natural language for MCP-compatible clients. It's a community project rather than a supported Fleet product, but its design choices are the instructive part:

- Live queries run through a prepare step that validates the target set and fetches the schema before any SQL executes, so a hallucinated table name fails cheaply instead of firing at ten thousand hosts.
- Queries run read-only. There is no tool for running arbitrary shell commands and no tool for deleting a host.
- Anything that changes state stays a proposal for a human to approve.
- Every answer ships with the SQL that produced it, because an answer you cannot review is an answer you cannot act on.

None of that comes from prompt wording. It comes from what the tool surface does and does not expose. If you wire an agent into a workflow that genuinely needs to run scripts, keep the approval gate. The discipline belongs in the boundary, not the instructions.

## Give agent-driven change an audit trail

Six months from now, someone will ask why a policy changed. "An agent suggested it and someone clicked yes" is not an answer you want to give a regulator or a colleague.

Because Fleet is API-first and GitOps-native, reports and policies can live in a Git repository as YAML, get reviewed in a pull request, and deploy through CI. An agent proposing a new detection or a tightened policy produces a diff with an author, a reviewer, a timestamp, and a revert path. That turns agent-assisted operations into something you can audit and roll back, which is a plainer and more durable answer than any log of prompts.

## Get the substrate right first

Agentic security is worth adopting, and the teams that get real leverage from it will not be the ones that waited. They will be the ones whose device data was already current, cross-platform, and honest about what it does not know, so the agents had something solid to stand on.

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
