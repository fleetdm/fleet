# Driving automations and reports based on available disk space on macOS

A customer's IT team built a disk space query in Fleet and noticed the numbers didn't match what macOS reports. They were right. The `mounts` table doesn't include purgeable space, so it consistently underreports available capacity. The fix is a one-line change: swap `mounts` for `disk_space`.

## The problem

macOS treats purgeable space (caches, Time Machine local snapshots, and other reclaimable data) as "available." When a user opens Finder and hits Get Info on their startup disk, macOS includes that purgeable space in the available number. The `mounts` table doesn't. It reports raw filesystem blocks only.

The customer's IT team spotted it immediately:

> "The query doesn't seem to include rewritable disk space."

They asked exactly the right question. The numbers were off because the table they were using was never designed to account for how macOS actually manages storage.

## Prerequisites

- Fleet with one or more macOS hosts enrolled
- Permission to run queries (observer+ or higher)

## Use the disk_space table

The [`disk_space`](https://fleetdm.com/tables/disk_space) table uses Apple's `NSURLVolumeAvailableCapacityForImportantUsageKey` API under the hood. This is the same API macOS uses to calculate what it shows users. It includes purgeable space.

Run this query on your macOS hosts:

```sql
SELECT bytes_available FROM disk_space;
```

That's it. The `bytes_available` column gives you the available disk capacity including purgeable space. It matches what your users see in Finder.

> **Note:** The `disk_space` table is macOS-only. For cross-platform disk reporting, you'll still need `mounts` on Linux and Windows, where purgeable space isn't a factor.

## Why mounts falls short on macOS

The `mounts` table reports what the filesystem layer knows: total blocks, free blocks, used blocks. On Linux and Windows, that's the full picture.

On macOS, it's not. macOS actively manages a layer of purgeable storage that sits between "used" and "truly free." The operating system will reclaim that space on demand when an app or download needs it. From the user's perspective, it's available. From the filesystem's perspective, it's allocated.

This means `mounts` can report a disk as 90% full while macOS tells the user they have 50 GB free. Both are technically correct, but only one matches what the user experiences.

## Build automations and reports

Now that your query returns accurate numbers, you can put them to work:

1. **Disk space alerts.** Create a policy using `disk_space` to flag hosts where `bytes_available` drops below a threshold. Users get notified before they run out of space, and the alert fires at the same threshold they'd notice in Finder.

2. **Capacity reports.** Schedule the query as a report in Fleet. Export the results to build dashboards or feed them into your ITSM tool. The numbers will match what your users report, which means fewer "but my Mac says I have space" tickets.

3. **Automated workflows.** Use Fleet's webhook integrations to trigger actions when available space crosses a boundary. Prompt users to clean up, open a ticket, or kick off a remediation script.

## Further reading

- [`disk_space` table documentation](https://fleetdm.com/tables/disk_space)
- [Fleet queries documentation](https://fleetdm.com/docs/using-fleet/fleet-ui#queries)

<meta name="articleTitle" value="Driving automations and reports based on available disk space on macOS">
<meta name="authorFullName" value="Gray Williams">
<meta name="authorGitHubUsername" value="gray-williams">
<meta name="publishedOn" value="2026-07-28">
<meta name="category" value="guides">
<meta name="description" value="How to get accurate disk space numbers on macOS with Fleet, including purgeable space.">
