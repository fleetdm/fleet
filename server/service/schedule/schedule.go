// Package schedule allows periodic run of a list of jobs.
//
// Type Schedule allows grouping a set of Jobs to run at specific intervals.
// Each Job is executed serially in the order they were added to the Schedule.
package schedule

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// ReloadInterval reloads and returns a new interval.
type ReloadInterval func(ctx context.Context) (time.Duration, error)

// Schedule runs a list of jobs serially at a given schedule.
//
// Each job is executed one after the other in the order they were added.
// If one of the job fails, an error is logged and the scheduler
// continues with the next.
type Schedule struct {
	ctx        context.Context
	name       string
	instanceID string
	logger     log.Logger

	defaultPrevRunCreatedAt time.Time // default timestamp of previous run for the schedule if none exists, time.Now if not set

	mu                sync.Mutex // protects schedInterval and intervalStartedAt
	schedInterval     time.Duration
	intervalStartedAt time.Time // start time of the most recent run of the scheduled jobs

	trigger chan struct{}
	done    chan struct{}

	configReloadInterval   time.Duration
	configReloadIntervalFn ReloadInterval

	locker Locker

	altLockName string

	jobs []Job

	statsStore CronStatsStore

	runOnce bool
}

// JobFn is the signature of a Job.
type JobFn func(context.Context) error

// Job represents a job that can be added to Scheduler.
type Job struct {
	// ID is the unique identifier for the job.
	ID string
	// Fn is the job itself.
	Fn JobFn
}

// Locker allows a Schedule to acquire a lock before running jobs.
type Locker interface {
	Lock(ctx context.Context, scheduleName string, scheduleInstanceID string, expiration time.Duration) (bool, error)
	Unlock(ctx context.Context, scheduleName string, scheduleInstanceID string) error
}

// CronStatsStore allows a Schedule to store and retrieve statistics pertaining to the Schedule
type CronStatsStore interface {
	// GetLatestCronStats returns a slice of no more than two cron stats records, where index 0 (if
	// present) is the most recently created scheduled run, and index 1 (if present) represents a
	// triggered run that is currently pending.
	GetLatestCronStats(ctx context.Context, name string) ([]fleet.CronStats, error)
	// InsertCronStats inserts cron stats for the named cron schedule
	InsertCronStats(ctx context.Context, statsType fleet.CronStatsType, name string, instance string, status fleet.CronStatsStatus) (int, error)
	// UpdateCronStats updates the status of the identified cron stats record
	UpdateCronStats(ctx context.Context, id int, status fleet.CronStatsStatus) error
}

// Option allows configuring a Schedule.
type Option func(*Schedule)

// WithLogger sets a logger for the Schedule.
func WithLogger(l log.Logger) Option {
	return func(s *Schedule) {
		s.logger = log.With(l, "schedule", s.name)
	}
}

// WithConfigReloadInterval allows setting a reload interval function,
// that will allow updating the interval of a running schedule.
//
// If not set, then the schedule performs no interval reloading.
func WithConfigReloadInterval(interval time.Duration, fn ReloadInterval) Option {
	return func(s *Schedule) {
		s.configReloadInterval = interval
		s.configReloadIntervalFn = fn
	}
}

// WithAltLockID sets an alternative identifier to use when acquiring the lock.
//
// If not set, then the Schedule's name is used for acquiring the lock.
func WithAltLockID(name string) Option {
	return func(s *Schedule) {
		s.altLockName = name
	}
}

// WithJob adds a job to the Schedule.
//
// Each job is executed in the order they are added.
func WithJob(id string, fn JobFn) Option {
	return func(s *Schedule) {
		s.jobs = append(s.jobs, Job{
			ID: id,
			Fn: fn,
		})
	}
}

// WithRunOnce sets the Schedule to run only once.
func WithRunOnce(once bool) Option {
	return func(s *Schedule) {
		s.runOnce = once
	}
}

// WithDefaultPrevRunCreatedAt sets the default time to use for the previous
// run of the schedule if it never ran yet. If not specified, the current time
// is used. This affects when the schedule starts running after Fleet is
// started, e.g. if the schedule has an interval of 1h and has no previous run
// recorded, by default its first run after Fleet starts will be in 1h.
func WithDefaultPrevRunCreatedAt(tm time.Time) Option {
	return func(s *Schedule) {
		s.defaultPrevRunCreatedAt = tm
	}
}

