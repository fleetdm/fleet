# Conditional access: Google

With Fleet's [conditional access](https://fleetdm.com/guides/conditional-access), Google blocks sign-in from iPhones and iPads that aren't managed by Fleet. Google calls this Context-Aware Access.

How it works:

1. When an end user signs in to a Google app on their iPhone or iPad, Google's basic mobile management adds the device to Google.
2. Every few minutes, a script tells Google which of these devices are managed by Fleet.
3. A Context-Aware Access level blocks sign-in from devices that aren't.

> **Note:** Built-in support for Google is [coming soon](https://github.com/fleetdm/fleet/issues/54888). Until then, the best practice is to run the sync script on a schedule.

> **Known limitation:** Google doesn't record serial numbers for iPhones and iPads, so the script matches Fleet hosts to Google devices by the end user's email and device type. If an end user has more than one iPhone (or iPad) signed in to the same Google account, such as a company-owned iPhone and a personal iPhone, the script can't tell which one Fleet manages. It doesn't mark either as managed, so Google blocks sign-in on both, including the company-owned one. Devices the script can't match are printed for review. If the end user's only iPhone signed in to Google is a personal one, for example before they sign in on their company-owned iPhone, the script marks the personal one as managed.

## Prerequisites

- Google Workspace Enterprise Standard, Enterprise Plus, or Enterprise for Education, or Cloud Identity Premium
- A Google super admin account
- Google's basic mobile management for iOS turned on (the default)
- iPhones and iPads enrolled to Fleet with MDM and [end user authentication](https://fleetdm.com/guides/setup-experience#end-user-authentication) turned on, so Fleet has each end user's identity provider (IdP) email
- A computer or continuous integration (CI) job that can reach Fleet and Google, with `curl`, `jq`, and `openssl`

## Step 1: Create a Fleet API user

Create an [API-only user](https://fleetdm.com/guides/fleetctl#create-api-only-user) with the **Observer** role. The sync script uses its API token.

Select **Specific API endpoints** and add only the endpoint the script calls: [List hosts](https://fleetdm.com/docs/rest-api/rest-api#list-hosts).

## Step 2: Create a Google service account

1. In the [Google Cloud console](https://console.cloud.google.com/), create a project and enable the **Cloud Identity API**.
2. Create a service account and add a JSON key.
3. In the Google Admin console, head to **Security > Access and data control > API controls > Domain-wide delegation** and select **Add new**. Enter the service account's client ID (its **Unique ID**, on the **Details** tab) and the `https://www.googleapis.com/auth/cloud-identity.devices` scope.
4. Choose an admin account for the service account to act as. The best practice is a dedicated admin account that can only manage mobile devices.

## Step 3: Sync hosts

1. In the Google Admin console, head to **Account > Account settings** and copy your **Customer ID**.
2. Download [`sync-fleet-hosts-to-google.sh`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/api-scripts/sync-fleet-hosts-to-google.sh) and set your Fleet URL, your customer ID, and the admin account from Step 2.
3. Run it with `DRY_RUN=true` to see the changes it would make. Set `FLEET_API_TOKEN` to the token from Step 1 and `GOOGLE_CREDENTIALS` to the contents of the JSON key from Step 2.
4. Schedule it to run every 5 minutes. The best practice is a [GitHub Actions](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#schedule) workflow in your GitOps repository, with the Fleet API token (`GOOGLE_SYNC_FLEET_API_TOKEN`) and the JSON key (`GOOGLE_CREDENTIALS`) stored as repository secrets:

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
      - run: bash sync-fleet-hosts-to-google.sh
        env:
          FLEET_API_TOKEN: ${{ secrets.GOOGLE_SYNC_FLEET_API_TOKEN }}
          GOOGLE_CREDENTIALS: ${{ secrets.GOOGLE_CREDENTIALS }}
```

GitHub can start scheduled runs late, or skip them, when it's busy. Check the workflow's run history to see how often it runs.

To check a sync, head to **Devices > Mobile & endpoints > Devices** in the Google Admin console, select the device, and select **Third-party services**. A device that's managed by Fleet shows **fleet (custom)**, **Managed**, its Fleet host ID, and its serial number under **Asset tags**.

> **Note:** The script matches devices to Fleet hosts by the end user's IdP email and device type (iPhone or iPad). If an end user has more than one iPhone (or iPad) in Google or Fleet, the script doesn't mark any of them as managed, and prints them for review. See the known limitation at the top of this guide.

To fix a device printed for review, ask the end user to remove their work account from the other iPhone (or iPad). Then delete that device in the Google Admin console under **Devices > Mobile & endpoints > Devices**. The next sync marks the company-owned one as managed.

By default, the script skips personally owned (BYOD) hosts, with the MDM status **On (personal)**, and only uses IdP emails. To change this, edit `ENROLLMENT_STATUSES` and `EMAIL_SOURCES` at the top of the script.

When a host is removed from Fleet, the next sync marks its device as unmanaged. If no host in Fleet has an email, the script stops without changing anything, and the run fails. This way, an outage or a wrong API token can't block everyone.

## Step 4: Turn on the access level

1. In the Google Admin console, head to **Security > Access and data control > Context-Aware Access** and select **Access levels > Create access level**.
2. Select **Advanced** and enter this condition:

```
device.vendors.exists(k, device.vendors[k].is_managed_device == true)
```

3. Select **Assign access levels**, choose a test organizational unit (OU), and assign the access level to the apps you want to protect, like Gmail and Drive. Don't assign it to **Admin Console**. Select **Apply to Google desktop and mobile apps**, and check that the assignment is **Active**, not **Monitor**.
4. After testing, assign the access level to all OUs.

> **Note:** This condition accepts any device partner that reports a device as managed. If you also connect another partner, for example a BeyondCorp Alliance partner, the devices it manages can sign in too.

## Step 5: Test

| Host | Expected result |
|---|---|
| iPhone or iPad in Fleet | Sign-in succeeds |
| iPhone or iPad not in Fleet | Sign-in denied |

Google applies a new state within seconds of a sync, including in apps that are already open. When a device becomes managed again, Google asks the end user to sign in again: close the message and reopen the app.

## Troubleshooting

- **Sign-in denied for a managed host**: Check that the host has the end user's IdP email in Fleet, and that the last sync didn't list the host for review.
- **New iPhone denied after an upgrade**: Check that the end user's old iPhone isn't still in the Google Admin console. If it is, delete it, so the script can match the new one.
- **Sync prints "No email" for a host**: Check that end user authentication is turned on for the host's fleet.
- **Sync stops with "Fleet returned no managed iPhones or iPads"**: Check the API token from Step 1, and that your hosts have an IdP email in Fleet.
- **Sync stops with "Google didn't issue a token: unauthorized_client"**: Check the domain-wide delegation from Step 2. A new delegation can take a few minutes to work.
- **Device missing from Google**: Check that the end user signed in to a Google app on the device, and that the device appears in the Google Admin console under **Devices > Mobile & endpoints**.

<meta name="articleTitle" value="Conditional access: Google">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-10-09">
<meta name="description" value="Use Google for conditional access, blocking sign-in from iPhones and iPads that aren't managed by Fleet.">
