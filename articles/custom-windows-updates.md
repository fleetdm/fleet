# Managing Windows updates with Fleet and the Windows Update CSP

Keeping Windows devices patched is one of the most important things an admin is responsible for, but also one of the easiest to get wrong. Push too hard and you're rebooting laptops mid-presentation. Go too soft and you're staring at a fleet full of devices three cumulative updates behind.

Windows exposes all of its update behavior through the [Update Policy CSP](https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-csp-update), a comprehensive set of controls that determine what updates devices get, when they install, and how much say the end user has in the process. Fleet can TODO TODO TODO

This guide covers the policies worth knowing, the ones worth skipping, and how to deploy and verify them with Fleet.

There are a few different types of updates in Windows that are important to know. Let's take a look at each of them and what they do.

**Feature updates:** These are released annually and contain new features and functionality. For example, **26H1** became generally available on February 10, 2026. Microsoft states 36 months of support for Enterprise and Education editions.

**Quality updates:** These updates deliver both security and non-security fixes, including security updates, critical updates, servicing stack updates, and driver updates. They're typically released on the second Tuesday of each month, though they can be released at any time. The second-Tuesday releases (the infamous "Patch Tuesday") are the ones that primarily focus on security updates.

It's important to note that quality updates are *cumulative*, so installing the latest quality update is sufficient to get all the available fixes for a specific feature update.

**Driver updates:** The mechanism behind updating device drivers. Admins have full control over whether these are installed or not.

**Microsoft product updates:** These update other Microsoft products, such as Office. You can enable or disable Microsoft updates by using policies controlled by various servicing tools.

**Servicing stack updates:** The servicing stack is the code component that actually installs Windows updates. Occasionally, the servicing stack itself needs to be updated in order to function smoothly. If you don't install the latest servicing stack update, there's a risk that your device can't be updated with the latest Microsoft security fixes.

## Start with Fleet's built-in enforcement

Before writing any custom profiles, know that Fleet handles the core case out of the box. In **Controls > OS updates**, you can set a **deadline** and **grace period** for Windows hosts on each team. Under the hood, this uses the same deadline policies described in the next section, so most teams never need to touch them directly.

Custom settings come in when you want to go beyond enforcement: rollout rings, pinning a release, patching Office, or locking down the end-user experience. That's what the rest of this guide is for.

## Deadlines vs. grace periods

