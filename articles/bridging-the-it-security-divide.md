# Bridging the IT-security divide: one source of truth for both teams

If you've spent time in a mid-to-large technology organization, you know the dynamic.

Security flags a critical vulnerability and needs to know which devices are exposed, now. IT needs time to assess impact before touching anything. Security escalates. IT pushes back.

Both teams are right. Both teams are frustrated. And somewhere in the middle, the vulnerability sits unmitigated while the org chart works out its differences.

This friction is so common it gets treated as structural. An inevitable consequence of two teams with different mandates, different tooling, and different definitions of success. Security wants control and visibility. IT wants stability and efficiency. The goals appear to conflict.

The friction isn't caused by incompatible goals. It's caused because they're working from different data. Remove the data problem and the friction largely goes away.

## The IT-security divide is a data problem, not a people problem

Most IT teams manage devices from an MDM or endpoint management platform. Most security teams work from a vulnerability scanner, a SIEM, and a separate asset inventory, often three tools that don't agree with each other. When security asks "how many devices are running this vulnerable software version?" and IT asks "which devices need this patch?", they're querying different systems that were last synchronized at different times and often produce different answers.

Every cross-team workflow begins with a dispute over whose data is right before the actual work can start. Who owns the asset inventory? Whose count of unpatched devices is authoritative? Is that device in IT's spreadsheet or security's scanner? The dispute isn't about competence or culture. It's about tooling that was never designed to produce a shared, authoritative answer.

Fleet removes this structural cause of friction by maintaining a single, real-time inventory across all device platforms: macOS, Windows, Linux, iOS, Android, and ChromeOS. Both IT and security work from the same data. When security asks which devices are running a vulnerable version of a package, they get the same answer IT would get. There's nothing to reconcile.

This matters more in 2026 than it ever has. The time from vulnerability disclosure to active exploitation has collapsed from weeks to hours. When the exposure window is that short, the time organizations used to spend reconciling competing inventories before beginning remediation is no longer a process inefficiency. It's an unacceptable risk.

A shared inventory isn't just an operational convenience. It's a security requirement.

## Real-time self-service queries eliminate the biggest source of security-to-IT interruptions

Ask any IT team to describe their relationship with the security team and one pattern comes up repeatedly: the urgent, disruptive information request.

"Which devices don't have CrowdStrike running?" "What software is installed on this specific hostname?" "How many Linux servers are still on this kernel version?" These questions are legitimate and often time-sensitive. But when security has to route every device question through IT, it creates a dependency that shapes the entire relationship. IT becomes a reporting service. Security becomes a source of interruptions. Neither team gets what they actually need.

Fleet's live query capability changes this dynamic directly. Security can query the entire device estate themselves, in real time, using Fleet's osquery-powered query console. A question like "which devices are missing this specific patch?" runs against every enrolled device and returns results in seconds, without involving IT at all.

The query library includes hundreds of pre-built queries covering common security checks: encryption status, agent health, software inventory, running processes, user accounts, open ports, and more. Security engineers can write custom SQL queries against the osquery schema to answer questions that go beyond the library. IT doesn't need to be involved unless the answer requires action, which is exactly how it should work.

This shifts the relationship in a meaningful way. IT stops being a bottleneck between security and the data they need. Security stops generating a stream of disruptive, urgent requests that interrupt IT's operational work. Both teams spend more time doing their actual jobs and less time managing the coordination overhead between them.

There's a secondary benefit that compounds over time. When security can verify device state themselves rather than relying on IT-generated reports, trust increases. Security knows the data is current because they queried it themselves. IT knows security isn't questioning their work, they're working from the same source. The dynamic stops being adversarial and starts being collaborative.

## Shared policies and shared compliance evidence transform audit season from a scramble to a continuous practice

The audit process is where IT-security friction reaches its most damaging expression.

