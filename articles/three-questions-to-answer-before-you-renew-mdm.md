# Three questions to answer before you renew your MDM

*Renewal is the one moment each cycle when you have real leverage over your device management stack. Here is what to check before you sign.*

## Key takeaways

- **Shadow AI is already on your endpoints, whether your MDM can see it or not.** Fleet inventories AI tooling, from MCP server configurations to browser extensions and IDE plug-ins, across macOS, Windows, and Linux, so AI governance covers what is actually installed, not just what a SaaS catalog happens to know about.
- **Vulnerability data means nothing without prioritization or depth.** Fleet matches installed software, down to browser extensions, IDE plug-ins, and developer packages, against CVE feeds from NVD, CISA's Known Exploited Vulnerabilities catalog, and EPSS exploit-probability scores, so your team patches what is actually being targeted instead of triaging a shallow, apps-only list by CVSS score.
- **You are probably renewing multiple contracts, not one.** Most teams pay for a separate tool per operating system, plus a vulnerability scanner and a compliance reporting tool. Fleet manages macOS, Windows, Linux, iOS, iPadOS, and Android, including devices most MDMs never reach, from one server and one API.
- **Configuration as code decides what your team can do for the next three years.** With Fleet, every profile, policy, and report lives in Git, gets reviewed in a pull request, and rolls back with one revert. That is also the only safe way to let AI draft changes to your fleet.
- **You can watch your own bugs and feature requests move, instead of waiting on a ticket number.** Fleet tracks issues and feature requests in public GitHub repos with a quarterly public roadmap, so you see what is being worked on rather than guessing at a support queue's priorities.
- **Leaving is possible even if you never plan to.** Fleet's core is open source and your configuration is portable YAML, which keeps switching a choice instead of a project a price hike forces on you.

<a purpose="cta-button" href="https://fleetdm.com/pricing">Compare Fleet</a>

Renewal season tends to arrive as a formality. A quote lands in your inbox, the number is higher than last year, and the path of least resistance is to sign it, because the alternative is a migration project nobody has budget or appetite for.

That instinct is understandable, and it is also the reason the number goes up every year. Renewal is the one point in the cycle where your incumbent has to answer questions, so it is worth spending a week on three of them before you sign.

## 1. Can you see what is actually running, and what is actually risky?

Most MDMs will tell you a device is "compliant" and stop there. That answer is only as good as the data behind it, and most MDM inventory is shallow by design: apps and OS version, refreshed on a slow check-in cycle. Three blind spots show up constantly: vulnerability data with no prioritization, an inventory that stops well short of everything actually installed, and AI tooling nobody classified as software in the first place.

### Vulnerability detection that tells you what to do next

A list of CVEs on a device is not the same as knowing what to fix first. Fleet identifies installed software fleet-wide and cross-references it against the National Vulnerability Database, CISA's Known Exploited Vulnerabilities catalog, and EPSS probability scores, so a critical CVSS score with no real-world exploitation ranks below a lower-severity bug that is actively being used in attacks. That prioritization, plus policy scoring and continuous compliance checks, is native to Fleet rather than something you buy from a third-party scanner and reconcile by hand.

### Inventory that goes deeper than an app list

Fleet is built on osquery, which means device state is a live, queryable database rather than a periodic snapshot. Beyond the operating system and installed applications, that includes package managers (Homebrew, deb and RPM packages, Python and npm packages), browser extensions across Chrome, Edge, Brave, Firefox, and Safari, with their permissions and whether they came from an official store, IDE extensions and forks like VS Code, Cursor, and Windsurf, installed certificates in both the system and user keychains (or the Windows certificate store), and disk encryption status for FileVault, BitLocker, and Linux LUKS. Any of it is a query away, fleet-wide, in seconds, not a report you schedule and wait on.

That depth matters because most of a device's real attack surface lives in exactly the categories a slow, apps-only inventory misses: a browser extension with broad permissions, a stale certificate, an unencrypted disk that never got flagged.

### AI governance starts with knowing what is installed

The newer version of this problem is AI tooling. A developer can install an AI coding assistant, wire it up to a handful of MCP servers, and hand an agent real access to code and credentials before lunch, with no help ticket and no entry in your SaaS catalog. Your identity provider does not see it because it was never a sanctioned app. Your EDR tends to wave it through, because a signed assistant calling an MCP server is not an attack.

You catch it the same way you catch anything else on a device: by inventorying what is there. Fleet's agent turns every macOS, Windows, and Linux host into something you can query live, including running MCP servers, the MCP client configurations in Claude Desktop, Claude Code, Cursor, VS Code, and similar tools, installed IDE extensions and forks, and the browser extensions, AI assistants, and "summarize this page" tools that quietly accumulate with broad permissions. That inventory rolls into the same software table Fleet already uses for vulnerability detection, so shadow AI is not a separate governance project. It is one more thing you already have visibility into.

## 2. How many contracts are you actually renewing?

The renewal on your desk is rarely the whole bill. Count the line items: one MDM for Mac, something else for Windows, a partly solved story for Linux, a vulnerability scanner, and a compliance reporting tool. Each has its own API, its own authentication, and its own idea of what a device is, which is why a question like "which devices are running an outdated version of Chrome" turns into three exports and a spreadsheet.

Fleet collapses that into one place: one server managing macOS, Windows, Linux, iOS, iPadOS, and Android, with one REST API that returns the same shape of data regardless of operating system.

### The devices nobody renewed for

