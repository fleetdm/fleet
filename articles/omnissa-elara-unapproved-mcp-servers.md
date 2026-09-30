# If Omnissa's Elara finds an unapproved MCP server on a laptop, what can it actually reach?

*Omnissa launched Elara on September 29 to discover the AI agents nobody approved and hold them to one policy. Discovery is the first half of governance; the second half is knowing what each agent can touch.*

## Key takeaways

- **A direct MDM competitor just made AI agent discovery a headline feature.** Omnissa's Elara looks for AI apps, agents, and MCP servers across managed devices, including ones outside the approved app inventory. Governing agents is now a mainstream device management requirement, not a niche worry.
- **Finding an agent is not the same as understanding it.** A list of names tells you something is installed. It doesn't tell you which files, credentials, or network locations that agent can reach.
- **A policy on paper is a claim, not a measurement.** Sanctioning or restricting an agent records a decision. Whether the device actually reflects that decision is a separate question that needs its own evidence.
- **Fleet's agent reports what is on the device itself.** The `ai_tools` table returns one row per AI tool, covering MCP servers, agent CLIs, desktop apps, and IDE plugins, with risk flags that describe what each one has been permitted to do.
- **Two independent views beat one.** Running an authority layer alongside device-level data gives you a way to notice when the two disagree, which is where surprises tend to hide.
- **Treat the beta as a signal, not a finish line.** Elara is in beta, so most teams can't rely on it yet. The device-level questions are worth answering now, whichever governance tool you eventually choose.

<a purpose="cta-button" href="https://fleetdm.com/visibility-and-reporting">See what Fleet can tell you</a>

On September 29, Omnissa introduced Elara at its Omnissa ONE conference and described it as an authority layer for AI. Omnissa says Elara discovers AI apps, models, agents, MCP servers, and tools running across managed devices, including tools installed outside standard app inventory, and lets IT sanction, restrict, or monitor what it finds.

Fleet has argued that [agentic security needs ground truth](https://fleetdm.com/articles/agentic-security-needs-ground-truth) about device state, not trust in a system's own account of itself. Elara's arrival suggests the wider market is heading toward the same standard. The interesting part is what "ground truth" has to include once an agent is on the device.

## What Omnissa announced

Omnissa positions Elara as a layer that sits above the tools a customer already runs. It brings together signals about people, identities, devices, apps, and AI, and correlates them with an organization's policy and permission structure. The stated aim, in the words of Omnissa's Product CTO Brian Link, is "control before consequence."

The first focus is AI governance: discovering AI applications and agents that are already operating, including unapproved ones, and managing the approved ones. Elara is available in beta, with a demo tour and waitlist on Omnissa's site. Omnissa's announcement doesn't publish detailed technical specifications, so how discovery works on each platform isn't something to assume yet.

## Discovery answers what, not how far

Suppose discovery turns up an MCP server on an engineer's laptop that nobody approved. You now know it exists. The questions that decide whether it matters come next.

Which tools does it expose? Which directories can it read? Does it hold credentials, and is it listening on a local socket that other processes can talk to? An inventory row with a name and a version can't answer those, and a sanction or restrict decision made on a name alone is a decision made with half the picture.

## A policy is not evidence

Approving, restricting, or monitoring an agent creates a record of intent. What the device is doing is a separate fact, and the gap between the two is where governance programs usually break. An agent gets restricted, and a user reinstalls it under another name. A tool gets approved at version one, and a later update widens what it can reach.

This is true of every device management policy, not only AI ones. The useful habit is to treat the policy as a claim and check it against what the device reports.

## What the device can tell you directly

Fleet's agent reads state from each device, so the answer comes from the machine rather than from a governance layer's account of it. The `ai_tools` table in fleetd returns one row per AI tool across macOS, Windows, and Linux. It covers MCP servers, agent CLIs, desktop apps, IDE plugins, sockets, and instruction files, and each row carries risk flags that describe what that tool has been permitted to do.

Because that data is queryable, you can ask the follow-up questions on demand: which hosts run an MCP server that isn't on the approved list, and which of those expose a local socket. A report can run on a schedule, and a policy can flag any host that drifts from the decision you recorded. Fleet doesn't replace an authority layer. It supplies an independent reading of the same devices, so you can tell when the two disagree.

## Where this leaves you

Omnissa's launch is a useful marker: unapproved agents on managed devices are now a problem worth building a product around. Elara is still in beta, and any authority layer will only be as good as the signals feeding it.

Whichever tool sets your policy, keep a way to check the result on the device itself. Knowing that a policy exists is a smaller thing than knowing which machines follow it.

## See it live

- **Get a demo** to see AI tools reported across your own devices: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Read the earlier piece on device data and agents:** [Agentic security is only as good as the device data underneath it](https://fleetdm.com/articles/agentic-security-needs-ground-truth)

<meta name="articleTitle" value="If Omnissa's Elara finds an unapproved MCP server on a laptop, what can it actually reach?">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-09-30">
<meta name="description" value="Omnissa's Elara discovers unapproved AI agents and MCP servers. Here's what to verify on the device itself once you know they exist.">