// New creates and returns a Schedule.
// Jobs are added with the WithJob Option.
//
// The jobs are executed serially in order at the provided interval.
//
// The provided locker is used to acquire/release a lock before running the jobs.
// The provided name and instanceID of the Schedule is used as the locking identifier.
func New(
	ctx context.Context,
	name string,
	instanceID string,
	interval time.Duration,
	locker Locker,
	statsStore CronStatsStore,
	opts ...Option,
) *Schedule {
	sch := &Schedule{
		ctx:                  ctx,
		name:                 name,
		instanceID:           instanceID,
		logger:               log.NewNopLogger(),
		trigger:              make(chan struct{}),
		done:                 make(chan struct{}),
		configReloadInterval: 1 * time.Hour, // by default we will check for updated config once per hour
		schedInterval:        truncateSecondsWithFloor(interval),
		locker:               locker,
		statsStore:           statsStore,
	}
	for _, fn := range opts {
		fn(sch)
	}
	if sch.logger == nil {
		sch.logger = log.NewNopLogger()
	}
	sch.logger = log.With(sch.logger, "instanceID", instanceID)
	return sch
}

// Start starts running the added jobs.
//
// All jobs must be added before calling Start.
func (s *Schedule) Start() {
	prevScheduledRun, _, err := s.GetLatestStats()
	if err != nil {
		level.Error(s.logger).Log("err", "start schedule", "details", err)
		ctxerr.Handle(s.ctx, err)
	}

	// if there is no previous run, set the start time to the specified default
	// time, falling back to current time.
	startedAt := prevScheduledRun.CreatedAt
	if startedAt.IsZero() {
		startedAt = s.defaultPrevRunCreatedAt
		if startedAt.IsZero() {
			startedAt = time.Now()
		}
	} else if s.runOnce && prevScheduledRun.Status == fleet.CronStatsStatusCompleted {
		// If job is set to run once, and it already ran, then nothing to do
		return
	}
	s.setIntervalStartedAt(startedAt)

	initialWait := 10 * time.Second
	if schedInterval := s.getSchedInterval(); schedInterval < initialWait {
		initialWait = schedInterval
	}
	schedTicker := time.NewTicker(initialWait)

	var g sync.WaitGroup
	g.Add(+1)
	go func() {
		defer func() {
			s.releaseLock()
			g.Done()
		}()

		for {
			level.Debug(s.logger).Log("msg", fmt.Sprintf("%v remaining until next tick", s.getRemainingInterval(s.intervalStartedAt)))

			select {
			case <-s.ctx.Done():
				close(s.trigger)
				schedTicker.Stop()
				return

			case <-s.trigger:
				level.Debug(s.logger).Log("msg", "done, trigger received")

				ok, cancelHold := s.holdLock()
				if !ok {
					level.Debug(s.logger).Log("msg", "unable to acquire lock")
					continue
				}

				s.runWithStats(fleet.CronStatsTypeTriggered)

				prevScheduledRun, _, err := s.GetLatestStats()
				if err != nil {
					level.Error(s.logger).Log("err", "trigger get cron stats", "details", err)
					ctxerr.Handle(s.ctx, err)
				}

				clearScheduleChannels(s.trigger, schedTicker.C) // in case another signal arrived during this run

				intervalStartedAt := s.getIntervalStartedAt()
				if prevScheduledRun.CreatedAt.After(intervalStartedAt) {
					// if there's a diff between the datastore and our local value, we use the
					// more recent timestamp and update our local value accordingly
					s.setIntervalStartedAt(prevScheduledRun.CreatedAt)
					intervalStartedAt = s.getIntervalStartedAt()
				}

				// if the triggered run spanned the schedule interval, we need to wait until the start of the next full interval
				schedInterval := s.getSchedInterval()
				if time.Since(intervalStartedAt) > schedInterval+1*time.Second { // we use 2s tolerance here because MySQL timestamps are truncated to 1s
					newStart := intervalStartedAt.Add(time.Since(intervalStartedAt).Truncate(schedInterval)) // advances start time by the number of full interval elasped
					s.setIntervalStartedAt(newStart)
					schedTicker.Reset(s.getRemainingInterval(newStart))
					level.Debug(s.logger).Log("msg", fmt.Sprintf("triggered run spanned schedule interval, new wait %v", s.getRemainingInterval(newStart)))
				}

				cancelHold()

			case <-schedTicker.C:
				level.Debug(s.logger).Log("msg", "done, tick received")

				schedInterval := s.getSchedInterval()

				prevScheduledRun, prevTriggeredRun, err := s.GetLatestStats()
				if err != nil {
					level.Error(s.logger).Log("err", "get cron stats", "details", err)
					ctxerr.Handle(s.ctx, err)
					// skip ahead to the next interval
					schedTicker.Reset(schedInterval)
					continue
				}

				if prevScheduledRun.Status == fleet.CronStatsStatusPending || prevTriggeredRun.Status == fleet.CronStatsStatusPending {
					// skip ahead to the next interval
					level.Info(s.logger).Log("msg", fmt.Sprintf("pending job might still be running, wait %v", schedInterval))
					schedTicker.Reset(schedInterval)
					continue
				}

				intervalStartedAt := s.getIntervalStartedAt()
				if prevScheduledRun.CreatedAt.After(intervalStartedAt) {
					// if there's a diff between the datastore and our local value, we use the
					// more recent timestamp and update our local value accordingly
					s.setIntervalStartedAt(prevScheduledRun.CreatedAt)
					intervalStartedAt = s.getIntervalStartedAt()
				}

				if time.Since(intervalStartedAt) < schedInterval {
					// wait for the remaining interval plus a small buffer
					newWait := s.getRemainingInterval(intervalStartedAt) + 100*time.Millisecond
					level.Info(s.logger).Log("msg", fmt.Sprintf("wait remaining interval %v", newWait))
					schedTicker.Reset(newWait)
					continue
				}

				// if the previous run took longer than the schedule interval, we wait until the start of the next full interval
				if time.Since(intervalStartedAt) > schedInterval+2*time.Second { // we use a 2s tolerance here because MySQL timestamps are truncated to 1s
					newStart := intervalStartedAt.Add(time.Since(intervalStartedAt).Truncate(schedInterval)) // advances start time by the number of full interval elasped
					s.setIntervalStartedAt(newStart)
					schedTicker.Reset(s.getRemainingInterval(newStart))
					level.Debug(s.logger).Log("msg", fmt.Sprintf("prior run spanned schedule interval, new wait %v", s.getRemainingInterval(newStart)))
					continue
				}

				ok, cancelHold := s.holdLock()
				if !ok {
					level.Debug(s.logger).Log("msg", "unable to acquire lock")
					schedTicker.Reset(schedInterval)
					continue
				}

				newStart := time.Now()
				s.setIntervalStartedAt(newStart)

				s.runWithStats(fleet.CronStatsTypeScheduled)

				// we need to re-synchronize this schedule instance so that the next scheduled run
				// starts at the beginning of the next full interval
				//
				// for example, if the interval is 1hr and the schedule takes 0.2 hrs to run
				// then we wait 0.8 hrs until the next time we run the schedule, or if the
				// the schedule takes 1.5 hrs to run then we wait 0.5 hrs (skipping the scheduled
				// tick that would have overlapped with the 1.5hrs running time)
				schedInterval = s.getSchedInterval()
				if time.Since(newStart) > schedInterval {
					level.Info(s.logger).Log("msg", fmt.Sprintf("total runtime (%v) exceeded schedule interval (%v)", time.Since(newStart), schedInterval))
					newStart = newStart.Add(time.Since(newStart).Truncate(schedInterval)) // advances start time by the number of full interval elasped
					s.setIntervalStartedAt(newStart)
				}
				clearScheduleChannels(s.trigger, schedTicker.C) // in case another signal arrived during this run

				schedTicker.Reset(s.getRemainingInterval(newStart))
				cancelHold()
			}
		}
	}()

	if s.configReloadIntervalFn != nil {
		// WithConfigReloadInterval option applies so we periodically check for config updates and
		// reset the schedInterval for the previous loop
		g.Add(+1)
		go func() {
			defer g.Done()

			configTicker := time.NewTicker(s.configReloadInterval)
			for {
				select {
				case <-s.ctx.Done():
					configTicker.Stop()
					return
				case <-configTicker.C:
					prevInterval := s.getSchedInterval()
					newInterval, err := s.configReloadIntervalFn(s.ctx)
					if err != nil {
						level.Error(s.logger).Log("err", "schedule interval config reload failed", "details", err)
						ctxerr.Handle(s.ctx, err)
						continue
					}

					newInterval = truncateSecondsWithFloor(newInterval)
					if newInterval <= 0 {
						level.Debug(s.logger).Log("msg", "config reload interval method returned invalid interval")
						continue
					}
					if prevInterval == newInterval {
						continue
					}
					s.setSchedInterval(newInterval)

					intervalStartedAt := s.getIntervalStartedAt()
					newWait := 10 * time.Millisecond
					if time.Since(intervalStartedAt) < newInterval {
						newWait = s.getRemainingInterval(intervalStartedAt)
					}

					clearScheduleChannels(s.trigger, schedTicker.C)
					schedTicker.Reset(newWait)

					level.Debug(s.logger).Log("msg", fmt.Sprintf("new schedule interval %v", newInterval))
					level.Debug(s.logger).Log("msg", fmt.Sprintf("time until next schedule tick %v", newWait))
				}
			}
		}()
	}

	go func() {
		g.Wait()
		level.Debug(s.logger).Log("msg", "close schedule")
		close(s.done) // communicates that the scheduler has finished running its goroutines
		schedTicker.Stop()
	}()
}

