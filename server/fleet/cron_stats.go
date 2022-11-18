package fleet

import "time"

// CronStats represents statistics recorded in connection with a named set of jobs (sometimes
// referred to as a "cron" or "schedule"). Each record represents a separate "run" of the named job set.
type CronStats struct {
	ID int `db:"id"`
	// StatsType denotes whether the stats are associated with a run of jobs that was "triggered"
	// (i.e. run on an ad-hoc basis) or "scheduled" (i.e. run on a regularly scheduled interval).
	StatsType CronStatsType `db:"stats_type"`
	// Name is the name of the set of jobs (i.e. the schedule name).
	Name string `db:"name"`
	// Instance is the unique id of the Fleet instance that performed the run of jobs represented by
	// the stats.
	Instance string `db:"instance"`
	// CreatedAt is the time the stats record was created. It is assumed to be the start of the run.
	CreatedAt time.Time `db:"created_at"`
	// UpdatedAt is the time the stats record was last updated. For a "completed" run, this assumed
	// to be the end of the run.
	UpdatedAt time.Time `db:"updated_at"`
	// Status is the current status of the run. Recognized statuses are "pending", "completed", and
	// "expired".
	Status CronStatsStatus `db:"status"`
}

// CronStatsType is one of two recognized types of cron stats (i.e. "scheduled" or "triggered")
type CronStatsType string

// List of recognized cron stats types.
const (
	CronStatsTypeScheduled CronStatsType = "scheduled"
	CronStatsTypeTriggered CronStatsType = "triggered"
)

// CronStatsStatus is one of three recognized statuses of cron stats (i.e. "pending", "expired" or "completed")
type CronStatsStatus string

// List of recognized cron stats statuses.
const (
	CronStatsStatusPending   CronStatsStatus = "pending"
	CronStatsStatusExpired   CronStatsStatus = "expired"
	CronStatsStatusCompleted CronStatsStatus = "completed"
)
