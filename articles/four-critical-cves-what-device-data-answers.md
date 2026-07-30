# Four critical CVEs in two weeks. Which ones can your device data answer for?

*July handed defenders two actively exploited critical flaws and two more that are only critical on paper so far. Here's an honest read on where device management helps, and where it doesn't.*

## Key takeaways

- **Two of July's four are under active attack, and two are not.** The SharePoint and Check Point flaws are being exploited and sit on CISA's Known Exploited Vulnerabilities list. The VMware and nginx flaws have no confirmed in-the-wild exploitation, which should change how you sequence the week.

- **Device management answers "what is installed, where, right now," and you should not expect more.** Fleet will not patch a SharePoint farm, a vCenter appliance, or a Check Point management server. Naming that boundary is what makes the parts it does answer worth trusting.

- **Some of this month's affected software genuinely lives on employee machines.** VMware Workstation and Fusion, and the management consoles your admins run, show up in device inventory across macOS and Windows, and that inventory is how you scope who is exposed.

- **A version match is not an exposure.** The nginx flaw only fires on a specific `map` directive pattern, so version-only scanning over-reports it. Reading the config tells you which servers are genuinely affected.

- **Patching does not close the SharePoint incident.** Attackers have been stealing machine keys, so a fully patched server can still be reachable by someone holding a forged token. Rotation, then verification, is the real close-out.

- **Anything you queried by hand this week can become a standing check.** Turn the queries into policies stored in Git, and next month's scramble starts from a dashboard instead of a blank terminal.

<a purpose="cta-button" href="/software-management">See your software exposure</a>

Four critical vulnerabilities landed in a two-week stretch this month, across SharePoint, Check Point, VMware, and nginx. If you run a device management platform, you probably got asked some version of "are we affected?" for all four, and the honest answer is different for each one.

That difference is worth spelling out, because the fastest way to lose credibility during a patch scramble is to over-claim what your tooling can see. Here's what's on the list, and then what your device data can and cannot tell you about it.

## What's on the list

| CVE | Affected software | Severity | Exploited in the wild | Fixed in |
|---|---|---|---|---|
| CVE-2026-50522 | SharePoint Server Subscription Edition, 2019, and 2016 Enterprise | CVSS 9.8 | Yes, following a public proof of concept | July 2026 Patch Tuesday |
| CVE-2026-16232 | Check Point Security Management Server and Multi-Domain Security Management | CVSS 9.1 | Yes, confirmed by the vendor, added to CISA KEV | Vendor advisory sk185169, July 22 |
| CVE-2026-59309 | VMware Directory Service, reached through vCenter | CVSS 9.8 | Not that Broadcom is aware of | VMSA-2026-0006, July 29 |
| CVE-2026-42533 | nginx, in configurations using a `map` directive with regex | CVSS v4 9.2, CVSS v3.1 8.1 | Not reported | nginx 1.30.4 and 1.31.3, NGINX Plus 37.0.3.1, July 15 |

A few details that matter for triage.

CVE-2026-50522 is a deserialization flaw in the `SessionSecurityTokenHandler` class, reachable with no authentication and no user interaction. It is the fourth SharePoint vulnerability exploited in roughly a month.

CVE-2026-16232 lets an unauthenticated attacker obtain an application login token and log in through SmartConsole with full administrator privileges, which means modifying firewall policy. Exploitation needs network access to the management server and a Trusted Clients configuration that does not restrict GUI clients, which researchers at Rapid7 found to be the default in their testing. CISA set a remediation due date of July 25, three days after the advisory.

VMSA-2026-0006 covers five CVEs, not one. Alongside CVE-2026-59309, there's CVE-2026-59310, a critical vCenter flaw allowing code execution with network access, and CVE-2026-47876, an out-of-bounds write in the ESXi VMXNET3 virtual network adapter that lets a local admin on a guest execute code on the host. CVE-2026-41703 is rated high and is the one that reaches beyond the data center: it affects ESXi, Workstation, and Fusion, and lets an attacker with VM deployment permissions obtain information.

CVE-2026-42533 is the most configuration-dependent of the four, and the most interesting for that reason. More on it below.

## The honest boundary

Fleet manages devices running fleetd: laptops, desktops, and Linux servers. It does not manage SharePoint farms as applications, ESXi hypervisors, vCenter appliances, or Check Point management servers. For the two most urgent items on that list, the server-side fix is a vendor patch applied by whoever owns that platform, and no amount of device inventory changes that.

So if you were hoping this article ends with "and Fleet remediates all four," it doesn't. What device data does well is narrower and still useful: it tells you which machines you own are running affected software, which ones you had forgotten about, and whether a fix landed. Three of this month's four have a real answer at that layer.

## Where your device data does answer

### VMware Workstation and Fusion on employee machines

CVE-2026-41703 affects Workstation and Fusion, which run on laptops and desktops rather than in a rack. That's ordinary software inventory, and it's the kind of install that spreads quietly through engineering teams.

On macOS:

```sql
SELECT name, bundle_short_version, path
FROM apps
WHERE name LIKE '%VMware Fusion%';
```

On Windows:

