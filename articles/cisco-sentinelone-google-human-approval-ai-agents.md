# Cisco, SentinelOne, and now Google: three vendors have put a human approval step in front of their AI agents

*Google's CodeMender can no longer write files or run commands on a host without sign-off. The pattern is spreading, and it raises the question of who approves the agents running in your own environment.*

## Key takeaways

- **Vendors are adding a human to the loop.** Google now requires approval before CodeMender, its AI vulnerability-fixing agent, can write files or run commands locally. Google's move follows similar ones from Cisco and SentinelOne earlier this year.

- **Sandboxing and approval solve different problems.** A sandbox limits what an agent can reach while it tests a fix. An approval step decides whether the fix is applied at all. Teams need both questions answered.

- **An agent with write access is a change to your endpoints.** An AI agent that edits files or runs commands on a device belongs in the same review process as any other software that changes that device.

- **A policy written up front won't hold on its own.** Approval rules decay when nobody can see what the agents are actually doing. Oversight has to be checkable after the fact.

- **Ground truth should be yours.** A vendor's guardrails describe how the vendor intends its agent to behave. Fleet's agent reports what is installed and running on each device, which lets a team verify that independently.

<a purpose="cta-button" href="https://fleetdm.com/software-management">See what's running on your devices</a>

Google added a sandboxed exploit simulation and a human approval step to CodeMender, its AI agent for finding and fixing vulnerabilities, as reported by Help Net Security. Before the agent can write files or run commands locally, a person has to sign off.

The change is small on its face, but it points at a larger shift: the people building agentic security tools are deciding that unattended write access is too much. If the vendors are adding checks, the teams running their software should know what they are checking.

## Two controls, two questions

A sandbox answers "what can the agent touch while it works?" Approval answers "who decides whether the result goes live?" Google's change adds both to CodeMender. Treat them separately when you evaluate any agent, because a product can have one without the other.

## An agent that writes files is a change to your devices

Once an AI tool can write files or run commands on a host, it behaves like any other software with administrative reach. It needs an owner, a change process, and a way to see where it is installed. Many teams already have that process for scripts and packages. Agents are easy to leave out because they arrive as developer tools or browser extensions rather than as managed software.

## Why a policy up front isn't enough

An approval rule written once, in a document, tells you what was intended. It doesn't tell you whether a particular device has an agent installed, which version it runs, or whether someone turned off the approval prompt. Oversight that holds up is oversight you can check.

## Verify on your own hosts

Fleet's agent reports installed software and running processes for each enrolled host across macOS, Windows, and Linux. That gives a team a vendor-independent way to answer basic questions: which devices have an AI agent, which version, and where it differs from the approved list. Check the [Fleet documentation](https://fleetdm.com/docs) for the tables available on each platform.

Fleet can't tell you how a given vendor's approval flow behaves internally. It can show you that the software is present and what changed on the device.

## The through-line

Three vendors adding approval steps in one year is a signal that unattended agents are a risk worth designing around. The vendor's guardrail is the first layer. Your own inventory is the second.

## See it live

- **Get a demo** to see installed software across your own devices: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Read how Fleet handles software management:** [fleetdm.com/software-management](https://fleetdm.com/software-management)

<meta name="articleTitle" value="Cisco, SentinelOne, and now Google: three vendors have put a human approval step in front of their AI agents">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-11-06">
<meta name="description" value="Google's CodeMender now needs human sign-off before it writes files or runs commands. Who approves the AI agents running on your devices?">
