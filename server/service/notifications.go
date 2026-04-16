package service

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

////////////////////////////////////////////////////////////////////////////////
// List notifications
////////////////////////////////////////////////////////////////////////////////

func listNotificationsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.ListNotificationsRequest)

	notifications, unread, err := svc.ListNotifications(ctx, fleet.NotificationListFilter{
		IncludeDismissed: req.IncludeDismissed,
		IncludeResolved:  req.IncludeResolved,
	})
	if err != nil {
		return fleet.ListNotificationsResponse{Err: err}, nil
	}
	return fleet.ListNotificationsResponse{
		Notifications: notifications,
		UnreadCount:   unread,
	}, nil
}

func (svc *Service) ListNotifications(
	ctx context.Context, filter fleet.NotificationListFilter,
) ([]*fleet.Notification, int, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Notification{}, fleet.ActionList); err != nil {
		return nil, 0, err
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, 0, fleet.ErrNoContext
	}

	// Refresh the notification set before reading so the list always reflects
	// the current state of license / token expiries. Each producer is
	// internally idempotent (upsert-by-dedupe-key).
	svc.runNotificationProducers(ctx)

	notifications, err := svc.ds.ListNotificationsForUser(ctx, vc.UserID(), filter)
	if err != nil {
		return nil, 0, ctxerr.Wrap(ctx, err, "list notifications")
	}

	// Compute unread from the already-filtered set when the filter matches
	// the active set; otherwise fetch a canonical count.
	if !filter.IncludeDismissed && !filter.IncludeResolved {
		unread := 0
		for _, n := range notifications {
			if n.ReadAt == nil {
				unread++
			}
		}
		return notifications, unread, nil
	}

	unread, _, err := svc.ds.CountActiveNotificationsForUser(ctx, vc.UserID())
	if err != nil {
		return nil, 0, ctxerr.Wrap(ctx, err, "count notifications for unread")
	}
	return notifications, unread, nil
}

////////////////////////////////////////////////////////////////////////////////
// Notification summary (avatar-badge data source)
////////////////////////////////////////////////////////////////////////////////

func notificationSummaryEndpoint(ctx context.Context, _ interface{}, svc fleet.Service) (fleet.Errorer, error) {
	unread, active, err := svc.NotificationSummary(ctx)
	if err != nil {
		return fleet.NotificationSummaryResponse{Err: err}, nil
	}
	return fleet.NotificationSummaryResponse{UnreadCount: unread, ActiveCount: active}, nil
}

func (svc *Service) NotificationSummary(ctx context.Context) (int, int, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Notification{}, fleet.ActionRead); err != nil {
		return 0, 0, err
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return 0, 0, fleet.ErrNoContext
	}
	// Refresh so the badge reflects current state. Cheap — summary is only
	// hit when an admin lands on a page (polled by the header).
	svc.runNotificationProducers(ctx)

	unread, active, err := svc.ds.CountActiveNotificationsForUser(ctx, vc.UserID())
	if err != nil {
		return 0, 0, ctxerr.Wrap(ctx, err, "notification summary")
	}
	return unread, active, nil
}

////////////////////////////////////////////////////////////////////////////////
// Dismiss notification
////////////////////////////////////////////////////////////////////////////////

func dismissNotificationEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.DismissNotificationRequest)
	if err := svc.DismissNotification(ctx, req.ID); err != nil {
		return fleet.DismissNotificationResponse{Err: err}, nil
	}
	return fleet.DismissNotificationResponse{}, nil
}

func (svc *Service) DismissNotification(ctx context.Context, notificationID uint) error {
	// Generic check first (wide net).
	if err := svc.authz.Authorize(ctx, &fleet.Notification{}, fleet.ActionWrite); err != nil {
		return err
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}
	// Loading the notification enforces audience-matching: user can only act
	// on notifications their role is allowed to see.
	if _, err := svc.ds.NotificationByIDForUser(ctx, notificationID, vc.UserID()); err != nil {
		return ctxerr.Wrap(ctx, err, "load notification for dismiss")
	}
	if err := svc.ds.DismissNotification(ctx, notificationID, vc.UserID()); err != nil {
		return ctxerr.Wrap(ctx, err, "dismiss notification")
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Restore notification
////////////////////////////////////////////////////////////////////////////////

func restoreNotificationEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.RestoreNotificationRequest)
	if err := svc.RestoreNotification(ctx, req.ID); err != nil {
		return fleet.RestoreNotificationResponse{Err: err}, nil
	}
	return fleet.RestoreNotificationResponse{}, nil
}

func (svc *Service) RestoreNotification(ctx context.Context, notificationID uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Notification{}, fleet.ActionWrite); err != nil {
		return err
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}
	if _, err := svc.ds.NotificationByIDForUser(ctx, notificationID, vc.UserID()); err != nil {
		return ctxerr.Wrap(ctx, err, "load notification for restore")
	}
	if err := svc.ds.RestoreNotification(ctx, notificationID, vc.UserID()); err != nil {
		return ctxerr.Wrap(ctx, err, "restore notification")
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Mark read
////////////////////////////////////////////////////////////////////////////////

func markNotificationReadEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.MarkNotificationReadRequest)
	if err := svc.MarkNotificationRead(ctx, req.ID); err != nil {
		return fleet.MarkNotificationReadResponse{Err: err}, nil
	}
	return fleet.MarkNotificationReadResponse{}, nil
}

