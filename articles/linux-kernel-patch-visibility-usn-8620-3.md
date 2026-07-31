# A routine Ubuntu kernel patch is a reminder that Linux visibility can't wait for a headline CVE

*USN-8620-3 doesn't have a catchy name or a logo. It's still the kind of patch that quietly separates fleets with real Linux visibility from fleets running on a spreadsheet.*

## Key takeaways

- **Niche Linux hardware is where patch tracking usually breaks down.** Intel IoT Gateway systems aren't part of anyone's standard laptop refresh cycle, so a kernel advisory that only affects them is easy to miss until an auditor or an attacker finds it first.
- **A patched kernel package isn't the same as a patched host.** USN-8620-3 requires a reboot to take effect, so the real question isn't "did the package install," it's "is the host still running the vulnerable kernel right now."
- **Fleet checks every installed kernel, not just the one that's running.** Fleet matches Ubuntu kernel vulnerabilities against installed `linux-image` packages, so a host that installed the fix but hasn't rebooted yet still shows up accurately, instead of looking falsely clean.
- **New advisories show up without anyone cross-referencing a CVE list by hand.** Fleet refreshes its Ubuntu OVAL data daily against Canonical's feed, so an advisory like USN-8620-3 appears in vulnerability results on its own.
- **Unusual kernel variants don't silently fall out of coverage.** Canonical's OVAL feed is written around the mainstream `-generic` kernel; Fleet falls back to NVD-based matching for variants like `-intel-iotg` so specialty hardware still gets flagged.
- **"Are we patched" becomes a query you can run today, on every Ubuntu host in the fleet.** Not just the ones IT remembers to check.

<a purpose="cta-button" href="/linux-management">See Linux management in Fleet</a>

Ubuntu shipped [USN-8620-3](https://ubuntu.com/security/notices/USN-8620-3) on July 31, fixing kernel vulnerabilities in the `linux-intel-iotg-5.15` and `linux-intel-iot-realtime` packages. It's not the kind of advisory that makes headlines. It targets Intel IoT Gateway class hardware, the kind of embedded, out-of-band Linux system that runs a point-of-sale terminal, a factory floor controller, or a retail kiosk rather than someone's laptop.

That's exactly why it's worth pausing on. Most Linux patch management effort goes toward the hosts everyone already tracks. The hosts that get missed are the ones running on hardware nobody thinks about until something goes wrong, and a routine kernel patch is the moment to check whether your tooling would have caught it.

## What USN-8620-3 fixes

The advisory covers Ubuntu 22.04 LTS (kernel 5.15.0-1104.106) and Ubuntu 20.04 LTS (kernel 5.15.0-1107.113~20.04.1) running the Intel IoT Gateway kernel packages. Among the vulnerabilities patched:

- **CVE-2023-45896**, an out-of-bounds read in the NTFS file system driver that can expose kernel memory.
- **CVE-2025-54505**, an AMD floating-point divider flaw that can leak sensitive information.
- **CVE-2025-54518**, a cache isolation issue in AMD Zen 2 processors that can enable privilege escalation.

Canonical's notice references more than 750 other CVEs fixed across various kernel subsystems in the same update. The patch also carries a reboot requirement and an unavoidable ABI change, meaning any third-party kernel modules on these hosts need to be recompiled against the new kernel.

## Why "the package is installed" isn't the finish line

A reboot requirement changes what "patched" means in practice. A host can have the fixed kernel package sitting on disk and still be running the vulnerable one, sometimes for days, if nobody flags the pending reboot. On a laptop, a user restarting for an OS update usually closes that gap on its own. On an IoT gateway that runs unattended in a back room, nothing forces that reboot, and the vulnerable kernel can keep running indefinitely.

This is where Fleet's approach to Ubuntu kernel vulnerabilities matters. Fleet matches vulnerabilities against every installed kernel package on a host that matches `linux-image.*`, not only the kernel that's actively running. That means a host sitting on a patched-but-not-yet-rebooted kernel, and a host still running the vulnerable kernel outright, are both visible for what they are, instead of one looking falsely clean because a package manager reported success.

## Coverage that doesn't stop at the mainstream kernel

Fleet's Linux vulnerability detection runs on OVAL, the open standard Canonical and other distribution maintainers use to publish machine-readable vulnerability definitions. Fleet downloads and refreshes the relevant OVAL data for each Ubuntu version in your fleet daily, so an advisory like USN-8620-3 becomes visible in Fleet without anyone manually tracking new USNs against a hardware inventory.

Canonical's OVAL feed is written primarily around the `-generic` kernel flavor most Ubuntu installs use. Intel IoT Gateway systems run a different kernel variant, `-intel-iotg`, which isn't always represented the same way in that feed. When Fleet detects a kernel variant that the OVAL data doesn't cover directly, it falls back to NVD-based matching against the kernel's CPE, so specialty hardware classes like IoT gateways stay covered instead of quietly falling outside the detection logic built for the mainstream case.

## The bottom line

USN-8620-3 will patch quietly, on hardware most patch management conversations never mention. That's the point: the value of Linux vulnerability detection shows up on the advisories nobody's watching for, not the ones already making the rounds. If Fleet's agent is already installed on your Ubuntu fleet, including the Intel IoT Gateway boxes, you already have the answer to whether any host is still running the kernel USN-8620-3 fixes.

*Fleet is the open-source platform for managing and securing macOS, Windows, Linux, and more from one place.* [*Get a demo*](https://fleetdm.com/contact) *or read more about how Fleet handles* [*Linux vulnerability detection with OVAL*](https://fleetdm.com/articles/linux-vulnerability-detection-with-oval-and-fleet)*.*

<meta name="articleTitle" value="A routine Ubuntu kernel patch is a reminder that Linux visibility can't wait for a headline CVE">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-07-31">
<meta name="description" value="USN-8620-3 patches Ubuntu kernel vulnerabilities on Intel IoT Gateway systems. Here's how Fleet's OVAL-based Linux vulnerability detection catches it automatically.">