// Trigger attempts to signal the schedule to start an ad-hoc run of all jobs after first checking
// whether another run is pending. If another run is already pending, it returns available status
// information for the pending run.
//
// Note that no distinction is made in the return value between the
// case where the signal is published to the trigger channel and the case where the trigger channel
// is blocked or otherwise unavailable to publish the signal. From the caller's perspective, both
// cases are deemed to be equivalent.
func (s *Schedule) Trigger() (*fleet.CronStats, error) {
	sched, trig, err := s.GetLatestStats()
	switch {
	case err != nil:
		return nil, err
	case sched.Status == fleet.CronStatsStatusPending:
		return &sched, nil
	case trig.Status == fleet.CronStatsStatusPending:
		return &trig, nil
	default:
		// ok
	}

	select {
	case s.trigger <- struct{}{}:
		// ok
	default:
		level.Debug(s.logger).Log("msg", "trigger channel not available")
	}
	return nil, nil
}

// Name returns the name of the schedule.
func (s *Schedule) Name() string {
	return s.name
}

// runWithStats runs all jobs in the schedule. Prior to starting the run, it creates a
// record in the database for the provided stats type with "pending" status. After completing the
// run, the stats record is updated to "completed" status.
func (s *Schedule) runWithStats(statsType fleet.CronStatsType) {
	statsID, err := s.insertStats(statsType, fleet.CronStatsStatusPending)
	if err != nil {
		level.Error(s.logger).Log("err", fmt.Sprintf("insert cron stats %s", s.name), "details", err)
		ctxerr.Handle(s.ctx, err)
	}
	level.Info(s.logger).Log("status", "pending")

	s.runAllJobs()

	if err := s.updateStats(statsID, fleet.CronStatsStatusCompleted); err != nil {
		level.Error(s.logger).Log("err", fmt.Sprintf("update cron stats %s", s.name), "details", err)
		ctxerr.Handle(s.ctx, err)
	}
	level.Info(s.logger).Log("status", "completed")
}

