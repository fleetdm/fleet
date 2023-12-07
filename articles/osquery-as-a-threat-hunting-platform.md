# Osquery… as a threat hunting platform?

![osquery… as a threat hunting platform?](../website/assets/images/articles/osquery-for-threat-hunting-1600x900@2x.jpg)

Detecting and containing a security incident isn’t easy even in the simplest of computer infrastructures. Responders need to piece together the timeline of events that led to an intrusion. And they need to do so quickly.

In order to reconstruct an incident, you want as much information as possible. One of the most comprehensive strategies is using forensic tools to identify suspicious activity from device memory lists. But this takes a lot of time and effort.

Traffic on network architectures comes from multiple machines and more “off-network” devices than ever before, many of which aren’t pre-approved. This limited visibility leaves endpoints vulnerable to a variety of attacks.

The more sophisticated systems are, the more taxing it is to detect and contain a security incident. For many organizations, dwell time (the duration between the initial compromise and detection) can range from several hours to a few months.

Security teams must use proactive threat detection strategies and reactive incident response plans to boost data security across the board and limit the risk of attacks. Beyond observing an incident in the making, cybersecurity professionals need real-time insights to defend their endpoints. But where should you start?

## Introducing osquery

![osquery… as a threat hunting platform?](../website/assets/images/articles/osquery-for-threat-hunting-2-1600x900@2x.jpg)

[Facebook engineers built osquery](https://fleetdm.com/podcasts/the-future-of-device-management-ep1) to inspect complex device inventories. This open-source agent makes it easy to monitor operating system internals for computers. It extracts a rich data set from a system that you can easily query to uncover specific artifacts linked to that system. But collecting quality data wasn’t the only reason for creating osquery.

Imagine how many endpoints an organization like Facebook has. Inspecting all these devices could strain systems and diminish performance if not cause downtime. That’s why osquery was designed to be lightweight. Security teams can identify, investigate, and proactively track threats on hundreds of thousands of devices — making osquery a powerful tool for triage.

Simply put, osquery acts as a single source of truth for security responders who need detailed data from every workstation and server. It’s a threat hunting platform for large-scale monitoring and detection of indicators of compromise (IoC) as well as Tactics, Techniques, and Procedures (TTP).

This provides an important link between analysts and operating system internals. Analysts can query running processes, changes in the file system, logged-in users, loaded kernel modules, installed packages, and Syslog messages — all from a database-like structure.

## Osquery for incident response

The osquery framework lets you explore an endpoint’s operating systems while using Windows, Linux, or Mac as a relational database. This allows incident responders to run standard SQL queries to retrieve information about computers.

You can view artifacts like running processes, bash history, open network sockets, listening ports, process trees, and Docker containers. Every artifact type is assigned its distinct table in the virtual database. Since it uses SQL, and many of the tables are cross-platform, the same queries can often be used across different operating systems.

With osquery, you can use queries to ask devices many different questions that help you identify, monitor, and manage threats. For instance, a query could be written to detect all processes currently running on a system or to flag servers with a root login during a specific time frame. Such queries are crucial when performing an audit of a system or investigating a breach.

Osquery lets you collect device data that could help you hunt for threats and respond to them when exploited. Security teams can install osquery and run scheduled or real-time queries. This reliable data helps blue teamers define a baseline and set flags for outlying behavior that might indicate a security threat. But first, you have to know which queries to run.

Maintained by [Recon InfoSec](https://twitter.com/Recon_InfoSec), the [Recon Hunt Queries](https://rhq.reconinfosec.com/) repo consolidates queries focused on incident response and threat hunting. Browse [general queries](https://rhq.reconinfosec.com/general/file_enumeration/) or find queries [by tactic](https://rhq.reconinfosec.com/tactics/initial_access/). The [Threat Hunting with Osquery](https://github.com/Kirtar22/ThreatHunting_with_Osquery) repo also has dozens of queries to help cyber threat analysts with their hunting or investigation exercises. Cloud Security Engineer [Pepe Burba](https://twitter.com/__pberba__) wrote a blog series about [hunting for persistence in Linux](https://pberba.github.io/security/2021/11/22/linux-threat-hunting-for-persistence-sysmon-auditd-webshell/#overview-of-blog-series). He explains how you can use osquery to [find evidence of web shells](https://pberba.github.io/security/2021/11/22/linux-threat-hunting-for-persistence-sysmon-auditd-webshell/#17-hunting-for-web-shells-using-osquery).

## Reduce security risk

Companies need to rethink the fragmented, siloed approaches to cybersecurity. Most solutions use separate proprietary agents for threat detection, incident response, and compliance before operating system sprawl. This increases complexity and could result in more points of failure.

Looking inside computers shouldn’t be this difficult. Your threat hunting platform should be a single source of truth. But osquery isn’t limited to endpoint security. It’s one solution that provides workstation and server visibility across IT, SRE, and even DevOps.

Osquery is a powerful platform. Like any new tool, it will take time and resources to make the most of it. Luckily, osquery managers simplify implementation and management for security teams. That’s where Fleet can help. Fleet makes it easy for companies to harness the power of osquery at scale. Fleet comes out of the box with a [query library](https://fleetdm.com/queries) that’s maintained by members of our community. So, you can start collecting accurate, actionable endpoint data right away. [Try `fleetctl preview`](https://fleetdm.com/try-fleet/register) to test Fleet on your device. Happy hunting.

<meta name="category" value="security">
<meta name="authorFullName" value="Chris McGillicuddy">
<meta name="authorGitHubUsername" value="chris-mcgillicuddy">
<meta name="publishedOn" value="2022-09-16">
<meta name="articleTitle" value="Osquery… as a threat hunting platform?">
<meta name="articleImageUrl" value="../website/assets/images/articles/osquery-for-threat-hunting-1600x900@2x.jpg">
