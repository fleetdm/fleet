package webhooks

import (
	"context"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
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

	total, unseen, err := ds.TotalAndUnseenHostsSince(appConfig.WebhookSettings.HostStatusWebhook.DaysCount)
	if err != nil {
		return errors.Wrap(err, "getting total and unseen hosts")
	}

	percentUnseen := float64(unseen) * 100.0 / float64(total)
	if percentUnseen >= appConfig.WebhookSettings.HostStatusWebhook.HostPercentage {
		url := appConfig.WebhookSettings.HostStatusWebhook.DestinationURL
		// TODOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOO HEREEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEE
		err2 := server.PostJSONWithTimeout(url)
		if err2 != nil {
			return err2
		}
	}

	return nil
}
