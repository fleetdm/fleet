# GitLab's CVSS 10.0 regex bug needs no login, and the fix is a version number you have to verify

*GitLab shipped emergency builds on October 12 for a CI/CD flaw that lets an unauthenticated attacker run code on a self-managed server. The patch is simple. Knowing every instance took it is the hard part.*

## Key takeaways

- **A crafted regular expression is enough.** CVE-2026-97229 is a CVSS 10.0 integer overflow in GitLab's CI/CD regular expression compiler. GitLab says it could let an unauthenticated user run code on the server through a specially crafted regular expression in a CI/CD configuration.
- **The remediation is a version, not a setting.** The fixed builds are 19.2.8, 19.3.5, and 19.4.2. Reporting on the advisory mentions no workaround, so upgrading is the only path.
- **Self-managed instances are the exposed ones.** GitLab.com is already patched. Every self-managed server you run, including the one a team stood up years ago, is yours to update.
- **A separate AI Gateway flaw has a narrower audience.** CVE-2026-90970 (CVSS 9.9) was disclosed on October 2 and needs an authenticated user with Duo Agent Platform access. It only affects organizations that host their own AI Gateway.
- **An advisory reaching you is not an update reaching the server.** Telling the difference takes version data from the hosts themselves, not a ticket that says "done."
- **Find the stragglers before an attacker does.** Last month a CVSS 10.0 GitLab file-read flaw saw in-the-wild probes within hours of disclosure, so the gap between "patched" and "believed patched" is the risk.

<a purpose="cta-button" href="https://fleetdm.com/software-catalog">See software inventory in Fleet</a>

GitLab announced patch releases on Monday, October 12, for GitLab Community Edition and Enterprise Edition. The headline fix is CVE-2026-97229, a CVSS 10.0 integer overflow that [The Hacker News](https://thehackernews.com/2026/10/gitlab-patches-cvss-100-flaw-that-could.html) reports GitLab attributes to insufficient regular expression validation.

For a team that runs its own GitLab, the advisory is the easy part. The question that follows is whether every instance, including the forgotten ones, is now on 19.2.8, 19.3.5, or 19.4.2.

## What the October 12 releases fix

The affected ranges are 19.2 before 19.2.8, 19.3 before 19.3.5, and 19.4 before 19.4.2. The release carries several other fixes, including a cross-site scripting bug (CVE-2026-92883, CVSS 8.7) and an improper authorization bug in downstream pipeline creation (CVE-2026-16301, CVSS 8.5). Both require an authenticated user.

The reporting I found doesn't say whether CVE-2026-97229 is being exploited. It also lists no workaround. Treat the fixed version as the whole answer, and note that secondary sources disagree on whether the 19.3 fix is 19.3.4 or 19.3.5. Use GitLab's own advisory as the final word.

## The AI Gateway flaw is a different problem

CVE-2026-90970 is a CVSS 9.9 sandbox escape in the self-hosted GitLab AI Gateway. [The Hacker News](https://thehackernews.com/2026/10/gitlab-patches-critical-self-hosted-ai.html) reports it was disclosed on October 2 and requires a logged-in user with Duo Agent Platform access, so it is not an unauthenticated bug. The fixed gateway versions are 19.2.4, 19.3.2, and 19.4.1.

GitLab says customers on GitLab.com, GitLab Dedicated, and self-managed instances that use a GitLab-hosted gateway don't need to act. If you run your own gateway, the advisory offers no workaround and no way to check whether it was attacked before the patch.

## Why "we sent the advisory" doesn't close this

Self-hosted GitLab tends to spread. A production instance is on the upgrade calendar, while a staging copy, a lab server, or an acquired team's instance may not be. The release also includes database migrations, so single-node instances are down until they finish, which is exactly when a maintenance window gets postponed and a server nobody revisits stays behind.

That is the gap to close. The question for each host is not "did we get the notice" but "what version is it running right now."

## Confirming the version with Fleet

**Start with software inventory.** For GitLab installed from Linux packages, Fleet's agent collects deb and rpm package names and versions, so a GitLab install can show up in the **Software** tab next to everything else. The [software inventory reference](https://fleetdm.com/guides/software-inventory-reference) lists what Fleet collects on each platform.

**Use a report for the rest.** Containers aren't part of software inventory, so a GitLab running in Docker needs a different check, such as a [report](https://fleetdm.com/guides/queries) against the hosts that run it. For package installs, a report against the package tables is a quick way to list every host with a `gitlab-ce` or `gitlab-ee` package and its version. Test any check on a host you know runs GitLab first. A result of zero rows only reassures you if the check has found a known install.

Once you have the list, compare each version to the fixed builds. Anything below 19.2.8, 19.3.5, or 19.4.2 on its line is still exposed. Anything on a line older than 19.2 isn't covered by this advisory's fixed versions, so check GitLab's guidance for those instances.

For the AI Gateway, check how it is deployed. If it runs as a container rather than a package, Fleet's software inventory may not see it, and you will need another source of truth for its version.

## Where this leaves you

Upgrade the self-managed instances first, then verify each one from the host. If you run your own AI Gateway, update it to the fixed version for your line as well. The patch is a version number, and a version number is something a host can report back to you.

## See it live

- **Get a demo** at [fleetdm.com/contact](https://fleetdm.com/contact) to see software inventory across Linux, macOS, and Windows hosts.

## Sources

- The Hacker News, [GitLab patches CVSS 10.0 flaw that could let unauthenticated users run server code](https://thehackernews.com/2026/10/gitlab-patches-cvss-100-flaw-that-could.html).
- The Hacker News, [GitLab patches critical 9.9 AI Gateway flaw allowing command execution on self-hosted servers](https://thehackernews.com/2026/10/gitlab-patches-critical-self-hosted-ai.html).

<meta name="articleTitle" value="GitLab's CVSS 10.0 regex bug needs no login, and the fix is a version number you have to verify">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-10-13">
<meta name="description" value="GitLab's CVSS 10.0 CI/CD flaw needs no login. Fixed builds are 19.2.8, 19.3.5, and 19.4.2. Here's how to confirm every instance took them.">
