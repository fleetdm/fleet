# CrowdStrike is investigating the Warp breach, but your list of laptops running Warp shouldn't wait for the report

*Warp says a compromised vendor exposed AI agent decision logs and telemetry. Until the investigation finishes, the part you can answer yourself is where Warp runs in your organization.*

## Key takeaways

- **The vendor's account is still incomplete.** Warp's CEO estimates roughly 1% of users were affected, and CrowdStrike is investigating. Warp hasn't said whether credentials or source code were included, so a team can't yet rule either out.
- **Agent logs can hold more than commands.** A coding agent's decision log records what a developer asked for and what the agent did about it, which can include file paths, project names, and pasted secrets. Treat exposed logs as sensitive until you know otherwise.
- **You can't scope your exposure without an inventory.** The first question after any vendor breach is which of your people use the product. A current list of the devices running Warp turns that into a lookup instead of a survey.
- **Fleet reports what is installed, across operating systems.** Fleet's agent inventories software on macOS, Windows, and Linux, so you can find Warp installs on the devices you manage and see which versions they run.
- **A report keeps the answer current.** Saved as a report, the same check tells you when a new install shows up after the breach, not just which ones existed on the day you looked.
- **Credential rotation starts from that list.** Once you know which developers ran Warp, you know whose tokens and API keys to review first, and you can decide whether to rotate before the vendor finishes its findings.

<a purpose="cta-button" href="https://fleetdm.com/software-management">See software inventory in Fleet</a>

Warp disclosed that a compromised third-party vendor exposed AI agent decision logs and user telemetry, according to BleepingComputer. The CEO's estimate is that roughly 1% of users were affected, and the company brought in CrowdStrike to investigate.

The open questions matter more than the disclosed facts. The company hasn't said whether credentials or source code were part of what was exposed. Security teams will be asked about their exposure before those answers arrive, and the first step doesn't depend on them.

## What a coding agent's logs can contain

A terminal built around an AI agent sits close to the work. Its decision logs describe what a developer asked the agent to do and how the agent responded. Telemetry adds how and when the tool is used.

Depending on how people work, that can include repository and directory names, error output, and anything pasted into a prompt. Some of that is routine. Some of it is a token someone pasted while debugging. Warp hasn't published what these logs held, so the cautious reading is that sensitive material may be present.

## The question you can answer today

Warp's investigation will take time, and your review shouldn't wait for it. What you control is the list of devices where Warp is installed, and who uses them.

Without an inventory, this is a message to the whole engineering team asking who uses Warp, followed by days of replies and people who forget. With one, it's a query. That difference is the gap between starting work this morning and starting it next week.

This is the same argument Fleet made in [agentic security is only as good as the device data underneath it](https://fleetdm.com/articles/agentic-security-needs-ground-truth): decisions about agents are only as sound as the ground truth you can verify yourself. A vendor that can't yet say what was exposed in its own breach is a fresh example.

## Finding Warp across your devices

Fleet's agent reports installed software for every enrolled device, including the application name and version. Search the software inventory for Warp, and you see which hosts have it, on which operating system, and at which version.

Fleet manages macOS, Windows, and Linux hosts, so the answer covers the whole fleet rather than one platform. Only devices enrolled in Fleet appear in the results, so laptops that aren't enrolled are a gap to close separately. Installs that aren't picked up as standard applications, such as a binary a developer placed in their home directory, may not show up in software inventory. If that's a concern, a report that looks for the binary on disk covers those cases.

## Keeping the answer current

A one-time list goes stale. Developers install tools on their own schedule, including after a breach announcement, when some will try a competitor and others will reinstall.

Save the check as a report and Fleet runs it on a schedule. Because reports and policies can live in Git as YAML, the definition is reviewed like any other change, and the history shows when the check began. If your organization decides that Warp isn't approved, a policy can flag every device where it's installed.

## From inventory to action

Once you have the list, the work is human. Ask those developers what they used Warp for. Review the tokens, API keys, and repositories they touched during the period, and decide which to rotate. If Warp later confirms that credentials weren't included, you will have lost a few hours. If it confirms they were, you will already be ahead.

## The takeaway for the next vendor breach

Vendors disclose in pieces, and the pieces rarely arrive in time. Teams that can name their affected devices in minutes get to choose their response, while the rest wait for the report.

## See it live

- **Get a demo** to see installed software across your own devices: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Read how Fleet handles vulnerability and software management:** [fleetdm.com/software-management](https://fleetdm.com/software-management)

<meta name="articleTitle" value="CrowdStrike is investigating the Warp breach, but your list of laptops running Warp shouldn't wait for the report">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-10-28">
<meta name="description" value="Warp says a vendor breach exposed AI agent logs. Find which of your laptops run Warp now, without waiting for the investigation to finish.">