// runAllJobs runs all jobs in the schedule.
func (s *Schedule) runAllJobs() {
	for _, job := range s.jobs {
		level.Debug(s.logger).Log("msg", "starting", "jobID", job.ID)
		if err := runJob(s.ctx, job.Fn); err != nil {
			level.Error(s.logger).Log("err", "running job", "details", err, "jobID", job.ID)
			ctxerr.Handle(s.ctx, err)
		}
	}
}

// runJob executes the job function with panic recovery.
func runJob(ctx context.Context, fn JobFn) (err error) {
	defer func() {
		if os.Getenv("TEST_CRON_NO_RECOVER") != "1" { // for detecting panics in tests
			if r := recover(); r != nil {
				err = fmt.Errorf("%v\n%s", r, string(debug.Stack()))
			}
		}
	}()

	if err := fn(ctx); err != nil {
		return err
	}
	return nil
}

// Done returns a channel that will be closed when the scheduler's context is done
// and it has finished running its goroutines.
func (s *Schedule) Done() <-chan struct{} {
	return s.done
}

// getScheduleInterval returns the schedule interval
func (s *Schedule) getSchedInterval() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.schedInterval
}

// setScheduleInterval sets the schedule interval after truncating the duration to seconds and
// applying a one second floor (e.g., 600ms becomes 1s, 1300ms becomes 2s, 1000ms becomes 2s)
func (s *Schedule) setSchedInterval(interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.schedInterval = truncateSecondsWithFloor(interval)
}

// getIntervalStartedAt returns the start time of the current schedule interval.
func (s *Schedule) getIntervalStartedAt() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.intervalStartedAt
}

