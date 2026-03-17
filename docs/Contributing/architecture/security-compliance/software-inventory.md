# Software inventory architecture

This document provides an overview of Fleet's Software Inventory architecture.

## Introduction

Software Inventory in Fleet provides visibility into the software installed on devices across the fleet. This document provides insights into the design decisions, system components, and interactions specific to the Software Inventory functionality.

## Software identity

A software row in the `software` table is uniquely identified by the combination of these fields:

- `name` - the software name as reported by osquery
- `version`
- `source` - e.g., `apps`, `deb_packages`, `programs`, `chrome_extensions`
- `bundle_identifier` - macOS bundle ID (e.g., `org.mozilla.firefox`)
- `release`, `vendor`, `arch` - included if any is non-empty
- `extension_id`, `extension_for` - included if any is non-empty
- `application_id` - included if non-empty
- `upgrade_code` - included if non-empty (Windows MSI)

These fields are combined into a checksum (`ComputeRawChecksum` in `server/fleet/software.go`) stored as a unique index on the `software` table. The same fields (in a different format) produce the `ToUniqueStr()` used for in-memory comparisons during ingestion.

### The software table is shared across hosts

The `software` table is a global catalog -each row represents a unique piece of software, and multiple hosts link to the same row via the `host_software` join table. This means modifying a software row affects all hosts that reference it.

### Software titles

Software titles (`software_titles` table) group related software versions under a single name for display in the UI. For macOS apps with a `bundle_identifier`, the title is matched by bundle ID (not name), so different software rows with different names but the same bundle ID share a single title.

### Why name is part of software identity

Name is included in the software identity (checksum and unique string) because multiple distinct software entries can share the same `bundle_identifier` and `version`. For example, macOS helper binaries inside an app bundle:

```
"Postman Helper (GPU)",       version="", bundle_id="com.postmanlabs.mac.helper"
"Postman Helper (Renderer)",  version="", bundle_id="com.postmanlabs.mac.helper"
```

These are different executables that osquery discovers independently. Without name in the identity, they would collapse into a single row, losing visibility into what's actually installed.

### Name changes and the rename problem

Because name is part of software identity, changing how Fleet computes the name (e.g., modifying the osquery query to use `display_name` instead of the raw filename) creates a mismatch between what's stored in the database and what osquery reports on the next check-in. This triggers the ingestion pipeline to treat the software as "uninstalled" (old name) and "newly installed" (new name), which:

1. Creates a new software row with a new ID
2. Deletes the old `host_software` link and creates a new one
3. Orphans the old software row (cleaned up later by `SyncHostsSoftware`)
4. Loses vulnerability associations until the next vulnerability scan

This is normally a non-issue during regular operation because osquery consistently reports the same name for a given app. It only becomes a problem when Fleet changes how software names are derived -such as modifying the osquery query to prefer `display_name` over the raw filename (see [#28584](https://github.com/fleetdm/fleet/issues/28584)). In those cases, a database migration should also update existing software rows to match the new naming convention.

There is no clean way to distinguish "same software, name changed" from "different software, same bundle_id" in the general case. A 1:1 rename heuristic (match by bundle_id+version when there's exactly one entry on each side) works for most apps but fails for the helper binary case described above.


## Ingestion pipeline

The software ingestion pipeline runs when a host checks in (approximately once per hour). The entry point is `applyChangesForNewSoftwareDB` in `server/datastore/mysql/software.go`.

### Flow

1. **Read current state** -`listSoftwareByHostIDShort` reads the host's current software from the replica DB
2. **Diff** -`nothingChanged` compares current vs incoming by `ToUniqueStr()`. If identical, no DB writes occur
3. **Lookup existing** -`getExistingSoftware` computes checksums for new incoming software and looks them up in the DB
4. **Phase 1 (outside transaction)** -`preInsertSoftwareInventory` inserts new software titles and software rows via `INSERT IGNORE` in small batches to reduce lock contention
5. **Phase 2 (transaction)** -Deletes `host_software` links for uninstalled software, creates links for new software, and updates `last_opened_at` timestamps

### Visibility

The UI endpoint (`ListHostSoftware`) uses `hostInstalledSoftware` which joins `host_software` → `software` → `software_titles` with `INNER JOIN`.

## Architecture overview

## Key components

## Architecture diagram
