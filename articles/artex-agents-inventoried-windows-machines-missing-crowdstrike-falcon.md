# ARTEX agents inventoried the Windows machines missing CrowdStrike Falcon. Do you have that list?

*A new FBI alert reports that AI agents built on the open-source ARTEX framework can reach privileged access in less than three hours, and that they searched for Windows machines without an endpoint sensor. Defenders can build that same list first.*

## Key takeaways

- **The response window is now measured in hours.** According to the FBI, some ARTEX intrusions reached privileged access less than three hours after the attackers first selected a target, faster than any weekly scan or monthly review can catch.
- **Attackers look for the machines your sensor missed.** The agents inventoried sensor coverage to find Windows machines without CrowdStrike Falcon, and the FBI's technique table also records that endpoint protection blocked their attempts to move laterally.
- **Fleet complements your EDR rather than replacing it.** Fleet does not block intrusions, but it shows IT and security teams which devices are missing a sensor across macOS, Windows, and Linux.

<a purpose="cta-button" href="https://fleetdm.com/reports">Browse ready-made reports</a>

On November 13, the FBI published a [FLASH alert](https://www.ic3.gov/CSA/2026/261113.pdf) on ARTEX, an open-source penetration testing framework that attackers now use to run intrusions with AI agents. The alert's central finding is speed: some intrusions reached privileged access in less than three hours.

A table of observed techniques later in the alert contains a less prominent detail. Once inside a network, the agents "inventoried endpoint sensor coverage to identify Windows machines without CrowdStrike Falcon." In other words, the attacker's software compiled a list of unprotected machines, and defenders can compile the same list first.

## What the FBI and Amazon reported

ARTEX is a multi-agent platform that turns plain-language goals into commands with little human input, using outside AI models. Its developer pulled the GitHub repository on October 8 after reports of abuse, but forks, mirrors, and a commercial version called ARTEX PRO remain in circulation. The FBI says attackers have used it since at least August 2026 against organizations in ten sectors. One campaign stole personal information belonging to approximately 67,000 people from South Korean financial institutions, and a supply chain compromise that hit ConnectWise Cloud Backup customers led to sign-in attempts against more than 2,250 downstream tenants.

Amazon published [its own findings](https://aws.amazon.com/blogs/security/amazon-threat-intelligence-observes-threat-actors-running-cybercrime-campaigns-with-artex-ai-agents/) the same day. Its decoy sensors captured requests from ARTEX agents that carried the agents' own records of two campaigns, one stealing AI model API keys and another trying to defeat identity checks at cryptocurrency exchanges. Amazon attributes the activity to ARTEX with moderate confidence and cautions that the results the agents claimed could not be independently confirmed.

## The agents looked for unprotected machines

The FBI describes ARTEX traffic as noisy enough that it may trigger alerts, "but attack speed provides limited time for coordinated defender response before objective completion." The weak point is what a team needs to know once an alert fires, such as which devices lack protection, and a weekly scan cannot answer that in time.

The same FBI table lists the agents' next move, attempts to reach domain controllers with Pass-the-Ticket, Pass-the-Hash, Kerberos relay, and WinRM, and notes that those attempts were "blocked by endpoint protection." The FBI does not connect the two entries, but read together they suggest that the sensor worked where it was running and that the agents looked for machines where it was not. An EDR console cannot report on hosts it has never seen, so finding those gaps requires a source that does not depend on the sensor.

## Building the list before an attacker does

Fleet's agent runs on the device and reports independently, so it can confirm whether another tool is present. Fleet's [report library](https://fleetdm.com/reports) includes a ready-made "Get CrowdStrike sensor status" report for macOS, Windows, and Linux, and running it as a [live report](https://fleetdm.com/guides/reports) returns an answer from every online host within seconds. For other EDR products, [Fleet support](https://fleetdm.com/support) can help adapt the check.

To keep that answer current, turn the check into a [policy](https://fleetdm.com/securing/what-are-fleet-policies) that fails on any host without a running sensor. Fleet checks policies every hour by default, and a [policy automation](https://fleetdm.com/guides/automations) can install the missing software or run a script on a failing host. Webhooks and tickets are sent once a day by default, an interval you can shorten. Fleet can only answer for enrolled devices, and offline hosts respond once they reconnect.

The same approach covers several host artifacts the FBI lists, including SSH key additions, new accounts, and remote access or tunneling tools such as vshell and Chisel. Fleet's agent can query tables such as `authorized_keys` on macOS and Linux, along with `users`, `processes`, and `listening_ports` on all three platforms, to show how far the activity has spread. This data provides context rather than detection: a renamed binary will not match a search by name, which leaves that work to your EDR, and the alert's cloud identity activity appears in your identity provider's logs rather than on devices.

## The fundamentals, checked faster

CJ Moses, the CISO of Amazon Integrated Security, wrote that none of the techniques in the campaigns were new and that the agents "succeeded where basic security hygiene was missing." The FBI's recommendations are just as basic, from inventorying internet-facing applications to rotating exposed credentials. ARTEX shortens how long a gap can stay open, so the most effective defense is to find your own gaps first and recheck them on a schedule that fits a three-hour window.

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
<meta name="description" value="The FBI says some ARTEX AI agent intrusions reach privileged access in under three hours and seek hosts without EDR. Find them first.">
