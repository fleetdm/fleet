# Chrome 153 starts a two-week release cycle, and Google says most enterprises don't need Extended Stable

*Chrome just cut its stable release cadence in half, from four weeks to two, while telling most enterprise customers to stay off the slower Extended Stable track. Here's what that means for whoever owns the Chrome patch calendar.*

## Key takeaways

- **Chrome's release cadence just doubled.** Chrome 153 began rolling out on September 8, 2026, with Chrome 154 due two weeks later on September 22. That replaces the four-week cycle Chrome had used since 2021, itself a cut from a six-week cadence before that.
- **Extended Stable didn't get faster, on purpose.** The eight-week Extended Stable track keeps its own schedule, moving from Chrome 156 to 160, for organizations that specifically need a longer testing window between updates.
- **Google is pointing most enterprises away from Extended Stable.** With the standard channel now shipping smaller, more frequent updates, Google recommends most enterprise customers stay on regular Stable rather than opt into the slower track, an unusual case of a vendor talking a customer out of its own "safer" option.
- **A version number is worth less if you don't know the channel behind it.** Two hosts on the same major version can be on different update tracks, so confirming Chrome's version across a fleet now means confirming which cadence produced it, not just the number itself.
- **Fleet's software inventory reports the exact Chrome version running on every host.** That turns "which cadence is this device actually on" from a guess into a number you can check across macOS, Windows, and Linux the same way you'd check any other installed software.
- **A saved policy keeps up with a cycle that won't wait for you to notice it.** A two-week cadence turns a "check every so often" habit into a losing bet; a Fleet policy checks the version on a schedule instead.

<a purpose="cta-button" href="https://fleetdm.com/software-catalog">See software inventory in Fleet</a>

Google shipped Chrome 153 on September 8, 2026, and with it, cut the major stable release cycle from four weeks to two. Chrome 154 is already scheduled for September 22. Google says the change is meant to ship features, performance work, and security fixes faster, and to help the browser respond to "fast-moving threats" as AI-assisted attacks accelerate.

For anyone who plans patch windows around Chrome's release calendar, that calendar just moved out from under them. A cadence built around a monthly-ish rhythm now runs on roughly half the clock, and the enterprise track built for teams that wanted things slower didn't change to match.

## Extended Stable stayed put, and Google wants it that way

Extended Stable is the channel Google built for organizations that need more runway between updates, an eight-week cycle that bundles more changes into fewer releases. It isn't going anywhere: the next builds on that track are Chrome 156, then 160, still eight weeks apart.

What changed is Google's guidance around it. With the standard channel now shipping smaller, more frequent updates instead of one large monthly jump, Google is telling most enterprise customers to stay on regular Stable rather than move to Extended Stable. That's a deliberate reversal of the usual enterprise instinct to pick the slower, "safer" track by default, and it means the two-week cycle isn't just a change for browsers that update on their own. It's a decision every fleet running Chrome now has to make on purpose.

## The version number alone doesn't tell you the channel

A build number used to be enough to know roughly where a device stood in Chrome's release calendar, because there was only one calendar. Now there are two: a two-week Stable cycle and an eight-week Extended Stable cycle running in parallel, each producing its own sequence of versions. Two hosts can carry different version numbers for entirely legitimate reasons that have nothing to do with one of them missing an update.

That makes "which channel is this host actually on" as relevant a question as "is this host patched." Answering it for one laptop is easy enough to eyeball. Answering it across a fleet, especially one that never made an explicit decision about Extended Stable, is a different problem.

## Confirming the version, and the channel, across every host

Fleet's software inventory already reports the exact Chrome version installed on every enrolled host, macOS, Windows, and Linux alike, the same way it reports any other piece of installed software. Searching the software inventory for Google Chrome returns every host running it, broken out by version, which turns "who's behind and who's on a different channel entirely" into something you can see rather than something you have to ask around about.

That visibility matters more now than it did under the old four-week cadence, because the gap between "current" and "one release behind" used to give a team roughly a month of runway. Under a two-week cycle, that runway is cut in half, and a host that's quietly drifted onto Extended Stable without anyone deciding it should be there won't look wrong at a glance. It'll just look like a different, equally valid version number, right up until it doesn't match what the rest of the fleet is running.

## Turning the check into a policy the cadence can't outrun

A one-time search answers where things stand today. A saved Fleet policy, a minimum-version check that runs on a schedule against every host that enrolls or changes, answers it continuously, which is what a cycle shipping every two weeks actually requires instead of a habit of remembering to look. Because Fleet policies live in Git as YAML and deploy through the same GitOps workflow as everything else Fleet manages, updating the minimum version when Chrome 154 ships is a reviewable pull request, not a console setting someone has to remember to touch.

## A faster cadence rewards a standing answer, not a periodic check

Chrome moving twice as fast is good news for how quickly fixes reach users. It's a harder problem for anyone whose patch tracking assumed a monthly rhythm that no longer exists, especially with a second, slower channel now running in parallel next to it. The teams that stay ahead of this won't be the ones checking more often. They'll be the ones who already have an answer, for every host, all the time.

## See it live

- **Get a demo** to see Chrome's version and channel drift reported across your own fleet: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Explore the software catalog** Fleet already builds from every host: [fleetdm.com/software-catalog](https://fleetdm.com/software-catalog)

## Sources

- 9to5Google, [Google Chrome starts two-week release cycle with version 153](https://9to5google.com/2026/09/08/chrome-updates-two-weeks/).
- gHacks Tech News, [Chrome 153 Launches, Beginning Google's Two-Week Release Cadence](https://www.ghacks.net/2026/09/09/chrome-153-launches-beginning-googles-two-week-release-cadence/).
- Google, [Extended Stable channel - Chrome Enterprise and Education Help](https://support.google.com/chrome/a/answer/16942104?hl=en).

<meta name="articleTitle" value="Chrome 153 starts a two-week release cycle, and Google says most enterprises don't need Extended Stable">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-09-09">
<meta name="description" value="Chrome 153 cuts its release cycle to two weeks, and Google steers enterprises off Extended Stable. Here's how to track each host's channel.">
