# CrowdStrike is investigating the Warp breach, but your list of laptops running Warp shouldn't wait for the report

*Warp says a compromised third-party vendor exposed AI agent decision logs and telemetry. Until the investigation finishes, the part you can answer yourself is where Warp runs in your organization.*

## Key takeaways

- **The vendor's account is still incomplete.** Warp's CEO estimates roughly 1% of users may have been affected, and CrowdStrike is investigating. Warp hasn't said whether the data held source code, prompts, or credentials, so a team can't yet rule any of them out.
- **Agent logs can hold more than commands.** A coding agent's decision log records what a developer asked for and what the agent did about it, which can include file paths, project names, and pasted secrets. Treat exposed logs as sensitive until you know otherwise.
- **You can't scope your exposure without an inventory.** The first question after any third-party vendor breach is which of your people use the product. A current list of the devices running Warp turns that into a lookup instead of a survey.
- **Fleet reports what is installed, across operating systems.** Fleet's agent inventories software on macOS, Windows, and Linux. On macOS and Linux, that means you can find Warp installs on the devices you manage and see which versions they run.
- **A policy flags new installs.** Software inventory refreshes on its own, and a policy flags a new Warp install instead of leaving it for your next search.

<a purpose="cta-button" href="https://fleetdm.com/software-management">See software inventory in Fleet</a>

Warp, the AI-powered terminal that developers use to direct coding agents, disclosed that attackers who compromised a third-party vendor exposed AI agent decision logs and user telemetry, [according to BleepingComputer](https://www.bleepingcomputer.com/news/security/warp-discloses-breach-exposing-ai-agent-logs-user-telemetry-data/). CEO Zach Lloyd put the share of possibly affected users at roughly 1%, adding that his team caught the activity within hours and mitigated it. Warp has notified the users it identified, informed law enforcement, and brought in CrowdStrike to investigate.

The open questions matter more than the disclosed facts. The company hasn't said whether the exposed data held source code, prompts, or credentials, and it hasn't named the vendor. Security teams will be asked about their exposure before those answers arrive, and the first step doesn't depend on them.

## What Warp's AI coding agent logs can contain

A terminal built around an AI agent sits close to the work. Its decision logs describe what a developer asked the agent to do and how the agent responded. Telemetry adds how and when the tool is used.

Depending on how people work, that can include repository and directory names, error output, and anything pasted into a prompt. Some of that is routine. Some of it is a token someone pasted while debugging. Warp's [privacy notice](https://www.warp.dev/privacy) says AI features like Agent Mode can send what a user submits, and what comes back, to Warp, and that users control this in settings. Warp hasn't published what the exposed logs held, so the cautious reading is that sensitive material may be present. Ask your Warp users how their AI data settings are configured.

## Scoping your Warp breach exposure today

Warp's investigation will take time, and your review shouldn't wait for it. What you control is the list of devices where Warp is installed, and who uses them.

Without an inventory, this is a message to the whole engineering team asking who uses Warp, followed by days of replies and people who forget. With one, it's a query. That difference is the gap between starting work this morning and starting it next week. Inventory covers the devices you manage, so personal laptops still depend on asking.

This is the same argument Fleet made in [agentic security is only as good as the device data underneath it](https://fleetdm.com/articles/agentic-security-needs-ground-truth): decisions about agents are only as sound as the ground truth you can verify yourself.

## How to find every Warp install with software inventory

Fleet's agent reports installed software for every enrolled device, including the application name and version. Search the software inventory for Warp, and you see which hosts have it, on which operating system, and at which version.

On macOS, Warp appears as an application. On Linux, it appears through the .deb and .rpm packages Warp ships. If your team keeps Warp, it's also a [Fleet-maintained app](https://fleetdm.com/guides/fleet-maintained-apps) on macOS, so Fleet can install it and keep it on the latest version. The [software inventory reference](https://fleetdm.com/guides/software-inventory-reference) lists which install types Fleet collects on each platform, including formats like AppImage that sit outside package managers.

## Keeping your Warp inventory current with policies

Developers install tools on their own schedule, including after a breach announcement, when some will try a competitor and others will reinstall. Unapproved AI coding tools are the newest form of shadow IT. Fleet's agent refreshes software inventory on its own, so the search stays accurate without a rerun.

If your organization decides that Warp isn't approved, or pauses it while the investigation runs, a policy can flag every device where it's installed. Write it so a device passes when Warp is absent and fails when it's present, and Fleet flags new installs within the hour on online devices. Because policies can live in Git as YAML, the definition is reviewed like any other change, and the history shows when the check began.

## Credential rotation after the Warp breach

Once you have the list, the work is human. Ask those developers whether they received a breach notice from Warp and what they used it for. Review the tokens, API keys, and repositories they used with Warp, and decide which to rotate. Warn them that breach news invites phishing, and that a message claiming to come from Warp deserves a second look. If Warp later confirms that credentials weren't included, you will have lost a few hours. If it confirms they were, you will already be ahead.

## The takeaway for the next AI developer tool breach

Warp's disclosure follows a breach at OpenCode, another AI coding platform, that exposed stored API keys. AI developer tools hold the access attackers want. Their vendors disclose in pieces, and the pieces rarely arrive in time. Teams that can list every device running Warp in minutes get to choose their response, while the rest wait for the report.

## See it live

- **Get a demo** to see installed software across your own devices: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Read how Fleet handles vulnerability and software management:** [fleetdm.com/software-management](https://fleetdm.com/software-management)
- **Join a GitOps workshop** to learn how to write policies like the Warp check, keep them in Git, and review every change before it ships: [fleetdm.com/workshops](https://fleetdm.com/workshops)

## Sources

- BleepingComputer: [Warp discloses breach exposing AI agent logs, user telemetry data](https://www.bleepingcomputer.com/news/security/warp-discloses-breach-exposing-ai-agent-logs-user-telemetry-data/)
- Warp: [Privacy notice](https://www.warp.dev/privacy)
- BleepingComputer: [OpenCode hack exposes API keys amid AI distillation attacks](https://www.bleepingcomputer.com/news/security/opencode-hack-exposes-api-keys-amid-ai-distillation-attacks/)
- Fleet: [Software inventory reference](https://fleetdm.com/guides/software-inventory-reference)

<meta name="articleTitle" value="CrowdStrike is investigating the Warp breach, but your list of laptops running Warp shouldn't wait for the report">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-10-28">
<meta name="description" value="A vendor breach at Warp exposed AI agent logs. Use software inventory to find every device running Warp and decide which credentials to rotate.">
