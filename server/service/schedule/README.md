# `schedule`: the Fleet cron job machinery

Fleet has several pieces of functionality that are implemented as cron jobs, which run on a
schedule. Package `schedule` implements the machinery needed for queueing and running these jobs.

## List of cron jobs

See [server/fleet/cron_schedules.go](../../../server/fleet/cron_schedules.go) for a list of the
currently implemented cron jobs and information about what they do.

Cron jobs are created and registered in the `cmd/fleet` package because they have to be run at
server start. The actual implementation of the cron job logic is usually elsewhere however,
typically in a service layer method (and related datastore methods).

## How to add a new cron job

See [this PR](https://github.com/fleetdm/fleet/pull/21959/files) for a nice example of how to add a
simple cron job.

1. **Do you need a new cron job?** You can add sub-jobs to an existing cron job; for example, if
   you're adding some functionality for cleaning up unused data, you might want to implement it as a
   sub-job in the [`cleanups_then_aggregation` cron](https://github.com/fleetdm/fleet/blob/65e374c85c32a7dd582aa1d438161663a4abc43c/cmd/fleet/cron.go#L793).
2. **Add a cron job name.** If you determine that you do need a new cron job, create a descriptive
   name in cron_schedules.go. Make sure you leave a comment explaining what the job does.
3. **Implement your functionality.** Do this wherever it makes sense. In the example PR, the
   functionality exists in the `server/mdm/maintainedapps/ingest.go` file. However, you'll most likely
   implement a service layer method and related datastore layer methods. 
4. **Add a function that returns a `*schedule.Schedule` in [`cmd/fleet/cron.go`](../../../cmd/fleet/cron.go).** This function will be used to
   register your cron job so it can actually run. This function should call whatever you implemented
   in step 3. This is also where you can set the interval on which your cron job will run.
5. **Register the cron job in [`cmd/fleet/serve.go`](../../../cmd/fleet/serve.go).** You'll use `cronSchedules.StartCronSchedule` to
   register the cron job by passing it an anonymous function that calls the function you wrote in
   step 3.
