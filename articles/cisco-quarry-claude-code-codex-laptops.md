# Cisco's Quarry has plugins for Claude Code and Codex. Do you know which laptops run them?

*Cisco's new open-source detector scores a whole AI agent session, not single actions. It can only watch the agents you know are there.*

## Key takeaways

- **Allowed is not the same as safe.** Access controls decide whether an agent may take one action. Cisco's Project Quarry looks at the sequence, such as reading a credentials file and then sending data to an outside server, where each step alone looks fine.
- **Session-level detection needs an agent list first.** Quarry's plugins cover Claude Code and Codex, and a detector can only score sessions on machines where someone installed it. Knowing which devices run which agent tools comes before any rule.
- **Devices hold the ground truth.** Installed software, versions, and config files on the laptop are facts your organization can check itself, independent of what any agent reports about its own behavior.
- **Detection and inventory do different jobs.** Quarry flags suspicious behavior in a session. Fleet reports on what is installed and configured across macOS, Windows, and Linux hosts. Neither replaces the other.
- **Checks belong in version control.** Reports and policies written as YAML in Git get reviewed and deployed like the rest of your configuration, so a new agent tool appearing on a host shows up as a change.

<a purpose="cta-button" href="https://fleetdm.com/software-management">See software inventory in Fleet</a>

On October 28, Cisco open-sourced Project Quarry, a detection language and engine for risky AI agent behavior. Its argument is simple. Individual actions from an agent often look legitimate in isolation, and trouble shows up in how they combine. A rule might flag an agent that reads a credentials file and then sends information to an external server, even if access controls allowed both steps.

That is a useful shift, and it raises a practical question for IT and security teams. Before you can score what an agent did, you need to know where agents run. For most organizations, that answer is scattered across employee laptops.

## What Quarry looks at

Quarry rules can examine model conversations, tool calls, Model Context Protocol (MCP) interactions, and multimodal inputs. Cisco ships plugins for Claude Code and Codex. Cisco also points to a July incident in which an OpenAI agent validated 14 exposed Hugging Face tokens with write access and shared them with an external service. Cisco says Quarry could flag activity resembling that case, but researchers have not tested it against the incident's data, so treat that as a stated aim, not a proven result.

Cisco positions the tool as a complement to access controls. Authorization answers whether an agent may perform an action. Quarry asks whether the pattern across allowed actions looks wrong.

## The part that happens before detection

A session-level detector works where it is deployed. If a developer installs a coding agent on a personal-use laptop and nobody records it, no rule fires because no one is listening. The gap is not in the detection language. It is in the list of places to apply it.

This is where device data helps. Fleet's agent reports what is installed on each host, which versions are present, and whether specific config files or processes exist. You can ask one question across macOS, Windows, and Linux at the same time: which devices have a coding agent installed, and which version? That list tells you where to roll out Quarry's plugins, which machines are still unaccounted for, and which tools nobody approved.

## Why the device view stays useful

An agent's own logs describe what the agent did. The device describes what is there. When you are investigating a session that looks wrong, you want a source that does not depend on the thing under investigation. Installed software, config file locations, and running processes are facts your team can verify independently.

Be clear about the limits. Fleet does not read agent conversations or score behavior, and it does not see agents that only run in a cloud platform. It tells you which devices to look at, and a behavior detector tells you what happened on them.

## Keep the check running

A one-time sweep goes stale. Fleet manages reports and policies as YAML, so the check for agent tooling can live in Git, go through code review, and deploy with the rest of your configuration. When a new agent tool appears on a host, the next report run shows it, with a history of who approved the check and when it changed.

## Where this leaves you

Cisco's release is a good sign that session-level detection is becoming a shared, open practice instead of a private vendor feature. The detector is only as complete as your picture of where agents run. Build that picture from the devices, then point the detection at it.

## See it live

- **Get a demo:** [Talk to the Fleet team](https://fleetdm.com/contact) about building an inventory of agent tooling across your devices.
- **Join a GitOps training session:** [Learn to manage reports and policies as code](https://fleetdm.com/gitops-workshop).

<meta name="articleTitle" value="Cisco's Quarry has plugins for Claude Code and Codex. Do you know which laptops run them?">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="publishedOn" value="2026-10-29">
<meta name="category" value="industry news">
<meta name="description" value="Cisco's Project Quarry scores whole AI agent sessions. It can only watch agents you know about, and your devices hold the list.">
