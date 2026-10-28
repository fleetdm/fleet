# Find and fix PHP2Shell exposure with Fleet

On October 21, 2026, the PHP project [released](https://www.php.net/archive/2026.php#2026-10-21-5) PHP 8.5.12, 8.4.27, 8.3.36, and 8.2.35 to fix [CVE-2026-109618](https://github.com/php/php-src/security/advisories/GHSA-w8hw-w9fx-j79w) and [CVE-2026-109619](https://github.com/php/php-src/security/advisories/GHSA-6wmf-4rv5-p3rp), together called PHP2Shell. Both are memory-safety bugs in the PHP runtime itself. An unauthenticated attacker can send crafted HTTP requests to any PHP installation that serves web requests and run code as the web server user, before the application sees the request. PHP-FPM, Apache mod_php, php-cgi, FrankenPHP, and LiteSpeed are all affected, and PHP rates mod_php as the highest risk. Exploitation started [before the disclosure](https://aws.amazon.com/blogs/security/amazon-threat-intelligence-identifies-php2shell-exploitation-prior-to-public-disclosure-in-madpot/). This guide shows you how to find every PHP runtime your hosts run, confirm each one is updated, restart the workers that still hold old code, apply PHP's interim mitigation where you can't update yet, and look for signs of exploitation, with Fleet reports, policies, and scripts.

## Why `php -v` isn't enough

Three things make PHP2Shell harder to close than a normal package update:

- Linux vendors backport the fix without changing the upstream version. Ubuntu 24.04's fixed package is still `8.3.6`, and RHEL 9's default PHP is still `8.0.30`. Only the full package version proves the fix, so every check in this guide compares the whole version string against the vendor's fixed build.
- PHP 8.1 and older are end of life. The PHP project published [source patches](https://www.php.net/archive/2026.php#2026-10-21-5) but no releases, so hosts on 7.4, 8.0, or 8.1 depend on a distro or LTS vendor backport, or need a newer branch.
- A worker that started before the update keeps running the old code. PHP-FPM pools, Apache with mod_php, and php-cgi all need a restart, and for mod_php a reload isn't enough.

PHP also lives outside the package manager: in containers, in FrankenPHP binaries, in Homebrew on Macs, and in zip files on Windows. The reports in step 1 find those too.

## Prerequisites

- A Fleet account with permission to run live reports, add policies, and run scripts.
- [Scripts enabled](https://fleetdm.com/guides/scripts) on fleetd for the hosts you want to fix.
- The fixed package for your distro. Ubuntu, Debian, Red Hat, AlmaLinux, Rocky, Amazon Linux, Fedora, and SUSE published updates on October 21 and 22, 2026. For PHP 7.4, 8.0, and 8.1, check your vendor's advisory first.

> **Warning:** A failing policy with a run-script automation restarts PHP-FPM and Apache on every failing host. Run the scripts on a test host first, then attach them to the policies.

## Step 1: Find every PHP runtime

Add the three reports in [`php2shell.reports.yml`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/all/reports/php2shell.reports.yml), one per OS, or paste a report's SQL into a live report in the Fleet UI. Each returns one row per finding with the columns `finding`, `item`, `detail`, and `status`. No row for a finding means it wasn't found.

| Finding | What it means |
|---|---|
| `php_package` (Linux) | An installed PHP package and its full version. `FIXED`, `UPDATE to <version>`, or `NO_VENDOR_FIX_KNOWN` for a branch with no published fix. |
| `homebrew_php` (macOS), `php_build` (Windows) | A PHP formula or `php.exe` and whether its version is at or above the fixed release. |
| `php_worker`, `mod_php` | A running PHP-FPM, php-cgi, LiteSpeed, or FrankenPHP worker, or Apache with `libphp` loaded. `STALE` means the binary on disk changed after the process started, so it still runs the old code. |
| `docker_container` | A running container whose image name suggests PHP (`php`, `wordpress`, `drupal`, and others). Package updates on the host don't reach it. |
| `mitigation_dropin` (Linux), `mitigation_ini` (Windows) | The interim mitigation from step 5 is in place. |
| `web_worker_child`, `php_in_tmp` | A shell or tool whose parent is a PHP worker, or a `.php` file in a temp directory. See step 6. |
| `mamp_php`, `xampp` | A bundled runtime that the policies don't check. Update the bundle. |
| `host` | The OS version, for context. |

To run a saved report from the command line:

```bash
fleetctl report --report-name "Linux - PHP2Shell exposure" --labels "Linux"
```

> **Note:** The reports show what is on the host right now. The `php_worker` and `web_worker_child` rows are a point-in-time view, not a history. For history, turn on [process auditing](https://fleetdm.com/guides/osquery-evented-tables-overview) and log `process_events`.

## Step 2: Update PHP on Linux

Add this policy. It passes when the host has no PHP packages, or when every installed PHP package is at or above the vendor's fixed build for that distro and PHP branch. It fails for a branch with no known fix, such as PHP 8.1 on Amazon Linux 2023, the `php:8.4` stream on RHEL 9 (no fix as of October 23), or a PHP you built from source.

```yaml
- name: Linux - PHP packages patched for PHP2Shell (CVE-2026-109618, CVE-2026-109619)
  query: |
    WITH os AS (
      SELECT platform, platform_like, major, version FROM os_version LIMIT 1
    ),
    php AS (
      SELECT name, version AS pkg
      FROM deb_packages
      WHERE status = 'install ok installed'
        AND (name GLOB 'php[0-9].[0-9]-common' OR name GLOB 'php[0-9].[0-9]-fpm'
             OR name GLOB 'php[0-9].[0-9]-cgi' OR name GLOB 'php[0-9].[0-9]-cli'
             OR name GLOB 'php[0-9].[0-9]-embed' OR name GLOB 'libapache2-mod-php[0-9].[0-9]'
             OR name GLOB 'lsphp[0-9][0-9]' OR name GLOB 'lsphp[0-9][0-9]-common'
             OR name IN ('php5', 'php5-common', 'php5-fpm', 'php5-cgi', 'php5-cli', 'libapache2-mod-php5'))
      UNION ALL
      SELECT name, version || '-' || release
      FROM rpm_packages
      WHERE version GLOB '[5-8].[0-9]*.[0-9]*'
        AND (name IN ('php', 'php-common', 'php-cli', 'php-fpm', 'php-cgi')
             OR name GLOB 'php[0-9]*-common' OR name GLOB 'php[0-9]*-fpm' OR name GLOB 'php[0-9]*-cli'
             OR name GLOB 'php[0-9]' OR name GLOB 'php[0-9][0-9]' OR name GLOB 'php[0-9].[0-9]'
             OR name GLOB 'apache2-mod_php*' OR name GLOB 'mod_php*'
             OR name GLOB 'lsphp[0-9][0-9]' OR name GLOB 'ea-php[0-9][0-9]-php-common' OR name GLOB 'alt-php[0-9][0-9]')
    ),
    branched AS (
      SELECT p.name, p.pkg,
             -- Strip a Debian epoch ("1:7.4.33-...") so the branch is always the first 3 chars.
             CASE WHEN p.pkg LIKE '%:%' THEN substr(p.pkg, instr(p.pkg, ':') + 1) ELSE p.pkg END AS v,
             (p.pkg LIKE '%sury%' OR p.pkg LIKE '%+0~20%') AS sury,
             -- Remi, LiteSpeed, cPanel EasyApache, and CloudLinux ship upstream version numbers
             (p.pkg LIKE '%.remi%' OR p.name GLOB 'lsphp*' OR p.name GLOB 'ea-php*' OR p.name GLOB 'alt-php*') AS upstream,
             (p.pkg LIKE '%.alma.%') AS alma,
             o.platform, o.platform_like, o.major, o.version AS os_version
      FROM php p, os o
    ),
    fixed AS (
      SELECT name, pkg, v, substr(v, 1, 3) AS branch,
        -- Ubuntu security updates always carry "ubuntu" in the version; a GA sync build like
        -- 8.3.6-0maysync1 does not, and version_compare can't order it against the fixed build
        (platform = 'ubuntu' AND NOT sury AND NOT upstream AND v NOT LIKE '%ubuntu%') AS ga_build,
        CASE
          -- packages.sury.org and ppa:ondrej/php ship upstream version numbers (EOL branches: patched revisions)
          WHEN sury THEN CASE substr(v, 1, 3)
            WHEN '8.5' THEN '8.5.12' WHEN '8.4' THEN '8.4.27' WHEN '8.3' THEN '8.3.36' WHEN '8.2' THEN '8.2.35'
            WHEN '8.1' THEN '8.1.34-9' WHEN '8.0' THEN '8.0.30-25' WHEN '7.4' THEN '7.4.33-31'
            WHEN '7.3' THEN '7.3.33-35' WHEN '7.2' THEN '7.2.34-66' WHEN '7.1' THEN '7.1.33-80'
            WHEN '7.0' THEN '7.0.33-92' WHEN '5.6' THEN '5.6.40-104' END
          WHEN upstream THEN CASE substr(v, 1, 3)
            WHEN '8.5' THEN '8.5.12' WHEN '8.4' THEN '8.4.27' WHEN '8.3' THEN '8.3.36' WHEN '8.2' THEN '8.2.35' END
          -- Ubuntu backports without changing the upstream version (USN-8948-1, USN-8948-2)
          WHEN platform = 'ubuntu' THEN CASE
            WHEN os_version LIKE '26.10%' AND substr(v, 1, 3) = '8.5' THEN '8.5.9-0ubuntu2.1'
            WHEN os_version LIKE '26.04%' AND substr(v, 1, 3) = '8.5' THEN '8.5.4-0ubuntu1.5'
            WHEN os_version LIKE '24.04%' AND substr(v, 1, 3) = '8.3' THEN '8.3.6-0ubuntu0.24.04.13'
            WHEN os_version LIKE '22.04%' AND substr(v, 1, 3) = '8.1' THEN '8.1.2-1ubuntu2.28'
            WHEN os_version LIKE '20.04%' AND substr(v, 1, 3) = '7.4' THEN '7.4.3-4ubuntu2.29+esm5'
            WHEN os_version LIKE '18.04%' AND substr(v, 1, 3) = '7.2' THEN '7.2.24-0ubuntu0.18.04.17+esm14'
            WHEN os_version LIKE '16.04%' AND substr(v, 1, 3) = '7.0' THEN '7.0.33-0ubuntu0.16.04.16+esm22'
            WHEN os_version LIKE '14.04%' AND substr(v, 1, 3) = '5.5' THEN '5.5.9+dfsg-1ubuntu4.29+esm18' END
          -- Debian (DSA-6574-1, DLA-4848-1) and Freexian ELTS (ELA-1853-1, ELA-1854-1)
          WHEN platform = 'debian' THEN CASE
            WHEN substr(v, 1, 3) = '8.5' THEN '8.5.12' WHEN substr(v, 1, 3) = '8.4' THEN '8.4.27'
            WHEN substr(v, 1, 3) = '8.3' THEN '8.3.36' WHEN substr(v, 1, 3) = '8.2' THEN '8.2.35'
            WHEN major = '11' AND substr(v, 1, 3) = '7.4' THEN '7.4.33-1+deb11u14'
            WHEN major = '10' AND substr(v, 1, 3) = '7.3' THEN '7.3.31-1~deb10u16' END
          -- osquery reports every distro with /etc/redhat-release as platform 'rhel', Fedora included, so match Fedora first
          WHEN platform = 'fedora' OR os_version LIKE 'Fedora%' THEN CASE substr(v, 1, 3)
            WHEN '8.5' THEN '8.5.12' WHEN '8.4' THEN '8.4.27' END
          -- AlmaLinux rebuilds keep the old version and add a .alma.1 release (ALSA 2026-10-21)
          WHEN platform = 'rhel' AND alma THEN CASE
            WHEN major = '10' AND substr(v, 1, 3) = '8.4' THEN '8.4.24-1.el10_2.alma.1'
            WHEN major = '10' AND substr(v, 1, 3) = '8.3' THEN '8.3.33-1.el10_2.alma.1'
            WHEN major = '9' AND substr(v, 1, 3) = '8.3' THEN '8.3.33-1.module_el9.8.0+328'
            WHEN major = '9' AND substr(v, 1, 3) = '8.2' THEN '8.2.33-1.module_el9.8.0+327'
            WHEN major = '9' AND substr(v, 1, 3) = '8.1' THEN '8.1.32-1.module_el9.8.0+326'
            WHEN major = '9' AND substr(v, 1, 3) = '8.0' THEN '8.0.30-8.el9_8.alma.1'
            WHEN major = '8' AND substr(v, 1, 3) = '8.2' THEN '8.2.33-1.module_el8.10.0+4320'
            WHEN major = '8' AND substr(v, 1, 3) = '8.0' THEN '8.0.30-1.module_el8.10.0+4319'
            WHEN major = '8' AND substr(v, 1, 3) = '7.4' THEN '7.4.33-8.module_el8.10.0+4321'
            WHEN major = '8' AND substr(v, 1, 3) = '7.3' THEN '7.3.20-1.module_el8.10.0+4316'
            WHEN major = '8' AND substr(v, 1, 3) = '7.2' THEN '7.2.24-1.module_el8.10.0+4315' END
          -- RHEL, Rocky, Oracle, CentOS Stream (RHSA-2026:87070 to 87151). RHEL 9 php:8.4 had no fix on 2026-10-23.
          WHEN platform = 'rhel' THEN CASE
            WHEN major = '10' AND substr(v, 1, 3) = '8.4' THEN '8.4.27'
            WHEN major = '10' AND substr(v, 1, 3) = '8.3' THEN '8.3.36'
            WHEN major = '9' AND substr(v, 1, 3) = '8.3' THEN '8.3.36'
            WHEN major = '9' AND substr(v, 1, 3) = '8.2' THEN '8.2.35'
            WHEN major = '9' AND substr(v, 1, 3) = '8.0' THEN '8.0.30-8.el9_8.1'
            WHEN major = '8' AND substr(v, 1, 3) = '8.2' THEN '8.2.35'
            WHEN major = '8' AND substr(v, 1, 3) = '7.4' THEN '7.4.33-8' END
          -- Amazon Linux 2023 (ALAS2023-2026-3301 to 3304) and the AL2 php8.2 extra. php8.1 and AL2 core php: no fix planned.
          WHEN platform = 'amzn' THEN CASE substr(v, 1, 3)
            WHEN '8.5' THEN '8.5.12' WHEN '8.4' THEN '8.4.27' WHEN '8.3' THEN '8.3.36' WHEN '8.2' THEN '8.2.35' END
          -- SUSE (SUSE-SU-2026:4843-1, 4844-1, 4852-1, 4853-1, 4870-1)
          WHEN platform_like LIKE '%suse%' THEN CASE
            WHEN substr(v, 1, 3) = '8.5' THEN '8.5.12' WHEN substr(v, 1, 3) = '8.4' THEN '8.4.27'
            WHEN substr(v, 1, 3) = '8.3' THEN '8.3.36' WHEN substr(v, 1, 3) = '8.2' THEN '8.2.35'
            WHEN substr(v, 1, 3) = '8.0' THEN '8.0.30-150400.4.80.1'
            WHEN substr(v, 1, 3) = '7.4' AND major = '15' THEN '7.4.33-150400.4.81.1'
            WHEN substr(v, 1, 3) = '7.4' AND major = '12' THEN '7.4.33-1.74.1' END
          -- Any other distro: supported branches use upstream numbers; EOL branches need a vendor check
          ELSE CASE substr(v, 1, 3)
            WHEN '8.5' THEN '8.5.12' WHEN '8.4' THEN '8.4.27' WHEN '8.3' THEN '8.3.36' WHEN '8.2' THEN '8.2.35' END
        END AS fixed
      FROM branched
    )
    SELECT 1 WHERE NOT EXISTS (
      SELECT 1 FROM fixed
      -- version_compare treats "9+ubuntu22" as one token and longer tokens win, so turn + and _ into dots first
      WHERE CASE WHEN fixed IS NULL THEN 1
                 WHEN ga_build THEN 1
                 WHEN version_compare(replace(replace(v, '+', '.'), '_', '.'), replace(replace(fixed, '+', '.'), '_', '.')) < 0 THEN 1
                 ELSE 0 END = 1
    );
  critical: true
  description: >-
    Passes when no PHP package is installed, or when every installed PHP package is at or above the vendor's
    fixed build for PHP2Shell (CVE-2026-109618, CVE-2026-109619). Only distro and repository packages are checked.
  resolution: >-
    Run remediate-php2shell-linux.sh. If the policy stays failing, this PHP branch has no vendor fix yet:
    switch to a supported branch or run mitigate-php2shell-linux.sh until you can.
  platform: linux
  calendar_events_enabled: false
```

These are the fixed package versions the policy checks, as published by each vendor on October 23, 2026:

| Distro | PHP branch | Fixed package version |
|---|---|---|
| Ubuntu 26.10, 26.04 | 8.5 | `8.5.9-0ubuntu2.1`, `8.5.4-0ubuntu1.5` ([USN-8948-1](https://ubuntu.com/security/notices/USN-8948-1)) |
| Ubuntu 24.04 | 8.3 | `8.3.6-0ubuntu0.24.04.13` |
| Ubuntu 22.04 | 8.1 | `8.1.2-1ubuntu2.28` |
| Ubuntu 20.04 and 18.04 (Pro) | 7.4, 7.2 | `7.4.3-4ubuntu2.29+esm5`, `7.2.24-0ubuntu0.18.04.17+esm14` ([USN-8948-2](https://ubuntu.com/security/notices/USN-8948-2)) |
| Debian 13, 12 | 8.4, 8.2 | `8.4.27-1~deb13u1`, `8.2.35-1~deb12u1` ([tracker](https://security-tracker.debian.org/tracker/CVE-2026-109618)) |
| RHEL 10 | 8.3, 8.4 | `8.3.36-1.el10_2`, `php8.4-8.4.27-1.el10_2` ([RHSB-2026-013](https://access.redhat.com/security/vulnerabilities/RHSB-2026-013)) |
| RHEL 9 | 8.0 (default), 8.2, 8.3 | `8.0.30-8.el9_8.1`, `8.2.35`, `8.3.36` module streams |
| RHEL 8 | 7.4, 8.2 | `7.4.33-8.module+el8.10.0...`, `8.2.35` module streams |
| AlmaLinux 8, 9, 10 | all shipped streams | Rebuilds with a `.alma.1` release, for example `8.2.33-1.module_el9.8.0+327...alma.1` ([blog](https://almalinux.org/blog/2026-10-21-php2shell-cve-2026-109618-cve-2026-109619/)) |
| Amazon Linux 2023 | 8.2 to 8.5 | `8.2.35`, `8.3.36`, `8.4.27`, `8.5.12` ([ALAS](https://explore.alas.aws.amazon.com/CVE-2026-109618.html)). No fix planned for `php8.1`. |
| Fedora 44 and 46 | 8.5 | `8.5.12` |
| SUSE 15 SP7, SP6, SP4 and SP5 LTSS | 8.3, 8.2, 8.0 | `8.3.36`, `8.2.35`, `8.0.30-150400.4.80.1` ([SUSE](https://www.suse.com/security/cve/CVE-2026-109618.html)) |
| packages.sury.org, ppa:ondrej/php, Remi, LiteSpeed, cPanel, CloudLinux | 8.2 to 8.5 | Upstream versions |

For hosts that fail, run [`remediate-php2shell-linux.sh`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/linux/scripts/remediate-php2shell-linux.sh), or set it as the policy's [run-script automation](https://fleetdm.com/guides/policy-automation-run-script). The script upgrades every installed PHP package with apt, dnf, yum, zypper, or apk, prints the full package versions and whether the changelog names the CVEs, restarts PHP-FPM pools and Apache when `libphp` is loaded, and warns about workers that still map a replaced binary. It also lists containers, FrankenPHP, and unpackaged PHP binaries it can't update. It exits `0` when the update went through, `2` when something needs a look, and `1` on error.

> **Note:** Fleet runs a policy's script automation up to three times while the host keeps failing. A host on a branch with no vendor fix stays red after the script runs. Its script output says so. Move it to a supported branch or an LTS vendor, and apply step 5 in the meantime.

Containers and FrankenPHP need their own update: pull or rebuild on `php:8.4.27` (or the matching 8.2, 8.3, or 8.5 tag) and recreate the container, or install FrankenPHP 1.13.2 or newer.

## Step 3: Update Homebrew PHP on macOS

A Mac is only exposed when it serves web requests with PHP, for example a developer's `php-fpm` or Homebrew's `httpd` with `mod_php`. The fix is the same. Add this policy:

```yaml
- name: macOS - Homebrew PHP patched for PHP2Shell (CVE-2026-109618, CVE-2026-109619)
  query: |
    SELECT 1 WHERE NOT EXISTS (
      SELECT 1 FROM homebrew_packages
      WHERE (name = 'php' OR name LIKE 'php@%')
        -- A Homebrew revision ("8.3.35_1") would otherwise outrank 8.3.36, so compare it as a 4th number
        AND CASE
              WHEN version LIKE '8.5.%' THEN version_compare(replace(version, '_', '.'), '8.5.12') < 0
              WHEN version LIKE '8.4.%' THEN version_compare(replace(version, '_', '.'), '8.4.27') < 0
              WHEN version LIKE '8.3.%' THEN version_compare(replace(version, '_', '.'), '8.3.36') < 0
              WHEN version LIKE '8.2.%' THEN version_compare(replace(version, '_', '.'), '8.2.35') < 0
              ELSE 1
            END
    );
  critical: false
  description: >-
    Passes when no Homebrew PHP formula is installed, or when every php or php@X.Y formula is at or above
    8.5.12, 8.4.27, 8.3.36, or 8.2.35. Fails for php@8.1 and older, which Homebrew no longer updates.
  resolution: >-
    Run remediate-php2shell-macos.sh. It upgrades each PHP formula as the Homebrew owner and restarts
    PHP services managed by launchd. Replace php@8.1 or older with a supported formula.
  platform: darwin
  calendar_events_enabled: false
```

For hosts that fail, run [`remediate-php2shell-macos.sh`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/macos/scripts/remediate-php2shell-macos.sh). Fleet runs scripts as root and `brew` refuses to run as root, so the script runs `brew update` and `brew upgrade` as the Homebrew owner, then restarts `php-fpm` and `httpd` services that launchd manages for Homebrew. It warns about MAMP, which bundles its own PHP, and about containers.

## Step 4: Replace PHP builds on Windows

Windows builds come from [windows.php.net](https://windows.php.net/download/) as zip files, so there is no package manager to drive. The policy finds `php.exe` in the common locations and in running processes, and checks each build's file version:

```yaml
- name: Windows - PHP builds patched for PHP2Shell (CVE-2026-109618, CVE-2026-109619)
  query: |
    WITH candidates AS (
      SELECT path FROM file
      WHERE path LIKE 'C:\php%\php.exe' OR path LIKE 'C:\Program Files\PHP\php.exe'
         OR path LIKE 'C:\Program Files\PHP\%\php.exe' OR path LIKE 'C:\Program Files (x86)\PHP\%\php.exe'
         OR path LIKE 'C:\xampp\php\php.exe' OR path LIKE 'C:\wamp64\bin\php\%\php.exe'
         OR path LIKE 'C:\wamp\bin\php\%\php.exe' OR path LIKE 'C:\laragon\bin\php\%\php.exe'
         OR path LIKE 'C:\tools\php%\php.exe' OR path LIKE 'C:\inetpub\PHP\%\php.exe'
         OR path LIKE 'C:\Program Files\iis express\PHP\%\php.exe'
      UNION
      SELECT DISTINCT path FROM processes
      WHERE name IN ('php.exe', 'php-cgi.exe', 'php-win.exe') AND path <> ''
    )
    SELECT 1 WHERE NOT EXISTS (
      SELECT 1 FROM candidates c
      JOIN file f ON f.path = c.path
      WHERE CASE
              WHEN f.file_version LIKE '8.5.%' THEN version_compare(f.file_version, '8.5.12') < 0
              WHEN f.file_version LIKE '8.4.%' THEN version_compare(f.file_version, '8.4.27') < 0
              WHEN f.file_version LIKE '8.3.%' THEN version_compare(f.file_version, '8.3.36') < 0
              WHEN f.file_version LIKE '8.2.%' THEN version_compare(f.file_version, '8.2.35') < 0
              ELSE 1
            END
    );
  critical: false
  description: >-
    Passes when no PHP build is found, or when every php.exe in the common install locations and every running
    php.exe, php-cgi.exe, or php-win.exe reports a file version at or above 8.5.12, 8.4.27, 8.3.36, or 8.2.35.
  resolution: >-
    Download the fixed build for the same branch and architecture from windows.php.net, replace the PHP
    directory, then restart IIS (iisreset) or the Apache service. Until then, run mitigate-php2shell-windows.ps1.
  platform: windows
  calendar_events_enabled: false
```

For hosts that fail:

1. Download the fixed build for the same branch, architecture, and thread-safety variant from [windows.php.net](https://windows.php.net/download/), and replace the PHP directory. For XAMPP, WAMP, or Laragon, install the bundle's updated PHP.
2. Restart IIS with `iisreset`, or restart the Apache service, so `php-cgi.exe` workers load the new build.
3. Until you can replace the build, run [`mitigate-php2shell-windows.ps1`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/windows/scripts/mitigate-php2shell-windows.ps1). It reports every build it finds, including the ones IIS's FastCGI handlers and Apache's `LoadModule` lines point at, without running any of them. It writes the step 5 settings into every ini file PHP can load for those builds, including the IIS `PHPRC`, Apache `PHPIniDir`, and registry locations, with a `.pre-php2shell` backup. It restarts IIS and Apache services when their PHP workers started before the change.

## Step 5: Apply the interim mitigation where you can't update yet

PHP's advisories describe partial mitigations for hosts that can't update. None of them makes a host safe, and PHP says so. They do close the confirmed paths:

- `session.upload_progress.enabled = 0` removes the session upload progress feature that three of the five known techniques use. It must be set in `php.ini`, an FPM pool, or the web server configuration, because PHP parses the request before `ini_set()` or `.user.ini` run. PHP recommends leaving it off even after you update, if your applications don't use it.
- `fatal_error_backtraces = 0` closes the confirmed mod_php path on PHP 8.5. Older releases ignore the directive.
- `memory_limit = -1`, enforced with `php_admin_value` so applications can't restore a finite limit, removes the predictable per-request limit that CVE-2026-109618 relies on. It also removes PHP's protection against runaway scripts, so PHP pairs it with a container or cgroup `memory.max` limit and a conservative `post_max_size`, and warns against `RLIMIT_AS` and `RLIMIT_DATA`, which make the vulnerable path more likely. The scripts in this guide don't apply it. Decide per host.

On Linux, add this policy. It passes when the host has no PHP configuration directories, or when every PHP configuration directory holds the drop-in that [`mitigate-php2shell-linux.sh`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/linux/scripts/mitigate-php2shell-linux.sh) writes:

```yaml
- name: Linux - PHP session upload progress disabled (PHP2Shell interim mitigation)
  query: |
    WITH conf_dirs AS (
      -- The file table returns directories matched with LIKE with a trailing slash, so the patterns end in / and rtrim drops it.
      SELECT rtrim(path, '/') AS path FROM file
      WHERE (path LIKE '/etc/php/%/fpm/conf.d/' OR path LIKE '/etc/php/%/apache2/conf.d/'
             OR path LIKE '/etc/php/%/cgi/conf.d/' OR path LIKE '/etc/php/%/litespeed/conf.d/'
             OR path LIKE '/etc/php/%/embed/conf.d/' OR path = '/etc/php.d'
             -- SUSE /etc/php8 and Alpine /etc/php83: LIKE has no [0-9], so GLOB keeps only numeric names
             OR (path LIKE '/etc/php%/conf.d/' AND path GLOB '/etc/php[0-9]*/conf.d/')
             OR path LIKE '/etc/opt/remi/php%/php.d/' OR path LIKE '/opt/cpanel/ea-php%/root/etc/php.d/'
             OR path LIKE '/opt/alt/php%/etc/php.d/')
        AND type = 'directory'
    )
    SELECT 1 WHERE NOT EXISTS (
      SELECT 1 FROM conf_dirs d
      -- file_lines aborts the query when the path doesn't exist, and a JOIN lets SQLite reach it
      -- before the file check, so CASE confirms the drop-in exists before reading it
      WHERE CASE
              WHEN NOT EXISTS (SELECT 1 FROM file f WHERE f.path = d.path || '/99-php2shell-mitigation.ini') THEN 1
              WHEN NOT EXISTS (SELECT 1 FROM file_lines l
                               WHERE l.path = d.path || '/99-php2shell-mitigation.ini'
                                 AND l.line LIKE 'session.upload_progress.enabled%0%') THEN 1
              ELSE 0
            END = 1
    );
  critical: false
  description: >-
    Passes when every PHP configuration directory contains 99-php2shell-mitigation.ini with
    session.upload_progress.enabled = 0. Partial mitigation for CVE-2026-109619; it doesn't cover the
    out-of-memory path. Updating PHP is the only complete fix.
  resolution: >-
    Run mitigate-php2shell-linux.sh. It writes the drop-in into every PHP configuration directory and
    restarts PHP-FPM, and Apache when mod_php is loaded. Then update PHP as soon as a fixed package is available.
  platform: linux
  calendar_events_enabled: false
```

Scope the policy to the hosts that need it with a label, or keep it everywhere if you want session upload progress off across the board. The script writes `99-php2shell-mitigation.ini` into Debian's `/etc/php/X.Y/<sapi>/conf.d`, RHEL's `/etc/php.d`, SUSE's `/etc/php8/conf.d`, and the Remi, cPanel, CloudLinux, and LiteSpeed layouts, then restarts PHP-FPM and Apache when `libphp` is loaded. To roll back, delete the drop-ins and restart the same services.

## Step 6: Look for signs of exploitation

Start the lookback on October 17, 2026, the first exploitation attempt Amazon [recorded](https://aws.amazon.com/blogs/security/amazon-threat-intelligence-identifies-php2shell-exploitation-prior-to-public-disclosure-in-madpot/), not the disclosure date. On the host, the reports from step 1 show two things worth a look:

- `web_worker_child`: a shell, interpreter, or download tool whose parent is `php-fpm`, `apache2`, `httpd`, `php-cgi`, or `lsphp`. Successful exploitation runs commands as the web server user, and that is what it looks like. Apache CGI scripts and cron-style maintenance can do this legitimately, so check the command line in `detail`.
- `php_in_tmp`: a `.php` file in `/tmp`, `/var/tmp`, or `/dev/shm`.

In your web server and PHP logs, look for:

- `POST` requests with multipart bodies that carry `PHP_SESSION_UPLOAD_PROGRESS` or `PHPSESSID` fields on endpoints that don't use them, deeply nested field names like `files[a][a][a]`, or many parts with empty filenames.
- PHP fatal errors `Allowed memory size of N bytes exhausted` at `Unknown on line 0`, which means the request failed during startup before any script ran, and Apache `AH00052` segmentation faults from failed address guessing.
- `Set-Cookie: PHPSESSID=` values that are unusually long or percent-encoded. The memory-disclosure variant leaks worker memory through that cookie.

Clean logs don't prove a host is safe. PHP's own advisory says so, and Datadog's [test exploitation](https://securitylabs.datadoghq.com/articles/php2shell-critical-vulnerabilities-in-php/) returned HTTP 200 without crashing the worker.

## Verify

1. Go to **Policies** and confirm the three patched-for-PHP2Shell policies show no failing hosts, apart from hosts you've documented as waiting on a vendor fix.
2. Re-run the step 1 report for each OS. Every `php_package`, `homebrew_php`, and `php_build` row should read `FIXED`, and no `php_worker` or `mod_php` row should read `STALE`.
3. On a Linux host, confirm the running PHP is the new one:

   ```bash
   dpkg -l 'php*' | awk '/^ii/ { print $2, $3 }'   # Debian and Ubuntu
   rpm -qa 'php*' | sort                             # RHEL family, Amazon Linux, Fedora, SUSE
   sudo grep -lE ' /(usr|lib|opt).*\(deleted\)$' /proc/[0-9]*/maps   # empty means no worker maps a replaced binary
   ```

## What Fleet's vulnerability detection shows

Fleet's software inventory lists the same packages the reports use, and Fleet's [vulnerability detection](https://fleetdm.com/software-management) matches them against several feeds. NVD lists CVE-2026-109618 and CVE-2026-109619 with CPE ranges for `php:php`, which covers Homebrew PHP on macOS. Ubuntu and RHEL hosts are matched through Canonical's and Red Hat's OVAL definitions, and Amazon Linux through ALAS, so expect the CVEs to appear under **Software > Vulnerabilities** for those hosts once the feeds include the advisories. Fleet refreshes them daily.

Three gaps to know about. Debian, SUSE, and third-party repository packages aren't matched by those feeds. Derived distros like AlmaLinux are evaluated with Red Hat's definitions, so a patched Alma backport with a lower version number can still show as vulnerable. And no feed sees a PHP inside a container or a FrankenPHP binary. The policies and reports in this guide don't depend on any feed, so use them as the source of truth for this one.

## Further reading

- [PHP: October 2026 security release](https://www.php.net/archive/2026.php#2026-10-21-5)
- [GHSA-w8hw-w9fx-j79w: RCE via OOM corruption (CVE-2026-109618)](https://github.com/php/php-src/security/advisories/GHSA-w8hw-w9fx-j79w)
- [GHSA-6wmf-4rv5-p3rp: RCE via stale global state (CVE-2026-109619)](https://github.com/php/php-src/security/advisories/GHSA-6wmf-4rv5-p3rp)
- [Red Hat security bulletin RHSB-2026-013](https://access.redhat.com/security/vulnerabilities/RHSB-2026-013)
- [Amazon Threat Intelligence: PHP2Shell exploitation before disclosure](https://aws.amazon.com/blogs/security/amazon-threat-intelligence-identifies-php2shell-exploitation-prior-to-public-disclosure-in-madpot/)
- [Reports in Fleet](https://fleetdm.com/guides/reports)
- [Scripts in Fleet](https://fleetdm.com/guides/scripts)
- [Automatically run scripts on failing policies](https://fleetdm.com/guides/policy-automation-run-script)

<meta name="articleTitle" value="Find and fix PHP2Shell exposure with Fleet">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="publishedOn" value="2026-10-23">
<meta name="category" value="guides">
<meta name="description" value="Find every PHP runtime, confirm each vendor's PHP2Shell fix, restart stale workers, and mitigate the rest with Fleet reports, policies, and scripts.">
