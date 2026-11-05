# Measure macOS time-to-patch with Fleet

Time-to-patch is how many days it takes your Macs to install a security update after Apple releases it. It's one of the clearest ways to show how well your patching works. Fleet doesn't keep a history of version changes, but every Mac keeps its own: macOS logs each update it installs, with a timestamp, in `/Library/Receipts/InstallHistory.plist`. This guide shows you how to collect that log with a Fleet report and compare it with Apple's release dates to get median time-to-patch, p90, and the share of patches installed within your SLA.

Because the log lives on the Mac, you get history from before the Mac enrolled in Fleet, too. That lets you compare patching before and after a move to Fleet with one set of data.


## How it works

1. A Fleet report reads each Mac's install log with osquery's `package_install_history` table. It returns one row per Mac with every macOS update the Mac installed and when.
2. The [SOFA](https://sofa.macadmins.io) feed, maintained by the Mac Admins community, provides the release date of every macOS security release and the CVEs it fixes.
3. A Python script joins the two. For each Mac and each security release that applied to it:

   > time-to-patch = first install of a version that includes the fix − Apple's release date


## Prerequisites

- Fleet with fleetd on your Macs. `package_install_history` is a macOS table in osquery, which ships with fleetd.
- An [API-only user](https://fleetdm.com/guides/fleetctl#create-api-only-user) with the observer role on the fleets with your Macs. Observers can read report data and the host list.
- Python 3.9 or later on the computer that runs the script. The script uses only Python's standard library.
- Network access to your Fleet server and to `sofafeed.macadmins.io` from that computer.


## Step 1: Add the report

Add the report in [`macos-patch-history.reports.yml`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/macos/reports/macos-patch-history.reports.yml) to each fleet that has Macs.

With GitOps, copy the file into your repository and reference it under `reports:` in each fleet's YAML:

```yaml
reports:
  - path: ../platforms/macos/reports/macos-patch-history.reports.yml
```

To use the Fleet UI instead, go to **Reports**, select the fleet, and select **Add report**. Paste the query below, select **Save report**, and set **Interval** to **Every 6 hours**.

```sql
SELECT
  (SELECT version FROM os_version) AS current_version,
  (SELECT MIN(time) FROM package_install_history) AS history_start,
  (SELECT group_concat(version || '@' || time, ',')
     FROM package_install_history
     WHERE source = 'softwareupdated'
       AND name LIKE 'macOS %'
       AND package_id = '') AS os_installs;
```

The report returns three columns:

| Column | What it means |
|---|---|
| `current_version` | The macOS version the Mac runs now. |
| `history_start` | The oldest entry in the Mac's install log, as a Unix timestamp. Releases before this date can't be measured for that Mac. |
| `os_installs` | Every macOS update the Mac installed, as `version@timestamp`, separated by commas. |

The filter keeps only macOS updates. Background updates, like XProtect, Gatekeeper data, and Rosetta, also come from `softwareupdated`, but they have a package ID, so they're left out.

The install times come from the Mac's log, not from when the report ran, so the report doesn't need to run often. Every 6 hours gets results from laptops that sleep part of the day without running the query more than needed. Because the report returns one row per host, Fleet never clips it, even with more hosts than the [report cap](https://fleetdm.com/docs/configuration/yaml-files#server-settings).


## Step 2: Wait for results

A report's interval counts how long fleetd has been running on the Mac, not time on the clock. While a Mac is asleep or turned off, the count stops. A Mac that's awake all day sends results within about 6 hours. A laptop that sleeps for much of the day can take a day or more.

Fleet keeps each Mac's latest results, so once a Mac has reported, it stays in your numbers even while it's asleep or offline.

To check, open the report in the Fleet UI and look at the number of hosts with results. Note the report's ID from the URL. If you added the report to several fleets, each fleet has its own report and ID.


## Step 3: Run the script

Download [`macos_time_to_patch.py`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/api-scripts/macos_time_to_patch.py). Then set your Fleet server URL and the API-only user's token, and pass each report ID with `--report-id`:

```bash
export FLEET_URL=https://fleet.example.com
export FLEET_API_TOKEN=<api_only_user_token>
python3 macos_time_to_patch.py --report-id 123 --report-id 456 --sla-days 14 --out-dir out
```

The script reads the reports and the host list from the [REST API](https://fleetdm.com/docs/rest-api/rest-api#get-report-data) and downloads the SOFA feed. It prints a summary and writes two files to `--out-dir`:

- `time_to_patch_by_host.csv`: one row per Mac per security release, for drill-down or a BI tool.
- `time_to_patch_summary.json`: the summary, plus a breakdown by month of Apple's release.

Useful options:

| Option | Default | What it does |
|---|---|---|
| `--sla-days` | `14` | Sets the patch SLA used for "within SLA." |
| `--days` | `365` | Sets how far back to look for Apple releases. |
| `--fleet` | All fleets | Only includes Macs in the fleet with this name. |
| `--cutover-date` | Each Mac's enrollment date | Sets one date, as `YYYY-MM-DD`, that splits results into before and after Fleet. |
| `--include-zero-cve` | Off | Counts releases with no published CVEs. |
| `--report-file`, `--hosts-file`, `--sofa-file` | None | Reads saved JSON instead of calling the APIs. |


## Step 4: Read the results

The summary looks like this:

```
macOS time-to-patch: 2 hosts, 28 security releases, SLA 14 days

| Scope | Measured | Median | p90 | Within SLA | Still exposed (past SLA) | No history |
|---|---|---|---|---|---|---|
| All | 16 | 8.7 | 9.7 | 93.8% | 0 (0) | 4 |
| Fleet | 15 | 8.7 | 9.7 | 93.3% | 0 (0) | 0 |
| pre-Fleet | 1 | 8.8 | 8.8 | 100.0% | 0 (0) | 4 |
```

| Column | What it means |
|---|---|
| Measured | The number of Mac and release pairs where the Mac installed the fix. |
| Median | The typical time-to-patch, in days. Half the patches were faster, and half were slower. |
| p90 | The time, in days, within which 90% of patches were installed. A p90 close to the median means patching is consistent. A p90 far above it means a few Macs patch much later than the rest. |
| Within SLA | The share of patches installed within `--sla-days`. Macs that are still exposed past the SLA count against it. |
| Still exposed | The number of releases a Mac hasn't installed yet. The number in parentheses is how many of those are past the SLA. |
| No history | The number of releases that can't be measured, because the Mac's install log starts after the release. |

Report the median, p90, and the share within SLA together. The median shows how fast a typical Mac patches, p90 shows how far behind the slowest Macs are, and the share within SLA shows how many meet your target.

Each row in the CSV has a `status`:

| Status | What it means |
|---|---|
| `patched` | The Mac installed the fix. `days` is the time-to-patch. |
| `exposed` | The Mac hasn't installed the fix. `days` is how long it's been exposed so far. |
| `patched_untimed` | The Mac runs a version with the fix, but its log has no dated install after the release. This usually means the Mac ran a release candidate that became the final build. |
| `no_history` | The Mac's install log starts after the release, usually because the Mac was erased or reinstalled. |


## How the script decides what counts

- Only security releases count. By default, releases with no published CVEs in SOFA are skipped.
- Betas don't count as patches. The install log doesn't record build numbers, so a beta of macOS 27.0 looks the same as the final release. The script ignores any install made before Apple's public release date. Betas still tell the script which major version the Mac moved to.
- Each Mac is measured against releases for the major version it ran. Apple ships fixes for the current and previous major versions on the same day, so a Mac on macOS 27 isn't measured against macOS 26 updates. Upgrading to a new major version counts as patching an older-version release only when the new version came out on the same day or later.
- Apple's release date decides whether a release counts as before or after Fleet. The script compares it with the Mac's `last_enrolled_at` in Fleet, or with `--cutover-date`. Re-enrolling a Mac resets `last_enrolled_at`, so use `--cutover-date` if your Macs have re-enrolled.


## Limitations

- This guide measures macOS updates only. Apps that update themselves, like Chrome and Zoom, don't write to the install log. To track app updates, use Fleet's [software inventory and vulnerability detection](https://fleetdm.com/guides/vulnerability-processing).
- A Mac that was erased or reinstalled only has history since then. Check the `no_history` count to see how much data is missing.
- Rapid Security Responses and Background Security Improvements haven't been tested with this report.
- When Apple stops shipping security updates for a major version, Macs that still run it never show as exposed. Track those with a policy that checks the macOS version.


## Troubleshooting

| Problem | What to check |
|---|---|
| The script finds 0 hosts. | The report doesn't have results yet. Wait for Macs to run it, and check that the report targets macOS and that **Interval** isn't set to **Never**. |
| A Mac is missing from the results. | The Mac hasn't run the report since you added it. Report intervals only count time when the Mac is awake, so a Mac that's often asleep or turned off reports later. |
| Every release for a Mac shows `no_history`. | The Mac's install log is newer than the releases, or the log is empty. Check `history_start` in the report. |


## Further reading

- [Reports in Fleet](https://fleetdm.com/guides/reports)
- [SOFA: Simple Organized Feed for Apple software updates](https://sofa.macadmins.io)
- [`package_install_history` table](https://fleetdm.com/tables/package_install_history)
- [REST API: Get report data](https://fleetdm.com/docs/rest-api/rest-api#get-report-data)

<meta name="articleTitle" value="Measure macOS time-to-patch with Fleet">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-11-05">
<meta name="description" value="Measure how many days your Macs take to install Apple security updates, with a Fleet report and Apple release dates from SOFA.">
