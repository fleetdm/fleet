# Unified queue implementation

Goal:
- Upcoming activities run in order as listed (one queue) and can be canceled.
- So that I have the ability to prioritize an important action (ex. lock/wipe) by cancelling all upcoming activity.

The following entities are part of the upcoming activities queue:
- **MDM commands**, including DDM "commands" (not the status checks), profile deployment and Lock/Wipe when those are based on MDM commands, but _excluding read-only queries_. Note that for iOS/iPadOS, we use MDM commands to query device information. Those should not be part of the ordered queue.
- **Script execution**, including Lock/Wipe when those are based on scripts.
- **Software install** and **uninstall**, including both Software packages (Fleet maintained apps or custom) and App-store apps.

Currently, these are handled by completely distinct processes that are not aware of / don't block each other:
- MDM commands are stored in the `nano_commands`/`nano_enrollment_queue`/... tables for Apple, and `windows_mdm_commands`/`windows_mdm_command_queue`/... for Windows. When an MDM session is started for a host, it gets the next command from the relevant queue.
- Script execution requests are sent to `fleetd` via the "Orbit notifications" returned by the `GET /config` orbit endpoint, multiple script execution requests may be sent at once and will then be processed in that order by `fleetd`: https://github.com/fleetdm/fleet/blob/b4a5a1fb49666dd3b10cfd11ccf26190ad9d2902/server/service/orbit.go#L256-L269. Those are stored in the `host_script_results` table (with `exit_code` still `NULL` when it is pending execution).
- Similarly, software installation requests are sent to `fleetd` via a different "Orbit notification" also returned by the `GET /config` orbit endpoint, and multiple software installation requests may be sent at once and will then be processed in that order by `fleetd`: https://github.com/fleetdm/fleet/blob/b4a5a1fb49666dd3b10cfd11ccf26190ad9d2902/server/service/orbit.go#L271-L277. Those are stored in the `host_software_installs` table (with `status` set to `pending`) for software packages installs/uninstalls. For App store apps, installation is via MDM commands, but the install request is saved in `host_vpp_software_installs` with the `command_uuid` pointing to the MDM command.

## New single table for all upcoming activities

The preferred implementation option at the moment is the table that sits in front of all actions that holds the arguments for these and the host they are intended for. This table is the unified queue for a host, and once items are processed they are removed and the next item in the list is marked as ready to process.

Notable challenges with that approach are:
- The table needs to be able to hold all the different types of actions and their arguments. Some actions require significant associated data (e.g. software installs can have a pre-install query, a custom install script and a post-install script).
- The logic today expects data to be immediately stored in the relevant tables even while it is pending. With that approach, we'd store it in the unified queue until it is ready to be processed, and only then we'd store it in the relevant tables (e.g. `nano_commands` or `host_script_results` etc.). If not properly addressed this may cause subtle issues in behavior, UI, summary counts, etc.
- Ensuring everything that is mutable still works after mutation (I don't think a lot of things are mutable - MDM commands certainly are not - but e.g. software install, what if the installer's information is updated?).
- Triggering execution of the next item in the queue - push or pull? In most cases, when we save the result of the previous item we could trigger (push) execution of the next one, but what about the very first item in the queue, or what if there are no more items at this moment?

### MDM commands

- For `nano_commands`/`nano_enrollment_queue`, I don't think we rely on a command being stored there while pending, AFAICT we mostly let the MDM protocol do its thing.
	* Of course it could still matter for some explicit flows, for example `fleetctl mdm run-command` followed by `fleetctl get mdm-command-results` would fail to find the command if it was in the upcoming queue but not in the MDM queue.
- Similar for Windows MDM tables.

### Script execution

- `host_script_results` is used during the "pending" phase:
	* Updating a software installer should cancel pending uninstall requests, which use script execution: https://github.com/fleetdm/fleet/blob/85dd21267700e76ee8fe184ddef2dc54692afe43/server/datastore/mysql/software_installers.go#L525
	* Batch-set software installers deletes pending uninstalls from scripts: https://github.com/fleetdm/fleet/blob/85dd21267700e76ee8fe184ddef2dc54692afe43/server/datastore/mysql/software_installers.go#L858
	* List pending host script executions: https://github.com/fleetdm/fleet/blob/85dd21267700e76ee8fe184ddef2dc54692afe43/server/datastore/mysql/scripts.go#L213
	* Check if saved script is pending execution on a host: https://github.com/fleetdm/fleet/blob/85dd21267700e76ee8fe184ddef2dc54692afe43/server/datastore/mysql/scripts.go#L238
	* Get host script execution result, which I believe can be called in the pending phase: https://github.com/fleetdm/fleet/blob/85dd21267700e76ee8fe184ddef2dc54692afe43/server/datastore/mysql/scripts.go#L274
	* Get the latest execution for a saved script on a host, which I believe has to handle the pending state: https://github.com/fleetdm/fleet/blob/85dd21267700e76ee8fe184ddef2dc54692afe43/server/datastore/mysql/scripts.go#L570
	* Cleanup of unused script contents deletes a script that isn't used, will need to take into account scripts in the new unified queue: https://github.com/fleetdm/fleet/blob/85dd21267700e76ee8fe184ddef2dc54692afe43/server/datastore/mysql/scripts.go#L1209

