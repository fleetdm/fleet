# Fleetctl trigger

## Goal

As a user, I would like to trigger a set of async jobs using `fleetctl`. For example, I’d like to
trigger a vuln scan, or an MDM dep sync. 

The proposed solution to accomplish this goal enables a new CLI command: 
`fleetctl trigger --name <NAME>`.   

## Background

Currently, the Fleet server uses the `schedule` package to create sets of defined jobs that are run
serially at defined intervals. The initial schedule interval must be specified at the point the
schedule is instantiated via `schedule.New`. Optionally, `schedule. WithConfigReloadInterval`
accepts a reload interval function. If specified, the reload interval function is periodically
called and its return value becomes the new the schedule interval. This mechanism allows the
schedule interval to be modified by user, for example, by changing the app config; however, there no
mechanism currently to trigger async jobs on an ad hoc basis.

## Proposal

### New CLI command `fleetctl trigger --name <NAME>`
- Upon this command, the CLI client makes a request to a new authenticated endpoint (see below) to
  trigger an ad hoc run of the named schedule. 

### New `schedule` option `WithTrigger` 
- This option adds a `trigger` channel on the `schedule` struct that will trigger an ad hoc run of
  the scheduled jobs.
- The trigger channel for each `schedule` is exposed via a new `schedules` map on the `Service` struct. 

### New endpoint `GET /trigger?name={:name}` 
- The request handler first calls `ds.Lock` to check if the named schedule is locked.
- If the named schedule is unlocked, request handler sends a trigger signal on the schedule's
  trigger channel and the server responds with status `202 Accepted`.  
- If the named schedule is locked (presumably because the schedule is currently running), the server
  responds with status `409 Conflict` and includes a message indicating the schedule is currently
  locked. It is up to the user to retry. To facilitate retries, the response message could be expanded to
  include additional status information, such as the expiration time of the current lock.

### Schedule locks
- Currently, lock duration is based on the schedule interval. 
  - Once an instance takes the lock, it will hold the lock for the duration of the interval, even
    after it has completed the jobs in the schedule. 
  - For long-running jobs, the lock may expire before the current instance completes its run,
    meaning that it is currently possible for another instance to start an overlapping job.
  - If the lockholder instance is terminated or killed, locks are not released, which may frustrate
    a user's attempt to configure a shorter schedule interval before the lock held by the dead
    instance expires.

- Under this proposal, locks become more dynamic.
  - Current lockholder releases its lock once it finishes running the schedule.
  - Graceful shutdown process handles release of locks upon termination signals. Jobs are
    preemptable and an interrupt function must be specified for each job, 
    e.g., `schedule.New(...).WithJob("job_name", jobFn, interruptFn)`.
  - As a fallback for cases that can't be handled via graceful shutdown (e.g., `SIGKILL`), the
    expiration for a lock is initially set to a relatively short default duration (e.g., 5 minutes).
    The expiration is then periodically extended by the current instance so long as scheduled jobs
    are running.  If the current instance dies without graceful shutdown, the lock will only be held
    by the dead instance for a short period.

### Additional UX considerations 
- What are some potential options that would be useful for the `fleetctl trigger` command?
  - Request the current status of the the named schedule without triggering a new run.
    For example, `--status` could provide the user with the running time of the schedule (this would
    require that we expand the `locks` table to include additional timestamp information, such as
    lock start time and lock release time).
  - Other useful options?

- What rules should determine when the interval ticker resets? Consider the following cases where
  `s.scheduleInterval = 1*time.Hour`: 
  - The schedule is triggered at 55 minutes into the 1-hour interval and takes 1 minute to complete.
    When should the schedule run again? 
    (a) after 4 minutes; 
    (b) after 1 hour; 
    (c) after 1 hour plus 4 minutes; 
    (d) other  
  - The schedule is triggered at 55 minutes into the 1-hour interval and takes 11 minutes to complete.
    When should the schedule run again? 
    (a) immediately; 
    (b) after 1 hour; 
    (c) after 54 minutes;
    (d) other

- What should be logged?
  - Debug log if schedule runtime exceeds schedule interval to aid detection/troubleshooting of
    long-running jobs.
  - Other useful logs? 







