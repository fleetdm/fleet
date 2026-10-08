package service

import (
	"context"
	_ "embed"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

// a batch is one notification for each of up to this many hosts
const endUserNotificationHostBatchSize = 500

// endUserNotificationScript is queued for every notification. It is the same
// script every time, so all notifications share one script_contents row and the
// URL is what tells them apart.
//
//go:embed embedded_scripts/end_user_notification.sh
var endUserNotificationScript string

func (s *Service) ExpireAndQueueNotifications(ctx context.Context) error {
	expired, err := s.ds.ExpireEndUserNotifications(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "expiring end user notifications")
	}
	if expired > 0 {
		s.logger.InfoContext(ctx, "expired end user notifications", "count", expired)
	}

	for {
		notifications, err := s.ds.ListEndUserNotificationsToDispatch(ctx, endUserNotificationHostBatchSize)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "listing end user notifications to dispatch")
		}
		if len(notifications) == 0 {
			return nil
		}

		hostIDs := make([]uint, 0, len(notifications))
		for _, notification := range notifications {
			hostIDs = append(hostIDs, notification.HostID)
		}

		// the script is the same for every notification, so nothing is rendered
		// here. GetHostScript puts each notification's URL into it when fleetd
		// fetches it.
		executionIDByHost, err := s.scriptQueue.QueueScriptForHosts(ctx, hostIDs, endUserNotificationScript)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "queueing end user notification scripts")
		}

		for _, notification := range notifications {
			executionID, ok := executionIDByHost[notification.HostID]
			if !ok {
				return ctxerr.Errorf(ctx, "no script was queued for end user notification %s on host %d",
					notification.UUID, notification.HostID)
			}
			notification.ExecutionID = &executionID
		}

		if err := s.ds.SetEndUserNotificationsDispatched(ctx, notifications); err != nil {
			return ctxerr.Wrap(ctx, err, "marking end user notifications dispatched")
		}

		// A host takes one notification at a time, so the rest of its queue is
		// now waiting on this one.
		if err := s.ds.DeferEndUserNotificationsForHosts(ctx, hostIDs); err != nil {
			return ctxerr.Wrap(ctx, err, "deferring end user notifications behind one in flight")
		}

		s.logger.InfoContext(ctx, "dispatched end user notifications", "count", len(notifications))

		// a short batch is not the end of the queue: hosts in it can have more than
		// one notification due, so this keeps going until a batch comes back empty
	}
}