```sql
SELECT name, version, install_location
FROM programs
WHERE name LIKE '%VMware Workstation%';
```

Everything those return also rolls into Fleet's software inventory automatically, where Fleet matches installed versions against published CVE data. The live query is for when you need the answer before the next collection cycle, which during an active scramble is most of the time.

### The management consoles your admins run

The Check Point flaw is server-side, so finding SmartConsole on a workstation does not fix anything. It does help you scope the surrounding risk, because exploitation depends on who can reach the management server, and the machines with the console installed are a reasonable proxy for the administrative surface around it.

```sql
SELECT name, version, install_location
FROM programs
WHERE name LIKE '%SmartConsole%'
   OR name LIKE '%Check Point%';
```

Treat that as scoping input for a conversation with your network team, not as a remediation step.

### The SharePoint servers nobody remembered

If your Windows servers run fleetd, you can ask which of them have SharePoint installed. In practice this is most valuable for finding the instance that never made it onto the official asset list, which is reliably the one that stays unpatched.

```sql
SELECT name, version
FROM programs
WHERE name LIKE '%SharePoint%';
```

Patching is still the platform owner's job. Knowing the full list of servers that need it is yours.

## A version match is not an exposure

CVE-2026-42533 is the one worth slowing down on, because it shows why version-only scanning gives you a misleading answer.

The flaw is a heap buffer overflow in how nginx handles the `map` directive when regular expression matching is involved. It surfaces when a `map` using regex references a capture variable such as `$1` before the map's own output variable. nginx sizes the output buffer in one pass and writes it in another, the two passes disagree about length, and the write runs past the end of the allocation. F5, acting as CNA, scored it 9.2 critical on CVSS v4.0 and 8.1 high on CVSS v3.1, and notes that code execution may be possible where ASLR is disabled or can be bypassed.

The practical consequence: a server running an unpatched nginx is only genuinely exposed if its configuration contains that pattern. Version matching flags every unpatched install. Reading the config narrows it to the ones that matter.

Start with the version, across both package families:

```sql
SELECT name, version FROM deb_packages WHERE name LIKE 'nginx%'
UNION ALL
SELECT name, version FROM rpm_packages WHERE name LIKE 'nginx%';
```

Then look at the configuration. The `file_lines` table reads a file one row per line, and it needs an exact path, so start with the main config:

```sql
SELECT path, line
FROM file_lines
WHERE path = '/etc/nginx/nginx.conf'
  AND line LIKE '%map %';
```

Two honest caveats. Most real deployments split configuration across `include` directives, so this query alone is a starting point rather than a complete audit; enumerate the included files and read those too. And a `map` line by itself is not the bug. You are looking for a `map` block using a regex where a capture variable is referenced ahead of the output variable, which means a human still reads the results. What the query buys you is a shortlist of servers worth reading, instead of every server running nginx.

That distinction generalizes. A CVE feed tells you which versions are implicated. A query tells you what your machines look like. When a flaw is conditional, only the second one answers the question you were asked.

## Patching isn't the close-out for SharePoint

CVE-2026-50522 comes with a detail that changes the remediation plan: attackers exploiting it have been stealing SharePoint machine keys. With a stolen key, an attacker can forge trusted data and keep access after the patch is applied.

That means the checklist is patch, rotate machine keys, then verify, and the verification step is the one teams skip. Device data helps at the first and last steps, by confirming which servers exist and which are running the patched build, but the key rotation itself happens in SharePoint. Anyone who reports this closed on patch deployment alone has closed it early.

## Make it a standing check

The queries above took an afternoon to assemble and will be needed again next month for a different set of CVEs. Rather than rebuilding them each time, save them as reports and turn the ones that represent policy into Fleet policies stored in Git as YAML, reviewed in a pull request, and deployed through CI.

The payoff is that the next advisory starts from a different place. Instead of "does anyone know how many machines run this," you already have a count, a trend, and a list of the hosts that were offline the last time you looked.

## The uncomfortable part

Two of this month's four critical flaws are being actively exploited, and for both of them the fix lives outside your device management platform. That's worth sitting with, because the instinct during a scramble is to reach for the tool you know rather than the tool that applies.

Device data earns its place in that week by being specific about a narrower question: which machines do we own, what is on them, is the fix there, and what did we forget. Answer those honestly and quickly, and the people patching the servers can spend their attention on the servers.

## See it live

- [**Get a demo**](https://fleetdm.com/contact)**.** We'll run these queries against real machines and show you what comes back before the next advisory forces the question.
- [**Join a GitOps training session**](https://fleetdm.com/gitops-workshop)**.** If you want this month's queries to still be running next month, managing them as code is how that happens.

*Fleet is the open-source device management platform for macOS, Windows, Linux, and more. Want to know what your devices are running?* [*Get a demo*](https://fleetdm.com/contact) *or explore the* [*reports library*](https://fleetdm.com/reports)*.*

<meta name="articleTitle" value="Four critical CVEs in two weeks. Which ones can your device data answer for?">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-07-30">
<meta name="description" value="July brought four critical CVEs. An honest look at which ones device inventory can answer for, and which ones it can't.">
