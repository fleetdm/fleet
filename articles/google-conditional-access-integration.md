# Conditional access: Google

With Fleet's [conditional access](https://fleetdm.com/guides/conditional-access), Google blocks sign-in from iPhones and iPads that aren't managed by Fleet. Google calls this Context-Aware Access.

How it works:

1. When an end user signs in to a Google app on their iPhone or iPad, Google's basic mobile management adds the device to Google.
2. Every few minutes, a script tells Google which of these devices are managed by Fleet.
3. A Context-Aware Access level blocks sign-in from devices that aren't.

> **Note:** Built-in support for Google is [coming soon](https://github.com/fleetdm/fleet/issues/54888). Until then, the best practice is to run the sync script on a schedule.

## Prerequisites

- Google Workspace Enterprise Standard, Enterprise Plus, or Enterprise for Education, or Cloud Identity Premium, and a super admin account.
- Google's basic mobile management for iOS turned on (the default).
- iPhones and iPads enrolled to Fleet with MDM turned on, with the end user's email in Fleet, for example from [end user authentication](https://fleetdm.com/guides/setup-experience#end-user-authentication).
- A computer or CI job that can reach Fleet and Google, with `curl` and `jq`.

## Step 1: Create a Fleet API user

Create an [API-only user](https://fleetdm.com/guides/fleetctl#create-api-only-user) with the **Observer** role. The sync script uses its API token.

Select **Specific API endpoints** and add only the endpoint the script calls: [List hosts](https://fleetdm.com/docs/rest-api/rest-api#list-hosts).

## Step 2: Create a Google service account

1. In the [Google Cloud console](https://console.cloud.google.com/), create a project and enable the **Cloud Identity API**.
2. Create a service account and add a JSON key.
3. In the Google Admin console, head to **Security > Access and data control > API controls > Domain-wide delegation** and select **Add new**. Enter the service account's client ID and the `https://www.googleapis.com/auth/cloud-identity.devices` scope.
4. Choose an admin account for the service account to act as. The best practice is a dedicated admin account that can only manage mobile devices.

## Step 3: Sync hosts

1. In the Google Admin console, head to **Account > Account settings** and copy your **Customer ID**.
2. Download [`sync-fleet-hosts-to-google.sh`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/api-scripts/sync-fleet-hosts-to-google.sh) and set your Fleet URL and customer ID.
3. Run it with `DRY_RUN=true` to see the changes it would make. Set `FLEET_API_TOKEN` to the token from Step 1 and `GOOGLE_ACCESS_TOKEN` to a token for the service account from Step 2.
4. Schedule it to run every 5 minutes. The best practice is a [GitHub Actions](https://docs.github.com/en/actions/writing-workflows/choosing-when-your-workflow-runs/events-that-trigger-workflows#schedule) workflow in your GitOps repository, with `FLEET_API_TOKEN` and the service account's JSON key (`GOOGLE_CREDENTIALS`) stored as repository secrets:

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
      - id: google
        uses: google-github-actions/auth@v2
        with:
          credentials_json: ${{ secrets.GOOGLE_CREDENTIALS }}
          token_format: access_token
          access_token_scopes: https://www.googleapis.com/auth/cloud-identity.devices
          access_token_subject: admin@example.com # The admin account from Step 2
      - run: ./sync-fleet-hosts-to-google.sh
        env:
          FLEET_API_TOKEN: ${{ secrets.FLEET_API_TOKEN }}
          GOOGLE_ACCESS_TOKEN: ${{ steps.google.outputs.access_token }}
```

Google doesn't record serial numbers for iPhones and iPads, so the script matches devices to Fleet hosts by end user email and device type (iPhone or iPad). If an end user has more than one iPhone (or iPad) in Google or Fleet, the script can't tell them apart. It doesn't mark them as managed, and prints them for review.

When a host is removed from Fleet, the next sync marks its device as unmanaged.

## Step 4: Turn on the access level

1. In the Google Admin console, head to **Security > Access and data control > Context-Aware Access** and select **Access levels > Create access level**.
2. Select **Advanced**, and enter this condition, replacing `<customer-ID>` with your customer ID:

```
device.vendors["<customer-ID>-fleet"].is_managed_device == true
```

3. Select **Assign access levels**, choose a test organizational unit (OU), and assign the access level to the apps you want to protect, like Gmail and Drive. Select **Apply to Google desktop and mobile apps**.
4. After testing, assign the access level to all OUs.

## Step 5: Test

| Host | Expected result |
|---|---|
| iPhone or iPad in Fleet | Sign-in succeeds |
| iPhone or iPad not in Fleet | Sign-in denied |

## Troubleshooting

- **Sign-in denied for a managed host**: Check that the host has the end user's email in Fleet, and that the last sync didn't list the host for review.
- **Device missing from Google**: Check that the end user signed in to a Google app on the device, and that the device appears in the Google Admin console under **Devices > Mobile & endpoints**.

<meta name="articleTitle" value="Conditional access: Google">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-10-08">
<meta name="description" value="Use Google for conditional access, blocking sign-in from iPhones and iPads that aren't managed by Fleet.">
