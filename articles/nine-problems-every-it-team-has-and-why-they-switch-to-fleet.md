# 9 problems every IT team has, and why they switch to Fleet

I have spoken with multiple IT leaders: CIOs at Fortune 500 companies, client platform engineers at 200-person startups, and everyone in between. The tools are different. The org charts are different. The pain is the same.

After hundreds of these conversations, I've noticed the same nine problems keep coming up. Not vague complaints, specific, structural problems with how legacy device management works. Problems that Jamf, Intune, and Workspace ONE cannot fix because they stem from the same architectural choices.

Here's what I keep hearing.

## 1\. The last mile: stuck at 75% patched

You push a patch. It hits 40,000 devices. Then it stops at 73%. Your MDM says "failed". That's it. No error output. No exit code. No explanation. You're stuck re-pushing to the entire fleet or manually hunting through logs on individual machines.

### How Fleet fixes this

**Full error output. Live queries. Targeted remediation.**

Fleet shows you the actual CLI output, raw MDM XML, and exact exit codes for every failed device, all in the UI. You paste the error into your AI assistant and get a root cause diagnosis in seconds. One team increased patch coverage from 73% to 99.8% in a single sprint after seeing the real errors for the first time.

 

## 2\. No peer review: someone ruined your weekend

A teammate pushes a config change directly from the Jamf GUI. No review. No approval. No diff. It breaks Wi-Fi on every Mac in the building. Friday night becomes a war room.

This happens because legacy MDMs lack version control. Changes go live the instant someone clicks "Save."

#### How Fleet fixes this

**GitOps. PR review. Instant rollback.**

Fleet's GitOps model puts every configuration in a YAML file in your Git repo. Changes require a pull request. A teammate reviews the diff. Only after approval does Fleet apply it. And if something goes wrong, one git revert rolls it back across your entire fleet in seconds, not hours.

 

## 3\. Left behind: Every team uses AI except IT

Engineering uses AI to write code and open PRs. Sales uses AI to draft outreach. Finance uses AI to model scenarios. But your MDM has no version control, so there's nowhere for AI to propose a change safely. You're locked out of the productivity gains every other department is getting.

### How Fleet fixes this

**AI proposes. Human approves. Fleet deploys.**

Fleet is the only MDM with a GitOps foundation. That means AI can draft a configuration change as a pull request, a teammate can review the diff, and Fleet applies it after approval. That's how a 3-person IT team ships like a 10-person team.

 

## 4\. The roach motel: you checked in, you can't check out

Jamf raises prices 30% at renewal. They strip features from on-prem to force you to their cloud. Switching feels impossible because they made it that way: proprietary config formats, no real export, migration complexity designed to keep you stuck.

### How Fleet fixes this

**Open source. Deploy anywhere. Leave anytime.**

Fleet’s core is MIT-licensed. Self-host it or let Fleet Cloud manage it. Identical features either way. Your device configurations live in your Git repo, not a proprietary database. You can read them, modify them, and take them to any future tool without a migration project. The vendor relationship is a choice, not a trap.

 

## 5\. Tool sprawl: six APIs for three operating systems

Jamf for Mac. Intune for Windows. Something else (maybe) for Linux. A separate vulnerability scanner. Another tool for compliance reporting. Each has its own API, authentication flow, and response schema.

Want to know which devices across your entire fleet are running an outdated version of Chrome? In the legacy world, that's three API calls, three data formats, and a spreadsheet to merge them. In Fleet, it's one SQL query across every OS, with results in about 3 seconds. Fleet's REST API and MCP server work identically regardless of whether the device runs macOS, Windows, or Linux.

### How Fleet fixes this

**One REST API. One MCP server. Every platform.**

Fleet's single API works identically whether you're querying a Mac, Windows, or Linux device. 400+ built-in osquery+fleet tables. Native MCP server for AI integration. Write your automation once, run it everywhere.

 

## 6\. The Linux blind spot: your most powerful users are invisible

