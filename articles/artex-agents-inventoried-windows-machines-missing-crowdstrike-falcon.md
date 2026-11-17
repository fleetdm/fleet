# ARTEX agents inventoried the Windows machines missing CrowdStrike Falcon. Do you have that list?

*An FBI alert says AI agents built on the open-source ARTEX framework can reach privileged access in under three hours. One line in it describes the agents checking which machines had no endpoint sensor. That's a list defenders can build first.*

## Key takeaways

- **The window to respond is now measured in hours.** The FBI says some ARTEX intrusions went from initial targeting to privileged access in under three hours. A weekly scan or a monthly coverage review can't confirm anything inside that window.
- **These attacks are loud, and they still finish.** The FBI describes ARTEX traffic as noisy enough to trigger alerts. The problem is the time left to act on an alert, so answers about device state have to be ready before it fires.
- **Attackers look for the machines your sensor missed.** The alert says agents inventoried sensor coverage to find Windows machines without CrowdStrike Falcon. Where endpoint protection was present, it blocked their lateral movement attempts. Coverage gaps are worth finding before someone else does.
- **The traces the FBI lists are things a device can report.** New SSH keys, new accounts, remote access tools, and tunneling software all show up in device data. You can ask every host about them in one pass.
- **None of the weaknesses are new.** Amazon, which observed two ARTEX campaigns, says the agents succeeded where basic security hygiene was missing. The fix is the same fundamentals, verified faster.
- **Fleet is not your EDR.** Fleet won't block an intrusion. It tells IT and security teams which devices are missing a sensor, across macOS, Windows, and Linux, and lets them ask new questions of every online host in seconds.

<a purpose="cta-button" href="https://fleetdm.com/reports">Browse ready-made reports</a>

