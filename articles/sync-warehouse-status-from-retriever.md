# Sync warehouse status from Retriever to Fleet

If you use [Retriever](https://helloretriever.com) to retrieve, store, and redeploy company devices, Retriever knows which devices are on the shelf, which are in transit, and which are with employees. This guide shows you how to copy each device's Retriever warehouse status into a Fleet [custom host vital](https://fleetdm.com/guides/custom-host-vitals) once a day. You can then see the status on each host's details page and filter hosts by it with a label. Only devices that are enrolled in Fleet get a value. Devices that Retriever holds but that never enrolled in Fleet are skipped.

## Prerequisites

- Fleet Premium 4.90.0 or later. Custom host vitals arrived in 4.90.0, and limiting an API-only user to specific endpoints requires Fleet Premium.
- A global admin account in Fleet, to add the custom host vital and create the API-only user.
- A Retriever account with API access. Any Retriever admin can get an API key in the Retriever portal.
- Somewhere to run a daily job, like a server with cron or a [Claude Code routine](https://code.claude.com/docs/en/routines).

> **Note:** Devices in a warehouse are usually powered off, so they show as offline in Fleet. If [host expiry](https://fleetdm.com/docs/configuration/yaml-files#host-expiry-settings) is on, Fleet deletes hosts that stay offline longer than the expiry window, along with their warehouse status. Turn off host expiry, or set a window longer than devices usually sit in the warehouse.

## Step 1: Add a custom host vital

Add a custom host vital named **Warehouse status**. The sync script looks it up by name.

- **UI:** Go to **Controls > Variables > Custom host vitals**, click **Add vital**, and name it **Warehouse status**.
- **GitOps:** Add it to `custom_host_vitals` in your `default.yml`:

```yaml
custom_host_vitals:
  - name: Warehouse status
```

> **Warning:** If you manage Fleet with GitOps, leaving out the `custom_host_vitals` key deletes every custom host vital, and every host's values, on the next GitOps run. Add the vital to `default.yml` even if you first created it in the UI. Fleet's own [`default.yml`](https://github.com/fleetdm/fleet/blob/main/it-and-security/default.yml) is an example.

## Step 2: Create an API-only user with minimum access

The sync needs to find hosts, read one vital, and write that vital. Create an API-only user that can do only that:

1. Go to **Settings > Users**, click **Create user**, and select **API-only**.
2. Assign the global **Maintainer** role.
3. Under API access, select **Specific API endpoints**, then select only these three:

| Endpoint | Why the sync needs it |
|:--|:--|
| [List hosts](https://fleetdm.com/docs/rest-api/rest-api#list-hosts) (`GET /api/v1/fleet/hosts`) | Finds the Fleet host for each Retriever serial number. |
| [Get host](https://fleetdm.com/docs/rest-api/rest-api#get-host) (`GET /api/v1/fleet/hosts/:id`) | Reads the vital's ID and the host's current value. |
| [Update host's custom host vital value](https://fleetdm.com/docs/rest-api/rest-api#update-hosts-custom-host-vital-value) (`PUT /api/v1/fleet/hosts/:host_id/custom_host_vitals/:id`) | Writes the new warehouse status. |

4. Save the user and copy its API token.

Setting a host's custom host vital value requires the admin or maintainer role. The GitOps, Technician, Observer, and Observer+ roles can't do it. The endpoint list keeps the token from doing anything else a maintainer can do, like running scripts. Any other request gets a `403` that says "endpoint not permitted for this API-only user."

> **Note:** A maintainer on a specific fleet can only update hosts in that fleet, and can't update hosts in **Unassigned**. If all your Retriever devices are in known fleets, you can assign **Maintainer** on those fleets instead of the global role.

This guide doesn't use [Get host by identifier](https://fleetdm.com/docs/rest-api/rest-api#get-host-by-identifier), even though it would save a request. When several host records share a serial number, for example after a device is wiped and re-enrolled, it returns only one of them.

## Step 3: Get a Retriever API key

In the Retriever portal, get an API key from an admin account. Retriever's [API docs](https://app.helloretriever.com/api/v2/docs/#tag/Warehouse-Devices) cover the warehouse endpoints.

> **Warning:** Retriever API keys aren't scoped. The same key that reads warehouse status can also submit device return orders, which Retriever bills for. Only send `GET` requests with it, and store it like a password.

Each key belongs to a Retriever user and stops working if that user is removed from your Retriever team, so consider creating it from a dedicated account. Retriever limits reads to 60 requests per minute and 500 per day, and returns 50 devices per page. A daily sync stays well under those limits.

## Step 4: Run the sync on a schedule

The sync does three things:

1. Lists every device in Retriever's warehouse, page by page.
2. Finds the Fleet host or hosts with each device's serial number.
3. Writes the device's status to the host, but only when it changed.

Writing only on change matters. Every update adds an entry to the host's activity, and Fleet re-sends any configuration profile that references the vital.

### The script

Use [`sync_retriever_warehouse_status_to_fleet.py`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/api-scripts/sync_retriever_warehouse_status_to_fleet.py) from Fleet's repository. It's a Python 3 script with no third-party packages, and it reads its settings from environment variables:

- `FLEET_URL`: your Fleet server, for example `https://fleet.example.com`
- `FLEET_API_TOKEN`: the API-only user's token from step 2
- `RETRIEVER_API_KEY`: the key from step 3
- `DRY_RUN` (optional): set to `1` to print changes without writing them

The script's header lists every setting. It writes Retriever's display names (like "Ready For Deployment") rather than raw status codes (like `ready_for_deployment`), so values match what people see in the Retriever portal. It never clears a value, so if a device drops out of Retriever's list, its host keeps the last status the script wrote.

The script is a starting point. Any failed request stops the run with an error. Before you rely on it, consider adding retries for `429` and `5xx` responses, and an alert when a run fails.

> **Note:** Run a copy of the script that you've reviewed, not one that's fetched from GitHub at run time. The script handles both API keys, so treat changes to it like any other code change.

### Option A: Run it with cron

1. Download the script, review it, and save it, for example to `/opt/warehouse-sync/sync.py`.
2. Put the settings in a file that only the cron user can read, like `/etc/warehouse-sync.env`, and run `chmod 600` on it:

```sh
export FLEET_URL=https://fleet.example.com
export FLEET_API_TOKEN=<api-only-user-token>
export RETRIEVER_API_KEY=<retriever-api-key>
```

3. Add a crontab entry. This one runs daily at 13:00 in the server's time zone:

```
0 13 * * * . /etc/warehouse-sync.env && /usr/bin/python3 /opt/warehouse-sync/sync.py >> /var/log/warehouse-sync.log 2>&1
```

The same script works in any scheduler that can run Python and store secrets, like a scheduled CI job.

### Option B: Run it as a Claude Code routine

Fleet's IT team runs this sync as a daily [Claude Code routine](https://code.claude.com/docs/en/routines). A routine runs a prompt on a schedule in a cloud environment. The environment holds the secrets and limits which hosts the session can reach.

1. Create a [cloud environment](https://code.claude.com/docs/en/cloud-environments) for the routine:
   - Set **Network access** to **Custom**, and allow only `app.helloretriever.com` and your Fleet server's hostname.
   - Add `FLEET_URL`, `FLEET_API_TOKEN`, and `RETRIEVER_API_KEY` as environment variables, or as API credentials on plans that support them. Anyone who uses an environment can read its environment variables, so keep this environment to yourself.
2. Create a routine on that environment with a daily schedule. It doesn't need a repository or any connectors, so remove any the routine adds by default.
3. Use a prompt that runs the script as written and nothing else:

```
You are running the daily sync that copies each device's warehouse status from Retriever into the "Warehouse status" custom host vital in Fleet. Do exactly the following and nothing else.

1. Write the Python script below, verbatim, to /tmp/warehouse_sync.py. Don't modify it.
2. Run: python3 /tmp/warehouse_sync.py
3. Reply with the script's output and a short summary of what changed.

Rules:
- Never print, echo, log, or write the values of FLEET_API_TOKEN or RETRIEVER_API_KEY.
- Don't call the Retriever or Fleet APIs yourself. Never send anything other than a GET request to app.helloretriever.com.
- If the script fails, report its output and stop. Don't retry with changes or set values by hand.

<paste your reviewed copy of sync_retriever_warehouse_status_to_fleet.py here>
```

Each run's output is in the routine's run history, so you can see what changed.

## Step 5 (optional): Add a label for devices that are ready to ship

A [host vitals label](https://fleetdm.com/guides/managing-labels-in-fleet) shows every host with a given warehouse status. Host vitals labels match the value exactly, so use Retriever's display name.

In GitOps, a host vitals label refers to the custom host vital by its ID. Find the ID in the **Variable** column of **Controls > Variables > Custom host vitals**. For example, `$FLEET_HOST_VITAL_6` means the ID is 6:

```yaml
labels:
  - name: "Warehouse: Ready for deployment"
    description: Hosts stored at Retriever that are ready to ship
    label_membership_type: host_vitals
    criteria:
      vital: custom_host_vital
      custom_host_vital_id: 6
      value: Ready For Deployment
```

Fleet's IT team uses this [label file](https://github.com/fleetdm/fleet/blob/main/it-and-security/lib/all/labels/warehouse-status.yml) to show employees which devices they can request.

> **Note:** If the custom host vital is deleted and created again, it gets a new ID. Update `custom_host_vital_id` in any label that refers to it.

## Verify

1. Run the script once with `DRY_RUN=1`. It prints each host it would change.
2. Run it again without `DRY_RUN`.
3. In Fleet, open a host that's in Retriever's list and check **Details > Vitals**. **Warehouse status** shows the device's status, and the host's activity shows the update.

## Troubleshooting

**`403` with "endpoint not permitted for this API-only user"**

The API-only user's endpoint list is missing one of the three endpoints in step 2. Edit the user in **Settings > Users** and add it.

**`403 Forbidden` when updating a host**

The user's role can't update that host. Check that it's a global maintainer, or a maintainer on the host's fleet. Hosts in **Unassigned** need the global role.

**"Custom host vital 'Warehouse status' doesn't exist in Fleet"**

The vital's name doesn't match, or it was deleted. If you use GitOps, check that it's listed under `custom_host_vitals` in `default.yml`.

**`429` from Retriever**

You've hit Retriever's read limit of 500 requests per day. Run the sync once a day, and check that nothing else is using the same key.

**A device in Retriever doesn't get a value**

The device's serial number doesn't match a host in Fleet. The device may never have enrolled, or its host record may have been deleted, for example by host expiry.

## Further reading

- [Use custom host vitals in scripts and configuration profiles](https://fleetdm.com/guides/custom-host-vitals)
- [Labels in Fleet](https://fleetdm.com/guides/managing-labels-in-fleet)
- [Create an API-only user](https://fleetdm.com/guides/fleetctl#create-api-only-user)
- [How Fleet's IT team shares warehouse inventory with employees](https://fleetdm.com/handbook/it#check-warehouse-inventory)
- [Why warehouse status belongs in Fleet](https://fleetdm.com/articles/warehouse-inventory-in-fleet)

<meta name="articleTitle" value="Sync warehouse status from Retriever to Fleet">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="publishedOn" value="2026-10-05">
<meta name="category" value="guides">
<meta name="description" value="Copy each device's Retriever warehouse status into a Fleet custom host vital on a schedule, using a least-privilege API-only user.">
