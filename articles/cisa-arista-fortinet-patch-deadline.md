# What CISA's Arista and Fortinet patch deadlines mean for your IT team

*CISA just gave federal agencies days, not weeks, to patch two actively exploited flaws. The bottleneck isn't reading the advisory: it's knowing which devices are still exposed.*

## Key takeaways
- **CISA's deadlines are shrinking, not just multiplying.** The Arista fix carried a 3-day window under Binding Operational Directive (BOD) 26-04, a preview of how fast every future Known Exploited Vulnerabilities (KEV) addition may need to move.
- **Neither flaw sits on a typical endpoint.** VeloCloud Orchestrator and FortiOS are network appliances, not devices a fleet of laptops and servers would normally include. That's worth naming plainly instead of stretching a product claim to fit.
- **The real lesson is about the software that *is* in your fleet.** The same KEV additions happen constantly for browsers, VPN clients, and OS packages that live on the Mac, Windows, and Linux devices IT already manages.
- **Manual "which hosts still have the old version" scrambles don't scale to 72-hour windows.** A point-in-time scan is stale before the meeting that reviews it ends.
- **Fleet flags KEV-listed exposure automatically, host by host.** Every CVE Fleet detects on managed software already carries a live CISA "known exploited" flag, so there's no separate spreadsheet to build when CISA adds one.
- **Filtering to "what's exploited right now" turns a scramble into a query.** A live query or a saved policy answers "are we exposed" in minutes, and keeps answering it every time inventory refreshes.

<a purpose="cta-button" href="/software-management">See vulnerability visibility in Fleet</a>

On July 27, 2026, CISA added two vulnerabilities to its Known Exploited Vulnerabilities (KEV) catalog: an unauthenticated OS command injection in Arista's VeloCloud Orchestrator On-Prem (CVE-2026-16812, CVSS 10.0) and an information-disclosure flaw in Fortinet's FortiOS SSL-VPN (CVE-2025-68686, CVSS 5.3). Under BOD 26-04, federal agencies had until July 30 to patch the Arista flaw and until August 10 for the Fortinet one: a three-day window for the more severe bug.

That kind of deadline compresses a process that used to take weeks into a scramble measured in hours. It's worth being precise about what these two flaws actually touch, and what that means for the tools most IT and security teams already run.

## What's actually vulnerable

CVE-2026-16812 lets an attacker with nothing more than network access to a VeloCloud Orchestrator's web interface run arbitrary commands. No credentials are required. Arista shipped fixes in VCO 5.2.3.14, 6.1.3.4, 6.4.2.4, and 7.0.0.1 and later. VeloCloud Orchestrator Hosted and Dedicated deployments were already patched before the advisory, so only self-managed, on-prem VCO instances are exposed.

CVE-2025-68686 is narrower. It lets an attacker bypass a prior FortiOS patch for symlink-based persistence, but only after the device has already been compromised at the filesystem level through a separate vulnerability. Fortinet's fix is a version upgrade, either 7.6.2 or later, or 7.4.7 or later, with 7.2, 7.0, and 6.4 needing migration to a supported release.

Both are appliances: a VeloCloud Orchestrator manages SD-WAN, and FortiOS runs on Fortinet's firewalls. Neither is the kind of general-purpose macOS, Windows, or Linux device that Fleet's agent would typically run on, and it wouldn't be honest to claim otherwise. If your organization runs either product, that patch cycle depends on Arista's and Fortinet's own management consoles, not Fleet.

## The part that does apply to your fleet

What's genuinely useful here isn't the specific CVEs, it's the pattern. CISA's KEV catalog adds entries like this constantly, and BOD 26-04 (like BOD 23-01 before it) exists because the agency has learned that "patch when convenient" doesn't hold up against active exploitation. Past KEV additions have landed on software that IT teams actually run fleet-wide: Log4Shell in 2021, and a steady stream of actively exploited Chrome and Windows CVEs since. The next one just as easily lands on a browser, a VPN client, or an npm package installed across thousands of the Mac, Windows, and Linux devices your team already manages with Fleet.

That's where a three-day deadline becomes solvable instead of a fire drill. Fleet's agent inventories installed software (apps, OS versions, browser plugins, and packages) across every managed host and cross-references it against multiple vulnerability sources, including the National Vulnerability Database and, for Fleet Premium customers, CISA's own KEV catalog. Every CVE Fleet surfaces already carries a `cisa_known_exploit` flag. When CISA adds a new entry, there's no tracking sheet to build from scratch. Filter your existing software inventory to `exploit: true` and you get the exact list of hosts still running the vulnerable version, right now.

## Query once, then keep querying

The difference between a scramble and a routine is repeatability. A one-time asset scan answers "are we exposed" for the moment you ran it. Save the same check as a Fleet policy instead, and it keeps running: patched devices drop off the failing list automatically, and newly enrolled or reimaged devices get checked the same way, with no one re-running a spreadsheet macro.

That's the actual shift BOD 26-04 is pushing federal agencies toward, and it's a reasonable one for any IT team to make on its own: stop treating each KEV addition as a one-off scramble, and instead keep a standing answer to "which of our devices are exposed right now" ready before CISA ever asks the question.

## See it live

Fleet Premium's [vulnerability processing](https://fleetdm.com/guides/vulnerability-processing) documents exactly what software is covered and how CVEs are matched, so you can check in advance whether it'll catch the next KEV addition on your fleet.

- **Get a demo** to see live exploited-vulnerability filtering on your own software inventory: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Browse [Fleet's policy library](https://fleetdm.com/policies)** for examples of standing checks you can adapt for KEV-driven patch compliance

---

*Ready to stop rebuilding your exposure list every time CISA updates the KEV catalog? [Get a demo](https://fleetdm.com/contact) or read how Fleet's [vulnerability processing](https://fleetdm.com/guides/vulnerability-processing) works.*

<meta name="category" value="articles">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="publishedOn" value="2026-07-30">
<meta name="articleTitle" value="What CISA's Arista and Fortinet patch deadlines mean for your IT team">
<meta name="description" value="CISA gave federal agencies days to patch actively exploited Arista and Fortinet flaws. Here's what that deadline pattern means for the software Fleet already tracks on your fleet.">
