# PHP 8.3.6 on Ubuntu 24.04 can be patched for PHP2Shell, and the version number won't tell you

*PHP's fixed releases are 8.3.36 and higher, yet Ubuntu 24.04 fixes the bugs in a package that still says 8.3.6. Here's how to read the build suffix and prove which hosts have it.*

## Key takeaways

- **A version check will call your patched Ubuntu hosts vulnerable.** PHP's fixed releases for the 8.3 line start at 8.3.36, but Ubuntu 24.04 backports the fix into a package that still reports 8.3.6. Comparing the upstream number alone produces false alarms, and it can also hide real gaps.
- **The proof is the full package version.** On Ubuntu 24.04, `8.3.6-0ubuntu0.24.04.13` is the build Ubuntu lists as fixed. Any lower revision on that release is still exposed.
- **Each Ubuntu release has its own fixed build.** 22.04, 24.04, 26.04, and 26.10 each have a different package and revision, and releases through 20.04 only get the fix through Ubuntu Pro.
- **Exploitation started before disclosure.** Amazon's honeypot recorded an attempt on October 17, four days before the public announcement, so "we patch on Tuesday" was already late.
- **Package inventory covers part of the problem.** Fleet's agent reports installed deb packages with their full versions, but PHP built from source or running in a container won't appear there and needs a different check.
- **A host report beats an advisory.** A list of hosts and revisions from the hosts themselves is evidence. A line in a change ticket is a claim.

<a purpose="cta-button" href="https://fleetdm.com/software-catalog">See software inventory in Fleet</a>

PHP2Shell is the name for two critical flaws in PHP itself, CVE-2026-109618 and CVE-2026-109619, disclosed on October 21. [Help Net Security](https://www.helpnetsecurity.com/2026/10/21/php2shell-php-vulnerabilities-patched-cve-2026-109618-cve-2026-109619/) reports that both allow unauthenticated code execution on a web server, and that a working exploit needs a single request before any application code runs.

The fixed upstream releases are PHP 8.5.12, 8.4.27, 8.3.36, and 8.2.35. If you run Ubuntu, that list is a trap. Ubuntu doesn't ship those numbers, and a scanner that compares them to what's installed can get the answer backwards.

## Why 8.3.6 can be the fixed version

Ubuntu LTS releases freeze the upstream version at release time and backport security fixes into it. The package name and version carry the proof: the upstream part stays the same, and the Ubuntu revision at the end changes.

[Ubuntu's page for CVE-2026-109618](https://ubuntu.com/security/CVE-2026-109618) lists these fixed builds:

| Release | Package | Fixed in |
| --- | --- | --- |
| 22.04 LTS | `php8.1` | `8.1.2-1ubuntu2.28` |
| 24.04 LTS | `php8.3` | `8.3.6-0ubuntu0.24.04.13` |
| 26.04 LTS | `php8.5` | `8.5.4-0ubuntu1.5` |
| 26.10 | `php8.5` | `8.5.9-0ubuntu2.1` |

Notice that none of these upstream numbers match the fixed releases. A 24.04 host running `8.3.6-0ubuntu0.24.04.13` is fixed. A host running `8.3.6-0ubuntu0.24.04.10` has the same upstream number and is not.

Older releases are different. Ubuntu's page lists fixes for 14.04 through 20.04 only as Ubuntu Pro (ESM) builds, such as `7.4.3-4ubuntu2.29+esm5` for 20.04. A 20.04 host without Ubuntu Pro has no fixed package to install, so for those hosts the question is whether the machine should still be serving PHP at all. Ubuntu lists USN-8948-1 and USN-8948-2 as the related notices, both dated October 21.

## Who is exposed and how fast

Help Net Security says every PHP installation serving web requests is affected, including PHP-FPM, Apache with mod_php, php-cgi, FrankenPHP, and LiteSpeed. It says mod_php setups face the greatest danger.

Timing matters too. According to Amazon Threat Intelligence, as reported in that same article, its MadPot honeypot recorded an exploitation attempt on October 17, while the flaws were still under embargo. Amazon doesn't know how the attacker got the details. The UK NCSC advises taking offline any system you can't fix right away.

So the real list is every host that serves PHP, with the exact package revision next to it.

## Getting the revision from each host

**Start with software inventory.** Fleet's agent collects deb packages on Ubuntu hosts, with name and version, so `php8.3` and its full version string appear in the **Software** tab. The [software inventory reference](https://fleetdm.com/guides/software-inventory-reference) lists what Fleet collects on each platform.

**Then run a report for a fleet-wide list.** A [report](https://fleetdm.com/guides/queries) can return the PHP packages on every Linux host in one result:

```sql
SELECT name, version
FROM deb_packages
WHERE name LIKE 'php%' OR name LIKE 'libapache2-mod-php%';
```

Compare each row against the table above for that host's release. Don't trust the query until it has returned a result on a host you know runs PHP. Also check the Apache module package separately: this article didn't verify whether Ubuntu versions it identically to the `php8.3` package.

**Know what package inventory misses.** PHP compiled from source, installed from a third-party repository, or running in a container won't show up as a deb package on the host. The Remi RPM repository, which Help Net Security says had no updates available on October 21, is one example of a third-party source. Those cases need their own check, such as looking for the `php` binary and running it with `-v`, or auditing your container images.

## What the ticket can't prove

A change ticket that says "PHP updated" records an intention. The package version on the host is the outcome. With a backported fix, the difference is easy to miss, because the line in the ticket says 8.3.6 both before and after.

## Where this leaves you

For Ubuntu hosts, the work is to read the full package version and compare it to the fixed build for that release. For unsupported releases and anything built outside the package manager, the work is finding it first. Either way, the starting point is a list of hosts and revisions that came from the hosts themselves.

## See it live

- **Talk to Fleet:** [Not sure which hosts serve PHP in your environment? Get a demo.](https://fleetdm.com/contact)
- **Read the argument for ground truth:** [fleetdm.com/articles/agentic-security-needs-ground-truth](https://fleetdm.com/articles/agentic-security-needs-ground-truth)

## Sources

- Help Net Security, [PHP2Shell: Critical PHP vulnerabilities patched, update now!](https://www.helpnetsecurity.com/2026/10/21/php2shell-php-vulnerabilities-patched-cve-2026-109618-cve-2026-109619/).
- Ubuntu, [CVE-2026-109618](https://ubuntu.com/security/CVE-2026-109618).

<meta name="articleTitle" value="PHP 8.3.6 on Ubuntu 24.04 can be patched for PHP2Shell, and the version number won't tell you">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-10-22">
<meta name="description" value="PHP2Shell's fixed releases are 8.3.36 and up, but Ubuntu 24.04 patches 8.3.6 in place. Read the package revision to see which hosts are fixed.">
