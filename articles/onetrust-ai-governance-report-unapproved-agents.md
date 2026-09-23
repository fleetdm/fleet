# Nine in 10 companies encourage AI agents. Fewer than half can govern them.

*OneTrust's new AI governance research finds most companies are adopting AI agents faster than they can oversee them. Fleet's inventory is how you find out where those agents landed, instead of waiting for the next incident to tell you.*

## Key takeaways

- **Adoption is already way ahead of oversight.** Almost nine in ten organizations have opened the door to AI agents, and fewer than half say governance covers what those agents do once they're in.
- **This isn't a hypothetical risk.** More than eight in ten organizations already had an AI-related incident this year, and over a quarter dealt with more than one.
- **You don't need a survey to know if you're exposed.** Fleet's agent already inventories the AI apps, browser extensions, IDE extensions, and MCP server configurations installed across your macOS, Windows, and Linux hosts, so you can check your own fleet instead of extrapolating from someone else's.
- **The unapproved agent is usually hiding in a familiar place.** A browser extension with broad permissions, a plugin wired into an IDE, or a desktop app installed without a ticket, all things Fleet's software inventory already tracks.
- **A one-time check doesn't close a governance gap, a saved policy does.** Turning a report into a scheduled policy means the next unsanctioned agent gets caught automatically instead of waiting for the next survey to ask about it.

<a purpose="cta-button" href="https://fleetdm.com/reports">Explore the reports library</a>

OneTrust's new AI-Ready Governance Report, based on a survey of 1,200 senior business decision-makers across eight countries, describes a gap a lot of security teams already feel: 87% of organizations encourage employees to use AI agents, but only 47% say they have clear, defined governance in place, and another 40% describe their controls as still developing. The report also found that 86% of organizations experienced at least one AI-related incident in the past year, and nearly half had an AI system or agent take an unapproved action, with 28% reporting two or more such incidents.

Those numbers describe an industry-wide pattern, but the actual exposure is specific to your own fleet, and it doesn't take another survey to find it.

## What "an unapproved action" looks like on the device

A survey response like "an AI agent did something nobody approved" is an abstraction of something much more concrete: a browser extension that summarizes pages and quietly sends their contents to a third-party API, an MCP server wired up to a coding assistant with more repository access than anyone signed off on, or a desktop AI client installed the same afternoon someone read about it. None of these show up in an identity provider or a SaaS catalog, because none of them were ever sanctioned apps to begin with.

That's exactly why the gap OneTrust measured persists: the tools that create it don't route through the systems most companies use to track software.

## Finding out where unapproved agents landed

Fleet's agent turns every macOS, Windows, and Linux host into something you can query directly, which means you don't have to guess whether an unsanctioned AI agent is running somewhere in your fleet. A browser extension inventory is one of the faster places to look, since AI assistants and "summarize this" tools tend to show up there first:

```sql
-- Chromium-family browsers (Chrome, Edge, Brave, Opera)
SELECT u.username,
       e.name,
       e.identifier,
       e.version,
       e.from_webstore,
       e.permissions
FROM users u
CROSS JOIN chrome_extensions e USING (uid);
```

The `permissions` and `from_webstore` columns are where to focus. A sideloaded extension requesting broad host permissions is the kind of thing a governance policy is supposed to catch before it becomes an incident, not after.

Browser extensions are one entry point among several. Fleet also reads MCP client configuration files, inventories IDE extensions, and tracks installed AI desktop apps across all three major operating systems. We covered the full starter pack of reports in [Shadow AI is already on your fleet](https://fleetdm.com/articles/shadow-ai-is-already-on-your-fleet), which is the fastest way to run the same checks OneTrust's respondents wish they'd had answers to.

## Turning the check into the governance the report says is missing

A single query answers "are we exposed today." The 47% gap OneTrust found is a gap in ongoing oversight, not a one-time audit. Saving these reports as scheduled Fleet policies means a new unsanctioned browser extension or MCP server gets flagged the day it appears, not the next time someone runs a survey. Because Fleet policies live in Git as YAML and deploy through your normal GitOps workflow, updating what counts as "approved" is a reviewable pull request instead of an undocumented exception someone remembers verbally.

## The gap is real, closing it doesn't require a survey

OneTrust's numbers are a reasonable description of the industry, but they're not a diagnosis of your own environment. A company with defined governance and one still building toward it can look identical on paper, unless someone goes and looks at what's installed. Fleet is how you do that look, and how you keep doing it as the next agent, extension, or MCP server shows up.

## See it live

- **Get a demo** to see AI tool visibility across your own fleet: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Read the shadow AI reports** for the full set of queries that surface AI apps, MCP configurations, and browser extensions: [fleetdm.com/articles/shadow-ai-is-already-on-your-fleet](https://fleetdm.com/articles/shadow-ai-is-already-on-your-fleet)

## Sources

- Help Net Security, [Your employees are already using AI tools you never approved](https://www.helpnetsecurity.com/2026/09/15/onetrust-enterprise-ai-governance-trends-report/).
- GlobeNewswire, [OneTrust Research: 86% of Organizations Experienced AI-Related Incidents, Yet Few Slowed Deployment](https://www.globenewswire.com/news-release/2026/09/14/3361166/0/en/onetrust-research-86-of-organizations-experienced-ai-related-incidents-yet-few-slowed-deployment.html).

<meta name="articleTitle" value="Nine in 10 companies encourage AI agents. Fewer than half can govern them.">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-09-15">
<meta name="description" value="OneTrust found nearly half of companies had an unapproved AI agent action this year. Here's how to find where those agents landed.">
