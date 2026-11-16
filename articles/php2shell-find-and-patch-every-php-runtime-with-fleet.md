# PHP2Shell: find and patch every PHP runtime with Fleet

*The bugs are in the PHP runtime, not in your application. The hard part is knowing which hosts run PHP, which packages carry the fix, and which workers still hold the old code.*

## Key takeaways

- **Application security doesn't help here.** PHP2Shell lives in how PHP parses a request and cleans up after a crash, before any application code runs. A login page, a WAF rule, or a hardened CMS doesn't change whether the runtime underneath is exploitable.
- **Version numbers lie on Linux.** Ubuntu's fixed package still reports PHP 8.3.6, and RHEL 9's still reports 8.0.30. Only the full package version proves the fix, so Fleet policies compare it against each vendor's published build.
- **Patched on disk isn't patched in memory.** PHP-FPM pools and Apache workers keep running the code they started with. Fleet's reports flag workers whose binary changed after they started, and the scripts restart them.
- **PHP hides in more places than the package manager.** Containers, FrankenPHP binaries, Homebrew on Macs, zip files on Windows, and vendor bundles like XAMPP and MAMP each need their own fix. One report per OS lists all of them.
- **A failing policy can fix itself.** Attach a script to the policy and each failing host updates, restarts its workers, and turns green on its next check-in. Hosts on an end-of-life branch stay red and say why.
- **Exploitation looks like a web server spawning a shell.** The same reports surface a shell or tool whose parent is a PHP worker, and the logs to search for the request patterns the exploits use.

<a purpose="cta-button" href="https://fleetdm.com/software-management">See software management in Fleet</a>

On October 21, 2026, the PHP project shipped an emergency release for two critical bugs that security researchers nicknamed PHP2Shell. The details had leaked the day before, proof-of-concept code was on GitHub within hours, and Amazon later found an exploitation attempt against its honeypots dated October 17, four days before anyone outside the embargo was supposed to know.

The scramble that followed was mostly a scramble for inventory. Which servers run PHP? Which of those serve web requests? Is the Ubuntu package that still says 8.3.6 fixed or not? Did anyone restart php-fpm? Those are host questions, and this piece is about answering them with Fleet.

## What PHP2Shell is

