# iOS 27 fixes 126 flaws, macOS 27 Golden Gate fixes 210, and most fleets can't say who's updated

*Apple's newest releases patched more vulnerabilities than almost any release in recent memory, including kernel bugs, a sandbox escape, and a Gatekeeper bypass. Here's how to find out which of your Macs, iPhones, and iPads actually installed it.*

## Key takeaways

- **You can already see the exact build every Mac, iPhone, and iPad is running.** Fleet's host inventory tracks OS version and build per device, so instead of assuming a major release reached your fleet on day one, you can check.
- **This is an unusually large patch load, even for Apple.** iOS 27 and iPadOS 27 fix about 126 security flaws, macOS 27 Golden Gate fixes 210, and roughly 100 of those overlap between the mobile and desktop releases.
- **The bugs aren't cosmetic.** Apple's advisories list fixes for kernel and root code execution, a sandbox escape, and a Gatekeeper bypass, the kind of flaws that turn a compromised app into a compromised device.
- **Enforcement beats hoping.** Fleet can set a minimum OS version and deadline for macOS, iOS, and iPadOS hosts, so devices that haven't updated get pushed to the new version instead of relying on a reminder email.
- **Coverage isn't uniform on day one.** Hardware compatibility, user-postponed prompts, and staggered MDM enforcement all mean some devices land on Golden Gate faster than others, and only host-level data tells you which ones are still behind.

<a purpose="cta-button" href="https://fleetdm.com/guides/enforce-os-updates">See how to enforce OS updates in Fleet</a>

Apple shipped iOS 27, iPadOS 27, and macOS 27 Golden Gate on September 14, and the security changelogs that came with them are long. iOS 27 and iPadOS 27 fix about 126 vulnerabilities, 20 of them in the kernel. macOS 27 Golden Gate fixes 210, roughly 100 of which are shared with the iOS release. Apple hasn't reported active exploitation of any of them, but the fix count alone makes this one of the larger patch days in recent memory.

For a fleet of any size, a release like this raises a question that a changelog can't answer: how many of your devices actually have it installed.

## What these bugs would have let an attacker do

Apple's advisories describe fixes across more than 90 platform components, including the kernel, Sandbox, AppleKeyStore, TCC, and WebKit. The specific flaw classes matter more than the count: some of the patched bugs could let a malicious app execute code with kernel or root privileges, escape the macOS sandbox, or bypass Gatekeeper's code-signing checks. Any one of those turns a single compromised app into control over the whole device, not just the app's own sandboxed data.

None of this is Apple-specific hyperbole. It's the difference between "install this when convenient" and "confirm this landed everywhere it needs to."

## Finding out who's actually on Golden Gate

Fleet already collects OS name, version, and build for every enrolled host, macOS, iOS, and iPadOS included. You don't have to wait on a survey or a help-desk ticket to know how far the rollout has gotten.

On macOS hosts, a straightforward query gives you the installed version and build directly from the host:

```sql
SELECT version, build FROM os_version;
```

Anything before Golden Gate's build number is still carrying the vulnerabilities patched on September 14.

iPhones and iPads don't run Fleet's agent the way macOS and Linux hosts do, since fleetd doesn't run on iOS or iPadOS, but Fleet still surfaces the reported `os_version` and build for those devices in the host list and API (collected through Apple's MDM protocol). You can filter the hosts list by `os_name` and `os_version` to pull every device still on iOS 26 or iPadOS 26 in one search, the same way you'd filter for an outdated macOS build.

## Enforcing instead of asking nicely

A one-time check tells you where things stand today. It doesn't stop the same gap from reopening at the next major release. Fleet can enforce a minimum OS version and deadline for macOS, iOS, iPadOS, and Windows hosts directly, so noncompliant devices are pushed to update automatically instead of waiting on a user to click "install."

To automatically enforce the latest macOS version and give hosts two weeks after Apple ships it:

```yaml
controls:
  macos_updates:
    minimum_version: "latest"
    deadline_days: 14
```

The same `ios_updates` and `ipados_updates` keys apply to iPhones and iPads. Because this lives in your Fleet GitOps YAML, updating the deadline for the next release is a reviewable pull request, not a setting someone has to remember to change in a console.

## The rollout isn't finished when Apple ships it

A record patch count is only good news for the devices that actually installed it. Hardware compatibility, deferred update prompts, and uneven MDM enforcement mean a meaningful share of any fleet is still running last year's OS weeks after a release like this one. Knowing the exact build on every device, and having a deadline in place to close the gap, is what turns "Apple shipped a fix" into "our fleet has it."

## See it live

- **Get a demo** to see OS version compliance across your own fleet: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Read the guide** on enforcing OS updates in Fleet: [fleetdm.com/guides/enforce-os-updates](https://fleetdm.com/guides/enforce-os-updates)

## Sources

- SecurityWeek, [Apple Patches 200 Vulnerabilities With New iOS 27, macOS Golden Gate 27 Releases](https://www.securityweek.com/apple-patches-200-vulnerabilities-with-new-ios-27-macos-golden-gate-27-releases/).
- 9to5Mac, [macOS 27 Golden Gate, macOS Tahoe 26.7, and macOS Sequoia 15.8 fix 200+ vulnerabilities](https://9to5mac.com/2026/09/14/macos-27-golden-gate-macos-tahoe-26-7-and-macos-sequoia-15-8-fix-200-vulnerabilities/).
- MacRumors, [iOS 27 and iOS 26.7 Fix More Than 100 Security Bugs](https://www.macrumors.com/2026/09/14/ios-27-bug-fixes/).

<meta name="articleTitle" value="iOS 27 fixes 126 flaws, macOS 27 Golden Gate fixes 210, and most fleets can't say who's updated">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-09-15">
<meta name="description" value="Apple patched 126 iOS and 210 macOS flaws in one release. See how to confirm which Macs, iPhones, and iPads actually installed the fix.">
