package webhooks

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

func TriggerHostStatusWebhook(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	appConfig *fleet.AppConfig,
) error {
	if !appConfig.WebhookSettings.HostStatusWebhook.Enable {
		return nil
	}

	level.Debug(logger).Log("enabled", "true")

	total, unseen, err := ds.TotalAndUnseenHostsSince(ctx, appConfig.WebhookSettings.HostStatusWebhook.DaysCount)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting total and unseen hosts")
	}

	percentUnseen := float64(unseen) * 100.0 / float64(total)
	if percentUnseen >= appConfig.WebhookSettings.HostStatusWebhook.HostPercentage {
		url := appConfig.WebhookSettings.HostStatusWebhook.DestinationURL

		message := fmt.Sprintf(
			"More than %.2f%% of your hosts have not checked into Fleet for more than %d days. "+
				"You've been sent this message because the Host status webhook is enabled in your Fleet instance.",
			percentUnseen, appConfig.WebhookSettings.HostStatusWebhook.DaysCount,
		)
		payload := map[string]interface{}{
			"text": message,
			"data": map[string]interface{}{
				"unseen_hosts": unseen,
				"total_hosts":  total,
				"days_unseen":  appConfig.WebhookSettings.HostStatusWebhook.DaysCount,
			},
		}

		err = server.PostJSONWithTimeout(ctx, url, &payload)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "posting to %s", url)
		}
	}

	return nil
}