Microsoft has gone through several generations of update enforcement (you'll find the fossil record in the "Legacy Policies" section of the docs). The current, recommended model is **deadline-driven**, built on four policies:

| Policy | Range | Default |
| --- | --- | --- |
| `ConfigureDeadlineForQualityUpdates` | 0–30 days | 7 |
| `ConfigureDeadlineForFeatureUpdates` | 0–30 days | 2 |
| `ConfigureDeadlineGracePeriod` | 0–7 days | 2 |
| `ConfigureDeadlineGracePeriodForFeatureUpdates` | 0–7 days | 7 |

With deadline policies configured, the download and install happen automatically as soon as the update is offered. The two knobs ann admin can turn are **deadline** and **grace period.**

Deadline: number of days from when the update is offered until the restart is forced (regardless of active hours, user can't reschedule).

Grace period: minimum days from when the update installs until an automatic restart can happen. It exists to protect users who were offline for a while: if a device is off for two weeks and installs an update past its deadline, the grace period still guarantees the user a couple of days to save work before the forced reboot.

The effective forced-restart moment is whichever comes later: deadline (from offer) or grace period (from install).

On Windows 11 22H2 and later, two companion policies — `ConfigureDeadlineNoAutoRebootForQualityUpdates` and `ConfigureDeadlineNoAutoRebootForFeatureUpdates` — tell the device not to attempt any automatic restart until both the deadline and grace period have expired. Users get maximum runway, but the backstop still lands.

> **Gotcha:** when deadline policies are configured, the download, install, and reboot behavior from `AllowAutoUpdate` is ignored. If you've inherited old profiles that set `AllowAutoUpdate`, the deadline policies win.

## Build rollout rings with deferrals

A deadline says "install within N days of being offered." Deferral policies control *when the update is offered in the first place*, which is how you build rollout rings without any extra tooling:

- `DeferQualityUpdatesPeriodInDays` — delay monthly quality updates by 0–30 days.
- `DeferFeatureUpdatesPeriodInDays` — delay feature updates (the annual releases) by 0–365 days.

A simple setup in Fleet might look something like this:

| Team | Quality deferral | Feature deferral |
| --- | --- | --- |
| 🧪 Testing + QA | 0 days | 0 days |
| 💻 Workstations | 7 days | 30 days |
| ☁️ IT servers | 14 days | 90 days |

Patch Tuesday lands on your testing ring the same day. If a week passes without anyone's audio driver catching fire, the same update reaches everyone else automatically.

## Pin the release you actually want

Windows Update will eventually offer devices the next feature release. If you'd rather move to Windows 11 26H1 when all the testing is complete and your devices are ready on your schedule, pin it.

- `ProductVersion` — the product to stay on or move to. Supported value type is a string containing a Windows product, for example, "Windows 11" or "11" or "Windows 10".
- `TargetReleaseVersion` — the specific release (e.g., `24H2` or `25H2`).

Devices stay on the pinned release until it reaches end of service or you change the policy. These two must be configured **together** — `TargetReleaseVersion` doesn't work on its own.

One warning: pinning is a commitment. If you pin a release and forget about it, Windows will keep the device there right up to (and past) end of service. Put a reminder on the calendar, or better, write a Fleet policy that flags hosts running a release within 90 days of end of service.

## Pausing updates

When a bad patch ships admins want a brake, not an entire config refactor:

- `PauseQualityUpdatesStartTime` — pauses quality updates for 35 days from the date you set (format: `2026-07-27`).
- `PauseFeatureUpdatesStartTime` — same, for feature updates channel.

Push the profile with today's date to the affected team, and updates stop being offered for 35 days or until you clear it. Because it's just a profile, un-pausing is deleting the profile. {HARRY WANT TO CONFIRM THIS ACTUALLY WORKS}

## Don't forget Office (and think about drivers)

Two policies that punch above their weight:

- `AllowMUUpdateService` — set to `1` and Windows Update also patches other Microsoft products, most notably Office. This is **off by default**, which surprises a lot of teams whose Office installs quietly stopped updating when they left WSUS or ConfigMgr behind. You can see the full list of Microsoft products that are in scope for this policy in the [offical documentation](https://learn.microsoft.com/en-us/windows/deployment/update/update-other-microsoft-products)
- `ExcludeWUDriversInQualityUpdate` — set to `1` to keep driver updates out of quality updates. Whether you want this depends on your hardware fleet; if your vendor ships driver updates through their own tooling, excluding Windows Update drivers avoids the two clashing.

## Managing the end-user experience

The remaining policies control what users see and how much they can interfere:

- `ActiveHoursStart` / `ActiveHoursEnd` / `ActiveHoursMaxRange` — define when automatic restarts won't happen. By default users set their own active hours (up to an 18-hour range); these policies let you set or constrain them. Be aware of the other policies you have configured. If either `AlwaysAutoRebootAtScheduledTimeMinutes` or `NoAutoRebootWithLoggedOnUsers` (a registry key, no CSP available), this policy has no effect.
- `SetDisablePauseUXAccess` — removes the user's ability to hit "Pause updates" in Settings. If you're enforcing a compliance window, this closes the loophole where a user pauses updates for 35 days and sails past your deadline.
- `SetDisableUXWUAccess` — If you enable this setting user access to Windows Update scan, download and install is removed. \
- `UpdateNotificationLevel` - This policy allows you to define what Windows Update notifications users see. This policy doesn't control how and when updates are downloaded and installed. Below maps the values an admin can set and the expected behavior.
-
| Value | Behavior |
| --- | --- |
| 0 (default) | Use the default Windows Update notifications |
| 1 | Turn off all notifications, excluding restart warnings |
| 2 | Turn off all notifications, including restart warnings |


- and `NoUpdateNotificationsDuringActiveHours` - Same values and behavior as the previous key, but can be helpful to dial down update toasts for conference room PCs, digital signage, and other devices where a notification isn't needed or is intrusive. Deadline warnings still appear once the deadline is reached, so enforcement stays visible.

## One to leave alone: safeguard holds

`DisableWUfBSafeguards` deserves a mention only as a warning. Safeguard holds are Microsoft's mechanism for blocking a feature update from devices with a known compatibility issue, for example, a driver that bluescreens on the new release. Setting this policy to `1` bypasses those holds.

Microsoft's own docs recommend using it only for validation in IT environments, and the policy deliberately resets to Not Configured after every feature update so nobody disables safeguards once and forgets (hey, good thinking there, Microsoft). Unless you're actively debugging why a specific device isn't being offered an update, leave it alone. More information on this key can be found [here.](https://learn.microsoft.com/en-us/windows/deployment/update/safeguard-holds). 

## Deploying with Fleet

Here's a profile that implements the workstation ring described above:

```xml
<Replace>
  <!-- Install quality updates within 7 days of being offered -->
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">int</Format>
    </Meta>
    <Target>
      <LocURI>./Device/Vendor/MSFT/Policy/Config/Update/ConfigureDeadlineForQualityUpdates</LocURI>
    </Target>
    <Data>7</Data>
  </Item>
</Replace>
<Replace>
  <!-- Guarantee users 2 days after install before a forced restart -->
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">int</Format>
    </Meta>
    <Target>
      <LocURI>./Device/Vendor/MSFT/Policy/Config/Update/ConfigureDeadlineGracePeriod</LocURI>
    </Target>
    <Data>2</Data>
  </Item>
</Replace>
<Replace>
  <!-- Defer quality updates for 7 days after release -->
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">int</Format>
    </Meta>
    <Target>
      <LocURI>./Device/Vendor/MSFT/Policy/Config/Update/DeferQualityUpdatesPeriodInDays</LocURI>
    </Target>
    <Data>7</Data>
  </Item>
</Replace>
<Replace>
  <!-- Also update other Microsoft products (e.g., Office) -->
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">int</Format>
    </Meta>
    <Target>
      <LocURI>./Device/Vendor/MSFT/Policy/Config/Update/AllowMUUpdateService</LocURI>
    </Target>
    <Data>1</Data>
  </Item>
</Replace>
<Replace>
  <!-- Remove the user's ability to pause updates in Settings -->
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">int</Format>
    </Meta>
    <Target>
      <LocURI>./Device/Vendor/MSFT/Policy/Config/Update/SetDisablePauseUXAccess</LocURI>
    </Target>
    <Data>1</Data>
  </Item>
</Replace>
```

Upload it under **Controls > OS settings > Custom settings** for the fleet you want, or manage it in your GitOps repo.

## Verifying with osquery

Profiles tell you what you *asked for*; osquery tells you what's *actually there*. Because MDM policies land in the registry, you can verify enforcement with the same tool you use for everything else.

One thing to know before you go looking: the values won't be where the Microsoft docs seem to point. Each policy's "Group policy mapping" lists a registry key under `Software\Policies\Microsoft\Windows\WindowsUpdate`, but that's where the *Group Policy* equivalent writes. Update policies delivered over MDM are native Policy CSP settings, so Windows stores them in the PolicyManager hive instead: what each enrollment requested lives under `PolicyManager\providers\<enrollmentGUID>`, and the effective, merged result, the one the Windows Update engine actually reads, lives under `PolicyManager\current`. (Some other CSP areas *do* stamp the classic GP keys; those are ADMX-backed policies, where MDM is essentially puppeting Group Policy for older components.)

`PolicyManager\current` is the source of truth, so that's what we query. This returns every Windows Update policy currently in effect on a host:

```sql
SELECT name, data
FROM registry
WHERE path LIKE 'HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\PolicyManager\current\device\Update\%';
```

And you can turn any individual setting into a Fleet policy. For example, "quality update deadline is 7 days or less":

```sql
SELECT 1 FROM registry
WHERE path = 'HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\PolicyManager\current\device\Update\ConfigureDeadlineForQualityUpdates'
AND CAST(data AS integer) <= 7;
```

## What about the rest of the CSP?

The Update CSP contains roughly 90 policies, and you've now seen the ~20 that matter for most environments. Of the remainder:

- **Legacy Policies** (engaged restart, `DeferUpdatePeriod`, auto-restart deadlines) are earlier generations of the enforcement model, superseded by the deadline policies. If you find them in inherited profiles, migrating to deadlines will simplify your life.
- **The WSUS section** (`UpdateServiceUrl`, scan-source policies) only matters if you're running WSUS. The one interesting corner is the `SetPolicyDrivenUpdateSourceFor*Updates` family, which lets you move update types to Windows Update one at a time, which can be useful for a gradual migration off WSUS or ConfigMgr.
- **The Maintenance Window section** is a newer addition that gives update installs a proper maintenance-window scheduler. Worth watching if you manage servers or kiosks with strict change windows.

## Conclusion

Windows update management doesn't need a dedicated product or a WSUS server. The Update CSP gives you deadlines, rings, release pinning, and an emergency brake, and Fleet gives you the delivery mechanism, fleet-based targeting, and the means to verify that what you configured is what's actually running. 

Set the deadline, build your rings, and let the machines patch themselves.

<meta name="category" value="guides">
<meta name="authorFullName" value="Harrison Ravazzolo">
<meta name="authorGitHubUsername" value="harrisonravazzolo">
<meta name="publishedOn" value="2026-07-27">
<meta name="articleTitle" value="Managing Windows updates with Fleet and the Windows Update CSP">
<meta name="description" value="Use Windows Update CSP policies with Fleet to enforce deadlines, build rollout rings, pin releases, and verify it all with osquery.">
