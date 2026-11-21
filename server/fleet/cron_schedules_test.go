package fleet

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeTriggerSchedule struct {
	stats *CronStats
	err   error
}

func (f fakeTriggerSchedule) Trigger(context.Context) (*CronStats, bool, error) {
	return f.stats, f.err == nil, f.err
}
func (f fakeTriggerSchedule) Name() string { return string(CronPolicyMembershipCleanup) }
func (f fakeTriggerSchedule) Start()       {}

func TestTriggerPolicyMembershipCleanup(t *testing.T) {
	triggerErr := errors.New("trigger failed")
	cases := []struct {
		name     string
		schedule CronSchedule
		wantErr  bool
	}{
		{name: "triggered", schedule: fakeTriggerSchedule{}},
		{name: "run already in progress", schedule: fakeTriggerSchedule{stats: &CronStats{Status: CronStatsStatusPending}}},
		{name: "trigger fails", schedule: fakeTriggerSchedule{err: triggerErr}, wantErr: true},
		{name: "schedule not registered", wantErr: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			schedules := NewCronSchedules()
			if c.schedule != nil {
				schedules.Schedules[string(CronPolicyMembershipCleanup)] = c.schedule
			}
			err := TriggerPolicyMembershipCleanup(t.Context(), schedules)
			if c.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
