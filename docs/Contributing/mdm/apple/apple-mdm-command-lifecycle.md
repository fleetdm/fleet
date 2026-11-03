# Apple MDM command lifecycle and cleanup

This document describes how an Apple MDM command moves through Fleet's `nano_*` tables, how Fleet cleans those rows up, and what to check when a feature adds a new automated command or stores a reference to one.

## Why this matters

Fleet enqueues Apple MDM commands constantly: profile installs and removals, refetches, VPP install verification, `DeclarativeManagement` sync, device name updates, and more. Each one adds rows that the "get next command" query and other hot paths have to walk. Without cleanup these tables grow without bound and degrade performance for every enrolled host.

Several features also derive live state from these rows (lock/wipe status, bootstrap package status, VPP install verification, recovery lock and managed local account rotation). Deleting a row a feature still reads silently breaks that feature. Because of this, cleanup is an opt-in allowlist: a command type is only deleted if it's explicitly listed, and anything else is kept indefinitely.

## Tables

| Table | One row per | Notes |
|---|---|---|
| `nano_commands` | command | The payload, `request_type`, and `command_uuid`. Shared by every host the command targets. |
| `nano_enrollment_queue` | command per enrollment | Delivery state. `active = 0` means the command will never be sent (see Deactivation below). |
| `nano_command_results` | command per enrollment | The device's response. Absent until the device responds. |

A queue row without a matching result row is considered unanswered, so nano re-serves it to the device. Any code that deletes these rows must delete the `nano_enrollment_queue` row before (or in the same transaction as) the `nano_command_results` row. Deleting the result first makes an already-answered command look pending and resends it.

## Lifecycle

1. **Enqueue.** Fleet writes the `nano_commands` row, then a `nano_enrollment_queue` row per target enrollment, then sends an APNs push. Fleet-generated commands that repeat often use a `command_uuid` prefix (for example `REFETCH-DEVICE-`, `DEVNAME-`, `VERIFY-VPP-INSTALLS-`) defined in `server/fleet/mdm.go`. The prefix lets cleanup tell Fleet's own recurring commands apart from user-run commands of the same request type.
2. **Delivery.** On check-in, nano returns the next active queue row that has no result (or a `NotNow` result).
3. **Result.** The device responds and nano writes `nano_command_results` with `Acknowledged`, `Error`, `CommandFormatError`, or `NotNow`. Fleet's results handlers update feature state (for example `host_mdm_apple_profiles`, `host_vpp_software_installs`) from the result.
4. **Deactivation.** Some flows set `active = 0` on queue rows instead of deleting them: re-enrollment clearing the queue, profile deletes and resends, SCEP renewal supersede, MDM reset, VPP and in-house install cancels, device name refresh, and GitOps batch-set. Inactive rows are hidden from the host's MDM command list and results.
5. **Cleanup.** The rows are deleted by the cleanup described below, or kept indefinitely if the command type isn't eligible.

## Cleanup

The `cleanup_apple_mdm_commands` job runs in the hourly `cleanups_then_aggregation` cron (`cmd/fleet/cron.go`). The datastore logic lives in `server/datastore/mysql/apple_mdm_cleanups.go`, and the classes and denylist live in `server/fleet/mdm.go`. Each run has a row budget and a command budget (see the `mdm.apple_command_cleanup_*` server settings) and a 10-minute time limit. Scan cursors are stored in Redis so the next run resumes where the last one stopped.

Sweeps run in this order and share one row budget:

1. **Inactive purge.** Deletes `active = 0` queue/result pairs deactivated longer ago than the short retention window. Request types in `AppleMDMInactivePurgeDenylist` are skipped because their inactive rows are still read back (for example, a canceled `DeviceLock` that the device acknowledges anyway).
2. **Retention sweep, short tier.** Deletes queue/result pairs with a terminal result (`Acknowledged`, `Error`, `CommandFormatError`) older than the short window, for commands matching `AppleMDMShortRetentionClasses` (request type plus optional `command_uuid` prefix).
3. **Retention sweep, standard tier.** Same, for request types in `AppleMDMStandardRetentionRequestTypes`, using the standard window.
4. **Command mop.** Deletes `nano_commands` rows that no longer have any queue or result rows. It first targets commands touched by the sweeps above, then walks the rest of the table oldest first. It skips commands younger than 24 hours, since nano writes the command row before its queue rows.

Any request type not listed in either tier is never deleted by the retention sweep. This includes commands whose state Fleet derives from the command itself, like `DeviceLock`, `EraseDevice`, and `EnableLostMode`, and any request type Apple adds in the future.

The per-acknowledgement refetch cleanup (`CleanupStaleNanoRefetchCommands`) still runs on refetch results and uses the short retention window.

### Reference guards

Regardless of class, a sweep skips a command while a feature still references it. The guards are the `NOT EXISTS` probes in `nanoCommandReferenceProbes` and, for the command mop, `nanoCommandMopFilter` and `nanoCommandDefensiveMopFilter`. They currently cover:

- `host_mdm_actions` lock, wipe, and unlock references
- the current `host_mdm_apple_profiles.command_uuid` (superseded commands aren't guarded)
- `host_mdm_apple_bootstrap_packages.command_uuid` (foreign key with `ON DELETE CASCADE`: deleting the command deletes the bootstrap record)
- `nano_cert_auth_associations.renew_command_uuid` (foreign key without cascade: an unguarded delete fails)
- VPP and in-house installs whose install or verification command hasn't reached a verified, failed, or canceled state
- current and pending recovery lock and managed local account password commands
- pending or running `setup_experience_status_results` commands
- `host_mdm_apple_device_names` (command mop only)

Every guarded column needs an index. The sweeps probe each guard for every candidate batch, so an unindexed guard turns each batch into a table scan.

## Adding a new command or a new reference

When a change adds a command that Fleet enqueues automatically (a new request type, or a new `command_uuid` prefix for an existing one):

- Decide whether the command is eligible for cleanup and add it to `AppleMDMShortRetentionClasses` or `AppleMDMStandardRetentionRequestTypes`. Short is for high-volume commands whose results are consumed right away. Standard is for commands users may want to see in host history for a while. If you're unsure, leave it unlisted: the command is kept, which is safe but adds to table growth.
- If the feature reads queue rows back after they're set to `active = 0`, add the request type to `AppleMDMInactivePurgeDenylist`.
- If the command repeats often, consider giving it a `command_uuid` prefix like our `REFETCH-` inventory commands so it can be classified and cleaned up separately and ideally, more frequently, from user-run commands of the same type.

When a change adds a table or column that stores a `nano_commands.command_uuid` that Fleet reads later:

- Add a guard probe for it to `nanoCommandReferenceProbes` in `server/datastore/mysql/apple_mdm_cleanups.go`. Guard only while the reference is live (for example, only the current profile command, or only unverified installs), so superseded rows can still be cleaned up.
- Add an index on the column in a migration.
- Add a test case to `server/datastore/mysql/apple_mdm_cleanups_test.go` showing the referenced command survives a cleanup run past its retention window.

These checks are also in the [pull request template](../../../../.github/pull_request_template.md).

## Windows

Windows MDM commands have a separate, simpler cleanup (`CleanupWindowsMDMCommandQueue` in `server/datastore/mysql/microsoft_mdm.go`), controlled by `mdm.windows_command_retention`. Commands a host's wiped status depends on are kept regardless of age, so the same reference-checking concern applies to Windows commands Fleet reads back.
