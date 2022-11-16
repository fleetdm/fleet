package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// // GetCronScheduleNames returns a list of the names of all cron schedules registered with the service.
// func (svc *Service) GetCronScheduleNames(ctx context.Context) []string {
// 	return svc.cronSchedulesService.GetCronScheduleNames()
// }

// TriggerCronSchedule attempts to trigger an ad-hoc run of the named cron schedule.
func (svc *Service) TriggerCronSchedule(ctx context.Context, name string) error {
	if err := svc.authz.Authorize(ctx, &fleet.CronSchedules{}, fleet.ActionWrite); err != nil {
		return err
	}
	return svc.cronSchedulesService.TriggerCronSchedule(name)
}
