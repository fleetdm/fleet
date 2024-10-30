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
###	`apple_mdm_iphone_ipad_refetcher`
###	`apple_mdm_apns_pusher`
###	`calendar`
###	`uninstall_software_migration`
###	`maintained_apps`