In most organizations, compliance evidence lives in multiple places. IT has configuration baselines and patch records. Security has vulnerability scan results and exception logs. The compliance team has its own tracking spreadsheets. When audit season arrives, all three groups race to reconcile their records, discover they're inconsistent, argue about which version is authoritative, and then spend weeks reconstructing evidence that should have been maintained continuously.

This isn't a compliance failure. It's a tooling failure. When policies are defined in one system, monitored in another, and reported through a third, continuous compliance is structurally impossible. You get point-in-time snapshots and periodic reconciliation instead of ongoing assurance.

Fleet approaches compliance differently. Policies are defined once, in code, using Fleet's infrastructure-as-code model, and they apply continuously across every enrolled device. Both IT and security can see policy status in real time: which devices are compliant, which are not, and exactly what the gap is. When a device falls out of compliance, Fleet surfaces it immediately. There's no waiting for the next scan cycle. There's no batch process to reconcile.

This matters for frameworks like SOC 2, CIS benchmarks, NIST, NIS2, and DORA. Evidence that used to be assembled under deadline pressure becomes continuously maintained data. The pre-audit reconciliation that consumed weeks of both IT and security's time doesn't need to happen because the records were never fragmented in the first place.

The practical change is significant. Both teams define compliance posture together, monitor it from the same platform, and can produce evidence on demand, not just during audit windows. IT's configuration baselines and security's control requirements are expressed in the same system, checked against the same inventory, and reported through the same interface. The audit becomes a review of ongoing practice rather than a reconstruction of historical state.

For organizations subject to multiple frameworks simultaneously, this is particularly valuable. Fleet's policy engine applies across all device platforms with the same query interface. A control that needs to be verified across macOS, Windows, and Linux servers doesn't require three separate tools and three separate reconciliation processes. It requires one query.

## What changes when the divide closes

When IT and security work from the same platform, the operational changes are concrete:

**Incident response accelerates.** When a critical vulnerability is disclosed, the exposure question is answered in seconds rather than days. IT and security are working from the same device inventory the moment the question is asked. There's no handoff, no reconciliation, no delay while the teams agree on which data to trust.

**Patch sequencing becomes rational.** With precise, real-time exposure data, IT can sequence remediation based on actual risk rather than worst-case estimates. Security doesn't need to pressure IT to patch everything immediately because both teams can see exactly which devices are exposed. The relationship becomes collaborative rather than adversarial.

**Compliance evidence is always current.** Neither team needs to scramble before an audit because the evidence was never scattered across disconnected systems. Both teams contributed to defining the policies. Both teams can see the compliance posture in real time. Both teams can speak to it confidently.

**The relationship changes character.** When IT and security work from the same data, the disputes over whose numbers are right disappear. Trust builds over time because each team can verify the other's work without accusation. The dynamic that was defined by friction starts to be defined by shared situational awareness.

## The bottom line

The IT-security divide is real. But it is not inevitable.

It is largely a consequence of fragmented tooling creating fragmented data creating fragmented decision-making. Remove the fragmentation and the divide starts to close.

Fleet gives IT and security teams a shared foundation: one inventory, one query interface, one policy framework, one view of every device across every platform. From that foundation, the workflows that used to generate friction become collaborative. The standoffs that used to delay security improvements become conversations. The scrambles that used to define audit season become routine operations.

Two teams. One platform. One shared understanding of what's running, what's configured, and what needs attention.

That's not just a better IT-security relationship. That's a better-secured organization.

[Talk to Fleet](https://fleetdm.com/contact) about how your IT and security teams can work from a single source of truth.

<meta name="articleTitle" value="Bridging the IT-security divide: one source of truth for both teams">
<meta name="authorFullName" value="Ashish Kuthiala">
<meta name="authorGitHubUsername" value="akuthiala">
<meta name="publishedOn" value="2026-06-11">
<meta name="category" value="articles">
<meta name="description" value="Why the friction between IT operations and security isn't inevitable, and what changes when both teams work from one platform.">