// setIntervalStartedAt sets the start time of the current schedule interval. The start time is
// rounded down to the nearest second.
func (s *Schedule) setIntervalStartedAt(start time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.intervalStartedAt = start.Truncate(1 * time.Second)
}

// getRemainingInterval returns the interval minus the remainder of dividing the time since state by
// the interval
func (s *Schedule) getRemainingInterval(start time.Time) time.Duration {
	interval := s.getSchedInterval()
	if interval == 0 {
		return 0
	}

	return interval - (time.Since(start) % interval)
}

func (s *Schedule) acquireLock() bool {
	ok, err := s.locker.Lock(s.ctx, s.getLockName(), s.instanceID, s.getSchedInterval())
	if err != nil {
		level.Error(s.logger).Log("msg", "lock failed", "err", err)
		ctxerr.Handle(s.ctx, err)
		return false
	}
	if !ok {
		level.Debug(s.logger).Log("msg", "not the lock leader, skipping")
		return false
	}
	return true
}

func (s *Schedule) releaseLock() {
	err := s.locker.Unlock(s.ctx, s.getLockName(), s.instanceID)
	if err != nil {
		level.Error(s.logger).Log("msg", "unlock failed", "err", err)
		ctxerr.Handle(s.ctx, err)
	}
}

// holdLock attempts to acquire a schedule lock. If it successfully acquires the lock, it starts a
// goroutine that periodically extends the lock, and it returns `true` along with a
// context.CancelFunc that will end the goroutine and release the lock. If it is unable to initially
// acquire a lock, it returns `false, nil`. The maximum duration of the hold is two hours.
func (s *Schedule) holdLock() (bool, context.CancelFunc) {
	if ok := s.acquireLock(); !ok {
		return false, nil
	}

	ctx, cancelFn := context.WithCancel(s.ctx)

	go func() {
		t := time.NewTimer(s.getSchedInterval() * 8 / 10) // hold timer is 80% of schedule interval
		for {
			select {
			case <-ctx.Done():
				if !t.Stop() {
					<-t.C
				}
				s.releaseLock()
				return
			case <-t.C:
				s.acquireLock()
				t.Reset(s.getSchedInterval() * 8 / 10)
			}
		}
	}()

	return true, cancelFn
}

func (s *Schedule) GetLatestStats() (fleet.CronStats, fleet.CronStats, error) {
	var scheduled, triggered fleet.CronStats

	cs, err := s.statsStore.GetLatestCronStats(s.ctx, s.name)
	if err != nil {
		return fleet.CronStats{}, fleet.CronStats{}, err
	}
	if len(cs) > 2 {
		return fleet.CronStats{}, fleet.CronStats{}, fmt.Errorf("get latest stats expected length to be no more than two but got length: %d", len(cs))
	}

	for _, stats := range cs {
		switch stats.StatsType {
		case fleet.CronStatsTypeScheduled:
			scheduled = stats
		case fleet.CronStatsTypeTriggered:
			triggered = stats
		default:
			level.Error(s.logger).Log("msg", fmt.Sprintf("get latest stats unexpected type: %s", stats.StatsType))
		}
	}

	return scheduled, triggered, nil
}

func (s *Schedule) insertStats(statsType fleet.CronStatsType, status fleet.CronStatsStatus) (int, error) {
	return s.statsStore.InsertCronStats(s.ctx, statsType, s.name, s.instanceID, status)
}

func (s *Schedule) updateStats(id int, status fleet.CronStatsStatus) error {
	return s.statsStore.UpdateCronStats(s.ctx, id, status)
}

func (s *Schedule) getLockName() string {
	name := s.name
	if s.altLockName != "" {
		name = s.altLockName
	}
	return name
}

// clearScheduleChannels performs a non-blocking select on the ticker and trigger channel in order
// to drain each channel. It is intended for use in cases where a signal may have been published to
// a channel during a pending run, in which case the expected behavior is for the signal to be dropped.
func clearScheduleChannels(trigger chan struct{}, ticker <-chan time.Time) {
	for {
		select {
		case <-trigger:
			// pull trigger signal from channel
		case <-ticker:
			// pull ticker signal from channel
		default:
			return
		}
	}
}

// truncateSecondsWithFloor returns the result of truncating the duration to seconds and
// and applying a one second floor (e.g., 600ms becomes 1s, 1300ms becomes 2s, 1000ms becomes 2s)
func truncateSecondsWithFloor(d time.Duration) time.Duration {
	if d <= 1*time.Second {
		return 1 * time.Second
	}
	return d.Truncate(time.Second)
}
