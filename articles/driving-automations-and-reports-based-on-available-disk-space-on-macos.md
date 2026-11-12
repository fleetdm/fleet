# Driving automations and reports based on available disk space on macOS

A customer's IT team built a disk space query in Fleet and noticed the numbers didn't match what macOS reports. They were right. The `mounts` table doesn't include purgeable space, so it consistently underreports what's available. That gap matters most in two situations: a Mac that won't download or install a software update, and a user who's convinced their disk isn't as full as your dashboard says. The fix for both is a one-line change: swap `mounts` for `disk_space`.

## The problem

macOS treats purgeable space (caches, Time Machine local snapshots, and other reclaimable data) as available. Open Finder, hit **Get Info** on the startup disk, and that purgeable space is included in the number macOS shows. The `mounts` table doesn't do this. It reports raw file system blocks: total, free, used. On Linux and Windows, that's the full picture. On macOS, it's not.

The customer's IT team spotted the mismatch immediately:

> "The query doesn't seem to include rewritable disk space."

They asked the right question. macOS actively manages a layer of purgeable storage that sits between "used" and "truly free", and the OS will reclaim it on demand when an app or installer needs room. From the user's perspective, that space is available. From the file system's perspective, it's already allocated. A disk can read as 90% full in `mounts` while macOS tells the user they have 50 GB free. Both numbers are technically correct, but only one matches what the user experiences, and only one matches what the macOS installer will use.

## Why this is relevant for update compliance

This is where the gap stops being a rounding error and starts causing real problems. macOS won't download or install an update without enough free space to stage it, and Apple doesn't publish hard numbers, but real-world testing puts minor updates at requiring roughly 15 GB free and major upgrades at roughly 35 GB (the installer itself plus working space for the upgrade process). If a host is short on either, the update silently fails to download, gets stuck partway through, or the user dismisses the "not enough space" prompt and moves on.

If you're troubleshooting why a fleet of Macs is behind on updates, checking `mounts` can point you in the wrong direction. It's the same undercounting issue: a host might have 30 GB of purgeable cache sitting there, macOS would reclaim it automatically during the update, but `mounts` reports that space as used. You'd have to dig into the host manually to find out it has plenty of room. `disk_space` gives you the number that matches what the installer sees, so a host failing a space check is genuinely short on space, not a false alarm from stale cache data.

## Prerequisites

- A Fleet instance with macOS hosts enrolled.
- Permission to run live queries and create policies in Fleet.

## Use the disk_space table

The [`disk_space`](https://fleetdm.com/tables/disk_space) table uses Apple's `NSURLVolumeAvailableCapacityForImportantUsageKey` API, the same API macOS uses to calculate what it shows users. It includes purgeable space.

Run this query on your macOS hosts:

```sql
SELECT bytes_available FROM disk_space;
```

That's it. The `bytes_available` column gives you available disk capacity including purgeable space, matching what your users see in Finder and what the macOS installer will have to work with.

> **Note:** `disk_space` is macOS-only. For cross-platform reporting, you'll still need `mounts` on Linux and Windows, where purgeable space isn't a factor.

## Build automations and reports

Once your query reflects reality, you can use it for update troubleshooting, not only for general alerting.

### Flag hosts that can't take the next update

Set separate thresholds for minor updates and major upgrades, since they need different amounts of headroom. Here's a policy for minor updates:

```yaml
- name: Sufficient disk space for macOS minor update
  query: SELECT 1 FROM disk_space WHERE bytes_available >= 15000000000;
  critical: false
  description: >-
    This policy checks whether a host has at least 15GB of available disk space, including purgeable space,
    which is roughly what macOS needs to download and install a minor update.
    Hosts that fail this policy are likely stuck on an older version because the update can't stage,
    not because anyone is ignoring the update prompt.
  resolution: |-
    Free up disk space by removing unnecessary files, emptying the Trash, or uninstalling unused applications.
    Once the host has enough free space, macOS should be able to download and install the pending update.

    If the issue persists, please reach out to support.
  platform: darwin
  webhooks_and_tickets_enabled: true
```

For major upgrades, raise the threshold to roughly 35 GB and adjust the name and description accordingly. Running both policies side by side tells you at a glance whether a host is behind because of disk space or for some other reason, which saves time when you're chasing down compliance gaps.

### Build capacity reports

Schedule `SELECT bytes_available FROM disk_space;` as a report in Fleet. Export the results to build dashboards or feed them into your ITSM tool. The numbers will match what your users report, which means fewer "but my Mac says I have space" tickets landing in your queue.

### Automate clean-up workflows

Use Fleet's webhook integrations to trigger action when available space crosses a boundary, before it becomes an update failure. Prompt the user to clean up, open a ticket automatically, or kick off a remediation script that clears known cache locations.

## Further reading

- [`disk_space` table documentation](https://fleetdm.com/tables/disk_space)
- [Fleet queries documentation](https://fleetdm.com/docs/using-fleet/fleet-ui#queries)

<meta name="articleTitle" value="Driving automations and reports based on available disk space on macOS">
<meta name="authorFullName" value="Gray Williams">
<meta name="authorGitHubUsername" value="GrayW">
<meta name="publishedOn" value="2026-08-18">
<meta name="category" value="guides">
<meta name="description" value="How to correctly measure available disk space on macOS in Fleet, and build policies and reports that catch real update blockers.">
