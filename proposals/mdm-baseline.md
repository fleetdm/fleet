# MDM baseline

This document is a high-level spec/guide of the technical challenges that don't
directly map to a product feature request.

We will create issues to tackle actionable bits, but the idea is to also use
this to guide the implementation of feature requests.


### Problem: Hosts can be enrolled via MDM

Currently, there's no reliable way to associate an MDM enrolled device with a
host in the `hosts` table.

- If a host is enrolled in MDM before its enrolled via `osquery` (eg: DEP
  enrollment) we will have a record in the `nano_enrollments` table before the
  host exists in the `hosts` table.

- If a host exists in both tables, we could `JOIN` them using `hosts.uuid =
  nano_enrollments.device_id`  but we need to make sure the right constraints
  in the application logic exist and the right indexes are created for fast
  access.

Notes:
    - lot of non-optional fields, probably have API changes.
    - so far osquery_host_id
    - add unique index to `hosts.uuid` (including when a host is enrolling via MDM)
    - find out if DEP APIs return the UUID
        - we might have to do the same for `hosts.hardware_serial`
    - add columns for MDM status (pending, ...)
    - update DEP syncer to insert hosts, and update its status
    - update host mdm status during enrollment
    - see what data we can populate in the `hosts` table when during MDM
      enrollment (both from DEP apis and during the enrollment check-in)
        - note: ProductName is the same as `hosts.hardware_model`
    - enrollment profiles for multiple users, see macOS extensions in spec

Currently, the code thinks of these two tables as two disjoint sets of elements
that represent the same entity: a host.

Motivation:

- In https://github.com/fleetdm/fleet/issues/8878 we're asked to return hosts
  that are scheduled for DEP enrollment (not even enrolled yet) to the `GET
  /hosts` response.
- In https://github.com/fleetdm/fleet/issues/8878 we're asked to retrieve a
  list of hosts enrolled via `osquery` that are pending to be enrolled via MDM.

#### Proposed solution

`hosts` should always be the source of truth, we could consider
`nano_enrollments` the same way we treat  the "auxiliary" `hosts_*` tables (eg:
`host_disks`): as tables holding metadata for a host. Consequentially we should
update MDM tables as necessary to have a `host_id` attribute (we don't use
foreign keys for the `host_*` tables).

When we get information through MDM channels (be it via an enrollment, or via
the DEP API notifying a new device purchased,) we always get a device UUID or a
serial number, and we should:

1. Ensure there's a host in the `hosts` table with that UUID or serial number (insert one if there isn't.)
1. Create a new entry in the related `nano_*` table pointing to the host.

If a host doesn't have a corresponding entry in the `nano_*` table, we can assume it's not MDM enrolled.


### Problem: Issue an MDM command and monitor it’s status in fleet

- assumption: when a command is issued, it only targets one device.
- have an endpoint to retrieve all the commands that were issued to a host and
  their status, results, etc.
- have and endpoint that retrieves a single command, and its status/results,
  etc.
- let the server handle communication, handle NotNow commands, duplicated commands, etc.
- the server will transform the XML responses from MDM to JSON and return that to clients.
    - double check if it's OK to blindly return the info.
- what permissions should we add  to execute commands.
    - define granularity: per command basis?
- investigate command status, check what are the product requirements and see
  if we can meet them accordingly see: https://github.com/fleetdm/fleet/issues/8815
    - Invesitage if this is a limitation with MDM and certain commands or something we can fix.
    - Look at this as we implement the commands.


### Problem: ingesting/storing data from MDM

- examples: recovery key, list of profiles, security info.
- keep this data in sync (even if it's not initiated by users), probably a cron job
- all of this involves issuing a command, monitoring its output and "ingesting"
  the output.
- for profiles:
    - make sure they're in sync in order to:
        - issue the command only to host that need it
        - validate that the host has the latest profile
- probably a new table? `host_profiles`, `profiles_last_sync_at`, `sync_status`
    - think if we need to store the profile raw contents.


### Problem: Storing installers, profile files and installers

We currently store everything in MySQL.

- Use a blob storage, start with S3
- Investigate into using signed URLs instead of buffering the content through fleet
    - Note: signed URLs is probably an Amazon
