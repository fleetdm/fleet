package webhooks

import (
	"context"
	"fmt"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/hashicorp/go-multierror"
)

func TriggerHostStatusWebhook(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
) error {
	multiErr := &multierror.Error{}
	multiErr = multierror.Append(multiErr, triggerGlobalHostStatusWebhook(ctx, ds, logger))
	multiErr = multierror.Append(multiErr, triggerTeamHostStatusWebhook(ctx, ds, logger))
	return multiErr.ErrorOrNil()
}

func triggerGlobalHostStatusWebhook(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger) error {
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting app config")
	}

	if !appConfig.WebhookSettings.HostStatusWebhook.Enable {
		return nil
	}

	level.Debug(logger).Log("global", "true", "enable_host_status_webhook", "true")

	return processWebhook(ctx, ds, nil, appConfig.WebhookSettings.HostStatusWebhook)
}

func processWebhook(ctx context.Context, ds fleet.Datastore, teamID *uint, settings fleet.HostStatusWebhookSettings) error {
	total, unseen, err := ds.TotalAndUnseenHostsSince(ctx, teamID, settings.DaysCount)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting total and unseen hosts")
	}

	unseenCount := len(unseen)
	percentUnseen := float64(unseenCount) * 100.0 / float64(total)
	if percentUnseen >= settings.HostPercentage {
		url := settings.DestinationURL

		message := fmt.Sprintf(
			"More than %.2f%% of your hosts have not checked into Fleet for more than %d days. "+
				"You've been sent this message because the Host status webhook is enabled in your Fleet instance.",
			percentUnseen, settings.DaysCount,
		)
		payload := map[string]interface{}{
			"text": message,
			"data": map[string]interface{}{
				"unseen_hosts": unseenCount,
				"total_hosts":  total,
				"days_unseen":  settings.DaysCount,
				"host_ids":     unseen,
			},
		}
		if teamID != nil {
			payload["data"].(map[string]interface{})["team_id"] = *teamID
		}

		err = server.PostJSONWithTimeout(ctx, url, &payload)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "posting to %s", url)
		}
	}

	return nil
}

func triggerTeamHostStatusWebhook(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger) error {

	teams, err := ds.TeamsSummary(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting teams summary")
	}
	// We try to send a webhook for each team. If one team fails, we continue.
	multiErr := &multierror.Error{}
	for _, teamSummary := range teams {
		id := teamSummary.ID
		team, err := ds.Team(ctx, id)
		if err != nil {
			multiErr = multierror.Append(multiErr, ctxerr.Wrap(ctx, err, "getting team"))
			continue
		}
		if team.Config.WebhookSettings.HostStatusWebhook == nil || !team.Config.WebhookSettings.HostStatusWebhook.Enable {
			continue
		}
		level.Debug(logger).Log("team", id, "enable_host_status_webhook", "true")
		err = processWebhook(ctx, ds, &id, *team.Config.WebhookSettings.HostStatusWebhook)
		if err != nil {
			multiErr = multierror.Append(multiErr, ctxerr.Wrap(ctx, err, "processing webhook"))
		}
	}

	return multiErr.ErrorOrNil()
}
