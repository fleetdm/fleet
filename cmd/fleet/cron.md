# Fleet cron jobs

## Architecture

### Relevant files

- `cron.go`
- `server/fleet/cron_schedules.go`

### How it works

Cron schedules are set up on server start, in the `fleet` command's `func main`. The
`fleet.CronSchedules` type has a `StartCronSchedule` method, which takes the functions defined in `cron.go`.

## List of cron schedules

> Taken from [`server/fleet/cron_schedules.go`](https://github.com/fleetdm/fleet/blob/main/server/fleet/cron_schedules.go#L14-L29)

###	`apple_mdm_dep_profile_assigner`
Takes care of 
- Importing devices from ABM (Apple Business Manager)
- Applying the currently configured ADE profile to them so that they enroll in Fleet during the ADE
  flow.

#### Default interval

###	`cleanups_then_aggregation`
Runs several sub-jobs that do data cleanup. Examples include removing unused script contents,
expired hosts, etc.

The cron also runs several aggregation sub-jobs. Examples include aggregating query statistics,
incrementing policy violation day counts, etc.

#### Default interval
1hr

###	`frequent_cleanups`
This job also runs cleanups, but at a faster frequency than `cleanups_then_aggregation`. It's mainly
focused on cleaning up old live queries that are still hanging around. 

#### Default interval
15m

###	`usage_statistics`


###	`vulnerabilities`
###	`automations`
###	`integrations`
###	`activities_streaming`
###	`mdm_apple_profile_manager`

Despite the name, this cron job actually manages profiles for both Apple devices _and_ Windows
devices. It has 3 sub-jobs:

- `ReconcileAppleProfiles`: handles Apple configuration profiles (`.mobileconfig` files)
- `ReconcileWindowsProfiles`: handles Windows profiles (`.xml` files)
- `ReconcileAppleDeclarations`: handles Apple DDM (Declarative Device Management) profiles (`.json` files)

Each of these jobs calculates the current desired state of profile assignment based on host
configuration: for example, which labels a host has or which team it's on. It then applies that
desired state and kicks off sending/executing those profiles on the hosts.

#### Default interval
30s

###	`apple_mdm_iphone_ipad_refetcher`
###	`apple_mdm_apns_pusher`
###	`calendar`
###	`uninstall_software_migration`
###	`maintained_apps`