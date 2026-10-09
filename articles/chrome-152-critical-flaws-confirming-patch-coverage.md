# What Chrome 152's two critical flaws mean for confirming browser patch coverage

*Chrome 152 fixes 26 externally reported vulnerabilities, including two critical bugs that could lead to remote code execution. Here is how to confirm the update reached every device instead of just trusting auto-update.*

## Key takeaways

- **"Chrome auto-updates" isn't the same as "Chrome is updated."** Chrome downloads patches automatically, but fixes only activate after a relaunch. A browser left open for days is still vulnerable in memory.
- **Two critical memory-safety flaws.** CVE-2026-84353 (Shared Tab Groups) and CVE-2026-84352 (WebGL) are use-after-free bugs that allow attackers to run code in the browser process.
- **Chrome ships different build numbers per platform.** Chrome 152 is version 152.0.7977.64 on Linux and 152.0.7977.64/.65 on Windows and macOS.
- **Find unpatched browsers instantly.** Fleet's software inventory lets you find the exact Chrome version running on every device.
- **The same inventory feeds vulnerability matching.** Once Chrome's version is in Fleet's software table, it's cross-referenced against CVE data the same way every other piece of software on the fleet is, so a missed update surfaces as a flagged vulnerability, not a browser tab nobody looked at again.

<a purpose="cta-button" href="https://fleetdm.com/software-catalog">See software inventory in Fleet</a>

Google just shipped Chrome 152 with fixes for 26 externally reported vulnerabilities. Two of these are critical flaws in Shared Tab Groups and WebGL. This flavor of memory-safety bug happens when code references freed memory. Attackers can leverage this to run unauthorized code instead of simply crashing the browser. Neither is reported as under active exploitation yet, which is the window where getting ahead of it is the difference between being protected or becoming a victim.

The hardest part of vulnerability management is not knowing if a patch exists. It is knowing if the patch actually applied to every single device in your fleet.

## Why an installed update and a running update aren't the same thing

Chrome's auto-update mechanism downloads and stages a new version in the background, but the fix doesn't take effect until the browser restarts. Whether a laptop has been asleep or an employee keeps ignoring the "relaunch to update" prompt, these devices sit on a patched-on-disk but vulnerable-in-memory version of Chrome.

From the outside, the device looks fully updated. The only way to know for sure is to check the version actively running on the machine.

## Confirming the version instead of assuming it

Fleet's software inventory reports the exact Chrome version installed on every enrolled host, macOS, Windows, and Linux alike, the same way it reports any other piece of installed software. Search the software inventory for Google Chrome and you get every host running it, broken out by the version each one reports, which turns "did the fix land" from a guess into a number: how many hosts are still below 152.0.7977.64 (Linux) or 152.0.7977.64/.65 (Windows and macOS), right now.

That platform split matters more than it looks. Chrome's release cadence doesn't always land the same build number across operating systems at the same moment, so a version check that doesn't distinguish platforms can tell you a Linux host is behind when it's current, or the reverse. Fleet's inventory already separates hosts by platform, so the comparison is against the right target either way.

## From a one-time check to an enforced policy

Running that check once tells you where you stand today. Saving it as a Fleet policy tells you where you stand every day going forward: a minimum-version comparison for Chrome that runs on a schedule and against every host that enrolls or changes, so the next critical Chrome release doesn't require remembering to go check again. Because policies live in Git as YAML and deploy through the same GitOps workflow as everything else Fleet manages, bumping the minimum version when Chrome 153 ships is a reviewable pull request, not a console setting someone has to remember to update.

## The same inventory does the vulnerability matching for you

Software inventory isn't just a version lookup. Once Chrome's version is in Fleet's software table, it's matched against CVE data from sources like the National Vulnerability Database the same way every other piece of installed software is, so a host still running a version affected by CVE-2026-84353 or CVE-2026-84352 shows up as a flagged vulnerability tied to that specific host, not a line in a browser release blog you'd have to cross-reference by hand.

## Patch cadence isn't slowing down

Software inventory is more than just a version lookup. Once Fleet logs a software version, the platform automatically cross-references it against CVE data. This means a host running a vulnerable version of Chrome immediately surfaces as a flagged vulnerability tied to that specific machine. You no longer have to cross-reference with other security tools or release notes by hand.

Patch cadence is not slowing down. The teams that handle browser security best are not always the ones who patch fastest. They are the ones who can identify every exposed device in seconds.

## See it live

- **Get a demo** to see Chrome's version reported across your own fleet: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Explore the software catalog** Fleet already builds from every host: [fleetdm.com/software-catalog](https://fleetdm.com/software-catalog)

## Sources

- Cyber Security News, [Google Chrome 152 Released With 327 Security Fixes, Including 10 Critical Vulnerabilities](https://cybersecuritynews.com/chrome-152-released-with-327-security-fixes/).
- SecurityWeek, [Chrome 152 Patches Over 300 Vulnerabilities](https://www.securityweek.com/chrome-152-patches-over-300-vulnerabilities/).

<meta name="articleTitle" value="What Chrome 152's two critical flaws mean for confirming browser patch coverage">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-09-02">
<meta name="description" value="Chrome 152 fixes two critical bugs. See how to confirm the update reached every device instead of assuming auto-update caught everyone.">
