# BT26-05-RS fixes the appliance, but Armory Castle used it to reach the Jump Item systems behind it

*BeyondTrust's patch closes CVE-2026-102764 on the Remote Support appliance. The intruders' real prize was the credentials on it, and those open doors the patch never touches.*

## Key takeaways

- **Patching is the start of the response, not the end.** The flaw was exploited as a zero-day on August 29, weeks before BeyondTrust's October 15 advisory. An appliance that was reachable and unpatched in that window should be treated as potentially compromised, even once it's on the fixed build.
- **Know which build you were running, and when.** Self-hosted installs from 24.3.1 through 26.2.3 need the BT26-05-RS patch, and anything older has to upgrade first. SaaS customers were patched as of October 9, so the open question sits with self-hosted teams.
- **The appliance is a credential vault.** Mandiant reports the intruders used the `bt-secrets` command to pull passwords, API keys, and endpoint credentials from the appliance, then used admin credentials to reach systems in the Jump Item list.
- **Hunt on the systems behind it.** Those downstream systems are where file access and data theft happened. Checking them is a device question, and it's one a team can answer for the computers and servers it manages.
- **Fleet covers the surrounding devices, not the appliance.** Don't expect Fleet's agent to run on a vendor appliance. It can show which machines the appliance could reach, who is logged in, and what's installed.
- **Rotate what the appliance held.** A patched appliance that still holds stolen credentials is still a problem. The finish line is a rotated secret list and a reviewed Jump Item list.

<a purpose="cta-button" href="https://fleetdm.com/guides/get-current-telemetry-from-your-devices-with-live-queries">See live reports in Fleet</a>

On October 15, 2026, BeyondTrust published advisory BT26-05 for CVE-2026-102764, a CVSS 9.3 improper origin validation flaw in an internal messaging component of Remote Support, formerly Bomgar. A remote attacker can use it to perform file operations on the appliance. Google's Mandiant attributes the activity to Armory Castle, a China-nexus espionage group also tracked as UNC5221, and says it exploited the flaw against a self-hosted appliance on August 29.

Most coverage stops at "apply the patch." That's necessary, and it's also the easy part. Remote Support exists to reach other computers, so the useful question is what an attacker with root on the appliance could do next, and what you can check about the systems it connects to.

## What the attackers did after the flaw

According to [The Hacker News](https://thehackernews.com/2026/10/china-linked-hackers-exploited.html), the group deployed SAWFLY, a lightweight PHP web shell, and gained root by copying `/bin/sh` to `/usr/bin/dbus-authd` and setting it to root-owned with the SUID bit. They then used the `bt-secrets` command to harvest passwords, API keys, and endpoint credentials from the appliance's secrets database. They also abused configuration templates to get passwordless root for the site service account, ran a SOCKS5 proxy called BARECLAW, edited `/ns/config/haproxy.conf` to expose it to the internet, and deleted system, web server, and application logs.

The step that matters for everyone downstream came next. The group used compromised admin credentials to pivot to systems in the Jump Item list, obtained full file-system access on them, and exfiltrated internal documents. The log deletion is the reason "we found nothing" on the appliance is weak comfort.

## Confirm which build you ran

The affected population is self-hosted Remote Support. Installs from 24.3.1 through 26.2.3 should apply the BT26-05-RS patch, and installs older than 24.3.1 have to upgrade before the fix is available. BeyondTrust pushed the fix to SaaS customers as of October 9.

Take the version numbers from [BeyondTrust's advisory](https://www.beyondtrust.com/trust-center/security-advisories/bt26-05), not from this article, and record three things per appliance: the version it ran, whether it was internet-reachable, and the date the patch landed. If an appliance was reachable and unpatched at any point since August 29, review it as a possible compromise rather than a closed ticket. The review itself belongs with BeyondTrust's guidance and your incident response process.

## Where Fleet fits, and where it doesn't

Fleet's agent runs on computers and servers. Don't count on it running on a vendor-supplied appliance, so BeyondTrust's own tooling has to answer questions about the appliance itself.

The systems behind it are different. Your Jump Item list names the machines the appliance could reach, and those are usually computers and servers that Fleet can report on. A [report](https://fleetdm.com/guides/get-current-telemetry-from-your-devices-with-live-queries) can pull current logged-in users, listening ports, and `authorized_keys` entries from those hosts, and [software inventory](https://fleetdm.com/guides/software-inventory-reference) shows what's installed on them and on the admin workstations whose credentials were stored on the appliance.

```sql
SELECT user, type, host, time FROM logged_in_users;
```

That won't prove a host is clean. It gives you a current, comparable picture of the machines an intruder with the appliance's credentials could have touched, so the review starts from data instead of guesses.

## Rotate what the appliance held

If the appliance stored credentials, assume anyone with root on it could read them. That includes passwords, API keys, and endpoint credentials. Work down the Jump Item list, rotate what each entry used, and check recent access on those systems. A team that patches and skips rotation has fixed the entry point and left the loot.

## What done looks like

For each self-hosted appliance, you want one page: the version it ran, the patch date, whether it was reachable before the patch, and who reviewed it. Next to it, you want the Jump Item list with a rotation date against every credential.

Patch day closes one door. The record of what was behind it is what tells you whether the work is finished.

## See it live

- [**Get a demo**](https://fleetdm.com/contact) to see current logged-in users, ports, and software reported across your own devices.
- [**Read how Fleet handles patch management**](https://fleetdm.com/guides/automated-patch-management) for the day-to-day version of this check.

<meta name="articleTitle" value="BT26-05-RS fixes the appliance, but Armory Castle used it to reach the Jump Item systems behind it">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-10-16">
<meta name="description" value="BeyondTrust's BT26-05-RS patch fixes the appliance. Armory Castle used it to steal credentials and reach other systems. Here's what to check.">
