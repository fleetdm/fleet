# Aur0ra ransomware: Old Active Directory attacks at AI agent speed

*Sophos says the Aur0ra crew let AI coding agents do the network and Active Directory recon. The techniques are old. The human doing the work is what changed, and the host is where it shows up first.*

## Key takeaways

- **The playbook is old, but AI agents now run it.** Sophos says the operators gave AI agents VPN credentials and a goal, then let them work through password spraying, Kerberoasting, and NTLM relay. Gambit Security found that most commands failed on the first try, so the agents adjusted and tried again. If you know these techniques, you know what to look for.
- **Domain admin came from credentials, not exploits.** In one case Sophos recovered, the actors used Cursor to go from stolen credentials to domain administrator. Reviewing who holds privileged access is still the highest-value fix.
- **Host-level signals are where this shows up first.** Sophos points to process, file, and named pipe detections on endpoints. Knowing what runs on each device, and who is logged in, gives you something to compare against.
- **AI tooling on a laptop is now part of your attack surface.** Sophos recovered Cursor artifacts tied to the operations. Fleet's agent can list every AI coding tool on your devices. Compare that list to the ones your team actually approved.
- **Fleet is not your EDR, and it does not need to be.** Fleet gives IT and security teams a cross-platform view of processes, accounts, and software that sits next to your detection stack.

<a purpose="cta-button" href="https://fleetdm.com/contact">Find every AI agent</a>

Every ransomware attack starts with homework. Someone scans the subnets, maps the domain, and hunts for the one account that opens every door. According to Sophos, the crew behind Aur0ra ransomware, which Sophos tracks as GOLD OGRE, handed that homework to AI coding agents and checked in only now and then.

That changes the clock more than the craft. Here is what Sophos found and what it means for the people who manage the laptops and servers underneath.

## What Sophos found

In its [report](https://www.sophos.com/blog/ai-used-in-aur0ra-ransomware-testing-and-execution), Sophos says the operators ran agents on Cursor's Composer model and on models from Anthropic, xAI, and Moonshot. To get past the models' safeguards, the operators told them the break-ins were sanctioned red team exercises or "diagnostics testing." [Gambit Security](https://gambit.security/blog-posts/aurora-ransomware-targets-esxi-abuses-cursor-agent-for-exploitation), which studied chat logs from the same crew, saw the agent running Anthropic's Claude Sonnet.

The agents did the discovery work: mapping the network and Active Directory, scanning internal subnets with Nmap or NetExec, enumerating the domain with BloodHound, and working out which credentials led where. Then they went after accounts with password spraying, Kerberoasting, and NTLM coercion and relay through PetitPotam, Coerce Plus, and PrinterBug. In a subset of environments, they got domain management access. In one late-August case, Sophos says the actors used Cursor to climb from stolen credentials to domain administrator, then collected data and deployed ransomware.

Victim counts vary by source. Sophos says three of the agent-driven intrusions ended with victims posted on Aur0ra's leak site. Gambit counted 10 targets between April 8 and May 21. [The Hacker News](https://thehackernews.com/2026/08/aurora-ransomware-operators-use-cursor.html), citing CloudSEK, reports more than 20 organizations in nine countries. Treat the real number as unknown.

## Why the old techniques matter more now

None of these techniques is new. An agent can repeat days of careful human work without getting tired. Sophos's advice stays the same: patch internet-facing devices, use phishing-resistant MFA, monitor EDR alerts, and respond quickly.

What changes is the time between the first foothold and domain admin. Detection that waits for a person to notice a slow trickle of odd behavior has less time to work.

## What the host can tell you

Sophos lists endpoint-level detections tied to this activity: Cobalt Strike named pipes and stager persistence, process-level detections for PetitPotam, NetExec, and PowerShell Kerberoasting scripts, and file-modification detections for Certipy output files.

Your EDR owns those detections. Fleet can add context an EDR alert does not carry by default. Reports can ask devices which processes are running, which local accounts exist, which ports are listening, and which software is installed, on macOS, Windows, and Linux. When an alert fires, the same questions run across the rest of the fleet show whether it is one machine or many.

Two limits are worth stating plainly. Fleet is not a replacement for endpoint detection and response, and it does not stop an intrusion. And a report only helps if you wrote the question before the incident.

## Treat AI tooling as inventory

Sophos recovered Cursor artifacts tied to Aur0ra's operations. Those were the attackers' tools, not a sign that Cursor is unsafe for your team. Still, the episode is a reminder that coding agents carry real capability and often run with the permissions of whoever installed them.

Start with a plain list of which AI coding tools run on which devices, and compare it to what your organization approved. Software inventory in Fleet answers that across operating systems. The list will not tell you whether an agent is being misused, but it tells you where to look.

## Narrow what a fast attacker can reach

Several of Sophos's recommendations are about limiting movement: segment critical servers, isolate backups, review domain admin privileges, use privileged access management, and apply application controls.

On the device side, reports on local accounts and group membership help you find stray administrators. Policies in Fleet flag hosts that drift from your baseline, so cleanup becomes a list instead of a hunt.

## The takeaway

AI agents gave a ransomware crew speed, not new tricks. That's good news for defenders: the fundamentals still work. Apply them assuming an attacker will move faster than your team can read an alert. When you know what runs on every device before an incident, you can answer the first question quickly: is it just this machine?

## See it live

- **Get a demo:** [Talk to the Fleet team](https://fleetdm.com/contact) about building host-level visibility across your devices.
- **Rather try it yourself?** Skip the demo and send your IT team to a [free, hands-on Fleet workshop](https://fleetdm.com/workshops). In four hours, they'll learn how to write policies like the stray-admin check, keep them in Git, and review every change before it ships. They'll also earn a Fleet Level 1 certificate.

## Sources

- Sophos Counter Threat Unit, [AI used in Aur0ra ransomware testing and execution](https://www.sophos.com/blog/ai-used-in-aur0ra-ransomware-testing-and-execution).
- Gambit Security, [Aurora ransomware targets ESXi, abuses Cursor Agent for exploitation](https://gambit.security/blog-posts/aurora-ransomware-targets-esxi-abuses-cursor-agent-for-exploitation), August 27, 2026.
- The Hacker News, [Aurora Ransomware Operators Use Cursor AI in Attacks Against 10 Targets](https://thehackernews.com/2026/08/aurora-ransomware-operators-use-cursor.html), August 31, 2026.

<meta name="articleTitle" value="Aur0ra ransomware: Old Active Directory attacks at AI agent speed">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="publishedOn" value="2026-11-02">
<meta name="category" value="industry news">
<meta name="description" value="Sophos says the Aur0ra ransomware crew used AI agents for Active Directory recon. Here is what to watch on the devices underneath.">
