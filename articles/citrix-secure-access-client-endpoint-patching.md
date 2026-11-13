# Two of Citrix's 12 November fixes belong on your laptops, not the gateway

*Citrix's latest bulletins cover NetScaler appliances and the Secure Access Client that runs on Windows and Linux devices. Upgrading the appliance doesn't touch the client, so someone has to confirm that part separately.*

## Key takeaways

- **The client is its own patch job.** Two of the 12 flaws are in Secure Access Client, the VPN software installed on Windows and Linux devices, so upgrading NetScaler appliances leaves them open.
- **A standard user can become an administrator.** Both client flaws let a low-privileged local user escalate, to SYSTEM on Windows and to root on Linux, which turns any foothold on a laptop into full control of it.
- **The fixed builds are specific.** Windows needs Secure Access Client 26.11.1.6 or later and Linux needs 26.10.4 or later, which gives you a clear line to measure each device against.
- **Software inventory answers the question directly.** Reading the installed client version from each device shows which hosts are still below the fixed build, instead of assuming the gateway upgrade covered them.
- **No known exploitation buys time, not safety.** Citrix says it isn't aware of exploits, which makes now the cheapest moment to close the gap on devices.

<a purpose="cta-button" href="https://fleetdm.com/software-management">See software inventory in Fleet</a>

Citrix published fixes for 12 vulnerabilities this week. Most of the attention is on NetScaler ADC and NetScaler Gateway, where nine memory overflow flaws are rated 9.3 or 9.5, and on NetScaler Console, where CVE-2026-120869 (CVSS 9.4) lets an unauthenticated attacker bypass authentication. Citrix says it isn't aware of exploitation for any of them.

Those are appliance problems, and the network team that owns the appliances will schedule upgrades. The two remaining flaws are different. They sit in Secure Access Client, the software on the Windows and Linux devices that connect through the gateway. Who confirms those got patched?

## What's in the client fixes

Citrix published two separate bulletins for Secure Access Client:

| CVE | Platform | CVSS | Impact | Fixed in |
|---|---|---|---|---|
| CVE-2026-118268 | Windows (CTX697280) | 8.5 | Low-privileged user escalates to SYSTEM | 26.11.1.6 and later |
| CVE-2026-118269 | Linux (CTX697281) | 8.5 | Low-privileged user gains root | 26.10.4 and later |

Both are improper privilege management flaws. Neither gives a remote attacker a way in by itself. What they do is remove the boundary between a standard user account and full control of the device. Phishing payloads, malicious browser extensions, and compromised developer tools often start with standard-user access, and a local privilege escalation is the step that turns that access into persistence, credential theft, and the ability to disable security tools.

## Why the gateway upgrade doesn't cover it

NetScaler upgrades happen on the appliance: 14.1-73.57 and later on the 14.1 branch, 13.1-64.39 and later on 13.1, with separate FIPS and NDcPP builds. When the appliance is done, the change ticket closes.

The client lives somewhere else. It's installed on each laptop and desktop, often pushed once during onboarding and updated whenever the VPN prompts the user or IT repackages it. Some organizations let the gateway offer client upgrades at connection time, others pin a version, and many have a mix. In every case, the version on a given device is a fact about that device, not about the gateway.

That's where gaps hide. A device that rarely connects, a Linux workstation someone set up by hand, or a contractor laptop that skipped an upgrade prompt can all keep running a vulnerable client long after the appliance work is complete.

## Check the installed version on each device

The fix is to measure the client the same way you'd measure a browser or an operating system: by reading the installed version from the device.

Fleet's agent collects software inventory on Windows and Linux hosts, including installed programs on Windows and deb and rpm packages on Linux. Search the **Software** page for Secure Access to see which versions are installed and how many hosts run each one. Any host below 26.11.1.6 on Windows or 26.10.4 on Linux needs the update.

To get the same answer as a report you can save and rerun, query the device directly. On Windows:

```sql
SELECT name, version, install_date
FROM programs
WHERE name LIKE '%Secure Access%';
```

On Linux, check the package manager your distributions use:

```sql
SELECT name, version FROM deb_packages WHERE name LIKE '%secure-access%'
UNION ALL
SELECT name, version FROM rpm_packages WHERE name LIKE '%secure-access%';
```

Display names and package names vary by version and packaging, so confirm the match on a known device before relying on either report. Neither query has been tested against the 26.x client builds.

## From finding to fixed

Once you know which hosts are behind, the work is the usual patch loop. Deploy the updated client through your existing software distribution, or upload the package to Fleet and install it on the hosts that need it. Then rerun the report or recheck the **Software** page until no host shows a version below the fixed build.

Fleet also matches installed software against published vulnerability data, so these CVEs may appear on affected hosts once the feeds include them. Don't wait on that. You already know the fixed versions, and a version check works today.

## The client is part of the attack surface

VPN appliances get patched because everyone knows they face the internet. The clients that connect to them are easier to forget, because they look like plumbing. This round of Citrix fixes is a reminder that the client is software running with elevated privileges on every device that has it. Track its version like any other app, and a gateway patch cycle won't leave half the job undone.

## See it live

- **Learn more:** Read how [Fleet processes vulnerabilities](https://fleetdm.com/guides/vulnerability-processing) across Windows, macOS, and Linux.
- **Get a demo:** [Talk to the Fleet team](https://fleetdm.com/contact) about tracking software versions across your devices.

## Sources

- Citrix security bulletins CTX697305 (NetScaler ADC and Gateway), CTX697320 (NetScaler Console), CTX697280 (Secure Access Client for Windows), and CTX697281 (Secure Access Client for Linux).
- The Hacker News, "Citrix Patches 12 NetScaler and Secure Access Client Flaws, Including Authentication Bypass."

<meta name="articleTitle" value="Two of Citrix's 12 November fixes belong on your laptops, not the gateway">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-11-13">
<meta name="description" value="Citrix fixed privilege escalation flaws in Secure Access Client for Windows and Linux. Check the client version on each device, not just the gateway.">