func (svc *Service) MarkNotificationRead(ctx context.Context, notificationID uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Notification{}, fleet.ActionWrite); err != nil {
		return err
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}
	if _, err := svc.ds.NotificationByIDForUser(ctx, notificationID, vc.UserID()); err != nil {
		return ctxerr.Wrap(ctx, err, "load notification for mark-read")
	}
	if err := svc.ds.MarkNotificationRead(ctx, notificationID, vc.UserID()); err != nil {
		return ctxerr.Wrap(ctx, err, "mark notification read")
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Mark all read
////////////////////////////////////////////////////////////////////////////////

func markAllNotificationsReadEndpoint(ctx context.Context, _ interface{}, svc fleet.Service) (fleet.Errorer, error) {
	if err := svc.MarkAllNotificationsRead(ctx); err != nil {
		return fleet.MarkAllNotificationsReadResponse{Err: err}, nil
	}
	return fleet.MarkAllNotificationsReadResponse{}, nil
}

func (svc *Service) MarkAllNotificationsRead(ctx context.Context) error {
	if err := svc.authz.Authorize(ctx, &fleet.Notification{}, fleet.ActionWrite); err != nil {
		return err
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}
	if err := svc.ds.MarkAllNotificationsRead(ctx, vc.UserID()); err != nil {
		return ctxerr.Wrap(ctx, err, "mark all notifications read")
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Demo: create random test notification (admin-only)
////////////////////////////////////////////////////////////////////////////////

func createDemoNotificationEndpoint(ctx context.Context, _ interface{}, svc fleet.Service) (fleet.Errorer, error) {
	n, err := svc.CreateDemoNotification(ctx)
	if err != nil {
		return fleet.CreateDemoNotificationResponse{Err: err}, nil
	}
	return fleet.CreateDemoNotificationResponse{Notification: n}, nil
}

// demoNotifications is the pool of sample notifications the demo endpoint
// picks from. Each has a unique dedupe-key suffix so calling the endpoint
// multiple times produces distinct rows.
var demoNotifications = []fleet.NotificationUpsert{
	{
		Type:     "demo_hosts_offline",
		Severity: fleet.NotificationSeverityWarning,
		Title:    "42 hosts offline for >7 days",
		Body:     "42 hosts have not checked in for more than 7 days. This may indicate network issues or decommissioned hardware.",
		CTAURL:   new("/hosts/manage?status=offline"),
		CTALabel: new("View offline hosts"),
		Audience: fleet.NotificationAudienceAdmin,
	},
	{
		Type:     "demo_policy_failures",
		Severity: fleet.NotificationSeverityError,
		Title:    "Critical policy failing on 128 hosts",
		Body:     "The policy \"Disk encryption enabled\" is failing on 128 hosts across 3 fleets.",
		CTAURL:   new("/policies"),
		CTALabel: new("View policies"),
		Audience: fleet.NotificationAudienceAdmin,
	},
	{
		Type:     "demo_vuln_cisa_kev",
		Severity: fleet.NotificationSeverityError,
		Title:    "New CISA KEV vulnerability detected",
		Body:     "CVE-2026-1234 (CVSS 9.8) was added to the CISA Known Exploited Vulnerabilities catalog and affects 57 hosts in your fleet.",
		CTAURL:   new("/software/vulnerabilities"),
		CTALabel: new("View vulnerability"),
		Audience: fleet.NotificationAudienceAdmin,
	},
	{
		Type:     "demo_software_failures",
		Severity: fleet.NotificationSeverityWarning,
		Title:    "Software install failures spiking",
		Body:     "The install failure rate for \"Google Chrome\" jumped to 23% in the last 24 hours (was 2%).",
		CTAURL:   new("/software/manage"),
		CTALabel: new("View software"),
		Audience: fleet.NotificationAudienceAdmin,
	},
	{
		Type:     "demo_fleet_update",
		Severity: fleet.NotificationSeverityInfo,
		Title:    "Fleet v5.2.0 available",
		Body:     "A new version of Fleet is available with performance improvements and 12 bug fixes.",
		CTAURL:   new("https://github.com/fleetdm/fleet/releases"),
		CTALabel: new("View release notes"),
		Audience: fleet.NotificationAudienceAdmin,
	},
	{
		Type:     "demo_seat_limit",
		Severity: fleet.NotificationSeverityWarning,
		Title:    "Approaching license seat limit",
		Body:     "You are using 475 of 500 licensed seats (95%). Contact sales to add more seats before enrollment is blocked.",
		CTAURL:   new("https://fleetdm.com/customers/register"),
		CTALabel: new("Contact sales"),
		Audience: fleet.NotificationAudienceAdmin,
	},
	{
		Type:     "demo_disk_encryption",
		Severity: fleet.NotificationSeverityWarning,
		Title:    "Disk encryption keys missing for 18 hosts",
		Body:     "18 macOS hosts have FileVault enabled but their recovery keys have not been escrowed to Fleet.",
		CTAURL:   new("/hosts/manage"),
		CTALabel: new("View hosts"),
		Audience: fleet.NotificationAudienceAdmin,
	},
	{
		Type:     "demo_webhook_failing",
		Severity: fleet.NotificationSeverityError,
		Title:    "Webhook delivery failing",
		Body:     "The vulnerability webhook to https://hooks.example.com/fleet has failed 15 consecutive times. Last error: connection timeout.",
		CTAURL:   new("/settings/integrations"),
		CTALabel: new("Check integration"),
		Audience: fleet.NotificationAudienceAdmin,
	},
}

func (svc *Service) CreateDemoNotification(ctx context.Context) (*fleet.Notification, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Notification{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	// Pick a random sample and give it a unique dedupe key so multiple calls
	// produce distinct notifications.
	sample := demoNotifications[rand.IntN(len(demoNotifications))]
	sample.DedupeKey = fmt.Sprintf("demo_%s_%d", sample.Type, rand.IntN(100000))

	n, err := svc.ds.UpsertNotification(ctx, sample)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create demo notification")
	}
	return n, nil
}
