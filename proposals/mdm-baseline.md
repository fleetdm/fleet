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
`host_disks`): as tables holding metadata for a host. 

When we get information through MDM channels (be it via an enrollment, or via
the DEP API notifying a new device purchased,) we always get a device serial
number.

1. Ensure there's a host in the `hosts` table with that serial number (insert
   one if there isn't.) Try to pre-populate as much info as possible from the
   MDM payload (eg: `ProductName` is `hosts.hardware_model`)
1. Create a new entry in the related `nano_*` table pointing to the host.

This implies that we must add an uniqueness constraint for the serial number in both tables.

We should also add a new column or table to track the MDM enrolling status
("pending", "enrolled") without having to `JOIN` both tables.

#### TODO:

1. Investigate what are the non-optional fields in the `hosts` table, and if
   having default empty values for some of them requires a breaking change.
1. Investigate the consequences of using a serial number to uniquely identify
   hosts, eg: what happens if two users on the same machine use different
   enrollment profiles? See "macOS extensions" in MDM spec for details.

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

Some features require information that can only be accessed through MDM, for example:

1. The recovery key for a host.
1. The list of currently installed profiles.
1. Security information.

We need to stablish a pattern to ingest and keep this data in sync, similarly
to how we do for data coming from `osquery`.

#### Proposed solution

Implement a cron job that:

1. Issues the necessary commands.
1. Monitors their output.
1. Ingests the data.

Driving example: we need a list of the currently installed profiles in a host
to check if its current status matches the desired status. To do this we need to:

1. Issue a `ProfileList` command to all devices.
1. Monitor its output until we get a response.
1. In a table (real names TBD) `host_profiles` set `profiles`, `last_sync_at` and `sync_status`.


#### TODO

1. Timeout for command responses.
1. Crash response, retries.?

### Problem: Storing scripts, profile files and installers

We currently store everything in MySQL, we should move all of this to a blob
storage.

We could start with S3, which is a standard API not tied to S3 to store the
installers. We already have S3 logic to store installers and carves.

#### TODO:

- Investigate using pre-signed URLs instead of buffering the file contents
  through Fleet (`n` installers * `m` devices.) We were worried about those in
  the past because while they are part of the S3 common API, they might not be
  supported by non-Amazon vendors.
    - A quick search yields that MinIO, Google and Azure support pre-signed
      URLs, but we need to confirm that we can use the same API to request the
      URLs.