PHP2Shell is two CVEs in the PHP runtime, both rated 9.2 under CVSS 4.0 by the PHP project and both [published](https://www.php.net/archive/2026.php#2026-10-21-5) on October 21.

[CVE-2026-109618](https://github.com/php/php-src/security/advisories/GHSA-w8hw-w9fx-j79w) is about out-of-memory handling. When a request runs PHP out of memory, PHP uses `longjmp()` to abandon the request. In some code paths that leaves request memory half-initialized, and the cleanup that follows trusts it anyway. An attacker can trigger the out-of-memory condition on purpose with multipart bodies: requests without a usable content length, parts with unlimited headers, empty file controls that don't count toward `max_file_uploads`, and deeply nested field names. PHP confirmed remote code execution against Apache with mod_php using default limits and a normal prefork configuration.

[CVE-2026-109619](https://github.com/php/php-src/security/advisories/GHSA-6wmf-4rv5-p3rp) is about stale state. PHP keeps global pointers into request memory, and a fatal error can skip the reset that normally clears them. The next request handled by the same worker then reads memory that was freed. One confirmed path runs through session upload progress, a feature that has been on by default since PHP 5.4. Another runs through fatal error backtraces, which PHP 8.5 turned on by default.

Both give an unauthenticated attacker code execution as the web server user, and watchTowr [reported](https://watchtowr.com/intelligence/php-remote-code-execution-php2shell/) exploitation in the wild on the day of the release. PHP says every installation that serves web requests is in scope: PHP-FPM, Apache mod_php, php-cgi, FrankenPHP, and LiteSpeed. Mod_php carries the highest risk, because the attack succeeds during request startup, before the application exists. The fixes are PHP 8.5.12, 8.4.27, 8.3.36, and 8.2.35. PHP 8.1 and older got [source patches](https://www.php.net/archive/2026.php#2026-10-21-5) but no release, which leaves those branches to distro and LTS vendors.

## Why "update PHP" is harder than it sounds

The PHP project's advice is one line: update, and restart every worker. Doing that across a fleet runs into four problems.

The first is inventory. PHP arrives on a host through the distro package manager, through third-party repositories like packages.sury.org and Remi, through control panels like cPanel and CloudLinux, through Docker images, as a FrankenPHP binary, through Homebrew on a Mac, and as a zip file on Windows. A host might have two of those at once, with one serving traffic and the other installed years ago for a script nobody remembers.

The second is version numbers. Linux vendors backport fixes without changing the upstream version. Ubuntu 24.04's fixed package is `8.3.6-0ubuntu0.24.04.13`, and RHEL 9's default PHP is `8.0.30-8.el9_8.1`. AlmaLinux rebuilt its streams with a `.alma.1` suffix. A check that reads the upstream version alone calls every one of those hosts vulnerable forever, or worse, calls them fixed before they are.

The third is memory. A PHP-FPM pool that started last Tuesday keeps running last Tuesday's code after the package updates. Apache with mod_php holds the old library until a full restart, and a graceful reload isn't enough. The package database says the host is fixed. The process serving requests isn't.

The fourth is end of life. The hosts most likely to run PHP 7.4 or 8.1 are the ones nobody updates on purpose. Some get a vendor backport. Amazon Linux 2023 said no fix is planned for `php8.1`. Those hosts need a mitigation, a migration, or a decision to take them offline.

## Find every PHP runtime with one report per OS

Fleet's agent already reports installed packages, running processes, loaded libraries, and containers. The [PHP2Shell reports](https://github.com/fleetdm/fleet/blob/main/docs/solutions/all/reports/php2shell.reports.yml) in Fleet's solutions library put those together into one row per finding, with a `status` column that says what to do.

On Linux, the `php_package` rows list every PHP package from the deb and rpm databases, including Sury, Remi, LiteSpeed, cPanel, and CloudLinux packages, and compare the full version against the vendor's fixed build for that distro and branch. The result is `FIXED`, `UPDATE to <version>`, or `NO_VENDOR_FIX_KNOWN` when the branch has no published fix. The `php_worker` rows list running PHP-FPM, php-cgi, LiteSpeed, and FrankenPHP processes, and the `mod_php` row shows Apache with `libphp` loaded. Both read `STALE` when the binary on disk changed after the process started. The `docker_container` rows list running containers whose image looks like PHP, WordPress, Drupal, or similar, because a package update on the host doesn't reach them.

On macOS, the report checks Homebrew's `php` and `php@X.Y` formulas against the fixed releases and lists MAMP's bundled runtimes. On Windows, it finds `php.exe` in the common install locations and in running processes, reads each build's file version, and lists XAMPP.

Every report also returns `web_worker_child` rows, which are covered below, and a `host` row with the OS version.

## Policies that know each distro's fixed version

A report tells you where you stand today. A policy keeps checking. The [PHP2Shell policies](https://github.com/fleetdm/fleet/blob/main/docs/solutions/all/policies/php2shell.policies.yml) pass when the host has no PHP, or when every PHP it has is at or above the fix.

The Linux policy carries a table of fixed builds for Ubuntu, Debian, RHEL, AlmaLinux, Rocky, Amazon Linux, Fedora, SUSE, and the common third-party repositories, keyed on the distro, the release, and the PHP branch. It compares full package versions with the agent's built-in version comparison, after normalizing the `+` and `_` separators that distro versions use, so `8.3.6-0ubuntu0.24.04.12` fails and `8.3.6-0ubuntu0.24.04.13` passes. A branch with no known fix fails on purpose, with a resolution that says so.

The macOS policy checks Homebrew formulas, and the Windows policy checks `php.exe` file versions. A fourth Linux policy verifies that the interim mitigation is in place, for hosts that can't update yet.

Each policy's version table reflects what vendors had published on October 23, 2026. If a vendor ships a later fix for a branch that shows no fix today, update the table and the policy picks it up on the next run.

## Patch, restart, re-check

Each policy has a matching script, so a failing host can fix itself through Fleet's [run-script automation](https://fleetdm.com/guides/policy-automation-run-script).

| Script | What it does |
|---|---|
| [`remediate-php2shell-linux.sh`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/linux/scripts/remediate-php2shell-linux.sh) | Upgrades every installed PHP package with apt, dnf, yum, zypper, or apk. Prints the full package versions and whether the changelog names the CVEs. Restarts PHP-FPM pools, and Apache when `libphp` is loaded. Warns about workers that still map a replaced binary, and about containers, FrankenPHP, and unpackaged PHP it can't update. |
| [`mitigate-php2shell-linux.sh`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/linux/scripts/mitigate-php2shell-linux.sh) | Writes PHP's interim mitigation (`session.upload_progress.enabled = 0`, and `fatal_error_backtraces = 0` for 8.5) as a drop-in into every PHP configuration directory, then restarts the same workers. For hosts waiting on a vendor fix. |
| [`remediate-php2shell-macos.sh`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/macos/scripts/remediate-php2shell-macos.sh) | Runs `brew update` and `brew upgrade` for each PHP formula as the Homebrew owner, restarts `php-fpm` and `httpd` services that launchd manages, and warns about MAMP and containers. |
| [`mitigate-php2shell-windows.ps1`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/windows/scripts/mitigate-php2shell-windows.ps1) | Reports every PHP build, including the ones IIS and Apache point at, without running them. Applies the interim mitigation, with a backup, to every ini file PHP can load for those builds, and restarts IIS and Apache services when their workers predate the change. Replacing a Windows build is a manual step. |

Each script is idempotent and exits `0` when the host is fixed, `2` when something needs a look, and `1` on error. The restart is the part people skip, and it's the part that matters: the Linux script checks `/proc/<pid>/exe` and the memory maps of every PHP worker and Apache process afterward, and warns if any still runs a replaced binary.

The interim mitigation deserves a caveat. PHP published it for hosts that can't update, and PHP says it's partial. Disabling session upload progress closes three of the five known techniques, and PHP suggests leaving it off even after you update, if your applications don't need it. The advisory's other suggestion, `memory_limit = -1`, removes the predictable limit the out-of-memory exploit relies on, but also removes PHP's defense against runaway scripts, and needs a cgroup limit alongside it. The scripts don't apply that one. It's a per-host decision.

## Hunt for signs of exploitation

Every public analysis of PHP2Shell lands on the same detection: a shell, an interpreter, or a download tool whose parent process is a PHP worker. Datadog's [write-up](https://securitylabs.datadoghq.com/articles/php2shell-critical-vulnerabilities-in-php/) shows `apache2` running as `www-data` spawning `sh -c` and then `bash` with an outbound connection. That's what successful exploitation looks like on the host, regardless of which technique got there.

The `web_worker_child` rows in the reports catch exactly that: a process named `sh`, `bash`, `python3`, `curl`, `nc`, `whoami`, and the like, whose parent is `php-fpm`, `apache2`, `httpd`, `php-cgi`, or `lsphp`. On Windows, the parent list is `w3wp.exe`, `httpd.exe`, and `php-cgi.exe`, and the children are `cmd.exe`, `powershell.exe`, `certutil.exe`, and friends. The `php_in_tmp` rows list `.php` files in `/tmp`, `/var/tmp`, and `/dev/shm`. Both are point-in-time views. For history, turn on process auditing and log `process_events` to your SIEM.

In the logs, the request side is distinctive. Amazon's honeypots saw multipart `POST` requests carrying `PHP_SESSION_UPLOAD_PROGRESS`. PHP's own advisory describes multipart bodies with no usable content length, many parts with empty filenames, and nested field names. On the response side, the memory-disclosure variant leaks worker memory into the `PHPSESSID` cookie, so unusually long or percent-encoded session cookies are a signal. In the PHP log, `Allowed memory size exhausted` at `Unknown on line 0` means a request died during startup, before any script ran. Start the lookback on October 17, not October 21.

Clean logs don't prove a host is clean. PHP's advisory says so directly, and Datadog's test exploitation returned HTTP 200 without crashing the worker.

## What Fleet's vulnerability detection will and won't show

Fleet's built-in vulnerability detection matches installed software against NVD, Canonical's and Red Hat's OVAL definitions, and Amazon's ALAS. NVD now carries CPE ranges for `php:php` on both CVEs, which covers Homebrew PHP on macOS. Ubuntu, RHEL, and Amazon Linux hosts show the CVEs once those feeds include the advisories, and Fleet refreshes the feeds daily.

Three gaps are worth stating plainly. Debian, SUSE, and third-party repository packages aren't matched. Derived distros like AlmaLinux are evaluated with Red Hat's definitions, so a patched Alma backport with a lower version number can still show as vulnerable. And no feed sees a PHP inside a container or a FrankenPHP binary. That's why the reports and policies here compare versions themselves. For this one, they're the source of truth, and the vulnerability page is the second opinion.

## The cost of a slow inventory

Qualys [measured](https://blog.qualys.com/vulnerabilities-threat-research/2026/10/20/php-announces-critical-security-release-for-two-vulnerabilities) patch coverage twelve hours after the release and found only a fraction of the installations it tracks had updated. The rest weren't negligent. Most didn't know yet which of their hosts served PHP, or trusted a version number that couldn't tell them. The exploitation that Amazon dated to October 17 means the window opened before the patch existed, and every day a worker keeps the old code open, it stays open.

Fleet can't make a vendor ship a backport faster. What it can do is answer the host questions in minutes instead of days: which hosts run PHP, from where, which packages are at the vendor's fixed build, which workers still hold old code, and which ones have a shell hanging off a web worker right now. The rest is a policy and a script.

## See it live

- [Find and fix PHP2Shell exposure with Fleet](https://fleetdm.com/guides/find-and-fix-php2shell-exposure-with-fleet) walks through the reports, policies, and scripts step by step.
- **Get a demo** of Fleet's software and vulnerability management at [fleetdm.com/contact](https://fleetdm.com/contact).

<meta name="articleTitle" value="PHP2Shell: find and patch every PHP runtime with Fleet">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="publishedOn" value="2026-10-23">
<meta name="category" value="articles">
<meta name="description" value="PHP2Shell is a runtime bug, so every PHP that serves web requests is exposed. How to find, patch, restart, and verify every host with Fleet.">
