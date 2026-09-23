<meta name="category" value="industry news">
<meta name="articleTitle" value="Ubuntu fixed 10 OpenSSH bugs and a sudo bypass in one day, but four LTS releases only get it through Ubuntu Pro">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="publishedOn" value="2026-09-23">
<meta name="description" value="Ubuntu fixed a sudo policy bypass and 10 OpenSSH bugs in one day. See which Ubuntu hosts got the fix, and which need Ubuntu Pro.">

# Ubuntu fixed 10 OpenSSH bugs and a sudo bypass in one day, but four LTS releases only get it through Ubuntu Pro

*Two Ubuntu security notices landed hours apart on September 22: one closing ten OpenSSH flaws, the other a sudo bug that let commands slip past policy checks and logging. Here's how to confirm which of your hosts got both, and why four LTS releases won't get there through a routine apt upgrade.*

## Key takeaways

- **You can confirm patch status per host instead of trusting apt caught it.** Fleet's software inventory reports the exact `openssh-client`, `openssh-server`, and `sudo` package versions installed on every Ubuntu host, so you're checking real data instead of assuming a changelog applied itself.
- **The sudo bug is a logging bypass, not a permissions slip.** [CVE-2026-82474](https://ubuntu.com/security/CVE-2026-82474) let a local user who's already allowed to run specific commands skip sudo's intercept policy checks, meaning the commands they ran wouldn't show up in policy enforcement or the audit log either.
- **The OpenSSH batch is ten CVEs, but only for releases past standard support.** [USN-8804-1](https://ubuntu.com/security/notices/USN-8804-1) covers 14.04, 16.04, 18.04, and 20.04 LTS, four releases that exited standard Ubuntu support years ago.
- **Those four releases don't get the fix through a normal apt upgrade.** Every fixed OpenSSH package in USN-8804-1 carries a `+esm` version suffix, meaning it ships only through Ubuntu Pro's Extended Security Maintenance.
- **The sudo fix is the opposite case: a standard update on supported releases.** [USN-8803-1](https://ubuntu.com/security/notices/USN-8803-1) covers 24.04 and 26.04 LTS, both still inside normal support, so a routine `apt upgrade` picks it up without any subscription involved.
- **A saved policy keeps the answer current past this one patch day.** Ubuntu ships new OpenSSH and sudo notices on a regular cadence, and a Fleet policy re-checks installed versions without anyone re-running the query by hand next time.

<a purpose="cta-button" href="https://fleetdm.com/security-and-control">See vulnerability management in Fleet</a>

Ubuntu published two security notices within hours of each other on September 22, 2026. [USN-8804-1](https://ubuntu.com/security/notices/USN-8804-1) fixes ten OpenSSH vulnerabilities. [USN-8803-1](https://ubuntu.com/security/notices/USN-8803-1) fixes a single sudo bug that undermines command logging. Same day, same distribution, and two fixes that reach your fleet in completely different ways.

That difference is the part worth checking for, not the CVE count. One notice patches through a standard package update. The other only reaches affected hosts if they're covered by a paid support contract, which is easy to overlook if you're used to `unattended-upgrades` quietly handling everything.

## Two notices, two different bugs

[CVE-2026-82474](https://ubuntu.com/security/CVE-2026-82474) is the more unusual of the two. Ubuntu's advisory says sudo "failed to apply intercept policy checks when commands were executed under certain circumstances," which let a local attacker already permitted to run specific commands bypass policy enforcement and logging to execute unauthorized programs. The bug isn't a full privilege escalation. It's a way for an authorized user to make sudo stop watching, which matters most to teams that rely on sudo's audit trail to know what happened on a host after the fact.

The OpenSSH batch in USN-8804-1 is broader: ten CVEs spanning ssh-agent, the ssh client, sshd, and internal-sftp, covering issues from a GSSAPI authentication denial-of-service to a bypass of `authorized_keys` principal restrictions. One of the ten, [CVE-2026-73282](https://ubuntu.com/security/CVE-2026-73282), a use-after-free in how the ssh client handles concurrent remote-forwarding requests, was already patched for 22.04, 24.04, and 26.04 LTS three weeks earlier, on September 3, through [USN-8721-1](https://ubuntu.com/security/notices/USN-8721-1). September 22's notice backports that same fix, plus nine more, to the four older releases still catching up.

## The catch: four releases need Ubuntu Pro to get it

This is the detail that's easy to miss if you only skim the CVE count. Every fixed OpenSSH package in USN-8804-1 carries a `+esm` suffix:

| Release | Fixed openssh-client / openssh-server version |
|---|---|
| 20.04 LTS | `1:8.2p1-4ubuntu0.13+esm3` |
| 18.04 LTS | `1:7.6p1-4ubuntu0.7+esm6` |
| 16.04 LTS | `1:7.2p2-4ubuntu2.10+esm10` |
| 14.04 LTS | `1:6.6p1-2ubuntu2.13+esm4` |

The `+esm` suffix means the fix ships through Ubuntu Pro's Extended Security Maintenance, not the standard archive. A host on one of these releases without an active Ubuntu Pro subscription attached won't pull these updates through a routine `apt upgrade`, no matter how current its other packages look. Because all four releases are already past standard support, it's easy to assume they're either fully patched or fully abandoned. USN-8804-1 shows a third state: still receiving fixes, but only for hosts enrolled in ESM.

The sudo fix doesn't have that complication. [USN-8803-1](https://ubuntu.com/security/notices/USN-8803-1) covers 24.04 LTS (`sudo` `1.9.15p5-3ubuntu5.24.04.3`) and 26.04 LTS (`sudo` `1.9.17p2-1ubuntu3.1`), both current releases inside standard support, so it lands through the same `apt upgrade` that handles everything else.

## Confirming what's installed

Fleet's software inventory reports installed package versions the same way for OpenSSH and sudo as it does for anything else on a host, which means you can check real data instead of a support-status assumption:

```sql
SELECT name, version FROM deb_packages WHERE name IN ('openssh-client', 'openssh-server', 'sudo');
```

Match the returned version and the host's Ubuntu release against the fixed version for that release: the ESM builds above for 14.04 through 20.04, or the standard builds for 24.04 and 26.04. A host still showing a pre-fix version on one of the ESM releases is either missing an active Ubuntu Pro subscription or hasn't run its ESM-aware update yet, both of which are worth knowing before you assume the September 22 batch is handled.

## Turning the sweep into a policy

A one-time query answers where things stand today. Ubuntu ships OpenSSH and sudo notices often enough that the answer won't stay current for long, and the next one may again split cleanly between ESM-only and standard-support releases. A saved Fleet policy checks installed versions against the current fixed build for each release automatically, so the next notice doesn't require someone to remember to re-run the query. Because Fleet policies live in Git as YAML and deploy through the same GitOps workflow as the rest of your configuration, updating the version thresholds when the next notice lands is a reviewable pull request, not a console edit someone has to remember to make.

## A patch day is only as good as its coverage

Two notices, published hours apart, patching the same distribution, reaching hosts through two entirely different mechanisms. That split is exactly why "Ubuntu shipped a fix" and "my fleet has the fix" are different claims. Knowing which of your Ubuntu hosts are on ESM, which have an active Pro subscription, and which package versions they're running is what closes the gap between the two.

## See it live

- **Get a demo** to see software inventory and vulnerability matching against your own fleet: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Read the guide** on using Fleet policies for patch management: [fleetdm.com/guides/how-to-use-policies-for-patch-management-in-fleet](https://fleetdm.com/guides/how-to-use-policies-for-patch-management-in-fleet)

## Sources

- Ubuntu, [USN-8804-1: OpenSSH vulnerabilities](https://ubuntu.com/security/notices/USN-8804-1).
- Ubuntu, [USN-8803-1: Sudo vulnerability](https://ubuntu.com/security/notices/USN-8803-1).
- Ubuntu, [USN-8721-1: OpenSSH vulnerabilities](https://ubuntu.com/security/notices/USN-8721-1).
