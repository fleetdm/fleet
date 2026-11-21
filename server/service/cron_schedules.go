package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// TriggerCronSchedule attempts to trigger an ad-hoc run of the named cron schedule.
func (svc *Service) TriggerCronSchedule(ctx context.Context, name string) error {
	if err := svc.authz.Authorize(ctx, &fleet.CronSchedules{}, fleet.ActionWrite); err != nil {
		return err
	}
	return svc.cronSchedulesService.TriggerCronSchedule(ctx, name)
}

// TriggerPolicyMembershipCleanup starts the cleanup of the policy results that a policy change invalidated. The change
// already requested the cleanup, which the schedule's hourly run does if this fails.
func (svc *Service) TriggerPolicyMembershipCleanup(ctx context.Context) {
	if err := fleet.TriggerPolicyMembershipCleanup(ctx, svc.cronSchedulesService); err != nil {
		svc.logger.ErrorContext(ctx, "trigger policy membership cleanup", "err", err)
	}
}