Linux is where tool sprawl gets uncomfortable. A meaningful share of engineers at technology companies run desktop Linux, and they tend to hold the most elevated access and the most sensitive data. Jamf does not manage Linux at all, and Intune's Linux support is narrow. So the highest-risk hardware in the building is often the least managed, and the renewal quote does not mention it. Fleet manages Linux as a first-class platform: inventory, vulnerabilities, script execution, remote lock and wipe, and CIS benchmarks, with the same API and dashboard as everything else. The same gap shows up with contractor and personal devices, where Fleet's BYOD enrollment gives you a defensible amount of visibility without taking over someone's phone.

## 3. What happens after you sign?

This is the question renewal conversations skip, and it is the one that determines whether the next three years feel like a partnership or a waiting room.

### Configuration as code, not a console click

In a console-driven MDM, a change goes live the moment somebody clicks save. There is no diff, no review, and no revert, which is why a Friday afternoon Wi-Fi profile can turn into a weekend. In Fleet, configuration profiles, policies, reports, and software live in your Git repository. A change arrives as a pull request, a teammate reads the diff, and Fleet applies it after approval. If it goes wrong, one revert rolls it back across the fleet.

That workflow is also the only responsible way to bring AI into device management. An assistant can draft a policy, scope it with the labels your repository already uses, and open a pull request. A human still reviews and merges. If your source of truth is a console, there is no diff to review and nowhere for the change to land, so the human-in-the-loop story does not exist. See [why AI-powered device management requires GitOps](https://fleetdm.com/articles/why-ai-powered-device-management-requires-gitops) for the longer argument.

### Where your bug reports and feature requests actually go

Ask your incumbent to show you the status of a bug you reported last quarter. For most MDMs, the honest answer is a ticket number and a support portal that does not say much. Fleet's bugs and feature requests live as issues in public GitHub repositories, tracked on boards you can look at, and the product team publishes a [quarterly roadmap preview](https://fleetdm.com/articles/roadmap-preview-july-2026) so you know what is being built before it ships. When something is broken enough to need an engineer, you are talking to the person who can fix it, not a tier-one queue reading from a script.

That transparency extends to diagnosing rollout failures. Most large deployments stall short of complete, and the remaining devices are disproportionately the ones belonging to executives, legal, and finance. If the only answer your MDM gives you is a red "failed" badge, you are stuck re-pushing to the whole fleet and opening a ticket. Fleet reports the actual exit code, script output, and raw MDM response for each device that did not take, directly in the UI, which is usually enough to fix the specific machines yourself.

### Leaving is a smaller decision when you have not lost anything

None of this requires self-hosting to matter, and most teams never will self-host. What matters is that the option exists: Fleet's core is open source, so your security team can read what runs on your devices, and self-hosting is there for the subset of teams with a real data-residency requirement. More practically, your configuration lives in your Git repository as YAML you own, not inside a vendor's proprietary format. That is what actually keeps a renewal negotiation honest. A vendor relationship you could leave is a different kind of relationship than one you cannot.

## The objection is migration, and it is smaller than it was

Every one of these questions runs into the same wall: switching sounds like a quarter of work you do not have.

It is worth separating visibility from management. You can deploy Fleet's agent through the MDM you already have, including Jamf, Intune, or Workspace ONE, and have every device reporting inventory, software, vulnerabilities, and policy status within minutes. Nothing is taken over and no user is disrupted. That alone answers most of what your CISO and your auditors ask for, and it is also how teams handle an acquisition whose devices they cannot see yet.

Moving management over comes later, on your schedule. For Apple devices, Fleet follows Apple's supported Automated Device Enrollment flow, and Fleet's [managed migration assistant](https://fleetdm.com/articles/managed-migration-assistant) handles the user-facing part of a Mac-to-Mac move. Exported profiles can be reapplied, smart groups become labels, and extension attributes become queries or policies. For Windows, Fleet's CSP converter turns exported Intune policies into Fleet configuration. The [macOS MDM migration guide](https://fleetdm.com/guides/mdm-migration) walks through the sequence, and [migrating to GitOps using fleetctl](https://fleetdm.com/guides/migrating-to-gitops-using-fleetctl) covers getting your existing configuration into a repository.

## Before you sign

You do not have to switch tools to get value out of a renewal. You do have to ask the questions while somebody has a reason to answer them: show me what you can tell me about AI tooling on my endpoints, show me how you prioritize vulnerabilities, show me the status of my open feature requests, and show me where Linux fits.

If those answers are good, sign with confidence. If they are not, you have just learned what you are paying for, and you have time to do something about it.

## See it live

- [Compare Fleet with your current MDM](https://fleetdm.com/pricing)
- **Get a demo** with the team who would run your migration: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Join a GitOps workshop** to see configuration as code against a real repository: [fleetdm.com/workshops](https://fleetdm.com/workshops)

---
*Ready to ask harder questions before your next renewal? [Talk to Fleet](https://fleetdm.com/contact), or read [9 problems every IT team has, and why they switch to Fleet](https://fleetdm.com/articles/nine-problems-every-it-team-has-and-why-they-switch-to-fleet).*

<meta name="articleTitle" value="Three questions to answer before you renew your MDM">
<meta name="authorFullName" value="Mitch Francese">
<meta name="authorGitHubUsername" value="tux234">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-08-20">
<meta name="description" value="Three questions to ask your MDM vendor at renewal: vulnerability prioritization, AI governance, tool sprawl, and how bugs and features get tracked.">
