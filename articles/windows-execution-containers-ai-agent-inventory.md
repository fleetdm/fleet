# Microsoft Execution Containers are generally available on Windows 11: find your AI agents before you contain them


*Microsoft's new containment layer lets policy limit what an AI agent can touch on a Windows PC. Policy only helps on devices where you know an agent is running.*

## Key takeaways

- **Microsoft now has a containment layer for agents.** Microsoft Execution Containers (MXC) lets a developer or administrator declare what an agent can reach, and Windows enforces those limits at runtime. It became generally available on Windows 11 on October 7, 2026.

- **The isolation options are different bets.** A process container restricts an agent inside the user's environment, and a session container moves it into a separate Windows account and session. Which one applies decides how much a compromised agent can do.

- **Policy is only as good as your list of agents.** Containment rules apply to the agents someone has configured. An agent installed outside that path never meets the rule.
- **Intune management isn't here yet.** Microsoft says Intune policy for MXC process containers on Windows 11 is coming soon, without a date. Plan on verifying before you depend on it.

- **Fleet's agent can report what's installed on each Windows device.** That gives you a per-host view of the software that could host an agent, whatever tool ends up enforcing the policy.
- **A report or policy turns that view into a standing check.** Define what an approved agent looks like once, and flag every PC that doesn't match.

<a purpose="cta-button" href="https://fleetdm.com/guides/windows-mdm-setup">See Windows device management in Fleet</a>

Microsoft announced Execution Containers at Build 2026 as a policy-driven execution layer for AI agents on Windows and the Windows Subsystem for Linux. It's in preview, so developers can build and test containment policies now.

The announcement answers how to limit an agent. IT also has to answer a prior question: on which of our devices is there an agent to limit?

## What Execution Containers do

According to [Microsoft's Windows Developer Blog](https://blogs.windows.com/windowsdeveloper/2026/10/07/microsoft-execution-containers-policy-driven-containment-for-ai-agents/), a developer or administrator declares what an agent can access, such as files and network, and Windows enforces those rules at runtime. Microsoft also ties agent activity to an identity, so IT can tell what an agent did from what the person using the device did. In his [Build 2026 post](https://blogs.windows.com/windowsdeveloper/2026/06/02/build-2026-furthering-windows-as-the-trusted-platform-for-development/), Pavan Davuluri, Microsoft's executive vice president for Windows and Devices, wrote that "every agent activity must be attributable and governed." Microsoft says Entra support for separating agent activity from user activity is coming soon.


On Windows 11, the two main isolation options sit at different points on the risk scale:

- **Process isolation** runs the agent in the same user session with its access restricted by policy, and suits low-risk work such as simple tool execution.
- **Session isolation** separates the agent from the user's desktop, UI, clipboard, and input devices, which is meant to limit UI spoofing, input injection, and cross-session data leakage.
Microsoft also lists a WSL container for Linux toolchains and an experimental MicroVM backend for higher-risk workloads.

## Where Intune fits

Coverage says IT can manage agent permissions centrally, including things like read-only file marking and access to the browser, screen capture, and location, with Intune enforcing device-level policy. The reports don't line up on what is available today versus coming later, and none of the sources I reviewed give Intune policy names or setup steps. Check Microsoft's documentation for the current state before you plan a rollout around it.

Whatever the timing, the pattern is familiar. A management tool enforces rules on the devices it knows about, in the shape you've configured.

## The gap before the policy

Agents arrive through developer tools, desktop apps, and command-line installs, and some of them are put there by individual employees. A containment policy for a sanctioned agent does nothing about an unsanctioned one running outside it.

The first useful question is an inventory question: which Windows devices have software that can run an agent, which versions, and which of them are on the list you've approved.

## Seeing agent software across Windows hosts

Fleet's agent reports installed software and versions for each enrolled Windows device, so you can search for the tools you care about by name and see which hosts have them. Fleet also manages Windows endpoints, so the same data sits next to the rest of the device record.
Inventory has blind spots, and agent tooling tends to land in them. On Windows, Fleet doesn't collect global npm packages, Scoop packages, or portable executables yet, so a command-line agent installed that way won't appear in software inventory. The [software inventory reference](https://fleetdm.com/guides/software-inventory-reference#limitations) lists what is and isn't collected.

This is inventory, not enforcement. Fleet's agent doesn't read what an Execution Container allows, and I'd treat anything that claims it does with caution. What it can do is tell you where agent software lives, so Intune or any other control is applied to a known list instead of a guess.

From there, a report can list hosts running unapproved agent software, and a policy can flag any PC that doesn't match your baseline. Keep either in Git as YAML and changes to the list go through review.

## The through-line

Containment is a strong idea, and Microsoft is putting it where the agents run. It works on the agents you've found. Start with the inventory, then let policy do its job.

## See it live

- **Get a demo** to see installed software reported across your own Windows devices: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Read how Fleet handles Windows device management:** [fleetdm.com/guides/windows-mdm-setup](https://fleetdm.com/guides/windows-mdm-setup)

<meta name="articleTitle" value="Microsoft Execution Containers are generally available on Windows 11: find your AI agents before you contain them">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-10-08">
<meta name="description" value="Microsoft's Execution Containers limit what AI agents can do on Windows. Here's how to see which of your PCs run agents before policy applies.">
