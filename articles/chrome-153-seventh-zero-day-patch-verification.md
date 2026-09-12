# Google shipped a fix for Chrome's seventh 2026 zero-day. Most fleets can't say which hosts got it.

*CVE-2026-87491 is an actively exploited V8 flaw, patched in Chrome 153.0.8010.36. Here is how to confirm the fix reached every host instead of assuming auto-update did.*

## Key takeaways

- **This is Chrome's seventh zero-day of the year, and it's already being exploited.** CVE-2026-87491 is a V8 engine flaw Google says attackers are using in the wild, not a theoretical bug researchers found first.
- **A patched build number and a patched fleet are two different claims.** Chrome 153.0.8010.36 fixes the flaw, but the fix only helps hosts that are running it.
- **Auto-update answers "did Chrome download it," not "is Chrome running it."** A browser that's been asleep, or sitting on an unclosed "relaunch to update" prompt, can still be executing the vulnerable version in memory.
- **Fleet's software inventory reports the exact Chrome build on every host.** Search it once and you get a straight answer: how many hosts are still below 153.0.8010.36, right now, broken out by macOS, Windows, and Linux.
- **Seven zero-days in a year is a pattern, not a one-off.** A version check you run once answers today's question. A Fleet policy that checks Chrome's version on a schedule answers it every time Google ships the next one.

<a purpose="cta-button" href="https://fleetdm.com/software-catalog">See software inventory in Fleet</a>

Google patched CVE-2026-87491 in Chrome 153.0.8010.36, the seventh actively exploited zero-day fixed in Chrome so far this year. Unlike a routine patch, active exploitation means the fix isn't optional homework for later. It's a race between the attackers who already have a working exploit and however long it takes your fleet to actually run the new build.

That race isn't won when Google ships the patch. It's won when every host on your network is running it.

## Auto-update tells you what shipped, not what's running

Chrome's auto-update mechanism downloads and stages a new version quietly in the background. The catch is that the fix doesn't take effect until the browser restarts. A laptop that's been asleep, or a browser tab someone's been ignoring the "relaunch to update" nudge on for a week, is still executing the vulnerable version in memory even though the installer says it's current.

From IT's vantage point, that host looks fine. Auto-update ran, the download completed, and there's nothing in a standard device inventory to distinguish it from a host that relaunched. The only way to tell the difference is to check the version Chrome is running, not the version it downloaded.

## Turning "probably patched" into a number

Fleet's software inventory reports the exact Chrome build installed on every enrolled host, macOS, Windows, and Linux alike, the same way it reports every other piece of installed software. Search the inventory for Google Chrome and the result isn't a guess. It's a list: every host running it, broken out by the version each one reports, so "did CVE-2026-87491's fix land" becomes a count of how many hosts are still below 153.0.8010.36 instead of a hope that auto-update caught everyone.

That inventory feeds Fleet's vulnerability matching automatically. Once a host's Chrome version is in the software table, it's cross-referenced against CVE data the same way every other installed package is, so a host still running a build affected by CVE-2026-87491 shows up as a flagged vulnerability tied to that specific machine, not a line in a release note someone has to remember to cross-check.

## Seven zero-days in, checking once isn't the finish line

CVE-2026-87491 is the seventh Chrome zero-day patched this year. Whatever number it lands on by December, another one is coming, and the version check that answers today's question won't answer the next one on its own. Saving that check as a Fleet policy, a minimum Chrome version compared against every host on a schedule, means the next actively exploited flaw doesn't start with someone remembering to go look. Because policies live in Git as YAML and deploy through the same GitOps workflow as everything else Fleet manages, bumping the minimum version when Chrome 154 ships is a reviewable pull request, not a task on a list nobody checks until the next incident.

## See it live

- **Get a demo** to see Chrome's version reported across your own fleet: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Explore the software catalog** Fleet already builds from every host: [fleetdm.com/software-catalog](https://fleetdm.com/software-catalog)

<meta name="articleTitle" value="Google shipped a fix for Chrome's seventh 2026 zero-day. Most fleets can't say which hosts got it.">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-09-10">
<meta name="description" value="CVE-2026-87491 is Chrome's seventh 2026 zero-day. See how to confirm the fix reached every host instead of trusting auto-update.">
