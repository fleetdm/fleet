# SailPoint's endpoint and browser sensors put MCP servers and agent credentials on the audit trail

*SailPoint now ships sensors that look for AI agents on the endpoint itself. The vendor move says something about what auditors will ask for next.*

## Key takeaways

- **Agent discovery is moving to the endpoint.** SailPoint's Agentic Fabric adds endpoint and browser sensors that surface AI agents, MCP servers, and the credentials they hold, because policy documents can't show what's actually installed.
- **The gap is large.** SailPoint's Horizons report says 79% of enterprises run AI agents in production while only 2% have purpose-built identity tools to govern them.
- **Inventory becomes evidence.** SailPoint's Agent Audit turns agent inventory and governance activity into exportable, framework-aligned reports, which means auditors can ask for proof directly.
- **Ground truth beats self-reporting.** The useful question is what an agent can reach, not what a policy says it shouldn't, and that answer comes from the device.
- **Your own device data can answer part of it.** Reports run on managed hosts can show installed tools, running processes, and extensions without waiting for a platform purchase.

<a purpose="cta-button" href="https://fleetdm.com/contact">Get a demo</a>

SailPoint announced general availability of Agentic Fabric on October 6. It covers discovery and governance for AI agents and other non-human identities, and it connects them to human owners and classifies their usage against policy.

The most telling piece is the sensors. Identity platforms have historically learned about access from connectors and directories. Here, SailPoint added SailPoint Endpoint Agent Security (SEAS) and SailPoint Browser Agent Security (SBAS) to find agents nobody approved.

## What the sensors look for

According to SailPoint's announcement, the sensors expose AI agents, MCP servers, tools, credentials, and data stores that traditional tools miss. SIEM and XDR telemetry and vault and pipeline discovery feed the same registry of non-human identities.

An MCP client matters here because it can extend an agent's reach to other tools, systems, data sources, and credentials. An unapproved one is an access path nobody reviewed.

## Why the audit feature is the real signal

Agent Audit exports inventory and governance activity as framework-aligned evidence reports, which can be customized or scheduled. Once a vendor packages agent inventory as audit evidence, the expectation that you can produce it follows.

SailPoint also added just-in-time provisioning, where access is granted only when needed, with conditions like a business justification or reauthentication.

## What an endpoint can tell you

Fleet has argued that agentic security needs ground truth rather than trust in a system's own account of itself. A managed device is where that truth lives: what's installed, what's running, and what's listening.

Fleet's agent runs reports on managed macOS, Windows, and Linux hosts, so a team can ask which machines have a given agent tool installed or running, and how that changed since last week. Fleet doesn't govern identities or issue access, and it doesn't replace an identity platform. It supplies the per-host facts such a platform needs to be right. Which agent-related tables fit your environment depends on the tools you care about, so build and test those reports against your own fleet.

## Evidence over assurances

Whichever platform you choose, the pattern is the same: discover first, attribute to an owner, then prove it on demand. Teams that can already show per-host facts will find the audit request easier to answer.

## See it live

- **Get a demo** to see device reports across your own hosts: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Explore the software catalog** Fleet builds from every host: [fleetdm.com/software-catalog](https://fleetdm.com/software-catalog)

<meta name="articleTitle" value="SailPoint's endpoint and browser sensors put MCP servers and agent credentials on the audit trail">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-10-06">
<meta name="description" value="SailPoint's new sensors find AI agents, MCP servers, and credentials on endpoints. See what that means for proving what an agent can reach.">
