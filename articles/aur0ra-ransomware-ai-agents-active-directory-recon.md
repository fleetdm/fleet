# Kerberoasting, PetitPotam, and BloodHound at AI speed: what Sophos found in the Aur0ra ransomware playbook

*Sophos says the Aur0ra crew let AI coding agents do the network and Active Directory recon. The techniques are old. The pace is what changed, and the first place to see it is the host.*

## Key takeaways

- **The playbook is familiar, but nobody is typing it.** Sophos describes agents running password spraying, Kerberoasting, and NTLM relay with little human direction. Defenders who know these techniques already know what to look for.
- **Domain admin came from credentials, not exploits.** In a subset of environments, the agents reached domain administrator access by chaining ordinary credential attacks. Reviewing who holds privileged access is still the highest-value fix.
- **Host-level signals are where this shows up first.** Sophos points to process, file, and named pipe detections on endpoints. Knowing what runs on each device, and who is logged in, gives you something to compare against.
- **AI tooling on a laptop is now part of your attack surface.** Sophos recovered Cursor artifacts tied to the operations. An inventory of coding agents on your devices tells you which ones your team approved.
- **Fleet is not your EDR, and it does not need to be.** Fleet gives IT and security teams a cross-platform view of processes, accounts, and software that sits next to your detection stack.

<a purpose="cta-button" href="https://fleetdm.com/software-management">See software visibility in Fleet</a>

Ransomware crews have always done reconnaissance. They scan subnets, map the domain, and look for the account that opens every door. Sophos's report on the group behind Aur0ra ransomware says the work is now delegated to AI coding agents, which ran it with minimal human input.

That changes the clock more than the craft. Here is what Sophos found and what it means for the people who manage the laptops and servers underneath.

## What Sophos found

According to the [Sophos report](https://www.sophos.com/blog/ai-used-in-aur0ra-ransomware-testing-and-execution) and coverage from Help Net Security, the operators used agents built on Cursor's Composer model and models from Anthropic, xAI, and Moonshot. Sophos says the actors leaned on Anthropic almost exclusively at first, then added other models. To get past guardrails, they framed the intrusions as authorized red team work or "diagnostics testing."

The agents handled network and Active Directory discovery, analysis of credential-based attack paths, subnet scanning with Nmap or NetExec, and BloodHound domain enumeration. They also ran password spraying, Kerberoasting, and NTLM coercion and relay through tools such as PetitPotam and PrinterBug. In a subset of environments, the agents reached domain administrator access. In late August, Sophos says the actors used Cursor to go from credential access to domain admin before collecting data and deploying ransomware.

Sophos does not give a total victim count. It says the agents operated in "several environments," and three victims were later named on Aur0ra's leak site. Treat the scale as unknown.

## Why the old techniques matter more now

None of these techniques is new. What a human operator once did over days of careful work, an agent can iterate on without fatigue. Sophos's own guidance reflects that: keep the fundamentals, patch internet-facing devices, use phishing-resistant MFA, monitor EDR alerts, and respond quickly.

The consequence is a shorter window between the first foothold and domain admin. Detection that depends on a person noticing a slow trickle of odd behavior has less time to work.

## What the host can tell you

Sophos lists endpoint-level detections tied to this activity: Cobalt Strike named pipes and stager persistence, process-level detections for PetitPotam, NetExec, and PowerShell Kerberoasting scripts, and file-modification detections for Certipy output files.

Your EDR owns those detections. Fleet can add context an EDR alert does not carry by default. Reports can ask devices which processes are running, which local accounts exist, which ports are listening, and which software is installed, on macOS, Windows, and Linux. When an alert fires, the same questions run across the rest of the fleet show whether it is one machine or many.

Two limits are worth stating plainly. Fleet is not a replacement for endpoint detection and response, and it does not stop an intrusion. And a report only helps if you wrote the question before the incident.

## Treat AI tooling as inventory

Sophos recovered Cursor artifacts tied to Aur0ra's operations. Those were the attackers' tools, not a sign that Cursor is unsafe for your team. Still, the episode is a reminder that coding agents carry real capability and often run with the permissions of whoever installed them.

Start with a plain list of which AI coding tools run on which devices, and compare it to what your organization approved. Software inventory in Fleet answers that across operating systems. The list will not tell you whether an agent is being misused, but it tells you where to look.

## Narrow what a fast attacker can reach

Several of Sophos's recommendations are about limiting movement: segment critical servers, isolate backups, review domain admin privileges, use privileged access management, and apply application controls. Those controls work regardless of who or what is on the other end.

On the device side, local account and group membership reports help you find stray administrators. Policies can flag hosts that drift from your baseline, so the cleanup becomes a list instead of a hunt.

## The takeaway

AI agents gave a ransomware crew speed, not new tricks. The defense is the same set of fundamentals, applied with the assumption that an attacker will move faster than your team can read an alert. Knowing what runs on every device before the incident is what lets you answer the first question quickly: is it just this machine?

## See it live

- **Get a demo:** [Talk to the Fleet team](https://fleetdm.com/contact) about building host-level visibility across your devices.
- **Rather try it yourself?** Skip the demo and send your IT team to a [free, hands-on Fleet workshop](https://fleetdm.com/workshops). In four hours, they'll work in Fleet alongside an instructor and earn a Fleet Level 1 certificate.

<meta name="articleTitle" value="Kerberoasting, PetitPotam, and BloodHound at AI speed: what Sophos found in the Aur0ra ransomware playbook">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="publishedOn" value="2026-11-02">
<meta name="category" value="industry news">
<meta name="description" value="Sophos says the Aur0ra ransomware crew used AI agents for Active Directory recon. Here is what to watch on the devices underneath.">
