# 76% of AI agents haven't run in 60 days, but their credentials still work. Start the cleanup on your laptops

*A new report says most dormant AI agents never lose their access. Here is how to find the ones your team left behind, starting with the devices where they were built and run.*

## Key takeaways

- **Quiet does not mean safe.** Arrakis Security's 2026 Autonomous Agents Threat Report found that 76% of active agents had not run in 60 days, yet their credentials and triggers still worked. An agent nobody watches is an identity nobody reviews.
- **Part of the trail starts on a laptop.** Coding agents, local MCP server configs, and API keys in dotfiles are created on employee devices, so a device inventory is the fastest way to see which agents exist at all.
- **You need a list before you can revoke anything.** Fleet's software inventory and reports give IT and security one view of the agent tooling installed across macOS, Windows, and Linux hosts.
- **Credentials live outside the device too.** Revocation happens in your identity provider, SaaS admin consoles, and secrets manager. The device view tells you where to look, and those systems do the actual revoking.
- **A recurring check beats a one-time sweep.** Run the same reports on a schedule, keep them in Git, and a new idle agent shows up as a diff instead of a surprise.

<a purpose="cta-button" href="https://fleetdm.com/contact">Talk to us about agent visibility</a>

Most security teams can name the AI agents they approved. Far fewer can name the ones they forgot. Someone wires up a proof of concept, connects it to a ticketing system or a code repository, and moves on to the next project. The agent stops running. The access it was granted does not stop existing.

Arrakis Security's report puts numbers on that gap. The question for IT is where to start looking.

## What the report found

The [Arrakis report](https://www.globenewswire.com/news-release/2026/10/22/3386053/0/en/arrakis-security-report-finds-76-of-ai-agents-are-dormant-but-keep-their-credentials-and-connections.html), published October 22, 2026, draws on 12 weeks of anonymized telemetry from customer environments. Its headline: 76% of active agents had not run in 60 days, yet their credentials and triggers still worked. According to [coverage of the report](https://www.helpnetsecurity.com/2026/10/26/arrakis-dormant-ai-agent-risks-report/), about 24% of those dormant agents could be started by an unsigned webhook from outside the organization, and most agents tied to sensitive integrations had no verified data loss prevention.

"When an agent stops being used, nobody removes anything, because in most companies nobody owns it," said Tal Baron, CEO of Arrakis Security.

Two caveats are worth stating plainly. The data comes from a single vendor's customers, and "exposure" is that vendor's own classification, not a count of breaches. Treat the exact percentages as directional. The direction is hard to argue with.

## Why dormant agents are a credential problem

A dormant agent is a service account with a personality. It holds a token, it can reach data, and nobody has a reason to look at it. The usual controls assume a person will notice when something is off, and a quiet agent gives them nothing to notice.

That makes the cleanup an inventory problem first. You cannot rotate a key you do not know exists, and you cannot decide an agent is unused until you know it was ever in use.

## Where the trail starts on a device

Plenty of agents are built and tested on employee machines before they move anywhere else. The evidence stays behind: a coding agent installed in a developer's toolchain, an MCP server entry in a local config file, an API key in a shell profile or `.env` file, a scheduled job that kicked off a script months ago.

Fleet's agent, built on osquery, reports on hosts across macOS, Windows, and Linux, so you can ask the same question of every device at once. [Software inventory](https://fleetdm.com/guides/software-inventory-reference) shows which AI tools and runtimes are installed and which versions, including the IDE extensions and npm and Python packages where many coding agents live. For installed applications, it also shows when each one was last opened, the same idle-time test the Arrakis report applied to agents. Reports can check for the specific config paths and processes your team cares about. Which paths matter depends on the tools in your environment, so build the list from what you already approved, then widen it.

This does not find agents that only ever ran in a cloud platform, and it does not read the contents of secrets. It narrows the search and gives each lead an owner.

## From list to revocation

Once you have a list, work it in order:

1. Match each agent to a person or team. If nobody claims it, that is a finding.
2. Find the credentials it holds: identity provider app registrations, SaaS API tokens, repository deploy keys, and entries in your secrets manager.
3. Revoke what is not needed, and rotate what is.
4. Remove the software or config from the device, so the agent cannot be quietly revived.

Steps 2 and 3 happen in those other systems. Fleet's part is steps 1 and 4: finding the agent on the device and confirming it is gone after cleanup. For step 4, [Fleet scripts](https://fleetdm.com/guides/scripts) can remove a tool or config file from every affected host at once.

## Make it repeatable

Fleet manages reports and policies as YAML, so the checks for agent tooling can live in Git, go through code review, and deploy the same way as the rest of your configuration. When a new tool shows up on a host, it appears in the next run, and a policy [automation](https://fleetdm.com/guides/automations) can open a ticket or send a webhook so the finding reaches an owner. That turns a one-off audit into a standing control, with a history of what changed and who approved it.

## The cost of leaving it

The risk is not exotic. A forgotten agent with a live token and a reachable trigger is an open door that no one is assigned to watch. Finding it takes an inventory, and the inventory is cheaper before an incident than after.

## See it live

- **Get a demo:** [Talk to the Fleet team](https://fleetdm.com/contact) about building an inventory of agent tooling across your devices.
- **Rather try it yourself?** Skip the demo and send your IT team to a [free, hands-on Fleet workshop](https://fleetdm.com/workshops). In four hours, they'll work in Fleet alongside an instructor and earn a Fleet Level 1 certificate.

<meta name="articleTitle" value="76% of AI agents haven't run in 60 days, but their credentials still work. Start the cleanup on your laptops">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="publishedOn" value="2026-10-27">
<meta name="category" value="industry news">
<meta name="description" value="Arrakis found 76% of AI agents haven't run in 60 days, but their credentials still work. Start the audit on the devices where agents were built.">