On November 13, the FBI published a [FLASH alert](https://www.ic3.gov/CSA/2026/261113.pdf) on ARTEX, an open-source penetration testing framework that attackers now use to run intrusions with AI agents. The headline number is the speed: under three hours from first targeting to privileged access in some cases.

Further down, in a table of observed techniques, there's a quieter detail. Once inside a network, the agents "inventoried endpoint sensor coverage to identify Windows machines without CrowdStrike Falcon." The attacker's software built a list of unprotected machines. This article looks at what the alert says and how to build that list yourself.

## What the FBI and Amazon reported

ARTEX is a multi-agent platform written in Go. An operator describes a goal in plain language, and the agents turn it into commands with little human input. It has no model of its own. It connects to outside models, and the FBI says DeepSeek Flash models were the most common on attacker-controlled installations. The developer, known as "Autumn-27," released ARTEX on GitHub in July 2026 for authorized testing and took the repository down on October 8 after reports of abuse. Forks, mirrors, and a paid version called ARTEX PRO still circulate. As of early November, researchers had counted more than 1,600 ARTEX consoles exposed on the internet.

The FBI says attackers have used ARTEX since at least August 2026 against organizations in ten sectors, including financial services, government, defense, healthcare, and legal. It names two campaigns. One hit South Korean financial institutions and took personal information on about 67,000 people. The other was a supply chain compromise affecting ConnectWise Cloud Backup customers, which led to attempted sign-ins to more than 2,250 downstream tenants.

Amazon published [its own findings](https://aws.amazon.com/blogs/security/amazon-threat-intelligence-observes-threat-actors-running-cybercrime-campaigns-with-artex-ai-agents/) the same day. Between October 3 and November 5, its decoy sensors recorded agents stealing AI model API keys in one campaign and trying to defeat identity checks at two cryptocurrency exchanges in another. Amazon assesses with moderate confidence that ARTEX was the tool, and notes it can't verify the outcomes the agents reported.

## Three hours changes what "regularly" means

The FBI's wording is precise. ARTEX "generates high volumes of forensically noisy traffic that may trigger alerts, but attack speed provides limited time for coordinated defender response before objective completion." The alert calls this a compressed detection window.

Detection is not the weak point here. The alerts may well fire. The weak point is everything a team needs to know after the alert: which devices are affected, which ones lack protection, and what changed on them in the last hour. If those answers come from a scan that runs weekly, or a spreadsheet someone reconciles each quarter, they describe a network from before the intrusion started.

## The agents looked for unprotected machines

The same FBI table that mentions the sensor inventory also lists what the agents tried next: lateral movement to domain controllers using Pass-the-Ticket, WinRM, Kerberos relay, and Pass-the-Hash. The alert notes those attempts were "blocked by endpoint protection."

The FBI doesn't connect the two lines, but the lesson is hard to miss. The sensor did its job where it was running, and the agents went looking for where it wasn't. Most organizations have a few of those machines: a laptop reimaged without the full software set, a server that predates the rollout, a device where the sensor stopped after an update.

Your EDR console can tell you about the hosts it knows. It can't tell you about hosts it has never seen. You need a second source of truth for that, one that doesn't depend on the sensor being installed.

## Building the list before an attacker does

Fleet's agent runs on the device and reports back on its own, so it can say whether another tool is present. Fleet's [report library](https://fleetdm.com/reports) includes a report called "Get CrowdStrike sensor status" for macOS, Windows, and Linux:

```sql
SELECT
  CASE WHEN COUNT(*) > 0 THEN 'yes' ELSE 'no' END AS crowdstrike_running,
  GROUP_CONCAT(DISTINCT name) AS sensor_processes
FROM processes
WHERE name IN ('com.crowdstrike.falcon.Agent', 'falcond', 'falcon-sensor', 'falcon-sensor-bpf', 'CSFalconService.exe', 'CSFalconContainer.exe');
```

Hosts that answer "no" have no sensor running. Run it as a [live report](https://fleetdm.com/guides/reports) and every online host answers within seconds. If you use a different EDR, swap in that product's process names.

A one-time answer goes stale, so turn the check into a [policy](https://fleetdm.com/securing/what-are-fleet-policies). Fleet checks policies every hour by default. When a host fails, a [policy automation](https://fleetdm.com/guides/automations) can install the missing software, run a script, call a webhook, or open a ticket.

Two limits are worth stating plainly. A device has to be enrolled in Fleet to answer, so an unmanaged machine is invisible to this check too. And offline hosts don't respond to a live report until they come back.

## Other traces a device can report

The FBI asks organizations reporting an ARTEX incident to look for specific artifacts. Several live on the host:

- SSH key additions
- Newly created accounts
- Open-source remote access tools: WISP, vshell, and AdaptixC2
- Tunneling tools: Chisel and SoftEther
- Web shells, as `.aspx` or `.jsp` files in unexpected web application paths

Fleet's agent exposes tables that cover most of this ground. `authorized_keys` lists SSH keys on macOS and Linux. `users` lists local accounts. `processes` and `listening_ports` show what's running and what's accepting connections on all three desktop and server platforms. When an alert fires on one machine, running the same question against every host shows whether it's one device or forty.

This is context, not detection. The FBI's technique list includes masquerading, so a renamed binary won't match a search by name. Much of the alert also concerns cloud identity: OAuth application credentials, service principal sign-ins, and Microsoft Entra audit records. Fleet doesn't see those. Your identity provider's logs and your EDR own that work.

## The fundamentals, checked faster

Amazon's CISO, CJ Moses, wrote that none of the techniques in the campaigns were new and that the agents "succeeded where basic security hygiene was missing." The FBI's recommendations read the same way: inventory every internet-facing application, keep management interfaces off the open internet, apply least privilege, and rotate exposed credentials.

What ARTEX changes is how long a gap can sit open before someone finds it. An agent that runs all night will check every machine for a missing sensor. The defense is to know your own gaps first, and to check them on a schedule that fits a three-hour window.

## See it live

- **Get a demo:** [Talk to the Fleet team](https://fleetdm.com/contact) about finding coverage gaps across your macOS, Windows, and Linux devices.
- **Rather try it yourself?** Send your IT team to a [free, hands-on Fleet workshop](https://fleetdm.com/workshops) to learn how to write policies like the sensor check and keep them in Git.

## Sources

- FBI, [Threat Overview of ARTEX Autonomous Penetration Testing Framework](https://www.ic3.gov/CSA/2026/261113.pdf), FLASH-20261113-001, November 13, 2026.
- Amazon Threat Intelligence, [Amazon Threat Intelligence observes threat actors running cybercrime campaigns with ARTEX AI agents](https://aws.amazon.com/blogs/security/amazon-threat-intelligence-observes-threat-actors-running-cybercrime-campaigns-with-artex-ai-agents/), November 13, 2026.

<meta name="articleTitle" value="ARTEX agents inventoried the Windows machines missing CrowdStrike Falcon. Do you have that list?">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="publishedOn" value="2026-11-17">
<meta name="category" value="industry news">
<meta name="description" value="The FBI says ARTEX AI agents reach privileged access in under three hours and look for hosts without EDR. Find those hosts first.">
