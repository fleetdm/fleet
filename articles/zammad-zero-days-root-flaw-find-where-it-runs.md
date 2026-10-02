# Zammad zero-days behind the DIVD breach, and how to find every install

*An AI agent chained two Zammad zero-days to breach a vulnerability disclosure nonprofit. The lesson for IT teams starts with an inventory question.*

## Key takeaways

- **An autonomous agent went from a regular user to root.** The Dutch Institute for Vulnerability Disclosure (DIVD) says an AI-driven attack chained two Zammad zero-days against its helpdesk, moving from remote code execution to root in seconds.
- **One of the two flaws affects every Zammad version.** CVE-2026-102490 is a local privilege escalation that, per DIVD, reaches all versions, including the latest alpha. As of October 2, Zammad hasn't published a fix, so upgrading doesn't close it.

- **The other flaw depends on your version.** CVE-2026-102489 is an unauthenticated remote code execution bug in Zammad 6.3.0 through 6.5.4, and DIVD says it isn't exploitable in 7.0.0 through 7.1.3.
- **You can't mitigate a server you don't know you run.** Helpdesks get stood up by one team and forgotten by the next. The first step is a list of where Zammad lives and which version each copy runs.
- **A vendor's word that a hole is closed isn't evidence.** Fleet has argued that agentic security needs ground truth from the device, and this incident shows why: an automated attacker moves faster than an advisory travels.
- **Containment limited the damage.** DIVD says it assumed breach and that network segmentation kept the impact smaller. Segmentation only works for hosts you've mapped.

+<a purpose="cta-button" href="https://fleetdm.com/software-catalog">See software inventory in Fleet</a>

DIVD discloses vulnerabilities for a living, and on September 21 its own Zammad helpdesk was breached. On October 1, DIVD [confirmed](https://www.helpnetsecurity.com/2026/10/01/divd-agentic-ai-attack-breach/) that the attacker used two Zammad zero-days, and that the behavior pointed to an autonomous AI agent choosing each next step on its own.


Most coverage focuses on the novelty of an AI attacker. The more useful angle for an IT team is plainer. Two flaws are known, one has no fix, and the first question is whether you run the software at all.

## What was exploited

[Help Net Security](https://www.helpnetsecurity.com/2026/10/01/divd-agentic-ai-attack-breach/) and [SecurityWeek](https://www.securityweek.com/zammad-zero-days-exploited-in-ai-powered-divd-hack/) describe two vulnerabilities:

- **CVE-2026-102489** is unauthenticated remote code execution that can also leak user sessions. DIVD lists Zammad 6.3.0 through 6.5.4 as affected. The bug is also present in 7.0.0 through 7.1.3, but DIVD says environment conditions keep it from being exploited there.

- **CVE-2026-102490** is a privilege escalation from the Zammad user to root. DIVD says all versions are affected, including the latest alpha.

DIVD's advice is to upgrade to Zammad 7 or take vulnerable installations offline. Upgrading addresses the first flaw. As of October 2, nothing fixes the second, and Zammad hasn't published an advisory for either CVE. Zammad [released 7.2.0](https://github.com/zammad/zammad/releases) on September 23, and the reporting on DIVD's advisory doesn't say how that release is affected.


[The Register](https://www.theregister.com/security/2026/10/01/ai-agents-hacked-the-hackers-stealing-email-addresses-from-security-research-org/5300652) reports that the attacker took email addresses. DIVD hasn't published a full account of everything accessed, so treat the scope as still open.

## Why the inventory comes first

Nobody can apply that advice to a server they've forgotten. Zammad is a helpdesk, which means it often sits with a support team, installed once on a Linux server or in a container, and rarely revisited. It doesn't show up in the software lists IT reviews for laptops.


The question to answer is concrete. Which hosts run Zammad, which version is each on, and is any of them reachable from somewhere it shouldn't be?

## Finding and containing Zammad with Fleet

**Find it with software inventory.** Fleet 's agent collects deb and rpm packages on Linux hosts, with name and version, so a package install of Zammad shows up in the **Software** tab alongside everything else. The [software inventory reference](https://fleetdm.com/guides/software-inventory-reference) lists exactly what Fleet collects on each platform. Docker containers aren't part of software inventory, so if Zammad runs in a container, use a [report](https://fleetdm.com/guides/queries) to look for it. Test any check on a host you know runs Zammad first. A check that returns nothing is only reassuring if you've seen it find a known install.


Test your check on a host you know runs Zammad before you trust it across the fleet. A query that returns nothing is only reassuring if you've seen it return something on a known install.

Once you have a list, version data tells you which copies fall in the affected range for the first flaw. For the second, every copy is on the list, so the useful follow-up is exposure: what can reach each server, and what could an attacker who got to the Zammad user reach next.

## Trusting the system's account of itself

Fleet has argued that [agentic security needs ground truth](https://fleetdm.com/articles/agentic-security-needs-ground-truth) rather than trust in a system's own account of itself. An automated attacker makes that concrete. It doesn't wait for an advisory, and it doesn't care what your ticket says about the patch status.

The same idea applies on the defensive side. A statement that "we upgraded" is a claim. A report from the host, showing the version it's running, is evidence.

## Where this leaves you

For the unpatched flaw, the work is containment: find every Zammad copy, reduce what can reach it, and watch it. For the version-dependent flaw, the work is verifying that each copy moved to Zammad 7. In both cases the starting point is the same list.

DIVD assumed breach, and segmentation limited what the attacker could reach. That only works for the servers you've mapped.

## See it live

-*Not sure where Zammad runs in your environment? [Talk to Fleet](https://fleetdm.com/contact)

- **Read the argument for ground truth:** [fleetdm.com/articles/agentic-security-needs-ground-truth](https://fleetdm.com/articles/agentic-security-needs-ground-truth)

## Sources

- Help Net Security, [AI agent used Zammad zero-days to breach Dutch vulnerability disclosure non-profit](https://www.helpnetsecurity.com/2026/10/01/divd-agentic-ai-attack-breach/).
- SecurityWeek, [Zammad zero-days exploited in AI-powered DIVD hack](https://www.securityweek.com/zammad-zero-days-exploited-in-ai-powered-divd-hack/).
- The Register, [AI agents hacked the hackers, stealing email addresses from security research org](https://www.theregister.com/security/2026/10/01/ai-agents-hacked-the-hackers-stealing-email-addresses-from-security-research-org/5300652).
- Zammad, [Releases](https://github.com/zammad/zammad/releases).
<meta name="articleTitle" value="Zammad zero-days behind the DIVD breach, and how to find every install">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-10-02">
<meta name="description" value="An AI agent chained two Zammad zero-days to reach root at DIVD. One flaw has no fix yet, so start by finding every Zammad server you run.">