About 15% of enterprise engineers use desktop Linux. They typically have the most elevated access, the most sensitive data, and the most creative security workarounds. Jamf doesn't support Linux at all. Intune offers limited Linux enrollment for compliance checking (Ubuntu Desktop and RHEL only), but not full device management. Those devices remain largely unmanaged by your security team.

A senior engineer's Ubuntu laptop gets lost at a conference. You can't remotely lock it. You can't wipe it. You don't even know what was on it. Fleet manages Linux the same way it manages Mac and Windows: full visibility, vulnerability management, script execution, remote lock and wipe, and CIS benchmarks. 

### How Fleet fixes this

**Full Linux management. Same API. Same dashboard.**

Fleet treats Linux as a first-class citizen. Ubuntu, CentOS, RHEL, Fedora, Debian, OpenSUSE: full device visibility, vulnerability management, script execution, remote lock-and-wipe, CIS benchmarks. Every feature available on Mac and Windows is available on Linux.

 
## 7\. Busywork: 20% of your team's time, gone

A client platform lead at a major technology company told us about his 13-person IT team. They spend roughly 20% of their time gathering data for auditors, security reviews, and leadership questions. That's the equivalent of 2.6 full-time engineers running scripts, exporting CSVs, and merging spreadsheets.

### How Fleet fixes this

**Live compliance dashboard. Zero prep.**

With Fleet, your auditor gets a URL. They open a live dashboard. Encryption status, EDR coverage, patch compliance, and CIS benchmark results. All current to the second. You didn't prepare anything because there's nothing to prepare. Fleet's policy engine runs continuously. You're always audit-ready, not just the week before the audit.

 
## 8\. Off limits: You acquired a company and can't see their devices

You close an acquisition. 500 employees. Different MDM. Different IT team. Your CISO wants to know their security posture immediately. But forcing a full MDM migration in week one destroys morale and takes months.

### How Fleet fixes this

**Visibility from day one. Migration on your timeline.**

Fleet solves this without forcing a migration. Deploy the Fleet agent via their existing MDM: Workspace ONE, Jamf, Intune, whatever they have. Within minutes, every device appears in Fleet. Hardware inventory, installed software, vulnerability exposure, and compliance status. Their existing MDM stays in place. Zero user disruption. Migrate to full Fleet management on your timeline, not under pressure.

 
## 9\. Stumped: you know what's happening but can't explain it

Your CISO wants a compliance summary before the investor call. Your CFO wants to justify the endpoint security budget. Your board audit committee wants a quarterly security posture. Your legal team needs M\&A due diligence data. Each question requires a different tool, a different export, and a different manual reconciliation process.

### How Fleet fixes this

**One source of truth. Every audience. Live.**

Fleet turns any device question into a live, shareable, always-current answer. A single dashboard URL replaces a week of audit prep. Fleet's MCP server enables non-technical stakeholders to query device data in plain English via an AI assistant. And the REST API delivers structured data to any downstream tool: BI dashboards, partner portals, executive reports.

 
## The common thread

These nine problems aren't nine separate issues. They're symptoms of the same root cause: legacy MDMs were built for a simpler era. They assumed one OS, one deployment model, one way of working. They assumed IT teams would click through GUIs forever.

Fleet was built for how IT actually works now:  multi-platform, code-first, AI-assisted, and accountable to stakeholders who expect real-time answers. It's open source, it runs anywhere, and it treats every operating system as a first-class citizen.

If even two or three of these problems sound familiar, it might be worth seeing what Fleet looks like in your environment.


**Get a demo:** [https://fleetdm.com/contact](https://fleetdm.com/contact)


<meta name="articleTitle" value="9 problems every IT team has, and why they switch to Fleet">
<meta name="authorGitHubUsername" value="akuthiala">
<meta name="authorFullName" value="Ashish Kuthiala, CMO at Fleet">
<meta name="publishedOn" value="2026-04-14">
<meta name="category" value="articles">
<meta name="description" value="After hundreds of conversations with IT leaders, nine structural problems keep coming up. Here's what they are and how Fleet solves them.">
