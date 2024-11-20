package fleet

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

type CronScheduleName string

// List of recognized cron schedule names.
const (
	CronAppleMDMDEPProfileAssigner  CronScheduleName = "apple_mdm_dep_profile_assigner"
	CronCleanupsThenAggregation     CronScheduleName = "cleanups_then_aggregation"
	CronFrequentCleanups            CronScheduleName = "frequent_cleanups"
	CronUsageStatistics             CronScheduleName = "usage_statistics"
	CronVulnerabilities             CronScheduleName = "vulnerabilities"
	CronAutomations                 CronScheduleName = "automations"
	CronWorkerIntegrations          CronScheduleName = "integrations"
	CronActivitiesStreaming         CronScheduleName = "activities_streaming"
	CronMDMAppleProfileManager      CronScheduleName = "mdm_apple_profile_manager"
	CronMDMWindowsProfileManager    CronScheduleName = "mdm_windows_profile_manager"
	CronAppleMDMIPhoneIPadRefetcher CronScheduleName = "apple_mdm_iphone_ipad_refetcher"
	CronAppleMDMAPNsPusher          CronScheduleName = "apple_mdm_apns_pusher"
	CronCalendar                    CronScheduleName = "calendar"
	CronUninstallSoftwareMigration  CronScheduleName = "uninstall_software_migration"
	CronMaintainedApps              CronScheduleName = "maintained_apps"
)

type CronSchedulesService interface {
	// TriggerCronSchedule attempts to trigger an ad-hoc run of the named cron schedule.
	TriggerCronSchedule(name string) error
}

func NewCronSchedules() *CronSchedules {
	return &CronSchedules{Schedules: make(map[string]CronSchedule)}
}

type CronSchedule interface {
	Trigger() (*CronStats, error)
	Name() string
	Start()
}

type CronSchedules struct {
	Schedules map[string]CronSchedule
}

// AuthzType implements authz.AuthzTyper.
func (cs *CronSchedules) AuthzType() string {
	return "cron_schedules"
}

type NewCronScheduleFunc func() (CronSchedule, error)

// StartCronSchedules starts a new cron schedule and registers it with the cron schedules struct.
func (cs *CronSchedules) StartCronSchedule(fn NewCronScheduleFunc) error {
	sched, err := fn()
	if err != nil {
		return err
	}
	sched.Start()
	cs.Schedules[sched.Name()] = sched
	return nil
}

// TriggerCronSchedule attempts to trigger an ad-hoc run of the named cron schedule.
func (cs *CronSchedules) TriggerCronSchedule(name string) error {
	sched, ok := cs.Schedules[name]
	if !ok {
		return triggerNotFoundError{name: name, msg: cs.formatSupportedTriggerNames()}
	}
	stats, err := sched.Trigger()
	switch {
	case err != nil:
		return err
	case stats != nil:
		return triggerConflictError{name: name, stats: stats}
	default:
		return nil
	}
}

// ScheduleNames returns a list of the names of all cron schedules registered with the service.
func (cs *CronSchedules) ScheduleNames() []string {
	var res []string
	for _, sched := range cs.Schedules {
		res = append(res, sched.Name())
	}
	sort.Strings(res)
	return res
}

func (cs *CronSchedules) formatSupportedTriggerNames() string {
	names := cs.ScheduleNames()
	switch len(names) {
	case 0:
		return "no supported triggers"
	case 1:
		return fmt.Sprintf("supported trigger name is %s", names[0])
	default:
		return fmt.Sprintf("supported trigger names are %s, and %s", strings.Join(names[:len(names)-1], ", "), names[len(names)-1])
	}
}

type triggerConflictError struct {
	name  string
	stats *CronStats
}

func (e triggerConflictError) Error() string {
	msg := "conflicts with current status of "
	if e.name != "" {
		msg += fmt.Sprintf("%s ", e.name)
	}
	msg += "schedule"
	if e.stats != nil {
		msg += fmt.Sprintf(": %s run started %v ago", e.stats.StatsType, time.Since(e.stats.CreatedAt).Round(time.Millisecond))
	}
	return msg
}

func (e triggerConflictError) IsConflict() bool {
	return true
}

func (e triggerConflictError) StatusCode() int {
	return http.StatusConflict
}

type triggerNotFoundError struct {
	name string
	msg  string
}

func (e triggerNotFoundError) Error() string {
	return fmt.Sprintf("invalid name; %s", e.msg)
}

func (e triggerNotFoundError) IsNotFound() bool {
	return true
}

func (e triggerNotFoundError) StatusCode() int {
	return http.StatusNotFound
}

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

// CronStatsStatus is one of four recognized statuses of cron stats (i.e. "pending", "expired", "canceled", or "completed")
type CronStatsStatus string

// List of recognized cron stats statuses.
const (
	CronStatsStatusPending   CronStatsStatus = "pending"
	CronStatsStatusExpired   CronStatsStatus = "expired"
	CronStatsStatusCompleted CronStatsStatus = "completed"
	CronStatsStatusCanceled  CronStatsStatus = "canceled"
)
