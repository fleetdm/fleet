# Upcoming activities

Introduced with the ["Upcoming activities run as listed (one queue)"](https://github.com/fleetdm/fleet/issues/22866) story, the upcoming activities feature (also known internally as the unified queue or the "uniq") consists of a single queue that holds the activities to execute for a specific host.

Those activities are processed in order (that is, the second activity being blocked until the first gets into a terminal state) and features like cancellation and prioritization are (or are planned to be) supported.

Types of activities that can be queued include:
* Script execution
* Software installation (custom installers or from the Fleet-maintained apps)
* VPP app installation
* Software uninstallation
* MDM commands are planned to be added to the queue (including commands to install or remove profiles)

## Implementation details

A high-level overview of the state _before_ the unified queue and the changes it brings are available in [this presentation](https://docs.google.com/presentation/d/1bIdE4wXNxDujLHu_DsO1U_0S9-vXCut4p5BgTkGME2Q/edit?usp=sharing).

The unified queue itself consists of the `upcoming_activities` table, and activity-type-specific secondary tables:
* `script_upcoming_activities` for scripts
* `software_install_upcoming_activities` for both software install and uninstall
* `vpp_app_upcoming_activities` for VPP app install

The primary table contains the meta information about the activity (user, host, priority, type, etc.) and a JSON `payload` column for secondary information that does not require any foreign key constraints or indexing. The secondary table contains foreign key references required by the activity and the corresponding `ON DELETE` behavior (e.g. if a VPP app gets deleted, the corresponding upcoming activity should be deleted as well).

The core of the unified queue lives in the `server/datastore/mysql` package and is almost an implementation detail to the outside layers - for example, when creating a script execution request, the service layer calls `Datastore.NewHostScriptExecutionRequest` and it automatically queues the request to the upcoming queue, and if the queue happens to be empty it "activates" it immediately, meaning that it makes it "in progress", ready to be executed instead of leaving it waiting for a previous activity to complete.

Something similar happens when an activity gets a (terminal state) result - regardless of if it succeeded or failed, as long as that state is terminal, the next activity, if there is one, will be "activated". Again, this is mostly transparent to the service layer: to use the same example of a script execution, when `Datastore.SetHostScriptExecutionResult` is called, the corresponding activity will be deleted (as it is not a pending activity anymore) and if there is a subsequent activity waiting, it will be "activated".

When an activity is ready to execute (to become "active"), it is updated in `upcoming_activities` to set a NON-NULL timestamp in the `activated_at` field, and in the same transaction, it is inserted in the proper table specific to the activity type, to then be picked up by the same flow that existed pre-unified queue to process those activities, that is:

* For scripts, it inserts a pending execution row in `host_script_results` with the same `execution_id` as the upcoming activity, and `fleetd` (orbit) will pick it up via its notifications;
* For software installs, it inserts a pending install row in `host_software_installs` with the same `execution_id` as the upcoming activity, and `fleetd` (orbit) will pick it up via its notifications;
* For VPP apps, since they are processed by an MDM command, it inserts a pending MDM command in `nano_commands` and `nano_enrollment_queue`, with the `command_uuid` set to the `execution_id` of the upcoming activity, and a push notification will be sent to the host to process it via MDM;
* For software uninstalls, it is a bit more complex but it inserts in both `host_script_results` and `host_software_installs` with the proper `uninstall = TRUE` flag, and the same `execution_id` is used in both tables to link them (as was done before the unified queue), and `fleetd` (orbit) will pick it up via its notifications.

The behavior described above is **very important** to ensure the queue does not become stuck, in fact those are the **two rules that every future change needs to keep in mind** when it affects the upcoming activities:

1. Whenever a new upcoming activity is enqueued, the code that creates the activity in `upcoming_activities` **MUST** call `Datastore.activateNextUpcomingActivity` inside the same transaction, with an empty string as last argument.
	* Why? This is to ensure that the activity is immediately activated if there is no other activity in the queue.
	* Example: `ds.activateNextUpcomingActivity(ctx, tx, hostID, "")`

2. Whenever an activity gets a result that is a terminal state (so it is not _pending_ anymore), the code that records the result **MUST** call `Datastore.activateNextUpcomingActivity` inside the same transaction (see notes below for an exception), with the `execution_id` of the activity that recorded the result as last argument.
	* Why? This is to delete the completed activity from the `upcoming_activities` table, and to "activate" the next activity if there is one waiting.
	* Example: `ds.activateNextUpcomingActivity(ctx, tx, hostID, result.ExecutionID)`

Whenever we add a new way to enqueue an activity or save an activity result (even if we fake a result due to e.g. a maximum number of attempts reached to process an activity), we need to make sure that these rules are followed.

Note that:

* To be extra clear, those rules are for _upcoming_, not _past_, activities.
* Queries that need to return _pending_ state must look into the `upcoming_activities` table, and depending on the query and activity type, may need to also UNION with the table that holds the in-progress state, e.g. `host_script_results` for scripts.
    - For example, `Datastore.GetHostScriptDetails` returns details of the latest execution request for a script, regardless of if it is pending, in progress or done. To do so, the query does a UNION in `upcoming_activities` and `host_script_results`, using the latest in `host_script_results` only if none exist in `upcoming_activities` (if there is a request in `upcoming_activities`, it is necessarily more recent than the ones that already have a result).
	- Many examples already exist that look into both tables, make sure to search for existing examples to help keep a consistent pattern (and fix any bugs in all places if one is found).
* For VPP app installs, the call to `activateNextUpcomingActivity` is done when the `ActivityInstalledAppStoreApp` past activity gets created, not in a transaction when the MDM command result gets saved.
    - This is due to our use of the third-party `nanomdm` package where saving MDM results is done by this package while our handling of the result is done separately in the `server/service/apple_mdm.go` file. Pretty much all of the state that Fleet saves in relation with MDM command results is not transactional with saving the MDM result itself in the `nano_*` tables.

### Cancellation

Starting with Fleet v4.67.0, cancellation of upcoming activities is supported. It is implemented as follows:

* If the upcoming activity was not _activated_ yet, then it simply deletes the row from `upcoming_activities`. A few more cleanup steps are done to ensure that if it was a Wipe/Lock script, the host's state is properly reset to "not pending wipe/lock".
* Otherwise if it was _activated_, then a new `canceled` boolean field was added to the `host_script_results`, `host_software_installs` and `host_vpp_software_installs` tables and it is set to `true` for the corresponding row. In addition to this:
    - If the software/VPP app install or script was part of the setup experience flow, the corresponding entry in setup experience is marked as "failed";
	- For VPP apps, the corresponding MDM command is marked as inactive (`active = 0`) so that it won't be sent to the host if it hasn't already been sent.

An _activated_ activity is not guaranteed to not run/execute, because the host may have already received request to process it. This is ok, Fleet will properly record any result of a canceled activity, it just won't show up in Fleet because it will just show as _canceled_ (there are new past activities for the cancelation of upcoming activities). Queries that return e.g. the last status of a software install or the last result of a saved script will ignore canceled executions.

## Testing

In general, testing is not too affected by the unified queue as we usually insert one activity, assert some things, and record a result, assert more things. This sequence works automatically because when we insert in a test, the queue is empty so the activity gets "activated" immediately, so we can record results right away.

If you do enqueue many activities, then you won't be able to record results for the subsequent ones until the previous one has a result. It's not really surprising once you understand the behavior of the queue, but it's worth mentioning.

There is an unexported Datastore field that can be used in tests, `Datastore.testActivateSpecificNextActivities`, when you need to control exactly what activity (or activities, this is a slice of strings) will be activated next. If you use this field, make sure to call `t.Cleanup(func() { ds.testActivateSpecificNextActivities = nil })` at the beginning of the test to avoid leaking this field to other tests.