### Software installs

Note that ["software uninstall" requests](https://github.com/fleetdm/fleet/blob/b4a5a1fb49666dd3b10cfd11ccf26190ad9d2902/server/datastore/mysql/software_installers.go#L550) are also saved in `host_software_installs`.

- `host_software_installs` is used during the "pending" phase:
	* List pending software installs to return to orbit: https://github.com/fleetdm/fleet/blob/b4a5a1fb49666dd3b10cfd11ccf26190ad9d2902/server/datastore/mysql/software_installers.go#L23
	* Updating a software installer should cancel pending install requests: https://github.com/fleetdm/fleet/blob/b4a5a1fb49666dd3b10cfd11ccf26190ad9d2902/server/datastore/mysql/software_installers.go#L523
	* Get install results by `execution_id` (not sure if it can get called while the request is pending): https://github.com/fleetdm/fleet/blob/b4a5a1fb49666dd3b10cfd11ccf26190ad9d2902/server/datastore/mysql/software_installers.go#L602
	* Get summary of software packages installs: https://github.com/fleetdm/fleet/blob/b4a5a1fb49666dd3b10cfd11ccf26190ad9d2902/server/datastore/mysql/software_installers.go#L646
	* Filter hosts by software install status: https://github.com/fleetdm/fleet/blob/b4a5a1fb49666dd3b10cfd11ccf26190ad9d2902/server/datastore/mysql/software_installers.go#L758
	* Get host last install data seems to be called for pending installs in some situations: https://github.com/fleetdm/fleet/blob/b4a5a1fb49666dd3b10cfd11ccf26190ad9d2902/server/datastore/mysql/software_installers.go#L783
	* Batch-set software installers deletes pending installs/uninstalls and marks as "removed": https://github.com/fleetdm/fleet/blob/b4a5a1fb49666dd3b10cfd11ccf26190ad9d2902/server/datastore/mysql/software_installers.go#L858
	* List host software returns the "latest install attempt", which may be a pending one: https://github.com/fleetdm/fleet/blob/b4a5a1fb49666dd3b10cfd11ccf26190ad9d2902/server/datastore/mysql/software.go#L2218
	* Setup experience references the `execution_id` of `host_software_installs`, should it bypass the unified queue as this _needs_ to happen during setup, not blocked on any other item that might be in the queue? https://github.com/fleetdm/fleet/blob/b4a5a1fb49666dd3b10cfd11ccf26190ad9d2902/server/datastore/mysql/setup_experience.go#L312

- `host_vpp_software_installs` is used during the "pending" phase:
	* Filter hosts by software install status : https://github.com/fleetdm/fleet/blob/b4a5a1fb49666dd3b10cfd11ccf26190ad9d2902/server/datastore/mysql/software_installers.go#L705
	* List host software that is available for install : https://github.com/fleetdm/fleet/blob/b4a5a1fb49666dd3b10cfd11ccf26190ad9d2902/server/datastore/mysql/software.go#L2290
	* Get summary of VPP software installs: https://github.com/fleetdm/fleet/blob/b4a5a1fb49666dd3b10cfd11ccf26190ad9d2902/server/datastore/mysql/vpp.go#L72

### DB schema changes and migrations

- Entries in the unified queue for a given host will need to be deleted if that host is deleted: https://github.com/fleetdm/fleet/blob/85dd21267700e76ee8fe184ddef2dc54692afe43/server/datastore/mysql/hosts.go#L563
- Should secondary data be created when creating the unified queue pending record entry? E.g. the `script_contents` row for a script in a software install or script execution request? 
	* I'd say yes, otherwise we could end up taking lots of storage place in this table, defeating the purpose of the `script_contents` deduplication table.
	* This means that if the script contents are updated, the pending row should be deleted (which I believe is how it works today).

The new table will need to have at least those columns:
- uuid (unique for each action, will allow cancelation support)
- host_id (but without FK for performance reasons, as per our DB performance patterns)
- actor id, email, name and avatar (copied into the table to persist even if user is deleted? a bit like we do for `activities`?)
- fleet-initiated vs user-initiated? (e.g. a disk encryption MDM command or OS upgrade is fleet-initiated via Fleet settings, while deploying a custom profile is user-initiated by the user that added the profile to the team)
- activity "type" (MDM command, script execution, software install, software uninstall, etc.)
- a creation time and a priority column to allow for easy re-ordering (by default all are at the same prority so ordering could be `priority DESC, created_at ASC`)
- an arbitrary JSON payload for the rest of the activity-specific payload, promoting columns as needed to support the "pending" behaviour that we have today via the other tables

