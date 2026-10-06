# Conditional access: Duo

With Fleet's [conditional access](https://fleetdm.com/guides/conditional-access), Duo blocks sign-in from hosts that aren't managed by Fleet. It works on macOS, Windows, and Linux.

How it works:

1. Duo Desktop on each host reports the host's device ID to Duo at sign-in.
2. Every few minutes, a script exports the device IDs of your Fleet hosts, and Duo's sync script uploads them to Duo.
3. Duo's [Trusted Endpoints](https://duo.com/docs/trusted-endpoints-generic-duo-desktop) policy blocks sign-in from hosts that aren't on the list.

Unlike [PingFederate](https://fleetdm.com/guides/pingfederate-conditional-access-integration), Duo doesn't need a certificate on the host.

> **Note:** Blocking hosts that are failing critical Fleet policies with Duo is coming soon. Fleet will add an activity when a host refetches ([macOS, iOS, and iPadOS](https://github.com/fleetdm/fleet/issues/50576), [Windows](https://github.com/fleetdm/fleet/issues/50577), [Linux](https://github.com/fleetdm/fleet/issues/50578)), so the sync can run as soon as an end user fixes an issue and selects **Refetch**. Until then, the best practice is to trust every host in Fleet. If you also block failing hosts today, an end user who fixes the issue can't sign in until the next sync, up to 5 minutes later. To block failing hosts anyway, [mark those policies as critical](https://fleetdm.com/guides/pingfederate-conditional-access-integration#step-2-mark-critical-policies) and set `REQUIRE_PASSING_CRITICAL_POLICIES=true` when you run the export script.

## Prerequisites

- A Duo Essentials, Advantage, or Premier plan, and an Owner, Administrator, or Application Manager role in the Duo Admin Panel.
- A computer or CI job that can reach Fleet and Duo, with `curl`, `jq`, Python 3.9 or later, and [`duo-client`](https://pypi.org/project/duo-client/) 4.3.0 or later.

## Step 1: Create a Fleet API user

Create an [API-only user](https://fleetdm.com/guides/fleetctl#create-api-only-user) with the **Observer** role. The export script uses its API token.

## Step 2: Install Duo Desktop

1. In Fleet, head to **Software** and select **Add software**. For macOS and Windows, add **Duo Desktop** from **Fleet-maintained**. For Linux, add Duo's [Duo Desktop package](https://duo.com/docs/duo-desktop) as a **Custom package**.
2. Install it on every host with a [policy automation](https://fleetdm.com/guides/automatic-software-install-in-fleet).

## Step 3: Collect Windows device IDs

Duo identifies Windows hosts by `MachineGuid`, not the UUID Fleet stores. Collect it with a report:

1. Head to **Reports** and select **Add report**. Use this query, target **Windows**, and select **Save**:

```sql
SELECT data AS machine_guid FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Cryptography\MachineGuid';
```

2. Note the report's ID in its URL.

## Step 4: Add Duo integrations

1. In the Duo Admin Panel, head to **Devices > Trusted Endpoints** and select **Add Integration**.
2. Select **Generic Integrations**, choose **macOS**, and select **Add**. Repeat for **Windows** and **Linux**.
3. On each integration's page, download Duo's `device_cache_sync.py`.

## Step 5: Sync hosts

1. Download [`export-fleet-hosts-for-duo.sh`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/api-scripts/export-fleet-hosts-for-duo.sh) and set your Fleet URL and the report ID from Step 3.
2. Run it with `FLEET_API_TOKEN` set to the token from Step 1. It writes `macos.csv`, `windows.csv`, and `linux.csv`.
3. Upload each file with the matching integration's sync script, for example `python device_cache_sync.py --infile macos.csv`.
4. Schedule these steps to run every 5 minutes. The best practice is a [GitHub Actions](https://docs.github.com/en/actions/writing-workflows/choosing-when-your-workflow-runs/events-that-trigger-workflows#schedule) workflow in your GitOps repository, with `FLEET_API_TOKEN` stored as a repository secret:

```yaml
on:
  schedule:
    - cron: "2-59/5 * * * *" # Every 5 minutes, avoiding the top of the hour when GitHub is busiest
  workflow_dispatch:
jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: "3.12"
      - run: pip install "duo-client>=4.3.0"
      - run: |
          ./export-fleet-hosts-for-duo.sh
          for os in macos windows linux; do python duo/$os/device_cache_sync.py --infile $os.csv; done
        env:
          FLEET_API_TOKEN: ${{ secrets.FLEET_API_TOKEN }}
```

Each sync replaces the previous list, so a newly enrolled host can sign in after the next sync.

## Step 6: Turn on the policy

1. On each integration's page, under **Change Integration Status**, activate the integration for a test group.
2. In your Duo policy, set **Trusted Endpoints** to block hosts that aren't trusted.
3. After testing, activate each integration for all users.

## Step 7: Test

| Host | Expected result |
|---|---|
| In Fleet | Sign-in succeeds |
| Not in Fleet | Sign-in denied |

## Troubleshooting

- **Sign-in denied for a managed host**: Check that Duo Desktop is installed and running, and that the last sync included the host.
- **Windows host missing from `windows.csv`**: Check that the Step 3 report has results for the host.

<meta name="articleTitle" value="Conditional access: Duo">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-09-29">
<meta name="description" value="Use Duo for conditional access, blocking sign-in from hosts that aren't managed by Fleet.">
