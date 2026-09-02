# What Chrome 152's two critical flaws mean for confirming browser patch coverage

*Chrome 152 fixes 26 externally reported vulnerabilities, including two critical use-after-free bugs in Shared Tab Groups and WebGL that could lead to remote code execution. Here's how to confirm the update reached every device instead of assuming auto-update caught everyone.*

## Key takeaways

- **"Chrome auto-updates" isn't the same as "Chrome is updated."** The update only takes effect after a relaunch, so a browser left open for days can carry a critical fix that installed but never activated. Fleet's software inventory reports the exact Chrome version running on every host, so you can confirm the update landed instead of assuming it did.
- **Two of the 26 fixes in this release are critical, and both are memory-safety bugs.** CVE-2026-84353 (Shared Tab Groups) and CVE-2026-84352 (WebGL) are both use-after-free vulnerabilities, the class of bug that can let an attacker run code in the browser process, not just crash it.
- **Chrome ships different build numbers per platform, so "up to date" means something different on each one.** Chrome 152 is 152.0.7977.64 on Linux and 152.0.7977.64/.65 on Windows and macOS. Fleet's inventory already separates hosts by platform, so the comparison is against the right build either way.
- **A version check becomes a standing policy, not a one-time sweep.** Save the minimum-version comparison as a Fleet policy and the next Chrome release gets checked the same way, automatically, across every host that enrolls or changes.
- **The same inventory feeds vulnerability matching.** Once Chrome's version is in Fleet's software table, it's cross-referenced against CVE data the same way every other piece of software on the fleet is, so a missed update surfaces as a flagged vulnerability, not a browser tab nobody looked at again.

<a purpose="cta-button" href="https://fleetdm.com/software-catalog">See software inventory in Fleet</a>

Google shipped Chrome 152 with fixes for 26 externally reported vulnerabilities, two of them critical. Both critical bugs, in Shared Tab Groups and in WebGL, are use-after-free flaws: the kind of memory-safety issue that occurs when code keeps referencing memory that's already been freed, and that attackers can sometimes turn into remote code execution rather than just a crash. Neither is reported as under active exploitation yet, which is the window where getting ahead of it counts for something.

The harder question isn't whether the fix exists. It's whether it reached every machine in your fleet, and that's a question "Chrome auto-updates" doesn't answer.

## Why an installed update and a running update aren't the same thing

Chrome's auto-update mechanism downloads and stages a new version in the background, but the fix doesn't take effect until the browser restarts. A laptop that's been asleep, a desktop where Chrome has stayed open through a dozen tabs and a long week, a session where someone dismissed the "relaunch to update" prompt again: all of these can be sitting on a patched-on-disk, vulnerable-in-memory version of Chrome. From the outside, that looks identical to a fully updated browser. The only way to tell the difference is to check the version running, not the version that shipped.

## Confirming the version instead of assuming it

Fleet's software inventory reports the exact Chrome version installed on every enrolled host, macOS, Windows, and Linux alike, the same way it reports any other piece of installed software. Search the software inventory for Google Chrome and you get every host running it, broken out by the version each one reports, which turns "did the fix land" from a guess into a number: how many hosts are still below 152.0.7977.64 (Linux) or 152.0.7977.64/.65 (Windows and macOS), right now.

That platform split matters more than it looks. Chrome's release cadence doesn't always land the same build number across operating systems at the same moment, so a version check that doesn't distinguish platforms can tell you a Linux host is behind when it's current, or the reverse. Fleet's inventory already separates hosts by platform, so the comparison is against the right target either way.

## From a one-time check to an enforced policy

Running that check once tells you where you stand today. Saving it as a Fleet policy tells you where you stand every day going forward: a minimum-version comparison for Chrome that runs on a schedule and against every host that enrolls or changes, so the next critical Chrome release doesn't require remembering to go check again. Because policies live in Git as YAML and deploy through the same GitOps workflow as everything else Fleet manages, bumping the minimum version when Chrome 153 ships is a reviewable pull request, not a console setting someone has to remember to update.

## The same inventory does the vulnerability matching for you

Software inventory isn't just a version lookup. Once Chrome's version is in Fleet's software table, it's matched against CVE data from sources like the National Vulnerability Database the same way every other piece of installed software is, so a host still running a version affected by CVE-2026-84353 or CVE-2026-84352 shows up as a flagged vulnerability tied to that specific host, not a line in a browser release blog you'd have to cross-reference by hand.

## Patch cadence isn't slowing down

Chrome ships security fixes on a tight, regular cycle, and this release is a reminder that "regular" doesn't mean "low stakes": two critical memory-safety bugs in a browser that's open on nearly every managed device, all day, every day. The teams that handle a release like this well aren't the ones that patch fastest. They're the ones who already know, in seconds, which of their devices are still running the vulnerable build.

## See it live

- **Get a demo** to see Chrome's version reported across your own fleet: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Explore the software catalog** Fleet already builds from every host: [fleetdm.com/software-catalog](https://fleetdm.com/software-catalog)

## Sources

- Cyber Security News, [Google Chrome 152 Released With 327 Security Fixes, Including 10 Critical Vulnerabilities](https://cybersecuritynews.com/chrome-152-released-with-327-security-fixes/).
- SecurityWeek, [Chrome 152 Patches Over 300 Vulnerabilities](https://www.securityweek.com/chrome-152-patches-over-300-vulnerabilities/).

---
*See which of your devices are still exposed. [Talk to Fleet](https://fleetdm.com/contact), or explore the [software catalog](https://fleetdm.com/software-catalog) Fleet already builds from your fleet.*

<meta name="articleTitle" value="What Chrome 152's two critical flaws mean for confirming browser patch coverage">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-09-02">
<meta name="description" value="Chrome 152 fixes two critical bugs. See how to confirm the update reached every device instead of assuming auto-update caught everyone.">
