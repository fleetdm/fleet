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

https://github.com/fleetdm/fleet/issues/8815

### Problem: Parsing and serializing XML data

All MDM-related APIs are handled using XML, but:

1. The UI will need some of that information, and we can't just respond with XML.
1. We can't output raw XML to the terminal like we currently do.

We need an structured way to parse and store this kind of information.

### Problem: Storing installers, profile files and installers

We currently store everything in MySQL.
