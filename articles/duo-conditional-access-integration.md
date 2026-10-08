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
- For Linux, x86-64 hosts. Duo Desktop doesn't support Linux on ARM.
- A computer or CI job that can reach Fleet and Duo, with `curl`, `jq`, Python 3.9 or later, and [`duo-client`](https://pypi.org/project/duo-client/) 4.3.0 or later.

## Step 1: Create a Fleet API user

Create an [API-only user](https://fleetdm.com/guides/fleetctl#create-api-only-user) with the **Observer** role. The export script uses its API token.

Select **Specific API endpoints** and add only the endpoints the script calls:

- [List hosts](https://fleetdm.com/docs/rest-api/rest-api#list-hosts)
- [Get report data](https://fleetdm.com/docs/rest-api/rest-api#get-report-data) (for the Windows report in Step 3)

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

1. Download [`export-fleet-hosts-for-duo.sh`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/api-scripts/export-fleet-hosts-for-duo.sh), set your Fleet URL and the report ID from Step 3, and save it in your repository as `duo/export-fleet-hosts-for-duo.sh`.
2. Run it with `FLEET_API_TOKEN` set to the token from Step 1. It writes `macos.csv`, `windows.csv`, and `linux.csv`.
3. Upload each file with the matching integration's sync script, for example `python device_cache_sync.py --infile macos.csv`. Duo's script refuses a file with no device IDs, so skip an operating system that has no hosts. Add `--dry_run` the first time to check the file without changing what Duo holds.
4. Schedule these steps to run every 5 minutes. The best practice is a [GitHub Actions](https://docs.github.com/en/actions/writing-workflows/choosing-when-your-workflow-runs/events-that-trigger-workflows#schedule) workflow. It can live in any repository. It doesn't need to be your GitOps repository.

> **Note:** GitHub runs scheduled workflows on a best-effort basis. A run can start late or be skipped when GitHub is busy. In testing, a 5-minute schedule started a run only twice in 11 hours. For a steady 5-minute sync, run the same steps from a server with cron or a systemd timer. A run every 5 minutes also uses about 8,600 minutes a month, more than the 2,000 free minutes of a private repository.

Duo's script has the integration's keys written into it, so don't commit it as is. Edit it to read the keys from environment variables instead. Replace the `MKEY_CREDENTIALS = {...}` block near the top with this:

```python
import os

MKEY_CREDENTIALS = {
    os.environ['DUO_MKEY']: {
        'API_IKEY': os.environ['DUO_IKEY'],
        'API_SKEY': os.environ['DUO_SKEY'],
        'API_HOST': os.environ['DUO_API_HOST'],
    }
}
```

The edited file holds no secrets, so you can save it in your repository as `duo/device_cache_sync.py`. Use that one file for all three integrations.

In your repository's **Settings > Secrets and variables > Actions**, add these secrets. Find the values in the `MKEY_CREDENTIALS` block of each script you downloaded in Step 4:

- `DUO_FLEET_API_TOKEN`: the token from Step 1. Use this name, not `FLEET_API_TOKEN`. If your repository is a GitOps repository, it already uses `FLEET_API_TOKEN` for a token with different permissions, and replacing it breaks your GitOps runs.
- `DUO_API_HOST`: your Duo API hostname.
- For each of `MACOS`, `WINDOWS`, and `LINUX`: `DUO_<OS>_MKEY` (the integration ID, which starts with `DM`), `DUO_<OS>_IKEY`, and `DUO_<OS>_SKEY`.

Then add this workflow, for example as `.github/workflows/duo-sync.yml`:

```yaml
name: Sync Fleet hosts to Duo Trusted Endpoints

on:
  schedule:
    - cron: "2-59/5 * * * *" # Every 5 minutes, avoiding the top of the hour when GitHub is busiest
  workflow_dispatch:

concurrency:
  group: ${{ github.workflow }}
  cancel-in-progress: false

permissions:
  contents: read

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: "3.12"
      - run: pip install "duo-client>=4.3.0"
      - name: Export Fleet hosts and sync to Duo
        env:
          FLEET_API_TOKEN: ${{ secrets.DUO_FLEET_API_TOKEN }}
          DUO_API_HOST: ${{ secrets.DUO_API_HOST }}
          DUO_MACOS_MKEY: ${{ secrets.DUO_MACOS_MKEY }}
          DUO_MACOS_IKEY: ${{ secrets.DUO_MACOS_IKEY }}
          DUO_MACOS_SKEY: ${{ secrets.DUO_MACOS_SKEY }}
          DUO_WINDOWS_MKEY: ${{ secrets.DUO_WINDOWS_MKEY }}
          DUO_WINDOWS_IKEY: ${{ secrets.DUO_WINDOWS_IKEY }}
          DUO_WINDOWS_SKEY: ${{ secrets.DUO_WINDOWS_SKEY }}
          DUO_LINUX_MKEY: ${{ secrets.DUO_LINUX_MKEY }}
          DUO_LINUX_IKEY: ${{ secrets.DUO_LINUX_IKEY }}
          DUO_LINUX_SKEY: ${{ secrets.DUO_LINUX_SKEY }}
        run: |
          bash duo/export-fleet-hosts-for-duo.sh
          for os in macos windows linux; do
            up=$(echo "$os" | tr a-z A-Z)
            mkey="DUO_${up}_MKEY"; ikey="DUO_${up}_IKEY"; skey="DUO_${up}_SKEY"
            if [ -n "${!mkey}" ] && [ "$(wc -l <"$os.csv")" -gt 1 ]; then
              DUO_MKEY="${!mkey}" DUO_IKEY="${!ikey}" DUO_SKEY="${!skey}" \
                python duo/device_cache_sync.py --infile "$os.csv" --device_id_column device_id
            else
              echo "$os: no credentials or no hosts, skipped"
            fi
          done
```

No GitHub? Run the same export and upload steps on a schedule anywhere that can reach Fleet and Duo, like cron, a systemd timer, or Windows Task Scheduler. Set the same environment variables in the job. On macOS, don't keep the scripts in Downloads, Documents, or Desktop. macOS blocks scheduled jobs from reading those folders.

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
- **macOS asks for local network access**: Duo Desktop talks to the browser over the local network. When Duo shows "Duo Desktop needs permission to access your local network", allow the browser's prompt. If you select **Skip for now**, Duo can't read the device ID and blocks the host. In **System Settings > Privacy & Security > Local Network**, make sure the browser is on.
- **Windows host missing from `windows.csv`**: Check that the Step 3 report has results for the host.

<meta name="articleTitle" value="Conditional access: Duo">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-09-29">
<meta name="description" value="Use Duo for conditional access, blocking sign-in from hosts that aren't managed by Fleet.">
