# Fleet cron jobs

## Architecture

### Relevant files

- `cron.go`
- `server/fleet/cron_schedules.go`

### How it works

Cron schedules are set up on server start, in the `fleet` command's `func main`. The
`fleet.CronSchedules` type has a `StartCronSchedule` method, which takes the functions defined in `cron.go`.

## List of cron schedules

###	`apple_mdm_dep_profile_assigner`
###	`cleanups_then_aggregation`
###	`frequent_cleanups`
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