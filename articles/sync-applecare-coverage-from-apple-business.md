# Sync AppleCare coverage from Apple Business to Fleet

Apple Business knows when each device's warranty and AppleCare coverage ends, but your team works in Fleet. This guide shows you how to copy each device's coverage end date from the Apple Business API into a Fleet [custom host vital](https://fleetdm.com/guides/custom-host-vitals) once a day. You can then see the date on each host's details page, and spot devices that are out of coverage before you send them for repair. Only devices that are in Apple Business and enrolled in Fleet get a value.

## Prerequisites

- Fleet Premium 4.90.0 or later. Custom host vitals arrived in 4.90.0, and limiting an API-only user to specific endpoints requires Fleet Premium.
- A global admin account in Fleet, to add the custom host vital and create the API-only user.
- An Organization Administrator account in [Apple Business](https://business.apple.com). Only an Organization Administrator can create an API account.
- A computer with Python 3 and the `openssl` command, to sign the client assertion in step 4. macOS includes both.
- Somewhere to run a daily job, like a server with cron or a [Claude Code routine](https://code.claude.com/docs/en/routines).

## Step 1: Add a custom host vital

Add a custom host vital named **AppleCare warranty expiration**. The sync script looks it up by name.

- **UI:** Go to **Controls > Variables > Custom host vitals**, click **Add vital**, and name it **AppleCare warranty expiration**.
- **GitOps:** Add it to `custom_host_vitals` in your `default.yml`:

```yaml
custom_host_vitals:
  - name: AppleCare warranty expiration
```

> **Warning:** If you manage Fleet with GitOps, leaving out the `custom_host_vitals` key deletes every custom host vital, and every host's values, on the next GitOps run. Add the vital to `default.yml` even if you first created it in the UI. Fleet's own [`default.yml`](https://github.com/fleetdm/fleet/blob/main/it-and-security/default.yml) is an example.

To use a different name, set the `VITAL_NAME` environment variable when you run the script in step 5.

## Step 2: Create an API-only user with minimum access

The sync needs to find hosts, read one vital, and write that vital. Create an API-only user that can do only that:

1. Go to **Settings > Users**, click **Create user**, and select **API-only**.
2. Assign the global **Maintainer** role.
3. Under API access, select **Specific API endpoints**, then select only these three:

| Endpoint | Why the sync needs it |
|:--|:--|
| [List hosts](https://fleetdm.com/docs/rest-api/rest-api#list-hosts) (`GET /api/v1/fleet/hosts`) | Matches Apple Business serial numbers to Fleet hosts. |
| [Get host](https://fleetdm.com/docs/rest-api/rest-api#get-host) (`GET /api/v1/fleet/hosts/:id`) | Reads the vital's ID and the host's current value. |
| [Update host's custom host vital value](https://fleetdm.com/docs/rest-api/rest-api#update-hosts-custom-host-vital-value) (`PUT /api/v1/fleet/hosts/:host_id/custom_host_vitals/:id`) | Writes the coverage end date. |

4. Save the user and copy its API token.

> **Warning:** Select **Update host's custom host vital value**, not **Update custom host vital**. The second one renames the vital itself. A dry run never writes, so it can't catch a missing write endpoint. Check the list before the first real run.

Setting a host's custom host vital value requires the admin or maintainer role. The GitOps role can't list every host, and the Observer and Observer+ roles can't write values. The endpoint list keeps the token from doing anything else a maintainer can do, like running scripts. Any other request gets a `403` that says "endpoint not permitted for this API-only user."

## Step 3: Create an Apple Business API account

The Apple Business API can also assign devices to other device management services and release devices from your organization. Give the API account a custom role that can only view devices.

1. Sign in to [Apple Business](https://business.apple.com) as an Organization Administrator.
2. Go to **Settings > Roles & Permissions**. Next to **Custom Roles**, click **Add**.
3. Name the role, for example "Device API read-only," and click **Next**.
4. Under **Devices**, select only permissions that view devices. Leave off any permission that assigns devices to device management services, releases devices from the organization, or removes Activation Lock. Leave the other groups off, then click **Done**.
5. Go to **Settings > API** and click **Add API Account**. Name the account, choose the custom role you just created, and click **Next**.
6. Click **Generate & Download** to download the account's private key (a `.pem` file).
7. Click **Edit** on the account and copy its **Client ID** (it starts with `BUSINESSAPI.`) and **Key ID**.

Apple's guide to [creating an API account](https://support.apple.com/guide/business/axm33189f66a/web) has more detail.

> **Warning:** The private key can get access tokens for the API account. Keep it on the computer you sign with, readable only by you, and don't copy it to the server that runs the sync.

## Step 4: Sign a client assertion

The Apple Business API doesn't take the private key directly. Instead, you sign a short token called a client assertion with the key, and the sync exchanges it for an access token on each run. A client assertion is valid for up to 180 days, so the server that runs the sync only needs the assertion, not the key.

1. Download [`sign_apple_business_client_assertion.py`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/api-scripts/sign_apple_business_client_assertion.py) and review it. It's a Python 3 script that uses the `openssl` command and no third-party packages.
2. Run it with the key, Client ID, and Key ID from step 3:

```sh
python3 sign_apple_business_client_assertion.py \
  --key ~/Downloads/<private-key>.pem \
  --client-id BUSINESSAPI.<uuid> \
  --key-id <key-id>
```

3. Copy the assertion it prints. It also prints the date the assertion expires, 179 days from now.

Set a reminder to sign a new assertion before that date. The sync script warns when the assertion has fewer than 30 days left, and stops with an error once it has expired. Signing a new assertion doesn't change the API account, so you only need to update `ABM_CLIENT_ASSERTION` wherever the sync runs.

> **Note:** Store the assertion like a password. Until it expires, anyone who has it can read your organization's devices and coverage. To cut it off early, revoke the private key in **Settings > API**, then generate a new key and sign a new assertion.

## Step 5: Run the sync on a schedule

The sync does four things:

1. Lists every device in your Apple Business organization, skipping devices that were released.
2. Finds the Fleet host or hosts with each device's serial number.
3. Reads the device's coverage, then takes the latest end date across coverage that wasn't canceled, including coverage that already ended.
4. Writes the date to the host, but only when it changed.

Writing only on change matters. Every update adds an entry to the host's activity, and Fleet re-sends any configuration profile that references the vital.

> **Note:** The script ignores coverage that has no end date, like an AppleCare+ for Business subscription. If a device has a subscription and a dated limited warranty, the vital shows the warranty's end date, even while the subscription covers the device.

The script makes one coverage request per matched device and one Get host request per matched host, so a run takes about two minutes for 170 devices. Larger organizations take proportionally longer.

### The script

Use [`sync_applecare_coverage_to_fleet.py`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/api-scripts/sync_applecare_coverage_to_fleet.py) from Fleet's repository. It's a Python 3 script with no third-party packages, and it reads its settings from environment variables:

- `FLEET_URL`: your Fleet server's `https://` address, for example `https://fleet.example.com`
- `FLEET_API_TOKEN`: the API-only user's token from step 2
- `ABM_CLIENT_ASSERTION`: the client assertion from step 4
- `VITAL_NAME` (optional): the vital's name, if it isn't **AppleCare warranty expiration**
- `DRY_RUN` (optional): set to `1` to print changes without writing them

The script's header lists every setting. It never clears a value, so a device that's released from Apple Business keeps the last date the script wrote. It retries `429` and `5xx` responses twice, and stops on the first `401` or `403` from either API, because a missing permission fails every device the same way. If one device fails for another reason, it prints the error, moves on, and exits with an error when it's done. It doesn't follow redirects, so set `FLEET_URL` to your Fleet server's final `https://` address.

> **Note:** Run a copy of the script that you've reviewed, not one that's fetched from GitHub at run time. The script handles both credentials, so treat changes to it like any other code change.

### Option A: Run it with cron

1. Download the script, review it, and save it, for example to `/opt/applecare-sync/sync.py`.
2. Put the settings in a file that only the cron user can read, like `/etc/applecare-sync.env`, and run `chmod 600` on it:

```sh
export FLEET_URL=https://fleet.example.com
export FLEET_API_TOKEN=<api-only-user-token>
export ABM_CLIENT_ASSERTION=<client-assertion>
```

3. Add a crontab entry. This one runs daily at 13:30 in the server's time zone:

```
30 13 * * * . /etc/applecare-sync.env && /usr/bin/python3 /opt/applecare-sync/sync.py >> /var/log/applecare-sync.log 2>&1
```

The same script works in any scheduler that can run Python and store secrets, like a scheduled CI job. Alert on a non-zero exit code: `1` means some devices failed, and `2` means the run stopped, for example because the assertion expired.

### Option B: Run it as a Claude Code routine

Fleet's IT team runs this sync as a daily [Claude Code routine](https://code.claude.com/docs/en/routines). A routine runs a prompt on a schedule in a cloud environment. The environment holds the secrets and limits which hosts the session can reach.

1. Create a [cloud environment](https://code.claude.com/docs/en/cloud-environments) for the routine:
   - Set **Network access** to **Custom**, and allow only `account.apple.com`, `api-business.apple.com`, and your Fleet server's hostname.
   - Add `FLEET_URL`, `FLEET_API_TOKEN`, and `ABM_CLIENT_ASSERTION` as environment variables, or as API credentials on plans that support them. Anyone who uses an environment can read its environment variables, so keep this environment to yourself.
2. Create a routine on that environment with a daily schedule. It doesn't need a repository or any connectors, so remove any the routine adds by default.
3. Use a prompt that runs the script as written and nothing else:

```
You are running the daily sync that copies each Apple device's AppleCare and warranty expiration date from the Apple Business API into the "AppleCare warranty expiration" custom host vital in Fleet. Do exactly the following and nothing else.

1. Write the Python script below, verbatim, to /tmp/applecare_sync.py. Don't modify it.
2. Run: python3 /tmp/applecare_sync.py
3. Reply with the script's output and a short summary of what changed. If it printed a warning, start your reply with it.

Rules:
- Never print, echo, log, or write the values of FLEET_API_TOKEN or ABM_CLIENT_ASSERTION.
- Don't call the Apple Business or Fleet APIs yourself. Never send anything to api-business.apple.com other than the script's GET requests.
- If the script fails, report its output and stop. Don't retry with changes or set values by hand.

<paste your reviewed copy of sync_applecare_coverage_to_fleet.py here>
```

The script can take a few minutes, so if the session moves it to the background, the routine should wait for it to finish before replying. Each run's output is in the routine's run history.

## Step 6 (optional): Find hosts by coverage date

The script writes dates as `YYYY-MM-DD`, for example `2028-05-27`, so they sort correctly as text. A date in the past means the device's coverage has ended.

- **Labels:** A [host vitals label](https://fleetdm.com/guides/managing-labels-in-fleet) matches a value exactly, so it can find hosts whose coverage ends on one date, but not before a date.
- **Sorting and filtering:** Sorting and filtering the **Hosts** page by custom host vitals is planned for a future release ([#49712](https://github.com/fleetdm/fleet/issues/49712)). Because the dates sort as text, you'll be able to sort hosts by coverage end date without changing the sync.
- **API:** To list hosts that are out of coverage today, read each host's `custom_host_vitals` from the [Get host](https://fleetdm.com/docs/rest-api/rest-api#get-host) endpoint and compare the date.

## Verify

1. Run the script once with `DRY_RUN=1`. It prints each host it would change, followed by a summary line.
2. Run it again without `DRY_RUN`.
3. In Fleet, open a host that's in Apple Business and check **Details > Vitals**. **AppleCare warranty expiration** shows a date, and the host's activity shows the update.
4. Run it a third time. The summary says it updated 0 hosts, because nothing changed.

## Troubleshooting

**`403` with "endpoint not permitted for this API-only user"**

The API-only user's endpoint list is missing one of the three endpoints in step 2. A missing **Update host's custom host vital value** only shows up on the first real run, because a dry run doesn't write. Edit the user in **Settings > Users** and add it.

**`403 Forbidden` when listing hosts**

The API-only user's role can't list hosts. Check that it's a global **Maintainer**, not GitOps or Observer.

**"Apple Business token request failed"**

Apple rejected the client assertion. Check that you signed it with the private key that's currently active for the API account. If the key was revoked or regenerated in **Settings > API**, sign a new assertion with the new key.

**"The Apple Business client assertion expired"**

Sign a new assertion (step 4) and update `ABM_CLIENT_ASSERTION` wherever the sync runs.

**`401` or `403` from `api-business.apple.com`**

The API account's role is missing a permission the device or coverage endpoint needs. Check the role in **Settings > Roles & Permissions**.

**"Custom host vital 'AppleCare warranty expiration' doesn't exist in Fleet"**

The vital's name doesn't match, or it was deleted. If you use GitOps, check that it's listed under `custom_host_vitals` in `default.yml`. If you named the vital something else, set `VITAL_NAME`.

**A device prints "no dated coverage, skipped"**

None of the device's coverage has an end date. AppleCare+ for Business subscriptions renew monthly or yearly and have no end date. Check the device's coverage in Apple Business.

**A device in Apple Business doesn't get a value**

The device's serial number doesn't match a host in Fleet. The device may never have enrolled, or its host record may have been deleted, for example by host expiry.

## Further reading

- [Use custom host vitals in scripts and configuration profiles](https://fleetdm.com/guides/custom-host-vitals)
- [Labels in Fleet](https://fleetdm.com/guides/managing-labels-in-fleet)
- [Create an API account in Apple Business](https://support.apple.com/guide/business/axm33189f66a/web)
- [Implementing OAuth for the Apple School Manager and Apple Business API](https://developer.apple.com/documentation/apple-school-and-business-manager-api/implementing-oauth-for-the-apple-school-manager-and-apple-business-api)
- [Sync warehouse status from Retriever to Fleet](https://fleetdm.com/guides/sync-warehouse-status-from-retriever)

<meta name="articleTitle" value="Sync AppleCare coverage from Apple Business to Fleet">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="publishedOn" value="2026-10-15">
<meta name="category" value="guides">
<meta name="description" value="Copy each device's AppleCare and warranty end date from the Apple Business API into a Fleet custom host vital on a schedule.">
