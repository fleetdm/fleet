package service

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/schedule"
	"github.com/stretchr/testify/require"
)

func TestTriggerCronScheduleAuth(t *testing.T) {
	ds := new(mock.Store)

	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{StartCronSchedules: []TestNewScheduleFunc{
		func(ctx context.Context, ds fleet.Datastore) fleet.NewCronScheduleFunc {
			return func() (fleet.CronSchedule, error) {
				s := schedule.New(
					ctx, "test_sched", "id", 1*time.Second, schedule.NopLocker{}, schedule.NopStatsStore{},
					schedule.WithJob("test_job", func(ctx context.Context) error {
						return nil
					}),
				)
				return s, nil
			}
		},
	}})

	testCases := []struct {
		name       string
		user       *fleet.User
		shouldFail bool
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			true,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
		},
		{
			"team admin",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			true,
		},
		{
			"team maintainer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			true,
		},
		{
			"team observer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
		},
		{
			"user",
			&fleet.User{ID: 777},
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			err := svc.TriggerCronSchedule(ctx, "test_sched")
			if tt.shouldFail {
				require.Error(t, err)
				require.Equal(t, (&authz.Forbidden{}).Error(), err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCronSchedulesService(t *testing.T) {
	ds := new(mock.Store)
	locker := schedule.SetupMockLocker("test_sched", "id", time.Now().Add(-1*time.Hour))
	statsStore := schedule.SetUpMockStatsStore("test_sched", fleet.CronStats{
		ID:        1,
		StatsType: fleet.CronStatsTypeScheduled,
		Name:      "test_sched",
		Instance:  "id",
		CreatedAt: time.Now().Add(-1 * time.Hour),
		UpdatedAt: time.Now().Add(-1 * time.Hour),
		Status:    fleet.CronStatsStatusCompleted,
	})
	jobsDone := uint32(0)

	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{StartCronSchedules: []TestNewScheduleFunc{
		func(ctx context.Context, ds fleet.Datastore) fleet.NewCronScheduleFunc {
			return func() (fleet.CronSchedule, error) {
				s := schedule.New(
					ctx, "test_sched", "id", 1*time.Second, locker, statsStore,
					schedule.WithJob("test_job", func(ctx context.Context) error {
						time.Sleep(100 * time.Millisecond)
						atomic.AddUint32(&jobsDone, 1)
						return nil
					}),
				)
				return s, nil
			}
		},
	}})

	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	require.NoError(t, svc.TriggerCronSchedule(ctx, "test_sched")) // first trigger sent ok and will run successfully

	time.Sleep(10 * time.Millisecond)
	require.Error(t, svc.TriggerCronSchedule(ctx, "test_sched")) // error because first job is pending

	require.Error(t, svc.TriggerCronSchedule(ctx, "test_sched")) // error because first job is pending

	time.Sleep(2 * time.Second)
	require.Error(t, svc.TriggerCronSchedule(ctx, "test_sched_2")) // error because unrecognized name

	time.Sleep(100 * time.Millisecond)
	require.Equal(t, uint32(3), atomic.LoadUint32(&jobsDone)) // 2 regularly scheduled (at 1s and 2s) plus 1 triggered
}
